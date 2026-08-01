package chatlifecycle_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/gowa-ui/internal/chatlifecycle"
	"github.com/shridarpatil/gowa-ui/internal/models"
	"github.com/shridarpatil/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// This file proves the core promise of the P0 refactor: the chat state
// machine is now unit-testable with a Postgres DB ALONE — no Redis, no *App,
// no fastglue. Each test builds a Service directly via chatlifecycle.New(db,
// nil, log) and seeds rows with the same testutil helpers the handler tests
// use.

// newService builds an isolated Service + DB + org for one test. wsHub is nil
// (broadcasts are no-ops) — WS behavior is verified at the integration level.
func newService(t *testing.T) (*chatlifecycle.Service, *gorm.DB, *models.Organization) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	svc := chatlifecycle.New(db, nil, testutil.NopLogger())
	org := testutil.CreateTestOrganization(t, db)
	return svc, db, org
}

// claimForTest mutates a contact into the assigned/open state, mirroring
// what ClaimChat does, so release tests start from a realistic row.
func claimForTest(t *testing.T, db *gorm.DB, c *models.Contact, assigneeID uuid.UUID) {
	t.Helper()
	c.AssignedUserID = &assigneeID
	c.SetStatus(models.ChatStatusOpen)
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", c.ID).Updates(map[string]any{
		"assigned_user_id": assigneeID,
		"metadata":         c.Metadata,
	}).Error)
}

// auditCount returns the number of audit_log rows for a (contact) resource,
// reading the count immediately (no polling). Use for pre-state measurement.
func auditCount(t *testing.T, db *gorm.DB, contactID uuid.UUID) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.AuditLog{}).
		Where("resource_type = ? AND resource_id = ?", "contact", contactID).
		Count(&n).Error)
	return n
}

// auditCountIs asserts the audit row count matches expected, polling briefly
// to absorb audit.LogAudit's async write (the DB create happens in a
// detached goroutine — see internal/audit/audit.go:146 — so a read
// immediately after Release() returns can race it).
func auditCountIs(t *testing.T, db *gorm.DB, contactID uuid.UUID, expected int64, msgAndArgs ...any) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		var n int64
		require.NoError(t, db.Model(&models.AuditLog{}).
			Where("resource_type = ? AND resource_id = ?", "contact", contactID).
			Count(&n).Error)
		if n >= expected || time.Now().After(deadline) {
			assert.EqualValues(t, expected, n, msgAndArgs...)
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestService_Release_Assignee_Success is the direct analog of the handler
// test TestApp_ReleaseChat_Assignee_Success, but exercises the service with
// no *App, no Redis, no HTTP. This is the regression contract for the
// extracted state machine.
func TestService_Release_Assignee_Success(t *testing.T) {
	svc, db, org := newService(t)
	agent := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Test Agent"))

	contact := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, contact, agent.ID)

	released, err := svc.Release(context.Background(), org.ID, agent.ID, contact, true, false)
	require.NoError(t, err)
	require.True(t, released, "Release must report a real release for an open assigned chat")

	// DB row: unassigned + pending + last_message_at bumped.
	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contact.ID).Error)
	assert.Nil(t, updated.AssignedUserID, "assigned_user_id must be cleared")
	assert.Equal(t, models.ChatStatusPending, updated.EffectiveStatus(),
		"chat_status must revert to pending")
	assert.NotZero(t, updated.LastMessageAt, "last_message_at must be bumped so the chat re-sorts")

	// Audit entry persisted — the extraChanges safeguard held.
	auditCountIs(t, db, contact.ID, 1, "release must persist exactly one audit entry")
}

// TestService_Release_ClosedChat_ByAgent_Forbidden pins the G2 closed-chat
// policy: an agent who is the assignee of a CLOSED chat must NOT be able to
// release it.
func TestService_Release_ClosedChat_ByAgent_Forbidden(t *testing.T) {
	svc, db, org := newService(t)
	agent := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Test Agent"))

	contact := testutil.CreateTestContact(t, db, org.ID)
	contact.AssignedUserID = &agent.ID
	contact.SetStatus(models.ChatStatusClosed)
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", contact.ID).Updates(map[string]any{
		"assigned_user_id": agent.ID,
		"metadata":         contact.Metadata,
	}).Error)

	released, err := svc.Release(context.Background(), org.ID, agent.ID, contact, true, false)
	require.ErrorIs(t, err, chatlifecycle.ErrClosedReleaseByAgent,
		"agent assignee of a closed chat must get the typed closed-release error")
	assert.False(t, released)

	// State untouched.
	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusClosed, updated.EffectiveStatus())
}

