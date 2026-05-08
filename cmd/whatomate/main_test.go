package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestVersionDefaults(t *testing.T) {
	assert.Equal(t, "dev", Version)
	assert.Equal(t, "unknown", BuildTime)
}

func TestPrintUsageDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		printUsage()
	})
}

func TestObservedHandlerNilManager(t *testing.T) {
	called := false
	inner := func(ctx *fasthttp.RequestCtx) {
		called = true
	}

	wrapped := observedHandler(inner, nil)
	var ctx fasthttp.RequestCtx
	wrapped(&ctx)

	assert.True(t, called)
}

func TestCorsWrapper(t *testing.T) {
	allowedOrigins := map[string]bool{
		"http://localhost:3000": true,
	}

	t.Run("sets CORS headers for allowed origin", func(t *testing.T) {
		called := false
		next := func(ctx *fasthttp.RequestCtx) {
			called = true
		}

		wrapped := corsWrapper(next, allowedOrigins)
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.Set("Origin", "http://localhost:3000")
		ctx.Request.SetRequestURI("/api/health")
		ctx.Request.Header.SetMethod("GET")
		wrapped(&ctx)

		assert.True(t, called)
		assert.Equal(t, "http://localhost:3000", string(ctx.Response.Header.Peek("Access-Control-Allow-Origin")))
		assert.Equal(t, "true", string(ctx.Response.Header.Peek("Access-Control-Allow-Credentials")))
	})

	t.Run("OPTIONS preflight returns 204", func(t *testing.T) {
		called := false
		next := func(ctx *fasthttp.RequestCtx) {
			called = true
		}

		wrapped := corsWrapper(next, allowedOrigins)
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.Set("Origin", "http://localhost:3000")
		ctx.Request.SetRequestURI("/api/health")
		ctx.Request.Header.SetMethod("OPTIONS")
		wrapped(&ctx)

		assert.False(t, called)
		assert.Equal(t, 204, ctx.Response.StatusCode())
		assert.Equal(t, "http://localhost:3000", string(ctx.Response.Header.Peek("Access-Control-Allow-Origin")))
	})

	t.Run("unknown origin does not set Access-Control-Allow-Origin", func(t *testing.T) {
		called := false
		next := func(ctx *fasthttp.RequestCtx) {
			called = true
		}

		wrapped := corsWrapper(next, allowedOrigins)
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.Set("Origin", "http://evil.com")
		ctx.Request.SetRequestURI("/api/health")
		ctx.Request.Header.SetMethod("GET")
		wrapped(&ctx)

		assert.True(t, called)
		assert.Empty(t, string(ctx.Response.Header.Peek("Access-Control-Allow-Origin")))
	})

	t.Run("always sets Access-Control-Allow-Methods", func(t *testing.T) {
		next := func(ctx *fasthttp.RequestCtx) {}
		wrapped := corsWrapper(next, allowedOrigins)
		var ctx fasthttp.RequestCtx
		wrapped(&ctx)

		methods := string(ctx.Response.Header.Peek("Access-Control-Allow-Methods"))
		assert.True(t, strings.Contains(methods, "GET"))
		assert.True(t, strings.Contains(methods, "POST"))
		assert.True(t, strings.Contains(methods, "OPTIONS"))
	})

	t.Run("no origin header still calls next", func(t *testing.T) {
		called := false
		next := func(ctx *fasthttp.RequestCtx) {
			called = true
		}

		wrapped := corsWrapper(next, allowedOrigins)
		var ctx fasthttp.RequestCtx
		ctx.Request.SetRequestURI("/api/health")
		ctx.Request.Header.SetMethod("GET")
		wrapped(&ctx)

		assert.True(t, called)
	})
}
