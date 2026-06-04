package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

const facebookOAuthDefaultAction = "connect"

var facebookOAuthScopes = []string{
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

type facebookOAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       any    `json:"error"`
}

type facebookOAuthUserResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
}

type facebookOAuthPagesResponse struct {
	Data []map[string]any `json:"data"`
}

type facebookPostPageRequest struct {
	Message  string `json:"message"`
	ImageURL string `json:"image_url"`
}

type facebookSendPageMessageRequest struct {
	RecipientID string `json:"recipient_id"`
	Message     string `json:"message"`
}

func (a *App) InitFacebookOAuth(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionWrite); err != nil {
		return nil
	}

	action := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("action")))
	if action == "" {
		action = facebookOAuthDefaultAction
	}
	if action != "connect" && action != "renew" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid OAuth action", nil, "action")
	}

	var accountID uuid.UUID
	if action == "renew" {
		rawAccountID := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("account_id")))
		if rawAccountID == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "account_id is required for renewal", nil, "account_id")
		}
		accountID, err = uuid.Parse(rawAccountID)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid account_id", nil, "account_id")
		}
		if _, err := findByIDAndOrg[models.FacebookAccount](requestDB, r, accountID, orgID, "Facebook account"); err != nil {
			return nil
		}
	}

	oauthCfg, err := a.facebookOAuthRuntimeConfig(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	stateToken, err := generateRandomString(64)
	if err != nil {
		a.Log.Error("Failed to generate Facebook OAuth state", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initiate Facebook OAuth", nil, "")
	}

	expiresAt := time.Now().Add(10 * time.Minute)
	state := models.FacebookOAuthState{
		OrganizationID: orgID,
		UserID:         userID,
		AccountID:      accountID,
		StateToken:     stateToken,
		Action:         action,
		ExpiresAt:      expiresAt,
	}
	if err := requestDB.Create(&state).Error; err != nil {
		a.Log.Error("Failed to store Facebook OAuth state", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initiate Facebook OAuth", nil, "")
	}

	authURL := facebookOAuthAuthURL(oauthCfg, stateToken)
	if string(r.RequestCtx.QueryArgs().Peek("redirect")) == "1" {
		r.RequestCtx.Redirect(authURL, fasthttp.StatusTemporaryRedirect)
		return nil
	}

	return r.SendEnvelope(map[string]any{
		"auth_url":   authURL,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func (a *App) CallbackFacebookOAuth(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	stateDB := requestDB.Session(&gorm.Session{})
	code := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("code")))
	stateToken := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("state")))
	errorParam := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("error")))

	if errorParam != "" {
		return a.redirectFacebookOAuthResult(r, "error", string(r.RequestCtx.QueryArgs().Peek("error_description")))
	}
	if code == "" || stateToken == "" {
		return a.redirectFacebookOAuthResult(r, "error", "Invalid callback parameters")
	}

	var state models.FacebookOAuthState
	err := stateDB.Where("state_token = ?", stateToken).First(&state).Error
	if err != nil {
		return a.redirectFacebookOAuthResult(r, "error", "Invalid or expired OAuth state")
	}
	if err := requestDB.Session(&gorm.Session{}).Delete(&state).Error; err != nil {
		a.Log.Warn("Failed to delete Facebook OAuth state", "error", err)
	}
	if time.Now().After(state.ExpiresAt) {
		return a.redirectFacebookOAuthResult(r, "error", "Invalid or expired OAuth state")
	}

	oauthCfg, err := a.facebookOAuthRuntimeConfig(r)
	if err != nil {
		return a.redirectFacebookOAuthResult(r, "error", err.Error())
	}

	shortToken, err := a.exchangeFacebookOAuthCode(oauthCfg, code)
	if err != nil {
		a.Log.Error("Failed to exchange Facebook OAuth code", "error", err)
		return a.redirectFacebookOAuthResult(r, "error", "Failed to connect Facebook account")
	}

	longToken, expiresIn, err := a.exchangeFacebookLongLivedToken(oauthCfg, shortToken)
	if err != nil {
		a.Log.Error("Failed to exchange Facebook long-lived token", "error", err)
		longToken = shortToken
		expiresIn = 60 * 24 * 3600
	}

	userInfo, err := a.fetchFacebookOAuthUser(oauthCfg, longToken)
	if err != nil {
		a.Log.Error("Failed to fetch Facebook OAuth profile", "error", err)
		return a.redirectFacebookOAuthResult(r, "error", "Failed to read Facebook profile")
	}
	if strings.TrimSpace(userInfo.ID) == "" {
		return a.redirectFacebookOAuthResult(r, "error", "Facebook did not return an account ID")
	}

	pages, pageTokens, err := a.fetchFacebookOAuthPages(oauthCfg, longToken)
	if err != nil {
		a.Log.Error("Failed to fetch Facebook pages", "error", err)
		pages = []map[string]any{}
		pageTokens = map[string]string{}
	}

	account, err := a.saveFacebookOAuthAccount(requestDB.Session(&gorm.Session{}), state, userInfo, longToken, expiresIn, pages, pageTokens)
	if err != nil {
		a.Log.Error("Failed to save Facebook OAuth account", "error", err)
		return a.redirectFacebookOAuthResult(r, "error", "Failed to save Facebook account")
	}

	status := "connected"
	if state.Action == "renew" {
		status = "renewed"
	}
	return a.redirectFacebookOAuthResult(r, status, account.ID.String())
}

