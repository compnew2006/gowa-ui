package handlers_test

import (
	"testing"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Agent-initiated transfer tests: the current assignee hands an open
// conversation to a colleague who shares one of the assignee's WhatsApp
// accounts (or has no account restriction). The transfer goes through
// ChatLifecycle.Assign, so the target gets the permanent access grant even
// when the conversation sits under an account they are not assigned to.

var transferChatPerms = []string{"chat:read", "chat:write", "chat.assign:write", "contacts:read"}

// transferFixture: accounts A, B, C; the owner is assigned A+B and holds an
// open conversation under A. Colleagues cover every eligibility branch.
type transferFixture struct {
	app          *handlers.App
	org          *models.Organization
	owner        *models.User // assigned A+B, owns the conversation
	sameA        *models.User // assigned A: shares the conversation's account
	onlyB        *models.User // assigned B: shares an owner account, not the chat's
	onlyC        *models.User // assigned C: shares nothing with the owner
	unrestricted *models.User // no account assignments: sees every account
	noChatPerm   *models.User // assigned A, role without chat:read
	inactive     *models.User // assigned A, deactivated
	accA         *models.WhatsAppAccount
	contact      *models.Contact
}

func newTransferFixture(t *testing.T) *transferFixture {
	t.Helper()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	accA := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "ta")))
	accB := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "tb")))
	accC := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName(accName(org, "tc")))

	owner := scopedUser(t, app, org, "Transfer Owner", transferChatPerms, accA.ID, accB.ID)

	inactiveRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "role-Transfer Inactive", transferChatPerms)
	inactive := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&inactiveRole.ID),
		testutil.WithFullName("Transfer Inactive"), testutil.WithInactive())
	testutil.AssignAccountToUser(t, app.DB, inactive.ID, accA.ID)

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(accA.Name), testutil.WithAssignee(owner.ID))

	return &transferFixture{
		app:          app,
		org:          org,
		owner:        owner,
		sameA:        scopedUser(t, app, org, "Transfer Same A", transferChatPerms, accA.ID),
		onlyB:        scopedUser(t, app, org, "Transfer Only B", transferChatPerms, accB.ID),
		onlyC:        scopedUser(t, app, org, "Transfer Only C", transferChatPerms, accC.ID),
		unrestricted: scopedUser(t, app, org, "Transfer Unrestricted", transferChatPerms),
		noChatPerm:   scopedUser(t, app, org, "Transfer No Chat", []string{"contacts:read"}, accA.ID),
		inactive:     inactive,
		accA:         accA,
		contact:      contact,
	}
}

