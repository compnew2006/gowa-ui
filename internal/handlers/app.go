package handlers

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shridarpatil/gowa-ui/internal/chatlifecycle"
	"github.com/shridarpatil/gowa-ui/internal/config"
	"github.com/shridarpatil/gowa-ui/internal/queue"
	"github.com/shridarpatil/gowa-ui/internal/websocket"
	"github.com/shridarpatil/gowa-ui/pkg/whatsapp"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// App holds all dependencies for handlers
type App struct {
	Config *config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	Log    logf.Logger
	// WARegistry resolves the GOWA provider per account.
	WARegistry        *whatsapp.Registry
	WSHub             *websocket.Hub
	Queue             queue.Queue
	CampaignSubCancel context.CancelFunc
	// HTTPClient is a shared HTTP client with connection pooling for external API calls
	HTTPClient *http.Client
	// ChatLifecycle owns the chat conversation state machine (claim/release/
	// close/reopen/join/leave/invite/remove) and its audit + system-message +
	// WS side effects. Handlers in chat_lifecycle.go are thin HTTP adapters
	// over this service. Nil only in tests that don't exercise chat lifecycle.
	ChatLifecycle *chatlifecycle.Service
	// wg tracks background goroutines for graceful shutdown
	wg sync.WaitGroup

	// gowaHistorySyncMu guards gowaHistoryLastSync, the per-account cooldown
	// state for automatic GOWA history backfills (see gowa_history_sync.go).
	gowaHistorySyncMu   sync.Mutex
	gowaHistoryLastSync map[uuid.UUID]time.Time
}

// WaitForBackgroundTasks blocks until all background goroutines complete.
// Call this during graceful shutdown to ensure all async work finishes.
func (a *App) WaitForBackgroundTasks() {
	a.wg.Wait()
}

// getOrgID extracts organization ID from request context (set by auth middleware)
// Super admins can override the org by passing X-Organization-ID header
// Super admins MUST select an organization - no "all organizations" view
func (a *App) getOrgID(r *fastglue.Request) (uuid.UUID, error) {
	// Get user's default organization ID from JWT
	var defaultOrgID uuid.UUID
	orgIDVal := r.RequestCtx.UserValue("organization_id")
	if orgIDVal == nil {
		return uuid.Nil, errors.New("organization_id not found in context")
	}
	switch v := orgIDVal.(type) {
	case uuid.UUID:
		defaultOrgID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, errors.New("organization_id is not a valid UUID")
		}
		defaultOrgID = parsed
	default:
		return uuid.Nil, errors.New("organization_id is not a valid UUID")
	}

	// Check for X-Organization-ID header to switch organizations
	userID, _ := r.RequestCtx.UserValue("user_id").(uuid.UUID)
	overrideOrgID := string(r.RequestCtx.Request.Header.Peek("X-Organization-ID"))
	if overrideOrgID != "" {
		parsedOrgID, err := uuid.Parse(overrideOrgID)
		if err == nil && parsedOrgID != defaultOrgID {
			if a.IsSuperAdmin(userID) {
				// Super admins can access any org
				var count int64
				if err := a.DB.Table("organizations").Where("id = ?", parsedOrgID).Count(&count).Error; err == nil && count > 0 {
					return parsedOrgID, nil
				}
			} else {
				// Non-super-admins can switch if they have membership
				var count int64
				if err := a.DB.Table("user_organizations").
					Where("user_id = ? AND organization_id = ? AND deleted_at IS NULL", userID, parsedOrgID).
					Count(&count).Error; err == nil && count > 0 {
					return parsedOrgID, nil
				}
			}
		}
	}

	return defaultOrgID, nil
}

// HealthCheck returns server health status
func (a *App) HealthCheck(r *fastglue.Request) error {
	return r.SendEnvelope(map[string]string{
		"status":  "ok",
		"service": "gowa-ui",
	})
}

