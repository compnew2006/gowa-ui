package facebookoauth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

const defaultAction = "connect"

var scopes = []string{
	"email",
	"pages_manage_posts",
	"pages_manage_engagement",
	"pages_manage_metadata",
	"pages_read_engagement",
	"pages_read_user_content",
	"pages_show_list",
	"pages_messaging",
	"public_profile",
	"read_insights",
	"business_management",
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       any    `json:"error"`
}

type tokenDebugResponse struct {
	Data struct {
		AppID     string   `json:"app_id"`
		Type      string   `json:"type"`
		IsValid   bool     `json:"is_valid"`
		ExpiresAt int64    `json:"expires_at"`
		Scopes    []string `json:"scopes"`
		Error     any      `json:"error"`
	} `json:"data"`
	Error any `json:"error"`
}

type userResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
}

type pagesResponse struct {
	Data []map[string]any `json:"data"`
}

type runtimeConfig struct {
	AppID       string
	AppSecret   string
	APIVersion  string
	RedirectURI string
	BaseURL     string
}

func (p *Plugin) InitFacebookOAuth(r *fastglue.Request) error {
	requestDB, organizationID, userID, ok := p.authContext(r)
	if !ok {
		return nil
	}

	action := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("action")))
	if action == "" {
		action = defaultAction
	}
	if action != "connect" && action != "renew" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid OAuth action", nil, "action")
	}

	accountID := uuid.Nil
	if action == "renew" {
		rawAccountID := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("account_id")))
		if rawAccountID == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "account_id is required for renewal", nil, "account_id")
		}
		var err error
		accountID, err = uuid.Parse(rawAccountID)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid account_id", nil, "account_id")
		}
		if _, ok := findAccount(r, requestDB, accountID, organizationID); !ok {
			return nil
		}
	}

	config, err := p.runtimeConfig(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	stateToken, err := randomString(64)
	if err != nil {
		p.app.Log.Error("Failed to generate Facebook OAuth state", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initiate Facebook OAuth", nil, "")
	}

	expiresAt := time.Now().Add(10 * time.Minute)
	state := OAuthState{
		OrganizationID: organizationID,
		UserID:         userID,
		AccountID:      accountID,
		StateToken:     stateToken,
		Action:         action,
		ExpiresAt:      expiresAt,
	}
	if err := requestDB.Create(&state).Error; err != nil {
		p.app.Log.Error("Failed to store Facebook OAuth state", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initiate Facebook OAuth", nil, "")
	}

	authURL := authorizationURL(config, stateToken)
	if string(r.RequestCtx.QueryArgs().Peek("redirect")) == "1" {
		r.RequestCtx.Redirect(authURL, fasthttp.StatusTemporaryRedirect)
		return nil
	}
	return r.SendEnvelope(map[string]any{
		"auth_url":   authURL,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func (p *Plugin) CallbackFacebookOAuth(r *fastglue.Request) error {
	code := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("code")))
	stateToken := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("state")))
	errorParam := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("error")))
	if errorParam != "" {
		return p.redirectResult(r, "error", string(r.RequestCtx.QueryArgs().Peek("error_description")))
	}
	if code == "" || stateToken == "" {
		return p.redirectResult(r, "error", "Invalid callback parameters")
	}
	if p.app == nil || p.app.DB == nil {
		return p.redirectResult(r, "error", "Facebook OAuth is unavailable")
	}

	db := p.app.DB.Session(&gorm.Session{})
	var state OAuthState
	if err := db.Where("state_token = ?", stateToken).First(&state).Error; err != nil {
		return p.redirectResult(r, "error", "Invalid or expired OAuth state")
	}
	if err := db.Delete(&state).Error; err != nil {
		p.app.Log.Warn("Failed to delete Facebook OAuth state", "error", err)
	}
	if time.Now().After(state.ExpiresAt) {
		return p.redirectResult(r, "error", "Invalid or expired OAuth state")
	}

	config, err := p.runtimeConfig(r)
	if err != nil {
		return p.redirectResult(r, "error", err.Error())
	}
	shortToken, err := p.exchangeCode(config, code)
	if err != nil {
		p.app.Log.Error("Failed to exchange Facebook OAuth code", "error", err)
		return p.redirectResult(r, "error", "Failed to connect Facebook account")
	}
	longToken, expiresIn, err := p.exchangeLongLivedToken(config, shortToken)
	if err != nil {
		p.app.Log.Error("Failed to exchange Facebook long-lived token", "error", err)
		return p.redirectResult(r, "error", "Failed to exchange Facebook long-lived user token")
	}
	debugPayload, err := p.debugToken(config, longToken)
	if err != nil {
		p.app.Log.Error("Failed to debug Facebook OAuth token", "error", err)
		return p.redirectResult(r, "error", "Failed to validate Facebook user token")
	}
	tokenType := strings.ToUpper(strings.TrimSpace(debugPayload.Data.Type))
	p.app.Log.Info(
		"Facebook OAuth token debug",
		"token_type", tokenType,
		"app_id", debugPayload.Data.AppID,
		"is_valid", debugPayload.Data.IsValid,
		"expires_at", debugPayload.Data.ExpiresAt,
		"scope_count", len(debugPayload.Data.Scopes),
	)
	if tokenType != "USER" {
		return p.redirectResult(r, "error", fmt.Sprintf("Facebook returned a %s token; reconnect with a user account that manages the pages", tokenType))
	}

	userInfo, err := p.fetchUser(config, longToken)
	if err != nil {
		p.app.Log.Error("Failed to fetch Facebook OAuth profile", "error", err)
		return p.redirectResult(r, "error", "Failed to read Facebook profile")
	}
	if strings.TrimSpace(userInfo.ID) == "" {
		return p.redirectResult(r, "error", "Facebook did not return an account ID")
	}
	pages, pageTokens, err := p.fetchPages(config, longToken)
	if err != nil {
		p.app.Log.Error("Failed to fetch Facebook pages", "error", err)
		return p.redirectResult(r, "error", "Connected with a user token, but failed to fetch managed Facebook pages")
	}
	account, err := p.saveAccount(db, state, userInfo, longToken, expiresIn, pages, pageTokens)
	if err != nil {
		p.app.Log.Error("Failed to save Facebook OAuth account", "error", err)
		return p.redirectResult(r, "error", "Failed to save Facebook account")
	}

	status := "connected"
	if state.Action == "renew" {
		status = "renewed"
	}
	return p.redirectResult(r, status, account.ID.String())
}

