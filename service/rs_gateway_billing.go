package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"
)

// RSGatewayBillingReceipt is an immutable, owner-authenticated report of the
// physical calls made for one new-api request. It carries usage, never prices.
type RSGatewayBillingReceipt struct {
	Version   int                       `json:"version"`
	RequestID string                    `json:"request_id"`
	Complete  bool                      `json:"complete"`
	Success   bool                      `json:"success"`
	Attempts  []RSGatewayBillingAttempt `json:"attempts"`
}

type RSGatewayBillingAttempt struct {
	PhysicalID    string                 `json:"physical_id"`
	Role          string                 `json:"role"`
	State         string                 `json:"state"`
	UsageObserved bool                   `json:"usage_observed"`
	Usage         *RSGatewayBillingUsage `json:"usage"`
}

type RSGatewayBillingUsage struct {
	InputTokens           int64 `json:"input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	CacheReadTokens       int64 `json:"cache_read_tokens"`
	CacheCreationTokens   int64 `json:"cache_creation_tokens"`
	CacheCreation5mTokens int64 `json:"cache_creation_5m_tokens"`
	CacheCreation1hTokens int64 `json:"cache_creation_1h_tokens"`
	InputIncludesCache    *bool `json:"input_includes_cache"`
	InputImageTokens      int64 `json:"input_image_tokens"`
	InputAudioTokens      int64 `json:"input_audio_tokens"`
	OutputImageTokens     int64 `json:"output_image_tokens"`
	OutputAudioTokens     int64 `json:"output_audio_tokens"`
}

type rsGatewayBillingContext struct {
	BaseURL        string
	KeyHash        string
	Username       string
	TokenName      string
	ModelName      string
	UsingGroup     string
	StartTime      time.Time
	IsStream       bool
	RelayFormat    types.RelayFormat
	Price          hosttypes.PriceData
	OtherRatios    map[string]float64
	Tiered         *billingexpr.BillingSnapshot
	RequestInput   billingexpr.RequestInput
	PreConsumed    int
	TokenConsumed  int
	Funding        string
	SubscriptionID int
}

// PrepareRSGatewayReceipt persists the existing reservation and user price
// before sending a supported request. The worker owns settlement after this.
func PrepareRSGatewayReceipt(c *gin.Context, info *relaycommon.RelayInfo) error {
	if info.ChannelType != constant.ChannelTypeRSGateway {
		return nil
	}
	input, err := freezeRSGatewayPriceInput(info.TieredBillingSnapshot, info.BillingRequestInput)
	if err != nil {
		return err
	}
	pricingTime := info.StartTime
	if pricingTime.IsZero() {
		pricingTime = time.Now()
	}
	input.PricingTime = &pricingTime
	snapshot := rsGatewayBillingContext{
		BaseURL: info.ChannelBaseUrl, KeyHash: fmt.Sprintf("%x", sha256.Sum256([]byte(info.ApiKey))),
		Username: c.GetString("username"), TokenName: info.TokenName,
		ModelName: info.GetBillingModelName(), UsingGroup: info.UsingGroup,
		StartTime: info.StartTime, IsStream: info.IsStream, RelayFormat: info.RelayFormat,
		Price: info.PriceData, OtherRatios: info.PriceData.OtherRatios(), Tiered: info.TieredBillingSnapshot,
		RequestInput: input, SubscriptionID: info.SubscriptionId,
	}
	quotaPerUnit := common.QuotaPerUnit
	audioPrice := operation_setting.GetGeminiInputAudioPricePerMillionTokens(snapshot.ModelName)
	snapshot.Price.QuotaPerUnit = &quotaPerUnit
	snapshot.Price.GeminiInputAudioPricePerMillion = &audioPrice
	if session, ok := info.Billing.(*BillingSession); ok {
		session.mu.Lock()
		snapshot.PreConsumed, snapshot.TokenConsumed = session.preConsumedQuota, session.tokenConsumed
		snapshot.Funding = session.funding.Source()
		session.mu.Unlock()
	}
	encoded, err := common.Marshal(snapshot)
	if err != nil {
		return err
	}
	// A process crash must not strand a reservation just because its response
	// defer never ran. Active streams report pending until the gateway seals them.
	result := model.DB.Model(&model.RSGatewaySettlement{}).Where("request_id = ? AND state = ?", info.RequestId, "pending").Updates(map[string]any{
		"context_json": string(encoded), "reconcile_after": common.GetTimestamp() + 60,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("gateway reservation is not pending")
	}
	info.RSGatewayBillingReceipt = true
	return nil
}

// QueueRSGatewayReceipt does not retain a Gin context or wait on client I/O.
func QueueRSGatewayReceipt(requestID string) {
	if err := model.DB.Model(&model.RSGatewaySettlement{}).Where("request_id = ? AND state = ?", requestID, "pending").Update("reconcile_after", common.GetTimestamp()).Error; err != nil {
		common.SysError(fmt.Sprintf("gateway billing request %s queue: %v", requestID, err))
	}
}

type rsGatewayPriceProbes struct {
	source billingexpr.RequestInput
	frozen billingexpr.RequestInput
	err    error
}

func (p *rsGatewayPriceProbes) Visit(node *ast.Node) {
	call, ok := (*node).(*ast.CallNode)
	if !ok || p.err != nil {
		return
	}
	identifier, ok := call.Callee.(*ast.IdentifierNode)
	if !ok || identifier.Value != "header" && identifier.Value != "param" {
		return
	}
	if len(call.Arguments) != 1 {
		p.err = errors.New("gateway durable pricing requires literal header/param names")
		return
	}
	literal, ok := call.Arguments[0].(*ast.StringNode)
	if !ok {
		p.err = errors.New("gateway durable pricing requires literal header/param names")
		return
	}
	key := strings.TrimSpace(literal.Value)
	if identifier.Value == "param" {
		p.frozen.FrozenParams[key] = gjson.GetBytes(p.source.Body, key).Value()
		return
	}
	lower := strings.ToLower(key)
	for _, part := range []string{"authorization", "cookie", "api-key", "apikey", "token", "secret"} {
		if strings.Contains(lower, part) {
			p.err = fmt.Errorf("gateway durable pricing cannot persist credential header %q", key)
			return
		}
	}
	for name, value := range p.source.Headers {
		if strings.EqualFold(strings.TrimSpace(name), key) {
			p.frozen.Headers[key] = value
			break
		}
	}
}

func freezeRSGatewayPriceInput(snapshot *billingexpr.BillingSnapshot, source *billingexpr.RequestInput) (billingexpr.RequestInput, error) {
	if snapshot == nil {
		return billingexpr.RequestInput{}, nil
	}
	if source == nil {
		source = &billingexpr.RequestInput{}
	}
	_, expression := billingexpr.ParseExprVersion(snapshot.ExprString)
	tree, err := parser.Parse(expression)
	if err != nil {
		return billingexpr.RequestInput{}, err
	}
	probes := rsGatewayPriceProbes{source: *source, frozen: billingexpr.RequestInput{
		Headers: map[string]string{}, FrozenParams: map[string]any{}, ImageCount: source.ImageCount, PricingTime: source.PricingTime,
	}}
	ast.Walk(&tree.Node, &probes)
	return probes.frozen, probes.err
}

var rsGatewayBillingWorker sync.Once

func StartRSGatewayBillingReconciler() {
	rsGatewayBillingWorker.Do(func() {
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := ReconcileRSGatewayBillingOnce(context.Background()); err != nil {
					common.SysError("gateway billing reconcile: " + err.Error())
				}
			}
		}()
	})
}

// ReconcileRSGatewayBillingOnce is shared by the production worker and isolated
// database tests. A lease bounds concurrent polling; the ledger transaction is
// the authority for duplicate delivery and money adjustments.
func ReconcileRSGatewayBillingOnce(ctx context.Context) error {
	now := common.GetTimestamp()
	var pending []model.RSGatewaySettlement
	if err := model.DB.Where("state = ? AND context_json <> ? AND reconcile_after <= ? AND lease_until <= ?", "pending", "", now, now).Order("reconcile_after, request_id").Limit(32).Find(&pending).Error; err != nil {
		return err
	}
	for _, record := range pending {
		now = common.GetTimestamp()
		lease := now + 60
		claimed := model.DB.Model(&model.RSGatewaySettlement{}).Where("request_id = ? AND state = ? AND lease_until <= ?", record.RequestId, "pending", now).Update("lease_until", lease)
		if claimed.Error != nil {
			return claimed.Error
		}
		if claimed.RowsAffected != 1 {
			continue
		}
		record.LeaseUntil = lease
		if err := reconcileRSGatewayReceipt(ctx, record); err != nil {
			common.SysError(fmt.Sprintf("gateway billing request %s remains pending: %v", record.RequestId, err))
			if updateErr := model.DB.Model(&model.RSGatewaySettlement{}).Where("request_id = ? AND state = ? AND lease_until = ?", record.RequestId, "pending", lease).Updates(map[string]any{
				"lease_until": int64(0), "reconcile_after": common.GetTimestamp() + 15,
			}).Error; updateErr != nil {
				return updateErr
			}
		}
	}
	return nil
}

func reconcileRSGatewayReceipt(ctx context.Context, record model.RSGatewaySettlement) error {
	var snapshot rsGatewayBillingContext
	if err := common.UnmarshalJsonStr(record.ContextJSON, &snapshot); err != nil {
		return err
	}
	channel, err := model.CacheGetChannel(record.ChannelId)
	if err != nil {
		return err
	}
	if channel.Type != constant.ChannelTypeRSGateway {
		return errors.New("original gateway channel type changed")
	}
	var key string
	keys := channel.GetKeys()
	if !channel.ChannelInfo.IsMultiKey {
		keys = []string{channel.Key}
	}
	for _, candidate := range keys {
		if fmt.Sprintf("%x", sha256.Sum256([]byte(candidate))) == snapshot.KeyHash {
			key = candidate
			break
		}
	}
	if key == "" {
		return errors.New("original gateway channel key is unavailable")
	}
	client, err := GetHttpClientWithProxySettings(channel.GetSetting().Proxy, channel.GetSetting())
	if err != nil {
		return err
	}
	// Billing credentials must never follow a redirect to a different endpoint.
	privateClient := *client
	privateClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(queryCtx, http.MethodGet, strings.TrimRight(snapshot.BaseURL, "/")+"/internal/new-api/billing/"+url.PathEscape(record.RequestId), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("X-RS-NewAPI-Billing-Version", "1")
	response, err := privateClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("gateway receipt HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return err
	}
	var receipt RSGatewayBillingReceipt
	if err := common.Unmarshal(body, &receipt); err != nil {
		return err
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: snapshot.ModelName, BillingModelName: snapshot.ModelName, StartTime: snapshot.StartTime,
		RelayFormat: snapshot.RelayFormat, PriceData: snapshot.Price, TieredBillingSnapshot: snapshot.Tiered,
		BillingRequestInput: &snapshot.RequestInput, ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeRSGateway},
	}
	info.PriceData.ReplaceOtherRatios(snapshot.OtherRatios)
	quota, charges, prompt, completion, err := priceRSGatewayReceipt(info, record.RequestId, receipt)
	if err != nil {
		return err
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(body))
	tokenKey := ""
	if record.TokenId > 0 {
		token, err := model.GetTokenByIds(record.TokenId, record.UserId)
		if err != nil {
			return err
		}
		tokenKey = token.Key
	}
	if err := model.CompleteRSGatewaySettlement(record.RequestId, model.RSGatewayCompletion{
		Quota: quota, PreConsumed: snapshot.PreConsumed, TokenConsumed: snapshot.TokenConsumed, TokenKey: tokenKey,
		Funding: snapshot.Funding, SubscriptionId: snapshot.SubscriptionID, Refunded: !receipt.Success && quota == 0,
		ReceiptJSON: string(body), ReceiptHash: hash, LeaseUntil: record.LeaseUntil,
	}); err != nil {
		return err
	}
	if !receipt.Success && quota == 0 {
		return nil
	}
	logContext := &gin.Context{Request: &http.Request{Header: make(http.Header)}}
	logContext.Set("username", snapshot.Username)
	logContext.Set(common.RequestIdKey, record.RequestId)
	other := model.NewLogOther()
	other.SetPublic("rs_gateway", true)
	other.SetPublic("gateway_billing_version", 1)
	other.SetPublic("gateway_attempts", charges)
	other.SetPublic("gateway_receipt_hash", hash)
	other.SetPublic("model_ratio", snapshot.Price.ModelRatio)
	other.SetPublic("group_ratio", snapshot.Price.GroupRatioInfo.GroupRatio)
	if snapshot.Tiered != nil {
		other.SetPublic("billing_mode", snapshot.Tiered.BillingMode)
	}
	attachQuotaSaturation(logContext, info, other)
	model.RecordConsumeLog(logContext, record.UserId, model.RecordConsumeLogParams{
		ChannelId: record.ChannelId, PromptTokens: prompt, CompletionTokens: completion,
		ModelName: snapshot.ModelName, TokenName: snapshot.TokenName, Quota: quota,
		Content: "网关完整逐路用量结算", TokenId: record.TokenId,
		UseTimeSeconds: int(time.Since(snapshot.StartTime).Seconds()), IsStream: snapshot.IsStream, Group: snapshot.UsingGroup, Other: other,
	})
	return nil
}

type rsGatewayAttemptCharge struct {
	PhysicalID    string `json:"physical_id"`
	Role          string `json:"role"`
	State         string `json:"state"`
	UsageObserved bool   `json:"usage_observed"`
	Quota         int    `json:"quota"`
	BillingUnit   string `json:"billing_unit"`
}

func priceRSGatewayReceipt(info *relaycommon.RelayInfo, requestID string, receipt RSGatewayBillingReceipt) (int, []rsGatewayAttemptCharge, int, int, error) {
	if receipt.Version != 1 || receipt.RequestID != requestID || !receipt.Complete {
		return 0, nil, 0, 0, errors.New("gateway receipt identity/version/completion mismatch")
	}
	seen := make(map[string]bool, len(receipt.Attempts))
	var charges []rsGatewayAttemptCharge
	total := decimal.Zero
	prompt, completion := int64(0), int64(0)
	fixedCharged := false
	for _, attempt := range receipt.Attempts {
		if attempt.PhysicalID == "" || seen[attempt.PhysicalID] ||
			(attempt.Role != "winner" && attempt.Role != "hedge") ||
			(attempt.State != "succeeded" && attempt.State != "failed" && attempt.State != "cancelled") {
			return 0, nil, 0, 0, errors.New("gateway receipt contains an invalid or duplicate physical attempt")
		}
		seen[attempt.PhysicalID] = true
		charge := rsGatewayAttemptCharge{PhysicalID: attempt.PhysicalID, Role: attempt.Role, State: attempt.State, UsageObserved: attempt.UsageObserved, BillingUnit: "token"}
		usage := &dto.Usage{UsageSemantic: "openai"}
		if attempt.UsageObserved {
			if attempt.Usage == nil || attempt.Usage.InputIncludesCache == nil {
				return 0, nil, 0, 0, errors.New("observed gateway usage lacks its cache semantics")
			}
			u := attempt.Usage
			for _, tokens := range []int64{u.InputTokens, u.OutputTokens, u.CacheReadTokens, u.CacheCreationTokens, u.CacheCreation5mTokens, u.CacheCreation1hTokens,
				u.InputImageTokens, u.InputAudioTokens, u.OutputImageTokens, u.OutputAudioTokens} {
				if tokens < 0 || tokens > int64(common.MaxQuota) {
					return 0, nil, 0, 0, errors.New("gateway usage token count is outside the billable range")
				}
			}
			if u.CacheCreation5mTokens+u.CacheCreation1hTokens > u.CacheCreationTokens {
				return 0, nil, 0, 0, errors.New("gateway cache-write breakdown exceeds its total")
			}
			usage.PromptTokens, usage.CompletionTokens = int(u.InputTokens), int(u.OutputTokens)
			usage.InputTokens, usage.OutputTokens = usage.PromptTokens, usage.CompletionTokens
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			usage.PromptTokensDetails = dto.InputTokenDetails{CachedTokens: int(u.CacheReadTokens), CachedCreationTokens: int(u.CacheCreationTokens), ImageTokens: int(u.InputImageTokens), AudioTokens: int(u.InputAudioTokens)}
			usage.CompletionTokenDetails = dto.OutputTokenDetails{ImageTokens: int(u.OutputImageTokens), AudioTokens: int(u.OutputAudioTokens)}
			if !*u.InputIncludesCache {
				usage.UsageSemantic = "anthropic"
				usage.ClaudeCacheCreation1hTokens = int(u.CacheCreation1hTokens)
				usage.ClaudeCacheCreation5mTokens = int(u.CacheCreationTokens - u.CacheCreation1hTokens)
			}
			prompt += u.InputTokens
			completion += u.OutputTokens
		}
		priceInfo := *info
		priceInfo.ResponsesUsageInfo = nil
		priceInfo.RelayFormat = types.RelayFormatOpenAI
		logContext := &gin.Context{}
		summary := calculateTextQuotaSummary(logContext, &priceInfo, usage)
		unit := billingexpr.BillingUnitToken
		quota := summary.Quota
		if info.TieredBillingSnapshot != nil {
			snap := info.TieredBillingSnapshot
			usedVars := billingexpr.UsedVarsByHash(snap.ExprString, snap.ExprHash)
			params := BuildTieredTokenParams(usage, usage.UsageSemantic == "anthropic", usedVars)
			result, err := billingexpr.ComputeTieredQuotaWithRequest(snap, params, *info.BillingRequestInput)
			if err != nil {
				return 0, nil, 0, 0, fmt.Errorf("gateway attempt %s price: %w", attempt.PhysicalID, err)
			}
			quota, unit = result.ActualQuotaAfterGroup, result.BillingUnit
			noteQuotaClamp(info, result.Clamp)
		} else if info.PriceData.UsePrice {
			unit = billingexpr.BillingUnitRequest
		}
		if unit == billingexpr.BillingUnitRequest {
			if !receipt.Success || attempt.Role != "winner" || fixedCharged {
				quota = 0
			} else {
				fixedCharged = true
			}
		} else if !attempt.UsageObserved {
			quota = 0
		}
		if quota < 0 {
			return 0, nil, 0, 0, fmt.Errorf("gateway attempt %s produced a negative charge", attempt.PhysicalID)
		}
		charge.Quota, charge.BillingUnit = quota, string(unit)
		charges = append(charges, charge)
		total = total.Add(decimal.NewFromInt(int64(quota)))
		noteQuotaClamp(info, priceInfo.QuotaClamp)
	}
	quota, clamp := common.QuotaFromDecimalChecked(total)
	noteQuotaClamp(info, clamp)
	return quota, charges, common.QuotaFromFloat(float64(prompt)), common.QuotaFromFloat(float64(completion)), nil
}
