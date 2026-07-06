package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestCorsWrapperAppliesSecurityHeadersToPreflight(t *testing.T) {
	t.Parallel()

	req := &fasthttp.Request{}
	req.Header.SetMethod(fasthttp.MethodOptions)
	req.SetRequestURI("/api/health")

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}, nil)

	handler := corsWrapper(func(ctx *fasthttp.RequestCtx) {
		t.Fatal("preflight should be handled before next handler")
	}, nil)
	handler(ctx)

	require.Equal(t, fasthttp.StatusNoContent, ctx.Response.StatusCode())
	require.Equal(t, "max-age=31536000; includeSubDomains", string(ctx.Response.Header.Peek("Strict-Transport-Security")))
	require.NotEmpty(t, string(ctx.Response.Header.Peek("Content-Security-Policy")))
	require.Equal(t, "nosniff", string(ctx.Response.Header.Peek("X-Content-Type-Options")))
}
