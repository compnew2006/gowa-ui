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

// --- GetOrganizationSettings Tests ---

func TestApp_GetOrganizationSettings_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("get-settings")),
		testutil.WithRoleID(&adminRole.ID),
	)

	// Set organization settings
	org.Settings = models.JSONB{
		"mask_phone_numbers":                  true,
		"strict_sending_restrictions_enabled": true,
		"timezone":                            "Asia/Kolkata",
		"date_format":                         "DD/MM/YYYY",
		"assigned_chat_reset_enabled":         true,
		"assigned_chat_reset_mode":            "custom_hour",
		"assigned_chat_reset_hour":            9,
	}
	require.NoError(t, app.DB.Save(org).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Settings map[string]any `json:"settings"`
			Name     string         `json:"name"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, true, resp.Data.Settings["mask_phone_numbers"])
	assert.Equal(t, true, resp.Data.Settings["strict_sending_restrictions_enabled"])
	assert.Equal(t, "Asia/Kolkata", resp.Data.Settings["timezone"])
	assert.Equal(t, "DD/MM/YYYY", resp.Data.Settings["date_format"])
	assert.NotContains(t, resp.Data.Settings, "assigned_chat_reset_enabled")
	assert.NotContains(t, resp.Data.Settings, "assigned_chat_reset_mode")
	assert.NotContains(t, resp.Data.Settings, "assigned_chat_reset_hour")
	assert.Equal(t, org.Name, resp.Data.Name)
}

func TestApp_GetOrganizationSettings_Defaults(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("get-settings-defaults")),
		testutil.WithRoleID(&adminRole.ID),
	)

	// Organization with nil settings should return defaults
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Settings map[string]any `json:"settings"`
			Name     string         `json:"name"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, false, resp.Data.Settings["mask_phone_numbers"])
	assert.Equal(t, false, resp.Data.Settings["strict_sending_restrictions_enabled"])
	assert.Equal(t, "UTC", resp.Data.Settings["timezone"])
	assert.Equal(t, "YYYY-MM-DD", resp.Data.Settings["date_format"])
	assert.NotContains(t, resp.Data.Settings, "assigned_chat_reset_enabled")
	assert.NotContains(t, resp.Data.Settings, "assigned_chat_reset_mode")
	assert.NotContains(t, resp.Data.Settings, "assigned_chat_reset_hour")
}

func TestApp_GetOrganizationSettings_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewGETRequest(t)
	// No auth context set

	err := app.GetOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

// --- UpdateOrganizationSettings Tests ---

func TestApp_UpdateOrganizationSettings_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("update-settings")),
		testutil.WithRoleID(&adminRole.ID),
	)

	maskEnabled := true
	timezone := "America/New_York"
	dateFormat := "MM/DD/YYYY"
	newName := "Updated Organization"
	org.Settings = models.JSONB{
		"assigned_chat_reset_enabled": true,
		"assigned_chat_reset_mode":    "custom_hour",
		"assigned_chat_reset_hour":    22,
	}
	require.NoError(t, app.DB.Save(org).Error)

	req := testutil.NewJSONRequest(t, map[string]any{
		"mask_phone_numbers":                  maskEnabled,
		"strict_sending_restrictions_enabled": true,
		"timezone":                            timezone,
		"date_format":                         dateFormat,
		"name":                                newName,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UpdateOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Settings updated successfully", resp.Data.Message)

	// Verify the settings were actually persisted
	var updatedOrg models.Organization
	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&updatedOrg).Error)

	assert.Equal(t, newName, updatedOrg.Name)
	assert.Equal(t, true, updatedOrg.Settings["mask_phone_numbers"])
	assert.Equal(t, true, updatedOrg.Settings["strict_sending_restrictions_enabled"])
	assert.Equal(t, "America/New_York", updatedOrg.Settings["timezone"])
	assert.Equal(t, "MM/DD/YYYY", updatedOrg.Settings["date_format"])
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_enabled")
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_mode")
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_hour")
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_last_date")
}

func TestApp_UpdateOrganizationSettings_PartialUpdate(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("partial-update")),
		testutil.WithRoleID(&adminRole.ID),
	)

	// Set initial settings
	org.Settings = models.JSONB{
		"mask_phone_numbers":          false,
		"timezone":                    "UTC",
		"date_format":                 "YYYY-MM-DD",
		"assigned_chat_reset_enabled": false,
		"assigned_chat_reset_mode":    "custom_hour",
		"assigned_chat_reset_hour":    8,
	}
	require.NoError(t, app.DB.Save(org).Error)
	originalName := org.Name

	// Only update timezone (partial update)
	req := testutil.NewJSONRequest(t, map[string]any{
		"timezone": "Europe/London",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UpdateOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Verify only timezone changed, other fields remain the same
	var updatedOrg models.Organization
	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&updatedOrg).Error)

	assert.Equal(t, originalName, updatedOrg.Name)
	assert.Equal(t, false, updatedOrg.Settings["mask_phone_numbers"])
	assert.Equal(t, "Europe/London", updatedOrg.Settings["timezone"])
	assert.Equal(t, "YYYY-MM-DD", updatedOrg.Settings["date_format"])
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_enabled")
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_mode")
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_hour")
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_last_date")
}

