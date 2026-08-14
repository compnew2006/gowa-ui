package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// newPUTRequest builds a fastglue PUT request for the chat-lifecycle tests.
// The route layer injects the {id} path param via SetUserValue; tests mirror
// that by calling SetPathParam(req, "id", contactID).
func newPUTRequest(t *testing.T) *fastglue.Request {
	t.Helper()
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("PUT")
	return &fastglue.Request{RequestCtx: ctx}
}

// newPOSTJSONRequest builds a fastglue POST request with a JSON body (for the
// bulk-release endpoint).
func newPOSTJSONRequest(t *testing.T, body any) *fastglue.Request {
	t.Helper()
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.Header.SetMethod("POST")
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		ctx.Request.SetBody(data)
	}
	return &fastglue.Request{RequestCtx: ctx}
}

// claimContactForTest sets a contact into the "assigned/open" state in the DB,
// mirroring what ClaimChat does, so release tests start from a realistic row.
// Returns a fresh copy of the contact after the write.
func claimContactForTest(t *testing.T, app *handlers.App, c *models.Contact, assigneeID uuid.UUID) *models.Contact {
	t.Helper()
	c.AssignedUserID = &assigneeID
	c.SetStatus(models.ChatStatusOpen)
	// Persist metadata + assignment atomically (same shape as ReleaseChat).
	require.NoError(t, app.DB.Model(c).Updates(map[string]any{
		"assigned_user_id": assigneeID,
		"metadata":         c.Metadata,
	}).Error)
	var fresh models.Contact
	require.NoError(t, app.DB.First(&fresh, "id = ?", c.ID).Error)
	return &fresh
}

// countAuditEntriesFor returns the number of audit_log rows for a given
// (resource_type, resource_id). Used to assert the extraChanges safeguard held
// — without it, audit.LogAudit silently drops the entry.
// countAuditEntriesFor returns the audit_log row count for a (resource_type,
// resource_id) pair. It polls briefly because audit.LogAudit writes
// asynchronously in a detached goroutine (internal/audit/audit.go:146) — a
// read immediately after the handler returns can race the write, which
// became visible after the chat-lifecycle extraction shifted call timing.
func countAuditEntriesFor(t *testing.T, app *handlers.App, resourceType string, resourceID uuid.UUID) int64 {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		var n int64
		require.NoError(t, app.DB.Model(&models.AuditLog{}).
			Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
			Count(&n).Error)
		if n > 0 || time.Now().After(deadline) {
			return n
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// createAssignAgent makes a user whose role grants chat.assign:write (the
// permission ReleaseChat requires via requireAuth). Reused across the release
// tests so each gets an isolated agent + role pair.
func createAssignAgent(t *testing.T, app *handlers.App, orgID uuid.UUID, fullName string) *models.User {
	t.Helper()
	role := testutil.CreateTestRoleWithKeys(t, app.DB, orgID, "assign-agent",
		[]string{"chat.assign:write", "chat:read"})
	return testutil.CreateTestUser(t, app.DB, orgID,
		testutil.WithRoleID(&role.ID), testutil.WithFullName(fullName))
}

// --- ReleaseChat ---

func TestApp_ReleaseChat_Assignee_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createAssignAgent(t, app, org.ID, "Test Agent")

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	contact = claimContactForTest(t, app, contact, agent.ID)

	req := newPUTRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ReleaseChat(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// DB row: unassigned + pending + collaborators cleared + last_message_at bumped.
	var updated models.Contact
	require.NoError(t, app.DB.First(&updated, "id = ?", contact.ID).Error)
	assert.Nil(t, updated.AssignedUserID, "assigned_user_id must be cleared")
	assert.Equal(t, models.ChatStatusPending, updated.EffectiveStatus(),
		"chat_status must revert to pending")
	assert.NotZero(t, updated.LastMessageAt, "last_message_at must be bumped so the chat re-sorts")

	// Audit entry persisted (the extraChanges safeguard is what makes this pass).
	assert.EqualValues(t, 1, countAuditEntriesFor(t, app, "contact", contact.ID),
		"release must persist exactly one audit entry")
}

func TestApp_ReleaseChat_NonOwnerNonAdmin_Forbidden(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	owner := createAssignAgent(t, app, org.ID, "Owner Agent")
	other := createAssignAgent(t, app, org.ID, "Other Agent")

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	contact = claimContactForTest(t, app, contact, owner.ID)

	req := newPUTRequest(t)
	testutil.SetAuthContext(req, org.ID, other.ID) // different agent
	testutil.SetPathParam(req, "id", contact.ID.String())

	_ = app.ReleaseChat(req)
	// loadContactByPath routes through scopeAssignedContact: an agent
	// without contacts:read cannot even see another agent's chat, so the
	// release is refused as "not found" (no existence leak).
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "Contact not found")
	// Contact must be untouched.
	var updated models.Contact
	require.NoError(t, app.DB.First(&updated, "id = ?", contact.ID).Error)
	require.NotNil(t, updated.AssignedUserID)
	assert.Equal(t, owner.ID, *updated.AssignedUserID)
}

func TestApp_ReleaseChat_ClosedChat_ByAgent_Forbidden(t *testing.T) {
	// G2: an agent who is the assignee of a CLOSED chat must NOT be able to
	// release it — that would silently transition closed → pending and lose the
	// closed/cleared state. Only admins/managers may release a closed chat.
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createAssignAgent(t, app, org.ID, "Owner Agent")

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	contact.AssignedUserID = &agent.ID
	contact.SetStatus(models.ChatStatusClosed)
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"assigned_user_id": agent.ID,
		"metadata":         contact.Metadata,
	}).Error)

	req := newPUTRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	_ = app.ReleaseChat(req)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "closed")
	// State untouched.
	var updated models.Contact
	require.NoError(t, app.DB.First(&updated, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusClosed, updated.EffectiveStatus())
}

