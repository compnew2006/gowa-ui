package main

import (
	"context"
	"fmt"
	"os"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/license"
	objectstorage "github.com/compnew2006/whatomate/internal/storage"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		runServer(os.Args[2:])
	case "worker":
		runWorker(os.Args[2:])
	case "crypto-migrate":
		runCryptoMigrate(os.Args[2:])
	case "queue-migrate-campaigns":
		runQueueMigrateCampaigns(os.Args[2:])
	case "inbound-media-reconcile":
		runInboundMediaReconcile(os.Args[2:])
	case "legacy-media-reconcile":
		runLegacyMediaReconcile(os.Args[2:])
	case "version":
		fmt.Printf("Whatomate %s (built %s)\n", Version, BuildTime)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Whatomate - WhatsApp Business API Platform

Usage:
  whatomate <command> [options]

Commands:
  server    Start the API server (with optional embedded workers)
  worker    Start background workers only (no API server)
  crypto-migrate  Upgrade encrypted secrets from enc:/enc2: to enc3:
  queue-migrate-campaigns  Redistribute legacy global campaign jobs into tenant streams
  inbound-media-reconcile  Reconcile stale queued inbound-media rows
  legacy-media-reconcile   Mark missing legacy local-media rows as unavailable
  version   Show version information
  help      Show this help message

Server Options:
  -config string    Path to config file (default "config.toml")
  -migrate          Run database migrations on startup
  -workers int      Global campaign worker budget for embedded scaling (0 to disable workers) (default 1)

Worker Options:
  -config string    Path to config file (default "config.toml")
  -workers int      Global campaign worker budget for scaling (default 1)

Crypto Migration Options:
  -config string       Path to config file (default "config.toml")
  -dry-run             Scan only; do not update rows
  -batch-size int      Number of rows per batch (default 500)
  -include-enc2        Upgrade enc2 payloads in addition to enc (default true)

Queue Campaign Migration Options:
  -config string       Path to config file (default "config.toml")
  -apply               Apply the migration; default is dry-run
  -batch-size int      Number of Redis stream entries per batch (default 100)
  -lock-ttl duration   TTL for the migration lock (default 5m)

Inbound Media Reconcile Options:
  -config string            Path to config file (default "config.toml")
  -instance-id string       Limit reconciliation to a single WhatsApp instance UUID
  -older-than duration      Only reconcile queued rows older than this age (default 15m)
  -limit int                Limit number of rows to reconcile (default 0 = all eligible)
  -apply                    Apply updates; default is dry-run
  -allow-active-queue       Bypass queue-idle safety checks

Legacy Media Reconcile Options:
  -config string            Path to config file (default "config.toml")
  -older-than duration      Only reconcile rows older than this age (default 1h)
  -limit int                Limit number of candidate rows scanned (default 0 = all eligible)
  -apply                    Apply updates; default is dry-run

Examples:
  whatomate server                     # API + dynamic embedded workers
  whatomate server -workers 0          # API only (no workers)
  whatomate server -workers 4          # API + campaign worker budget 4
  whatomate server -migrate            # Run migrations and start server
  whatomate worker -workers 4          # campaign worker budget 4 (no API)
  whatomate crypto-migrate -dry-run    # Scan for legacy encrypted secrets
  whatomate queue-migrate-campaigns -config config.toml -apply
  whatomate inbound-media-reconcile -config config.toml -apply
  whatomate legacy-media-reconcile -config config.toml -apply

Deployment Scenarios:
  All-in-one:    whatomate server
  Separate:      whatomate server -workers 0  (on API server)
                 whatomate worker -workers 4  (on worker server)`)
}

func initLogger(appName string, level logf.Level) logf.Logger {
	return logf.New(logf.Opts{
		EnableColor:     true,
		Level:           level,
		EnableCaller:    true,
		TimestampFormat: "2006-01-02 15:04:05",
		DefaultFields:   []any{"app", appName},
	})
}

func loadAndValidateConfig(configPath string, lo logf.Logger) *config.Config {
	cfg, err := config.Load(configPath)
	if err != nil {
		lo.Fatal("Failed to load config", "error", err)
	}
	if err := config.ValidateJWTSecret(cfg); err != nil {
		lo.Fatal("Invalid JWT configuration", "error", err)
	}
	if err := config.ValidateEncryptionKey(cfg); err != nil {
		lo.Fatal("Invalid encryption configuration", "error", err)
	}
	if err := config.ValidateDefaultAdmin(cfg); err != nil {
		lo.Fatal("Invalid default admin configuration", "error", err)
	}
	if err := config.ValidateLicenseConfig(cfg); err != nil {
		lo.Fatal("Invalid license configuration", "error", err)
	}
	if err := config.ValidateDatabaseCredentials(cfg); err != nil {
		lo.Fatal("Invalid database configuration", "error", err)
	}
	if err := config.ValidateWebhookVerifyToken(cfg); err != nil {
		lo.Fatal("Invalid webhook configuration", "error", err)
	}
	return cfg
}

func adjustLogLevelForProduction(cfg *config.Config, lo logf.Logger, appName string) logf.Logger {
	if cfg.App.Environment == "production" {
		return logf.New(logf.Opts{
			Level:           logf.InfoLevel,
			TimestampFormat: "2006-01-02 15:04:05",
			DefaultFields:   []any{"app", appName},
		})
	}
	return lo
}

func initObjectStorage(cfg *config.Config, lo logf.Logger) objectstorage.ObjectStorage {
	storedObjects, err := objectstorage.NewObjectStorage(&cfg.Storage)
	if err != nil {
		lo.Fatal("Failed to initialize object storage", "error", err)
	}
	return storedObjects
}

func initLicenseService(cfg *config.Config, db *gorm.DB, rdb *redis.Client, lo logf.Logger) (*license.Service, context.CancelFunc) {
	licenseService, err := license.NewService(cfg, db, rdb, lo)
	if err != nil {
		lo.Fatal("Failed to initialize license service", "error", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	licenseService.Start(ctx)
	return licenseService, cancel
}

func capWorkerCount(workerCount int, licenseService *license.Service, lo logf.Logger) int {
	state := licenseService.CurrentState()
	if state.LicenseID != "" && state.MaxWorkers > 0 && workerCount > state.MaxWorkers {
		lo.Warn("Requested workers exceed licensed maximum; capping worker count",
			"requested", workerCount,
			"licensed_max", state.MaxWorkers)
		return state.MaxWorkers
	}
	return workerCount
}
