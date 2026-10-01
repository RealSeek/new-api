package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 日志里的"首字"对外统一展示上游数值（网关上报），
// 端到端实测值与网关缓冲耗时只写进 admin_info（普通用户视图会剥离 admin_info）。
func TestGenerateTextOtherInfoPrefersUpstreamFirstByte(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
		ChannelMeta:         &relaycommon.ChannelMeta{},
		StartTime:           time.Now().Add(-73 * time.Second),
		FirstResponseTime:   time.Now().Add(-16 * time.Second),
		UpstreamFirstByteMs: 29528,
	}

	other := GenerateTextOtherInfo(c, info, 0, 1, 0, 0, 0, 0, 0)

	assert.Equal(t, float64(29528), other["frt"])
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	assert.InDelta(t, 57000, adminInfo["frt_client_ms"].(float64), 2000)
	assert.InDelta(t, 57000-29528, adminInfo["frt_gateway_buffer_ms"].(float64), 2000)
	assert.InDelta(t, 73000, adminInfo["duration_ms"].(int64), 2000)
}

// 非网关通道没有上游上报值，保持原有端到端口径，且不写拆分字段。
func TestGenerateTextOtherInfoKeepsMeasuredFirstByteWithoutUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
		ChannelMeta:       &relaycommon.ChannelMeta{},
		StartTime:         time.Now().Add(-10 * time.Second),
		FirstResponseTime: time.Now().Add(-7 * time.Second),
	}

	other := GenerateTextOtherInfo(c, info, 0, 1, 0, 0, 0, 0, 0)

	assert.InDelta(t, 3000, other["frt"].(float64), 2000)
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	assert.NotContains(t, adminInfo, "frt_client_ms")
	assert.NotContains(t, adminInfo, "frt_gateway_buffer_ms")
}