// TestService_Release_NonOwnerNonAdmin_Forbidden: a caller who is neither
// the assignee nor an admin/manager is rejected by the service's
// defense-in-depth authorization check.
func TestService_Release_NonOwnerNonAdmin_Forbidden(t *testing.T) {
	svc, db, org := newService(t)
	owner := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Owner"))
	contact := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, contact, owner.ID)

	// A random unrelated user, isAssignee=false, isAdminOrManager=false.
	released, err := svc.Release(context.Background(), org.ID, uuid.New(), contact, false, false)
	require.ErrorIs(t, err, chatlifecycle.ErrNotAuthorized)
	assert.False(t, released)
}

// TestService_Release_AlreadyPending_Idempotent: re-releasing an
// already-pending + unassigned chat is a safe no-op — no audit, no system
// message, no broadcast.
func TestService_Release_AlreadyPending_Idempotent(t *testing.T) {
	svc, db, org := newService(t)
	contact := testutil.CreateTestContact(t, db, org.ID)
	contact.SetStatus(models.ChatStatusPending)
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", contact.ID).
		Update("metadata", contact.Metadata).Error)

	before := auditCount(t, db, contact.ID)
	released, err := svc.Release(context.Background(), org.ID, uuid.New(), contact, true, true)
	require.NoError(t, err)
	assert.False(t, released, "idempotent release must report released=false")
	assert.EqualValues(t, before, auditCount(t, db, contact.ID),
		"idempotent release must not write a duplicate audit entry")
}

// TestService_Release_Admin_CanReleaseClosed: an admin/manager CAN release
// a closed chat (the closed-chat guard only applies to non-admins).
func TestService_Release_Admin_CanReleaseClosed(t *testing.T) {
	svc, db, org := newService(t)
	admin := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Admin"))

	contact := testutil.CreateTestContact(t, db, org.ID)
	contact.AssignedUserID = &admin.ID
	contact.SetStatus(models.ChatStatusClosed)
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", contact.ID).Updates(map[string]any{
		"assigned_user_id": admin.ID,
		"metadata":         contact.Metadata,
	}).Error)

	// isAdminOrManager=true bypasses the closed-chat guard.
	released, err := svc.Release(context.Background(), org.ID, admin.ID, contact, false, true)
	require.NoError(t, err)
	require.True(t, released)

	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusPending, updated.EffectiveStatus())
	assert.Nil(t, updated.AssignedUserID)
}

// TestService_CustomerReopen_SubsumesReopenBlock pins the service
// API behind ensureClaimableChatStatus (internal/handlers/chat_lifecycle.go),
// which routes both the incoming and phone-sent reopen paths through here.
func TestService_CustomerReopen_SubsumesReopenBlock(t *testing.T) {
	svc, db, org := newService(t)
	contact := testutil.CreateTestContact(t, db, org.ID)
	contact.SetStatus(models.ChatStatusClosed)
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", contact.ID).
		Update("metadata", contact.Metadata).Error)

	reopened := svc.CustomerReopen(context.Background(), org.ID, contact, "")
	assert.True(t, reopened, "CustomerReopen on a closed chat must reopen it")

	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusPending, updated.EffectiveStatus())
	assert.Nil(t, updated.AssignedUserID, "reopen must unassign the owner")

	// Idempotent: a second call on the now-pending chat must NOT re-fire.
	reopenedAgain := svc.CustomerReopen(context.Background(), org.ID, &updated, "")
	assert.False(t, reopenedAgain, "CustomerReopen on a non-closed chat must be a no-op")
}

// TestService_BulkRelease_OwnerReleasesOwn mirrors the handler bulk test:
// an agent bulk-releases their own chats; chats owned by others are rejected
// per-item without aborting the batch.
func TestService_BulkRelease_OwnerReleasesOwn(t *testing.T) {
	svc, db, org := newService(t)
	agent := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Test Agent"))
	other := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Other"))

	c1 := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, c1, agent.ID)
	c2 := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, c2, agent.ID)
	c3 := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, c3, other.ID)

	result := svc.BulkRelease(context.Background(), org.ID, agent.ID,
		[]string{c1.ID.String(), c2.ID.String(), c3.ID.String()}, false)

	assert.ElementsMatch(t,
		[]string{c1.ID.String(), c2.ID.String()}, result.ReleasedIDs,
		"agent must be able to bulk-release their own chats")
	require.Len(t, result.Failed, 1, "the other-agent chat must fail, not silently release")
	assert.Equal(t, c3.ID.String(), result.Failed[0]["contact_id"])
	assert.Equal(t, "not authorized", result.Failed[0]["reason"])

	// c3 stays assigned to `other`.
	var c3After models.Contact
	require.NoError(t, db.First(&c3After, "id = ?", c3.ID).Error)
	require.NotNil(t, c3After.AssignedUserID)
	assert.Equal(t, other.ID, *c3After.AssignedUserID)
}

