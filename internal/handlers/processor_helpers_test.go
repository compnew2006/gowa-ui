package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/compnew2006/gowa-ui/test/testutil"
)

// newProcessorTestApp creates a minimal App suitable for incoming-message
// processor tests. It connects to the test database and Redis, provides a mock
// GOWA server that accepts all sends, and uses a no-op logger.
func newProcessorTestApp(t *testing.T) *App {
	t.Helper()
	db := testutil.SetupTestDB(t)
	log := testutil.NopLogger()

	// Mock GOWA API server that accepts all requests.
	waServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    "SUCCESS",
			"message": "Success",
			"results": map[string]any{"message_id": "wamid.mock_" + uuid.New().String()[:8], "status": "ok"},
		})
	}))
	t.Cleanup(waServer.Close)

	// Route every account through the mock server, regardless of its base URL.
	whatsapp.RegisterGowaFactory(
		func(baseURL string) (string, string) { return "", "" },
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(waServer.URL, username, password)
		},
	)

	app := &App{
		DB:         db,
		Log:        log,
		WARegistry: whatsapp.NewRegistry(log),
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
	if rdb := testutil.SetupTestRedis(t); rdb != nil {
		app.Redis = rdb
	}
	return app
}

// createProcessorTestOrg creates an organization and WhatsApp account for processor tests.
func createProcessorTestOrg(t *testing.T, app *App) (*models.Organization, *models.WhatsAppAccount) {
	t.Helper()
	org := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	return org, account
}
