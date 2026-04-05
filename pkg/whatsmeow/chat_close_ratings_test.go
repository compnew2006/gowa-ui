package whatsmeow

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
	waClient "go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

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
			gotValue, gotOK := parseInboundRatingValue(tc.input)
			assert.Equal(t, tc.wantOK, gotOK)
			assert.Equal(t, tc.wantValue, gotValue)
		})
	}
}

func TestPersistParsedMessage_DoesNotReopenClosedChatForPendingRatingReply(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Rating Org",
		Slug:      "rating-org-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Primary",
		PhoneNumber:    "15550004444",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	assignee := testutil.CreateTestUser(t, db, org.ID)
	closingAgent := testutil.CreateTestUser(t, db, org.ID)
	closedAt := time.Now().UTC().Add(-2 * time.Minute)

	contact := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instance.ID,
		PhoneNumber:    "15550001111",
		ProfileName:    "Customer",
		Status:         models.ChatStatusClosed,
		AssignedUserID: &assignee.ID,
		ClosedAt:       &closedAt,
		ClosedByUserID: &closingAgent.ID,
	}
	require.NoError(t, db.Create(&contact).Error)

	cycle := models.ChatClosureRating{
		BaseModel:            models.BaseModel{ID: uuid.New()},
		OrganizationID:       org.ID,
		ContactID:            contact.ID,
		ChatID:               contact.ID,
		AgentUserID:          &assignee.ID,
		ClosingAgentID:       closingAgent.ID,
		ClosedAt:             closedAt,
		State:                models.ChatClosureRatingStatePending,
		CloseMessage:         "Please rate 1-10",
		CloseMessageLanguage: "en",
		ContextMessages:      models.JSONB{},
	}
	require.NoError(t, db.Create(&cycle).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")

	myJID, err := types.ParseJID(instance.PhoneNumber + "@s.whatsapp.net")
	require.NoError(t, err)
	client := &waClient.Client{
		Store: &store.Device{ID: &myJID},
	}

	chatJID, err := types.ParseJID(contact.PhoneNumber + "@s.whatsapp.net")
	require.NoError(t, err)
	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   chatJID,
				IsFromMe: false,
			},
			ID:        "wamid.rating.reply.1",
			Timestamp: time.Now().UTC(),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("٧ ممتاز"),
		},
	}

	savedMessage, err := cm.persistParsedMessage(context.Background(), client, evt, instance.ID, org.ID, persistMessageOptions{
		AllowFromMe:   false,
		Broadcast:     false,
		HistorySync:   false,
		UpdateMetrics: false,
	})
	require.NoError(t, err)
	require.NotNil(t, savedMessage)

	var refreshedContact models.Contact
	require.NoError(t, db.First(&refreshedContact, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusClosed, refreshedContact.EffectiveStatus())
	require.NotNil(t, refreshedContact.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshedContact.AssignedUserID)
	assert.Equal(t, instance.PhoneNumber, refreshedContact.WhatsAppAccount)

	var refreshedCycle models.ChatClosureRating
	require.NoError(t, db.First(&refreshedCycle, "id = ?", cycle.ID).Error)
	assert.Equal(t, models.ChatClosureRatingStateRated, refreshedCycle.State)
	require.NotNil(t, refreshedCycle.Rating)
	assert.Equal(t, 7, *refreshedCycle.Rating)
	assert.Equal(t, "٧ ممتاز", refreshedCycle.RatingMessage)
	require.NotNil(t, refreshedCycle.RatingMessageID)
	assert.Equal(t, savedMessage.ID, *refreshedCycle.RatingMessageID)
	assert.Equal(t, instance.PhoneNumber, savedMessage.WhatsAppAccount)
}

func TestConnectionManagerLoadChatCloseRatingSettings_UsesInstanceSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Instance Settings Org",
		Slug:      "instance-settings-org-" + uuid.NewString(),
		Settings: models.JSONB{
			"chat_close_rating_enabled":                 true,
			"chat_close_rating_followup_window_minutes": 5,
		},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Primary",
		PhoneNumber:    "15550009999",
		Settings: models.JSONB{
			"chat_close_rating_enabled":                 false,
			"chat_close_rating_followup_window_minutes": 45,
		},
	}
	require.NoError(t, db.Create(&instance).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")

	settings, err := cm.loadChatCloseRatingSettings(context.Background(), &instance.ID)
	require.NoError(t, err)
	assert.False(t, settings.Enabled)
	assert.Equal(t, 45, settings.FollowupWindowMinutes)
}

