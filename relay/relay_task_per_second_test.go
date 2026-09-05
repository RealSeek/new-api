package relay

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
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
			ModelPrice:     0.2,
			UsePrice:       true,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	}
	applyPerSecondBilling(info, relaycommon.TaskSubmitReq{Duration: 8, Size: "1920x1080"})

	assert.Equal(t, map[string]float64{
		"seconds":          10,
		"resolution_price": 2.5,
	}, info.PriceData.OtherRatios())
	assert.InDelta(t, 5.0, info.PriceData.ApplyOtherRatiosToFloat(info.PriceData.ModelPrice), 0.000001)
	quota, clamp := calculateTaskSubmitQuota(info, true)
	assert.Nil(t, clamp)
	assert.Equal(t, 2_500_000, quota)
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
