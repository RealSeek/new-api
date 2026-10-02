package model

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/model_alias_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
)

type PricingPluginVariant struct {
	PluginKey            string                               `json:"plugin_key"`
	PluginName           string                               `json:"plugin_name"`
	Icon                 string                               `json:"icon,omitempty"`
	BillingExpr          string                               `json:"billing_expr"`
	BillingMode          string                               `json:"billing_mode"`
	BillingUsageSchema   map[string]jsplugin.UsageFieldSchema `json:"billing_usage_schema"`
	BillingUsageExamples []jsplugin.UsageExample              `json:"billing_usage_examples,omitempty"`
}

type Pricing struct {
	ChannelVideoCapabilities []PricingChannelVideoCapabilities    `json:"channel_video_capabilities,omitempty"`
	BillingPluginVariants    []PricingPluginVariant               `json:"billing_plugin_variants,omitempty"`
	ModelName                string                               `json:"model_name"`
	Description              string                               `json:"description,omitempty"`
	Icon                     string                               `json:"icon,omitempty"`
	Tags                     string                               `json:"tags,omitempty"`
	VendorID                 int                                  `json:"vendor_id,omitempty"`
	QuotaType                int                                  `json:"quota_type"`
	ModelRatio               float64                              `json:"model_ratio"`
	ModelPrice               float64                              `json:"model_price"`
	OwnerBy                  string                               `json:"owner_by"`
	CompletionRatio          float64                              `json:"completion_ratio"`
	CacheRatio               *float64                             `json:"cache_ratio,omitempty"`
	CreateCacheRatio         *float64                             `json:"create_cache_ratio,omitempty"`
	ImageRatio               *float64                             `json:"image_ratio,omitempty"`
	AudioRatio               *float64                             `json:"audio_ratio,omitempty"`
	AudioCompletionRatio     *float64                             `json:"audio_completion_ratio,omitempty"`
	EnableGroup              []string                             `json:"enable_groups"`
	SupportedEndpointTypes   []constant.EndpointType              `json:"supported_endpoint_types"`
	BillingMode              string                               `json:"billing_mode,omitempty"`
	BillingExpr              string                               `json:"billing_expr,omitempty"`
	BillingUsageSchema       map[string]jsplugin.UsageFieldSchema `json:"billing_usage_schema,omitempty"`
	BillingUsageExamples     []jsplugin.UsageExample              `json:"billing_usage_examples,omitempty"`
	TaskPricingUnit          string                               `json:"task_pricing_unit,omitempty"`
	PricingVersion           string                               `json:"pricing_version,omitempty"`
	VideoPrice               *ratio_setting.VideoPriceConfig      `json:"video_price,omitempty"`
	ImagePrice               ratio_setting.ImagePriceConfig       `json:"image_price,omitempty"`
	ResolutionAliasPrices    []ResolutionAliasPrice               `json:"resolution_alias_prices,omitempty"`
}

type PricingChannelVideoCapabilities struct {
	ChannelID    int                         `json:"channel_id"`
	ChannelName  string                      `json:"channel_name"`
	Groups       []string                    `json:"groups"`
	Capabilities *dto.VideoModelCapabilities `json:"capabilities"`
}

type ResolutionAliasPrice struct {
	Resolution string  `json:"resolution"`
	Price      float64 `json:"price"`
	Unit       string  `json:"unit"`
}

type PricingVendor struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

var (
	pricingMap           []Pricing
	vendorsList          []PricingVendor
	supportedEndpointMap map[string]common.EndpointInfo
	lastGetPricingTime   time.Time
	updatePricingLock    sync.Mutex

	// 缓存映射：模型名 -> 启用分组 / 计费类型
	modelEnableGroups     = make(map[string][]string)
	modelQuotaTypeMap     = make(map[string]int)
	modelEnableGroupsLock = sync.RWMutex{}
)

var (
	modelSupportEndpointTypes = make(map[string][]constant.EndpointType)
	modelSupportEndpointsLock = sync.RWMutex{}
)

