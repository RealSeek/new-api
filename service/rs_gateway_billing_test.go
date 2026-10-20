package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type rsGatewaySettlementBeforeReceipts struct {
	RequestId                  string `gorm:"primaryKey;type:varchar(64)"`
	UserId, TokenId, ChannelId int
	Quota                      int64
	State                      string `gorm:"type:varchar(16);not null"`
	Funding                    string `gorm:"type:varchar(16)"`
	SubscriptionId             int
	CreatedAt, UpdatedAt       int64
}

func (rsGatewaySettlementBeforeReceipts) TableName() string { return "rs_gateway_settlements" }

func TestRSGatewayReceiptPerAttemptPricesAndRequestProbes(t *testing.T) {
	input := billingexpr.RequestInput{Headers: map[string]string{"Authorization": "Bearer must-not-persist", "X-Discount": "yes"}, Body: []byte(`{"service_tier":"priority","input":"private prompt"}`)}
	expression := `tier("base", p * 2) * (param("service_tier") == "priority" ? 2 : 1) * (header("x-discount") == "yes" ? 0.5 : 1)`
	snapshot := &billingexpr.BillingSnapshot{ExprString: expression}
	frozen, err := freezeRSGatewayPriceInput(snapshot, &input)
	require.NoError(t, err)
	encoded, err := common.Marshal(frozen)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "must-not-persist")
	assert.NotContains(t, string(encoded), "private prompt")
	before, _, err := billingexpr.RunExprWithRequest(expression, billingexpr.TokenParams{P: 80}, input)
	require.NoError(t, err)
	var reopened billingexpr.RequestInput
	require.NoError(t, common.Unmarshal(encoded, &reopened))
	after, _, err := billingexpr.RunExprWithRequest(expression, billingexpr.TokenParams{P: 80}, reopened)
	require.NoError(t, err)
	assert.Equal(t, before, after)
	for _, bad := range []string{`tier("x", p * 2) * (header("Authorization") == "x" ? 2 : 1)`, `tier("x", p * 2) * (header(param("header")) == "x" ? 2 : 1)`} {
		_, err := freezeRSGatewayPriceInput(&billingexpr.BillingSnapshot{ExprString: bad}, &input)
		require.Error(t, err)
	}
	pricingTime := time.Date(1997, time.January, 2, 4, 5, 0, 0, time.UTC)
	clockExpression := `tier("historical", p * (hour("UTC") + minute("UTC") + weekday("UTC") + month("UTC") + day("UTC")))`
	clockInput, err := freezeRSGatewayPriceInput(&billingexpr.BillingSnapshot{ExprString: clockExpression}, &billingexpr.RequestInput{PricingTime: &pricingTime})
	require.NoError(t, err)
	encoded, err = common.Marshal(clockInput)
	require.NoError(t, err)
	require.NoError(t, common.Unmarshal(encoded, &reopened))
	cost, _, err := billingexpr.RunExprWithRequest(clockExpression, billingexpr.TokenParams{P: 10}, reopened)
	require.NoError(t, err)
	assert.Equal(t, float64(160), cost, "reconciliation uses the original request time, including its timezone")

	cacheIncluded := true
	receipt := RSGatewayBillingReceipt{Version: 1, RequestID: "pricing", Complete: true, Success: true, Attempts: []RSGatewayBillingAttempt{
		{PhysicalID: "flow:1", Role: "winner", State: "succeeded", UsageObserved: true, Usage: &RSGatewayBillingUsage{InputTokens: 80, OutputTokens: 10, InputIncludesCache: &cacheIncluded}},
		{PhysicalID: "flow:2", Role: "hedge", State: "cancelled", UsageObserved: true, Usage: &RSGatewayBillingUsage{InputTokens: 80, OutputTokens: 5, InputIncludesCache: &cacheIncluded}},
		{PhysicalID: "flow:3", Role: "hedge", State: "failed"},
	}}
	for _, test := range []struct {
		name, expression string
		success          bool
		want             int
	}{
		{"each attempt keeps its own context tier", `len <= 100 ? tier("s", p * 2 + c * 4) : tier("l", p * 10 + c * 20)`, true, 190},
		{"successful request price is charged once", `tier("request", fixed(0.01))`, true, 5000},
		{"failed request does not charge a request price", `tier("request", fixed(0.01))`, false, 0},
		{"failed calls retain their observed tokens", `tier("base", p * 2 + c * 4)`, false, 190},
	} {
		t.Run(test.name, func(t *testing.T) {
			receipt.Success = test.success
			info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatOpenAI, PriceData: hosttypes.PriceData{GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}},
				TieredBillingSnapshot: &billingexpr.BillingSnapshot{BillingMode: "tiered_expr", ExprString: test.expression, ExprHash: billingexpr.ExprHashString(test.expression), QuotaPerUnit: 500000, GroupRatio: 1}, BillingRequestInput: &billingexpr.RequestInput{}}
			quota, charges, _, _, err := priceRSGatewayReceipt(info, receipt.RequestID, receipt)
			require.NoError(t, err)
			assert.Equal(t, test.want, quota)
			require.Len(t, charges, 3)
			assert.Zero(t, charges[2].Quota, "no token estimate for an unobserved loser")
		})
	}
	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })
	for _, test := range []struct {
		name       string
		audioPrice float64
		fixed      bool
		want       int
	}{
		{"legacy audio uses frozen actual price", 2, false, 180},
		{"legacy audio retains an explicit zero price", 0, false, 220},
		{"legacy fixed price retains the quota conversion", 0, true, 5000},
	} {
		t.Run(test.name, func(t *testing.T) {
			frozenUnit := float64(500000)
			info := &relaycommon.RelayInfo{OriginModelName: "gemini-2.5-flash", PriceData: hosttypes.PriceData{
				ModelRatio: 2, CompletionRatio: 3, ModelPrice: 0.01, UsePrice: test.fixed,
				GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}, QuotaPerUnit: &frozenUnit, GeminiInputAudioPricePerMillion: &test.audioPrice,
			}}
			one := RSGatewayBillingReceipt{Version: 1, RequestID: "legacy", Complete: true, Success: true, Attempts: []RSGatewayBillingAttempt{
				{PhysicalID: "legacy:1", Role: "winner", State: "succeeded", UsageObserved: true, Usage: &RSGatewayBillingUsage{InputTokens: 80, OutputTokens: 10, InputAudioTokens: 40, InputIncludesCache: &cacheIncluded}},
			}}
			quota, _, _, _, err := priceRSGatewayReceipt(info, "legacy", one)
			require.NoError(t, err)
			assert.Equal(t, test.want, quota)
		})
	}
	receipt.Attempts = append(receipt.Attempts, receipt.Attempts[0])
	_, _, _, _, err = priceRSGatewayReceipt(&relaycommon.RelayInfo{}, receipt.RequestID, receipt)
	require.Error(t, err, "a duplicate physical ID must not be billed twice")
}

