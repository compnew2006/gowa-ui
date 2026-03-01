package whatsmeow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	"github.com/zerodha/logf"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waWa6"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"gorm.io/gorm"
)

// ConnectionManager manages whatsmeow clients for multiple instances
type ConnectionManager struct {
	db            *gorm.DB
	store         *sqlstore.Container
	clients       map[uuid.UUID]*whatsmeow.Client
	clientsMu     sync.RWMutex
	metrics       sync.Map // map[uuid.UUID]*instanceMetrics
	avatarSync    sync.Map // map[uuid.UUID]struct{}
	activeCallsMu sync.Mutex
	activeCallIDs map[uuid.UUID]map[string]struct{}
	logger        logf.Logger
	cfg           *config.WhatsmeowConfig
	hub           *websocket.Hub
	connectFn     func(context.Context, uuid.UUID) error
	qrCodesMu     sync.RWMutex
	qrCodes       map[uuid.UUID]cachedQRCode
	typingIndicator *typingIndicatorPlanner
	// mediaStoragePath is the local root directory where inbound media is persisted.
	mediaStoragePath string
}

type cachedQRCode struct {
	code       string
	timeoutSec int
	receivedAt time.Time
}

const orgAutoConnectBootstrapSettingsKey = "whatsmeow_auto_connect_bootstrap_done"

// NewConnectionManager creates a new ConnectionManager
func NewConnectionManager(db *gorm.DB, store *sqlstore.Container, logger logf.Logger, cfg *config.WhatsmeowConfig, hub *websocket.Hub, mediaStoragePath string) *ConnectionManager {
	if mediaStoragePath == "" {
		mediaStoragePath = "./uploads"
	}
	cm := &ConnectionManager{
		db:               db,
		store:            store,
		clients:          make(map[uuid.UUID]*whatsmeow.Client),
		logger:           logger,
		cfg:              cfg,
		hub:              hub,
		mediaStoragePath: mediaStoragePath,
		activeCallIDs:    make(map[uuid.UUID]map[string]struct{}),
		qrCodes:          make(map[uuid.UUID]cachedQRCode),
		typingIndicator:  newTypingIndicatorPlanner(cfg),
	}
	cm.connectFn = cm.Connect
	return cm
}

