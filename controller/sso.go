package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

const artSSOFlowPurpose = "art_sso"

type artSSOTokenRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
}

type artSSOAccountRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Subject      string `json:"subject"`
}

type artSSOProvisionTokenRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Subject      string `json:"subject"`
	Group        string `json:"group"`
	Name         string `json:"name"`
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

// ArtSSOAccount 向受信任的 OnlyArt 服务端返回对应用户的当前余额。
func ArtSSOAccount(c *gin.Context) {
	var request artSSOAccountRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || !validArtSSOClient(request.ClientID, request.ClientSecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "SSO 客户端无效"})
		return
	}
	userID, err := strconv.Atoi(strings.TrimSpace(request.Subject))
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "SSO 用户标识无效"})
		return
	}
	user, err := model.GetUserById(userID, false)
	if err != nil || user.Status != common.UserStatusEnabled {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "new-api 用户不存在或不可用"})
		return
	}
	displayType := operation_setting.GetQuotaDisplayType()
	balance := float64(user.Quota)
	if displayType != operation_setting.QuotaDisplayTypeTokens {
		if common.QuotaPerUnit <= 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "额度换算配置无效"})
			return
		}
		balance = balance / common.QuotaPerUnit * operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"balance":      balance,
		"symbol":       operation_setting.GetCurrencySymbol(),
		"display_type": displayType,
	}})
}

// ArtSSOGroups 向受信任的 OnlyArt 服务端返回当前可用的真实分组名。
func ArtSSOGroups(c *gin.Context) {
	var request artSSOAccountRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || !validArtSSOClient(request.ClientID, request.ClientSecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "SSO 客户端无效"})
		return
	}
	groups := make([]string, 0, len(ratio_setting.GetGroupRatioCopy()))
	for group := range ratio_setting.GetGroupRatioCopy() {
		if group != "" && group != "auto" {
			groups = append(groups, group)
		}
	}
	sort.Strings(groups)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": groups})
}

// ArtSSOProvisionToken 为 OnlyArt 当前用户创建或复用指定真实分组的 Token。
func ArtSSOProvisionToken(c *gin.Context) {
	var request artSSOProvisionTokenRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || !validArtSSOClient(request.ClientID, request.ClientSecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "SSO 客户端无效"})
		return
	}
	userID, err := strconv.Atoi(strings.TrimSpace(request.Subject))
	group := strings.TrimSpace(request.Group)
	name := strings.TrimSpace(request.Name)
	if err != nil || userID <= 0 || group == "" || group == "auto" || name == "" || len(name) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "用户或分组无效"})
		return
	}
	user, err := model.GetUserById(userID, false)
	if err != nil || user.Status != common.UserStatusEnabled {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "new-api 用户不存在或不可用"})
		return
	}
	if _, exists := ratio_setting.GetGroupRatioCopy()[group]; !exists {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "分组不存在"})
		return
	}
	var token model.Token
	if err := model.DB.Where(&model.Token{UserId: userID, Group: group, Name: name}).First(&token).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"token_id": token.Id, "key": token.Key, "group": group, "name": name}})
		return
	}
	count, err := model.CountUserTokens(userID)
	if err != nil || int(count) >= operation_setting.GetMaxUserTokens() {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "用户令牌数量已达上限"})
		return
	}
	key, err := common.GenerateKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "令牌生成失败"})
		return
	}
	token = model.Token{UserId: userID, Name: name, Key: key, CreatedTime: common.GetTimestamp(), AccessedTime: common.GetTimestamp(), ExpiredTime: -1, UnlimitedQuota: true, Group: group}
	if err := token.Insert(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "令牌创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"token_id": token.Id, "key": key, "group": group, "name": name}})
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
