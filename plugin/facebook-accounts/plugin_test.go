package facebookaccounts

import (
	"log/slog"
	"testing"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPluginRoutes(t *testing.T) {
	app := newRouteTestApp(t)
	plugin := &Plugin{}

	require.NoError(t, plugin.Init(app, app.DB, nil, slog.Default()))
	require.Equal(t, "facebook-accounts", plugin.Name())
	require.Equal(t, core.ModuleManifest{
		Key: "facebook-accounts", DisplayName: "Facebook Accounts", Version: "1.0.0",
		SchemaVersion: 1, Dependencies: []string{"facebook-core"}, DefaultEnabled: true,
	}, plugin.Manifest())
	require.NoError(t, plugin.Migrate(app.DB))

	glue := fastglue.NewGlue()
	glue.Before(func(r *fastglue.Request) *fastglue.Request {
		r.RequestCtx.SetStatusCode(fasthttp.StatusUnauthorized)
		return nil
	})
	plugin.Routes(glue)
	handler := glue.Handler()

	routes := []struct {
		method string
		path   string
	}{
		{method: fasthttp.MethodGet, path: "/api/facebook/accounts"},
		{method: fasthttp.MethodPost, path: "/api/facebook/accounts"},
		{method: fasthttp.MethodGet, path: "/api/facebook/accounts/test-account"},
		{method: fasthttp.MethodPut, path: "/api/facebook/accounts/test-account"},
		{method: fasthttp.MethodDelete, path: "/api/facebook/accounts/test-account"},
		{method: fasthttp.MethodPost, path: "/api/facebook/accounts/test-account/pages/refresh"},
		{method: fasthttp.MethodPost, path: "/api/facebook/accounts/test-account/pages/test-page/connect"},
		{method: fasthttp.MethodPost, path: "/api/facebook/accounts/test-account/pages/test-page/disconnect"},
		{method: fasthttp.MethodDelete, path: "/api/facebook/accounts/test-account/pages/test-page"},
		{method: fasthttp.MethodPost, path: "/api/facebook/accounts/test-account/pages/test-page/feed"},
		{method: fasthttp.MethodGet, path: "/api/facebook/accounts/test-account/pages/test-page/insights"},
		{method: fasthttp.MethodPost, path: "/api/facebook/accounts/test-account/pages/test-page/messages"},
	}
	for _, route := range routes {
		assertRouteRegistered(t, handler, route.method, route.path)
	}
	assertRouteNotRegistered(t, handler, fasthttp.MethodGet, "/api/facebook/accounts/test-account/oauth/renew")
	assertRouteNotRegistered(t, handler, fasthttp.MethodGet, "/api/facebook/comments")
}

func newRouteTestApp(t *testing.T) *handlers.App {
	t.Helper()

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=127.0.0.1 user=whatomate_test dbname=whatomate_test sslmode=disable",
	}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)

	return &handlers.App{
		Config: &config.Config{},
		DB:     db,
		Log:    testutil.NopLogger(),
	}
}

func assertRouteRegistered(t *testing.T, handler fasthttp.RequestHandler, method, path string) {
	t.Helper()

	request := testutil.NewRequest(t)
	request.RequestCtx.Request.Header.SetMethod(method)
	request.RequestCtx.Request.SetRequestURI(path)
	handler(request.RequestCtx)

	require.NotEqual(t, fasthttp.StatusNotFound, request.RequestCtx.Response.StatusCode())
}

func assertRouteNotRegistered(t *testing.T, handler fasthttp.RequestHandler, method, path string) {
	t.Helper()

	request := testutil.NewRequest(t)
	request.RequestCtx.Request.Header.SetMethod(method)
	request.RequestCtx.Request.SetRequestURI(path)
	handler(request.RequestCtx)

	require.Equal(t, fasthttp.StatusNotFound, request.RequestCtx.Response.StatusCode())
}
