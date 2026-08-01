package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/gowa-ui/internal/handlers"
	"github.com/shridarpatil/gowa-ui/internal/models"
	"github.com/shridarpatil/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// --- ListAccounts Tests ---

func TestApp_ListAccounts_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	// Create two accounts for this org
	acc1 := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	acc2 := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListAccounts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Accounts []handlers.AccountResponse `json:"accounts"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Data.Accounts, 2)

	// Accounts are ordered by created_at DESC, so acc2 should be first
	assert.Equal(t, acc2.ID, resp.Data.Accounts[0].ID)
	assert.Equal(t, acc1.ID, resp.Data.Accounts[1].ID)

	// Verify GOWA fields are exposed but the webhook secret is not
	assert.Equal(t, acc2.GowaDeviceID, resp.Data.Accounts[0].GowaDeviceID)
	assert.False(t, resp.Data.Accounts[0].HasGowaWebhookSecret)
}

func TestApp_ListAccounts_Empty(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListAccounts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Accounts []handlers.AccountResponse `json:"accounts"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Data.Accounts, 0)
}

func TestApp_ListAccounts_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewGETRequest(t)
	// No auth context set

	err := app.ListAccounts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_ListAccounts_OrgIsolation(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	user1 := createAdminUser(t, app, org1.ID)
	user2 := createAdminUser(t, app, org2.ID)

	// Create accounts for org1
	testutil.CreateTestWhatsAppAccount(t, app.DB, org1.ID)
	testutil.CreateTestWhatsAppAccount(t, app.DB, org1.ID)

	// Create one account for org2
	testutil.CreateTestWhatsAppAccount(t, app.DB, org2.ID)

	// org1 user should see 2 accounts
	req1 := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req1, org1.ID, user1.ID)

	err := app.ListAccounts(req1)
	require.NoError(t, err)

	var resp1 struct {
		Data struct {
			Accounts []handlers.AccountResponse `json:"accounts"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req1), &resp1)
	require.NoError(t, err)
	assert.Len(t, resp1.Data.Accounts, 2)

	// org2 user should see 1 account
	req2 := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req2, org2.ID, user2.ID)

	err = app.ListAccounts(req2)
	require.NoError(t, err)

	var resp2 struct {
		Data struct {
			Accounts []handlers.AccountResponse `json:"accounts"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req2), &resp2)
	require.NoError(t, err)
	assert.Len(t, resp2.Data.Accounts, 1)
}

// --- CreateAccount Tests ---

func TestApp_CreateAccount_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"name":           "My WhatsApp Account",
		"gowa_base_url":  "http://gowa.local:3000",
		"gowa_device_id": "device-abc",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AccountResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, "My WhatsApp Account", resp.Data.Name)
	assert.Equal(t, "http://gowa.local:3000", resp.Data.GowaBaseURL)
	assert.Equal(t, "device-abc", resp.Data.GowaDeviceID)
	assert.Equal(t, "active", resp.Data.Status)
	assert.True(t, resp.Data.HasGowaWebhookSecret) // auto-generated
	assert.NotEqual(t, uuid.Nil, resp.Data.ID)
}

func TestApp_CreateAccount_WithOptionalFields(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"name":                "Full Account",
		"gowa_base_url":       "http://gowa.local:3001",
		"gowa_device_id":      "device-full",
		"gowa_webhook_secret": "custom-webhook-secret",
		"is_default_incoming": true,
		"is_default_outgoing": true,
		"auto_read_receipt":   true,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AccountResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Full Account", resp.Data.Name)
	assert.Equal(t, "http://gowa.local:3001", resp.Data.GowaBaseURL)
	assert.Equal(t, "device-full", resp.Data.GowaDeviceID)
	assert.True(t, resp.Data.IsDefaultIncoming)
	assert.True(t, resp.Data.IsDefaultOutgoing)
	assert.True(t, resp.Data.AutoReadReceipt)
	assert.True(t, resp.Data.HasGowaWebhookSecret)
}

