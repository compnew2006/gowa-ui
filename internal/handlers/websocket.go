package handlers

import (
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/internal/middleware"
	ws "github.com/compnew2006/whatomate/internal/websocket"
	"github.com/fasthttp/websocket"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// newUpgrader creates a WebSocket upgrader that validates origins against the
// configured allowed origins and safe defaults.
func newUpgrader(allowedOrigins map[string]bool) websocket.FastHTTPUpgrader {
	return websocket.FastHTTPUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
			origin := string(ctx.Request.Header.Peek("Origin"))
			return middleware.IsOriginAllowedForRequest(origin, allowedOrigins, string(ctx.Host()), ctx.IsTLS())
		},
	}
}

// wsUpgrader returns a WebSocket upgrader configured with the app's allowed origins.
func (a *App) wsUpgrader() websocket.FastHTTPUpgrader {
	allowedOrigins := middleware.ParseAllowedOrigins(a.Config.Server.AllowedOrigins)
	return newUpgrader(allowedOrigins)
}

// WebSocketHandler handles WebSocket connections.
// Authentication is performed via message-based auth after the upgrade:
// the client must send {"type":"auth","payload":{"token":"<jwt>"}} within 5 seconds.
func (a *App) WebSocketHandler(r *fastglue.Request) error {
	// Require a valid token in the HTTP handshake request before upgrade to
	// prevent unauthenticated connection exhaustion.
	tokenString, err := wsTokenFromRequest(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid authorization header format", nil, "")
	}
	if tokenString == "" {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Missing WebSocket token", nil, "")
	}
	if _, _, err := a.validateWSToken(tokenString); err != nil {
		a.Log.Warn("WebSocket handshake authentication failed", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid or expired WebSocket token", nil, "")
	}

	// Upgrade to WebSocket only after successful handshake authentication.
	up := a.wsUpgrader()
	err = up.Upgrade(r.RequestCtx, func(conn *websocket.Conn) {
		// Create unauthenticated client — auth happens via first message
		client := ws.NewUnauthenticatedClient(a.WSHub, conn, a.validateWSTokenFn())

		// Start pumps in goroutines
		// Client self-registers with hub after successful auth message
		go client.WritePump()
		client.ReadPump() // Blocking - runs until connection closes
	})

	if err != nil {
		a.Log.Error("WebSocket upgrade failed", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "WebSocket upgrade failed", nil, "")
	}

	return nil
}

func wsTokenFromRequest(r *fastglue.Request) (string, error) {
	token := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("token")))
	if token != "" {
		return token, nil
	}

	authHeader := strings.TrimSpace(string(r.RequestCtx.Request.Header.Peek("Authorization")))
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			return "", fmt.Errorf("invalid authorization header format")
		}
		return strings.TrimSpace(parts[1]), nil
	}

	token = strings.TrimSpace(string(r.RequestCtx.Request.Header.Cookie(cookieAccessName)))
	return token, nil
}

// validateWSTokenFn returns a function that validates a JWT token
// and returns user ID and organization ID.
func (a *App) validateWSTokenFn() ws.AuthenticateFn {
	return a.validateWSToken
}

func (a *App) validateWSToken(tokenString string) (uuid.UUID, uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		signingMethod, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok || signingMethod.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected JWT signing method: %s", token.Method.Alg())
		}
		return a.jwtSecretBytes()
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil || !token.Valid {
		return uuid.Nil, uuid.Nil, err
	}

	claims, ok := token.Claims.(*middleware.JWTClaims)
	if !ok {
		return uuid.Nil, uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	return claims.UserID, claims.OrganizationID, nil
}
