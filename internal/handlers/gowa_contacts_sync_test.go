package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// newChatsMock returns a GOWA server that serves GET /chats with a fixed chat
// list (one 1:1 chat and one group chat), so the contact-sync handler has
// deterministic data to import.
func newChatsMock(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chats" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    "SUCCESS",
			"message": "Success get chat list",
			"results": map[string]any{
				"data": []map[string]any{
					{"jid": "16505551234@s.whatsapp.net", "name": "Alice One"},
					{"jid": "120363group@g.us", "name": "Team Group"},
				},
				"pagination": map[string]any{"total": 2, "limit": 100, "offset": 0},
			},
		})
	}))
	return server
}

// TestSyncGowaInstanceDeviceContacts_ImportsChats verifies the happy path: a
// connected device with a provisioned account imports its chat list into the
// contacts table, with group chats flagged as groups and 1:1 chats keyed by
// phone number.
func TestSyncGowaInstanceDeviceContacts_ImportsChats(t *testing.T) {
	t.Parallel()

	mock := newChatsMock(t)
	defer mock.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)

	// DB GowaInstance whose BaseURL points at the mock. resolveGowaInstance
	// builds gowa.New(baseURL, user, pass) directly from this row.
	inst := &models.GowaInstance{
		OrganizationID: org.ID,
		Name:           "test-server-" + uuid.New().String()[:8],
		BaseURL:        mock.URL,
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(inst).Error)

	// Provisioned WhatsAppAccount row for the device (the /sync endpoint's job).
	// Name AND device ID must be unique: idx_wa_org_name enforces name uniqueness,
	// and getGowaAccountByDeviceID is global (not org-scoped), so parallel tests
	// sharing "dev-1" would cross-resolve into each other's accounts/orgs.
	deviceID := "dev-imports-" + uuid.New().String()[:8]
	accountName := "gowa-dev-1-" + uuid.New().String()[:8]
	acc := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           accountName,
		GowaDeviceID:   deviceID,
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(acc).Error)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	testutil.SetPathParam(req, "deviceId", deviceID)

	require.NoError(t, app.SyncGowaInstanceDeviceContacts(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"want 200, body=%s", string(testutil.GetResponseBody(req)))

	// Response reports both chats touched, both new.
	var env struct {
		Data struct {
			Synced  int `json:"synced"`
			Created int `json:"created"`
			Total   int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &env))
	assert.Equal(t, 2, env.Data.Synced)
	assert.Equal(t, 2, env.Data.Created)
	assert.Equal(t, 2, env.Data.Total)

	// 1:1 chat → contact keyed by phone digits, account stamped.
	var c1 models.Contact
	require.NoError(t, app.DB.Where("organization_id = ? AND phone_number = ?", org.ID, "16505551234").First(&c1).Error)
	assert.Equal(t, "Alice One", c1.ProfileName)
	assert.Equal(t, accountName, c1.WhatsAppAccount, "whats_app_account should be stamped on sync")

	// Group chat → contact keyed by the full @g.us JID, flagged as group.
	var c2 models.Contact
	require.NoError(t, app.DB.Where("organization_id = ? AND phone_number = ?", org.ID, "120363group@g.us").First(&c2).Error)
	assert.Equal(t, "Team Group", c2.ProfileName)
	assert.Equal(t, true, c2.Metadata["is_group_chat"], "group chat must carry is_group_chat metadata")
}

// TestSyncGowaInstanceDeviceContacts_NeedsAccountProvisioning verifies that
// when no WhatsAppAccount row exists for the device, the handler returns 409
// with a clear message instead of side-effecting GOWA's webhook config.
func TestSyncGowaInstanceDeviceContacts_NeedsAccountProvisioning(t *testing.T) {
	t.Parallel()

	mock := newChatsMock(t)
	defer mock.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)
	inst := &models.GowaInstance{OrganizationID: org.ID, Name: "s", BaseURL: mock.URL, IsActive: true}
	require.NoError(t, app.DB.Create(inst).Error)
	// NOTE: no WhatsAppAccount row created → handler must 409.

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	testutil.SetPathParam(req, "deviceId", "dev-unprovisioned")

	require.NoError(t, app.SyncGowaInstanceDeviceContacts(req))
	assert.Equal(t, fasthttp.StatusConflict, testutil.GetResponseStatusCode(req))
	assert.Contains(t, string(testutil.GetResponseBody(req)), "Sync")
}

