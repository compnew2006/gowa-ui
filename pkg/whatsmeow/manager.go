package whatsmeow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
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

const (
	orgAutoConnectBootstrapSettingsKey = "whatsmeow_auto_connect_bootstrap_done"
	defaultHealthMonitorInterval       = 30 * time.Second
	defaultReconnectTimeout            = 45 * time.Second
	baseReconnectBackoff               = 5 * time.Second
	maxReconnectBackoff                = 5 * time.Minute
)

// ConnectionManager manages whatsmeow clients for multiple instances.
type ConnectionManager struct {
	db    *gorm.DB
	store *sqlstore.Container
	pool  *ConnectionPool

	metrics       sync.Map // map[uuid.UUID]*instanceMetrics
	avatarSync    sync.Map // map[uuid.UUID]struct{}
	activeCallsMu sync.Mutex
	activeCallIDs map[uuid.UUID]map[string]struct{}
	logger        logf.Logger
	cfg           *config.WhatsmeowConfig
	hub           *websocket.Hub
	connectFn     func(context.Context, uuid.UUID) error

	qrCodesMu       sync.RWMutex
	qrCodes         map[uuid.UUID]cachedQRCode
	typingIndicator *typingIndicatorPlanner
	// mediaStoragePath is the local root directory where inbound media is persisted.
	mediaStoragePath  string
	inboundMediaQueue inboundMediaJobEnqueuer

	healthMonitorMu     sync.Mutex
	healthMonitorCancel context.CancelFunc
	healthMonitorDone   chan struct{}
	mediaService        *MediaService

	// disableAvatarSync allows tests to bypass avatar sync to avoid panics on mock clients
	disableAvatarSync bool
}

type inboundMediaJobEnqueuer interface {
	EnqueueInboundMedia(ctx context.Context, job *queue.InboundMediaJob) error
}

type cachedQRCode struct {
	code       string
	timeoutSec int
	receivedAt time.Time
}

// NewConnectionManager creates a new ConnectionManager.
func NewConnectionManager(db *gorm.DB, store *sqlstore.Container, logger logf.Logger, cfg *config.WhatsmeowConfig, hub *websocket.Hub, mediaStoragePath string) *ConnectionManager {
	if mediaStoragePath == "" {
		mediaStoragePath = "./uploads"
	}

	cm := &ConnectionManager{
		db:               db,
		store:            store,
		pool:             NewConnectionPool(),
		logger:           logger,
		cfg:              cfg,
		hub:              hub,
		mediaStoragePath: mediaStoragePath,
		activeCallIDs:    make(map[uuid.UUID]map[string]struct{}),
		qrCodes:          make(map[uuid.UUID]cachedQRCode),
		typingIndicator:  newTypingIndicatorPlanner(cfg),
	}
	if cm.typingIndicator != nil {
		cm.typingIndicator.warn = logger.Warn
	}
	cm.connectFn = cm.Connect
	return cm
}

// SetInboundMediaQueue configures the queue used for async inbound-media recovery jobs.
func (cm *ConnectionManager) SetInboundMediaQueue(q inboundMediaJobEnqueuer) {
	if cm == nil {
		return
	}
	cm.inboundMediaQueue = q
}

// SetMediaService configures the zero-disk inbound media pipeline.
func (cm *ConnectionManager) SetMediaService(service *MediaService) {
	if cm == nil {
		return
	}
	cm.mediaService = service
}

// Connect initializes and connects a WhatsApp instance.
// If the instance is already connected, it returns nil immediately.
func (cm *ConnectionManager) Connect(ctx context.Context, instanceID uuid.UUID) error {
	instance, err := cm.loadInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to load instance: %w", err)
	}

	entry, err := cm.ensurePoolEntry(ctx, instance)
	if err != nil {
		return err
	}

	entry.connectMu.Lock()
	defer entry.connectMu.Unlock()

	if existingClient := entry.client(); existingClient != nil {
		return cm.connectExistingClient(ctx, instance, existingClient)
	}

	return cm.connectNewClient(ctx, instance, entry)
}

