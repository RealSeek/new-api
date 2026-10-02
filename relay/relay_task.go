package relay

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/model_alias_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type TaskSubmitResult struct {
	UpstreamTaskID string
	TaskData       []byte
	ClientResponse any
	Platform       constant.TaskPlatform
	Quota          int
	Immediate      *relaycommon.TaskInfo
	PluginState    []byte
	//PerCallPrice   types.PriceData
}

// ResolveOriginTask 处理基于已有任务的提交（remix / continuation）：
// 查找原始任务、从中提取模型名称、将渠道锁定到原始任务的渠道
// （通过 info.LockedChannel，重试时复用同一渠道并轮换 key），
// 以及提取 OtherRatios（时长、分辨率）。
// 该函数在控制器的重试循环之前调用一次，其结果通过 info 字段和上下文持久化。
func ResolveOriginTask(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	// 检测 remix action
	path := c.Request.URL.Path
	if strings.Contains(path, "/v1/videos/") && strings.HasSuffix(path, "/remix") {
		info.Action = constant.TaskActionRemix
	}

	// 提取 remix 任务的 video_id
	if info.Action == constant.TaskActionRemix {
		videoID := c.Param("video_id")
		if strings.TrimSpace(videoID) == "" {
			return service.TaskErrorWrapperLocal(fmt.Errorf("video_id is required"), "invalid_request", http.StatusBadRequest)
		}
		info.OriginTaskID = videoID
	}

	if info.OriginTaskID == "" {
		return nil
	}

	// 查找原始任务
	originTask, exist, err := model.GetByTaskId(info.UserId, info.OriginTaskID)
	if err != nil {
		return service.TaskErrorWrapper(err, "get_origin_task_failed", http.StatusInternalServerError)
	}
	if !exist {
		return service.TaskErrorWrapperLocal(errors.New("task_origin_not_exist"), "task_not_exist", http.StatusBadRequest)
	}

	// 从原始任务推导模型名称
	if info.OriginModelName == "" {
		if originTask.Properties.OriginModelName != "" {
			info.OriginModelName = originTask.Properties.OriginModelName
		} else if originTask.Properties.UpstreamModelName != "" {
			info.OriginModelName = originTask.Properties.UpstreamModelName
		} else {
			var taskData map[string]any
			_ = common.Unmarshal(originTask.Data, &taskData)
			if m, ok := taskData["model"].(string); ok && m != "" {
				info.OriginModelName = m
			}
		}
	}

	// 锁定到原始任务的渠道（重试时复用同一渠道，轮换 key）
	ch, err := model.GetChannelById(originTask.ChannelId, true)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "channel_not_found", http.StatusBadRequest)
	}
	if ch.Status != common.ChannelStatusEnabled {
		return service.TaskErrorWrapperLocal(errors.New("the channel of the origin task is disabled"), "task_channel_disable", http.StatusBadRequest)
	}
	info.LockedChannel = ch

	if originTask.ChannelId != info.ChannelId {
		key, _, newAPIError := ch.GetNextEnabledKey()
		if newAPIError != nil {
			return service.TaskErrorWrapper(newAPIError, "channel_no_available_key", newAPIError.StatusCode)
		}
		common.SetContextKey(c, constant.ContextKeyChannelKey, key)
		common.SetContextKey(c, constant.ContextKeyChannelType, ch.Type)
		common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, ch.GetBaseURL())
		common.SetContextKey(c, constant.ContextKeyChannelId, originTask.ChannelId)

		info.ChannelBaseUrl = ch.GetBaseURL()
		info.ChannelId = originTask.ChannelId
		info.ChannelType = ch.Type
		info.ApiKey = key
	}

	// 提取 remix 参数（时长、分辨率 → OtherRatios）
	if info.Action == constant.TaskActionRemix {
		if bc := originTask.PrivateData.BillingContext; bc != nil {
			// 新的 remix 逻辑：直接从原始任务的 BillingContext 中提取 OtherRatios（如果存在）。
			// 按次计费的任务不复制乘数：重新提交同样只收一次固定价格。
			if !bc.PerCallBilling {
				for s, f := range bc.OtherRatios {
					info.PriceData.AddOtherRatio(s, f)
				}
			}
		} else {
			// 旧的 remix 逻辑：直接从 task data 解析 seconds 和 size（如果存在）
			var taskData map[string]any
			_ = common.Unmarshal(originTask.Data, &taskData)
			secondsStr, _ := taskData["seconds"].(string)
			seconds, _ := strconv.Atoi(secondsStr)
			if seconds <= 0 {
				seconds = 4
			}
			// 历史任务数据可能包含未经校验的时长，作为计费乘数前必须钳制
			if seconds > relaycommon.MaxTaskDurationSeconds {
				seconds = relaycommon.MaxTaskDurationSeconds
			}
			sizeStr, _ := taskData["size"].(string)
			info.PriceData.AddOtherRatio("seconds", float64(seconds))
			info.PriceData.AddOtherRatio("size", 1)
			if sizeStr == "1792x1024" || sizeStr == "1024x1792" {
				info.PriceData.AddOtherRatio("size", 1.666667)
			}
		}
	}

	return nil
}

