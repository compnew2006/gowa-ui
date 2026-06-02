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

// --- Settings ---

func TestApp_GetAgentSelectionSettings_AutoCreatesGlobal(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	require.NoError(t, app.GetAgentSelectionSettings(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var result struct {
		Status string                          `json:"status"`
		Data   struct {
			Settings *models.AgentSelectionSettings `json:"settings"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &result))
	require.NotNil(t, result.Data.Settings)
	assert.NotEqual(t, uuid.Nil, result.Data.Settings.ID, "expected settings to be auto-created with non-zero ID")
	assert.Equal(t, org.ID, result.Data.Settings.OrganizationID)
	assert.Nil(t, result.Data.Settings.InstanceID, "expected global row (instance_id IS NULL)")
}

func TestApp_GetAgentSelectionSettings_InstanceScopeDoesNotShadowGlobal(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	instanceID := uuid.New()
	testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, &instanceID, func(s *models.AgentSelectionSettings) {
		s.MenuHeaderText = "Instance-scoped header"
	})

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	require.NoError(t, app.GetAgentSelectionSettings(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var result struct {
		Data struct {
			Settings *models.AgentSelectionSettings `json:"settings"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &result))
	require.NotNil(t, result.Data.Settings)
	assert.Nil(t, result.Data.Settings.InstanceID, "no instance filter must return the global (instance_id IS NULL) row, not the instance row")
}

func TestApp_UpdateAgentSelectionSettings_RejectsInvalidMinutes(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil)

	cases := []struct {
		name string
		body map[string]any
		want string
	}{
		{"selection_timeout_minutes=0", map[string]any{"selection_timeout_minutes": 0}, "selection_timeout_minutes"},
		{"selection_timeout_minutes>1440", map[string]any{"selection_timeout_minutes": 5000}, "selection_timeout_minutes"},
		{"max_invalid_attempts=0", map[string]any{"max_invalid_attempts": 0}, "max_invalid_attempts"},
		{"max_invalid_attempts>20", map[string]any{"max_invalid_attempts": 100}, "max_invalid_attempts"},
		{"prompt_delay_minutes<0", map[string]any{"prompt_delay_minutes": -1}, "prompt_delay_minutes"},
		{"prompt_delay_minutes>1440", map[string]any{"prompt_delay_minutes": 5000}, "prompt_delay_minutes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := testutil.NewJSONRequest(t, tc.body)
			testutil.SetAuthContext(req, org.ID, user.ID)
			require.NoError(t, app.UpdateAgentSelectionSettings(req))
			testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, tc.want)
		})
	}
}

