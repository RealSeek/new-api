package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupArtSSOTest(t *testing.T) *model.User {
	t.Helper()
	previousDB := model.DB
	previousType := common.MainDatabaseType()
	previousSecret := common.SessionSecret
	previousQuotaPerUnit := common.QuotaPerUnit
	previousDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalGroupRatios := ratio_setting.GroupRatio2JSONString()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.AuthFlow{}, &model.Token{}, &model.Ability{}))
	model.DB = db
	common.SessionSecret = "art-sso-test-session-secret"
	common.QuotaPerUnit = 500000
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Setenv("ART_SSO_CLIENT_ID", "onlyart")
	t.Setenv("ART_SSO_CLIENT_SECRET", "test-client-secret-at-least-32-chars")
	t.Setenv("ART_SSO_REDIRECT_URIS", "https://art.example.com/v1/auth/new-api/callback")
	t.Cleanup(func() {
		model.DB = previousDB
		common.SessionSecret = previousSecret
		common.QuotaPerUnit = previousQuotaPerUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = previousDisplayType
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatios))
		common.SetMainDatabaseType(previousType)
	})
	user := &model.User{Username: "sso-user", DisplayName: "SSO 用户", Email: "sso@example.com", Password: "unused-password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, Quota: 1000000}
	require.NoError(t, db.Create(user).Error)
	return user
}

func queryArtSSOGroups(t *testing.T, request artSSOGroupsRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := common.Marshal(request)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/sso/art/groups", bytes.NewReader(body))
	ArtSSOGroups(context)
	return recorder
}

func queryArtSSOAccount(t *testing.T, subject, secret string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := common.Marshal(artSSOAccountRequest{ClientID: "onlyart", ClientSecret: secret, Subject: subject})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/sso/art/account", bytes.NewReader(body))
	ArtSSOAccount(context)
	return recorder
}

func exchangeArtSSOCode(t *testing.T, code, secret string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := common.Marshal(artSSOTokenRequest{ClientID: "onlyart", ClientSecret: secret, Code: code, RedirectURI: "https://art.example.com/v1/auth/new-api/callback"})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/sso/art/token", bytes.NewReader(body))
	ArtSSOToken(context)
	return recorder
}

func TestArtSSOAuthorizeRedirectsToSameOriginSignInWithoutSession(t *testing.T) {
	setupArtSSOTest(t)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/user/auth/sso/art/authorize?client_id=onlyart&redirect_uri=https%3A%2F%2Fart.example.com%2Fv1%2Fauth%2Fnew-api%2Fcallback&state=state-value", nil)

	ArtSSOAuthorize(context)

	require.Equal(t, http.StatusFound, recorder.Code)
	location := recorder.Header().Get("Location")
	assert.True(t, strings.HasPrefix(location, "/sign-in?redirect="))
	assert.Contains(t, location, "%2Fapi%2Fuser%2Fauth%2Fsso%2Fart%2Fauthorize")
}

func TestArtSSOTokenReturnsMinimalIdentityAndRejectsReplay(t *testing.T) {
	user := setupArtSSOTest(t)
	payload, err := common.Marshal(map[string]string{"client_id": "onlyart", "redirect_uri": "https://art.example.com/v1/auth/new-api/callback", "state": "state-value"})
	require.NoError(t, err)
	code, _, err := model.CreateAuthFlow(model.AuthFlowCreate{Purpose: artSSOFlowPurpose, Provider: "art", Intent: model.AuthFlowIntentLogin, UserId: user.Id, SessionId: "session-id", Payload: string(payload), ExpiresAt: time.Now().Add(time.Minute)})
	require.NoError(t, err)

	response := exchangeArtSSOCode(t, code, "test-client-secret-at-least-32-chars")
	require.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"subject":"1"`)
	assert.NotContains(t, response.Body.String(), "password")

	replay := exchangeArtSSOCode(t, code, "test-client-secret-at-least-32-chars")
	assert.Equal(t, http.StatusUnauthorized, replay.Code)
}

func TestArtSSOTokenRejectsInvalidClientSecretWithoutConsumingCode(t *testing.T) {
	user := setupArtSSOTest(t)
	payload := `{"client_id":"onlyart","redirect_uri":"https://art.example.com/v1/auth/new-api/callback","state":"state-value"}`
	code, _, err := model.CreateAuthFlow(model.AuthFlowCreate{Purpose: artSSOFlowPurpose, Provider: "art", Intent: model.AuthFlowIntentLogin, UserId: user.Id, SessionId: "session-id", Payload: payload, ExpiresAt: time.Now().Add(time.Minute)})
	require.NoError(t, err)

	response := exchangeArtSSOCode(t, code, "wrong-secret")
	assert.Equal(t, http.StatusUnauthorized, response.Code)
	_, err = model.GetAuthFlow(code, model.AuthFlowMatch{Purpose: artSSOFlowPurpose})
	assert.NoError(t, err)
}

func TestArtSSOAccountReturnsCurrentBalance(t *testing.T) {
	user := setupArtSSOTest(t)

	response := queryArtSSOAccount(t, strconv.Itoa(user.Id), "test-client-secret-at-least-32-chars")

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"success":true,"data":{"balance":2,"symbol":"$","display_type":"USD"}}`, response.Body.String())
}

