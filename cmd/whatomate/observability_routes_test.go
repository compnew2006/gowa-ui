package main

import (
	"net"
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/observability"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestSetupRoutes_ObservabilityDisabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: testutil.TestJWTSecret,
		},
	}

	app := &handlers.App{
		Config: cfg,
		Log:    testutil.NopLogger(),
	}

	g := fastglue.NewGlue()
	setupRoutes(g, app, testutil.NopLogger(), "", nil, cfg, nil)
	handler := g.Handler()

	metricsReq := newRemoteGETRequest(t, "/metrics", "127.0.0.1")
	handler(metricsReq.RequestCtx)
	require.NotEqual(t, "text/plain; version=0.0.4; charset=utf-8", string(metricsReq.RequestCtx.Response.Header.ContentType()))
	require.NotContains(t, string(metricsReq.RequestCtx.Response.Body()), "# HELP whatomate_uptime_seconds")

	pprofReq := newRemoteGETRequest(t, "/debug/pprof/", "127.0.0.1")
	handler(pprofReq.RequestCtx)
	require.NotContains(t, string(pprofReq.RequestCtx.Response.Body()), "Types of profiles available:")
}

func TestSetupRoutes_MetricsRequireTokenWhenConfigured(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: testutil.TestJWTSecret,
		},
		Observability: config.ObservabilityConfig{
			EnableMetrics: true,
			AccessToken:   "debug-token",
		},
	}

	app := &handlers.App{
		Config: cfg,
		Log:    testutil.NopLogger(),
	}

	manager := observability.NewManager(cfg.Observability, nil, nil)
	g := fastglue.NewGlue()
	setupRoutes(g, app, testutil.NopLogger(), "", nil, cfg, manager)
	handler := observedHandler(g.Handler(), manager, testutil.NopLogger())

	unauthorized := newRemoteGETRequest(t, "/metrics", "127.0.0.1")
	handler(unauthorized.RequestCtx)
	require.Equal(t, fasthttp.StatusUnauthorized, unauthorized.RequestCtx.Response.StatusCode())
	require.Contains(t, string(unauthorized.RequestCtx.Response.Body()), "Observability token required")

	authorized := newRemoteGETRequest(t, "/metrics", "127.0.0.1")
	authorized.RequestCtx.Request.Header.Set("Authorization", "Bearer debug-token")
	handler(authorized.RequestCtx)
	require.Equal(t, fasthttp.StatusOK, authorized.RequestCtx.Response.StatusCode())
	require.Contains(t, string(authorized.RequestCtx.Response.Body()), "# HELP whatomate_uptime_seconds")
}

func TestSetupRoutes_PprofAllowsLoopbackWhenEnabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: testutil.TestJWTSecret,
		},
		Observability: config.ObservabilityConfig{
			EnablePprof: true,
		},
	}

	app := &handlers.App{
		Config: cfg,
		Log:    testutil.NopLogger(),
	}

	manager := observability.NewManager(cfg.Observability, nil, nil)
	g := fastglue.NewGlue()
	setupRoutes(g, app, testutil.NopLogger(), "", nil, cfg, manager)
	handler := g.Handler()

	req := newRemoteGETRequest(t, "/debug/pprof/", "127.0.0.1")
	handler(req.RequestCtx)

	require.Equal(t, fasthttp.StatusOK, req.RequestCtx.Response.StatusCode())
	require.Contains(t, string(req.RequestCtx.Response.Body()), "Types of profiles available:")
}

func newRemoteGETRequest(t *testing.T, path string, remoteIP string) *fastglue.Request {
	t.Helper()

	req := &fasthttp.Request{}
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI(path)

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, &net.TCPAddr{IP: net.ParseIP(remoteIP), Port: 12345}, nil)
	return &fastglue.Request{RequestCtx: ctx}
}
