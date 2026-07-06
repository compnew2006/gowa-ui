package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	waManager "github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	waClient "go.mau.fi/whatsmeow"
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

func createInstanceManagerUser(t *testing.T, app *handlers.App, orgID uuid.UUID, roleName string) *models.User {
	t.Helper()
	return createUserWithPermissionKeys(t, app, orgID, roleName, []string{
		"accounts:read",
		"accounts:write",
		"accounts:delete",
	})
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
		"enabled":              restrictionsEnabled,
		"allowed_instance_id":  allowedInstanceID.String(),
		"allowed_instance_ids": []string{allowedInstanceID.String()},
		"authorized_numbers":   []string{},
	}
	require.NoError(t, app.DB.Model(&user).Update("settings", user.Settings).Error)
}

func enableRestrictedInstanceVisibilityMultiple(
	t *testing.T,
	app *handlers.App,
	orgID uuid.UUID,
	userID uuid.UUID,
	allowedInstanceIDs ...uuid.UUID,
) {
	t.Helper()

	app.Config.WhatsApp.Provider = "whatsmeow"

	var org models.Organization
	require.NoError(t, app.DB.Where("id = ?", orgID).First(&org).Error)
	if org.Settings == nil {
		org.Settings = models.JSONB{}
	}
	org.Settings["strict_sending_restrictions_enabled"] = true
	require.NoError(t, app.DB.Model(&org).Update("settings", org.Settings).Error)

	ids := make([]string, 0, len(allowedInstanceIDs))
	for _, id := range allowedInstanceIDs {
		ids = append(ids, id.String())
	}

	var user models.User
	require.NoError(t, app.DB.Where("id = ?", userID).First(&user).Error)
	if user.Settings == nil {
		user.Settings = models.JSONB{}
	}
	var legacy any
	if len(ids) > 0 {
		legacy = ids[0]
	}
	user.Settings["send_restrictions"] = models.JSONB{
		"enabled":              true,
		"allowed_instance_id":  legacy,
		"allowed_instance_ids": ids,
		"authorized_numbers":   []string{},
	}
	require.NoError(t, app.DB.Model(&user).Update("settings", user.Settings).Error)
}