func TestApp_CreateAccount_ValidationErrors(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	tests := []struct {
		name string
		body map[string]any
	}{
		{
			name: "missing_name",
			body: map[string]any{
				"gowa_base_url":  "http://gowa.local:3000",
				"gowa_device_id": "device-1",
			},
		},
		{
			name: "missing_gowa_base_url",
			body: map[string]any{
				"name":           "Test",
				"gowa_device_id": "device-1",
			},
		},
		{
			name: "missing_gowa_device_id",
			body: map[string]any{
				"name":          "Test",
				"gowa_base_url": "http://gowa.local:3000",
			},
		},
		{
			name: "all_fields_empty",
			body: map[string]any{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := testutil.NewJSONRequest(t, tc.body)
			testutil.SetAuthContext(req, org.ID, user.ID)

			err := app.CreateAccount(req)
			require.NoError(t, err)
			assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
		})
	}
}

func TestApp_CreateAccount_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]any{
		"name":           "Test",
		"gowa_base_url":  "http://gowa.local:3000",
		"gowa_device_id": "device-1",
	})
	// No auth context set

	err := app.CreateAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

// --- GetAccount Tests ---

func TestApp_GetAccount_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.GetAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AccountResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, account.ID, resp.Data.ID)
	assert.Equal(t, account.Name, resp.Data.Name)
	assert.Equal(t, account.GowaBaseURL, resp.Data.GowaBaseURL)
	assert.Equal(t, account.GowaDeviceID, resp.Data.GowaDeviceID)
	assert.Equal(t, "active", resp.Data.Status)
	assert.False(t, resp.Data.HasGowaWebhookSecret)
}

func TestApp_GetAccount_NotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.GetAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestApp_GetAccount_InvalidID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", "not-a-uuid")

	err := app.GetAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_GetAccount_CrossOrgIsolation(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	// Create two separate organizations
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)

	user2 := createAdminUser(t, app, org2.ID)

	// Create account in org1
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org1.ID)

	// User from org2 tries to access org1's account
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org2.ID, user2.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.GetAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

// --- UpdateAccount Tests ---

func TestApp_UpdateAccount_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"name":                "Updated Account Name",
		"gowa_base_url":       "http://gowa.local:3999",
		"gowa_device_id":      "device-updated",
		"gowa_webhook_secret": "new-webhook-secret",
		"auto_read_receipt":   true,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.UpdateAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AccountResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, account.ID, resp.Data.ID)
	assert.Equal(t, "Updated Account Name", resp.Data.Name)
	assert.Equal(t, "http://gowa.local:3999", resp.Data.GowaBaseURL)
	assert.Equal(t, "device-updated", resp.Data.GowaDeviceID)
	assert.True(t, resp.Data.AutoReadReceipt)
	assert.True(t, resp.Data.HasGowaWebhookSecret)

	// Verify the update persisted in the database
	var updated models.WhatsAppAccount
	require.NoError(t, app.DB.Where("id = ?", account.ID).First(&updated).Error)
	assert.Equal(t, "Updated Account Name", updated.Name)
	assert.Equal(t, "http://gowa.local:3999", updated.GowaBaseURL)
	updated.DecryptSecrets(app.Config.App.EncryptionKey)
	assert.Equal(t, "new-webhook-secret", updated.GowaWebhookSecret)
}

func TestApp_UpdateAccount_PartialUpdate(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	// Only update the name, leave other fields unchanged
	req := testutil.NewJSONRequest(t, map[string]any{
		"name": "Only Name Changed",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.UpdateAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AccountResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Only Name Changed", resp.Data.Name)
	// Original values should be preserved
	assert.Equal(t, account.GowaBaseURL, resp.Data.GowaBaseURL)
	assert.Equal(t, account.GowaDeviceID, resp.Data.GowaDeviceID)
}

func TestApp_UpdateAccount_NotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"name": "Updated Name",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.UpdateAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestApp_UpdateAccount_InvalidID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"name": "Updated",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", "not-a-uuid")

	err := app.UpdateAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

// --- DeleteAccount Tests ---

func TestApp_DeleteAccount_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.DeleteAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Account deleted successfully", resp.Data.Message)

	// Verify account is soft-deleted (GORM default with DeletedAt)
	var count int64
	app.DB.Model(&models.WhatsAppAccount{}).Where("id = ?", account.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestApp_DeleteAccount_NotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.DeleteAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestApp_DeleteAccount_InvalidID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", "not-a-uuid")

	err := app.DeleteAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_DeleteAccount_CrossOrgIsolation(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)

	user2 := createAdminUser(t, app, org2.ID)

	// Create account in org1
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org1.ID)

	// User from org2 tries to delete org1's account
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org2.ID, user2.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.DeleteAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))

	// Verify the account still exists in org1
	var count int64
	app.DB.Model(&models.WhatsAppAccount{}).Where("id = ?", account.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

// --- Account Assignment Scoping Tests ---

func TestApp_ListAccounts_AssignedOnly(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	acc1 := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	acc2 := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID) // unassigned

	testutil.AssignAccountToUser(t, app.DB, user.ID, acc1.ID)
	testutil.AssignAccountToUser(t, app.DB, user.ID, acc2.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListAccounts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Accounts []handlers.AccountResponse `json:"accounts"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	require.Len(t, resp.Data.Accounts, 2)

	got := []uuid.UUID{resp.Data.Accounts[0].ID, resp.Data.Accounts[1].ID}
	assert.ElementsMatch(t, []uuid.UUID{acc1.ID, acc2.ID}, got)
}

func TestApp_ListAccounts_NoAssignmentsFallback(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	// No assignments for this user — full org visibility applies.
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListAccounts(req)
	require.NoError(t, err)

	var resp struct {
		Data struct {
			Accounts []handlers.AccountResponse `json:"accounts"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Data.Accounts, 2)
}

