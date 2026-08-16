package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/internal/websocket"
	"github.com/google/uuid"
)

// Device-health alerting: a WhatsApp business number that silently goes
// offline costs the org every message until someone notices. Connection
// state changes already broadcast over WebSocket (gowa_connection) and land
// in the status column; this layer adds the durable surfaces — an audit-log
// entry, a device_alert WebSocket push, and an optional Telegram message —
// with a cooldown so reconnect storms don't spam.

const (
	// deviceAlertCooldown suppresses repeat alerts for the same account
	// while it flaps. A logout that persists past the cooldown re-alerts.
	deviceAlertCooldown = 15 * time.Minute
	// telegramTimeout bounds the push so a slow Telegram API can't hold
	// webhook processing hostage (the call runs on the background wg).
	telegramTimeout = 5 * time.Second
)

// notifyDeviceStatusChange fans out a device state change to the alerting
// surfaces. It is called after the DB status update in both the webhook path
// (processGowaConnection) and the API-triggered path
// (updateGowaDeviceAccountStatus), and is safe to call for every event:
// alerts fire only on logout/disconnect (rate-limited by cooldown) and the
// matching recovery notice fires only if an alert was previously sent.
func (a *App) notifyDeviceStatusChange(account *models.WhatsAppAccount, deviceID, newStatus, reason string) {
	if account == nil {
		return
	}
	ctx := context.Background()
	alertedKey := fmt.Sprintf("device_alerted:%s", account.ID)

	switch newStatus {
	case "disconnected":
		if !a.claimDeviceAlertSlot(ctx, account.ID) {
			return // cooldown active — already alerted recently
		}
		a.sendDeviceAlert(ctx, account, deviceID, true, reason)
		if a.Redis != nil {
			// Remember that we alerted so recovery can clear it. Errors are
			// non-fatal: worst case a recovery notice is skipped.
			_ = a.Redis.Set(ctx, alertedKey, "1", 7*24*time.Hour).Err()
		}
	case "active":
		// Recovery notice — only when an outage alert was actually delivered.
		if a.Redis == nil {
			return
		}
		existed, err := a.Redis.Del(ctx, alertedKey).Result()
		if err != nil || existed == 0 {
			return
		}
		a.sendDeviceAlert(ctx, account, deviceID, false, "")
	}
}

// claimDeviceAlertSlot atomically claims the cooldown slot for an account's
// outage alert. Without Redis (or on Redis errors) it fails open — a spammed
// alert beats a silent outage.
func (a *App) claimDeviceAlertSlot(ctx context.Context, accountID uuid.UUID) bool {
	if a.Redis == nil {
		return true
	}
	ok, err := a.Redis.SetNX(ctx, fmt.Sprintf("device_alert_cd:%s", accountID), "1", deviceAlertCooldown).Result()
	if err != nil {
		a.Log.Warn("Device alert cooldown check failed; alerting anyway", "error", err, "account_id", accountID)
		return true
	}
	return ok
}

// sendDeviceAlert delivers one alert through every configured surface.
func (a *App) sendDeviceAlert(ctx context.Context, account *models.WhatsAppAccount, deviceID string, outage bool, reason string) {
	status := "active"
	if outage {
		status = "disconnected"
	}
	var text string
	if outage {
		text = fmt.Sprintf("⛔ WhatsApp account “%s” is DISCONNECTED (device %s). Re-scan the QR code from Settings → Accounts to bring it back online.", account.Name, deviceID)
		if reason != "" {
			text += " Reason: " + reason
		}
	} else {
		text = fmt.Sprintf("✅ WhatsApp account “%s” is back ONLINE (device %s).", account.Name, deviceID)
	}

	// Durable record in the org's audit trail (resource: devices).
	a.logAudit(account.OrganizationID, uuid.Nil,
		"devices", account.ID, models.AuditActionUpdated, nil,
		map[string]any{
			"device_status": map[string]any{
				"outage": outage,
				"status": status,
				"reason": reason,
			},
		})

	// Live push to connected frontends.
	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(account.OrganizationID, websocket.WSMessage{
			Type: "device_alert",
			Payload: map[string]any{
				"account_id":   account.ID.String(),
				"account_name": account.Name,
				"device_id":    deviceID,
				"outage":       outage,
				"reason":       reason,
			},
		})
	}

	// Optional Telegram push.
	cfg := a.Config
	if cfg == nil || cfg.Alerts.TelegramBotToken == "" || cfg.Alerts.TelegramChatID == "" {
		return
	}
	a.wg.Add(1)
	go func(text, accountName string) {
		defer a.wg.Done()
		tctx, cancel := context.WithTimeout(ctx, telegramTimeout)
		defer cancel()
		body, _ := json.Marshal(map[string]string{
			"chat_id": cfg.Alerts.TelegramChatID,
			"text":    text,
		})
		req, err := http.NewRequestWithContext(tctx, http.MethodPost,
			"https://api.telegram.org/bot"+cfg.Alerts.TelegramBotToken+"/sendMessage",
			bytes.NewReader(body))
		if err != nil {
			a.Log.Error("Device alert: failed to build Telegram request", "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := NewSharedHTTPClient().Do(req)
		if err != nil {
			a.Log.Error("Device alert: Telegram push failed", "error", err, "account", accountName)
			return
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			a.Log.Warn("Device alert: Telegram returned non-OK", "status", resp.StatusCode, "account", accountName)
		}
	}(text, account.Name)
}