// ApplyChannelPin copies plugin-declared origin-task facts from the prepare
// context onto RelayInfo and, when the resolved pin retries on the same
// channel, writes LockedChannel. ResolveOriginTask is unchanged.
func ApplyChannelPin(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if info == nil {
		return nil
	}
	if info.TaskRelayInfo == nil {
		info.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	}
	if tasks, ok := common.GetContextKeyType[[]*model.Task](c, constant.ContextKeyOriginTasks); ok {
		refs := make([]relaycommon.OriginTaskRef, 0, len(tasks))
		for _, task := range tasks {
			if task == nil {
				continue
			}
			refs = append(refs, relaycommon.OriginTaskRef{
				TaskID:         task.TaskID,
				UpstreamTaskID: task.GetUpstreamTaskID(),
				Action:         task.Action,
				Status:         string(task.Status),
				Data:           append([]byte(nil), task.Data...),
			})
		}
		info.OriginTasks = refs
	}
	pin, found, _ := service.GetChannelConstraints(c).ResolvedPin()
	if !found || pin.RetryMode != dto.PinRetrySameChannel {
		return nil
	}
	ch, err := model.CacheGetChannel(pin.ChannelId)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "origin_task_channel_disabled", http.StatusBadRequest)
	}
	if ch.Status != common.ChannelStatusEnabled {
		return service.TaskErrorWrapperLocal(errors.New("the channel of the origin task is disabled"), "origin_task_channel_disabled", http.StatusBadRequest)
	}
	info.LockedChannel = ch
	return nil
}

// ApplyOriginTaskAffinity is the compatibility name for ApplyChannelPin.
func ApplyOriginTaskAffinity(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return ApplyChannelPin(c, info)
}

