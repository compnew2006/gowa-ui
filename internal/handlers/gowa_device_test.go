package handlers_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/handlers"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

// mockGowaDeviceAPI is a configurable GOWA REST API double. It routes by path
// so a single server can answer the login, status, and media-download calls the
// device handlers make. The QR image is served as raw bytes (as a real GOWA
// instance would) so the handler's base64-data-URI transformation is exercised.
type mockGowaDeviceAPI struct {
	*httptest.Server
	qrImage    []byte
	statusCode int // non-zero overrides every path with an error response
	qrLink     string
	appStatus  gowa.AppStatus
	lastQuery  string
	lastPath   string
}

func newMockGowaDeviceAPI() *mockGowaDeviceAPI {
	m := &mockGowaDeviceAPI{
		qrImage: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A}, // fake PNG header
		appStatus: gowa.AppStatus{
			IsConnected: true,
			IsLoggedIn:  true,
			DeviceID:    "dev-1",
			JID:         "16505551234@s.whatsapp.net",
		},
	}
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.lastPath = r.URL.Path
		m.lastQuery = r.URL.RawQuery

		if m.statusCode != 0 {
			w.WriteHeader(m.statusCode)
			_, _ = w.Write([]byte(`{"code":500,"message":"gowa down"}`))
			return
		}

		switch r.URL.Path {
		case "/app/login":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": map[string]any{
					"qr_link":     m.qrLink,
					"qr_duration": 20,
				},
			})
		case "/app/status":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"results": m.appStatus})
		case "/app/login-with-code":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": map[string]any{"pair_code": "12345678"},
			})
		case "/statics/qr/abc.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(m.qrImage)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	// GOWA returns an absolute URL for the QR image; point it back at this
	// mock so DownloadMedia fetches the raw bytes through the same server.
	m.qrLink = m.Server.URL + "/statics/qr/abc.png"
	return m
}

func (m *mockGowaDeviceAPI) close()      { m.Server.Close() }
func (m *mockGowaDeviceAPI) url() string { return m.Server.URL }

// newGowaDeviceApp wires an App whose registry resolves GOWA accounts to a real
// gowa.Client pointing at the mock server. The factory is registered per App so
// each test controls its own target URL.
func newGowaDeviceApp(t *testing.T, mock *mockGowaDeviceAPI) *handlers.App {
	t.Helper()
	app := newTestApp(t)
	whatsapp.RegisterGowaFactory(
		func(baseURL string) (string, string) { return "user", "pass" },
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(baseURL, username, password)
		},
	)
	meta := whatsapp.New(logf.New(logf.Opts{Level: logf.ErrorLevel}))
	app.WhatsApp = meta
	app.WARegistry = whatsapp.NewRegistry(meta, logf.New(logf.Opts{Level: logf.ErrorLevel}))
	return app
}

// createGowaAccountInDB inserts a real GOWA-backed WhatsAppAccount row so the
// handler's DB lookup, IsGowa() check, and provider resolution all run against
// real state (not a mocked object).
func createGowaAccountInDB(t *testing.T, app *handlers.App, orgID uuid.UUID, mockURL, deviceID string) *models.WhatsAppAccount {
	t.Helper()
	acc := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           "gowa-acc-" + deviceID + "-" + uuid.New().String()[:8],
		ProviderType:   "gowa",
		GowaBaseURL:    mockURL,
		GowaDeviceID:   deviceID,
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(acc).Error)
	return acc
}

// authedGowaRequest builds a GET request carrying the auth context ({id} path
// param, org+user in context) that the router would supply.
func authedGowaRequest(t *testing.T, orgID, accID uuid.UUID) *fastglue.Request {
	t.Helper()
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgID, uuid.New())
	testutil.SetPathParam(req, "id", accID.String())
	return req
}

// --- Tests ---

