package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// rebaseGowaQRLink fixes GOWA returning a qr_link with an internal host
// (e.g. "http://localhost:3000/statics/...") that whatomate cannot reach.
// It keeps the path+query but swaps the scheme/host onto the instance's
// configured BaseURL, so DownloadMedia hits the server the user actually
// configured. If qrLink is relative or already on the right host, it is
// returned unchanged.
func rebaseGowaQRLink(qrLink, baseURL string) string {
	if qrLink == "" || baseURL == "" {
		return qrLink
	}
	parsed, err := url.Parse(qrLink)
	if err != nil || (parsed.IsAbs() && parsed.Host != "") {
		// If it's already absolute, only rebase when the host differs from the
		// configured base. This preserves the normal same-host case (where the
		// returned link is already correct) and only intervenes on mismatches.
		if err != nil {
			return qrLink
		}
		base, bErr := url.Parse(baseURL)
		if bErr != nil || base.Host == "" || base.Host == parsed.Host {
			return qrLink
		}
		parsed.Scheme = base.Scheme
		parsed.Host = base.Host
		return parsed.String()
	}
	// Relative link: prefix the configured base URL.
	base, err := url.Parse(baseURL)
	if err != nil || base.Host == "" {
		return qrLink
	}
	parsed.Scheme = base.Scheme
	parsed.Host = base.Host
	return parsed.String()
}

// gowaInstanceBundle holds the resolved DB instance and a live gowa.Client built
// from its decrypted credentials.
type gowaInstanceBundle struct {
	instance *models.GowaInstance
	client   *gowa.Client
}

// resolveGowaInstance loads the DB instance by {id} scoped to orgID, decrypts
// its credentials, and builds a gowa.Client. On error it sends the HTTP
// response and returns ok=false — callers should `return nil` immediately.
// Permission checks must already have been done by the caller.
func (a *App) resolveGowaInstance(r *fastglue.Request, orgID uuid.UUID) (gowaInstanceBundle, bool) {
	id, err := parsePathUUID(r, "id", "GOWA instance")
	if err != nil {
		return gowaInstanceBundle{}, false
	}
	instance, err := findByIDAndOrg[models.GowaInstance](a.DB, r, id, orgID, "GOWA instance")
	if err != nil {
		return gowaInstanceBundle{}, false
	}
	instance.DecryptCredentials(a.Config.App.EncryptionKey)
	client := gowa.New(instance.BaseURL, instance.Username, instance.Password)
	return gowaInstanceBundle{instance: instance, client: client}, true
}

// ListGowaInstances returns all DB-managed GOWA instances for the caller's org,
// without credentials.
// GET /api/gowa/servers
func (a *App) ListGowaInstances(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceGowaInstances, models.ActionRead)
	if err != nil {
		return nil
	}

	var instances []models.GowaInstance
	if err := a.DB.Where("organization_id = ?", orgID).Order("created_at DESC").Find(&instances).Error; err != nil {
		a.Log.Error("Failed to list GOWA instances", "error", err, "org", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list GOWA instances", nil, "")
	}

	out := make([]models.GowaInstanceResponse, 0, len(instances))
	for i := range instances {
		out = append(out, instances[i].ToResponse())
	}
	return r.SendEnvelope(map[string]any{"instances": out})
}

// GetGowaInstance returns a single DB-managed GOWA instance without credentials.
// GET /api/gowa/servers/{id}
func (a *App) GetGowaInstance(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceGowaInstances, models.ActionRead)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "GOWA instance")
	if err != nil {
		return nil
	}
	instance, err := findByIDAndOrg[models.GowaInstance](a.DB, r, id, orgID, "GOWA instance")
	if err != nil {
		return nil
	}
	return r.SendEnvelope(map[string]any{"instance": instance.ToResponse()})
}

