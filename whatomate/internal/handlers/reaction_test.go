package handlers_test

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendReaction_InvalidMessageID_Returns400(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]string{"emoji": "👍"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetPathParam(req, "message_id", "not-a-uuid")

	err := app.SendReaction(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, 400, "Invalid message")
}

func TestSendReaction_MissingMessageID_Returns400(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]string{"emoji": "👍"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.SendReaction(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, 400, "Invalid message")
}

func TestSendReaction_InvalidContactID_Returns400(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	msgID := uuid.New()
	req := testutil.NewJSONRequest(t, map[string]string{"emoji": "👍"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", "not-a-uuid")
	testutil.SetPathParam(req, "message_id", msgID.String())

	err := app.SendReaction(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, 400, "Invalid contact")
}

func TestSendReaction_MessageNotFound_Returns404(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	randomMsgID := uuid.New()
	req := testutil.NewJSONRequest(t, map[string]string{"emoji": "👍"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetPathParam(req, "message_id", randomMsgID.String())

	err := app.SendReaction(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, 404, "Message not found")
}

func TestSendReaction_ContactNotFound_Returns404(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	msg := createReactionTestMessage(t, app, org.ID, contact.ID, account.Name)

	randomContactID := uuid.New()
	req := testutil.NewJSONRequest(t, map[string]string{"emoji": "👍"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", randomContactID.String())
	testutil.SetPathParam(req, "message_id", msg.ID.String())

	err := app.SendReaction(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, 404, "Contact not found")
}

func TestSendReaction_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	msg := createReactionTestMessage(t, app, org.ID, contact.ID, account.Name)

	req := testutil.NewJSONRequest(t, map[string]string{"emoji": "👍"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetPathParam(req, "message_id", msg.ID.String())

	err := app.SendReaction(req)
	require.NoError(t, err)
	assert.Equal(t, 200, testutil.GetResponseStatusCode(req))

	var resp map[string]interface{}
	testutil.ParseEnvelopeResponse(t, req, &resp)
	assert.Equal(t, msg.ID.String(), resp["message_id"])

	reactions, ok := resp["reactions"].([]interface{})
	require.True(t, ok, "expected reactions array")
	require.Len(t, reactions, 1, "expected one reaction")

	r := reactions[0].(map[string]interface{})
	assert.Equal(t, "👍", r["emoji"])
}

func TestSendReaction_RemoveReaction(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	msg := createReactionTestMessage(t, app, org.ID, contact.ID, account.Name)

	req := testutil.NewJSONRequest(t, map[string]string{"emoji": ""})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetPathParam(req, "message_id", msg.ID.String())

	err := app.SendReaction(req)
	require.NoError(t, err)
	assert.Equal(t, 200, testutil.GetResponseStatusCode(req))

	var resp map[string]interface{}
	testutil.ParseEnvelopeResponse(t, req, &resp)
	reactions, ok := resp["reactions"].([]interface{})
	require.True(t, ok, "expected reactions array")
	assert.Empty(t, reactions, "expected empty reactions when emoji is empty")
}

func createReactionTestMessage(t *testing.T, app *handlers.App, orgID, contactID uuid.UUID, accountName string) *models.Message {
	t.Helper()
	msg := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    orgID,
		ContactID:         contactID,
		WhatsAppAccount:   accountName,
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeText,
		Content:           "test message",
		Status:            models.MessageStatusDelivered,
		WhatsAppMessageID: "wamid.test-" + uuid.New().String()[:8],
	}
	require.NoError(t, app.DB.Create(msg).Error)
	return msg
}
