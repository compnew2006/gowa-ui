package main

import (
	"context"
	"flag"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/database"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/google/uuid"
	"github.com/zerodha/logf"
)

// INBOUND MEDIA RECONCILE COMMAND
// ============================================================================

func runInboundMediaReconcile(args []string) {
	reconcileFlags := flag.NewFlagSet("inbound-media-reconcile", flag.ExitOnError)
	configPath := reconcileFlags.String("config", "config.toml", "Path to config file")
	instanceIDText := reconcileFlags.String("instance-id", "", "Limit reconciliation to a single WhatsApp instance UUID")
	olderThan := reconcileFlags.Duration("older-than", 15*time.Minute, "Only reconcile queued rows older than this age")
	limit := reconcileFlags.Int("limit", 0, "Limit number of rows to reconcile (0 = all eligible)")
	apply := reconcileFlags.Bool("apply", false, "Apply updates; default is dry-run")
	allowActiveQueue := reconcileFlags.Bool("allow-active-queue", false, "Bypass queue-idle safety checks")
	_ = reconcileFlags.Parse(args)

	lo := logf.New(logf.Opts{
		EnableColor:     true,
		Level:           logf.InfoLevel,
		EnableCaller:    true,
		TimestampFormat: "2006-01-02 15:04:05",
		DefaultFields:   []any{"app", "whatomate-inbound-media-reconcile"},
	})

	cfg, err := config.Load(*configPath)
	if err != nil {
		lo.Fatal("Failed to load config", "error", err)
	}
	if err := config.ValidateDatabaseCredentials(cfg); err != nil {
		lo.Fatal("Invalid database configuration", "error", err)
	}
	if cfg.WhatsApp.Provider != "whatsmeow" {
		lo.Fatal("Inbound media reconciliation is only supported for whatsmeow provider", "provider", cfg.WhatsApp.Provider)
	}

	db, err := database.NewPostgres(&cfg.Database, cfg.App.Debug)
	if err != nil {
		lo.Fatal("Failed to connect to database", "error", err)
	}
	lo.Info("Connected to PostgreSQL")

	rdb, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		lo.Fatal("Failed to connect to Redis", "error", err)
	}
	lo.Info("Connected to Redis")

	var instanceID *uuid.UUID
	if trimmed := strings.TrimSpace(*instanceIDText); trimmed != "" {
		parsedID, parseErr := uuid.Parse(trimmed)
		if parseErr != nil {
			lo.Fatal("Invalid instance-id", "error", parseErr, "value", trimmed)
		}
		instanceID = &parsedID
	}

	summary, err := whatsmeow.ReconcileStaleQueuedInboundMedia(
		context.Background(),
		db,
		rdb,
		whatsmeow.InboundMediaReconcileOptions{
			InstanceID:       instanceID,
			OlderThan:        *olderThan,
			Limit:            *limit,
			Apply:            *apply,
			AllowActiveQueue: *allowActiveQueue,
		},
		lo,
	)
	if err != nil {
		lo.Fatal("Inbound media reconciliation failed", "error", err)
	}

	lo.Info(
		"Inbound media reconciliation completed",
		"dry_run", summary.DryRun,
		"cutoff", summary.Cutoff.Format(time.RFC3339),
		"queue_pending", summary.QueuePending,
		"queue_lag", summary.QueueLag,
		"active_pending_ids", summary.ActivePendingIDs,
		"skipped_active_queued", summary.SkippedActiveQueued,
		"total_queued", summary.TotalQueued,
		"eligible_queued", summary.EligibleQueued,
		"requeued", summary.Requeued,
		"marked_failed", summary.MarkedFailed,
		"updated", summary.Updated,
		"sample_ids", strings.Join(summary.SampleIDs, ","),
	)
}

// ============================================================================
// LEGACY MEDIA RECONCILE COMMAND
// ============================================================================

func runLegacyMediaReconcile(args []string) {
	reconcileFlags := flag.NewFlagSet("legacy-media-reconcile", flag.ExitOnError)
	configPath := reconcileFlags.String("config", "config.toml", "Path to config file")
	olderThan := reconcileFlags.Duration("older-than", time.Hour, "Only reconcile rows older than this age")
	limit := reconcileFlags.Int("limit", 0, "Limit number of candidate rows scanned (0 = all eligible)")
	apply := reconcileFlags.Bool("apply", false, "Apply updates; default is dry-run")
	_ = reconcileFlags.Parse(args)

	lo := logf.New(logf.Opts{
		EnableColor:     true,
		Level:           logf.InfoLevel,
		EnableCaller:    true,
		TimestampFormat: "2006-01-02 15:04:05",
		DefaultFields:   []any{"app", "whatomate-legacy-media-reconcile"},
	})

	cfg, err := config.Load(*configPath)
	if err != nil {
		lo.Fatal("Failed to load config", "error", err)
	}
	if err := config.ValidateDatabaseCredentials(cfg); err != nil {
		lo.Fatal("Invalid database configuration", "error", err)
	}
	if strings.ToLower(strings.TrimSpace(cfg.Storage.Type)) != "" && strings.ToLower(strings.TrimSpace(cfg.Storage.Type)) != "local" {
		lo.Fatal("Legacy media reconciliation is only supported for local storage", "storage_type", cfg.Storage.Type)
	}

	db, err := database.NewPostgres(&cfg.Database, cfg.App.Debug)
	if err != nil {
		lo.Fatal("Failed to connect to database", "error", err)
	}
	lo.Info("Connected to PostgreSQL")

	summary, err := handlers.ReconcileMissingLegacyMedia(
		context.Background(),
		db,
		cfg.Storage.LocalPath,
		handlers.LegacyMediaReconcileOptions{
			OlderThan: *olderThan,
			Limit:     *limit,
			Apply:     *apply,
		},
	)
	if err != nil {
		lo.Fatal("Legacy media reconciliation failed", "error", err)
	}

	lo.Info(
		"Legacy media reconciliation completed",
		"dry_run", summary.DryRun,
		"cutoff", summary.Cutoff.Format(time.RFC3339),
		"candidates", summary.CandidateCount,
		"missing", summary.MissingCount,
		"updated", summary.UpdatedCount,
		"sample_ids", strings.Join(summary.SampleIDs, ","),
	)
}
