package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
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
	// InboundDLQ stores failed inbound messages for retry with exponential backoff.
	InboundDLQ *queue.InboundDLQ
	// OutgoingRetryQueue stores failed outgoing messages for retry with exponential backoff.
	OutgoingRetryQueue *queue.OutgoingRetryQueue
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
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Database connection error", nil, "")
	}
	if err := sqlDB.Ping(); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Database ping failed", nil, "")
	}

	// Check Redis connection
	if err := a.Redis.Ping(r.RequestCtx).Err(); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Redis connection error", nil, "")
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
		return db
	}

	orgID, err := a.getOrgID(r)
	if err != nil {
		return a.DB
	}

	return tenant.ScopedDB(a.DB, orgID)
}

// requirePermission checks if the user has the required permission.
// Returns nil if permitted, otherwise sends a 403 error envelope and returns errEnvelopeSent.
// Automatically extracts orgID from the request for org-aware permission checks.
func (a *App) requirePermission(r *fastglue.Request, userID uuid.UUID, resource, action string) error {
	orgID, _ := a.getOrgID(r)
	if !a.HasPermission(userID, resource, action, orgID) {
		_ = r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
		return errEnvelopeSent
	}
	return nil
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

// RetryOutgoingMessage re-sends a failed outgoing message by loading the
// persisted message and contact from the database and routing through the
// active provider (whatsmeow or Meta).
func (a *App) RetryOutgoingMessage(ctx context.Context, msgID uuid.UUID) error {
	var msg models.Message
	if err := a.DB.WithContext(ctx).First(&msg, msgID).Error; err != nil {
		return queue.NewPermanentError(fmt.Errorf("retry outgoing: message not found: %w", err))
	}

	if msg.Status != models.MessageStatusFailed {
		a.Log.Debug("Outgoing retry: message not in failed status, skipping",
			"message_id", msgID,
			"status", msg.Status,
		)
		return nil
	}

	var contact models.Contact
	if err := a.DB.WithContext(ctx).First(&contact, msg.ContactID).Error; err != nil {
		return queue.NewPermanentError(fmt.Errorf("retry outgoing: contact not found: %w", err))
	}

	orgID := msg.OrganizationID

	if a.isWhatsmeowProvider() && a.MessageProvider != nil {
		if msg.InstanceID == nil {
			return fmt.Errorf("retry outgoing: missing instance_id for message %s", msgID)
		}
		instanceID := msg.InstanceID.String()
		to := contact.PhoneNumber

		if _, err := a.retrySendViaProvider(ctx, &msg, instanceID, to); err != nil {
			return err
		}
	} else {
		waAccount, err := a.resolveWhatsAppAccountByOrg(ctx, orgID, msg.WhatsAppAccount)
		if err != nil {
			return fmt.Errorf("retry outgoing: resolve account: %w", err)
		}
		account := a.toWhatsAppAccount(waAccount)

		if _, err := a.retrySendViaMeta(ctx, &msg, account, contact.PhoneNumber); err != nil {
			return err
		}
	}

	a.DB.Model(&models.Message{}).Where("id = ?", msg.ID).Updates(map[string]any{
		"status":        models.MessageStatusSent,
		"error_message": "",
	})
	msg.Status = models.MessageStatusSent

	a.Log.Info("Outgoing retry: message re-sent successfully",
		"message_id", msgID,
		"type", msg.MessageType,
	)
	return nil
}

func (a *App) retrySendViaProvider(ctx context.Context, msg *models.Message, instanceID, to string) (string, error) {
	switch msg.MessageType {
	case models.MessageTypeText:
		return a.MessageProvider.SendText(ctx, instanceID, to, msg.Content)
	case models.MessageTypeImage:
		return a.MessageProvider.SendImage(ctx, instanceID, to, msg.MediaURL, msg.Content)
	case models.MessageTypeVideo:
		return a.MessageProvider.SendVideo(ctx, instanceID, to, msg.MediaURL, msg.Content)
	case models.MessageTypeAudio:
		return a.MessageProvider.SendAudio(ctx, instanceID, to, msg.MediaURL)
	case models.MessageTypeDocument:
		return a.MessageProvider.SendDocument(ctx, instanceID, to, msg.MediaURL, msg.MediaFilename, msg.Content)
	case models.MessageTypeInteractive:
		fallbackText := msg.Content
		if fallbackText == "" {
			fallbackText = msg.Content
		}
		return a.MessageProvider.SendText(ctx, instanceID, to, fallbackText)
	case models.MessageTypeTemplate:
		return a.MessageProvider.SendText(ctx, instanceID, to, msg.Content)
	default:
		return "", fmt.Errorf("unsupported message type for retry: %s", msg.MessageType)
	}
}

func (a *App) retrySendViaMeta(ctx context.Context, msg *models.Message, account *whatsapp.Account, to string) (string, error) {
	switch msg.MessageType {
	case models.MessageTypeText:
		return a.WhatsApp.SendTextMessage(ctx, account, to, msg.Content, "")
	case models.MessageTypeImage:
		return a.WhatsApp.SendImageMessage(ctx, account, to, msg.MediaURL, msg.Content)
	case models.MessageTypeVideo:
		return a.WhatsApp.SendVideoMessage(ctx, account, to, msg.MediaURL, msg.Content)
	case models.MessageTypeAudio:
		return a.WhatsApp.SendAudioMessage(ctx, account, to, msg.MediaURL)
	case models.MessageTypeDocument:
		return a.WhatsApp.SendDocumentMessage(ctx, account, to, msg.MediaURL, msg.MediaFilename, msg.Content)
	case models.MessageTypeTemplate:
		return a.WhatsApp.SendTemplateMessage(ctx, account, to, msg.TemplateName, "", nil)
	default:
		return "", fmt.Errorf("unsupported message type for retry: %s", msg.MessageType)
	}
}

// resolveWhatsAppAccountByOrg finds a WhatsApp account by org ID and optional name.
func (a *App) resolveWhatsAppAccountByOrg(ctx context.Context, orgID uuid.UUID, accountName string) (*models.WhatsAppAccount, error) {
	if accountName != "" {
		var account models.WhatsAppAccount
		if err := a.DB.WithContext(ctx).Where("organization_id = ? AND name = ?", orgID, accountName).First(&account).Error; err != nil {
			return nil, fmt.Errorf("whatsapp account %q not found: %w", accountName, err)
		}
		return &account, nil
	}

	var account models.WhatsAppAccount
	if err := a.DB.WithContext(ctx).Where("organization_id = ?", orgID).First(&account).Error; err != nil {
		return nil, fmt.Errorf("no whatsapp account found for org %s: %w", orgID, err)
	}
	return &account, nil
}

func (a *App) RetryInboundDLQEntry(ctx context.Context, entry *queue.InboundDLQEntry) error {
	var msg IncomingTextMessage
	if err := json.Unmarshal(entry.RawMessage, &msg); err != nil {
		return queue.NewPermanentError(fmt.Errorf("dlq retry: unmarshal message: %w", err))
	}

	var count int64
	a.DB.WithContext(ctx).Model(&models.Message{}).Where("whats_app_message_id = ?", msg.ID).Count(&count)
	if count > 0 {
		a.Log.Info("DLQ retry: message already exists, skipping", "wa_msg_id", msg.ID)
		return nil
	}

	a.processIncomingMessageFull(entry.PhoneNumberID, msg, entry.ProfileName)

	a.DB.WithContext(ctx).Model(&models.Message{}).Where("whats_app_message_id = ?", msg.ID).Count(&count)
	if count == 0 {
		return fmt.Errorf("dlq retry: message still not saved after re-processing (wa_msg_id=%s)", msg.ID)
	}

	return nil
}