func (a *App) RenewFacebookOAuth(r *fastglue.Request) error {
	id, err := parsePathUUID(r, "id", "Facebook account")
	if err != nil {
		return nil
	}
	r.RequestCtx.QueryArgs().Set("action", "renew")
	r.RequestCtx.QueryArgs().Set("account_id", id.String())
	return a.InitFacebookOAuth(r)
}

func (a *App) PostFacebookPage(r *fastglue.Request) error {
	account, pageID, pageToken, err := a.facebookPageOperationContext(r, models.ActionWrite)
	if err != nil {
		return err
	}
	if account == nil {
		return nil
	}

	var req facebookPostPageRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	req.Message = strings.TrimSpace(req.Message)
	req.ImageURL = strings.TrimSpace(req.ImageURL)
	if req.Message == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "message is required", nil, "message")
	}

	oauthCfg, err := a.facebookOAuthRuntimeConfig(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	endpoint := fmt.Sprintf("%s/%s/%s/feed", oauthCfg.BaseURL, oauthCfg.APIVersion, url.PathEscape(pageID))
	form := url.Values{"message": {req.Message}, "access_token": {pageToken}}
	if req.ImageURL != "" {
		endpoint = fmt.Sprintf("%s/%s/%s/photos", oauthCfg.BaseURL, oauthCfg.APIVersion, url.PathEscape(pageID))
		form.Set("url", req.ImageURL)
		form.Set("caption", req.Message)
	}

	payload, err := a.facebookGraphFormPost(endpoint, form)
	if err != nil {
		return a.facebookGraphError(r, err, account.ID)
	}
	return r.SendEnvelope(payload)
}

func (a *App) GetFacebookPageInsights(r *fastglue.Request) error {
	account, pageID, pageToken, err := a.facebookPageOperationContext(r, models.ActionRead)
	if err != nil {
		return err
	}
	if account == nil {
		return nil
	}

	metric := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("metric")))
	if metric == "" {
		metric = "page_impressions"
	}
	days := 30
	if rawDays := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("days"))); rawDays != "" {
		parsed, parseErr := strconv.Atoi(rawDays)
		if parseErr != nil || parsed < 1 || parsed > 90 {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "days must be between 1 and 90", nil, "days")
		}
		days = parsed
	}

	oauthCfg, err := a.facebookOAuthRuntimeConfig(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	now := time.Now()
	query := url.Values{
		"metric":       {metric},
		"period":       {"day"},
		"since":        {strconv.FormatInt(now.AddDate(0, 0, -days).Unix(), 10)},
		"until":        {strconv.FormatInt(now.Unix(), 10)},
		"access_token": {pageToken},
	}
	endpoint := fmt.Sprintf("%s/%s/%s/insights?%s", oauthCfg.BaseURL, oauthCfg.APIVersion, url.PathEscape(pageID), query.Encode())
	payload, err := a.facebookGraphGet(endpoint)
	if err != nil {
		return a.facebookGraphError(r, err, account.ID)
	}
	return r.SendEnvelope(payload)
}

