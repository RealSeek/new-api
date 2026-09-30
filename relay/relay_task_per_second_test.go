package relay

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPerSecondBillingUsesDurationStepAndResolutionPrice(t *testing.T) {
	original := ratio_setting.VideoPrice2JSONString()
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateVideoPriceByJSONString(original)) })
	require.NoError(t, ratio_setting.UpdateVideoPriceByJSONString(`{
		"video-test": {
			"default_price": 0.2,
			"default_duration": 5,
			"billing_step": 5,
			"minimum_duration": 5,
			"resolution_prices": {"720p": 0.3, "1080p": 0.5}
		}
	}`))

	info := &relaycommon.RelayInfo{
		OriginModelName: "video-test",
		PriceData: types.PriceData{
			ModelPrice:     0.5, // ModelPriceHelperPerCall 取 "default" 分辨率价（排序第一个：1080p）
			UsePrice:       true,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	}
	applyPerSecondBilling(info, relaycommon.TaskSubmitReq{Duration: 8, Size: "1280x720"})

	// 8s 向上取整到 billing_step=5 的 10s；720p 价 0.3 / 基准 0.5 = 0.6
	assert.Equal(t, map[string]float64{
		"seconds":          10,
		"resolution_price": 0.6,
	}, info.PriceData.OtherRatios())
	assert.InDelta(t, 3.0, info.PriceData.ApplyOtherRatiosToFloat(info.PriceData.ModelPrice), 0.000001)
	quota, clamp := calculateTaskSubmitQuota(info, true)
	assert.Nil(t, clamp)
	assert.Equal(t, 1_500_000, quota)
}

func TestApplyPerSecondBillingUsesConfiguredDefaultDuration(t *testing.T) {
	original := ratio_setting.VideoPrice2JSONString()
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateVideoPriceByJSONString(original)) })
	require.NoError(t, ratio_setting.UpdateVideoPriceByJSONString(`{
		"video-default": {
			"default_price": 0.1,
			"default_duration": 6,
			"billing_step": 1,
			"minimum_duration": 1
		}
	}`))

	info := &relaycommon.RelayInfo{OriginModelName: "video-default"}
	applyPerSecondBilling(info, relaycommon.TaskSubmitReq{})

	assert.Equal(t, map[string]float64{"seconds": 6, "resolution_price": 1}, info.PriceData.OtherRatios())
}

// default_price=0（只配分辨率价）时也必须按分辨率加价，
// 否则所有分辨率都会被当成基准分辨率的价格。
func TestApplyPerSecondBillingUsesResolutionPriceWhenDefaultPriceIsZero(t *testing.T) {
	original := ratio_setting.VideoPrice2JSONString()
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateVideoPriceByJSONString(original)) })
	require.NoError(t, ratio_setting.UpdateVideoPriceByJSONString(`{
		"video-no-default": {
			"default_price": 0,
			"default_duration": 5,
			"billing_step": 1,
			"minimum_duration": 1,
			"resolution_prices": {"480p": 0.25, "720p": 0.5}
		}
	}`))

	info := &relaycommon.RelayInfo{
		OriginModelName: "video-no-default",
		PriceData: types.PriceData{
			ModelPrice:     0.25, // GetVideoPrice(..., "default") → 480p
			UsePrice:       true,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	}
	applyPerSecondBilling(info, relaycommon.TaskSubmitReq{Duration: 5, Size: "1280x720"})

	assert.Equal(t, map[string]float64{
		"seconds":          5,
		"resolution_price": 2,
	}, info.PriceData.OtherRatios())
	assert.InDelta(t, 2.5, info.PriceData.ApplyOtherRatiosToFloat(info.PriceData.ModelPrice), 0.000001)
	quota, clamp := calculateTaskSubmitQuota(info, true)
	assert.Nil(t, clamp)
	assert.Equal(t, 1_250_000, quota)
}

