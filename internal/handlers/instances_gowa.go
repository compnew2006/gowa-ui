package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/gowa"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// isGowaProvider returns true if the configured provider is the GOWA HTTP
// backend. Mirrors the existing isWhatsmeowProvider() helper.
func (a *App) isGowaProvider() bool {
	return a.Config != nil && a.Config.WhatsApp.Provider == "gowa"
}

// gowaDeviceID returns the device_id whatomate uses to address this instance
// on the GOWA server. We use instance.Name (operator-chosen, unique per org)
// rather than the UUID because GOWA's device_id is a free-form string that
// reads better in logs and GOWA's UI. If Name is empty we fall back to the
// UUID string so the lookup never returns an unusable identifier.
func gowaDeviceID(instance *models.WhatsAppInstance) string {
	if instance == nil {
		return ""
	}
	if instance.Name != "" {
		return instance.Name
	}
	return instance.ID.String()
}

// gowaWebhookURLForInstance builds the per-instance callback URL that GOWA
// will POST inbound events to. We append the instance UUID as a path segment
// so the receiver can resolve the source instance without parsing JSON first.
func (a *App) gowaWebhookURLForInstance(instanceID uuid.UUID) string {
	base := a.Config.Gowa.WebhookCallbackURL
	if base == "" {
		base = "/api/gowa/webhook"
	}
	return fmt.Sprintf("%s/api/gowa/webhook/%s", base, instanceID.String())
}

// ----- Lifecycle helpers (called from instances.go) -----

// gowaCreateDevice provisions a device slot on the GOWA server for the given
// instance. It is called after the instance row is committed locally so a
// GOWA failure never leaves an orphan local row, and a local failure never
// orphans a device on GOWA (we delete the local row on provisioning failure
// to keep the two systems consistent — the operator can retry).
//
// The webhook_url is wired so GOWA immediately starts pushing inbound events
// for this device back to whatomate. WebhookSecret is the shared HMAC key
// validated on receipt (Stage 6).
func (a *App) gowaCreateDevice(ctx context.Context, instance *models.WhatsAppInstance) error {
	if a.GowaClient == nil {
		return errors.New("gowa client not initialized")
	}
	req := gowa.CreateDeviceRequest{
		DeviceID:      gowaDeviceID(instance),
		WebhookURL:    a.gowaWebhookURLForInstance(instance.ID),
		WebhookSecret: a.Config.Gowa.WebhookSecret,
	}
	dev, err := a.GowaClient.CreateDevice(ctx, req)
	if err != nil {
		return fmt.Errorf("gowa: create device %q: %w", req.DeviceID, err)
	}
	a.Log.Info("GOWA device provisioned",
		"instance_id", instance.ID,
		"device_id", dev.ID,
		"state", dev.State,
	)
	return nil
}

// gowaDeleteDevice purges the device from the GOWA server. It is best-effort
// during instance deletion: a failure is logged but does not block local
// cleanup, because the local row is the source of truth and a stale GOWA
// device is preferable to a hanging delete request.
func (a *App) gowaDeleteDevice(ctx context.Context, instance *models.WhatsAppInstance) {
	if a.GowaClient == nil {
		return
	}
	deviceID := gowaDeviceID(instance)
	if err := a.GowaClient.DeleteDevice(ctx, deviceID); err != nil {
		a.Log.Warn("GOWA device delete failed during instance deletion; proceeding with local cleanup",
			"instance_id", instance.ID, "device_id", deviceID, "error", err)
		return
	}
	a.Log.Info("GOWA device purged", "instance_id", instance.ID, "device_id", deviceID)
}

// gowaFetchStatus queries the GOWA device status and maps it to a whatomate
// InstanceStatus. Used by GetInstanceHealth and any code that needs the
// current connection state.
func (a *App) gowaFetchStatus(ctx context.Context, instance *models.WhatsAppInstance) (models.InstanceStatus, error) {
	if a.GowaClient == nil {
		return models.InstanceStatusDisconnected, errors.New("gowa client not initialized")
	}
	status, err := a.GowaClient.GetStatus(ctx, gowaDeviceID(instance))
	if err != nil {
		return models.InstanceStatusDisconnected, err
	}
	switch {
	case status.IsLoggedIn:
		return models.InstanceStatusConnected, nil
	case status.IsConnected:
		return models.InstanceStatusConnecting, nil
	default:
		return models.InstanceStatusDisconnected, nil
	}
}

// ----- GOWA-mode lifecycle handlers -----
//
// These are standalone fastglue handlers invoked instead of the whatsmeow
// branches when provider=gowa. They are registered in main.go under the same
// /api/instances/* paths but only wired when cfg.WhatsApp.Provider == "gowa".

