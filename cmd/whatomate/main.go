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

	"github.com/redis/go-redis/v9"
	"github.com/shridarpatil/whatomate/internal/assignment"
	"github.com/shridarpatil/whatomate/internal/chatlifecycle"
	"github.com/shridarpatil/whatomate/internal/config"
	"github.com/shridarpatil/whatomate/internal/database"
	"github.com/shridarpatil/whatomate/internal/frontend"
	"github.com/shridarpatil/whatomate/internal/handlers"
	"github.com/shridarpatil/whatomate/internal/middleware"
	"github.com/shridarpatil/whatomate/internal/queue"
	"github.com/shridarpatil/whatomate/internal/websocket"
	"github.com/shridarpatil/whatomate/internal/worker"
	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
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
  whatomate server                     # API + 1 embedded worker
  whatomate server -workers 0          # API only (no workers)
  whatomate server -workers 4          # API + 4 embedded workers
  whatomate server -migrate            # Run migrations and start server
  whatomate worker -workers 4          # 4 workers only (no API)

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

	// Validate JWT secret
	if cfg.App.Environment == "production" && len(cfg.JWT.Secret) < 32 {
		lo.Fatal("JWT secret must be at least 32 characters in production")
	}
	if cfg.JWT.Secret == "" {
		lo.Warn("JWT secret is empty, using a random secret (tokens will not persist across restarts)")
	}

	// Warn if debug mode is on in production
	if cfg.App.Environment == "production" && cfg.App.Debug {
		lo.Warn("Debug mode is enabled in production! This may expose sensitive information.")
	}

	// Require explicit CORS origins in production
	if cfg.App.Environment == "production" && cfg.Server.AllowedOrigins == "" {
		lo.Fatal("server.allowed_origins must be set in production (e.g. \"https://app.example.com\")")
	}

	// Set log level based on environment
	if cfg.App.Environment == "production" {
		lo = logf.New(logf.Opts{
			Level:           logf.InfoLevel,
			TimestampFormat: "2006-01-02 15:04:05",
			DefaultFields:   []any{"app", "whatomate"},
		})
	}

	// Connect to PostgreSQL
	db, err := database.NewPostgres(&cfg.Database, cfg.App.Debug)
	if err != nil {
		lo.Fatal("Failed to connect to database", "error", err)
	}
	lo.Info("Connected to PostgreSQL")

	// Run migrations if requested
	if *migrate {
		if err := database.RunMigrationWithProgress(db, &cfg.DefaultAdmin); err != nil {
			lo.Fatal("Migration failed", "error", err)
		}
		// Backfill v2 graph for any legacy chatbot flow still on Steps[].
		// Idempotent — re-running is a no-op once every row is converted.
		if err := handlers.BackfillChatbotFlowGraph(db, lo); err != nil {
			lo.Fatal("Chatbot flow graph backfill failed", "error", err)
		}
		// Backfill GOWA webhook secrets for accounts created without one (FR-017).
		// Ensures no GOWA account is left webhook-unprotected. Idempotent.
		if err := handlers.BackfillGowaWebhookSecrets(db, cfg, lo); err != nil {
			lo.Fatal("GOWA webhook secret backfill failed", "error", err)
		}
	}

	// Connect to Redis
	rdb, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		lo.Fatal("Failed to connect to Redis", "error", err)
	}
	lo.Info("Connected to Redis")

	// Initialize job queue
	jobQueue := queue.NewRedisQueue(rdb, lo)
	lo.Info("Job queue initialized")

	// Initialize Fastglue
	g := fastglue.NewGlue()

	// Initialize the GOWA provider registry. The factory resolves per-instance
	// Basic Auth credentials from the DB (UI-managed) with a config-file fallback.
	whatsapp.RegisterGowaFactory(
		func(baseURL string) (string, string) {
			return handlers.ResolveGowaCreds(db, cfg, baseURL)
		},
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(baseURL, username, password)
		},
	)
	waRegistry := whatsapp.NewRegistry(lo)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(lo)
	go wsHub.Run()
	lo.Info("WebSocket hub started")

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
		Config:     cfg,
		DB:         db,
		Redis:      rdb,
		Log:        lo,
		WARegistry: waRegistry,
		WSHub:      wsHub,
		Queue:      jobQueue,
		HTTPClient: httpClient,
	}

	// Initialize shared assignment engine (used by chat transfers)
	assigner := assignment.New(db, rdb, lo)
	app.Assigner = assigner

	// Chat-lifecycle state machine: claim/release/close/reopen/join/leave and
	// the audit + system-message + WS side effects they emit. The handlers in
	// chat_lifecycle.go are thin HTTP adapters over this service.
	app.ChatLifecycle = chatlifecycle.New(db, wsHub, lo)

	// Start campaign stats subscriber for real-time WebSocket updates from worker
	if err := app.StartCampaignStatsSubscriber(); err != nil {
		lo.Error("Failed to start campaign stats subscriber", "error", err)
	}

	// Parse allowed origins for CORS
	allowedOrigins := middleware.ParseAllowedOrigins(cfg.Server.AllowedOrigins)

	// Setup middleware (CORS is handled by corsWrapper at fasthttp level)
	g.Before(middleware.SecurityHeaders())
	g.Before(middleware.RequestLogger(lo))
	g.Before(middleware.Recovery(lo))
	g.Before(middleware.CSRFProtection())

	// Setup routes
	setupRoutes(g, app, lo, cfg.Server.BasePath, rdb, cfg)

	// Create server with CORS wrapper
	server := &fasthttp.Server{
		Handler:            corsWrapper(g.Handler(), allowedOrigins),
		ReadTimeout:        time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:       time.Duration(cfg.Server.WriteTimeout) * time.Second,
		MaxRequestBodySize: 15 * 1024 * 1024,
		// fasthttp's default ReadBufferSize is 4 KB. Cookie-based auth stores the
		// access JWT, refresh JWT, and CSRF token in cookies — together these can
		// exceed 4 KB, which fasthttp rejects with HTTP 431 (Request Header
		// Fields Too Large). 16 KB comfortably accommodates JWT cookies while
		// bounding per-connection memory.
		ReadBufferSize:  16 * 1024,
		WriteBufferSize: 16 * 1024,
		Name:            "Whatomate",
	}

	// Start server in goroutine
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	go func() {
		lo.Info("Server listening", "address", addr)
		if err := server.ListenAndServe(addr); err != nil {
			lo.Fatal("Server failed", "error", err)
		}
	}()

	// Start SLA processor (runs every minute)
	slaProcessor := handlers.NewSLAProcessor(app, time.Minute)
	slaCtx, slaCancel := context.WithCancel(context.Background())
	go slaProcessor.Start(slaCtx)
	lo.Info("SLA processor started")

	// Start daily chat-reset processor (polls every minute, resets assigned
	// chats to pending per account schedule).
	chatResetProcessor := handlers.NewChatResetProcessor(app, time.Minute)
	chatResetCtx, chatResetCancel := context.WithCancel(context.Background())
	go chatResetProcessor.Start(chatResetCtx)
	lo.Info("Chat reset processor started")

	// Start scheduled-message processor (polls every minute, fires due
	// scheduled messages through the unified sender).
	scheduledMsgProcessor := handlers.NewScheduledMessageProcessor(app, time.Minute)
	scheduledMsgCtx, scheduledMsgCancel := context.WithCancel(context.Background())
	go scheduledMsgProcessor.Start(scheduledMsgCtx)
	lo.Info("Scheduled message processor started")

	// Start GOWA history-sync processor (initial pass at startup, then periodic
	// re-sync). GOWA syncs message history itself but never replays it via
	// webhook, so this pulls synced history into the messages table
	// automatically. A device "connected" webhook also triggers an immediate
	// sync; the per-account cooldown keeps overlapping triggers cheap.
	gowaHistoryProcessor := handlers.NewGowaHistorySyncProcessor(app, 15*time.Minute)
	gowaHistoryCtx, gowaHistoryCancel := context.WithCancel(context.Background())
	go gowaHistoryProcessor.Start(gowaHistoryCtx)
	lo.Info("GOWA history sync processor started")

	// Start embedded workers
	var workers []*worker.Worker
	var workerCancel context.CancelFunc
	if *numWorkers > 0 {
		var workerCtx context.Context
		workerCtx, workerCancel = context.WithCancel(context.Background())

		for i := 0; i < *numWorkers; i++ {
			w, err := worker.New(cfg, db, rdb, lo, waRegistry)
			if err != nil {
				lo.Fatal("Failed to create worker", "error", err, "worker_num", i+1)
			}
			workers = append(workers, w)

			workerNum := i + 1
			go func() {
				lo.Info("Worker started", "worker_num", workerNum)
				if err := w.Run(workerCtx); err != nil && err != context.Canceled {
					lo.Error("Worker error", "error", err, "worker_num", workerNum)
				}
			}()
		}
		lo.Info("Embedded workers started", "count", *numWorkers)
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

	// Stop SLA processor
	lo.Info("Stopping SLA processor...")
	slaCancel()
	slaProcessor.Stop()
	lo.Info("SLA processor stopped")

	// Stop chat reset processor
	lo.Info("Stopping chat reset processor...")
	chatResetCancel()
	chatResetProcessor.Stop()
	lo.Info("Chat reset processor stopped")

	// Stop scheduled message processor
	lo.Info("Stopping scheduled message processor...")
	scheduledMsgCancel()
	scheduledMsgProcessor.Stop()
	lo.Info("Scheduled message processor stopped")

	// Stop GOWA history sync processor
	lo.Info("Stopping GOWA history sync processor...")
	gowaHistoryCancel()
	gowaHistoryProcessor.Stop()
	lo.Info("GOWA history sync processor stopped")

	// Stop workers first
	if workerCancel != nil {
		lo.Info("Stopping workers...", "count", len(workers))
		workerCancel()
		for _, w := range workers {
			_ = w.Close()
		}
		lo.Info("Workers stopped")
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

	// Initialize the GOWA provider registry. The factory resolves per-instance
	// Basic Auth credentials from the DB (UI-managed) with a config-file fallback.
	whatsapp.RegisterGowaFactory(
		func(baseURL string) (string, string) {
			return handlers.ResolveGowaCreds(db, cfg, baseURL)
		},
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(baseURL, username, password)
		},
	)
	waRegistry := whatsapp.NewRegistry(lo)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Create and run workers
	workers := make([]*worker.Worker, *workerCount)
	errCh := make(chan error, *workerCount)

	for i := 0; i < *workerCount; i++ {
		w, err := worker.New(cfg, db, rdb, lo, waRegistry)
		if err != nil {
			lo.Fatal("Failed to create worker", "error", err, "worker_num", i+1)
		}
		workers[i] = w

		go func(workerNum int) {
			lo.Info("Worker started", "worker_num", workerNum)
			errCh <- w.Run(ctx)
		}(i + 1)
	}

	lo.Info("Workers started", "count", *workerCount)

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
	for _, w := range workers {
		if w != nil {
			if err := w.Close(); err != nil {
				lo.Error("Error closing worker", "error", err)
			}
		}
	}
	lo.Info("Workers stopped")
}

