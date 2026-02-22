package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestNewUpgrader_CheckOrigin(t *testing.T) {
	t.Parallel()

	allowlist := middleware.ParseAllowedOrigins("https://app.example.com")

	tests := []struct {
		name           string
		host           string
		origin         string
		allowedOrigins map[string]bool
		wantAllowed    bool
	}{
		{
			name:           "allowlist permits configured origin",
			host:           "api.example.com",
			origin:         "https://app.example.com",
			allowedOrigins: allowlist,
			wantAllowed:    true,
		},
		{
			name:           "allowlist blocks unknown origin",
			host:           "api.example.com",
			origin:         "https://evil.example",
			allowedOrigins: allowlist,
			wantAllowed:    false,
		},
		{
			name:           "fallback permits same origin",
			host:           "example.com",
			origin:         "http://example.com",
			allowedOrigins: nil,
			wantAllowed:    true,
		},
		{
			name:           "fallback permits localhost origin",
			host:           "api.example.com",
			origin:         "http://localhost:3000",
			allowedOrigins: nil,
			wantAllowed:    true,
		},
		{
			name:           "fallback blocks foreign origin",
			host:           "api.example.com",
			origin:         "https://evil.example",
			allowedOrigins: nil,
			wantAllowed:    false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			upgrader := newUpgrader(tc.allowedOrigins)
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.SetHost(tc.host)
			ctx.Request.Header.Set("Origin", tc.origin)

			assert.Equal(t, tc.wantAllowed, upgrader.CheckOrigin(ctx))
		})
	}
}
