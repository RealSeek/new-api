package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	relayhelper "github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// RSGatewayHelper 将已完成鉴权和渠道选择的请求原样交给网关。
func RSGatewayHelper(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	startTime := time.Now()
	info.TokenId = c.GetInt("token_id")
	if info.TokenId <= 0 {
		info.TokenId = common.GetContextKeyInt(c, constant.ContextKeyTokenId)
	}
	info.TokenName = c.GetString("token_name")
	if strings.TrimSpace(info.TokenName) == "" && info.TokenId > 0 {
		if token, err := model.GetTokenById(info.TokenId); err == nil {
			info.TokenName = strings.TrimSpace(token.Name)
		}
	}
	info.InitChannelMeta(c)
	recordError := func(content string, statusCode int) {
		if !constant.ErrorLogEnabled {
			return
		}
		model.RecordErrorLog(
			c,
			info.UserId,
			info.ChannelId,
			info.OriginModelName,
			c.GetString("token_name"),
			content,
			info.TokenId,
			int(time.Since(startTime).Seconds()),
			info.IsStream,
			info.UsingGroup,
			map[string]interface{}{
				"request_path": c.Request.URL.Path,
				"status_code":  statusCode,
				"rs_gateway":   true,
			},
		)
	}
	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		recordError(fmt.Sprintf("网关不支持 API 类型 %d", info.ApiType), http.StatusBadGateway)
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	storage, err := common.GetBodyStorage(c)
	if err != nil {
		recordError("读取请求体失败", http.StatusBadRequest)
		return types.NewError(err, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
	}
	requestBody, err := storage.Bytes()
	if err != nil {
		recordError("读取请求体失败", http.StatusBadRequest)
		return types.NewError(err, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
	}
	info.IsStream = rsGatewayRequestIsStream(requestBody)
	var imageRequest *dto.ImageRequest
	meta := &types.TokenCountMeta{}
	if info.RelayMode == relayconstant.RelayModeImagesGenerations || info.RelayMode == relayconstant.RelayModeImagesEdits {
		// 复用图片参数校验与价格倍率，转发仍使用原始请求体。
		imageRequest, err = relayhelper.GetAndValidOpenAIImageRequest(c, info.RelayMode)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		info.IsStream = imageRequest.IsStream(c.Request)
		meta = imageRequest.GetTokenCountMeta()
	}
	c.Set(string(constant.ContextKeyIsStream), info.IsStream)
	priceData, err := relayhelper.ModelPriceHelper(c, info, 0, meta)
	if err != nil {
		return types.NewErrorWithStatusCode(err, types.ErrorCodeModelPriceError, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	tokenId := info.TokenId
	if info.IsPlayground {
		tokenId = 0
	}
	if err := model.BeginRSGatewaySettlement(info.RequestId, info.UserId, tokenId, info.ChannelId); err != nil {
		logger.LogError(c, fmt.Sprintf("gateway ledger unavailable, request_id=%s: %v", info.RequestId, err))
		return types.NewErrorWithStatusCode(errors.New("gateway settlement unavailable"), types.ErrorCodeUpdateDataError,
			http.StatusServiceUnavailable, types.ErrOptionWithSkipRetry())
	}
	settled := false
	defer func() {
		if settled {
			return
		}
		if info.Billing != nil {
			info.Billing.Refund(c)
			return
		}
		if err := model.CompleteRSGatewaySettlement(info.RequestId, model.RSGatewayCompletion{Refunded: true}); err != nil {
			logger.LogError(c, fmt.Sprintf("gateway refund pending, request_id=%s: %v", info.RequestId, err))
		}
	}()
	if !priceData.FreeModel {
		if apiErr := service.PreConsumeBilling(c, priceData.QuotaToPreConsume, info); apiErr != nil {
			return apiErr
		}
	}
	response, err := adaptor.DoRequest(c, info, common.NewReplayableBodyReader(storage))
	if err != nil {
		message, statusCode, clientCanceled := classifyRSGatewayRequestError(err, c.Request.Context().Err())
		if clientCanceled {
			logger.LogInfo(c, message)
			c.Status(statusCode)
			return nil
		}
		recordError(message, statusCode)
		return types.NewErrorWithStatusCode(
			errors.New(message),
			types.ErrorCodeDoRequestFailed,
			statusCode,
			types.ErrOptionWithSkipRetry(),
		)
	}
	httpResponse, ok := response.(*http.Response)
	if !ok || httpResponse == nil {
		recordError("网关返回了无效响应", http.StatusBadGateway)
		return types.NewError(errors.New("invalid http response"), types.ErrorCodeDoRequestFailed, types.ErrOptionWithSkipRetry())
	}
	defer httpResponse.Body.Close()
	if strings.HasPrefix(strings.ToLower(httpResponse.Header.Get("Content-Type")), "text/event-stream") {
		info.IsStream = true
		c.Set(string(constant.ContextKeyIsStream), true)
	}
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		newAPIError := service.RelayErrorHandler(c.Request.Context(), httpResponse, false)
		recordError(newAPIError.MaskSensitiveErrorWithStatusCode(), newAPIError.StatusCode)
		return newAPIError
	}
	usageTracker := newRSGatewayUsageTracker(info.IsStream)
	var imageUsage *dto.Usage
	streamInterrupted := false
	defer func() {
		usage := imageUsage
		if imageRequest != nil {
			if usage == nil {
				return
			}
		} else {
			usage = usageTracker.Usage()
			if shouldRefundRSGatewayPreConsumed(usage, streamInterrupted, usageTracker.failed) {
				logger.LogInfo(c, "网关流未完成且没有计费用量，退还预扣额度")
				return
			}
		}
		quota := service.PostTextConsumeQuota(c, info, usage, nil)
		if quota < 0 {
			return
		}
		settled = true
	}()
	if imageRequest != nil {
		var apiErr *types.NewAPIError
		imageUsage, apiErr = rsGatewayImageResponse(c, info, httpResponse)
		if apiErr != nil {
			recordError(apiErr.MaskSensitiveErrorWithStatusCode(), apiErr.StatusCode)
		}
		return apiErr
	}

	connectionHeaders := make(map[string]struct{})
	for _, name := range strings.Split(httpResponse.Header.Get("Connection"), ",") {
		if name = strings.ToLower(strings.TrimSpace(name)); name != "" {
			connectionHeaders[name] = struct{}{}
		}
	}
	for name, values := range httpResponse.Header {
		lowerName := strings.ToLower(name)
		if _, skip := connectionHeaders[lowerName]; skip {
			continue
		}
		switch lowerName {
		case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
			"te", "trailer", "transfer-encoding", "upgrade":
			continue
		}
		c.Writer.Header().Del(name)
		for _, value := range values {
			c.Writer.Header().Add(name, value)
		}
	}
	c.Writer.WriteHeader(httpResponse.StatusCode)

	if strings.HasPrefix(strings.ToLower(httpResponse.Header.Get("Content-Type")), "text/event-stream") {
		flusher, canFlush := c.Writer.(http.Flusher)
		if canFlush {
			buffer := make([]byte, 32*1024)
			for {
				read, readErr := httpResponse.Body.Read(buffer)
				if read > 0 {
					// 网关使用自定义直通循环，首个实际响应块需要在这里记录首字时间。
					info.SetFirstResponseTime()
					_, _ = usageTracker.Write(buffer[:read])
					if _, writeErr := c.Writer.Write(buffer[:read]); writeErr != nil {
						streamInterrupted = true
						logger.LogError(c, "网关响应写入失败: "+writeErr.Error())
						return nil
					}
					flusher.Flush()
				}
				if readErr != nil {
					if !errors.Is(readErr, io.EOF) {
						streamInterrupted = true
						logger.LogError(c, "网关响应读取失败: "+readErr.Error())
					}
					return nil
				}
			}
		}
	}
	if _, err = io.Copy(io.MultiWriter(c.Writer, usageTracker), httpResponse.Body); err != nil {
		streamInterrupted = true
		logger.LogError(c, "网关响应转发失败: "+err.Error())
	}
	return nil
}

func shouldRefundRSGatewayPreConsumed(usage *dto.Usage, interrupted, failed bool) bool {
	return usage == nil && (interrupted || failed)
}

// 图片响应复用原有计数和用量解析，避免大图被通用用量捕获上限截断。
func rsGatewayImageResponse(c *gin.Context, info *relaycommon.RelayInfo, response *http.Response) (*dto.Usage, *types.NewAPIError) {
	var usage *dto.Usage
	var apiErr *types.NewAPIError
	if info.IsStream {
		usage, apiErr = openai.OpenaiImageStreamHandler(c, info, response)
	} else {
		usage, apiErr = openai.OpenaiImageHandler(c, info, response)
	}
	if apiErr != nil {
		return nil, apiErr
	}
	// 按张计费不依赖 token；沿用普通图片通道的结算标记。
	if info.PriceData.UsePrice && usage.TotalTokens == 0 {
		usage.PromptTokens = 1
		usage.TotalTokens = 1
	}
	return usage, nil
}

func classifyRSGatewayRequestError(requestErr, requestContextErr error) (string, int, bool) {
	if errors.Is(requestContextErr, context.Canceled) || errors.Is(requestErr, context.Canceled) {
		// 499 与 Nginx 的“客户端关闭请求”状态保持一致。
		return "客户端已取消网关请求", 499, true
	}
	if errors.Is(requestContextErr, context.DeadlineExceeded) || errors.Is(requestErr, context.DeadlineExceeded) {
		return "网关请求超时", http.StatusGatewayTimeout, false
	}
	return "网关连接失败", http.StatusBadGateway, false
}

const rsGatewayUsageCaptureLimit = 8 << 20

type rsGatewayUsageTracker struct {
	stream  bool
	pending []byte
	body    []byte
	usage   dto.Usage
	found   bool
	failed  bool
}

func newRSGatewayUsageTracker(stream bool) *rsGatewayUsageTracker {
	return &rsGatewayUsageTracker{stream: stream}
}

func (t *rsGatewayUsageTracker) Write(data []byte) (int, error) {
	if !t.stream {
		remaining := rsGatewayUsageCaptureLimit - len(t.body)
		if remaining > 0 {
			t.body = append(t.body, data[:min(len(data), remaining)]...)
		}
		return len(data), nil
	}
	t.pending = append(t.pending, data...)
	for {
		lineEnd := bytes.IndexByte(t.pending, '\n')
		if lineEnd < 0 {
			break
		}
		t.consumeSSELine(t.pending[:lineEnd])
		t.pending = t.pending[lineEnd+1:]
	}
	if len(t.pending) > rsGatewayUsageCaptureLimit {
		t.pending = append([]byte(nil), t.pending[len(t.pending)-rsGatewayUsageCaptureLimit:]...)
	}
	return len(data), nil
}

func (t *rsGatewayUsageTracker) consumeSSELine(line []byte) {
	line = bytes.TrimSpace(line)
	if !bytes.HasPrefix(line, []byte("data:")) {
		return
	}
	payload := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
	if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
		return
	}
	t.mergeJSON(payload)
}

func (t *rsGatewayUsageTracker) mergeJSON(data []byte) {
	var value interface{}
	if json.Unmarshal(data, &value) != nil {
		return
	}
	t.failed = t.failed || rsGatewayPayloadFailed(value)
	walkRSGatewayUsage(value, func(usage dto.Usage) {
		normalizeRSGatewayUsage(&usage)
		mergeRSGatewayUsage(&t.usage, &usage)
		t.found = t.usage.TotalTokens > 0
	})
}

func rsGatewayPayloadFailed(value interface{}) bool {
	payload, ok := value.(map[string]interface{})
	if !ok {
		return false
	}
	eventType, _ := payload["type"].(string)
	switch eventType {
	case "error", "response.error", "response.failed":
		return true
	}
	errorValue, exists := payload["error"]
	return exists && errorValue != nil
}

func (t *rsGatewayUsageTracker) Usage() *dto.Usage {
	if t.stream {
		if len(t.pending) > 0 {
			t.consumeSSELine(t.pending)
			t.pending = nil
		}
	} else {
		t.mergeJSON(t.body)
	}
	if !t.found {
		return nil
	}
	usage := t.usage
	return &usage
}

func rsGatewayRequestIsStream(body []byte) bool {
	var request struct {
		Stream bool `json:"stream"`
	}
	return json.Unmarshal(body, &request) == nil && request.Stream
}

func walkRSGatewayUsage(value interface{}, visit func(dto.Usage)) {
	switch current := value.(type) {
	case map[string]interface{}:
		for key, child := range current {
			if key == "usageMetadata" {
				if raw, err := json.Marshal(child); err == nil {
					var metadata dto.GeminiUsageMetadata
					if json.Unmarshal(raw, &metadata) == nil {
						if usage := relayconvert.UsageFromGeminiMetadata(&metadata, 0); usage != nil {
							visit(*usage)
						}
					}
				}
			}
			if key == "usage" {
				if raw, err := json.Marshal(child); err == nil {
					var usage dto.Usage
					if json.Unmarshal(raw, &usage) == nil {
						if childMap, ok := child.(map[string]interface{}); ok && hasRSGatewayClaudeCacheUsage(childMap) {
							var claudeUsage dto.ClaudeUsage
							if json.Unmarshal(raw, &claudeUsage) == nil {
								if claudeUsage.CacheReadInputTokens > 0 {
									usage.PromptTokensDetails.CachedTokens = claudeUsage.CacheReadInputTokens
								}
								cacheCreationTokens := claudeUsage.GetCacheCreationTotalTokens()
								if cacheCreationTokens == 0 {
									cacheCreationTokens = claudeUsage.ClaudeCacheCreation5mTokens + claudeUsage.ClaudeCacheCreation1hTokens
								}
								if cacheCreationTokens > 0 {
									usage.PromptTokensDetails.CachedCreationTokens = cacheCreationTokens
								}
							}
						}
						visit(usage)
					}
				}
			}
			walkRSGatewayUsage(child, visit)
		}
	case []interface{}:
		for _, child := range current {
			walkRSGatewayUsage(child, visit)
		}
	}
}

func hasRSGatewayClaudeCacheUsage(usage map[string]interface{}) bool {
	for _, key := range []string{
		"cache_read_input_tokens",
		"cache_creation_input_tokens",
		"cache_creation",
		"claude_cache_creation_5_m_tokens",
		"claude_cache_creation_1_h_tokens",
	} {
		if _, ok := usage[key]; ok {
			return true
		}
	}
	return false
}

func normalizeRSGatewayUsage(usage *dto.Usage) {
	if usage.PromptTokens == 0 {
		usage.PromptTokens = usage.InputTokens
	}
	if usage.CompletionTokens == 0 {
		usage.CompletionTokens = usage.OutputTokens
	}
	if usage.InputTokens == 0 {
		usage.InputTokens = usage.PromptTokens
	}
	if usage.OutputTokens == 0 {
		usage.OutputTokens = usage.CompletionTokens
	}
	if usage.InputTokensDetails != nil && usage.PromptTokensDetails.CachedTokens == 0 {
		usage.PromptTokensDetails = *usage.InputTokensDetails
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
}

func mergeRSGatewayUsage(target, next *dto.Usage) {
	if next.PromptTokens > 0 {
		target.PromptTokens = next.PromptTokens
		target.InputTokens = next.InputTokens
		target.PromptTokensDetails = next.PromptTokensDetails
		target.InputTokensDetails = next.InputTokensDetails
		target.ClaudeCacheCreation5mTokens = next.ClaudeCacheCreation5mTokens
		target.ClaudeCacheCreation1hTokens = next.ClaudeCacheCreation1hTokens
	}
	if next.CompletionTokens > 0 {
		target.CompletionTokens = next.CompletionTokens
		target.OutputTokens = next.OutputTokens
		target.CompletionTokenDetails = next.CompletionTokenDetails
	}
	if next.BillingUsage != nil {
		target.BillingUsage = dto.CloneBillingUsage(next.BillingUsage)
	}
	if next.UsageSemantic != "" {
		target.UsageSemantic = next.UsageSemantic
	}
	if next.UsageSource != "" {
		target.UsageSource = next.UsageSource
	}
	target.TotalTokens = target.PromptTokens + target.CompletionTokens
}