// ─── Coverage gap fillers: the 7 previously-untested handlers ───
//
// These tests exist ONLY because the refactor made them possible. Before P0,
// the claim/close/reopen/join/leave/invite/remove logic was inlined in
// *App handler methods and required the full Redis+*App harness to exercise.
// Now the service is unit-testable with Postgres alone.

// TestService_Claim_AssignsPendingChat pins the happy path: claiming a
// pending chat assigns it to the caller and sets status to open.
func TestService_Claim_AssignsPendingChat(t *testing.T) {
	svc, db, org := newService(t)
	agent := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Claimer"))
	contact := testutil.CreateTestContact(t, db, org.ID)
	contact.SetStatus(models.ChatStatusPending)
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", contact.ID).
		Update("metadata", contact.Metadata).Error)

	outcome, agentName, otherName, err := svc.Claim(context.Background(), org.ID, agent.ID, contact, false)
	require.NoError(t, err)
	assert.Equal(t, chatlifecycle.ClaimDone, outcome)
	assert.Equal(t, "Claimer", agentName)
	assert.Empty(t, otherName)

	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contact.ID).Error)
	require.NotNil(t, updated.AssignedUserID)
	assert.Equal(t, agent.ID, *updated.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, updated.EffectiveStatus())
}

// TestService_Claim_AlreadySelf_Idempotent: re-claiming one's own chat is a
// no-op success that just normalizes status to open.
func TestService_Claim_AlreadySelf_Idempotent(t *testing.T) {
	svc, db, org := newService(t)
	agent := testutil.CreateTestUser(t, db, org.ID)
	contact := testutil.CreateTestContact(t, db, org.ID)
	contact.AssignedUserID = &agent.ID
	contact.SetStatus(models.ChatStatusOpen)
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", contact.ID).Updates(map[string]any{
		"assigned_user_id": agent.ID,
		"metadata":         contact.Metadata,
	}).Error)

	outcome, _, _, err := svc.Claim(context.Background(), org.ID, agent.ID, contact, false)
	require.NoError(t, err)
	assert.Equal(t, chatlifecycle.ClaimAlreadySelf, outcome)
}

// TestService_Claim_OtherAssignee_Conflict: claiming a chat owned by another
// agent, without collaborate permission, yields ClaimConflictOther with the
// current owner's name for the 409 body.
func TestService_Claim_OtherAssignee_Conflict(t *testing.T) {
	svc, db, org := newService(t)
	owner := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Owner"))
	claimer := testutil.CreateTestUser(t, db, org.ID)
	contact := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, contact, owner.ID)

	outcome, _, otherName, err := svc.Claim(context.Background(), org.ID, claimer.ID, contact, false)
	require.NoError(t, err)
	assert.Equal(t, chatlifecycle.ClaimConflictOther, outcome)
	assert.Equal(t, "Owner", otherName)
}

// TestService_Claim_OtherAssignee_ReroutesToJoin: claiming a chat owned by
// another agent WITH collaborate permission reroutes to join-as-collaborator.
func TestService_Claim_OtherAssignee_ReroutesToJoin(t *testing.T) {
	svc, db, org := newService(t)
	owner := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Owner"))
	joiner := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Joiner"))
	contact := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, contact, owner.ID)

	outcome, _, _, err := svc.Claim(context.Background(), org.ID, joiner.ID, contact, true)
	require.NoError(t, err)
	assert.Equal(t, chatlifecycle.ClaimRerouteJoin, outcome,
		"with collaborate perm, claim on another's chat must signal reroute-to-join")
}

// TestService_Close_SetsClosedAndBroadcasts: closing an open chat flips status
// to closed. Idempotency: closing an already-closed chat returns ErrAlreadyClosed.
func TestService_Close_SetsClosedAndBroadcasts(t *testing.T) {
	svc, db, org := newService(t)
	agent := testutil.CreateTestUser(t, db, org.ID)
	contact := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, contact, agent.ID)

	require.NoError(t, svc.Close(context.Background(), org.ID, agent.ID, contact))
	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusClosed, updated.EffectiveStatus())

	// Idempotent: second close returns ErrAlreadyClosed, no second system msg.
	err := svc.Close(context.Background(), org.ID, agent.ID, &updated)
	require.ErrorIs(t, err, chatlifecycle.ErrAlreadyClosed)
}

// TestService_Reopen_SetsOpen: admin reopens a closed chat; idempotent on open.
func TestService_Reopen_SetsOpen(t *testing.T) {
	svc, db, org := newService(t)
	admin := testutil.CreateTestUser(t, db, org.ID)
	contact := testutil.CreateTestContact(t, db, org.ID)
	contact.SetStatus(models.ChatStatusClosed)
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", contact.ID).
		Update("metadata", contact.Metadata).Error)

	reopened, err := svc.Reopen(context.Background(), org.ID, admin.ID, contact)
	require.NoError(t, err)
	require.True(t, reopened)
	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusOpen, updated.EffectiveStatus())

	// Idempotent.
	reopened2, err := svc.Reopen(context.Background(), org.ID, admin.ID, &updated)
	require.NoError(t, err)
	assert.False(t, reopened2)
}

