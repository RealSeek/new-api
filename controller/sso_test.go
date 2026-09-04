package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
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
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.AuthFlow{}))
	model.DB = db
	common.SessionSecret = "art-sso-test-session-secret"
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Setenv("ART_SSO_CLIENT_ID", "onlyart")
	t.Setenv("ART_SSO_CLIENT_SECRET", "test-client-secret-at-least-32-chars")
	t.Setenv("ART_SSO_REDIRECT_URIS", "https://art.example.com/v1/auth/new-api/callback")
	t.Cleanup(func() {
		model.DB = previousDB
		common.SessionSecret = previousSecret
		common.SetMainDatabaseType(previousType)
	})
	user := &model.User{Username: "sso-user", DisplayName: "SSO 用户", Email: "sso@example.com", Password: "unused-password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1}
	require.NoError(t, db.Create(user).Error)
	return user
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