func TestApp_GowaDeviceStatus_ReturnsConnectionState(t *testing.T) {
	mock := newMockGowaDeviceAPI()
	defer mock.close()
	app := newGowaDeviceApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	acc := createGowaAccountInDB(t, app, org.ID, mock.url(), "dev-1")

	req := authedGowaRequest(t, org.ID, acc.ID)

	require.NoError(t, app.GowaDeviceStatus(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data map[string]any `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, true, resp.Data["is_connected"])
	assert.Equal(t, true, resp.Data["is_logged_in"])
	assert.Equal(t, "16505551234@s.whatsapp.net", resp.Data["jid"])
	assert.Equal(t, "dev-1", resp.Data["device_id"])
}

func TestApp_GowaDeviceStatus_GOWAErrorReturnsBadGateway(t *testing.T) {
	mock := newMockGowaDeviceAPI()
	mock.statusCode = http.StatusInternalServerError
	defer mock.close()
	app := newGowaDeviceApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	acc := createGowaAccountInDB(t, app, org.ID, mock.url(), "dev-1")

	req := authedGowaRequest(t, org.ID, acc.ID)

	require.NoError(t, app.GowaDeviceStatus(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadGateway, "Failed to get device status")
}

func TestApp_GowaLoginQR_ReturnsBase64DataURI(t *testing.T) {
	mock := newMockGowaDeviceAPI()
	defer mock.close()
	app := newGowaDeviceApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	acc := createGowaAccountInDB(t, app, org.ID, mock.url(), "dev-1")

	req := authedGowaRequest(t, org.ID, acc.ID)

	require.NoError(t, app.GowaLoginQR(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			QRLink     string `json:"qr_link"`
			QRDuration int    `json:"qr_duration"`
		} `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)

	// The handler must wrap the raw PNG bytes as a data URI so the browser
	// <img> tag renders without Basic Auth — the one piece of logic that is
	// the handler's, not GOWA's.
	require.NotEmpty(t, resp.Data.QRLink)
	const prefix = "data:image/png;base64,"
	require.True(t, len(resp.Data.QRLink) > len(prefix), "qr_link must be a data URI")
	decoded, err := base64.StdEncoding.DecodeString(resp.Data.QRLink[len(prefix):])
	require.NoError(t, err, "data URI payload must be valid base64")
	assert.Equal(t, mock.qrImage, decoded, "data URI must carry the raw QR image bytes")
	assert.Equal(t, 20, resp.Data.QRDuration)
}

func TestApp_GowaPairCode_ReturnsCodeFromGOWA(t *testing.T) {
	mock := newMockGowaDeviceAPI()
	defer mock.close()
	app := newGowaDeviceApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	acc := createGowaAccountInDB(t, app, org.ID, mock.url(), "dev-1")

	req := testutil.NewJSONRequest(t, map[string]any{"phone": "16505551234"})
	testutil.SetAuthContext(req, org.ID, uuid.New())
	testutil.SetPathParam(req, "id", acc.ID.String())

	require.NoError(t, app.GowaPairCode(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			PairCode string `json:"pair_code"`
		} `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, "12345678", resp.Data.PairCode)

	// The handler must forward the phone as a query param to GOWA.
	assert.Contains(t, mock.lastQuery, "phone=16505551234")
}

func TestApp_GowaPairCode_EmptyPhoneRejected(t *testing.T) {
	mock := newMockGowaDeviceAPI()
	defer mock.close()
	app := newGowaDeviceApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	acc := createGowaAccountInDB(t, app, org.ID, mock.url(), "dev-1")

	req := testutil.NewJSONRequest(t, map[string]any{"phone": ""})
	testutil.SetAuthContext(req, org.ID, uuid.New())
	testutil.SetPathParam(req, "id", acc.ID.String())

	require.NoError(t, app.GowaPairCode(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Phone number is required")
	// GOWA must never have been called for an invalid request.
	assert.Equal(t, "", mock.lastPath)
}

// TestApp_GowaHandler_RejectsNonGowaAccount verifies the type guard in
// resolveGowaAccount: a Meta account hitting a GOWA-only endpoint gets a 400,
// not a provider misfire. This is the assertion that stops a routing bug from
// silently calling Meta methods on a GOWA endpoint (or vice versa).
func TestApp_GowaHandler_RejectsNonGowaAccount(t *testing.T) {
	mock := newMockGowaDeviceAPI()
	defer mock.close()
	app := newGowaDeviceApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)

	acc := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "meta-acc-" + uuid.New().String()[:8],
		ProviderType:   "meta",
		PhoneID:        "phone-1",
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(acc).Error)

	req := authedGowaRequest(t, org.ID, acc.ID)

	require.NoError(t, app.GowaDeviceStatus(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "not a GOWA account")
	assert.Equal(t, "", mock.lastPath, "GOWA mock must not be hit for a Meta account")
}