// TestService_Join_AddsCollaborator: a user joins as collaborator.
// Idempotent for owner and existing collaborator.
func TestService_Join_AddsCollaborator(t *testing.T) {
	svc, db, org := newService(t)
	owner := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Owner"))
	joiner := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Joiner"))
	contact := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, contact, owner.ID)

	res, err := svc.Join(context.Background(), org.ID, joiner.ID, contact)
	require.NoError(t, err)
	assert.Equal(t, chatlifecycle.JoinDone, res.Outcome)
	assert.Equal(t, "Joiner", res.UserName)

	// Joining again is idempotent.
	res2, err := svc.Join(context.Background(), org.ID, joiner.ID, contact)
	require.NoError(t, err)
	assert.Equal(t, chatlifecycle.JoinAlreadyCollaborator, res2.Outcome)

	// Owner joining their own chat is idempotent.
	res3, err := svc.Join(context.Background(), org.ID, owner.ID, contact)
	require.NoError(t, err)
	assert.Equal(t, chatlifecycle.JoinAlreadyOwner, res3.Outcome)
}

// TestService_Invite_AddsTargetAsCollaborator: an admin invites another user.
func TestService_Invite_AddsTargetAsCollaborator(t *testing.T) {
	svc, db, org := newService(t)
	inviter := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Inviter"))
	target := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Target"))
	contact := testutil.CreateTestContact(t, db, org.ID)

	res, err := svc.Invite(context.Background(), org.ID, inviter.ID, target.ID, contact)
	require.NoError(t, err)
	assert.Equal(t, chatlifecycle.InviteDone, res.Outcome)
	assert.Equal(t, "Target", res.TargetName)
}

// TestService_Leave_OwnerLastParticipant_ClosesChat: the owner leaving a chat
// with no collaborators closes it.
func TestService_Leave_OwnerLastParticipant_ClosesChat(t *testing.T) {
	svc, db, org := newService(t)
	owner := testutil.CreateTestUser(t, db, org.ID)
	contact := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, contact, owner.ID) // no collaborators

	res, err := svc.Leave(context.Background(), org.ID, owner.ID, contact, true, false, false)
	require.NoError(t, err)
	assert.Equal(t, chatlifecycle.LeaveClosedChat, res.Outcome)

	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusClosed, updated.EffectiveStatus())
	assert.Nil(t, updated.AssignedUserID)
}

// TestService_Leave_GhostExit_NoStateChange: an admin who isn't a participant
// leaves — no state change, no system message.
func TestService_Leave_GhostExit_NoStateChange(t *testing.T) {
	svc, db, org := newService(t)
	owner := testutil.CreateTestUser(t, db, org.ID)
	ghost := testutil.CreateTestUser(t, db, org.ID)
	contact := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, contact, owner.ID)

	res, err := svc.Leave(context.Background(), org.ID, ghost.ID, contact, false, false, true)
	require.NoError(t, err)
	assert.Equal(t, chatlifecycle.LeaveGhostExit, res.Outcome)

	// Owner + status unchanged.
	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contact.ID).Error)
	require.NotNil(t, updated.AssignedUserID)
	assert.Equal(t, owner.ID, *updated.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, updated.EffectiveStatus())
}

// TestService_RemoveCollaborator_OwnerRejected: removing the primary owner is
// rejected with ErrCannotRemoveOwner.
func TestService_RemoveCollaborator_OwnerRejected(t *testing.T) {
	svc, db, org := newService(t)
	owner := testutil.CreateTestUser(t, db, org.ID)
	manager := testutil.CreateTestUser(t, db, org.ID)
	contact := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, contact, owner.ID)

	_, err := svc.RemoveCollaborator(context.Background(), org.ID, manager.ID, owner.ID, contact)
	require.ErrorIs(t, err, chatlifecycle.ErrCannotRemoveOwner)
}

// TestService_RemoveCollaborator_NotCollaborator: removing someone who isn't
// a collaborator is rejected with ErrNotCollaborator.
func TestService_RemoveCollaborator_NotCollaborator(t *testing.T) {
	svc, db, org := newService(t)
	manager := testutil.CreateTestUser(t, db, org.ID)
	stranger := testutil.CreateTestUser(t, db, org.ID)
	contact := testutil.CreateTestContact(t, db, org.ID)

	_, err := svc.RemoveCollaborator(context.Background(), org.ID, manager.ID, stranger.ID, contact)
	require.ErrorIs(t, err, chatlifecycle.ErrNotCollaborator)
}
