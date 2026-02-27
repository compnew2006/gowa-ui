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

func createUserWithPermissionKeys(t *testing.T, app *handlers.App, orgID uuid.UUID, roleName string, permissionKeys []string) *models.User {
	t.Helper()

	role := testutil.CreateTestRoleWithKeys(t, app.DB, orgID, roleName, permissionKeys)
	return testutil.CreateTestUser(t, app.DB, orgID, testutil.WithRoleID(&role.ID))
}

func TestSensitiveRBAC_ListAccounts_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "no-accounts", []string{"chat:read"})

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListAccounts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestSensitiveRBAC_ListWebhooks_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "no-webhooks", []string{"chat:read"})

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListWebhooks(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestSensitiveRBAC_ListCustomActions_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "no-custom-actions", []string{"chat:read"})

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListCustomActions(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestSensitiveRBAC_GetSSOSettings_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "no-sso-settings", []string{"accounts:read"})

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetSSOSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestSensitiveRBAC_GetBusinessProfile_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "no-business-profile", []string{"webhooks:read"})
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.GetBusinessProfile(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestSensitiveRBAC_ExecuteCustomAction_AllowsChatWritePermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-writer", []string{"chat:write"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	action := createTestCustomAction(t, app, org.ID, "Open CRM", models.ActionTypeURL, map[string]interface{}{
		"url":             "https://crm.example.com/contact/{{contact.id}}",
		"open_in_new_tab": true,
	}, true, 0)

	req := testutil.NewJSONRequest(t, map[string]any{
		"contact_id": contact.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", action.ID.String())

	err := app.ExecuteCustomAction(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.ActionResult `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Data.Success)
	assert.Contains(t, resp.Data.RedirectURL, "/api/custom-actions/redirect/")
}
