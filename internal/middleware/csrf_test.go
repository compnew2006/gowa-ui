package middleware_test

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestCSRFProtection_RefreshCookieRequiresMatchingToken(t *testing.T) {
	t.Parallel()

	req := newTestRequest()
	req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodPost)
	req.RequestCtx.Request.Header.SetCookie("whm_refresh", "refresh-token")
	req.RequestCtx.Request.Header.SetCookie("whm_csrf", "cookie-token")
	req.RequestCtx.Request.Header.Set("X-CSRF-Token", "different-token")

	result := middleware.CSRFProtection()(req)

	require.Nil(t, result)
	assert.Equal(t, fasthttp.StatusForbidden, req.RequestCtx.Response.StatusCode())
	assert.Contains(t, string(req.RequestCtx.Response.Body()), "CSRF token mismatch")
}

func TestCSRFProtection_RefreshCookieAllowsMatchingToken(t *testing.T) {
	t.Parallel()

	req := newTestRequest()
	req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodPost)
	req.RequestCtx.Request.Header.SetCookie("whm_refresh", "refresh-token")
	req.RequestCtx.Request.Header.SetCookie("whm_csrf", "cookie-token")
	req.RequestCtx.Request.Header.Set("X-CSRF-Token", "cookie-token")

	result := middleware.CSRFProtection()(req)

	require.NotNil(t, result)
	assert.Equal(t, fasthttp.StatusOK, req.RequestCtx.Response.StatusCode())
}
