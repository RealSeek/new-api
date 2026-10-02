package model

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func resetPricingEndpointTestTables(t *testing.T) {
	t.Helper()
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &Model{}, &Vendor{}))
	for _, table := range []string{"abilities", "channels", "models", "vendors"} {
		require.NoError(t, DB.Exec("DELETE FROM "+table).Error)
	}
	InitChannelCache()
	InvalidatePricingCache()
	t.Cleanup(func() {
		for _, table := range []string{"abilities", "channels", "models", "vendors"} {
			require.NoError(t, DB.Exec("DELETE FROM "+table).Error)
		}
		InitChannelCache()
		InvalidatePricingCache()
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})
}

func insertPricingEndpointChannel(t *testing.T, channelID int, channelType int, settings dto.ChannelOtherSettings) {
	t.Helper()
	channel := &Channel{
		Id:     channelID,
		Type:   channelType,
		Key:    fmt.Sprintf("key-%d", channelID),
		Status: common.ChannelStatusEnabled,
		Name:   fmt.Sprintf("channel-%d", channelID),
	}
	if settings.AdvancedCustom != nil || len(settings.SupportedEndpointTypes) > 0 || len(settings.VideoModelCapabilities) > 0 {
		channel.SetOtherSettings(settings)
	}
	require.NoError(t, DB.Create(channel).Error)
}

func TestPricingVideoCapabilitiesByChannel(t *testing.T) {
	resetPricingEndpointTestTables(t)
	limits := dto.VideoModelCapabilities{ReferenceImages: 9, FirstLastFrames: 2, ReferenceVideos: 3}
	insertPricingEndpointChannel(t, 136, constant.ChannelTypeRSGateway, dto.ChannelOtherSettings{
		SupportedEndpointTypes: []string{string(constant.EndpointTypeOpenAIVideo)},
		VideoModelCapabilities: map[string]dto.VideoModelCapabilities{"seedance-2.0": limits},
	})
	insertPricingEndpointChannel(t, 137, constant.ChannelTypeRSGateway, dto.ChannelOtherSettings{SupportedEndpointTypes: []string{string(constant.EndpointTypeOpenAIVideo)}})
	insertPricingEndpointChannel(t, 138, constant.ChannelTypeRSGateway, dto.ChannelOtherSettings{SupportedEndpointTypes: []string{string(constant.EndpointTypeOpenAIVideo)}})
	require.NoError(t, DB.Model(&Channel{}).Where("id = ?", 138).Update("status", common.ChannelStatusManuallyDisabled).Error)
	insertPricingEndpointChannel(t, 139, constant.ChannelTypeOpenAI, dto.ChannelOtherSettings{})
	for _, id := range []int{136, 137, 138, 139} {
		insertPricingEndpointAbility(t, id, "seedance-2.0")
	}
	require.NoError(t, DB.Create(&Ability{Group: "premium", Model: "seedance-2.0", ChannelId: 136, Enabled: true}).Error)
	InitChannelCache()
	var video Pricing
	for _, pricing := range GetPricing() {
		if pricing.ModelName == "seedance-2.0" {
			video = pricing
		}
	}
	require.Len(t, video.ChannelVideoCapabilities, 2)
	assert.Equal(t, PricingChannelVideoCapabilities{ChannelID: 136, ChannelName: "channel-136", Groups: []string{"default", "premium"}, Capabilities: &limits}, video.ChannelVideoCapabilities[0])
	assert.Nil(t, video.ChannelVideoCapabilities[1].Capabilities)
	encoded, err := common.Marshal(video)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "key-136")

	var channel Channel
	require.NoError(t, DB.First(&channel, 136).Error)
	settings := channel.GetOtherSettings()
	limits.ReferenceImages = 8
	settings.VideoModelCapabilities["seedance-2.0"] = limits
	channel.SetOtherSettings(settings)
	require.NoError(t, DB.Model(&Channel{}).Where("id = ?", channel.Id).Update("settings", channel.OtherSettings).Error)
	CacheUpdateChannel(&channel)
	for _, pricing := range GetPricing() {
		if pricing.ModelName == "seedance-2.0" {
			assert.Equal(t, 8, pricing.ChannelVideoCapabilities[0].Capabilities.ReferenceImages)
		}
	}
}

