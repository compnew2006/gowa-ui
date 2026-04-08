package handlers

import (
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
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
			origin := strings.TrimSpace(string(ctx.Request.Header.Peek("Origin")))
			if origin == "" {
				// Explicitly reject missing Origin to prevent browser-based CSWSH bypass.
				return false
			}
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
	if a.License != nil && a.License.IsLocked() {
		return r.SendErrorEnvelope(fasthttp.StatusLocked, "A valid license is required to open WebSocket connections", nil, "")
	}
	if a.License != nil && a.License.RequiresQuotaCleanup() {
		return r.SendErrorEnvelope(fasthttp.StatusLocked, "License quota overage requires cleanup before WebSocket connections can resume", map[string]any{
			"code":         "license_quota_overage",
			"cleanup_url":  "/license-cleanup",
			"activate_url": "/activate",
		}, "")
	}

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
	up.Subprotocols = []string{"whm.v1"}
	err = up.Upgrade(r.RequestCtx, func(conn *websocket.Conn) {
		// Create unauthenticated client — auth happens via first message
		client := ws.NewUnauthenticatedClient(a.WSHub, conn, a.validateWSTokenFn())
		client.SetContactAccessFn(a.canSubscribeToContactUpdates)

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
	authHeader := strings.TrimSpace(string(r.RequestCtx.Request.Header.Peek("Authorization")))
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			return "", fmt.Errorf("invalid authorization header format")
		}
		return strings.TrimSpace(parts[1]), nil
	}

	wsProtocols := strings.TrimSpace(string(r.RequestCtx.Request.Header.Peek("Sec-WebSocket-Protocol")))
	if token := wsTokenFromProtocols(wsProtocols); token != "" {
		return token, nil
	}

	return "", nil
}

func wsTokenFromProtocols(protocolHeader string) string {
	if strings.TrimSpace(protocolHeader) == "" {
		return ""
	}
	for _, raw := range strings.Split(protocolHeader, ",") {
		protocol := strings.TrimSpace(raw)
		lower := strings.ToLower(protocol)
		switch {
		case strings.HasPrefix(lower, "auth.") && len(protocol) > len("auth."):
			return protocol[len("auth."):]
		case strings.HasPrefix(lower, "bearer.") && len(protocol) > len("bearer."):
			return protocol[len("bearer."):]
		}
	}
	return ""
}

// validateWSTokenFn returns a function that validates a JWT token
// and returns user ID and organization ID.
func (a *App) validateWSTokenFn() ws.AuthenticateFn {
	return a.validateWSToken
}

func (a *App) canSubscribeToContactUpdates(userID, orgID, contactID uuid.UUID) bool {
	if userID == uuid.Nil || orgID == uuid.Nil || contactID == uuid.Nil {
		return false
	}

	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if a.shouldRestrictChatVisibilityToAgentScope(userID, orgID) {
		query = applyAgentVisibleChatAccessFilter(query, userID)
	}

	restrictedInstanceIDs, err := a.getRestrictedInstancesForUser(orgID, userID)
	if err != nil {
		a.Log.Warn("Failed to resolve restricted instances for websocket contact subscription",
			"error", err,
			"org_id", orgID,
			"user_id", userID,
			"contact_id", contactID)
		return false
	}

	query = applyRestrictedInstanceVisibilityFilter(query, restrictedInstanceIDs)
	if err := query.First(&contact).Error; err != nil {
		return false
	}

	normalizeContactStatus(&contact)
	if isChatRestrictedForMessageRead(contact) && !a.canAccessRestrictedChatWithoutClaim(contact, userID, orgID) {
		return false
	}

	return true
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
	if claims.Subject != wsTokenSubject || claims.UserID == uuid.Nil || claims.OrganizationID == uuid.Nil {
		return uuid.Nil, uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	return claims.UserID, claims.OrganizationID, nil
}
