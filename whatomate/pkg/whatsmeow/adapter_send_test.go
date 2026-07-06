package whatsmeow

import (
	"path/filepath"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	waTypes "go.mau.fi/whatsmeow/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildTextMessageUsesConversationForPlainText(t *testing.T) {
	text := "زاهد : الرجاء تواصل مع قسم القرطاسية"

	msg := buildTextMessage(text)
	if msg == nil {
		t.Fatal("expected message")
	}
	if got := msg.GetConversation(); got != text {
		t.Fatalf("expected conversation text %q, got %q", text, got)
	}
	if msg.GetExtendedTextMessage() != nil {
		t.Fatal("expected no extended text message")
	}
}

func TestBuildTextMessageUsesExtendedTextForURLs(t *testing.T) {
	text := "قيم تجربتك / Rate your experience:\n5 ممتاز (😍Excellent)\nhttps://g.page/r/example/review"

	msg := buildTextMessage(text)
	if msg == nil {
		t.Fatal("expected message")
	}
	if msg.GetExtendedTextMessage() == nil {
		t.Fatal("expected extended text message")
	}
	if got := msg.GetExtendedTextMessage().GetText(); got != text {
		t.Fatalf("expected text %q, got %q", text, got)
	}
	if got := msg.GetConversation(); got != "" {
		t.Fatalf("expected empty conversation payload, got %q", got)
	}
}

func setupTestAdapterDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "whatsmeow-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)
	err = db.Exec(`
		CREATE TABLE messages (
			id TEXT PRIMARY KEY,
			organization_id TEXT,
			instance_id TEXT,
			conversation_id TEXT,
			whats_app_message_id TEXT,
			direction TEXT,
			message_type TEXT,
			content TEXT,
			metadata TEXT
		)
	`).Error
	require.NoError(t, err)
	return db
}

func TestResolveReplyContext_DirectChat_Incoming(t *testing.T) {
	t.Parallel()
	db := setupTestAdapterDB(t)

	orgID := uuid.New()
	instanceID := uuid.New().String()
	contactPhone := "1234567890"
	quotedMsgID := "wamid.incoming123"
	jid := waTypes.NewJID(contactPhone, waTypes.DefaultUserServer)
	conversationID := jid.String()

	// Seed an incoming message using raw SQL
	err := db.Exec(`
		INSERT INTO messages (id, organization_id, instance_id, conversation_id, whats_app_message_id, direction, message_type, content, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), orgID.String(), instanceID, conversationID, quotedMsgID, string(models.DirectionIncoming), string(models.MessageTypeText), "Hello agent!", "{}",
	).Error
	require.NoError(t, err)

	adapter := &WhatsmeowAdapter{db: db}
	myJID := waTypes.NewJID("9999999999", waTypes.DefaultUserServer)

	participant, quotedText := adapter.resolveReplyContext(jid, quotedMsgID, myJID, instanceID)

	assert.Equal(t, jid.String(), participant)
	assert.Equal(t, "Hello agent!", quotedText)
}

func TestResolveReplyContext_DirectChat_Outgoing(t *testing.T) {
	t.Parallel()
	db := setupTestAdapterDB(t)

	orgID := uuid.New()
	instanceID := uuid.New().String()
	contactPhone := "1234567890"
	quotedMsgID := "wamid.outgoing456"
	jid := waTypes.NewJID(contactPhone, waTypes.DefaultUserServer)
	conversationID := jid.String()

	// Seed an outgoing message using raw SQL
	err := db.Exec(`
		INSERT INTO messages (id, organization_id, instance_id, conversation_id, whats_app_message_id, direction, message_type, content, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), orgID.String(), instanceID, conversationID, quotedMsgID, string(models.DirectionOutgoing), string(models.MessageTypeText), "Hello customer! How can I assist?", "{}",
	).Error
	require.NoError(t, err)

	adapter := &WhatsmeowAdapter{db: db}
	myJID := waTypes.NewJID("9999999999", waTypes.DefaultUserServer)

	participant, quotedText := adapter.resolveReplyContext(jid, quotedMsgID, myJID, instanceID)

	assert.Equal(t, myJID.ToNonAD().String(), participant)
	assert.Equal(t, "Hello customer! How can I assist?", quotedText)
}