func (a *App) SendFacebookPageMessage(r *fastglue.Request) error {
	account, pageID, pageToken, err := a.facebookPageOperationContext(r, models.ActionWrite)
	if err != nil {
		return err
	}
	if account == nil {
		return nil
	}

	var req facebookSendPageMessageRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	req.RecipientID = strings.TrimSpace(req.RecipientID)
	req.Message = strings.TrimSpace(req.Message)
	if req.RecipientID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "recipient_id is required", nil, "recipient_id")
	}
	if req.Message == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "message is required", nil, "message")
	}

	oauthCfg, err := a.facebookOAuthRuntimeConfig(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	endpoint := fmt.Sprintf("%s/%s/%s/messages", oauthCfg.BaseURL, oauthCfg.APIVersion, url.PathEscape(pageID))
	body := map[string]any{
		"recipient":    map[string]string{"id": req.RecipientID},
		"message":      map[string]string{"text": req.Message},
		"access_token": pageToken,
	}
	payload, err := a.facebookGraphJSONPost(endpoint, body)
	if err != nil {
		return a.facebookGraphError(r, err, account.ID)
	}
	return r.SendEnvelope(payload)
}

type facebookOAuthRuntimeConfig struct {
	AppID       string
	AppSecret   string
	APIVersion  string
	RedirectURI string
	BaseURL     string
}

func (a *App) facebookOAuthRuntimeConfig(r *fastglue.Request) (facebookOAuthRuntimeConfig, error) {
	if a == nil || a.Config == nil {
		return facebookOAuthRuntimeConfig{}, errors.New("Facebook OAuth is not configured")
	}

	cfg := facebookOAuthRuntimeConfig{
		AppID:       strings.TrimSpace(a.Config.FacebookOAuth.AppID),
		AppSecret:   strings.TrimSpace(a.Config.FacebookOAuth.AppSecret),
		APIVersion:  strings.TrimSpace(a.Config.FacebookOAuth.APIVersion),
		RedirectURI: strings.TrimSpace(a.Config.FacebookOAuth.RedirectURI),
		BaseURL:     strings.TrimRight(strings.TrimSpace(a.Config.FacebookOAuth.BaseURL), "/"),
	}
	if cfg.APIVersion == "" {
		cfg.APIVersion = "v20.0"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://graph.facebook.com"
	}
	if cfg.RedirectURI == "" {
		cfg.RedirectURI = a.facebookOAuthCallbackURL(r)
	}
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return facebookOAuthRuntimeConfig{}, errors.New("facebook_oauth.app_id and facebook_oauth.app_secret must be configured")
	}
	return cfg, nil
}

func (a *App) facebookOAuthCallbackURL(r *fastglue.Request) string {
	scheme := "https"
	if a != nil && a.Config != nil && r != nil && !r.RequestCtx.IsTLS() && a.Config.App.Environment == "development" {
		scheme = "http"
	}
	host := "localhost"
	if r != nil {
		host = string(r.RequestCtx.Host())
	}
	basePath := ""
	if a != nil && a.Config != nil {
		basePath = sanitizeRedirectPath(a.Config.Server.BasePath)
	}
	return fmt.Sprintf("%s://%s%s/api/facebook/oauth/callback", scheme, host, basePath)
}

func facebookOAuthAuthURL(cfg facebookOAuthRuntimeConfig, state string) string {
	query := url.Values{
		"client_id":     {cfg.AppID},
		"redirect_uri":  {cfg.RedirectURI},
		"state":         {state},
		"response_type": {"code"},
		"scope":         {strings.Join(facebookOAuthScopes, ",")},
	}
	return fmt.Sprintf("https://www.facebook.com/%s/dialog/oauth?%s", cfg.APIVersion, query.Encode())
}

func (a *App) exchangeFacebookOAuthCode(cfg facebookOAuthRuntimeConfig, code string) (string, error) {
	query := url.Values{
		"client_id":     {cfg.AppID},
		"client_secret": {cfg.AppSecret},
		"redirect_uri":  {cfg.RedirectURI},
		"code":          {code},
	}
	endpoint := fmt.Sprintf("%s/%s/oauth/access_token?%s", cfg.BaseURL, cfg.APIVersion, query.Encode())
	payload, err := a.facebookTokenGet(endpoint)
	if err != nil {
		return "", err
	}
	return payload.AccessToken, nil
}