// gowaInstanceInput is the create/update payload. Username/Password are plain
// at the API boundary and encrypted before persistence.
type gowaInstanceInput struct {
	Name       string `json:"name"`
	BaseURL    string `json:"base_url"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	WebhookURL string `json:"webhook_url"`
	IsActive   *bool  `json:"is_active"`
}

// CreateGowaInstance creates a DB-managed GOWA instance. Before persisting it
// probes the server with ListDevices to validate the URL/credentials.
// POST /api/gowa/servers
func (a *App) CreateGowaInstance(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceGowaInstances, models.ActionWrite)
	if err != nil {
		return nil
	}

	var in gowaInstanceInput
	if err := r.Decode(&in, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	if in.Name == "" || in.BaseURL == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "name and base_url are required", nil, "")
	}

	// Probe the GOWA server before saving — validate URL + auth work.
	probe := gowa.New(in.BaseURL, in.Username, in.Password)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := probe.ListDevices(ctx); err != nil {
		a.Log.Error("GOWA instance probe failed", "error", err, "base_url", in.BaseURL)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Could not reach GOWA server with these credentials", nil, "")
	}

	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	inst := &models.GowaInstance{
		OrganizationID: orgID,
		Name:           in.Name,
		BaseURL:        in.BaseURL,
		Username:       in.Username,
		Password:       in.Password,
		WebhookURL:     in.WebhookURL,
		IsActive:       active,
	}
	if err := inst.EncryptCredentials(a.Config.App.EncryptionKey); err != nil {
		a.Log.Error("Failed to encrypt GOWA instance credentials", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to secure credentials", nil, "")
	}
	if err := a.DB.Create(inst).Error; err != nil {
		a.Log.Error("Failed to create GOWA instance", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create GOWA instance", nil, "")
	}

	a.logAudit(orgID, userID, "gowa_instances", inst.ID, models.AuditActionCreated, nil, map[string]any{
		"name": inst.Name, "base_url": inst.BaseURL,
	})
	return r.SendEnvelope(map[string]any{"instance": inst.ToResponse()})
}

// UpdateGowaInstance updates a DB-managed GOWA instance. Empty username/password
// in the payload means "keep existing".
// PUT /api/gowa/servers/{id}
func (a *App) UpdateGowaInstance(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceGowaInstances, models.ActionWrite)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "GOWA instance")
	if err != nil {
		return nil
	}
	instance, err := findByIDAndOrg[models.GowaInstance](a.DB, r, id, orgID, "GOWA instance")
	if err != nil {
		return nil
	}

	var in gowaInstanceInput
	if err := r.Decode(&in, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	old := *instance
	instance.DecryptCredentials(a.Config.App.EncryptionKey)

	if in.Name != "" {
		instance.Name = in.Name
	}
	if in.BaseURL != "" {
		instance.BaseURL = in.BaseURL
	}
	if in.Username != "" {
		instance.Username = in.Username
	}
	if in.Password != "" {
		instance.Password = in.Password
	}
	instance.WebhookURL = in.WebhookURL
	if in.IsActive != nil {
		instance.IsActive = *in.IsActive
	}

	if err := instance.EncryptCredentials(a.Config.App.EncryptionKey); err != nil {
		a.Log.Error("Failed to encrypt GOWA instance credentials", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to secure credentials", nil, "")
	}
	if err := a.DB.Save(instance).Error; err != nil {
		a.Log.Error("Failed to update GOWA instance", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update GOWA instance", nil, "")
	}

	a.logAudit(orgID, userID, "gowa_instances", instance.ID, models.AuditActionUpdated,
		map[string]any{"name": old.Name, "base_url": old.BaseURL, "is_active": old.IsActive},
		map[string]any{"name": instance.Name, "base_url": instance.BaseURL, "is_active": instance.IsActive})
	return r.SendEnvelope(map[string]any{"instance": instance.ToResponse()})
}

// DeleteGowaInstance soft-deletes a DB-managed GOWA instance (does not touch
// devices on the remote GOWA server — that's an explicit per-device action).
// DELETE /api/gowa/servers/{id}
func (a *App) DeleteGowaInstance(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceGowaInstances, models.ActionDelete)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "GOWA instance")
	if err != nil {
		return nil
	}
	instance, err := findByIDAndOrg[models.GowaInstance](a.DB, r, id, orgID, "GOWA instance")
	if err != nil {
		return nil
	}
	if err := a.DB.Delete(instance).Error; err != nil {
		a.Log.Error("Failed to delete GOWA instance", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete GOWA instance", nil, "")
	}

	a.logAudit(orgID, userID, "gowa_instances", instance.ID, models.AuditActionDeleted, map[string]any{
		"name": instance.Name, "base_url": instance.BaseURL,
	}, nil)
	return r.SendEnvelope(map[string]any{"deleted": true})
}

// parseDeviceID extracts the {deviceId} path param (a GOWA device string id).
func parseDeviceID(r *fastglue.Request) string {
	v, _ := r.RequestCtx.UserValue("deviceId").(string)
	return v
}

// ListGowaInstanceDevices lists all devices on a GOWA instance, enriching each
// with its live connection status.
// GET /api/gowa/servers/{id}/devices
func (a *App) ListGowaInstanceDevices(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceDevices, models.ActionRead)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}

	ctx := context.Background()
	devices, err := bundle.client.ListDevices(ctx)
	if err != nil {
		a.Log.Error("Failed to list GOWA devices", "error", err, "instance", bundle.instance.Name)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to list devices from GOWA", nil, "")
	}

	type deviceWithStatus struct {
		gowa.DeviceInfo
		IsConnected bool   `json:"is_connected"`
		IsLoggedIn  bool   `json:"is_logged_in"`
		JID         string `json:"jid"`
	}
	out := make([]deviceWithStatus, 0, len(devices))
	for _, d := range devices {
		entry := deviceWithStatus{DeviceInfo: d}
		if st, err := bundle.client.GetDeviceStatus(ctx, d.ID); err == nil {
			entry.IsConnected = st.IsConnected
			entry.IsLoggedIn = st.IsLoggedIn
		}
		out = append(out, entry)
	}
	return r.SendEnvelope(map[string]any{"devices": out})
}

// ensureGowaAccountOpts parameterizes ensureGowaAccountForDevice.
type ensureGowaAccountOpts struct {
	InstanceName  string // GowaInstance.Name, used to build a unique account name
	BaseURL       string // GowaInstance.BaseURL
	DeviceID      string // GOWA device id (becomes WhatsAppAccount.GowaDeviceID)
	WebhookSecret string // HMAC secret (encrypted at rest)
	PreferredName string // optional human label; defaults to InstanceName
}

// ensureGowaAccountForDevice creates the WhatsAppAccount row that the GOWA
// webhook receiver needs to resolve a device (getGowaAccountByDeviceID). It is
// idempotent on GowaDeviceID: if an account already exists for this device it
// is returned unchanged. The per-org unique Name constraint (idx_wa_org_name)
// is satisfied by suffixing; the device id is the real identity key.
//
// This mirrors App.CreateAccount's GOWA branch (accounts.go:159-181) but is
// scoped to the GOWA-Servers provisioning path, which previously created the
// device on the remote GOWA server without ever writing the account row.
func (a *App) ensureGowaAccountForDevice(orgID, userID uuid.UUID, opts ensureGowaAccountOpts) (*models.WhatsAppAccount, error) {
	// Idempotent: a device may already have an account (retry, re-provision).
	var existing models.WhatsAppAccount
	if err := a.DB.Where("organization_id = ? AND gowa_device_id = ?", orgID, opts.DeviceID).First(&existing).Error; err == nil {
		a.decryptAccountSecrets(&existing)
		return &existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("lookup existing gowa account: %w", err)
	}

	// Build a per-org unique Name. idx_wa_org_name requires uniqueness within
	// an org, so suffix with the device id's short tail until free.
	label := opts.PreferredName
	if label == "" {
		label = opts.InstanceName
	}
	if label == "" {
		label = "gowa"
	}
	name := label
	for i := 0; i < 8; i++ {
		var count int64
		if err := a.DB.Model(&models.WhatsAppAccount{}).
			Where("organization_id = ? AND name = ?", orgID, name).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("check gowa account name uniqueness: %w", err)
		}
		if count == 0 {
			break
		}
		// e.g. "my-server-a3f9" — short suffix keeps it readable.
		tail := opts.DeviceID
		if len(tail) > 8 {
			tail = tail[len(tail)-8:]
		}
		name = fmt.Sprintf("%s-%s-%d", label, tail, i+1)
	}

	account := models.WhatsAppAccount{
		BaseModel:         models.BaseModel{},
		OrganizationID:    orgID,
		Name:              name,
		ProviderType:      "gowa",
		GowaBaseURL:       opts.BaseURL,
		GowaDeviceID:      opts.DeviceID,
		GowaWebhookSecret: opts.WebhookSecret,
		Status:            "active",
		CreatedByID:       &userID,
		UpdatedByID:       &userID,
	}
	if err := a.encryptAccountSecrets(&account); err != nil {
		return nil, fmt.Errorf("encrypt gowa webhook secret: %w", err)
	}
	if err := a.DB.Create(&account).Error; err != nil {
		return nil, fmt.Errorf("create gowa whatsapp account: %w", err)
	}
	a.decryptAccountSecrets(&account)
	return &account, nil
}

// SyncGowaInstanceDevice backfills the WhatsAppAccount row for a device that
// already exists on the GOWA server but has no account row in whatomate
// (e.g. devices created via the GOWA Servers UI before this fix). It reads the
// device's current webhook config from GOWA so the stored secret matches what
// GOWA is actually signing webhooks with.
//
// POST /api/gowa/servers/{id}/devices/{deviceId}/sync
func (a *App) SyncGowaInstanceDevice(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}

	// Pull the live webhook config so the stored secret is authoritative. If
	// GOWA has no secret configured, generate one and push it so the account
	// row and GOWA agree before any webhook arrives.
	cfg, err := bundle.client.GetDeviceWebhook(context.Background(), deviceID)
	if err != nil {
		a.Log.Error("Failed to read GOWA device webhook during sync", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to read device webhook from GOWA", nil, "")
	}
	secret := cfg.WebhookSecret
	if secret == "" {
		secret = gowa.GenerateWebhookSecret()
		webhookURL := bundle.instance.WebhookURL
		if webhookURL == "" {
			webhookURL = fmt.Sprintf("%s://%s%s", "http", r.RequestCtx.Host(), a.Config.GOWA.WebhookPath)
		}
		if _, err := bundle.client.SetDeviceWebhook(context.Background(), deviceID, gowa.WebhookConfig{
			WebhookURL:    webhookURL,
			WebhookSecret: secret,
			WebhookEvents: "message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited",
		}); err != nil {
			a.Log.Error("Failed to set GOWA device webhook during sync", "error", err, "device", deviceID)
			return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to set webhook on GOWA", nil, "")
		}
	}

	account, err := a.ensureGowaAccountForDevice(orgID, userID, ensureGowaAccountOpts{
		InstanceName:  bundle.instance.Name,
		BaseURL:       bundle.instance.BaseURL,
		DeviceID:      deviceID,
		WebhookSecret: secret,
	})
	if err != nil {
		a.Log.Error("Failed to backfill GOWA WhatsAppAccount", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to sync device", nil, "")
	}

	a.logAudit(orgID, userID, "devices", account.ID, models.AuditActionUpdated, nil, map[string]any{
		"device_id": deviceID, "synced_account": true,
	})
	return r.SendEnvelope(map[string]any{
		"device_id":    deviceID,
		"account_id":   account.ID,
		"account_name": account.Name,
	})
}

// CreateGowaInstanceDevice provisions a new device on a GOWA instance (mirrors
// the legacy GowaCreateDevice webhook/device-id generation).
// POST /api/gowa/servers/{id}/devices  body: {"device_name": "..."}
func (a *App) CreateGowaInstanceDevice(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}

	var req struct {
		DeviceName string `json:"device_name"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	if req.DeviceName == "" {
		req.DeviceName = "whatomate-device"
	}

	ctx := context.Background()
	webhookURL := bundle.instance.WebhookURL
	if webhookURL == "" {
		webhookURL = fmt.Sprintf("%s://%s%s", "http", r.RequestCtx.Host(), a.Config.GOWA.WebhookPath)
	}
	webhookSecret := gowa.GenerateWebhookSecret()
	deviceID := gowa.GenerateDeviceID(req.DeviceName)

	device, err := bundle.client.CreateDevice(ctx, deviceID, gowa.WebhookConfig{
		WebhookURL:    webhookURL,
		WebhookSecret: webhookSecret,
		WebhookEvents: "message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited",
	})
	if err != nil {
		a.Log.Error("Failed to create GOWA device", "error", err, "instance", bundle.instance.Name)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to create device on GOWA", nil, "")
	}

	// Provision the matching WhatsAppAccount row so incoming webhooks for this
	// device can resolve an account (getGowaAccountByDeviceID). Without it the
	// webhook receiver returns 403 and no chat/message is ever stored, so the
	// device appears connected but its chats never reach the inbox.
	account, accountErr := a.ensureGowaAccountForDevice(orgID, userID, ensureGowaAccountOpts{
		InstanceName:  bundle.instance.Name,
		BaseURL:       bundle.instance.BaseURL,
		DeviceID:      device.ID,
		WebhookSecret: webhookSecret,
		PreferredName: req.DeviceName,
	})

	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionCreated, nil, map[string]any{
		"device_id": device.ID, "instance": bundle.instance.Name, "base_url": bundle.instance.BaseURL,
	})
	resp := map[string]any{
		"device_id":      device.ID,
		"webhook_secret": webhookSecret,
	}
	if accountErr != nil {
		// The device was created on GOWA but the account row was not. Surface
		// the error so the operator can retry via the sync endpoint, but still
		// return the device_id so QR pairing can proceed.
		a.Log.Error("GOWA device provisioned but WhatsAppAccount row creation failed", "error", accountErr, "device_id", device.ID)
		resp["account_error"] = accountErr.Error()
	} else {
		resp["account_id"] = account.ID
	}
	return r.SendEnvelope(resp)
}

