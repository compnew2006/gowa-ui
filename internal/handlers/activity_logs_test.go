package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestApp_CreateActivityLog_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	req := testutil.NewJSONRequest(t, map[string]any{
		"category":   "custom",
		"event_type": "ui.button_click",
		"action":     "export_contacts",
		"metadata": map[string]any{
			"button": "export",
		},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateActivityLog(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data models.ActivityLog `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, "custom", resp.Data.Category)
	assert.Equal(t, "ui.button_click", resp.Data.EventType)
	assert.Equal(t, "export_contacts", resp.Data.Action)
	assert.Equal(t, "success", resp.Data.Status)
	assert.Equal(t, "custom", resp.Data.Source)
	require.NotNil(t, resp.Data.UserID)
	assert.Equal(t, user.ID, *resp.Data.UserID)

	var count int64
	require.NoError(t, app.DB.Model(&models.ActivityLog{}).Where("user_id = ? AND event_type = ?", user.ID, "ui.button_click").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestApp_CreateActivityLog_InvalidPayload(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	req := testutil.NewJSONRequest(t, map[string]any{
		"category": "custom",
		"action":   "missing_event_type",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateActivityLog(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "event_type is required")
}

func TestApp_ListActivityLogs_OwnEventsOnly(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user1 := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	user2 := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID), testutil.WithEmail(testutil.UniqueEmail("other-user")))

	orgID := org.ID
	user1ID := user1.ID
	user2ID := user2.ID

	require.NoError(t, app.DB.Create(&models.ActivityLog{
		OrganizationID: &orgID,
		UserID:         &user1ID,
		Category:       "system",
		EventType:      "system.api_interaction",
		Action:         "api_request",
		Status:         "success",
		Source:         "system",
		Metadata:       models.JSONB{"k": "v1"},
	}).Error)
	require.NoError(t, app.DB.Create(&models.ActivityLog{
		OrganizationID: &orgID,
		UserID:         &user2ID,
		Category:       "system",
		EventType:      "system.api_interaction",
		Action:         "api_request",
		Status:         "success",
		Source:         "system",
		Metadata:       models.JSONB{"k": "v2"},
	}).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user1.ID)

	err := app.ListActivityLogs(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Logs  []models.ActivityLog `json:"logs"`
			Total int64                `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
	require.Len(t, resp.Data.Logs, 1)
	require.NotNil(t, resp.Data.Logs[0].UserID)
	assert.Equal(t, user1.ID, *resp.Data.Logs[0].UserID)
}

func TestApp_ListActivityLogs_FilterAndPagination(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	orgID := org.ID
	userID := user.ID

	createLog := func(category, eventType, status, source string, createdAt time.Time) {
		log := models.ActivityLog{
			OrganizationID: &orgID,
			UserID:         &userID,
			Category:       category,
			EventType:      eventType,
			Action:         "api_request",
			Status:         status,
			Source:         source,
		}
		require.NoError(t, app.DB.Create(&log).Error)
		require.NoError(t, app.DB.Model(&models.ActivityLog{}).Where("id = ?", log.ID).Update("created_at", createdAt).Error)
	}

	now := time.Now().UTC()
	createLog("system", "system.api_interaction", "success", "system", now)
	createLog("system", "system.api_interaction", "failure", "system", now)
	createLog("custom", "ui.click", "success", "custom", now.Add(-48*time.Hour))

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "category", "system")
	testutil.SetQueryParam(req, "status", "success")
	testutil.SetQueryParam(req, "source", "system")
	testutil.SetQueryParam(req, "event_type", "system.api_interaction")
	testutil.SetQueryParam(req, "limit", 1)
	testutil.SetQueryParam(req, "page", 1)
	testutil.SetQueryParam(req, "start_date", now.Add(-24*time.Hour).Format("2006-01-02"))

	err := app.ListActivityLogs(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Logs  []models.ActivityLog `json:"logs"`
			Total int64                `json:"total"`
			Page  int                  `json:"page"`
			Limit int                  `json:"limit"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
	assert.Equal(t, 1, resp.Data.Page)
	assert.Equal(t, 1, resp.Data.Limit)
	require.Len(t, resp.Data.Logs, 1)
	assert.Equal(t, "system", resp.Data.Logs[0].Category)
	assert.Equal(t, "success", resp.Data.Logs[0].Status)
}

func TestApp_CreateActivityLog_InvalidContactID(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	req := testutil.NewJSONRequest(t, map[string]any{
		"category":   "custom",
		"event_type": "ui.button_click",
		"action":     "export_contacts",
		"contact_id": "not-a-uuid",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateActivityLog(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Invalid contact ID")
}

func TestApp_CreateActivityLog_InvalidMessageID(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	req := testutil.NewJSONRequest(t, map[string]any{
		"category":   "custom",
		"event_type": "ui.button_click",
		"action":     "export_contacts",
		"message_id": uuid.Nil.String() + "-invalid",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateActivityLog(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Invalid message ID")
}

func TestApp_ActivityLogs_ForbiddenForAgent(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))

	createReq := testutil.NewJSONRequest(t, map[string]any{
		"category":   "custom",
		"event_type": "ui.button_click",
		"action":     "export_contacts",
	})
	testutil.SetAuthContext(createReq, org.ID, user.ID)

	err := app.CreateActivityLog(createReq)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, createReq, fasthttp.StatusForbidden, "Access denied")

	listReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(listReq, org.ID, user.ID)

	err = app.ListActivityLogs(listReq)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, listReq, fasthttp.StatusForbidden, "Access denied")
}
