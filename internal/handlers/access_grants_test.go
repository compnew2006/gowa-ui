package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// Assignment access-grant tests: direct admin assignment creates a permanent
// per-conversation grant; the grant survives close and reassignment as
// READ-ONLY access; Release (revoke) is the only way to end it.

// grantFixture is the standard scenario factory: account A (user's own) +
// account B (foreign), an admin who can assign, and a contact under B.
type grantFixture struct {
	app     *handlers.App
	org     *models.Organization
	admin   *models.User
	agent   *models.User // scoped to account A only
	other   *models.User // second account-A agent ( bystander )
	accA    *models.WhatsAppAccount
	accB    *models.WhatsAppAccount
	contact *models.Contact // under account B
}

func newGrantFixture(t *testing.T) *grantFixture {
	t.Helper()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	accA := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "a")))
	accB := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "b")))

	adminRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "grants-admin",
		[]string{"contacts:read", "contacts:write", "chat.assign:write"})
	admin := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithRoleID(&adminRole.ID), testutil.WithFullName("Grants Admin"))

	// The agent gets a broad role so every refusal below provably comes from
	// the read-only gate, not from a missing role permission.
	agentRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "grants-agent",
		[]string{"contacts:read", "contacts:write", "chat:read", "chat:write", "chat.assign:write", "chat.collaborate:write"})
	agent := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithRoleID(&agentRole.ID), testutil.WithFullName("Scoped Agent"))
	testutil.AssignAccountToUser(t, app.DB, agent.ID, accA.ID)

	other := scopedUser(t, app, org, "Bystander Agent", []string{"contacts:read"}, accA.ID)

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accB.Name))

	return &grantFixture{app: app, org: org, admin: admin, agent: agent, other: other, accA: accA, accB: accB, contact: contact}
}

// assignViaAPI runs PUT /contacts/{id}/assign as the admin.
func (f *grantFixture) assignViaAPI(t *testing.T, target *models.User) {
	t.Helper()
	req := testutil.NewJSONRequest(t, map[string]any{"user_id": target.ID.String()})
	req.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(req, f.org.ID, f.admin.ID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	require.NoError(t, f.app.AssignContact(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"admin assignment must succeed, got: %s", string(testutil.GetResponseBody(req)))
}

func (f *grantFixture) activeGrantCount(t *testing.T, userID uuid.UUID) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.app.DB.Model(&models.ContactAssignmentAccessGrant{}).
		Where("contact_id = ? AND user_id = ? AND revoked_at IS NULL", f.contact.ID, userID).
		Count(&n).Error)
	return n
}

// loadFresh reloads the contact from the DB.
func (f *grantFixture) loadFresh(t *testing.T) *models.Contact {
	t.Helper()
	var c models.Contact
	require.NoError(t, f.app.DB.First(&c, "id = ?", f.contact.ID).Error)
	return &c
}

// TestAssignContact_CreatesGrantOnce: direct admin assignment creates exactly
// one active grant; repeating the assignment stays at one.
func TestAssignContact_CreatesGrantOnce(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)
	assert.EqualValues(t, 1, f.activeGrantCount(t, f.agent.ID), "assignment must create exactly one grant")

	f.assignViaAPI(t, f.agent) // idempotent repeat
	assert.EqualValues(t, 1, f.activeGrantCount(t, f.agent.ID), "repeat assignment must not duplicate the grant")
}

// TestReAssignAfterRelease_ReactivatesGrant: Release (revoke) then re-assign
// the same user — the idempotent-looking assignment must reactivate the grant.
func TestReAssignAfterRelease_ReactivatesGrant(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)

	del := testutil.NewRequest(t)
	del.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(del, f.org.ID, f.admin.ID)
	testutil.SetPathParam(del, "id", f.contact.ID.String())
	testutil.SetPathParam(del, "target_user_id", f.agent.ID.String())
	require.NoError(t, f.app.RevokeAccessGrant(del))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(del))
	assert.EqualValues(t, 0, f.activeGrantCount(t, f.agent.ID), "release must revoke")

	f.assignViaAPI(t, f.agent) // back to the same user (idempotent branch)
	assert.EqualValues(t, 1, f.activeGrantCount(t, f.agent.ID),
		"re-assigning the same user must reactivate the grant")
}

