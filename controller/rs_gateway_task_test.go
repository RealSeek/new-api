package controller

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	taskjsplugin "github.com/QuantumNous/new-api/relay/channel/task/jsplugin"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRSGatewayVideoDurationValidation(t *testing.T) {
	source, err := os.ReadFile("../plugins/tasks/rs-gateway/plugin.js")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(string(source), jsplugin.Options{})
	require.NoError(t, err)
	for _, tc := range []struct {
		name    string
		body    map[string]any
		invalid bool
	}{
		{"opaque metadata", map[string]any{"seconds": "8", "metadata": "not a JSON document"}, false},
		{"provider metadata is not native duration", map[string]any{"seconds": "8", "metadata": map[string]any{"seconds": -1, "resolution": "4k"}}, false},
		{"provider aliases do not change native duration", map[string]any{"seconds": "8", "durationSeconds": 1}, false},
		{"maximum duration", map[string]any{"seconds": "3600"}, false},
		{"boolean duration", map[string]any{"seconds": true}, true},
		{"hexadecimal duration", map[string]any{"seconds": "0x8"}, true},
		{"oversized duration", map[string]any{"seconds": 3601}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.body["prompt"] = "a wave"
			_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{"requestBody": tc.body, "baseUrl": "https://gateway.example", "upstreamModel": "video"})
			if tc.invalid {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRSGatewayNativeVideoMultipartPreservesBinary(t *testing.T) {
	source, err := os.ReadFile("../plugins/tasks/rs-gateway/plugin.js")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(string(source), jsplugin.Options{})
	require.NoError(t, err)
	service.InitHttpClient()
	fileBytes := []byte{0, 255, 1, 128, '\r', '\n', 0}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/videos", r.URL.Path)
		assert.Equal(t, "Bearer gateway-key", r.Header.Get("Authorization"))
		mediaType, parameters, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if !assert.NoError(t, err) || !assert.Equal(t, "multipart/form-data", mediaType) || !assert.NotEmpty(t, parameters["boundary"]) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !assert.NoError(t, r.ParseMultipartForm(1<<20)) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.MultipartForm.RemoveAll()
		assert.Equal(t, []string{"first", "second"}, r.MultipartForm.Value["custom_field"])
		assert.Equal(t, []string{`{"enabled":false,"seed":0}`}, r.MultipartForm.Value["metadata"])
		assert.Equal(t, "native-video", r.FormValue("model"))
		assert.Equal(t, "8", r.FormValue("seconds"))
		files := r.MultipartForm.File["input_reference"]
		if !assert.Len(t, files, 1) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		assert.Equal(t, "reference.png", files[0].Filename)
		assert.Equal(t, "image/png", files[0].Header.Get("Content-Type"))
		file, err := files[0].Open()
		if !assert.NoError(t, err) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()
		actual, err := io.ReadAll(file)
		assert.NoError(t, err)
		assert.Equal(t, fileBytes, actual)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"id":"video_gateway-task","status":"queued"}`))
	}))
	t.Cleanup(server.Close)
	var input bytes.Buffer
	writer := multipart.NewWriter(&input)
	for _, field := range []struct{ name, value string }{
		{"model", "native-video"}, {"prompt", "from a binary reference"}, {"seconds", "8"}, {"size", "1280x720"},
		{"custom_field", "first"}, {"custom_field", "second"}, {"metadata", `{"enabled":false,"seed":0}`},
	} {
		require.NoError(t, writer.WriteField(field.name, field.value))
	}
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="input_reference"; filename="reference.png"`},
		"Content-Type":        {"image/png"},
	})
	require.NoError(t, err)
	_, err = part.Write(fileBytes)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	c := taskSubmissionTestContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(input.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	info := taskSubmissionRelayInfo(nil)
	info.OriginModelName, info.UpstreamModelName = "native-video", "native-video"
	info.ChannelType, info.ChannelBaseUrl, info.ApiKey = constant.ChannelTypeRSGateway, server.URL, "gateway-key"
	adaptor := taskjsplugin.New(plugin)
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
	body, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	response, err := adaptor.DoRequest(c, info, body)
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusAccepted, response.StatusCode)
	parsed, taskErr := adaptor.ParseResponse(c, response, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "video_gateway-task", parsed.UpstreamTaskID)
}

