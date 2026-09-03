package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyRSGatewayRequestError(t *testing.T) {
	tests := []struct {
		name              string
		requestErr        error
		requestContextErr error
		wantMessage       string
		wantStatus        int
		wantCanceled      bool
	}{
		{
			name:              "客户端主动取消",
			requestErr:        errors.New("上游请求失败"),
			requestContextErr: context.Canceled,
			wantMessage:       "客户端已取消网关请求",
			wantStatus:        499,
			wantCanceled:      true,
		},
		{
			name:         "传输层收到取消",
			requestErr:   fmt.Errorf("发送失败: %w", context.Canceled),
			wantMessage:  "客户端已取消网关请求",
			wantStatus:   499,
			wantCanceled: true,
		},
		{
			name:         "网关请求超时",
			requestErr:   fmt.Errorf("发送失败: %w", context.DeadlineExceeded),
			wantMessage:  "网关请求超时",
			wantStatus:   http.StatusGatewayTimeout,
			wantCanceled: false,
		},
		{
			name:         "网关连接失败",
			requestErr:   errors.New("connection refused"),
			wantMessage:  "网关连接失败",
			wantStatus:   http.StatusBadGateway,
			wantCanceled: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message, statusCode, clientCanceled := classifyRSGatewayRequestError(test.requestErr, test.requestContextErr)
			assert.Equal(t, test.wantMessage, message)
			assert.Equal(t, test.wantStatus, statusCode)
			assert.Equal(t, test.wantCanceled, clientCanceled)
		})
	}
}

func TestReportRSGatewayUsage(t *testing.T) {
	received := make(chan map[string]interface{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer gateway-key", r.Header.Get("Authorization"))
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		received <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	reportRSGatewayUsage(server.URL+"/api/new-api-usage", "gateway-key", "request-42", 1250)
	select {
	case payload := <-received:
		assert.Equal(t, "request-42", payload["request_id"])
		assert.Equal(t, float64(1250), payload["quota"])
	case <-time.After(2 * time.Second):
		t.Fatal("等待网关结算回写超时")
	}
}

func TestRSGatewayRequestIsStream(t *testing.T) {
	assert.True(t, rsGatewayRequestIsStream([]byte(`{"model":"gpt-5.6-sol","stream":true}`)))
	assert.False(t, rsGatewayRequestIsStream([]byte(`{"model":"gpt-5.6-sol"}`)))
	assert.False(t, rsGatewayRequestIsStream([]byte(`not-json`)))
}

func TestRSGatewayFirstResponseTimeOnlyRecordsOnce(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	info := relaycommon.GenRSGatewayRelayInfo(c, types.RelayFormatOpenAIResponses, nil)

	time.Sleep(time.Millisecond)
	info.SetFirstResponseTime()
	first := info.FirstResponseTime
	require.True(t, first.After(info.StartTime))

	time.Sleep(time.Millisecond)
	info.SetFirstResponseTime()
	assert.Equal(t, first, info.FirstResponseTime)
}

func TestRSGatewayUsageTrackerReadsResponsesSSE(t *testing.T) {
	tracker := newRSGatewayUsageTracker(true)
	chunks := []string{
		"event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n",
		"event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":53654,\"output_tokens\":367,\"total_tokens\":54021,\"input_tokens_details\":{\"cached_tokens\":39424}}}}\n\n",
		"data: [DONE]\n\n",
	}
	for _, chunk := range chunks {
		_, err := tracker.Write([]byte(chunk))
		require.NoError(t, err)
	}

	usage := tracker.Usage()
	require.NotNil(t, usage)
	assert.Equal(t, 53654, usage.PromptTokens)
	assert.Equal(t, 367, usage.CompletionTokens)
	assert.Equal(t, 54021, usage.TotalTokens)
	assert.Equal(t, 39424, usage.PromptTokensDetails.CachedTokens)
}

func TestRSGatewayUsageTrackerMergesClaudeStreamUsage(t *testing.T) {
	tracker := newRSGatewayUsageTracker(true)
	_, _ = tracker.Write([]byte("data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":7948,\"cache_read_input_tokens\":0,\"output_tokens\":0}}}\n\n"))
	_, _ = tracker.Write([]byte("data: {\"type\":\"message_delta\",\"usage\":{\"input_tokens\":16314,\"cache_read_input_tokens\":14656,\"output_tokens\":66,\"billing_usage\":{\"source\":\"oai_chat\",\"semantic\":\"openai\",\"openai_usage\":{\"prompt_tokens\":16314,\"completion_tokens\":66,\"total_tokens\":16380,\"prompt_tokens_details\":{\"cached_tokens\":14656}}}}}\n\n"))

	usage := tracker.Usage()
	require.NotNil(t, usage)
	assert.Equal(t, 16314, usage.PromptTokens)
	assert.Equal(t, 14656, usage.PromptTokensDetails.CachedTokens)
	assert.Equal(t, 66, usage.CompletionTokens)
	assert.Equal(t, 16380, usage.TotalTokens)
	require.NotNil(t, usage.BillingUsage)
	require.NotNil(t, usage.BillingUsage.OpenAIUsage)
	assert.Equal(t, 14656, usage.BillingUsage.OpenAIUsage.PromptTokensDetails.CachedTokens)
}

func TestRSGatewayUsageTrackerReadsNonStreamUsage(t *testing.T) {
	tracker := newRSGatewayUsageTracker(false)
	_, err := tracker.Write([]byte(`{"id":"resp_1","usage":{"prompt_tokens":30,"completion_tokens":7,"total_tokens":37}}`))
	require.NoError(t, err)

	usage := tracker.Usage()
	require.NotNil(t, usage)
	assert.Equal(t, 30, usage.PromptTokens)
	assert.Equal(t, 7, usage.CompletionTokens)
	assert.Equal(t, 37, usage.TotalTokens)
}
