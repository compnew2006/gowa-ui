package modulemanagement

import (
	"context"
	"strings"
	"testing"

	"github.com/compnew2006/whatomate/internal/core"
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
		{fasthttp.MethodGet, "/api/organizations/" + uuid.NewString() + "/modules"},
		{fasthttp.MethodPut, "/api/organizations/" + uuid.NewString() + "/modules/facebook-core"},
	}
	for _, route := range routes {
		request := testutil.NewRequest(t)
		request.RequestCtx.Request.Header.SetMethod(route.method)
		request.RequestCtx.Request.SetRequestURI(route.path)
		handler(request.RequestCtx)
		assert.NotEqual(t, fasthttp.StatusNotFound, request.RequestCtx.Response.StatusCode())
	}
}