func TestRSGatewayVideoSubmissionPersistsAndChargesOnce(t *testing.T) {
	db, dialect := openTaskDialectDatabase(t, &model.User{}, &model.Token{}, &model.Channel{}, &model.Task{}, &model.Log{}, &model.RSGatewaySettlement{})
	oldDB, oldLogDB := model.DB, model.LOG_DB
	oldMain, oldLog := common.MainDatabaseType(), common.LogDatabaseType()
	oldRedis, oldMemory, oldBatch, oldConsume, oldExport := common.RedisEnabled, common.MemoryCacheEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled, common.DataExportEnabled
	model.DB, model.LOG_DB = db, db
	common.SetDatabaseTypes(dialect, dialect)
	common.RedisEnabled, common.MemoryCacheEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled, common.DataExportEnabled = false, false, false, true, false
	oldVideoPrice := ratio_setting.VideoPrice2JSONString()
	t.Cleanup(func() {
		model.DB, model.LOG_DB = oldDB, oldLogDB
		common.SetDatabaseTypes(oldMain, oldLog)
		common.RedisEnabled, common.MemoryCacheEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled, common.DataExportEnabled = oldRedis, oldMemory, oldBatch, oldConsume, oldExport
		require.NoError(t, ratio_setting.UpdateVideoPriceByJSONString(oldVideoPrice))
	})
	withTieredBillingConfig(t, map[string]string{"gateway-video": "per_second"}, map[string]string{})
	require.NoError(t, ratio_setting.UpdateVideoPriceByJSONString(`{"gateway-video":{"default_price":0.125,"default_duration":5,"minimum_duration":1,"billing_step":1,"resolution_prices":{"720p":0.125}}}`))
	source, err := os.ReadFile("../plugins/tasks/rs-gateway/plugin.js")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(string(source), jsplugin.Options{})
	require.NoError(t, err)
	service.InitHttpClient()
	var captured map[string]any
	var tokenName string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := common.DecodeJson(r.Body, &captured); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		tokenName = r.Header.Get("X-RS-NewAPI-Token-Name")
		assert.Equal(t, "/v1/videos", r.URL.Path)
		assert.Equal(t, "Bearer gateway-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"id":"vendor-video","status":"queued","seconds":8}`))
	}))
	t.Cleanup(server.Close)
	initial := int(10 * common.QuotaPerUnit)
	user := model.User{Username: "gateway-video-owner", AffCode: "gateway-video-aff", Quota: initial}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "playground-context-token", RemainQuota: initial}
	require.NoError(t, db.Create(&token).Error)
	ch := model.Channel{Name: "RS Gateway", Type: constant.ChannelTypeRSGateway}
	require.NoError(t, db.Create(&ch).Error)
	c := taskSubmissionTestContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(`{"model":"gateway-video","prompt":"a wave","seconds":"8","size":"1280x720","metadata":"not a JSON document","resolution":"4k","duration":1,"custom_field":{"enabled":false,"value":0}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("group", "default")
	c.Set("username", user.Username)
	c.Set("token_name", "video-token")
	c.Set("model_mapping", `{"gateway-video":"provider-video"}`)
	c.Set(jsplugin.ContextKeyPinnedPlugin, jsplugin.PinnedPlugin{Plugin: plugin})
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "gateway-video")
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, server.URL)
	common.SetContextKey(c, constant.ContextKeyChannelId, ch.Id)
	common.SetContextKey(c, constant.ContextKeyChannelType, ch.Type)
	common.SetContextKey(c, constant.ContextKeyChannelKey, "gateway-key")
	info := taskSubmissionRelayInfo(nil)
	info.UserId, info.RequestId = user.Id, "gateway-video-request"
	info.TokenId = token.Id
	c.Set(common.RequestIdKey, info.RequestId)
	info.OriginModelName, info.UserGroup = "gateway-video", "default"
	info.TokenName = "video-token"
	info.IsPlayground = true
	info.UserSetting.BillingPreference = "wallet_only"
	info.PublicTaskID, info.LockedChannel = model.GenerateTaskID(), &ch
	outcome, taskErr := executeTaskSubmission(c, info)
	require.Nil(t, taskErr)
	require.NotNil(t, outcome)
	assert.Equal(t, "provider-video", captured["model"])
	assert.Equal(t, "a wave", captured["prompt"])
	assert.Equal(t, "8", captured["seconds"])
	assert.Equal(t, "1280x720", captured["size"])
	assert.Equal(t, "not a JSON document", captured["metadata"])
	assert.Equal(t, "4k", captured["resolution"], "provider extensions stay on the wire but do not override native size pricing")
	assert.Equal(t, map[string]any{"enabled": false, "value": float64(0)}, captured["custom_field"])
	assert.Equal(t, "video-token", tokenName)
	want := int(common.QuotaPerUnit)
	assert.Equal(t, want, outcome.Task.Quota)
	var stored model.Task
	require.NoError(t, db.Where("task_id = ?", info.PublicTaskID).First(&stored).Error)
	assert.Equal(t, constant.TaskPlatform("rs-gateway"), stored.Platform)
	assert.Equal(t, "vendor-video", stored.GetUpstreamTaskID())
	assert.Equal(t, "gateway-key", stored.PrivateData.Key)
	require.NotNil(t, stored.PrivateData.Execution.TaskPlugin)
	assert.Equal(t, "rs-gateway", stored.PrivateData.Execution.TaskPlugin.Key)
	var ledger model.RSGatewaySettlement
	require.NoError(t, db.Where("request_id = ?", info.RequestId).First(&ledger).Error)
	assert.Equal(t, "settled", ledger.State)
	assert.Equal(t, int64(want), ledger.Quota)
	assert.Zero(t, ledger.TokenId, "playground did not reserve API token quota")
	require.NoError(t, info.Billing.Settle(want))
	info.Billing.Refund(c)
	var updated model.User
	require.NoError(t, db.First(&updated, user.Id).Error)
	assert.Equal(t, initial-want, updated.Quota)
	assert.Equal(t, want, updated.UsedQuota)
	var logs []model.Log
	require.NoError(t, db.Where("user_id = ?", user.Id).Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, want, logs[0].Quota)

	// A failed task update must roll back the wallet and ledger too.
	stored.Status = model.TaskStatusFailure
	require.NoError(t, db.Model(&stored).Update("status", stored.Status).Error)
	stale := stored
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("gateway:reject-task-quota", func(tx *gorm.DB) {
		if tx.Statement.Model != nil {
			if _, ok := tx.Statement.Model.(*model.Task); ok {
				tx.AddError(errors.New("simulated task storage failure"))
			}
		}
	}))
	assert.False(t, service.RefundTaskQuota(t.Context(), &stored, "provider failed"))
	require.NoError(t, db.Callback().Update().Remove("gateway:reject-task-quota"))
	require.NoError(t, db.First(&updated, user.Id).Error)
	assert.Equal(t, initial-want, updated.Quota)
	assert.Equal(t, want, stored.Quota)
	require.NoError(t, db.Where("request_id = ?", info.RequestId).First(&ledger).Error)
	assert.Equal(t, "settled", ledger.State)
	require.True(t, service.RefundTaskQuota(t.Context(), &stored, "provider failed"))
	require.True(t, service.RefundTaskQuota(t.Context(), &stale, "retry"))
	require.NoError(t, db.First(&updated, user.Id).Error)
	assert.Equal(t, initial, updated.Quota)
	assert.Zero(t, updated.UsedQuota)
	assert.Equal(t, 1, updated.RequestCount)
	require.NoError(t, db.Where("request_id = ?", info.RequestId).First(&ledger).Error)
	assert.Equal(t, "refunded", ledger.State)
	assert.Zero(t, ledger.Quota)
	require.NoError(t, db.First(&stored, stored.ID).Error)
	assert.Zero(t, stored.Quota)
	require.NoError(t, db.First(&token, token.Id).Error)
	assert.Equal(t, initial, token.RemainQuota)
	assert.Zero(t, token.UsedQuota)
}
