package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// closeChatUpdates — pure unit tests (no DB required)
// ---------------------------------------------------------------------------

func TestCloseChatUpdates_PreservesAssignee(t *testing.T) {
	t.Parallel()

	closerID := uuid.New()
	assigneeID := uuid.New()

	// clearAssignee=false, currentAssignee is set → preserve assignee
	updates := closeChatUpdates(closerID, &assigneeID, false)

	assert.Equal(t, models.ChatStatusClosed, updates["status"])
	assert.Equal(t, &assigneeID, updates["assigned_user_id"])
	assert.NotNil(t, updates["closed_at"])
	assert.Equal(t, &closerID, updates["closed_by_user_id"])
}

func TestCloseChatUpdates_FallsBackToCloser(t *testing.T) {
	t.Parallel()

	closerID := uuid.New()

	// clearAssignee=false, no current assignee → fall back to closer
	updates := closeChatUpdates(closerID, nil, false)

	assert.Equal(t, models.ChatStatusClosed, updates["status"])
	assert.Equal(t, &closerID, updates["assigned_user_id"])
	assert.NotNil(t, updates["closed_at"])
	assert.Equal(t, &closerID, updates["closed_by_user_id"])
}

func TestCloseChatUpdates_ClearsAssignee(t *testing.T) {
	t.Parallel()

	closerID := uuid.New()

	// clearAssignee=true → always nil assignee
	updates := closeChatUpdates(closerID, nil, true)

	assert.Equal(t, models.ChatStatusClosed, updates["status"])
	assert.Nil(t, updates["assigned_user_id"])
	assert.NotNil(t, updates["closed_at"])
	assert.Equal(t, &closerID, updates["closed_by_user_id"])
}

func TestCloseChatUpdates_ClearsAssigneeEvenWhenAssigned(t *testing.T) {
	t.Parallel()

	closerID := uuid.New()
	previousAssignee := uuid.New()

	// clearAssignee=true → nil regardless of previous assignee
	updates := closeChatUpdates(closerID, &previousAssignee, true)

	assert.Equal(t, models.ChatStatusClosed, updates["status"])
	assert.Nil(t, updates["assigned_user_id"])
	assert.NotNil(t, updates["closed_at"])
	assert.Equal(t, &closerID, updates["closed_by_user_id"])
}

func TestCloseChatUpdates_ClosedAtIsRecent(t *testing.T) {
	t.Parallel()

	beforeCall := time.Now().UTC()
	updates := closeChatUpdates(uuid.New(), nil, false)
	afterCall := time.Now().UTC()

	closedAt, ok := updates["closed_at"].(*time.Time)
	require.True(t, ok, "closed_at must be *time.Time")
	require.NotNil(t, closedAt)

	// closedAt should be between beforeCall and afterCall (within tolerance)
	assert.False(t, closedAt.Before(beforeCall.Add(-time.Second)), "closed_at too far in past")
	assert.False(t, closedAt.After(afterCall.Add(time.Second)), "closed_at too far in future")
}

// ---------------------------------------------------------------------------
// chatAssignmentUpdates — pure unit tests (no DB required)
// ---------------------------------------------------------------------------

func TestChatAssignmentUpdates_AssignsUser(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	updates := chatAssignmentUpdates(&userID)

	assert.Equal(t, models.ChatStatusOpen, updates["status"])
	assert.Equal(t, &userID, updates["assigned_user_id"])
	assert.Nil(t, updates["closed_at"])
	assert.Nil(t, updates["closed_by_user_id"])
}

func TestChatAssignmentUpdates_UnassignsUser(t *testing.T) {
	t.Parallel()

	updates := chatAssignmentUpdates(nil)

	assert.Equal(t, models.ChatStatusPending, updates["status"])
	assert.Nil(t, updates["assigned_user_id"])
	assert.Nil(t, updates["closed_at"])
	assert.Nil(t, updates["closed_by_user_id"])
}

// ---------------------------------------------------------------------------
// reopenChatUpdates — delegates to chatAssignmentUpdates(nil)
// ---------------------------------------------------------------------------

