package modulemanagement

import (
	"context"
	"strings"
	"testing"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newManagementTestManager(t *testing.T) (*core.ModuleManager, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(
		"CREATE TABLE organizations (id text PRIMARY KEY, name text NOT NULL, slug text NOT NULL, deleted_at datetime)",
	).Error)
	// module_events table is plugin-owned; ensure it exists for audit tests.
	require.NoError(t, MigrateModuleEvents(db))

	organizationID := uuid.New()
	require.NoError(t, db.Exec(
		"INSERT INTO organizations (id, name, slug) VALUES (?, ?, ?)",
		organizationID,
		"Test",
		"test-"+uuid.NewString(),
	).Error)

	manager := core.NewModuleManager(db, []core.ModuleManifest{{
		Key: "facebook-core", DisplayName: "Facebook Core", Version: "1.0.0",
		SchemaVersion: 1, DefaultEnabled: true, Technical: true,
	}})
	require.NoError(t, manager.Migrate(context.Background()))
	require.NoError(t, manager.Sync(context.Background()))
	return manager, organizationID
}

// newManagementTestApp constructs an App wired to the same in-memory DB so the
// plugin's audit writer and events endpoints can be exercised end-to-end.
func newManagementTestApp(t *testing.T, manager *core.ModuleManager) (*handlers.App, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, MigrateModuleEvents(db))
	app := newTestAppWithDB(t, db)
	return app, db
}

// newTestAppWithDB builds a minimal handlers.App carrying only the DB field
// that this plugin needs. We avoid handlers.newTestApp (internal) by
// constructing the struct directly; recordEvent only touches app.DB and app.Log.
func newTestAppWithDB(t *testing.T, db *gorm.DB) *handlers.App {
	t.Helper()
	app := &handlers.App{DB: db}
	return app
}

func TestListEffectiveModulesUsesRequestOrganization(t *testing.T) {
	manager, organizationID := newManagementTestManager(t)
	plugin := &Plugin{manager: manager}
	request := testutil.NewRequest(t)
	request.RequestCtx.SetUserValue(middleware.ContextKeyOrganizationID, organizationID)

	require.NoError(t, plugin.listEffective(request))
	assert.Equal(t, fasthttp.StatusOK, request.RequestCtx.Response.StatusCode())
	assert.True(t, strings.Contains(string(request.RequestCtx.Response.Body()), "facebook-core"))
	assert.True(t, strings.Contains(string(request.RequestCtx.Response.Body()), "effective_enabled"))
}

func TestUpdateGlobalModuleRequiresSuperAdmin(t *testing.T) {
	manager, _ := newManagementTestManager(t)
	plugin := &Plugin{manager: manager}

	request := testutil.NewRequest(t)
	request.RequestCtx.SetUserValue("key", "facebook-core")
	request.RequestCtx.Request.SetBodyString(`{"enabled":false}`)
	require.NoError(t, plugin.updateGlobal(request))
	assert.Equal(t, fasthttp.StatusForbidden, request.RequestCtx.Response.StatusCode())

	request = testutil.NewRequest(t)
	request.RequestCtx.SetUserValue("key", "facebook-core")
	request.RequestCtx.SetUserValue(middleware.ContextKeyIsSuperAdmin, true)
	request.RequestCtx.Request.SetBodyString(`{"enabled":false}`)
	require.NoError(t, plugin.updateGlobal(request))
	assert.Equal(t, fasthttp.StatusOK, request.RequestCtx.Response.StatusCode())

	enabled, err := manager.IsGloballyEnabled(context.Background(), "facebook-core")
	require.NoError(t, err)
	assert.False(t, enabled)
}

// TestUpdateGlobalWritesAuditEvent confirms every successful give/ungive is
// recorded in the module_events audit table (req G: "audit every give/ungive").
// licenseAllows returns true here because app.License is nil (no restriction).
func TestUpdateGlobalWritesAuditEvent(t *testing.T) {
	manager, _ := newManagementTestManager(t)
	app, db := newManagementTestApp(t, manager)
	plugin := &Plugin{manager: manager, app: app}
	actor := uuid.New()

	request := testutil.NewRequest(t)
	request.RequestCtx.SetUserValue("key", "facebook-core")
	request.RequestCtx.SetUserValue(middleware.ContextKeyIsSuperAdmin, true)
	request.RequestCtx.SetUserValue(middleware.ContextKeyUserID, actor)
	request.RequestCtx.SetUserValue(middleware.ContextKeyEmail, "root@example.com")
	request.RequestCtx.Request.SetBodyString(`{"enabled":false}`)
	require.NoError(t, plugin.updateGlobal(request))
	assert.Equal(t, fasthttp.StatusOK, request.RequestCtx.Response.StatusCode())

	var events []ModuleEvent
	require.NoError(t, db.Find(&events).Error)
	require.Len(t, events, 1)
	assert.Equal(t, ModuleActionDisable, events[0].Action)
	assert.Equal(t, "facebook-core", events[0].ModuleKey)
	assert.Equal(t, moduleScopeGlobal, events[0].Scope)
	require.NotNil(t, events[0].Enabled)
	assert.False(t, *events[0].Enabled)
	require.NotNil(t, events[0].ActorUserID)
	assert.Equal(t, actor, *events[0].ActorUserID)
	assert.Equal(t, "root@example.com", events[0].ActorEmail)
}

