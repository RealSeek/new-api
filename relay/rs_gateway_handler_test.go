package relay

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		t.Fatal("等待 RS Gateway 结算回写超时")
	}
}

func TestRSGatewayRequestIsStream(t *testing.T) {
	assert.True(t, rsGatewayRequestIsStream([]byte(`{"model":"gpt-5.6-sol","stream":true}`)))
	assert.False(t, rsGatewayRequestIsStream([]byte(`{"model":"gpt-5.6-sol"}`)))
	assert.False(t, rsGatewayRequestIsStream([]byte(`not-json`)))
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
	_, _ = tracker.Write([]byte("data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":120,\"cache_read_input_tokens\":80}}}\n\n"))
	_, _ = tracker.Write([]byte("data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":15}}\n\n"))

	usage := tracker.Usage()
	require.NotNil(t, usage)
	assert.Equal(t, 120, usage.PromptTokens)
	assert.Equal(t, 15, usage.CompletionTokens)
	assert.Equal(t, 135, usage.TotalTokens)
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
