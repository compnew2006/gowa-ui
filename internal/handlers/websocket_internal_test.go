package handlers

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/logf"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/test/testutil"
)

// A rejected WebSocket handshake answers with the status the upgrader chose
// (403 for a foreign origin, 400 for a malformed handshake) instead of a
// blanket 500, and the log line names the origin. Without it, a page opened
// on an unlisted host (www vs apex) showed up only as an anonymous
// "WebSocket upgrade failed" in the logs.
func TestWebSocketHandler_RejectedHandshake(t *testing.T) {
	tests := []struct {
		name       string
		origin     string
		noUpgrade  bool
		wantStatus int
	}{
		{name: "origin not in allowed_origins", origin: "https://www.app.example.com", wantStatus: fasthttp.StatusForbidden},
		{name: "not a websocket handshake", origin: "https://app.example.com", noUpgrade: true, wantStatus: fasthttp.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			app := &App{
				Config: &config.Config{Server: config.ServerConfig{AllowedOrigins: "https://app.example.com"}},
				Log:    logf.New(logf.Opts{Writer: &logs, Level: logf.DebugLevel}),
			}

			req := testutil.NewGETRequest(t)
			h := &req.RequestCtx.Request.Header
			h.Set("Connection", "Upgrade")
			if !tc.noUpgrade {
				h.Set("Upgrade", "websocket")
			}
			h.Set("Sec-WebSocket-Version", "13")
			h.Set("Sec-WebSocket-Key", base64.StdEncoding.EncodeToString([]byte("the sample nonce"))) // RFC 6455 example key
			h.Set("Origin", tc.origin)

			assert.NoError(t, app.WebSocketHandler(req))
			assert.Equal(t, tc.wantStatus, testutil.GetResponseStatusCode(req))
			assert.Contains(t, logs.String(), tc.origin, "the log line must name the rejected origin")
		})
	}
}
