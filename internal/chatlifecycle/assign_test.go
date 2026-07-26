package chatlifecycle_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// systemMessagesFor returns the system messages of a contact filtered by
// system_type, oldest first.
func systemMessagesFor(t *testing.T, db *gorm.DB, contactID uuid.UUID, systemType string) []models.Message {
	t.Helper()
	var msgs []models.Message
	require.NoError(t, db.
		Where("contact_id = ? AND metadata->>'system_type' = ?", contactID, systemType).
		Order("created_at ASC").
		Find(&msgs).Error)
	return msgs
}

// TestService_Assign_WritesSystemMessage locks the admin "Assign to agent"
// side effects: status open, owner set, and a system message crediting the
// admin who assigned and the agent who received the chat.
func TestService_Assign_WritesSystemMessage(t *testing.T) {
	svc, db, org := newService(t)
	admin := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Ada Admin"))
	agent := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Gary Agent"))

	contact := testutil.CreateTestContact(t, db, org.ID)
	contact.SetStatus(models.ChatStatusPending)
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", contact.ID).
		Update("metadata", contact.Metadata).Error)

	require.NoError(t, svc.Assign(context.Background(), org.ID, admin.ID, contact, &agent.ID))

	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contact.ID).Error)
	require.NotNil(t, updated.AssignedUserID)
	assert.Equal(t, agent.ID, *updated.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, updated.EffectiveStatus())

	msgs := systemMessagesFor(t, db, contact.ID, "chat_assigned")
	require.Len(t, msgs, 1, "assignment must write exactly one system message")
	assert.Contains(t, msgs[0].Content, "Ada Admin")
	assert.Contains(t, msgs[0].Content, "Gary Agent")
	assert.Equal(t, true, msgs[0].Metadata["is_system_message"])
	assert.Equal(t, agent.ID.String(), msgs[0].Metadata["agent_id"])
	assert.Equal(t, admin.ID.String(), msgs[0].Metadata["assigned_by"])

	auditCountIs(t, db, contact.ID, 1, "assignment must persist an audit entry")
}

// TestService_Assign_SameOwner_Idempotent: re-assigning an open chat to its
// current owner must not spam a duplicate system message.
func TestService_Assign_SameOwner_Idempotent(t *testing.T) {
	svc, db, org := newService(t)
	admin := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Ada Admin"))
	agent := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Gary Agent"))

	contact := testutil.CreateTestContact(t, db, org.ID)
	require.NoError(t, svc.Assign(context.Background(), org.ID, admin.ID, contact, &agent.ID))
	require.NoError(t, svc.Assign(context.Background(), org.ID, admin.ID, contact, &agent.ID))

	msgs := systemMessagesFor(t, db, contact.ID, "chat_assigned")
	assert.Len(t, msgs, 1, "the idempotent re-assign must not write a second system message")
}

// TestService_Assign_Reassign_WritesNewSystemMessage: moving the chat from one
// agent to another records a second system message crediting the new agent.
func TestService_Assign_Reassign_WritesNewSystemMessage(t *testing.T) {
	svc, db, org := newService(t)
	admin := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Ada Admin"))
	first := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("First Agent"))
	second := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Second Agent"))

	contact := testutil.CreateTestContact(t, db, org.ID)
	require.NoError(t, svc.Assign(context.Background(), org.ID, admin.ID, contact, &first.ID))
	require.NoError(t, svc.Assign(context.Background(), org.ID, admin.ID, contact, &second.ID))

	msgs := systemMessagesFor(t, db, contact.ID, "chat_assigned")
	require.Len(t, msgs, 2)
	assert.Contains(t, msgs[1].Content, "Second Agent")

	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contact.ID).Error)
	require.NotNil(t, updated.AssignedUserID)
	assert.Equal(t, second.ID, *updated.AssignedUserID)
}

// TestService_Assign_NilTarget_DelegatesToRelease: unassigning through the
// assign endpoint behaves as an admin release — pending, unassigned, with the
// chat_released system message.
func TestService_Assign_NilTarget_DelegatesToRelease(t *testing.T) {
	svc, db, org := newService(t)
	admin := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Ada Admin"))
	agent := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Gary Agent"))

	contact := testutil.CreateTestContact(t, db, org.ID)
	claimForTest(t, db, contact, agent.ID)

	require.NoError(t, svc.Assign(context.Background(), org.ID, admin.ID, contact, nil))

	var updated models.Contact
	require.NoError(t, db.First(&updated, "id = ?", contact.ID).Error)
	assert.Nil(t, updated.AssignedUserID)
	assert.Equal(t, models.ChatStatusPending, updated.EffectiveStatus())

	msgs := systemMessagesFor(t, db, contact.ID, "chat_released")
	assert.Len(t, msgs, 1, "unassign must reuse the release system message")
}
