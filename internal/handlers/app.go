package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/queue"
	objectstorage "github.com/compnew2006/whatomate/internal/storage"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

// App holds all dependencies for handlers
type App struct {
	Config            *config.Config
	DB                *gorm.DB
	Redis             *redis.Client
	Log               logf.Logger
	WhatsApp          *whatsapp.Client
	WhatsmeowStore    *sqlstore.Container
	WhatsmeowManager  *whatsmeow.ConnectionManager
	ObjectStorage     objectstorage.ObjectStorage
	WSHub             *websocket.Hub
	Queue             queue.Queue
	CampaignSubCancel context.CancelFunc
	// HTTPClient is a shared HTTP client with connection pooling for external API calls
	HTTPClient *http.Client
	// MessageProvider is the abstraction for sending messages (Meta or Whatsmeow)
	MessageProvider provider.MessageProvider
	// WhatsmeowContactResolver resolves ad-hoc chat recipients for WhatsMeow start-chat flows.
	WhatsmeowContactResolver WhatsmeowContactResolver
	// WhatsmeowQueue is the per-instance message queue for whatsmeow rate limiting
	WhatsmeowQueue *whatsmeow.QueueManager
	// License enforces host-bound activation and runtime quotas.
	License *license.Service
	// legacyMediaRestoreGroup deduplicates concurrent restore attempts per message within this process.
	legacyMediaRestoreGroup singleflight.Group
	// legacyMediaRestoreLimiter caps concurrent restore work across distinct messages.
	legacyMediaRestoreLimiter chan struct{}
	// legacyMediaRestoreLimiterOnce lazily initializes the limiter.
	legacyMediaRestoreLimiterOnce sync.Once
	// legacyMediaRestoreMetrics tracks in-process restore counters for observability.
	legacyMediaRestoreMetrics legacyMediaRestoreMetrics
	// wg tracks background goroutines for graceful shutdown
	wg sync.WaitGroup
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
	return tenant.ResolveOrganizationID(r, a.DB)
}

// HealthCheck returns server health status
func (a *App) HealthCheck(r *fastglue.Request) error {
	return r.SendEnvelope(map[string]string{
		"status":  "ok",
		"service": "whatomate",
	})
}

// ReadyCheck returns server readiness status
func (a *App) ReadyCheck(r *fastglue.Request) error {
	// Check database connection
	sqlDB, err := a.DB.DB()
	if err != nil {
		return r.SendErrorEnvelope(500, "Database connection error", nil, "")
	}
	if err := sqlDB.Ping(); err != nil {
		return r.SendErrorEnvelope(500, "Database ping failed", nil, "")
	}

	// Check Redis connection
	if err := a.Redis.Ping(r.RequestCtx).Err(); err != nil {
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
			Payload: map[string]interface{}{
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
	userID, ok := parseContextUUID(userIDVal)
	if !ok {
		return uuid.Nil, uuid.Nil, errors.New("user_id is not a valid UUID")
	}

	return orgID, userID, nil
}

func parseContextUUID(value any) (uuid.UUID, bool) {
	switch typed := value.(type) {
	case uuid.UUID:
		if typed == uuid.Nil {
			return uuid.Nil, false
		}
		return typed, true
	case string:
		parsed, err := uuid.Parse(strings.TrimSpace(typed))
		if err != nil || parsed == uuid.Nil {
			return uuid.Nil, false
		}
		return parsed, true
	default:
		return uuid.Nil, false
	}
}

func (a *App) requestDB(r *fastglue.Request) *gorm.DB {
	if db, ok := tenant.GetScopedDB(r); ok && db != nil {
		return db.Session(&gorm.Session{})
	}

	orgID, err := a.getOrgID(r)
	if err != nil {
		return a.DB.Session(&gorm.Session{})
	}

	return tenant.ScopedDB(a.DB, orgID)
}

type authenticatedRequest struct {
	DB     *gorm.DB
	OrgID  uuid.UUID
	UserID uuid.UUID
	Ctx    context.Context
	Cancel context.CancelFunc
}

func (a *App) requireAuthenticatedRequest(r *fastglue.Request, timeout time.Duration) (*authenticatedRequest, error) {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return nil, err
	}

	requestDB := a.requestDB(r)
	ctx := context.Context(r.RequestCtx)
	cancel := context.CancelFunc(func() {})
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(r.RequestCtx, timeout)
		requestDB = requestDB.WithContext(ctx)
	}

	return &authenticatedRequest{
		DB:     requestDB,
		OrgID:  orgID,
		UserID: userID,
		Ctx:    ctx,
		Cancel: cancel,
	}, nil
}

// requirePermission checks if the user has the required permission.
// Returns nil if permitted, otherwise sends a 403 error envelope and returns errEnvelopeSent.
// Automatically extracts orgID from the request for org-aware permission checks.
func (a *App) requirePermission(r *fastglue.Request, userID uuid.UUID, resource, action string) error {
	orgID, _ := a.getOrgID(r)
	if !a.HasPermission(userID, resource, action, orgID) {
		return a.sendForbidden(r, resource, action)
	}
	return nil
}

func (a *App) requireRequestPermission(r *fastglue.Request, resource, action string) (uuid.UUID, uuid.UUID, bool) {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
		return uuid.Nil, uuid.Nil, false
	}
	if err := a.requirePermission(r, userID, resource, action); err != nil {
		return uuid.Nil, uuid.Nil, false
	}
	return orgID, userID, true
}

// authorizeRequest combines authentication extraction and permission check into one call.
// Returns the org and user IDs, and whether the request is authorized.
// When ok is false, the error response has already been sent.
// authorizeRequest combines authentication extraction and permission check into one call.
// Returns the org and user IDs, and whether the request is authorized.
// When ok is false, the error response has already been sent.
func (a *App) authorizeRequest(r *fastglue.Request, resource, action string) (orgID, userID uuid.UUID, ok bool) {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
		return uuid.Nil, uuid.Nil, false
	}
	if !a.HasPermission(userID, resource, action, orgID) {
		_ = r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions: "+resource+":"+action, nil, "")
		return uuid.Nil, uuid.Nil, false
	}
	return orgID, userID, true
}

// sendForbidden sends a standardized 403 Forbidden response for permission failures.
// Use this instead of manually constructing SendErrorEnvelope(fasthttp.StatusForbidden, ...)
// to ensure consistent error messages across all endpoints.
func (a *App) sendForbidden(r *fastglue.Request, resource, action string) error {
	_ = r.SendErrorEnvelope(fasthttp.StatusForbidden,
		"Insufficient permissions: "+resource+":"+action, nil, "")
	return errEnvelopeSent
}

// decodeRequest decodes a JSON request body into the provided struct.
// Returns nil on success, otherwise sends a 400 error envelope and returns errEnvelopeSent.
func (a *App) decodeRequest(r *fastglue.Request, v interface{}) error {
	if err := r.Decode(v, "json"); err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
		return errEnvelopeSent
	}
	return nil
}
