package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"golang.org/x/oauth2"
)

const ssoStateCookieName = "whm_sso_state"

func (a *App) allowLocalSSOEndpoints() bool {
	if a == nil || a.Config == nil {
		return false
	}

	environment := strings.ToLower(strings.TrimSpace(a.Config.App.Environment))
	return environment == "development" || environment == "test"
}

func isPrivateOrLocalIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

func (a *App) validateCustomSSOEndpoint(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", fmt.Errorf("must not be empty")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("must be a valid absolute URL")
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("must use http or https")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("must not include userinfo")
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return "", fmt.Errorf("must include a host")
	}

	if !a.allowLocalSSOEndpoints() {
		if host == "localhost" || strings.HasSuffix(host, ".localhost") {
			return "", fmt.Errorf("must use a public host")
		}

		if ip := net.ParseIP(host); ip != nil && isPrivateOrLocalIP(ip) {
			return "", fmt.Errorf("must use a public host")
		}
	}

	parsed.Fragment = ""
	return parsed.String(), nil
}

func (a *App) validateCustomSSOProviderConfig(ssoConfig *models.SSOProvider) error {
	if ssoConfig == nil {
		return fmt.Errorf("missing SSO provider config")
	}

	authURL, err := a.validateCustomSSOEndpoint(ssoConfig.AuthURL)
	if err != nil {
		return fmt.Errorf("invalid auth URL: %w", err)
	}
	tokenURL, err := a.validateCustomSSOEndpoint(ssoConfig.TokenURL)
	if err != nil {
		return fmt.Errorf("invalid token URL: %w", err)
	}
	userInfoURL, err := a.validateCustomSSOEndpoint(ssoConfig.UserInfoURL)
	if err != nil {
		return fmt.Errorf("invalid user info URL: %w", err)
	}

	ssoConfig.AuthURL = authURL
	ssoConfig.TokenURL = tokenURL
	ssoConfig.UserInfoURL = userInfoURL
	return nil
}

func (a *App) oauthHTTPClient() *http.Client {
	if a != nil && a.HTTPClient != nil {
		return a.HTTPClient
	}

	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: SSRFSafeDialer(),
		},
	}
}

func (a *App) OAuthHTTPClient() *http.Client {
	return a.oauthHTTPClient()
}

func (a *App) oauthContext() context.Context {
	return context.WithValue(context.Background(), oauth2.HTTPClient, a.oauthHTTPClient())
}

func (a *App) setSSOStateCookie(r *fastglue.Request, browserToken string) {
	secure := false
	domain := ""
	basePath := ""
	if a != nil && a.Config != nil {
		secure = a.Config.Cookie.Secure
		domain = a.Config.Cookie.Domain
		basePath = sanitizeRedirectPath(a.Config.Server.BasePath)
		if strings.EqualFold(strings.TrimSpace(a.Config.App.Environment), "production") {
			secure = true
		}
	}

	cookie := fasthttp.AcquireCookie()
	cookie.SetKey(ssoStateCookieName)
	cookie.SetValue(browserToken)
	cookie.SetHTTPOnly(true)
	cookie.SetSecure(secure)
	cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	cookie.SetPath(basePath + "/api/auth/sso")
	cookie.SetMaxAge(int((5 * time.Minute).Seconds()))
	if domain != "" {
		cookie.SetDomain(domain)
	}
	r.RequestCtx.Response.Header.SetCookie(cookie)
	fasthttp.ReleaseCookie(cookie)
}

func (a *App) clearSSOStateCookie(r *fastglue.Request) {
	secure := false
	domain := ""
	basePath := ""
	if a != nil && a.Config != nil {
		secure = a.Config.Cookie.Secure
		domain = a.Config.Cookie.Domain
		basePath = sanitizeRedirectPath(a.Config.Server.BasePath)
		if strings.EqualFold(strings.TrimSpace(a.Config.App.Environment), "production") {
			secure = true
		}
	}

	cookie := fasthttp.AcquireCookie()
	cookie.SetKey(ssoStateCookieName)
	cookie.SetValue("")
	cookie.SetHTTPOnly(true)
	cookie.SetSecure(secure)
	cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	cookie.SetPath(basePath + "/api/auth/sso")
	cookie.SetMaxAge(-1)
	if domain != "" {
		cookie.SetDomain(domain)
	}
	r.RequestCtx.Response.Header.SetCookie(cookie)
	fasthttp.ReleaseCookie(cookie)
}

func (a *App) ssoStateCookieValue(r *fastglue.Request) string {
	return strings.TrimSpace(string(r.RequestCtx.Request.Header.Cookie(ssoStateCookieName)))
}

func generatePKCEVerifier() (string, error) {
	return generateRandomString(64)
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
