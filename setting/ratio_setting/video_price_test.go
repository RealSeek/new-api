package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoPriceConfigLookup(t *testing.T) {
	original := VideoPrice2JSONString()
	t.Cleanup(func() { require.NoError(t, UpdateVideoPriceByJSONString(original)) })

	require.NoError(t, UpdateVideoPriceByJSONString(`{
		"video-test": {
			"default_price": 0.2,
			"default_duration": 5,
			"billing_step": 1,
			"minimum_duration": 1,
			"resolution_prices": {"720P": 0.41}
		}
	}`))

	price, ok := GetVideoPrice("video-test", "720p")
	require.True(t, ok)
	assert.InDelta(t, 0.41, price, 0.000001)

	price, ok = GetVideoPrice("video-test", "1080p")
	require.True(t, ok)
	assert.InDelta(t, 0.2, price, 0.000001)
}

func TestValidateVideoPriceJSONString(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "默认价格必须为正数",
			value: `{"video":{"default_price":0,"default_duration":5,"billing_step":1,"minimum_duration":1}}`,
		},
		{
			name:  "时长不能超过上限",
			value: `{"video":{"default_price":0.2,"default_duration":3601,"billing_step":1,"minimum_duration":1}}`,
		},
		{
			name:  "分辨率价格必须为正数",
			value: `{"video":{"default_price":0.2,"default_duration":5,"billing_step":1,"minimum_duration":1,"resolution_prices":{"1080p":0}}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t, ValidateVideoPriceJSONString(test.value))
		})
	}
}
