package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/compnew2006/gowa-ui/internal/middleware"
	"github.com/zerodha/fastglue"
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
	case "version":
		fmt.Printf("Gowa-UI %s (built %s)\n", Version, BuildTime)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Gowa-UI - WhatsApp Business API Platform

Usage:
  gowa-ui <command> [options]

Commands:
  server    Start the API server (with optional embedded workers)
  worker    Start background workers only (no API server)
  version   Show version information
  help      Show this help message

Server Options:
  -config string    Path to config file (default "config.toml")
  -migrate          Run database migrations on startup
  -workers int      Number of embedded workers (0 to disable) (default 1)

Worker Options:
  -config string    Path to config file (default "config.toml")
  -workers int      Number of workers to run (default 1)

Examples:
  gowa-ui server                     # API + 1 embedded worker
  gowa-ui server -workers 0          # API only (no workers)
  gowa-ui server -workers 4          # API + 4 embedded workers
  gowa-ui server -migrate            # Run migrations and start server
  gowa-ui worker -workers 4          # 4 workers only (no API)

Deployment Scenarios:
  All-in-one:    gowa-ui server
  Separate:      gowa-ui server -workers 0  (on API server)
                 gowa-ui worker -workers 4  (on worker server)`)
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

	lo := setupLogger("", "gowa-ui")
	lo.Info("Starting Gowa-UI server...", "version", Version)

	// Load + validate config (validates env-based invariants, may Fatal).
	cfg := loadAndValidateConfig(*configPath, lo)

	// Logger may downgrade to info in production now that the environment is known.
	lo = setupLogger(cfg.App.Environment, "gowa-ui")

	db := setupDB(cfg, lo)
	if *migrate {
		runMigrations(db, cfg, lo)
	}

	rdb := setupRedis(&cfg.Redis, lo)

	jobQueue := setupQueue(rdb, lo)
	lo.Info("Job queue initialized")

	waRegistry := setupWARegistry(db, cfg, lo)
	wsHub := setupWSHub(lo)
	httpClient := setupHTTPClient()
	app := setupApp(cfg, db, rdb, lo, waRegistry, wsHub, httpClient, jobQueue)

	// Parse allowed origins for CORS (CORS is handled by corsWrapper at fasthttp level).
	allowedOrigins := middleware.ParseAllowedOrigins(cfg.Server.AllowedOrigins)

	g := fastglue.NewGlue()
	setupMiddleware(g, lo)
	setupRoutes(g, app, lo, cfg.Server.BasePath, rdb, cfg)

	server := setupHTTPServer(corsWrapper(g.Handler(), allowedOrigins), cfg)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	go func() {
		lo.Info("Server listening", "address", addr)
		if err := server.ListenAndServe(addr); err != nil {
			lo.Fatal("Server failed", "error", err)
		}
	}()

	procs := startProcessors(app, lo)
	workers, workerCancel, _ := startEmbeddedWorkers(cfg, db, rdb, lo, waRegistry, *numWorkers)

	_ = waitForShutdownSignal()
	gracefulShutdown(lo, app, procs, workers, workerCancel, server)
}

// ============================================================================
// WORKER COMMAND
// ============================================================================

func runWorker(args []string) {
	workerFlags := flag.NewFlagSet("worker", flag.ExitOnError)
	configPath := workerFlags.String("config", "config.toml", "Path to config file")
	workerCount := workerFlags.Int("workers", 1, "Number of workers to run")
	_ = workerFlags.Parse(args)

	lo := setupLogger("", "gowa-ui-worker")
	lo.Info("Starting Gowa-UI worker...", "version", Version)

	cfg := loadAndValidateConfig(*configPath, lo)
	lo = setupLogger(cfg.App.Environment, "gowa-ui-worker")

	db := setupDB(cfg, lo)
	rdb := setupRedis(&cfg.Redis, lo)
	waRegistry := setupWARegistry(db, cfg, lo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, *workerCount)
	workers := spawnStandaloneWorkers(cfg, db, rdb, lo, waRegistry, *workerCount, ctx, errCh)
	lo.Info("Workers started", "count", *workerCount)

	// Wait for shutdown signal or worker error. The signal channel is created
	// here (rather than via waitForShutdownSignal's blocking helper) so the
	// select can race it against errCh.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
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

	stopWorkers(workers, lo)
}
