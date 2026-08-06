package main

import (
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/frontend"
	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

// setupRoutes wires every public + protected API route plus the embedded
// frontend catch-all. It is a thin conductor: each domain is registered by a
// dedicated setup*Routes helper (see below) called in the exact order they
// appeared in the prior monolithic block, and the three route-section
// g.Before middlewares (auth-skip, global /api rate limit, RBAC no-op) are
// registered by setupRouteMiddleware as a single order-preserving unit
// between the public auth routes and the protected routes — exactly where
// they sat before the refactor. fastglue applies Before handlers in
// registration order, so this ordering is observable behavior.
func setupRoutes(g *fastglue.Fastglue, app *handlers.App, lo logf.Logger, basePath string, rdb *redis.Client, cfg *config.Config) {
	setupHealthRoutes(g, app)
	setupAuthRoutes(g, app, cfg, rdb, lo)
	setupGowaWebhookRoutes(g, app, cfg, rdb, lo)
	setupWebSocketRoute(g, app)

	// Register the three route-section Before middlewares as one
	// order-preserving unit, exactly between the public auth/webhook routes
	// and the protected /api routes — same position as before the refactor.
	setupRouteMiddleware(g, app, cfg, rdb, lo)

	setupCurrentUserRoutes(g, app)
	setupUserRoutes(g, app)
	setupRoleRoutes(g, app)
	setupAPIKeyRoutes(g, app)
	setupAccountRoutes(g, app)
	setupGowaServerRoutes(g, app)
	setupContactRoutes(g, app)
	setupScheduledMessageRoutes(g, app)
	setupChatLifecycleRoutes(g, app)
	setupImportExportRoutes(g, app)
	setupTagRoutes(g, app)
	setupMessageRoutes(g, app)
	setupConversationNoteRoutes(g, app)
	setupMediaRoutes(g, app)
	setupTemplateRoutes(g, app)
	setupCampaignRoutes(g, app)
	setupTeamRoutes(g, app)
	setupAuditLogRoutes(g, app)
	setupCannedResponseRoutes(g, app)
	setupAnalyticsRoutes(g, app)
	setupWidgetRoutes(g, app)
	setupOrganizationRoutes(g, app)
	setupSSOSettingsRoutes(g, app)
	setupWebhookRoutes(g, app)
	setupCustomActionRoutes(g, app)
	setupFrontendCatchAll(g, lo, basePath)
}

// setupHealthRoutes registers the unauthenticated liveness/readiness probes.
func setupHealthRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Health check
	g.GET("/health", app.HealthCheck)
	g.GET("/ready", app.ReadyCheck)
}

// setupAuthRoutes registers the public auth + SSO routes, applying per-route
// rate limiting when cfg.RateLimit.Enabled.
func setupAuthRoutes(g *fastglue.Fastglue, app *handlers.App, cfg *config.Config, rdb *redis.Client, lo logf.Logger) {
	// Auth routes (public, optionally rate-limited)
	if cfg.RateLimit.Enabled {
		window := time.Duration(cfg.RateLimit.WindowSeconds) * time.Second
		lo.Info("Rate limiting enabled on auth endpoints",
			"login_max", cfg.RateLimit.LoginMaxAttempts,
			"register_max", cfg.RateLimit.RegisterMaxAttempts,
			"refresh_max", cfg.RateLimit.RefreshMaxAttempts,
			"sso_max", cfg.RateLimit.SSOMaxAttempts,
			"window_seconds", cfg.RateLimit.WindowSeconds)

		g.POST("/api/auth/login", withRateLimit(app.Login, middleware.RateLimitOpts{
			Redis: rdb, Log: lo, Max: cfg.RateLimit.LoginMaxAttempts, Window: window, KeyPrefix: "login", TrustProxy: cfg.RateLimit.TrustProxy,
		}))
		g.POST("/api/auth/register", withRateLimit(app.Register, middleware.RateLimitOpts{
			Redis: rdb, Log: lo, Max: cfg.RateLimit.RegisterMaxAttempts, Window: window, KeyPrefix: "register", TrustProxy: cfg.RateLimit.TrustProxy,
		}))
		g.POST("/api/auth/refresh", withRateLimit(app.RefreshToken, middleware.RateLimitOpts{
			Redis: rdb, Log: lo, Max: cfg.RateLimit.RefreshMaxAttempts, Window: window, KeyPrefix: "refresh", TrustProxy: cfg.RateLimit.TrustProxy,
		}))
	} else {
		g.POST("/api/auth/login", app.Login)
		g.POST("/api/auth/register", app.Register)
		g.POST("/api/auth/refresh", app.RefreshToken)
	}
	g.POST("/api/auth/logout", app.Logout)
	g.POST("/api/auth/switch-org", app.SwitchOrg)
	g.GET("/api/auth/ws-token", app.GetWSToken)

	// SSO routes (public, optionally rate-limited)
	g.GET("/api/auth/sso/providers", app.GetPublicSSOProviders)
	if cfg.RateLimit.Enabled {
		window := time.Duration(cfg.RateLimit.WindowSeconds) * time.Second
		g.GET("/api/auth/sso/{provider}/init", withRateLimit(app.InitSSO, middleware.RateLimitOpts{
			Redis: rdb, Log: lo, Max: cfg.RateLimit.SSOMaxAttempts, Window: window, KeyPrefix: "sso_init", TrustProxy: cfg.RateLimit.TrustProxy,
		}))
		g.GET("/api/auth/sso/{provider}/callback", withRateLimit(app.CallbackSSO, middleware.RateLimitOpts{
			Redis: rdb, Log: lo, Max: cfg.RateLimit.SSOMaxAttempts, Window: window, KeyPrefix: "sso_callback", TrustProxy: cfg.RateLimit.TrustProxy,
		}))
	} else {
		g.GET("/api/auth/sso/{provider}/init", app.InitSSO)
		g.GET("/api/auth/sso/{provider}/callback", app.CallbackSSO)
	}
}

