package chatlifecycle_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// resetAssignedForTest is the reset-test analog of claimForTest: it mutates a
// contact into the assigned/open state with a WhatsApp account set so the
// reset query can find it.
func resetAssignedForTest(t *testing.T, db *gorm.DB, c *models.Contact, assigneeID uuid.UUID) {
	t.Helper()
	c.AssignedUserID = &assigneeID
	c.SetStatus(models.ChatStatusOpen)
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", c.ID).Updates(map[string]any{
		"assigned_user_id": assigneeID,
		"metadata":         c.Metadata,
	}).Error)
}

// TestService_ResetAssignedChats_ResetsAllOpenAssigned verifies the happy path:
// every assigned+open conversation for the target account is returned to
// pending, unassigned, with a system message in its timeline.
func TestService_ResetAssignedChats_ResetsAllOpenAssigned(t *testing.T) {
	svc, db, org := newService(t)
	account := testutil.CreateTestWhatsAppAccount(t, db, org.ID)
	agent := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Agent"))

	c1 := testutil.CreateTestContactWith(t, db, org.ID, testutil.WithContactAccount(account.Name))
	resetAssignedForTest(t, db, c1, agent.ID)
	c2 := testutil.CreateTestContactWith(t, db, org.ID, testutil.WithContactAccount(account.Name))
	resetAssignedForTest(t, db, c2, agent.ID)

	summary, err := svc.ResetAssignedChats(context.Background(), org.ID, account.Name, "System")
	require.NoError(t, err)
	assert.Equal(t, 2, summary.ResetCount, "both assigned chats must be reset")
	assert.Empty(t, summary.Skipped, "no chats should have failed")

	for _, c := range []*models.Contact{c1, c2} {
		var updated models.Contact
		require.NoError(t, db.First(&updated, "id = ?", c.ID).Error)
		assert.Nil(t, updated.AssignedUserID, "assigned_user_id must be cleared")
		assert.Equal(t, models.ChatStatusPending, updated.EffectiveStatus(),
			"status must revert to pending")

		msgs := systemMessagesFor(t, db, c.ID, "chat_daily_reset")
		require.Len(t, msgs, 1, "each reset chat must have exactly one reset system message")
	}
}

// TestService_ResetAssignedChats_SkipsClosedChats ensures explicitly-closed
// conversations are NOT reset — they should remain closed for audit/history.
func TestService_ResetAssignedChats_SkipsClosedChats(t *testing.T) {
	svc, db, org := newService(t)
	account := testutil.CreateTestWhatsAppAccount(t, db, org.ID)
	agent := testutil.CreateTestUser(t, db, org.ID)

	// Assigned but explicitly closed → must NOT be reset.
	closed := testutil.CreateTestContactWith(t, db, org.ID, testutil.WithContactAccount(account.Name))
	closed.AssignedUserID = &agent.ID
	closed.SetStatus(models.ChatStatusClosed)
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", closed.ID).Updates(map[string]any{
		"assigned_user_id": agent.ID,
		"metadata":         closed.Metadata,
	}).Error)

	// Assigned + open → must be reset.
	open := testutil.CreateTestContactWith(t, db, org.ID, testutil.WithContactAccount(account.Name))
	resetAssignedForTest(t, db, open, agent.ID)

	summary, err := svc.ResetAssignedChats(context.Background(), org.ID, account.Name, "System")
	require.NoError(t, err)
	assert.Equal(t, 1, summary.ResetCount, "only the open chat should be reset")

	var closedAfter models.Contact
	require.NoError(t, db.First(&closedAfter, "id = ?", closed.ID).Error)
	assert.Equal(t, models.ChatStatusClosed, closedAfter.EffectiveStatus(),
		"closed chat must stay closed")
	require.NotNil(t, closedAfter.AssignedUserID, "closed chat must keep its assignee")
}

// TestService_ResetAssignedChats_SkipsPendingUnassigned ensures already-pending
// (unassigned) chats are a no-op — they are already in the target state.
func TestService_ResetAssignedChats_SkipsPendingUnassigned(t *testing.T) {
	svc, db, org := newService(t)
	account := testutil.CreateTestWhatsAppAccount(t, db, org.ID)

	pending := testutil.CreateTestContactWith(t, db, org.ID, testutil.WithContactAccount(account.Name))
	pending.SetStatus(models.ChatStatusPending)
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", pending.ID).
		Update("metadata", pending.Metadata).Error)

	summary, err := svc.ResetAssignedChats(context.Background(), org.ID, account.Name, "System")
	require.NoError(t, err)
	assert.Equal(t, 0, summary.ResetCount, "pending chats must not be counted")
}

