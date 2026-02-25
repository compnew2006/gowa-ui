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

func createTestInstance(t *testing.T, app *handlers.App, orgID uuid.UUID, name string) *models.WhatsAppInstance {
	t.Helper()

	instance := &models.WhatsAppInstance{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		Name:            name,
		Status:          models.InstanceStatusDisconnected,
		AutoReadReceipt: false,
	}
	require.NoError(t, app.DB.Create(instance).Error)
	return instance
}

func enableRestrictedInstanceVisibility(
	t *testing.T,
	app *handlers.App,
	orgID uuid.UUID,
	userID uuid.UUID,
	allowedInstanceID uuid.UUID,
) {
	t.Helper()
	enableRestrictedInstanceVisibilityWithStrictAndEnabled(t, app, orgID, userID, allowedInstanceID, true, true)
}

func enableRestrictedInstanceVisibilityWithStrict(
	t *testing.T,
	app *handlers.App,
	orgID uuid.UUID,
	userID uuid.UUID,
	allowedInstanceID uuid.UUID,
	strictOrgMode bool,
) {
	t.Helper()
	enableRestrictedInstanceVisibilityWithStrictAndEnabled(t, app, orgID, userID, allowedInstanceID, strictOrgMode, true)
}

func enableRestrictedInstanceVisibilityWithStrictAndEnabled(
	t *testing.T,
	app *handlers.App,
	orgID uuid.UUID,
	userID uuid.UUID,
	allowedInstanceID uuid.UUID,
	strictOrgMode bool,
	restrictionsEnabled bool,
) {
	t.Helper()

	app.Config.WhatsApp.Provider = "whatsmeow"

	var org models.Organization
	require.NoError(t, app.DB.Where("id = ?", orgID).First(&org).Error)
	if org.Settings == nil {
		org.Settings = models.JSONB{}
	}
	org.Settings["strict_sending_restrictions_enabled"] = strictOrgMode
	require.NoError(t, app.DB.Model(&org).Update("settings", org.Settings).Error)

	var user models.User
	require.NoError(t, app.DB.Where("id = ?", userID).First(&user).Error)
	if user.Settings == nil {
		user.Settings = models.JSONB{}
	}
	user.Settings["send_restrictions"] = models.JSONB{
		"enabled":             restrictionsEnabled,
		"allowed_instance_id": allowedInstanceID.String(),
		"authorized_numbers":  []string{},
	}
	require.NoError(t, app.DB.Model(&user).Update("settings", user.Settings).Error)
}

func TestApp_CreateInstance_DuplicateNameConflict(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	createTestInstance(t, app, org.ID, "Support Line")

	req := testutil.NewJSONRequest(t, map[string]any{
		"name": "  support line  ",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateInstance(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusConflict, "already exists")

	var count int64
	require.NoError(t, app.DB.Model(&models.WhatsAppInstance{}).Where("organization_id = ?", org.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestApp_UpdateInstance_DuplicateNameConflict(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	createTestInstance(t, app, org.ID, "Sales")
	second := createTestInstance(t, app, org.ID, "Support")

	req := testutil.NewJSONRequest(t, map[string]any{
		"name": " sales ",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", second.ID.String())

	err := app.UpdateInstance(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusConflict, "already exists")

	var refreshed models.WhatsAppInstance
	require.NoError(t, app.DB.First(&refreshed, "id = ?", second.ID).Error)
	assert.Equal(t, "Support", refreshed.Name)
}

func TestApp_ListInstances_RestrictedUserOnlySeesAllowedInstance(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	allowed := createTestInstance(t, app, org.ID, "Allowed")
	_ = createTestInstance(t, app, org.ID, "Hidden")
	enableRestrictedInstanceVisibility(t, app, org.ID, user.ID, allowed.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListInstances(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data []models.WhatsAppInstance `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, allowed.ID, resp.Data[0].ID)
}

func TestApp_GetInstance_RestrictedUserCannotAccessOtherInstance(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	allowed := createTestInstance(t, app, org.ID, "Allowed")
	other := createTestInstance(t, app, org.ID, "Other")
	enableRestrictedInstanceVisibility(t, app, org.ID, user.ID, allowed.ID)

	forbiddenReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(forbiddenReq, org.ID, user.ID)
	testutil.SetPathParam(forbiddenReq, "id", other.ID.String())

	err := app.GetInstance(forbiddenReq)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, forbiddenReq, fasthttp.StatusNotFound, "Instance not found")

	allowedReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(allowedReq, org.ID, user.ID)
	testutil.SetPathParam(allowedReq, "id", allowed.ID.String())

	err = app.GetInstance(allowedReq)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(allowedReq))

	var resp struct {
		Data models.WhatsAppInstance `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(allowedReq), &resp))
	assert.Equal(t, allowed.ID, resp.Data.ID)
}
