package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/compnew2006/whatomate/internal/database"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/internal/worker"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/zerodha/logf"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// WORKER COMMAND
// ============================================================================

func runWorker(args []string) {
	workerFlags := flag.NewFlagSet("worker", flag.ExitOnError)
	configPath := workerFlags.String("config", "config.toml", "Path to config file")
	workerCount := workerFlags.Int("workers", 1, "Number of workers to run")
	_ = workerFlags.Parse(args)

	// Initialize logger
	lo := initLogger("whatomate-worker", logf.DebugLevel)

	lo.Info("Starting Whatomate worker...", "version", Version)

	// Load configuration
	cfg := loadAndValidateConfig(*configPath, lo)

	lo = adjustLogLevelForProduction(cfg, lo, "whatomate-worker")

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

	storedObjects := initObjectStorage(cfg, lo)
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

	licenseService, licenseCancel := initLicenseService(cfg, db, rdb, lo)
	defer licenseCancel()

	cappedWorkers := capWorkerCount(*workerCount, licenseService, lo)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var (
		inboundWorker  *worker.Worker
		workerScaler   *worker.WorkerScaler
		dlqRetryWorker *worker.DLQRetyWorker
		dlqRetryCancel context.CancelFunc
		errCh          = make(chan error, 2)
	)

	if cappedWorkers > 0 {
		inboundWorker, err = worker.New(cfg, db, rdb, lo, messageProvider, licenseService, worker.WorkerOptions{
			EnableCampaignConsumer: false,
			EnableInboundMedia:     true,
		})
		if err != nil {
			lo.Fatal("Failed to create inbound media worker", "error", err)
		}

		workerScaler = worker.NewWorkerScaler(cfg, db, rdb, lo, messageProvider, licenseService, cappedWorkers)

		go func() {
			lo.Info("Inbound media worker started")
			errCh <- inboundWorker.Run(ctx)
		}()

		go func() {
			lo.Info("Worker scaler started", "budget", cappedWorkers)
			errCh <- workerScaler.Start(ctx)
		}()

		lo.Info("Dynamic worker runtime started", "campaign_worker_budget", cappedWorkers)
	} else {
		lo.Info("Workers disabled", "campaign_worker_budget", cappedWorkers)
	}

	dlqRetryWorker = worker.NewDLQRetyWorker(db, rdb, lo, (&handlers.App{
		DB:         db,
		Redis:      rdb,
		Log:        lo,
		Config:     cfg,
		License:    licenseService,
		InboundDLQ: queue.NewInboundDLQ(rdb, lo),
	}).RetryInboundDLQEntry)
	var dlqCtx context.Context
	dlqCtx, dlqRetryCancel = context.WithCancel(context.Background())
	go dlqRetryWorker.Run(dlqCtx)
	lo.Info("Inbound DLQ retry worker started")

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
	if dlqRetryCancel != nil {
		lo.Info("Stopping DLQ retry worker...")
		dlqRetryCancel()
	}
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