// TestService_ResetAssignedChats_AccountScoped verifies only chats for the
// specified account are reset — a chat assigned to a different account must be
// left untouched.
func TestService_ResetAssignedChats_AccountScoped(t *testing.T) {
	svc, db, org := newService(t)
	account1 := testutil.CreateTestWhatsAppAccount(t, db, org.ID)
	account2 := testutil.CreateTestWhatsAppAccount(t, db, org.ID)
	agent := testutil.CreateTestUser(t, db, org.ID)

	// account1 chat — should be reset.
	c1 := testutil.CreateTestContactWith(t, db, org.ID, testutil.WithContactAccount(account1.Name))
	resetAssignedForTest(t, db, c1, agent.ID)

	// account2 chat — should NOT be reset.
	c2 := testutil.CreateTestContactWith(t, db, org.ID, testutil.WithContactAccount(account2.Name))
	resetAssignedForTest(t, db, c2, agent.ID)

	summary, err := svc.ResetAssignedChats(context.Background(), org.ID, account1.Name, "System")
	require.NoError(t, err)
	assert.Equal(t, 1, summary.ResetCount, "only account1's chat should be reset")

	var c2After models.Contact
	require.NoError(t, db.First(&c2After, "id = ?", c2.ID).Error)
	require.NotNil(t, c2After.AssignedUserID, "account2 chat must keep its assignee")
	assert.Equal(t, models.ChatStatusOpen, c2After.EffectiveStatus(),
		"account2 chat must stay open")
}

// TestService_ResetAssignedChats_Idempotent verifies that running the reset
// twice is safe: the second pass finds zero assigned chats and is a no-op.
func TestService_ResetAssignedChats_Idempotent(t *testing.T) {
	svc, db, org := newService(t)
	account := testutil.CreateTestWhatsAppAccount(t, db, org.ID)
	agent := testutil.CreateTestUser(t, db, org.ID)

	c := testutil.CreateTestContactWith(t, db, org.ID, testutil.WithContactAccount(account.Name))
	resetAssignedForTest(t, db, c, agent.ID)

	// First reset: 1 chat.
	summary1, err := svc.ResetAssignedChats(context.Background(), org.ID, account.Name, "System")
	require.NoError(t, err)
	assert.Equal(t, 1, summary1.ResetCount)

	// Second reset: 0 chats (already pending).
	summary2, err := svc.ResetAssignedChats(context.Background(), org.ID, account.Name, "System")
	require.NoError(t, err)
	assert.Equal(t, 0, summary2.ResetCount, "second reset must find nothing to do")

	// Only one system message should exist.
	msgs := systemMessagesFor(t, db, c.ID, "chat_daily_reset")
	assert.Len(t, msgs, 1, "no duplicate system message on idempotent second run")
}

// TestService_ResetAssignedChats_NoChats verifies an account with zero assigned
// chats returns a zero-count summary without error and without audit.
func TestService_ResetAssignedChats_NoChats(t *testing.T) {
	svc, db, org := newService(t)
	account := testutil.CreateTestWhatsAppAccount(t, db, org.ID)

	summary, err := svc.ResetAssignedChats(context.Background(), org.ID, account.Name, "System")
	require.NoError(t, err)
	assert.Equal(t, 0, summary.ResetCount)
	assert.Empty(t, summary.ContactIDs)
}

// TestService_ResetAssignedChats_DefaultsActorName verifies that an empty
// actorName defaults to "System".
func TestService_ResetAssignedChats_DefaultsActorName(t *testing.T) {
	svc, db, org := newService(t)
	account := testutil.CreateTestWhatsAppAccount(t, db, org.ID)
	agent := testutil.CreateTestUser(t, db, org.ID)

	c := testutil.CreateTestContactWith(t, db, org.ID, testutil.WithContactAccount(account.Name))
	resetAssignedForTest(t, db, c, agent.ID)

	summary, err := svc.ResetAssignedChats(context.Background(), org.ID, account.Name, "")
	require.NoError(t, err)
	assert.Equal(t, 1, summary.ResetCount)

	// The audit entry should exist with user_name "System", scoped to this
	// test's org. audit.LogAudit writes asynchronously (detached goroutine),
	// so poll briefly to absorb the race.
	deadline := time.Now().Add(2 * time.Second)
	for {
		var auditEntries []models.AuditLog
		require.NoError(t, db.Where("organization_id = ? AND resource_type = ?",
			org.ID, models.ResourceSettingsChatReset).Find(&auditEntries).Error)
		if len(auditEntries) >= 1 || time.Now().After(deadline) {
			require.Len(t, auditEntries, 1, "one summary audit entry expected")
			assert.Equal(t, "System", auditEntries[0].UserName)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}