// RelayTaskSubmit 完成 task 提交的全部流程（每次尝试调用一次）：
// 刷新渠道元数据 → 确定 platform/adaptor → 验证请求 →
// 估算计费(EstimateBilling) → 计算价格 → 预扣费（仅首次）→
// 构建/发送/解析上游请求 → 提交后计费调整(AdjustBillingOnSubmit)。
// 共享控制器编排负责未落库退款、最终额度预留、落库和结算。
func RelayTaskSubmit(c *gin.Context, info *relaycommon.RelayInfo) (*TaskSubmitResult, *dto.TaskError) {
	info.InitChannelMeta(c)

	// 1. 确定 platform → 创建适配器 → 验证请求
	platform := constant.TaskPlatform(c.GetString("platform"))
	if platform == "" {
		platform = GetTaskPlatform(c)
	}
	platform, adaptor := getTaskAdaptorForRequest(c, platform)
	if adaptor == nil {
		code, message := TaskPlatformUnavailableError(platform)
		return nil, service.TaskErrorWrapperLocal(errors.New(message), code, http.StatusBadRequest)
	}
	// buildSubmitRequest runs during validation and the unreleased plugin
	// contract exposes this host-generated id to that hook.
	if info.PublicTaskID == "" {
		info.PublicTaskID = model.GenerateTaskID()
	}
	adaptor.Init(info)
	// Plugin submit hooks run during ValidateRequestAndSetAction and cache the
	// upstream body. OriginModelName is already seeded on that line (protocol
	// resolved_task_model, legacy submit, or GenRelayInfo original_model), so
	// map before validation. The empty-name CoverTaskActionToModelName
	// synthesis happens after validate and cannot move; skip the late block
	// when early mapping ran so a chain is never applied twice.
	mappedBeforeValidate := info.OriginModelName != ""
	if mappedBeforeValidate {
		info.UpstreamModelName = info.OriginModelName
		if err := helper.ModelMappedHelper(c, info, nil); err != nil {
			return nil, service.TaskErrorWrapperLocal(err, "model_mapping_failed", http.StatusBadRequest)
		}
	}
	pricingModelName := info.OriginModelName
	if alias, ok := model_alias_setting.GetResolutionAlias(info.OriginModelName); ok {
		explicit, fallback := taskResolutionCandidates(c)
		variant, _, aliasErr := model_alias_setting.ResolveAliasVariant(alias, explicit, fallback)
		if aliasErr != nil {
			return nil, service.TaskErrorWrapperLocal(aliasErr, "invalid_resolution", http.StatusBadRequest)
		}
		pricingModelName = variant
		info.UpstreamModelName = variant
	}
	if taskErr := adaptor.ValidateRequestAndSetAction(c, info); taskErr != nil {
		return nil, taskErr
	}

	// 2. 确定模型名称
	modelName := info.OriginModelName
	if modelName == "" {
		modelName = service.CoverTaskActionToModelName(platform, info.Action)
	}

	if !mappedBeforeValidate {
		info.OriginModelName = modelName
		info.UpstreamModelName = modelName
		if err := helper.ModelMappedHelper(c, info, nil); err != nil {
			return nil, service.TaskErrorWrapperLocal(err, "model_mapping_failed", http.StatusBadRequest)
		}
	}

	// 4. 价格计算：基础模型价格
	if pricingModelName == "" {
		pricingModelName = modelName
	}
	info.OriginModelName = pricingModelName
	var priceData types.PriceData
	var err error
	pluginKey := c.GetString("task_plugin_key")
	pinnedValue, _ := c.Get(jsplugin.ContextKeyPinnedPlugin)
	pinnedPlugin, _ := pinnedValue.(jsplugin.PinnedPlugin)
	if pinnedPlugin.Plugin != nil {
		pluginKey = pinnedPlugin.Plugin.Meta.Key
	}
	exprStr, exists := billing_setting.ResolveTaskBillingExpr(pluginKey, pricingModelName, info.UpstreamModelName)
	// Freeze the configured per-second tariff as a usage expression for Seedance.
	// Automatic duration reserves the model ceiling; completion overlays the
	// actual output seconds without multiplying by reference-video duration.
	if !exists && info.ChannelType == constant.ChannelTypeRSGateway && relaycommon.IsSeedance2VideoModel(info.UpstreamModelName) &&
		billing_setting.GetBillingMode(pricingModelName) == billing_setting.BillingModePerSecond {
		req, requestErr := relaycommon.GetTaskRequest(c)
		if requestErr != nil {
			return nil, service.TaskErrorWrapperLocal(requestErr, "invalid_request", http.StatusBadRequest)
		}
		resolution := relaycommon.NormalizeVideoResolution(req.Size)
		if req.Resolution != "" {
			resolution = relaycommon.NormalizeVideoResolution(req.Resolution)
		}
		if value, ok := req.Metadata["resolution"].(string); ok && value != "" {
			resolution = relaycommon.NormalizeVideoResolution(value)
		}
		if resolution == "" {
			resolution = model_alias_setting.DefaultAliasResolution
		}
		price, priced := ratio_setting.GetVideoPrice(pricingModelName, resolution)
		config, configured := ratio_setting.GetVideoPriceConfig(pricingModelName)
		if !priced || !configured {
			return nil, service.TaskErrorWrapperLocal(fmt.Errorf("Seedance %s has no price for %s", pricingModelName, resolution), "model_price_error", http.StatusBadRequest)
		}
		exprStr = fmt.Sprintf("tier(\"video\", ceil(max(u(\"seconds\"), %d) / %d) * %d * %s)", config.MinimumDuration, config.BillingStep, config.BillingStep, strconv.FormatFloat(price, 'g', -1, 64))
		exists = true
	}
	useTiered := exists || billing_setting.GetBillingMode(pricingModelName) == billing_setting.BillingModeTieredExpr
	if useTiered {
		provider, supported := adaptor.(channel.TaskUsageFactsProvider)
		if billingexpr.UsesFixedPricing(exprStr) {
			return nil, service.TaskErrorWrapper(fmt.Errorf("fixed pricing is not supported for task usage expressions"), "model_price_error", http.StatusBadRequest)
		}
		if !exists || !supported {
			return nil, service.TaskErrorWrapper(fmt.Errorf("task model %s has no usage expression or meter", modelName), "model_price_error", http.StatusBadRequest)
		}
		sharedModel := pinnedPlugin.Generation.SharedModel(modelName) || pinnedPlugin.Generation.SharedModel(info.UpstreamModelName)
		if sharedModel && pinnedPlugin.Plugin != nil {
			schema, _ := pinnedPlugin.Plugin.Meta.UsageForModels(info.UpstreamModelName, modelName)
			if !billing_setting.TaskExprCompatible(exprStr, schema) {
				return nil, service.TaskErrorWrapper(fmt.Errorf("task model %s pricing is not configured for plugin %s", modelName, pluginKey), "model_price_error", http.StatusBadRequest)
			}
		}
		var facts map[string]any
		if validatedProvider, ok := adaptor.(channel.TaskValidatedUsageFactsProvider); ok {
			facts, err = validatedProvider.ExtractUsageFactsValidated(c, info)
			if err != nil {
				return nil, service.TaskErrorWrapperLocal(err, "plugin_usage_invalid", http.StatusBadRequest)
			}
		} else {
			facts = provider.ExtractUsageFacts(c, info)
		}
		cost, trace, runErr := billingexpr.RunExprWithRequest(exprStr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
		if runErr != nil || cost < 0 {
			if runErr == nil {
				runErr = fmt.Errorf("negative task expression result")
			}
			return nil, service.TaskErrorWrapper(runErr, "model_price_error", http.StatusBadRequest)
		}
		groupRatioInfo := helper.HandleGroupRatio(c, info)
		quota, clamp := common.QuotaRoundChecked(cost * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
		noteTaskQuotaClamp(info, clamp)
		priceData = types.PriceData{Quota: quota, QuotaToPreConsume: quota, GroupRatioInfo: groupRatioInfo}
		info.TieredBillingSnapshot = &billingexpr.BillingSnapshot{BillingMode: billing_setting.BillingModeTieredExpr, ModelName: modelName, ExprString: exprStr, ExprHash: billingexpr.ExprHashString(exprStr), GroupRatio: groupRatioInfo.GroupRatio, EstimatedQuotaBeforeGroup: cost * common.QuotaPerUnit, EstimatedQuotaAfterGroup: quota, EstimatedTier: trace.MatchedTier, QuotaPerUnit: common.QuotaPerUnit, ExprVersion: billingexpr.ExprVersion(exprStr), TaskUsageBilling: true, UsageFacts: facts}
	} else {
		priceData, err = helper.ModelPriceHelperPerCall(c, info)
		if err != nil {
			return nil, service.TaskErrorWrapper(err, "model_price_error", http.StatusBadRequest)
		}
	}
	info.PriceData = priceData

	// 5. 计费估算：汇总计费乘数（时长、分辨率等）
	//    必须在 ModelPriceHelperPerCall 之后调用（它会重建 PriceData）。
	//    ResolveOriginTask 可能已在 remix 路径中预设了 OtherRatios，此处合并。
	isPerSecond := billing_setting.GetBillingMode(info.OriginModelName) == billing_setting.BillingModePerSecond
	if info.TieredBillingSnapshot == nil {
		if isPerSecond {
			req, reqErr := relaycommon.GetTaskRequest(c)
			if reqErr != nil {
				return nil, service.TaskErrorWrapperLocal(reqErr, "invalid_request", http.StatusBadRequest)
			}
			applyPerSecondBilling(info, req)
		} else if !info.PriceData.UsePrice {
			var ratios map[string]float64
			if provider, ok := adaptor.(channel.TaskValidatedBillingProvider); ok {
				ratios, err = provider.EstimateBillingValidated(c, info)
				if err != nil {
					return nil, service.TaskErrorWrapperLocal(err, "plugin_usage_invalid", http.StatusBadRequest)
				}
			} else {
				ratios = adaptor.EstimateBilling(c, info)
			}
			for k, v := range ratios {
				info.PriceData.AddOtherRatio(k, v)
			}
		}
	}
	if info.TieredBillingSnapshot == nil && (isPerSecond || !common.StringsContains(constant.TaskPricePatches, modelName)) {
		quota, clamp := calculateTaskSubmitQuota(info, isPerSecond)
		info.PriceData.Quota = quota
		noteTaskQuotaClamp(info, clamp)
	}

	// 计价结束：对外（日志、任务、账单上下文）恢复展示合并名，转发仍用变体名。
	info.OriginModelName = modelName

	// 7. 预扣费（仅首次 — 重试时 info.Billing 已存在，跳过）
	if info.Billing == nil && (!info.PriceData.FreeModel || info.ChannelType == constant.ChannelTypeRSGateway) {
		if info.ChannelType == constant.ChannelTypeRSGateway {
			tokenId := info.TokenId
			if info.IsPlayground {
				tokenId = 0
			}
			if err := model.BeginRSGatewaySettlement(info.RequestId, info.UserId, tokenId, info.ChannelId); err != nil {
				return nil, service.TaskErrorWrapperLocal(err, "gateway_settlement_unavailable", http.StatusServiceUnavailable)
			}
		}
		info.ForcePreConsume = true
		if apiErr := service.PreConsumeBilling(c, info.PriceData.Quota, info); apiErr != nil {
			if info.ChannelType == constant.ChannelTypeRSGateway {
				if err := model.CompleteRSGatewaySettlement(info.RequestId, model.RSGatewayCompletion{Refunded: true}); err != nil {
					logger.LogError(c, fmt.Sprintf("gateway task refund pending, request_id=%s: %v", info.RequestId, err))
				}
			}
			return nil, service.TaskErrorFromAPIError(apiErr)
		}
	}

	// 8. 构建请求体
	requestBody, err := adaptor.BuildRequestBody(c, info)
	if err != nil {
		return nil, service.TaskErrorWrapper(err, "build_request_failed", http.StatusInternalServerError)
	}

	// 9. 发送请求
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return nil, service.TaskErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}
	if resp == nil {
		return nil, service.TaskErrorWrapperLocal(errors.New("upstream returned an empty response"), "fail_to_fetch_task", http.StatusBadGateway)
	}
	defer resp.Body.Close()
	// Any 2xx is a successful submission: task APIs commonly answer 201 Created
	// or 202 Accepted, and parseSubmitResponse receives the exact status code.
	if resp.StatusCode/100 != 2 {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, service.TaskErrorWrapper(fmt.Errorf("%s", string(responseBody)), "fail_to_fetch_task", resp.StatusCode)
	}

	// 10. Parse only. The controller presents the response after the durable
	// task barrier and billing settlement.
	parsed, taskErr := adaptor.ParseResponse(c, resp, info)
	if taskErr != nil {
		return nil, taskErr
	}
	if parsed == nil {
		return nil, service.TaskErrorWrapperLocal(errors.New("task adaptor returned an empty response"), "plugin_submit_response_invalid", http.StatusBadGateway)
	}

	// 11. 提交后计费调整：让适配器根据上游实际返回调整 OtherRatios
	finalQuota := info.PriceData.Quota
	if parsed.Immediate != nil && parsed.Immediate.Status == model.TaskStatusFailure {
		finalQuota = 0
	} else if snap := info.TieredBillingSnapshot; snap != nil {
		if parsed.Immediate != nil && parsed.Immediate.Status == model.TaskStatusSuccess && len(parsed.Immediate.UsageFacts) > 0 {
			settlement, facts, err := service.EvaluateTaskCompletionUsage(snap, parsed.Immediate.UsageFacts)
			if err != nil {
				logger.LogWarn(c, fmt.Sprintf("task immediate usage settlement failed; retaining reserved quota: %v", err))
			} else {
				finalQuota = settlement.ActualQuotaAfterGroup
				snap.UsageFacts = facts
				snap.EstimatedTier = settlement.MatchedTier
				noteTaskQuotaClamp(info, settlement.Clamp)
			}
		}
	} else {
		if adjustedRatios := adaptor.AdjustBillingOnSubmit(info, parsed.TaskData); len(adjustedRatios) > 0 {
			if adjustedQuota, ok := recalcQuotaFromRatios(info, adjustedRatios); ok {
				// 基于调整后的 ratios 重新计算 quota
				finalQuota = adjustedQuota
				info.PriceData.ReplaceOtherRatios(adjustedRatios)
				info.PriceData.Quota = finalQuota
			}
		}
	}

	info.PriceData.Quota = finalQuota

	return &TaskSubmitResult{
		UpstreamTaskID: parsed.UpstreamTaskID,
		TaskData:       parsed.TaskData,
		ClientResponse: parsed.ClientResponse,
		Platform:       platform,
		Quota:          finalQuota,
		Immediate:      parsed.Immediate,
		PluginState:    parsed.PluginState,
	}, nil
}

// taskResolutionCandidates 汇总任务请求里可能携带分辨率的字段。
// explicit 是文档约定的 resolution 字段（与 metadata.resolution）；
// fallback 是 size 归一化结果，仅在能命中配置的档位时才使用。
func taskResolutionCandidates(c *gin.Context) (explicit []string, fallback []string) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		if strings.Contains(c.GetHeader("Content-Type"), "multipart/form-data") {
			form, parseErr := common.ParseMultipartFormReusable(c)
			if parseErr != nil {
				return nil, nil
			}
			defer form.RemoveAll()
			if values := form.Value["resolution"]; len(values) > 0 {
				req.Resolution = values[0]
			}
			if values := form.Value["size"]; len(values) > 0 {
				req.Size = values[0]
			}
			if values := form.Value["metadata"]; len(values) > 0 {
				_ = common.UnmarshalJsonStr(values[0], &req.Metadata)
			}
		} else {
			var resolutionRequest struct {
				Resolution string         `json:"resolution"`
				Size       string         `json:"size"`
				Metadata   map[string]any `json:"metadata"`
			}
			if common.UnmarshalBodyReusable(c, &resolutionRequest) != nil {
				return nil, nil
			}
			req.Resolution, req.Size, req.Metadata = resolutionRequest.Resolution, resolutionRequest.Size, resolutionRequest.Metadata
		}
	}
	if value := strings.TrimSpace(req.Resolution); value != "" {
		explicit = append(explicit, value)
	}
	if value, ok := req.Metadata["resolution"].(string); ok && strings.TrimSpace(value) != "" {
		explicit = append(explicit, strings.TrimSpace(value))
	}
	if value := strings.TrimSpace(req.Size); value != "" {
		fallback = append(fallback, relaycommon.NormalizeVideoResolution(value))
	}
	return explicit, fallback
}

