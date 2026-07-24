package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/config"
	"github.com/shridarpatil/whatomate/internal/handlers"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// fakeMetaServer stands up an httptest server that simulates Meta's Graph API for
// account validation flows. Each request increments hits[path] so tests can verify
// which endpoints were called. Override individual endpoint handlers via the maps.
type fakeMetaServer struct {
	mu          sync.Mutex
	hits        map[string]int
	server      *httptest.Server
	phoneFn     func(w http.ResponseWriter, r *http.Request)
	bizFn       func(w http.ResponseWriter, r *http.Request)
	listFn      func(w http.ResponseWriter, r *http.Request)
	subFn       func(w http.ResponseWriter, r *http.Request)
	wabaLimitFn func(w http.ResponseWriter, r *http.Request)
}

func newFakeMetaServer(t *testing.T) *fakeMetaServer {
	t.Helper()
	f := &fakeMetaServer{hits: make(map[string]int)}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.hits[r.URL.Path]++
		f.mu.Unlock()

		switch {
		case strings.Contains(r.URL.RawQuery, "whatsapp_business_manager_messaging_limit") && !strings.Contains(r.URL.RawQuery, "display_phone_number"):
			if f.wabaLimitFn != nil {
				f.wabaLimitFn(w, r)
				return
			}
			_, _ = w.Write([]byte(`{"whatsapp_business_manager_messaging_limit":"TIER_10K"}`))
		case strings.HasSuffix(r.URL.Path, "/subscribed_apps") && r.Method == http.MethodPost:
			if f.subFn != nil {
				f.subFn(w, r)
				return
			}
			_, _ = w.Write([]byte(`{"success": true}`))
		case strings.HasSuffix(r.URL.Path, "/phone_numbers"):
			if f.listFn != nil {
				f.listFn(w, r)
				return
			}
			// Default: return one phone whose ID matches the path's BusinessID parent — but
			// the simplest default is: caller must override.
			_, _ = w.Write([]byte(`{"data":[]}`))
		case strings.Contains(r.URL.Path, "fields=id,name") || strings.Contains(r.URL.RawQuery, "fields=id,name"):
			if f.bizFn != nil {
				f.bizFn(w, r)
				return
			}
			_, _ = w.Write([]byte(`{"id":"biz","name":"Biz Co"}`))
		default:
			// Catch-all for the phone-number details endpoint.
			if f.phoneFn != nil {
				f.phoneFn(w, r)
				return
			}
			_, _ = w.Write([]byte(`{"display_phone_number":"+1234567890","verified_name":"Test","account_mode":"LIVE","code_verification_status":"VERIFIED","quality_rating":"GREEN","messaging_limit_tier":"TIER_250"}`))
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeMetaServer) URL() string { return f.server.URL }

// newAppWithMeta returns an App pointing at the fake Meta server.
func newAppWithMeta(t *testing.T, meta *fakeMetaServer) *handlers.App {
	t.Helper()
	app := newTestApp(t)
	app.WhatsApp = whatsapp.NewWithBaseURL(logf.New(logf.Opts{Level: logf.ErrorLevel}), meta.URL())
	app.Config = &config.Config{
		JWT: config.JWTConfig{
			Secret:            testutil.TestJWTSecret,
			AccessExpiryMins:  15,
			RefreshExpiryDays: 7,
		},
		WhatsApp: config.WhatsAppConfig{BaseURL: meta.URL(), APIVersion: "v18.0"},
	}
	return app
}

// createTestAccountForValidation inserts a WhatsApp account directly with plaintext credentials
// (encryption is a no-op when EncryptionKey == "").
func createTestAccountForValidation(t *testing.T, db *gorm.DB, orgID uuid.UUID, phoneID, businessID string) *models.WhatsAppAccount {
	t.Helper()
	acc := &models.WhatsAppAccount{
		BaseModel:          models.BaseModel{ID: uuid.New()},
		OrganizationID:     orgID,
		Name:               "acc-" + uuid.New().String()[:6],
		PhoneID:            phoneID,
		BusinessID:         businessID,
		AccessToken:        "plain-test-token",
		WebhookVerifyToken: "verify-token",
		APIVersion:         "v18.0",
		Status:             "active",
	}
	require.NoError(t, db.Create(acc).Error)
	return acc
}

// --- TestAccountConnection ---

// --- SubscribeApp ---

// --- CreateAccount: encryption + defaulting + IsDefaultIncoming flip ---

func TestApp_CreateAccount_AccessTokenEncryptedAtRest(t *testing.T) {
	app := newTestApp(t)
	// Set a real encryption key so Encrypt is not a no-op.
	app.Config = &config.Config{
		App: config.AppConfig{EncryptionKey: "this-is-a-32-character-test-key-XX"},
		JWT: config.JWTConfig{Secret: testutil.TestJWTSecret, AccessExpiryMins: 15, RefreshExpiryDays: 7},
	}
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"name":         "secure",
		"phone_id":     "phone-z",
		"business_id":  "biz-z",
		"access_token": "PLAIN-TOKEN",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	require.NoError(t, app.CreateAccount(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var stored models.WhatsAppAccount
	require.NoError(t, app.DB.Where("phone_id = ?", "phone-z").First(&stored).Error)
	assert.NotEqual(t, "PLAIN-TOKEN", stored.AccessToken, "access token must be encrypted at rest")
	assert.True(t, strings.HasPrefix(stored.AccessToken, "enc:"), "stored token should carry the enc: prefix")
}

func TestApp_CreateAccount_DefaultAPIVersionAppliedWhenMissing(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"name":         "no-api-ver",
		"phone_id":     "p",
		"business_id":  "b",
		"access_token": "tok",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	require.NoError(t, app.CreateAccount(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var stored models.WhatsAppAccount
	require.NoError(t, app.DB.Where("name = ?", "no-api-ver").First(&stored).Error)
	assert.Equal(t, "v21.0", stored.APIVersion)
}

func TestApp_CreateAccount_VerifyTokenAutoGeneratedWhenEmpty(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"name":         "auto-vt",
		"phone_id":     "p",
		"business_id":  "b",
		"access_token": "tok",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	require.NoError(t, app.CreateAccount(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var stored models.WhatsAppAccount
	require.NoError(t, app.DB.Where("name = ?", "auto-vt").First(&stored).Error)
	assert.NotEmpty(t, stored.WebhookVerifyToken, "verify token must be auto-generated when not supplied")
}

func TestApp_CreateAccount_DefaultIncomingFlipsExisting(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	// Existing default incoming.
	existing := createTestAccountForValidation(t, app.DB, org.ID, "p1", "b1")
	require.NoError(t, app.DB.Model(existing).Update("is_default_incoming", true).Error)

	req := testutil.NewJSONRequest(t, map[string]any{
		"name":                "new-default",
		"phone_id":            "p2",
		"business_id":         "b2",
		"access_token":        "tok",
		"is_default_incoming": true,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	require.NoError(t, app.CreateAccount(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var prev models.WhatsAppAccount
	require.NoError(t, app.DB.Where("id = ?", existing.ID).First(&prev).Error)
	assert.False(t, prev.IsDefaultIncoming, "existing default-incoming must be unset when a new default is created")

	var fresh models.WhatsAppAccount
	require.NoError(t, app.DB.Where("name = ?", "new-default").First(&fresh).Error)
	assert.True(t, fresh.IsDefaultIncoming)
}
