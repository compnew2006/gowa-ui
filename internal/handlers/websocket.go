package handlers

import (
	"github.com/compnew2006/gowa-ui/internal/middleware"
	ws "github.com/compnew2006/gowa-ui/internal/websocket"
	"github.com/fasthttp/websocket"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// newUpgrader creates a WebSocket upgrader that validates origins against the
// configured allowed origins. If no origins are configured, all are allowed.
func newUpgrader(allowedOrigins map[string]bool) websocket.FastHTTPUpgrader {
	return websocket.FastHTTPUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
			origin := string(ctx.Request.Header.Peek("Origin"))
			return middleware.IsOriginAllowed(origin, allowedOrigins)
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
	// Upgrade to WebSocket immediately (unauthenticated)
	up := a.wsUpgrader()
	err := up.Upgrade(r.RequestCtx, func(conn *websocket.Conn) {
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

// validateWSTokenFn returns a function that validates a JWT token
// and returns user ID and organization ID.
func (a *App) validateWSTokenFn() ws.AuthenticateFn {
	return func(tokenString string) (uuid.UUID, uuid.UUID, error) {
		token, err := jwt.ParseWithClaims(tokenString, &middleware.JWTClaims{}, middleware.HMACKeyFunc(a.Config.JWT.Secret))

		if err != nil || !token.Valid {
			return uuid.Nil, uuid.Nil, err
		}

		claims, ok := token.Claims.(*middleware.JWTClaims)
		if !ok {
			return uuid.Nil, uuid.Nil, jwt.ErrTokenInvalidClaims
		}

		// Reject refresh tokens: they are long-lived and single-use (JTI) —
		// the socket must authenticate with the short-lived access/WS token.
		// JTI presence also catches legacy refresh tokens issued before
		// token_type existed.
		if claims.TokenType == middleware.TokenTypeRefresh || claims.ID != "" {
			return uuid.Nil, uuid.Nil, jwt.ErrTokenInvalidClaims
		}

		return claims.UserID, claims.OrganizationID, nil
	}
}