func TestCalculateTaskSubmitQuotaPreservesSmallPerSecondPrice(t *testing.T) {
	info := &relaycommon.RelayInfo{PriceData: types.PriceData{
		ModelPrice:     0.000001,
		UsePrice:       true,
		Quota:          0,
		GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
	}}
	info.PriceData.AddOtherRatio("seconds", 100)

	quota, clamp := calculateTaskSubmitQuota(info, true)

	assert.Nil(t, clamp)
	assert.Equal(t, 50, quota)
}

// stubBillingAdaptor 只实现 EstimateBilling，用于验证乘数是否被采用。
type stubBillingAdaptor struct {
	channel.TaskAdaptor
	estimated map[string]float64
}

func (s stubBillingAdaptor) EstimateBilling(*gin.Context, *relaycommon.RelayInfo) map[string]float64 {
	return s.estimated
}

// 按次（固定价格）模型不得叠加适配器估算的时长/分辨率乘数：
// 网关视频模型走 Sora 适配器，EstimateBilling 会返回 seconds，叠加后一次调用被放大成按秒计费。
func TestApplyTaskOtherRatiosSkipsEstimatesForPerCallModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	info := &relaycommon.RelayInfo{
		OriginModelName: "video-per-call",
		PriceData: types.PriceData{
			ModelPrice:     1.55,
			UsePrice:       true,
			Quota:          775_000,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	}
	adaptor := stubBillingAdaptor{estimated: map[string]float64{"seconds": 15, "size": 1}}

	applyTaskOtherRatios(c, adaptor, info, false)

	assert.Empty(t, info.PriceData.OtherRatios())
	quota, clamp := calculateTaskSubmitQuota(info, false)
	assert.Nil(t, clamp)
	assert.Equal(t, 775_000, quota)
}

func TestApplyTaskOtherRatiosUsesAdaptorEstimateForRatioModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	info := &relaycommon.RelayInfo{
		OriginModelName: "video-ratio",
		PriceData: types.PriceData{
			ModelRatio:     0.1,
			Quota:          30_000,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	}
	adaptor := stubBillingAdaptor{estimated: map[string]float64{"seconds": 5, "size": 1}}

	applyTaskOtherRatios(c, adaptor, info, false)

	assert.Equal(t, map[string]float64{"seconds": 5, "size": 1}, info.PriceData.OtherRatios())
}

// 按秒计费模型必须继续按 VideoPrice（时长步长 + 分辨率价格）计费，
// 不能受适配器估算影响。
func TestApplyTaskOtherRatiosKeepsVideoPriceForPerSecondModel(t *testing.T) {
	original := ratio_setting.VideoPrice2JSONString()
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateVideoPriceByJSONString(original)) })
	require.NoError(t, ratio_setting.UpdateVideoPriceByJSONString(`{
		"video-second": {
			"default_price": 0.2,
			"default_duration": 5,
			"billing_step": 5,
			"minimum_duration": 5,
			"resolution_prices": {"480p": 0.1, "720p": 0.5}
		}
	}`))

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("task_request", relaycommon.TaskSubmitReq{Duration: 8, Size: "1280x720"})

	info := &relaycommon.RelayInfo{
		OriginModelName: "video-second",
		PriceData: types.PriceData{
			ModelPrice:     0.1, // GetVideoPrice(..., "default") → 480p
			UsePrice:       true,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	}
	adaptor := stubBillingAdaptor{estimated: map[string]float64{"seconds": 999, "size": 9}}

	applyTaskOtherRatios(c, adaptor, info, true)

	// 8s 向上取整到 billing_step=5 的 10s；720p 价格 0.5 / 基准 0.1 = 5
	assert.Equal(t, map[string]float64{"seconds": 10, "resolution_price": 5}, info.PriceData.OtherRatios())
	quota, clamp := calculateTaskSubmitQuota(info, true)
	assert.Nil(t, clamp)
	assert.Equal(t, 2_500_000, quota)
}