func TestReopenChatUpdates_ReturnsPendingState(t *testing.T) {
	t.Parallel()

	updates := reopenChatUpdates()

	assert.Equal(t, models.ChatStatusPending, updates["status"])
	assert.Nil(t, updates["assigned_user_id"])
	assert.Nil(t, updates["closed_at"])
	assert.Nil(t, updates["closed_by_user_id"])
}

func TestReopenChatUpdates_SameAsAssignmentUpdatesWithNil(t *testing.T) {
	t.Parallel()

	fromReopen := reopenChatUpdates()
	fromAssignment := chatAssignmentUpdates(nil)

	// Both should produce identical maps
	assert.Equal(t, fromAssignment, fromReopen)
}

// ---------------------------------------------------------------------------
// normalizeContactStatus — pure unit tests (no DB required)
// ---------------------------------------------------------------------------

func TestNormalizeContactStatus_NilContact(t *testing.T) {
	assert.Equal(t, models.ChatStatusPending, normalizeContactStatus(nil))
}

func TestNormalizeContactStatus_OpenContact(t *testing.T) {
	contact := &models.Contact{
		Status:         models.ChatStatusPending,
		AssignedUserID: ptr(uuid.New()),
	}
	// EffectiveStatus() returns open when assigned
	status := normalizeContactStatus(contact)
	assert.Equal(t, models.ChatStatusOpen, status)
	assert.Equal(t, models.ChatStatusOpen, contact.Status, "should mutate contact.Status")
}

func TestNormalizeContactStatus_ClosedContact(t *testing.T) {
	contact := &models.Contact{
		Status: models.ChatStatusClosed,
	}
	status := normalizeContactStatus(contact)
	assert.Equal(t, models.ChatStatusClosed, status)
}

// ---------------------------------------------------------------------------
// canManageChatLifecycle — DB-required integration tests
// ---------------------------------------------------------------------------

func setupLifecycleTestApp(t *testing.T) *App {
	t.Helper()
	db := testutil.SetupTestDB(t)
	log := testutil.NopLogger()
	return &App{
		DB:  db,
		Log: log,
	}
}

func TestCanManageChatLifecycle_NilApp(t *testing.T) {
	t.Parallel()

	var app *App
	assert.False(t, app.canManageChatLifecycle(uuid.New(), uuid.New()))
}

func TestCanManageChatLifecycle_NoPermissions(t *testing.T) {
	db := testutil.SetupTestDB(t)
	org := testutil.CreateTestOrganization(t, db)
	user := testutil.CreateTestUser(t, db, org.ID)

	app := &App{DB: db, Log: testutil.NopLogger()}

	assert.False(t, app.canManageChatLifecycle(user.ID, org.ID),
		"user with no roles should not have chat lifecycle permissions")
}

func TestCanManageChatLifecycle_WithChatAssignPermission(t *testing.T) {
	db := testutil.SetupTestDB(t)
	org := testutil.CreateTestOrganization(t, db)
	adminRole := testutil.CreateAdminRole(t, db, org.ID)
	user := testutil.CreateTestUser(t, db, org.ID, testutil.WithRoleID(&adminRole.ID))

	app := &App{DB: db, Log: testutil.NopLogger()}

	assert.True(t, app.canManageChatLifecycle(user.ID, org.ID),
		"admin role should have chat lifecycle permissions")
}

func TestCanManageChatLifecycle_DifferentOrg(t *testing.T) {
	db := testutil.SetupTestDB(t)
	org1 := testutil.CreateTestOrganization(t, db)
	org2 := testutil.CreateTestOrganization(t, db)
	adminRole := testutil.CreateAdminRole(t, db, org1.ID)
	user := testutil.CreateTestUser(t, db, org1.ID, testutil.WithRoleID(&adminRole.ID))

	app := &App{DB: db, Log: testutil.NopLogger()}

	// User from org1 should NOT have permissions in org2
	assert.False(t, app.canManageChatLifecycle(user.ID, org2.ID),
		"user should not have lifecycle permissions in a different org")
}

// ptr returns a pointer to v.
func ptr[T any](v T) *T {
	return &v
}