func (p *Plugin) RenewFacebookOAuth(r *fastglue.Request) error {
	rawID, ok := r.RequestCtx.UserValue("id").(string)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid Facebook account ID", nil, "id")
	}
	accountID, err := uuid.Parse(rawID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid Facebook account ID", nil, "id")
	}
	r.RequestCtx.QueryArgs().Set("action", "renew")
	r.RequestCtx.QueryArgs().Set("account_id", accountID.String())
	return p.InitFacebookOAuth(r)
}

func (p *Plugin) authContext(r *fastglue.Request) (*gorm.DB, uuid.UUID, uuid.UUID, bool) {
	organizationID, organizationOK := middleware.GetOrganizationID(r)
	userID, userOK := middleware.GetUserID(r)
	if !organizationOK || !userOK || organizationID == uuid.Nil || userID == uuid.Nil || p.app == nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
		return nil, uuid.Nil, uuid.Nil, false
	}
	if !p.app.HasPermission(userID, models.ResourceAccounts, models.ActionWrite, organizationID) {
		_ = r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
		return nil, uuid.Nil, uuid.Nil, false
	}
	return tenant.ScopedDB(p.app.DB, organizationID), organizationID, userID, true
}

func findAccount(r *fastglue.Request, db *gorm.DB, accountID, organizationID uuid.UUID) (*models.FacebookAccount, bool) {
	var account models.FacebookAccount
	err := db.Where("id = ? AND organization_id = ?", accountID, organizationID).First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_ = r.SendErrorEnvelope(fasthttp.StatusNotFound, "Facebook account not found", nil, "")
		return nil, false
	}
	if err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load Facebook account", nil, "")
		return nil, false
	}
	return &account, true
}

