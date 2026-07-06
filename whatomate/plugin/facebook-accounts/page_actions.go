package facebookaccounts

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"

	"github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/models"
	facebookgraph "github.com/compnew2006/whatomate/plugin/facebook-core/graph"
)

type postPageRequest struct {
	Message  string `json:"message"`
	ImageURL string `json:"image_url"`
}

type sendPageMessageRequest struct {
	RecipientID string `json:"recipient_id"`
	Message     string `json:"message"`
}

type graphRuntimeConfig struct {
	APIVersion string
	BaseURL    string
}

func (p *Plugin) PostFacebookPage(r *fastglue.Request) error {
	account, pageID, pageToken, ok := p.pageOperationContext(r, models.ActionWrite)
	if !ok {
		return nil
	}

	var request postPageRequest
	if err := r.Decode(&request, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	request.Message = strings.TrimSpace(request.Message)
	request.ImageURL = strings.TrimSpace(request.ImageURL)
	if request.Message == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "message is required", nil, "message")
	}

	cfg, err := p.graphRuntimeConfig()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	endpoint := fmt.Sprintf("%s/%s/%s/feed", cfg.BaseURL, cfg.APIVersion, url.PathEscape(pageID))
	form := url.Values{"message": {request.Message}, "access_token": {pageToken}}
	if request.ImageURL != "" {
		endpoint = fmt.Sprintf("%s/%s/%s/photos", cfg.BaseURL, cfg.APIVersion, url.PathEscape(pageID))
		form.Set("url", request.ImageURL)
		form.Set("caption", request.Message)
	}

	payload, err := facebookgraph.New(p.app.OAuthHTTPClient()).FormPost(endpoint, form)
	if err != nil {
		return p.graphError(r, err, account.ID)
	}
	return r.SendEnvelope(payload)
}

func (p *Plugin) GetFacebookPageInsights(r *fastglue.Request) error {
	account, pageID, pageToken, ok := p.pageOperationContext(r, models.ActionRead)
	if !ok {
		return nil
	}

	metric := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("metric")))
	if metric == "" {
		metric = "page_impressions"
	}
	days := 30
	if rawDays := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("days"))); rawDays != "" {
		parsed, err := strconv.Atoi(rawDays)
		if err != nil || parsed < 1 || parsed > 90 {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "days must be between 1 and 90", nil, "days")
		}
		days = parsed
	}

	cfg, err := p.graphRuntimeConfig()
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
	endpoint := fmt.Sprintf("%s/%s/%s/insights?%s", cfg.BaseURL, cfg.APIVersion, url.PathEscape(pageID), query.Encode())
	payload, err := facebookgraph.New(p.app.OAuthHTTPClient()).Get(endpoint)
	if err != nil {
		return p.graphError(r, err, account.ID)
	}
	return r.SendEnvelope(payload)
}

func (p *Plugin) SendFacebookPageMessage(r *fastglue.Request) error {
	account, pageID, pageToken, ok := p.pageOperationContext(r, models.ActionWrite)
	if !ok {
		return nil
	}

	var request sendPageMessageRequest
	if err := r.Decode(&request, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	request.RecipientID = strings.TrimSpace(request.RecipientID)
	request.Message = strings.TrimSpace(request.Message)
	if request.RecipientID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "recipient_id is required", nil, "recipient_id")
	}
	if request.Message == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "message is required", nil, "message")
	}

	cfg, err := p.graphRuntimeConfig()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	endpoint := fmt.Sprintf("%s/%s/%s/messages", cfg.BaseURL, cfg.APIVersion, url.PathEscape(pageID))
	body := map[string]any{
		"recipient":    map[string]string{"id": request.RecipientID},
		"message":      map[string]string{"text": request.Message},
		"access_token": pageToken,
	}
	payload, err := facebookgraph.New(p.app.OAuthHTTPClient()).JSONPost(endpoint, body)
	if err != nil {
		return p.graphError(r, err, account.ID)
	}
	return r.SendEnvelope(payload)
}

func (p *Plugin) pageOperationContext(r *fastglue.Request, action string) (*models.FacebookAccount, string, string, bool) {
	requestDB, organizationID, ok := p.accountContext(r, action)
	if !ok {
		return nil, "", "", false
	}

	pageID := strings.TrimSpace(fmt.Sprint(r.RequestCtx.UserValue("page_id")))
	if pageID == "" {
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "page_id is required", nil, "page_id")
		return nil, "", "", false
	}
	account, ok := findAccount(r, requestDB, organizationID)
	if !ok {
		return nil, "", "", false
	}

	pageTokens, err := p.decryptPageTokens(*account)
	if err != nil {
		p.app.Log.Error("Failed to decrypt Facebook page tokens", "error", err, "account_id", account.ID)
		_ = r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read Facebook page token", nil, "")
		return nil, "", "", false
	}
	pageToken := pageTokens[pageID]
	if pageToken == "" {
		_ = r.SendErrorEnvelope(fasthttp.StatusNotFound, "Page token not found for this account", nil, "page_id")
		return nil, "", "", false
	}
	return account, pageID, pageToken, true
}

func (p *Plugin) decryptPageTokens(account models.FacebookAccount) (map[string]string, error) {
	if account.PageTokens == "" {
		return map[string]string{}, nil
	}
	decrypted, err := crypto.Decrypt(account.PageTokens, p.app.Config.App.EncryptionKey)
	if err != nil {
		return nil, err
	}
	var pageTokens map[string]string
	if err := json.Unmarshal([]byte(decrypted), &pageTokens); err != nil {
		return nil, err
	}
	return pageTokens, nil
}

func (p *Plugin) graphRuntimeConfig() (graphRuntimeConfig, error) {
	if p == nil || p.app == nil || p.app.Config == nil {
		return graphRuntimeConfig{}, errors.New("Facebook OAuth is not configured")
	}
	cfg := graphRuntimeConfig{
		APIVersion: strings.TrimSpace(p.app.Config.FacebookOAuth.APIVersion),
		BaseURL:    strings.TrimRight(strings.TrimSpace(p.app.Config.FacebookOAuth.BaseURL), "/"),
	}
	if cfg.APIVersion == "" {
		cfg.APIVersion = "v20.0"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://graph.facebook.com"
	}
	if strings.TrimSpace(p.app.Config.FacebookOAuth.AppID) == "" || strings.TrimSpace(p.app.Config.FacebookOAuth.AppSecret) == "" {
		return graphRuntimeConfig{}, errors.New("facebook_oauth.app_id and facebook_oauth.app_secret must be configured")
	}
	return cfg, nil
}

func (p *Plugin) graphError(r *fastglue.Request, err error, accountID uuid.UUID) error {
	p.app.Log.Error("Facebook Graph API request failed", "error", err, "account_id", accountID)
	return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Facebook Graph API request failed", nil, "")
}
