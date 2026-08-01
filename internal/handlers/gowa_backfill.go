package handlers

import (
	"github.com/shridarpatil/gowa-ui/internal/config"
	"github.com/shridarpatil/gowa-ui/internal/crypto"
	"github.com/shridarpatil/gowa-ui/internal/models"
	"github.com/shridarpatil/gowa-ui/pkg/gowa"
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
	}

	log.Info("GOWA webhook secret backfill complete", "backfilled", len(accounts))
	return nil
}
