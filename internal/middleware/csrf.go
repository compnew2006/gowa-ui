package middleware

import (
	"strings"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// CSRFProtection returns a middleware that validates CSRF tokens on mutating
// requests that use cookie-based authentication.
func CSRFProtection() fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		method := string(r.RequestCtx.Method())

		// Only validate mutating methods.
		if method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
			return r
		}

		// Skip the auth endpoints that establish/recover a session. These MUST
		// work when the browser still holds a stale access cookie (e.g. after a
		// server restart), otherwise the access-recovery flow (token refresh)
		// itself 403s on CSRF and the user can never recover — every page load
		// then floods the console with 401s instead of redirecting to login.
		path := string(r.RequestCtx.Path())
		if path == "/api/auth/login" || path == "/api/auth/register" ||
			path == "/api/auth/refresh" || path == "/api/auth/logout" ||
			strings.HasPrefix(path, "/api/auth/sso/") {
			return r
		}

		// Skip if the request uses header-based auth (API key or Bearer token).
		// These are not automatically attached by the browser, so CSRF is not a concern.
		if len(r.RequestCtx.Request.Header.Peek("Authorization")) > 0 ||
			len(r.RequestCtx.Request.Header.Peek("X-API-Key")) > 0 {
			return r
		}

		// Skip if there is no access cookie — the auth middleware will reject
		// the request with 401 anyway.
		cookieVal := r.RequestCtx.Request.Header.Cookie("whm_access")
		if len(cookieVal) == 0 {
			return r
		}

		// Double-submit: compare whm_csrf cookie with X-CSRF-Token header.
		csrfCookie := string(r.RequestCtx.Request.Header.Cookie("whm_csrf"))
		csrfHeader := string(r.RequestCtx.Request.Header.Peek("X-CSRF-Token"))

		if csrfCookie == "" || csrfHeader == "" || csrfCookie != csrfHeader {
			r.RequestCtx.SetStatusCode(fasthttp.StatusForbidden)
			r.RequestCtx.SetContentType("application/json")
			r.RequestCtx.SetBodyString(`{"status":"error","message":"CSRF token mismatch"}`)
			return nil
		}

		return r
	}
}
