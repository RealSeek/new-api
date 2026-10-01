package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRSGatewayConcurrentTaskRefundCreditsOnce(t *testing.T) {
	if common.UsingMainDatabase(common.DatabaseTypeSQLite) {
		t.Skip("concurrent row locking requires a real MySQL or PostgreSQL test database")
	}
	for _, funding := range []string{BillingSourceWallet, BillingSourceSubscription} {
		t.Run(funding, func(t *testing.T) {
			truncate(t)
			seedUser(t, 1, 10000)
			seedToken(t, 1, 1, "concurrent-gateway-token", 5000)
			seedChannel(t, 1)
			seedChargedAccounting(t, 1, 1, 1, 3000, 1)
			subscriptionID := 0
			if funding == BillingSourceSubscription {
				subscriptionID = 1
				seedSubscription(t, 1, 1, 10000, 3000)
			}
			require.NoError(t, model.DB.AutoMigrate(&model.RSGatewaySettlement{}, &model.SubscriptionPreConsumeRecord{}))
			task := makeTask(1, 1, 3000, 1, funding, subscriptionID)
			task.Platform, task.Status = "rs-gateway", model.TaskStatusFailure
			require.NoError(t, model.DB.Create(task).Error)
			requestID := fmt.Sprintf("task-refund:%d", task.ID)
			t.Cleanup(func() {
				require.NoError(t, model.DB.Where("request_id = ?", requestID).Delete(&model.RSGatewaySettlement{}).Error)
			})
			start := make(chan struct{})
			results := make(chan struct {
				quota int
				err   error
			}, 2)
			for range 2 {
				stale := *task
				go func() {
					<-start
					quota, err := model.RefundRSGatewayTaskQuota(&stale)
					results <- struct {
						quota int
						err   error
					}{quota, err}
				}()
			}
			close(start)
			totalRefund := 0
			for range 2 {
				result := <-results
				require.NoError(t, result.err)
				totalRefund += result.quota
			}
			assert.Equal(t, 3000, totalRefund)
			wantWallet := 13000
			if funding == BillingSourceSubscription {
				wantWallet = 10000
				var sub model.UserSubscription
				require.NoError(t, model.DB.First(&sub, subscriptionID).Error)
				assert.Zero(t, sub.AmountUsed)
			}
			assert.Equal(t, wantWallet, getUserQuota(t, 1))
			assert.Zero(t, getTaskQuota(t, task.ID))
			var token model.Token
			require.NoError(t, model.DB.First(&token, 1).Error)
			assert.Equal(t, 8000, token.RemainQuota)
			assert.Zero(t, token.UsedQuota)
			var ledger model.RSGatewaySettlement
			require.NoError(t, model.DB.Where("request_id = ?", requestID).First(&ledger).Error)
			assert.Equal(t, "refunded", ledger.State)
			assert.Zero(t, ledger.Quota)
		})
	}
}