// TestClaim_DoesNotCreateGrant: claiming (the non-admin path) must never mint
// a permanent grant.
func TestClaim_DoesNotCreateGrant(t *testing.T) {
	f := newGrantFixture(t)
	// Account-B agent claims their own account's pending contact.
	bAgent := scopedUser(t, f.app, f.org, "B Agent", []string{"contacts:read", "chat.assign:write", "chat:read"}, f.accB.ID)

	req := newPUTRequest(t)
	testutil.SetAuthContext(req, f.org.ID, bAgent.ID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	require.NoError(t, f.app.ClaimChat(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	assert.EqualValues(t, 0, f.activeGrantCount(t, bAgent.ID), "claim must not create an access grant")
}

// TestGrant_PersistsAfterCloseAndReassign: close then reassign to someone
// else — the original holder keeps ONE active grant and read-only access.
func TestGrant_PersistsAfterCloseAndReassign(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)

	// Agent closes it (allowed — current assignee).
	req := newPUTRequest(t)
	testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	require.NoError(t, f.app.CloseChat(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Admin reopens + reassigns to someone else.
	fresh := f.loadFresh(t)
	require.NoError(t, f.app.DB.Model(&models.Contact{}).Where("id = ?", fresh.ID).
		Update("metadata", map[string]any{"chat_status": "open"}).Error)
	f.assignViaAPI(t, f.other)

	assert.EqualValues(t, 1, f.activeGrantCount(t, f.agent.ID),
		"close + reassign must keep the original holder's grant")

	// Still visible to the agent (historical read-only), with the right mode.
	visible := visibleContactIDs(t, f.app, f.org.ID, f.agent.ID)
	assert.True(t, visible[f.contact.ID], "grant holder must still see the conversation")

	mode := accessModeFor(t, f.app, f.org.ID, f.agent.ID, f.contact.ID)
	assert.Equal(t, "historical_assignment_read_only", mode)
}

// TestHistoricalGrant_ReadOnlyMatrix: after the assignment moves on, the
// grant holder may search and read but EVERY write/lifecycle action is 403 —
// even with a role that would otherwise allow it.
func TestHistoricalGrant_ReadOnlyMatrix(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)
	f.assignViaAPI(t, f.other) // assignment moves on; agent keeps grant only

	// Read path: GetMessages is allowed.
	getReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(getReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(getReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.GetMessages(getReq))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(getReq),
		"historical grant holder must be able to READ messages")

	// Write paths: every one 403 with the read-only envelope.
	sendReq := testutil.NewJSONRequest(t, map[string]any{
		"type": "text", "content": map[string]any{"body": "hi"},
	})
	testutil.SetAuthContext(sendReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(sendReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.SendMessage(sendReq))
	testutil.AssertErrorResponse(t, sendReq, fasthttp.StatusForbidden, "Read-only access")

	typingReq := testutil.NewJSONRequest(t, map[string]any{"action": "start"})
	testutil.SetAuthContext(typingReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(typingReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.SendTypingIndicator(typingReq))
	testutil.AssertErrorResponse(t, typingReq, fasthttp.StatusForbidden, "Read-only access")

	claimReq := newPUTRequest(t)
	testutil.SetAuthContext(claimReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(claimReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.ClaimChat(claimReq))
	testutil.AssertErrorResponse(t, claimReq, fasthttp.StatusForbidden, "Read-only access")

	closeReq := newPUTRequest(t)
	testutil.SetAuthContext(closeReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(closeReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.CloseChat(closeReq))
	testutil.AssertErrorResponse(t, closeReq, fasthttp.StatusForbidden, "Read-only access")

	reopenReq := newPUTRequest(t)
	testutil.SetAuthContext(reopenReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(reopenReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.ReopenChat(reopenReq))
	testutil.AssertErrorResponse(t, reopenReq, fasthttp.StatusForbidden, "Read-only access")

	// Self-assignment loophole: historical holder assigning to themselves.
	selfReq := testutil.NewJSONRequest(t, map[string]any{"user_id": f.agent.ID.String()})
	selfReq.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(selfReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(selfReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.AssignContact(selfReq))
	testutil.AssertErrorResponse(t, selfReq, fasthttp.StatusForbidden, "Read-only access")

	// Contact mutation (tags) also refused.
	tagsReq := testutil.NewJSONRequest(t, map[string]any{"tags": []string{"x"}})
	tagsReq.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(tagsReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(tagsReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.UpdateContactTags(tagsReq))
	testutil.AssertErrorResponse(t, tagsReq, fasthttp.StatusForbidden, "Read-only access")
}

// TestCurrentAssignee_FullAccess: while the grant holder IS the assignee, the
// read-only gate must not fire (close succeeds).
func TestCurrentAssignee_FullAccess(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)

	// While still the assignee (before closing): full access mode.
	mode := accessModeFor(t, f.app, f.org.ID, f.agent.ID, f.contact.ID)
	assert.Equal(t, "current_assignee", mode)

	req := newPUTRequest(t)
	testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	require.NoError(t, f.app.CloseChat(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"current assignee must be able to close")
}

// TestRevokeAccessGrant_OpenAssignee: releasing the CURRENT assignee of an
// OPEN conversation → pending + unassigned + grant revoked + agent loses
// visibility entirely.
func TestRevokeAccessGrant_OpenAssignee(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(req, f.org.ID, f.admin.ID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	testutil.SetPathParam(req, "target_user_id", f.agent.ID.String())
	require.NoError(t, f.app.RevokeAccessGrant(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	fresh := f.loadFresh(t)
	assert.Nil(t, fresh.AssignedUserID, "release must unassign")
	assert.Equal(t, models.ChatStatusPending, fresh.EffectiveStatus(), "open conversation becomes pending")
	assert.EqualValues(t, 0, f.activeGrantCount(t, f.agent.ID), "grant must be revoked")

	visible := visibleContactIDs(t, f.app, f.org.ID, f.agent.ID)
	assert.False(t, visible[f.contact.ID], "released user must lose visibility entirely")
}

// TestRevokeAccessGrant_ClosedAssignee: releasing the assignee of a CLOSED
// conversation keeps it closed with closed_by intact.
func TestRevokeAccessGrant_ClosedAssignee(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)

	closeReq := newPUTRequest(t)
	testutil.SetAuthContext(closeReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(closeReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.CloseChat(closeReq))

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(req, f.org.ID, f.admin.ID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	testutil.SetPathParam(req, "target_user_id", f.agent.ID.String())
	require.NoError(t, f.app.RevokeAccessGrant(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	fresh := f.loadFresh(t)
	assert.Equal(t, models.ChatStatusClosed, fresh.EffectiveStatus(), "closed stays closed")
	assert.Equal(t, f.agent.ID.String(), fresh.ClosedByUserID(), "closed_by stamp preserved")
	assert.Nil(t, fresh.AssignedUserID)
	assert.EqualValues(t, 0, f.activeGrantCount(t, f.agent.ID))
}

// TestRevokeAccessGrant_OtherAssignee: when assigned to someone else, only
// the target's grant is revoked — the current assignee is untouched.
func TestRevokeAccessGrant_OtherAssignee(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)
	f.assignViaAPI(t, f.other) // now assigned to other; both hold grants

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(req, f.org.ID, f.admin.ID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	testutil.SetPathParam(req, "target_user_id", f.agent.ID.String())
	require.NoError(t, f.app.RevokeAccessGrant(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	fresh := f.loadFresh(t)
	require.NotNil(t, fresh.AssignedUserID)
	assert.Equal(t, f.other.ID, *fresh.AssignedUserID, "current assignee untouched")
	assert.EqualValues(t, 1, f.activeGrantCount(t, f.other.ID), "other's grant untouched")
	assert.EqualValues(t, 0, f.activeGrantCount(t, f.agent.ID), "target's grant revoked")
}

// TestRevokeAccessGrant_IdempotentAndGuards: second revoke is a 200 no-op; a
// permission-less user gets 403; another org's admin gets 404.
func TestRevokeAccessGrant_IdempotentAndGuards(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)

	del := func(as *models.User, target *models.User) int {
		req := testutil.NewRequest(t)
		req.RequestCtx.Request.Header.SetMethod("DELETE")
		testutil.SetAuthContext(req, f.org.ID, as.ID)
		testutil.SetPathParam(req, "id", f.contact.ID.String())
		testutil.SetPathParam(req, "target_user_id", target.ID.String())
		require.NoError(t, f.app.RevokeAccessGrant(req))
		return testutil.GetResponseStatusCode(req)
	}

	require.Equal(t, fasthttp.StatusOK, del(f.admin, f.agent))
	assert.Equal(t, fasthttp.StatusOK, del(f.admin, f.agent), "second revoke is an idempotent 200")

	// Permission-less user: the grant-management permission check runs before
	// the scoped load, so the refusal is 403 (no existence leak either way).
	viewer := scopedUser(t, f.app, f.org, "No Perm Viewer", []string{"contacts:read"}, f.accA.ID)
	assert.Equal(t, fasthttp.StatusForbidden, del(viewer, f.agent))

	// Cross-org isolation: another org's admin sees nothing.
	org2 := testutil.CreateTestOrganization(t, f.app.DB)
	adminRole2 := testutil.CreateAdminRole(t, f.app.DB, org2.ID)
	admin2 := testutil.CreateTestUser(t, f.app.DB, org2.ID, testutil.WithRoleID(&adminRole2.ID))
	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(req, org2.ID, admin2.ID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	testutil.SetPathParam(req, "target_user_id", f.agent.ID.String())
	require.NoError(t, f.app.RevokeAccessGrant(req))
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req), "cross-org must be isolated")
}

// TestListAccessGrants: the assign dialog's data source.
func TestListAccessGrants(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)
	f.assignViaAPI(t, f.other)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, f.org.ID, f.admin.ID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	require.NoError(t, f.app.ListAccessGrants(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			AccessGrants []struct {
				UserID     string `json:"user_id"`
				UserName   string `json:"user_name"`
				IsAssignee bool   `json:"is_current_assignee"`
			} `json:"access_grants"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	require.Len(t, resp.Data.AccessGrants, 2, "both assignees hold grants")

	byUser := map[string]bool{}
	assignee := ""
	for _, g := range resp.Data.AccessGrants {
		byUser[g.UserName] = true
		if g.IsAssignee {
			assignee = g.UserID
		}
	}
	assert.True(t, byUser["Scoped Agent"])
	assert.True(t, byUser["Bystander Agent"])
	assert.Equal(t, f.other.ID.String(), assignee, "only the current assignee is flagged")
}

// accessModeFor fetches the single contact response and returns access_mode.
func accessModeFor(t *testing.T, app *handlers.App, orgID, userID uuid.UUID, contactID uuid.UUID) string {
	t.Helper()
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgID, userID)
	testutil.SetPathParam(req, "id", contactID.String())
	require.NoError(t, app.GetContact(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	var resp struct {
		Data struct {
			AccessMode string `json:"access_mode"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	return resp.Data.AccessMode
}

// TestAccessModes_InListResponse: the list response carries the mode + flags
// for each of the three states.
func TestAccessModes_InListResponse(t *testing.T) {
	f := newGrantFixture(t)
	// Own-account contact for the agent → standard.
	ownContact := testutil.CreateTestContactWith(t, f.app.DB, f.org.ID, testutil.WithContactAccount(f.accA.Name))
	f.assignViaAPI(t, f.agent)

	listModes := func(user *models.User) map[string]handlers.ContactResponse {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, f.org.ID, user.ID)
		require.NoError(t, f.app.ListContacts(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		out := map[string]handlers.ContactResponse{}
		for _, c := range resp.Data.Contacts {
			out[c.ID.String()] = c
		}
		return out
	}

	modes := listModes(f.agent)
	assert.Equal(t, "standard", modes[ownContact.ID.String()].AccessMode)
	assert.True(t, modes[ownContact.ID.String()].CanReply)
	assigned := modes[f.contact.ID.String()]
	assert.Equal(t, "current_assignee", assigned.AccessMode)
	assert.True(t, assigned.CanReply)
	assert.True(t, assigned.CanClose)

	// Move assignment on → historical read-only.
	f.assignViaAPI(t, f.other)
	modes = listModes(f.agent)
	historical := modes[f.contact.ID.String()]
	assert.Equal(t, "historical_assignment_read_only", historical.AccessMode)
	assert.False(t, historical.CanReply)
	assert.False(t, historical.CanClose)
}
