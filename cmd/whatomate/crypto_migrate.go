package main

import (
	"flag"
	"strings"

	"github.com/compnew2006/whatomate/internal/config"
	appcrypto "github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/database"
	"github.com/zerodha/logf"
)

// CRYPTO MIGRATION COMMAND
// ============================================================================

func runCryptoMigrate(args []string) {
	migrateFlags := flag.NewFlagSet("crypto-migrate", flag.ExitOnError)
	configPath := migrateFlags.String("config", "config.toml", "Path to config file")
	dryRun := migrateFlags.Bool("dry-run", false, "Scan only; do not update rows")
	batchSize := migrateFlags.Int("batch-size", 500, "Number of rows per batch")
	includeEnc2 := migrateFlags.Bool("include-enc2", true, "Upgrade enc2 payloads in addition to enc")
	_ = migrateFlags.Parse(args)

	lo := logf.New(logf.Opts{
		EnableColor:     true,
		Level:           logf.InfoLevel,
		EnableCaller:    true,
		TimestampFormat: "2006-01-02 15:04:05",
		DefaultFields:   []any{"app", "whatomate-crypto-migrate"},
	})

	lo.Info("Starting crypto migration", "dry_run", *dryRun, "batch_size", *batchSize, "include_enc2", *includeEnc2)

	cfg, err := config.Load(*configPath)
	if err != nil {
		lo.Fatal("Failed to load config", "error", err)
	}
	if err := config.ValidateEncryptionKey(cfg); err != nil {
		lo.Fatal("Invalid encryption configuration", "error", err)
	}
	if strings.TrimSpace(cfg.App.EncryptionKey) == "" {
		lo.Fatal("Encryption key is required for migration")
	}

	db, err := database.NewPostgres(&cfg.Database, cfg.App.Debug)
	if err != nil {
		lo.Fatal("Failed to connect to database", "error", err)
	}
	lo.Info("Connected to PostgreSQL")

	opts := appcrypto.MigrationOptions{
		DryRun:      *dryRun,
		BatchSize:   *batchSize,
		IncludeEnc2: *includeEnc2,
	}

	summaries, err := appcrypto.MigrateEncryptedColumns(db, cfg.App.EncryptionKey, opts, lo)
	if err != nil {
		lo.Fatal("Crypto migration failed", "error", err)
	}

	totalUpdated := 0
	totalFailed := 0
	for _, summary := range summaries {
		totalUpdated += summary.Updated
		totalFailed += summary.Failed
	}

	lo.Info("Crypto migration completed", "updated", totalUpdated, "failed", totalFailed, "dry_run", *dryRun)
}
