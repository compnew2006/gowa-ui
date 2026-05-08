package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	appcrypto "github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"golang.org/x/oauth2"
)

// UserInfo represents normalized user info from OAuth providers
type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (a *App) buildOAuthConfig(provider string, ssoConfig *models.SSOProvider, r *fastglue.Request) *oauth2.Config {
	var endpoint oauth2.Endpoint
	var scopes []string

	if provider == "custom" {
		if err := a.validateCustomSSOProviderConfig(ssoConfig); err != nil {
			a.Log.Error("Invalid custom SSO provider configuration", "error", err)
			return nil
		}
		endpoint = oauth2.Endpoint{
			AuthURL:  ssoConfig.AuthURL,
			TokenURL: ssoConfig.TokenURL,
		}
		scopes = []string{"openid", "email", "profile"}
	} else {
		providerCfg := oauthProviders[provider]
		endpoint = providerCfg.Endpoint
		scopes = providerCfg.Scopes
	}

	// Build callback URL from request
	scheme := "https"
	if !r.RequestCtx.IsTLS() && a.Config.App.Environment == "development" {
		scheme = "http"
	}
	host := string(r.RequestCtx.Host())
	basePath := sanitizeRedirectPath(a.Config.Server.BasePath)
	callbackURL := fmt.Sprintf("%s://%s%s/api/auth/sso/%s/callback", scheme, host, basePath, provider)

	// Decrypt SSO client secret
	allowLegacy := true
	if a.Config != nil && a.Config.App.AllowLegacyEncryption != nil {
		allowLegacy = *a.Config.App.AllowLegacyEncryption
	}
	if !appcrypto.IsEncrypted(ssoConfig.ClientSecret) {
		a.Log.Error("SSO client secret is not encrypted", "provider", provider)
		return nil
	}
	decryptedSecret, err := appcrypto.DecryptWithPolicy(ssoConfig.ClientSecret, a.Config.App.EncryptionKey, allowLegacy)
	if err != nil {
		a.Log.Error("Failed to decrypt SSO client secret", "error", err)
		return nil
	}

	return &oauth2.Config{
		ClientID:     ssoConfig.ClientID,
		ClientSecret: decryptedSecret,
		Endpoint:     endpoint,
		Scopes:       scopes,
		RedirectURL:  callbackURL,
	}
}

func (a *App) fetchUserInfo(provider string, ssoConfig *models.SSOProvider, token *oauth2.Token) (*UserInfo, error) {
	var userInfoURL string

	if provider == "custom" {
		if err := a.validateCustomSSOProviderConfig(ssoConfig); err != nil {
			return nil, err
		}
		userInfoURL = ssoConfig.UserInfoURL
	} else {
		userInfoURL = oauthProviders[provider].UserInfoURL
	}

	req, err := http.NewRequest(http.MethodGet, userInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	if provider == "github" {
		req.Header.Set("Accept", "application/vnd.github+json")
	}

	resp, err := a.oauthHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user info request failed: %s", string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse based on provider
	var userInfo UserInfo
	var rawData map[string]any
	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, err
	}

	switch provider {
	case "google":
		userInfo.ID = getString(rawData, "id")
		userInfo.Email = getString(rawData, "email")
		userInfo.Name = getString(rawData, "name")
	case "microsoft":
		userInfo.ID = getString(rawData, "id")
		userInfo.Email = getString(rawData, "mail")
		if userInfo.Email == "" {
			userInfo.Email = getString(rawData, "userPrincipalName")
		}
		userInfo.Name = getString(rawData, "displayName")
	case "github":
		userInfo.ID = fmt.Sprintf("%v", rawData["id"])
		userInfo.Email = getString(rawData, "email")
		userInfo.Name = getString(rawData, "name")
		if userInfo.Name == "" {
			userInfo.Name = getString(rawData, "login")
		}
		// GitHub might not return email in user info, need separate API call
		if userInfo.Email == "" {
			email, err := a.fetchGitHubEmail(token)
			if err == nil {
				userInfo.Email = email
			}
		}
	case "facebook":
		userInfo.ID = getString(rawData, "id")
		userInfo.Email = getString(rawData, "email")
		userInfo.Name = getString(rawData, "name")
	default: // custom
		userInfo.ID = getString(rawData, "sub")
		if userInfo.ID == "" {
			userInfo.ID = getString(rawData, "id")
		}
		userInfo.Email = getString(rawData, "email")
		userInfo.Name = getString(rawData, "name")
		if userInfo.Name == "" {
			userInfo.Name = getString(rawData, "preferred_username")
		}
	}

	if userInfo.Email == "" {
		return nil, fmt.Errorf("email not provided by SSO provider")
	}

	return &userInfo, nil
}

func (a *App) fetchGitHubEmail(token *oauth2.Token) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := a.oauthHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	// Find primary verified email
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	// Fallback to first verified email
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no verified email found")
}

func (a *App) redirectWithError(r *fastglue.Request, message string) {
	basePath := sanitizeRedirectPath(a.Config.Server.BasePath)
	encodedMsg := url.QueryEscape(message)
	redirectURL := fmt.Sprintf("%s/login?sso_error=%s", basePath, encodedMsg)
	r.RequestCtx.Redirect(redirectURL, fasthttp.StatusTemporaryRedirect)
}

// sanitizeRedirectPath ensures the path is safe for redirects by preventing
// open redirect vulnerabilities (e.g., //evil.com or /\evil.com)
func sanitizeRedirectPath(path string) string {
	if path == "" {
		return ""
	}
	// Ensure path starts with /
	if path[0] != '/' {
		path = "/" + path
	}
	// Prevent protocol-relative URLs (//...) and backslash escapes (/\...)
	// by stripping dangerous characters after the leading slash
	for len(path) > 1 && (path[1] == '/' || path[1] == '\\') {
		path = "/" + path[2:]
	}
	return path
}

func getString(data map[string]any, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
