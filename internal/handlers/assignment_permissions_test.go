package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestApp_AssignContact_AllowsContactsWritePermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	assigner := createUserWithPermissionKeys(t, app, org.ID, "contacts-writer", []string{"contacts:write"})
	targetUser := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": targetUser.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, assigner.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	require.NotNil(t, refreshed.AssignedUserID)
	assert.Equal(t, targetUser.ID, *refreshed.AssignedUserID)
}

func TestApp_AssignContact_RejectsWithoutAssignmentPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	assigner := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	targetUser := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": targetUser.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, assigner.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to assign contacts")
}

func TestApp_AssignContact_RejectsAssigneeWithoutInstanceAccess(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	assigner := createUserWithPermissionKeys(t, app, org.ID, "chat-assigner", []string{"chat.assign:write"})
	targetUser := testutil.CreateTestUser(t, app.DB, org.ID)
	allowed := createTestInstance(t, app, org.ID, "Allowed Contact Assignment")
	blocked := createTestInstance(t, app, org.ID, "Blocked Contact Assignment")
	enableRestrictedInstanceVisibility(t, app, org.ID, targetUser.ID, allowed.ID)

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.InstanceID = &blocked.ID
	})

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": targetUser.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, assigner.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "does not have access")

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	assert.Nil(t, refreshed.AssignedUserID)
}

func TestApp_ListAgentTransfers_IncludesInstanceID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	instance := createTestInstance(t, app, org.ID, "Transfers Instance")

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.InstanceID = &instance.ID
	})
	transfer := createTestTransfer(t, app, org.ID, contact.ID, account.Name, models.TransferStatusActive, nil)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListAgentTransfers(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var result struct {
		Data struct {
			Transfers []handlers.AgentTransferResponse `json:"transfers"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &result))
	require.Len(t, result.Data.Transfers, 1)
	assert.Equal(t, transfer.ID.String(), result.Data.Transfers[0].ID)
	require.NotNil(t, result.Data.Transfers[0].InstanceID)
	assert.Equal(t, instance.ID.String(), *result.Data.Transfers[0].InstanceID)
}

func TestApp_AssignAgentTransfer_RejectsAssigneeWithoutInstanceAccess(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	assigner := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	allowed := createTestInstance(t, app, org.ID, "Allowed Transfer Assignment")
	blocked := createTestInstance(t, app, org.ID, "Blocked Transfer Assignment")

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.InstanceID = &blocked.ID
	})
	targetAgent := createTestAgent(t, app, org.ID)
	enableRestrictedInstanceVisibility(t, app, org.ID, targetAgent.ID, allowed.ID)
	transfer := createTestTransfer(t, app, org.ID, contact.ID, account.Name, models.TransferStatusActive, nil)

	req := testutil.NewJSONRequest(t, map[string]any{
		"agent_id": targetAgent.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, assigner.ID)
	testutil.SetPathParam(req, "id", transfer.ID.String())

	err := app.AssignAgentTransfer(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "does not have access")

	var refreshed models.AgentTransfer
	require.NoError(t, app.DB.Where("id = ?", transfer.ID).First(&refreshed).Error)
	assert.Nil(t, refreshed.AgentID)
}
