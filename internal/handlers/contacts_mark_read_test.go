package handlers_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/handlers"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// newMarkReadRequest builds a fastglue POST request for MarkContactRead tests.
func newMarkReadRequest(t *testing.T) *fastglue.Request {
	t.Helper()
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("POST")
	return &fastglue.Request{RequestCtx: ctx}
}

// createUnreadIncomingMessage inserts an incoming, not-yet-read message for
// the contact so the mark-read effect is observable.
func createUnreadIncomingMessage(t *testing.T, app *handlers.App, orgID, contactID uuid.UUID) *models.Message {
	t.Helper()
	msg := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		ContactID:      contactID,
		Direction:      models.DirectionIncoming,
		MessageType:    models.MessageTypeText,
		Content:        "hello",
		Status:         models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msg).Error)
	return msg
}

// TestApp_MarkContactRead_PendingUnclaimed_Forbidden locks the claim-gate
// guard: an agent who cannot view a pending unclaimed chat's content must not
// be able to mark it read either — that would clear the unread badge and send
// read receipts for messages nobody has seen.
func TestApp_MarkContactRead_PendingUnclaimed_Forbidden(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	// Mirrors the real "agent" role: contacts:read but neither contacts:write
	// nor chat.collaborate:write.
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "read-agent",
		[]string{"contacts:read", "chat:read", "chat.assign:write"})
	agent := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	contact.SetStatus(models.ChatStatusPending)
	require.NoError(t, app.DB.Model(contact).Update("metadata", contact.Metadata).Error)
	msg := createUnreadIncomingMessage(t, app, org.ID, contact.ID)

	req := newMarkReadRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	_ = app.MarkContactRead(req)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "Claim this chat")

	// The message must remain unread.
	var updated models.Message
	require.NoError(t, app.DB.First(&updated, "id = ?", msg.ID).Error)
	assert.Equal(t, models.MessageStatusDelivered, updated.Status,
		"message must stay unread behind the claim gate")
}

// TestApp_MarkContactRead_Assignee_Success verifies the happy path is intact:
// the assigned agent of an open chat can mark it read.
func TestApp_MarkContactRead_Assignee_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "read-agent",
		[]string{"contacts:read", "chat:read", "chat.assign:write"})
	agent := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	contact.AssignedUserID = &agent.ID
	contact.SetStatus(models.ChatStatusOpen)
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"assigned_user_id": agent.ID,
		"metadata":         contact.Metadata,
	}).Error)
	msg := createUnreadIncomingMessage(t, app, org.ID, contact.ID)

	req := newMarkReadRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.MarkContactRead(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var updated models.Message
	require.NoError(t, app.DB.First(&updated, "id = ?", msg.ID).Error)
	assert.Equal(t, models.MessageStatusRead, updated.Status)
}
