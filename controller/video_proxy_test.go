package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestVideoProxyConfiguredGatewayAllowsPrivateEndpointWithoutRedirects(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Task{}))
	originalDB, originalCache := model.DB, common.MemoryCacheEnabled
	settings := system_setting.GetFetchSetting()
	originalSettings := *settings
	model.DB, common.MemoryCacheEnabled = db, false
	settings.EnableSSRFProtection, settings.AllowPrivateIp = true, false
	t.Cleanup(func() {
		model.DB, common.MemoryCacheEnabled = originalDB, originalCache
		*settings = originalSettings
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	service.InitHttpClient()
	redirect := false
	followed := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirected" {
			followed = true
			w.WriteHeader(http.StatusOK)
			return
		}
		require.Equal(t, "/v1/videos/upstream-id/content", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		if redirect {
			http.Redirect(w, r, "/redirected", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("video-content"))
	}))
	defer upstream.Close()
	channel := model.Channel{Type: constant.ChannelTypeRSGateway, BaseURL: &upstream.URL, Key: "test-key"}
	require.NoError(t, db.Create(&channel).Error)
	task := model.Task{UserId: 1, ChannelId: channel.Id, TaskID: "public-id", Status: model.TaskStatusSuccess, PrivateData: model.TaskPrivateData{UpstreamTaskID: "upstream-id"}}
	require.NoError(t, db.Create(&task).Error)
	request := func(userID int) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(response)
		c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/public-id/content", nil)
		c.Set("id", userID)
		c.Params = gin.Params{{Key: "task_id", Value: "public-id"}}
		VideoProxy(c)
		return response
	}
	response := request(1)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "video-content", response.Body.String())
	require.Equal(t, http.StatusNotFound, request(2).Code)
	redirect = true
	require.Equal(t, http.StatusBadGateway, request(1).Code)
	require.False(t, followed)
	// 普通外部结果地址仍须通过私网校验。
	require.NoError(t, db.Model(&channel).Update("type", constant.ChannelTypeMiniMax).Error)
	task.PrivateData.ResultURL = upstream.URL + "/v1/videos/upstream-id/content"
	require.NoError(t, db.Save(&task).Error)
	require.Equal(t, http.StatusForbidden, request(1).Code)
}
