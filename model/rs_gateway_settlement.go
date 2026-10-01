package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

// RSGatewaySettlement 与最终资金调整在主数据库中一起提交。
// pending 表示尚未完成结算或退款，不能据此推断收入为零。
type RSGatewaySettlement struct {
	RequestId      string `gorm:"primaryKey;type:varchar(64)"`
	UserId         int
	TokenId        int
	ChannelId      int
	Quota          int64
	State          string `gorm:"type:varchar(16);not null"`
	Funding        string `gorm:"type:varchar(16)"`
	SubscriptionId int
	CreatedAt      int64
	UpdatedAt      int64
}

func (RSGatewaySettlement) TableName() string { return "rs_gateway_settlements" }

func BeginRSGatewaySettlement(requestId string, userId, tokenId, channelId int) error {
	if requestId == "" || len(requestId) > 64 || userId <= 0 {
		return errors.New("invalid gateway settlement identity")
	}
	now := common.GetTimestamp()
	return DB.Create(&RSGatewaySettlement{
		RequestId: requestId, UserId: userId, TokenId: tokenId, ChannelId: channelId,
		State: "pending", CreatedAt: now, UpdatedAt: now,
	}).Error
}

type RSGatewayCompletion struct {
	Quota          int
	PreConsumed    int
	TokenConsumed  int
	TokenKey       string
	Funding        string
	SubscriptionId int
	Refunded       bool
}

// CompleteRSGatewaySettlement 只用于网关渠道，绕过内存额度批处理。
// 预扣已完成时只调整差额；资金、令牌和最终账目任一步失败均回滚。
func CompleteRSGatewaySettlement(requestId string, input RSGatewayCompletion) error {
	for _, quota := range []int{input.Quota, input.PreConsumed, input.TokenConsumed} {
		if err := common.ValidateWalletQuota(quota); err != nil {
			return err
		}
	}
	if input.Refunded && input.Quota != 0 {
		return errors.New("refund must have zero final quota")
	}
	state := "settled"
	if input.Refunded {
		state = "refunded"
	}
	fundingDelta := int64(input.Quota) - int64(input.PreConsumed)
	tokenDelta := int64(input.Quota) - int64(input.TokenConsumed)
	var record RSGatewaySettlement
	applied := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("request_id = ?", requestId).First(&record).Error; err != nil {
			return err
		}
		if record.State != "pending" {
			if record.State == state && record.Quota == int64(input.Quota) &&
				record.Funding == input.Funding && record.SubscriptionId == input.SubscriptionId {
				return nil
			}
			return errors.New("gateway settlement is already finalized with different values")
		}
		switch input.Funding {
		case "wallet":
			if fundingDelta != 0 {
				query := tx.Model(&User{}).Where("id = ?", record.UserId)
				if fundingDelta < 0 {
					query = query.Where("quota <= ?", int64(common.MaxWalletQuota)+fundingDelta)
				}
				result := query.Update("quota", gorm.Expr("quota - ?", fundingDelta))
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected != 1 {
					return errors.New("gateway wallet adjustment failed")
				}
			}
		case "subscription":
			var sub UserSubscription
			if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", input.SubscriptionId, record.UserId).First(&sub).Error; err != nil {
				return err
			}
			newUsed := sub.AmountUsed + fundingDelta
			if newUsed < 0 {
				newUsed = 0
			}
			if sub.AmountTotal > 0 && newUsed > sub.AmountTotal {
				return fmt.Errorf("subscription used exceeds total, used=%d total=%d", newUsed, sub.AmountTotal)
			}
			sub.AmountUsed = newUsed
			if err := tx.Save(&sub).Error; err != nil {
				return err
			}
			if input.Refunded {
				if err := tx.Model(&SubscriptionPreConsumeRecord{}).Where("request_id = ?", requestId).
					Update("status", "refunded").Error; err != nil {
					return err
				}
			}
		case "":
			if input.Quota != 0 || input.PreConsumed != 0 || input.TokenConsumed != 0 {
				return errors.New("gateway paid settlement has no funding source")
			}
		default:
			return errors.New("unsupported gateway funding source")
		}
		if record.TokenId > 0 && tokenDelta != 0 {
			result := tx.Model(&Token{}).Where("id = ? AND user_id = ?", record.TokenId, record.UserId).Updates(map[string]interface{}{
				"remain_quota":  gorm.Expr("remain_quota - ?", tokenDelta),
				"used_quota":    gorm.Expr("used_quota + ?", tokenDelta),
				"accessed_time": common.GetTimestamp(),
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("gateway token adjustment failed")
			}
		}
		record.Quota = int64(input.Quota)
		record.State = state
		record.Funding = input.Funding
		record.SubscriptionId = input.SubscriptionId
		record.UpdatedAt = common.GetTimestamp()
		if err := tx.Save(&record).Error; err != nil {
			return err
		}
		applied = true
		return nil
	})
	if err != nil || !applied {
		return err
	}
	// 缓存是资金数据库的派生数据，只有事务提交成功后才能调整。
	gopool.Go(func() {
		if input.Funding == "wallet" && fundingDelta != 0 {
			if err := cacheIncrUserQuota(record.UserId, -fundingDelta); err != nil {
				common.SysLog(fmt.Sprintf("gateway settlement %s: update wallet cache: %v", requestId, err))
			}
		}
		if common.RedisEnabled && record.TokenId > 0 && tokenDelta != 0 {
			if _, err := cacheApplyTokenQuotaDelta(record.TokenId, input.TokenKey, -tokenDelta); err != nil {
				common.SysLog(fmt.Sprintf("gateway settlement %s: update token cache: %v", requestId, err))
			}
		}
	})
	return nil
}