func applyPerSecondBilling(info *relaycommon.RelayInfo, req relaycommon.TaskSubmitReq) {
	config, ok := ratio_setting.GetVideoPriceConfig(info.OriginModelName)
	if !ok {
		return
	}
	seconds := relaycommon.ResolveTaskDuration(req, config.DefaultDuration)
	seconds = max(seconds, config.MinimumDuration)
	seconds = ((seconds + config.BillingStep - 1) / config.BillingStep) * config.BillingStep
	seconds = min(seconds, relaycommon.MaxTaskDurationSeconds)
	info.PriceData.AddOtherRatio("seconds", float64(seconds))

	resolution := relaycommon.NormalizeVideoResolution(req.Size)
	if req.Resolution != "" {
		resolution = relaycommon.NormalizeVideoResolution(req.Resolution)
	}
	if raw, ok := req.Metadata["resolution"].(string); ok && raw != "" {
		resolution = relaycommon.NormalizeVideoResolution(raw)
	}
	// 基准价与 ModelPriceHelperPerCall 的取法一致（"default" 分辨率价），
	// 这样 resolution_price 比例 × 基准价 恰好等于该分辨率的实际单价；
	// default_price=0（只配分辨率价）时也不会丢失分辨率加价。
	basePrice, _ := ratio_setting.GetVideoPrice(info.OriginModelName, "default")
	resolutionPrice, hasResolutionPrice := ratio_setting.GetVideoPrice(info.OriginModelName, resolution)
	if hasResolutionPrice && basePrice > 0 {
		info.PriceData.AddOtherRatio("resolution_price", resolutionPrice/basePrice)
	}
}

