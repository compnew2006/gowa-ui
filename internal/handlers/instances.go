package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
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
	ErrorRatePercent      float64 `json:"error_rate_percent"`
	QueueDepth            int64   `json:"queue_depth"`
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
	restrictedInstanceID, err := a.getRestrictedInstanceForUser(orgID, userID)
	if err != nil {
		return nil, err
	}
	if restrictedInstanceID != nil {
		query = query.Where("id = ?", *restrictedInstanceID)
	}
	return query, nil
}

// CreateInstance creates a new WhatsApp instance
func (a *App) CreateInstance(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var req CreateInstanceRequest
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	name := normalizeInstanceName(req.Name)
	if name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name is required", nil, "name")
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

	if err := a.DB.Create(&instance).Error; err != nil {
		a.Log.Error("Failed to create instance", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create instance", nil, "")
	}

	return r.SendEnvelope(instance)
}

// ListInstances returns all instances for the organization
func (a *App) ListInstances(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	query := a.DB.Where("organization_id = ?", orgID)
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

	return r.SendEnvelope(instances)
}

// GetInstance returns a single instance
func (a *App) GetInstance(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := a.DB.Where("id = ? AND organization_id = ?", id, orgID)
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

	return r.SendEnvelope(instance)
}

// UpdateInstance updates an instance
func (a *App) UpdateInstance(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
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

	query := a.DB.Where("id = ? AND organization_id = ?", id, orgID)
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
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
	}
	if req.AutoReadReceipt != nil {
		updates["auto_read_receipt"] = *req.AutoReadReceipt
	}
	if req.Settings != nil {
		settings := waManager.EnsureInstanceSettingsDefaults(*req.Settings)
		if settingsErr := waManager.ValidateInstanceSettings(settings); settingsErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, settingsErr.Error(), nil, "settings")
		}
		updates["settings"] = settings
	}

	if len(updates) > 0 {
		if err := a.DB.Model(&instance).Updates(updates).Error; err != nil {
			a.Log.Error("Failed to update instance", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update instance", nil, "")
		}
	}

	return r.SendEnvelope(instance)
}

// DeleteInstance logs out/unlinks an instance and then deletes it.
func (a *App) DeleteInstance(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := a.DB.Where("id = ? AND organization_id = ?", id, orgID)
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

	if a.WhatsmeowManager == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Whatsmeow manager not initialized", nil, "")
	}

	// Ensure WhatsApp session is explicitly logged out before deleting the instance.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := a.WhatsmeowManager.Logout(ctx, instance.ID); err != nil {
		a.Log.Error("Failed to log out instance during deletion", "error", err, "instance_id", instance.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to log out instance", nil, "")
	}

	if err := a.DB.Delete(&instance).Error; err != nil {
		a.Log.Error("Failed to delete instance", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete instance", nil, "")
	}

	return r.SendEnvelope(map[string]string{"status": "deleted"})
}

// ConnectInstance initiates connection (and QR generation) for an instance
func (a *App) ConnectInstance(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := a.DB.Where("id = ? AND organization_id = ?", id, orgID)
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
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := a.DB.Where("id = ? AND organization_id = ?", id, orgID)
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
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := a.DB.Where("id = ? AND organization_id = ?", id, orgID)
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

// PairPhoneInstance requests a phone linking code for an unpaired instance.
func (a *App) PairPhoneInstance(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := a.DB.Where("id = ? AND organization_id = ?", id, orgID)
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

	if a.WhatsmeowManager == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Whatsmeow manager not initialized", nil, "")
	}

	var req PairPhoneInstanceRequest
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	phoneDigits := normalizePairingPhoneNumber(req.PhoneNumber)
	if phoneDigits == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "phone_number is required", nil, "phone_number")
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
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}

	query := a.DB.Where("id = ? AND organization_id = ?", id, orgID)
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
	if a.WhatsmeowManager != nil {
		current := a.WhatsmeowManager.GetInstanceHealth(instance.ID)
		health = InstanceHealthResponse{
			UptimeSeconds:         current.UptimeSeconds,
			MessagesSentToday:     current.MessagesSentToday,
			MessagesReceivedToday: current.MessagesReceivedToday,
			MessagesFailedToday:   current.MessagesFailedToday,
			ErrorRatePercent:      current.ErrorRatePercent,
			QueueDepth:            current.QueueDepth,
		}
	}

	return r.SendEnvelope(health)
}
