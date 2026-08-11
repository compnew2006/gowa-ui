package handlers_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// statusMockGowa is a minimal GOWA mock that records the last request's phone
// field (the field that must equal "status@broadcast"). It reuses the same
// response envelope shape as newMockGowaServer so gowa.doJSON/doMultipart parse
// it correctly.
type statusMockGowa struct {
	server   *httptest.Server
	mu       sync.Mutex
	lastPhone string
	lastPath  string
	fail      bool
}

func newStatusMockGowa() *statusMockGowa {
	m := &statusMockGowa{}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		phone := r.FormValue("phone")
		m.mu.Lock()
		m.lastPhone = phone
		m.lastPath = r.URL.Path
		fail := m.fail
		m.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if fail {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    "BAD_REQUEST",
				"message": "status rejected",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    "SUCCESS",
			"message": "Success",
			"results": map[string]any{"message_id": "status-msg-123", "status": "ok"},
		})
	}))
	return m
}

func (m *statusMockGowa) close() { m.server.Close() }

func (m *statusMockGowa) lastRequest() (phone, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastPhone, m.lastPath
}

// newStatusTestApp wires the App's WARegistry to the mock GOWA server so every
// account routes through it regardless of base URL.
func newStatusTestApp(t *testing.T, m *statusMockGowa) *handlers.App {
	t.Helper()
	app := newTestApp(t)
	// RegisterGowaFactory is process-global — this wrapper passes its own
	// mock closures + logger to NewRegistryWithFactory.
	app.WARegistry = whatsapp.NewRegistryWithFactory(
		testutil.NopLogger(),
		func(_ uuid.UUID, baseURL string) (string, string) { return "", "" },
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(m.server.URL, username, password)
		},
	)
	return app
}

// newJSONStatusRequest builds an authenticated POST /status/send JSON request.
func newJSONStatusRequest(t *testing.T, orgID, userID uuid.UUID, body any) *fastglue.Request {
	t.Helper()
	req := testutil.NewJSONRequest(t, body)
	testutil.SetFullAuthContext(req, orgID, userID, nil, true)
	return req
}

// newMultipartStatusRequest builds an authenticated multipart POST /status/send
// request carrying a single file part named "file".
func newMultipartStatusRequest(t *testing.T, orgID, userID uuid.UUID, fields map[string]string, filename, contentType string, fileData []byte) *fastglue.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for k, v := range fields {
		require.NoError(t, writer.WriteField(k, v))
	}
	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(fileData)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.Header.SetContentType(writer.FormDataContentType())
	ctx.Request.SetBody(buf.Bytes())
	req := &fastglue.Request{RequestCtx: ctx}
	testutil.SetFullAuthContext(req, orgID, userID, nil, true)
	return req
}

// --- SendStatus handler tests ---

