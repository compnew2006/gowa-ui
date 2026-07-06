package handlers

import (
	"net"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	cookieAccessName  = "whm_access"
	cookieRefreshName = "whm_refresh"
	cookieCSRFName    = "whm_csrf"
)

var legacyAuthCookieNames = []string{
	cookieAccessName,
	cookieRefreshName,
	cookieCSRFName,
	"whm_token",
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}

	return unique
}

func authCookiePathVariants(basePath string) []string {
	paths := []string{"/", "/api", "/api/auth/refresh"}

	trimmedBasePath := strings.TrimSpace(basePath)
	trimmedBasePath = strings.TrimRight(trimmedBasePath, "/")
	if trimmedBasePath != "" {
		paths = append(paths,
			trimmedBasePath,
			trimmedBasePath+"/",
			trimmedBasePath+"/api",
			trimmedBasePath+"/api/auth/refresh",
		)
	}

	return uniqueStrings(paths)
}

func parentCookieDomain(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if strings.HasPrefix(host, "[") && strings.Contains(host, "]") {
		host = strings.Trim(host, "[]")
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	if ip := net.ParseIP(host); ip != nil {
		return ""
	}

	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return ""
	}

	return strings.Join(parts[len(parts)-2:], ".")
}

func (a *App) authCookieDomainsForResponse(r *fastglue.Request) []string {
	domains := []string{strings.TrimSpace(a.Config.Cookie.Domain)}
	if r != nil {
		domains = append(domains, parentCookieDomain(string(r.RequestCtx.Host())))
	}
	return uniqueStrings(domains)
}

func (a *App) expireAuthCookie(r *fastglue.Request, name, path, domain string, secure, httpOnly bool) {
	c := fasthttp.AcquireCookie()
	c.SetKey(name)
	c.SetValue("")
	c.SetExpire(time.Unix(1, 0))
	c.SetMaxAge(-1)
	c.SetHTTPOnly(httpOnly)
	c.SetSecure(secure)
	c.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	c.SetPath(path)
	if domain != "" {
		c.SetDomain(domain)
	}
	r.RequestCtx.Response.Header.SetCookie(c)
	fasthttp.ReleaseCookie(c)
}

func (a *App) clearAuthCookieVariants(r *fastglue.Request, names []string) {
	secure := a.Config.Cookie.Secure
	if a.Config != nil && strings.EqualFold(strings.TrimSpace(a.Config.App.Environment), "production") {
		secure = true
	}

	paths := authCookiePathVariants(a.Config.Server.BasePath)
	domains := append([]string{""}, a.authCookieDomainsForResponse(r)...)
	for _, name := range uniqueStrings(names) {
		httpOnly := name != cookieCSRFName
		for _, path := range paths {
			for _, domain := range domains {
				a.expireAuthCookie(r, name, path, domain, secure, httpOnly)
			}
		}
	}
}

// setAuthCookies sets httpOnly auth cookies and a JS-readable CSRF cookie.
func (a *App) setAuthCookies(r *fastglue.Request, accessToken string, accessTokenExpiresAt time.Time, refreshToken string) error {
	secure := a.Config.Cookie.Secure
	if a.Config != nil && strings.EqualFold(strings.TrimSpace(a.Config.App.Environment), "production") {
		secure = true
	}
	domain := a.Config.Cookie.Domain
	bp := a.Config.Server.BasePath // e.g. "/whatomate" or ""

	// Expire legacy host-only/root-scoped cookie variants before issuing fresh tokens.
	a.clearAuthCookieVariants(r, legacyAuthCookieNames)

	// Access token cookie — httpOnly, scoped to basePath/api
	ac := fasthttp.AcquireCookie()
	ac.SetKey(cookieAccessName)
	ac.SetValue(accessToken)
	ac.SetHTTPOnly(true)
	ac.SetSecure(secure)
	ac.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	ac.SetPath(bp + "/api")
	ac.SetMaxAge(accessTokenTTLSeconds(time.Now(), accessTokenExpiresAt))
	if domain != "" {
		ac.SetDomain(domain)
	}
	r.RequestCtx.Response.Header.SetCookie(ac)
	fasthttp.ReleaseCookie(ac)

	// Refresh token cookie — httpOnly, narrow path
	rc := fasthttp.AcquireCookie()
	rc.SetKey(cookieRefreshName)
	rc.SetValue(refreshToken)
	rc.SetHTTPOnly(true)
	rc.SetSecure(secure)
	rc.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	rc.SetPath(bp + "/api/auth/refresh")
	rc.SetMaxAge(a.Config.JWT.RefreshExpiryDays * 86400)
	if domain != "" {
		rc.SetDomain(domain)
	}
	r.RequestCtx.Response.Header.SetCookie(rc)
	fasthttp.ReleaseCookie(rc)

	// CSRF token cookie — NOT httpOnly (JS-readable), broad path
	csrfToken, err := generateCSRFToken()
	if err != nil {
		return err
	}
	cc := fasthttp.AcquireCookie()
	cc.SetKey(cookieCSRFName)
	cc.SetValue(csrfToken)
	cc.SetHTTPOnly(false)
	cc.SetSecure(secure)
	cc.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	cc.SetPath(bp + "/")
	cc.SetMaxAge(a.Config.JWT.RefreshExpiryDays * 86400)
	if domain != "" {
		cc.SetDomain(domain)
	}
	r.RequestCtx.Response.Header.SetCookie(cc)
	fasthttp.ReleaseCookie(cc)
	return nil
}

// clearAuthCookies expires all auth cookies.
func (a *App) clearAuthCookies(r *fastglue.Request) {
	a.clearAuthCookieVariants(r, legacyAuthCookieNames)
}
