package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wsTestApp builds a DB-backed App for ws-scoping tests. Redis is optional:
// permission lookups fall back to the database when it is absent.
func wsTestApp(t *testing.T) *App {
	t.Helper()
	return &App{
		DB:  testutil.SetupTestDB(t),
		Log: testutil.NopLogger(),
	}
}

// TestWSContactRecipients_Matrix pins the recipient rule for conversation
// broadcasts: involvement wins on ANY account; account scope only counts for
// users holding contacts:read; super admins and users without account
// assignments get everything; revoked grants exclude; a contacts:read user
// assigned an account SUBSET must not receive other accounts' events (the
// review finding — BroadcastToOrg leaked them).
func TestWSContactRecipients_Matrix(t *testing.T) {
	app := wsTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	suffix := org.ID.String()[:8]
	accA := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("ws-a-"+suffix))
	accB := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("ws-b-"+suffix))

	readRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "ws-read",
		[]string{"contacts:read", "chat:read"})
	noReadRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "ws-noread",
		[]string{"chat:write"})

	// A real super admin carries a role too — the permission cache errors out
	// on roleless users, so the fixture mirrors production shape.
	super := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithSuperAdmin(), testutil.WithRoleID(&readRole.ID))
	fullNoAssign := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&readRole.ID))
	subsetReader := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&readRole.ID))
	testutil.AssignAccountToUser(t, app.DB, subsetReader.ID, accA.ID)
	subsetNoRead := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&noReadRole.ID))
	testutil.AssignAccountToUser(t, app.DB, subsetNoRead.ID, accA.ID)

	contactA := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accA.Name))
	contactB := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accB.Name))

	// Involvement fixtures on contactB (cross-account for the subset users).
	assignee := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&readRole.ID))
	testutil.AssignAccountToUser(t, app.DB, assignee.ID, accA.ID)
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", contactB.ID).
		Update("assigned_user_id", assignee.ID).Error)

	collab := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&readRole.ID))
	testutil.AssignAccountToUser(t, app.DB, collab.ID, accA.ID)
	contactB.Metadata = models.JSONB{"collaborators": []any{
		map[string]any{"user_id": collab.ID.String(), "name": "C", "role": "agent", "joined_at": time.Now()},
	}}
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", contactB.ID).
		Update("metadata", contactB.Metadata).Error)

	grantHolder := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&readRole.ID))
	testutil.AssignAccountToUser(t, app.DB, grantHolder.ID, accA.ID)
	require.NoError(t, app.DB.Create(&models.ContactAssignmentAccessGrant{
		OrganizationID: org.ID, ContactID: contactB.ID, UserID: grantHolder.ID,
		GrantedBy: super.ID, GrantedAt: time.Now(),
	}).Error)

	revoked := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&readRole.ID))
	testutil.AssignAccountToUser(t, app.DB, revoked.ID, accA.ID)
	now := time.Now()
	require.NoError(t, app.DB.Create(&models.ContactAssignmentAccessGrant{
		OrganizationID: org.ID, ContactID: contactB.ID, UserID: revoked.ID,
		GrantedBy: super.ID, GrantedAt: now, RevokedAt: &now,
	}).Error)

	in := func(t *testing.T, list []uuid.UUID, id uuid.UUID) bool {
		t.Helper()
		for _, u := range list {
			if u == id {
				return true
			}
		}
		return false
	}

	// freshContact reloads contactB so in-memory mutation state matches the
	// persisted rows under test.
	freshContact := func(t *testing.T) *models.Contact {
		t.Helper()
		var c models.Contact
		require.NoError(t, app.DB.First(&c, "id = ?", contactB.ID).Error)
		return &c
	}

	// Account-A conversation: org-wide viewers + the account-A subset reader.
	recA := app.wsContactRecipients(contactA, org.ID)
	assert.True(t, in(t, recA, super.ID), "super admin always receives")
	assert.True(t, in(t, recA, fullNoAssign.ID), "no assignments = org-wide fallback")
	assert.True(t, in(t, recA, subsetReader.ID), "assigned account matches")
	assert.False(t, in(t, recA, subsetNoRead.ID),
		"contacts:read-less subset user has involvement-only access")

	// Account-B conversation: involvement beats the account subset.
	recB := app.wsContactRecipients(freshContact(t), org.ID)
	assert.True(t, in(t, recB, super.ID))
	assert.True(t, in(t, recB, fullNoAssign.ID))
	assert.True(t, in(t, recB, assignee.ID), "assignee receives even cross-account")
	assert.True(t, in(t, recB, collab.ID), "active collaborator receives")
	assert.True(t, in(t, recB, grantHolder.ID), "active grant holder receives")
	assert.False(t, in(t, recB, subsetReader.ID),
		"account-subset contacts:read user must NOT receive other accounts' messages")
	assert.False(t, in(t, recB, subsetNoRead.ID))
	assert.False(t, in(t, recB, revoked.ID), "revoked grant excludes")

	// Involvement on contactB for the contacts:read-less user: assign them.
	require.NoError(t, app.DB.Model(&models.Contact{}).Where("id = ?", contactB.ID).
		Update("assigned_user_id", subsetNoRead.ID).Error)
	recB2 := app.wsContactRecipients(freshContact(t), org.ID)
	assert.True(t, in(t, recB2, subsetNoRead.ID),
		"assignee receives even without contacts:read")
}

// TestWSContactRecipients_FailsClosed: a contact whose account no longer
// exists still classifies exactly like scopeAssignedContact (org-wide
// fallback viewers included, assigned-subset users excluded) — and missing
// inputs drop to an empty recipient list rather than a wildcard.
func TestWSContactRecipients_FailsClosed(t *testing.T) {
	app := wsTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	readRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "ws-closed",
		[]string{"contacts:read"})
	super := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithSuperAdmin(), testutil.WithRoleID(&readRole.ID))
	subset := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&readRole.ID))
	accA := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID,
		testutil.WithAccountName("ws-closed-a-"+org.ID.String()[:8]))
	testutil.AssignAccountToUser(t, app.DB, subset.ID, accA.ID)

	ghost := &models.Contact{
		OrganizationID:  org.ID,
		WhatsAppAccount: "account-that-does-not-exist",
	}
	rec := app.wsContactRecipients(ghost, org.ID)
	contains := func(list []uuid.UUID, id uuid.UUID) bool {
		for _, u := range list {
			if u == id {
				return true
			}
		}
		return false
	}
	assert.True(t, contains(rec, super.ID),
		"org-wide fallback viewers still receive (same as the list endpoint)")
	assert.False(t, contains(rec, subset.ID),
		"account-subset users are excluded by the unmatched account")

	assert.Empty(t, app.wsContactRecipients(nil, org.ID), "nil contact drops")
	assert.Empty(t, app.wsContactRecipientsByID(org.ID, uuid.New()),
		"missing contact drops")
}