func TestApp_ReleaseChat_AlreadyPending_Idempotent(t *testing.T) {
	// Idempotency: the assignee re-releases a chat that is already pending +
	// unassigned (e.g. a WS race landed the state change before the PUT arrived).
	// Must be a safe no-op success: no duplicate system message, no duplicate
	// audit entry. An admin is used here because a non-admin cannot target a
	// chat they don't own (the auth guard correctly 403s that — covered by
	// TestApp_ReleaseChat_NonOwnerNonAdmin_Forbidden). The idempotent branch
	// is the same code path regardless of who reaches it.
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	// Already pending + unassigned.
	contact.SetStatus(models.ChatStatusPending)
	require.NoError(t, app.DB.Model(contact).Update("metadata", contact.Metadata).Error)

	before := countAuditEntriesFor(t, app, "contact", contact.ID)
	req := newPUTRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ReleaseChat(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	// No new audit entry on the idempotent path — must not duplicate the trail.
	assert.EqualValues(t, before, countAuditEntriesFor(t, app, "contact", contact.ID),
		"idempotent release must not write a duplicate audit entry")
}

func TestApp_ReleaseChat_NotFound(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	req := newPUTRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", uuid.New().String()) // nonexistent

	_ = app.ReleaseChat(req)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "not found")
}

// --- BulkReleaseChats ---

func TestApp_BulkReleaseChats_OwnerReleasesOwn(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createAssignAgent(t, app, org.ID, "Test Agent")

	c1 := claimContactForTest(t, app, testutil.CreateTestContact(t, app.DB, org.ID), agent.ID)
	c2 := claimContactForTest(t, app, testutil.CreateTestContact(t, app.DB, org.ID), agent.ID)
	// A third chat owned by someone else — must show up in `failed`, not released.
	other := createAssignAgent(t, app, org.ID, "Other Agent")
	c3 := claimContactForTest(t, app, testutil.CreateTestContact(t, app.DB, org.ID), other.ID)

	req := newPOSTJSONRequest(t, map[string]any{
		"contact_ids": []string{c1.ID.String(), c2.ID.String(), c3.ID.String()},
	})
	testutil.SetAuthContext(req, org.ID, agent.ID)

	err := app.BulkReleaseChats(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			ReleasedIDs []string `json:"released_ids"`
			Failed      []struct {
				ContactID string `json:"contact_id"`
				Reason    string `json:"reason"`
			} `json:"failed"`
		} `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)

	assert.ElementsMatch(t,
		[]string{c1.ID.String(), c2.ID.String()}, resp.Data.ReleasedIDs,
		"agent must be able to bulk-release their own chats")
	require.Len(t, resp.Data.Failed, 1, "the other-agent chat must fail, not silently release")
	assert.Equal(t, c3.ID.String(), resp.Data.Failed[0].ContactID)
	// The handler pre-filters the batch through scopeAssignedContact: an
	// agent without contacts:read cannot see another agent's chat, so it is
	// reported as "not found" rather than "not authorized".
	assert.Equal(t, "not found", resp.Data.Failed[0].Reason)

	// c3 stays assigned to `other`.
	var c3After models.Contact
	require.NoError(t, app.DB.First(&c3After, "id = ?", c3.ID).Error)
	require.NotNil(t, c3After.AssignedUserID)
	assert.Equal(t, other.ID, *c3After.AssignedUserID)
}

func TestApp_BulkReleaseChats_EmptyList_BadRequest(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	req := newPOSTJSONRequest(t, map[string]any{"contact_ids": []string{}})
	testutil.SetAuthContext(req, org.ID, admin.ID)

	_ = app.BulkReleaseChats(req)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "contact_ids")
}