// applyTaskOtherRatios 计算任务提交时的计费乘数。
// 按秒计费模型使用 VideoPrice 配置；按次计费（固定价格）模型一次调用只收一次
// 固定价格，不叠加适配器估算的时长/分辨率乘数；其余模型沿用适配器估算。
func applyTaskOtherRatios(c *gin.Context, adaptor channel.TaskAdaptor, info *relaycommon.RelayInfo, isPerSecond bool) {
	if isPerSecond {
		if req, reqErr := relaycommon.GetTaskRequest(c); reqErr == nil {
			applyPerSecondBilling(info, req)
		}
		return
	}
	if info.PriceData.UsePrice {
		return
	}
	for k, v := range adaptor.EstimateBilling(c, info) {
		info.PriceData.AddOtherRatio(k, v)
	}
}

func calculateTaskSubmitQuota(info *relaycommon.RelayInfo, preserveUnitPricePrecision bool) (int, *common.QuotaClamp) {
	baseQuota := float64(info.PriceData.Quota)
	if preserveUnitPricePrecision && info.PriceData.UsePrice {
		baseQuota = info.PriceData.ModelPrice * common.QuotaPerUnit * info.PriceData.GroupRatioInfo.GroupRatio
	}
	return common.QuotaFromFloatChecked(info.PriceData.ApplyOtherRatiosToFloat(baseQuota))
}