func TestVideoCapabilitySettingsValidationAndPersistence(t *testing.T) {
	for _, raw := range []string{
		`{"video_model_capabilities":{"seedance-2.0":{"reference_images":-1}}}`,
		`{"video_model_capabilities":{"seedance-2.0":{"first_last_frames":3}}}`,
		`{"video_model_capabilities":{"seedance-2.0":{"reference_videos":1.5}}}`,
		`{"video_model_capabilities":{" ":{}}}`,
	} {
		channel := Channel{OtherSettings: raw}
		require.Error(t, channel.ValidateSettings())
	}
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(filepath.Join(t.TempDir(), "video.db"))
			case "mysql":
				dsn := os.Getenv("TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("TEST_MYSQL_DSN is not configured")
				}
				driver = mysql.Open(dsn)
			case "postgres":
				dsn := os.Getenv("TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN is not configured")
				}
				driver = postgres.Open(dsn)
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
			tableName := "video_capability_channels"
			require.NoError(t, db.Table(tableName).AutoMigrate(&Channel{}))
			t.Cleanup(func() { require.NoError(t, db.Migrator().DropTable(tableName)) })
			channel := Channel{Type: constant.ChannelTypeRSGateway, Key: "private", Name: "video", OtherSettings: `{"preserved_setting":"keep","video_model_capabilities":{"seedance-2.0":{"reference_images":9,"first_last_frames":2,"reference_videos":3,"reference_audios":0},"seedance-2.5":{"reference_images":4}}}`}
			require.NoError(t, channel.ValidateSettings())
			require.NoError(t, db.Table(tableName).Create(&channel).Error)
			var saved Channel
			require.NoError(t, db.Table(tableName).First(&saved, channel.Id).Error)
			assert.Equal(t, channel.OtherSettings, saved.OtherSettings)
			assert.Equal(t, dto.VideoModelCapabilities{ReferenceImages: 9, FirstLastFrames: 2, ReferenceVideos: 3}, saved.GetOtherSettings().VideoModelCapabilities["seedance-2.0"])
			assert.Equal(t, 4, saved.GetOtherSettings().VideoModelCapabilities["seedance-2.5"].ReferenceImages)
			var version string
			if dialect == "sqlite" {
				require.NoError(t, db.Raw("select sqlite_version()").Scan(&version).Error)
			} else {
				require.NoError(t, db.Raw("select version()").Scan(&version).Error)
			}
			t.Logf("%s version: %s", dialect, version)
		})
	}
}

func TestPricingRSGatewayUsesConfiguredEndpointTypes(t *testing.T) {
	resetPricingEndpointTestTables(t)
	insertPricingEndpointChannel(t, 105, constant.ChannelTypeRSGateway, dto.ChannelOtherSettings{
		SupportedEndpointTypes: []string{string(constant.EndpointTypeOpenAIVideo)},
	})
	insertPricingEndpointAbility(t, 105, "grok-imagine-video-1.5")

	byModel := pricingEndpointTypesByModel(t)
	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIVideo}, byModel["grok-imagine-video-1.5"])
}

func TestPricingRSGatewayImageEndpointUsesConfiguredAlias(t *testing.T) {
	resetPricingEndpointTestTables(t)
	insertPricingEndpointChannel(t, 106, constant.ChannelTypeRSGateway, dto.ChannelOtherSettings{
		SupportedEndpointTypes: []string{string(constant.EndpointTypeImageGeneration)},
	})
	insertPricingEndpointAbility(t, 106, "my-image-alias")
	byModel := pricingEndpointTypesByModel(t)
	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeImageGeneration}, byModel["my-image-alias"])
	endpoint, ok := common.GetDefaultEndpointInfo(constant.EndpointTypeImageGeneration)
	require.True(t, ok)
	assert.Equal(t, "/v1/images/generations", endpoint.Path)
}

func insertPricingEndpointAbility(t *testing.T, channelID int, modelName string) {
	t.Helper()
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     modelName,
		ChannelId: channelID,
		Enabled:   true,
	}).Error)
}

func pricingEndpointAdvancedCustomConfig(routes ...dto.AdvancedCustomRoute) dto.ChannelOtherSettings {
	return dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{
			Routes: routes,
		},
	}
}

func pricingEndpointTypesByModel(t *testing.T) map[string][]constant.EndpointType {
	t.Helper()
	InitChannelCache()
	return pricingEndpointTypesFromPricing(GetPricing())
}

func pricingEndpointTypesFromPricing(pricings []Pricing) map[string][]constant.EndpointType {
	byModel := make(map[string][]constant.EndpointType)
	for _, pricing := range pricings {
		byModel[pricing.ModelName] = pricing.SupportedEndpointTypes
	}
	return byModel
}

