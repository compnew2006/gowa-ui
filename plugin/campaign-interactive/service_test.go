package campaigninteractive

import (
	"log/slog"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupPollTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE bulk_message_campaigns (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			organization_id TEXT,
			instance_id TEXT,
			name TEXT,
			status TEXT DEFAULT 'draft',
			body_content TEXT,
			poll_question TEXT DEFAULT '',
			poll_options JSON DEFAULT '[]',
			poll_max_selections INTEGER DEFAULT 0
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE bulk_message_recipients (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			organization_id TEXT,
			campaign_id TEXT,
			phone_number TEXT,
			whats_app_message_id TEXT DEFAULT ''
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE messages (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			organization_id TEXT,
			instance_id TEXT,
			contact_id TEXT,
			whats_app_message_id TEXT,
			conversation_id TEXT,
			direction TEXT,
			message_type TEXT,
			content TEXT,
			status TEXT,
			is_reply BOOLEAN DEFAULT FALSE,
			reply_to_message_id TEXT,
			metadata JSON,
			interactive_data JSON
		)
	`).Error)

	return db
}

func newTestPlugin(db *gorm.DB) *Plugin {
	return &Plugin{
		db:  db,
		log: slog.Default(),
	}
}

func seedPollCampaign(t *testing.T, db *gorm.DB) (orgID, campaignID uuid.UUID) {
	t.Helper()

	orgID = uuid.New()
	campaignID = uuid.New()

	require.NoError(t, db.Exec(`
		INSERT INTO bulk_message_campaigns (id, organization_id, name, status, poll_question, poll_options, poll_max_selections)
		VALUES (?, ?, 'Test Poll Campaign', 'completed', 'Did you enjoy this?', '["Yes","No","Maybe"]', 1)
	`, campaignID, orgID).Error)

	return orgID, campaignID
}

func TestGetPollVotes_CampaignNotFound(t *testing.T) {
	db := setupPollTestDB(t)
	p := newTestPlugin(db)

	resp, err := p.getPollVotes(db, uuid.New())
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestGetPollVotes_CampaignHasNoPoll(t *testing.T) {
	db := setupPollTestDB(t)
	p := newTestPlugin(db)

	campaignID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bulk_message_campaigns (id, organization_id, name, status, poll_question, poll_options)
		VALUES (?, ?, 'Text Campaign', 'completed', '', '[]')
	`, campaignID, uuid.New()).Error)

	resp, err := p.getPollVotes(db, campaignID)
	assert.NoError(t, err)
	assert.Nil(t, resp)
}

func TestGetPollVotes_NoRecipients(t *testing.T) {
	db := setupPollTestDB(t)
	p := newTestPlugin(db)

	_, campaignID := seedPollCampaign(t, db)

	resp, err := p.getPollVotes(db, campaignID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "Did you enjoy this?", resp.Question)
	assert.Equal(t, []string{"Yes", "No", "Maybe"}, resp.Options)
	assert.Equal(t, int64(0), resp.Total)
	require.Len(t, resp.Results, 3)
	for _, r := range resp.Results {
		assert.Equal(t, "0.0%", r.Percentage)
	}
}

func TestGetPollVotes_WithVotes(t *testing.T) {
	db := setupPollTestDB(t)
	p := newTestPlugin(db)

	orgID, campaignID := seedPollCampaign(t, db)

	recipientID := uuid.New()
	pollMsgID := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO bulk_message_recipients (id, organization_id, campaign_id, phone_number, whats_app_message_id)
		VALUES (?, ?, ?, '+1234567890', 'wa_poll_msg_1')
	`, recipientID, orgID, campaignID).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO messages (id, organization_id, message_type, content, direction, status, whats_app_message_id, interactive_data)
		VALUES (?, ?, 'poll', 'Did you enjoy this?', 'outgoing', 'sent', 'wa_poll_msg_1', '{"type":"poll","question":"Did you enjoy this?","options":["Yes","No","Maybe"],"max_selections":1}')
	`, pollMsgID, orgID).Error)

	vote1 := uuid.New()
	vote2 := uuid.New()
	vote3 := uuid.New()
	require.NoError(t, db.Exec(`
		INSERT INTO messages (id, organization_id, message_type, content, direction, status, is_reply, reply_to_message_id, interactive_data)
		VALUES
			(?, ?, 'poll', 'Voted: Yes', 'incoming', 'received', 1, ?, '{"type":"poll_vote","selected_options":["Yes"]}'),
			(?, ?, 'poll', 'Voted: Yes', 'incoming', 'received', 1, ?, '{"type":"poll_vote","selected_options":["Yes"]}'),
			(?, ?, 'poll', 'Voted: No', 'incoming', 'received', 1, ?, '{"type":"poll_vote","selected_options":["No"]}')
	`, vote1, orgID, pollMsgID, vote2, orgID, pollMsgID, vote3, orgID, pollMsgID).Error)

	resp, err := p.getPollVotes(db, campaignID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, int64(3), resp.Total)
	require.Len(t, resp.Results, 3)

	byOption := make(map[string]PollVote)
	for _, r := range resp.Results {
		byOption[r.Option] = r
	}
	assert.Equal(t, int64(2), byOption["Yes"].Count)
	assert.Equal(t, int64(1), byOption["No"].Count)
	assert.Equal(t, int64(0), byOption["Maybe"].Count)
	assert.Equal(t, "66.7%", byOption["Yes"].Percentage)
	assert.Equal(t, "33.3%", byOption["No"].Percentage)
}

func TestZeroResults(t *testing.T) {
	options := []string{"A", "B", "C"}
	results := zeroResults(options)

	require.Len(t, results, 3)
	for i, opt := range options {
		assert.Equal(t, opt, results[i].Option)
		assert.Equal(t, "0.0%", results[i].Percentage)
		assert.Equal(t, int64(0), results[i].Count)
	}
}

func TestPollOptionStrings_NilInput(t *testing.T) {
	var arr models.JSONBArray
	assert.Nil(t, arr.Strings())
}
