package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"gorm.io/gorm"
)

// auditLogsRole creates a role with audit_logs:read permission.
func auditLogsRole(t *testing.T, db *gorm.DB, orgID uuid.UUID) *models.CustomRole {
	t.Helper()
	return testutil.CreateTestRoleWithKeys(t, db, orgID, "audit-reader", []string{"audit_logs:read"})
}

func makeAuditLog(t *testing.T, db *gorm.DB, orgID, userID, resourceID uuid.UUID, resType string, action models.AuditAction, when time.Time) *models.AuditLog {
	t.Helper()
	log := &models.AuditLog{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ResourceType:   resType,
		ResourceID:     resourceID,
		UserID:         userID,
		UserName:       "tester",
		Action:         action,
		Changes:        models.JSONBArray{map[string]any{"field": "name", "old_value": nil, "new_value": "x"}},
	}
	require.NoError(t, db.Create(log).Error)
	// Force created_at since GORM autoCreateTime overrides on Create.
	require.NoError(t, db.Model(log).Update("created_at", when).Error)
	return log
}

// --- ListAuditLogs ---

func TestApp_ListAuditLogs_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := auditLogsRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	now := time.Now()
	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "contact", models.AuditActionCreated, now.Add(-2*time.Hour))
	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "contact", models.AuditActionUpdated, now.Add(-1*time.Hour))
	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "user", models.AuditActionDeleted, now)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	require.NoError(t, app.ListAuditLogs(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			AuditLogs []handlers.AuditLogResponse `json:"audit_logs"`
			Total     int                         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, 3, resp.Data.Total)
	assert.Len(t, resp.Data.AuditLogs, 3)
	// DESC by created_at: newest first.
	assert.Equal(t, models.AuditActionDeleted, resp.Data.AuditLogs[0].Action)
}

func TestApp_ListAuditLogs_FilterByResourceType(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := auditLogsRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "contact", models.AuditActionCreated, time.Now())
	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "user", models.AuditActionCreated, time.Now())
	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "contact", models.AuditActionUpdated, time.Now())

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "resource_type", "contact")

	require.NoError(t, app.ListAuditLogs(req))
	var resp struct {
		Data struct {
			AuditLogs []handlers.AuditLogResponse `json:"audit_logs"`
			Total     int                         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, 2, resp.Data.Total)
	for _, l := range resp.Data.AuditLogs {
		assert.Equal(t, "contact", l.ResourceType)
	}
}

func TestApp_ListAuditLogs_FilterByResourceTypeMulti(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := auditLogsRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	// An account and its per-account settings blocks (close_rating, call_auto_reject)
	// all share the same resource_id but use distinct resource_type strings. The
	// account detail page aggregates them into a single Activity Log via a
	// comma-separated resource_type filter.
	accountID := uuid.New()
	makeAuditLog(t, app.DB, org.ID, user.ID, accountID, "account", models.AuditActionCreated, time.Now())
	makeAuditLog(t, app.DB, org.ID, user.ID, accountID, "settings.close_rating", models.AuditActionUpdated, time.Now())
	makeAuditLog(t, app.DB, org.ID, user.ID, accountID, "settings.call_auto_reject", models.AuditActionUpdated, time.Now())
	// Noise: same resource_id but a type not in the filter, plus a different id.
	makeAuditLog(t, app.DB, org.ID, user.ID, accountID, "contact", models.AuditActionCreated, time.Now())
	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "account", models.AuditActionCreated, time.Now())

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "resource_type", "account,settings.close_rating,settings.call_auto_reject")
	testutil.SetQueryParam(req, "resource_id", accountID.String())

	require.NoError(t, app.ListAuditLogs(req))
	var resp struct {
		Data struct {
			AuditLogs []handlers.AuditLogResponse `json:"audit_logs"`
			Total     int                         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, 3, resp.Data.Total, "comma-separated filter must match the 3 selected types for the account")
	types := map[string]bool{}
	for _, l := range resp.Data.AuditLogs {
		types[l.ResourceType] = true
		assert.Equal(t, accountID, l.ResourceID)
	}
	assert.True(t, types["account"])
	assert.True(t, types["settings.close_rating"])
	assert.True(t, types["settings.call_auto_reject"])
}

func TestApp_ListAuditLogs_HideContactUpdates(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := auditLogsRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	now := time.Now()
	// The flood: routine contact updates. These must be hideable without
	// hiding creates/deletes or other resources' updates.
	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "contact", models.AuditActionUpdated, now.Add(-2*time.Hour))
	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "contact", models.AuditActionCreated, now.Add(-1*time.Hour))
	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "user", models.AuditActionUpdated, now)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "hide_contact_updates", "1")

	require.NoError(t, app.ListAuditLogs(req))
	var resp struct {
		Data struct {
			AuditLogs []handlers.AuditLogResponse `json:"audit_logs"`
			Total     int                         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, 2, resp.Data.Total, "hide_contact_updates must filter out routine contact updates")
	for _, l := range resp.Data.AuditLogs {
		assert.False(t, l.ResourceType == "contact" && l.Action == models.AuditActionUpdated,
			"no routine contact update may survive the filter")
	}

	// Without the flag the full history is still listed (nothing is deleted
	// or hidden by default).
	req2 := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req2, org.ID, user.ID)
	require.NoError(t, app.ListAuditLogs(req2))
	var resp2 struct {
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req2), &resp2))
	assert.Equal(t, 3, resp2.Data.Total, "without the flag all rows must be listed")
}

// countContactAuditLogs returns the number of audit rows for one contact.
func countContactAuditLogs(t *testing.T, db *gorm.DB, contactID uuid.UUID) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.AuditLog{}).
		Where("resource_type = ? AND resource_id = ?", "contact", contactID).
		Count(&n).Error)
	return n
}

