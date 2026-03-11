package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/chat_close_ratings"
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

func TestMaybeCaptureChatCloseRating_CapturesFollowupMessagesWithinLimit(t *testing.T) {
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
		Content:         "very good agent",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&secondIncoming).Error)

	capturedSecond := app.maybeCaptureChatCloseRating(org.ID, contact, incomingMessagePayload{MessageType: "text", MessageText: "very good agent"}, &secondIncoming)
	assert.True(t, capturedSecond)

	thirdIncoming := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: closedAt.Add(3 * time.Minute)},
		OrganizationID:  org.ID,
		WhatsAppAccount: account.Name,
		ContactID:       contact.ID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         "9",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&thirdIncoming).Error)

	capturedThird := app.maybeCaptureChatCloseRating(org.ID, contact, incomingMessagePayload{MessageType: "text", MessageText: "9"}, &thirdIncoming)
	assert.True(t, capturedThird)

	require.NoError(t, app.DB.Where("id = ?", cycle.ID).First(&updated).Error)
	require.NotNil(t, updated.Rating)
	assert.Equal(t, 9, *updated.Rating)
	assert.Equal(t, "9", updated.RatingMessage)

	rawFollowup, ok := updated.ContextMessages[chatCloseRatingFollowupContextKey]
	require.True(t, ok)
	require.NotNil(t, rawFollowup)

	var followup struct {
		RemainingMessages int              `json:"remaining_messages"`
		Entries           []map[string]any `json:"entries"`
		Comments          []string         `json:"comments"`
	}
	encodedFollowup, err := json.Marshal(rawFollowup)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(encodedFollowup, &followup))
	assert.Equal(t, 0, followup.RemainingMessages)
	assert.Len(t, followup.Entries, 3)
	assert.Equal(t, []string{"very good agent"}, followup.Comments)
}

func TestParseInboundRatingValue_SupportsLocalizedDigitsAndComments(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		input     string
		wantValue int
		wantOK    bool
	}{
		{name: "ascii integer", input: "7", wantValue: 7, wantOK: true},
		{name: "arabic indic integer", input: "٧", wantValue: 7, wantOK: true},
		{name: "persian integer", input: "۷", wantValue: 7, wantOK: true},
		{name: "ascii with comment", input: "7 great service", wantValue: 7, wantOK: true},
		{name: "arabic with comment", input: "٧ ممتاز", wantValue: 7, wantOK: true},
		{name: "leading rtl mark", input: "\u200f٧ ممتاز", wantValue: 7, wantOK: true},
		{name: "out of range", input: "11", wantValue: 0, wantOK: false},
		{name: "not leading number", input: "rating 7", wantValue: 0, wantOK: false},
		{name: "non rating token", input: "7pm", wantValue: 0, wantOK: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotValue, gotOK := chat_close_ratings.ParseInboundRatingValue(tc.input)
			assert.Equal(t, tc.wantOK, gotOK)
			assert.Equal(t, tc.wantValue, gotValue)
		})
	}
}

func TestMaybeCaptureChatCloseRating_AcceptsArabicDigitsAndMessage(t *testing.T) {
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
		CloseMessageLanguage: "ar",
		ContextMessages:      models.JSONB{},
	}
	require.NoError(t, app.DB.Create(&cycle).Error)

	incoming := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: closedAt.Add(1 * time.Minute)},
		OrganizationID:  org.ID,
		WhatsAppAccount: account.Name,
		ContactID:       contact.ID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         "٧ ممتاز",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&incoming).Error)

	captured := app.maybeCaptureChatCloseRating(org.ID, contact, incomingMessagePayload{
		MessageType: "text",
		MessageText: "٧ ممتاز",
	}, &incoming)
	assert.True(t, captured)

	var updated models.ChatClosureRating
	require.NoError(t, app.DB.Where("id = ?", cycle.ID).First(&updated).Error)
	require.NotNil(t, updated.Rating)
	assert.Equal(t, 7, *updated.Rating)
	assert.Equal(t, "٧ ممتاز", updated.RatingMessage)
	require.NotNil(t, updated.RatingMessageID)
	assert.Equal(t, incoming.ID, *updated.RatingMessageID)
}

