package handlers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// gowaAccount holds the resolved GOWA client and its DB account.
type gowaAccount struct {
	client  *gowa.Client
	account *models.WhatsAppAccount
}

// resolveGowaAccount resolves the account from the URL, validates it's a GOWA
// account, and type-asserts the provider to *gowa.Client. On error it sends the
// HTTP response and returns ok=false — callers should `return nil` immediately.
// Callers must have already authenticated (and authorized) via requireAuth and
// pass the resulting orgID here — this helper does NOT re-check permissions.
func (a *App) resolveGowaAccount(r *fastglue.Request, orgID uuid.UUID) (gowaAccount, bool) {
	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return gowaAccount{}, false
	}

	account, err := a.resolveWhatsAppAccountByID(r, id, orgID)
	if err != nil {
		return gowaAccount{}, false
	}

	if account.GowaBaseURL == "" || account.GowaDeviceID == "" {
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Account has no GOWA configuration", nil, "")
		return gowaAccount{}, false
	}

	provider := a.resolveProvider(account)
	gowaClient, ok := provider.(*gowa.Client)
	if !ok {
		_ = r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "GOWA provider not available", nil, "")
		return gowaAccount{}, false
	}

	return gowaAccount{client: gowaClient, account: account}, true
}

// GowaLoginQR retrieves a QR code for pairing a GOWA device.
// If the device is already connected, returns already_connected=true with the JID
// instead of attempting to fetch a new QR (which GOWA would reject).
// The QR image is fetched from GOWA and returned as a base64 data URI so the
// browser can render it directly without needing Basic Auth credentials.
// GET /api/accounts/{id}/gowa/qr

// GowaPairCode requests a phone-code pairing for a GOWA device.
// POST /api/accounts/{id}/gowa/pair-code  body: {"phone": "16505551234"}
func (a *App) GowaPairCode(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}

	ga, ok := a.resolveGowaAccount(r, orgID)
	if !ok {
		return nil
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

	ctx := context.Background()
	result, err := ga.client.LoginWithCode(ctx, ga.account.GowaDeviceID, req.Phone)
	if err != nil {
		a.Log.Error("Failed to get GOWA pair code", "error", err, "account", ga.account.Name)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to get pair code from GOWA", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"pair_code": result.PairCode,
	})
}

// GowaInstances lists the configured GOWA instances (for the account-creation dropdown).
// GET /api/gowa/instances
func (a *App) GowaInstances(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceDevices, models.ActionRead)
	if err != nil {
		return nil
	}

	type gowaInstanceResponse struct {
		Name    string `json:"name"`
		BaseURL string `json:"base_url"`
	}

	allowed := a.Config.FindGOWAInstancesForOrg(orgID.String())
	instances := make([]gowaInstanceResponse, 0, len(allowed))
	for _, inst := range allowed {
		instances = append(instances, gowaInstanceResponse{
			Name:    inst.Name,
			BaseURL: inst.BaseURL,
		})
	}

	return r.SendEnvelope(map[string]any{
		"instances": instances,
	})
}

// GowaCreateDevice creates a new device on a GOWA instance and returns the device ID
// and webhook secret. Used during account creation to auto-provision a device.
// POST /api/gowa/create-device  body: {"base_url": "...", "device_name": "..."}
func (a *App) GowaCreateDevice(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceDevices, models.ActionWrite)
	if err != nil {
		return nil
	}

	var req struct {
		BaseURL    string `json:"base_url"`
		DeviceName string `json:"device_name"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	if req.BaseURL == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "base_url is required", nil, "")
	}
	if req.DeviceName == "" {
		req.DeviceName = "gowa-ui-device"
	}

	// Find credentials for this instance, scoped to the caller's org.
	inst := a.Config.FindGOWAInstance(req.BaseURL, orgID.String())
	if inst == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Unknown GOWA instance for your organization", nil, "")
	}

	// Create a GOWA client for this instance.
	gowaClient := gowa.New(inst.BaseURL, inst.Username, inst.Password)

	ctx := context.Background()

	// Create the device on GOWA with webhook pointing back to gowa-ui.
	// Prefer the instance-configured webhook URL; fall back to deriving from
	// the request host (works when GOWA and gowa-ui are on the same host).
	webhookURL := inst.WebhookURL
	if webhookURL == "" {
		webhookURL = fmt.Sprintf("%s://%s%s", "http", r.RequestCtx.Host(), a.Config.GOWA.WebhookPath)
	}
	webhookSecret := gowa.GenerateWebhookSecret()
	deviceID := gowa.GenerateDeviceID(req.DeviceName)

	device, err := gowaClient.CreateDevice(ctx, deviceID, gowa.WebhookConfig{
		WebhookURL:    webhookURL,
		WebhookSecret: webhookSecret,
		WebhookEvents: "message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited,call.offer",
	})
	if err != nil {
		a.Log.Error("Failed to create GOWA device", "error", err, "base_url", req.BaseURL)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to create device on GOWA", nil, "")
	}

	// GOWA device IDs are external string identifiers, not DB UUIDs, so use
	// uuid.Nil as the resource ID; the actual device_id is captured in the
	// audit change metadata below.
	a.logAudit(orgID, userID, "devices", uuid.Nil, models.AuditActionCreated, nil, map[string]any{
		"device_id": device.ID,
		"base_url":  req.BaseURL,
	})

	return r.SendEnvelope(map[string]any{
		"device_id":      device.ID,
		"webhook_secret": webhookSecret,
		"base_url":       req.BaseURL,
	})
}

// GowaDeviceStatus retrieves the connection status of a GOWA device.
// GET /api/accounts/{id}/gowa/status