func TestApp_CreateInstance_DuplicateNameConflict(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createInstanceManagerUser(t, app, org.ID, "instance-manager-create")

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
	user := createInstanceManagerUser(t, app, org.ID, "instance-manager-update")

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

func TestApp_UpdateInstance_MergesSettingsAndPersistsAutoDownloadIncomingMedia(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createInstanceManagerUser(t, app, org.ID, "instance-manager-auto-download")
	instance := createTestInstance(t, app, org.ID, "Support")

	instance.Settings = models.JSONB{
		"auto_sync_history":     true,
		"custom_existing_key":   "keep-me",
		"chat_tag_display_mode": "name",
	}
	require.NoError(t, app.DB.Model(instance).Update("settings", instance.Settings).Error)

	req := testutil.NewJSONRequest(t, map[string]any{
		"settings": map[string]any{
			"auto_download_incoming_media": true,
		},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", instance.ID.String())

	err := app.UpdateInstance(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data models.WhatsAppInstance `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, true, resp.Data.Settings["auto_download_incoming_media"])
	assert.Equal(t, true, resp.Data.Settings["auto_sync_history"])
	assert.Equal(t, "keep-me", resp.Data.Settings["custom_existing_key"])
	assert.Equal(t, "name", resp.Data.Settings["chat_tag_display_mode"])

	var updated models.WhatsAppInstance
	require.NoError(t, app.DB.First(&updated, "id = ?", instance.ID).Error)
	assert.Equal(t, true, updated.Settings["auto_download_incoming_media"])
	assert.Equal(t, true, updated.Settings["auto_sync_history"])
	assert.Equal(t, "keep-me", updated.Settings["custom_existing_key"])
	assert.Equal(t, "name", updated.Settings["chat_tag_display_mode"])
}

func TestApp_UpdateInstance_ReindexesConnectedRuntimeKey(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createInstanceManagerUser(t, app, org.ID, "instance-manager-reindex")
	instance := createTestInstance(t, app, org.ID, "Sales")

	manager := waManager.NewConnectionManager(app.DB, nil, app.Log, &config.WhatsmeowConfig{}, nil, "./uploads")
	app.WhatsmeowManager = manager

	client := &waClient.Client{}
	require.NoError(t, manager.RegisterInstanceClient(*instance, client))
	assert.Same(t, client, manager.GetClient(instance.ID))
	assert.Same(t, client, manager.GetClientByKey(waManager.NewInstanceKey(org.ID, "Sales")))

	req := testutil.NewJSONRequest(t, map[string]any{
		"name": " Growth ",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", instance.ID.String())

	err := app.UpdateInstance(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var updated models.WhatsAppInstance
	require.NoError(t, app.DB.First(&updated, "id = ?", instance.ID).Error)
	assert.Equal(t, "Growth", updated.Name)

	assert.Nil(t, manager.GetClientByKey(waManager.NewInstanceKey(org.ID, "Sales")))
	assert.Same(t, client, manager.GetClientByKey(waManager.NewInstanceKey(org.ID, "Growth")))
	assert.Same(t, client, manager.GetClient(instance.ID))
}

func TestApp_ListInstances_RestrictedUserOnlySeesAllowedInstance(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createInstanceManagerUser(t, app, org.ID, "instance-manager-list-one")

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

func TestApp_ListInstances_RestrictedUserSeesMultipleAllowedInstances(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createInstanceManagerUser(t, app, org.ID, "instance-manager-list-many")

	allowedA := createTestInstance(t, app, org.ID, "Allowed A")
	allowedB := createTestInstance(t, app, org.ID, "Allowed B")
	hidden := createTestInstance(t, app, org.ID, "Hidden")
	enableRestrictedInstanceVisibilityMultiple(t, app, org.ID, user.ID, allowedA.ID, allowedB.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListInstances(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data []models.WhatsAppInstance `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	require.Len(t, resp.Data, 2)

	gotIDs := []uuid.UUID{resp.Data[0].ID, resp.Data[1].ID}
	assert.ElementsMatch(t, []uuid.UUID{allowedA.ID, allowedB.ID}, gotIDs)
	for _, instance := range resp.Data {
		assert.NotEqual(t, hidden.ID, instance.ID)
	}
}

func TestApp_GetInstance_RestrictedUserCannotAccessOtherInstance(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createInstanceManagerUser(t, app, org.ID, "instance-manager-get")

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

func TestApp_ListInstances_InjectsAssignedChatResetDefaults(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createInstanceManagerUser(t, app, org.ID, "instance-manager-list-defaults")

	instance := createTestInstance(t, app, org.ID, "Support")
	instance.Settings = models.JSONB{"custom_existing_setting": "keep-me"}
	require.NoError(t, app.DB.Model(instance).Update("settings", instance.Settings).Error)

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
	assert.Equal(t, "keep-me", resp.Data[0].Settings["custom_existing_setting"])
	assert.Equal(t, true, resp.Data[0].Settings["assigned_chat_reset_enabled"])
	assert.Equal(t, "midnight", resp.Data[0].Settings["assigned_chat_reset_mode"])
	assert.Equal(t, float64(0), resp.Data[0].Settings["assigned_chat_reset_hour"])
}

func TestApp_GetInstance_InjectsAssignedChatResetDefaults(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createInstanceManagerUser(t, app, org.ID, "instance-manager-get-defaults")

	instance := createTestInstance(t, app, org.ID, "Support")
	instance.Settings = models.JSONB{"custom_existing_setting": "keep-me"}
	require.NoError(t, app.DB.Model(instance).Update("settings", instance.Settings).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", instance.ID.String())

	err := app.GetInstance(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data models.WhatsAppInstance `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, instance.ID, resp.Data.ID)
	assert.Equal(t, "keep-me", resp.Data.Settings["custom_existing_setting"])
	assert.Equal(t, true, resp.Data.Settings["assigned_chat_reset_enabled"])
	assert.Equal(t, "midnight", resp.Data.Settings["assigned_chat_reset_mode"])
	assert.Equal(t, float64(0), resp.Data.Settings["assigned_chat_reset_hour"])
}
