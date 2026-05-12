package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestApp_DismissNotification_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	instance := createTestInstance(t, app, org.ID, "Dismiss Notification")
	notification := &models.InstanceNotification{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     instance.ID,
		EventType:      "logged_out",
		Message:        "WhatsApp session was logged out.",
	}
	require.NoError(t, app.DB.Create(notification).Error)

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", notification.ID.String())

	err := app.DismissNotification(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var result struct {
		Status string                      `json:"status"`
		Data   models.InstanceNotification `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &result))
	assert.Equal(t, "success", result.Status)
	assert.True(t, result.Data.IsDismissed)

	var refreshed models.InstanceNotification
	require.NoError(t, app.DB.Where("id = ? AND organization_id = ?", notification.ID, org.ID).First(&refreshed).Error)
	assert.True(t, refreshed.IsDismissed)
}
