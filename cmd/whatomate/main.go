package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	appcrypto "github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/database"
	"github.com/compnew2006/whatomate/internal/frontend"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/observability"
	"github.com/compnew2006/whatomate/internal/queue"
	objectstorage "github.com/compnew2006/whatomate/internal/storage"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/compnew2006/whatomate/internal/worker"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
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

// ============================================================================
// SERVER COMMAND
// ============================================================================

func runServer(args []string) {
	serverFlags := flag.NewFlagSet("server", flag.ExitOnError)
	configPath := serverFlags.String("config", "config.toml", "Path to config file")
	migrate := serverFlags.Bool("migrate", false, "Run database migrations")
	numWorkers := serverFlags.Int("workers", 1, "Number of workers to run (0 to disable embedded workers)")
	_ = serverFlags.Parse(args)

	// Initialize logger
	lo := logf.New(logf.Opts{
		EnableColor:     true,
		Level:           logf.DebugLevel,
		EnableCaller:    true,
		TimestampFormat: "2006-01-02 15:04:05",
		DefaultFields:   []any{"app", "whatomate"},
	})

	lo.Info("Starting Whatomate server...", "version", Version)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		lo.Fatal("Failed to load config", "error", err)
	}

	// Validate JWT secret and fail fast on insecure/missing settings.
	if err := config.ValidateJWTSecret(cfg); err != nil {
		lo.Fatal("Invalid JWT configuration", "error", err)
	}
	if err := config.ValidateEncryptionKey(cfg); err != nil {
		lo.Fatal("Invalid encryption configuration", "error", err)
	}
	if err := config.ValidateDefaultAdmin(cfg); err != nil {
		lo.Fatal("Invalid default admin configuration", "error", err)
	}
	if err := config.ValidateDatabaseCredentials(cfg); err != nil {
		lo.Fatal("Invalid database configuration", "error", err)
	}
	if err := config.ValidateWebhookVerifyToken(cfg); err != nil {
		lo.Fatal("Invalid WhatsApp configuration", "error", err)
	}
	if err := config.ValidateLicenseConfig(cfg); err != nil {
		lo.Fatal("Invalid license configuration", "error", err)
	}
	if err := config.ValidateDatabaseCredentials(cfg); err != nil {
		lo.Fatal("Invalid database configuration", "error", err)
	}
	if err := config.ValidateWebhookVerifyToken(cfg); err != nil {
		lo.Fatal("Invalid WhatsApp configuration", "error", err)
	}

	// Warn if debug mode is on in production
	if cfg.App.Environment == "production" && cfg.App.Debug {
		lo.Warn("Debug mode is enabled in production! This may expose sensitive information.")
	}

	// Set log level based on environment
	if cfg.App.Environment == "production" {
		lo = logf.New(logf.Opts{
			Level:           logf.InfoLevel,
			TimestampFormat: "2006-01-02 15:04:05",
			DefaultFields:   []any{"app", "whatomate"},
		})
	}

	sandboxMode := cfg.App.SandboxMode
	if sandboxMode {
		if *migrate {
			lo.Fatal("Sandbox mode forbids -migrate to avoid shared-environment schema changes")
		}
		if *numWorkers != 0 {
			lo.Warn("Sandbox mode forces embedded workers off", "requested", *numWorkers)
			*numWorkers = 0
		}
		lo.Warn("Sandbox mode enabled: startup upgrades, reconnect automation, recurring background jobs, and embedded workers are disabled")
	}

	// Connect to PostgreSQL
	db, err := database.NewPostgres(&cfg.Database, cfg.App.Debug)
	if err != nil {
		lo.Fatal("Failed to connect to database", "error", err)
	}
	lo.Info("Connected to PostgreSQL")

	// Initialize whatsmeow sqlstore
	sqlDB, err := db.DB()
	if err != nil {
		lo.Fatal("Failed to get underlying SQL DB for whatsmeow", "error", err)
	}
	storeContainer := sqlstore.NewWithDB(sqlDB, "postgres", waLog.Stdout("Database", "DEBUG", true))
	if sandboxMode {
		lo.Warn("Sandbox mode: skipping whatsmeow sqlstore upgrade")
	} else if err := storeContainer.Upgrade(context.Background()); err != nil {
		lo.Fatal("Failed to upgrade whatsmeow store", "error", err)
	}
	lo.Info("Whatsmeow sqlstore initialized")

	// Run migrations if requested
	if *migrate {
		if err := database.RunMigrationWithProgress(db, &cfg.DefaultAdmin); err != nil {
			lo.Fatal("Migration failed", "error", err)
		}
	}
	if sandboxMode {
		lo.Info("Sandbox mode: skipping assigned chat reset startup backfill")
	} else if err := database.BackfillInstanceAssignedChatResetSettings(db); err != nil {
		lo.Fatal("Failed to backfill per-instance assigned chat reset settings", "error", err)
	}

	// Connect to Redis
	rdb, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		lo.Fatal("Failed to connect to Redis", "error", err)
	}
	lo.Info("Connected to Redis")

	storedObjects, err := objectstorage.NewObjectStorage(&cfg.Storage)
	if err != nil {
		lo.Fatal("Failed to initialize object storage", "error", err)
	}

	// Initialize job queue
	jobQueue := queue.NewRedisQueue(rdb, lo)
	lo.Info("Job queue initialized")

	// Initialize Fastglue
	g := fastglue.NewGlue()

	// Initialize WhatsApp client
	waClient := whatsapp.NewWithBaseURL(lo, cfg.WhatsApp.BaseURL)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(lo)
	go wsHub.Run()
	lo.Info("WebSocket hub started")

	// Initialize whatsmeow manager
	whatsmeowManager := whatsmeow.NewConnectionManager(db, storeContainer, lo, &cfg.Whatsmeow, wsHub, cfg.Storage.LocalPath)
	whatsmeowManager.SetInboundMediaQueue(jobQueue)
	whatsmeowManager.SetCampaignStatsPublisher(queue.NewPublisher(rdb, lo))
	whatsmeowManager.SetMediaService(whatsmeow.NewMediaService(db, storedObjects, lo, whatsmeowManager.GetClient))

	// Auto-connect linked sessions and reconnect active instances in background.
	if cfg.WhatsApp.Provider == "whatsmeow" {
		if sandboxMode {
			lo.Warn("Sandbox mode: skipping whatsmeow health monitor and auto-reconnect lifecycle")
		} else {
			whatsmeowManager.StartHealthMonitor(context.Background())
			defer whatsmeowManager.StopHealthMonitor()

			// Reconcile stale transient states before serving API traffic.
			reconcileCtx, reconcileCancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := whatsmeowManager.ReconcileStartupStatuses(reconcileCtx); err != nil {
				lo.Warn("Failed to reconcile stale instance statuses on startup", "error", err)
			}
			reconcileCancel()

			go func() {
				// Wait a bit for server to start.
				time.Sleep(2 * time.Second)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()

				if err := whatsmeowManager.AutoConnectLinkedInstancesOnFirstRun(ctx); err != nil {
					lo.Warn("First-run auto-connect completed with issues", "error", err)
				}
				if err := whatsmeowManager.ReconnectAll(ctx); err != nil {
					lo.Error("Failed to reconnect instances", "error", err)
				}
			}()
		}
	}
	lo.Info("Whatsmeow manager initialized")

	// Initialize app with dependencies
	// Shared HTTP client with connection pooling for external API calls
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext:         handlers.SSRFSafeDialer(),
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	app := &handlers.App{
		Config:           cfg,
		DB:               db,
		Redis:            rdb,
		Log:              lo,
		WhatsApp:         waClient,
		WSHub:            wsHub,
		WhatsmeowStore:   storeContainer,
		WhatsmeowManager: whatsmeowManager,
		ObjectStorage:    storedObjects,
		Queue:            jobQueue,
		HTTPClient:       httpClient,
	}

	licenseService, err := license.NewService(cfg, db, rdb, lo)
	if err != nil {
		lo.Fatal("Failed to initialize license service", "error", err)
	}
	app.License = licenseService
	licenseCtx, licenseCancel := context.WithCancel(context.Background())
	licenseService.Start(licenseCtx)

	// Wire MessageProvider based on configured provider
	switch cfg.WhatsApp.Provider {
	case "whatsmeow":
		adapter := whatsmeow.NewWhatsmeowAdapter(whatsmeowManager, db, lo)
		app.MessageProvider = adapter
		app.WhatsmeowQueue = whatsmeow.NewQueueManager(cfg.Whatsmeow, lo)
		app.WhatsmeowQueue.SetDepthObserver(func(instanceID string, depth int64) {
			parsedInstanceID, err := uuid.Parse(strings.TrimSpace(instanceID))
			if err != nil {
				return
			}
			whatsmeowManager.SetQueueDepth(parsedInstanceID, depth)
		})
		lo.Info("MessageProvider set to whatsmeow")
	default: // "meta" or empty
		metaAdapter := whatsapp.NewMetaAdapter(waClient, db, lo)
		app.MessageProvider = metaAdapter
		lo.Info("MessageProvider set to meta")
	}

	// Start campaign stats subscriber for real-time WebSocket updates from worker
	if err := app.StartCampaignStatsSubscriber(); err != nil {
		lo.Error("Failed to start campaign stats subscriber", "error", err)
	}

	// Parse allowed origins for CORS
	allowedOrigins := middleware.ParseAllowedOrigins(cfg.Server.AllowedOrigins)
	observabilityManager := observability.NewManager(cfg.Observability, db, rdb)

	// Setup middleware (CORS is handled by corsWrapper at fasthttp level)
	g.Before(middleware.SecurityHeaders(strings.EqualFold(strings.TrimSpace(cfg.App.Environment), "production")))
	g.Before(middleware.RequestLogger(lo))
	g.Before(middleware.Recovery(lo))
	g.Before(func(r *fastglue.Request) *fastglue.Request {
		if string(r.RequestCtx.Method()) == "OPTIONS" {
			return r
		}
		if app.LicenseBlocksRequest(string(r.RequestCtx.Method()), string(r.RequestCtx.Path())) {
			return app.SendLicenseBlocked(r)
		}
		return r
	})
	g.Before(middleware.CSRFProtection())

	// Setup routes
	setupRoutes(g, app, lo, cfg.Server.BasePath, rdb, cfg, observabilityManager)

	// Create server with CORS wrapper
	maxRequestBodySizeMB := cfg.Server.MaxRequestBodySizeMB
	if maxRequestBodySizeMB <= 0 {
		maxRequestBodySizeMB = 110
	}
	maxRequestBodySize := maxRequestBodySizeMB * 1024 * 1024
	server := &fasthttp.Server{
		Handler:            observedHandler(corsWrapper(g.Handler(), allowedOrigins), observabilityManager),
		ReadTimeout:        time.Duration(cfg.Server.ReadTimeout) * time.Second,
		ReadBufferSize:     32 * 1024,
		WriteTimeout:       time.Duration(cfg.Server.WriteTimeout) * time.Second,
		MaxRequestBodySize: maxRequestBodySize,
		Name:               "Whatomate",
	}

	// Start server in goroutine
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	go func() {
		lo.Info("Server listening", "address", addr)
		if err := server.ListenAndServe(addr); err != nil {
			lo.Fatal("Server failed", "error", err)
		}
	}()

	var (
		slaProcessor               *handlers.SLAProcessor
		slaCancel                  context.CancelFunc
		chatAssignmentResetWorker  *handlers.ChatAssignmentResetWorker
		chatAssignmentResetCancel  context.CancelFunc
		campaignScheduler          *handlers.CampaignScheduler
		campaignSchedulerCancel    context.CancelFunc
		instanceAutoCampaignWorker *handlers.InstanceAutoCampaignWorker
		instanceAutoCampaignCancel context.CancelFunc
		mediaRetentionWorker       *handlers.MediaRetentionWorker
		mediaRetentionCancel       context.CancelFunc
		uploadsCleanupWorker       *handlers.UploadsCleanupWorker
		uploadsCleanupCancel       context.CancelFunc
	)
	if sandboxMode {
		lo.Warn("Sandbox mode: skipping recurring background workers")
	} else {
		// Start SLA processor (runs every minute)
		slaProcessor = handlers.NewSLAProcessor(app, time.Minute)
		var slaCtx context.Context
		slaCtx, slaCancel = context.WithCancel(context.Background())
		go slaProcessor.Start(slaCtx)
		lo.Info("SLA processor started")

		// Start assigned chat reset worker (checks schedule every minute).
		chatAssignmentResetWorker = handlers.NewChatAssignmentResetWorker(app, time.Minute)
		var chatAssignmentResetCtx context.Context
		chatAssignmentResetCtx, chatAssignmentResetCancel = context.WithCancel(context.Background())
		go chatAssignmentResetWorker.Start(chatAssignmentResetCtx)
		lo.Info("Assigned chat reset worker started")

		campaignScheduler = handlers.NewCampaignScheduler(app, time.Minute)
		var campaignSchedulerCtx context.Context
		campaignSchedulerCtx, campaignSchedulerCancel = context.WithCancel(context.Background())
		go campaignScheduler.Start(campaignSchedulerCtx)
		lo.Info("Campaign scheduler started")

		// Start instance auto campaign worker (checks interval every minute).
		instanceAutoCampaignWorker = handlers.NewInstanceAutoCampaignWorker(app, time.Minute)
		var instanceAutoCampaignCtx context.Context
		instanceAutoCampaignCtx, instanceAutoCampaignCancel = context.WithCancel(context.Background())
		go instanceAutoCampaignWorker.Start(instanceAutoCampaignCtx)
		lo.Info("Instance auto campaign worker started")

		mediaRetentionWorker = handlers.NewMediaRetentionWorker(app, 24*time.Hour)
		var mediaRetentionCtx context.Context
		mediaRetentionCtx, mediaRetentionCancel = context.WithCancel(context.Background())
		go mediaRetentionWorker.Start(mediaRetentionCtx)
		lo.Info("Media retention worker started")

		uploadsCleanupWorker = handlers.NewUploadsCleanupWorker(app, time.Minute)
		var uploadsCleanupCtx context.Context
		uploadsCleanupCtx, uploadsCleanupCancel = context.WithCancel(context.Background())
		go uploadsCleanupWorker.Start(uploadsCleanupCtx)
		lo.Info("Uploads cleanup worker started")
	}

	// Start embedded workers
	var (
		inboundWorker       *worker.Worker
		inboundWorkerCancel context.CancelFunc
		inboundWorkerDone   chan error
		workerScaler        *worker.WorkerScaler
		workerScalerCancel  context.CancelFunc
		workerScalerDone    chan error
	)
	if *numWorkers > 0 {
		state := licenseService.CurrentState()
		if state.LicenseID != "" && state.MaxWorkers > 0 && *numWorkers > state.MaxWorkers {
			lo.Warn("Requested embedded workers exceed licensed maximum; capping worker count",
				"requested", *numWorkers,
				"licensed_max", state.MaxWorkers)
			*numWorkers = state.MaxWorkers
		}
		inboundWorker, err = worker.New(cfg, db, rdb, lo, app.MessageProvider, licenseService, worker.WorkerOptions{
			EnableCampaignConsumer: false,
			EnableInboundMedia:     true,
		})
		if err != nil {
			lo.Fatal("Failed to create inbound media worker", "error", err)
		}
		var inboundCtx context.Context
		inboundCtx, inboundWorkerCancel = context.WithCancel(context.Background())
		inboundWorkerDone = make(chan error, 1)
		go func() {
			lo.Info("Inbound media worker started")
			err := inboundWorker.Run(inboundCtx)
			if err != nil && err != context.Canceled {
				lo.Error("Inbound media worker error", "error", err)
			}
			inboundWorkerDone <- err
		}()

		workerScaler = worker.NewWorkerScaler(cfg, db, rdb, lo, app.MessageProvider, licenseService, *numWorkers)
		var scalerCtx context.Context
		scalerCtx, workerScalerCancel = context.WithCancel(context.Background())
		workerScalerDone = make(chan error, 1)
		go func() {
			lo.Info("Worker scaler started", "budget", *numWorkers)
			err := workerScaler.Start(scalerCtx)
			if err != nil && err != context.Canceled {
				lo.Error("Worker scaler error", "error", err)
			}
			workerScalerDone <- err
		}()

		lo.Info("Embedded worker runtime started", "campaign_worker_budget", *numWorkers)
	} else {
		lo.Info("Embedded workers disabled, run workers separately")
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	lo.Info("Shutting down...")

	// Stop campaign stats subscriber
	lo.Info("Stopping campaign stats subscriber...")
	app.StopCampaignStatsSubscriber()
	lo.Info("Campaign stats subscriber stopped")

	licenseCancel()

	if slaCancel != nil && slaProcessor != nil {
		lo.Info("Stopping SLA processor...")
		slaCancel()
		slaProcessor.Stop()
		lo.Info("SLA processor stopped")
	}

	if chatAssignmentResetCancel != nil && chatAssignmentResetWorker != nil {
		lo.Info("Stopping assigned chat reset worker...")
		chatAssignmentResetCancel()
		chatAssignmentResetWorker.Stop()
		lo.Info("Assigned chat reset worker stopped")
	}

	if campaignSchedulerCancel != nil && campaignScheduler != nil {
		lo.Info("Stopping campaign scheduler...")
		campaignSchedulerCancel()
		campaignScheduler.Stop()
		lo.Info("Campaign scheduler stopped")
	}

	if instanceAutoCampaignCancel != nil && instanceAutoCampaignWorker != nil {
		lo.Info("Stopping instance auto campaign worker...")
		instanceAutoCampaignCancel()
		instanceAutoCampaignWorker.Stop()
		lo.Info("Instance auto campaign worker stopped")
	}

	if mediaRetentionCancel != nil && mediaRetentionWorker != nil {
		lo.Info("Stopping media retention worker...")
		mediaRetentionCancel()
		mediaRetentionWorker.Stop()
		lo.Info("Media retention worker stopped")
	}

	if uploadsCleanupCancel != nil && uploadsCleanupWorker != nil {
		lo.Info("Stopping uploads cleanup worker...")
		uploadsCleanupCancel()
		uploadsCleanupWorker.Stop()
		lo.Info("Uploads cleanup worker stopped")
	}

	// Stop workers first
	if workerScalerCancel != nil {
		lo.Info("Stopping worker scaler...")
		workerScalerCancel()
		if workerScaler != nil {
			workerScaler.Stop()
		}
		if workerScalerDone != nil {
			<-workerScalerDone
		}
		lo.Info("Worker scaler stopped")
	}
	if inboundWorkerCancel != nil {
		lo.Info("Stopping inbound media worker...")
		inboundWorkerCancel()
		if inboundWorkerDone != nil {
			<-inboundWorkerDone
		}
		if inboundWorker != nil {
			_ = inboundWorker.Close()
		}
		lo.Info("Inbound media worker stopped")
	}

	// Then stop server
	lo.Info("Stopping server...")
	if err := server.Shutdown(); err != nil {
		lo.Error("Server shutdown error", "error", err)
	}
	lo.Info("Server stopped")
}

