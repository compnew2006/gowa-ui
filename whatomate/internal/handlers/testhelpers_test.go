package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/audit"
	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/compnew2006/whatomate/test/testutil"
)

// appOption configures an App for testing.
type appOption func(*handlers.App)

// withQueue sets the queue on the test App.
func withQueue(q queue.Queue) appOption {
	return func(a *handlers.App) {
		a.Queue = q
	}
}

// withWhatsApp sets the WhatsApp client on the test App.
func withWhatsApp(wa *whatsapp.Client) appOption {
	return func(a *handlers.App) {
		a.WhatsApp = wa
	}
}

// withHTTPClient sets the HTTP client on the test App.
func withHTTPClient(client *http.Client) appOption {
	return func(a *handlers.App) {
		a.HTTPClient = client
	}
}

// withWSHub starts a real websocket.Hub on the test App so tests can observe broadcasts.
func withWSHub() appOption {
	return func(a *handlers.App) {
		a.WSHub = websocket.NewHub(testutil.NopLogger(), nil)
		go a.WSHub.Run()
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
			EncryptionKey: testutil.TestEncryptionKey,
			Environment:   "test",
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
		Audit:  audit.New(db, log),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}
