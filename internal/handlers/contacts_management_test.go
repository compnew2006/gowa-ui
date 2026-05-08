package handlers_test

import (
	"encoding/json"
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

func createContactMgmtUser(t *testing.T, app *handlers.App, orgID uuid.UUID, roleName string, permissionKeys []string) *models.User {
	t.Helper()
	return createUserWithPermissionKeys(t, app, orgID, roleName, permissionKeys)
}

func createContactMgmtAdmin(t *testing.T, app *handlers.App, orgID uuid.UUID) *models.User {
	t.Helper()
	return createUserWithPermissionKeys(t, app, orgID, "contact-admin", []string{
		"contacts:read",
		"contacts:write",
		"contacts:delete",
		"contacts:soft_delete",
		"chat_assign:write",
		"chat:read",
		"chat:write",
	})
}

func createAssignOnlyUser(t *testing.T, app *handlers.App, orgID uuid.UUID) *models.User {
	t.Helper()
	return createUserWithPermissionKeys(t, app, orgID, "assign-only", []string{
		"contacts:read",
		"chat_assign:write",
	})
}

func createReadonlyUser(t *testing.T, app *handlers.App, orgID uuid.UUID) *models.User {
	t.Helper()
	return createUserWithPermissionKeys(t, app, orgID, "readonly", []string{
		"contacts:read",
	})
}

// --- AssignContact Tests ---

func TestApp_AssignContact_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createReadonlyUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": uuid.New().String(),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to assign contacts")
}

func TestApp_AssignContact_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": uuid.New().String(),
	})

	err := app.AssignContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_AssignContact_InvalidContactID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": uuid.New().String(),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", "not-a-uuid")

	err := app.AssignContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_AssignContact_ContactNotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": uuid.New().String(),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestApp_AssignContact_AssignToUser(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	actor := createContactMgmtAdmin(t, app, org.ID)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": assignee.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, actor.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	require.NotNil(t, refreshed.AssignedUserID)
	assert.Equal(t, assignee.ID, *refreshed.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, refreshed.Status)
}

func TestApp_AssignContact_AssignToNonExistentUser(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	actor := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": uuid.New().String(),
	})
	testutil.SetAuthContext(req, org.ID, actor.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "User not found")
}

func TestApp_AssignContact_UnassignContact(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	actor := createContactMgmtAdmin(t, app, org.ID)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &assignee.ID
		c.Status = models.ChatStatusOpen
	})

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": nil,
	})
	testutil.SetAuthContext(req, org.ID, actor.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	assert.Nil(t, refreshed.AssignedUserID)
	assert.Equal(t, models.ChatStatusPending, refreshed.Status)
}

func TestApp_AssignContact_WithChatAssignPermissionOnly(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	actor := createAssignOnlyUser(t, app, org.ID)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": assignee.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, actor.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
}

// --- ClaimChat Tests ---

func TestApp_ClaimChat_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createReadonlyUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ClaimChat(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to claim chats")
}

func TestApp_ClaimChat_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, nil)

	err := app.ClaimChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_ClaimChat_ChatNotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.ClaimChat(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "Chat not found")
}

func TestApp_ClaimChat_AlreadyAssignedToAnotherUser(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	claimer := createContactMgmtAdmin(t, app, org.ID)
	otherUser := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &otherUser.ID
		c.Status = models.ChatStatusOpen
	})

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, claimer.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ClaimChat(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusConflict, "already assigned to another user")
}

func TestApp_ClaimChat_ClosedChatCannotBeClaimed(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.Status = models.ChatStatusClosed
		c.ClosedAt = ptrTime(time.Now().UTC().Add(-1 * time.Hour))
	})

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ClaimChat(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusConflict, "Closed chats cannot be claimed")
}

func TestApp_ClaimChat_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.Status = models.ChatStatusPending
	})

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ClaimChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	require.NotNil(t, refreshed.AssignedUserID)
	assert.Equal(t, user.ID, *refreshed.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, refreshed.Status)
}

func TestApp_ClaimChat_AlreadyClaimedBySelf(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &user.ID
		c.Status = models.ChatStatusOpen
	})

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ClaimChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp handlers.ContactResponse
	testutil.ParseEnvelopeResponse(t, req, &resp)
	assert.Equal(t, contact.ID, resp.ID)
}

// --- CloseChat Tests ---

func TestApp_CloseChat_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createReadonlyUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.CloseChat(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to close chats")
}

func TestApp_CloseChat_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, nil)

	err := app.CloseChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_CloseChat_ChatNotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.CloseChat(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "Chat not found")
}

func TestApp_CloseChat_AlreadyClosed(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.Status = models.ChatStatusClosed
		c.ClosedAt = ptrTime(time.Now().UTC().Add(-1 * time.Hour))
	})

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.CloseChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	assert.Equal(t, models.ChatStatusClosed, refreshed.Status)
}

func TestApp_CloseChat_OnlyAssignedUserCanClose(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	closer := createContactMgmtAdmin(t, app, org.ID)
	otherUser := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &otherUser.ID
		c.Status = models.ChatStatusOpen
	})

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, closer.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.CloseChat(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "Only the assigned user can close")
}