// GowaGetInstanceQR fetches a fresh QR code from GOWA for the given instance.
// GET /api/instances/{id}/qr
func (a *App) GowaGetInstanceQR(r *fastglue.Request) error {
	if !a.isGowaProvider() || a.GowaClient == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not active", nil, "")
	}
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceReadPermission(r, userID); err != nil {
		return nil
	}
	instance, glucErr := a.fetchOwnedInstance(r, orgID, userID)
	if glucErr != nil {
		return glucErr
	}

	ctx, cancel := context.WithTimeout(r.RequestCtx, 30*time.Second)
	defer cancel()
	login, err := a.gowaGetLoginQRWithSelfHealing(ctx, instance)
	if err != nil {
		a.Log.Error("GOWA login QR failed", "error", err, "instance_id", instance.ID)
		return gowaSendError(r, err, "Failed to fetch QR code from GOWA")
	}
	return r.SendEnvelope(map[string]any{
		"instance_id": instance.ID.String(),
		"qr_link":     login.QRLink,
		"qr_duration": login.QRDuration,
		"device_id":   login.DeviceID,
	})
}

// GowaConnectInstance initiates pairing on GOWA. In GOWA, "connect" means
// asking for a QR or pair code; the device slot must already exist.
// POST /api/instances/{id}/connect
func (a *App) GowaConnectInstance(r *fastglue.Request) error {
	if !a.isGowaProvider() || a.GowaClient == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not active", nil, "")
	}
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceWritePermission(r, userID); err != nil {
		return nil
	}
	instance, glucErr := a.fetchOwnedInstance(r, orgID, userID)
	if glucErr != nil {
		return glucErr
	}

	ctx, cancel := context.WithTimeout(r.RequestCtx, 30*time.Second)
	defer cancel()
	login, err := a.gowaGetLoginQRWithSelfHealing(ctx, instance)
	if err != nil {
		a.Log.Error("GOWA connect failed", "error", err, "instance_id", instance.ID)
		return gowaSendError(r, err, "Failed to initiate GOWA connection")
	}

	// Mark locally as connecting so the UI reflects the in-flight pairing.
	if err := a.DB.Model(&models.WhatsAppInstance{}).
		Where("id = ?", instance.ID).
		Update("status", models.InstanceStatusConnecting).Error; err != nil {
		a.Log.Warn("Failed to mark instance as connecting", "error", err, "instance_id", instance.ID)
	}

	return r.SendEnvelope(map[string]any{
		"status":      "connection_initiated",
		"qr_link":     login.QRLink,
		"qr_duration": login.QRDuration,
	})
}

// GowaDisconnectInstance logs out the GOWA device (preserving the slot, per
// GOWA's Logout semantics — unlike whatsmeow-mode whatomate which used the
// destructive Logout call).
// POST /api/instances/{id}/disconnect
func (a *App) GowaDisconnectInstance(r *fastglue.Request) error {
	if !a.isGowaProvider() || a.GowaClient == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not active", nil, "")
	}
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceWritePermission(r, userID); err != nil {
		return nil
	}
	instance, glucErr := a.fetchOwnedInstance(r, orgID, userID)
	if glucErr != nil {
		return glucErr
	}

	ctx, cancel := context.WithTimeout(r.RequestCtx, 20*time.Second)
	defer cancel()
	if err := a.GowaClient.Logout(ctx, gowaDeviceID(instance)); err != nil {
		a.Log.Error("GOWA logout failed", "error", err, "instance_id", instance.ID)
		return gowaSendError(r, err, "Failed to log out GOWA device")
	}

	// Reflect the disconnected state locally.
	if err := a.DB.Model(&models.WhatsAppInstance{}).
		Where("id = ?", instance.ID).
		Updates(map[string]any{
			"status":      models.InstanceStatusDisconnected,
			"jid":         "",
			"phone_number": "",
		}).Error; err != nil {
		a.Log.Warn("Failed to update instance status after GOWA logout", "error", err, "instance_id", instance.ID)
	}

	return r.SendEnvelope(map[string]string{"status": "logged_out"})
}

// GowaReconnectInstance forces a fresh connection on the GOWA side.
// POST /api/instances/{id}/reconnect
func (a *App) GowaReconnectInstance(r *fastglue.Request) error {
	if !a.isGowaProvider() || a.GowaClient == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not active", nil, "")
	}
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceWritePermission(r, userID); err != nil {
		return nil
	}
	instance, glucErr := a.fetchOwnedInstance(r, orgID, userID)
	if glucErr != nil {
		return glucErr
	}

	ctx, cancel := context.WithTimeout(r.RequestCtx, 30*time.Second)
	defer cancel()
	if err := a.GowaClient.Reconnect(ctx, gowaDeviceID(instance)); err != nil {
		a.Log.Error("GOWA reconnect failed", "error", err, "instance_id", instance.ID)
		return gowaSendError(r, err, "Failed to reconnect GOWA device")
	}
	return r.SendEnvelope(map[string]string{"status": "reconnection_initiated"})
}