// Connect initializes and connects a WhatsApp instance
// If the instance is already connected, it returns nil immediately.
func (cm *ConnectionManager) Connect(ctx context.Context, instanceID uuid.UUID) error {
	// 1. Load instance from DB
	var instance models.WhatsAppInstance
	if err := cm.db.WithContext(ctx).First(&instance, "id = ?", instanceID).Error; err != nil {
		return fmt.Errorf("failed to load instance: %w", err)
	}

	cm.clientsMu.Lock()
	defer cm.clientsMu.Unlock()

	// 2. Reuse existing client when available.
	if existingClient, ok := cm.clients[instanceID]; ok {
		if existingClient.IsConnected() {
			newStatus := models.InstanceStatusConnecting
			if existingClient.Store != nil && existingClient.Store.ID != nil {
				newStatus = models.InstanceStatusConnected
				cm.ClearCachedQRCode(instanceID)
			}

			if err := cm.updateInstanceStatus(ctx, instanceID, newStatus); err != nil {
				cm.logger.Error("Failed to update instance status", "component", "whatsmeow", "event", "status_update_error", "error", err)
			}
			if newStatus == models.InstanceStatusConnected {
				cm.MarkConnected(instanceID)
				cm.broadcastInstanceConnected(instance.OrganizationID, instanceID, instance.PhoneNumber)
			}

			cm.logger.Debug("Instance already connected", "component", "whatsmeow", "event", "connect_skip", "instance_id", instanceID, "status", newStatus)
			return nil
		}

		if err := existingClient.Connect(); err != nil {
			return fmt.Errorf("failed to reconnect existing client: %w", err)
		}

		newStatus := models.InstanceStatusConnecting
		if existingClient.Store != nil && existingClient.Store.ID != nil {
			newStatus = models.InstanceStatusConnected
			cm.ClearCachedQRCode(instanceID)
		}

		if err := cm.updateInstanceStatus(ctx, instanceID, newStatus); err != nil {
			cm.logger.Error("Failed to update instance status", "component", "whatsmeow", "event", "status_update_error", "error", err)
		}
		if newStatus == models.InstanceStatusConnected {
			cm.MarkConnected(instanceID)
			cm.broadcastInstanceConnected(instance.OrganizationID, instanceID, instance.PhoneNumber)
		}

		cm.logger.Info("Instance reconnected using existing client", "component", "whatsmeow", "event", "reconnect_existing_client", "instance_id", instanceID, "status", newStatus)
		return nil
	}

	// 3. Load or create device in store
	var deviceStore *store.Device
	var err error

	if instance.JID != "" {
		var jid types.JID
		jid, err = types.ParseJID(instance.JID)
		if err != nil {
			return fmt.Errorf("invalid JID in database: %w", err)
		}

		deviceStore, err = cm.store.GetDevice(ctx, jid)
		if err != nil {
			return fmt.Errorf("failed to get device from store: %w", err)
		}
		if deviceStore == nil {
			cm.logger.Warn(
				"No stored device found for persisted JID, resetting instance identity for fresh pairing",
				"component", "whatsmeow",
				"event", "stale_device_identity",
				"instance_id", instanceID,
				"jid", instance.JID,
			)
			if clearErr := cm.updateInstanceIdentity(ctx, instanceID, "", ""); clearErr != nil {
				return fmt.Errorf("failed to reset stale instance identity: %w", clearErr)
			}
			instance.JID = ""
			instance.PhoneNumber = ""
			deviceStore = cm.store.NewDevice()
		}
	} else {
		// New device (not paired yet)
		deviceStore = cm.store.NewDevice()
	}

	if deviceStore == nil {
		return fmt.Errorf("device store is nil (should not happen)")
	}

	// 4. Create whatsmeow client
	// Use a sub-logger for whatsmeow
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	identityPrefix := ""
	if cm.cfg != nil {
		identityPrefix = cm.cfg.Identity
	}
	linkedDeviceName := buildLinkedDeviceName(identityPrefix, instance.Name, instance.ID)
	client.GetClientPayload = func() *waWa6.ClientPayload {
		payload := deviceStore.GetClientPayload()
		applyLinkedDeviceName(payload, linkedDeviceName)
		return payload
	}

	// 5. Register event handler
	// handleEvent will be defined in events.go (same package)
	client.AddEventHandler(func(evt interface{}) {
		cm.handleEvent(evt, instanceID, instance.OrganizationID)
	})

	// 6. Connect
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to WhatsApp: %w", err)
	}

	// 7. Store client
	cm.clients[instanceID] = client

	// 8. Update status to connecting (or connected if already logged in)
	newStatus := models.InstanceStatusConnecting
	if client.IsConnected() && client.Store != nil && client.Store.ID != nil {
		newStatus = models.InstanceStatusConnected
	}

	if err := cm.updateInstanceStatus(ctx, instanceID, newStatus); err != nil {
		cm.logger.Error("Failed to update instance status", "component", "whatsmeow", "event", "status_update_error", "error", err)
	}
	if newStatus == models.InstanceStatusConnected {
		cm.ClearCachedQRCode(instanceID)
		cm.MarkConnected(instanceID)
		cm.broadcastInstanceConnected(instance.OrganizationID, instanceID, instance.PhoneNumber)
	}

	cm.logger.Info("Instance connected", "component", "whatsmeow", "event", "connected", "instance_id", instanceID, "status", newStatus)
	return nil
}

// Disconnect disconnects a WhatsApp instance
func (cm *ConnectionManager) Disconnect(ctx context.Context, instanceID uuid.UUID) error {
	cm.clientsMu.Lock()
	defer cm.clientsMu.Unlock()

	client, ok := cm.clients[instanceID]
	if !ok {
		return nil // Already disconnected
	}

	client.Disconnect()
	delete(cm.clients, instanceID)
	cm.clearActiveCalls(instanceID)
	cm.ClearCachedQRCode(instanceID)

	if err := cm.updateInstanceStatus(ctx, instanceID, models.InstanceStatusDisconnected); err != nil {
		cm.logger.Error("Failed to update instance status on disconnect", "component", "whatsmeow", "event", "disconnect_error", "instance_id", instanceID, "error", err)
	}
	cm.MarkDisconnected(instanceID)

	cm.logger.Info("Instance disconnected", "component", "whatsmeow", "event", "disconnected", "instance_id", instanceID)
	return nil
}

