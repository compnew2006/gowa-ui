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

// Visibility-matrix tests for the involvement grant in scopeAssignedContact:
// being the assignee, a collaborator, or the agent who closed a conversation
// makes that ONE conversation visible even under a WhatsApp account the user
// is not assigned to, while everything else outside the user's accounts stays
// hidden.

// accName builds an account name unique to the test's org — WhatsAppAccount
// names are globally unique (idx_wa_org_name), so tests sharing the DB must
// not reuse the same literal names.
func accName(org *models.Organization, suffix string) string {
	return "scope-" + suffix + "-" + org.ID.String()[:8]
}

// scopedUser creates a user with the given permission keys and WhatsApp
// account assignments — the production /chat setup (agents assigned a subset
// of accounts via user_whatsapp_accounts).
func scopedUser(t *testing.T, app *handlers.App, org *models.Organization, name string, permKeys []string, accountIDs ...uuid.UUID) *models.User {
	t.Helper()
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "role-"+name, permKeys)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithRoleID(&role.ID), testutil.WithFullName(name))
	for _, accID := range accountIDs {
		testutil.AssignAccountToUser(t, app.DB, user.ID, accID)
	}
	return user
}

// visibleContactIDs runs ListContacts as the user and returns the set of
// contact IDs the visibility scope returned.
func visibleContactIDs(t *testing.T, app *handlers.App, orgID, userID uuid.UUID) map[uuid.UUID]bool {
	t.Helper()
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgID, userID)
	require.NoError(t, app.ListContacts(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Contacts []handlers.ContactResponse `json:"contacts"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	ids := make(map[uuid.UUID]bool, len(resp.Data.Contacts))
	for _, c := range resp.Data.Contacts {
		ids[c.ID] = true
	}
	return ids
}

// assignForScopeTest puts a contact into the assigned/open state, mirroring
// what AssignContact does.
func assignForScopeTest(t *testing.T, app *handlers.App, c *models.Contact, userID uuid.UUID) {
	t.Helper()
	c.AssignedUserID = &userID
	c.SetStatus(models.ChatStatusOpen)
	require.NoError(t, app.DB.Model(c).Updates(map[string]any{
		"assigned_user_id": userID,
		"metadata":         c.Metadata,
	}).Error)
}

// TestScopeAssignedContact_CrossAccountAssignment: the production scenario —
// an agent scoped to account A is assigned ONE conversation from account B.
// That conversation becomes visible (and actionable) to them; the REST of
// account B stays hidden.
func TestScopeAssignedContact_CrossAccountAssignment(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	accA := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "a")))
	accB := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "b")))

	reader := scopedUser(t, app, org, "Scoped Reader",
		[]string{"contacts:read"}, accA.ID)
	otherAgent := scopedUser(t, app, org, "Account B Agent",
		[]string{"contacts:read"}, accB.ID)

	assignedToReader := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accB.Name))
	assignForScopeTest(t, app, assignedToReader, reader.ID)

	uninvolvedB := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accB.Name))
	ownedByOtherB := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accB.Name))
	assignForScopeTest(t, app, ownedByOtherB, otherAgent.ID)
	ownAccountA := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accA.Name))

	visible := visibleContactIDs(t, app, org.ID, reader.ID)
	assert.True(t, visible[assignedToReader.ID],
		"a conversation assigned to the user must be visible even outside their accounts")
	assert.True(t, visible[ownAccountA.ID],
		"own-account conversations stay visible")
	assert.False(t, visible[uninvolvedB.ID],
		"uninvolved cross-account conversations must stay hidden")
	assert.False(t, visible[ownedByOtherB.ID],
		"cross-account conversations assigned to someone else must stay hidden")
}

// TestScopeAssignedContact_NonReaderInvolvementOnly: without contacts:read,
// involvement (assignment) is the sole source of access — but it works across
// account boundaries, so a cross-account assignment still shows the chat.
func TestScopeAssignedContact_NonReaderInvolvementOnly(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	accA := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "a")))
	accB := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "b")))

	agent := scopedUser(t, app, org, "Scoped Agent",
		[]string{"chat:read"}, accA.ID)

	crossAccountAssigned := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accB.Name))
	assignForScopeTest(t, app, crossAccountAssigned, agent.ID)
	ownAccountUnassigned := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accA.Name))

	visible := visibleContactIDs(t, app, org.ID, agent.ID)
	assert.True(t, visible[crossAccountAssigned.ID],
		"a cross-account conversation assigned to the agent must be visible without contacts:read")
	assert.False(t, visible[ownAccountUnassigned.ID],
		"without contacts:read, unassigned own-account conversations stay hidden")
}

// TestScopeAssignedContact_ClosedByKeepsClosedChatFindable: close releases
// the assignment; metadata.closed_by keeps the closed conversation searchable
// for the agent who handled it (parity with users of that account), while it
// stays hidden for everyone else outside the account.
func TestScopeAssignedContact_ClosedByKeepsClosedChatFindable(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	accA := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "a")))
	accB := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "b")))

	closer := scopedUser(t, app, org, "Closer", []string{"contacts:read"}, accA.ID)
	bystander := scopedUser(t, app, org, "Bystander", []string{"contacts:read"}, accA.ID)
	accountBUser := scopedUser(t, app, org, "Account B Member", []string{"contacts:read"}, accB.ID)

	closed := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accB.Name))
	assignForScopeTest(t, app, closed, closer.ID)
	closed.AssignedUserID = nil
	closed.SetStatus(models.ChatStatusClosed)
	closed.SetClosedBy(closer.ID.String(), "Closer")
	require.NoError(t, app.DB.Model(closed).Updates(map[string]any{
		"assigned_user_id": nil,
		"metadata":         closed.Metadata,
	}).Error)

	closerView := visibleContactIDs(t, app, org.ID, closer.ID)
	assert.True(t, closerView[closed.ID],
		"the agent who closed the conversation must still find it (closed_by grant)")

	bystanderView := visibleContactIDs(t, app, org.ID, bystander.ID)
	assert.False(t, bystanderView[closed.ID],
		"a closed cross-account conversation stays hidden for non-involved users")

	accountBView := visibleContactIDs(t, app, org.ID, accountBUser.ID)
	assert.True(t, accountBView[closed.ID],
		"the conversation's own account users keep seeing it after close")
}

// TestScopeAssignedContact_ClosedChatVisibleThroughLifecycle: end-to-end —
// the scoped agent closes the cross-account conversation through CloseChat and
// can still load it afterwards through the scoped read path (the same path
// GetMessages/search use).
func TestScopeAssignedContact_ClosedChatVisibleThroughLifecycle(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	accA := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "a")))
	accB := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "b")))

	agent := scopedUser(t, app, org, "Handler", []string{"contacts:read", "chat:read"}, accA.ID)

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accB.Name))
	assignForScopeTest(t, app, contact, agent.ID)

	// Close through the real endpoint (PUT /contacts/{id}/close).
	req := newPUTRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	require.NoError(t, app.CloseChat(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// The closed conversation is still in the agent's scoped list.
	visible := visibleContactIDs(t, app, org.ID, agent.ID)
	assert.True(t, visible[contact.ID],
		"after close, the handler must still find the conversation via closed_by")
}