func TestRSGatewayReceiptUsageAndBillingMetadata(t *testing.T) {
	included, separate := true, false
	expression := `len <= 100 ? tier("short", p * 2 + c * 4 + cr * 0.5 + cc * 3 + cc1h * 6) : tier("long", p * 10 + c * 20 + cr + cc * 6 + cc1h * 12)`
	info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatOpenAI,
		PriceData:             hosttypes.PriceData{GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{BillingMode: "tiered_expr", ExprString: expression, ExprHash: billingexpr.ExprHashString(expression), QuotaPerUnit: 1000000, GroupRatio: 1},
		BillingRequestInput:   &billingexpr.RequestInput{},
	}
	for _, test := range []struct {
		name          string
		input         int64
		includesCache *bool
	}{
		{"inclusive input", 80, &included},
		{"separate Anthropic cache", 55, &separate},
	} {
		t.Run(test.name, func(t *testing.T) {
			receipt := RSGatewayBillingReceipt{Version: 1, RequestID: "metadata", Complete: true, Success: true, Attempts: []RSGatewayBillingAttempt{
				{PhysicalID: "winner", Role: "winner", State: "succeeded", UsageObserved: true, Usage: &RSGatewayBillingUsage{
					InputTokens: test.input, OutputTokens: 10, CacheReadTokens: 20, CacheCreationTokens: 5, CacheCreation5mTokens: 5, InputIncludesCache: test.includesCache,
				}},
				{PhysicalID: "hedge", Role: "hedge", State: "cancelled", UsageObserved: true, Usage: &RSGatewayBillingUsage{
					InputTokens: 120, OutputTokens: 5, InputIncludesCache: &included,
				}},
			}}
			quota, charges, prompt, completion, err := priceRSGatewayReceipt(info, receipt.RequestID, receipt)
			require.NoError(t, err)
			assert.Equal(t, 200, prompt, "log inputs include separately reported cache exactly once")
			assert.Equal(t, 15, completion)
			assert.Equal(t, 1475, quota)
			require.Len(t, charges, 2)
			winner := charges[0].Billing.Snapshot()
			loser := charges[1].Billing.Snapshot()
			assert.Equal(t, "short", winner["matched_tier"])
			assert.Equal(t, "long", loser["matched_tier"])
			assert.Equal(t, map[string]float64{"p": 55, "c": 10, "len": 80, "cr": 20, "cc": 5, "cc1h": 0,
				"img": 0, "img_cr": 0, "img_o": 0, "ai": 0, "ao": 0}, winner["billing_tokens"])
			assert.Equal(t, test.input, charges[0].Usage.InputTokens)
			assert.Equal(t, *test.includesCache, *charges[0].Usage.InputIncludesCache)
			encoded, err := common.Marshal(charges)
			require.NoError(t, err)
			assert.Contains(t, string(encoded), `"matched_tier":"short"`)
			assert.Contains(t, string(encoded), `"matched_tier":"long"`)
			assert.Contains(t, string(encoded), `"expr_b64":`)
		})
	}
}

