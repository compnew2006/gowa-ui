package handlers_test

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func createChatSoftDeleteUser(t *testing.T, app *handlers.App, orgID uuid.UUID, roleName string) *models.User {
	t.Helper()

	return createUserWithPermissionKeys(t, app, orgID, roleName, []string{
		"contacts:read",
		"contacts:soft_delete",
	})
}

func createExactAdminUser(t *testing.T, app *handlers.App, orgID uuid.UUID, fullName string) *models.User {
	t.Helper()

	allPerms := testutil.GetOrCreateTestPermissions(t, app.DB)
	role := testutil.CreateTestRoleExact(t, app.DB, orgID, "admin", false, false, allPerms)
	return testutil.CreateTestUser(t, app.DB, orgID, testutil.WithRoleID(&role.ID), testutil.WithFullName(fullName))
}

func createChatMessage(
	t *testing.T,
	app *handlers.App,
	orgID uuid.UUID,
	contact *models.Contact,
	createdAt time.Time,
	content string,
) *models.Message {
	t.Helper()

	message := &models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: createdAt, UpdatedAt: createdAt},
		OrganizationID:  orgID,
		InstanceID:      contact.InstanceID,
		WhatsAppAccount: contact.WhatsAppAccount,
		ContactID:       contact.ID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         content,
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(message).Error)
	return message
}

func TestApp_SoftDeleteContactForUser_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader-only", []string{"contacts:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.SoftDeleteContactForUser(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to hide chats")
}

func TestApp_SoftDeleteContactForUser_CreatesDeletionAndAdminNotification(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	actor := createChatSoftDeleteUser(t, app, org.ID, "chat-soft-delete-user")
	admin := createExactAdminUser(t, app, org.ID, "Admin User")
	instance := createTestInstance(t, app, org.ID, "Soft Delete Support")
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(account.Name),
		func(c *models.Contact) {
			c.InstanceID = &instance.ID
			c.AssignedUserID = &actor.ID
			c.Status = models.ChatStatusOpen
		},
	)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, actor.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.SoftDeleteContactForUser(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var deletion models.ContactUserDeletion
	require.NoError(t, app.DB.Where("organization_id = ? AND contact_id = ? AND user_id = ?", org.ID, contact.ID, actor.ID).First(&deletion).Error)
	assert.Equal(t, org.ID, deletion.OrganizationID)
	assert.Equal(t, contact.ID, deletion.ContactID)
	assert.Equal(t, actor.ID, deletion.UserID)

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	assert.Equal(t, models.ChatStatusClosed, refreshed.Status)
	assert.Nil(t, refreshed.AssignedUserID)
	require.NotNil(t, refreshed.ClosedByUserID)
	assert.Equal(t, actor.ID, *refreshed.ClosedByUserID)

	var notification models.InstanceNotification
	require.NoError(t, app.DB.Where("organization_id = ? AND contact_id = ? AND event_type = ?", org.ID, contact.ID, "chat_deleted_by_user").First(&notification).Error)
	assert.Equal(t, instance.ID, notification.InstanceID)
	require.NotNil(t, notification.ContactID)
	assert.Equal(t, contact.ID, *notification.ContactID)
	assert.Equal(t, actor.ID.String(), notification.Metadata["actor_id"])
	assert.Equal(t, contact.ID.String(), notification.Metadata["contact_id"])

	nonAdminReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(nonAdminReq, org.ID, actor.ID)
	err = app.ListNotifications(nonAdminReq)
	require.NoError(t, err)

	var nonAdminNotifications []models.InstanceNotification
	testutil.ParseEnvelopeResponse(t, nonAdminReq, &nonAdminNotifications)
	assert.Empty(t, nonAdminNotifications)

	adminReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(adminReq, org.ID, admin.ID)
	err = app.ListNotifications(adminReq)
	require.NoError(t, err)

	var adminNotifications []models.InstanceNotification
	testutil.ParseEnvelopeResponse(t, adminReq, &adminNotifications)
	require.Len(t, adminNotifications, 1)
	assert.Equal(t, "chat_deleted_by_user", adminNotifications[0].EventType)
	require.NotNil(t, adminNotifications[0].ContactID)
	assert.Equal(t, contact.ID, *adminNotifications[0].ContactID)
}