func TestApp_UpdateOrganizationSettings_IgnoresLegacyAssignedChatResetRequestFields(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("invalid-reset-mode")),
		testutil.WithRoleID(&adminRole.ID),
	)

	req := testutil.NewJSONRequest(t, map[string]any{
		"timezone":                    "Europe/Berlin",
		"assigned_chat_reset_enabled": false,
		"assigned_chat_reset_mode":    "custom_hour",
		"assigned_chat_reset_hour":    11,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UpdateOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var updatedOrg models.Organization
	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&updatedOrg).Error)
	assert.Equal(t, "Europe/Berlin", updatedOrg.Settings["timezone"])
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_enabled")
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_mode")
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_hour")
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_last_date")
}

func TestApp_UpdateOrganizationSettings_RemovesLegacyChatCloseRatingSettings(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("invalid-followup-window")),
		testutil.WithRoleID(&adminRole.ID),
	)

	org.Settings = models.JSONB{
		"chat_close_rating_enabled":                 true,
		"chat_close_rating_window_days":             2,
		"chat_close_rating_followup_window_minutes": 25,
		"chat_close_rating_templates": models.JSONB{
			"en": "Legacy template",
		},
	}
	require.NoError(t, app.DB.Save(org).Error)

	req := testutil.NewJSONRequest(t, map[string]any{
		"timezone": "Europe/Cairo",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UpdateOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var updatedOrg models.Organization
	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&updatedOrg).Error)
	assert.Equal(t, "Europe/Cairo", updatedOrg.Settings["timezone"])
	_, hasEnabled := updatedOrg.Settings["chat_close_rating_enabled"]
	_, hasWindowDays := updatedOrg.Settings["chat_close_rating_window_days"]
	_, hasFollowupWindow := updatedOrg.Settings["chat_close_rating_followup_window_minutes"]
	_, hasTemplates := updatedOrg.Settings["chat_close_rating_templates"]
	assert.False(t, hasEnabled)
	assert.False(t, hasWindowDays)
	assert.False(t, hasFollowupWindow)
	assert.False(t, hasTemplates)
}

func TestApp_UpdateOrganizationSettings_RemovesLegacyAssignedChatResetSettings(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("midnight-reset-hour")),
		testutil.WithRoleID(&adminRole.ID),
	)

	org.Settings = models.JSONB{
		"assigned_chat_reset_enabled":   true,
		"assigned_chat_reset_mode":      "custom_hour",
		"assigned_chat_reset_hour":      17,
		"assigned_chat_reset_last_date": "2026-04-04",
	}
	require.NoError(t, app.DB.Save(org).Error)

	req := testutil.NewJSONRequest(t, map[string]any{
		"timezone": "UTC",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UpdateOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var updatedOrg models.Organization
	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&updatedOrg).Error)
	assert.Equal(t, "UTC", updatedOrg.Settings["timezone"])
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_enabled")
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_mode")
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_hour")
	assert.NotContains(t, updatedOrg.Settings, "assigned_chat_reset_last_date")
}

func TestApp_UpdateOrganizationSettings_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]any{
		"timezone": "UTC",
	})
	// No auth context set

	err := app.UpdateOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_UpdateOrganizationSettings_EmptyNameIgnored(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("empty-name")),
		testutil.WithRoleID(&adminRole.ID),
	)
	originalName := org.Name

	// Send an empty name -- should be ignored
	req := testutil.NewJSONRequest(t, map[string]any{
		"name": "",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UpdateOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Verify name was not changed
	var updatedOrg models.Organization
	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&updatedOrg).Error)
	assert.Equal(t, originalName, updatedOrg.Name)
}

func TestApp_UpdateOrganizationSettings_InvalidJSON(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("invalid-json")),
		testutil.WithRoleID(&adminRole.ID),
	)

	// Create a request with invalid JSON body
	req := testutil.NewGETRequest(t)
	req.RequestCtx.Request.Header.SetMethod("POST")
	req.RequestCtx.Request.Header.SetContentType("application/json")
	req.RequestCtx.Request.SetBody([]byte(`{invalid json`))
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UpdateOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

// --- GetCurrentOrganization Tests ---

func TestApp_GetCurrentOrganization_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("get-current-org")),
		testutil.WithRoleID(&adminRole.ID),
	)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetCurrentOrganization(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.OrganizationResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, org.ID, resp.Data.ID)
	assert.Equal(t, org.Name, resp.Data.Name)
	assert.Equal(t, org.Slug, resp.Data.Slug)
	assert.NotEmpty(t, resp.Data.CreatedAt)
}

func TestApp_GetCurrentOrganization_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewGETRequest(t)
	// No auth context set

	err := app.GetCurrentOrganization(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_GetCurrentOrganization_NotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("get-org-404")),
		testutil.WithRoleID(&adminRole.ID),
	)

	// Set auth context with a non-existent organization ID
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, uuid.New(), user.ID)

	err := app.GetCurrentOrganization(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}
