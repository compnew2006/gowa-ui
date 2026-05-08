package middleware

import (
	"crypto/subtle"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func CSRFProtection() fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		method := string(r.RequestCtx.Method())

		if method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
			return r
		}

		if len(r.RequestCtx.Request.Header.Peek("Authorization")) > 0 ||
			len(r.RequestCtx.Request.Header.Peek("X-API-Key")) > 0 {
			return r
		}

		accessCookie := r.RequestCtx.Request.Header.Cookie("whm_access")
		refreshCookie := r.RequestCtx.Request.Header.Cookie("whm_refresh")
		if len(accessCookie) == 0 && len(refreshCookie) == 0 {
			return r
		}

		csrfCookie := string(r.RequestCtx.Request.Header.Cookie("whm_csrf"))
		csrfHeader := string(r.RequestCtx.Request.Header.Peek("X-CSRF-Token"))

		if csrfCookie == "" || csrfHeader == "" || subtle.ConstantTimeCompare([]byte(csrfCookie), []byte(csrfHeader)) != 1 {
			r.RequestCtx.SetStatusCode(fasthttp.StatusForbidden)
			r.RequestCtx.SetContentType("application/json")
			r.RequestCtx.SetBodyString(`{"status":"error","message":"CSRF token mismatch"}`)
			return nil
		}

		return r
	}
}