func TestApp_CloseChat_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &user.ID
		c.Status = models.ChatStatusOpen
	})

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.CloseChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	assert.Equal(t, models.ChatStatusClosed, refreshed.Status)
	require.NotNil(t, refreshed.ClosedAt)
	require.NotNil(t, refreshed.ClosedByUserID)
	assert.Equal(t, user.ID, *refreshed.ClosedByUserID)
}

// --- ReopenChat Tests ---

func TestApp_ReopenChat_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createReadonlyUser(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.Status = models.ChatStatusClosed
		c.ClosedAt = ptrTime(time.Now().UTC().Add(-1 * time.Hour))
	})

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ReopenChat(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to reopen chats")
}

func TestApp_ReopenChat_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, nil)

	err := app.ReopenChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_ReopenChat_ChatNotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.ReopenChat(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "Chat not found")
}

func TestApp_ReopenChat_OnlyClosedChatsCanBeReopened(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.Status = models.ChatStatusOpen
	})

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ReopenChat(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusConflict, "Only closed chats can be reopened")
}

func TestApp_ReopenChat_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	closedAt := time.Now().UTC().Add(-1 * time.Hour)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.Status = models.ChatStatusClosed
		c.ClosedAt = &closedAt
		c.ClosedByUserID = &user.ID
	})

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ReopenChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	assert.Equal(t, models.ChatStatusPending, refreshed.Status)
	assert.Nil(t, refreshed.AssignedUserID)
	assert.Nil(t, refreshed.ClosedAt)
	assert.Nil(t, refreshed.ClosedByUserID)
}

// --- SetChatPublic Tests ---

func TestApp_SetChatPublic_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createReadonlyUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{"is_public": true})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.SetChatPublic(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to change chat visibility")
}

func TestApp_SetChatPublic_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]any{"is_public": true})

	err := app.SetChatPublic(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_SetChatPublic_ChatNotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{"is_public": true})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.SetChatPublic(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "Chat not found")
}

func TestApp_SetChatPublic_MakePublic(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.IsPublic = false
	})

	req := testutil.NewJSONRequest(t, map[string]any{"is_public": true})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.SetChatPublic(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	assert.True(t, refreshed.IsPublic)
}

func TestApp_SetChatPublic_MakePrivate(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.IsPublic = true
	})

	req := testutil.NewJSONRequest(t, map[string]any{"is_public": false})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.SetChatPublic(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	assert.False(t, refreshed.IsPublic)
}

func TestApp_SetChatPublic_NoChange(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.IsPublic = true
	})

	req := testutil.NewJSONRequest(t, map[string]any{"is_public": true})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.SetChatPublic(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp handlers.ContactResponse
	testutil.ParseEnvelopeResponse(t, req, &resp)
	assert.True(t, resp.IsPublic)
}

// --- UpdateContactTags Tests ---

func TestApp_UpdateContactTags_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createReadonlyUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"tags": []string{"vip"},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.UpdateContactTags(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to update contact tags")
}

func TestApp_UpdateContactTags_ContactNotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"tags": []string{"vip"},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.UpdateContactTags(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestApp_UpdateContactTags_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"tags": []string{"vip", "customer"},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.UpdateContactTags(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	require.NotNil(t, refreshed.Tags)
	assert.Len(t, refreshed.Tags, 2)
}

func TestApp_UpdateContactTags_ClearTags(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.Tags = models.JSONBArray{"old-tag"}
	})

	req := testutil.NewJSONRequest(t, map[string]any{
		"tags": []string{},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.UpdateContactTags(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	require.Nil(t, refreshed.Tags)
}

// --- CreateContact Tests ---

func TestApp_CreateContact_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createReadonlyUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"phone_number": "+1234567890",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to create contacts")
}

func TestApp_CreateContact_MissingPhoneNumber(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"profile_name": "Test",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "phone_number is required")
}

func TestApp_CreateContact_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	phone := "+1234567" + uuid.New().String()[:4]
	req := testutil.NewJSONRequest(t, map[string]any{
		"phone_number": phone,
		"profile_name": "New Contact",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp handlers.ContactResponse
	testutil.ParseEnvelopeResponse(t, req, &resp)
	assert.Equal(t, "New Contact", resp.ProfileName)
	assert.Equal(t, models.ChatStatusPending, resp.Status)
}

func TestApp_CreateContact_WithTags(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	phone := "+1234567" + uuid.New().String()[:4]
	req := testutil.NewJSONRequest(t, map[string]any{
		"phone_number": phone,
		"tags":         []string{"lead", "priority"},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp handlers.ContactResponse
	testutil.ParseEnvelopeResponse(t, req, &resp)
	assert.ElementsMatch(t, []string{"lead", "priority"}, resp.Tags)
}

func TestApp_CreateContact_DuplicatePhone(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	phone := "+1234567" + uuid.New().String()[:4]
	testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithPhoneNumber(phone))

	req := testutil.NewJSONRequest(t, map[string]any{
		"phone_number": phone,
		"profile_name": "Duplicate",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusConflict, "already exists")
}

// --- UpdateContact Tests ---

func TestApp_UpdateContact_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createReadonlyUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"profile_name": "Updated",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.UpdateContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to update contacts")
}

func TestApp_UpdateContact_ContactNotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"profile_name": "Updated",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.UpdateContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestApp_UpdateContact_NoFieldsToUpdate(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.UpdateContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "No fields to update")
}

func TestApp_UpdateContact_UpdateProfileName(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"profile_name": "Updated Name",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.UpdateContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	assert.Equal(t, "Updated Name", refreshed.ProfileName)
}

func TestApp_UpdateContact_UpdateTags(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"tags": []string{"new-tag-1", "new-tag-2"},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.UpdateContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	require.NotNil(t, refreshed.Tags)
	assert.Len(t, refreshed.Tags, 2)
}

// --- DeleteContact Tests ---

func TestApp_DeleteContact_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.DeleteContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to delete chats")
}

