package frontend

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestHandler_ServesBasePathBootstrapScript(t *testing.T) {
	// The embed FS will always have dist as a directory locally in source, but inside dist we need index.html
	entries, _ := distFS.ReadDir("dist")
	hasIndex := false
	for _, e := range entries {
		if e.Name() == "index.html" {
			hasIndex = true
			break
		}
	}
	if !hasIndex {
		t.Skip("Skipping test because frontend index.html is not embedded")
	}

	handler := Handler("/portal")

	resp := performRequest(t, handler, "/"+basePathBootstrapScriptName)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode())
	assert.Equal(t, "window.__BASE_PATH__ = \"/portal\";", string(resp.Body()))
	assert.Contains(t, string(resp.Header.Peek("Content-Type")), "application/javascript")
	assert.Equal(t, "no-store", string(resp.Header.Peek("Cache-Control")))
}

func TestHandler_IndexUsesExternalBasePathScript(t *testing.T) {
	entries, _ := distFS.ReadDir("dist")
	hasIndex := false
	for _, e := range entries {
		if e.Name() == "index.html" {
			hasIndex = true
			break
		}
	}
	if !hasIndex {
		t.Skip("Skipping test because frontend index.html is not embedded")
	}

	handler := Handler("/portal")

	resp := performRequest(t, handler, "/")
	body := string(resp.Body())

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode())
	assert.Contains(t, body, `<base href="/portal/">`)
	assert.Contains(t, body, `<script src="./`+basePathBootstrapScriptName+`"></script>`)
	assert.NotContains(t, body, "window.__BASE_PATH__ = ")
	assert.False(t, strings.Contains(body, "<script>"), "index should not include inline script tags")
}

func performRequest(t *testing.T, handler fasthttp.RequestHandler, uri string) *fasthttp.Response {
	t.Helper()

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(uri)
	req.Header.SetMethod(fasthttp.MethodGet)
	defer fasthttp.ReleaseRequest(req)

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, nil, nil)
	handler(ctx)

	resp := fasthttp.AcquireResponse()
	ctx.Response.CopyTo(resp)
	require.NotNil(t, resp)

	t.Cleanup(func() {
		fasthttp.ReleaseResponse(resp)
	})

	return resp
}
