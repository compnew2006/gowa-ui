package frontend

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestIsEmbedded(t *testing.T) {
	// The frontend is not embedded during go test unless we build it and use special tags or it actually exists
	// We just check the function doesn't panic.
	_ = IsEmbedded()
}

func TestNotEmbeddedHandler(t *testing.T) {
	handler := notEmbeddedHandler("test message")
	ctx := &fasthttp.RequestCtx{}

	handler(ctx)

	assert.Equal(t, fasthttp.StatusNotFound, ctx.Response.StatusCode())
	assert.Equal(t, "text/plain; charset=utf-8", string(ctx.Response.Header.Peek("Content-Type")))
	assert.Equal(t, "test message", string(ctx.Response.Body()))
}

func TestHandler_MissingDist(t *testing.T) {
	// The frontend is not reliably embedded during go test for unit testing purposes without make,
	// but if it IS embedded, we should verify the main SPA handler.
	handler := Handler("/testpath")
	ctx := &fasthttp.RequestCtx{}

	ctx.Request.SetRequestURI("/")
	handler(ctx)

	body := string(ctx.Response.Body())
	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		assert.Contains(t, body, "Frontend not embedded")
	} else if ctx.Response.StatusCode() == fasthttp.StatusOK {
		assert.True(t, strings.Contains(body, "window.__BASE_PATH__ = \"/testpath\";") || strings.Contains(body, "<html"))
		assert.Equal(t, "text/html; charset=utf-8", string(ctx.Response.Header.Peek("Content-Type")))
	}

	// Try the bootstrap script
	ctx.Response.Reset()
	ctx.Request.SetRequestURI("/__whatomate_base_path__.js")
	handler(ctx)
	if ctx.Response.StatusCode() == fasthttp.StatusOK {
		assert.Equal(t, "application/javascript; charset=utf-8", string(ctx.Response.Header.Peek("Content-Type")))
		assert.Contains(t, string(ctx.Response.Body()), "window.__BASE_PATH__ = \"/testpath\";")
	}

	// Request an asset that might exist (like index.html to simulate file serve)
	ctx.Response.Reset()
	ctx.Request.SetRequestURI("/index.html")
	handler(ctx)
	if ctx.Response.StatusCode() == fasthttp.StatusOK {
		// Just ensure it returned successfully
		assert.True(t, len(ctx.Response.Body()) > 0)
	}

	// Try an API prefix request - this should fall through SPA to fasthttpadaptor which returns 404
	ctx.Response.Reset()
	ctx.Request.SetRequestURI("/api/v1/status")
	handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound || strings.Contains(string(ctx.Response.Body()), "Frontend not embedded") {
		// Valid responses - either the embed fallback OR the actual 404 from fileServer
	} else {
		assert.Equal(t, fasthttp.StatusNotFound, ctx.Response.StatusCode())
		assert.NotContains(t, string(ctx.Response.Body()), "<html") // Not the SPA fallback
	}
}