// ============================================================================
// WORKER COMMAND
// ============================================================================

func runWorker(args []string) {
	workerFlags := flag.NewFlagSet("worker", flag.ExitOnError)
	configPath := workerFlags.String("config", "config.toml", "Path to config file")
	workerCount := workerFlags.Int("workers", 1, "Number of workers to run")
	_ = workerFlags.Parse(args)

	// Initialize logger
	lo := logf.New(logf.Opts{
		EnableColor:     true,
		Level:           logf.DebugLevel,
		EnableCaller:    true,
		TimestampFormat: "2006-01-02 15:04:05",
		DefaultFields:   []any{"app", "whatomate-worker"},
	})

	lo.Info("Starting Whatomate worker...", "version", Version)

	// Load configuration
	cfg, err := config.Load(*configPath)
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

	// Set log level based on environment
	if cfg.App.Environment == "production" {
		lo = logf.New(logf.Opts{
			Level:           logf.InfoLevel,
			TimestampFormat: "2006-01-02 15:04:05",
			DefaultFields:   []any{"app", "whatomate-worker"},
		})
	}

	// Connect to PostgreSQL
	db, err := database.NewPostgres(&cfg.Database, cfg.App.Debug)
	if err != nil {
		lo.Fatal("Failed to connect to database", "error", err)
	}
	lo.Info("Connected to PostgreSQL")

	// Connect to Redis
	rdb, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		lo.Fatal("Failed to connect to Redis", "error", err)
	}
	lo.Info("Connected to Redis")

	storedObjects, err := objectstorage.NewObjectStorage(&cfg.Storage)
	if err != nil {
		lo.Fatal("Failed to initialize object storage", "error", err)
	}
	if cfg.WhatsApp.Provider == "whatsmeow" && storedObjects == nil {
		lo.Fatal("Whatsmeow inbound media requires storage.type=s3")
	}

	var messageProvider provider.MessageProvider
	if cfg.WhatsApp.Provider == "whatsmeow" {
		sqlDB, err := db.DB()
		if err != nil {
			lo.Fatal("Failed to get underlying SQL DB for whatsmeow", "error", err)
		}
		storeContainer := sqlstore.NewWithDB(sqlDB, "postgres", waLog.Stdout("Database", "DEBUG", true))
		if err := storeContainer.Upgrade(context.Background()); err != nil {
			lo.Fatal("Failed to upgrade whatsmeow store", "error", err)
		}

		whatsmeowManager := whatsmeow.NewConnectionManager(db, storeContainer, lo, &cfg.Whatsmeow, nil, cfg.Storage.LocalPath)
		whatsmeowQueue := queue.NewRedisQueue(rdb, lo)
		whatsmeowManager.SetInboundMediaQueue(whatsmeowQueue)
		whatsmeowManager.SetCampaignStatsPublisher(queue.NewPublisher(rdb, lo))
		whatsmeowManager.StartHealthMonitor(context.Background())
		defer whatsmeowManager.StopHealthMonitor()
		whatsmeowManager.SetMediaService(whatsmeow.NewMediaService(db, storedObjects, lo, whatsmeowManager.GetClient))
		reconcileCtx, reconcileCancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := whatsmeowManager.ReconcileStartupStatuses(reconcileCtx); err != nil {
			lo.Warn("Failed to reconcile stale instance statuses on startup", "error", err)
		}
		reconcileCancel()

		reconnectCtx, reconnectCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		if err := whatsmeowManager.AutoConnectLinkedInstancesOnFirstRun(reconnectCtx); err != nil {
			lo.Warn("First-run auto-connect completed with issues", "error", err)
		}
		if err := whatsmeowManager.ReconnectAll(reconnectCtx); err != nil {
			lo.Error("Failed to reconnect instances", "error", err)
		}
		reconnectCancel()

		messageProvider = whatsmeow.NewWhatsmeowAdapter(whatsmeowManager, db, lo)
		lo.Info("Worker MessageProvider set to whatsmeow")
	} else {
		waClient := whatsapp.NewWithBaseURL(lo, cfg.WhatsApp.BaseURL)
		messageProvider = whatsapp.NewMetaAdapter(waClient, db, lo)
		lo.Info("Worker MessageProvider set to meta")
	}

	licenseService, err := license.NewService(cfg, db, rdb, lo)
	if err != nil {
		lo.Fatal("Failed to initialize license service", "error", err)
	}
	licenseCtx, licenseCancel := context.WithCancel(context.Background())
	defer licenseCancel()
	licenseService.Start(licenseCtx)

	state := licenseService.CurrentState()
	if state.LicenseID != "" && state.MaxWorkers > 0 && *workerCount > state.MaxWorkers {
		lo.Warn("Requested workers exceed licensed maximum; capping worker count",
			"requested", *workerCount,
			"licensed_max", state.MaxWorkers)
		*workerCount = state.MaxWorkers
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var (
		inboundWorker *worker.Worker
		workerScaler  *worker.WorkerScaler
		errCh         = make(chan error, 2)
	)

	if *workerCount > 0 {
		inboundWorker, err = worker.New(cfg, db, rdb, lo, messageProvider, licenseService, worker.WorkerOptions{
			EnableCampaignConsumer: false,
			EnableInboundMedia:     true,
		})
		if err != nil {
			lo.Fatal("Failed to create inbound media worker", "error", err)
		}

		workerScaler = worker.NewWorkerScaler(cfg, db, rdb, lo, messageProvider, licenseService, *workerCount)

		go func() {
			lo.Info("Inbound media worker started")
			errCh <- inboundWorker.Run(ctx)
		}()

		go func() {
			lo.Info("Worker scaler started", "budget", *workerCount)
			errCh <- workerScaler.Start(ctx)
		}()

		lo.Info("Dynamic worker runtime started", "campaign_worker_budget", *workerCount)
	} else {
		lo.Info("Workers disabled", "campaign_worker_budget", *workerCount)
	}

	// Wait for shutdown signal or error
	select {
	case sig := <-quit:
		lo.Info("Received shutdown signal", "signal", sig)
		cancel()
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			lo.Error("Worker error", "error", err)
			cancel()
		}
	}

	// Cleanup
	lo.Info("Shutting down workers...")
	if workerScaler != nil {
		workerScaler.Stop()
	}
	if inboundWorker != nil {
		if err := inboundWorker.Close(); err != nil {
			lo.Error("Error closing inbound worker", "error", err)
		}
	}
	lo.Info("Workers stopped")
}

