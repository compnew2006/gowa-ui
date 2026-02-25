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

func TestApp_GetUserSendRestrictions_AutoMergesIncomingNumbers(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	targetUser := testutil.CreateTestUser(t, app.DB, org.ID)

	var target models.User
	require.NoError(t, app.DB.Where("id = ?", targetUser.ID).First(&target).Error)
	target.Settings = models.JSONB{
		"send_restrictions": models.JSONB{
			"enabled":              true,
			"include_all_contacts": true,
			"authorized_numbers":   []string{},
		},
	}
	require.NoError(t, app.DB.Model(&target).Update("settings", target.Settings).Error)

	testutil.CreateTestContactWith(
		t,
		app.DB,
		org.ID,
		testutil.WithPhoneNumber("+15550112233"),
	)

	contact := testutil.CreateTestContactWith(
		t,
		app.DB,
		org.ID,
		testutil.WithPhoneNumber("+15550987654"),
		func(c *models.Contact) {
			c.AssignedUserID = &targetUser.ID
		},
	)

	require.NoError(t, app.DB.Create(&models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		ContactID:      contact.ID,
		Direction:      models.DirectionIncoming,
		MessageType:    models.MessageTypeText,
		Content:        "Incoming hello",
		Status:         models.MessageStatusReceived,
	}).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, adminUser.ID)
	testutil.SetPathParam(req, "id", targetUser.ID.String())

	err := app.GetUserSendRestrictions(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Enabled           bool     `json:"enabled"`
			AuthorizedNumbers []string `json:"authorized_numbers"`
			PrefixAgentName   bool     `json:"prefix_agent_name"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.True(t, resp.Data.Enabled)
	assert.True(t, resp.Data.PrefixAgentName)
	assert.Contains(t, resp.Data.AuthorizedNumbers, "15550112233")
	assert.Contains(t, resp.Data.AuthorizedNumbers, "15550987654")

	var reloaded models.User
	require.NoError(t, app.DB.Where("id = ?", targetUser.ID).First(&reloaded).Error)
	settingsJSON, err := json.Marshal(reloaded.Settings)
	require.NoError(t, err)
	assert.Contains(t, string(settingsJSON), "15550987654")
}

func TestApp_UpdateUserSendRestrictions_NormalizesNumbers(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	targetUser := testutil.CreateTestUser(t, app.DB, org.ID)
	instance := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "restricted-instance",
		Status:         models.InstanceStatusConnected,
	}
	require.NoError(t, app.DB.Create(instance).Error)

	req := testutil.NewJSONRequest(t, map[string]interface{}{
		"enabled":              true,
		"include_all_contacts": true,
		"allowed_instance_id":  instance.ID.String(),
		"prefix_agent_name":    false,
		"authorized_numbers": []string{
			"+1 555-000-1111",
			"15550001111",
			"invalid-number",
		},
	})
	testutil.SetAuthContext(req, org.ID, adminUser.ID)
	testutil.SetPathParam(req, "id", targetUser.ID.String())

	err := app.UpdateUserSendRestrictions(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Enabled            bool     `json:"enabled"`
			IncludeAllContacts bool     `json:"include_all_contacts"`
			AllowedInstanceID  string   `json:"allowed_instance_id"`
			AuthorizedNumbers  []string `json:"authorized_numbers"`
			PrefixAgentName    bool     `json:"prefix_agent_name"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.True(t, resp.Data.Enabled)
	assert.True(t, resp.Data.IncludeAllContacts)
	assert.Equal(t, instance.ID.String(), resp.Data.AllowedInstanceID)
	assert.Equal(t, []string{"15550001111"}, resp.Data.AuthorizedNumbers)
	assert.False(t, resp.Data.PrefixAgentName)
}