// ReadyCheck returns server readiness status
func (a *App) ReadyCheck(r *fastglue.Request) error {
	// Check database connection
	sqlDB, err := a.DB.DB()
	if err != nil {
		a.Log.Error("Database connection error", "error", err)
		return r.SendErrorEnvelope(500, "Database connection error", nil, "")
	}
	if err := sqlDB.Ping(); err != nil {
		a.Log.Error("Database ping failed", "error", err)
		return r.SendErrorEnvelope(500, "Database ping failed", nil, "")
	}

	// Check Redis connection
	if err := a.Redis.Ping(r.RequestCtx).Err(); err != nil {
		a.Log.Error("Redis connection error", "error", err)
		return r.SendErrorEnvelope(500, "Redis connection error", nil, "")
	}

	return r.SendEnvelope(map[string]string{
		"status": "ready",
	})
}

// StartCampaignStatsSubscriber starts listening for campaign stats updates from Redis pub/sub
// and broadcasts them via WebSocket
func (a *App) StartCampaignStatsSubscriber() error {
	if a.WSHub == nil {
		a.Log.Warn("WebSocket hub not initialized, skipping campaign stats subscriber")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.CampaignSubCancel = cancel

	subscriber := queue.NewSubscriber(a.Redis, a.Log)

	err := subscriber.SubscribeCampaignStats(ctx, func(update *queue.CampaignStatsUpdate) {
		a.Log.Debug("Received campaign stats update from Redis",
			"campaign_id", update.CampaignID,
			"status", update.Status,
			"sent", update.SentCount,
		)

		// Broadcast to organization via WebSocket
		a.WSHub.BroadcastToOrg(update.OrganizationID, websocket.WSMessage{
			Type: websocket.TypeCampaignStatsUpdate,
			Payload: map[string]any{
				"campaign_id":     update.CampaignID,
				"status":          update.Status,
				"sent_count":      update.SentCount,
				"delivered_count": update.DeliveredCount,
				"read_count":      update.ReadCount,
				"failed_count":    update.FailedCount,
			},
		})
	})

	if err != nil {
		cancel()
		return err
	}

	a.Log.Info("Campaign stats subscriber started")
	return nil
}

// StopCampaignStatsSubscriber stops the campaign stats subscriber
func (a *App) StopCampaignStatsSubscriber() {
	if a.CampaignSubCancel != nil {
		a.CampaignSubCancel()
	}
}

// getOrgAndUserID extracts both organization ID and user ID from the request context.
// Returns an error if either is missing or invalid.
func (a *App) getOrgAndUserID(r *fastglue.Request) (orgID, userID uuid.UUID, err error) {
	orgID, err = a.getOrgID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	userIDVal := r.RequestCtx.UserValue("user_id")
	if userIDVal == nil {
		return uuid.Nil, uuid.Nil, errors.New("user_id not found in context")
	}
	switch v := userIDVal.(type) {
	case uuid.UUID:
		userID = v
	case string:
		userID, err = uuid.Parse(v)
		if err != nil {
			return uuid.Nil, uuid.Nil, errors.New("user_id is not a valid UUID")
		}
	default:
		return uuid.Nil, uuid.Nil, errors.New("user_id is not a valid UUID")
	}

	return orgID, userID, nil
}

// requireAuth extracts the organization ID and user ID from the request and
// verifies the user holds the given permission. On failure it writes the
// appropriate error envelope (401 if unauthenticated, 403 if the permission is
// missing) and returns errEnvelopeSent, so callers should `return nil` early.
func (a *App) requireAuth(r *fastglue.Request, resource, action string) (orgID, userID uuid.UUID, err error) {
	orgID, userID, err = a.getOrgAndUserID(r)
	if err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
		return uuid.Nil, uuid.Nil, errEnvelopeSent
	}
	if !a.HasPermission(userID, resource, action, orgID) {
		_ = r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
		return uuid.Nil, uuid.Nil, errEnvelopeSent
	}
	return orgID, userID, nil
}

// decodeRequest decodes a JSON request body into the provided struct.
// Returns nil on success, otherwise sends a 400 error envelope and returns errEnvelopeSent.
func (a *App) decodeRequest(r *fastglue.Request, v any) error {
	if err := r.Decode(v, "json"); err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
		return errEnvelopeSent
	}
	return nil
}
