package handlers

import (
	"context"
	"strings"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/google/uuid"
)

// RepairGowaDeviceIDs is a one-shot startup self-heal. GOWA registers devices
// under their CURRENT display name, and renaming a device in WhatsApp/linking
// changes that id — while whatsapp_accounts.gowa_device_id keeps the stale
// value forever (no DB integrity links the two). Every gowa-ui call for the
// renamed device (send, receipts, media recovery) then fails with
// DEVICE_NOT_FOUND. This pass lists the engine's devices once per (org,
// server) pair and re-links each account whose stored device id no longer
// exists but whose gowa_jid still matches a live engine device.
//
// It never mutates a matching id, never invents one (accounts whose JID
// matches nothing are only logged), and every failure is non-fatal so a
// temporarily unreachable GOWA server cannot block startup.
func (a *App) RepairGowaDeviceIDs(ctx context.Context) {
	var accounts []models.WhatsAppAccount
	if err := a.DB.Where("gowa_base_url <> '' OR gowa_device_id <> ''").Find(&accounts).Error; err != nil {
		a.Log.Error("GOWA device-id repair: failed to load accounts", "error", err)
		return
	}
	if len(accounts) == 0 {
		return
	}

	// Engine device lists are per server, but Basic Auth credentials are
	// org-scoped — group by (org, dial base URL) and cache the successful
	// list per dial URL so orgs sharing a server don't re-fetch it.
	type groupKey struct {
		orgID uuid.UUID
		base  string
	}
	devicesByBase := make(map[string][]gowa.DeviceInfo)
	repaired, unmatched := 0, 0
	for _, acct := range accounts {
		base := GowaDialBaseURL(a.Config, acct.GowaBaseURL)
		devices, ok := devicesByBase[base]
		if !ok {
			client := a.gowaClientForAccount(&acct)
			if client == nil {
				continue
			}
			devices, err := client.ListDevices(ctx)
			if err != nil {
				a.Log.Warn("GOWA device-id repair: device list failed",
					"base_url", base, "account", acct.Name, "error", err)
				devicesByBase[base] = nil
				continue
			}
			devicesByBase[base] = devices
		}
		if devices == nil {
			continue
		}

		newID := planGowaDeviceRepair(devices, acct)
		if newID == acct.GowaDeviceID {
			continue
		}
		if newID == "" {
			unmatched++
			a.Log.Warn("GOWA device-id repair: stored device id not found and no JID match",
				"account", acct.Name, "stored_device_id", acct.GowaDeviceID, "stored_jid", acct.GowaJID)
			continue
		}
		if err := a.DB.Model(&models.WhatsAppAccount{}).Where("id = ?", acct.ID).
			Update("gowa_device_id", newID).Error; err != nil {
			a.Log.Error("GOWA device-id repair: update failed", "account", acct.Name, "error", err)
			continue
		}
		repaired++
		a.Log.Info("GOWA device-id repair: re-linked account to renamed device",
			"account", acct.Name, "old_device_id", acct.GowaDeviceID, "new_device_id", newID, "jid", acct.GowaJID)
	}
	if repaired > 0 || unmatched > 0 {
		a.Log.Info("GOWA device-id repair pass complete", "repaired", repaired, "unmatched", unmatched)
	}
}

// planGowaDeviceRepair returns the engine device id the account should be
// re-linked to: the stored id when it still exists (no-op), the id of the
// live device sharing the account's JID when the stored id is gone, or ""
// when nothing matches (caller logs and leaves the row alone).
func planGowaDeviceRepair(devices []gowa.DeviceInfo, acct models.WhatsAppAccount) string {
	jid := strings.TrimSpace(acct.GowaJID)
	var byJID string
	for _, d := range devices {
		if d.ID == acct.GowaDeviceID {
			return acct.GowaDeviceID
		}
		if jid != "" && strings.TrimSpace(d.JID) == jid {
			byJID = d.ID
		}
	}
	return byJID
}
