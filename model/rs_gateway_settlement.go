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