// Logout unlinks the WhatsApp session and clears local runtime/session state.
func (cm *ConnectionManager) Logout(ctx context.Context, instanceID uuid.UUID) error {
	var instance models.WhatsAppInstance
	if err := cm.db.WithContext(ctx).
		Select("id", "organization_id").
		Where("id = ?", instanceID).
		First(&instance).Error; err != nil {
		return fmt.Errorf("failed to load instance: %w", err)
	}

	var logoutErr error

	cm.clientsMu.Lock()
	client, ok := cm.clients[instanceID]
	if ok && client != nil {
		if err := client.Logout(ctx); err != nil {
			logoutErr = err
			cm.logger.Warn("Failed graceful WhatsApp logout; forcing local cleanup", "component", "whatsmeow", "event", "logout_force_cleanup", "instance_id", instanceID, "error", err)
			client.Disconnect()
			if client.Store != nil {
				if deleteErr := client.Store.Delete(ctx); deleteErr != nil {
					cm.logger.Warn("Failed to delete local store during forced logout", "component", "whatsmeow", "event", "logout_store_delete_error", "instance_id", instanceID, "error", deleteErr)
				}
			}
		}
	}
	delete(cm.clients, instanceID)
	cm.clientsMu.Unlock()
	cm.clearActiveCalls(instanceID)
	cm.ClearCachedQRCode(instanceID)

	if err := cm.db.WithContext(ctx).Model(&models.WhatsAppInstance{}).
		Where("id = ?", instanceID).
		Updates(map[string]any{
			"status":       models.InstanceStatusLoggedOut,
			"jid":          "",
			"phone_number": "",
			"session_id":   "",
		}).Error; err != nil {
		return fmt.Errorf("failed to persist logged out state: %w", err)
	}
	cm.MarkDisconnected(instanceID)

	if cm.hub != nil {
		cm.hub.BroadcastToOrg(instance.OrganizationID, websocket.WSMessage{
			Type: websocket.TypeInstanceLoggedOut,
			Payload: websocket.InstancePayload{
				InstanceID: instanceID.String(),
				Status:     string(models.InstanceStatusLoggedOut),
			},
		})
	}

	cm.logger.Info("Instance logged out", "component", "whatsmeow", "event", "logout", "instance_id", instanceID)

	if logoutErr != nil {
		return fmt.Errorf("failed to logout WhatsApp session cleanly: %w", logoutErr)
	}

	return nil
}

// GetClient returns the whatsmeow client for an instance if connected
func (cm *ConnectionManager) GetClient(instanceID uuid.UUID) *whatsmeow.Client {
	cm.clientsMu.RLock()
	defer cm.clientsMu.RUnlock()
	return cm.clients[instanceID]
}

// updateInstanceStatus updates the status of an instance in the database
func (cm *ConnectionManager) updateInstanceStatus(ctx context.Context, instanceID uuid.UUID, status models.InstanceStatus) error {
	return cm.db.WithContext(ctx).Model(&models.WhatsAppInstance{}).
		Where("id = ?", instanceID).
		Update("status", status).Error
}

func (cm *ConnectionManager) updateInstanceSendBlock(ctx context.Context, instanceID uuid.UUID, blockedUntil *time.Time, reason string) error {
	updates := map[string]any{
		"send_blocked_until": blockedUntil,
		"send_block_reason":  strings.TrimSpace(reason),
	}
	return cm.db.WithContext(ctx).Model(&models.WhatsAppInstance{}).
		Where("id = ?", instanceID).
		Updates(updates).Error
}

func (cm *ConnectionManager) clearInstanceSendBlock(ctx context.Context, instanceID uuid.UUID) error {
	return cm.updateInstanceSendBlock(ctx, instanceID, nil, "")
}

// ReconcileStartupStatuses resets stale transient statuses after process restarts.
// At startup there are no in-memory clients yet, so non-linked instances left in
// "connecting" from a previous run must be moved back to "disconnected".
func (cm *ConnectionManager) ReconcileStartupStatuses(ctx context.Context) error {
	if cm.db == nil {
		return fmt.Errorf("database is not initialized")
	}

	result := cm.db.WithContext(ctx).
		Model(&models.WhatsAppInstance{}).
		Where("status = ? AND (jid = '' OR jid IS NULL)", models.InstanceStatusConnecting).
		Update("status", models.InstanceStatusDisconnected)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		cm.logger.Info(
			"Reset stale connecting instances after startup",
			"component", "whatsmeow",
			"event", "startup_status_reconcile",
			"count", result.RowsAffected,
		)
	}

	return nil
}