func GetPricing() []Pricing {
	if time.Since(lastGetPricingTime) > time.Minute*1 || len(pricingMap) == 0 {
		updatePricingLock.Lock()
		defer updatePricingLock.Unlock()
		// Double check after acquiring the lock
		if time.Since(lastGetPricingTime) > time.Minute*1 || len(pricingMap) == 0 {
			modelSupportEndpointsLock.Lock()
			defer modelSupportEndpointsLock.Unlock()
			updatePricing()
		}
	}
	return pricingMap
}

func InvalidatePricingCache() {
	updatePricingLock.Lock()
	defer updatePricingLock.Unlock()

	pricingMap = nil
	vendorsList = nil
	lastGetPricingTime = time.Time{}
}

// GetVendors 返回当前定价接口使用到的供应商信息
func GetVendors() []PricingVendor {
	if time.Since(lastGetPricingTime) > time.Minute*1 || len(pricingMap) == 0 {
		// 保证先刷新一次
		GetPricing()
	}
	return vendorsList
}

func GetModelSupportEndpointTypes(model string) []constant.EndpointType {
	if model == "" {
		return make([]constant.EndpointType, 0)
	}
	modelSupportEndpointsLock.RLock()
	defer modelSupportEndpointsLock.RUnlock()
	if endpoints, ok := modelSupportEndpointTypes[model]; ok {
		return endpoints
	}
	return make([]constant.EndpointType, 0)
}

func getPricingEndpointTypesForAbility(ability AbilityWithChannel, advancedCustomConfigs map[int]*dto.AdvancedCustomConfig) []constant.EndpointType {
	if ability.ChannelType == constant.ChannelTypeRSGateway {
		var channel Channel
		if err := DB.First(&channel, ability.ChannelId).Error; err == nil {
			configured := channel.GetOtherSettings().SupportedEndpointTypes
			if len(configured) > 0 {
				endpoints := make([]constant.EndpointType, 0, len(configured))
				for _, endpoint := range configured {
					if _, ok := common.GetDefaultEndpointInfo(constant.EndpointType(endpoint)); ok {
						endpoints = append(endpoints, constant.EndpointType(endpoint))
					}
				}
				if len(endpoints) > 0 {
					return endpoints
				}
			}
		}
	}
	if ability.ChannelType != constant.ChannelTypeAdvancedCustom {
		return common.GetEndpointTypesByChannelType(ability.ChannelType, ability.Model)
	}
	if config := advancedCustomConfigs[ability.ChannelId]; config != nil {
		return config.SupportedEndpointTypesForModel(ability.Model)
	}
	return common.GetEndpointTypesByChannelType(ability.ChannelType, ability.Model)
}

// loadPricingAdvancedCustomConfigs runs inside updatePricing while
// updatePricingLock is held, and nests channelSyncLock.RLock. This defines the
// global lock order updatePricingLock -> channelSyncLock: any code path holding
// channelSyncLock must release it before touching the pricing cache (see
// InitChannelCache / CacheUpdateChannel), otherwise it deadlocks.
// The returned configs are pointers shared with the channel cache; they are
// replaced wholesale on update and never mutated in place, so reading them after
// RUnlock is safe.
func loadPricingAdvancedCustomConfigs(enableAbilities []AbilityWithChannel) map[int]*dto.AdvancedCustomConfig {
	channelIDs := make([]int, 0)
	seen := make(map[int]struct{})
	for _, ability := range enableAbilities {
		if ability.ChannelType != constant.ChannelTypeAdvancedCustom {
			continue
		}
		if _, exists := seen[ability.ChannelId]; exists {
			continue
		}
		seen[ability.ChannelId] = struct{}{}
		channelIDs = append(channelIDs, ability.ChannelId)
	}
	if len(channelIDs) == 0 {
		return nil
	}

	configs := make(map[int]*dto.AdvancedCustomConfig, len(channelIDs))
	if common.MemoryCacheEnabled {
		channelSyncLock.RLock()
		defer channelSyncLock.RUnlock()
		for _, channelID := range channelIDs {
			if config := channel2advancedCustomConfig[channelID]; config != nil {
				configs[channelID] = config
			}
		}
		return configs
	}

	for _, channelID := range channelIDs {
		channel, err := CacheGetChannel(channelID)
		if err != nil {
			common.SysLog(fmt.Sprintf("load advanced custom channel settings error: channel_id=%d, error=%v", channelID, err))
			continue
		}
		if channel.Type != constant.ChannelTypeAdvancedCustom {
			continue
		}
		if config := channel.GetOtherSettings().AdvancedCustom; config != nil {
			configs[channelID] = config
		}
	}
	return configs
}

