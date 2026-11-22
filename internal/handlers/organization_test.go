package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// --- GetOrganizationSettings Tests ---

func TestApp_GetOrganizationSettings_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	// Set organization settings
	org.Settings = models.JSONB{
		"mask_phone_numbers": true,
		"timezone":           "Asia/Kolkata",
		"date_format":        "DD/MM/YYYY",
	}
	require.NoError(t, app.DB.Save(org).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Settings handlers.OrganizationSettings `json:"settings"`
			Name     string                        `json:"name"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, true, resp.Data.Settings.MaskPhoneNumbers)
	assert.Equal(t, "Asia/Kolkata", resp.Data.Settings.Timezone)
	assert.Equal(t, "DD/MM/YYYY", resp.Data.Settings.DateFormat)
	assert.Equal(t, org.Name, resp.Data.Name)
}

func TestApp_GetOrganizationSettings_Defaults(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	// Organization with nil settings should return defaults
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Settings handlers.OrganizationSettings `json:"settings"`
			Name     string                        `json:"name"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, false, resp.Data.Settings.MaskPhoneNumbers)
	assert.Equal(t, "UTC", resp.Data.Settings.Timezone)
	assert.Equal(t, "YYYY-MM-DD", resp.Data.Settings.DateFormat)
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
	user := createAdminUser(t, app, org.ID)

	maskEnabled := true
	timezone := "America/New_York"
	dateFormat := "MM/DD/YYYY"
	newName := "Updated Organization"

	req := testutil.NewJSONRequest(t, map[string]any{
		"mask_phone_numbers": maskEnabled,
		"timezone":           timezone,
		"date_format":        dateFormat,
		"name":               newName,
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
	assert.Equal(t, "America/New_York", updatedOrg.Settings["timezone"])
	assert.Equal(t, "MM/DD/YYYY", updatedOrg.Settings["date_format"])
}

func TestApp_UpdateOrganizationSettings_PartialUpdate(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	// Set initial settings
	org.Settings = models.JSONB{
		"mask_phone_numbers": false,
		"timezone":           "UTC",
		"date_format":        "YYYY-MM-DD",
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
	user := createAdminUser(t, app, org.ID)
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
	user := createAdminUser(t, app, org.ID)

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

// --- Task 12: single source for org name / timezone ---

func TestApp_UpdateOrganizationSettings_ReturnsUpdatedSettingsAndName(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"timezone": "America/New_York",
		"name":     "Renamed Organization",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UpdateOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// The write response is authoritative: it echoes the stored values so
	// the UI never has to guess what the server kept.
	var resp struct {
		Data struct {
			Message  string                        `json:"message"`
			Settings handlers.OrganizationSettings `json:"settings"`
			Name     string                        `json:"name"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, "America/New_York", resp.Data.Settings.Timezone)
	assert.Equal(t, "Renamed Organization", resp.Data.Name)

	// GET shows the same values: one source (org.Name + settings["timezone"]).
	getReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(getReq, org.ID, user.ID)
	require.NoError(t, app.GetOrganizationSettings(getReq))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(getReq))

	var getResp struct {
		Data struct {
			Settings handlers.OrganizationSettings `json:"settings"`
			Name     string                        `json:"name"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(getReq), &getResp)
	require.NoError(t, err)
	assert.Equal(t, "America/New_York", getResp.Data.Settings.Timezone)
	assert.Equal(t, "Renamed Organization", getResp.Data.Name)

	// The switcher source (my organizations) shows the same name.
	myReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(myReq, org.ID, user.ID)
	require.NoError(t, app.ListMyOrganizations(myReq))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(myReq))

	var myResp struct {
		Data struct {
			Organizations []struct {
				Name string `json:"name"`
			} `json:"organizations"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(myReq), &myResp)
	require.NoError(t, err)
	require.Len(t, myResp.Data.Organizations, 1)
	assert.Equal(t, "Renamed Organization", myResp.Data.Organizations[0].Name)
}

func TestApp_UpdateOrganizationSettings_InvalidTimezone(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	org.Settings = models.JSONB{"timezone": "UTC"}
	require.NoError(t, app.DB.Save(org).Error)

	req := testutil.NewJSONRequest(t, map[string]any{
		"timezone": "Not/AZone",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UpdateOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))

	// Nothing was stored: the previous value is untouched.
	var stored models.Organization
	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&stored).Error)
	assert.Equal(t, "UTC", stored.Settings["timezone"])
}

func TestApp_UpdateOrganizationSettings_EmptyTimezoneNormalizesToUTC(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	org.Settings = models.JSONB{"timezone": "America/New_York"}
	require.NoError(t, app.DB.Save(org).Error)

	// An empty timezone used to be stored as "" while reads showed the UTC
	// default — a silent read/write divergence. It now normalizes to UTC.
	req := testutil.NewJSONRequest(t, map[string]any{
		"timezone": "",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UpdateOrganizationSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var stored models.Organization
	require.NoError(t, app.DB.Where("id = ?", org.ID).First(&stored).Error)
	assert.Equal(t, "UTC", stored.Settings["timezone"])

	getReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(getReq, org.ID, user.ID)
	require.NoError(t, app.GetOrganizationSettings(getReq))

	var getResp struct {
		Data struct {
			Settings handlers.OrganizationSettings `json:"settings"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(getReq), &getResp))
	assert.Equal(t, "UTC", getResp.Data.Settings.Timezone)
}
