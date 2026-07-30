package handlers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/shridarpatil/whatomate/internal/audit"
	"github.com/shridarpatil/whatomate/internal/contactutil"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Call auto-reject: when enabled for a WhatsApp account, incoming voice/video
// calls (GOWA "call.offer" webhook) are rejected via POST /call/reject while
// still ringing, and an optional automated text is sent to the caller.
//
// Per-account on purpose (same rationale as close_rating): each number belongs
// to a different branch with its own policy and wording, so the toggle and the
// message live in WhatsAppAccount.Settings["call_auto_reject"], edited from
// Settings → Accounts → account.

const defaultCallAutoRejectMessage = "عذرًا، لا نستقبل مكالمات على هذا الرقم. من فضلك أرسل رسالة نصية وسنرد عليك في أقرب وقت.\nSorry, we can't take calls on this number. Please send a text message and we'll get back to you shortly."

// callAutoRejectSettings is read from
// WhatsAppAccount.Settings["call_auto_reject"]:
//
//	{"enabled": true, "message": "..."}
//
// Disabled by default. An explicit empty message disables the automated
// text while keeping the rejection itself (parse-side contract, mirroring
// close_rating's empty-thanks semantics).
type callAutoRejectSettings struct {
	Enabled bool
	Message string
}

func callAutoRejectSettingsForAccount(account *models.WhatsAppAccount) callAutoRejectSettings {
	s := callAutoRejectSettings{
		Message: defaultCallAutoRejectMessage,
	}
	return parseCallAutoRejectSettings(account.Settings, s)
}

// parseCallAutoRejectSettings applies the "call_auto_reject" block of the
// account settings JSONB on top of the defaults. Split out for table-driven
// tests.
func parseCallAutoRejectSettings(settings models.JSONB, s callAutoRejectSettings) callAutoRejectSettings {
	raw, ok := settings["call_auto_reject"].(map[string]any)
	if !ok {
		return s
	}
	if v, ok := raw["enabled"].(bool); ok {
		s.Enabled = v
	}
	if v, ok := raw["message"].(string); ok {
		// An explicit empty string disables the automated message.
		s.Message = strings.TrimSpace(v)
	}
	return s
}

// processGowaCallOffer handles the GOWA "call.offer" webhook event: while the
// call is still ringing it is rejected through the GOWA REST API, then the
// account's automated message (if any) is sent to the caller. Runs in a
// goroutine from handleGowaWebhook.
func (a *App) processGowaCallOffer(account *models.WhatsAppAccount, envelope *gowa.WebhookPayload) {
	defer func() {
		if rv := recover(); rv != nil {
			a.Log.Error("Panic in processGowaCallOffer", "panic", rv, "device_id", envelope.DeviceID)
		}
	}()

	settings := callAutoRejectSettingsForAccount(account)
	if !settings.Enabled {
		return
	}

	// Dump the raw payload at INFO level so we can see every field GOWA sends
	// — the `from` can be a WhatsApp LID (internal privacy ID) that is NOT a
	// phone-number JID, and we need to find the real caller identity.
	a.Log.Info("GOWA call.offer raw payload",
		"device_id", envelope.DeviceID, "payload", string(envelope.Payload))

	var call gowa.CallOfferPayload
	if err := json.Unmarshal(envelope.Payload, &call); err != nil {
		a.Log.Error("Failed to parse GOWA call.offer payload", "error", err)
		return
	}
	a.Log.Info("GOWA call.offer parsed",
		"from", call.From, "call_id", call.CallID, "chat_id", call.ChatID)
	if call.CallID == "" || call.From == "" {
		a.Log.Warn("GOWA call.offer missing call_id or from", "device_id", envelope.DeviceID)
		return
	}

	provider := a.resolveProvider(account)
	gowaClient, ok := provider.(*gowa.Client)
	if !ok {
		a.Log.Error("GOWA provider not available for call reject", "account", account.Name)
		return
	}

	// Rejection only succeeds while the call is still ringing; if it already
	// ended GOWA returns an error — skip the automated message in that case
	// so callers who hung up (or were answered) don't get a rejection text.
	if err := gowaClient.RejectCall(context.Background(), account.GowaDeviceID, call.From, call.CallID); err != nil {
		a.Log.Error("Failed to reject incoming call", "error", err,
			"account", account.Name, "call_id", call.CallID)
		return
	}
	a.Log.Info("Auto-rejected incoming call",
		"account", account.Name, "caller", call.From, "call_id", call.CallID)

	if settings.Message == "" {
		return
	}

	// Determine the messaging phone number for the caller.
	//
	// call.From is always used for /call/reject (GOWA requires the exact
	// signaling address), but it may be a WhatsApp LID (Linked ID — a
	// privacy-preserving internal ID) rather than a phone-number JID.
	// /send/message only accepts phone-number JIDs and rejects LIDs with
	// "is not on whatsapp".
	//
	// Resolution order:
	//  1. chat_id (if GOWA includes it — phone-based conversation JID)
	//  2. Resolve the `from` LID via GOWA's /user/info endpoint
	//  3. Fall back to the raw `from` digits (may fail at send time)
	var phone string
	if call.ChatID != "" {
		phone = gowa.PhoneFromJID(call.ChatID)
	}
	if phone == "" {
		// Try to resolve the `from` field as a LID. GOWA's /user/info
		// endpoint accepts "<number>@lid" and returns the real phone number.
		resolved, err := gowaClient.ResolveLID(context.Background(), account.GowaDeviceID, call.From)
		if err != nil {
			a.Log.Warn("Failed to resolve caller LID via GOWA /user/info",
				"error", err, "from", call.From, "account", account.Name)
		}
		if resolved != "" {
			phone = resolved
			a.Log.Info("Resolved caller LID to phone number",
				"from", call.From, "phone", phone, "account", account.Name)
		}
	}
	if phone == "" {
		// Last resort: extract digits from `from`. This will likely fail at
		// send time if `from` is a LID, but try anyway in case it's a real JID.
		phone = gowa.PhoneFromJID(call.From)
		if phone == "" {
			return
		}
		a.Log.Warn("Could not resolve caller LID; using raw 'from' digits — message may fail",
			"from", call.From, "phone", phone, "account", account.Name)
	}
	contact, isNew, err := contactutil.GetOrCreateContact(a.DB, account.OrganizationID, phone, "")
	if err != nil {
		a.Log.Error("Failed to get or create contact for rejected call", "error", err, "caller", call.From)
		return
	}
	// Stamp the receiving account on fresh contacts so the conversation shows
	// up under the correct account tab (mirrors the GOWA contact sync).
	if isNew && contact.WhatsAppAccount == "" {
		if err := contactutil.StampAccountName(a.DB, contact, account.Name); err != nil {
			a.Log.Error("Failed to stamp whats_app_account for rejected call contact", "error", err, "contact_id", contact.ID)
		}
	}

	if err := a.sendAndSaveTextMessage(account, contact, settings.Message); err != nil {
		a.Log.Error("Failed to send call auto-reject message", "error", err, "contact_id", contact.ID)
	}
}

