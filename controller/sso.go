package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const artSSOFlowPurpose = "art_sso"

type artSSOTokenRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
}

// ArtSSOAuthorize 启动 OnlyArt SSO 浏览器流程，只向 OnlyArt 返回一次性 code。
func ArtSSOAuthorize(c *gin.Context) {
	clientID := strings.TrimSpace(c.Query("client_id"))
	redirectURI := strings.TrimSpace(c.Query("redirect_uri"))
	state := strings.TrimSpace(c.Query("state"))
	if clientID == "" || clientID != strings.TrimSpace(os.Getenv("ART_SSO_CLIENT_ID")) || len(state) > 200 || state == "" || !allowedArtSSORedirect(redirectURI) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "SSO 回调地址或状态无效"})
		return
	}
	rawRefresh, err := c.Cookie(service.RefreshCookieName)
	if err != nil || strings.TrimSpace(rawRefresh) == "" {
		loginPath := "/sign-in?redirect=" + url.QueryEscape(c.Request.URL.RequestURI())
		c.Redirect(http.StatusFound, loginPath)
		return
	}

	// 刷新一次会话以验证 HttpOnly cookie，并保持 new-api 的会话轮换策略。
	bundle, user, err := service.RefreshLoginSession(rawRefresh, c.GetHeader("X-Auth-Session"), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		service.ClearRefreshCookie(c)
		loginPath := "/sign-in?redirect=" + url.QueryEscape(c.Request.URL.RequestURI())
		c.Redirect(http.StatusFound, loginPath)
		return
	}
	service.WriteRefreshCookie(c, bundle.RefreshToken)

	payload, err := common.Marshal(map[string]string{"client_id": clientID, "redirect_uri": redirectURI, "state": state})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "SSO 票据创建失败"})
		return
	}
	code, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
		Purpose:   artSSOFlowPurpose,
		Provider:  "art",
		Intent:    model.AuthFlowIntentLogin,
		UserId:    user.Id,
		SessionId: bundle.Session.SID,
		Payload:   string(payload),
		ExpiresAt: time.Now().Add(60 * time.Second),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "SSO 票据创建失败"})
		return
	}
	location, _ := url.Parse(redirectURI)
	query := location.Query()
	query.Set("code", code)
	query.Set("state", state)
	location.RawQuery = query.Encode()
	c.Redirect(http.StatusFound, location.String())
}

// ArtSSOToken 使用一次性 code 换取最小用户身份，不暴露 new-api Token。
func ArtSSOToken(c *gin.Context) {
	var request artSSOTokenRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || !validArtSSOClient(request.ClientID, request.ClientSecret) || strings.TrimSpace(request.Code) == "" || !allowedArtSSORedirect(request.RedirectURI) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "SSO 客户端或票据无效"})
		return
	}
	flow, err := model.GetAuthFlow(request.Code, model.AuthFlowMatch{Purpose: artSSOFlowPurpose, Provider: "art", Intent: model.AuthFlowIntentLogin})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "SSO 票据无效或已过期"})
		return
	}
	var payload struct {
		ClientID    string `json:"client_id"`
		RedirectURI string `json:"redirect_uri"`
		State       string `json:"state"`
	}
	if common.UnmarshalJsonStr(flow.Payload, &payload) != nil || payload.ClientID != request.ClientID || payload.RedirectURI != strings.TrimSpace(request.RedirectURI) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "SSO 票据绑定信息无效"})
		return
	}
	claimed, err := model.ConsumeAuthFlow(request.Code, model.AuthFlowMatch{Purpose: artSSOFlowPurpose, Provider: "art", Intent: model.AuthFlowIntentLogin, UserId: flow.UserId, SessionId: flow.SessionId})
	if err != nil || claimed == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "SSO 票据已被使用"})
		return
	}
	user, err := model.GetUserById(flow.UserId, false)
	if err != nil || user.Status != common.UserStatusEnabled {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "new-api 用户不可用"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"subject":      strconv.Itoa(user.Id),
		"username":     user.Username,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"avatar_url":   "",
		"role":         user.Role,
	}})
}

func validArtSSOClient(clientID, clientSecret string) bool {
	expectedID := strings.TrimSpace(os.Getenv("ART_SSO_CLIENT_ID"))
	expectedSecret := strings.TrimSpace(os.Getenv("ART_SSO_CLIENT_SECRET"))
	if expectedID == "" || len(expectedSecret) < 32 || strings.TrimSpace(clientID) != expectedID {
		return false
	}
	actual := sha256.Sum256([]byte(clientSecret))
	expected := sha256.Sum256([]byte(expectedSecret))
	return hmac.Equal(actual[:], expected[:])
}

func allowedArtSSORedirect(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, item := range strings.Split(os.Getenv("ART_SSO_REDIRECT_URIS"), ",") {
		if strings.TrimSpace(item) == value {
			return true
		}
	}
	return false
}
