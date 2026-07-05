package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/compnew2006/whatomate/internal/websocket"
	waManager "github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	waClient "go.mau.fi/whatsmeow"
	"gorm.io/gorm"
)

// CreateInstanceRequest represents the request body for creating an instance
type CreateInstanceRequest struct {
	Name            string       `json:"name"`
	IsDefault       bool         `json:"is_default"`
	AutoReadReceipt bool         `json:"auto_read_receipt"`
	Settings        models.JSONB `json:"settings"`
}

// UpdateInstanceRequest represents the request body for updating an instance
type UpdateInstanceRequest struct {
	Name            *string       `json:"name"`
	IsDefault       *bool         `json:"is_default"`
	AutoReadReceipt *bool         `json:"auto_read_receipt"`
	Settings        *models.JSONB `json:"settings"`
}

func mergeInstanceSettings(current models.JSONB, updates models.JSONB) models.JSONB {
	merged := make(models.JSONB, len(current)+len(updates))
	for key, value := range current {
		merged[key] = value
	}
	for key, value := range updates {
		merged[key] = value
	}
	return merged
}

// PairPhoneInstanceRequest represents request body for phone-code pairing.
type PairPhoneInstanceRequest struct {
	PhoneNumber          string `json:"phone_number"`
	ShowPushNotification *bool  `json:"show_push_notification"`
	ClientType           string `json:"client_type"`
	ClientDisplayName    string `json:"client_display_name"`
}

// PairPhoneInstanceResponse represents phone-code pairing response.
type PairPhoneInstanceResponse struct {
	Status      string `json:"status"`
	PairingCode string `json:"pairing_code"`
	PhoneNumber string `json:"phone_number"`
	TimeoutSec  int    `json:"timeout_seconds"`
}

// InstanceHealthResponse represents health stats for one instance.
type InstanceHealthResponse struct {
	UptimeSeconds         int64   `json:"uptime_seconds"`
	MessagesSentToday     uint64  `json:"messages_sent_today"`
	MessagesReceivedToday uint64  `json:"messages_received_today"`
	MessagesFailedToday   uint64  `json:"messages_failed_today"`
	EventsDroppedToday    uint64  `json:"events_dropped_today"`
	ErrorRatePercent      float64 `json:"error_rate_percent"`
	QueueDepth            int64   `json:"queue_depth"`
}

// InstanceQRCodeSnapshotResponse returns the latest cached QR code for an instance, if available.
type InstanceQRCodeSnapshotResponse struct {
	InstanceID string `json:"instance_id"`
	Available  bool   `json:"available"`
	QRCode     string `json:"qr_code,omitempty"`
	TimeoutSec int    `json:"timeout_seconds,omitempty"`
	ReceivedAt string `json:"received_at,omitempty"`
	ExpiresAt  string `json:"expires_at,omitempty"`
}

func (a *App) broadcastInstanceConnectFailure(orgID, instanceID uuid.UUID, reason, message string) {
	if a.WSHub == nil {
		return
	}

	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		trimmedReason = "connect_failed"
	}

	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		trimmedMessage = "Failed to connect this instance. Please try regenerating QR code."
	}

	a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type: websocket.TypeInstanceReconnectFailed,
		Payload: websocket.InstanceReconnectFailedPayload{
			InstanceID: instanceID.String(),
			Reason:     trimmedReason,
			Message:    trimmedMessage,
		},
	})
}

func (a *App) scopeInstancesQueryToUserRestriction(query *gorm.DB, orgID, userID uuid.UUID) (*gorm.DB, error) {
	restrictedInstanceIDs, err := a.getRestrictedInstancesForUser(orgID, userID)
	if err != nil {
		return nil, err
	}
	if len(restrictedInstanceIDs) > 0 {
		query = query.Where("id IN ?", restrictedInstanceIDs)
	}
	return query, nil
}

func (a *App) requireInstanceReadPermission(r *fastglue.Request, userID uuid.UUID) error {
	return a.requirePermission(r, userID, models.ResourceAccounts, models.ActionRead)
}

func (a *App) requireInstanceWritePermission(r *fastglue.Request, userID uuid.UUID) error {
	return a.requirePermission(r, userID, models.ResourceAccounts, models.ActionWrite)
}

func (a *App) requireInstanceDeletePermission(r *fastglue.Request, userID uuid.UUID) error {
	return a.requirePermission(r, userID, models.ResourceAccounts, models.ActionDelete)
}

