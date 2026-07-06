package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func stringPtr(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func stringSlicePtr(values []string) *[]string {
	return &values
}

func createRolesUserWithPermissions(t *testing.T, app *handlers.App, orgID uuid.UUID, permissionKeys ...string) *models.User {
	t.Helper()

	role := testutil.CreateTestRoleWithKeys(t, app.DB, orgID, "roles-user", permissionKeys)
	return testutil.CreateTestUser(
		t,
		app.DB,
		orgID,
		testutil.WithEmail(testutil.UniqueEmail("roles-user")),
		testutil.WithRoleID(&role.ID),
	)
}

func TestApp_ListRoles_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	permissions := testutil.GetOrCreateTestPermissions(t, app.DB)

	// Create some roles
	adminRole := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Admin", true, false, permissions)
	agentRole := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Agent", false, true, permissions[:3])

	// Create a user to make the request
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:read")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListRoles(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Roles []handlers.RoleResponse `json:"roles"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.Len(t, resp.Data.Roles, 2)

	// Check that roles are sorted (system first, then by name)
	assert.Equal(t, adminRole.Name, resp.Data.Roles[0].Name)
	assert.True(t, resp.Data.Roles[0].IsSystem)
	assert.Equal(t, agentRole.Name, resp.Data.Roles[1].Name)
	assert.True(t, resp.Data.Roles[1].IsDefault)
}

func TestApp_GetRole_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	permissions := testutil.GetOrCreateTestPermissions(t, app.DB)

	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Test Role", false, false, permissions[:2])
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:read")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", role.ID.String())

	err := app.GetRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string                `json:"status"`
		Data   handlers.RoleResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, role.ID, resp.Data.ID)
	assert.Equal(t, role.Name, resp.Data.Name)
	assert.Len(t, resp.Data.Permissions, 2)
}

func TestApp_GetRole_NotFound(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:read")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", uuid.New().String())

	err := app.GetRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestApp_CreateRole_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	permissions := testutil.GetOrCreateTestPermissions(t, app.DB)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:write")

	reqBody := handlers.CreateRoleRequest{
		Name:        "New Role",
		Description: "A new custom role",
		IsDefault:   false,
		Permissions: []string{"users:read", "users:write"},
	}

	req := testutil.NewJSONRequest(t, reqBody)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string                `json:"status"`
		Data   handlers.RoleResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, "New Role", resp.Data.Name)
	assert.Equal(t, "A new custom role", resp.Data.Description)
	assert.False(t, resp.Data.IsSystem)
	assert.Len(t, resp.Data.Permissions, 2)

	// Verify permissions were assigned correctly
	var dbRole models.CustomRole
	require.NoError(t, app.DB.Preload("Permissions").First(&dbRole, "id = ?", resp.Data.ID).Error)
	assert.Len(t, dbRole.Permissions, 2)

	// Clean up permissions for next test
	_ = permissions
}

func TestApp_CreateRole_DuplicateName(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	_ = testutil.GetOrCreateTestPermissions(t, app.DB)

	testutil.CreateTestRoleExact(t, app.DB, org.ID, "Existing Role", false, false, nil)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:write")

	reqBody := handlers.CreateRoleRequest{
		Name:        "Existing Role",
		Description: "Trying to create duplicate",
		Permissions: []string{},
	}

	req := testutil.NewJSONRequest(t, reqBody)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusConflict, testutil.GetResponseStatusCode(req))
}

func TestApp_CreateRole_MissingName(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:write")

	reqBody := handlers.CreateRoleRequest{
		Name:        "",
		Description: "Role without name",
		Permissions: []string{},
	}

	req := testutil.NewJSONRequest(t, reqBody)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_CreateRole_WithDefaultFlag(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	_ = testutil.GetOrCreateTestPermissions(t, app.DB)

	// Create an existing default role
	existingDefault := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Old Default", false, true, nil)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:write")

	reqBody := handlers.CreateRoleRequest{
		Name:        "New Default Role",
		Description: "This will be the new default",
		IsDefault:   true,
		Permissions: []string{},
	}

	req := testutil.NewJSONRequest(t, reqBody)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Verify the old default was unset
	var oldDefault models.CustomRole
	require.NoError(t, app.DB.First(&oldDefault, "id = ?", existingDefault.ID).Error)
	assert.False(t, oldDefault.IsDefault)
}

func TestApp_UpdateRole_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	permissions := testutil.GetOrCreateTestPermissions(t, app.DB)

	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Editable Role", false, false, permissions[:1])
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:write")

	reqBody := handlers.UpdateRoleRequest{
		Name:        stringPtr("Updated Role Name"),
		Description: stringPtr("Updated description"),
		Permissions: stringSlicePtr([]string{"users:read", "users:write", "contacts:read"}),
	}

	req := testutil.NewJSONRequest(t, reqBody)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", role.ID.String())

	err := app.UpdateRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string                `json:"status"`
		Data   handlers.RoleResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "Updated Role Name", resp.Data.Name)
	assert.Equal(t, "Updated description", resp.Data.Description)
	assert.Len(t, resp.Data.Permissions, 3)
}

