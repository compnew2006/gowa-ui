package frontend

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestHandler_ServesBasePathBootstrapScript(t *testing.T) {
	handler := Handler("/portal")

	resp := performRequest(t, handler, "/"+basePathBootstrapScriptName)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode())
	assert.Equal(t, "window.__BASE_PATH__ = \"/portal\";", string(resp.Body()))
	assert.Contains(t, string(resp.Header.Peek("Content-Type")), "application/javascript")
	assert.Equal(t, "no-store", string(resp.Header.Peek("Cache-Control")))
}

func TestHandler_IndexUsesExternalBasePathScript(t *testing.T) {
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