// GowaGetInstanceHealth returns live status from the GOWA device.
// GET /api/instances/{id}/health
func (a *App) GowaGetInstanceHealth(r *fastglue.Request) error {
	if !a.isGowaProvider() || a.GowaClient == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not active", nil, "")
	}
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceReadPermission(r, userID); err != nil {
		return nil
	}
	instance, glucErr := a.fetchOwnedInstance(r, orgID, userID)
	if glucErr != nil {
		return glucErr
	}

	ctx, cancel := context.WithTimeout(r.RequestCtx, 15*time.Second)
	defer cancel()
	status, err := a.GowaClient.GetStatus(ctx, gowaDeviceID(instance))
	if err != nil {
		a.Log.Error("GOWA status failed", "error", err, "instance_id", instance.ID)
		return gowaSendError(r, err, "Failed to fetch GOWA device status")
	}
	// Return the SAME InstanceHealthResponse shape the whatsmeow path returns,
	// so the embedded frontend's instance page (which calls .toFixed() on
	// error_rate_percent and reads uptime_seconds etc.) does not crash.
	// GOWA only exposes connected/logged_in booleans, so we map them to the
	// numeric fields: nonzero uptime = live; counters stay at zero (the
	// operator can read richer metrics from the GOWA side directly).
	health := InstanceHealthResponse{}
	if status.IsLoggedIn {
		health.UptimeSeconds = 1
	}
	return r.SendEnvelope(health)
}

// GowaListDevicesForAdmin is a debug/admin endpoint that proxies the raw GOWA
// device list. Useful for operators verifying the GOWA side during setup.
// GET /api/gowa/devices
func (a *App) GowaListDevicesForAdmin(r *fastglue.Request) error {
	if !a.isGowaProvider() || a.GowaClient == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not active", nil, "")
	}
	ctx, cancel := context.WithTimeout(r.RequestCtx, 15*time.Second)
	defer cancel()
	devices, err := a.GowaClient.ListDevices(ctx)
	if err != nil {
		return gowaSendError(r, err, "Failed to list GOWA devices")
	}
	return r.SendEnvelope(devices)
}

// ----- shared helpers -----

// fetchOwnedInstance loads a single instance by path :id, scoped to the org
// and the user's instance restrictions. Returns a fastglue error envelope
// already formatted for the client on failure.
func (a *App) fetchOwnedInstance(r *fastglue.Request, orgID, userID uuid.UUID) (*models.WhatsAppInstance, error) {
	idStr, ok := r.RequestCtx.UserValue("id").(string)
	if !ok || idStr == "" {
		return nil, r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Missing instance id", nil, "")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID", nil, "")
	}
	query := a.requestDB(r).Where("id = ? AND organization_id = ?", id, orgID)
	query, err = a.scopeInstancesQueryToUserRestriction(query, orgID, userID)
	if err != nil {
		a.Log.Error("Failed to resolve restricted instance", "error", err, "org_id", orgID, "user_id", userID)
		return nil, r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}
	var instance models.WhatsAppInstance
	if err := query.First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, r.SendErrorEnvelope(fasthttp.StatusNotFound, "Instance not found", nil, "")
		}
		return nil, r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch instance", nil, "")
	}
	return &instance, nil
}

// gowaSendError maps a gowa.Error to an HTTP error envelope. Network failures
// (no status code) become 502 Bad Gateway; GOWA's own status code is preserved
// for everything else so 404s and 409s surface correctly to the API client.
func gowaSendError(r *fastglue.Request, err error, fallback string) error {
	var ge *gowa.Error
	if errors.As(err, &ge) {
		status := ge.StatusCode
		if status == 0 {
			// Transport-level failure — GOWA unreachable.
			status = fasthttp.StatusBadGateway
		}
		msg := ge.Message
		if msg == "" {
			msg = fallback
		}
		return r.SendErrorEnvelope(status, msg, nil, "")
	}
	return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, fallback, nil, "")
}

// compile-time guard: ensure we keep net/http imported (used by fasthttp
// status constants indirectly through the file).
var _ = http.StatusOK

func (a *App) gowaGetLoginQRWithSelfHealing(ctx context.Context, instance *models.WhatsAppInstance) (*gowa.LoginResponse, error) {
	deviceID := gowaDeviceID(instance)
	login, err := a.GowaClient.GetLoginQR(ctx, deviceID)
	if err != nil {
		var gowaErr *gowa.Error
		if errors.As(err, &gowaErr) && (gowaErr.StatusCode == 404 || gowaErr.StatusCode == 500) {
			a.Log.Warn("GOWA login QR failed, trying self-healing (re-create device)...", "error", err, "status_code", gowaErr.StatusCode, "device_id", deviceID)
			
			// Best-effort delete device on GOWA
			_ = a.GowaClient.DeleteDevice(ctx, deviceID)
			
			// Create device on GOWA
			if createErr := a.gowaCreateDevice(ctx, instance); createErr == nil {
				// Retry GetLoginQR
				login, err = a.GowaClient.GetLoginQR(ctx, deviceID)
			}
		}
	}
	return login, err
}