func TestApp_ListAccounts_CrossOrgAssignmentIgnored(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org1.ID)

	testutil.CreateTestWhatsAppAccount(t, app.DB, org1.ID)
	testutil.CreateTestWhatsAppAccount(t, app.DB, org1.ID)
	otherOrgAcc := testutil.CreateTestWhatsAppAccount(t, app.DB, org2.ID)

	// Assignment in another org must not restrict visibility in org1.
	testutil.AssignAccountToUser(t, app.DB, user.ID, otherOrgAcc.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org1.ID, user.ID)

	err := app.ListAccounts(req)
	require.NoError(t, err)

	var resp struct {
		Data struct {
			Accounts []handlers.AccountResponse `json:"accounts"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Data.Accounts, 2)
}

func TestApp_ListAccounts_SuperAdminUnrestricted(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	// A role is still required for permission resolution; the super admin
	// flag is what bypasses assignment scoping.
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)
	superAdmin := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("list-superadmin")),
		testutil.WithRoleID(&agentRole.ID),
		testutil.WithSuperAdmin(),
	)

	acc1 := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	// Even with an explicit assignment, super admins see all org accounts.
	testutil.AssignAccountToUser(t, app.DB, superAdmin.ID, acc1.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, superAdmin.ID)

	err := app.ListAccounts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Accounts []handlers.AccountResponse `json:"accounts"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Data.Accounts, 2)
}

func TestApp_GetAccount_AssignmentScoping(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	assigned := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	unassigned := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	testutil.AssignAccountToUser(t, app.DB, user.ID, assigned.ID)

	// Assigned account is accessible
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", assigned.ID.String())

	err := app.GetAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Unassigned account returns 404 (existence is not leaked)
	req2 := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req2, org.ID, user.ID)
	testutil.SetPathParam(req2, "id", unassigned.ID.String())

	err = app.GetAccount(req2)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req2))
}

func TestApp_UpdateAccount_AssignmentScoping(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	assigned := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	unassigned := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	testutil.AssignAccountToUser(t, app.DB, user.ID, assigned.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"name": "Should Not Update",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", unassigned.ID.String())

	err := app.UpdateAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))

	// Account name unchanged in DB
	var current models.WhatsAppAccount
	require.NoError(t, app.DB.Where("id = ?", unassigned.ID).First(&current).Error)
	assert.Equal(t, unassigned.Name, current.Name)
}

func TestApp_DeleteAccount_AssignmentScoping(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	assigned := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	unassigned := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	testutil.AssignAccountToUser(t, app.DB, user.ID, assigned.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", unassigned.ID.String())

	err := app.DeleteAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))

	// Account still exists
	var count int64
	app.DB.Model(&models.WhatsAppAccount{}).Where("id = ?", unassigned.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestApp_DeleteAccount_RemovesAssignments(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)
	agent := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("del-assigned")),
	)

	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	testutil.AssignAccountToUser(t, app.DB, agent.ID, account.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.DeleteAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Dangling assignment rows are cleaned up with the account
	var count int64
	app.DB.Model(&models.UserWhatsAppAccount{}).
		Where("whats_app_account_id = ?", account.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}