func TestMaybeCaptureChatCloseRating_UpdatesRatingMessageWithFollowupComment(t *testing.T) {
	app := newProcessorTestApp(t)
	org, account := createProcessorTestOrg(t, app)
	closingAgent := testutil.CreateTestUser(t, app.DB, org.ID)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	closedAt := time.Now().UTC()
	rating := 8
	ratedAt := closedAt.Add(1 * time.Minute)
	cycle := models.ChatClosureRating{
		BaseModel:            models.BaseModel{ID: uuid.New()},
		OrganizationID:       org.ID,
		ContactID:            contact.ID,
		ChatID:               contact.ID,
		AgentUserID:          &agent.ID,
		ClosingAgentID:       closingAgent.ID,
		ClosedAt:             closedAt,
		State:                models.ChatClosureRatingStateRated,
		Rating:               &rating,
		RatedAt:              &ratedAt,
		RatingMessage:        "8",
		CloseMessage:         "Please rate 1-10",
		CloseMessageLanguage: "en",
		ContextMessages: models.JSONB{
			chatCloseRatingFollowupContextKey: models.JSONB{
				"expires_at":                       closedAt.Add(15 * time.Minute).Format(time.RFC3339),
				"remaining_messages":               2,
				chatCloseRatingFollowupEntriesKey:  []any{},
				chatCloseRatingFollowupCommentsKey: []any{},
			},
		},
	}
	require.NoError(t, app.DB.Create(&cycle).Error)

	commentIncoming := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: closedAt.Add(2 * time.Minute)},
		OrganizationID:  org.ID,
		WhatsAppAccount: account.Name,
		ContactID:       contact.ID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         "very good agent",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&commentIncoming).Error)

	captured := app.maybeCaptureChatCloseRating(org.ID, contact, incomingMessagePayload{
		MessageType: "text",
		MessageText: "very good agent",
	}, &commentIncoming)
	assert.True(t, captured)

	var updated models.ChatClosureRating
	require.NoError(t, app.DB.Where("id = ?", cycle.ID).First(&updated).Error)
	require.NotNil(t, updated.Rating)
	assert.Equal(t, 8, *updated.Rating)
	assert.Equal(t, "very good agent", updated.RatingMessage)
	require.NotNil(t, updated.RatingMessageID)
	assert.Equal(t, commentIncoming.ID, *updated.RatingMessageID)
}

func TestListAgentRatingRecords_MergesFollowupCommentsIntoRatingMessage(t *testing.T) {
	app := newProcessorTestApp(t)
	org, _ := createProcessorTestOrg(t, app)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)
	closingAgent := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	rating := 8
	ratedAt := time.Now().UTC().Add(-2 * time.Minute)
	cycle := models.ChatClosureRating{
		BaseModel:            models.BaseModel{ID: uuid.New()},
		OrganizationID:       org.ID,
		ContactID:            contact.ID,
		ChatID:               contact.ID,
		AgentUserID:          &agent.ID,
		ClosingAgentID:       closingAgent.ID,
		ClosedAt:             ratedAt.Add(-5 * time.Minute),
		State:                models.ChatClosureRatingStateRated,
		Rating:               &rating,
		RatedAt:              &ratedAt,
		RatingMessage:        "8",
		CloseMessage:         "Please rate 1-10",
		CloseMessageLanguage: "en",
		ContextMessages: models.JSONB{
			chatCloseRatingFollowupContextKey: models.JSONB{
				chatCloseRatingFollowupCommentsKey: []any{"very good agent", "quick support"},
				chatCloseRatingFollowupEntriesKey: []any{
					map[string]any{
						"kind":    "comment",
						"content": "quick support",
					},
				},
			},
		},
	}
	require.NoError(t, app.DB.Create(&cycle).Error)

	records, err := app.listAgentRatingRecords(
		org.ID,
		time.Now().UTC().Add(-24*time.Hour),
		time.Now().UTC(),
		nil,
		nil,
		nil,
		nil,
		10,
	)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "8 | very good agent | quick support", records[0].RatingMessage)
	require.NotNil(t, records[0].ContextMessages)
	_, hasFollowup := records[0].ContextMessages[chatCloseRatingFollowupContextKey]
	assert.True(t, hasFollowup)
}

func TestDecodeRatingContextMessages_ParsesJSONBPayload(t *testing.T) {
	decoded := decodeRatingContextMessages(json.RawMessage(`{"followup":{"comments":["text 1","text 2"]}}`))
	require.NotNil(t, decoded)

	rawFollowup, ok := decoded[chatCloseRatingFollowupContextKey]
	require.True(t, ok)
	followup, ok := rawFollowup.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []string{"text 1", "text 2"}, asStringSlice(followup[chatCloseRatingFollowupCommentsKey]))
}

func TestReadChatCloseRatingSettings(t *testing.T) {
	orgSettings := models.JSONB{
		"chat_close_rating_enabled":                 true,
		"chat_close_rating_followup_window_minutes": 20,
		"chat_close_rating_templates": map[string]interface{}{
			"en": "Org English",
		},
	}

	instanceSettings := models.JSONB{
		"chat_close_rating_enabled":                 false,
		"chat_close_rating_followup_window_minutes": 10,
		"chat_close_rating_templates": map[string]interface{}{
			"en": "Instance English",
			"es": "Instance Spanish",
		},
	}

	// Test org settings only
	settingsOrgOnly := readChatCloseRatingSettings(orgSettings, nil)
	assert.True(t, settingsOrgOnly.Enabled)
	assert.Equal(t, 20, settingsOrgOnly.FollowupWindowMinutes)
	assert.Equal(t, "Org English", settingsOrgOnly.Templates["en"])
	assert.Equal(t, defaultChatCloseRatingTemplates["es"], settingsOrgOnly.Templates["es"])

	// Test override with instance settings
	settingsOverride := readChatCloseRatingSettings(orgSettings, instanceSettings)
	assert.False(t, settingsOverride.Enabled)
	assert.Equal(t, 10, settingsOverride.FollowupWindowMinutes)
	assert.Equal(t, "Instance English", settingsOverride.Templates["en"])
	assert.Equal(t, "Instance Spanish", settingsOverride.Templates["es"])
}