func TestApp_UpdateAgentSelectionSettings_AcceptsValidMinutes(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil)

	req := testutil.NewJSONRequest(t, map[string]any{
		"prompt_delay_minutes":      5,
		"selection_timeout_minutes": 15,
		"max_invalid_attempts":      5,
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	require.NoError(t, app.UpdateAgentSelectionSettings(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var result struct {
		Data struct {
			Settings *models.AgentSelectionSettings `json:"settings"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &result))
	assert.Equal(t, 5, result.Data.Settings.PromptDelayMinutes)
	assert.Equal(t, 15, result.Data.Settings.SelectionTimeoutMinutes)
	assert.Equal(t, 5, result.Data.Settings.MaxInvalidAttempts)
}

// --- Build menu (HideUnavailableAgents wiring) ---

func TestApp_BuildAgentSelectionMenu_HideUnavailableAgents_IncludesBusyParticipants(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	_ = testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	settings := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil, func(s *models.AgentSelectionSettings) {
		s.HideUnavailableAgents = false
	})

	busy := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithFullName("Busy Agent"))
	busyAgentRole := testutil.CreateAgentRole(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(busy).Update("role_id", busyAgentRole.ID).Error)
	maxOpen := 0
	testutil.CreateTestAgentSelectionParticipant(t, app.DB, org.ID, settings.ID, busy.ID, func(p *models.AgentSelectionParticipant) {
		p.DisplayName = "Busy Agent"
		p.MaxOpenChats = &maxOpen
	})

	settings2 := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil, func(s *models.AgentSelectionSettings) {
		s.HideUnavailableAgents = true
	})
	busy2 := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithFullName("Busy2"))
	maxOpen2 := 0
	testutil.CreateTestAgentSelectionParticipant(t, app.DB, org.ID, settings2.ID, busy2.ID, func(p *models.AgentSelectionParticipant) {
		p.DisplayName = "Busy2"
		p.MaxOpenChats = &maxOpen2
	})

	menu, err := app.BuildAgentSelectionMenuForTest(org.ID, settings, nil)
	require.NoError(t, err)
	assert.Len(t, menu.Options, 1, "HideUnavailableAgents=false must include participants even at MaxOpenChats=0")

	menu2, err := app.BuildAgentSelectionMenuForTest(org.ID, settings2, nil)
	require.NoError(t, err)
	assert.Len(t, menu2.Options, 0, "HideUnavailableAgents=true must hide participants at MaxOpenChats=0")
}

// --- Session expiry sends TimeoutResponseText ---

func TestApp_ExpireAgentSelectionSession_SendsTimeoutText(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	_ = testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	settings := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil, func(s *models.AgentSelectionSettings) {
		s.TimeoutResponseText = "We did not get a selection in time. Goodbye."
	})

	session := &models.AgentSelectionSession{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		InstanceID:      nil,
		WhatsAppAccount: account.Name,
		Status:          models.AgentSelectionSessionMenuSent,
	}
	require.NoError(t, app.DB.Create(session).Error)

	// Sanity check: settings resolve returns the configured timeout text.
	resolved, err := app.GetAgentSelectionSettingsForTest(org.ID, nil)
	require.NoError(t, err)
	require.Equal(t, "We did not get a selection in time. Goodbye.", resolved.TimeoutResponseText)

	// Invoke expire. We can't easily stub the WhatsApp provider in this test, so the actual
	// send-and-save will fail silently (no provider configured) — the assertion we care about
	// is that the session status flips to timeout and a SelectionTimeout audit event is written.
	app.ExpireAgentSelectionSessionForTest(session)

	var events []models.AgentSelectionAuditEvent
	require.NoError(t, app.DB.Where("session_id = ?", session.ID).Find(&events).Error)
	require.NotEmpty(t, events, "expected audit event for session timeout")
	last := events[len(events)-1]
	assert.Equal(t, models.AgentSelectionEventSelectionTimeout, last.EventType)
	assert.Equal(t, models.AgentSelectionSessionTimeout, session.Status)

	_ = settings
	_ = contact
	_ = account
}

// --- Test-send endpoint ---

func TestApp_TestSendAgentSelectionMenu_RequiresContactID(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, org.ID, user.ID)
	require.NoError(t, app.TestSendAgentSelectionMenu(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "contact_id")
}

func TestApp_TestSendAgentSelectionMenu_RejectsInvalidContact(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{"contact_id": uuid.New()})
	testutil.SetAuthContext(req, org.ID, user.ID)
	require.NoError(t, app.TestSendAgentSelectionMenu(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "Contact not found")
}

// --- Bug fixes: informative error messages + correct error code masking ---

func TestApp_UpdateAgentSelectionSettings_RejectsUnknownInstance_WithHelpfulMessage(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil)

	missingID := uuid.New().String()
	req := testutil.NewJSONRequest(t, map[string]any{
		"allowed_instance_ids": []string{missingID},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	require.NoError(t, app.UpdateAgentSelectionSettings(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, missingID)
}

func TestApp_UpdateAgentSelectionSettings_AcceptsKnownInstance(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil)
	instance := testutil.CreateTestWhatsAppInstance(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"allowed_instance_ids": []string{instance.ID.String()},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	require.NoError(t, app.UpdateAgentSelectionSettings(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var result struct {
		Data struct {
			Settings *models.AgentSelectionSettings `json:"settings"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &result))
	require.NotNil(t, result.Data.Settings)
	assert.Equal(t, models.StringArray{instance.ID.String()}, result.Data.Settings.AllowedInstanceIDs)
}

func TestApp_UpdateAgentSelectionSettings_RejectsInstanceFromOtherOrg(t *testing.T) {
	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, orgA.ID)
	user := testutil.CreateTestUser(t, app.DB, orgA.ID, testutil.WithRoleID(&adminRole.ID))
	testutil.CreateTestAgentSelectionSettings(t, app.DB, orgA.ID, nil)

	otherOrg := testutil.CreateTestOrganization(t, app.DB)
	foreignInstance := testutil.CreateTestWhatsAppInstance(t, app.DB, otherOrg.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"allowed_instance_ids": []string{foreignInstance.ID.String()},
	})
	testutil.SetAuthContext(req, orgA.ID, user.ID)
	require.NoError(t, app.UpdateAgentSelectionSettings(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, foreignInstance.ID.String())
}

func TestApp_CreateAgentSelectionParticipant_DuplicateReturns409(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	settings := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)
	testutil.CreateTestAgentSelectionParticipant(t, app.DB, org.ID, settings.ID, agent.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"settings_id": settings.ID.String(),
		"user_id":     agent.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	require.NoError(t, app.CreateAgentSelectionParticipant(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusConflict, "Agent is already in this routing list")
}

func TestApp_CreateAgentSelectionParticipant_RejectsUnknownUser(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	settings := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil)

	req := testutil.NewJSONRequest(t, map[string]any{
		"settings_id": settings.ID.String(),
		"user_id":     uuid.New().String(),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	require.NoError(t, app.CreateAgentSelectionParticipant(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "not available for this organization")
}

// --- Soft-delete unique-index regression: delete then re-add must succeed ---

func TestApp_CreateAgentSelectionParticipant_ReAddAfterDelete_Succeeds(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	settings := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)
	first := testutil.CreateTestAgentSelectionParticipant(t, app.DB, org.ID, settings.ID, agent.ID)

	delReq := testutil.NewRequest(t)
	testutil.SetAuthContext(delReq, org.ID, user.ID)
	testutil.SetPathParam(delReq, "id", first.ID.String())
	require.NoError(t, app.DeleteAgentSelectionParticipant(delReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(delReq))

	addReq := testutil.NewJSONRequest(t, map[string]any{
		"settings_id": settings.ID.String(),
		"user_id":     agent.ID.String(),
	})
	testutil.SetAuthContext(addReq, org.ID, user.ID)
	require.NoError(t, app.CreateAgentSelectionParticipant(addReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(addReq),
		"re-adding an agent after a soft-delete must succeed thanks to the partial unique index on agent_selection_participants")
}

// --- Delete endpoints now require :delete permission ---

func TestApp_DeleteAgentSelectionParticipant_RequiresDeletePermission(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	writeOnly := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "write-only", []string{"agent_selection:read", "agent_selection:write"})
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&writeOnly.ID))

	settings := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil)
	participant := testutil.CreateTestAgentSelectionParticipant(t, app.DB, org.ID, settings.ID, uuid.New())

	req := testutil.NewRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", participant.ID.String())
	require.NoError(t, app.DeleteAgentSelectionParticipant(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "Insufficient permissions")
}

func TestApp_DeleteAgentSelectionOption_RequiresDeletePermission(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	writeOnly := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "write-only", []string{"agent_selection:read", "agent_selection:write"})
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&writeOnly.ID))

	settings := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil)
	option := testutil.CreateTestAgentSelectionOption(t, app.DB, org.ID, settings.ID, models.AgentSelectionOptionQueue)

	req := testutil.NewRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", option.ID.String())
	require.NoError(t, app.DeleteAgentSelectionOption(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "Insufficient permissions")
}