func TestApp_UpdateRole_SystemRoleRejectedWithoutSuperAdmin(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	permissions := testutil.GetOrCreateTestPermissions(t, app.DB)

	// Create a system role
	systemRole := testutil.CreateTestRoleExact(t, app.DB, org.ID, "System Admin", true, false, permissions)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:write")

	reqBody := handlers.UpdateRoleRequest{
		Description: stringPtr("Updated description"),
	}

	req := testutil.NewJSONRequest(t, reqBody)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", systemRole.ID.String())

	err := app.UpdateRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestApp_UpdateRole_SystemRoleSuperAdminCanEditDescriptionAndPermissions(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	permissions := testutil.GetOrCreateTestPermissions(t, app.DB)

	systemRole := testutil.CreateTestRoleExact(t, app.DB, org.ID, "System Admin", true, false, permissions[:2])
	user := testutil.CreateTestUser(
		t,
		app.DB,
		org.ID,
		testutil.WithEmail(testutil.UniqueEmail("update-sys-role")),
		testutil.WithSuperAdmin(),
	)

	reqBody := handlers.UpdateRoleRequest{
		Description: stringPtr("Updated description"),
		Permissions: stringSlicePtr([]string{"users:read"}),
	}

	req := testutil.NewJSONRequest(t, reqBody)
	testutil.SetFullAuthContext(req, org.ID, user.ID, user.RoleID, true)
	req.RequestCtx.SetUserValue("id", systemRole.ID.String())

	err := app.UpdateRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string                `json:"status"`
		Data   handlers.RoleResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "System Admin", resp.Data.Name)
	assert.Equal(t, "Updated description", resp.Data.Description)
	assert.Equal(t, []string{"users:read"}, resp.Data.Permissions)
}

func TestApp_UpdateRole_NotFound(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:write")

	reqBody := handlers.UpdateRoleRequest{
		Name: stringPtr("Updated Name"),
	}

	req := testutil.NewJSONRequest(t, reqBody)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", uuid.New().String())

	err := app.UpdateRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestApp_DeleteRole_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Deletable Role", false, false, nil)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:delete")

	req := testutil.NewGETRequest(t)
	req.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", role.ID.String())

	err := app.DeleteRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Verify role was deleted
	var dbRole models.CustomRole
	err = app.DB.First(&dbRole, "id = ?", role.ID).Error
	assert.Error(t, err) // Should be not found
}

func TestApp_DeleteRole_SystemRole(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	systemRole := testutil.CreateTestRoleExact(t, app.DB, org.ID, "System Role", true, false, nil)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:delete")

	req := testutil.NewGETRequest(t)
	req.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", systemRole.ID.String())

	err := app.DeleteRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))

	// Verify role still exists
	var dbRole models.CustomRole
	require.NoError(t, app.DB.First(&dbRole, "id = ?", systemRole.ID).Error)
}

func TestApp_DeleteRole_WithAssignedUsers(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Role With Users", false, false, nil)
	// Create a user with this role
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("assigned-user")), testutil.WithRoleID(&role.ID))
	adminUser := createRolesUserWithPermissions(t, app, org.ID, "roles:delete")

	req := testutil.NewGETRequest(t)
	req.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(req, org.ID, adminUser.ID)
	req.RequestCtx.SetUserValue("id", role.ID.String())

	err := app.DeleteRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_ListPermissions_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	permissions := testutil.GetOrCreateTestPermissions(t, app.DB)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:read")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListPermissions(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Permissions []handlers.PermissionResponse `json:"permissions"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.GreaterOrEqual(t, len(resp.Data.Permissions), len(permissions))

	// Verify permission format
	for _, perm := range resp.Data.Permissions {
		assert.NotEmpty(t, perm.Resource)
		assert.NotEmpty(t, perm.Action)
		assert.Equal(t, perm.Resource+":"+perm.Action, perm.Key)
	}
}

func TestApp_ListRoles_ForbiddenWithoutReadPermission(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("list-no-read")))

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListRoles(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestApp_CreateRole_ForbiddenWithoutWritePermission(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:read")

	req := testutil.NewJSONRequest(t, handlers.CreateRoleRequest{
		Name:        "Blocked Role",
		Description: "Should not be created",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestApp_DeleteRole_ForbiddenWithoutDeletePermission(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Protected Role", false, false, nil)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:read", "roles:write")

	req := testutil.NewGETRequest(t)
	req.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", role.ID.String())

	err := app.DeleteRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestApp_ListPermissions_ForbiddenWithoutReadPermission(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("perms-no-read")))

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListPermissions(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestApp_UpdateRole_ClearPermissionsWithExplicitEmptySlice(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	permissions := testutil.GetOrCreateTestPermissions(t, app.DB)
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Clearable Role", false, false, permissions[:2])
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:write")

	req := testutil.NewJSONRequest(t, handlers.UpdateRoleRequest{
		Permissions: stringSlicePtr([]string{}),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", role.ID.String())

	err := app.UpdateRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var updated models.CustomRole
	require.NoError(t, app.DB.First(&updated, "id = ?", role.ID).Error)
	require.NoError(t, app.DB.Model(&updated).Association("Permissions").Find(&updated.Permissions))
	assert.Empty(t, updated.Permissions)
}

func TestApp_UpdateRole_ExplicitFalseUnsetsDefault(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Default Role", false, true, nil)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:write")

	req := testutil.NewJSONRequest(t, handlers.UpdateRoleRequest{
		IsDefault: boolPtr(false),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", role.ID.String())

	err := app.UpdateRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var updated models.CustomRole
	require.NoError(t, app.DB.First(&updated, "id = ?", role.ID).Error)
	assert.False(t, updated.IsDefault)
}

func TestApp_UpdateRole_OmittedDefaultKeepsExistingValue(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Stable Default Role", false, true, nil)
	user := createRolesUserWithPermissions(t, app, org.ID, "roles:write")

	req := testutil.NewJSONRequest(t, handlers.UpdateRoleRequest{
		Description: stringPtr("Still default"),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", role.ID.String())

	err := app.UpdateRole(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var updated models.CustomRole
	require.NoError(t, app.DB.First(&updated, "id = ?", role.ID).Error)
	assert.True(t, updated.IsDefault)
	assert.Equal(t, "Still default", updated.Description)
}
