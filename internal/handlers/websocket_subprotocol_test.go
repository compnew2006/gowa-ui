package handlers

import (
	"strings"
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSocketHandler_EchoesSubprotocol(t *testing.T) {
	app := &App{
		Config: &config.Config{
			JWT: config.JWTConfig{Secret: testutil.TestJWTSecret},
			Server: config.ServerConfig{
				AllowedOrigins: "", // Allow all for test
			},
		},
		Log: testutil.NopLogger(),
	}

	req := testutil.NewGETRequest(t)
	// WS Handshake Headers
	req.RequestCtx.Request.Header.Set("Connection", "Upgrade")
	req.RequestCtx.Request.Header.Set("Upgrade", "websocket")
	req.RequestCtx.Request.Header.Set("Sec-WebSocket-Version", "13")
	req.RequestCtx.Request.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.RequestCtx.Request.Header.Set("Origin", "http://localhost")

	// Set Token and Subprotocol
	token := signedWSTokenForTest(t, testutil.TestJWTSecret, jwt.SigningMethodHS256)
	req.RequestCtx.Request.Header.Set("Sec-WebSocket-Protocol", "whm.v1, auth."+token)

	err := app.WebSocketHandler(req)
	require.NoError(t, err)

	// Since we are not actually wiring up a real TCP connection in the test glue,
	// fastHTTPUpgrader will either return an error, or set the response headers for 101 Switching Protocols.

	// Check if the response includes Sec-WebSocket-Protocol
	// fastHTTP handles the upgrade by setting a response header and a Connection=Upgrade
	protocol := string(req.RequestCtx.Response.Header.Peek("Sec-WebSocket-Protocol"))

	// Verify that the server echoed "whm.v1" or whatever subprotocol it intends to support
	assert.NotEmpty(t, protocol, "Server must echo a Sec-WebSocket-Protocol, otherwise browsers will drop the connection")
	assert.True(t, strings.HasPrefix(protocol, "whm.v1") || strings.Contains(protocol, "whm.v1"), "Subprotocol should include whm.v1")
}