// setupGowaWebhookRoutes registers the public GOWA webhook endpoints (single
// + per-device), both HMAC-verified in the handler and rate-limited per-IP to
// blunt brute-force signature attempts (FR-020).
func setupGowaWebhookRoutes(g *fastglue.Fastglue, app *handlers.App, cfg *config.Config, rdb *redis.Client, lo logf.Logger) {
	// GOWA webhook routes (public — HMAC verified in handler)
	// Rate-limited per-IP to prevent brute-force signature attempts (FR-020).
	gowaWebhookRL := middleware.RateLimitOpts{
		Redis:      rdb,
		Log:        lo,
		Max:        100,
		Window:     time.Minute,
		KeyPrefix:  "gowa_webhook",
		TrustProxy: cfg.RateLimit.TrustProxy,
	}
	// Single endpoint: resolves device from the payload's device_id field.
	g.POST(cfg.GOWA.WebhookPath, withRateLimit(app.GowaWebhookHandler, gowaWebhookRL))
	// Per-device endpoint: GOWA v8.10.0 supports device-specific webhook URLs.
	// The path device_id overrides the payload's device_id field.
	g.POST(cfg.GOWA.WebhookPath+"/{device_id}", withRateLimit(app.GowaWebhookHandlerDevice, gowaWebhookRL))
}

// setupWebSocketRoute registers the WS upgrade route (auth via message-based
// flow after upgrade).
func setupWebSocketRoute(g *fastglue.Fastglue, app *handlers.App) {
	// WebSocket route (auth via message-based flow after upgrade)
	g.GET("/ws", app.WebSocketHandler)
}