// TestSyncGowaInstanceDeviceContacts_Idempotent verifies a second sync does
// not create duplicate contacts (GetOrCreateContact is the upsert path).
func TestSyncGowaInstanceDeviceContacts_Idempotent(t *testing.T) {
	t.Parallel()

	mock := newChatsMock(t)
	defer mock.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)
	inst := &models.GowaInstance{OrganizationID: org.ID, Name: "s-" + uuid.New().String()[:8], BaseURL: mock.URL, IsActive: true}
	require.NoError(t, app.DB.Create(inst).Error)
	// Unique device ID: getGowaAccountByDeviceID is global (not org-scoped), so
	// parallel tests sharing "dev-1" would cross-resolve. A unique tail avoids that.
	deviceID := "dev-idem-" + uuid.New().String()[:8]
	acc := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "gowa-dev-1-" + uuid.New().String()[:8],
		GowaDeviceID:   deviceID,
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(acc).Error)

	run := func() {
		req := testutil.NewJSONRequest(t, nil)
		testutil.SetAuthContext(req, org.ID, admin.ID)
		testutil.SetPathParam(req, "id", inst.ID.String())
		testutil.SetPathParam(req, "deviceId", deviceID)
		require.NoError(t, app.SyncGowaInstanceDeviceContacts(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	}

	run() // first sync: 2 created
	run() // second sync: 0 created, still 2 touched

	var count int64
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("organization_id = ?", org.ID).Count(&count).Error)
	assert.Equal(t, int64(2), count, "no duplicate contacts after re-sync")
}

// TestSyncGowaInstanceDeviceContacts_OverwritesStaleAccount verifies that a
// pre-existing contact carrying an empty or stale whats_app_account gets
// re-stamped with the syncing device's account: a sync from a specific GOWA
// server is authoritative about which account owns the chat.
func TestSyncGowaInstanceDeviceContacts_OverwritesStaleAccount(t *testing.T) {
	t.Parallel()

	mock := newChatsMock(t)
	defer mock.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)
	inst := &models.GowaInstance{OrganizationID: org.ID, Name: "s-" + uuid.New().String()[:8], BaseURL: mock.URL, IsActive: true}
	require.NoError(t, app.DB.Create(inst).Error)

	deviceID := "dev-stale-" + uuid.New().String()[:8]
	accountName := "gowa-dev-1-" + uuid.New().String()[:8]
	acc := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           accountName,
		GowaDeviceID:   deviceID,
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(acc).Error)

	// Pre-existing contacts: one stamped with a stale account name, one with
	// an empty account (the webhook-created shape). Both must end up on the
	// syncing device's account.
	stale := &models.Contact{
		OrganizationID:  org.ID,
		PhoneNumber:     "16505551234",
		WhatsAppAccount: "some-old-account",
	}
	require.NoError(t, app.DB.Create(stale).Error)
	empty := &models.Contact{
		OrganizationID: org.ID,
		PhoneNumber:    "120363group@g.us",
	}
	require.NoError(t, app.DB.Create(empty).Error)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	testutil.SetPathParam(req, "deviceId", deviceID)

	require.NoError(t, app.SyncGowaInstanceDeviceContacts(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"want 200, body=%s", string(testutil.GetResponseBody(req)))

	var c1 models.Contact
	require.NoError(t, app.DB.First(&c1, "id = ?", stale.ID).Error)
	assert.Equal(t, accountName, c1.WhatsAppAccount, "stale account must be overwritten on sync")

	var c2 models.Contact
	require.NoError(t, app.DB.First(&c2, "id = ?", empty.ID).Error)
	assert.Equal(t, accountName, c2.WhatsAppAccount, "empty account must be stamped on sync")
}

// TestSyncGowaInstanceDeviceContacts_AgentDenied verifies the devices:write
// permission is enforced.
func TestSyncGowaInstanceDeviceContacts_AgentDenied(t *testing.T) {
	t.Parallel()

	mock := newChatsMock(t)
	defer mock.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createTestAgent(t, app, org.ID)
	inst := &models.GowaInstance{OrganizationID: org.ID, Name: "s", BaseURL: mock.URL, IsActive: true}
	require.NoError(t, app.DB.Create(inst).Error)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	testutil.SetPathParam(req, "deviceId", "dev-1")

	require.NoError(t, app.SyncGowaInstanceDeviceContacts(req))
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

// reference the package to keep the import if helpers ever move.
var _ = handlers.App{}