func TestPricingAdvancedCustomUsesConfiguredEndpointTypes(t *testing.T) {
	resetPricingEndpointTestTables(t)

	insertPricingEndpointChannel(t, 101, constant.ChannelTypeAdvancedCustom, pricingEndpointAdvancedCustomConfig(
		dto.AdvancedCustomRoute{
			IncomingPath: "/v1/chat/completions",
			UpstreamPath: "/v1/chat/completions",
		},
		dto.AdvancedCustomRoute{
			IncomingPath: "/v1/responses",
			UpstreamPath: "/v1beta/models/{model}:generateContent",
			Converter:    "openai_responses_to_gemini_generate_content",
			Models:       []string{"re:^gemini-"},
		},
	))
	insertPricingEndpointAbility(t, 101, "gemini-2.5-flash")
	insertPricingEndpointAbility(t, 101, "gpt-4o")

	byModel := pricingEndpointTypesByModel(t)

	assert.Equal(t, []constant.EndpointType{
		constant.EndpointTypeOpenAI,
		constant.EndpointTypeOpenAIResponse,
	}, byModel["gemini-2.5-flash"])
	assert.Equal(t, []constant.EndpointType{
		constant.EndpointTypeOpenAI,
	}, byModel["gpt-4o"])
}

func TestPricingModelMetadataEndpointsMergeWithAdvancedCustomInference(t *testing.T) {
	resetPricingEndpointTestTables(t)

	insertPricingEndpointChannel(t, 103, constant.ChannelTypeAdvancedCustom, pricingEndpointAdvancedCustomConfig(
		dto.AdvancedCustomRoute{
			IncomingPath: "/v1/responses",
			UpstreamPath: "/v1beta/models/{model}:generateContent",
			Converter:    "openai_responses_to_gemini_generate_content",
			Models:       []string{"re:^gemini-"},
		},
	))
	insertPricingEndpointAbility(t, 103, "gemini-2.5-flash")
	require.NoError(t, DB.Create(&Model{
		ModelName: "gemini-2.5-flash",
		Endpoints: `{
			"openai": "/v1/chat/completions"
		}`,
		Status:   1,
		NameRule: NameRuleExact,
	}).Error)

	byModel := pricingEndpointTypesByModel(t)

	assert.Equal(t, []constant.EndpointType{
		constant.EndpointTypeOpenAIResponse,
		constant.EndpointTypeOpenAI,
	}, byModel["gemini-2.5-flash"])
}

func TestPricingModelMetadataEndpointsCanProvideEndpointWithoutChannelInference(t *testing.T) {
	resetPricingEndpointTestTables(t)

	insertPricingEndpointChannel(t, 104, constant.ChannelTypeAdvancedCustom, pricingEndpointAdvancedCustomConfig(
		dto.AdvancedCustomRoute{
			IncomingPath: "/v1/responses",
			UpstreamPath: "/v1beta/models/{model}:generateContent",
			Converter:    "openai_responses_to_gemini_generate_content",
			Models:       []string{"re:^gemini-"},
		},
	))
	insertPricingEndpointAbility(t, 104, "metadata-only-model")
	require.NoError(t, DB.Create(&Model{
		ModelName: "metadata-only-model",
		Endpoints: `{
			"openai": "/v1/chat/completions"
		}`,
		Status:   1,
		NameRule: NameRuleExact,
	}).Error)

	byModel := pricingEndpointTypesByModel(t)

	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAI}, byModel["metadata-only-model"])
}

func TestPricingAdvancedCustomMissingConfigFallsBackToChannelType(t *testing.T) {
	resetPricingEndpointTestTables(t)

	insertPricingEndpointChannel(t, 102, constant.ChannelTypeAdvancedCustom, dto.ChannelOtherSettings{})
	insertPricingEndpointAbility(t, 102, "gpt-4o")

	byModel := pricingEndpointTypesByModel(t)

	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAI}, byModel["gpt-4o"])
}

func TestPricingNativeChannelEndpointTypesUnchanged(t *testing.T) {
	resetPricingEndpointTestTables(t)

	insertPricingEndpointChannel(t, 201, constant.ChannelTypeOpenAI, dto.ChannelOtherSettings{})
	insertPricingEndpointChannel(t, 202, constant.ChannelTypeGemini, dto.ChannelOtherSettings{})
	insertPricingEndpointChannel(t, 203, constant.ChannelTypeAnthropic, dto.ChannelOtherSettings{})
	insertPricingEndpointAbility(t, 201, "gpt-4o")
	insertPricingEndpointAbility(t, 202, "gemini-2.5-flash")
	insertPricingEndpointAbility(t, 203, "claude-3-5-sonnet")

	byModel := pricingEndpointTypesByModel(t)

	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAI}, byModel["gpt-4o"])
	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeGemini, constant.EndpointTypeOpenAI}, byModel["gemini-2.5-flash"])
	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeAnthropic, constant.EndpointTypeOpenAI}, byModel["claude-3-5-sonnet"])
}