// assertNoExtraAuditRows fails as soon as more than max audit rows appear
// for the contact, polling briefly because UpdateContact's own audit write
// is async (audit.LogAudit runs in a detached goroutine).
func assertNoExtraAuditRows(t *testing.T, db *gorm.DB, contactID uuid.UUID, max int64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if n := countContactAuditLogs(t, db, contactID); n > max {
			t.Fatalf("one action wrote %d audit rows for the contact, want at most %d", n, max)
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestApp_UpdateContact_AssignmentOnly_WritesSingleAuditRow: one PUT that
// only changes the assignee must produce exactly one audit row. Before the
// fix ChatLifecycle.Assign wrote one row and UpdateContact wrote a second,
// near-identical "Updated Contact" row for the same action.
func TestApp_UpdateContact_AssignmentOnly_WritesSingleAuditRow(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	assignee := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"assigned_user_id": assignee.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.UpdateContact(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	assertNoExtraAuditRows(t, app.DB, contact.ID, 1)
}

// TestApp_UpdateContact_FieldUpdate_StillAudited: the dedupe above must not
// swallow genuine field edits — a name change with no assignment change
// still writes exactly one audit row.
func TestApp_UpdateContact_FieldUpdate_StillAudited(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"profile_name": "Renamed Contact",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.UpdateContact(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// The async audit write must land exactly once.
	require.Eventually(t, func() bool {
		return countContactAuditLogs(t, app.DB, contact.ID) == 1
	}, 2*time.Second, 50*time.Millisecond, "a field update must write exactly one audit row")
	assertNoExtraAuditRows(t, app.DB, contact.ID, 1)
}

func TestApp_ListAuditLogs_FilterByAction(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := auditLogsRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "contact", models.AuditActionCreated, time.Now())
	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "contact", models.AuditActionDeleted, time.Now())

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "action", "deleted")

	require.NoError(t, app.ListAuditLogs(req))
	var resp struct {
		Data struct {
			AuditLogs []handlers.AuditLogResponse `json:"audit_logs"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	require.Len(t, resp.Data.AuditLogs, 1)
	assert.Equal(t, models.AuditActionDeleted, resp.Data.AuditLogs[0].Action)
}

func TestApp_ListAuditLogs_DateRangeFilter(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := auditLogsRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	now := time.Now()
	old := now.Add(-72 * time.Hour)
	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "x", models.AuditActionCreated, old)
	makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "x", models.AuditActionCreated, now)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "from", now.Add(-24*time.Hour).Format(time.RFC3339))

	require.NoError(t, app.ListAuditLogs(req))
	var resp struct {
		Data struct {
			AuditLogs []handlers.AuditLogResponse `json:"audit_logs"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	require.Len(t, resp.Data.AuditLogs, 1, "from-filter must exclude logs older than 24h ago")
}

func TestApp_ListAuditLogs_CrossOrgIsolation(t *testing.T) {
	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	orgB := testutil.CreateTestOrganization(t, app.DB)
	roleA := auditLogsRole(t, app.DB, orgA.ID)
	roleB := auditLogsRole(t, app.DB, orgB.ID)
	userA := testutil.CreateTestUser(t, app.DB, orgA.ID, testutil.WithRoleID(&roleA.ID))
	userB := testutil.CreateTestUser(t, app.DB, orgB.ID, testutil.WithRoleID(&roleB.ID))

	makeAuditLog(t, app.DB, orgA.ID, userA.ID, uuid.New(), "x", models.AuditActionCreated, time.Now())
	makeAuditLog(t, app.DB, orgA.ID, userA.ID, uuid.New(), "x", models.AuditActionUpdated, time.Now())

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgB.ID, userB.ID)
	require.NoError(t, app.ListAuditLogs(req))
	var resp struct {
		Data struct {
			AuditLogs []handlers.AuditLogResponse `json:"audit_logs"`
			Total     int                         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, 0, resp.Data.Total, "audit logs from another org must not be returned")
}

func TestApp_ListAuditLogs_PermissionDenied(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	// Role has no audit_logs:read.
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "no-audit-perms", false, false, nil)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	require.NoError(t, app.ListAuditLogs(req))
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

// --- GetAuditLog ---

func TestApp_GetAuditLog_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := auditLogsRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	log := makeAuditLog(t, app.DB, org.ID, user.ID, uuid.New(), "contact", models.AuditActionUpdated, time.Now())

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", log.ID.String())

	require.NoError(t, app.GetAuditLog(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AuditLogResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, log.ID, resp.Data.ID)
	assert.Equal(t, "contact", resp.Data.ResourceType)
}

func TestApp_GetAuditLog_NotFound(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := auditLogsRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	require.NoError(t, app.GetAuditLog(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "Audit log not found")
}

func TestApp_GetAuditLog_CrossOrgIsolation(t *testing.T) {
	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	orgB := testutil.CreateTestOrganization(t, app.DB)
	roleB := auditLogsRole(t, app.DB, orgB.ID)
	userA := testutil.CreateTestUser(t, app.DB, orgA.ID)
	userB := testutil.CreateTestUser(t, app.DB, orgB.ID, testutil.WithRoleID(&roleB.ID))
	log := makeAuditLog(t, app.DB, orgA.ID, userA.ID, uuid.New(), "x", models.AuditActionCreated, time.Now())

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgB.ID, userB.ID)
	testutil.SetPathParam(req, "id", log.ID.String())

	require.NoError(t, app.GetAuditLog(req))
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req),
		"cross-org access must look like not-found")
}