// setupRouteMiddleware registers the three route-section Before middlewares
// in their original order: (1) path-based auth-skip + JWT/API-key enforcement,
// (2) global /api rate limit keyed by user id, (3) RBAC no-op (route-level
// permission checks now live in handlers via HasPermission). The auth-skip
// path list is a security boundary — every entry must be preserved verbatim.
//
// The closures register here as one order-preserving unit, exactly as they
// appeared inline in the prior setupRoutes body, because fastglue applies
// Before handlers in registration order.
func setupRouteMiddleware(g *fastglue.Fastglue, app *handlers.App, cfg *config.Config, rdb *redis.Client, lo logf.Logger) {
	// (1) Path-based auth middleware.
	// For protected routes, we'll use a path-based middleware approach
	// Apply auth middleware globally but check path in the middleware
	g.Before(func(r *fastglue.Request) *fastglue.Request {
		// Skip auth for OPTIONS preflight requests (handled by CORS middleware)
		if string(r.RequestCtx.Method()) == "OPTIONS" {
			return r
		}
		path := string(r.RequestCtx.Path())
		// Skip auth for public routes.
		// GOWA per-device webhooks use a prefix match since the path includes
		// a dynamic device_id segment (e.g. /api/gowa/webhook/628123@s.whatsapp.net).
		if path == "/health" || path == "/ready" ||
			path == "/api/auth/login" || path == "/api/auth/register" || path == "/api/auth/refresh" ||
			path == "/api/auth/logout" || path == "/ws" ||
			path == "/api/gowa/webhook" || strings.HasPrefix(path, "/api/gowa/webhook/") {
			return r
		}
		// Skip auth for SSO routes (they handle their own auth via state tokens)
		if len(path) >= 13 && path[:13] == "/api/auth/sso" {
			return r
		}
		// Skip auth for custom action redirects (uses one-time token)
		if len(path) >= 28 && path[:28] == "/api/custom-actions/redirect" {
			return r
		}
		// Apply auth for all other /api routes (supports both JWT and API key).
		// AuthWithDBAndRedis wires the Redis client so the per-user token-version
		// revocation check (H3) runs on every authenticated request.
		if len(path) > 4 && path[:4] == "/api" {
			return middleware.AuthWithDBAndRedis(app.Config.JWT.Secret, app.DB, rdb, lo)(r)
		}
		return r
	})

	// (2) Global rate limit on all /api/ routes, keyed by user ID (or IP if unauthenticated).
	// Runs after auth so the user identity is available.
	if cfg.RateLimit.Enabled {
		apiMax := cfg.RateLimit.APIMaxRequests
		if apiMax == 0 {
			apiMax = 200
		}
		apiWindow := cfg.RateLimit.APIWindowSeconds
		if apiWindow == 0 {
			apiWindow = 60
		}
		apiRL := middleware.UserAwareRateLimit(middleware.RateLimitOpts{
			Redis:      rdb,
			Log:        lo,
			Max:        apiMax,
			Window:     time.Duration(apiWindow) * time.Second,
			KeyPrefix:  "api_global",
			TrustProxy: cfg.RateLimit.TrustProxy,
		})
		g.Before(func(r *fastglue.Request) *fastglue.Request {
			path := string(r.RequestCtx.Path())
			if len(path) > 4 && path[:4] == "/api" {
				return apiRL(r)
			}
			return r
		})
	}

	// (3) Role-based access control middleware
	g.Before(func(r *fastglue.Request) *fastglue.Request {
		method := string(r.RequestCtx.Method())

		// Skip OPTIONS preflight requests
		if method == "OPTIONS" {
			return r
		}

		// Route-level permission checks are now handled at the handler level
		// using the granular permission system (HasPermission checks)
		return r
	})
}

// setupCurrentUserRoutes registers /api/me endpoints (all authenticated users).
func setupCurrentUserRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Current User (all authenticated users)
	g.GET("/api/me", app.GetCurrentUser)
	g.PUT("/api/me/settings", app.UpdateCurrentUserSettings)
	g.PUT("/api/me/password", app.ChangePassword)
	g.PUT("/api/me/availability", app.UpdateAvailability)
	g.GET("/api/me/organizations", app.ListMyOrganizations)
}

// setupUserRoutes registers user-management endpoints (admin only, enforced in handler).
func setupUserRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// User Management (admin only - enforced by middleware)
	g.GET("/api/users", app.ListUsers)
	g.POST("/api/users", app.CreateUser)
	g.GET("/api/users/{id}", app.GetUser)
	g.PUT("/api/users/{id}", app.UpdateUser)
	g.DELETE("/api/users/{id}", app.DeleteUser)
}

// setupRoleRoutes registers roles & permissions endpoints (admin only).
func setupRoleRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Roles & Permissions (admin only - enforced by middleware)
	g.GET("/api/roles", app.ListRoles)
	g.POST("/api/roles", app.CreateRole)
	g.GET("/api/roles/{id}", app.GetRole)
	g.PUT("/api/roles/{id}", app.UpdateRole)
	g.DELETE("/api/roles/{id}", app.DeleteRole)
	g.GET("/api/permissions", app.ListPermissions)
}

// setupAPIKeyRoutes registers API-key management endpoints (admin only).
func setupAPIKeyRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// API Keys (admin only - enforced by middleware)
	g.GET("/api/api-keys", app.ListAPIKeys)
	g.GET("/api/api-keys/{id}", app.GetAPIKey)
	g.POST("/api/api-keys", app.CreateAPIKey)
	g.PUT("/api/api-keys/{id}", app.UpdateAPIKey)
	g.DELETE("/api/api-keys/{id}", app.DeleteAPIKey)
}

