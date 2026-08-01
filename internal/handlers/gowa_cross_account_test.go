package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/gowa-ui/internal/config"
	"github.com/shridarpatil/gowa-ui/internal/models"
	"github.com/shridarpatil/gowa-ui/pkg/gowa"
	"github.com/shridarpatil/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests lock the account-scoped wamid dedup: when two WhatsApp accounts
// of the same org message each other, the identical WhatsApp message ID exists
// once per account (the sender's outgoing copy and the recipient's incoming
// copy). A global dedup used to drop the recipient's copy entirely, so the
// receiving account's conversation stayed empty.

// textIncoming builds a plain-text IncomingTextMessage.
func textIncoming(wamid, fromPhone, toPhone, body string) IncomingTextMessage {
	return IncomingTextMessage{
		From:      fromPhone,
		To:        toPhone,
		ID:        wamid,
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      "text",
		Text: &struct {
			Body string `json:"body"`
		}{Body: body},
	}
}

// createSenderCopy stores the sender account's outgoing row for a wamid, as
// the UI send path / phone echo would.
func createSenderCopy(t *testing.T, app *App, orgID uuid.UUID, accountName, wamid string, contactID uuid.UUID) *models.Message {
	t.Helper()
	msg := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    orgID,
		WhatsAppAccount:   accountName,
		ContactID:         contactID,
		WhatsAppMessageID: wamid,
		Direction:         models.DirectionOutgoing,
		MessageType:       models.MessageTypeText,
		Content:           "cross-account test",
		Status:            models.MessageStatusSent,
	}
	require.NoError(t, app.DB.Create(msg).Error)
	return msg
}

// newCrossAccountApp builds a lifecycle test app with a non-nil config, since
// the incoming path resolves the account via getWhatsAppAccountCached which
// decrypts secrets using Config.App.EncryptionKey.
func newCrossAccountApp(t *testing.T) *App {
	t.Helper()
	app := newOutgoingLifecycleApp(t)
	app.Config = &config.Config{}
	return app
}

func TestProcessIncomingMessage_CrossAccountSameWamidKeepsBothCopies(t *testing.T) {
	app := newCrossAccountApp(t)
	if app.Redis == nil {
		t.Skip("TEST_REDIS_URL not set, skipping cached-account test")
	}
	org, sender := createProcessorTestOrg(t, app)
	receiver := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID,
		testutil.WithAccountName("receiver-"+uuid.New().String()[:8]))

	// The webhook resolves the receiving device by its connected JID.
	senderPhone := uniquePhone()
	receiverPhone := uniquePhone()
	receiverJID := receiverPhone + "@s.whatsapp.net"
	require.NoError(t, app.DB.Model(receiver).Update("gowa_jid", receiverJID).Error)

	// The sender account already stored its outgoing copy under this wamid.
	wamid := "XWAMID_" + uuid.New().String()[:12]
	senderContact := testutil.CreateTestContact(t, app.DB, org.ID)
	createSenderCopy(t, app, org.ID, sender.Name, wamid, senderContact.ID)

	// The receiving device now delivers the same wamid as an incoming message.
	incoming := textIncoming(wamid, senderPhone, receiverPhone, "hello across accounts")
	app.processIncomingMessage(receiver, receiverJID, incoming, "Sender", false, false, "", "")

	var stored models.Message
	require.NoError(t, app.DB.Where(
		"whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
		wamid, org.ID, receiver.Name).First(&stored).Error,
		"the receiving account's incoming copy must be saved despite the sender's copy sharing the wamid")
	assert.Equal(t, models.DirectionIncoming, stored.Direction)

	// The sender's copy must be untouched — exactly two rows share the wamid.
	var count int64
	app.DB.Model(&models.Message{}).
		Where("whats_app_message_id = ? AND organization_id = ?", wamid, org.ID).
		Count(&count)
	assert.EqualValues(t, 2, count)
}

func TestProcessIncomingMessage_SameAccountDuplicateStillSkipped(t *testing.T) {
	app := newCrossAccountApp(t)
	if app.Redis == nil {
		t.Skip("TEST_REDIS_URL not set, skipping cached-account test")
	}
	org, account := createProcessorTestOrg(t, app)

	senderPhone := uniquePhone()
	accountJID := uniquePhone() + "@s.whatsapp.net"
	require.NoError(t, app.DB.Model(account).Update("gowa_jid", accountJID).Error)

	wamid := "XWAMID_" + uuid.New().String()[:12]
	incoming := textIncoming(wamid, senderPhone, gowa.PhoneFromJID(accountJID), "hi")
	app.processIncomingMessage(account, accountJID, incoming, "Sender", false, false, "", "")
	// Redelivery of the same webhook must not create a second row.
	app.processIncomingMessage(account, accountJID, incoming, "Sender", false, false, "", "")

	var count int64
	app.DB.Model(&models.Message{}).
		Where("whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
			wamid, org.ID, account.Name).
		Count(&count)
	assert.EqualValues(t, 1, count)
}

