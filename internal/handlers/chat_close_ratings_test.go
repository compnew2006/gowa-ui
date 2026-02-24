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

func TestHandleManualChatCloseRatingPrompt_CreatesPendingCycleAndPromptMessage(t *testing.T) {
	app := newProcessorTestApp(t)
	org, account := createProcessorTestOrg(t, app)
	closingAgent := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	contact.AssignedUserID = &closingAgent.ID

	app.handleManualChatCloseRatingPrompt(org.ID, closingAgent.ID, contact)

	var cycle models.ChatClosureRating
	require.NoError(t, app.DB.Where("organization_id = ? AND contact_id = ?", org.ID, contact.ID).First(&cycle).Error)

	assert.Equal(t, models.ChatClosureRatingStatePending, cycle.State)
	assert.Equal(t, closingAgent.ID, cycle.ClosingAgentID)
	require.NotNil(t, cycle.AgentUserID)
	assert.Equal(t, closingAgent.ID, *cycle.AgentUserID)
	assert.Contains(t, cycle.CloseMessage, contact.ID.String())
	assert.Contains(t, cycle.CloseMessage, "1 to 10")
	require.NotNil(t, cycle.CloseMessageID)

	var promptMessage models.Message
	require.NoError(t, app.DB.Where("id = ?", *cycle.CloseMessageID).First(&promptMessage).Error)
	assert.Equal(t, models.DirectionOutgoing, promptMessage.Direction)
	assert.Equal(t, models.MessageTypeText, promptMessage.MessageType)
	assert.Equal(t, cycle.CloseMessage, promptMessage.Content)
}

func TestMaybeCaptureChatCloseRating_StoresSingleRatingPerCycle(t *testing.T) {
	app := newProcessorTestApp(t)
	org, account := createProcessorTestOrg(t, app)
	closingAgent := testutil.CreateTestUser(t, app.DB, org.ID)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	closedAt := time.Now().UTC()
	cycle := models.ChatClosureRating{
		BaseModel:            models.BaseModel{ID: uuid.New()},
		OrganizationID:       org.ID,
		ContactID:            contact.ID,
		ChatID:               contact.ID,
		AgentUserID:          &agent.ID,
		ClosingAgentID:       closingAgent.ID,
		ClosedAt:             closedAt,
		State:                models.ChatClosureRatingStatePending,
		CloseMessage:         "Please rate 1-10",
		CloseMessageLanguage: "en",
		ContextMessages:      models.JSONB{},
	}
	require.NoError(t, app.DB.Create(&cycle).Error)

	firstIncoming := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: closedAt.Add(1 * time.Minute)},
		OrganizationID:  org.ID,
		WhatsAppAccount: account.Name,
		ContactID:       contact.ID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         "8",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&firstIncoming).Error)

	captured := app.maybeCaptureChatCloseRating(org.ID, contact, incomingMessagePayload{MessageType: "text", MessageText: "8"}, &firstIncoming)
	assert.True(t, captured)

	var updated models.ChatClosureRating
	require.NoError(t, app.DB.Where("id = ?", cycle.ID).First(&updated).Error)
	assert.Equal(t, models.ChatClosureRatingStateRated, updated.State)
	require.NotNil(t, updated.Rating)
	assert.Equal(t, 8, *updated.Rating)
	require.NotNil(t, updated.RatedAt)
	require.NotNil(t, updated.RatingMessageID)
	assert.Equal(t, firstIncoming.ID, *updated.RatingMessageID)
	assert.Equal(t, "8", updated.RatingMessage)

	secondIncoming := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: closedAt.Add(2 * time.Minute)},
		OrganizationID:  org.ID,
		WhatsAppAccount: account.Name,
		ContactID:       contact.ID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         "9",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&secondIncoming).Error)

	capturedSecond := app.maybeCaptureChatCloseRating(org.ID, contact, incomingMessagePayload{MessageType: "text", MessageText: "9"}, &secondIncoming)
	assert.False(t, capturedSecond)

	require.NoError(t, app.DB.Where("id = ?", cycle.ID).First(&updated).Error)
	require.NotNil(t, updated.Rating)
	assert.Equal(t, 8, *updated.Rating)
}