func (a *App) exchangeFacebookLongLivedToken(cfg facebookOAuthRuntimeConfig, shortToken string) (string, int, error) {
	query := url.Values{
		"grant_type":        {"fb_exchange_token"},
		"client_id":         {cfg.AppID},
		"client_secret":     {cfg.AppSecret},
		"fb_exchange_token": {shortToken},
	}
	endpoint := fmt.Sprintf("%s/%s/oauth/access_token?%s", cfg.BaseURL, cfg.APIVersion, query.Encode())
	payload, err := a.facebookTokenGet(endpoint)
	if err != nil {
		return "", 0, err
	}
	expiresIn := payload.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 60 * 24 * 3600
	}
	return payload.AccessToken, expiresIn, nil
}

func (a *App) facebookTokenGet(endpoint string) (facebookOAuthTokenResponse, error) {
	var payload facebookOAuthTokenResponse
	if err := a.facebookJSONRequest(http.MethodGet, endpoint, nil, &payload); err != nil {
		return payload, err
	}
	if payload.AccessToken == "" {
		return payload, errors.New("Facebook token response did not include an access token")
	}
	return payload, nil
}

func (a *App) fetchFacebookOAuthUser(cfg facebookOAuthRuntimeConfig, accessToken string) (facebookOAuthUserResponse, error) {
	query := url.Values{
		"fields":       {"id,name,email,picture.width(200)"},
		"access_token": {accessToken},
	}
	endpoint := fmt.Sprintf("%s/%s/me?%s", cfg.BaseURL, cfg.APIVersion, query.Encode())
	var payload facebookOAuthUserResponse
	err := a.facebookJSONRequest(http.MethodGet, endpoint, nil, &payload)
	return payload, err
}

func (a *App) fetchFacebookOAuthPages(cfg facebookOAuthRuntimeConfig, accessToken string) ([]map[string]any, map[string]string, error) {
	query := url.Values{
		"fields":       {"id,name,access_token,category,picture.width(200)"},
		"access_token": {accessToken},
	}
	endpoint := fmt.Sprintf("%s/%s/me/accounts?%s", cfg.BaseURL, cfg.APIVersion, query.Encode())
	var payload facebookOAuthPagesResponse
	if err := a.facebookJSONRequest(http.MethodGet, endpoint, nil, &payload); err != nil {
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
		}
		delete(page, "access_token")
		safePages = append(safePages, page)
	}
	return safePages, pageTokens, nil
}

func (a *App) saveFacebookOAuthAccount(db *gorm.DB, state models.FacebookOAuthState, userInfo facebookOAuthUserResponse, accessToken string, expiresIn int, pages []map[string]any, pageTokens map[string]string) (*models.FacebookAccount, error) {
	encKey := a.Config.App.EncryptionKey
	encAccessToken, err := crypto.Encrypt(accessToken, encKey)
	if err != nil {
		return nil, err
	}
	pageTokensJSON, err := json.Marshal(pageTokens)
	if err != nil {
		return nil, err
	}
	encPageTokens, err := crypto.Encrypt(string(pageTokensJSON), encKey)
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
	query := db.Where("organization_id = ? AND method = ? AND account_uid = ?", state.OrganizationID, models.FBAccountMethodOAuth, userInfo.ID)
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
			AccessToken:    encAccessToken,
			PageTokens:     encPageTokens,
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
		"access_token":     encAccessToken,
		"page_tokens":      encPageTokens,
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

func (a *App) facebookPageOperationContext(r *fastglue.Request, action string) (*models.FacebookAccount, string, string, error) {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return nil, "", "", r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, action); err != nil {
		return nil, "", "", err
	}

	accountID, err := parsePathUUID(r, "id", "Facebook account")
	if err != nil {
		return nil, "", "", nil
	}
	pageID := strings.TrimSpace(fmt.Sprint(r.RequestCtx.UserValue("page_id")))
	if pageID == "" {
		return nil, "", "", r.SendErrorEnvelope(fasthttp.StatusBadRequest, "page_id is required", nil, "page_id")
	}

	account, err := findByIDAndOrg[models.FacebookAccount](requestDB, r, accountID, orgID, "Facebook account")
	if err != nil {
		return nil, "", "", nil
	}
	pageTokens, err := a.decryptFacebookPageTokens(*account)
	if err != nil {
		a.Log.Error("Failed to decrypt Facebook page tokens", "error", err, "account_id", account.ID)
		return nil, "", "", r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read Facebook page token", nil, "")
	}
	pageToken := pageTokens[pageID]
	if pageToken == "" {
		return nil, "", "", r.SendErrorEnvelope(fasthttp.StatusNotFound, "Page token not found for this account", nil, "page_id")
	}
	return account, pageID, pageToken, nil
}

