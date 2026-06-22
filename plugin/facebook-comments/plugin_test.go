package facebookcomments

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
	for _, rateLimitEnabled := range []bool{false, true} {
		t.Run(rateLimitTestName(rateLimitEnabled), func(t *testing.T) {
			app := newRouteTestApp(t, rateLimitEnabled)
			plugin := &Plugin{}

			require.NoError(t, plugin.Init(app, app.DB, nil, slog.Default()))
			require.Equal(t, "facebook-comments", plugin.Name())
			require.Equal(t, core.ModuleManifest{
				Key: "facebook-comments", DisplayName: "Facebook Comments", Version: "1.0.0",
				SchemaVersion: 1, Dependencies: []string{"facebook-accounts"}, DefaultEnabled: true,
			}, plugin.Manifest())
			require.NoError(t, plugin.Migrate(app.DB))

			glue := fastglue.NewGlue()
			plugin.Routes(glue)
			handler := glue.Handler()

			routes := []struct {
				method string
				path   string
			}{
				{method: fasthttp.MethodGet, path: "/api/facebook/comments"},
				{method: fasthttp.MethodGet, path: "/api/facebook/comments/pages"},
				{method: fasthttp.MethodPost, path: "/api/facebook/comments/sync"},
				{method: fasthttp.MethodGet, path: "/api/facebook/comments/settings"},
				{method: fasthttp.MethodPut, path: "/api/facebook/comments/settings"},
				{method: fasthttp.MethodGet, path: "/api/facebook/comments/pages/test-page/settings"},
				{method: fasthttp.MethodPut, path: "/api/facebook/comments/pages/test-page/settings"},
				{method: fasthttp.MethodPost, path: "/api/facebook/comments/test-comment/reply"},
				{method: fasthttp.MethodPut, path: "/api/facebook/comments/test-comment/status"},
				{method: fasthttp.MethodGet, path: "/api/facebook/comments/webhook"},
				{method: fasthttp.MethodPost, path: "/api/facebook/comments/webhook"},
			}
			for _, route := range routes {
				assertRouteRegistered(t, handler, route.method, route.path)
			}
			assertRouteNotRegistered(t, handler, fasthttp.MethodGet, "/api/facebook/accounts")
		})
	}
}

func TestSettingsHandlersRequireAuthentication(t *testing.T) {
	app := newRouteTestApp(t, false)
	plugin := &Plugin{}
	require.NoError(t, plugin.Init(app, app.DB, nil, slog.Default()))

	tests := []struct {
		name    string
		handler fastglue.FastRequestHandler
	}{
		{name: "get settings", handler: plugin.GetFacebookCommentSettings},
		{name: "update settings", handler: plugin.UpdateFacebookCommentSettings},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := testutil.NewRequest(t)
			require.NoError(t, test.handler(request))
			require.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(request))
		})
	}
}

func rateLimitTestName(enabled bool) string {
	if enabled {
		return "rate_limit_enabled"
	}
	return "rate_limit_disabled"
}

func newRouteTestApp(t *testing.T, rateLimit bool) *handlers.App {
	t.Helper()

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=127.0.0.1 user=whatomate_test dbname=whatomate_test sslmode=disable",
	}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)

	return &handlers.App{
		Config: &config.Config{
			RateLimit: config.RateLimitConfig{
				Enabled:            rateLimit,
				WebhookMaxAttempts: 5,
				WindowSeconds:      60,
			},
		},
		DB:  db,
		Log: testutil.NopLogger(),
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