// Disconnect disconnects a WhatsApp instance.
func (cm *ConnectionManager) Disconnect(ctx context.Context, instanceID uuid.UUID) error {
	if cm == nil || cm.pool == nil {
		return nil
	}
	if cm.pool.entry(instanceID) == nil {
		return nil
	}

	client := cm.pool.removeInstance(instanceID)
	if client != nil {
		client.Disconnect()
	}
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
	client := cm.pool.removeInstance(instanceID)
	if client != nil {
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

// GetClient returns the whatsmeow client for an instance if connected.
func (cm *ConnectionManager) GetClient(instanceID uuid.UUID) *whatsmeow.Client {
	if cm == nil || cm.pool == nil {
		return nil
	}
	return cm.pool.GetClient(instanceID)
}

// GetClientByKey returns the runtime client bound to the tenant/account key.
func (cm *ConnectionManager) GetClientByKey(key InstanceKey) *whatsmeow.Client {
	if cm == nil || cm.pool == nil {
		return nil
	}
	return cm.pool.GetClientByKey(key)
}

// RegisterInstanceClient registers a runtime client inside the tenant-aware pool.
func (cm *ConnectionManager) RegisterInstanceClient(instance models.WhatsAppInstance, client *whatsmeow.Client) error {
	if cm == nil || cm.pool == nil {
		return fmt.Errorf("connection pool is not initialized")
	}
	if err := cm.pool.RegisterInstanceClient(instance, client); err != nil {
		return err
	}
	if client != nil && client.IsConnected() {
		cm.pool.markConnected(instance.ID, cm.connectedPhoneNumber(instance.PhoneNumber, client))
	}
	return nil
}

// ReindexInstance updates the runtime tenant/account key after instance metadata changes.
func (cm *ConnectionManager) ReindexInstance(instance models.WhatsAppInstance) error {
	if cm == nil || cm.pool == nil {
		return nil
	}
	if cm.pool.entry(instance.ID) == nil {
		return nil
	}

	_, err := cm.ensurePoolEntry(context.Background(), instance)
	return err
}

// StartHealthMonitor launches the runtime reconnect worker for managed instances.
func (cm *ConnectionManager) StartHealthMonitor(ctx context.Context) {
	if cm == nil {
		return
	}

	cm.healthMonitorMu.Lock()
	defer cm.healthMonitorMu.Unlock()

	if cm.healthMonitorCancel != nil {
		return
	}

	baseCtx := context.Background()
	if ctx != nil {
		baseCtx = ctx
	}

	monitorCtx, cancel := context.WithCancel(baseCtx)
	done := make(chan struct{})
	cm.healthMonitorCancel = cancel
	cm.healthMonitorDone = done

	go func() {
		defer close(done)
		cm.healthMonitor(monitorCtx)
	}()
}

// StopHealthMonitor stops the runtime reconnect worker.
func (cm *ConnectionManager) StopHealthMonitor() {
	if cm == nil {
		return
	}

	cm.healthMonitorMu.Lock()
	cancel := cm.healthMonitorCancel
	done := cm.healthMonitorDone
	cm.healthMonitorCancel = nil
	cm.healthMonitorDone = nil
	cm.healthMonitorMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// updateInstanceStatus updates the status of an instance in the database.
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

// ReconnectAll reconnects all instances marked as connected in the DB.
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
			// Don't stop on single failure.
		}
	}
	return nil
}

// AutoConnectLinkedInstancesOnFirstRun auto-connects linked instances (jid != "")
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

func (cm *ConnectionManager) loadInstance(ctx context.Context, instanceID uuid.UUID) (models.WhatsAppInstance, error) {
	var instance models.WhatsAppInstance
	if err := cm.db.WithContext(ctx).First(&instance, "id = ?", instanceID).Error; err != nil {
		return models.WhatsAppInstance{}, err
	}
	return instance, nil
}

func (cm *ConnectionManager) ensurePoolEntry(ctx context.Context, instance models.WhatsAppInstance) (*connectionEntry, error) {
	if cm.pool == nil {
		return nil, fmt.Errorf("connection pool is not initialized")
	}

	for attempts := 0; attempts < 2; attempts++ {
		entry, conflictID, err := cm.pool.ensureEntry(instance)
		if err != nil {
			return nil, err
		}
		if conflictID == uuid.Nil {
			return entry, nil
		}

		resolved, resolveErr := cm.resolvePoolConflict(ctx, conflictID)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if !resolved {
			key := NewInstanceKey(instance.OrganizationID, instance.Name)
			return nil, fmt.Errorf("instance key %q for organization %s is already owned by instance %s", key.AccountName, key.OrganizationID, conflictID)
		}
	}

	key := NewInstanceKey(instance.OrganizationID, instance.Name)
	return nil, fmt.Errorf("failed to resolve runtime connection key %q for organization %s", key.AccountName, key.OrganizationID)
}