func (f *transferFixture) transfer(t *testing.T, callerID uuid.UUID, body map[string]any) *fastglue.Request {
	t.Helper()
	req := testutil.NewJSONRequest(t, body)
	req.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(req, f.org.ID, callerID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	require.NoError(t, f.app.TransferChat(req))
	return req
}

func transferTargetIDs(t *testing.T, app *handlers.App, orgID, callerID uuid.UUID) map[uuid.UUID]bool {
	t.Helper()
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgID, callerID)
	require.NoError(t, app.ListTransferTargets(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"list transfer targets: %s", string(testutil.GetResponseBody(req)))

	var resp struct {
		Users []handlers.TransferTarget `json:"users"`
	}
	testutil.ParseEnvelopeResponse(t, req, &resp)
	ids := make(map[uuid.UUID]bool, len(resp.Users))
	for _, u := range resp.Users {
		ids[u.ID] = true
	}
	return ids
}

// TestListTransferTargets_SharedAccountsOnly: a scoped caller sees colleagues
// on any of their accounts plus unrestricted users, never themselves, users
// on unrelated accounts, users without chat access, or inactive users.
func TestListTransferTargets_SharedAccountsOnly(t *testing.T) {
	f := newTransferFixture(t)
	got := transferTargetIDs(t, f.app, f.org.ID, f.owner.ID)

	assert.Equal(t, map[uuid.UUID]bool{
		f.sameA.ID:        true,
		f.onlyB.ID:        true,
		f.unrestricted.ID: true,
	}, got)
}

// TestListTransferTargets_UnrestrictedCallerSeesAllChatUsers: a caller with no
// account assignment sees every account, so every active chat user qualifies.
func TestListTransferTargets_UnrestrictedCallerSeesAllChatUsers(t *testing.T) {
	f := newTransferFixture(t)
	got := transferTargetIDs(t, f.app, f.org.ID, f.unrestricted.ID)

	for _, u := range []*models.User{f.owner, f.sameA, f.onlyB, f.onlyC} {
		assert.True(t, got[u.ID], "%s must be a target for an unrestricted caller", u.FullName)
	}
	for _, u := range []*models.User{f.unrestricted, f.noChatPerm, f.inactive} {
		assert.False(t, got[u.ID], "%s must not be a target", u.FullName)
	}
}

// TestListTransferTargets_RequiresAssignPermission: chat.assign:write gates
// the colleague list.
func TestListTransferTargets_RequiresAssignPermission(t *testing.T) {
	f := newTransferFixture(t)
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, f.org.ID, f.noChatPerm.ID)
	require.NoError(t, f.app.ListTransferTargets(req))
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

// TestTransferChat_SameAccountColleague: the assignee hands the conversation
// to a colleague on the same account.
func TestTransferChat_SameAccountColleague(t *testing.T) {
	f := newTransferFixture(t)
	req := f.transfer(t, f.owner.ID, map[string]any{"user_id": f.sameA.ID.String()})
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req), string(testutil.GetResponseBody(req)))

	var c models.Contact
	require.NoError(t, f.app.DB.First(&c, "id = ?", f.contact.ID).Error)
	require.NotNil(t, c.AssignedUserID)
	assert.Equal(t, f.sameA.ID, *c.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, c.EffectiveStatus())

	var sysMsgs int64
	require.NoError(t, f.app.DB.Model(&models.Message{}).
		Where("contact_id = ? AND metadata->>'system_type' = ?", f.contact.ID, "chat_assigned").
		Count(&sysMsgs).Error)
	assert.EqualValues(t, 1, sysMsgs, "transfer must write the assignment system message")
}

// TestTransferChat_CrossAccountColleagueGetsGrant: a colleague who shares an
// owner account but not the conversation's account receives the access grant
// and can see the conversation after the transfer.
func TestTransferChat_CrossAccountColleagueGetsGrant(t *testing.T) {
	f := newTransferFixture(t)
	require.False(t, visibleContactIDs(t, f.app, f.org.ID, f.onlyB.ID)[f.contact.ID],
		"precondition: the account-B colleague cannot see an account-A conversation")

	req := f.transfer(t, f.owner.ID, map[string]any{"user_id": f.onlyB.ID.String()})
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req), string(testutil.GetResponseBody(req)))

	var grants int64
	require.NoError(t, f.app.DB.Model(&models.ContactAssignmentAccessGrant{}).
		Where("contact_id = ? AND user_id = ? AND revoked_at IS NULL", f.contact.ID, f.onlyB.ID).
		Count(&grants).Error)
	assert.EqualValues(t, 1, grants, "transfer must mint the target's access grant")
	assert.True(t, visibleContactIDs(t, f.app, f.org.ID, f.onlyB.ID)[f.contact.ID],
		"the new assignee must see the conversation")
	assert.Equal(t, "current_assignee", accessModeFor(t, f.app, f.org.ID, f.onlyB.ID, f.contact.ID))
}

// TestTransferChat_PromotesCollaborator: transferring to a collaborator makes
// them the owner and drops them from the collaborators list.
func TestTransferChat_PromotesCollaborator(t *testing.T) {
	f := newTransferFixture(t)
	f.contact.AddCollaborator(models.Collaborator{UserID: f.sameA.ID.String(), Name: f.sameA.FullName})
	require.NoError(t, f.app.DB.Model(&models.Contact{}).Where("id = ?", f.contact.ID).
		Update("metadata", f.contact.Metadata).Error)

	req := f.transfer(t, f.owner.ID, map[string]any{"user_id": f.sameA.ID.String()})
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req), string(testutil.GetResponseBody(req)))

	var c models.Contact
	require.NoError(t, f.app.DB.First(&c, "id = ?", f.contact.ID).Error)
	require.NotNil(t, c.AssignedUserID)
	assert.Equal(t, f.sameA.ID, *c.AssignedUserID)
	assert.False(t, c.IsCollaborator(f.sameA.ID.String()), "the new owner must leave the collaborators list")
}

