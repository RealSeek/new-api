package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGroupStatusOmitsFallbackAndKeepsCacheRatesWithoutHealthSamples(t *testing.T) {
	oldDB, oldLogDB := model.DB, model.LOG_DB
	oldMainType, oldLogType := common.MainDatabaseType(), common.LogDatabaseType()
	oldRedis, oldMaster := common.RedisEnabled, common.IsMasterNode
	oldGroups := setting.UserUsableGroups2JSONString()
	oldRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		model.DB, model.LOG_DB = oldDB, oldLogDB
		common.SetDatabaseTypes(oldMainType, oldLogType)
		common.RedisEnabled, common.IsMasterNode = oldRedis, oldMaster
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(oldGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldRatios))
		model.InvalidatePricingCache()
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	model.DB = db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled, common.IsMasterNode = false, false
	t.Setenv("LOG_SQL_DSN", "")
	require.NoError(t, model.InitLogDB())
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Channel{}, &model.Ability{}, &model.Model{}, &model.Vendor{}, &model.ChannelModelDescription{}, &model.PerfMetric{}, &model.Log{}))
	model.InvalidatePricingCache()
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","auto":"Auto","board-cached":"Cached","board-miss":"Miss","board-idle":"Idle","board-unpriced":"Unpriced"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"auto":1,"board-cached":1,"board-miss":1,"board-idle":1}`))
	windowEnd := time.Now().Add(-10 * time.Minute).Unix()
	require.NoError(t, db.Create(&[]model.Log{
		{CreatedAt: windowEnd, Type: model.LogTypeConsume, Group: "board-cached", PromptTokens: 100, Other: `{"usage_semantic":"openai","input_tokens_total":100,"cache_tokens":75}`},
		{CreatedAt: windowEnd, Type: model.LogTypeConsume, Group: "board-miss", PromptTokens: 100, Other: `{"usage_semantic":"openai","input_tokens_total":100,"cache_tokens":0}`},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/perf-metrics/groups", nil)
	GetPerfMetricsGroups(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Groups []map[string]any `json:"groups"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	groups := make(map[string]map[string]any, len(response.Data.Groups))
	for _, group := range response.Data.Groups {
		name, ok := group["name"].(string)
		require.True(t, ok)
		groups[name] = group
	}
	require.Len(t, groups, 3)
	assert.NotContains(t, groups, "default")
	assert.NotContains(t, groups, "auto")
	assert.NotContains(t, groups, "board-unpriced")
	for name, rate := range map[string]float64{"board-cached": 75, "board-miss": 0} {
		require.Contains(t, groups, name)
		assert.Nil(t, groups[name]["status"], "retained cache usage is independent of health samples")
		assert.Equal(t, rate, groups[name]["cache_hit_rate"])
		assert.EqualValues(t, windowEnd, groups[name]["cache_window_end"])
	}
	require.Contains(t, groups, "board-idle")
	assert.Contains(t, groups["board-idle"], "cache_hit_rate")
	assert.Nil(t, groups["board-idle"]["cache_hit_rate"], "no cache samples must remain unknown")
	assert.EqualValues(t, 0, groups["board-idle"]["cache_window_end"])
}