func appendPricingEndpoint(endpoints []string, endpoint string) []string {
	if endpoint == "" || common.StringsContains(endpoints, endpoint) {
		return endpoints
	}
	return append(endpoints, endpoint)
}

// buildResolutionAliasPrices 汇总合并模型各分辨率变体的展示单价。
// 变体的计费方式不一致时以第一档为准（单位取首个有价档位）。
func buildResolutionAliasPrices(modelName string, alias model_alias_setting.ResolutionAlias) []ResolutionAliasPrice {
	rows := make([]ResolutionAliasPrice, 0, len(alias.Resolutions))
	for _, resolution := range alias.ResolutionLabels() {
		variant := alias.Resolutions[resolution]
		pricingModel := model_alias_setting.ResolutionPricingModel(modelName, resolution, variant)
		var price float64
		var unit string
		var ok bool
		if pricingModel == modelName && billing_setting.GetBillingMode(modelName) == billing_setting.BillingModePerSecond {
			price, ok = ratio_setting.GetVideoPrice(modelName, resolution)
			unit = "second"
		} else if pricingModel == variant {
			price, unit, ok = resolutionVariantPrice(variant)
		}
		if !ok {
			continue
		}
		rows = append(rows, ResolutionAliasPrice{Resolution: resolution, Price: price, Unit: unit})
	}
	return rows
}

// resolutionVariantPrice 返回某个变体模型的展示单价与单位（second/request）。
func resolutionVariantPrice(variant string) (float64, string, bool) {
	if billing_setting.GetBillingMode(variant) == billing_setting.BillingModePerSecond {
		if price, ok := ratio_setting.GetVideoPrice(variant, "default"); ok {
			return price, "second", true
		}
		return 0, "", false
	}
	if price, ok := ratio_setting.GetModelPrice(variant, false); ok {
		return price, "request", true
	}
	return 0, "", false
}