// The isolated databases are supplied by the runner; no server is installed or
// production DSN inferred. This exercises migrations, HTTP polling and money.
func TestRSGatewayReceiptDatabaseMatrix(t *testing.T) {
	for _, dialect := range []struct {
		name common.DatabaseType
		env  string
	}{
		{common.DatabaseTypeSQLite, ""},
		{common.DatabaseTypePostgreSQL, "TEST_POSTGRES_DSN"},
		{common.DatabaseTypeMySQL, "TEST_MYSQL_DSN"},
	} {
		t.Run(string(dialect.name), func(t *testing.T) {
			var driver gorm.Dialector = sqlite.Open(":memory:")
			if dialect.env != "" {
				dsn := os.Getenv(dialect.env)
				if dsn == "" {
					t.Skip(dialect.env + " is not configured")
				}
				if dialect.name == common.DatabaseTypeMySQL {
					driver = mysql.Open(dsn)
				} else {
					driver = postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
				}
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			oldDB, oldLogDB := model.DB, model.LOG_DB
			oldMainType, oldLogType := common.MainDatabaseType(), common.LogDatabaseType()
			oldMemory, oldRedis, oldBatch, oldLog := common.MemoryCacheEnabled, common.RedisEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled
			model.DB, model.LOG_DB = db, db
			common.SetDatabaseTypes(dialect.name, dialect.name)
			common.MemoryCacheEnabled, common.RedisEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled = false, false, false, true
			t.Cleanup(func() {
				model.DB, model.LOG_DB = oldDB, oldLogDB
				common.SetDatabaseTypes(oldMainType, oldLogType)
				common.MemoryCacheEnabled, common.RedisEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled = oldMemory, oldRedis, oldBatch, oldLog
				require.NoError(t, sqlDB.Close())
			})
			require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.Channel{}, &model.UserSubscription{}, &model.SubscriptionPreConsumeRecord{}, &model.Log{}))
			// Use the actual old schema, then run the production model migration twice.
			if !db.Migrator().HasTable(&model.RSGatewaySettlement{}) {
				require.NoError(t, db.Migrator().CreateTable(&rsGatewaySettlementBeforeReceipts{}))
			}
			legacyID := "legacy-receipt-" + common.NewRequestId()
			require.NoError(t, db.Create(&rsGatewaySettlementBeforeReceipts{RequestId: legacyID, UserId: 1, Quota: 31, State: "settled", Funding: "wallet"}).Error)
			t.Cleanup(func() {
				require.NoError(t, db.Where("request_id = ?", legacyID).Delete(&model.RSGatewaySettlement{}).Error)
			})
			require.NoError(t, db.AutoMigrate(&model.RSGatewaySettlement{}))
			require.NoError(t, db.AutoMigrate(&model.RSGatewaySettlement{}))
			var legacy model.RSGatewaySettlement
			require.NoError(t, db.First(&legacy, "request_id = ?", legacyID).Error)
			assert.EqualValues(t, 31, legacy.Quota)
			assert.Equal(t, "settled", legacy.State)
			var version string
			query := "select version()"
			if dialect.name == common.DatabaseTypeSQLite {
				query = "select sqlite_version()"
			}
			require.NoError(t, db.Raw(query).Scan(&version).Error)
			t.Logf("database: %s", version)
			for _, funding := range []string{"wallet", "subscription"} {
				t.Run(funding, func(t *testing.T) { runRSGatewayReceiptAccounting(t, db, funding) })
			}
		})
	}
}

