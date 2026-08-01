package main

import (
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/compnew2006/gowa-ui/internal/chatlifecycle"
	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/database"
	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/queue"
	"github.com/compnew2006/gowa-ui/internal/websocket"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// setupLogger builds the leveled logger used by both runServer (component
// "gowa-ui") and runWorker (component "gowa-ui-worker"). In development the
// logger defaults to debug level with color + caller; in production it
// downgrades to info and drops color/caller. This collapses the two logf.New
// blocks that were open-coded in both commands.
func setupLogger(environment, component string) logf.Logger {
	if environment == "production" {
		return logf.New(logf.Opts{
			Level:           logf.InfoLevel,
			TimestampFormat: "2006-01-02 15:04:05",
			DefaultFields:   []any{"app", component},
		})
	}
	return logf.New(logf.Opts{
		EnableColor:     true,
		Level:           logf.DebugLevel,
		EnableCaller:    true,
		TimestampFormat: "2006-01-02 15:04:05",
		DefaultFields:   []any{"app", component},
	})
}

// loadAndValidateConfig loads the TOML config and runs the four environment
// validations (JWT>=32 in production, empty-JWT warn, debug-in-prod warn,
// allowed-origins required in production). On fatal conditions it calls
// lo.Fatal (preserving the exact messages from the prior inline block).
func loadAndValidateConfig(path string, lo logf.Logger) *config.Config {
	cfg, err := config.Load(path)
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

	return cfg
}

// setupDB connects to PostgreSQL with the same fatal-on-error + "Connected to
// PostgreSQL" log that runServer/runWorker open-coded.
func setupDB(cfg *config.Config, lo logf.Logger) *gorm.DB {
	db, err := database.NewPostgres(&cfg.Database, cfg.App.Debug)
	if err != nil {
		lo.Fatal("Failed to connect to database", "error", err)
	}
	lo.Info("Connected to PostgreSQL")
	return db
}

// runMigrations runs schema migrations plus the GOWA webhook-secret backfill
// (FR-017). Called only from the server command's *migrate branch.
func runMigrations(db *gorm.DB, cfg *config.Config, lo logf.Logger) {
	if err := database.RunMigrationWithProgress(db, &cfg.DefaultAdmin); err != nil {
		lo.Fatal("Migration failed", "error", err)
	}
	// Backfill GOWA webhook secrets for accounts created without one (FR-017).
	// Ensures no GOWA account is left webhook-unprotected. Idempotent.
	if err := handlers.BackfillGowaWebhookSecrets(db, cfg, lo); err != nil {
		lo.Fatal("GOWA webhook secret backfill failed", "error", err)
	}
}

// setupRedis connects to Redis with fatal-on-error + log, matching the prior
// inline block in both commands.
func setupRedis(cfg *config.RedisConfig, lo logf.Logger) *redis.Client {
	rdb, err := database.NewRedis(cfg)
	if err != nil {
		lo.Fatal("Failed to connect to Redis", "error", err)
	}
	lo.Info("Connected to Redis")
	return rdb
}

// setupQueue constructs the Redis-backed job queue. The "Job queue
// initialized" log lives at the call site (runServer) since the worker command
// does not log this.
func setupQueue(rdb *redis.Client, lo logf.Logger) queue.Queue {
	return queue.NewRedisQueue(rdb, lo)
}

// setupHTTPClient returns the single production HTTP client (pooled, SSRF-safe).
// One definition lives in handlers.NewSharedHTTPClient and is reused here.
func setupHTTPClient() *http.Client {
	return handlers.NewSharedHTTPClient()
}

// setupWARegistry wires the GOWA provider registry with the production
// factory closures: per-instance Basic Auth credentials resolved from the DB
// (UI-managed) with a config-file fallback, and a gowa.New client constructor.
// This eliminates the verbatim duplication between runServer and runWorker.
// The factory is process-global (mutates pkg/whatsapp package state) — this
// helper registers the production resolver, so it must be called exactly once
// per process at startup of each command.
func setupWARegistry(db *gorm.DB, cfg *config.Config, lo logf.Logger) *whatsapp.Registry {
	return whatsapp.NewRegistryWithFactory(
		lo,
		func(baseURL string) (string, string) {
			return handlers.ResolveGowaCreds(db, cfg, baseURL)
		},
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(baseURL, username, password)
		},
	)
}

// setupWSHub constructs the WebSocket hub and starts its Run goroutine.
func setupWSHub(lo logf.Logger) *websocket.Hub {
	wsHub := websocket.NewHub(lo)
	go wsHub.Run()
	lo.Info("WebSocket hub started")
	return wsHub
}

// setupApp assembles the handlers.App struct, wires the ChatLifecycle service
// (production uses the real WS hub), and starts the campaign-stats subscriber
// (real-time WS updates from workers). The subscriber-start error is logged
// but non-fatal, exactly as the prior inline code did.
func setupApp(
	cfg *config.Config,
	db *gorm.DB,
	rdb *redis.Client,
	lo logf.Logger,
	waRegistry *whatsapp.Registry,
	wsHub *websocket.Hub,
	httpClient *http.Client,
	jobQueue queue.Queue,
) *handlers.App {
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

	// Chat-lifecycle state machine: claim/release/close/reopen/join/leave and
	// the audit + system-message + WS side effects they emit. The handlers in
	// chat_lifecycle.go are thin HTTP adapters over this service.
	app.ChatLifecycle = chatlifecycle.New(db, wsHub, lo)

	// Start campaign stats subscriber for real-time WebSocket updates from worker
	if err := app.StartCampaignStatsSubscriber(); err != nil {
		lo.Error("Failed to start campaign stats subscriber", "error", err)
	}

	return app
}

// setupHTTPServer builds the *fasthttp.Server with the exact timeouts,
// MaxRequestBodySize, 16KB read/write buffers, and Name: "Gowa-UI" used by
// the server command. The cookie-size comment is preserved.
func setupHTTPServer(handler fasthttp.RequestHandler, cfg *config.Config) *fasthttp.Server {
	return &fasthttp.Server{
		Handler:            handler,
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
		Name:            "Gowa-UI",
	}
}