func TestInitChannelCacheInvalidatesPricingCache(t *testing.T) {
	resetPricingEndpointTestTables(t)

	insertPricingEndpointChannel(t, 301, constant.ChannelTypeAdvancedCustom, pricingEndpointAdvancedCustomConfig(
		dto.AdvancedCustomRoute{
			IncomingPath: "/v1/chat/completions",
			UpstreamPath: "/v1/chat/completions",
		},
	))
	insertPricingEndpointAbility(t, 301, "gemini-3.5-flash")
	InitChannelCache()

	initial := pricingEndpointTypesByModel(t)
	require.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAI}, initial["gemini-3.5-flash"])

	var channel Channel
	require.NoError(t, DB.First(&channel, "id = ?", 301).Error)
	channel.SetOtherSettings(pricingEndpointAdvancedCustomConfig(
		dto.AdvancedCustomRoute{
			IncomingPath: "/v1/chat/completions",
			UpstreamPath: "/v1/chat/completions",
		},
		dto.AdvancedCustomRoute{
			IncomingPath: "/v1/responses",
			UpstreamPath: "/v1beta/models/{model}:generateContent",
			Converter:    "openai_responses_to_gemini_generate_content",
			Models:       []string{"re:^gemini-"},
		},
	))
	require.NoError(t, DB.Model(&Channel{}).Where("id = ?", 301).Update("settings", channel.OtherSettings).Error)
	InitChannelCache()

	updated := pricingEndpointTypesByModel(t)
	assert.Equal(t, []constant.EndpointType{
		constant.EndpointTypeOpenAI,
		constant.EndpointTypeOpenAIResponse,
	}, updated["gemini-3.5-flash"])
}

func TestInitChannelCacheInvalidatesStartupPricingBuiltBeforeChannelCache(t *testing.T) {
	resetPricingEndpointTestTables(t)

	insertPricingEndpointChannel(t, 302, constant.ChannelTypeAdvancedCustom, pricingEndpointAdvancedCustomConfig(
		dto.AdvancedCustomRoute{
			IncomingPath: "/v1/chat/completions",
			UpstreamPath: "/v1/chat/completions",
		},
		dto.AdvancedCustomRoute{
			IncomingPath: "/v1/responses",
			UpstreamPath: "/v1beta/models/{model}:generateContent",
			Converter:    "openai_responses_to_gemini_generate_content",
			Models:       []string{"re:^gemini-"},
		},
	))
	insertPricingEndpointAbility(t, 302, "gemini-3.5-flash")

	staleByModel := pricingEndpointTypesFromPricing(GetPricing())
	require.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAI}, staleByModel["gemini-3.5-flash"])

	InitChannelCache()

	rebuiltByModel := pricingEndpointTypesFromPricing(GetPricing())
	assert.Equal(t, []constant.EndpointType{
		constant.EndpointTypeOpenAI,
		constant.EndpointTypeOpenAIResponse,
	}, rebuiltByModel["gemini-3.5-flash"])
}

func TestCacheUpdateChannelSyncsAdvancedCustomConfig(t *testing.T) {
	resetPricingEndpointTestTables(t)

	channel := &Channel{
		Id:     401,
		Type:   constant.ChannelTypeAdvancedCustom,
		Key:    "key-401",
		Status: common.ChannelStatusEnabled,
		Name:   "channel-401",
	}
	channel.SetOtherSettings(pricingEndpointAdvancedCustomConfig(dto.AdvancedCustomRoute{
		IncomingPath: "/v1/responses",
		UpstreamPath: "/v1beta/models/{model}:generateContent",
		Converter:    "openai_responses_to_gemini_generate_content",
	}))
	CacheUpdateChannel(channel)

	require.NotNil(t, channel2advancedCustomConfig[401])
	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIResponse}, channel2advancedCustomConfig[401].SupportedEndpointTypesForModel("gemini-3.5-flash"))

	channel.SetOtherSettings(pricingEndpointAdvancedCustomConfig(dto.AdvancedCustomRoute{
		IncomingPath: "/v1/chat/completions",
		UpstreamPath: "/v1/chat/completions",
	}))
	CacheUpdateChannel(channel)

	require.NotNil(t, channel2advancedCustomConfig[401])
	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAI}, channel2advancedCustomConfig[401].SupportedEndpointTypesForModel("gemini-3.5-flash"))

	channel.Type = constant.ChannelTypeOpenAI
	CacheUpdateChannel(channel)

	assert.Nil(t, channel2advancedCustomConfig[401])
}
