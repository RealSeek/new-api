package relay

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

// RSGatewayHelper 将已完成鉴权和渠道选择的请求原样交给 RS Gateway。
func RSGatewayHelper(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	info.InitChannelMeta(c)
	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return types.NewError(err, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
	}
	response, err := adaptor.DoRequest(c, info, common.NewReplayableBodyReader(storage))
	if err != nil {
		return types.NewError(err, types.ErrorCodeDoRequestFailed, types.ErrOptionWithSkipRetry())
	}
	httpResponse, ok := response.(*http.Response)
	if !ok || httpResponse == nil {
		return types.NewError(errors.New("invalid http response"), types.ErrorCodeDoRequestFailed, types.ErrOptionWithSkipRetry())
	}
	defer httpResponse.Body.Close()

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
