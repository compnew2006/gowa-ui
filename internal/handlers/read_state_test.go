package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// The unread badge is per user: one agent opening a conversation used to
// mark its messages read for EVERY user, so the count vanished for agents who
// never saw them. Now each user's own read position drives their badge.
func TestUnreadCount_IsPerUser(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	userA := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	userB := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	createUnreadIncomingMessage(t, app, org.ID, contact.ID)
	createUnreadIncomingMessage(t, app, org.ID, contact.ID)

	unreadFor := func(userID uuid.UUID) int {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, userID)
		require.NoError(t, app.ListContacts(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
		var resp struct {
			Data struct {
				Contacts []handlers.ContactResponse `json:"contacts"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		for _, c := range resp.Data.Contacts {
			if c.ID == contact.ID {
				return c.UnreadCount
			}
		}
		t.Fatalf("contact missing from list")
		return -1
	}
	markRead := func(userID uuid.UUID) {
		req := newMarkReadRequest(t)
		testutil.SetAuthContext(req, org.ID, userID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		require.NoError(t, app.MarkContactRead(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	}

	require.Equal(t, 2, unreadFor(userA.ID))
	require.Equal(t, 2, unreadFor(userB.ID))

	// A opens the chat: A's badge clears, B keeps it.
	markRead(userA.ID)
	assert.Equal(t, 0, unreadFor(userA.ID))
	assert.Equal(t, 2, unreadFor(userB.ID), "another user reading must not clear B's badge")

	// The WhatsApp-side read state (blue ticks) is still conversation-wide.
	var stillUnread int64
	app.DB.Model(&models.Message{}).
		Where("contact_id = ? AND direction = ? AND status <> ?", contact.ID, models.DirectionIncoming, models.MessageStatusRead).
		Count(&stillUnread)
	assert.Zero(t, stillUnread, "messages are marked read for the customer on the first open")

	// A new message counts for both.
	createUnreadIncomingMessage(t, app, org.ID, contact.ID)
	assert.Equal(t, 1, unreadFor(userA.ID))
	assert.Equal(t, 3, unreadFor(userB.ID))

	// A message delivered late, stamped with an older time than messages A
	// already read (history sync / delayed webhook), is still new to A.
	markRead(userA.ID)
	late := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New(), CreatedAt: time.Now().Add(-48 * time.Hour)},
		OrganizationID: org.ID,
		ContactID:      contact.ID,
		Direction:      models.DirectionIncoming,
		MessageType:    models.MessageTypeText,
		Content:        "late",
		Status:         models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(late).Error)
	assert.Equal(t, 1, unreadFor(userA.ID), "a late-inserted message must not hide behind A's read position")

	// B opening the chat only clears B.
	markRead(userB.ID)
	assert.Equal(t, 0, unreadFor(userB.ID))
	assert.Equal(t, 1, unreadFor(userA.ID))
}

// Revoked and failed messages never count, whatever the read position —
// the read sweep skips them, so counting them would pin the badge forever.
func TestUnreadCount_IgnoresRevokedAndFailed(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	for _, status := range []models.MessageStatus{models.MessageStatusRevoked, models.MessageStatusFailed, models.MessageStatusDelivered} {
		msg := createUnreadIncomingMessage(t, app, org.ID, contact.ID)
		require.NoError(t, app.DB.Model(msg).Update("status", status).Error)
	}

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	require.NoError(t, app.GetContact(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	var resp struct {
		Data handlers.ContactResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, 1, resp.Data.UnreadCount)
}