// CreateInstance creates a new WhatsApp instance
func (a *App) CreateInstance(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceWritePermission(r, userID); err != nil {
		return nil
	}

	var req CreateInstanceRequest
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	name := normalizeInstanceName(req.Name)
	if name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name is required", nil, "name")
	}
	if !a.checkQuotaOrRespond(r, license.ResourceEndpoints, orgID) {
		return nil
	}

	settings := waManager.EnsureInstanceSettingsDefaults(req.Settings)
	if err := waManager.ValidateInstanceSettings(settings); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "settings")
	}

	nameTaken, err := a.isInstanceNameTaken(context.Background(), orgID, name, nil)
	if err != nil {
		a.Log.Error("Failed to validate instance name uniqueness", "error", err, "organization_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to validate instance name", nil, "")
	}
	if nameTaken {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Instance with this name already exists", nil, "name")
	}

	instance := models.WhatsAppInstance{
		OrganizationID:  orgID,
		Name:            name,
		IsDefault:       req.IsDefault,
		AutoReadReceipt: req.AutoReadReceipt,
		Settings:        settings,
		Status:          models.InstanceStatusDisconnected,
	}

	if err := requestDB.Create(&instance).Error; err != nil {
		a.Log.Error("Failed to create instance", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create instance", nil, "")
	}

	// GOWA provider: mirror the instance as a device on the GOWA server. On
	// failure we roll back the local row so the operator gets a clean error
	// and can retry — never leave an instance locally that GOWA doesn't know
	// about, otherwise all subsequent lifecycle calls would 404 on GOWA.
	if a.isGowaProvider() {
		provCtx, provCancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := a.gowaCreateDevice(provCtx, &instance); err != nil {
			provCancel()
			a.Log.Error("GOWA device provisioning failed; rolling back local instance",
				"error", err, "instance_id", instance.ID, "name", instance.Name)
			if delErr := requestDB.Delete(&instance).Error; delErr != nil {
				a.Log.Error("Failed to roll back local instance after GOWA provisioning failure",
					"error", delErr, "instance_id", instance.ID)
			}
			return r.SendErrorEnvelope(fasthttp.StatusBadGateway,
				"Failed to provision device on GOWA backend", nil, "")
		}
		provCancel()
	}

	return r.SendEnvelope(instance)
}

// ListInstances returns all instances for the organization
func (a *App) ListInstances(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceReadPermission(r, userID); err != nil {
		return nil
	}

	query := requestDB.Where("organization_id = ?", orgID)
	query, err = a.scopeInstancesQueryToUserRestriction(query, orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance for list", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list instances", nil, "")
	}

	var instances []models.WhatsAppInstance
	if err := query.Find(&instances).Error; err != nil {
		a.Log.Error("Failed to list instances", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list instances", nil, "")
	}

	for idx := range instances {
		instances[idx].Settings = waManager.EnsureInstanceSettingsDefaults(instances[idx].Settings)
	}

	return r.SendEnvelope(instances)
}

// GetInstance returns a single instance
func (a *App) GetInstance(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceReadPermission(r, userID); err != nil {
		return nil
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := requestDB.Where("id = ? AND organization_id = ?", id, orgID)
	query, err = a.scopeInstancesQueryToUserRestriction(query, orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance for read", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to get instance", nil, "")
	}

	var instance models.WhatsAppInstance
	if err := query.First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Instance not found", nil, "")
		}
		a.Log.Error("Failed to get instance", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to get instance", nil, "")
	}

	instance.Settings = waManager.EnsureInstanceSettingsDefaults(instance.Settings)

	return r.SendEnvelope(instance)
}