func (p *Plugin) runtimeConfig(r *fastglue.Request) (runtimeConfig, error) {
	if p == nil || p.app == nil || p.app.Config == nil {
		return runtimeConfig{}, errors.New("Facebook OAuth is not configured")
	}
	config := runtimeConfig{
		AppID:       strings.TrimSpace(p.app.Config.FacebookOAuth.AppID),
		AppSecret:   strings.TrimSpace(p.app.Config.FacebookOAuth.AppSecret),
		APIVersion:  strings.TrimSpace(p.app.Config.FacebookOAuth.APIVersion),
		RedirectURI: strings.TrimSpace(p.app.Config.FacebookOAuth.RedirectURI),
		BaseURL:     strings.TrimRight(strings.TrimSpace(p.app.Config.FacebookOAuth.BaseURL), "/"),
	}
	if config.APIVersion == "" {
		config.APIVersion = "v20.0"
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://graph.facebook.com"
	}
	if config.RedirectURI == "" {
		config.RedirectURI = p.callbackURL(r)
	}
	if config.AppID == "" || config.AppSecret == "" {
		return runtimeConfig{}, errors.New("facebook_oauth.app_id and facebook_oauth.app_secret must be configured")
	}
	return config, nil
}

func (p *Plugin) callbackURL(r *fastglue.Request) string {
	scheme := "https"
	if p.app != nil && p.app.Config != nil && r != nil &&
		!r.RequestCtx.IsTLS() && p.app.Config.App.Environment == "development" {
		scheme = "http"
	}
	host := "localhost"
	if r != nil {
		host = string(r.RequestCtx.Host())
	}
	basePath := ""
	if p.app != nil && p.app.Config != nil {
		basePath = sanitizePath(p.app.Config.Server.BasePath)
	}
	return fmt.Sprintf("%s://%s%s/api/facebook/oauth/callback", scheme, host, basePath)
}

func authorizationURL(config runtimeConfig, state string) string {
	query := url.Values{
		"client_id":     {config.AppID},
		"redirect_uri":  {config.RedirectURI},
		"state":         {state},
		"response_type": {"code"},
		"scope":         {strings.Join(scopes, ",")},
		"auth_type":     {"rerequest"},
	}
	return fmt.Sprintf("https://www.facebook.com/%s/dialog/oauth?%s", config.APIVersion, query.Encode())
}

func (p *Plugin) exchangeCode(config runtimeConfig, code string) (string, error) {
	query := url.Values{
		"client_id":     {config.AppID},
		"client_secret": {config.AppSecret},
		"redirect_uri":  {config.RedirectURI},
		"code":          {code},
	}
	endpoint := fmt.Sprintf("%s/%s/oauth/access_token?%s", config.BaseURL, config.APIVersion, query.Encode())
	payload, err := p.tokenGet(endpoint)
	if err != nil {
		return "", err
	}
	p.app.Log.Info("Facebook OAuth code exchange", "token_type", payload.TokenType, "expires_in", payload.ExpiresIn)
	return payload.AccessToken, nil
}

func (p *Plugin) exchangeLongLivedToken(config runtimeConfig, shortToken string) (string, int, error) {
	query := url.Values{
		"grant_type":        {"fb_exchange_token"},
		"client_id":         {config.AppID},
		"client_secret":     {config.AppSecret},
		"fb_exchange_token": {shortToken},
	}
	endpoint := fmt.Sprintf("%s/%s/oauth/access_token?%s", config.BaseURL, config.APIVersion, query.Encode())
	payload, err := p.tokenGet(endpoint)
	if err != nil {
		return "", 0, err
	}
	expiresIn := payload.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 60 * 24 * 3600
	}
	p.app.Log.Info("Facebook OAuth long-lived token exchange", "token_type", payload.TokenType, "expires_in", expiresIn)
	return payload.AccessToken, expiresIn, nil
}