func runRSGatewayReceiptAccounting(t *testing.T, db *gorm.DB, funding string) {
	t.Helper()
	requestID := "receipt-" + funding + "-" + common.NewRequestId()
	key := "gateway-receipt-test-key"
	var responseStatus atomic.Int64
	responseStatus.Store(http.StatusAccepted)
	cacheIncluded := true
	firstByteMs, winnerDurationMs, hedgeDurationMs := int64(250), int64(7500), int64(9200)
	receipt := RSGatewayBillingReceipt{Version: 1, RequestID: requestID, Complete: true, Success: true, Attempts: []RSGatewayBillingAttempt{
		{PhysicalID: "flow:1", Role: "winner", State: "succeeded", FirstByteMs: &firstByteMs, DurationMs: &winnerDurationMs, UsageObserved: true, Usage: &RSGatewayBillingUsage{
			InputTokens: 80, OutputTokens: 10, CacheReadTokens: 20, CacheCreationTokens: 5, CacheCreation5mTokens: 2, CacheCreation1hTokens: 3, InputIncludesCache: &cacheIncluded,
		}},
		{PhysicalID: "flow:2", Role: "hedge", State: "cancelled", DurationMs: &hedgeDurationMs, UsageObserved: true, Usage: &RSGatewayBillingUsage{InputTokens: 80, OutputTokens: 5, InputIncludesCache: &cacheIncluded}},
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer "+key, r.Header.Get("Authorization"))
		assert.Equal(t, "/internal/new-api/billing/"+requestID, r.URL.Path)
		if status := int(responseStatus.Load()); status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		body, err := common.Marshal(receipt)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	const initial, reserved, want = 10000, 20, 190
	user := model.User{Username: "receipt_" + strings.ReplaceAll(requestID, "-", ""), Quota: initial}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "t-" + requestID, Name: "receipt", RemainQuota: initial}
	require.NoError(t, db.Create(&token).Error)
	channel := model.Channel{Type: constant.ChannelTypeRSGateway, Key: key, Name: "receipt", BaseURL: &server.URL}
	require.NoError(t, db.Create(&channel).Error)
	sub := model.UserSubscription{UserId: user.Id, AmountTotal: initial, AmountUsed: reserved}
	if funding == "subscription" {
		require.NoError(t, db.Create(&sub).Error)
	}
	t.Cleanup(func() {
		require.NoError(t, db.Where("request_id = ?", requestID).Delete(&model.Log{}).Error)
		require.NoError(t, db.Where("request_id = ?", requestID).Delete(&model.RSGatewaySettlement{}).Error)
		require.NoError(t, db.Unscoped().Delete(&token).Error)
		require.NoError(t, db.Unscoped().Delete(&user).Error)
		require.NoError(t, db.Unscoped().Delete(&channel).Error)
		if sub.Id != 0 {
			require.NoError(t, db.Delete(&sub).Error)
		}
	})
	if funding == "wallet" {
		ok, err := model.TryReserveUserQuota(user.Id, reserved, true)
		require.NoError(t, err)
		require.True(t, ok)
	}
	ok, err := model.TryReserveTokenQuota(token.Id, token.Key, reserved, false, true)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, model.BeginRSGatewaySettlement(requestID, user.Id, token.Id, channel.Id))
	expression := `len <= 100 ? tier("s", p * 2 + c * 4) : tier("l", p * 10 + c * 20)`
	info := &relaycommon.RelayInfo{RequestId: requestID, UserId: user.Id, TokenId: token.Id, TokenName: token.Name, OriginModelName: "receipt-model", StartTime: time.Now().Add(-time.Hour), RelayFormat: types.RelayFormatOpenAI, IsStream: true,
		RequestURLPath:        "/v1/chat/completions?private=value",
		ChannelMeta:           &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeRSGateway, ChannelId: channel.Id, ChannelBaseUrl: server.URL, ApiKey: key},
		PriceData:             hosttypes.PriceData{GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{BillingMode: "tiered_expr", ExprString: expression, ExprHash: billingexpr.ExprHashString(expression), QuotaPerUnit: 500000, GroupRatio: 1},
		SubscriptionId:        sub.Id,
	}
	var source FundingSource = &WalletFunding{userId: user.Id, immediate: true}
	if funding == "subscription" {
		source = &SubscriptionFunding{userId: user.Id, subscriptionId: sub.Id}
	}
	info.Billing = &BillingSession{relayInfo: info, funding: source, preConsumedQuota: reserved, tokenConsumed: reserved}
	c := &gin.Context{}
	c.Set("username", user.Username)
	if funding == "subscription" {
		info.RelayFormat = types.RelayFormatClaude
		info.RequestURLPath = "/v1/messages?beta=true"
	}
	info.InitRequestConversionChain()
	require.NoError(t, PrepareRSGatewayReceipt(c, info))
	QueueRSGatewayReceipt(requestID)
	require.NoError(t, ReconcileRSGatewayBillingOnce(context.Background()))
	var ledger model.RSGatewaySettlement
	require.NoError(t, db.First(&ledger, "request_id = ?", requestID).Error)
	assert.Equal(t, "pending", ledger.State)
	assert.NotEmpty(t, ledger.ContextJSON)
	assert.NotContains(t, ledger.ContextJSON, key)
	assert.NotContains(t, ledger.ContextJSON, "private=value")
	assert.NotContains(t, ledger.ContextJSON, "beta=true")
	assert.Empty(t, ledger.ReceiptHash)
	responseStatus.Store(http.StatusNotFound)
	QueueRSGatewayReceipt(requestID)
	require.NoError(t, ReconcileRSGatewayBillingOnce(context.Background()))
	require.NoError(t, db.First(&ledger, "request_id = ?", requestID).Error)
	assert.Equal(t, "pending", ledger.State, "404 cannot erase a bill or refund a reservation")
	// Only persisted context remains. Changing live pricing cannot change this bill.
	info.Billing, info.TieredBillingSnapshot = nil, nil
	responseStatus.Store(http.StatusOK)
	QueueRSGatewayReceipt(requestID)
	require.NoError(t, ReconcileRSGatewayBillingOnce(context.Background()))
	require.NoError(t, ReconcileRSGatewayBillingOnce(context.Background()))
	require.NoError(t, db.First(&ledger, "request_id = ?", requestID).Error)
	assert.Equal(t, "settled", ledger.State)
	assert.EqualValues(t, want, ledger.Quota)
	assert.NotEmpty(t, ledger.ReceiptHash)
	var logs []model.Log
	require.NoError(t, db.Where("request_id = ?", requestID).Find(&logs).Error)
	require.Len(t, logs, 1, "repeated receipt reconciliation records one consume log")
	assert.Equal(t, 160, logs[0].PromptTokens)
	assert.Equal(t, 15, logs[0].CompletionTokens)
	assert.Equal(t, 7, logs[0].UseTime, "receipt polling delay and hedge completion do not inflate model duration")
	assert.True(t, logs[0].IsStream)
	assert.Empty(t, logs[0].Content, "settlement diagnostics are not model response content")
	var other map[string]any
	require.NoError(t, common.UnmarshalJsonStr(logs[0].Other, &other))
	if funding == "subscription" {
		assert.Equal(t, "/v1/messages", other["request_path"])
		assert.Equal(t, []any{"Claude Messages"}, other["request_conversion"])
	} else {
		assert.Equal(t, "/v1/chat/completions", other["request_path"])
		assert.Equal(t, []any{"OpenAI Compatible"}, other["request_conversion"])
	}
	assert.Equal(t, float64(250), other["frt"])
	admin, ok := other["admin_info"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(7500), admin["duration_ms"])
	assert.Equal(t, float64(20), other["cache_tokens"])
	assert.Equal(t, float64(5), other["cache_creation_tokens"])
	assert.Equal(t, float64(2), other["cache_creation_tokens_5m"])
	assert.Equal(t, float64(3), other["cache_creation_tokens_1h"])
	assert.Equal(t, float64(160), other["input_tokens_total"])
	assert.Equal(t, "s", other["matched_tier"], "combined input must not select the long-context tier")
	assert.Equal(t, "flow:1", other["gateway_pricing_physical_id"])
	assert.NotEmpty(t, other["expr_b64"])
	billingTokens, ok := other["billing_tokens"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(80), billingTokens["len"], "the displayed pricing keeps the winner's actual context length")
	require.NoError(t, db.First(&user, user.Id).Error)
	require.NoError(t, db.First(&token, token.Id).Error)
	require.NoError(t, db.First(&channel, channel.Id).Error)
	assert.Equal(t, initial-want, token.RemainQuota)
	assert.Equal(t, want, token.UsedQuota)
	assert.Equal(t, want, user.UsedQuota)
	assert.Equal(t, 1, user.RequestCount)
	assert.EqualValues(t, want, channel.UsedQuota)
	if funding == "wallet" {
		assert.Equal(t, initial-want, user.Quota)
	} else {
		assert.Equal(t, initial, user.Quota)
		require.NoError(t, db.First(&sub, sub.Id).Error)
		assert.EqualValues(t, want, sub.AmountUsed)
	}
}

