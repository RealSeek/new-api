package controller

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayRSGatewaySkipsRequestValidationAndPreservesResponse(t *testing.T) {
	service.InitHttpClient()

	requestBody := `{"model":"gateway-model","custom_field":{"kept":true}}`
	type capturedRequest struct {
		body             string
		path             string
		trace            string
		authorization    string
		turnMetadata     string
		requestReadError error
	}
	received := make(chan capturedRequest, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		received <- capturedRequest{
			body:             string(body),
			path:             r.URL.Path,
			trace:            r.URL.Query().Get("trace"),
			authorization:    r.Header.Get("Authorization"),
			turnMetadata:     r.Header.Get("X-Codex-Turn-Metadata"),
			requestReadError: err,
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Upstream-Test", "preserved")
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("upstream raw response"))
	}))
	t.Cleanup(upstream.Close)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions?trace=trace-value",
		strings.NewReader(requestBody),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("X-Codex-Turn-Metadata", "turn-123")
	common.SetContextKey(ctx, constant.ContextKeyChannelType, constant.ChannelTypeRSGateway)
	common.SetContextKey(ctx, constant.ContextKeyChannelId, 61)
	common.SetContextKey(ctx, constant.ContextKeyChannelBaseUrl, upstream.URL)
	common.SetContextKey(ctx, constant.ContextKeyChannelKey, "upstream-key")
	common.SetContextKey(ctx, constant.ContextKeyOriginalModel, "gateway-model")

	Relay(ctx, types.RelayFormatOpenAI)

	captured := <-received
	require.NoError(t, captured.requestReadError)
	assert.Equal(t, requestBody, captured.body)
	assert.Equal(t, "/v1/chat/completions", captured.path)
	assert.Equal(t, "trace-value", captured.trace)
	assert.Equal(t, "Bearer upstream-key", captured.authorization)
	assert.Equal(t, "turn-123", captured.turnMetadata)
	assert.Equal(t, http.StatusTeapot, recorder.Code)
	assert.Equal(t, "text/plain", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "preserved", recorder.Header().Get("X-Upstream-Test"))
	assert.Equal(t, "upstream raw response", recorder.Body.String())
}
