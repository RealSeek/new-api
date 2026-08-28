package relay

import (
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
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

// RSGatewayHelper 将已完成鉴权和渠道选择的请求原样交给 RS Gateway。
func RSGatewayHelper(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	startTime := time.Now()
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
	// RS Gateway 负责实际计费和用量解析，New-API 仍需保留一条消费日志，
	// 方便按 New-API Key 在管理端追踪请求。用量字段由网关侧日志提供，这里不重复估算。
	defer func() {
		if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
			recordError(fmt.Sprintf("RS Gateway 返回 HTTP %d", httpResponse.StatusCode), httpResponse.StatusCode)
			return
		}
		model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
			ChannelId:      info.ChannelId,
			ModelName:      info.OriginModelName,
			TokenName:      c.GetString("token_name"),
			TokenId:        info.TokenId,
			UseTimeSeconds: int(time.Since(startTime).Seconds()),
			IsStream:       info.IsStream,
			Group:          info.UsingGroup,
			Other: map[string]interface{}{
				"rs_gateway":  true,
				"status_code": httpResponse.StatusCode,
			},
		})
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
	if _, err = io.Copy(c.Writer, httpResponse.Body); err != nil {
		logger.LogError(c, "RS Gateway 响应转发失败: "+err.Error())
	}
	return nil
}