func (p *Plugin) tokenGet(endpoint string) (tokenResponse, error) {
	var payload tokenResponse
	if err := p.jsonRequest(http.MethodGet, endpoint, nil, &payload); err != nil {
		return payload, err
	}
	if payload.Error != nil {
		return payload, fmt.Errorf("Facebook token exchange error: %s", oauthErrorString(payload.Error))
	}
	if payload.AccessToken == "" {
		return payload, errors.New("Facebook token response did not include an access token")
	}
	return payload, nil
}

func (p *Plugin) debugToken(config runtimeConfig, accessToken string) (tokenDebugResponse, error) {
	query := url.Values{
		"input_token":  {accessToken},
		"access_token": {config.AppID + "|" + config.AppSecret},
	}
	endpoint := fmt.Sprintf("%s/%s/debug_token?%s", config.BaseURL, config.APIVersion, query.Encode())
	var payload tokenDebugResponse
	if err := p.jsonRequest(http.MethodGet, endpoint, nil, &payload); err != nil {
		return payload, err
	}
	if payload.Error != nil {
		return payload, fmt.Errorf("Facebook debug_token error: %s", oauthErrorString(payload.Error))
	}
	if payload.Data.Error != nil {
		return payload, fmt.Errorf("Facebook debug_token error: %s", oauthErrorString(payload.Data.Error))
	}
	if !payload.Data.IsValid {
		return payload, errors.New("Facebook debug_token reported an invalid token")
	}
	if strings.TrimSpace(payload.Data.Type) == "" {
		return payload, errors.New("Facebook debug_token response did not include a token type")
	}
	return payload, nil
}

func (p *Plugin) fetchUser(config runtimeConfig, accessToken string) (userResponse, error) {
	query := url.Values{
		"fields":       {"id,name,email,picture.width(200)"},
		"access_token": {accessToken},
	}
	endpoint := fmt.Sprintf("%s/%s/me?%s", config.BaseURL, config.APIVersion, query.Encode())
	var payload userResponse
	err := p.jsonRequest(http.MethodGet, endpoint, nil, &payload)
	return payload, err
}

func (p *Plugin) fetchPages(config runtimeConfig, accessToken string) ([]map[string]any, map[string]string, error) {
	query := url.Values{
		"fields":       {"id,name,access_token,category,picture.width(200)"},
		"access_token": {accessToken},
	}
	endpoint := fmt.Sprintf("%s/%s/me/accounts?%s", config.BaseURL, config.APIVersion, query.Encode())
	var payload pagesResponse
	if err := p.jsonRequest(http.MethodGet, endpoint, nil, &payload); err != nil {
		return nil, nil, err
	}

	pageTokens := make(map[string]string, len(payload.Data))
	safePages := make([]map[string]any, 0, len(payload.Data))
	for _, page := range payload.Data {
		pageID, _ := page["id"].(string)
		if pageID == "" {
			continue
		}
		if token, _ := page["access_token"].(string); token != "" {
			pageTokens[pageID] = token
			page["connected"] = true
		} else {
			page["connected"] = false
		}
		delete(page, "access_token")
		safePages = append(safePages, page)
	}
	return safePages, pageTokens, nil
}