func TestApp_SendStatus_Text_ForwardsStatusBroadcastJID(t *testing.T) {
	mock := newStatusMockGowa()
	defer mock.close()

	app := newStatusTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithSuperAdmin())

	req := newJSONStatusRequest(t, org.ID, user.ID, handlers.SendStatusRequest{
		Message:         "My status update",
		WhatsAppAccount: account.Name,
	})

	require.NoError(t, app.SendStatus(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// The handler MUST address GOWA with the well-known status@broadcast JID.
	phone, path := mock.lastRequest()
	assert.Equal(t, gowaStatusJID, phone, "status post must target status@broadcast")
	assert.Equal(t, "/send/message", path)

	var resp struct {
		Data struct {
			MessageID string `json:"message_id"`
			Status    string `json:"status"`
		} `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, "status-msg-123", resp.Data.MessageID)
	assert.Equal(t, "sent", resp.Data.Status)
}

func TestApp_SendStatus_Text_RejectsEmptyMessage(t *testing.T) {
	mock := newStatusMockGowa()
	defer mock.close()

	app := newStatusTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithSuperAdmin())

	req := newJSONStatusRequest(t, org.ID, user.ID, handlers.SendStatusRequest{
		Message:         "   ",
		WhatsAppAccount: account.Name,
	})

	require.NoError(t, app.SendStatus(req))
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))

	// No GOWA call should have happened.
	phone, _ := mock.lastRequest()
	assert.Empty(t, phone)
}

func TestApp_SendStatus_RejectsUnsupportedType(t *testing.T) {
	mock := newStatusMockGowa()
	defer mock.close()

	app := newStatusTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	createTestAccount(t, app, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithSuperAdmin())

	req := newJSONStatusRequest(t, org.ID, user.ID, map[string]any{
		"type":    "audio",
		"message": "x",
	})

	require.NoError(t, app.SendStatus(req))
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_SendStatus_PropagatesGOWAError(t *testing.T) {
	mock := newStatusMockGowa()
	defer mock.close()
	mock.fail = true

	app := newStatusTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithSuperAdmin())

	req := newJSONStatusRequest(t, org.ID, user.ID, handlers.SendStatusRequest{
		Message:         "will fail",
		WhatsAppAccount: account.Name,
	})

	require.NoError(t, app.SendStatus(req))
	assert.Equal(t, fasthttp.StatusBadGateway, testutil.GetResponseStatusCode(req))
}

func TestApp_SendStatus_AccountResolutionFailure(t *testing.T) {
	mock := newStatusMockGowa()
	defer mock.close()

	app := newStatusTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithSuperAdmin())

	// Ask for an account that does not exist in this org.
	req := newJSONStatusRequest(t, org.ID, user.ID, handlers.SendStatusRequest{
		Message:         "x",
		WhatsAppAccount: "nonexistent-account",
	})

	require.NoError(t, app.SendStatus(req))
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_SendStatus_Image_Multipart(t *testing.T) {
	mock := newStatusMockGowa()
	defer mock.close()

	app := newStatusTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithSuperAdmin())

	png := minimalPNG()
	req := newMultipartStatusRequest(t, org.ID, user.ID, map[string]string{
		"type":              "image",
		"caption":           "status caption",
		"whatsapp_account":  account.Name,
	}, "status.png", "image/png", png)

	require.NoError(t, app.SendStatus(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req), "body: %s", string(testutil.GetResponseBody(req)))

	phone, path := mock.lastRequest()
	assert.Equal(t, gowaStatusJID, phone)
	assert.Equal(t, "/send/image", path)
}

func TestApp_SendStatus_DoesNotPersistMessageOrContact(t *testing.T) {
	mock := newStatusMockGowa()
	defer mock.close()

	app := newStatusTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithSuperAdmin())

	beforeMessages := countRows(t, app.DB, "messages")
	beforeContacts := countRows(t, app.DB, "contacts")

	req := newJSONStatusRequest(t, org.ID, user.ID, handlers.SendStatusRequest{
		Message:         "ephemeral status",
		WhatsAppAccount: account.Name,
	})
	require.NoError(t, app.SendStatus(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// A status post must NOT create a contact or a message row — it is not a
	// conversation. This is the core design contract of the dedicated path.
	assert.Equal(t, beforeMessages, countRows(t, app.DB, "messages"), "status must not persist a message")
	assert.Equal(t, beforeContacts, countRows(t, app.DB, "contacts"), "status must not create a contact")
}

// --- helpers ---

// gowaStatusJID mirrors pkg/gowa.StatusBroadcastJID here to avoid an import
// cycle surface; the test asserts the handler forwards exactly this string.
const gowaStatusJID = "status@broadcast"

func countRows(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	rows, err := db.Raw("SELECT count(*) FROM " + table).Rows()
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		require.NoError(t, rows.Scan(&n))
	}
	return n
}

// minimalPNG returns the bytes of a valid 1x1 transparent PNG.
func minimalPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
}