// ReconnectAll reconnects all instances marked as connected in the DB
// This should be called at startup.
func (cm *ConnectionManager) ReconnectAll(ctx context.Context) error {
	if err := cm.ReconcileStartupStatuses(ctx); err != nil {
		return fmt.Errorf("failed to reconcile startup statuses: %w", err)
	}

	var instances []models.WhatsAppInstance
	// Only reconnect instances that were connected or connecting, and have a JID (valid session)
	if err := cm.db.WithContext(ctx).Where("status IN ? AND jid != ''", []models.InstanceStatus{models.InstanceStatusConnected, models.InstanceStatusConnecting}).Find(&instances).Error; err != nil {
		return fmt.Errorf("failed to list instances for reconnect: %w", err)
	}

	connectInstance := cm.connectFn
	if connectInstance == nil {
		connectInstance = cm.Connect
	}

	for _, instance := range instances {
		cm.logger.Info("Reconnecting instance", "component", "whatsmeow", "event", "reconnect_start", "instance_id", instance.ID)
		if err := connectInstance(ctx, instance.ID); err != nil {
			cm.logger.Error("Failed to reconnect instance", "component", "whatsmeow", "event", "reconnect_error", "instance_id", instance.ID, "error", err)
			if statusErr := cm.updateInstanceStatus(ctx, instance.ID, models.InstanceStatusDisconnected); statusErr != nil {
				cm.logger.Error("Failed to set reconnect-failed instance status", "component", "whatsmeow", "event", "reconnect_status_update_error", "instance_id", instance.ID, "error", statusErr)
			}
			// Don't stop on single failure
		}
	}
	return nil
}

// AutoConnectLinkedInstancesOnFirstRun auto-connects linked instances (jid != ”)
// for organizations that haven't completed the first-run bootstrap yet.
func (cm *ConnectionManager) AutoConnectLinkedInstancesOnFirstRun(ctx context.Context) error {
	if cm.db == nil {
		return fmt.Errorf("database is not initialized")
	}

	bootstrapPendingFilter := fmt.Sprintf(`COALESCE(settings->>'%s', 'false') <> 'true'`, orgAutoConnectBootstrapSettingsKey)
	linkedInstancesSubQuery := cm.db.WithContext(ctx).
		Model(&models.WhatsAppInstance{}).
		Select("1").
		Where("whatsapp_instances.organization_id = organizations.id").
		Where("jid <> ''")

	var orgIDs []uuid.UUID
	if err := cm.db.WithContext(ctx).
		Model(&models.Organization{}).
		Where("EXISTS (?)", linkedInstancesSubQuery).
		Where(bootstrapPendingFilter).
		Pluck("id", &orgIDs).Error; err != nil {
		return fmt.Errorf("failed to list organizations for first-run auto-connect: %w", err)
	}

	if len(orgIDs) == 0 {
		return nil
	}

	connectInstance := cm.connectFn
	if connectInstance == nil {
		connectInstance = cm.Connect
	}

	hasFailures := false
	for _, orgID := range orgIDs {
		if err := ctx.Err(); err != nil {
			return err
		}

		var instances []models.WhatsAppInstance
		if err := cm.db.WithContext(ctx).
			Where("organization_id = ? AND jid <> '' AND status NOT IN ?", orgID, []models.InstanceStatus{models.InstanceStatusBanned, models.InstanceStatusLoggedOut}).
			Find(&instances).Error; err != nil {
			return fmt.Errorf("failed to list linked instances for organization %s: %w", orgID, err)
		}

		orgFailed := false
		for _, instance := range instances {
			cm.logger.Info("First-run auto-connect for linked instance", "component", "whatsmeow", "event", "bootstrap_autoconnect_start", "organization_id", orgID, "instance_id", instance.ID)
			if err := connectInstance(ctx, instance.ID); err != nil {
				hasFailures = true
				orgFailed = true
				cm.logger.Error("First-run auto-connect failed", "component", "whatsmeow", "event", "bootstrap_autoconnect_error", "organization_id", orgID, "instance_id", instance.ID, "error", err)
			}
		}

		if orgFailed {
			// Skip marker update so startup can retry this org on next run.
			continue
		}

		if err := cm.markOrgAutoConnectBootstrapDone(ctx, orgID); err != nil {
			return fmt.Errorf("failed to mark first-run auto-connect as complete for organization %s: %w", orgID, err)
		}
	}

	if hasFailures {
		return fmt.Errorf("first-run auto-connect completed with errors")
	}

	return nil
}

func (cm *ConnectionManager) markOrgAutoConnectBootstrapDone(ctx context.Context, orgID uuid.UUID) error {
	var org models.Organization
	if err := cm.db.WithContext(ctx).
		Select("id", "settings").
		Where("id = ?", orgID).
		First(&org).Error; err != nil {
		return err
	}

	if org.Settings == nil {
		org.Settings = models.JSONB{}
	}
	org.Settings[orgAutoConnectBootstrapSettingsKey] = true

	return cm.db.WithContext(ctx).
		Model(&models.Organization{}).
		Where("id = ?", orgID).
		Update("settings", org.Settings).Error
}