func (cm *ConnectionManager) resolvePoolConflict(ctx context.Context, conflictID uuid.UUID) (bool, error) {
	if cm.pool == nil {
		return false, fmt.Errorf("connection pool is not initialized")
	}

	var conflictInstance models.WhatsAppInstance
	err := cm.db.WithContext(ctx).
		Select("id", "organization_id", "name", "phone_number").
		Where("id = ?", conflictID).
		First(&conflictInstance).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			client := cm.pool.removeInstance(conflictID)
			if client != nil {
				client.Disconnect()
			}
			return true, nil
		}
		return false, fmt.Errorf("failed to inspect conflicting instance %s: %w", conflictID, err)
	}

	if err := cm.pool.ReindexInstance(conflictInstance); err != nil {
		return false, fmt.Errorf("failed to reindex conflicting instance %s: %w", conflictID, err)
	}

	return true, nil
}

func (cm *ConnectionManager) connectExistingClient(ctx context.Context, instance models.WhatsAppInstance, client *whatsmeow.Client) error {
	if client.IsConnected() {
		cm.syncRuntimeEntry(instance.ID, client, instance.PhoneNumber)
		cm.markInstanceConnectionState(ctx, instance, client)
		cm.logger.Debug("Instance already connected", "component", "whatsmeow", "event", "connect_skip", "instance_id", instance.ID)
		return nil
	}

	if err := client.Connect(); err != nil {
		cm.pool.markDisconnected(instance.ID)
		return fmt.Errorf("failed to reconnect existing client: %w", err)
	}

	cm.syncRuntimeEntry(instance.ID, client, instance.PhoneNumber)
	cm.markInstanceConnectionState(ctx, instance, client)
	cm.logger.Info("Instance reconnected using existing client", "component", "whatsmeow", "event", "reconnect_existing_client", "instance_id", instance.ID)
	return nil
}

func (cm *ConnectionManager) connectNewClient(ctx context.Context, instance models.WhatsAppInstance, entry *connectionEntry) error {
	deviceStore, err := cm.resolveDeviceStore(ctx, &instance)
	if err != nil {
		return err
	}
	if deviceStore == nil {
		return fmt.Errorf("device store is nil (should not happen)")
	}

	client := cm.newClient(instance, deviceStore)
	if err := client.Connect(); err != nil {
		cm.pool.removeInstance(instance.ID)
		return fmt.Errorf("failed to connect to WhatsApp: %w", err)
	}

	entry.attachClient(client, instance.PhoneNumber)
	cm.syncRuntimeEntry(instance.ID, client, instance.PhoneNumber)
	cm.markInstanceConnectionState(ctx, instance, client)

	cm.logger.Info("Instance connected", "component", "whatsmeow", "event", "connected", "instance_id", instance.ID)
	return nil
}

func (cm *ConnectionManager) resolveDeviceStore(ctx context.Context, instance *models.WhatsAppInstance) (*store.Device, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	}

	var deviceStore *store.Device
	var err error

	if instance.JID != "" {
		var jid types.JID
		jid, err = types.ParseJID(instance.JID)
		if err != nil {
			return nil, fmt.Errorf("invalid JID in database: %w", err)
		}

		deviceStore, err = cm.store.GetDevice(ctx, jid)
		if err != nil {
			return nil, fmt.Errorf("failed to get device from store: %w", err)
		}
		if deviceStore == nil {
			cm.logger.Warn(
				"No stored device found for persisted JID, resetting instance identity for fresh pairing",
				"component", "whatsmeow",
				"event", "stale_device_identity",
				"instance_id", instance.ID,
				"jid", instance.JID,
			)
			if clearErr := cm.updateInstanceIdentity(ctx, instance.ID, "", ""); clearErr != nil {
				return nil, fmt.Errorf("failed to reset stale instance identity: %w", clearErr)
			}
			instance.JID = ""
			instance.PhoneNumber = ""
			deviceStore = cm.store.NewDevice()
		}
	} else {
		// New device (not paired yet).
		deviceStore = cm.store.NewDevice()
	}

	return deviceStore, nil
}

func (cm *ConnectionManager) newClient(instance models.WhatsAppInstance, deviceStore *store.Device) *whatsmeow.Client {
	clientLog := newClientLogger(waLog.Stdout("Client", "DEBUG", true))
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

	client.AddEventHandler(func(evt interface{}) {
		cm.handleEvent(evt, instance.ID, instance.OrganizationID)
	})

	return client
}

func (cm *ConnectionManager) syncRuntimeEntry(instanceID uuid.UUID, client *whatsmeow.Client, phoneNumber string) {
	if cm.pool == nil {
		return
	}
	entry := cm.pool.entry(instanceID)
	if entry == nil {
		return
	}
	entry.attachClient(client, phoneNumber)
}

