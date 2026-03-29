package crypto

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

type MigrationOptions struct {
	DryRun      bool
	BatchSize   int
	IncludeEnc2 bool
}

type MigrationSummary struct {
	Table   string
	Column  string
	Scanned int
	Updated int
	Skipped int
	Failed  int
}

type migrationTarget struct {
	Table  string
	Column string
}

var migrationTargets = []migrationTarget{
	{Table: "whatsapp_accounts", Column: "access_token"},
	{Table: "whatsapp_accounts", Column: "app_secret"},
	{Table: "chatbot_settings", Column: "ai_api_key"},
	{Table: "sso_providers", Column: "client_secret"},
}

func MigrateEncryptedColumns(db *gorm.DB, key string, opts MigrationOptions, logger logf.Logger) ([]MigrationSummary, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if strings.TrimSpace(key) == "" {
		return nil, ErrMissingEncryptionKey
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 500
	}

	summaries := make([]MigrationSummary, 0, len(migrationTargets))
	hadFailure := false

	for _, target := range migrationTargets {
		summary := MigrationSummary{Table: target.Table, Column: target.Column}
		likeClause := fmt.Sprintf("%s LIKE ?", target.Column)

		offset := 0
		for {
			query := db.Table(target.Table).
				Select("id", target.Column).
				Where(likeClause, legacyPrefix+"%")
			if opts.IncludeEnc2 {
				query = query.Or(likeClause, prefixV2+"%")
			}

			rows, err := query.Limit(opts.BatchSize).Offset(offset).Rows()
			if err != nil {
				return nil, err
			}

			rowCount := 0
			for rows.Next() {
				rowCount++
				summary.Scanned++

				var id string
				var value sql.NullString
				if err := rows.Scan(&id, &value); err != nil {
					summary.Failed++
					hadFailure = true
					logger.Error("Failed to scan encrypted row", "table", target.Table, "column", target.Column, "error", err)
					continue
				}
				if !value.Valid || strings.TrimSpace(value.String) == "" {
					summary.Skipped++
					continue
				}

				updatedValue, changed, err := UpgradeCiphertext(value.String, key, opts.IncludeEnc2)
				if err != nil {
					summary.Failed++
					hadFailure = true
					logger.Error("Failed to upgrade ciphertext", "table", target.Table, "column", target.Column, "id", id, "error", err)
					continue
				}
				if !changed {
					summary.Skipped++
					continue
				}

				summary.Updated++
				if !opts.DryRun {
					if err := db.Table(target.Table).
						Where("id = ?", id).
						UpdateColumn(target.Column, updatedValue).Error; err != nil {
						summary.Failed++
						hadFailure = true
						logger.Error("Failed to persist upgraded ciphertext", "table", target.Table, "column", target.Column, "id", id, "error", err)
					}
				}
			}
			_ = rows.Close()

			if rowCount == 0 {
				break
			}
			offset += opts.BatchSize
		}

		summaries = append(summaries, summary)
		logger.Info("Crypto migration summary", "table", summary.Table, "column", summary.Column, "scanned", summary.Scanned, "updated", summary.Updated, "skipped", summary.Skipped, "failed", summary.Failed, "dry_run", opts.DryRun)
	}

	if hadFailure {
		return summaries, fmt.Errorf("crypto migration completed with failures")
	}
	return summaries, nil
}

func UpgradeCiphertext(ciphertext, key string, includeEnc2 bool) (string, bool, error) {
	trimmed := strings.TrimSpace(ciphertext)
	if trimmed == "" {
		return ciphertext, false, nil
	}
	if strings.HasPrefix(trimmed, prefixV3) {
		return ciphertext, false, nil
	}
	if strings.HasPrefix(trimmed, legacyPrefix) || (includeEnc2 && strings.HasPrefix(trimmed, prefixV2)) {
		plaintext, err := DecryptWithPolicy(ciphertext, key, true)
		if err != nil {
			return "", false, err
		}
		upgraded, err := Encrypt(plaintext, key)
		if err != nil {
			return "", false, err
		}
		return upgraded, true, nil
	}
	return ciphertext, false, nil
}