func TestPersistParsedMessage_DoesNotReopenClosedChatForFollowupComment(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Reopen Org",
		Slug:      "reopen-org-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Primary",
		PhoneNumber:    "15550005555",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	assignee := testutil.CreateTestUser(t, db, org.ID)
	closingAgent := testutil.CreateTestUser(t, db, org.ID)
	closedAt := time.Now().UTC().Add(-2 * time.Minute)

	contact := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instance.ID,
		PhoneNumber:    "15550002222",
		ProfileName:    "Customer",
		Status:         models.ChatStatusClosed,
		AssignedUserID: &assignee.ID,
		ClosedAt:       &closedAt,
		ClosedByUserID: &closingAgent.ID,
	}
	require.NoError(t, db.Create(&contact).Error)

	cycle := models.ChatClosureRating{
		BaseModel:            models.BaseModel{ID: uuid.New()},
		OrganizationID:       org.ID,
		ContactID:            contact.ID,
		ChatID:               contact.ID,
		AgentUserID:          &assignee.ID,
		ClosingAgentID:       closingAgent.ID,
		ClosedAt:             closedAt,
		State:                models.ChatClosureRatingStatePending,
		CloseMessage:         "Please rate 1-10",
		CloseMessageLanguage: "en",
		ContextMessages:      models.JSONB{},
	}
	require.NoError(t, db.Create(&cycle).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")

	myJID, err := types.ParseJID(instance.PhoneNumber + "@s.whatsapp.net")
	require.NoError(t, err)
	client := &waClient.Client{
		Store: &store.Device{ID: &myJID},
	}

	chatJID, err := types.ParseJID(contact.PhoneNumber + "@s.whatsapp.net")
	require.NoError(t, err)
	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   chatJID,
				IsFromMe: false,
			},
			ID:        "wamid.rating.reply.2",
			Timestamp: time.Now().UTC(),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("Thanks for your help"),
		},
	}

	savedMessage, err := cm.persistParsedMessage(context.Background(), client, evt, instance.ID, org.ID, persistMessageOptions{
		AllowFromMe:   false,
		Broadcast:     false,
		HistorySync:   false,
		UpdateMetrics: false,
	})
	require.NoError(t, err)
	require.NotNil(t, savedMessage)

	var refreshedContact models.Contact
	require.NoError(t, db.First(&refreshedContact, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusClosed, refreshedContact.EffectiveStatus())
	require.NotNil(t, refreshedContact.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshedContact.AssignedUserID)
	require.NotNil(t, refreshedContact.ClosedAt)
	require.NotNil(t, refreshedContact.ClosedByUserID)
	assert.Equal(t, closingAgent.ID, *refreshedContact.ClosedByUserID)

	var refreshedCycle models.ChatClosureRating
	require.NoError(t, db.First(&refreshedCycle, "id = ?", cycle.ID).Error)
	assert.Equal(t, models.ChatClosureRatingStatePending, refreshedCycle.State)
	assert.Nil(t, refreshedCycle.Rating)
	assert.Nil(t, refreshedCycle.RatedAt)
	assert.Nil(t, refreshedCycle.RatingMessageID)
	rawFollowup, ok := refreshedCycle.ContextMessages[chatCloseRatingFollowupContextKey]
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
	assert.Equal(t, 2, followup.RemainingMessages)
	assert.Len(t, followup.Entries, 1)
	assert.Equal(t, []string{"Thanks for your help"}, followup.Comments)
}

