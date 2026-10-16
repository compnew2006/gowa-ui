package handlers

import (
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Per-user unread state. Each user keeps their own read position per
// conversation (models.ContactReadState), so opening a chat clears the
// unread badge for that user only. Positions are messages.read_seq values —
// INSERT order, not created_at, so history-synced or late-delivered messages
// stamped with an older time still count as unread. contacts.read_watermark_seq
// is the conversation-wide floor (automated replies, and the read state from
// before per-user tracking).
//
// The WhatsApp-side read state (messages.status = read, blue ticks) stays
// conversation-wide: the customer sees one read receipt, sent the first time
// anyone opens the chat (markMessagesAsRead).

// unreadMessagesQuery selects the incoming messages of contactIDs still
// unread for userID. Terminal statuses (revoked/failed) never count: the read
// sweep deliberately skips them, so counting them would pin the badge forever.
func (a *App) unreadMessagesQuery(orgID, userID uuid.UUID, contactIDs []uuid.UUID) *gorm.DB {
	return a.DB.Table("messages AS m").
		Joins("JOIN contacts c ON c.id = m.contact_id").
		Joins("LEFT JOIN contact_read_states r ON r.contact_id = m.contact_id AND r.user_id = ?", userID).
		Where("m.organization_id = ? AND m.contact_id IN ? AND m.direction = ? AND m.status NOT IN ? AND m.deleted_at IS NULL",
			orgID, contactIDs, models.DirectionIncoming,
			[]models.MessageStatus{models.MessageStatusRevoked, models.MessageStatusFailed}).
		Where("m.read_seq > COALESCE(GREATEST(c.read_watermark_seq, r.last_read_seq), 0)")
}

// unreadCountsForUser returns the unread incoming message count per contact
// as seen by userID (contacts without unread messages are absent).
func (a *App) unreadCountsForUser(orgID, userID uuid.UUID, contactIDs []uuid.UUID) map[uuid.UUID]int64 {
	counts := make(map[uuid.UUID]int64, len(contactIDs))
	if len(contactIDs) == 0 {
		return counts
	}
	var rows []struct {
		ContactID   uuid.UUID
		UnreadCount int64
	}
	if err := a.unreadMessagesQuery(orgID, userID, contactIDs).
		Select("m.contact_id, COUNT(*) AS unread_count").
		Group("m.contact_id").
		Scan(&rows).Error; err != nil {
		a.Log.Error("Failed to count unread messages", "error", err, "user_id", userID)
		return counts
	}
	for _, row := range rows {
		counts[row.ContactID] = row.UnreadCount
	}
	return counts
}

// markReadForUser moves userID's read position in the conversation to its
// newest message, clearing the unread badge for this user only.
func (a *App) markReadForUser(orgID, userID, contactID uuid.UUID) {
	if err := a.DB.Exec(`
		INSERT INTO contact_read_states (contact_id, user_id, organization_id, last_read_seq, updated_at)
		SELECT ?, ?, ?, COALESCE(MAX(read_seq), 0), NOW() FROM messages WHERE contact_id = ?
		ON CONFLICT (contact_id, user_id) DO UPDATE
			SET last_read_seq = GREATEST(contact_read_states.last_read_seq, EXCLUDED.last_read_seq),
			    updated_at = NOW()`,
		contactID, userID, orgID, contactID).Error; err != nil {
		a.Log.Error("Failed to update read position", "error", err, "contact_id", contactID, "user_id", userID)
	}
}

// markReadForEveryone raises the conversation-wide read watermark to its
// newest message: messages handled automatically (an automated reply went
// out) stop counting as unread for every user.
func (a *App) markReadForEveryone(contactID uuid.UUID) {
	if err := a.DB.Exec(`
		UPDATE contacts SET read_watermark_seq = GREATEST(read_watermark_seq,
			(SELECT MAX(read_seq) FROM messages WHERE contact_id = ?))
		WHERE id = ?`, contactID, contactID).Error; err != nil {
		a.Log.Error("Failed to update read watermark", "error", err, "contact_id", contactID)
	}
}
