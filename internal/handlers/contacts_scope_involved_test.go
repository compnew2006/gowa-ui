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

// TestScopeAssignedContact_ClosedFindableOnlyViaGrant: close releases the
// assignment and metadata.closed_by is AUDIT-ONLY — it must NOT keep the
// conversation visible. The durable access is the assignment grant (minted by
// direct admin assignment): a closed conversation stays findable for a holder
// of an active grant, and disappears for a closed_by-only "closer" once the
// grant is revoked.
func TestScopeAssignedContact_ClosedFindableOnlyViaGrant(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	accA := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "a")))
	accB := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "b")))

	closer := scopedUser(t, app, org, "Closer", []string{"contacts:read"}, accA.ID)
	bystander := scopedUser(t, app, org, "Bystander", []string{"contacts:read"}, accA.ID)
	accountBUser := scopedUser(t, app, org, "Account B Member", []string{"contacts:read"}, accB.ID)

	closed := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accB.Name))
	assignForScopeTest(t, app, closed, closer.ID)
	// Mint the grant the way AssignContact does (direct admin assignment).
	require.NoError(t, app.DB.Create(&models.ContactAssignmentAccessGrant{
		OrganizationID: org.ID,
		ContactID:      closed.ID,
		UserID:         closer.ID,
		GrantedBy:      closer.ID,
	}).Error)
	closed.AssignedUserID = nil
	closed.SetStatus(models.ChatStatusClosed)
	closed.SetClosedBy(closer.ID.String(), "Closer")
	require.NoError(t, app.DB.Model(closed).Updates(map[string]any{
		"assigned_user_id": nil,
		"metadata":         closed.Metadata,
	}).Error)

	closerView := visibleContactIDs(t, app, org.ID, closer.ID)
	assert.True(t, closerView[closed.ID],
		"the grant holder must still find the closed conversation")

	// Revoke the grant → closed_by alone must NOT keep it visible.
	require.NoError(t, app.DB.Model(&models.ContactAssignmentAccessGrant{}).
		Where("contact_id = ? AND user_id = ?", closed.ID, closer.ID).
		Update("revoked_at", time.Now()).Error)
	revokedView := visibleContactIDs(t, app, org.ID, closer.ID)
	assert.False(t, revokedView[closed.ID],
		"after Release, closed_by must not substitute for the revoked grant")

	bystanderView := visibleContactIDs(t, app, org.ID, bystander.ID)
	assert.False(t, bystanderView[closed.ID],
		"a closed cross-account conversation stays hidden for non-involved users")

	accountBView := visibleContactIDs(t, app, org.ID, accountBUser.ID)
	assert.True(t, accountBView[closed.ID],
		"the conversation's own account users keep seeing it after close")
}

// TestScopeAssignedContact_ClosedChatVisibleThroughLifecycle: end-to-end —
// the scoped agent (holding the assignment grant) closes the cross-account
// conversation through CloseChat and can still load it afterwards through the
// scoped read path, now as read-only.
func TestScopeAssignedContact_ClosedChatVisibleThroughLifecycle(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	accA := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "a")))
	accB := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "b")))

	agent := scopedUser(t, app, org, "Handler", []string{"contacts:read", "chat:read"}, accA.ID)

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accB.Name))
	assignForScopeTest(t, app, contact, agent.ID)
	// The grant that AssignContact would mint — post-close access comes from
	// THIS, not from closed_by.
	require.NoError(t, app.DB.Create(&models.ContactAssignmentAccessGrant{
		OrganizationID: org.ID,
		ContactID:      contact.ID,
		UserID:         agent.ID,
		GrantedBy:      agent.ID,
	}).Error)

	// Close through the real endpoint (PUT /contacts/{id}/close).
	req := newPUTRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	require.NoError(t, app.CloseChat(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// The closed conversation is still in the agent's scoped list, read-only.
	visible := visibleContactIDs(t, app, org.ID, agent.ID)
	assert.True(t, visible[contact.ID],
		"after close, the grant holder must still find the conversation")
}
