package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/handlers"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/require"
)

// --- Catalog Test Helpers ---

// mockCatalogServer creates a mock WhatsApp API server for catalog operations.
type mockCatalogServer struct {
	server *httptest.Server

	// Configurable responses
	nextCatalogID string
	nextProductID string
	returnError   bool
	errorMessage  string
}

func newMockCatalogServer() *mockCatalogServer {
	m := &mockCatalogServer{
		nextCatalogID: "meta-catalog-" + uuid.New().String()[:8],
		nextProductID: "meta-product-" + uuid.New().String()[:8],
	}

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"message": "Invalid access token",
					"code":    190,
				},
			})
			return
		}

		if m.returnError {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"message": m.errorMessage,
					"code":    100,
				},
			})
			return
		}

		switch r.Method {
		case http.MethodPost:
			// Handle catalog or product creation
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": m.nextCatalogID,
			})
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
			})
		case http.MethodGet:
			// List catalogs
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return m
}

func (m *mockCatalogServer) close() {
	m.server.Close()
}

// newCatalogTestApp creates an App instance for catalog testing with a mock WhatsApp server.
func newCatalogTestApp(t *testing.T, mockServer *mockCatalogServer) *handlers.App {
	t.Helper()

	log := testutil.NopLogger()
	waClient := whatsapp.NewWithBaseURL(log, mockServer.server.URL)

	return newTestApp(t, withWhatsApp(waClient))
}

// createTestCatalog creates a test catalog directly in the database.
func createTestCatalog(t *testing.T, app *handlers.App, orgID uuid.UUID, accountName, name string) *models.Catalog {
	t.Helper()

	catalog := &models.Catalog{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		WhatsAppAccount: accountName,
		MetaCatalogID:   "meta-catalog-" + uuid.New().String()[:8],
		Name:            name,
		IsActive:        true,
	}
	require.NoError(t, app.DB.Create(catalog).Error)
	return catalog
}

// createTestCatalogProduct creates a test catalog product directly in the database.
func createTestCatalogProduct(t *testing.T, app *handlers.App, orgID, catalogID uuid.UUID, name string, price int64) *models.CatalogProduct {
	t.Helper()

	product := &models.CatalogProduct{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		CatalogID:      catalogID,
		MetaProductID:  "meta-product-" + uuid.New().String()[:8],
		Name:           name,
		Description:    "Test product description",
		Price:          price,
		Currency:       "USD",
		URL:            "https://example.com/product",
		ImageURL:       "https://example.com/image.jpg",
		RetailerID:     "SKU-" + uuid.New().String()[:8],
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(product).Error)
	return product
}

// createCatalogTestAccount creates a WhatsApp account with predictable fields for catalog tests.
func createCatalogTestAccount(t *testing.T, app *handlers.App, orgID uuid.UUID) *models.WhatsAppAccount {
	t.Helper()

	account := &models.WhatsAppAccount{
		BaseModel:          models.BaseModel{ID: uuid.New()},
		OrganizationID:     orgID,
		Name:               "test-account-" + uuid.New().String()[:8],
		PhoneID:            "phone-" + uuid.New().String()[:8],
		BusinessID:         "business-" + uuid.New().String()[:8],
		AccessToken:        "test-token",
		WebhookVerifyToken: "webhook-token",
		APIVersion:         "v18.0",
		Status:             "active",
	}
	require.NoError(t, app.DB.Create(account).Error)
	return account
}

// --- ListCatalogs Tests ---

// --- CreateCatalog Tests ---

// --- GetCatalog Tests ---

// --- DeleteCatalog Tests ---

// --- ListCatalogProducts Tests ---

// --- CreateCatalogProduct Tests ---

// --- GetCatalogProduct Tests ---

// --- UpdateCatalogProduct Tests ---

// --- DeleteCatalogProduct Tests ---
