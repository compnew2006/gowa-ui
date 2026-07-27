package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/contactutil"
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

	// Drop any cached GOWA client for this base URL so the messaging registry
	// picks up the new credentials on the next send.
	if a.WARegistry != nil {
		a.WARegistry.InvalidateGowa(inst.BaseURL)
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

	// Drop cached GOWA clients for both the old and new base URLs so the
	// messaging registry re-resolves credentials on the next send. The base
	// URL may have changed, and credentials were just (re)encrypted.
	if a.WARegistry != nil {
		a.WARegistry.InvalidateGowa(old.BaseURL)
		a.WARegistry.InvalidateGowa(instance.BaseURL)
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

	// Drop the cached GOWA client so future sends no longer resolve to a
	// provider whose credentials were just removed.
	if a.WARegistry != nil {
		a.WARegistry.InvalidateGowa(instance.BaseURL)
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

// lookupGowaDeviceJID returns the connected WhatsApp JID for a GOWA device
// (e.g. "966561853319@s.whatsapp.net") by listing devices on the GOWA server
// and matching by ID. Returns "" when the device is not found or has not yet
// paired (no JID assigned).
//
// This is required because GOWA webhooks send the connected JID as the
// top-level device_id, while whatomate stores the device's custom id as
// GowaDeviceID. The JID must be persisted as WhatsAppAccount.GowaJID for the
// webhook resolver (getGowaAccountByDeviceID) to match incoming messages.
func lookupGowaDeviceJID(ctx context.Context, client *gowa.Client, deviceID string) string {
	if client == nil || deviceID == "" {
		return ""
	}
	devices, err := client.ListDevices(ctx)
	if err != nil {
		return ""
	}
	for _, d := range devices {
		if d.ID == deviceID {
			return strings.TrimSpace(d.JID)
		}
	}
	return ""
}

// ensureGowaAccountOpts parameterizes ensureGowaAccountForDevice.
type ensureGowaAccountOpts struct {
	InstanceName  string // GowaInstance.Name, used to build a unique account name
	BaseURL       string // GowaInstance.BaseURL
	DeviceID      string // GOWA device id (becomes WhatsAppAccount.GowaDeviceID)
	WebhookSecret string // HMAC secret (encrypted at rest)
	PreferredName string // optional human label; defaults to InstanceName
	// DeviceJID is the connected WhatsApp JID for this device (e.g.
	// "966561853319@s.whatsapp.net"). GOWA webhooks send this JID as the
	// top-level device_id, so it must be stored as WhatsAppAccount.GowaJID for
	// getGowaAccountByDeviceID to resolve the account. May be empty for a
	// freshly created device that has not yet paired; it is backfilled on the
	// first connection webhook and on subsequent syncs.
	DeviceJID string
}

// ensureGowaAccountForDevice creates the WhatsAppAccount row that the GOWA
// webhook receiver needs to resolve a device (getGowaAccountByDeviceID). It is
// idempotent on GowaDeviceID: if an account already exists for this device it
// is returned unchanged (and its PhoneID is backfilled if a JID is now known).
// The per-org unique Name constraint (idx_wa_org_name) is satisfied by
// suffixing; the device id is the real identity key.
//
// This mirrors App.CreateAccount's GOWA branch (accounts.go:159-181) but is
// scoped to the GOWA-Servers provisioning path, which previously created the
// device on the remote GOWA server without ever writing the account row.
func (a *App) ensureGowaAccountForDevice(orgID, userID uuid.UUID, opts ensureGowaAccountOpts) (*models.WhatsAppAccount, error) {
	// Idempotent: a device may already have an account (retry, re-provision).
	var existing models.WhatsAppAccount
	if err := a.DB.Where("organization_id = ? AND gowa_device_id = ?", orgID, opts.DeviceID).First(&existing).Error; err == nil {
		// Backfill the JID if we now know it but the row predates pairing.
		// This is the key fix for "device connected but no chats": GOWA sends
		// the JID as device_id, so GowaJID must be set or the webhook lookup
		// fails with "Unknown GOWA device" on every real message.
		if opts.DeviceJID != "" && existing.GowaJID != opts.DeviceJID {
			if err := a.DB.Model(&existing).Update("gowa_jid", opts.DeviceJID).Error; err != nil {
				return nil, fmt.Errorf("update gowa account gowa_jid: %w", err)
			}
			existing.GowaJID = opts.DeviceJID
		}
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
		BaseModel:      models.BaseModel{},
		OrganizationID: orgID,
		Name:           name,
		// GOWA webhooks key device_id by the connected JID, not the custom
		// device id. Storing the JID as GowaJID here lets the webhook lookup
		// match on the first message (see getGowaAccountByDeviceID).
		GowaJID:           opts.DeviceJID,
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
			WebhookEvents: "message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited,call.offer",
		}); err != nil {
			a.Log.Error("Failed to set GOWA device webhook during sync", "error", err, "device", deviceID)
			return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to set webhook on GOWA", nil, "")
		}
	}

	// Fetch the connected JID from GOWA so we can store it as PhoneID. Without
	// this, GOWA webhooks (keyed by JID) can't resolve the account and every
	// real message is rejected with "Unknown GOWA device".
	jid := lookupGowaDeviceJID(context.Background(), bundle.client, deviceID)
	if jid == "" {
		a.Log.Warn("GOWA device has no connected JID; PhoneID will be backfilled on connect", "device", deviceID)
	}

	account, err := a.ensureGowaAccountForDevice(orgID, userID, ensureGowaAccountOpts{
		InstanceName:  bundle.instance.Name,
		BaseURL:       bundle.instance.BaseURL,
		DeviceID:      deviceID,
		WebhookSecret: secret,
		DeviceJID:     jid,
	})
	if err != nil {
		a.Log.Error("Failed to backfill GOWA WhatsAppAccount", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to sync device", nil, "")
	}

	a.logAudit(orgID, userID, "devices", account.ID, models.AuditActionUpdated, nil, map[string]any{
		"device_id": deviceID, "synced_account": true, "jid": jid,
	})
	return r.SendEnvelope(map[string]any{
		"device_id":    deviceID,
		"account_id":   account.ID,
		"account_name": account.Name,
		"jid":          jid,
	})
}

// SyncGowaInstanceDeviceContacts imports a device's chat list from GOWA and
// upserts the corresponding contact rows, so the Contacts page populates
// immediately for a connected device instead of waiting for the next inbound
// webhook. It is read-only with respect to GOWA (GET /chats) and idempotent
// with respect to whatomate (reuses contactutil.GetOrCreateContact).
//
// POST /api/gowa/servers/{id}/devices/{deviceId}/sync-contacts
func (a *App) SyncGowaInstanceDeviceContacts(r *fastglue.Request) error {
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

	// Resolve the WhatsAppAccount for this device. It must already exist (the
	// /sync endpoint provisions it); importing contacts never mutates the
	// webhook config, so we surface a clear 409 if the account is missing
	// rather than silently side-effecting GOWA.
	account, err := a.getGowaAccountByDeviceID(deviceID)
	if err != nil {
		a.Log.Warn("GOWA contact sync requested for device with no account row", "device_id", deviceID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusConflict,
			"Device account is not provisioned yet. Click the Sync button first to set up the device account.", nil, "")
	}
	// Guard against cross-org device-id reuse: getGowaAccountByDeviceID is not
	// org-scoped (it predates multi-org), so re-check ownership here.
	if account.OrganizationID != orgID {
		a.Log.Warn("GOWA contact sync device/account org mismatch", "device_id", deviceID,
			"account_org", account.OrganizationID, "request_org", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "GOWA instance not found", nil, "")
	}

	ctx := context.Background()
	chats, total, err := bundle.client.ListChats(ctx, deviceID, gowa.ListChatsOptions{})
	if err != nil {
		a.Log.Error("Failed to list GOWA chats for contact sync", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to fetch chats from GOWA", nil, "")
	}

	created, touched := 0, 0
	for _, ch := range chats {
		jid := strings.TrimSpace(ch.JID)
		if jid == "" {
			continue
		}
		name := strings.TrimSpace(ch.Name)

		isGroup := strings.HasSuffix(jid, "@g.us")
		isNewsletter := strings.HasSuffix(jid, "@newsletter")
		// Match the webhook convention: group/newsletter chats are keyed by their
		// full JID (the @g.us/@newsletter suffix is part of the identity), while
		// 1:1 chats use the bare phone digits.
		identity := jid
		if !isGroup && !isNewsletter {
			identity = gowa.PhoneFromJID(jid)
			if identity == "" {
				continue
			}
		}

		contact, isNew, err := contactutil.GetOrCreateContact(a.DB, orgID, identity, name)
		if err != nil {
			a.Log.Error("Failed to upsert contact during GOWA sync", "error", err, "jid", jid)
			continue
		}
		touched++

		// Set group/newsletter metadata to match how the webhook path marks
		// these chats (chatbot_processor.go), so the contact list renders the
		// correct badge consistently. Groups and newsletters are distinct
		// categories — a @newsletter JID is NOT a group. We only write when the
		// flag isn't already set.
		metaKey := ""
		if isGroup {
			metaKey = "is_group_chat"
		} else if isNewsletter {
			metaKey = "is_newsletter"
		}
		if metaKey != "" {
			needsMetaUpdate := false
			if contact.Metadata == nil {
				contact.Metadata = models.JSONB{}
				needsMetaUpdate = true
			}
			// Groups and newsletters are mutually exclusive. Setting one clears
			// the other so legacy contacts that carry both flags self-heal.
			otherKey := ""
			if metaKey == "is_group_chat" {
				otherKey = "is_newsletter"
			} else if metaKey == "is_newsletter" {
				otherKey = "is_group_chat"
			}
			_, hasOther := contact.Metadata[otherKey]
			if contact.Metadata[metaKey] != true {
				contact.Metadata[metaKey] = true
				needsMetaUpdate = true
			}
			if hasOther {
				delete(contact.Metadata, otherKey)
				needsMetaUpdate = true
			}
			if needsMetaUpdate {
				if err := a.DB.Model(contact).Update("metadata", contact.Metadata).Error; err != nil {
					a.Log.Error("Failed to set chat metadata during GOWA sync", "error", err, "jid", jid)
				}
			}
		}

		// Stamp the owning account name. A sync from a specific GOWA server
		// means the contact belongs to that server's account, so always
		// overwrite — webhook-created rows can carry an empty or stale value
		// that would otherwise never self-heal, breaking the Contacts UI's
		// account filter. NOTE: the column is whats_app_account (GORM maps
		// the WhatsAppAccount field to whats_app_account, not
		// whatsapp_account), so the raw Update must use the DB column name.
		if contact.WhatsAppAccount != account.Name {
			if err := a.DB.Model(contact).Update("whats_app_account", account.Name).Error; err != nil {
				a.Log.Error("Failed to stamp whats_app_account during GOWA sync", "error", err, "jid", jid)
			} else {
				contact.WhatsAppAccount = account.Name
			}
		}

		// Populate the contact's WhatsApp profile picture (or group icon) so
		// the chat list shows real avatars instead of colored initials. This
		// is best-effort and skips contacts that already have an avatar_url,
		// keeping re-syncs cheap. Done here rather than per-webhook because the
		// sync already iterates every chat and GOWA's /user/avatar is a
		// per-contact round-trip we don't want on every inbound message.
		a.refreshContactAvatar(bundle.client, account, contact, deviceID, false)

		if isNew {
			created++
		}
	}

	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionUpdated, nil, map[string]any{
		"device_id":      deviceID,
		"action":         "sync_contacts",
		"chats_seen":     total,
		"contacts_total": touched,
		"contacts_new":   created,
	})
	return r.SendEnvelope(map[string]any{
		"device_id": deviceID,
		"synced":    touched,
		"created":   created,
		"total":     total,
	})
}

// gowaMsgTypeToWhatomate maps a GOWA chat-message media_type to whatomate's
// MessageType. Empty media_type means a plain-text message.
func gowaMsgTypeToWhatomate(mediaType string) models.MessageType {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image":
		return models.MessageTypeImage
	case "video":
		return models.MessageTypeVideo
	case "audio", "voice", "ptt":
		return models.MessageTypeAudio
	case "document":
		return models.MessageTypeDocument
	default:
		return models.MessageTypeText
	}
}

// SyncGowaInstanceMessages backfills a device's message history from GOWA into
// the messages table. It iterates the device's chat list (GET /chats), and for
// each chat pulls recent messages (GET /chat/{jid}/messages), upserting them as
// messages rows keyed by whats_app_message_id (idempotent), and stamps each
// contact's last_message_at/preview from its newest message.
//
// This is required because GOWA only delivers NEW messages via webhook; the
// device's existing chat history is never replayed. Without this backfill,
// contacts imported via /sync-contacts have zero messages and the chat view
// shows an empty conversation even though the device has history.
//
// PerChatLimit caps messages fetched per chat (default 50, newest-first) so
// the operation stays bounded for large histories.
//
// POST /api/gowa/servers/{id}/devices/{deviceId}/sync-messages  body: {"per_chat_limit": 50, "max_chats": 0}
func (a *App) SyncGowaInstanceMessages(r *fastglue.Request) error {
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

	// The WhatsAppAccount must already exist (provisioned by /sync) so we can
	// stamp the owning account name on every message.
	account, err := a.getGowaAccountByDeviceID(deviceID)
	if err != nil {
		a.Log.Warn("GOWA message sync requested for device with no account row", "device_id", deviceID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusConflict,
			"Device account is not provisioned yet. Click the Sync button first to set up the device account.", nil, "")
	}
	if account.OrganizationID != orgID {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "GOWA instance not found", nil, "")
	}

	var req struct {
		PerChatLimit int `json:"per_chat_limit"`
		MaxChats     int `json:"max_chats"`
	}
	_ = r.Decode(&req, "json")
	if req.PerChatLimit <= 0 || req.PerChatLimit > 100 {
		req.PerChatLimit = 50
	}

	ctx := context.Background()
	chats, totalChats, err := bundle.client.ListChats(ctx, deviceID, gowa.ListChatsOptions{Limit: 100})
	if err != nil {
		a.Log.Error("Failed to list GOWA chats for message sync", "error", err, "device", deviceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to fetch chat list from GOWA", nil, "")
	}
	if req.MaxChats > 0 && len(chats) > req.MaxChats {
		chats = chats[:req.MaxChats]
	}

	chatsWithMsgs := 0
	msgsStored := 0
	for _, ch := range chats {
		jid := strings.TrimSpace(ch.JID)
		if jid == "" {
			continue
		}

		isGroup := strings.HasSuffix(jid, "@g.us")
		isNewsletter := strings.HasSuffix(jid, "@newsletter")
		identity := jid
		if !isGroup && !isNewsletter {
			identity = gowa.PhoneFromJID(jid)
			if identity == "" {
				continue
			}
		}

		contact, _, err := contactutil.GetOrCreateContact(a.DB, orgID, identity, strings.TrimSpace(ch.Name))
		if err != nil {
			a.Log.Error("Failed to upsert contact during GOWA message sync", "error", err, "jid", jid)
			continue
		}

		// Fetch recent history (newest-first). GOWA returns the newest page
		// first at offset 0, so per_chat_limit gives us the tail of the convo.
		msgs, _, err := bundle.client.GetChatMessages(ctx, deviceID, jid, gowa.ChatMessagesOptions{Limit: req.PerChatLimit})
		if err != nil {
			a.Log.Error("Failed to fetch GOWA chat messages", "error", err, "device", deviceID, "jid", jid)
			continue
		}
		if len(msgs) == 0 {
			continue
		}

		// Stamp the owning account name (mirrors /sync-contacts): always
		// overwrite so an empty or stale value self-heals. The DB column is
		// whats_app_account (GORM's mapping of the WhatsAppAccount field),
		// not whatsapp_account.
		if contact.WhatsAppAccount != account.Name {
			if err := a.DB.Model(contact).Update("whats_app_account", account.Name).Error; err != nil {
				a.Log.Error("Failed to stamp whats_app_account during GOWA message sync", "error", err, "jid", jid)
			} else {
				contact.WhatsAppAccount = account.Name
			}
		}

		// Stamp group/newsletter metadata if not already set (mirrors
		// /sync-contacts). Groups and newsletters are distinct categories.
		metaKey := ""
		if isGroup {
			metaKey = "is_group_chat"
		} else if isNewsletter {
			metaKey = "is_newsletter"
		}
		if metaKey != "" {
			needsMetaUpdate := false
			if contact.Metadata == nil {
				contact.Metadata = models.JSONB{}
				needsMetaUpdate = true
			}
			// Groups and newsletters are mutually exclusive. Setting one clears
			// the other so legacy contacts that carry both flags self-heal.
			otherKey := ""
			if metaKey == "is_group_chat" {
				otherKey = "is_newsletter"
			} else if metaKey == "is_newsletter" {
				otherKey = "is_group_chat"
			}
			_, hasOther := contact.Metadata[otherKey]
			if contact.Metadata[metaKey] != true {
				contact.Metadata[metaKey] = true
				needsMetaUpdate = true
			}
			if hasOther {
				delete(contact.Metadata, otherKey)
				needsMetaUpdate = true
			}
			if needsMetaUpdate {
				a.DB.Model(contact).Update("metadata", contact.Metadata)
			}
		}

		// Bulk-insert messages, skipping any whose whats_app_message_id already
		// exists (idempotent re-sync). GORM Clause OnConflict would need a
		// unique constraint on the message id; we instead pre-filter by querying
		// existing ids for this chat to avoid a schema change. Scoped to this
		// account: two org accounts chatting with each other share wamids across
		// their copies, and syncing one device must not skip messages that only
		// exist as the other account's copy.
		existing := make(map[string]bool, len(msgs))
		{
			ids := make([]string, 0, len(msgs))
			for _, m := range msgs {
				if m.ID != "" {
					ids = append(ids, m.ID)
				}
			}
			if len(ids) > 0 {
				var found []string
				a.DB.Model(&models.Message{}).
					Where("whats_app_message_id IN ? AND organization_id = ? AND whats_app_account = ?",
						ids, account.OrganizationID, account.Name).
					Pluck("whats_app_message_id", &found)
				for _, id := range found {
					existing[id] = true
				}
			}
		}

		var newest models.Message
		for _, m := range msgs {
			if m.ID == "" || existing[m.ID] {
				continue
			}
			ts := gowa.ParseTimestamp(m.Timestamp)
			if ts.IsZero() {
				ts = time.Now()
			}
			direction := models.DirectionIncoming
			status := models.MessageStatusReceived
			if m.IsFromMe {
				direction = models.DirectionOutgoing
				status = models.MessageStatusSent
			}
			msgType := gowaMsgTypeToWhatomate(m.MediaType)
			// For media messages, prefer the stored content as caption only if
			// it's a text body; otherwise leave content empty (media lives in
			// MediaURL/Filename).
			content := m.Content
			if msgType != models.MessageTypeText {
				content = "" // caption is not reliably separated by GOWA history
			}

			msg := models.Message{
				// Set CreatedAt to the message's real timestamp so historical
				// messages render in chronological order in the chat view
				// (GORM's autoCreateTime honors an explicitly-set value).
				BaseModel:         models.BaseModel{ID: uuid.New(), CreatedAt: ts},
				OrganizationID:    orgID,
				WhatsAppAccount:   account.Name,
				ContactID:         contact.ID,
				WhatsAppMessageID: m.ID,
				Direction:         direction,
				MessageType:       msgType,
				Content:           content,
				// NOTE: MediaURL is intentionally left empty here. History sync
				// gives us GOWA's server-side URL (m.URL), NOT a local file path —
				// the bytes were never downloaded to disk. Writing m.URL into
				// media_url would create a lying row: ServeMedia would try to serve
				// a non-existent local file and 404. Instead, leave media_url empty
				// so the row is honest ("no local media yet"), and let ServeMedia's
				// auto-recovery lazily download the bytes via WhatsAppMessageID on
				// first view. See internal/handlers/media.go ServeMedia for the
				// recovery path. Stash the original GOWA URL in metadata as a
				// fallback in case the message-ID-based recovery is unavailable.
				MediaURL:      "",
				MediaFilename: m.Filename,
				Status:        status,
			}
			// Preserve the original timestamp via Metadata so the UI can render
			// historical order even though created_at is now.
			if msg.Metadata == nil {
				msg.Metadata = models.JSONB{}
			}
			msg.Metadata["synced_from_history"] = true
			msg.Metadata["gowa_timestamp"] = m.Timestamp
			if m.URL != "" {
				// Keep GOWA's original URL for potential lazy recovery. Not used
				// as a local path — only as a hint for future download attempts.
				msg.Metadata["gowa_media_url"] = m.URL
			}

			if err := a.DB.Create(&msg).Error; err != nil {
				a.Log.Error("Failed to store GOWA history message", "error", err, "msg_id", m.ID)
				continue
			}
			msgsStored++

			// Track the newest message (largest timestamp) for the contact stamp.
			// GOWA returns newest-first, so the first non-skipped msg is newest.
			if newest.ID == uuid.Nil {
				newest = msg
			}
		}

		// Stamp the contact's last_message_at/preview from the newest message.
		if newest.ID != uuid.Nil {
			preview := getMessagePreviewFromContent(newest.MessageType, newest.Content)
			// Use the chat's last_message_time (authoritative from GOWA) when available.
			lastAt := gowa.ParseTimestamp(ch.LastMessageTime)
			if lastAt.IsZero() {
				lastAt = time.Now()
			}
			a.DB.Model(contact).Updates(map[string]any{
				"last_message_at":      lastAt,
				"last_message_preview": preview,
			})
			chatsWithMsgs++
		}
	}

	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionUpdated, nil, map[string]any{
		"device_id":       deviceID,
		"action":          "sync_messages",
		"chats_seen":      totalChats,
		"chats_with_msgs": chatsWithMsgs,
		"messages_stored": msgsStored,
		"per_chat_limit":  req.PerChatLimit,
	})
	return r.SendEnvelope(map[string]any{
		"device_id":       deviceID,
		"chats_seen":      totalChats,
		"chats_with_msgs": chatsWithMsgs,
		"messages_stored": msgsStored,
	})
}

