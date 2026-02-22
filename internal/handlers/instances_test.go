package handlers_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
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