func TestReopenClosedContactOnIncoming_ReopensForNonTextFollowupMessage(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Reopen Media Org",
		Slug:      "reopen-media-org-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Primary",
		PhoneNumber:    "15550006666",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	assignee := testutil.CreateTestUser(t, db, org.ID)
	closingAgent := testutil.CreateTestUser(t, db, org.ID)
	closedAt := time.Now().UTC().Add(-2 * time.Minute)

	contact := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instance.ID,
		PhoneNumber:    "15550003333",
		ProfileName:    "Customer",
		Status:         models.ChatStatusClosed,
		AssignedUserID: &assignee.ID,
		ClosedAt:       &closedAt,
		ClosedByUserID: &closingAgent.ID,
	}
	require.NoError(t, db.Create(&contact).Error)

	cycle := models.ChatClosureRating{
		BaseModel:            models.BaseModel{ID: uuid.New()},
		OrganizationID:       org.ID,
		ContactID:            contact.ID,
		ChatID:               contact.ID,
		AgentUserID:          &assignee.ID,
		ClosingAgentID:       closingAgent.ID,
		ClosedAt:             closedAt,
		State:                models.ChatClosureRatingStatePending,
		CloseMessage:         "Please rate 1-10",
		CloseMessageLanguage: "en",
		ContextMessages:      models.JSONB{},
	}
	require.NoError(t, db.Create(&cycle).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")
	require.NoError(t, cm.reopenClosedContactOnIncoming(context.Background(), &contact, models.MessageTypeImage, "file"))

	var refreshedContact models.Contact
	require.NoError(t, db.First(&refreshedContact, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusPending, refreshedContact.EffectiveStatus())
	assert.Nil(t, refreshedContact.AssignedUserID)
	assert.Nil(t, refreshedContact.ClosedAt)
	assert.Nil(t, refreshedContact.ClosedByUserID)
}

func TestPersistParsedMessage_UpdatesRatingMessageWithFollowupComment(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Rated Followup Org",
		Slug:      "rated-followup-org-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Primary",
		PhoneNumber:    "15550007777",
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instance).Error)

	assignee := testutil.CreateTestUser(t, db, org.ID)
	closingAgent := testutil.CreateTestUser(t, db, org.ID)
	closedAt := time.Now().UTC().Add(-2 * time.Minute)

	contact := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instance.ID,
		PhoneNumber:    "15550004444",
		ProfileName:    "Customer",
		Status:         models.ChatStatusClosed,
		AssignedUserID: &assignee.ID,
		ClosedAt:       &closedAt,
		ClosedByUserID: &closingAgent.ID,
	}
	require.NoError(t, db.Create(&contact).Error)

	rating := 8
	ratedAt := closedAt.Add(1 * time.Minute)
	cycle := models.ChatClosureRating{
		BaseModel:            models.BaseModel{ID: uuid.New()},
		OrganizationID:       org.ID,
		ContactID:            contact.ID,
		ChatID:               contact.ID,
		AgentUserID:          &assignee.ID,
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
	require.NoError(t, db.Create(&cycle).Error)

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")

	myJID, err := types.ParseJID(instance.PhoneNumber + "@s.whatsapp.net")
	require.NoError(t, err)
	client := &waClient.Client{
		Store: &store.Device{ID: &myJID},
	}

	chatJID, err := types.ParseJID(contact.PhoneNumber + "@s.whatsapp.net")
	require.NoError(t, err)
	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   chatJID,
				IsFromMe: false,
			},
			ID:        "wamid.rating.reply.followup.comment",
			Timestamp: time.Now().UTC(),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("very good agent"),
		},
	}

	savedMessage, err := cm.persistParsedMessage(context.Background(), client, evt, instance.ID, org.ID, persistMessageOptions{
		AllowFromMe:   false,
		Broadcast:     false,
		HistorySync:   false,
		UpdateMetrics: false,
	})
	require.NoError(t, err)
	require.NotNil(t, savedMessage)

	var refreshedCycle models.ChatClosureRating
	require.NoError(t, db.First(&refreshedCycle, "id = ?", cycle.ID).Error)
	require.NotNil(t, refreshedCycle.Rating)
	assert.Equal(t, 8, *refreshedCycle.Rating)
	assert.Equal(t, "very good agent", refreshedCycle.RatingMessage)
	require.NotNil(t, refreshedCycle.RatingMessageID)
	assert.Equal(t, savedMessage.ID, *refreshedCycle.RatingMessageID)
}