// recalcQuotaFromRatios 根据 adjustedRatios 重新计算 quota。
// 公式: baseQuota × ∏(ratio) — 其中 baseQuota 是不含 OtherRatios 的基础额度。
func recalcQuotaFromRatios(info *relaycommon.RelayInfo, ratios map[string]float64) (int, bool) {
	// 从 PriceData 获取不含 OtherRatios 的基础价格
	baseQuota := info.PriceData.RemoveOtherRatiosFromFloat(float64(info.PriceData.Quota))
	priceData := info.PriceData
	if !priceData.ReplaceOtherRatios(ratios) {
		return 0, false
	}
	// 应用新的 ratios
	result := priceData.ApplyOtherRatiosToFloat(baseQuota)
	quota, clamp := common.QuotaFromFloatChecked(result)
	noteTaskQuotaClamp(info, clamp)
	return quota, true
}

// noteTaskQuotaClamp records the first quota saturation event onto the task's
// RelayInfo so LogTaskConsumption can surface it on the submit log's
// admin_info. First non-nil clamp wins.
func noteTaskQuotaClamp(info *relaycommon.RelayInfo, clamp *common.QuotaClamp) {
	if clamp == nil || info == nil {
		return
	}
	if info.QuotaClamp == nil {
		info.QuotaClamp = clamp
	}
}

