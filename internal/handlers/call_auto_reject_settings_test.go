package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/handlers"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// --- GetCallAutoRejectSettings Tests ---

func TestApp_GetCallAutoRejectSettings_Defaults(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.GetCallAutoRejectSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.CallAutoRejectSettingsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	assert.False(t, resp.Data.Enabled)
	assert.NotEmpty(t, resp.Data.Message)
}

func TestApp_GetCallAutoRejectSettings_AccountOverrides(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	settings := models.JSONB{
		"call_auto_reject": map[string]any{
			"enabled": true,
			"message": "",
		},
	}
	require.NoError(t, app.DB.Model(account).Update("settings", settings).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.GetCallAutoRejectSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.CallAutoRejectSettingsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	assert.True(t, resp.Data.Enabled)
	assert.Empty(t, resp.Data.Message) // explicit empty disables the automated text
}

func TestApp_GetCallAutoRejectSettings_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewGETRequest(t)
	// No auth context set

	err := app.GetCallAutoRejectSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_GetCallAutoRejectSettings_NotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.GetCallAutoRejectSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

// --- UpdateCallAutoRejectSettings Tests ---

func TestApp_UpdateCallAutoRejectSettings_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"enabled": true,
		"message": "  معلش، ابعت رسالة وهنرد عليك  ",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.UpdateCallAutoRejectSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var updated models.WhatsAppAccount
	require.NoError(t, app.DB.Where("id = ?", account.ID).First(&updated).Error)

	block, ok := updated.Settings["call_auto_reject"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, block["enabled"])
	assert.Equal(t, "معلش، ابعت رسالة وهنرد عليك", block["message"]) // trimmed

	// Settings on another account must stay untouched (per-account isolation).
	other := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	var otherReloaded models.WhatsAppAccount
	require.NoError(t, app.DB.Where("id = ?", other.ID).First(&otherReloaded).Error)
	_, has := otherReloaded.Settings["call_auto_reject"]
	assert.False(t, has)
}

func TestApp_UpdateCallAutoRejectSettings_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]any{"enabled": true})
	// No auth context set

	err := app.UpdateCallAutoRejectSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_UpdateCallAutoRejectSettings_NotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{"enabled": true})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.UpdateCallAutoRejectSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}
