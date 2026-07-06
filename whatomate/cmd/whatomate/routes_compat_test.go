package main

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestSetupRoutes_CompatibilityAliasesRequireAuth(t *testing.T) {
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

	tests := []struct {
		name string
		path string
	}{
		{name: "auth me alias", path: "/api/auth/me"},
		{name: "chat sessions alias", path: "/api/chat/sessions"},
		{name: "chat session detail alias", path: "/api/chat/sessions/test-session"},
		{name: "analytics root alias", path: "/api/analytics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := testutil.NewGETRequest(t)
			req.RequestCtx.Request.SetRequestURI(tt.path)

			handler(req.RequestCtx)

			require.Equal(t, fasthttp.StatusUnauthorized, req.RequestCtx.Response.StatusCode(), "expected alias route to exist and hit auth middleware")
		})
	}
}