// setupAccountRoutes registers WhatsApp-account CRUD plus the per-account
// settings endpoints (close-rating, call-auto-reject, daily-reset, GOWA pair-code).
func setupAccountRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Accounts
	g.GET("/api/accounts", app.ListAccounts)
	g.POST("/api/accounts", app.CreateAccount)
	g.GET("/api/accounts/{id}", app.GetAccount)
	g.PUT("/api/accounts/{id}", app.UpdateAccount)
	g.DELETE("/api/accounts/{id}", app.DeleteAccount)

	// Per-account chat-close rating (settings live on the WhatsApp account)
	g.GET("/api/accounts/{id}/close-rating", app.GetCloseRatingSettings)
	g.PUT("/api/accounts/{id}/close-rating", app.UpdateCloseRatingSettings)
	g.GET("/api/accounts/{id}/close-rating/stats", app.GetCloseRatingStats)

	// Per-account call auto-reject (settings live on the WhatsApp account)
	g.GET("/api/accounts/{id}/call-auto-reject", app.GetCallAutoRejectSettings)
	g.PUT("/api/accounts/{id}/call-auto-reject", app.UpdateCallAutoRejectSettings)

	// Per-account daily assigned-chat reset schedule (settings live on the WhatsApp account)
	g.GET("/api/accounts/{id}/daily-reset", app.GetChatResetSettings)
	g.PUT("/api/accounts/{id}/daily-reset", app.UpdateChatResetSettings)

	// GOWA device management (QR code, pair code, connection status)
	g.GET("/api/accounts/{id}/gowa/qr", app.GowaLoginQR)
	g.GET("/api/accounts/{id}/gowa/status", app.GowaStatus)
	g.POST("/api/accounts/{id}/gowa/pair-code", app.GowaPairCode)
}

// setupGowaServerRoutes registers GOWA-instance management plus per-server
// device management.
func setupGowaServerRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// GOWA instance management (multi-instance dropdown + device provisioning)
	g.GET("/api/gowa/instances", app.GowaInstances)
	g.POST("/api/gowa/create-device", app.GowaCreateDevice)

	// GOWA servers (DB-managed instances + per-instance device management)
	g.GET("/api/gowa/servers", app.ListGowaInstances)
	g.POST("/api/gowa/servers", app.CreateGowaInstance)
	g.GET("/api/gowa/servers/{id}", app.GetGowaInstance)
	g.PUT("/api/gowa/servers/{id}", app.UpdateGowaInstance)
	g.DELETE("/api/gowa/servers/{id}", app.DeleteGowaInstance)

	// Devices within a DB-managed GOWA server
	g.GET("/api/gowa/servers/{id}/devices", app.ListGowaInstanceDevices)
	g.POST("/api/gowa/servers/{id}/devices", app.CreateGowaInstanceDevice)
	g.DELETE("/api/gowa/servers/{id}/devices/{deviceId}", app.DeleteGowaInstanceDevice)
	g.GET("/api/gowa/servers/{id}/devices/{deviceId}/qr", app.GowaInstanceDeviceQR)
	g.GET("/api/gowa/servers/{id}/devices/{deviceId}/status", app.GowaInstanceDeviceStatus)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/pair-code", app.GowaInstanceDevicePairCode)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/logout", app.GowaInstanceDeviceLogout)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/reconnect", app.GowaInstanceDeviceReconnect)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/sync", app.SyncGowaInstanceDevice)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/sync-contacts", app.SyncGowaInstanceDeviceContacts)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/sync-messages", app.SyncGowaInstanceMessages)
	g.GET("/api/gowa/servers/{id}/devices/{deviceId}/webhook", app.GetGowaInstanceDeviceWebhook)
	g.PUT("/api/gowa/servers/{id}/devices/{deviceId}/webhook", app.SetGowaInstanceDeviceWebhook)
}

// setupContactRoutes registers contact CRUD + assignment/tags + avatar endpoints.
func setupContactRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Contacts
	g.GET("/api/contacts", app.ListContacts)
	g.POST("/api/contacts", app.CreateContact)
	g.GET("/api/contacts/{id}", app.GetContact)
	g.GET("/api/contacts/{id}/avatar", app.RefreshContactAvatar)
	g.GET("/api/contacts/{id}/avatar/image", app.ServeContactAvatar)
	g.PUT("/api/contacts/{id}", app.UpdateContact)
	g.DELETE("/api/contacts/{id}", app.DeleteContact)
	g.PUT("/api/contacts/{id}/assign", app.AssignContact)
	g.PUT("/api/contacts/{id}/tags", app.UpdateContactTags)
}

