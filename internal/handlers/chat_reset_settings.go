package handlers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/shridarpatil/whatomate/internal/audit"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Daily chat-reset schedule: when enabled for a WhatsApp account, the
// ChatResetProcessor returns every assigned (open) conversation back to the
// pending pool once per day at the configured wall-clock time. This prevents
// chats from being stuck with inactive agents indefinitely.
//
// Per-account on purpose (same rationale as close_rating / call_auto_reject):
// each number belongs to a different branch with its own operating hours and
// staffing, so the toggle and the reset time live in
// WhatsAppAccount.Settings["daily_reset"], edited from Settings → Accounts.

// ChatResetSettingsResponse is the effective daily-reset configuration
// (account overrides merged over the built-in defaults) as shown in the UI.
type ChatResetSettingsResponse struct {
	Enabled  bool   `json:"enabled"`
	Time     string `json:"time"`     // HH:MM, 24-hour, server-local unless timezone is set
	Timezone string `json:"timezone"` // IANA timezone name (e.g. "Asia/Dubai"); empty = server local
}

// chatResetSettings is read from WhatsAppAccount.Settings["daily_reset"]:
//
//	{"enabled": true, "time": "02:00", "timezone": "Asia/Dubai"}
//
// Disabled by default.
type chatResetSettings struct {
	Enabled  bool
	Time     string
	Timezone string
}

func chatResetSettingsForAccount(account *models.WhatsAppAccount) chatResetSettings {
	return parseChatResetSettings(account.Settings)
}

// parseChatResetSettings applies the "daily_reset" block of the account
// settings JSONB on top of the defaults. Split out for table-driven tests.
func parseChatResetSettings(settings models.JSONB) chatResetSettings {
	s := chatResetSettings{
		Time: "02:00", // sensible default: early morning, low traffic
	}
	raw, ok := settings["daily_reset"].(map[string]any)
	if !ok {
		return s
	}
	if v, ok := raw["enabled"].(bool); ok {
		s.Enabled = v
	}
	if v, ok := raw["time"].(string); ok {
		s.Time = strings.TrimSpace(v)
	}
	if v, ok := raw["timezone"].(string); ok {
		s.Timezone = strings.TrimSpace(v)
	}
	return s
}

// chatResetSnapshot extracts the daily_reset block for audit diffing.
func chatResetSnapshot(settings models.JSONB) map[string]any {
	block, _ := settings["daily_reset"].(map[string]any)
	return map[string]any{"daily_reset": block}
}

// GetChatResetSettings returns the effective daily-reset settings for a
// WhatsApp account. Route: GET /api/accounts/{id}/daily-reset
func (a *App) GetChatResetSettings(r *fastglue.Request) error {
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

	s := chatResetSettingsForAccount(account)
	return r.SendEnvelope(ChatResetSettingsResponse{
		Enabled:  s.Enabled,
		Time:     s.Time,
		Timezone: s.Timezone,
	})
}

// UpdateChatResetSettings replaces the account's daily_reset settings block.
// The frontend always sends the full block, so this is a full replacement.
// Route: PUT /api/accounts/{id}/daily-reset
func (a *App) UpdateChatResetSettings(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceAccounts, models.ActionWrite)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	var req struct {
		Enabled  bool   `json:"enabled"`
		Time     string `json:"time"`
		Timezone string `json:"timezone"`
	}
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Validate time format (HH:MM, 24-hour).
	req.Time = strings.TrimSpace(req.Time)
	if req.Time == "" {
		req.Time = "02:00"
	}
	if _, err := time.Parse("15:04", req.Time); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"time must be in HH:MM 24-hour format (e.g. \"02:00\")", nil, "")
	}

	// Validate timezone if provided.
	req.Timezone = strings.TrimSpace(req.Timezone)
	if req.Timezone != "" {
		if _, err := time.LoadLocation(req.Timezone); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
				"timezone must be a valid IANA timezone name (e.g. \"Asia/Dubai\")", nil, "")
		}
	}

	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil
	}

	if account.Settings == nil {
		account.Settings = models.JSONB{}
	}
	oldSnapshot := chatResetSnapshot(account.Settings)

	account.Settings["daily_reset"] = map[string]any{
		"enabled":  req.Enabled,
		"time":     req.Time,
		"timezone": req.Timezone,
	}

	if err := a.DB.Model(account).Update("settings", account.Settings).Error; err != nil {
		a.Log.Error("Failed to update daily-reset settings", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update settings", nil, "")
	}

	userName := audit.GetUserName(a.DB, userID)
	audit.LogAudit(a.DB, orgID, userID, userName,
		models.ResourceSettingsChatReset, account.ID, models.AuditActionUpdated,
		oldSnapshot, chatResetSnapshot(account.Settings))

	return r.SendEnvelope(map[string]any{
		"message": "Settings updated successfully",
	})
}
