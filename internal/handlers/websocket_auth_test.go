package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestWSTokenFromRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(req *fastglue.Request)
		wantToken string
		wantErr   bool
	}{
		{
			name: "authorization header takes precedence",
			setup: func(req *fastglue.Request) {
				req.RequestCtx.Request.Header.Set("Authorization", "Bearer bearer-token")
				req.RequestCtx.Request.Header.Set("Sec-WebSocket-Protocol", "whm.v1, auth.protocol-token")
			},
			wantToken: "bearer-token",
		},
		{
			name: "uses bearer token from authorization header",
			setup: func(req *fastglue.Request) {
				req.RequestCtx.Request.Header.Set("Authorization", "Bearer bearer-token")
			},
			wantToken: "bearer-token",
		},
		{
			name: "uses auth token from websocket subprotocol",
			setup: func(req *fastglue.Request) {
				req.RequestCtx.Request.Header.Set("Sec-WebSocket-Protocol", "whm.v1, auth.protocol-token")
			},
			wantToken: "protocol-token",
		},
		{
			name: "uses bearer token from websocket subprotocol",
			setup: func(req *fastglue.Request) {
				req.RequestCtx.Request.Header.Set("Sec-WebSocket-Protocol", "whm.v1, bearer.protocol-token")
			},
			wantToken: "protocol-token",
		},
		{
			name: "returns error for invalid authorization header",
			setup: func(req *fastglue.Request) {
				req.RequestCtx.Request.Header.Set("Authorization", "Token abc")
			},
			wantErr: true,
		},
		{
			name: "returns empty token when no auth is provided",
			setup: func(req *fastglue.Request) {
				// intentionally empty
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := testutil.NewGETRequest(t)
			tc.setup(req)

			token, err := wsTokenFromRequest(req)
			if tc.wantErr {
				require.Error(t, err)
				assert.Empty(t, token)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantToken, token)
		})
	}
}

func TestWebSocketHandler_RejectsMissingTokenBeforeUpgrade(t *testing.T) {
	t.Parallel()

	app := &App{
		Config: &config.Config{JWT: config.JWTConfig{Secret: testutil.TestJWTSecret}},
		Log:    testutil.NopLogger(),
	}
	req := testutil.NewGETRequest(t)

	err := app.WebSocketHandler(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Missing WebSocket token")
}

func TestWebSocketHandler_RejectsInvalidTokenBeforeUpgrade(t *testing.T) {
	t.Parallel()

	app := &App{
		Config: &config.Config{JWT: config.JWTConfig{Secret: testutil.TestJWTSecret}},
		Log:    testutil.NopLogger(),
	}
	req := testutil.NewGETRequest(t)
	req.RequestCtx.Request.Header.Set("Authorization", "Bearer not-a-jwt")

	err := app.WebSocketHandler(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid or expired WebSocket token")
}

func TestWebSocketHandler_ValidTokenPassesPreUpgradeAuth(t *testing.T) {
	t.Parallel()

	app := &App{
		Config: &config.Config{JWT: config.JWTConfig{Secret: testutil.TestJWTSecret}},
		Log:    testutil.NopLogger(),
	}
	req := testutil.NewGETRequest(t)
	req.RequestCtx.Request.Header.Set("Authorization", "Bearer "+signedWSTokenForTest(t, testutil.TestJWTSecret, jwt.SigningMethodHS256))

	err := app.WebSocketHandler(req)
	require.NoError(t, err)
	assert.NotEqual(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
	assert.Contains(t, string(testutil.GetResponseBody(req)), "WebSocket upgrade failed")
}

func TestValidateWSToken_RejectsUnexpectedSigningMethod(t *testing.T) {
	t.Parallel()

	app := &App{
		Config: &config.Config{JWT: config.JWTConfig{Secret: testutil.TestJWTSecret}},
		Log:    testutil.NopLogger(),
	}

	_, _, err := app.validateWSToken(signedWSTokenForTest(t, testutil.TestJWTSecret, jwt.SigningMethodHS384))
	require.Error(t, err)
}

func signedWSTokenForTest(t *testing.T, secret string, method jwt.SigningMethod) string {
	t.Helper()

	claims := middleware.JWTClaims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "whatomate",
			Subject:   "ws",
		},
	}

	token := jwt.NewWithClaims(method, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}
