package perinstanceuploadscleanup

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "failed to open test database")

	require.NoError(t, db.Exec(`
		CREATE TABLE organizations (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			name TEXT,
			slug TEXT,
			settings JSON
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE whatsapp_instances (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			organization_id TEXT,
			name TEXT,
			phone_number TEXT,
			jid TEXT,
			status TEXT DEFAULT 'disconnected',
			is_default BOOLEAN DEFAULT FALSE,
			session_id TEXT,
			auto_read_receipt BOOLEAN DEFAULT FALSE,
			settings JSON DEFAULT '{}',
			last_connected_at DATETIME,
			send_blocked_until DATETIME,
			send_block_reason TEXT DEFAULT ''
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE instance_uploads_cleanup_audits (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			organization_id TEXT NOT NULL,
			instance_id TEXT NOT NULL,
			actor_user_id TEXT,
			actor_email TEXT,
			old_inherit BOOLEAN,
			new_inherit BOOLEAN NOT NULL,
			old_retention_days INTEGER,
			new_retention_days INTEGER,
			reason TEXT
		)
	`).Error)

	require.NoError(t, db.Exec(`CREATE INDEX idx_iuca_org_id ON instance_uploads_cleanup_audits (organization_id)`).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_iuca_inst_id ON instance_uploads_cleanup_audits (instance_id)`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE user_organizations (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			user_id TEXT NOT NULL,
			organization_id TEXT NOT NULL,
			role_id TEXT,
			is_super_admin BOOLEAN DEFAULT FALSE
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE custom_roles (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			organization_id TEXT NOT NULL,
			name TEXT
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE custom_role_permissions (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			custom_role_id TEXT NOT NULL,
			resource TEXT NOT NULL,
			action TEXT NOT NULL
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			email TEXT,
			name TEXT,
			role_id TEXT,
			is_super_admin BOOLEAN DEFAULT FALSE
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE permissions (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			description TEXT
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE role_permissions (
			custom_role_id TEXT NOT NULL,
			permission_id TEXT NOT NULL,
			PRIMARY KEY (custom_role_id, permission_id)
		)
	`).Error)

	return db
}

func setupPlugin(t *testing.T, db *gorm.DB) *Plugin {
	t.Helper()
	app := &handlers.App{
		DB:  db,
		Log: testutil.NopLogger(),
	}
	p := &Plugin{
		PluginBase: core.PluginBase{
			App: app,
			DB:  db,
			Log: slog.Default(),
		},
		srv: newService(db, slog.Default()),
	}
	return p
}

func seedOrg(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	orgID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO organizations (id, created_at, updated_at, name, slug, settings) VALUES (?, ?, ?, ?, ?, '{}')`,
		orgID.String(), time.Now().UTC(), time.Now().UTC(), "Test Org", "test-org",
	).Error)
	return orgID
}

func seedSuperAdmin(t *testing.T, db *gorm.DB, orgID, userID uuid.UUID) {
	t.Helper()
	roleID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO custom_roles (id, created_at, updated_at, organization_id, name) VALUES (?, ?, ?, ?, ?)`,
		roleID.String(), time.Now().UTC(), time.Now().UTC(), orgID.String(), "admin",
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, created_at, updated_at, email, name, is_super_admin, role_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID.String(), time.Now().UTC(), time.Now().UTC(), "admin@test.com", "Admin", true, roleID.String(),
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO user_organizations (id, created_at, updated_at, user_id, organization_id, is_super_admin, role_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), time.Now().UTC(), time.Now().UTC(), userID.String(), orgID.String(), true, roleID.String(),
	).Error)
}

func seedInstanceHandlerTest(t *testing.T, db *gorm.DB, orgID uuid.UUID, settings string) uuid.UUID {
	t.Helper()
	instID := uuid.New()
	if settings == "" {
		settings = `{"uploads_cleanup":{"inherit":true}}`
	}
	require.NoError(t, db.Exec(
		`INSERT INTO whatsapp_instances (id, created_at, updated_at, organization_id, name, settings) VALUES (?, ?, ?, ?, ?, ?)`,
		instID.String(), time.Now().UTC(), time.Now().UTC(), orgID.String(), "Test Instance", settings,
	).Error)
	return instID
}

