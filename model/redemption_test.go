package model

import (
	"fmt"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSearchRedemptionsFiltersAndPaginates(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	})

	now := common.GetTimestamp()
	redemptions := []Redemption{
		{Id: 1, Name: "alpha-active", Key: "00000000000000000000000000000001", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: 0},
		{Id: 2, Name: "alpha-future", Key: "00000000000000000000000000000002", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: now + 3600},
		{Id: 3, Name: "alpha-expired", Key: "00000000000000000000000000000003", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: now - 10},
		{Id: 4, Name: "beta-disabled", Key: "00000000000000000000000000000004", Status: common.RedemptionCodeStatusDisabled, ExpiredTime: 0},
		{Id: 5, Name: "beta-used", Key: "00000000000000000000000000000005", Status: common.RedemptionCodeStatusUsed, ExpiredTime: 0},
	}
	require.NoError(t, DB.Create(&redemptions).Error)

	tests := []struct {
		name      string
		keyword   string
		status    string
		startIdx  int
		num       int
		wantTotal int64
		wantIds   []int
	}{
		{
			name:      "no filters returns all rows",
			num:       10,
			wantTotal: 5,
			wantIds:   []int{5, 4, 3, 2, 1},
		},
		{
			name:      "keyword filters by name prefix",
			keyword:   "alpha",
			num:       10,
			wantTotal: 3,
			wantIds:   []int{3, 2, 1},
		},
		{
			name:      "enabled status excludes expired rows",
			status:    "1",
			num:       10,
			wantTotal: 2,
			wantIds:   []int{2, 1},
		},
		{
			name:      "expired status returns enabled expired rows",
			status:    "expired",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{3},
		},
		{
			name:      "disabled status",
			status:    "2",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{4},
		},
		{
			name:      "used status",
			status:    "3",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{5},
		},
		{
			name:      "pagination keeps unpaged total",
			startIdx:  1,
			num:       2,
			wantTotal: 5,
			wantIds:   []int{4, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, total, err := SearchRedemptions(tt.keyword, tt.status, tt.startIdx, tt.num)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			gotIds := make([]int, 0, len(rows))
			for _, row := range rows {
				gotIds = append(gotIds, row.Id)
			}
			assert.Equal(t, tt.wantIds, gotIds)
		})
	}
}

func setupRedeemFixtures(t *testing.T, quotas []int) (userId int, keys []string) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
		DB.Exec("DELETE FROM users")
		DB.Exec("DELETE FROM logs")
	})

	user := &User{Username: "redeem-user", Password: "password", Status: common.UserStatusEnabled, Quota: 0}
	require.NoError(t, DB.Create(user).Error)

	redemptions := make([]Redemption, 0, len(quotas))
	keys = make([]string, 0, len(quotas))
	for i, quota := range quotas {
		key := fmt.Sprintf("%032d", i+1)
		keys = append(keys, key)
		redemptions = append(redemptions, Redemption{
			Name:        "redeem-test",
			Key:         key,
			Status:      common.RedemptionCodeStatusEnabled,
			Quota:       quota,
			CreatedTime: common.GetTimestamp(),
		})
	}
	require.NoError(t, DB.Create(&redemptions).Error)
	return user.Id, keys
}

func setupRedeemFixture(t *testing.T, quota int) (userId int, key string) {
	t.Helper()
	userId, keys := setupRedeemFixtures(t, []int{quota})
	return userId, keys[0]
}

func TestRedeemCreditsQuotaExactlyOnce(t *testing.T) {
	userId, key := setupRedeemFixture(t, 500)

	quota, err := Redeem(key, userId)
	require.NoError(t, err)
	assert.Equal(t, 500, quota)

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 500, user.Quota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "name = ?", "redeem-test").Error)
	assert.Equal(t, common.RedemptionCodeStatusUsed, redemption.Status)
	assert.Equal(t, userId, redemption.UsedUserId)

	// Redeeming the same code again must fail and must not credit quota.
	_, err = Redeem(key, userId)
	require.Error(t, err)
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 500, user.Quota)
}

func TestRedeemBatchCreditsAllCodesAtomically(t *testing.T) {
	userId, keys := setupRedeemFixtures(t, []int{100, 200, 300})

	quota, err := RedeemBatch(keys, userId)
	require.NoError(t, err)
	assert.Equal(t, 600, quota)

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 600, user.Quota)

	var redemptions []Redemption
	require.NoError(t, DB.Where(commonKeyCol+" IN ?", keys).Find(&redemptions).Error)
	require.Len(t, redemptions, 3)
	for _, redemption := range redemptions {
		assert.Equal(t, common.RedemptionCodeStatusUsed, redemption.Status)
		assert.Equal(t, userId, redemption.UsedUserId)
	}
}

func TestRedeemBatchRollsBackWhenAnyCodeIsInvalid(t *testing.T) {
	userId, keys := setupRedeemFixtures(t, []int{100, 200})

	_, err := RedeemBatch(append(keys, "missing-redemption-code"), userId)
	require.Error(t, err)

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Zero(t, user.Quota)

	var redemptions []Redemption
	require.NoError(t, DB.Where(commonKeyCol+" IN ?", keys).Find(&redemptions).Error)
	require.Len(t, redemptions, 2)
	for _, redemption := range redemptions {
		assert.Equal(t, common.RedemptionCodeStatusEnabled, redemption.Status)
		assert.Zero(t, redemption.UsedUserId)
	}
}

func TestRedeemBatchRejectsDuplicateCodes(t *testing.T) {
	userId, keys := setupRedeemFixtures(t, []int{100})

	_, err := RedeemBatch([]string{keys[0], keys[0]}, userId)
	require.Error(t, err)

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Zero(t, user.Quota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, commonKeyCol+" = ?", keys[0]).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, redemption.Status)
}

func TestRedeemBatchRejectsQuotaOverflow(t *testing.T) {
	userId, keys := setupRedeemFixtures(t, []int{100})
	require.NoError(t, DB.Model(&User{}).Where("id = ?", userId).Update("quota", common.MaxQuota-50).Error)

	_, err := RedeemBatch(keys, userId)
	require.Error(t, err)

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, common.MaxQuota-50, user.Quota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, commonKeyCol+" = ?", keys[0]).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, redemption.Status)
}

// Exactly one of several concurrent redeems of the same code may win, and
// quota must be credited exactly once.
func TestRedeemConcurrentSingleSuccess(t *testing.T) {
	userId, key := setupRedeemFixture(t, 300)

	const goroutines = 5
	successes := make([]bool, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			if _, err := Redeem(key, userId); err == nil {
				successes[idx] = true
			}
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, ok := range successes {
		if ok {
			successCount++
		}
	}
	assert.Equal(t, 1, successCount, "exactly one concurrent redeem should succeed")

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 300, user.Quota, "quota must be credited exactly once")
}
