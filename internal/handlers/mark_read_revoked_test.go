package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
)

// TestMarkMessagesAsRead_PreservesRevoked pins the refresh-un-deletes bug fix:
// opening/refreshing a chat sweeps its incoming messages to "read", and that
// sweep must NEVER touch revoked (or failed) messages. Before the fix the
// bulk update had no status filter, so every refresh flipped revoked rows
// back to read — and since revoke preserves the original content, the chat
// rendered the deleted message as if it were never unsent.
func TestMarkMessagesAsRead_PreservesRevoked(t *testing.T) {
	db := testutil.SetupTestDB(t)

	app := &App{DB: db}
	org := testutil.CreateTestOrganization(t, db)
	orgID := org.ID
	contact := &models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
	}
	require.NoError(t, db.Create(contact).Error)

	makeMsg := func(wamid string, status models.MessageStatus) *models.Message {
		msg := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New()},
			OrganizationID:    orgID,
			ContactID:         contact.ID,
			WhatsAppAccount:   "", // no account → read-receipt goroutine is skipped
			WhatsAppMessageID: wamid,
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "x",
			Status:            status,
		}
		require.NoError(t, db.Create(msg).Error)
		return msg
	}

	makeMsg("MR-REV-1", models.MessageStatusRevoked)
	makeMsg("MR-FAIL-1", models.MessageStatusFailed)
	makeMsg("MR-RECV-1", models.MessageStatusReceived)

	app.markMessagesAsRead(orgID, contact.ID, contact)

	statusOf := func(wamid string) models.MessageStatus {
		var msg models.Message
		require.NoError(t, db.Where("whats_app_message_id = ?", wamid).First(&msg).Error)
		return msg.Status
	}
	assert.Equal(t, models.MessageStatusRevoked, statusOf("MR-REV-1"), "revoked must survive the read sweep")
	assert.Equal(t, models.MessageStatusFailed, statusOf("MR-FAIL-1"), "failed must survive the read sweep")
	assert.Equal(t, models.MessageStatusRead, statusOf("MR-RECV-1"), "normal incoming must still become read")
}

// TestUnreadCount_ExcludesRevokedAndFailed pins the badge semantics: the
// unread counter (sidebar badge) must not count revoked/failed messages.
// markMessagesAsRead never sweeps those (revoked stays revoked by design),
// so counting them made the badge reappear on every refresh forever for any
// conversation containing one deleted message.
func TestUnreadCount_ExcludesRevokedAndFailed(t *testing.T) {
	db := testutil.SetupTestDB(t)
	app := &App{DB: db}
	org := testutil.CreateTestOrganization(t, db)
	contact := &models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
	}
	require.NoError(t, db.Create(contact).Error)

	for _, tc := range []struct{ wamid, status string }{
		{"UR-RECV", "received"},
		{"UR-REV", "revoked"},
		{"UR-FAIL", "failed"},
		{"UR-READ", "read"},
	} {
		require.NoError(t, db.Create(&models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New()},
			OrganizationID:    org.ID,
			ContactID:         contact.ID,
			WhatsAppMessageID: tc.wamid,
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "x",
			Status:            models.MessageStatus(tc.status),
		}).Error)
	}

	resp := app.buildContactResponse(contact, org.ID, uuid.New())
	assert.Equal(t, 1, resp.UnreadCount,
		"only the plain 'received' message counts as unread")
}