var fetchRespBuilders = map[int]func(c *gin.Context) (respBody []byte, taskResp *dto.TaskError){
	relayconstant.RelayModeVideoFetchByID: videoFetchByIDRespBodyBuilder,
}

func RelayTaskFetch(c *gin.Context, relayMode int) (taskResp *dto.TaskError) {
	respBuilder, ok := fetchRespBuilders[relayMode]
	if !ok {
		taskResp = service.TaskErrorWrapperLocal(errors.New("invalid_relay_mode"), "invalid_relay_mode", http.StatusBadRequest)
	}

	respBody, taskErr := respBuilder(c)
	if taskErr != nil {
		return taskErr
	}
	if len(respBody) == 0 {
		respBody = []byte("{\"code\":\"success\",\"data\":null}")
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	_, err := io.Copy(c.Writer, bytes.NewBuffer(respBody))
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
		return
	}
	return
}

func videoFetchByIDRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	taskId := c.Param("task_id")
	if taskId == "" {
		taskId = c.GetString("task_id")
	}
	userId := c.GetInt("id")

	originTask, exist, err := model.GetByTaskId(userId, taskId)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "get_task_failed", http.StatusInternalServerError)
		return
	}
	if !exist || !originTask.ResultRetrievable() {
		taskResp = service.TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusBadRequest)
		return
	}

	isOpenAIVideoAPI := strings.HasPrefix(c.Request.RequestURI, "/v1/videos/")

	// Gemini/Vertex 支持实时查询：用户 fetch 时直接从上游拉取最新状态
	if realtimeResp := tryRealtimeFetch(originTask, isOpenAIVideoAPI); len(realtimeResp) > 0 {
		respBody = realtimeResp
		return
	}

	// OpenAI Video API 格式: 走各 adaptor 的 ConvertToOpenAIVideo
	if isOpenAIVideoAPI {
		adaptor := GetTaskAdaptor(originTask.Platform)
		if adaptor == nil {
			taskResp = service.TaskErrorWrapperLocal(fmt.Errorf("invalid channel id: %d", originTask.ChannelId), "invalid_channel_id", http.StatusBadRequest)
			return
		}
		if converter, ok := adaptor.(channel.OpenAIVideoConverter); ok {
			openAIVideoData, err := converter.ConvertToOpenAIVideo(originTask)
			if err != nil {
				taskResp = service.TaskErrorWrapper(err, "convert_to_openai_video_failed", http.StatusInternalServerError)
				return
			}
			respBody = openAIVideoData
			return
		}
		taskResp = service.TaskErrorWrapperLocal(fmt.Errorf("not_implemented:%s", originTask.Platform), "not_implemented", http.StatusNotImplemented)
		return
	}

	// 通用 TaskDto 格式
	respBody, err = common.Marshal(dto.TaskResponse[any]{
		Code: "success",
		Data: TaskModel2Dto(originTask),
	})
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
	}
	return
}