func updatePricing() {
	//modelRatios := common.GetModelRatios()
	enableAbilities, err := GetAllEnableAbilityWithChannels()
	if err != nil {
		common.SysLog(fmt.Sprintf("GetAllEnableAbilityWithChannels error: %v", err))
		return
	}
	// 预加载模型元数据与供应商一次，避免循环查询
	var allMeta []Model
	_ = DB.Find(&allMeta).Error
	names := make([]string, 0, len(enableAbilities))
	for _, ability := range enableAbilities {
		names = append(names, ability.Model)
	}
	metaMap := resolveModelMetadata(allMeta, names)

	// 预加载供应商
	var vendors []Vendor
	_ = DB.Find(&vendors).Error
	vendorMap := make(map[int]*Vendor)
	for i := range vendors {
		vendorMap[vendors[i].Id] = &vendors[i]
	}

	// 初始化默认供应商映射
	initDefaultVendorMapping(metaMap, vendorMap, enableAbilities)

	// 构建对前端友好的供应商列表
	vendorsList = make([]PricingVendor, 0, len(vendorMap))
	for _, v := range vendorMap {
		vendorsList = append(vendorsList, PricingVendor{
			ID:          v.Id,
			Name:        v.Name,
			Description: v.Description,
			Icon:        v.Icon,
		})
	}

	modelGroupsMap := make(map[string]*types.Set[string])

	for _, ability := range enableAbilities {
		groups, ok := modelGroupsMap[ability.Model]
		if !ok {
			groups = types.NewSet[string]()
			modelGroupsMap[ability.Model] = groups
		}
		groups.Add(ability.Group)
	}

	//这里使用切片而不是Set，因为一个模型可能支持多个端点类型，并且第一个端点是优先使用端点
	modelSupportEndpointsStr := make(map[string][]string)
	advancedCustomConfigs := loadPricingAdvancedCustomConfigs(enableAbilities)
	videoAbilityChannels := make(map[string]map[int]bool)

	// 先根据已有能力填充原生端点
	for _, ability := range enableAbilities {
		endpoints := modelSupportEndpointsStr[ability.Model]
		channelTypes := getPricingEndpointTypesForAbility(ability, advancedCustomConfigs)
		if slices.Contains(channelTypes, constant.EndpointTypeOpenAIVideo) {
			if videoAbilityChannels[ability.Model] == nil {
				videoAbilityChannels[ability.Model] = make(map[int]bool)
			}
			videoAbilityChannels[ability.Model][ability.ChannelId] = true
		}
		for _, channelType := range channelTypes {
			if !common.StringsContains(endpoints, string(channelType)) {
				endpoints = append(endpoints, string(channelType))
			}
		}
		modelSupportEndpointsStr[ability.Model] = endpoints
	}

	// 再补充模型自定义端点：若配置有效则追加到已有推断，不再裁剪渠道真实能力
	for modelName, meta := range metaMap {
		if strings.TrimSpace(meta.Endpoints) == "" {
			continue
		}
		var raw map[string]any
		if err := common.Unmarshal([]byte(meta.Endpoints), &raw); err == nil {
			endpoints := modelSupportEndpointsStr[modelName]
			for k, v := range raw {
				switch v.(type) {
				case string, map[string]any:
					endpoints = appendPricingEndpoint(endpoints, k)
				}
			}
			if len(endpoints) > 0 {
				modelSupportEndpointsStr[modelName] = endpoints
			}
		}
	}

	modelSupportEndpointTypes = make(map[string][]constant.EndpointType)
	for model, endpoints := range modelSupportEndpointsStr {
		supportedEndpoints := make([]constant.EndpointType, 0)
		for _, endpointStr := range endpoints {
			endpointType := constant.EndpointType(endpointStr)
			supportedEndpoints = append(supportedEndpoints, endpointType)
		}
		modelSupportEndpointTypes[model] = supportedEndpoints
	}

	videoChannels := make(map[int]*Channel)
	videoSettings := make(map[int]dto.ChannelOtherSettings)
	modelVideoCapabilities := make(map[string]map[int]PricingChannelVideoCapabilities)
	for _, ability := range enableAbilities {
		if !videoAbilityChannels[ability.Model][ability.ChannelId] {
			continue
		}
		channel, loaded := videoChannels[ability.ChannelId]
		if !loaded {
			channel, err = CacheGetChannel(ability.ChannelId)
			if err != nil {
				common.SysLog(fmt.Sprintf("pricing video capabilities: channel_id=%d, error=%v", ability.ChannelId, err))
				continue
			}
			videoChannels[channel.Id] = channel
			var settings dto.ChannelOtherSettings
			if channel.OtherSettings != "" {
				if err := common.UnmarshalJsonStr(channel.OtherSettings, &settings); err != nil {
					common.SysLog(fmt.Sprintf("pricing video capabilities: channel_id=%d, invalid settings: %v", channel.Id, err))
				}
			}
			videoSettings[channel.Id] = settings
		}
		if channel.Status != common.ChannelStatusEnabled {
			continue
		}
		entries := modelVideoCapabilities[ability.Model]
		if entries == nil {
			entries = make(map[int]PricingChannelVideoCapabilities)
			modelVideoCapabilities[ability.Model] = entries
		}
		entry, exists := entries[channel.Id]
		if !exists {
			entry = PricingChannelVideoCapabilities{ChannelID: channel.Id, ChannelName: channel.Name}
			if capabilities, configured := videoSettings[channel.Id].VideoModelCapabilities[ability.Model]; configured {
				entry.Capabilities = &capabilities
			}
		}
		if !slices.Contains(entry.Groups, ability.Group) {
			entry.Groups = append(entry.Groups, ability.Group)
			slices.Sort(entry.Groups)
		}
		entries[channel.Id] = entry
	}

	// 构建全局 supportedEndpointMap（默认 + 自定义覆盖）
	supportedEndpointMap = make(map[string]common.EndpointInfo)
	// 1. 默认端点
	for _, endpoints := range modelSupportEndpointTypes {
		for _, et := range endpoints {
			if info, ok := common.GetDefaultEndpointInfo(et); ok {
				if _, exists := supportedEndpointMap[string(et)]; !exists {
					supportedEndpointMap[string(et)] = info
				}
			}
		}
	}
	// 2. 自定义端点（models 表）覆盖默认
	for _, meta := range metaMap {
		if strings.TrimSpace(meta.Endpoints) == "" {
			continue
		}
		var raw map[string]any
		if err := common.Unmarshal([]byte(meta.Endpoints), &raw); err == nil {
			for k, v := range raw {
				switch val := v.(type) {
				case string:
					supportedEndpointMap[k] = common.EndpointInfo{Path: val, Method: "POST"}
				case map[string]any:
					ep := common.EndpointInfo{Method: "POST"}
					if p, ok := val["path"].(string); ok {
						ep.Path = p
					}
					if m, ok := val["method"].(string); ok {
						ep.Method = strings.ToUpper(m)
					}
					supportedEndpointMap[k] = ep
				default:
					// ignore unsupported types
				}
			}
		}
	}

	pricingMap = make([]Pricing, 0)
	pluginGeneration := jsplugin.DefaultRegistry.Generation()
	for model, groups := range modelGroupsMap {
		pricing := Pricing{
			ModelName:              model,
			EnableGroup:            groups.Items(),
			SupportedEndpointTypes: modelSupportEndpointTypes[model],
		}

		for _, capabilities := range modelVideoCapabilities[model] {
			pricing.ChannelVideoCapabilities = append(pricing.ChannelVideoCapabilities, capabilities)
		}
		slices.SortFunc(pricing.ChannelVideoCapabilities, func(a, b PricingChannelVideoCapabilities) int {
			return a.ChannelID - b.ChannelID
		})

		// 补充模型元数据（描述、标签、供应商、状态）
		if meta, ok := metaMap[model]; ok {
			// 若模型被禁用(status!=1)，则直接跳过，不返回给前端
			if meta.Status != 1 {
				continue
			}
			pricing.Description = meta.Description
			pricing.Icon = meta.Icon
			pricing.Tags = meta.Tags
			pricing.VendorID = meta.VendorID
		}
		modelPrice, findPrice := ratio_setting.GetModelPrice(model, false)
		if findPrice {
			pricing.ModelPrice = modelPrice
			pricing.QuotaType = 1
		} else {
			modelRatio, _, _ := ratio_setting.GetModelRatio(model)
			pricing.ModelRatio = modelRatio
			pricing.CompletionRatio = ratio_setting.GetCompletionRatio(model)
			pricing.QuotaType = 0
		}
		if cacheRatio, ok := ratio_setting.GetCacheRatio(model); ok {
			pricing.CacheRatio = &cacheRatio
		}
		if createCacheRatio, ok := ratio_setting.GetCreateCacheRatio(model); ok {
			pricing.CreateCacheRatio = &createCacheRatio
		}
		if imageRatio, ok := ratio_setting.GetImageRatio(model); ok {
			pricing.ImageRatio = &imageRatio
		}
		if ratio_setting.ContainsAudioRatio(model) {
			audioRatio := ratio_setting.GetAudioRatio(model)
			pricing.AudioRatio = &audioRatio
		}
		if ratio_setting.ContainsAudioCompletionRatio(model) {
			audioCompletionRatio := ratio_setting.GetAudioCompletionRatio(model)
			pricing.AudioCompletionRatio = &audioCompletionRatio
		}
		if billingMode := billing_setting.GetBillingMode(model); billingMode == "tiered_expr" {
			if expr, ok := billing_setting.GetBillingExpr(model); ok && strings.TrimSpace(expr) != "" {
				pricing.BillingMode = billingMode
				pricing.BillingExpr = expr
			}
		} else if target, resolved := ResolveTaskModelAlias(pluginGeneration, model); resolved && target.Declared != "" {
			if tailMode := billing_setting.GetBillingMode(target.Declared); tailMode == "tiered_expr" {
				if expr, ok := billing_setting.GetBillingExpr(target.Declared); ok && strings.TrimSpace(expr) != "" {
					pricing.BillingMode = tailMode
					pricing.BillingExpr = expr
				}
			}
		}
		plugin, usageModel, ok := ResolveTaskUsagePlugin(pluginGeneration, model)
		if ok && plugin != nil {
			usageSchema, usageExamples := plugin.Meta.UsageForModel(usageModel)
			pricing.BillingUsageSchema = jsplugin.CloneUsageSchema(usageSchema)
			pricing.BillingUsageExamples = jsplugin.CloneUsageExamples(usageExamples)
		}
		if unit, ok := billing_setting.GetTaskPricingUnit(model); ok {
			pricing.TaskPricingUnit = unit
		}
		providers := pluginGeneration.PluginsByModel(model)
		hasProviderOverride := false
		for _, provider := range providers {
			if _, configured := billing_setting.GetPluginBillingExpr(provider.Meta.Key, model); configured {
				hasProviderOverride = true
				break
			}
		}
		if hasProviderOverride || (len(providers) >= 2 && pricing.BillingMode == billing_setting.BillingModeTieredExpr) {
			for _, provider := range providers {
				schema, examples := provider.Meta.UsageForModel(model)
				if schema == nil {
					schema = map[string]jsplugin.UsageFieldSchema{}
				}
				expression, hasExpression := billing_setting.ResolveTaskBillingExpr(provider.Meta.Key, model, "")
				mode := billing_setting.BillingModeRatio
				if hasExpression || billing_setting.GetBillingMode(model) == billing_setting.BillingModeTieredExpr {
					mode = billing_setting.BillingModeTieredExpr
				}
				if mode == billing_setting.BillingModeTieredExpr && !billing_setting.TaskExprCompatible(expression, schema) {
					expression = ""
				}
				pricing.BillingPluginVariants = append(pricing.BillingPluginVariants, PricingPluginVariant{
					PluginKey: provider.Meta.Key, PluginName: provider.Meta.Name, Icon: provider.Meta.Icon,
					BillingExpr: expression, BillingMode: mode,
					BillingUsageSchema: jsplugin.CloneUsageSchema(schema), BillingUsageExamples: jsplugin.CloneUsageExamples(examples),
				})
			}
		}
		if billing_setting.GetBillingMode(model) == billing_setting.BillingModePerSecond {
			pricing.BillingMode = billing_setting.BillingModePerSecond
			pricing.QuotaType = 2
			if videoPrice, ok := ratio_setting.GetVideoPriceConfig(model); ok {
				pricing.VideoPrice = &videoPrice
				pricing.ModelPrice = videoPrice.DefaultPrice
			}
		}
		if imagePrice, ok := ratio_setting.GetImagePriceConfig(model); ok && len(imagePrice) > 0 {
			pricing.ImagePrice = imagePrice
		}
		if alias, ok := model_alias_setting.GetResolutionAlias(model); ok {
			pricing.ResolutionAliasPrices = buildResolutionAliasPrices(model, alias)
			if len(pricing.ResolutionAliasPrices) > 0 {
				if pricing.ResolutionAliasPrices[0].Unit == "second" {
					pricing.QuotaType = 2
				} else {
					pricing.QuotaType = 1
				}
			}
		}
		pricingMap = append(pricingMap, pricing)
	}

	// 防止大更新后数据不通用
	if len(pricingMap) > 0 {
		pricingMap[0].PricingVersion = "818f58180f058c012f3c88b96e86afb5801a3251ba8f5e29bf3b3bf4b72c2435"
	}

	// 刷新缓存映射，供高并发快速查询
	modelEnableGroupsLock.Lock()
	modelEnableGroups = make(map[string][]string)
	modelQuotaTypeMap = make(map[string]int)
	for _, p := range pricingMap {
		modelEnableGroups[p.ModelName] = p.EnableGroup
		modelQuotaTypeMap[p.ModelName] = p.QuotaType
	}
	modelEnableGroupsLock.Unlock()

	lastGetPricingTime = time.Now()
}

// GetSupportedEndpointMap 返回全局端点到路径的映射
func GetSupportedEndpointMap() map[string]common.EndpointInfo {
	return supportedEndpointMap
}