// ============================================================================
// ROUTES
// ============================================================================

func setupRoutes(g *fastglue.Fastglue, app *handlers.App, lo logf.Logger, basePath string, rdb *redis.Client, cfg *config.Config) {
	// Health check
	g.GET("/health", app.HealthCheck)
	g.GET("/ready", app.ReadyCheck)

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
	g.POST("/api/auth/logout", app.Logout)
	g.POST("/api/auth/switch-org", app.SwitchOrg)
	g.GET("/api/auth/ws-token", app.GetWSToken)

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

	// GOWA webhook routes (public — HMAC verified in handler)
	// Rate-limited per-IP to prevent brute-force signature attempts (FR-020).
	gowaWebhookRL := middleware.RateLimitOpts{
		Redis:      rdb,
		Log:        lo,
		Max:        100,
		Window:     time.Minute,
		KeyPrefix:  "gowa_webhook",
		TrustProxy: cfg.RateLimit.TrustProxy,
	}
	// Single endpoint: resolves device from the payload's device_id field.
	g.POST(cfg.GOWA.WebhookPath, withRateLimit(app.GowaWebhookHandler, gowaWebhookRL))
	// Per-device endpoint: GOWA v8.10.0 supports device-specific webhook URLs.
	// The path device_id overrides the payload's device_id field.
	g.POST(cfg.GOWA.WebhookPath+"/{device_id}", withRateLimit(app.GowaWebhookHandlerDevice, gowaWebhookRL))

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
		// Skip auth for public routes.
		// GOWA per-device webhooks use a prefix match since the path includes
		// a dynamic device_id segment (e.g. /api/gowa/webhook/628123@s.whatsapp.net).
		if path == "/health" || path == "/ready" ||
			path == "/api/auth/login" || path == "/api/auth/register" || path == "/api/auth/refresh" ||
			path == "/api/auth/logout" || path == "/ws" ||
			path == "/api/gowa/webhook" || strings.HasPrefix(path, "/api/gowa/webhook/") {
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

	// Global rate limit on all /api/ routes, keyed by user ID (or IP if unauthenticated).
	// Runs after auth so the user identity is available.
	if cfg.RateLimit.Enabled {
		apiMax := cfg.RateLimit.APIMaxRequests
		if apiMax == 0 {
			apiMax = 200
		}
		apiWindow := cfg.RateLimit.APIWindowSeconds
		if apiWindow == 0 {
			apiWindow = 60
		}
		apiRL := middleware.UserAwareRateLimit(middleware.RateLimitOpts{
			Redis:      rdb,
			Log:        lo,
			Max:        apiMax,
			Window:     time.Duration(apiWindow) * time.Second,
			KeyPrefix:  "api_global",
			TrustProxy: cfg.RateLimit.TrustProxy,
		})
		g.Before(func(r *fastglue.Request) *fastglue.Request {
			path := string(r.RequestCtx.Path())
			if len(path) > 4 && path[:4] == "/api" {
				return apiRL(r)
			}
			return r
		})
	}

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
	g.PUT("/api/me/password", app.ChangePassword)
	g.PUT("/api/me/availability", app.UpdateAvailability)
	g.GET("/api/me/organizations", app.ListMyOrganizations)

	// User Management (admin only - enforced by middleware)
	g.GET("/api/users", app.ListUsers)
	g.POST("/api/users", app.CreateUser)
	g.GET("/api/users/{id}", app.GetUser)
	g.PUT("/api/users/{id}", app.UpdateUser)
	g.DELETE("/api/users/{id}", app.DeleteUser)

	// Roles & Permissions (admin only - enforced by middleware)
	g.GET("/api/roles", app.ListRoles)
	g.POST("/api/roles", app.CreateRole)
	g.GET("/api/roles/{id}", app.GetRole)
	g.PUT("/api/roles/{id}", app.UpdateRole)
	g.DELETE("/api/roles/{id}", app.DeleteRole)
	g.GET("/api/permissions", app.ListPermissions)

	// API Keys (admin only - enforced by middleware)
	g.GET("/api/api-keys", app.ListAPIKeys)
	g.GET("/api/api-keys/{id}", app.GetAPIKey)
	g.POST("/api/api-keys", app.CreateAPIKey)
	g.PUT("/api/api-keys/{id}", app.UpdateAPIKey)
	g.DELETE("/api/api-keys/{id}", app.DeleteAPIKey)

	// Accounts
	g.GET("/api/accounts", app.ListAccounts)
	g.POST("/api/accounts", app.CreateAccount)
	g.GET("/api/accounts/{id}", app.GetAccount)
	g.PUT("/api/accounts/{id}", app.UpdateAccount)
	g.DELETE("/api/accounts/{id}", app.DeleteAccount)

	// Per-account chat-close rating (settings live on the WhatsApp account)
	g.GET("/api/accounts/{id}/close-rating", app.GetCloseRatingSettings)
	g.PUT("/api/accounts/{id}/close-rating", app.UpdateCloseRatingSettings)
	g.GET("/api/accounts/{id}/close-rating/stats", app.GetCloseRatingStats)

	// Per-account call auto-reject (settings live on the WhatsApp account)
	g.GET("/api/accounts/{id}/call-auto-reject", app.GetCallAutoRejectSettings)
	g.PUT("/api/accounts/{id}/call-auto-reject", app.UpdateCallAutoRejectSettings)

	// Per-account daily assigned-chat reset schedule (settings live on the WhatsApp account)
	g.GET("/api/accounts/{id}/daily-reset", app.GetChatResetSettings)
	g.PUT("/api/accounts/{id}/daily-reset", app.UpdateChatResetSettings)

	// GOWA device management (QR code, pair code, connection status)
	g.POST("/api/accounts/{id}/gowa/pair-code", app.GowaPairCode)

	// GOWA instance management (multi-instance dropdown + device provisioning)
	g.GET("/api/gowa/instances", app.GowaInstances)
	g.POST("/api/gowa/create-device", app.GowaCreateDevice)

	// GOWA servers (DB-managed instances + per-instance device management)
	g.GET("/api/gowa/servers", app.ListGowaInstances)
	g.POST("/api/gowa/servers", app.CreateGowaInstance)
	g.GET("/api/gowa/servers/{id}", app.GetGowaInstance)
	g.PUT("/api/gowa/servers/{id}", app.UpdateGowaInstance)
	g.DELETE("/api/gowa/servers/{id}", app.DeleteGowaInstance)

	// Devices within a DB-managed GOWA server
	g.GET("/api/gowa/servers/{id}/devices", app.ListGowaInstanceDevices)
	g.POST("/api/gowa/servers/{id}/devices", app.CreateGowaInstanceDevice)
	g.DELETE("/api/gowa/servers/{id}/devices/{deviceId}", app.DeleteGowaInstanceDevice)
	g.GET("/api/gowa/servers/{id}/devices/{deviceId}/qr", app.GowaInstanceDeviceQR)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/pair-code", app.GowaInstanceDevicePairCode)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/logout", app.GowaInstanceDeviceLogout)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/reconnect", app.GowaInstanceDeviceReconnect)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/sync", app.SyncGowaInstanceDevice)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/sync-contacts", app.SyncGowaInstanceDeviceContacts)
	g.POST("/api/gowa/servers/{id}/devices/{deviceId}/sync-messages", app.SyncGowaInstanceMessages)
	g.GET("/api/gowa/servers/{id}/devices/{deviceId}/webhook", app.GetGowaInstanceDeviceWebhook)
	g.PUT("/api/gowa/servers/{id}/devices/{deviceId}/webhook", app.SetGowaInstanceDeviceWebhook)

	// Contacts
	g.GET("/api/contacts", app.ListContacts)
	g.POST("/api/contacts", app.CreateContact)
	g.GET("/api/contacts/{id}", app.GetContact)
	g.GET("/api/contacts/{id}/avatar", app.RefreshContactAvatar)
	g.PUT("/api/contacts/{id}", app.UpdateContact)
	g.DELETE("/api/contacts/{id}", app.DeleteContact)
	g.PUT("/api/contacts/{id}/assign", app.AssignContact)
	g.PUT("/api/contacts/{id}/tags", app.UpdateContactTags)
	g.GET("/api/contacts/{id}/session-data", app.GetContactSessionData)

	// Scheduled messages
	g.POST("/api/contacts/{id}/scheduled-messages", app.CreateScheduledMessage)
	g.GET("/api/contacts/{id}/scheduled-messages", app.ListContactScheduledMessages)
	g.GET("/api/scheduled-messages", app.ListScheduledMessages)
	g.PUT("/api/scheduled-messages/{id}", app.UpdateScheduledMessage)
	g.DELETE("/api/scheduled-messages/{id}", app.CancelScheduledMessage)

	// Chat Lifecycle
	g.PUT("/api/contacts/{id}/claim", app.ClaimChat)
	g.PUT("/api/contacts/{id}/release", app.ReleaseChat)
	g.POST("/api/contacts/bulk-release", app.BulkReleaseChats)
	g.PUT("/api/contacts/{id}/close", app.CloseChat)
	g.PUT("/api/contacts/{id}/reopen", app.ReopenChat)
	g.POST("/api/contacts/{id}/join", app.JoinChat)
	g.DELETE("/api/contacts/{id}/join", app.LeaveChat)
	g.DELETE("/api/contacts/{id}/collaborators/{user_id}", app.RemoveCollaborator)
	g.POST("/api/contacts/{id}/collaborators/{user_id}", app.InviteCollaborator)

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
	g.POST("/api/contacts/{id}/messages", app.SendMessage)
	g.POST("/api/contacts/{id}/mark-read", app.MarkContactRead)
	g.POST("/api/contacts/{id}/messages/{message_id}/reaction", app.SendReaction)
	g.POST("/api/contacts/{id}/messages/{message_id}/revoke", app.RevokeMessage)
	g.POST("/api/contacts/{id}/typing", app.SendTypingIndicator)
	g.POST("/api/messages", app.SendMessage) // Legacy route
	g.POST("/api/messages/template", app.SendTemplateMessage)
	g.POST("/api/messages/media", app.SendMediaMessage)
	g.PUT("/api/messages/{id}/read", app.MarkMessageRead)

	// Conversation Notes
	g.GET("/api/contacts/{id}/notes", app.ListConversationNotes)
	g.POST("/api/contacts/{id}/notes", app.CreateConversationNote)
	g.PUT("/api/contacts/{id}/notes/{note_id}", app.UpdateConversationNote)
	g.DELETE("/api/contacts/{id}/notes/{note_id}", app.DeleteConversationNote)

	// Media (serves media files for messages, auth-protected)
	g.GET("/api/media/{message_id}", app.ServeMedia)
	// Media burst download — zips the media of the given message IDs together
	g.GET("/api/media/zip", app.ServeMediaZip)
	// Re-download a message's media from its provider (e.g. after a failed fetch)
	g.POST("/api/media/{message_id}/redownload", app.RedownloadMedia)

	// Templates (local blueprints — full CRUD, no remote sync)
	g.GET("/api/templates", app.ListTemplates)
	g.POST("/api/templates", app.CreateTemplate)
	g.GET("/api/templates/{id}", app.GetTemplate)
	g.PUT("/api/templates/{id}", app.UpdateTemplate)
	g.DELETE("/api/templates/{id}", app.DeleteTemplate)

	// Bulk Campaigns
	g.GET("/api/campaigns", app.ListCampaigns)
	g.POST("/api/campaigns", app.CreateCampaign)
	g.GET("/api/campaigns/{id}", app.GetCampaign)
	g.PUT("/api/campaigns/{id}", app.UpdateCampaign)
	g.DELETE("/api/campaigns/{id}", app.DeleteCampaign)
	g.POST("/api/campaigns/{id}/start", app.StartCampaign)
	g.POST("/api/campaigns/{id}/pause", app.PauseCampaign)
	g.POST("/api/campaigns/{id}/cancel", app.CancelCampaign)
	g.POST("/api/campaigns/{id}/retry-failed", app.RetryFailed)
	g.GET("/api/campaigns/{id}/progress", app.GetCampaign)
	g.POST("/api/campaigns/{id}/recipients/import", app.ImportRecipients)
	g.GET("/api/campaigns/{id}/recipients", app.GetCampaignRecipients)
	g.DELETE("/api/campaigns/{id}/recipients/{recipientId}", app.DeleteCampaignRecipient)
	g.POST("/api/campaigns/{id}/media", app.UploadCampaignMedia)
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

	// Audit Logs
	g.GET("/api/audit-logs", app.ListAuditLogs)
	g.GET("/api/audit-logs/{id}", app.GetAuditLog)

	// Canned Responses
	g.GET("/api/canned-responses", app.ListCannedResponses)
	g.POST("/api/canned-responses", app.CreateCannedResponse)
	g.GET("/api/canned-responses/{id}", app.GetCannedResponse)
	g.PUT("/api/canned-responses/{id}", app.UpdateCannedResponse)
	g.DELETE("/api/canned-responses/{id}", app.DeleteCannedResponse)
	g.POST("/api/canned-responses/{id}/use", app.IncrementCannedResponseUsage)

	// Sessions (admin/debug)

	// Analytics
	g.GET("/api/analytics/agents", app.GetAgentAnalytics)

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

	// Organizations
	g.GET("/api/organizations", app.ListOrganizations)
	g.POST("/api/organizations", app.CreateOrganization)
	g.POST("/api/organizations/members", app.AddOrganizationMember)

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

// corsWrapper wraps a handler with CORS support at the fasthttp level.
// This ensures CORS headers are set even for auto-handled OPTIONS requests.
func corsWrapper(next fasthttp.RequestHandler, allowedOrigins map[string]bool) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		origin := string(ctx.Request.Header.Peek("Origin"))

		if origin != "" && middleware.IsOriginAllowed(origin, allowedOrigins) {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
			ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		} else if len(allowedOrigins) == 0 && origin != "" {
			// Development: no whitelist configured
			ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
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
