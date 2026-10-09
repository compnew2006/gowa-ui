package main

import (
	"strings"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixturePhoneNumber(t *testing.T) {
	t.Parallel()

	const (
		runA = "0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d"
		runB = "f0e1d2c3-b4a5-4968-8776-655443322110"
	)

	tests := []struct {
		name    string
		runID   string
		wantErr bool
	}{
		{name: "valid run id", runID: runA},
		{name: "another run id", runID: runB},
		{name: "not hex", runID: "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz", wantErr: true},
		{name: "empty", runID: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := fixturePhoneNumber(tt.runID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, 15, "4-digit prefix plus 11 digits")
			assert.True(t, strings.HasPrefix(got, "1555"))
			assert.Regexp(t, `^\d+$`, got)

			again, err := fixturePhoneNumber(tt.runID)
			require.NoError(t, err)
			assert.Equal(t, got, again, "the number is derived, not random")
		})
	}

	a, err := fixturePhoneNumber(runA)
	require.NoError(t, err)
	b, err := fixturePhoneNumber(runB)
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "different runs must not share a number")
}

// seedAdmin creates the organization and super-admin the seeder looks up, and
// points the seeder at it through the same env var the e2e runner uses.
func seedAdmin(t *testing.T) *models.Organization {
	t.Helper()
	db := testutil.SetupTestDB(t)
	org := testutil.CreateTestOrganization(t, db)
	email := testutil.UniqueEmail("e2e-fixture-admin")
	testutil.CreateTestUser(t, db, org.ID, testutil.WithEmail(email), testutil.WithSuperAdmin())
	t.Setenv("E2E_USER_SUPER_ADMIN_USERNAME", email)
	return org
}

func TestSeedChatConversation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	org := seedAdmin(t)
	runID := uuid.NewString()

	got, err := seed(db, "chat-conversation", runID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cleanup(db, runID, nil) })

	contactID, err := uuid.Parse(got.ResourceID)
	require.NoError(t, err, "the resource id is the contact the route opens")
	assert.Equal(t, "/chat/"+got.ResourceID, got.Path)
	assert.Equal(t, "E2E fixture "+runID+" contact", got.Heading)

	var contact models.Contact
	require.NoError(t, db.First(&contact, "id = ?", contactID).Error)
	assert.Equal(t, org.ID, contact.OrganizationID)
	assert.Equal(t, got.Heading, contact.ProfileName)
	assert.Equal(t, "E2E fixture "+runID+" account", contact.WhatsAppAccount)

	var account models.WhatsAppAccount
	require.NoError(t, db.First(&account, "name = ? AND organization_id = ?", contact.WhatsAppAccount, org.ID).Error)
	assert.Empty(t, account.GowaBaseURL, "no GOWA server: nothing sent from the fixture can leave the machine")
	assert.Empty(t, account.GowaDeviceID, "no GOWA device")

	require.NoError(t, cleanup(db, runID, []uuid.UUID{contactID}))
	var left int64
	require.NoError(t, db.Unscoped().Model(&models.Contact{}).Where("id = ?", contactID).Count(&left).Error)
	assert.Zero(t, left, "cleanup removes the contact")
	require.NoError(t, db.Unscoped().Model(&models.WhatsAppAccount{}).
		Where("name LIKE ?", "E2E fixture "+runID+" account%").Count(&left).Error)
	assert.Zero(t, left, "cleanup removes the account")
}

func TestSeedChatConversationRunsDoNotCollide(t *testing.T) {
	db := testutil.SetupTestDB(t)
	seedAdmin(t)
	first, second := uuid.NewString(), uuid.NewString()
	t.Cleanup(func() {
		_ = cleanup(db, first, nil)
		_ = cleanup(db, second, nil)
	})

	a, err := seed(db, "chat-conversation", first)
	require.NoError(t, err)
	b, err := seed(db, "chat-conversation", second)
	require.NoError(t, err)

	assert.NotEqual(t, a.ResourceID, b.ResourceID)
	assert.NotEqual(t, a.Path, b.Path)
}

func TestSeedRejectsUnknownScenario(t *testing.T) {
	db := testutil.SetupTestDB(t)
	seedAdmin(t)

	_, err := seed(db, "no-such-scenario", uuid.NewString())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported E2E fixture scenario")
}