func (cm *ConnectionManager) markInstanceConnectionState(ctx context.Context, instance models.WhatsAppInstance, client *whatsmeow.Client) {
	newStatus := models.InstanceStatusConnecting
	phoneNumber := cm.connectedPhoneNumber(instance.PhoneNumber, client)
	if client != nil && client.Store != nil && client.Store.ID != nil {
		newStatus = models.InstanceStatusConnected
		cm.ClearCachedQRCode(instance.ID)
	}

	if err := cm.updateInstanceStatus(ctx, instance.ID, newStatus); err != nil {
		cm.logger.Error("Failed to update instance status", "component", "whatsmeow", "event", "status_update_error", "instance_id", instance.ID, "error", err)
	}

	if newStatus == models.InstanceStatusConnected {
		cm.pool.markConnected(instance.ID, phoneNumber)
		cm.MarkConnected(instance.ID)
		cm.broadcastInstanceConnected(instance.OrganizationID, instance.ID, phoneNumber)
	}
}

func (cm *ConnectionManager) connectedPhoneNumber(current string, client *whatsmeow.Client) string {
	phoneNumber := strings.TrimSpace(current)
	if client != nil && client.Store != nil && client.Store.ID != nil {
		if candidate := strings.TrimSpace(client.Store.ID.User); candidate != "" {
			phoneNumber = candidate
		}
	}
	return phoneNumber
}

func (cm *ConnectionManager) healthMonitor(ctx context.Context) {
	cm.runHealthMonitorPass(ctx)

	ticker := time.NewTicker(cm.healthMonitorInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.runHealthMonitorPass(ctx)
		}
	}
}

func (cm *ConnectionManager) runHealthMonitorPass(ctx context.Context) {
	if cm == nil || cm.pool == nil || cm.db == nil {
		return
	}

	connectInstance := cm.connectFn
	if connectInstance == nil {
		connectInstance = cm.Connect
	}

	for _, entry := range cm.pool.snapshotEntries() {
		if entry == nil {
			continue
		}

		instance, err := cm.loadInstance(ctx, entry.InstanceID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				client := cm.pool.removeInstance(entry.InstanceID)
				if client != nil {
					client.Disconnect()
				}
				continue
			}
			cm.logger.Warn("Health monitor failed to load instance", "component", "whatsmeow", "event", "health_monitor_load_error", "instance_id", entry.InstanceID, "error", err)
			continue
		}

		if err := cm.ReindexInstance(instance); err != nil {
			cm.logger.Warn("Health monitor failed to reindex instance", "component", "whatsmeow", "event", "health_monitor_reindex_error", "instance_id", instance.ID, "error", err)
			continue
		}

		client := entry.client()
		if !cm.shouldHealthMonitorReconnect(instance, client) {
			if instance.Status == models.InstanceStatusLoggedOut || instance.Status == models.InstanceStatusBanned || (strings.TrimSpace(instance.JID) == "" && (client == nil || !client.IsConnected())) {
				removedClient := cm.pool.removeInstance(instance.ID)
				if removedClient != nil && removedClient != client {
					removedClient.Disconnect()
				}
			}
			continue
		}

		if client == nil {
			continue
		}
		if client.IsConnected() {
			cm.pool.markConnected(instance.ID, cm.connectedPhoneNumber(instance.PhoneNumber, client))
			continue
		}
		if !entry.beginReconnect(time.Now().UTC(), baseReconnectBackoff, maxReconnectBackoff) {
			continue
		}

		reconnectCtx, cancel := context.WithTimeout(ctx, cm.reconnectTimeout())
		err = connectInstance(reconnectCtx, instance.ID)
		cancel()
		entry.finishReconnect(err, instance.PhoneNumber)
		if err != nil {
			cm.logger.Warn("Health monitor reconnect failed", "component", "whatsmeow", "event", "health_monitor_reconnect_failed", "instance_id", instance.ID, "error", err)
		}
	}
}

func (cm *ConnectionManager) shouldHealthMonitorReconnect(instance models.WhatsAppInstance, client *whatsmeow.Client) bool {
	if client == nil {
		return false
	}
	if strings.TrimSpace(instance.JID) == "" {
		return false
	}
	switch instance.Status {
	case models.InstanceStatusLoggedOut, models.InstanceStatusBanned:
		return false
	default:
		return true
	}
}

func (cm *ConnectionManager) healthMonitorInterval() time.Duration {
	if cm != nil && cm.cfg != nil && cm.cfg.HealthMonitorIntervalSeconds > 0 {
		return time.Duration(cm.cfg.HealthMonitorIntervalSeconds) * time.Second
	}
	return defaultHealthMonitorInterval
}

func (cm *ConnectionManager) reconnectTimeout() time.Duration {
	if cm != nil && cm.cfg != nil && cm.cfg.ReconnectTimeoutSeconds > 0 {
		return time.Duration(cm.cfg.ReconnectTimeoutSeconds) * time.Second
	}
	return defaultReconnectTimeout
}