// setupScheduledMessageRoutes registers scheduled-message CRUD.
func setupScheduledMessageRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Scheduled messages
	g.POST("/api/contacts/{id}/scheduled-messages", app.CreateScheduledMessage)
	g.GET("/api/contacts/{id}/scheduled-messages", app.ListContactScheduledMessages)
	g.GET("/api/scheduled-messages", app.ListScheduledMessages)
	g.PUT("/api/scheduled-messages/{id}", app.UpdateScheduledMessage)
	g.DELETE("/api/scheduled-messages/{id}", app.CancelScheduledMessage)
}

// setupChatLifecycleRoutes registers chat claim/release/close/reopen/join/leave
// + collaborator endpoints.
func setupChatLifecycleRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Chat Lifecycle
	g.PUT("/api/contacts/{id}/claim", app.ClaimChat)
	g.PUT("/api/contacts/{id}/release", app.ReleaseChat)
	g.POST("/api/contacts/bulk-release", app.BulkReleaseChats)
	g.PUT("/api/contacts/{id}/close", app.CloseChat)
	g.PUT("/api/contacts/{id}/reopen", app.ReopenChat)
	g.POST("/api/contacts/{id}/join", app.JoinChat)
	g.DELETE("/api/contacts/{id}/join", app.LeaveChat)
	g.DELETE("/api/contacts/{id}/collaborators/{user_id}", app.RemoveCollaborator)
	g.POST("/api/contacts/{id}/collaborators/{user_id}", app.InviteCollaborator)
}

// setupImportExportRoutes registers the generic import/export config + data endpoints.
func setupImportExportRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Generic Import/Export
	g.POST("/api/export", app.ExportData)
	g.POST("/api/import", app.ImportData)
	g.GET("/api/export/{table}/config", app.GetExportConfig)
	g.GET("/api/import/{table}/config", app.GetImportConfig)
}

// setupTagRoutes registers tag CRUD.
func setupTagRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Tags
	g.GET("/api/tags", app.ListTags)
	g.POST("/api/tags", app.CreateTag)
	g.PUT("/api/tags/{name}", app.UpdateTag)
	g.DELETE("/api/tags/{name}", app.DeleteTag)
}

// setupMessageRoutes registers message list/send/revoke/react/typing/read-state
// + status send + template/media send.
func setupMessageRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Messages
	g.GET("/api/contacts/{id}/messages", app.GetMessages)
	g.POST("/api/contacts/{id}/messages", app.SendMessage)
	g.POST("/api/contacts/{id}/mark-read", app.MarkContactRead)
	g.POST("/api/contacts/{id}/messages/{message_id}/reaction", app.SendReaction)
	g.POST("/api/contacts/{id}/messages/{message_id}/revoke", app.RevokeMessage)
	g.POST("/api/contacts/{id}/typing", app.SendTypingIndicator)
	g.POST("/api/messages", app.SendMessage) // Legacy route
	g.POST("/api/messages/template", app.SendTemplateMessage)
	g.POST("/api/messages/media", app.SendMediaMessage)
	g.POST("/api/status/send", app.SendStatus)
	g.PUT("/api/messages/{id}/read", app.MarkMessageRead)
}

// setupConversationNoteRoutes registers conversation-note CRUD.
func setupConversationNoteRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Conversation Notes
	g.GET("/api/contacts/{id}/notes", app.ListConversationNotes)
	g.POST("/api/contacts/{id}/notes", app.CreateConversationNote)
	g.PUT("/api/contacts/{id}/notes/{note_id}", app.UpdateConversationNote)
	g.DELETE("/api/contacts/{id}/notes/{note_id}", app.DeleteConversationNote)
}

// setupMediaRoutes registers media serve/zip/redownload endpoints.
func setupMediaRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Media (serves media files for messages, auth-protected)
	g.GET("/api/media/{message_id}", app.ServeMedia)
	// Media burst download — zips the media of the given message IDs together
	g.GET("/api/media/zip", app.ServeMediaZip)
	// Re-download a message's media from its provider (e.g. after a failed fetch)
	g.POST("/api/media/{message_id}/redownload", app.RedownloadMedia)
}

