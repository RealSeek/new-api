package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCustomModelPricingPersistsAtomicallyAndRejectsInvalidPrices(t *testing.T) {
	require.NoError(t, validateOptionValue(billing_setting.TaskPricingOption, `{"video":"request"}`))
	require.Error(t, validateOptionValue(billing_setting.TaskPricingOption, `{"video":"invalid"}`))
	require.Error(t, validateOptionValue(billing_setting.TaskPricingOption, `null`))

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Option{}))
	connection, err := db.DB()
	require.NoError(t, err)
	oldDB := DB
	oldOptions := common.OptionMap
	common.OptionMap = make(map[string]string)
	savedConfig := make(map[string]string)
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error { savedConfig[key] = value; return nil }))
	savedPrices := map[string]string{
		"VideoPrice": ratio_setting.VideoPrice2JSONString(), "ImagePrice": ratio_setting.ImagePrice2JSONString(),
		"ModelPrice": ratio_setting.ModelPrice2JSONString(), "ModelRatio": ratio_setting.ModelRatio2JSONString(),
		"CompletionRatio": ratio_setting.CompletionRatio2JSONString(), "CacheRatio": ratio_setting.CacheRatio2JSONString(),
		"CreateCacheRatio": ratio_setting.CreateCacheRatio2JSONString(), "ImageRatio": ratio_setting.ImageRatio2JSONString(),
		"AudioRatio": ratio_setting.AudioRatio2JSONString(), "AudioCompletionRatio": ratio_setting.AudioCompletionRatio2JSONString(),
	}
	DB = db
	t.Cleanup(func() {
		DB = oldDB
		require.NoError(t, connection.Close())
		for key, value := range savedPrices {
			require.NoError(t, updateOptionMap(key, value))
		}
		require.NoError(t, config.GlobalConfig.LoadFromDB(savedConfig))
		common.OptionMap = oldOptions
	})
	var video, image map[string]any
	require.NoError(t, common.UnmarshalJsonStr(`{"default_price":0.125,"default_duration":5,"billing_step":1,"minimum_duration":1}`, &video))
	require.NoError(t, common.UnmarshalJsonStr(`{"1k":0.05,"2k":0.1,"4k":0.2}`, &image))
	empty := ModelPricingVersion(PricingValues{})
	require.NoError(t, UpdateModelPricing([]ModelPricingChange{
		{ModelName: "custom-video", ExpectedVersion: empty, Pricing: PricingValues{"VideoPrice": video, "billing_setting.billing_mode": "per_second", billing_setting.TaskPricingOption: billing_setting.TaskPricingUnitSecond}},
		{ModelName: "custom-image", ExpectedVersion: empty, Pricing: PricingValues{"ImagePrice": image, "ModelPrice": 0.05, billing_setting.TaskPricingOption: billing_setting.TaskPricingUnitRequest}},
	}))
	snapshot, err := GetModelPricingSnapshot([]string{"custom-video", "custom-image"})
	require.NoError(t, err)
	entries := make(map[string]ModelPricingEntry)
	for _, entry := range snapshot.Entries {
		entries[entry.ModelName] = entry
	}
	assert.Equal(t, video, entries["custom-video"].Configured["VideoPrice"])
	assert.Equal(t, image, entries["custom-image"].Configured["ImagePrice"])
	assert.Equal(t, billing_setting.TaskPricingUnitSecond, entries["custom-video"].Configured[billing_setting.TaskPricingOption])
	assert.Equal(t, billing_setting.TaskPricingUnitRequest, entries["custom-image"].Configured[billing_setting.TaskPricingOption])
	assert.Error(t, UpdateModelPricing([]ModelPricingChange{
		{ModelName: "custom-image", ExpectedVersion: entries["custom-image"].Version, Pricing: PricingValues{billing_setting.TaskPricingOption: "invalid"}},
	}))
	assert.Error(t, UpdateModelPricing([]ModelPricingChange{
		{ModelName: "custom-image", ExpectedVersion: entries["custom-image"].Version, Pricing: PricingValues{"ImagePrice": map[string]any{"1k": -1.0}}},
	}))
	unchanged, err := GetModelPricingSnapshot([]string{"custom-image"})
	require.NoError(t, err)
	assert.Equal(t, entries["custom-image"].Version, unchanged.Entries[0].Version)
	require.NoError(t, UpdateModelPricing([]ModelPricingChange{
		{ModelName: "custom-video", ExpectedVersion: entries["custom-video"].Version, Reset: true},
	}))
	snapshot, err = GetModelPricingSnapshot([]string{"custom-video", "custom-image"})
	require.NoError(t, err)
	for _, entry := range snapshot.Entries {
		if entry.ModelName == "custom-video" {
			assert.NotContains(t, entry.Configured, "VideoPrice")
			assert.NotContains(t, entry.Configured, billing_setting.TaskPricingOption)
		}
		if entry.ModelName == "custom-image" {
			assert.Equal(t, image, entry.Configured["ImagePrice"])
		}
	}
}
