package main

import (
	"context"
	"flag"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/database"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/zerodha/logf"
)

// QUEUE CAMPAIGN MIGRATION COMMAND
// ============================================================================

func runQueueMigrateCampaigns(args []string) {
	migrateFlags := flag.NewFlagSet("queue-migrate-campaigns", flag.ExitOnError)
	configPath := migrateFlags.String("config", "config.toml", "Path to config file")
	apply := migrateFlags.Bool("apply", false, "Apply the migration; default is dry-run")
	batchSize := migrateFlags.Int64("batch-size", 100, "Number of Redis stream entries per batch")
	lockTTL := migrateFlags.Duration("lock-ttl", 5*time.Minute, "TTL for the migration lock")
	_ = migrateFlags.Parse(args)

	lo := logf.New(logf.Opts{
		EnableColor:     true,
		Level:           logf.InfoLevel,
		EnableCaller:    true,
		TimestampFormat: "2006-01-02 15:04:05",
		DefaultFields:   []any{"app", "whatomate-queue-migrate-campaigns"},
	})

	cfg, err := config.Load(*configPath)
	if err != nil {
		lo.Fatal("Failed to load config", "error", err)
	}

	rdb, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		lo.Fatal("Failed to connect to Redis", "error", err)
	}

	summary, err := queue.MigrateLegacyCampaignStream(context.Background(), rdb, lo, queue.CampaignMigrationOptions{
		Apply:     *apply,
		BatchSize: *batchSize,
		LockTTL:   *lockTTL,
	})
	if err != nil {
		lo.Fatal("Campaign queue migration failed", "error", err)
	}

	lo.Info("Campaign queue migration completed",
		"dry_run", summary.DryRun,
		"legacy_stream_exists", summary.LegacyStreamExists,
		"legacy_group_found", summary.ConsumerGroupFound,
		"temporary_group_used", summary.TemporaryGroupUsed,
		"unread", summary.Unread,
		"pending", summary.Pending,
		"migrated", summary.Migrated,
		"invalid", summary.Invalid,
		"skipped", summary.Skipped,
		"invalid_samples", summary.InvalidMessageIDs,
		"migrated_samples", summary.MigratedMessageIDs,
	)
}