func TestRSGatewayTaskRefundRollsBackTokenFailureAndPollingRetries(t *testing.T) {
	for _, funding := range []string{BillingSourceWallet, BillingSourceSubscription} {
		t.Run(funding, func(t *testing.T) {
			truncate(t)
			seedUser(t, 1, 10000)
			seedToken(t, 1, 1, "gateway-task-token", 5000)
			seedChannel(t, 1)
			seedChargedAccounting(t, 1, 1, 1, 3000, 1)
			subscriptionID := 0
			if funding == BillingSourceSubscription {
				subscriptionID = 1
				seedSubscription(t, 1, 1, 10000, 3000)
			}
			require.NoError(t, model.DB.AutoMigrate(&model.RSGatewaySettlement{}, &model.SubscriptionPreConsumeRecord{}))
			requestID := "gateway-task-refund-" + funding
			t.Cleanup(func() {
				require.NoError(t, model.DB.Where("request_id = ?", requestID).Delete(&model.RSGatewaySettlement{}).Error)
			})
			require.NoError(t, model.BeginRSGatewaySettlement(requestID, 1, 1, 1))
			require.NoError(t, model.CompleteRSGatewaySettlement(requestID, model.RSGatewayCompletion{
				Quota: 3000, PreConsumed: 3000, TokenConsumed: 3000, Funding: funding, SubscriptionId: subscriptionID,
			}))
			task := makeTask(1, 1, 3000, 1, funding, subscriptionID)
			task.Platform, task.Status = "rs-gateway", model.TaskStatusFailure
			task.PrivateData.Execution = &model.TaskExecutionSnapshot{RequestID: requestID}
			require.NoError(t, model.DB.Create(task).Error)
			stale := *task
			require.NoError(t, model.DB.Callback().Update().Before("gorm:update").Register("gateway:reject-token", func(tx *gorm.DB) {
				if tx.Statement.Table == "tokens" {
					tx.AddError(errors.New("simulated token storage failure"))
				}
			}))
			assert.False(t, RefundTaskQuota(t.Context(), task, "provider failed"))
			require.NoError(t, model.DB.Callback().Update().Remove("gateway:reject-token"))
			assert.Equal(t, 10000, getUserQuota(t, 1))
			assert.Equal(t, 3000, getTaskQuota(t, task.ID))
			var ledger model.RSGatewaySettlement
			require.NoError(t, model.DB.Where("request_id = ?", requestID).First(&ledger).Error)
			assert.Equal(t, "settled", ledger.State)

			oldFactory := GetTaskAdaptorFunc
			GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return nil }
			t.Cleanup(func() { GetTaskAdaptorFunc = oldFactory })
			RunTaskPollingOnce(t.Context(), nil)
			require.True(t, RefundTaskQuota(t.Context(), &stale, "stale retry"))
			wantWallet := 13000
			if funding == BillingSourceSubscription {
				wantWallet = 10000
				var sub model.UserSubscription
				require.NoError(t, model.DB.First(&sub, subscriptionID).Error)
				assert.Zero(t, sub.AmountUsed)
			}
			assert.Equal(t, wantWallet, getUserQuota(t, 1))
			assert.Zero(t, getTaskQuota(t, task.ID))
			var token model.Token
			require.NoError(t, model.DB.First(&token, 1).Error)
			assert.Equal(t, 8000, token.RemainQuota)
			assert.Zero(t, token.UsedQuota)
			require.NoError(t, model.DB.Where("request_id = ?", requestID).First(&ledger).Error)
			assert.Equal(t, "refunded", ledger.State)
			assert.Zero(t, ledger.Quota)
			var refundLogs int64
			require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("type = ?", model.LogTypeRefund).Count(&refundLogs).Error)
			assert.Equal(t, int64(1), refundLogs)
		})
	}
}

func TestRSGatewayMigratedTaskRefundUsesDurableTaskIdentity(t *testing.T) {
	truncate(t)
	seedUser(t, 1, 10000)
	seedToken(t, 1, 1, "legacy-gateway-token", 5000)
	seedChannel(t, 1)
	seedChargedAccounting(t, 1, 1, 1, 3000, 1)
	require.NoError(t, model.DB.AutoMigrate(&model.RSGatewaySettlement{}))
	task := makeTask(1, 1, 3000, 1, "", 0)
	task.TaskID, task.Platform, task.Status = "task_legacy_gateway", "rs-gateway", model.TaskStatusFailure
	task.PrivateData.Execution = &model.TaskExecutionSnapshot{TaskPlugin: &model.TaskPluginSnapshot{Key: "rs-gateway"}}
	require.NoError(t, model.DB.Create(task).Error)
	stale := *task
	requestID := fmt.Sprintf("task-refund:%d", task.ID)
	t.Cleanup(func() {
		require.NoError(t, model.DB.Where("request_id = ?", requestID).Delete(&model.RSGatewaySettlement{}).Error)
	})
	require.True(t, RefundTaskQuota(t.Context(), task, "legacy provider failed"))
	require.True(t, RefundTaskQuota(t.Context(), &stale, "stale retry"))
	assert.Equal(t, 13000, getUserQuota(t, 1))
	assert.Zero(t, getTaskQuota(t, task.ID))
	var ledger model.RSGatewaySettlement
	require.NoError(t, model.DB.Where("request_id = ?", requestID).First(&ledger).Error)
	assert.Equal(t, "refunded", ledger.State)
}
