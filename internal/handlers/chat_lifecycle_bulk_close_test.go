package handlers_test

import (
	"fmt"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// bulkCloseResponse mirrors the BulkCloseChats envelope.
type bulkCloseResponse struct {
	Data struct {
		ClosedIDs []string `json:"closed_ids"`
		Failed    []struct {
			ContactID string `json:"contact_id"`
			Reason    string `json:"reason"`
		} `json:"failed"`
		Requested int `json:"requested"`
	} `json:"data"`
}

func callBulkClose(t *testing.T, app *handlers.App, orgID, userID uuid.UUID, ids []string) bulkCloseResponse {
	t.Helper()
	req := newPOSTJSONRequest(t, map[string]any{"contact_ids": ids})
	testutil.SetAuthContext(req, orgID, userID)
	err := app.BulkCloseChats(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	var resp bulkCloseResponse
	testutil.ParseJSONResponse(t, req, &resp)
	return resp
}

// createCloseAgent makes a user with the seeded agent permissions
// (contacts:read included) so the caller can SEE other agents' chats — the
// per-conversation gate, not invisibility, must refuse them.
func createCloseAgent(t *testing.T, app *handlers.App, orgID uuid.UUID, fullName string) *models.User {
	t.Helper()
	role := testutil.CreateAgentRole(t, app.DB, orgID)
	return testutil.CreateTestUser(t, app.DB, orgID,
		testutil.WithRoleID(&role.ID), testutil.WithFullName(fullName))
}

func TestApp_BulkCloseChats_OwnerClosesOwn(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createCloseAgent(t, app, org.ID, "Test Agent")

	c1 := claimContactForTest(t, app, testutil.CreateTestContact(t, app.DB, org.ID), agent.ID)
	c2 := claimContactForTest(t, app, testutil.CreateTestContact(t, app.DB, org.ID), agent.ID)

	resp := callBulkClose(t, app, org.ID, agent.ID,
		[]string{c1.ID.String(), c2.ID.String()})

	assert.ElementsMatch(t,
		[]string{c1.ID.String(), c2.ID.String()}, resp.Data.ClosedIDs,
		"agent must be able to bulk-close their own open chats")
	assert.Empty(t, resp.Data.Failed)
	assert.Equal(t, 2, resp.Data.Requested)

	// DB rows: closed + assignment released + collaborators cleared.
	for _, c := range []*models.Contact{c1, c2} {
		var updated models.Contact
		require.NoError(t, app.DB.First(&updated, "id = ?", c.ID).Error)
		assert.Equal(t, models.ChatStatusClosed, updated.EffectiveStatus())
		assert.Nil(t, updated.AssignedUserID, "assigned_user_id must be cleared")
	}
}

func TestApp_BulkCloseChats_MixedBatch_PartialResults(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createCloseAgent(t, app, org.ID, "Test Agent")
	other := createCloseAgent(t, app, org.ID, "Other Agent")

	mine := claimContactForTest(t, app, testutil.CreateTestContact(t, app.DB, org.ID), agent.ID)
	// Another agent's open chat: the caller can SEE it (contacts:read, no
	// account assignment = full org visibility) but is not a participant.
	theirs := claimContactForTest(t, app, testutil.CreateTestContact(t, app.DB, org.ID), other.ID)
	// Unclaimed (pending) conversation: same gate as single close refuses it.
	pending := testutil.CreateTestContact(t, app.DB, org.ID)
	pending.SetStatus(models.ChatStatusPending)
	require.NoError(t, app.DB.Model(pending).Update("metadata", pending.Metadata).Error)

	resp := callBulkClose(t, app, org.ID, agent.ID,
		[]string{mine.ID.String(), theirs.ID.String(), pending.ID.String()})

	assert.Equal(t, []string{mine.ID.String()}, resp.Data.ClosedIDs,
		"only the caller's own open chat may close")
	require.Len(t, resp.Data.Failed, 2, "the other two must be refused, not silently skipped")
	failedByID := map[string]string{}
	for _, f := range resp.Data.Failed {
		failedByID[f.ContactID] = f.Reason
	}
	// A non-participant viewer reaches these only as read-only, so the
	// handler refuses them before the service gate.
	assert.Equal(t, "read-only", failedByID[theirs.ID.String()])
	assert.Equal(t, "read-only", failedByID[pending.ID.String()])

	// Neither refused conversation changed state.
	var theirsAfter models.Contact
	require.NoError(t, app.DB.First(&theirsAfter, "id = ?", theirs.ID).Error)
	assert.Equal(t, models.ChatStatusOpen, theirsAfter.EffectiveStatus())
	require.NotNil(t, theirsAfter.AssignedUserID)
	assert.Equal(t, other.ID, *theirsAfter.AssignedUserID)
	var pendingAfter models.Contact
	require.NoError(t, app.DB.First(&pendingAfter, "id = ?", pending.ID).Error)
	assert.Equal(t, models.ChatStatusPending, pendingAfter.EffectiveStatus())
	assert.Nil(t, pendingAfter.AssignedUserID)
}

func TestApp_BulkCloseChats_Admin_UnclaimedPending_RefusedNotOpen(t *testing.T) {
	// The admin passes the handler's scope/read-only pre-filter (admins see
	// everything), so the unclaimed conversation reaches the service gate —
	// which refuses it for not being open. The other agent's open chat closes
	// under the admin/manager branch of the same gate.
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	agent := createCloseAgent(t, app, org.ID, "Test Agent")

	theirs := claimContactForTest(t, app, testutil.CreateTestContact(t, app.DB, org.ID), agent.ID)
	pending := testutil.CreateTestContact(t, app.DB, org.ID)
	pending.SetStatus(models.ChatStatusPending)
	require.NoError(t, app.DB.Model(pending).Update("metadata", pending.Metadata).Error)

	resp := callBulkClose(t, app, org.ID, admin.ID,
		[]string{theirs.ID.String(), pending.ID.String()})

	assert.Equal(t, []string{theirs.ID.String()}, resp.Data.ClosedIDs)
	require.Len(t, resp.Data.Failed, 1)
	assert.Equal(t, pending.ID.String(), resp.Data.Failed[0].ContactID)
	assert.Equal(t, "not open", resp.Data.Failed[0].Reason)

	var pendingAfter models.Contact
	require.NoError(t, app.DB.First(&pendingAfter, "id = ?", pending.ID).Error)
	assert.Equal(t, models.ChatStatusPending, pendingAfter.EffectiveStatus())
}

func TestApp_BulkCloseChats_CollaboratorCloses(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	owner := createCloseAgent(t, app, org.ID, "Owner Agent")
	collab := createCloseAgent(t, app, org.ID, "Collaborator Agent")

	contact := claimContactForTest(t, app, testutil.CreateTestContact(t, app.DB, org.ID), owner.ID)
	contact.AddCollaborator(models.Collaborator{UserID: collab.ID.String(), Name: "Collaborator Agent"})
	require.NoError(t, app.DB.Model(contact).Update("metadata", contact.Metadata).Error)

	resp := callBulkClose(t, app, org.ID, collab.ID, []string{contact.ID.String()})

	assert.Equal(t, []string{contact.ID.String()}, resp.Data.ClosedIDs)
	assert.Empty(t, resp.Data.Failed)

	var updated models.Contact
	require.NoError(t, app.DB.First(&updated, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusClosed, updated.EffectiveStatus())
}

func TestApp_BulkCloseChats_AlreadyClosed_Failed(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createCloseAgent(t, app, org.ID, "Test Agent")

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	contact.AssignedUserID = &agent.ID
	contact.SetStatus(models.ChatStatusClosed)
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"assigned_user_id": agent.ID,
		"metadata":         contact.Metadata,
	}).Error)

	resp := callBulkClose(t, app, org.ID, agent.ID, []string{contact.ID.String()})

	assert.Empty(t, resp.Data.ClosedIDs)
	require.Len(t, resp.Data.Failed, 1)
	assert.Equal(t, "already closed", resp.Data.Failed[0].Reason)

	var updated models.Contact
	require.NoError(t, app.DB.First(&updated, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusClosed, updated.EffectiveStatus())
}

func TestApp_BulkCloseChats_DuplicateIds_Deduped(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createCloseAgent(t, app, org.ID, "Test Agent")

	contact := claimContactForTest(t, app, testutil.CreateTestContact(t, app.DB, org.ID), agent.ID)

	resp := callBulkClose(t, app, org.ID, agent.ID,
		[]string{contact.ID.String(), contact.ID.String()})

	assert.Equal(t, []string{contact.ID.String()}, resp.Data.ClosedIDs,
		"a duplicated id must close (and be reported) once")
	assert.Empty(t, resp.Data.Failed)
}

func TestApp_BulkCloseChats_InvalidUUID_FailedNotFatal(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createCloseAgent(t, app, org.ID, "Test Agent")

	mine := claimContactForTest(t, app, testutil.CreateTestContact(t, app.DB, org.ID), agent.ID)

	resp := callBulkClose(t, app, org.ID, agent.ID,
		[]string{"not-a-uuid", mine.ID.String()})

	assert.Equal(t, []string{mine.ID.String()}, resp.Data.ClosedIDs)
	require.Len(t, resp.Data.Failed, 1)
	assert.Equal(t, "not-a-uuid", resp.Data.Failed[0].ContactID)
	assert.Equal(t, "invalid uuid", resp.Data.Failed[0].Reason)
}

func TestApp_BulkCloseChats_EmptyList_BadRequest(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	req := newPOSTJSONRequest(t, map[string]any{"contact_ids": []string{}})
	testutil.SetAuthContext(req, org.ID, admin.ID)

	_ = app.BulkCloseChats(req)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "contact_ids")
}

func TestApp_BulkCloseChats_OverCap_BadRequest(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	ids := make([]string, 0, 501)
	for i := 0; i < 501; i++ {
		ids = append(ids, fmt.Sprintf("00000000-0000-0000-0000-%012d", i))
	}
	req := newPOSTJSONRequest(t, map[string]any{"contact_ids": ids})
	testutil.SetAuthContext(req, org.ID, admin.ID)

	_ = app.BulkCloseChats(req)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "500")
}