// DeleteGowaInstanceDevice removes a device from a GOWA instance.
// DELETE /api/gowa/servers/{id}/devices/{deviceId}
func (a *App) DeleteGowaInstanceDevice(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.ActionDelete)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}

	if err := bundle.client.DeleteDevice(context.Background(), deviceID); err != nil {
		a.Log.Error("Failed to delete GOWA device", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to delete device on GOWA", nil, "")
	}
	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionDeleted, nil, map[string]any{
		"device_id": deviceID, "instance": bundle.instance.Name,
	})
	return r.SendEnvelope(map[string]any{"deleted": true})
}

// GowaInstanceDeviceQR fetches a login QR (as a base64 data URI) for a device.
// GET /api/gowa/servers/{id}/devices/{deviceId}/qr
func (a *App) GowaInstanceDeviceQR(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}

	ctx := context.Background()
	// If already connected, short-circuit like the account-bound handler does.
	if st, err := bundle.client.GetDeviceStatus(ctx, deviceID); err == nil && st.IsConnected {
		return r.SendEnvelope(map[string]any{"already_connected": true})
	}
	qr, err := bundle.client.GetLoginQR(ctx, deviceID)
	if err != nil {
		a.Log.Error("Failed to get GOWA login QR", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to get QR code from GOWA", nil, "")
	}
	// GOWA returns a qr_link with its own internal host (often localhost:3000);
	// rebase it onto the instance's configured BaseURL so whatomate can reach it.
	qrLink := rebaseGowaQRLink(qr.QRLink, bundle.instance.BaseURL)
	qrData, err := bundle.client.DownloadMedia(ctx, qrLink, "")
	if err != nil {
		a.Log.Error("Failed to download GOWA QR image", "error", err, "device", deviceID, "qr_link", qrLink, "original", qr.QRLink, "base_url", bundle.instance.BaseURL)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to download QR image", nil, "")
	}
	dataURI := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(qrData))
	return r.SendEnvelope(map[string]any{"qr_link": dataURI, "qr_duration": qr.QRDuration})
}