func (p *Plugin) saveAccount(
	db *gorm.DB,
	state OAuthState,
	userInfo userResponse,
	accessToken string,
	expiresIn int,
	pages []map[string]any,
	pageTokens map[string]string,
) (*models.FacebookAccount, error) {
	encryptionKey := p.app.Config.App.EncryptionKey
	encryptedAccessToken, err := crypto.Encrypt(accessToken, encryptionKey)
	if err != nil {
		return nil, err
	}
	pageTokensJSON, err := json.Marshal(pageTokens)
	if err != nil {
		return nil, err
	}
	encryptedPageTokens, err := crypto.Encrypt(string(pageTokensJSON), encryptionKey)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(expiresIn) * time.Second)
	data := models.JSONB{
		"pages":        pages,
		"page_count":   len(pages),
		"raw_response": userInfo,
	}
	var account models.FacebookAccount
	query := db.Where(
		"organization_id = ? AND method = ? AND account_uid = ?",
		state.OrganizationID,
		models.FBAccountMethodOAuth,
		userInfo.ID,
	)
	if state.Action == "renew" && state.AccountID != uuid.Nil {
		query = db.Where("organization_id = ? AND id = ?", state.OrganizationID, state.AccountID)
	}
	err = query.First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		account = models.FacebookAccount{
			OrganizationID: state.OrganizationID,
			UserID:         state.UserID,
			Platform:       "facebook",
			Name:           userInfo.Name,
			AccountUID:     userInfo.ID,
			Email:          userInfo.Email,
			AvatarURL:      userInfo.Picture.Data.URL,
			Status:         models.FBAccountStatusActive,
			Method:         models.FBAccountMethodOAuth,
			AccessToken:    encryptedAccessToken,
			PageTokens:     encryptedPageTokens,
			TokenExpiresAt: &expiresAt,
			ConnectedAt:    &now,
			LastRenewedAt:  &now,
			Data:           data,
		}
		if account.Name == "" {
			account.Name = "Facebook Account"
		}
		if err := db.Create(&account).Error; err != nil {
			return nil, err
		}
		return &account, nil
	}
	if err != nil {
		return nil, err
	}

	updates := map[string]any{
		"user_id":          state.UserID,
		"platform":         "facebook",
		"name":             nonEmpty(userInfo.Name, account.Name),
		"account_uid":      userInfo.ID,
		"email":            userInfo.Email,
		"avatar_url":       userInfo.Picture.Data.URL,
		"status":           models.FBAccountStatusActive,
		"method":           models.FBAccountMethodOAuth,
		"access_token":     encryptedAccessToken,
		"page_tokens":      encryptedPageTokens,
		"token_expires_at": &expiresAt,
		"last_renewed_at":  &now,
		"data":             data,
	}
	if account.ConnectedAt == nil {
		updates["connected_at"] = &now
	}
	if err := db.Model(&account).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := db.First(&account, "id = ?", account.ID).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (p *Plugin) jsonRequest(method, endpoint string, body io.Reader, out any) error {
	request, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return err
	}
	response, err := p.app.OAuthHTTPClient().Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(responseBody, out); err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Facebook Graph API returned status %d", response.StatusCode)
	}
	return nil
}

func (p *Plugin) redirectResult(r *fastglue.Request, status, message string) error {
	basePath := ""
	if p.app != nil && p.app.Config != nil {
		basePath = sanitizePath(p.app.Config.Server.BasePath)
	}
	target := basePath + "/facebook/accounts"
	query := url.Values{"facebook_oauth": {status}}
	if strings.TrimSpace(message) != "" {
		query.Set("message", message)
	}
	r.RequestCtx.Redirect(target+"?"+query.Encode(), fasthttp.StatusFound)
	return nil
}

func randomString(length int) (string, error) {
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("crypto/rand.Read failed: %w", err)
	}
	return base64.URLEncoding.EncodeToString(buffer)[:length], nil
}

func sanitizePath(path string) string {
	if path == "" {
		return ""
	}
	if path[0] != '/' {
		path = "/" + path
	}
	for len(path) > 1 && (path[1] == '/' || path[1] == '\\') {
		path = "/" + path[2:]
	}
	return path
}

func oauthErrorString(raw any) string {
	if raw == nil {
		return "unknown error"
	}
	if rawError, ok := raw.(map[string]any); ok {
		parts := []string{}
		for _, item := range []struct {
			key    string
			prefix string
		}{
			{key: "message"},
			{key: "code", prefix: "code="},
			{key: "error_subcode", prefix: "subcode="},
			{key: "type", prefix: "type="},
		} {
			value := strings.TrimSpace(fmt.Sprint(rawError[item.key]))
			if value != "" && value != "<nil>" {
				parts = append(parts, item.prefix+value)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, ": ")
		}
	}
	return fmt.Sprint(raw)
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
