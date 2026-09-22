package handlers

import (
	"fmt"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/zerodha/fastglue"
)

// Per-account media retention: after `retention_days` days, a message's local
// media file is deleted from disk and its row's media_url cleared (with a
// metadata tombstone) by the MediaRetentionProcessor (media_retention_
// processor.go). Absent block or enabled=false keeps files forever — the
// historical behavior. Age is the message row's created_at, not the file's
// download time: a file recovered today for a months-old message is already
// past retention and is purged on the next pass.

// mediaRetentionSettings is the account's media_retention settings block.
type mediaRetentionSettings struct {
	Enabled       bool `json:"enabled"`
	RetentionDays int  `json:"retention_days"`
	// Effective reports whether retention actually applies (block present,
	// enabled, valid day range) — the same condition the processor scans on.
	Effective bool `json:"effective"`
}

// GetMediaRetentionSettings returns the account's stored media-retention
// block plus whether it is effective.
// GET /api/accounts/{id}/media-retention
func (a *App) GetMediaRetentionSettings(r *fastglue.Request) error {
	account, ok := a.getAccountSettingsBlock(r)
	if !ok {
		return nil
	}
	s := mediaRetentionSettings{}
	if block, ok := account.Settings["media_retention"].(map[string]any); ok {
		if v, ok := block["enabled"].(bool); ok {
			s.Enabled = v
		}
		if v, ok := block["retention_days"].(float64); ok {
			s.RetentionDays = int(v)
		}
	}
	s.Effective = s.Enabled && s.RetentionDays >= 1 && s.RetentionDays <= 3650
	return r.SendEnvelope(s)
}

// UpdateMediaRetentionSettings replaces the account's media_retention block.
// PUT /api/accounts/{id}/media-retention
func (a *App) UpdateMediaRetentionSettings(r *fastglue.Request) error {
	return a.updateAccountSettingsBlock(r, accountSettingsBlock{
		Key:      "media_retention",
		Resource: models.ResourceSettingsMediaRetention,
		Decode: func(body []byte) (map[string]any, error) {
			var req mediaRetentionSettings
			if err := decodeJSONSettingsBody(body, &req); err != nil {
				return nil, err
			}
			if req.Enabled && (req.RetentionDays < 1 || req.RetentionDays > 3650) {
				return nil, fmt.Errorf("retention_days must be between 1 and 3650 when retention is enabled")
			}
			return map[string]any{
				"enabled":        req.Enabled,
				"retention_days": req.RetentionDays,
			}, nil
		},
		// Disabling retention must not leave stale keep-until windows behind:
		// a restored file's retention_keep_until would otherwise survive the
		// off period and shield the file after a later re-enable. Best-effort
		// cleanup scoped to this account's messages (matched by org + the
		// account name string, the same reference the purge scan uses). The
		// key-existence predicate keeps this from rewriting every message
		// row of the account — only rows that actually carry retention keys
		// are touched (no locks/WAL storm on disable).
		AfterSave: func(a *App, account *models.WhatsAppAccount, block map[string]any) {
			enabled, _ := block["enabled"].(bool)
			if !enabled {
				if err := a.DB.Exec(
					`UPDATE messages SET metadata = COALESCE(metadata, '{}'::jsonb) - 'retention_keep_until' - 'retention_restored_at' WHERE organization_id = ? AND whats_app_account = ? AND ((metadata -> 'retention_keep_until') IS NOT NULL OR (metadata -> 'retention_restored_at') IS NOT NULL)`,
					account.OrganizationID, account.Name).Error; err != nil {
					a.Log.Warn("Media retention: failed to clear keep-until windows on disable",
						"account_id", account.ID, "error", err)
				}
				return
			}
			// Clamp previously granted keep-until windows to the (possibly
			// lowered) window: without this, lowering 3650 → 30 days would
			// leave old far-future windows shielding files for years. Only
			// rows carrying the key are rewritten. The block here comes from
			// Decode (Go int), not from a JSONB round-trip (float64) — accept
			// both.
			days := 0
			switch v := block["retention_days"].(type) {
			case int:
				days = v
			case float64:
				days = int(v)
			}
			if days < 1 {
				return
			}
			if err := a.DB.Exec(
				`UPDATE messages SET metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object('retention_keep_until', LEAST((metadata ->> 'retention_keep_until')::timestamptz, now() + (? * interval '1 day'))) WHERE organization_id = ? AND whats_app_account = ? AND (metadata ->> 'retention_keep_until') IS NOT NULL AND (metadata ->> 'retention_keep_until')::timestamptz > now() + (? * interval '1 day')`,
				int(days), account.OrganizationID, account.Name, int(days)).Error; err != nil {
				a.Log.Warn("Media retention: failed to clamp keep-until windows",
					"account_id", account.ID, "error", err)
			}
		},
	})
}
