package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
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

// isGowaDevicePaired reports whether the GOWA device slot for the given
// instance is already paired with WhatsApp (i.e. the user has scanned the QR
// and whatsmeow has authenticated). This is the critical pre-check the QR and
// connect handlers must perform before asking GOWA for a login QR: calling
// GET /devices/:id/login on an already-logged-in device returns 500 ("device
// login not found"), which the legacy self-heal path misinterpreted as
// "device slot corrupt" — it then deleted the live paired device and
// re-provisioned an empty slot, destroying the user's session.
//
// IMPORTANT: GOWA's Device.State string is misleading here — "connected" only
// means the websocket is up, NOT that the device is paired. A brand-new empty
// slot shows state="connected" immediately after creation, before any QR scan.
// The reliable signal is the explicit is_logged_in boolean from
// GET /devices/:id/status, which is what the poller's
// syncInstanceStatusFromGowa already uses. We mirror that contract here.
//
// Returns the live Device (for state + JID context in logs) and a boolean
// indicating actual pairing. Errors are returned only on transport/lookup
// failure.
func (a *App) isGowaDevicePaired(ctx context.Context, deviceID string) (*gowa.Device, bool, error) {
	if deviceID == "" || a.GowaClient == nil {
		return nil, false, nil
	}
	// Use /status for the authoritative is_logged_in signal. Fall back to
	// /devices/:id only for the JID/state metadata (not for the pairing
	// decision).
	status, err := a.GowaClient.GetStatus(ctx, deviceID)
	if err != nil {
		return nil, false, err
	}
	device, _ := a.GowaClient.GetDevice(ctx, deviceID) // best-effort, for JID/state
	paired := status != nil && status.IsLoggedIn
	return device, paired, nil
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
	deviceID := gowaDeviceID(instance)
	status, err := a.GowaClient.GetStatus(ctx, deviceID)
	if err != nil {
		// GetStatus uses a path parameter which may fail for non-ASCII device IDs
		// (e.g. Arabic-Indic numerals). Fall back to scanning ListDevices.
		var ge *gowa.Error
		if errors.As(err, &ge) && ge.StatusCode >= 500 {
			if devices, listErr := a.GowaClient.ListDevices(ctx); listErr == nil {
				for _, dev := range devices {
					if dev.ID == deviceID {
						switch dev.State {
						case "logged_in":
							return models.InstanceStatusConnected, nil
						case "connecting":
							return models.InstanceStatusConnecting, nil
						default:
							return models.InstanceStatusDisconnected, nil
						}
					}
				}
			}
		}
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

	// Pre-check: never ask for a login QR when the device is already paired.
	// GOWA returns 500 ("device login not found") for a logged-in device, which
	// the legacy self-heal path misread as a corrupt slot and "fixed" by
	// deleting the live session. If we are already paired, surface that to the
	// caller instead of generating a QR.
	if device, paired, statusErr := a.isGowaDevicePaired(ctx, gowaDeviceID(instance)); statusErr == nil && paired {
		a.markGowaInstanceConnected(ctx, instance, device)
		return r.SendEnvelope(map[string]any{
			"instance_id":      instance.ID.String(),
			"available":        false,
			"already_connected": true,
			"state":            device.State,
			"jid":              device.JID,
		})
	}

	login, err := a.gowaGetLoginQRWithSelfHealing(ctx, instance)
	if err != nil {
		a.Log.Error("GOWA login QR failed", "error", err, "instance_id", instance.ID)
		return gowaSendError(r, err, "Failed to fetch QR code from GOWA")
	}

	qrCodeVal := login.QRLink
	if login.QRLink != "" {
		if bytes, fetchErr := a.GowaClient.FetchBytes(ctx, login.QRLink, gowaDeviceID(instance)); fetchErr == nil {
			base64Data := base64.StdEncoding.EncodeToString(bytes)
			qrCodeVal = fmt.Sprintf("data:image/png;base64,%s", base64Data)
		} else {
			a.Log.Warn("GOWA login QR download failed; passing raw link", "error", fetchErr, "url", login.QRLink)
		}
	}

	return r.SendEnvelope(map[string]any{
		"instance_id":     instance.ID.String(),
		"available":       true,
		"qr_code":         qrCodeVal,
		"timeout_seconds": login.QRDuration,
		"device_id":       login.DeviceID,
	})
}

// markGowaInstanceConnected syncs the local instance row to "connected" using
// the live GOWA device state, mirroring what syncInstanceStatusFromGowa does
// in the poller. This is invoked on the QR/connect short-circuit path so the
// DB catches up to reality without waiting for the next poll sweep.
func (a *App) markGowaInstanceConnected(ctx context.Context, instance *models.WhatsAppInstance, device *gowa.Device) {
	if instance == nil || device == nil {
		return
	}
	updates := map[string]any{"status": models.InstanceStatusConnected}
	if device.JID != "" && instance.JID == "" {
		updates["jid"] = device.JID
		if phone := gowaJIDToPhone(device.JID); phone != "" {
			updates["phone_number"] = phone
		}
	}
	if instance.LastConnectedAt == nil {
		now := time.Now()
		updates["last_connected_at"] = &now
	}
	if err := a.DB.WithContext(ctx).Model(&models.WhatsAppInstance{}).
		Where("id = ?", instance.ID).Updates(updates).Error; err != nil {
		a.Log.Warn("GOWA: failed to mark instance connected on QR short-circuit",
			"error", err, "instance_id", instance.ID)
	}
}

// gowaJIDToPhone extracts the phone digits from a JID like
// "12025550100@s.whatsapp.net". For group/newsletter JIDs it returns "".
func gowaJIDToPhone(jid string) string {
	if at := strings.IndexByte(jid, '@'); at > 0 {
		local := jid[:at]
		if strings.HasSuffix(jid, "@s.whatsapp.net") && local != "" {
			return local
		}
	}
	return ""
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

	// Pre-check: if the device is already paired, do NOT request a new login
	// QR — GOWA returns 500 for a logged-in device and the self-heal path
	// would destroy the live session. Mirror the QR handler's short-circuit.
	if device, paired, statusErr := a.isGowaDevicePaired(ctx, gowaDeviceID(instance)); statusErr == nil && paired {
		a.markGowaInstanceConnected(ctx, instance, device)
		return r.SendEnvelope(map[string]any{
			"status":           "already_connected",
			"already_connected": true,
			"state":            device.State,
			"jid":              device.JID,
		})
	}

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

	qrCodeVal := login.QRLink
	if login.QRLink != "" {
		if bytes, fetchErr := a.GowaClient.FetchBytes(ctx, login.QRLink, gowaDeviceID(instance)); fetchErr == nil {
			base64Data := base64.StdEncoding.EncodeToString(bytes)
			qrCodeVal = fmt.Sprintf("data:image/png;base64,%s", base64Data)
		} else {
			a.Log.Warn("GOWA connect QR download failed; passing raw link", "error", fetchErr, "url", login.QRLink)
		}
	}

	return r.SendEnvelope(map[string]any{
		"status":      "connection_initiated",
		"qr_link":     qrCodeVal,
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
			"status":       models.InstanceStatusDisconnected,
			"jid":          "",
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

	deviceID := gowaDeviceID(instance)
	err = a.GowaClient.Reconnect(ctx, deviceID)
	if err != nil {
		var gowaErr *gowa.Error
		if errors.As(err, &gowaErr) {
			// If device is not logged in or session is deleted on GOWA, we self-heal
			// by purging and re-creating the device slot, then triggering the login flow.
			isNotLoggedIn := gowaErr.StatusCode == 404 ||
				gowaErr.StatusCode == 500 ||
				strings.Contains(strings.ToLower(gowaErr.Message), "not logged in") ||
				strings.Contains(strings.ToLower(gowaErr.Message), "session deleted")

			if isNotLoggedIn {
				a.Log.Warn("GOWA reconnect failed due to missing or deleted session; self-healing device slot",
					"error", err, "device_id", deviceID)

				_ = a.GowaClient.DeleteDevice(ctx, deviceID)
				if createErr := a.gowaCreateDevice(ctx, instance); createErr != nil {
					a.Log.Error("GOWA reconnect self-heal: re-create device failed", "error", createErr, "device_id", deviceID)
					return gowaSendError(r, createErr, "Failed to re-provision GOWA device")
				}

				// Give GOWA a moment to initialise the new slot
				time.Sleep(500 * time.Millisecond)

				// Call app/login to initialize the login session (so that the QR code is generated)
				if _, loginErr := a.GowaClient.GetLoginQR(ctx, deviceID); loginErr != nil {
					a.Log.Error("GOWA reconnect self-heal: start login flow failed", "error", loginErr, "device_id", deviceID)
					return gowaSendError(r, loginErr, "Failed to initialize login QR code")
				}

				return r.SendEnvelope(map[string]string{"status": "reconnection_initiated"})
			}
		}

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
	gowaStatus, err := a.gowaFetchStatus(ctx, instance)
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
	if gowaStatus == models.InstanceStatusConnected {
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
	if err == nil {
		return login, nil
	}

	var gowaErr *gowa.Error
	if !errors.As(err, &gowaErr) {
		return nil, err
	}

	a.Log.Warn("GOWA login QR failed, attempting self-healing",
		"error", err, "status_code", gowaErr.StatusCode, "device_id", deviceID)

	// Defense-in-depth: never enter the destructive self-heal path while the
	// device is currently paired/logged in. GOWA returns 500 ("device login
	// not found") for a logged-in device because there is no active login
	// flow — that is not a corrupt slot, and deleting the device would
	// destroy the user's live WhatsApp session. Bail out with the original
	// error so the caller surfaces a clear "already connected" failure
	// instead of nuking the pairing.
	if device, paired, _ := a.isGowaDevicePaired(ctx, deviceID); paired {
		a.Log.Warn("GOWA self-heal aborted: device is already paired, refusing to delete live session",
			"device_id", deviceID, "state", device.State, "jid", device.JID)
		return nil, fmt.Errorf("gowa: device %s is already %s — refusing to request a new login QR (would destroy the live session); use /disconnect first if you really want to re-pair",
			deviceID, device.State)
	}

	// Step 1: try reconnect first (device exists but websocket dropped)
	if reconnErr := a.GowaClient.Reconnect(ctx, deviceID); reconnErr == nil {
		// Brief pause for GOWA to re-establish the WS session
		time.Sleep(1 * time.Second)
		login, err = a.GowaClient.GetLoginQR(ctx, deviceID)
		if err == nil {
			a.Log.Info("GOWA self-heal via reconnect succeeded", "device_id", deviceID)
			return login, nil
		}
		a.Log.Warn("GOWA reconnect succeeded but QR still failed; escalating to re-provision", "error", err, "device_id", deviceID)
	}

	// Step 2: device slot gone or corrupt — delete and re-create
	if gowaErr.StatusCode == 404 || gowaErr.StatusCode == 500 {
		_ = a.GowaClient.DeleteDevice(ctx, deviceID)
		if createErr := a.gowaCreateDevice(ctx, instance); createErr != nil {
			return nil, fmt.Errorf("GOWA self-heal: re-create device failed: %w", createErr)
		}
		// Give GOWA a moment to initialise the new slot
		time.Sleep(500 * time.Millisecond)
		login, err = a.GowaClient.GetLoginQR(ctx, deviceID)
		if err == nil {
			a.Log.Info("GOWA self-heal via re-provision succeeded", "device_id", deviceID)
		}
	}

	return login, err
}
