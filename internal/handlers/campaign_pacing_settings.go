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
	// Stored is the raw account-level value (0 = not configured on the
	// account). Effective is what actually applies (stored → config
	// default → 0 = unlimited). Source reports which one won
	// ("account" or "server" or "unlimited"). MessagesPerMinute keeps
	// carrying the effective value for backward compatibility.
	Stored    int    `json:"stored_messages_per_minute,omitempty"`
	Effective int    `json:"effective_messages_per_minute,omitempty"`
	Source    string `json:"source,omitempty"`
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
	source := "unlimited"
	stored := 0
	if block, ok := account.Settings["send_pacing"].(map[string]any); ok {
		if v, ok := block["messages_per_minute"].(float64); ok && int(v) > 0 {
			stored = int(v)
		}
	}
	if stored > 0 {
		effective = stored
		source = "account"
	} else if a.Config != nil && a.Config.Campaigns.DefaultPacingPerMinute > 0 {
		effective = a.Config.Campaigns.DefaultPacingPerMinute
		source = "server"
	}
	return r.SendEnvelope(sendPacingSettings{
		MessagesPerMinute: effective,
		Stored:            stored,
		Effective:         effective,
		Source:            source,
	})
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