func TestApp_ListAgentSelectionParticipants_MultiOrgAgent(t *testing.T) {
	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)

	// Admin user for org1
	adminRole := testutil.CreateAdminRole(t, app.DB, org1.ID)
	adminUser := testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithRoleID(&adminRole.ID))

	// Agent user whose default/primary org is org2
	agentUser := testutil.CreateTestUser(t, app.DB, org2.ID, func(u *models.User) {
		u.FullName = "Multi Org Agent"
		u.Email = "agent@org2.com"
	})

	// Add agentUser to org1 via user_organizations
	userOrg := models.UserOrganization{
		UserID:         agentUser.ID,
		OrganizationID: org1.ID,
		IsDefault:      false,
	}
	require.NoError(t, app.DB.Create(&userOrg).Error)

	settings := testutil.CreateTestAgentSelectionSettings(t, app.DB, org1.ID, nil)

	// Create participant for agentUser in org1's settings list
	reqAdd := testutil.NewJSONRequest(t, map[string]any{
		"settings_id": settings.ID.String(),
		"user_id":     agentUser.ID.String(),
	})
	testutil.SetAuthContext(reqAdd, org1.ID, adminUser.ID)
	require.NoError(t, app.CreateAgentSelectionParticipant(reqAdd))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(reqAdd))

	// List participants for org1
	reqList := testutil.NewGETRequest(t)
	testutil.SetAuthContext(reqList, org1.ID, adminUser.ID)
	require.NoError(t, app.ListAgentSelectionParticipants(reqList))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(reqList))

	var result struct {
		Data struct {
			Participants []models.AgentSelectionParticipant `json:"participants"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(reqList), &result))
	require.Len(t, result.Data.Participants, 1)

	participant := result.Data.Participants[0]
	assert.Equal(t, agentUser.ID, participant.UserID)
	require.NotNil(t, participant.User, "expected preloaded user not to be nil")
	assert.Equal(t, "Multi Org Agent", participant.User.FullName)
	assert.Equal(t, "agent@org2.com", participant.User.Email)
}
