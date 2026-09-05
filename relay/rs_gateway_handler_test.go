package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	relayhelper "github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRSGatewayImageResponse(t *testing.T) {
	previousTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = previousTimeout })
	for _, test := range []struct {
		name       string
		body       string
		stream     bool
		usePrice   bool
		wantTokens int
		wantN      float64
		wantError  bool
	}{
		{name: "无用量按实际图片数量计费", body: `{"data":[{"url":"https://example.com/1.png"},{"url":"https://example.com/2.png"}]}`, usePrice: true, wantTokens: 1, wantN: 2},
		{name: "大图尾部用量不丢失", body: `{"data":[{"b64_json":"` + strings.Repeat("A", 9<<20) + `"}],"usage":{"input_tokens":12,"output_tokens":25,"total_tokens":37}}`, usePrice: true, wantTokens: 37, wantN: 1},
		{name: "按token计费保留图片用量", body: `{"data":[{"b64_json":"image"}],"usage":{"input_tokens":12,"output_tokens":25}}`, wantTokens: 37, wantN: 3},
		{name: "按token计费不虚构用量", body: `{"data":[{"b64_json":"image"}]}`, wantN: 3},
		{name: "流式图片无用量仍计费", body: "event: image_generation.completed\ndata: {\"type\":\"image_generation.completed\",\"b64_json\":\"image\"}\n\ndata: [DONE]\n\n", stream: true, usePrice: true, wantTokens: 1, wantN: 1},
		{name: "错误响应不进入结算", body: `{"error":{"type":"server_error","message":"failed"}}`, usePrice: true, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
			info := relaycommon.GenRSGatewayRelayInfo(c, types.RelayFormatOpenAIImage, nil)
			info.ChannelMeta = &relaycommon.ChannelMeta{}
			info.IsStream = test.stream
			info.PriceData = hosttypes.PriceData{UsePrice: test.usePrice}
			info.PriceData.AddOtherRatio("n", 3)
			contentType := "application/json"
			if test.stream {
				contentType = "text/event-stream"
			}
			response := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {contentType}},
				Body:       io.NopCloser(strings.NewReader(test.body)),
			}
			usage, apiErr := rsGatewayImageResponse(c, info, response)
			if test.wantError {
				require.NotNil(t, apiErr)
				assert.Nil(t, usage)
				assert.False(t, c.Writer.Written())
				return
			}
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, test.wantTokens, usage.TotalTokens)
			assert.Equal(t, test.wantN, info.PriceData.OtherRatios()["n"])
			if test.stream {
				assert.Contains(t, recorder.Body.String(), "data: [DONE]")
			} else {
				assert.True(t, test.body == recorder.Body.String(), "原始图片响应必须完整保留")
			}
		})
	}
}

func TestRSGatewayImageSettlementWithoutUsage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.Channel{}))
	previousDB := model.DB
	previousBatch, previousLog, previousRedis := common.BatchUpdateEnabled, common.LogConsumeEnabled, common.RedisEnabled
	model.DB = db
	common.BatchUpdateEnabled, common.LogConsumeEnabled, common.RedisEnabled = false, false, false
	t.Cleanup(func() {
		model.DB = previousDB
		common.BatchUpdateEnabled, common.LogConsumeEnabled, common.RedisEnabled = previousBatch, previousLog, previousRedis
	})
	const initialQuota = 1000000
	require.NoError(t, db.Create(&model.User{Id: 1, Username: "image-test", Quota: initialQuota}).Error)
	require.NoError(t, db.Create(&model.Token{Id: 1, UserId: 1, Key: "image-test-key", RemainQuota: initialQuota}).Error)
	require.NoError(t, db.Create(&model.Channel{Id: 1, Name: "gateway"}).Error)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	info := relaycommon.GenRSGatewayRelayInfo(c, types.RelayFormatOpenAIImage, nil)
	info.ChannelMeta = &relaycommon.ChannelMeta{ChannelId: 1, ChannelType: constant.ChannelTypeRSGateway}
	info.UserId, info.TokenId, info.TokenKey = 1, 1, "image-test-key"
	info.UserQuota = initialQuota
	info.OriginModelName = "gpt-image-2"
	info.PriceData = hosttypes.PriceData{
		UsePrice: true, ModelPrice: 0.08,
		GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1},
	}
	// 复现信任用户实际预扣为零、上游也不返回 usage 的情况。
	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"data":[{"b64_json":"image"}]}`)),
	}
	usage, apiErr := rsGatewayImageResponse(c, info, response)
	require.Nil(t, apiErr)
	quota := service.PostTextConsumeQuota(c, info, usage, nil)
	wantQuota := common.QuotaFromFloat(0.08 * common.QuotaPerUnit)
	assert.Equal(t, wantQuota, quota)
	var user model.User
	var token model.Token
	var channel model.Channel
	require.NoError(t, db.First(&user, 1).Error)
	require.NoError(t, db.First(&token, 1).Error)
	require.NoError(t, db.First(&channel, 1).Error)
	assert.Equal(t, initialQuota-wantQuota, user.Quota)
	assert.Equal(t, wantQuota, user.UsedQuota)
	assert.Equal(t, initialQuota-wantQuota, token.RemainQuota)
	assert.Equal(t, int64(wantQuota), channel.UsedQuota)
}

func TestRSGatewayImagePricingMetadata(t *testing.T) {
	for _, test := range []struct {
		name      string
		body      string
		wantN     float64
		wantError bool
	}{
		{name: "默认一张", body: `{"model":"gpt-image-2"}`, wantN: 1},
		{name: "多张图片", body: `{"model":"gpt-image-2","n":3}`, wantN: 3},
		{name: "数量超过上限", body: fmt.Sprintf(`{"model":"gpt-image-2","n":%d}`, dto.MaxImageN+1), wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(test.body))
			c.Request.Header.Set("Content-Type", "application/json")
			t.Cleanup(func() { common.CleanupBodyStorage(c) })
			request, err := relayhelper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantN, request.GetTokenCountMeta().BillingRatios["n"])
			storage, err := common.GetBodyStorage(c)
			require.NoError(t, err)
			body, err := storage.Bytes()
			require.NoError(t, err)
			assert.Equal(t, test.body, string(body))
		})
	}
}

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

func TestRSGatewayUsageTrackerReadsGeminiUsageMetadata(t *testing.T) {
	tracker := newRSGatewayUsageTracker(false)
	_, err := tracker.Write([]byte(`{"usageMetadata":{"promptTokenCount":120,"candidatesTokenCount":17,"totalTokenCount":137,"cachedContentTokenCount":32}}`))
	require.NoError(t, err)

	usage := tracker.Usage()
	require.NotNil(t, usage)
	assert.Equal(t, 120, usage.PromptTokens)
	assert.Equal(t, 17, usage.CompletionTokens)
	assert.Equal(t, 137, usage.TotalTokens)
	require.NotNil(t, usage.BillingUsage)
	assert.Equal(t, dto.BillingUsageSemanticGemini, usage.BillingUsage.Semantic)
	require.NotNil(t, usage.BillingUsage.GeminiUsageMetadata)
	assert.Equal(t, 32, usage.BillingUsage.GeminiUsageMetadata.CachedContentTokenCount)
}
