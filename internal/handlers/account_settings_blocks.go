package handlers

import (
	"encoding/json"
	"errors"

	"github.com/compnew2006/gowa-ui/internal/audit"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Per-account settings blocks (call_auto_reject, close_rating, daily_reset,
// …) all persist one JSON object under WhatsAppAccount.Settings[key] and
// share the same handler skeleton: auth → parse/validate body → load account
// → snapshot old block → replace block → save → audit → respond. This file
// owns that skeleton so each block only declares its key, audit resource,
// body decoding/validation, and an optional post-save hook.

// errInvalidSettingsBody is the shared 400 message for unparseable bodies.
var errInvalidSettingsBody = errors.New("Invalid request body")

// settingsBlockSnapshot extracts one settings block for audit diffing.
func settingsBlockSnapshot(settings models.JSONB, key string) map[string]any {
	block, _ := settings[key].(map[string]any)
	return map[string]any{key: block}
}

// loadAccountByIDPath resolves the "id" path parameter to a WhatsApp account
// scoped to orgID. Error responses are already sent on failure; the second
// return value reports whether the account was loaded.
func (a *App) loadAccountByIDPath(r *fastglue.Request, orgID uuid.UUID) (*models.WhatsAppAccount, bool) {
	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil, false
	}
	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil, false
	}
	return account, true
}

// accountSettingsBlock describes one per-account settings block endpoint.
type accountSettingsBlock struct {
	// Key is the WhatsAppAccount.Settings key (e.g. "close_rating"); also the
	// audit-snapshot key and the default name used in error logs.
	Key string

	// Resource is the audit resource logged on update (e.g.
	// models.ResourceSettingsCloseRating).
	Resource string

	// Decode parses and validates the request body and returns the block
	// value to store. A non-nil error is surfaced as a 400 with its message.
	Decode func(body []byte) (map[string]any, error)

	// AfterSave runs after a successful persist (e.g. repairing the GOWA
	// webhook subscription when call auto-reject is switched on). Optional.
	AfterSave func(a *App, account *models.WhatsAppAccount, block map[string]any)
}

// getAccountSettingsBlock serves the GET side of a per-account settings
// block: auth + account resolution, leaving the response shape to the caller.
func (a *App) getAccountSettingsBlock(r *fastglue.Request) (*models.WhatsAppAccount, bool) {
	orgID, _, err := a.requireAuth(r, models.ResourceAccounts, models.ActionRead)
	if err != nil {
		return nil, false
	}
	return a.loadAccountByIDPath(r, orgID)
}

// updateAccountSettingsBlock serves the PUT side of a per-account settings
// block. The frontend always sends the full block, so this is a full
// replacement — no per-field partial-update semantics.
func (a *App) updateAccountSettingsBlock(r *fastglue.Request, blk accountSettingsBlock) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceAccounts, models.ActionWrite)
	if err != nil {
		return nil
	}

	block, err := blk.Decode(r.RequestCtx.PostBody())
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	account, ok := a.loadAccountByIDPath(r, orgID)
	if !ok {
		return nil
	}

	if account.Settings == nil {
		account.Settings = models.JSONB{}
	}
	oldSnapshot := settingsBlockSnapshot(account.Settings, blk.Key)
	account.Settings[blk.Key] = block

	if err := a.DB.Model(account).Update("settings", account.Settings).Error; err != nil {
		a.Log.Error("Failed to update "+blk.Key+" settings", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update settings", nil, "")
	}

	userName := audit.GetUserName(a.DB, userID)
	audit.LogAudit(a.DB, orgID, userID, userName,
		blk.Resource, account.ID, models.AuditActionUpdated,
		oldSnapshot, settingsBlockSnapshot(account.Settings, blk.Key))

	if blk.AfterSave != nil {
		blk.AfterSave(a, account, block)
	}

	return r.SendEnvelope(map[string]any{
		"message": "Settings updated successfully",
	})
}

// decodeJSONSettingsBody unmarshals a settings-block request body, mapping
// parse failures to the shared 400 message.
func decodeJSONSettingsBody(body []byte, v any) error {
	if err := json.Unmarshal(body, v); err != nil {
		return errInvalidSettingsBody
	}
	return nil
}
