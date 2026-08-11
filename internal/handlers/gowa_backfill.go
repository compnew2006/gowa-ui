package handlers

import (
	"context"
	"time"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/crypto"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// BackfillGowaWebhookSecrets generates and stores a webhook secret for every
// GOWA-type WhatsApp account that doesn't have one. This ensures no GOWA
// account is left webhook-unprotected (FR-017). The function is idempotent —
// accounts that already have a secret are skipped.
//
// This runs at startup after migrations, before the server accepts traffic.
func BackfillGowaWebhookSecrets(db *gorm.DB, cfg *config.Config, log logf.Logger) error {
	var accounts []models.WhatsAppAccount
	if err := db.Where("provider_type = ? AND (gowa_webhook_secret = ? OR gowa_webhook_secret IS NULL)", "gowa", "").Find(&accounts).Error; err != nil {
		return err
	}

	if len(accounts) == 0 {
		return nil
	}

	log.Info("Backfilling GOWA webhook secrets", "count", len(accounts))

	for i := range accounts {
		secret := gowa.GenerateWebhookSecret()
		enc, err := crypto.Encrypt(secret, cfg.App.EncryptionKey)
		if err != nil {
			log.Error("Failed to encrypt backfilled webhook secret", "account_id", accounts[i].ID, "error", err)
			continue
		}
		if err := db.Model(&models.WhatsAppAccount{}).Where("id = ?", accounts[i].ID).Update("gowa_webhook_secret", enc).Error; err != nil {
			log.Error("Failed to update backfilled webhook secret", "account_id", accounts[i].ID, "error", err)
			continue
		}
		// Best-effort: push the new secret to GOWA so it signs with the same
		// secret we verify with. Without this, every webhook for this account
		// 403s (gap #6). Per-account so one bad instance can't abort the batch.
		pushBackfilledSecretToGOWA(db, cfg, log, &accounts[i], secret)
	}

	log.Info("GOWA webhook secret backfill complete", "backfilled", len(accounts))
	return nil
}

// syncDeviceWebhookSecret pushes plaintextSecret to a GOWA device via client,
// preserving the device's existing webhook URL + event whitelist (PATCH needs
// the full WebhookConfig). Best-effort: failures are logged by `log` but never
// returned — the DB save that precedes every caller must not depend on GOWA
// being reachable (same principle as ensureCallOfferSubscription).
//
// This closes the manual-provisioning half of gap #6: without it, a
// backend-generated or backend-rotated secret diverges from what GOWA signs
// with, and every webhook for the account becomes a 403. The modern
// GOWA-Servers provisioning path already pushes the secret at device creation;
// this covers manual account CRUD + startup backfill.
func syncDeviceWebhookSecret(ctx context.Context, client *gowa.Client, log logf.Logger, accountName, deviceID, plaintextSecret string) {
	if deviceID == "" || plaintextSecret == "" {
		return
	}
	cfg, err := client.GetDeviceWebhook(ctx, deviceID)
	if err != nil {
		log.Warn("Could not read GOWA device webhook to sync secret (best-effort)",
			"error", err, "account", accountName, "device_id", deviceID)
		return
	}
	cfg.WebhookSecret = plaintextSecret
	if _, err := client.SetDeviceWebhook(ctx, deviceID, *cfg); err != nil {
		log.Warn("Failed to push webhook secret to GOWA (best-effort); webhooks may 403 until synced",
			"error", err, "account", accountName, "device_id", deviceID)
		return
	}
	log.Info("Synced webhook secret to GOWA", "account", accountName, "device_id", deviceID)
}

// pushBackfilledSecretToGOWA constructs a one-off GOWA client for an account
// (resolved with the SAME org-scoped credentials the runtime uses) and pushes a
// freshly-backfilled plaintext secret. Used by the startup backfill, which has
// no *App / registry — per-account so one unreachable GOWA instance can't abort
// the whole backfill.
func pushBackfilledSecretToGOWA(db *gorm.DB, cfg *config.Config, log logf.Logger, account *models.WhatsAppAccount, plaintextSecret string) {
	if account.GowaDeviceID == "" || account.GowaBaseURL == "" || plaintextSecret == "" {
		return
	}
	user, pass := ResolveGowaCreds(db, cfg, account.OrganizationID, account.GowaBaseURL)
	client := gowa.New(account.GowaBaseURL, user, pass)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	syncDeviceWebhookSecret(ctx, client, log, account.Name, account.GowaDeviceID, plaintextSecret)
}