// ============================================================================
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

// ============================================================================
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

// ============================================================================
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

// ============================================================================
// ROUTES
// ============================================================================

func setupRoutes(g *fastglue.Fastglue, app *handlers.App, lo logf.Logger, basePath string, rdb *redis.Client, cfg *config.Config, observabilityManager *observability.Manager) {
	sendMessageHandler := app.SendMessage
	sendMediaMessageHandler := app.SendMediaMessage
	sendTemplateMessageHandler := app.SendTemplateMessage
	sendCannedResponseHandler := app.SendCannedResponse
	createCampaignHandler := app.CreateCampaign
	updateCampaignHandler := app.UpdateCampaign
	deleteCampaignHandler := app.DeleteCampaign
	startCampaignHandler := app.StartCampaign
	pauseCampaignHandler := app.PauseCampaign
	cancelCampaignHandler := app.CancelCampaign
	retryFailedCampaignHandler := app.RetryFailed
	importRecipientsHandler := app.ImportRecipients
	deleteCampaignRecipientHandler := app.DeleteCampaignRecipient
	uploadCampaignMediaHandler := app.UploadCampaignMedia

	if cfg.RateLimit.OutboundPerUserPS > 0 || cfg.RateLimit.OutboundPerIPPS > 0 {
		if cfg.RateLimit.OutboundPerUserPS > 0 {
			opts := middleware.RateLimitOpts{
				Redis:      rdb,
				Log:        lo,
				Max:        cfg.RateLimit.OutboundPerUserPS,
				Window:     time.Second,
				KeyPrefix:  "outbound_user",
				TrustProxy: cfg.RateLimit.TrustProxy,
				KeyFunc:    outboundRateLimitUserKey,
			}
			sendMessageHandler = withRateLimit(sendMessageHandler, opts)
			sendMediaMessageHandler = withRateLimit(sendMediaMessageHandler, opts)
			sendTemplateMessageHandler = withRateLimit(sendTemplateMessageHandler, opts)
			sendCannedResponseHandler = withRateLimit(sendCannedResponseHandler, opts)
		}
		if cfg.RateLimit.OutboundPerIPPS > 0 {
			opts := middleware.RateLimitOpts{
				Redis:      rdb,
				Log:        lo,
				Max:        cfg.RateLimit.OutboundPerIPPS,
				Window:     time.Second,
				KeyPrefix:  "outbound_ip",
				TrustProxy: cfg.RateLimit.TrustProxy,
			}
			sendMessageHandler = withRateLimit(sendMessageHandler, opts)
			sendMediaMessageHandler = withRateLimit(sendMediaMessageHandler, opts)
			sendTemplateMessageHandler = withRateLimit(sendTemplateMessageHandler, opts)
			sendCannedResponseHandler = withRateLimit(sendCannedResponseHandler, opts)
		}

		lo.Info("Outbound message rate limiting enabled",
			"per_user_per_second", cfg.RateLimit.OutboundPerUserPS,
			"per_ip_per_second", cfg.RateLimit.OutboundPerIPPS)
	}
	if cfg.RateLimit.Enabled && cfg.RateLimit.CampaignMutatingMaxAttempts > 0 {
		campaignMutatingOpts := middleware.RateLimitOpts{
			Redis:      rdb,
			Log:        lo,
			Max:        cfg.RateLimit.CampaignMutatingMaxAttempts,
			Window:     time.Duration(cfg.RateLimit.WindowSeconds) * time.Second,
			KeyPrefix:  "campaign_mutating",
			TrustProxy: cfg.RateLimit.TrustProxy,
			KeyFunc:    outboundRateLimitUserKey,
		}
		createCampaignHandler = withRateLimit(createCampaignHandler, campaignMutatingOpts)
		updateCampaignHandler = withRateLimit(updateCampaignHandler, campaignMutatingOpts)
		deleteCampaignHandler = withRateLimit(deleteCampaignHandler, campaignMutatingOpts)
		startCampaignHandler = withRateLimit(startCampaignHandler, campaignMutatingOpts)
		pauseCampaignHandler = withRateLimit(pauseCampaignHandler, campaignMutatingOpts)
		cancelCampaignHandler = withRateLimit(cancelCampaignHandler, campaignMutatingOpts)
		retryFailedCampaignHandler = withRateLimit(retryFailedCampaignHandler, campaignMutatingOpts)
		importRecipientsHandler = withRateLimit(importRecipientsHandler, campaignMutatingOpts)
		deleteCampaignRecipientHandler = withRateLimit(deleteCampaignRecipientHandler, campaignMutatingOpts)
		uploadCampaignMediaHandler = withRateLimit(uploadCampaignMediaHandler, campaignMutatingOpts)
	}

	// Health check
	g.GET("/health", app.HealthCheck)
	g.GET("/ready", app.ReadyCheck)
	if observabilityManager != nil && observabilityManager.MetricsEnabled() {
		g.GET("/metrics", observabilityManager.MetricsHandler())
	}
	if observabilityManager != nil {
		observabilityManager.RegisterPprofRoutes(g)
	}
	g.GET("/api/license/bootstrap", app.GetLicenseBootstrap)
	g.POST("/api/license/activate", app.ActivateLicense)

	// Auth routes (public, optionally rate-limited)
	if cfg.RateLimit.Enabled {
		window := time.Duration(cfg.RateLimit.WindowSeconds) * time.Second
		lo.Info("Rate limiting enabled on auth endpoints",
			"login_max", cfg.RateLimit.LoginMaxAttempts,
			"register_max", cfg.RateLimit.RegisterMaxAttempts,
			"refresh_max", cfg.RateLimit.RefreshMaxAttempts,
			"sso_max", cfg.RateLimit.SSOMaxAttempts,
			"window_seconds", cfg.RateLimit.WindowSeconds)

		g.POST("/api/auth/login", withRateLimit(app.Login, middleware.RateLimitOpts{
			Redis: rdb, Log: lo, Max: cfg.RateLimit.LoginMaxAttempts, Window: window, KeyPrefix: "login", TrustProxy: cfg.RateLimit.TrustProxy,
		}))
		g.POST("/api/auth/register", withRateLimit(app.Register, middleware.RateLimitOpts{
			Redis: rdb, Log: lo, Max: cfg.RateLimit.RegisterMaxAttempts, Window: window, KeyPrefix: "register", TrustProxy: cfg.RateLimit.TrustProxy,
		}))
		g.POST("/api/auth/refresh", withRateLimit(app.RefreshToken, middleware.RateLimitOpts{
			Redis: rdb, Log: lo, Max: cfg.RateLimit.RefreshMaxAttempts, Window: window, KeyPrefix: "refresh", TrustProxy: cfg.RateLimit.TrustProxy,
		}))
	} else {
		g.POST("/api/auth/login", app.Login)
		g.POST("/api/auth/register", app.Register)
		g.POST("/api/auth/refresh", app.RefreshToken)
	}
	// Authenticated endpoint: generate signed registration invite for current org.
	g.POST("/api/auth/register/invite", app.CreateRegisterInvite)
	g.POST("/api/auth/logout", app.Logout)
	g.POST("/api/auth/switch-org", app.SwitchOrg)
	g.GET("/api/auth/ws-token", app.GetWSToken)
	g.GET("/api/auth/me", app.GetCurrentUser) // Legacy alias for /api/me

	// SSO routes (public, optionally rate-limited)
	g.GET("/api/auth/sso/providers", app.GetPublicSSOProviders)
	if cfg.RateLimit.Enabled {
		window := time.Duration(cfg.RateLimit.WindowSeconds) * time.Second
		g.GET("/api/auth/sso/{provider}/init", withRateLimit(app.InitSSO, middleware.RateLimitOpts{
			Redis: rdb, Log: lo, Max: cfg.RateLimit.SSOMaxAttempts, Window: window, KeyPrefix: "sso_init", TrustProxy: cfg.RateLimit.TrustProxy,
		}))
		g.GET("/api/auth/sso/{provider}/callback", withRateLimit(app.CallbackSSO, middleware.RateLimitOpts{
			Redis: rdb, Log: lo, Max: cfg.RateLimit.SSOMaxAttempts, Window: window, KeyPrefix: "sso_callback", TrustProxy: cfg.RateLimit.TrustProxy,
		}))
	} else {
		g.GET("/api/auth/sso/{provider}/init", app.InitSSO)
		g.GET("/api/auth/sso/{provider}/callback", app.CallbackSSO)
	}

	// Webhook routes (public - for Meta)
	g.GET("/api/webhook", app.WebhookVerify)
	if cfg.RateLimit.Enabled {
		window := time.Duration(cfg.RateLimit.WindowSeconds) * time.Second
		g.POST("/api/webhook", withRateLimit(app.WebhookHandler, middleware.RateLimitOpts{
			Redis: rdb, Log: lo, Max: cfg.RateLimit.WebhookMaxAttempts, Window: window, KeyPrefix: "webhook", TrustProxy: cfg.RateLimit.TrustProxy,
		}))
	} else {
		g.POST("/api/webhook", app.WebhookHandler)
	}

	// WebSocket route (auth via message-based flow after upgrade)
	g.GET("/ws", app.WebSocketHandler)

	// For protected routes, we'll use a path-based middleware approach
	// Apply auth middleware globally but check path in the middleware
	g.Before(func(r *fastglue.Request) *fastglue.Request {
		// Skip auth for OPTIONS preflight requests (handled by CORS middleware)
		if string(r.RequestCtx.Method()) == "OPTIONS" {
			return r
		}
		path := string(r.RequestCtx.Path())
		// Skip auth for public routes
		if path == "/health" || path == "/ready" ||
			path == "/api/license/bootstrap" || path == "/api/license/activate" ||
			path == "/api/auth/login" || path == "/api/auth/register" || path == "/api/auth/refresh" ||
			path == "/api/auth/logout" || path == "/api/webhook" || path == "/ws" {
			return r
		}
		// Skip auth for SSO routes (they handle their own auth via state tokens)
		if len(path) >= 13 && path[:13] == "/api/auth/sso" {
			return r
		}
		// Skip auth for custom action redirects (uses one-time token)
		if len(path) >= 28 && path[:28] == "/api/custom-actions/redirect" {
			return r
		}
		// Apply auth for all other /api routes (supports both JWT and API key)
		if len(path) > 4 && path[:4] == "/api" {
			return middleware.AuthWithDB(app.Config.JWT.Secret, app.DB)(r)
		}
		return r
	})

	g.Before(func(r *fastglue.Request) *fastglue.Request {
		if string(r.RequestCtx.Method()) == "OPTIONS" {
			return r
		}

		path := string(r.RequestCtx.Path())
		if path == "/health" || path == "/ready" ||
			path == "/api/license/bootstrap" || path == "/api/license/activate" ||
			path == "/api/auth/login" || path == "/api/auth/register" || path == "/api/auth/refresh" ||
			path == "/api/auth/logout" || path == "/api/webhook" || path == "/ws" {
			return r
		}
		if len(path) >= 13 && path[:13] == "/api/auth/sso" {
			return r
		}
		if len(path) >= 28 && path[:28] == "/api/custom-actions/redirect" {
			return r
		}
		if len(path) > 4 && path[:4] == "/api" {
			return middleware.TenantScope(app.DB)(r)
		}

		return r
	})

	// Role-based access control middleware
	g.Before(func(r *fastglue.Request) *fastglue.Request {
		method := string(r.RequestCtx.Method())

		// Skip OPTIONS preflight requests
		if method == "OPTIONS" {
			return r
		}

		// Route-level permission checks are now handled at the handler level
		// using the granular permission system (HasPermission checks)
		return r
	})

	// Current User (all authenticated users)
	g.GET("/api/me", app.GetCurrentUser)
	g.PUT("/api/me/settings", app.UpdateCurrentUserSettings)
	g.POST("/api/me/chat-background", app.UploadCurrentUserChatBackground)
	g.GET("/api/me/chat-background", app.GetCurrentUserChatBackground)
	g.PUT("/api/me/password", app.ChangePassword)
	g.PUT("/api/me/availability", app.UpdateAvailability)
	g.GET("/api/me/organizations", app.ListMyOrganizations)

	// User Management (admin only - enforced by middleware)
	g.GET("/api/users", app.ListUsers)
	g.POST("/api/users", app.CreateUser)
	g.GET("/api/users/{id}", app.GetUser)
	g.PUT("/api/users/{id}", app.UpdateUser)
	g.DELETE("/api/users/{id}", app.DeleteUser)
	g.GET("/api/users/{id}/send-restrictions", app.GetUserSendRestrictions)
	g.PUT("/api/users/{id}/send-restrictions", app.UpdateUserSendRestrictions)

	// Roles & Permissions (admin only - enforced by middleware)
	g.GET("/api/roles", app.ListRoles)
	g.POST("/api/roles", app.CreateRole)
	g.GET("/api/roles/{id}", app.GetRole)
	g.PUT("/api/roles/{id}", app.UpdateRole)
	g.DELETE("/api/roles/{id}", app.DeleteRole)
	g.GET("/api/permissions", app.ListPermissions)

	// API Keys (admin only - enforced by middleware)
	g.GET("/api/api-keys", app.ListAPIKeys)
	g.POST("/api/api-keys", app.CreateAPIKey)
	g.DELETE("/api/api-keys/{id}", app.DeleteAPIKey)

	// Accounts
	g.GET("/api/accounts", app.ListAccounts)
	g.POST("/api/accounts", app.CreateAccount)
	g.GET("/api/accounts/{id}", app.GetAccount)
	g.PUT("/api/accounts/{id}", app.UpdateAccount)
	g.DELETE("/api/accounts/{id}", app.DeleteAccount)
	g.POST("/api/accounts/{id}/test", app.TestAccountConnection)
	g.POST("/api/accounts/{id}/subscribe", app.SubscribeApp)
	g.GET("/api/accounts/{id}/business_profile", app.GetBusinessProfile)
	g.PUT("/api/accounts/{id}/business_profile", app.UpdateBusinessProfile)
	g.POST("/api/accounts/{id}/business_profile/photo", app.UpdateProfilePicture)

	// Contacts
	g.GET("/api/contacts", app.ListContacts)
	g.POST("/api/contacts", app.CreateContact)
	g.GET("/api/contacts/{id}", app.GetContact)
	g.PUT("/api/contacts/{id}", app.UpdateContact)
	g.DELETE("/api/contacts/{id}", app.DeleteContact)
	g.POST("/api/contacts/{id}/soft-delete", app.SoftDeleteContactForUser)
	g.PUT("/api/contacts/{id}/assign", app.AssignContact)
	g.GET("/api/contacts/{id}/collaborators", app.ListContactCollaborators)
	g.POST("/api/contacts/{id}/collaborators", app.InviteContactCollaborator)
	g.PUT("/api/contacts/{id}/collaborators/{user_id}/accept", app.AcceptContactCollaborator)
	g.PUT("/api/contacts/{id}/collaborators/{user_id}/decline", app.DeclineContactCollaborator)
	g.DELETE("/api/contacts/{id}/collaborators/{user_id}", app.RemoveContactCollaborator)
	g.PUT("/api/contacts/{id}/tags", app.UpdateContactTags)
	g.GET("/api/contacts/{id}/session-data", app.GetContactSessionData)

	// Chats (contact-backed alias + lifecycle endpoints)
	g.GET("/api/chats", app.ListContacts)
	g.PUT("/api/chats/{id}/claim", app.ClaimChat)
	g.PUT("/api/chats/{id}/close", app.CloseChat)
	g.PUT("/api/chats/{id}/reopen", app.ReopenChat)
	g.PUT("/api/chats/{id}/public", app.SetChatPublic)
	g.GET("/api/chats/{id}/messages", app.GetMessages)

	// Generic Import/Export
	g.POST("/api/export", app.ExportData)
	g.POST("/api/import", app.ImportData)
	g.GET("/api/export/{table}/config", app.GetExportConfig)
	g.GET("/api/import/{table}/config", app.GetImportConfig)

	// Tags
	g.GET("/api/tags", app.ListTags)
	g.POST("/api/tags", app.CreateTag)
	g.PUT("/api/tags/{name}", app.UpdateTag)
	g.DELETE("/api/tags/{name}", app.DeleteTag)

	// Messages
	g.GET("/api/contacts/{id}/messages", app.GetMessages)
	g.POST("/api/contacts/{id}/messages", sendMessageHandler)
	g.POST("/api/contacts/{id}/typing", app.SendTypingPresence)
	g.POST("/api/contacts/{id}/messages/{message_id}/reaction", app.SendReaction)
	g.POST("/api/contacts/{id}/messages/{message_id}/revoke", app.RevokeMessage)
	g.POST("/api/messages", sendMessageHandler) // Legacy route
	g.POST("/api/messages/template", sendTemplateMessageHandler)
	g.POST("/api/messages/media", sendMediaMessageHandler)
	g.PUT("/api/messages/{id}/read", app.MarkMessageRead)
	g.GET("/api/statuses", app.ListStatuses)
	g.GET("/api/statuses/{id}/media", app.ServeStatusMedia)
	g.POST("/api/statuses/{id}/reply", app.ReplyToStatus)
	g.POST("/api/statuses/{id}/mark-seen", app.MarkStatusSeen)

	// WhatsApp Instances (whatsmeow)
	g.GET("/api/instances", app.ListInstances)
	g.POST("/api/instances", app.CreateInstance)
	g.GET("/api/instances/{id}", app.GetInstance)
	g.PUT("/api/instances/{id}", app.UpdateInstance)
	g.DELETE("/api/instances/{id}", app.DeleteInstance)
	g.GET("/api/instances/{id}/health", app.GetInstanceHealth)
	g.GET("/api/instances/{id}/qr", app.GetInstanceQRCodeSnapshot)
	g.POST("/api/instances/{id}/connect", app.ConnectInstance)
	g.POST("/api/instances/{id}/pair-phone", app.PairPhoneInstance)
	g.POST("/api/instances/{id}/disconnect", app.DisconnectInstance)
	g.POST("/api/instances/{id}/reconnect", app.ReconnectInstance)
	g.POST("/api/instances/{id}/status/send", app.SendStatus)
	g.POST("/api/instances/{id}/auto-campaign/media", app.UploadInstanceAutoCampaignMedia)
	g.GET("/api/notifications", app.ListNotifications)
	g.PUT("/api/notifications/{id}/dismiss", app.DismissNotification)

	// App Config (provider & feature flags)
	g.GET("/api/config", app.GetAppConfig)

	// Conversation Notes
	g.GET("/api/contacts/{id}/notes", app.ListConversationNotes)
	g.POST("/api/contacts/{id}/notes", app.CreateConversationNote)
	g.PUT("/api/contacts/{id}/notes/{note_id}", app.UpdateConversationNote)
	g.DELETE("/api/contacts/{id}/notes/{note_id}", app.DeleteConversationNote)

	// Media (serves media files for messages, auth-protected)
	g.GET("/api/media/{message_id}", app.ServeMedia)

	// Templates (Meta only)
	meta := func(h fastglue.FastRequestHandler) fastglue.FastRequestHandler { return app.ProviderGuard("meta", h) }
	g.GET("/api/templates", meta(app.ListTemplates))
	g.POST("/api/templates", meta(app.CreateTemplate))
	g.GET("/api/templates/{id}", meta(app.GetTemplate))
	g.PUT("/api/templates/{id}", meta(app.UpdateTemplate))
	g.DELETE("/api/templates/{id}", meta(app.DeleteTemplate))
	g.POST("/api/templates/sync", meta(app.SyncTemplates))
	g.POST("/api/templates/{id}/publish", meta(app.SubmitTemplate))
	g.POST("/api/templates/upload-media", meta(app.UploadTemplateMedia))

	// WhatsApp Flows (Meta only)
	g.GET("/api/flows", meta(app.ListFlows))
	g.POST("/api/flows", meta(app.CreateFlow))
	g.GET("/api/flows/{id}", meta(app.GetFlow))
	g.PUT("/api/flows/{id}", meta(app.UpdateFlow))
	g.DELETE("/api/flows/{id}", meta(app.DeleteFlow))
	g.POST("/api/flows/{id}/save-to-meta", meta(app.SaveFlowToMeta))
	g.POST("/api/flows/{id}/publish", meta(app.PublishFlow))
	g.POST("/api/flows/{id}/deprecate", meta(app.DeprecateFlow))
	g.POST("/api/flows/{id}/duplicate", meta(app.DuplicateFlow))
	g.POST("/api/flows/sync", meta(app.SyncFlows))

	// Bulk Campaigns (supported for Meta and whatsmeow)
	g.GET("/api/campaigns", app.ListCampaigns)
	g.POST("/api/campaigns", createCampaignHandler)
	g.GET("/api/campaigns/{id}", app.GetCampaign)
	g.PUT("/api/campaigns/{id}", updateCampaignHandler)
	g.DELETE("/api/campaigns/{id}", deleteCampaignHandler)
	g.POST("/api/campaigns/{id}/start", startCampaignHandler)
	g.POST("/api/campaigns/{id}/pause", pauseCampaignHandler)
	g.POST("/api/campaigns/{id}/cancel", cancelCampaignHandler)
	g.POST("/api/campaigns/{id}/retry-failed", retryFailedCampaignHandler)
	g.GET("/api/campaigns/{id}/progress", app.GetCampaign)
	g.POST("/api/campaigns/{id}/recipients/import", importRecipientsHandler)
	g.GET("/api/campaigns/{id}/recipients", app.GetCampaignRecipients)
	g.DELETE("/api/campaigns/{id}/recipients/{recipientId}", deleteCampaignRecipientHandler)
	g.POST("/api/campaigns/{id}/media", uploadCampaignMediaHandler)
	g.GET("/api/campaigns/{id}/media", app.ServeCampaignMedia)

	// Chatbot Settings
	g.GET("/api/chatbot/settings", app.GetChatbotSettings)
	g.PUT("/api/chatbot/settings", app.UpdateChatbotSettings)

	// Keyword Rules
	g.GET("/api/chatbot/keywords", app.ListKeywordRules)
	g.POST("/api/chatbot/keywords", app.CreateKeywordRule)
	g.GET("/api/chatbot/keywords/{id}", app.GetKeywordRule)
	g.PUT("/api/chatbot/keywords/{id}", app.UpdateKeywordRule)
	g.DELETE("/api/chatbot/keywords/{id}", app.DeleteKeywordRule)

	// Chatbot Flows
	g.GET("/api/chatbot/flows", app.ListChatbotFlows)
	g.POST("/api/chatbot/flows", app.CreateChatbotFlow)
	g.GET("/api/chatbot/flows/{id}", app.GetChatbotFlow)
	g.PUT("/api/chatbot/flows/{id}", app.UpdateChatbotFlow)
	g.DELETE("/api/chatbot/flows/{id}", app.DeleteChatbotFlow)

	// AI Contexts
	g.GET("/api/chatbot/ai-contexts", app.ListAIContexts)
	g.POST("/api/chatbot/ai-contexts", app.CreateAIContext)
	g.GET("/api/chatbot/ai-contexts/{id}", app.GetAIContext)
	g.PUT("/api/chatbot/ai-contexts/{id}", app.UpdateAIContext)
	g.DELETE("/api/chatbot/ai-contexts/{id}", app.DeleteAIContext)

	// Agent Transfers
	g.GET("/api/chatbot/transfers", app.ListAgentTransfers)
	g.POST("/api/chatbot/transfers", app.CreateAgentTransfer)
	g.POST("/api/chatbot/transfers/pick", app.PickNextTransfer)
	g.PUT("/api/chatbot/transfers/{id}/resume", app.ResumeFromTransfer)
	g.PUT("/api/chatbot/transfers/{id}/assign", app.AssignAgentTransfer)

	// Teams (admin/manager - access control in handler)
	g.GET("/api/teams", app.ListTeams)
	g.POST("/api/teams", app.CreateTeam)
	g.GET("/api/teams/{id}", app.GetTeam)
	g.PUT("/api/teams/{id}", app.UpdateTeam)
	g.DELETE("/api/teams/{id}", app.DeleteTeam)
	g.GET("/api/teams/{id}/members", app.ListTeamMembers)
	g.POST("/api/teams/{id}/members", app.AddTeamMember)
	g.DELETE("/api/teams/{id}/members/{member_user_id}", app.RemoveTeamMember)

	// Canned Responses
	g.GET("/api/canned-responses", app.ListCannedResponses)
	g.POST("/api/canned-responses", app.CreateCannedResponse)
	g.GET("/api/canned-responses/{id}", app.GetCannedResponse)
	g.PUT("/api/canned-responses/{id}", app.UpdateCannedResponse)
	g.DELETE("/api/canned-responses/{id}", app.DeleteCannedResponse)
	g.POST("/api/canned-responses/{id}/send", sendCannedResponseHandler)
	g.POST("/api/canned-responses/{id}/use", app.IncrementCannedResponseUsage)

	// Sessions (admin/debug)
	g.GET("/api/chatbot/sessions", app.ListChatbotSessions)
	g.GET("/api/chatbot/sessions/{id}", app.GetChatbotSession)
	g.GET("/api/chat/sessions", app.ListChatbotSessions)    // Legacy alias for /api/chatbot/sessions
	g.GET("/api/chat/sessions/{id}", app.GetChatbotSession) // Legacy alias for /api/chatbot/sessions/{id}

	// Analytics
	g.GET("/api/analytics", app.GetDashboardStats) // Dashboard alias for /api/analytics/dashboard
	g.GET("/api/analytics/dashboard", app.GetDashboardStats)
	g.GET("/api/analytics/messages", app.GetMessageAnalytics)
	g.GET("/api/analytics/chatbot", app.GetChatbotAnalytics)
	g.GET("/api/analytics/agents", app.GetAgentAnalytics)
	g.GET("/api/analytics/agents/comparison", app.GetAgentComparison)
	g.GET("/api/analytics/agents/ratings/export", app.ExportAgentRatings)
	g.GET("/api/analytics/agents/{id}", app.GetAgentDetails)

	// Meta WhatsApp Analytics (Meta only)
	g.GET("/api/analytics/meta", meta(app.GetMetaAnalytics))
	g.GET("/api/analytics/meta/accounts", meta(app.ListMetaAccountsForAnalytics))
	g.POST("/api/analytics/meta/refresh", meta(app.RefreshMetaAnalyticsCache))

	// Widgets (customizable analytics)
	g.GET("/api/widgets", app.ListWidgets)
	g.POST("/api/widgets", app.CreateWidget)
	g.GET("/api/widgets/data-sources", app.GetWidgetDataSources)
	g.GET("/api/widgets/data", app.GetAllWidgetsData)
	g.GET("/api/widgets/{id}", app.GetWidget)
	g.PUT("/api/widgets/{id}", app.UpdateWidget)
	g.DELETE("/api/widgets/{id}", app.DeleteWidget)
	g.GET("/api/widgets/{id}/data", app.GetWidgetData)
	g.POST("/api/widgets/layout", app.SaveWidgetLayout)

	// Organization Settings
	g.GET("/api/org/settings", app.GetOrganizationSettings)
	g.PUT("/api/org/settings", app.UpdateOrganizationSettings)
	g.POST("/api/org/uploads-cleanup/run", app.RunUploadsCleanupNow)

	// Organizations
	g.GET("/api/organizations", app.ListOrganizations)
	g.POST("/api/organizations", app.CreateOrganization)
	g.DELETE("/api/organizations/{id}", app.DeleteOrganization)
	g.GET("/api/organizations/current", app.GetCurrentOrganization)
	g.GET("/api/organizations/members", app.ListOrganizationMembers)
	g.POST("/api/organizations/members", app.AddOrganizationMember)
	g.PUT("/api/organizations/members/{member_id}", app.UpdateOrganizationMemberRole)
	g.DELETE("/api/organizations/members/{member_id}", app.RemoveOrganizationMember)

	// SSO Settings (admin only - enforced by middleware)
	g.GET("/api/settings/sso", app.GetSSOSettings)
	g.PUT("/api/settings/sso/{provider}", app.UpdateSSOProvider)
	g.DELETE("/api/settings/sso/{provider}", app.DeleteSSOProvider)

	// Webhooks
	g.GET("/api/webhooks", app.ListWebhooks)
	g.POST("/api/webhooks", app.CreateWebhook)
	g.GET("/api/webhooks/{id}", app.GetWebhook)
	g.PUT("/api/webhooks/{id}", app.UpdateWebhook)
	g.DELETE("/api/webhooks/{id}", app.DeleteWebhook)
	g.POST("/api/webhooks/{id}/test", app.TestWebhook)

	// Custom Actions
	g.GET("/api/custom-actions", app.ListCustomActions)
	g.POST("/api/custom-actions", app.CreateCustomAction)
	g.GET("/api/custom-actions/{id}", app.GetCustomAction)
	g.PUT("/api/custom-actions/{id}", app.UpdateCustomAction)
	g.DELETE("/api/custom-actions/{id}", app.DeleteCustomAction)
	g.POST("/api/custom-actions/{id}/execute", app.ExecuteCustomAction)
	g.GET("/api/custom-actions/redirect/{token}", app.CustomActionRedirect)

	// Catalogs (Meta only)
	g.GET("/api/catalogs", meta(app.ListCatalogs))
	g.POST("/api/catalogs", meta(app.CreateCatalog))
	g.GET("/api/catalogs/{id}", meta(app.GetCatalog))
	g.DELETE("/api/catalogs/{id}", meta(app.DeleteCatalog))
	g.POST("/api/catalogs/sync", meta(app.SyncCatalogs))

	// Catalog Products (Meta only)
	g.GET("/api/catalogs/{id}/products", meta(app.ListCatalogProducts))
	g.POST("/api/catalogs/{id}/products", meta(app.CreateCatalogProduct))
	g.GET("/api/products/{id}", meta(app.GetCatalogProduct))
	g.PUT("/api/products/{id}", meta(app.UpdateCatalogProduct))
	g.DELETE("/api/products/{id}", meta(app.DeleteCatalogProduct))

	// Serve embedded frontend (SPA)
	if frontend.IsEmbedded() {
		lo.Info("Serving embedded frontend", "base_path", basePath)
		frontendHandler := frontend.Handler(basePath)
		// Catch-all for frontend routes
		g.GET("/{path:*}", func(r *fastglue.Request) error {
			frontendHandler(r.RequestCtx)
			return nil
		})
		g.GET("/", func(r *fastglue.Request) error {
			frontendHandler(r.RequestCtx)
			return nil
		})
	} else {
		lo.Info("Frontend not embedded, API-only mode")
	}
}

