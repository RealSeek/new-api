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
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayhelper "github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

// RSGatewayHelper 将已完成鉴权和渠道选择的请求原样交给 RS Gateway。
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
		recordError(fmt.Sprintf("RS Gateway 不支持 API 类型 %d", info.ApiType), http.StatusBadGateway)
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
	c.Set(string(constant.ContextKeyIsStream), info.IsStream)
	priceData, err := relayhelper.ModelPriceHelper(c, info, 0, &types.TokenCountMeta{})
	if err != nil {
		return types.NewErrorWithStatusCode(err, types.ErrorCodeModelPriceError, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	if !priceData.FreeModel {
		if apiErr := service.PreConsumeBilling(c, priceData.QuotaToPreConsume, info); apiErr != nil {
			return apiErr
		}
	}
	settled := false
	defer func() {
		if !settled && info.Billing != nil {
			info.Billing.Refund(c)
		}
	}()
	response, err := adaptor.DoRequest(c, info, common.NewReplayableBodyReader(storage))
	if err != nil {
		recordError("RS Gateway 连接失败", http.StatusBadGateway)
		return types.NewError(err, types.ErrorCodeDoRequestFailed, types.ErrOptionWithSkipRetry())
	}
	httpResponse, ok := response.(*http.Response)
	if !ok || httpResponse == nil {
		recordError("RS Gateway 返回了无效响应", http.StatusBadGateway)
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
	defer func() {
		usage := usageTracker.Usage()
		quota := service.PostTextConsumeQuota(c, info, usage, nil)
		if httpResponse.Request != nil && httpResponse.Request.URL != nil {
			callbackURL := *httpResponse.Request.URL
			callbackURL.Path = "/api/new-api-usage"
			callbackURL.RawPath = ""
			callbackURL.RawQuery = ""
			reportRSGatewayUsage(callbackURL.String(), info.ApiKey, info.RequestId, quota)
		}
		settled = true
	}()

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
					// RS Gateway 使用自定义直通循环，首个实际响应块需要在这里记录首字时间。
					info.SetFirstResponseTime()
					_, _ = usageTracker.Write(buffer[:read])
					if _, writeErr := c.Writer.Write(buffer[:read]); writeErr != nil {
						logger.LogError(c, "RS Gateway 响应写入失败: "+writeErr.Error())
						return nil
					}
					flusher.Flush()
				}
				if readErr != nil {
					if !errors.Is(readErr, io.EOF) {
						logger.LogError(c, "RS Gateway 响应读取失败: "+readErr.Error())
					}
					return nil
				}
			}
		}
	}
	if _, err = io.Copy(io.MultiWriter(c.Writer, usageTracker), httpResponse.Body); err != nil {
		logger.LogError(c, "RS Gateway 响应转发失败: "+err.Error())
	}
	return nil
}

func reportRSGatewayUsage(callbackURL, apiKey, requestID string, quota int) {
	if callbackURL == "" || apiKey == "" || requestID == "" || quota < 0 {
		return
	}
	payload, err := json.Marshal(map[string]interface{}{
		"request_id": requestID,
		"quota":      quota,
	})
	if err != nil {
		return
	}
	client := service.GetHttpClient()
	if client == nil {
		client = http.DefaultClient
	}
	gopool.Go(func() {
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Duration(attempt*attempt) * 250 * time.Millisecond)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(payload))
			if reqErr != nil {
				cancel()
				lastErr = reqErr
				break
			}
			req.Header.Set("Authorization", "Bearer "+apiKey)
			req.Header.Set("Content-Type", "application/json")
			resp, requestErr := client.Do(req)
			if requestErr == nil {
				_ = resp.Body.Close()
				if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
					cancel()
					return
				}
				lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			} else {
				lastErr = requestErr
			}
			cancel()
		}
		logger.LogError(context.Background(), fmt.Sprintf("RS Gateway 结算回写失败: %v", lastErr))
	})
}

const rsGatewayUsageCaptureLimit = 8 << 20

type rsGatewayUsageTracker struct {
	stream  bool
	pending []byte
	body    []byte
	usage   dto.Usage
	found   bool
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
	walkRSGatewayUsage(value, func(usage dto.Usage) {
		normalizeRSGatewayUsage(&usage)
		mergeRSGatewayUsage(&t.usage, &usage)
		t.found = t.usage.TotalTokens > 0
	})
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
			if key == "usage" {
				if raw, err := json.Marshal(child); err == nil {
					var usage dto.Usage
					if json.Unmarshal(raw, &usage) == nil {
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
	}
	if next.CompletionTokens > 0 {
		target.CompletionTokens = next.CompletionTokens
		target.OutputTokens = next.OutputTokens
		target.CompletionTokenDetails = next.CompletionTokenDetails
	}
	target.TotalTokens = target.PromptTokens + target.CompletionTokens
}