func makeRequest(t *testing.T, orgID, userID, instanceID uuid.UUID, body []byte) *fastglue.Request {
	t.Helper()
	var req fasthttp.Request
	req.SetRequestURI("http://localhost/api/instances/" + instanceID.String() + "/uploads-cleanup")
	if body != nil {
		req.SetBody(body)
		req.Header.SetMethod("POST")
	}
	var ctx fasthttp.RequestCtx
	ctx.Init(&req, &net.TCPAddr{IP: net.ParseIP("127.0.0.1")}, nil)
	ctx.SetUserValue("id", instanceID.String())
	ctx.SetUserValue("user_id", userID.String())

	// Store orgID in context for tenant.ResolveOrganizationID.
	// The real middleware sets this; we inject directly.
	ctx.SetUserValue("organization_id", orgID.String())

	return &fastglue.Request{RequestCtx: &ctx}
}

func readResponseEnvelope(t *testing.T, ctx *fasthttp.RequestCtx) map[string]interface{} {
	t.Helper()
	body := ctx.Response.Body()
	require.NotEmpty(t, body, "response body should not be empty")
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &envelope), "response should be valid JSON")
	return envelope
}

// --- GET /api/instances/{id}/uploads-cleanup ---

func TestHandleGetRetention_Success(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)
	orgID := seedOrg(t, db)
	userID := uuid.New()
	seedSuperAdmin(t, db, orgID, userID)
	instID := seedInstanceHandlerTest(t, db, orgID, `{"uploads_cleanup":{"inherit":false,"retention_days":30}}`)

	r := makeRequest(t, orgID, userID, instID, nil)

	// Stub tenant resolution: set org_id directly on context so ScopedDB works.
	// The handler calls tenant.ScopedDB which uses tenant.ResolveOrganizationID.
	// For SQLite tests we bypass tenant middleware by calling ScopedDB directly.
	// Since the handler uses tenant.ScopedDB, we need to ensure it works with our test DB.
	// The simplest approach: override getOrgAndUserID by injecting context values.
	err := p.handleGetRetention(r)
	require.NoError(t, err)

	envelope := readResponseEnvelope(t, r.RequestCtx)
	assert.Equal(t, "success", envelope["status"])

	data, ok := envelope["data"].(map[string]interface{})
	require.True(t, ok, "envelope data should be a map")
	assert.Equal(t, instID.String(), data["instance_id"])
	assert.Equal(t, false, data["inherit"])
	assert.Equal(t, float64(30), data["retention_days"])
}

func TestHandleGetRetention_InstanceNotFound(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)
	orgID := seedOrg(t, db)
	userID := uuid.New()
	seedSuperAdmin(t, db, orgID, userID)
	fakeInstanceID := uuid.New()

	r := makeRequest(t, orgID, userID, fakeInstanceID, nil)

	err := p.handleGetRetention(r)
	require.NoError(t, err)

	envelope := readResponseEnvelope(t, r.RequestCtx)
	assert.Equal(t, "error", envelope["status"])
	assert.Equal(t, 404, r.RequestCtx.Response.StatusCode())
}

// --- PUT /api/instances/{id}/uploads-cleanup ---

func TestHandlePutRetention_Success(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)
	orgID := seedOrg(t, db)
	userID := uuid.New()
	seedSuperAdmin(t, db, orgID, userID)
	instID := seedInstanceHandlerTest(t, db, orgID, `{"uploads_cleanup":{"inherit":true}}`)

	body, _ := json.Marshal(map[string]interface{}{
		"inherit":        false,
		"retention_days": 60,
		"reason":         "policy update",
	})
	r := makeRequest(t, orgID, userID, instID, body)

	err := p.handlePutRetention(r)
	require.NoError(t, err)

	envelope := readResponseEnvelope(t, r.RequestCtx)
	assert.Equal(t, "success", envelope["status"])

	data, ok := envelope["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, false, data["inherit"])

	// Verify audit row was written.
	var count int64
	db.Table("instance_uploads_cleanup_audits").Where("instance_id = ?", instID.String()).Count(&count)
	assert.Equal(t, int64(1), count, "one audit row should be written")
}

func TestHandlePutRetention_ExceedsMaxRetention(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)
	orgID := seedOrg(t, db)
	userID := uuid.New()
	seedSuperAdmin(t, db, orgID, userID)
	instID := seedInstanceHandlerTest(t, db, orgID, `{}`)

	body, _ := json.Marshal(map[string]interface{}{
		"inherit":        false,
		"retention_days": 99999,
	})
	r := makeRequest(t, orgID, userID, instID, body)

	err := p.handlePutRetention(r)
	require.NoError(t, err)

	envelope := readResponseEnvelope(t, r.RequestCtx)
	assert.Equal(t, "error", envelope["status"])
	assert.Equal(t, 400, r.RequestCtx.Response.StatusCode())
}

