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

	app.markMessagesAsRead(orgID, contact.ID, contact, nil)

	statusOf := func(wamid string) models.MessageStatus {
		var msg models.Message
		require.NoError(t, db.Where("whats_app_message_id = ?", wamid).First(&msg).Error)
		return msg.Status
	}
	assert.Equal(t, models.MessageStatusRevoked, statusOf("MR-REV-1"), "revoked must survive the read sweep")
	assert.Equal(t, models.MessageStatusFailed, statusOf("MR-FAIL-1"), "failed must survive the read sweep")
	assert.Equal(t, models.MessageStatusRead, statusOf("MR-RECV-1"), "normal incoming must still become read")
}

// TestMarkReadForEveryone_WatermarkShapes pins the GREATEST behaviour in
// markReadForEveryone (read_state.go): a NULL watermark is raised to the
// newest message's read_seq, an existing watermark above the newest message
// is preserved, and a conversation with no messages keeps its watermark
// instead of being wiped (MAX over no rows is NULL, and GREATEST(watermark,
// NULL) is the watermark in PostgreSQL).
func TestMarkReadForEveryone_WatermarkShapes(t *testing.T) {
	db := testutil.SetupTestDB(t)
	app := &App{DB: db}
	org := testutil.CreateTestOrganization(t, db)

	watermarkOf := func(contactID uuid.UUID) *int64 {
		var watermark *int64
		require.NoError(t, db.Raw("SELECT read_watermark_seq FROM contacts WHERE id = ?", contactID).Scan(&watermark).Error)
		return watermark
	}
	newContact := func() *models.Contact {
		contact := &models.Contact{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: org.ID,
			// Contacts are unique per (org, phone): a blank phone collides
			// across the subtests' contacts.
			PhoneNumber: internalTestPhone(),
		}
		require.NoError(t, db.Create(contact).Error)
		return contact
	}
	addMessage := func(contactID uuid.UUID) int64 {
		msg := &models.Message{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: org.ID,
			ContactID:      contactID,
			Direction:      models.DirectionIncoming,
			MessageType:    models.MessageTypeText,
			Content:        "x",
			Status:         models.MessageStatusDelivered,
		}
		require.NoError(t, db.Create(msg).Error)
		var seq int64
		require.NoError(t, db.Raw("SELECT read_seq FROM messages WHERE id = ?", msg.ID).Scan(&seq).Error)
		return seq
	}

	t.Run("NULL watermark is raised to the newest message", func(t *testing.T) {
		contact := newContact()
		require.Nil(t, watermarkOf(contact.ID), "a fresh contact has no watermark")
		seq1 := addMessage(contact.ID)
		seq2 := addMessage(contact.ID)
		require.Greater(t, seq2, seq1)

		app.markReadForEveryone(contact.ID)

		got := watermarkOf(contact.ID)
		require.NotNil(t, got)
		assert.Equal(t, seq2, *got)
	})

	t.Run("watermark above the newest message is preserved", func(t *testing.T) {
		contact := newContact()
		seq := addMessage(contact.ID)
		require.NoError(t, db.Exec("UPDATE contacts SET read_watermark_seq = ? WHERE id = ?", seq+100, contact.ID).Error)

		app.markReadForEveryone(contact.ID)

		got := watermarkOf(contact.ID)
		require.NotNil(t, got)
		assert.Equal(t, seq+100, *got, "the watermark must never move backwards")
	})

	t.Run("no messages preserves the watermark", func(t *testing.T) {
		contact := newContact()
		require.NoError(t, db.Exec("UPDATE contacts SET read_watermark_seq = ? WHERE id = ?", 42, contact.ID).Error)

		app.markReadForEveryone(contact.ID)

		got := watermarkOf(contact.ID)
		require.NotNil(t, got)
		assert.Equal(t, int64(42), *got)
	})

	t.Run("no messages and no watermark stays NULL", func(t *testing.T) {
		contact := newContact()
		require.Nil(t, watermarkOf(contact.ID))

		app.markReadForEveryone(contact.ID)

		assert.Nil(t, watermarkOf(contact.ID), "nothing to raise the watermark to, and nothing to wipe")
	})
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

	create := func(wamid, status string) {
		require.NoError(t, db.Create(&models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New()},
			OrganizationID:    org.ID,
			ContactID:         contact.ID,
			WhatsAppMessageID: wamid,
			Direction:         models.DirectionIncoming,
			MessageType:       models.MessageTypeText,
			Content:           "x",
			Status:            models.MessageStatus(status),
		}).Error)
	}

	// Read state is per user: the viewer reads the first message...
	viewer := uuid.New()
	create("UR-READ", "read")
	app.markReadForUser(org.ID, viewer, contact.ID)
	// ...then three more arrive, two of them terminal.
	create("UR-RECV", "received")
	create("UR-REV", "revoked")
	create("UR-FAIL", "failed")

	resp := app.buildContactResponse(contact, org.ID, viewer)
	assert.Equal(t, 1, resp.UnreadCount,
		"only the plain 'received' message counts as unread")
}