// CallAutoRejectSettingsResponse is the effective call-auto-reject
// configuration (account overrides merged over the built-in defaults).
type CallAutoRejectSettingsResponse struct {
	Enabled bool   `json:"enabled"`
	Message string `json:"message"`
}

// GetCallAutoRejectSettings returns the effective call-auto-reject settings
// for a WhatsApp account.
func (a *App) GetCallAutoRejectSettings(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceAccounts, models.ActionRead)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil
	}

	s := callAutoRejectSettingsForAccount(account)
	return r.SendEnvelope(CallAutoRejectSettingsResponse{
		Enabled: s.Enabled,
		Message: s.Message,
	})
}

// callAutoRejectSnapshot extracts the call_auto_reject block for audit diffing.
func callAutoRejectSnapshot(settings models.JSONB) map[string]any {
	block, _ := settings["call_auto_reject"].(map[string]any)
	return map[string]any{"call_auto_reject": block}
}

// UpdateCallAutoRejectSettings replaces the account's call_auto_reject
// settings block. The frontend always sends the full block, so this is a
// full replacement — no per-field partial-update semantics.
func (a *App) UpdateCallAutoRejectSettings(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceAccounts, models.ActionWrite)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	var req struct {
		Enabled bool   `json:"enabled"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil
	}

	if account.Settings == nil {
		account.Settings = models.JSONB{}
	}
	oldSnapshot := callAutoRejectSnapshot(account.Settings)

	// An explicit empty message disables the automated text; the rejection
	// itself is governed solely by the enabled flag.
	account.Settings["call_auto_reject"] = map[string]any{
		"enabled": req.Enabled,
		"message": strings.TrimSpace(req.Message),
	}

	if err := a.DB.Model(account).Update("settings", account.Settings).Error; err != nil {
		a.Log.Error("Failed to update call-auto-reject settings", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update settings", nil, "")
	}

	userName := audit.GetUserName(a.DB, userID)
	audit.LogAudit(a.DB, orgID, userID, userName,
		models.ResourceSettingsCallAutoReject, account.ID, models.AuditActionUpdated,
		oldSnapshot, callAutoRejectSnapshot(account.Settings))

	// Devices registered before call.offer existed keep their old webhook
	// subscription on the GOWA server and never receive call events — repair
	// the subscription whenever the feature is switched on.
	if req.Enabled {
		a.ensureCallOfferSubscription(account)
	}

	return r.SendEnvelope(map[string]any{
		"message": "Settings updated successfully",
	})
}

// ensureCallOfferSubscription makes sure the account's GOWA device webhook
// subscription includes the "call.offer" event. Best-effort: failures are
// logged, not surfaced — the settings save must not depend on GOWA being up.
func (a *App) ensureCallOfferSubscription(account *models.WhatsAppAccount) {
	if account.GowaDeviceID == "" {
		return
	}
	gowaClient, ok := a.resolveProvider(account).(*gowa.Client)
	if !ok {
		return
	}

	ctx := context.Background()
	cfg, err := gowaClient.GetDeviceWebhook(ctx, account.GowaDeviceID)
	if err != nil {
		a.Log.Warn("Could not read GOWA device webhook to verify call.offer subscription",
			"error", err, "account", account.Name, "device_id", account.GowaDeviceID)
		return
	}
	// An empty events string means the device receives every event.
	if cfg.WebhookEvents == "" || strings.Contains(cfg.WebhookEvents, "call.offer") {
		return
	}

	cfg.WebhookEvents += ",call.offer"
	if _, err := gowaClient.SetDeviceWebhook(ctx, account.GowaDeviceID, *cfg); err != nil {
		a.Log.Error("Failed to add call.offer to GOWA device webhook subscription",
			"error", err, "account", account.Name, "device_id", account.GowaDeviceID)
		return
	}
	a.Log.Info("Subscribed GOWA device to call.offer events",
		"account", account.Name, "device_id", account.GowaDeviceID)
}