func TestResolveReplyContext_GroupChat_Incoming(t *testing.T) {
	t.Parallel()
	db := setupTestAdapterDB(t)

	orgID := uuid.New()
	instanceID := uuid.New().String()
	groupJidStr := "12345-group@g.us"
	quotedMsgID := "wamid.groupincoming789"
	senderPhone := "5556667777"
	jid, err := waTypes.ParseJID(groupJidStr)
	require.NoError(t, err)
	conversationID := jid.String()

	// Seed an incoming group message with sender_phone in metadata JSON
	err = db.Exec(`
		INSERT INTO messages (id, organization_id, instance_id, conversation_id, whats_app_message_id, direction, message_type, content, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), orgID.String(), instanceID, conversationID, quotedMsgID, string(models.DirectionIncoming), string(models.MessageTypeText), "Hello group!", `{"sender_phone":"5556667777"}`,
	).Error
	require.NoError(t, err)

	adapter := &WhatsmeowAdapter{db: db}
	myJID := waTypes.NewJID("9999999999", waTypes.DefaultUserServer)

	participant, quotedText := adapter.resolveReplyContext(jid, quotedMsgID, myJID, instanceID)

	expectedParticipant := senderPhone + "@s.whatsapp.net"
	assert.Equal(t, expectedParticipant, participant)
	assert.Equal(t, "Hello group!", quotedText)
}

func TestResolveReplyContext_EmptyContentFallback(t *testing.T) {
	t.Parallel()
	db := setupTestAdapterDB(t)

	orgID := uuid.New()
	instanceID := uuid.New().String()
	contactPhone := "1234567890"
	quotedMsgID := "wamid.media_no_caption"
	jid := waTypes.NewJID(contactPhone, waTypes.DefaultUserServer)
	conversationID := jid.String()

	// Seed a media message with empty content using raw SQL
	err := db.Exec(`
		INSERT INTO messages (id, organization_id, instance_id, conversation_id, whats_app_message_id, direction, message_type, content, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), orgID.String(), instanceID, conversationID, quotedMsgID, string(models.DirectionIncoming), string(models.MessageTypeImage), "", "{}",
	).Error
	require.NoError(t, err)

	adapter := &WhatsmeowAdapter{db: db}
	myJID := waTypes.NewJID("9999999999", waTypes.DefaultUserServer)

	_, quotedText := adapter.resolveReplyContext(jid, quotedMsgID, myJID, instanceID)

	assert.Equal(t, "Media", quotedText)
}

func TestResolvePollSender_OutgoingDirectPoll_UsesOwnJID(t *testing.T) {
	t.Parallel()
	chatJID := waTypes.NewJID("1234567890", waTypes.DefaultUserServer)
	ownJID := waTypes.NewJID("9999999999", waTypes.DefaultUserServer)

	msg := models.Message{Direction: models.DirectionOutgoing, Metadata: map[string]interface{}{}}

	got := resolvePollSender(msg, chatJID, ownJID, false)
	assert.Equal(t, ownJID, got, "outgoing poll should use own JID as sender")
}

func TestResolvePollSender_IncomingDirectPoll_UsesChatJID(t *testing.T) {
	t.Parallel()
	chatJID := waTypes.NewJID("1234567890", waTypes.DefaultUserServer)
	ownJID := waTypes.NewJID("9999999999", waTypes.DefaultUserServer)

	msg := models.Message{Direction: models.DirectionIncoming, Metadata: map[string]interface{}{}}

	got := resolvePollSender(msg, chatJID, ownJID, false)
	assert.Equal(t, chatJID, got, "incoming direct poll should use chat JID as sender")
}

func TestResolvePollSender_IncomingGroupPoll_UsesParticipantSender(t *testing.T) {
	t.Parallel()
	groupJID := waTypes.JID{User: "12345-group", Server: "g.us"}
	ownJID := waTypes.NewJID("9999999999", waTypes.DefaultUserServer)
	senderPhone := "5556667777"

	msg := models.Message{
		Direction: models.DirectionIncoming,
		Metadata:  map[string]interface{}{"sender_phone": senderPhone},
	}

	got := resolvePollSender(msg, groupJID, ownJID, true)
	expectedSender, _ := waTypes.ParseJID(senderPhone + "@s.whatsapp.net")
	assert.Equal(t, expectedSender, got, "incoming group poll should use participant sender")
}

func TestResolvePollSender_OutgoingGroupPoll_UsesOwnJID(t *testing.T) {
	t.Parallel()
	groupJID := waTypes.JID{User: "12345-group", Server: "g.us"}
	ownJID := waTypes.NewJID("9999999999", waTypes.DefaultUserServer)

	msg := models.Message{
		Direction: models.DirectionOutgoing,
		Metadata:  map[string]interface{}{"sender_phone": "5556667777"},
	}

	got := resolvePollSender(msg, groupJID, ownJID, true)
	assert.Equal(t, ownJID, got, "outgoing group poll should still use own JID")
}