// withRateLimit wraps a handler with the rate limit middleware.
func withRateLimit(handler fastglue.FastRequestHandler, opts middleware.RateLimitOpts) fastglue.FastRequestHandler {
	rl := middleware.RateLimit(opts)
	return func(r *fastglue.Request) error {
		if rl(r) == nil {
			return nil // Rate limited — response already sent.
		}
		return handler(r)
	}
}

func observedHandler(handler fasthttp.RequestHandler, observabilityManager *observability.Manager) fasthttp.RequestHandler {
	if observabilityManager == nil {
		return handler
	}
	return observabilityManager.Wrap(handler)
}

func outboundRateLimitUserKey(r *fastglue.Request, clientIP string) string {
	userID, ok := r.RequestCtx.UserValue(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return "anonymous:" + strings.TrimSpace(clientIP)
	}

	orgID, ok := r.RequestCtx.UserValue(middleware.ContextKeyOrganizationID).(uuid.UUID)
	if !ok || orgID == uuid.Nil {
		return userID.String()
	}

	return orgID.String() + ":" + userID.String()
}

// corsWrapper wraps a handler with CORS support at the fasthttp level.
// This ensures CORS headers are set even for auto-handled OPTIONS requests.
func corsWrapper(next fasthttp.RequestHandler, allowedOrigins map[string]bool) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		origin := string(ctx.Request.Header.Peek("Origin"))

		if origin != "" && middleware.IsOriginAllowedForRequest(origin, allowedOrigins, string(ctx.Host()), ctx.IsTLS()) {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
			ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
			ctx.Response.Header.Set("Vary", "Origin")
		}

		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Organization-ID, X-CSRF-Token")
		ctx.Response.Header.Set("Access-Control-Max-Age", "86400")

		// Handle preflight OPTIONS requests
		if string(ctx.Method()) == "OPTIONS" {
			ctx.SetStatusCode(fasthttp.StatusNoContent)
			return
		}

		next(ctx)
	}
}
