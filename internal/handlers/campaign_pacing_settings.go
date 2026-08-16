package handlers

import (
	"fmt"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/zerodha/fastglue"
)

// Campaign send pacing settings: the account-level messages/minute budget the
// worker enforces before each campaign send (see internal/worker/pacing.go).
// 0/absent = fall back to [campaigns].default_pacing_per_minute from the
// config file, and 0 there too = unlimited (historical behavior).

// sendPacingSettings is the account's send_pacing settings block.
type sendPacingSettings struct {
	// MessagesPerMinute caps campaign sends for this account. 0 = inherit
	// the config-file default.
	MessagesPerMinute int `json:"messages_per_minute"`
}

// GetSendPacingSettings returns the account's effective pacing: the settings
// block value, or the config default when unset (so the UI can show what
// actually applies).
// GET /api/accounts/{id}/send-pacing
func (a *App) GetSendPacingSettings(r *fastglue.Request) error {
	account, ok := a.getAccountSettingsBlock(r)
	if !ok {
		return nil
	}
	effective := 0
	if block, ok := account.Settings["send_pacing"].(map[string]any); ok {
		if v, ok := block["messages_per_minute"].(float64); ok && int(v) > 0 {
			effective = int(v)
		}
	}
	if effective == 0 && a.Config != nil {
		effective = a.Config.Campaigns.DefaultPacingPerMinute
	}
	return r.SendEnvelope(sendPacingSettings{MessagesPerMinute: effective})
}

// UpdateSendPacingSettings replaces the account's send_pacing block.
// PUT /api/accounts/{id}/send-pacing
func (a *App) UpdateSendPacingSettings(r *fastglue.Request) error {
	return a.updateAccountSettingsBlock(r, accountSettingsBlock{
		Key:      "send_pacing",
		Resource: models.ResourceAccounts,
		Decode: func(body []byte) (map[string]any, error) {
			var req sendPacingSettings
			if err := decodeJSONSettingsBody(body, &req); err != nil {
				return nil, err
			}
			if req.MessagesPerMinute < 0 || req.MessagesPerMinute > 1000 {
				return nil, fmt.Errorf("messages_per_minute must be between 0 and 1000 (0 = use the server default)")
			}
			return map[string]any{
				"messages_per_minute": req.MessagesPerMinute,
			}, nil
		},
	})
}
