package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/shridarpatil/gowa-ui/internal/chatlifecycle"
	"github.com/shridarpatil/gowa-ui/internal/config"
	"github.com/shridarpatil/gowa-ui/internal/handlers"
	"github.com/shridarpatil/gowa-ui/internal/queue"
	"github.com/shridarpatil/gowa-ui/test/testutil"
)

// appOption configures an App for testing.
type appOption func(*handlers.App)

// withQueue sets the queue on the test App.
func withQueue(q queue.Queue) appOption {
	return func(a *handlers.App) {
		a.Queue = q
	}
}

// withHTTPClient sets the HTTP client on the test App.
func withHTTPClient(client *http.Client) appOption {
	return func(a *handlers.App) {
		a.HTTPClient = client
	}
}

// newTestApp creates an App instance for testing with a test database, Redis, and default config.
// Skips the test if TEST_REDIS_URL is not set.
func newTestApp(t *testing.T, opts ...appOption) *handlers.App {
	t.Helper()

	db := testutil.SetupTestDB(t)
	log := testutil.NopLogger()

	redisClient := testutil.SetupTestRedis(t)
	if redisClient == nil {
		t.Skip("TEST_REDIS_URL not set, skipping test")
	}

	cfg := &config.Config{
		App: config.AppConfig{
			EncryptionKey: "test-encryption-key-for-handlers-longer-than-32-chars",
		},
		JWT: config.JWTConfig{
			Secret:            testutil.TestJWTSecret,
			AccessExpiryMins:  15,
			RefreshExpiryDays: 7,
		},
	}

	app := &handlers.App{
		Config: cfg,
		DB:     db,
		Log:    log,
		Redis:  redisClient,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	// Chat-lifecycle service — mirrors cmd/gowa-ui/main.go wiring so the
	// ReleaseChat/ClaimChat/etc handlers have a real service to delegate to.
	// wsHub is nil here (WS is verified at integration level, not in unit tests).
	app.ChatLifecycle = chatlifecycle.New(db, nil, log)

	for _, opt := range opts {
		opt(app)
	}

	return app
}