func (a *App) decryptFacebookPageTokens(account models.FacebookAccount) (map[string]string, error) {
	if account.PageTokens == "" {
		return map[string]string{}, nil
	}
	decrypted, err := crypto.Decrypt(account.PageTokens, a.Config.App.EncryptionKey)
	if err != nil {
		return nil, err
	}
	var pageTokens map[string]string
	if err := json.Unmarshal([]byte(decrypted), &pageTokens); err != nil {
		return nil, err
	}
	return pageTokens, nil
}

func (a *App) facebookGraphGet(endpoint string) (map[string]any, error) {
	var payload map[string]any
	err := a.facebookJSONRequest(http.MethodGet, endpoint, nil, &payload)
	return payload, err
}

func (a *App) facebookGraphFormPost(endpoint string, form url.Values) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return a.facebookGraphRequest(req)
}

func (a *App) facebookGraphJSONPost(endpoint string, body map[string]any) (map[string]any, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return a.facebookGraphRequest(req)
}

func (a *App) facebookGraphRequest(req *http.Request) (map[string]any, error) {
	resp, err := a.oauthHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return payload, facebookGraphResponseError(resp.StatusCode, payload)
	}
	return payload, nil
}

func facebookGraphResponseError(statusCode int, payload map[string]any) error {
	if rawErr, ok := payload["error"].(map[string]any); ok {
		parts := []string{fmt.Sprintf("Facebook Graph API returned status %d", statusCode)}
		if message := strings.TrimSpace(fmt.Sprint(rawErr["message"])); message != "" && message != "<nil>" {
			parts = append(parts, message)
		}
		if code := strings.TrimSpace(fmt.Sprint(rawErr["code"])); code != "" && code != "<nil>" {
			parts = append(parts, "code="+code)
		}
		if subcode := strings.TrimSpace(fmt.Sprint(rawErr["error_subcode"])); subcode != "" && subcode != "<nil>" {
			parts = append(parts, "subcode="+subcode)
		}
		if typ := strings.TrimSpace(fmt.Sprint(rawErr["type"])); typ != "" && typ != "<nil>" {
			parts = append(parts, "type="+typ)
		}
		return errors.New(strings.Join(parts, ": "))
	}
	return fmt.Errorf("Facebook Graph API returned status %d", statusCode)
}

func (a *App) facebookJSONRequest(method, endpoint string, body io.Reader, out any) error {
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return err
	}
	resp, err := a.oauthHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Facebook Graph API returned status %d", resp.StatusCode)
	}
	return nil
}

func (a *App) facebookGraphError(r *fastglue.Request, err error, accountID uuid.UUID) error {
	a.Log.Error("Facebook Graph API request failed", "error", err, "account_id", accountID)
	return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Facebook Graph API request failed", nil, "")
}

func (a *App) redirectFacebookOAuthResult(r *fastglue.Request, status, message string) error {
	basePath := ""
	if a != nil && a.Config != nil {
		basePath = sanitizeRedirectPath(a.Config.Server.BasePath)
	}
	target := basePath + "/facebook/accounts"
	query := url.Values{"facebook_oauth": {status}}
	if strings.TrimSpace(message) != "" {
		query.Set("message", message)
	}
	r.RequestCtx.Redirect(target+"?"+query.Encode(), fasthttp.StatusFound)
	return nil
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
