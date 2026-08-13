package handlers_test

import (
	"net/http"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// TestDeleteAccount_LogsOutGowaDevice verifies DeleteAccount best-effort logs
// the GOWA device out before soft-deleting the row (gap #4). Not parallel:
// newMsgTestApp mutates the process-global GOWA factory.
func TestDeleteAccount_LogsOutGowaDevice(t *testing.T) {

	mock := newMockGowaServer()
	defer mock.close()

	app := newMsgTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)

	account := createTestAccount(t, app, org.ID)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", account.ID.String())
	require.NoError(t, app.DeleteAccount(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// The gateway must have received the logout call.
	var logoutSent bool
	for _, r := range mock.sentRequests() {
		if r.method == http.MethodPost && r.path == "/devices/"+account.GowaDeviceID+"/logout" {
			logoutSent = true
		}
	}
	assert.True(t, logoutSent, "expected POST /devices/<id>/logout on the gateway")

	// The row must be soft-deleted.
	var deleted models.WhatsAppAccount
	require.NoError(t, app.DB.Unscoped().First(&deleted, "id = ?", account.ID).Error)
	assert.NotNil(t, deleted.DeletedAt)
}

// TestDeleteAccount_WithoutGowaDevice_SkipsLogout verifies accounts without a
// GOWA device id are deleted without any gateway call. Not parallel: shares
// the process-global GOWA factory with TestDeleteAccount_LogsOutGowaDevice.
func TestDeleteAccount_WithoutGowaDevice_SkipsLogout(t *testing.T) {

	mock := newMockGowaServer()
	defer mock.close()

	app := newMsgTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)

	account := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "plain-account-" + uuid.New().String()[:8],
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(account).Error)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", account.ID.String())
	require.NoError(t, app.DeleteAccount(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	assert.Empty(t, mock.sentRequests(), "no gateway call expected for a device-less account")
}

// TestDeleteGowaInstanceDevice_CleansUpLinkedAccounts verifies deleting a GOWA
// device also soft-deletes its WhatsAppAccount row (gap #5) and the user
// assignment rows, while leaving accounts bound to other devices untouched.
// The DB enforces one account per gowa_device_id (idx_wa_accounts_gowa_device),
// so a device maps 1:1 to an account.
func TestDeleteGowaInstanceDevice_CleansUpLinkedAccounts(t *testing.T) {
	t.Parallel()

	mock := newMockGowaServer()
	defer mock.close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)

	inst := &models.GowaInstance{
		OrganizationID: org.ID,
		Name:           "cleanup-" + uuid.New().String()[:8],
		BaseURL:        mock.server.URL,
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(inst).Error)

	deviceID := "dev-cleanup-" + uuid.New().String()[:8]
	otherDeviceID := "dev-other-" + uuid.New().String()[:8]

	makeAccount := func(name, devID string) *models.WhatsAppAccount {
		acc := &models.WhatsAppAccount{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: org.ID,
			Name:           name,
			GowaBaseURL:    inst.BaseURL,
			GowaDeviceID:   devID,
			Status:         "active",
		}
		require.NoError(t, app.DB.Create(acc).Error)
		return acc
	}

	linked := makeAccount("linked-"+uuid.New().String()[:8], deviceID)
	unrelated := makeAccount("unrelated-"+uuid.New().String()[:8], otherDeviceID)

	// A user assignment must be removed alongside the linked account.
	assignment := &models.UserWhatsAppAccount{
		UserID:            admin.ID,
		WhatsAppAccountID: linked.ID,
	}
	require.NoError(t, app.DB.Create(assignment).Error)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	testutil.SetPathParam(req, "deviceId", deviceID)
	require.NoError(t, app.DeleteGowaInstanceDevice(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Linked account: soft-deleted.
	var linkedRow models.WhatsAppAccount
	require.NoError(t, app.DB.Unscoped().First(&linkedRow, "id = ?", linked.ID).Error)
	assert.True(t, linkedRow.DeletedAt.Valid, "linked account must be soft-deleted")

	// Unrelated device account: untouched.
	var unrelatedRow models.WhatsAppAccount
	require.NoError(t, app.DB.First(&unrelatedRow, "id = ?", unrelated.ID).Error)
	assert.False(t, unrelatedRow.DeletedAt.Valid, "account on a different device must survive")

	// Assignments removed.
	var assignCount int64
	require.NoError(t, app.DB.Model(&models.UserWhatsAppAccount{}).
		Where("whats_app_account_id = ?", linked.ID).Count(&assignCount).Error)
	assert.Zero(t, assignCount, "user assignments must be removed with the account")
}

// TestDeleteGowaInstanceDevice_GatewayError_KeepsAccount verifies that when the
// GOWA gateway rejects the device deletion, linked accounts are left intact.
func TestDeleteGowaInstanceDevice_GatewayError_KeepsAccount(t *testing.T) {
	t.Parallel()

	mock := newMockGowaServer()
	defer mock.close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)

	inst := &models.GowaInstance{
		OrganizationID: org.ID,
		Name:           "fail-" + uuid.New().String()[:8],
		BaseURL:        mock.server.URL,
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(inst).Error)

	deviceID := "dev-fail-" + uuid.New().String()[:8]
	acc := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "survivor-" + uuid.New().String()[:8],
		GowaBaseURL:    inst.BaseURL,
		GowaDeviceID:   deviceID,
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(acc).Error)

	mock.setError("gateway down")

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	testutil.SetPathParam(req, "deviceId", deviceID)
	require.NoError(t, app.DeleteGowaInstanceDevice(req))
	assert.Equal(t, fasthttp.StatusBadGateway, testutil.GetResponseStatusCode(req))

	var survivor models.WhatsAppAccount
	require.NoError(t, app.DB.First(&survivor, "id = ?", acc.ID).Error)
	assert.False(t, survivor.DeletedAt.Valid, "account must survive a gateway failure")
}

// TestListGowaInstanceDevices_EnrichesLinkedAccount verifies the device list
// names the WhatsApp account bound to each device (used by the UI for the
// delete warning and the linked-account link), and leaves unbound devices
// without a linked_account field. Not parallel: newMsgTestApp mutates the
// process-global GOWA factory.
func TestListGowaInstanceDevices_EnrichesLinkedAccount(t *testing.T) {
	mock := newMockGowaServer()
	defer mock.close()

	mock.setDevicesResponse([]gowa.DeviceInfo{
		{ID: "dev-enrich-bound", State: "logged_in", JID: "111111@s.whatsapp.net"},
		{ID: "dev-enrich-free", State: "unpaired"},
	})

	app := newMsgTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)

	inst := &models.GowaInstance{
		OrganizationID: org.ID,
		Name:           "enrich-" + uuid.New().String()[:8],
		BaseURL:        mock.server.URL,
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(inst).Error)

	account := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "bound-account-" + uuid.New().String()[:8],
		GowaBaseURL:    inst.BaseURL,
		GowaDeviceID:   "dev-enrich-bound",
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(account).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	require.NoError(t, app.ListGowaInstanceDevices(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var body struct {
		Devices []struct {
			ID            string `json:"id"`
			LinkedAccount *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"linked_account"`
		} `json:"devices"`
	}
	testutil.ParseEnvelopeResponse(t, req, &body)
	require.Len(t, body.Devices, 2, "both mocked devices must be returned")

	var bound, free *struct {
		ID            string `json:"id"`
		LinkedAccount *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"linked_account"`
	}
	for i := range body.Devices {
		switch body.Devices[i].ID {
		case "dev-enrich-bound":
			bound = &body.Devices[i]
		case "dev-enrich-free":
			free = &body.Devices[i]
		}
	}
	require.NotNil(t, bound, "bound device must be present")
	require.NotNil(t, free, "free device must be present")

	require.NotNil(t, bound.LinkedAccount, "bound device must carry its account")
	assert.Equal(t, account.ID.String(), bound.LinkedAccount.ID)
	assert.Equal(t, account.Name, bound.LinkedAccount.Name)
	assert.Nil(t, free.LinkedAccount, "unbound device must not carry a linked_account")

	raw := string(testutil.GetResponseBody(req))
	assert.Contains(t, raw, `"name":"`+account.Name+`"`)
	assert.NotContains(t, raw, "access_token", "account secrets must never leak in the device list")
}

// TestListGowaInstanceDevices_OmitsSoftDeletedAccounts verifies that a
// soft-deleted account no longer appears as the linked account of its device.
func TestListGowaInstanceDevices_OmitsSoftDeletedAccounts(t *testing.T) {
	mock := newMockGowaServer()
	defer mock.close()

	mock.setDevicesResponse([]gowa.DeviceInfo{
		{ID: "dev-enrich-gone", State: "logged_in", JID: "222222@s.whatsapp.net"},
	})

	app := newMsgTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)

	inst := &models.GowaInstance{
		OrganizationID: org.ID,
		Name:           "enrich-gone-" + uuid.New().String()[:8],
		BaseURL:        mock.server.URL,
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(inst).Error)

	account := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "deleted-account-" + uuid.New().String()[:8],
		GowaBaseURL:    inst.BaseURL,
		GowaDeviceID:   "dev-enrich-gone",
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(account).Error)
	require.NoError(t, app.DB.Delete(account).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	require.NoError(t, app.ListGowaInstanceDevices(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var body struct {
		Devices []struct {
			ID            string `json:"id"`
			LinkedAccount *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"linked_account"`
		} `json:"devices"`
	}
	testutil.ParseEnvelopeResponse(t, req, &body)
	require.Len(t, body.Devices, 1)
	assert.Nil(t, body.Devices[0].LinkedAccount,
		"soft-deleted account must not be reported as linked")
}