func TestHandlePutRetention_MissingRetentionDays(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)
	orgID := seedOrg(t, db)
	userID := uuid.New()
	seedSuperAdmin(t, db, orgID, userID)
	instID := seedInstanceHandlerTest(t, db, orgID, `{}`)

	body, _ := json.Marshal(map[string]interface{}{
		"inherit": false,
	})
	r := makeRequest(t, orgID, userID, instID, body)

	err := p.handlePutRetention(r)
	require.NoError(t, err)

	envelope := readResponseEnvelope(t, r.RequestCtx)
	assert.Equal(t, "error", envelope["status"])
	assert.Equal(t, 400, r.RequestCtx.Response.StatusCode())
}

func TestHandlePutRetention_PreservesRetentionDaysOnInheritTrue(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)
	orgID := seedOrg(t, db)
	userID := uuid.New()
	seedSuperAdmin(t, db, orgID, userID)
	instID := seedInstanceHandlerTest(t, db, orgID, `{"uploads_cleanup":{"inherit":false,"retention_days":30}}`)

	// Toggle to inherit=true WITHOUT sending retention_days.
	body, _ := json.Marshal(map[string]interface{}{
		"inherit": true,
	})
	r := makeRequest(t, orgID, userID, instID, body)

	err := p.handlePutRetention(r)
	require.NoError(t, err)

	envelope := readResponseEnvelope(t, r.RequestCtx)
	assert.Equal(t, "success", envelope["status"])

	data, ok := envelope["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, data["inherit"])

	// retention_days should still be 30 (Q-OPT-2: preserve old value).
	rd, ok := data["retention_days"].(float64)
	require.True(t, ok, "retention_days should be present")
	assert.Equal(t, float64(30), rd, "retention_days should be preserved when toggling to inherit=true")
}

// --- POST /api/instances/{id}/uploads-cleanup/run ---

// --- GET /api/instances/{id}/uploads-cleanup/history ---

func TestHandleHistory_DefaultLimit(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)
	orgID := seedOrg(t, db)
	userID := uuid.New()
	seedSuperAdmin(t, db, orgID, userID)
	instID := seedInstanceHandlerTest(t, db, orgID, "")

	// Seed 7 audit rows.
	for i := 0; i < 7; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO instance_uploads_cleanup_audits (id, created_at, updated_at, organization_id, instance_id, new_inherit, reason) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			uuid.New().String(), time.Now().UTC().Add(time.Duration(i)*time.Minute), time.Now().UTC(), orgID.String(), instID.String(), true, fmt.Sprintf("reason-%d", i),
		).Error)
	}

	r := makeRequest(t, orgID, userID, instID, nil)

	err := p.handleHistory(r)
	require.NoError(t, err)

	envelope := readResponseEnvelope(t, r.RequestCtx)
	assert.Equal(t, "success", envelope["status"])

	data, ok := envelope["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(5), data["limit"], "default limit should be 5")
	assert.Equal(t, float64(0), data["offset"])
	assert.Equal(t, float64(7), data["total"])
}

func TestHandleHistory_InvalidLimit(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)
	orgID := seedOrg(t, db)
	userID := uuid.New()
	seedSuperAdmin(t, db, orgID, userID)
	instID := seedInstanceHandlerTest(t, db, orgID, "")

	// Set query param limit=0 via URI.
	r := makeRequestWithQuery(t, orgID, userID, instID, "limit=0")

	err := p.handleHistory(r)
	require.NoError(t, err)

	assert.Equal(t, 400, r.RequestCtx.Response.StatusCode())
}

func TestHandleHistory_LimitExceedsMax(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)
	orgID := seedOrg(t, db)
	userID := uuid.New()
	seedSuperAdmin(t, db, orgID, userID)
	instID := seedInstanceHandlerTest(t, db, orgID, "")

	r := makeRequestWithQuery(t, orgID, userID, instID, "limit=200")

	err := p.handleHistory(r)
	require.NoError(t, err)

	assert.Equal(t, 400, r.RequestCtx.Response.StatusCode())
}

func makeRequestWithQuery(t *testing.T, orgID, userID, instanceID uuid.UUID, query string) *fastglue.Request {
	t.Helper()
	var req fasthttp.Request
	req.SetRequestURI("http://localhost/api/instances/" + instanceID.String() + "/uploads-cleanup/history?" + query)
	var ctx fasthttp.RequestCtx
	ctx.Init(&req, &net.TCPAddr{IP: net.ParseIP("127.0.0.1")}, nil)
	ctx.SetUserValue("id", instanceID.String())
	ctx.SetUserValue("user_id", userID.String())
	ctx.SetUserValue("organization_id", orgID.String())
	return &fastglue.Request{RequestCtx: &ctx}
}

