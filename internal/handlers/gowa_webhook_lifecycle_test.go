package handlers

import (
	"fmt"

	"github.com/google/uuid"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/chatlifecycle"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newOutgoingLifecycleApp builds a processor test app with the chat lifecycle
// service wired in (needed for the reopen system message).
func newOutgoingLifecycleApp(t *testing.T) *App {
	t.Helper()
	app := newProcessorTestApp(t)
	app.ChatLifecycle = chatlifecycle.New(app.DB, nil, testutil.NopLogger())
	return app
}

// uniquePhone returns a phone number unlikely to collide across tests sharing
// the test database.
func uniquePhone() string {
	return fmt.Sprintf("96650%010d", time.Now().UnixNano()%1e10)
}

// phoneSentPayload builds a text message sent from the connected phone
// (is_from_me=true) to the given recipient phone.
func phoneSentPayload(recipientPhone string) gowa.MessagePayload {
	return gowa.MessagePayload{
		ID:        "PHONE_MSG_" + recipientPhone,
		ChatID:    recipientPhone + "@s.whatsapp.net",
		From:      recipientPhone + "@s.whatsapp.net",
		Timestamp: time.Now().Format(time.RFC3339),
		IsFromMe:  true,
		Body:      "sent from the phone",
	}
}

// TestProcessGowaOutgoingMessage_NewContactBecomesPending locks the claim-gate
// fix: a conversation that exists only because the connected number's phone
// sent messages must be claimable (pending), exactly like a received message —
// not silently readable by every agent.
func TestProcessGowaOutgoingMessage_NewContactBecomesPending(t *testing.T) {
	app := newOutgoingLifecycleApp(t)
	org, account := createProcessorTestOrg(t, app)

	phone := uniquePhone()
	msg := phoneSentPayload(phone)
	app.processGowaOutgoingMessage(account, &msg, "device-1")

	var contact models.Contact
	require.NoError(t, app.DB.
		Where("organization_id = ? AND phone_number = ?", org.ID, phone).
		First(&contact).Error)
	// The claim gate keys off EffectiveStatus: a fresh unassigned contact is
	// pending by inference (defaultStatus), so no explicit key write is
	// needed — same as the incoming path.
	assert.Equal(t, models.ChatStatusPending, contact.EffectiveStatus())

	var stored models.Message
	require.NoError(t, app.DB.
		Where("contact_id = ? AND whats_app_message_id = ?", contact.ID, msg.ID).
		First(&stored).Error)
	assert.Equal(t, models.DirectionOutgoing, stored.Direction)
}

// TestProcessGowaOutgoingMessage_StaleOpenUnassignedBecomesPending: an
// unassigned contact carrying a stale explicit 'open' status (e.g. legacy
// data) must be normalized back to pending when the phone sends a message —
// otherwise it stays readable without a claim.
func TestProcessGowaOutgoingMessage_StaleOpenUnassignedBecomesPending(t *testing.T) {
	app := newOutgoingLifecycleApp(t)
	org, account := createProcessorTestOrg(t, app)

	phone := uniquePhone()
	contact := &models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    phone,
	}
	contact.SetStatus(models.ChatStatusOpen)
	require.NoError(t, app.DB.Create(contact).Error)

	msg := phoneSentPayload(phone)
	app.processGowaOutgoingMessage(account, &msg, "device-1")

	var updated models.Contact
	require.NoError(t, app.DB.First(&updated, "id = ?", contact.ID).Error)
	assert.Equal(t, models.ChatStatusPending, updated.EffectiveStatus())
	assert.Equal(t, "pending", updated.Metadata["chat_status"],
		"the stale open key must be overwritten")
}

// TestProcessGowaOutgoingMessage_ClosedChatReopensPending mirrors the incoming
// path: new activity from the phone on a closed conversation reopens it as
// pending with a system message.
func TestProcessGowaOutgoingMessage_ClosedChatReopensPending(t *testing.T) {
	app := newOutgoingLifecycleApp(t)
	org, account := createProcessorTestOrg(t, app)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)

	phone := uniquePhone()
	contact := &models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    phone,
		AssignedUserID: &agent.ID,
	}
	contact.SetStatus(models.ChatStatusClosed)
	require.NoError(t, app.DB.Create(contact).Error)

	msg := phoneSentPayload(phone)
	app.processGowaOutgoingMessage(account, &msg, "device-1")

	var updated models.Contact
	require.NoError(t, app.DB.First(&updated, "id = ?", contact.ID).Error)
	assert.Nil(t, updated.AssignedUserID, "reopen must clear the stale assignment")
	assert.Equal(t, models.ChatStatusPending, updated.EffectiveStatus())

	var sysCount int64
	app.DB.Model(&models.Message{}).
		Where("contact_id = ? AND metadata->>'system_type' = ?", contact.ID, "chat_reopened").
		Count(&sysCount)
	assert.EqualValues(t, 1, sysCount, "reopen must leave a system message")
}

// TestProcessGowaOutgoingMessage_AssignedOpenUntouched: a phone-sent message
// on an assigned open conversation must not disturb ownership or status.
func TestProcessGowaOutgoingMessage_AssignedOpenUntouched(t *testing.T) {
	app := newOutgoingLifecycleApp(t)
	org, account := createProcessorTestOrg(t, app)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)

	phone := uniquePhone()
	contact := &models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    phone,
		AssignedUserID: &agent.ID,
	}
	contact.SetStatus(models.ChatStatusOpen)
	require.NoError(t, app.DB.Create(contact).Error)

	msg := phoneSentPayload(phone)
	app.processGowaOutgoingMessage(account, &msg, "device-1")

	var updated models.Contact
	require.NoError(t, app.DB.First(&updated, "id = ?", contact.ID).Error)
	require.NotNil(t, updated.AssignedUserID)
	assert.Equal(t, agent.ID, *updated.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, updated.EffectiveStatus())
}