func TestApp_ListContacts_HidesSoftDeletedContactUntilNewActivity(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createChatSoftDeleteUser(t, app, org.ID, "chat-soft-delete-list")
	instance := createTestInstance(t, app, org.ID, "List Soft Delete")
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	closedAt := time.Now().UTC().Add(-3 * time.Hour)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(account.Name),
		func(c *models.Contact) {
			c.InstanceID = &instance.ID
			c.Status = models.ChatStatusClosed
			c.ClosedAt = &closedAt
		},
	)

	preDeleteMessageAt := time.Now().UTC().Add(-2 * time.Hour)
	createChatMessage(t, app, org.ID, contact, preDeleteMessageAt, "before hide")
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"last_message_at":      preDeleteMessageAt,
		"last_message_preview": "before hide",
	}).Error)

	softDeleteReq := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(softDeleteReq, org.ID, user.ID)
	testutil.SetPathParam(softDeleteReq, "id", contact.ID.String())
	require.NoError(t, app.SoftDeleteContactForUser(softDeleteReq))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(softDeleteReq))

	listReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(listReq, org.ID, user.ID)
	testutil.SetQueryParam(listReq, "status", "closed")
	require.NoError(t, app.ListContacts(listReq))

	var hidden struct {
		Contacts []handlers.ContactResponse `json:"contacts"`
		Total    int                        `json:"total"`
	}
	testutil.ParseEnvelopeResponse(t, listReq, &hidden)
	assert.Empty(t, hidden.Contacts)
	assert.Zero(t, hidden.Total)

	var deletion models.ContactUserDeletion
	require.NoError(t, app.DB.Where("organization_id = ? AND contact_id = ? AND user_id = ?", org.ID, contact.ID, user.ID).First(&deletion).Error)
	postDeleteMessageAt := deletion.DeletedAt.Add(2 * time.Minute)
	postDeleteMessage := createChatMessage(t, app, org.ID, contact, postDeleteMessageAt, "after hide")
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"last_message_at":      postDeleteMessageAt,
		"last_message_preview": postDeleteMessage.Content,
	}).Error)

	visibleReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(visibleReq, org.ID, user.ID)
	testutil.SetQueryParam(visibleReq, "status", "closed")
	require.NoError(t, app.ListContacts(visibleReq))

	var visible struct {
		Contacts []handlers.ContactResponse `json:"contacts"`
		Total    int                        `json:"total"`
	}
	testutil.ParseEnvelopeResponse(t, visibleReq, &visible)
	require.Len(t, visible.Contacts, 1)
	assert.Equal(t, contact.ID, visible.Contacts[0].ID)
	assert.Equal(t, 1, visible.Contacts[0].UnreadCount)
	assert.Equal(t, 1, visible.Total)
}

func TestApp_GetMessages_ExcludesSoftDeletedHistory(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createChatSoftDeleteUser(t, app, org.ID, "chat-soft-delete-messages")
	instance := createTestInstance(t, app, org.ID, "Messages Soft Delete")
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	closedAt := time.Now().UTC().Add(-2 * time.Hour)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(account.Name),
		func(c *models.Contact) {
			c.InstanceID = &instance.ID
			c.Status = models.ChatStatusClosed
			c.ClosedAt = &closedAt
		},
	)

	beforeMessage := createChatMessage(t, app, org.ID, contact, time.Now().UTC().Add(-90*time.Minute), "before soft delete")

	softDeleteReq := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(softDeleteReq, org.ID, user.ID)
	testutil.SetPathParam(softDeleteReq, "id", contact.ID.String())
	require.NoError(t, app.SoftDeleteContactForUser(softDeleteReq))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(softDeleteReq))

	var deletion models.ContactUserDeletion
	require.NoError(t, app.DB.Where("organization_id = ? AND contact_id = ? AND user_id = ?", org.ID, contact.ID, user.ID).First(&deletion).Error)

	afterMessage := createChatMessage(t, app, org.ID, contact, deletion.DeletedAt.Add(time.Minute), "after soft delete")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.GetMessages(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response struct {
		Messages []handlers.MessageResponse `json:"messages"`
		Total    int                        `json:"total"`
	}
	testutil.ParseEnvelopeResponse(t, req, &response)
	require.Len(t, response.Messages, 1)
	assert.Equal(t, afterMessage.ID, response.Messages[0].ID)
	assert.NotEqual(t, beforeMessage.ID, response.Messages[0].ID)
	assert.Equal(t, 1, response.Total)
}
