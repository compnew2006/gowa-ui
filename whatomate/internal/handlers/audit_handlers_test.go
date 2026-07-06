package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// seedAuditEvent inserts one audit row owned by orgID (nil for a global event).
func seedAuditEvent(t *testing.T, app *handlers.App, orgID *uuid.UUID, action string) models.AuditEvent {
	t.Helper()
	evt := models.AuditEvent{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Category:       "admin",
		Action:         action,
		Source:         "user",
		Success:        true,
		Details:        models.JSONB{},
	}
	require.NoError(t, app.DB.Create(&evt).Error)
	return evt
}

// superAdminAuditRequest builds a GET /api/audit-events request authenticated
// as a super admin (bypasses requirePermission; IsSuperAdmin=true).
func superAdminAuditRequest(query string) *fastglue.Request {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api/audit-events")
	ctx.Request.Header.SetMethod("GET")
	ctx.SetUserValue("user_id", uuid.New())
	ctx.SetUserValue("organization_id", uuid.New())
	ctx.SetUserValue(middleware.ContextKeyIsSuperAdmin, true)
	if query != "" {
		ctx.URI().SetQueryString(query)
	}
	return &fastglue.Request{RequestCtx: ctx}
}

func TestListAuditEvents_SuperAdmin_SeesAllOrgsAndGlobal(t *testing.T) {
	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	orgB := testutil.CreateTestOrganization(t, app.DB)
	seedAuditEvent(t, app, &orgA.ID, "user_created")
	seedAuditEvent(t, app, &orgB.ID, "user_created")
	seedAuditEvent(t, app, nil, "server_started") // global event

	req := superAdminAuditRequest("")
	require.NoError(t, app.ListAuditEvents(req))

	var resp struct {
		Data struct {
			Events []models.AuditEvent `json:"events"`
			Total  int64               `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, int64(3), resp.Data.Total, "super admin must see all orgs + global events")
	assert.Len(t, resp.Data.Events, 3)
}

func TestListAuditEvents_SuperAdmin_FiltersByOrgID(t *testing.T) {
	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	orgB := testutil.CreateTestOrganization(t, app.DB)
	seedAuditEvent(t, app, &orgA.ID, "user_created")
	seedAuditEvent(t, app, &orgB.ID, "user_created")

	req := superAdminAuditRequest("organization_id=" + orgA.ID.String())
	require.NoError(t, app.ListAuditEvents(req))

	var resp struct {
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, int64(1), resp.Data.Total, "organization_id filter must narrow to org A only")
}

func TestListAuditEvents_FiltersByCategory(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	seedAuditEvent(t, app, &org.ID, "user_created") // category admin
	authEvt := models.AuditEvent{
		ID:             uuid.New(),
		OrganizationID: &org.ID,
		Category:       "auth",
		Action:         "login_success",
		Source:         "user",
		Success:        true,
		Details:        models.JSONB{},
	}
	require.NoError(t, app.DB.Create(&authEvt).Error)

	req := superAdminAuditRequest("category=auth")
	require.NoError(t, app.ListAuditEvents(req))

	var resp struct {
		Data struct {
			Events []models.AuditEvent `json:"events"`
			Total  int64               `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
	require.Len(t, resp.Data.Events, 1)
	assert.Equal(t, "login_success", resp.Data.Events[0].Action)
}

func TestListAuditEvents_Pagination_CapsPerPage(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	for i := 0; i < 3; i++ {
		seedAuditEvent(t, app, &org.ID, "user_created")
	}

	req := superAdminAuditRequest("per_page=999")
	require.NoError(t, app.ListAuditEvents(req))

	var resp struct {
		Data struct {
			PerPage int `json:"per_page"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, 200, resp.Data.PerPage, "per_page must be capped at 200")
}

func TestListAuditEvents_NonAdmin_Returns403(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	seedAuditEvent(t, app, &org.ID, "user_created")

	// Non-super-admin user with no audit:read permission.
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api/audit-events")
	ctx.Request.Header.SetMethod("GET")
	ctx.SetUserValue("user_id", uuid.New())
	ctx.SetUserValue("organization_id", org.ID)
	// is_super_admin not set → IsSuperAdmin returns false.
	req := &fastglue.Request{RequestCtx: ctx}

	_ = app.ListAuditEvents(req)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req),
		"non-admin without audit:read must be rejected at the handler")
}

func TestListAuditEvents_OrdersNewestFirst(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	seedAuditEvent(t, app, &org.ID, "first")
	seedAuditEvent(t, app, &org.ID, "second")

	req := superAdminAuditRequest("per_page=2")
	require.NoError(t, app.ListAuditEvents(req))

	var resp struct {
		Data struct {
			Events []models.AuditEvent `json:"events"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	require.Len(t, resp.Data.Events, 2)
	assert.Equal(t, "second", resp.Data.Events[0].Action, "newest event must come first")
}