// getMessagePreviewFromContent returns a short preview string for a message,
// mirroring updateContactLastMessage/getMessagePreview logic but for history.
func getMessagePreviewFromContent(msgType models.MessageType, content string) string {
	switch msgType {
	case models.MessageTypeText:
		return truncateString(content, 100)
	case models.MessageTypeImage:
		return "[Image]"
	case models.MessageTypeVideo:
		return "[Video]"
	case models.MessageTypeAudio:
		return "[Audio]"
	case models.MessageTypeDocument:
		return "[Document]"
	default:
		return truncateString(content, 100)
	}
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
		WebhookEvents: "message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited,call.offer",
	})
	if err != nil {
		a.Log.Error("Failed to create GOWA device", "error", err, "instance", bundle.instance.Name)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to create device on GOWA", nil, "")
	}

	// Provision the matching WhatsAppAccount row so incoming webhooks for this
	// device can resolve an account (getGowaAccountByDeviceID). Without it the
	// webhook receiver returns 403 and no chat/message is ever stored, so the
	// device appears connected but its chats never reach the inbox. A freshly
	// created device has no JID yet; PhoneID stays empty and is backfilled by
	// the connect webhook or a later /sync.
	jid := lookupGowaDeviceJID(ctx, bundle.client, device.ID)
	account, accountErr := a.ensureGowaAccountForDevice(orgID, userID, ensureGowaAccountOpts{
		InstanceName:  bundle.instance.Name,
		BaseURL:       bundle.instance.BaseURL,
		DeviceID:      device.ID,
		WebhookSecret: webhookSecret,
		PreferredName: req.DeviceName,
		DeviceJID:     jid,
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