func TestRSGatewayReceiptNonbillableMediaAttemptsKeepUsageWithoutCharging(t *testing.T) {
	included, billable, nonbillable := true, true, false
	for _, expression := range []string{`tier("token", p * 2 + c * 4)`, `tier("request", fixed(0.01))`} {
		t.Run(expression, func(t *testing.T) {
			info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatOpenAI,
				PriceData: hosttypes.PriceData{GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}},
				TieredBillingSnapshot: &billingexpr.BillingSnapshot{BillingMode: "tiered_expr", ExprString: expression,
					ExprHash: billingexpr.ExprHashString(expression), QuotaPerUnit: 500000, GroupRatio: 1}, BillingRequestInput: &billingexpr.RequestInput{}}
			receipt := RSGatewayBillingReceipt{Version: 1, RequestID: "media", Complete: true, Success: false,
				Attempts: []RSGatewayBillingAttempt{{PhysicalID: "no-image", Role: "hedge", State: "failed", BillToUser: &nonbillable,
					UsageObserved: true, Usage: &RSGatewayBillingUsage{InputTokens: 80, OutputTokens: 10, InputIncludesCache: &included}}}}
			quota, charges, prompt, completion, err := priceRSGatewayReceipt(info, receipt.RequestID, receipt)
			require.NoError(t, err)
			assert.Zero(t, quota)
			assert.Equal(t, 80, prompt)
			assert.Equal(t, 10, completion)
			require.Len(t, charges, 1)
			require.NotNil(t, charges[0].Usage)
			assert.EqualValues(t, 80, charges[0].Usage.InputTokens)
			receipt.Success = true
			receipt.Attempts = append(receipt.Attempts, RSGatewayBillingAttempt{PhysicalID: "retry-image", Role: "winner", State: "succeeded",
				BillToUser: &billable, UsageObserved: true, Usage: &RSGatewayBillingUsage{InputTokens: 20, OutputTokens: 5, InputIncludesCache: &included}})
			quota, charges, _, _, err = priceRSGatewayReceipt(info, receipt.RequestID, receipt)
			require.NoError(t, err)
			assert.Zero(t, charges[0].Quota)
			if strings.Contains(expression, "fixed") {
				assert.Equal(t, 5000, quota)
			} else {
				assert.Equal(t, 30, quota)
			}
			assert.Equal(t, charges[1].Quota, quota)
		})
	}
}
