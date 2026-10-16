package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
)

// An automated reply (MarkIncomingRead, issue #280) handles the conversation
// for everyone: its unread messages stop counting for every user, while a
// person reading only clears their own badge.
func TestMarkMessagesAsRead_AutomatedReplyClearsForEveryone(t *testing.T) {
	db := testutil.SetupTestDB(t)
	app := &App{DB: db}
	org := testutil.CreateTestOrganization(t, db)
	contact := &models.Contact{BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: org.ID}
	require.NoError(t, db.Create(contact).Error)
	userA, userB := uuid.New(), uuid.New()

	incoming := func() {
		require.NoError(t, db.Create(&models.Message{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: org.ID,
			ContactID:      contact.ID,
			Direction:      models.DirectionIncoming,
			MessageType:    models.MessageTypeText,
			Content:        "hi",
			Status:         models.MessageStatusDelivered,
		}).Error)
	}
	unread := func(user uuid.UUID) int64 {
		return app.unreadCountsForUser(org.ID, user, []uuid.UUID{contact.ID})[contact.ID]
	}

	incoming()
	incoming()
	app.markMessagesAsRead(org.ID, contact.ID, contact, &userA)
	assert.Equal(t, int64(0), unread(userA))
	assert.Equal(t, int64(2), unread(userB))

	incoming()
	app.markMessagesAsRead(org.ID, contact.ID, contact, nil) // automated reply went out
	assert.Equal(t, int64(0), unread(userA))
	assert.Equal(t, int64(0), unread(userB), "an automated reply clears the badge for every user")

	incoming()
	assert.Equal(t, int64(1), unread(userA))
	assert.Equal(t, int64(1), unread(userB))
}
