package model

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRSGatewaySettlementAtomicAndIdempotent(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&RSGatewaySettlement{}, &SubscriptionPreConsumeRecord{}))
	for _, test := range []struct {
		name       string
		funding    string
		quota      int
		refund     bool
		wantWallet int
		wantUsed   int64
		wantToken  int
	}{
		{"wallet_charge", "wallet", 30, false, 970, 0, 970},
		{"wallet_refund", "wallet", 0, true, 1000, 0, 1000},
		{"subscription_charge", "subscription", 30, false, 1000, 30, 970},
		{"subscription_refund", "subscription", 0, true, 1000, 0, 1000},
	} {
		t.Run(test.name, func(t *testing.T) {
			truncateTables(t)
			resetBatchUpdateTestState(t)
			user := createReserveTestUser(t, test.wantWallet)
			if test.funding == "wallet" {
				require.NoError(t, DB.Model(&user).Update("quota", 980).Error)
			}
			token := Token{UserId: user.Id, Key: "gateway-" + test.name, RemainQuota: 980, UsedQuota: 20}
			require.NoError(t, DB.Create(&token).Error)
			sub := UserSubscription{UserId: user.Id, AmountTotal: 1000, AmountUsed: 20}
			if test.funding == "subscription" {
				require.NoError(t, DB.Create(&sub).Error)
			}
			requestId := "gateway-" + test.name
			require.NoError(t, BeginRSGatewaySettlement(requestId, user.Id, token.Id, 1))
			input := RSGatewayCompletion{Quota: test.quota, PreConsumed: 20, TokenConsumed: 20, TokenKey: token.Key,
				Funding: test.funding, SubscriptionId: sub.Id, Refunded: test.refund}
			require.NoError(t, CompleteRSGatewaySettlement(requestId, input))
			require.NoError(t, CompleteRSGatewaySettlement(requestId, input))
			assert.Equal(t, test.wantWallet, getUserQuotaFromDB(t, user.Id))
			assert.Equal(t, test.wantToken, getTokenFromDB(t, token.Id).RemainQuota)
			if test.funding == "subscription" {
				require.NoError(t, DB.First(&sub, sub.Id).Error)
				assert.Equal(t, test.wantUsed, sub.AmountUsed)
			}
			var ledger RSGatewaySettlement
			require.NoError(t, DB.First(&ledger, "request_id = ?", requestId).Error)
			assert.Equal(t, int64(test.quota), ledger.Quota)
			if test.refund {
				assert.Equal(t, "refunded", ledger.State)
			} else {
				assert.Equal(t, "settled", ledger.State)
			}
			input.Quota++
			assert.Error(t, CompleteRSGatewaySettlement(requestId, input))
		})
	}
}

func TestRSGatewaySettlementRollsBackWhenTokenAdjustmentFails(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	require.NoError(t, DB.AutoMigrate(&RSGatewaySettlement{}))
	user := createReserveTestUser(t, 980)
	require.NoError(t, BeginRSGatewaySettlement("gateway-failed", user.Id, 999999, 1))
	err := CompleteRSGatewaySettlement("gateway-failed", RSGatewayCompletion{
		Quota: 30, PreConsumed: 20, TokenConsumed: 20, Funding: "wallet",
	})
	require.Error(t, err)
	assert.Equal(t, 980, getUserQuotaFromDB(t, user.Id))
	var ledger RSGatewaySettlement
	require.NoError(t, DB.First(&ledger, "request_id = ?", "gateway-failed").Error)
	assert.Equal(t, "pending", ledger.State)
	assert.Zero(t, ledger.Quota)
}

func TestRSGatewayReservationPersistsWithBatchMode(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	useUserCacheMiniRedis(t)
	common.BatchUpdateEnabled = true
	user := createReserveTestUser(t, 100)
	require.NoError(t, populateUserCache(user))
	reserved, err := TryReserveUserQuota(user.Id, 20, true)
	require.NoError(t, err)
	require.True(t, reserved)
	assert.Equal(t, 80, getUserQuotaFromDB(t, user.Id))
	token := createReserveTestToken(t, 100)
	for _, unlimited := range []bool{false, true} {
		reserved, err = TryReserveTokenQuota(token.Id, token.Key, 20, unlimited, true)
		require.NoError(t, err)
		require.True(t, reserved)
	}
	assert.Equal(t, 60, getTokenFromDB(t, token.Id).RemainQuota)
}