// setupTemplateRoutes registers local-template CRUD (no remote sync).
func setupTemplateRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Templates (local blueprints — full CRUD, no remote sync)
	g.GET("/api/templates", app.ListTemplates)
	g.POST("/api/templates", app.CreateTemplate)
	g.GET("/api/templates/{id}", app.GetTemplate)
	g.PUT("/api/templates/{id}", app.UpdateTemplate)
	g.DELETE("/api/templates/{id}", app.DeleteTemplate)
}

// setupCampaignRoutes registers bulk-campaign CRUD + lifecycle + recipients + media.
func setupCampaignRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Bulk Campaigns
	g.GET("/api/campaigns", app.ListCampaigns)
	g.POST("/api/campaigns", app.CreateCampaign)
	g.GET("/api/campaigns/{id}", app.GetCampaign)
	g.PUT("/api/campaigns/{id}", app.UpdateCampaign)
	g.DELETE("/api/campaigns/{id}", app.DeleteCampaign)
	g.POST("/api/campaigns/{id}/start", app.StartCampaign)
	g.POST("/api/campaigns/{id}/pause", app.PauseCampaign)
	g.POST("/api/campaigns/{id}/cancel", app.CancelCampaign)
	g.POST("/api/campaigns/{id}/retry-failed", app.RetryFailed)
	g.GET("/api/campaigns/{id}/progress", app.GetCampaign)
	g.POST("/api/campaigns/{id}/recipients/import", app.ImportRecipients)
	g.GET("/api/campaigns/{id}/recipients", app.GetCampaignRecipients)
	g.DELETE("/api/campaigns/{id}/recipients/{recipientId}", app.DeleteCampaignRecipient)
	g.POST("/api/campaigns/{id}/media", app.UploadCampaignMedia)
	g.GET("/api/campaigns/{id}/media", app.ServeCampaignMedia)
}

// setupTeamRoutes registers team CRUD + membership endpoints.
func setupTeamRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Teams (admin/manager - access control in handler)
	g.GET("/api/teams", app.ListTeams)
	g.POST("/api/teams", app.CreateTeam)
	g.GET("/api/teams/{id}", app.GetTeam)
	g.PUT("/api/teams/{id}", app.UpdateTeam)
	g.DELETE("/api/teams/{id}", app.DeleteTeam)
	g.GET("/api/teams/{id}/members", app.ListTeamMembers)
	g.POST("/api/teams/{id}/members", app.AddTeamMember)
	g.DELETE("/api/teams/{id}/members/{member_user_id}", app.RemoveTeamMember)
}

// setupAuditLogRoutes registers audit-log list/get endpoints.
func setupAuditLogRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Audit Logs
	g.GET("/api/audit-logs", app.ListAuditLogs)
	g.GET("/api/audit-logs/{id}", app.GetAuditLog)
}

// setupCannedResponseRoutes registers canned-response CRUD + usage-counter endpoint.
func setupCannedResponseRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Canned Responses
	g.GET("/api/canned-responses", app.ListCannedResponses)
	g.POST("/api/canned-responses", app.CreateCannedResponse)
	g.GET("/api/canned-responses/{id}", app.GetCannedResponse)
	g.PUT("/api/canned-responses/{id}", app.UpdateCannedResponse)
	g.DELETE("/api/canned-responses/{id}", app.DeleteCannedResponse)
	g.POST("/api/canned-responses/{id}/use", app.IncrementCannedResponseUsage)
}

// setupAnalyticsRoutes registers analytics endpoints.
func setupAnalyticsRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Sessions (admin/debug)

	// Analytics
	g.GET("/api/analytics/agents", app.GetAgentAnalytics)
}

// setupWidgetRoutes registers customizable-analytics widget endpoints.
func setupWidgetRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Widgets (customizable analytics)
	g.GET("/api/widgets", app.ListWidgets)
	g.POST("/api/widgets", app.CreateWidget)
	g.GET("/api/widgets/data-sources", app.GetWidgetDataSources)
	g.GET("/api/widgets/data", app.GetAllWidgetsData)
	g.GET("/api/widgets/{id}", app.GetWidget)
	g.PUT("/api/widgets/{id}", app.UpdateWidget)
	g.DELETE("/api/widgets/{id}", app.DeleteWidget)
	g.GET("/api/widgets/{id}/data", app.GetWidgetData)
	g.POST("/api/widgets/layout", app.SaveWidgetLayout)
}