func TestArtSSOAccountRejectsInvalidClient(t *testing.T) {
	user := setupArtSSOTest(t)

	response := queryArtSSOAccount(t, strconv.Itoa(user.Id), "wrong-secret")

	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestArtSSOGroupsKeepsDefaultStringResponse(t *testing.T) {
	setupArtSSOTest(t)
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"zeta":2,"":3,"auto":4,"alpha":1}`))

	response := queryArtSSOGroups(t, artSSOGroupsRequest{ClientID: "onlyart", ClientSecret: "test-client-secret-at-least-32-chars"})

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"success":true,"data":["alpha","zeta"]}`, response.Body.String())
}

func TestArtSSOGroupsDetailsIncludesSortedRatioAndModels(t *testing.T) {
	setupArtSSOTest(t)
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"zeta":2,"alpha":1,"auto":4,"":3}`))
	abilities := []model.Ability{
		{Group: "zeta", Model: "model-z", ChannelId: 1, Enabled: true},
		{Group: "zeta", Model: "model-a", ChannelId: 2, Enabled: true},
		{Group: "zeta", Model: "model-z", ChannelId: 3, Enabled: true},
		{Group: "alpha", Model: "model-b", ChannelId: 4, Enabled: true},
		{Group: "alpha", Model: "model-a", ChannelId: 5, Enabled: false},
	}
	require.NoError(t, model.DB.Create(&abilities).Error)

	response := queryArtSSOGroups(t, artSSOGroupsRequest{ClientID: "onlyart", ClientSecret: "test-client-secret-at-least-32-chars", Details: true})

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"success":true,"data":[{"name":"alpha","ratio":1,"models":["model-b"]},{"name":"zeta","ratio":2,"models":["model-a","model-z"]}]}`, response.Body.String())
}

func TestArtSSOGroupsRejectsInvalidClient(t *testing.T) {
	setupArtSSOTest(t)
	response := queryArtSSOGroups(t, artSSOGroupsRequest{ClientID: "onlyart", ClientSecret: "wrong-secret", Details: true})
	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestArtSSOProvisionTokenCreatesReusableGroupedToken(t *testing.T) {
	user := setupArtSSOTest(t)
	body, err := common.Marshal(artSSOProvisionTokenRequest{ClientID: "onlyart", ClientSecret: "test-client-secret-at-least-32-chars", Subject: strconv.Itoa(user.Id), Group: "default", Name: "onlyart-sso-user-default"})
	require.NoError(t, err)

	request := func() *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(http.MethodPost, "/api/sso/art/provision-token", bytes.NewReader(body))
		ArtSSOProvisionToken(context)
		return recorder
	}

	first := request()
	require.Equal(t, http.StatusOK, first.Code)
	assert.Contains(t, first.Body.String(), `"group":"default"`)
	assert.Contains(t, first.Body.String(), `"key":"`)
	second := request()
	require.Equal(t, http.StatusOK, second.Code)
	assert.JSONEq(t, first.Body.String(), second.Body.String())
	var count int64
	require.NoError(t, model.DB.Model(&model.Token{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestArtSSOProvisionTokenRejectsUnknownGroup(t *testing.T) {
	user := setupArtSSOTest(t)
	body, err := common.Marshal(artSSOProvisionTokenRequest{ClientID: "onlyart", ClientSecret: "test-client-secret-at-least-32-chars", Subject: strconv.Itoa(user.Id), Group: "missing-group", Name: "onlyart-sso-user-missing"})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/sso/art/provision-token", bytes.NewReader(body))

	ArtSSOProvisionToken(context)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}