// tryRealtimeFetch 尝试从上游实时拉取 Gemini/Vertex 任务状态。
// 仅当渠道类型为 Gemini 或 Vertex 时触发；其他渠道或出错时返回 nil。
// 当非 OpenAI Video API 时，还会构建自定义格式的响应体。
func tryRealtimeFetch(task *model.Task, isOpenAIVideoAPI bool) []byte {
	channelModel, err := model.GetChannelById(task.ChannelId, true)
	if err != nil {
		return nil
	}
	if channelModel.Type != constant.ChannelTypeVertexAi && channelModel.Type != constant.ChannelTypeGemini {
		return nil
	}

	baseURL := constant.GetChannelBaseURL(channelModel.Type)
	if channelModel.GetBaseURL() != "" {
		baseURL = channelModel.GetBaseURL()
	}
	proxy := channelModel.GetSetting().Proxy
	adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(channelModel.Type)))
	if adaptor == nil {
		return nil
	}

	resp, err := adaptor.FetchTask(baseURL, channelModel.Key, task, proxy)
	if err != nil || resp == nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	ti, err := adaptor.ParseTaskResult(task, resp, body)
	if err != nil || ti == nil {
		return nil
	}

	snap := task.Snapshot()

	// 将上游最新状态更新到 task
	if ti.Status != "" {
		task.Status = model.TaskStatus(ti.Status)
	}
	if ti.Progress != "" {
		task.Progress = ti.Progress
	}
	if strings.HasPrefix(ti.Url, "data:") {
		// data: URI — kept in Data, not ResultURL
	} else if ti.Url != "" {
		task.PrivateData.ResultURL = ti.Url
	} else if task.Status == model.TaskStatusSuccess {
		// No URL from adaptor — construct proxy URL using public task ID
		task.PrivateData.ResultURL = taskcommon.BuildProxyURL(task.TaskID)
	}

	if !snap.Equal(task.Snapshot()) {
		_, _ = task.UpdateWithStatus(snap.Status)
	}

	// OpenAI Video API 由调用者的 ConvertToOpenAIVideo 分支处理
	if isOpenAIVideoAPI {
		return nil
	}

	// 非 OpenAI Video API: 构建自定义格式响应
	format := detectVideoFormat(body)
	out := map[string]any{
		"error":    nil,
		"format":   format,
		"metadata": nil,
		"status":   mapTaskStatusToSimple(task.Status),
		"task_id":  task.TaskID,
		"url":      task.GetResultURL(),
	}
	respBody, _ := common.Marshal(dto.TaskResponse[any]{
		Code: "success",
		Data: out,
	})
	return respBody
}

// detectVideoFormat 从 Gemini/Vertex 原始响应中探测视频格式
func detectVideoFormat(rawBody []byte) string {
	var raw map[string]any
	if err := common.Unmarshal(rawBody, &raw); err != nil {
		return "mp4"
	}
	respObj, ok := raw["response"].(map[string]any)
	if !ok {
		return "mp4"
	}
	vids, ok := respObj["videos"].([]any)
	if !ok || len(vids) == 0 {
		return "mp4"
	}
	v0, ok := vids[0].(map[string]any)
	if !ok {
		return "mp4"
	}
	mt, ok := v0["mimeType"].(string)
	if !ok || mt == "" || strings.Contains(mt, "mp4") {
		return "mp4"
	}
	return mt
}

// mapTaskStatusToSimple 将内部 TaskStatus 映射为简化状态字符串
func mapTaskStatusToSimple(status model.TaskStatus) string {
	switch status {
	case model.TaskStatusSuccess:
		return "succeeded"
	case model.TaskStatusFailure:
		return "failed"
	case model.TaskStatusQueued, model.TaskStatusSubmitted:
		return "queued"
	default:
		return "processing"
	}
}

func TaskModel2Dto(task *model.Task) *dto.TaskDto {
	return &dto.TaskDto{
		ID:         task.ID,
		CreatedAt:  task.CreatedAt,
		UpdatedAt:  task.UpdatedAt,
		TaskID:     task.TaskID,
		Platform:   string(task.Platform),
		UserId:     task.UserId,
		Group:      task.Group,
		ChannelId:  task.ChannelId,
		Quota:      task.Quota,
		Action:     constant.NormalizeTaskAction(task.Action),
		Status:     string(task.Status),
		FailReason: task.FailReason,
		ResultURL:  task.GetResultURL(),
		SubmitTime: task.SubmitTime,
		StartTime:  task.StartTime,
		FinishTime: task.FinishTime,
		Progress:   task.Progress,
		Properties: task.Properties,
		Username:   task.Username,
		Data:       task.Data,
	}
}