func TestProcessGowaOutgoingMessage_CrossAccountSameWamidKeepsBothCopies(t *testing.T) {
	app := newOutgoingLifecycleApp(t)
	org, receiver := createProcessorTestOrg(t, app)
	sender := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID,
		testutil.WithAccountName("sender-"+uuid.New().String()[:8]))

	// The receiving account already stored its incoming copy under this wamid.
	wamid := "XWAMID_" + uuid.New().String()[:12]
	receiverContact := testutil.CreateTestContact(t, app.DB, org.ID)
	incomingCopy := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    org.ID,
		WhatsAppAccount:   receiver.Name,
		ContactID:         receiverContact.ID,
		WhatsAppMessageID: wamid,
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeText,
		Content:           "cross-account test",
		Status:            models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(incomingCopy).Error)

	// The sender's device now echoes the same wamid as an is_from_me message.
	recipientPhone := uniquePhone()
	msg := gowa.MessagePayload{
		ID:        wamid,
		ChatID:    recipientPhone + "@s.whatsapp.net",
		From:      recipientPhone + "@s.whatsapp.net",
		Timestamp: time.Now().Format(time.RFC3339),
		IsFromMe:  true,
		Body:      "sent from the phone",
	}
	app.processGowaOutgoingMessage(sender, &msg, "device-cross")

	var stored models.Message
	require.NoError(t, app.DB.Where(
		"whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
		wamid, org.ID, sender.Name).First(&stored).Error,
		"the sender account's outgoing echo must be saved despite the receiver's copy sharing the wamid")
	assert.Equal(t, models.DirectionOutgoing, stored.Direction)
}

// TestCrossAccountMirrorContactIDs locks the mirror-view resolution used by
// GetMessages: selecting the tab of the account whose own number IS the page
// contact must resolve to the counterpart contacts of the other org accounts
// (where that account's copies of the thread actually live).
func TestCrossAccountMirrorContactIDs(t *testing.T) {
	app := newProcessorTestApp(t)
	org, saudi := createProcessorTestOrg(t, app)
	egypt := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID,
		testutil.WithAccountName("egypt-"+uuid.New().String()[:8]))

	saudiPhone := uniquePhone()
	egyptPhone := uniquePhone()
	require.NoError(t, app.DB.Model(saudi).Update("gowa_jid", saudiPhone+"@s.whatsapp.net").Error)
	require.NoError(t, app.DB.Model(egypt).Update("gowa_jid", egyptPhone+"@s.whatsapp.net").Error)

	saudiContact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber(saudiPhone))
	egyptContact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber(egyptPhone))

	// Page = saudi number, tab = saudi account → mirror to the egypt contact.
	ids := app.crossAccountMirrorContactIDs(org.ID, saudi.Name, saudiContact)
	require.Len(t, ids, 1)
	assert.Equal(t, egyptContact.ID, ids[0])

	// Page = egypt number, tab = egypt account → mirror to the saudi contact.
	ids = app.crossAccountMirrorContactIDs(org.ID, egypt.Name, egyptContact)
	require.Len(t, ids, 1)
	assert.Equal(t, saudiContact.ID, ids[0])

	// Normal tab (account's number differs from the page contact) → no mirror.
	assert.Nil(t, app.crossAccountMirrorContactIDs(org.ID, saudi.Name, egyptContact))

	// Unknown account or a regular customer contact → no mirror.
	customer := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber(uniquePhone()))
	assert.Nil(t, app.crossAccountMirrorContactIDs(org.ID, saudi.Name, customer))
	assert.Nil(t, app.crossAccountMirrorContactIDs(org.ID, "no-such-account", saudiContact))
}

func TestProcessGowaOutgoingMessage_SameAccountEchoStillDeduped(t *testing.T) {
	app := newOutgoingLifecycleApp(t)
	org, account := createProcessorTestOrg(t, app)

	// A UI send already created the local row with the GOWA-returned wamid.
	wamid := "XWAMID_" + uuid.New().String()[:12]
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	createSenderCopy(t, app, org.ID, account.Name, wamid, contact.ID)

	recipientPhone := uniquePhone()
	msg := gowa.MessagePayload{
		ID:        wamid,
		ChatID:    recipientPhone + "@s.whatsapp.net",
		From:      recipientPhone + "@s.whatsapp.net",
		Timestamp: time.Now().Format(time.RFC3339),
		IsFromMe:  true,
		Body:      "echo of a UI send",
	}
	app.processGowaOutgoingMessage(account, &msg, "device-echo")

	var count int64
	app.DB.Model(&models.Message{}).
		Where("whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
			wamid, org.ID, account.Name).
		Count(&count)
	assert.EqualValues(t, 1, count, "the echo of a UI-sent message must still dedup within the same account")
}
