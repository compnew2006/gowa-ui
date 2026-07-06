package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	facebookgraph "github.com/compnew2006/whatomate/plugin/facebook-core/graph"
	"github.com/zerodha/fastglue"
)

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

// withFacebookPageManagement is a consolidation helper for connect, disconnect, and remove page handlers.
// It handles: auth context resolution, page token decryption, running the mutation action,
// page existence validation, marking connectedness, persistence, and response.

func (a *App) facebookGraphFormPost(endpoint string, form url.Values) (map[string]any, error) {
	return facebookgraph.New(a.OAuthHTTPClient()).FormPost(endpoint, form)
}

func (a *App) facebookGraphJSONPost(endpoint string, body map[string]any) (map[string]any, error) {
	return facebookgraph.New(a.OAuthHTTPClient()).JSONPost(endpoint, body)
}

func (a *App) facebookJSONRequest(method, endpoint string, body io.Reader, out any) error {
	return facebookgraph.New(a.OAuthHTTPClient()).JSONRequest(method, endpoint, body, out)
}