// UpdateInstance updates an instance
func (a *App) UpdateInstance(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceWritePermission(r, userID); err != nil {
		return nil
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	var req UpdateInstanceRequest
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	query := requestDB.Where("id = ? AND organization_id = ?", id, orgID)
	query, err = a.scopeInstancesQueryToUserRestriction(query, orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance for update", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	var instance models.WhatsAppInstance
	if err := query.First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Instance not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		normalizedName := normalizeInstanceName(*req.Name)
		if normalizedName == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name is required", nil, "name")
		}

		nameTaken, nameErr := a.isInstanceNameTaken(context.Background(), orgID, normalizedName, &instance.ID)
		if nameErr != nil {
			a.Log.Error("Failed to validate instance name uniqueness", "error", nameErr, "organization_id", orgID, "instance_id", instance.ID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to validate instance name", nil, "")
		}
		if nameTaken {
			return r.SendErrorEnvelope(fasthttp.StatusConflict, "Instance with this name already exists", nil, "name")
		}

		updates["name"] = normalizedName
		instance.Name = normalizedName
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
		instance.IsDefault = *req.IsDefault
	}
	if req.AutoReadReceipt != nil {
		updates["auto_read_receipt"] = *req.AutoReadReceipt
		instance.AutoReadReceipt = *req.AutoReadReceipt
	}
	if req.Settings != nil {
		settings := mergeInstanceSettings(instance.Settings, *req.Settings)
		settings = waManager.EnsureInstanceSettingsDefaults(settings)
		if settingsErr := waManager.ValidateInstanceSettings(settings); settingsErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, settingsErr.Error(), nil, "settings")
		}
		updates["settings"] = settings
		instance.Settings = settings
	}

	if len(updates) > 0 {
		updateQuery := tenant.ScopedDB(a.DB.Session(&gorm.Session{}), orgID).
			Model(&models.WhatsAppInstance{}).
			Where("id = ?", instance.ID)
		if err := updateQuery.Updates(updates).Error; err != nil {
			a.Log.Error("Failed to update instance", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update instance", nil, "")
		}
	}

	if req.Name != nil && a.WhatsmeowManager != nil {
		if err := a.WhatsmeowManager.ReindexInstance(instance); err != nil {
			a.Log.Error("Failed to reindex instance runtime connection", "error", err, "instance_id", instance.ID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to refresh runtime connection", nil, "")
		}
	}

	instance.Settings = waManager.EnsureInstanceSettingsDefaults(instance.Settings)

	return r.SendEnvelope(instance)
}

// DeleteInstance logs out/unlinks an instance and then deletes it.
func (a *App) DeleteInstance(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceDeletePermission(r, userID); err != nil {
		return nil
	}

	deleteChats, err := parseDeleteChatsQueryFlag(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "delete_chats")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := requestDB.Where("id = ? AND organization_id = ?", id, orgID)
	query, err = a.scopeInstancesQueryToUserRestriction(query, orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance for delete", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	var instance models.WhatsAppInstance
	if err := query.First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Instance not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	if a.WhatsmeowManager == nil && !a.isGowaProvider() {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Whatsmeow manager not initialized", nil, "")
	}

	// Ensure WhatsApp session is explicitly logged out before deleting the instance.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if a.isGowaProvider() {
		// Best-effort GOWA purge: log any failure but proceed with local
		// cleanup. The local row is the source of truth, and a stale GOWA
		// device is preferable to a hanging delete request that traps the
		// user's quota. gowaDeleteDevice already logs.
		a.gowaDeleteDevice(ctx, &instance)
	} else {
		if err := a.WhatsmeowManager.Logout(ctx, instance.ID); err != nil {
			a.Log.Warn("Failed to log out WhatsApp session cleanly during deletion, proceeding with deletion", "error", err, "instance_id", instance.ID)
		}
	}

	if err := a.deleteWhatsAppInstanceWithOptionalChatPurge(&instance, orgID, deleteChats); err != nil {
		a.Log.Error("Failed to delete instance", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete instance", nil, "")
	}

	return r.SendEnvelope(map[string]string{"status": "deleted"})
}

// ConnectInstance initiates connection (and QR generation) for an instance
func (a *App) ConnectInstance(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceWritePermission(r, userID); err != nil {
		return nil
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := requestDB.Where("id = ? AND organization_id = ?", id, orgID)
	query, err = a.scopeInstancesQueryToUserRestriction(query, orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance for connect", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	// Verify ownership
	var instance models.WhatsAppInstance
	if err := query.First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Instance not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	if a.WhatsmeowManager == nil {
		if a.isGowaProvider() {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
				"Use POST /api/instances/{id}/connect on the GOWA provider (provider=gowa does not use this endpoint)", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Whatsmeow manager not initialized", nil, "")
	}

	// Trigger connection asynchronously
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := a.WhatsmeowManager.Connect(ctx, instance.ID); err != nil {
			a.Log.Error("Failed to connect instance", "error", err, "instance_id", instance.ID)
			a.broadcastInstanceConnectFailure(
				instance.OrganizationID,
				instance.ID,
				"connect_failed",
				"Failed to establish WhatsApp connection. Please retry or regenerate QR code.",
			)
		}
	}()

	return r.SendEnvelope(map[string]string{"status": "connection_initiated"})
}

// DisconnectInstance disconnects an instance
func (a *App) DisconnectInstance(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceWritePermission(r, userID); err != nil {
		return nil
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := requestDB.Where("id = ? AND organization_id = ?", id, orgID)
	query, err = a.scopeInstancesQueryToUserRestriction(query, orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance for disconnect", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	// Verify ownership
	var instance models.WhatsAppInstance
	if err := query.First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Instance not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	if a.WhatsmeowManager == nil {
		if a.isGowaProvider() {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
				"Use POST /api/instances/{id}/disconnect on the GOWA provider (provider=gowa does not use this endpoint)", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Whatsmeow manager not initialized", nil, "")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := a.WhatsmeowManager.Logout(ctx, instance.ID); err != nil {
		a.Log.Error("Failed to log out instance", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to log out instance", nil, "")
	}

	return r.SendEnvelope(map[string]string{"status": "logged_out"})
}

// ReconnectInstance reconnects an instance
func (a *App) ReconnectInstance(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceWritePermission(r, userID); err != nil {
		return nil
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := requestDB.Where("id = ? AND organization_id = ?", id, orgID)
	query, err = a.scopeInstancesQueryToUserRestriction(query, orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance for reconnect", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	var instance models.WhatsAppInstance
	if err := query.First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Instance not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	if a.WhatsmeowManager == nil {
		if a.isGowaProvider() {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
				"Use POST /api/instances/{id}/reconnect on the GOWA provider (provider=gowa does not use this endpoint)", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Whatsmeow manager not initialized", nil, "")
	}

	disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer disconnectCancel()
	if err := a.WhatsmeowManager.Disconnect(disconnectCtx, instance.ID); err != nil {
		a.Log.Warn("Failed to disconnect instance before reconnect", "error", err, "instance_id", instance.ID)
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := a.WhatsmeowManager.Connect(ctx, instance.ID); err != nil {
			a.Log.Error("Failed to reconnect instance", "error", err, "instance_id", instance.ID)
			a.broadcastInstanceConnectFailure(
				instance.OrganizationID,
				instance.ID,
				"reconnect_failed",
				"Failed to refresh WhatsApp connection. Please regenerate QR code and try again.",
			)
		}
	}()

	return r.SendEnvelope(map[string]string{"status": "reconnection_initiated"})
}

// GetInstanceQRCodeSnapshot returns the latest cached QR code (if still valid) for an instance.
func (a *App) GetInstanceQRCodeSnapshot(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceReadPermission(r, userID); err != nil {
		return nil
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := requestDB.Where("id = ? AND organization_id = ?", id, orgID)
	query, err = a.scopeInstancesQueryToUserRestriction(query, orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance for qr snapshot", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	var instance models.WhatsAppInstance
	if err := query.Select("id").First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Instance not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	if a.WhatsmeowManager == nil {
		if a.isGowaProvider() {
			// GOWA does not cache QR codes server-side the way whatsmeow does;
			// every QR request triggers a fresh GET /devices/:id/login. Point
			// the caller at the live endpoint instead of returning an empty
			// snapshot.
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
				"Use GET /api/instances/{id}/qr on the GOWA provider to fetch a live QR code", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Whatsmeow manager not initialized", nil, "")
	}

	snapshot, ok := a.WhatsmeowManager.GetCachedQRCode(instance.ID)
	if !ok {
		return r.SendEnvelope(InstanceQRCodeSnapshotResponse{
			InstanceID: instance.ID.String(),
			Available:  false,
		})
	}

	receivedAt := snapshot.ReceivedAt.UTC()
	expiresAt := receivedAt.Add(time.Duration(snapshot.TimeoutSec) * time.Second)
	return r.SendEnvelope(InstanceQRCodeSnapshotResponse{
		InstanceID: instance.ID.String(),
		Available:  true,
		QRCode:     snapshot.Code,
		TimeoutSec: snapshot.TimeoutSec,
		ReceivedAt: receivedAt.Format(time.RFC3339),
		ExpiresAt:  expiresAt.Format(time.RFC3339),
	})
}

// PairPhoneInstance requests a phone linking code for an unpaired instance.
func (a *App) PairPhoneInstance(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceWritePermission(r, userID); err != nil {
		return nil
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := requestDB.Where("id = ? AND organization_id = ?", id, orgID)
	query, err = a.scopeInstancesQueryToUserRestriction(query, orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance for pairing", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	var instance models.WhatsAppInstance
	if err := query.First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Instance not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	var req PairPhoneInstanceRequest
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	phoneDigits := normalizePairingPhoneNumber(req.PhoneNumber)
	if phoneDigits == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "phone_number is required", nil, "phone_number")
	}

	if a.isGowaProvider() {
		if a.GowaClient == nil {
			return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA client not initialized", nil, "")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		pairResp, err := a.GowaClient.LoginWithCode(ctx, gowaDeviceID(&instance), phoneDigits)
		if err != nil {
			a.Log.Error("GOWA pair failed", "error", err, "instance_id", instance.ID)
			return gowaSendError(r, err, "Failed to request pairing code from GOWA")
		}

		if err := a.DB.Model(&models.WhatsAppInstance{}).
			Where("id = ?", instance.ID).
			Update("status", models.InstanceStatusConnecting).Error; err != nil {
			a.Log.Warn("Failed to update instance status for GOWA pairing", "error", err, "instance_id", instance.ID)
		}

		return r.SendEnvelope(PairPhoneInstanceResponse{
			Status:      "pair_code_generated",
			PairingCode: pairResp.PairCode,
			PhoneNumber: phoneDigits,
			TimeoutSec:  60,
		})
	}

	if a.WhatsmeowManager == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Whatsmeow manager not initialized", nil, "")
	}

	opts := waManager.DefaultPairPhoneOptions()
	if req.ShowPushNotification != nil {
		opts.ShowPushNotification = *req.ShowPushNotification
	}

	clientType, ok := parsePairClientType(req.ClientType)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid client_type", nil, "client_type")
	}
	opts.ClientType = clientType
	opts.ClientDisplayName = normalizePairClientDisplayName(
		req.ClientDisplayName,
		buildPairClientDisplayName(instance.Name, instance.ID),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	code, pairErr := a.WhatsmeowManager.RequestPhonePairingCode(ctx, instance.ID, phoneDigits, opts)
	if pairErr != nil {
		switch {
		case errors.Is(pairErr, waClient.ErrPhoneNumberTooShort),
			errors.Is(pairErr, waClient.ErrPhoneNumberIsNotInternational):
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, pairErr.Error(), nil, "phone_number")
		case strings.Contains(strings.ToLower(pairErr.Error()), "already paired"):
			return r.SendErrorEnvelope(fasthttp.StatusConflict, "Instance is already paired. Disconnect/log out before requesting a new pairing code.", nil, "")
		default:
			a.Log.Error("Failed to request phone pairing code", "error", pairErr, "instance_id", instance.ID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to request phone pairing code", nil, "")
		}
	}

	return r.SendEnvelope(PairPhoneInstanceResponse{
		Status:      "pair_code_generated",
		PairingCode: code,
		PhoneNumber: phoneDigits,
		TimeoutSec:  waManager.PhonePairingTimeoutSec(),
	})
}

// GetInstanceHealth returns runtime health metrics for an instance.
func (a *App) GetInstanceHealth(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceReadPermission(r, userID); err != nil {
		return nil
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := requestDB.Where("id = ? AND organization_id = ?", id, orgID)
	query, err = a.scopeInstancesQueryToUserRestriction(query, orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance for health", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	var instance models.WhatsAppInstance
	if err := query.First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Instance not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}

	health := InstanceHealthResponse{}
	if a.isGowaProvider() && a.GowaClient != nil {
		// GOWA exposes only connected/logged_in booleans, not the rich
		// counters whatsmeow tracks. Surface the live state as the connected
		// flag and leave counters at zero — the operator can read richer
		// metrics directly from the GOWA server if needed.
		statusCtx, statusCancel := context.WithTimeout(r.RequestCtx, 10*time.Second)
		status, err := a.GowaClient.GetStatus(statusCtx, gowaDeviceID(&instance))
		statusCancel()
		if err != nil {
			a.Log.Warn("GOWA status probe failed during health check", "error", err, "instance_id", instance.ID)
		} else {
			if status.IsLoggedIn {
				health.UptimeSeconds = 1 // nonzero signals "live"
			}
		}
	} else if a.WhatsmeowManager != nil {
		current := a.WhatsmeowManager.GetInstanceHealth(instance.ID)
		health = InstanceHealthResponse{
			UptimeSeconds:         current.UptimeSeconds,
			MessagesSentToday:     current.MessagesSentToday,
			MessagesReceivedToday: current.MessagesReceivedToday,
			MessagesFailedToday:   current.MessagesFailedToday,
			EventsDroppedToday:    current.EventsDroppedToday,
			ErrorRatePercent:      current.ErrorRatePercent,
			QueueDepth:            current.QueueDepth,
		}
	}

	return r.SendEnvelope(health)
}