// TestTransferChat_Refusals: every refusal leaves the assignment untouched.
func TestTransferChat_Refusals(t *testing.T) {
	tests := []struct {
		name       string
		caller     func(f *transferFixture) uuid.UUID
		body       func(f *transferFixture) map[string]any
		setup      func(t *testing.T, f *transferFixture)
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "colleague on unrelated account",
			caller:     func(f *transferFixture) uuid.UUID { return f.owner.ID },
			body:       func(f *transferFixture) map[string]any { return map[string]any{"user_id": f.onlyC.ID.String()} },
			wantStatus: fasthttp.StatusBadRequest,
			wantMsg:    "cannot receive",
		},
		{
			name:       "colleague without chat access",
			caller:     func(f *transferFixture) uuid.UUID { return f.owner.ID },
			body:       func(f *transferFixture) map[string]any { return map[string]any{"user_id": f.noChatPerm.ID.String()} },
			wantStatus: fasthttp.StatusBadRequest,
			wantMsg:    "cannot receive",
		},
		{
			name:       "inactive colleague",
			caller:     func(f *transferFixture) uuid.UUID { return f.owner.ID },
			body:       func(f *transferFixture) map[string]any { return map[string]any{"user_id": f.inactive.ID.String()} },
			wantStatus: fasthttp.StatusBadRequest,
			wantMsg:    "cannot receive",
		},
		{
			name:       "unknown user",
			caller:     func(f *transferFixture) uuid.UUID { return f.owner.ID },
			body:       func(f *transferFixture) map[string]any { return map[string]any{"user_id": uuid.New().String()} },
			wantStatus: fasthttp.StatusBadRequest,
			wantMsg:    "cannot receive",
		},
		{
			name:       "self",
			caller:     func(f *transferFixture) uuid.UUID { return f.owner.ID },
			body:       func(f *transferFixture) map[string]any { return map[string]any{"user_id": f.owner.ID.String()} },
			wantStatus: fasthttp.StatusBadRequest,
			wantMsg:    "already own",
		},
		{
			name:       "missing user_id",
			caller:     func(f *transferFixture) uuid.UUID { return f.owner.ID },
			body:       func(f *transferFixture) map[string]any { return map[string]any{} },
			wantStatus: fasthttp.StatusBadRequest,
			wantMsg:    "user_id is required",
		},
		{
			name:       "caller is not the assignee",
			caller:     func(f *transferFixture) uuid.UUID { return f.sameA.ID },
			body:       func(f *transferFixture) map[string]any { return map[string]any{"user_id": f.onlyB.ID.String()} },
			wantStatus: fasthttp.StatusForbidden,
			wantMsg:    "Only the assigned agent",
		},
		{
			name: "caller without chat.assign:write",
			caller: func(f *transferFixture) uuid.UUID {
				return f.noChatPerm.ID
			},
			body:       func(f *transferFixture) map[string]any { return map[string]any{"user_id": f.sameA.ID.String()} },
			wantStatus: fasthttp.StatusForbidden,
			wantMsg:    "Insufficient permissions",
		},
		{
			name:   "closed conversation",
			caller: func(f *transferFixture) uuid.UUID { return f.owner.ID },
			body:   func(f *transferFixture) map[string]any { return map[string]any{"user_id": f.sameA.ID.String()} },
			setup: func(t *testing.T, f *transferFixture) {
				require.NoError(t, f.app.DB.Model(&models.Contact{}).Where("id = ?", f.contact.ID).
					Update("metadata", map[string]any{"chat_status": "closed"}).Error)
			},
			wantStatus: fasthttp.StatusConflict,
			wantMsg:    "Only open conversations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTransferFixture(t)
			if tt.setup != nil {
				tt.setup(t, f)
			}

			req := f.transfer(t, tt.caller(f), tt.body(f))
			testutil.AssertErrorResponse(t, req, tt.wantStatus, tt.wantMsg)

			var c models.Contact
			require.NoError(t, f.app.DB.First(&c, "id = ?", f.contact.ID).Error)
			require.NotNil(t, c.AssignedUserID)
			assert.Equal(t, f.owner.ID, *c.AssignedUserID, "a refused transfer must not change the assignee")
		})
	}
}
