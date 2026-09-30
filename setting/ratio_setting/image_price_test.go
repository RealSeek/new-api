package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImagePriceLookupByResolution(t *testing.T) {
	original := ImagePrice2JSONString()
	t.Cleanup(func() { require.NoError(t, UpdateImagePriceByJSONString(original)) })

	require.NoError(t, UpdateImagePriceByJSONString(`{"gpt-image-2":{"1k":0.05,"2k":0.1,"4k":0.2}}`))

	price, ok := GetImagePrice("gpt-image-2", "2k")
	require.True(t, ok)
	assert.Equal(t, 0.1, price)

	_, ok = GetImagePrice("gpt-image-2", "8k")
	assert.False(t, ok)
	_, ok = GetImagePrice("unconfigured-model", "1k")
	assert.False(t, ok)
}

func TestImagePriceValidationRejectsInvalidTiers(t *testing.T) {
	require.Error(t, ValidateImagePriceJSONString(`{"gpt-image-2":{"8k":0.1}}`))
	require.Error(t, ValidateImagePriceJSONString(`{"gpt-image-2":{"1k":0}}`))
	require.NoError(t, ValidateImagePriceJSONString(`{"gpt-image-2":{"1k":0.05,"2k":0.1,"4k":0.2}}`))
}