// RefundRSGatewayTaskQuota refunds the durable charge once. The task quota is
// the retry marker; the original request ledger remains the accounting record.
func RefundRSGatewayTaskQuota(task *Task) (int, error) {
	var stored Task
	var token Token
	quota := 0
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(&stored, task.ID).Error; err != nil {
			return err
		}
		if stored.Quota == 0 {
			return nil
		}
		if stored.Platform != "rs-gateway" || stored.Status != TaskStatusFailure {
			return errors.New("gateway task is not refundable")
		}
		if err := common.ValidateWalletQuota(stored.Quota); err != nil {
			return err
		}
		// Migrated tasks predate request provenance. Their durable task row
		// identifies the refund; do not guess which historical request charged it.
		ledger := RSGatewaySettlement{
			RequestId: fmt.Sprintf("task-refund:%d", stored.ID), UserId: stored.UserId, TokenId: stored.PrivateData.TokenId, ChannelId: stored.ChannelId,
			Quota: int64(stored.Quota), State: "settled", Funding: stored.PrivateData.BillingSource, SubscriptionId: stored.PrivateData.SubscriptionId,
			CreatedAt: common.GetTimestamp(),
		}
		legacy := stored.PrivateData.Execution == nil || stored.PrivateData.Execution.RequestID == ""
		if !legacy {
			ledger = RSGatewaySettlement{}
			if err := lockForUpdate(tx).Where("request_id = ?", stored.PrivateData.Execution.RequestID).First(&ledger).Error; err != nil {
				return err
			}
		} else if ledger.Funding == "" {
			ledger.Funding = "wallet"
		}
		if ledger.State != "settled" || ledger.Quota != int64(stored.Quota) || ledger.UserId != stored.UserId || ledger.ChannelId != stored.ChannelId ||
			(!legacy && ledger.Funding != stored.PrivateData.BillingSource) || ledger.SubscriptionId != stored.PrivateData.SubscriptionId {
			return errors.New("gateway task charge does not match settlement ledger")
		}
		amount := int64(stored.Quota)
		if ledger.Funding == "subscription" {
			result := tx.Model(&UserSubscription{}).Where("id = ? AND user_id = ? AND amount_used >= ?", ledger.SubscriptionId, stored.UserId, amount).
				Update("amount_used", gorm.Expr("amount_used - ?", amount))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("gateway task subscription refund failed")
			}
			if err := tx.Model(&SubscriptionPreConsumeRecord{}).Where("request_id = ?", ledger.RequestId).Update("status", "refunded").Error; err != nil {
				return err
			}
		} else if ledger.Funding != "wallet" {
			return errors.New("unsupported gateway task funding source")
		}
		userUpdates := map[string]any{"used_quota": gorm.Expr("used_quota - ?", amount)}
		userQuery := tx.Model(&User{}).Where("id = ?", stored.UserId)
		if ledger.Funding == "wallet" {
			userQuery = userQuery.Where("quota <= ?", int64(common.MaxWalletQuota)-amount)
			userUpdates["quota"] = gorm.Expr("quota + ?", amount)
		}
		result := userQuery.Updates(userUpdates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("gateway task wallet refund failed")
		}
		if ledger.TokenId > 0 {
			if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", ledger.TokenId, stored.UserId).First(&token).Error; err != nil {
				return err
			}
			result = tx.Model(&token).Updates(map[string]any{
				"remain_quota": gorm.Expr("remain_quota + ?", amount), "used_quota": gorm.Expr("used_quota - ?", amount),
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("gateway task token refund failed")
			}
		}
		if err := tx.Model(&Channel{}).Where("id = ?", stored.ChannelId).Update("used_quota", gorm.Expr("used_quota - ?", amount)).Error; err != nil {
			return err
		}
		if err := tx.Model(&stored).Update("quota", 0).Error; err != nil {
			return err
		}
		ledger.Quota, ledger.State, ledger.UpdatedAt = 0, "refunded", common.GetTimestamp()
		if legacy {
			if err := tx.Create(&ledger).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Save(&ledger).Error; err != nil {
				return err
			}
		}
		quota = int(amount)
		return nil
	})
	if err != nil {
		return 0, err
	}
	task.Quota = 0
	if quota == 0 {
		return 0, nil
	}
	if stored.PrivateData.BillingSource != "subscription" {
		if err := cacheIncrUserQuota(stored.UserId, int64(quota)); err != nil {
			common.SysError(fmt.Sprintf("gateway task %s wallet cache refund: %v", stored.TaskID, err))
		}
	}
	if common.RedisEnabled && token.Id > 0 {
		if _, err := cacheApplyTokenQuotaDelta(token.Id, token.Key, int64(quota)); err != nil {
			common.SysError(fmt.Sprintf("gateway task %s token cache refund: %v", stored.TaskID, err))
		}
	}
	return quota, nil
}
