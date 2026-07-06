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
	assert.Equal(t, account.Name, result.Data.Transfers[0].WhatsAppAccount)
}

func TestApp_ListAgentTransfers_FiltersBlockedInstances(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "restricted-transfer-reader", []string{"transfers:read", "transfers:pickup"})
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	allowed := createTestInstance(t, app, org.ID, "Allowed Transfer List")
	blocked := createTestInstance(t, app, org.ID, "Blocked Transfer List")
	enableRestrictedInstanceVisibility(t, app, org.ID, user.ID, allowed.ID)

	allowedContact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.InstanceID = &allowed.ID
	})
	blockedContact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.InstanceID = &blocked.ID
	})

	allowedTransfer := createTestTransfer(t, app, org.ID, allowedContact.ID, account.Name, models.TransferStatusActive, nil)
	_ = createTestTransfer(t, app, org.ID, blockedContact.ID, account.Name, models.TransferStatusActive, nil)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListAgentTransfers(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var result struct {
		Data struct {
			Transfers         []handlers.AgentTransferResponse `json:"transfers"`
			GeneralQueueCount int64                            `json:"general_queue_count"`
			TotalCount        int64                            `json:"total_count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &result))
	require.Len(t, result.Data.Transfers, 1)
	assert.Equal(t, allowedTransfer.ID.String(), result.Data.Transfers[0].ID)
	assert.Equal(t, int64(1), result.Data.GeneralQueueCount)
	assert.Equal(t, int64(1), result.Data.TotalCount)
}

func TestApp_ListAgentTransfers_RejectsWithoutTransferReadPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "no-transfer-read", []string{"chat:read"})

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListAgentTransfers(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to view transfers")
}

func TestApp_CreateAgentTransfer_RejectsWithoutTransferWritePermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "transfer-reader-only", []string{"transfers:read"})
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"contact_id":       contact.ID.String(),
		"whatsapp_account": account.Name,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateAgentTransfer(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to create transfers")
}

func TestApp_CreateAgentTransfer_RejectsAssigneeWithoutInstanceAccess(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	assigner := createUserWithPermissionKeys(t, app, org.ID, "transfer-writer", []string{"transfers:write"})
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	allowed := createTestInstance(t, app, org.ID, "Allowed Transfer Create")
	blocked := createTestInstance(t, app, org.ID, "Blocked Transfer Create")

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.InstanceID = &blocked.ID
	})
	targetAgent := createTestAgent(t, app, org.ID)
	enableRestrictedInstanceVisibility(t, app, org.ID, targetAgent.ID, allowed.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"contact_id":       contact.ID.String(),
		"whatsapp_account": account.Name,
		"agent_id":         targetAgent.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, assigner.ID)

	err := app.CreateAgentTransfer(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "does not have access")

	var count int64
	require.NoError(t, app.DB.Model(&models.AgentTransfer{}).
		Where("organization_id = ? AND contact_id = ?", org.ID, contact.ID).
		Count(&count).Error)
	assert.Zero(t, count)
}

func TestApp_ClaimChat_FiltersBlockedInstances(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "restricted-chat-claimer", []string{"chat:write"})
	allowed := createTestInstance(t, app, org.ID, "Allowed Claim")
	blocked := createTestInstance(t, app, org.ID, "Blocked Claim")
	enableRestrictedInstanceVisibility(t, app, org.ID, user.ID, allowed.ID)

	blockedContact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.InstanceID = &blocked.ID
	})

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodPut)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", blockedContact.ID.String())

	err := app.ClaimChat(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "Chat not found")

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", blockedContact.ID).First(&refreshed).Error)
	assert.Nil(t, refreshed.AssignedUserID)
}

func TestApp_ClaimChat_AllowsAllowedRestrictedInstance(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "allowed-chat-claimer", []string{"chat:write"})
	allowed := createTestInstance(t, app, org.ID, "Allowed Claim Success")
	enableRestrictedInstanceVisibility(t, app, org.ID, user.ID, allowed.ID)

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.InstanceID = &allowed.ID
	})

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodPut)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.ClaimChat(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	require.NotNil(t, refreshed.AssignedUserID)
	assert.Equal(t, user.ID, *refreshed.AssignedUserID)
}

func TestApp_ResumeFromTransfer_RejectsWithoutTransferWritePermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "transfer-reader", []string{"transfers:read"})
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	transfer := createTestTransfer(t, app, org.ID, contact.ID, account.Name, models.TransferStatusActive, nil)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", transfer.ID.String())

	err := app.ResumeFromTransfer(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission to resume transfers")
}

func TestApp_PickNextTransfer_FiltersBlockedInstances(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "restricted-transfer-picker", []string{"transfers:read", "transfers:pickup"})
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	allowed := createTestInstance(t, app, org.ID, "Allowed Transfer Pickup")
	blocked := createTestInstance(t, app, org.ID, "Blocked Transfer Pickup")
	enableRestrictedInstanceVisibility(t, app, org.ID, user.ID, allowed.ID)

	blockedContact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.InstanceID = &blocked.ID
	})
	allowedContact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.InstanceID = &allowed.ID
	})

	_ = createTestTransfer(t, app, org.ID, blockedContact.ID, account.Name, models.TransferStatusActive, nil)
	allowedTransfer := createTestTransfer(t, app, org.ID, allowedContact.ID, account.Name, models.TransferStatusActive, nil)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.PickNextTransfer(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var result struct {
		Data struct {
			Transfer handlers.AgentTransferResponse `json:"transfer"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &result))
	assert.Equal(t, allowedTransfer.ID.String(), result.Data.Transfer.ID)
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

// --- Multi-org and lifecycle tests ---

func TestApp_AssignContact_AllowsMultiOrgUser(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	homeOrg := testutil.CreateTestOrganization(t, app.DB)
	targetOrg := testutil.CreateTestOrganization(t, app.DB)

	// Create a user whose home org is homeOrg but who is also a member of targetOrg
	// via user_organizations.
	homeUser := testutil.CreateTestUser(t, app.DB, homeOrg.ID)

	// Add a user_organizations entry for targetOrg so the user appears in the
	// targetOrg's user list (like ListUsers does).
	userOrg := &models.UserOrganization{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		UserID:         homeUser.ID,
		OrganizationID: targetOrg.ID,
		IsDefault:      false,
	}
	require.NoError(t, app.DB.Create(userOrg).Error)

	assigner := createUserWithPermissionKeys(t, app, targetOrg.ID, "contacts-writer-multi", []string{"contacts:write"})
	contact := testutil.CreateTestContact(t, app.DB, targetOrg.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": homeUser.ID.String(),
	})
	testutil.SetAuthContext(req, targetOrg.ID, assigner.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	require.NotNil(t, refreshed.AssignedUserID)
	assert.Equal(t, homeUser.ID, *refreshed.AssignedUserID)
}

func TestApp_AssignContact_RejectsUserNotInOrg(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	otherOrg := testutil.CreateTestOrganization(t, app.DB)

	// Create a user in otherOrg only — no membership in org at all.
	otherOrgUser := testutil.CreateTestUser(t, app.DB, otherOrg.ID)

	assigner := createUserWithPermissionKeys(t, app, org.ID, "contacts-writer-reject", []string{"contacts:write"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": otherOrgUser.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, assigner.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "not a member")

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	assert.Nil(t, refreshed.AssignedUserID)
}

func TestApp_AssignContact_UnassignOpenChat(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	assigner := createUserWithPermissionKeys(t, app, org.ID, "contacts-writer-unassign", []string{"contacts:write"})
	targetUser := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// First, assign the contact
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"assigned_user_id": targetUser.ID,
		"status":           models.ChatStatusOpen,
	}).Error)

	// Now unassign by sending user_id: null
	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": nil,
	})
	req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodPut)
	testutil.SetAuthContext(req, org.ID, assigner.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", contact.ID).First(&refreshed).Error)
	assert.Nil(t, refreshed.AssignedUserID)
	assert.Equal(t, string(models.ChatStatusPending), refreshed.Status)
}

func TestApp_AssignContact_RejectsAssignClosedChat(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	assigner := createUserWithPermissionKeys(t, app, org.ID, "contacts-writer-closed", []string{"contacts:write"})
	targetUser := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Close the contact
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"status":            models.ChatStatusClosed,
		"assigned_user_id":  targetUser.ID,
		"closed_at":         time.Now().UTC(),
		"closed_by_user_id": assigner.ID,
	}).Error)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": targetUser.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, assigner.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusConflict, "closed chat")
}

func TestApp_AssignContact_RejectsUnassignClosedChat(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	assigner := createUserWithPermissionKeys(t, app, org.ID, "contacts-writer-unassign-closed", []string{"contacts:write"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Close the contact
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"status":            models.ChatStatusClosed,
		"closed_at":         time.Now().UTC(),
		"closed_by_user_id": assigner.ID,
	}).Error)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": nil,
	})
	req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodPut)
	testutil.SetAuthContext(req, org.ID, assigner.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.AssignContact(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusConflict, "closed chat")
}