// GowaInstanceDevicePairCode requests a phone pair code.
// POST /api/gowa/servers/{id}/devices/{deviceId}/pair-code  body: {"phone":"..."}
func (a *App) GowaInstanceDevicePairCode(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}

	var req struct {
		Phone string `json:"phone"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	if req.Phone == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Phone number is required", nil, "")
	}
	result, err := bundle.client.LoginWithCode(context.Background(), deviceID, req.Phone)
	if err != nil {
		a.Log.Error("Failed to get GOWA pair code", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to get pair code from GOWA", nil, "")
	}
	return r.SendEnvelope(map[string]any{"pair_code": result.PairCode})
}

// GowaInstanceDeviceLogout logs out a device (keeps the slot).
// POST /api/gowa/servers/{id}/devices/{deviceId}/logout
func (a *App) GowaInstanceDeviceLogout(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}
	if err := bundle.client.LogoutDevice(context.Background(), deviceID); err != nil {
		a.Log.Error("Failed to logout GOWA device", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to logout device on GOWA", nil, "")
	}
	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionUpdated, nil, map[string]any{
		"device_id": deviceID, "action": "logout",
	})
	return r.SendEnvelope(map[string]any{"ok": true})
}

// GowaInstanceDeviceReconnect triggers a reconnect for a device.
// POST /api/gowa/servers/{id}/devices/{deviceId}/reconnect
func (a *App) GowaInstanceDeviceReconnect(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}
	if err := bundle.client.ReconnectDevice(context.Background(), deviceID); err != nil {
		a.Log.Error("Failed to reconnect GOWA device", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to reconnect device on GOWA", nil, "")
	}
	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionUpdated, nil, map[string]any{
		"device_id": deviceID, "action": "reconnect",
	})
	return r.SendEnvelope(map[string]any{"ok": true})
}

// GetGowaInstanceDeviceWebhook returns the webhook config for a device.
// GET /api/gowa/servers/{id}/devices/{deviceId}/webhook
func (a *App) GetGowaInstanceDeviceWebhook(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceDevices, models.ActionRead)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}
	cfg, err := bundle.client.GetDeviceWebhook(context.Background(), deviceID)
	if err != nil {
		a.Log.Error("Failed to get GOWA device webhook", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to get webhook from GOWA", nil, "")
	}
	return r.SendEnvelope(map[string]any{"webhook": cfg})
}

// SetGowaInstanceDeviceWebhook updates the webhook config for a device.
// PUT /api/gowa/servers/{id}/devices/{deviceId}/webhook
func (a *App) SetGowaInstanceDeviceWebhook(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}
	bundle, ok := a.resolveGowaInstance(r, orgID)
	if !ok {
		return nil
	}
	deviceID := parseDeviceID(r)
	if deviceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "deviceId is required", nil, "")
	}

	var cfg gowa.WebhookConfig
	if err := r.Decode(&cfg, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	updated, err := bundle.client.SetDeviceWebhook(context.Background(), deviceID, cfg)
	if err != nil {
		a.Log.Error("Failed to set GOWA device webhook", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to set webhook on GOWA", nil, "")
	}
	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionUpdated, nil, map[string]any{
		"device_id": deviceID, "webhook_url": updated.WebhookURL,
	})
	return r.SendEnvelope(map[string]any{"webhook": updated})
}