func TestApp_DeleteContact_ContactNotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtUser(t, app, org.ID, "deleter", []string{
		"contacts:read",
		"contacts:write",
		"contacts:delete",
	})

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.DeleteContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestApp_DeleteContact_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtUser(t, app, org.ID, "deleter", []string{
		"contacts:read",
		"contacts:write",
		"contacts:delete",
	})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.DeleteContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var found models.Contact
	err = app.DB.Unscoped().Where("id = ?", contact.ID).First(&found).Error
	require.NoError(t, err)
	assert.NotNil(t, found.DeletedAt)
}

func TestApp_DeleteContact_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, nil)

	err := app.DeleteContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

// --- GetContactSessionData Tests ---

func TestApp_GetContactSessionData_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewGETRequest(t)

	err := app.GetContactSessionData(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_GetContactSessionData_ContactNotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.GetContactSessionData(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "Contact not found")
}

func TestApp_GetContactSessionData_NoSession(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.GetContactSessionData(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp handlers.ContactSessionDataResponse
	testutil.ParseEnvelopeResponse(t, req, &resp)
	assert.Nil(t, resp.SessionID)
	assert.Nil(t, resp.FlowID)
	assert.Equal(t, "", resp.FlowName)
	assert.NotNil(t, resp.SessionData)
	assert.NotNil(t, resp.PanelConfig)
}

// --- Lifecycle Flow Tests ---

func TestApp_ContactLifecycle_FullCycle(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.Status = models.ChatStatusPending
	})

	claimReq := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(claimReq, org.ID, user.ID)
	testutil.SetPathParam(claimReq, "id", contact.ID.String())
	err := app.ClaimChat(claimReq)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(claimReq))

	var afterClaim models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&afterClaim).Error)
	assert.Equal(t, models.ChatStatusOpen, afterClaim.Status)
	require.NotNil(t, afterClaim.AssignedUserID)
	assert.Equal(t, user.ID, *afterClaim.AssignedUserID)

	closeReq := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(closeReq, org.ID, user.ID)
	testutil.SetPathParam(closeReq, "id", contact.ID.String())
	err = app.CloseChat(closeReq)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(closeReq))

	var afterClose models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&afterClose).Error)
	assert.Equal(t, models.ChatStatusClosed, afterClose.Status)
	require.NotNil(t, afterClose.ClosedAt)

	reopenReq := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(reopenReq, org.ID, user.ID)
	testutil.SetPathParam(reopenReq, "id", contact.ID.String())
	err = app.ReopenChat(reopenReq)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(reopenReq))

	var afterReopen models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&afterReopen).Error)
	assert.Equal(t, models.ChatStatusPending, afterReopen.Status)
	assert.Nil(t, afterReopen.AssignedUserID)
	assert.Nil(t, afterReopen.ClosedAt)
	assert.Nil(t, afterReopen.ClosedByUserID)
}

// --- AssignContactRequest deserialization ---

func TestAssignContactRequest_NilUserID(t *testing.T) {
	raw := `{"user_id": null}`
	var req handlers.AssignContactRequest
	require.NoError(t, json.Unmarshal([]byte(raw), &req))
	assert.Nil(t, req.UserID)
}

func TestAssignContactRequest_WithUserID(t *testing.T) {
	id := uuid.New()
	raw := `{"user_id":"` + id.String() + `"}`
	var req handlers.AssignContactRequest
	require.NoError(t, json.Unmarshal([]byte(raw), &req))
	require.NotNil(t, req.UserID)
	assert.Equal(t, id, *req.UserID)
}

func TestSetChatPublicRequest_Deserialization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		expected bool
	}{
		{"public", `{"is_public":true}`, true},
		{"private", `{"is_public":false}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req handlers.SetChatPublicRequest
			require.NoError(t, json.Unmarshal([]byte(tt.raw), &req))
			assert.Equal(t, tt.expected, req.IsPublic)
		})
	}
}

func TestUpdateContactTagsRequest_Deserialization(t *testing.T) {
	raw := `{"tags":["vip","customer"]}`
	var req handlers.UpdateContactTagsRequest
	require.NoError(t, json.Unmarshal([]byte(raw), &req))
	assert.Equal(t, []string{"vip", "customer"}, req.Tags)
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