// TestUpdateOrganizationWritesAuditEvent mirrors the above at the org scope,
// proving the per-tenant give/ungive is audited with the right organization_id.
func TestUpdateOrganizationWritesAuditEvent(t *testing.T) {
	manager, orgID := newManagementTestManager(t)
	app, db := newManagementTestApp(t, manager)

	// Super-admin bypasses authorizeOrganization.
	plugin := &Plugin{manager: manager, app: app}
	request := testutil.NewRequest(t)
	request.RequestCtx.SetUserValue("id", orgID.String())
	request.RequestCtx.SetUserValue("key", "facebook-core")
	request.RequestCtx.SetUserValue(middleware.ContextKeyIsSuperAdmin, true)
	request.RequestCtx.SetUserValue(middleware.ContextKeyOrganizationID, orgID)
	request.RequestCtx.Request.SetBodyString(`{"enabled":false}`)
	require.NoError(t, plugin.updateOrganization(request))
	assert.Equal(t, fasthttp.StatusOK, request.RequestCtx.Response.StatusCode())

	var events []ModuleEvent
	require.NoError(t, db.Find(&events).Error)
	require.Len(t, events, 1)
	assert.Equal(t, moduleScopeOrganization, events[0].Scope)
	require.NotNil(t, events[0].OrganizationID)
	assert.Equal(t, orgID, *events[0].OrganizationID)
}

// TestListGlobalEventsRequiresSuperAdmin confirms the new audit endpoint is
// super-admin gated (req G: "prevent privilege escalation").
func TestListGlobalEventsRequiresSuperAdmin(t *testing.T) {
	manager, _ := newManagementTestManager(t)
	app, _ := newManagementTestApp(t, manager)
	plugin := &Plugin{manager: manager, app: app}

	// Non-super-admin → 403.
	request := testutil.NewRequest(t)
	require.NoError(t, plugin.listGlobalEvents(request))
	assert.Equal(t, fasthttp.StatusForbidden, request.RequestCtx.Response.StatusCode())

	// Super-admin → 200.
	request = testutil.NewRequest(t)
	request.RequestCtx.SetUserValue(middleware.ContextKeyIsSuperAdmin, true)
	require.NoError(t, plugin.listGlobalEvents(request))
	assert.Equal(t, fasthttp.StatusOK, request.RequestCtx.Response.StatusCode())
}

func TestPluginRegistersModuleManagementRoutes(t *testing.T) {
	plugin := &Plugin{}
	glue := fastglue.NewGlue()
	glue.Before(func(r *fastglue.Request) *fastglue.Request {
		r.RequestCtx.SetStatusCode(fasthttp.StatusUnauthorized)
		return nil
	})
	plugin.Routes(glue)
	handler := glue.Handler()

	routes := []struct {
		method string
		path   string
	}{
		{fasthttp.MethodGet, "/api/modules/effective"},
		{fasthttp.MethodGet, "/api/admin/modules"},
		{fasthttp.MethodPut, "/api/admin/modules/facebook-core"},
		{fasthttp.MethodGet, "/api/admin/modules/events"},
		{fasthttp.MethodGet, "/api/organizations/" + uuid.NewString() + "/modules"},
		{fasthttp.MethodPut, "/api/organizations/" + uuid.NewString() + "/modules/facebook-core"},
		{fasthttp.MethodGet, "/api/organizations/" + uuid.NewString() + "/modules/events"},
	}
	for _, route := range routes {
		request := testutil.NewRequest(t)
		request.RequestCtx.Request.Header.SetMethod(route.method)
		request.RequestCtx.Request.SetRequestURI(route.path)
		handler(request.RequestCtx)
		assert.NotEqual(t, fasthttp.StatusNotFound, request.RequestCtx.Response.StatusCode(),
			"route %s %s must be registered (got 404)", route.method, route.path)
	}
}