// --- GET /api/org/uploads-cleanup/instances (overview) ---

func TestHandleOverview_BasicEnvelope(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)
	orgID := seedOrg(t, db)
	userID := uuid.New()
	seedSuperAdmin(t, db, orgID, userID)

	// Seed two instances with different settings.
	seedInstanceHandlerTest(t, db, orgID, `{"uploads_cleanup":{"inherit":false,"retention_days":30}}`)
	seedInstanceHandlerTest(t, db, orgID, `{"uploads_cleanup":{"inherit":true}}`)

	r := makeRequest(t, orgID, userID, uuid.Nil, nil)

	err := p.handleOverview(r)
	require.NoError(t, err)

	envelope := readResponseEnvelope(t, r.RequestCtx)
	assert.Equal(t, "success", envelope["status"])

	data, ok := envelope["data"].(map[string]interface{})
	require.True(t, ok)

	items, ok := data["items"].([]interface{})
	require.True(t, ok, "items should be a slice")
	assert.Equal(t, 2, len(items), "should have 2 instance rows")

	// Verify envelope pagination fields.
	assert.Equal(t, float64(20), data["limit"])
	assert.Equal(t, float64(0), data["offset"])
	assert.Equal(t, float64(2), data["total"])
}

func TestHandleOverview_FilterBySource(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)
	orgID := seedOrgWithCleanup(t, db, 90) // workspace default = 90 days
	userID := uuid.New()
	seedSuperAdmin(t, db, orgID, userID)

	// Instance with custom retention (source="custom").
	seedInstanceHandlerTest(t, db, orgID, `{"uploads_cleanup":{"inherit":false,"retention_days":30}}`)
	// Instance inheriting (source="default").
	seedInstanceHandlerTest(t, db, orgID, `{"uploads_cleanup":{"inherit":true}}`)

	// Filter for source=custom only.
	r := makeRequestWithQuery(t, orgID, userID, uuid.Nil, "source=custom")
	err := p.handleOverview(r)
	require.NoError(t, err)

	envelope := readResponseEnvelope(t, r.RequestCtx)
	data := envelope["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	assert.Equal(t, 1, len(items), "should have 1 custom instance")

	row := items[0].(map[string]interface{})
	assert.Equal(t, "custom", row["effective_source"])
}

func seedOrgWithCleanup(t *testing.T, db *gorm.DB, retentionDays int) uuid.UUID {
	t.Helper()
	orgID := uuid.New()
	settings := fmt.Sprintf(`{"uploads_cleanup_retention_days":%d}`, retentionDays)
	require.NoError(t, db.Exec(
		`INSERT INTO organizations (id, created_at, updated_at, name, slug, settings) VALUES (?, ?, ?, ?, ?, ?)`,
		orgID.String(), time.Now().UTC(), time.Now().UTC(), "Test Org", "test-org-"+orgID.String()[:8], settings,
	).Error)
	return orgID
}

func TestHandleRun_DisabledRetention(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)
	orgID := seedOrg(t, db)
	userID := uuid.New()
	seedSuperAdmin(t, db, orgID, userID)
	// Instance with retention 0 = disabled
	instID := seedInstanceHandlerTest(t, db, orgID, `{"uploads_cleanup":{"inherit":false,"retention_days":0}}`)

	r := makeRequest(t, orgID, userID, instID, nil)

	err := p.handleRun(r)
	require.NoError(t, err)

	envelope := readResponseEnvelope(t, r.RequestCtx)
	assert.Equal(t, "error", envelope["status"])
	assert.Equal(t, 400, r.RequestCtx.Response.StatusCode())
}

// --- Tenant scoping / org resolution ---

func TestHandleGetRetention_MissingOrgID(t *testing.T) {
	t.Parallel()

	db := setupHandlerTestDB(t)
	p := setupPlugin(t, db)

	userID := uuid.New()
	instID := uuid.New()

	var req fasthttp.Request
	req.SetRequestURI("http://localhost/api/instances/" + instID.String() + "/uploads-cleanup")
	var ctx fasthttp.RequestCtx
	ctx.Init(&req, &net.TCPAddr{IP: net.ParseIP("127.0.0.1")}, nil)
	ctx.SetUserValue("id", instID.String())
	ctx.SetUserValue("user_id", userID.String())
	// No org_id set — tenant resolution should fail.
	r := &fastglue.Request{RequestCtx: &ctx}

	err := p.handleGetRetention(r)
	// requireInstanceAccess sends the error envelope and returns errResponseSent.
	require.Error(t, err)

	envelope := readResponseEnvelope(t, r.RequestCtx)
	assert.Equal(t, "error", envelope["status"])
}