// setupOrganizationRoutes registers org settings + organization CRUD/member endpoints.
func setupOrganizationRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Organization Settings
	g.GET("/api/org/settings", app.GetOrganizationSettings)
	g.PUT("/api/org/settings", app.UpdateOrganizationSettings)

	// Organizations
	g.GET("/api/organizations", app.ListOrganizations)
	g.POST("/api/organizations", app.CreateOrganization)
	g.POST("/api/organizations/members", app.AddOrganizationMember)
}

// setupSSOSettingsRoutes registers SSO settings endpoints (admin only).
func setupSSOSettingsRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// SSO Settings (admin only - enforced by middleware)
	g.GET("/api/settings/sso", app.GetSSOSettings)
	g.PUT("/api/settings/sso/{provider}", app.UpdateSSOProvider)
	g.DELETE("/api/settings/sso/{provider}", app.DeleteSSOProvider)
}

// setupWebhookRoutes registers outgoing-webhook CRUD + test endpoint.
func setupWebhookRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Webhooks
	g.GET("/api/webhooks", app.ListWebhooks)
	g.POST("/api/webhooks", app.CreateWebhook)
	g.GET("/api/webhooks/{id}", app.GetWebhook)
	g.PUT("/api/webhooks/{id}", app.UpdateWebhook)
	g.DELETE("/api/webhooks/{id}", app.DeleteWebhook)
	g.POST("/api/webhooks/{id}/test", app.TestWebhook)
}

// setupCustomActionRoutes registers custom-action CRUD + execute + redirect endpoints.
func setupCustomActionRoutes(g *fastglue.Fastglue, app *handlers.App) {
	// Custom Actions
	g.GET("/api/custom-actions", app.ListCustomActions)
	g.POST("/api/custom-actions", app.CreateCustomAction)
	g.GET("/api/custom-actions/{id}", app.GetCustomAction)
	g.PUT("/api/custom-actions/{id}", app.UpdateCustomAction)
	g.DELETE("/api/custom-actions/{id}", app.DeleteCustomAction)
	g.POST("/api/custom-actions/{id}/execute", app.ExecuteCustomAction)
	g.GET("/api/custom-actions/redirect/{token}", app.CustomActionRedirect)
}

// setupFrontendCatchAll registers the SPA catch-all when the frontend is
// embedded in the binary. The frontend.IsEmbedded() conditional is preserved
// — the catch-all is NOT registered in API-only mode.
func setupFrontendCatchAll(g *fastglue.Fastglue, lo logf.Logger, basePath string) {
	// Serve embedded frontend (SPA)
	if frontend.IsEmbedded() {
		lo.Info("Serving embedded frontend", "base_path", basePath)
		frontendHandler := frontend.Handler(basePath)
		// Catch-all for frontend routes
		g.GET("/{path:*}", func(r *fastglue.Request) error {
			frontendHandler(r.RequestCtx)
			return nil
		})
		g.GET("/", func(r *fastglue.Request) error {
			frontendHandler(r.RequestCtx)
			return nil
		})
	} else {
		lo.Info("Frontend not embedded, API-only mode")
	}
}

// withRateLimit wraps a handler with the rate limit middleware.
func withRateLimit(handler fastglue.FastRequestHandler, opts middleware.RateLimitOpts) fastglue.FastRequestHandler {
	rl := middleware.RateLimit(opts)
	return func(r *fastglue.Request) error {
		if rl(r) == nil {
			return nil // Rate limited — response already sent.
		}
		return handler(r)
	}
}

// corsWrapper wraps a handler with CORS support at the fasthttp level.
// This ensures CORS headers are set even for auto-handled OPTIONS requests.
func corsWrapper(next fasthttp.RequestHandler, allowedOrigins map[string]bool) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		origin := string(ctx.Request.Header.Peek("Origin"))

		if origin != "" && middleware.IsOriginAllowed(origin, allowedOrigins) {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
			ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		} else if len(allowedOrigins) == 0 && origin != "" {
			// Development: no whitelist configured
			ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
		}

		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Organization-ID, X-CSRF-Token")
		ctx.Response.Header.Set("Access-Control-Max-Age", "86400")

		// Handle preflight OPTIONS requests
		if string(ctx.Method()) == "OPTIONS" {
			ctx.SetStatusCode(fasthttp.StatusNoContent)
			return
		}

		next(ctx)
	}
}
