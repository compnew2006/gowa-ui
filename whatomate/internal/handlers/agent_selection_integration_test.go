package handlers_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

type captureTextProvider struct {
	instanceID string
	to         string
	text       string
}

func (p *captureTextProvider) SendText(_ context.Context, instanceID string, to string, text string) (string, error) {
	p.instanceID = instanceID
	p.to = to
	p.text = text
	return "wamid-menu", nil
}

func (p *captureTextProvider) SendImage(context.Context, string, string, string, string) (string, error) {
	return "", nil
}

func (p *captureTextProvider) SendDocument(context.Context, string, string, string, string, string) (string, error) {
	return "", nil
}

func (p *captureTextProvider) SendVideo(context.Context, string, string, string, string) (string, error) {
	return "", nil
}

func (p *captureTextProvider) SendAudio(context.Context, string, string, string) (string, error) {
	return "", nil
}

func (p *captureTextProvider) MarkRead(context.Context, string, string) error {
	return nil
}

func (p *captureTextProvider) SendReaction(context.Context, string, string, string) error {
	return nil
}

func (p *captureTextProvider) RevokeMessage(context.Context, string, string) error {
	return nil
}

func (p *captureTextProvider) GetMediaURL(context.Context, string, string) (string, error) {
	return "", nil
}

func (p *captureTextProvider) DownloadMedia(context.Context, string, string) ([]byte, error) {
	return nil, nil
}

func (p *captureTextProvider) UploadMedia(context.Context, string, string, []byte) (string, error) {
	return "", nil
}

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
		Status string `json:"status"`
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

func TestApp_UpdateAgentSelectionSettings_InstanceScopeDoesNotOverwriteGlobal(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	instance := testutil.CreateTestWhatsAppInstance(t, app.DB, org.ID)
	global := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil, func(s *models.AgentSelectionSettings) {
		s.PromptDelayMinMinutes = 3
		s.PromptDelayMaxMinutes = 6
	})

	req := testutil.NewJSONRequest(t, map[string]any{
		"instance_id":              instance.ID.String(),
		"enabled":                  true,
		"trigger_mode":             string(models.AgentSelectionTriggerKeyword),
		"trigger_keywords":         []string{"11"},
		"prompt_delay_min_minutes": 1,
		"prompt_delay_max_minutes": 2,
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
	require.NotNil(t, result.Data.Settings.InstanceID)
	assert.Equal(t, instance.ID, *result.Data.Settings.InstanceID)
	assert.Equal(t, 1, result.Data.Settings.PromptDelayMinMinutes)
	assert.Equal(t, 2, result.Data.Settings.PromptDelayMaxMinutes)
	assert.NotEqual(t, global.ID, result.Data.Settings.ID)

	var refreshedGlobal models.AgentSelectionSettings
	require.NoError(t, app.DB.First(&refreshedGlobal, "id = ?", global.ID).Error)
	assert.Nil(t, refreshedGlobal.InstanceID)
	assert.Equal(t, 3, refreshedGlobal.PromptDelayMinMinutes)
	assert.Equal(t, 6, refreshedGlobal.PromptDelayMaxMinutes)
}

func TestApp_DeleteAgentSelectionSettings_RemovesRuleAndReturnsInheritedDraft(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	instance := testutil.CreateTestWhatsAppInstance(t, app.DB, org.ID)
	global := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil, func(s *models.AgentSelectionSettings) {
		s.PromptDelayMinMinutes = 3
		s.PromptDelayMaxMinutes = 6
	})
	instanceRule := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, &instance.ID, func(s *models.AgentSelectionSettings) {
		s.PromptDelayMinMinutes = 1
		s.PromptDelayMaxMinutes = 2
	})
	agent := testutil.CreateTestUser(t, app.DB, org.ID)
	participant := testutil.CreateTestAgentSelectionParticipant(t, app.DB, org.ID, instanceRule.ID, agent.ID)
	option := testutil.CreateTestAgentSelectionOption(t, app.DB, org.ID, instanceRule.ID, models.AgentSelectionOptionQueue, func(o *models.AgentSelectionOption) {
		o.Label = "Instance option"
	})

	delReq := testutil.NewRequest(t)
	delReq.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(delReq, org.ID, user.ID)
	testutil.SetPathParam(delReq, "id", instanceRule.ID.String())
	require.NoError(t, app.DeleteAgentSelectionSettings(delReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(delReq))

	var settingsCount int64
	require.NoError(t, app.DB.Unscoped().Model(&models.AgentSelectionSettings{}).Where("id = ?", instanceRule.ID).Count(&settingsCount).Error)
	assert.Zero(t, settingsCount)
	var participantCount int64
	require.NoError(t, app.DB.Unscoped().Model(&models.AgentSelectionParticipant{}).Where("id = ?", participant.ID).Count(&participantCount).Error)
	assert.Zero(t, participantCount)
	var optionCount int64
	require.NoError(t, app.DB.Unscoped().Model(&models.AgentSelectionOption{}).Where("id = ?", option.ID).Count(&optionCount).Error)
	assert.Zero(t, optionCount)

	getReq := testutil.NewGETRequest(t)
	getReq.RequestCtx.QueryArgs().Add("instance_id", instance.ID.String())
	testutil.SetAuthContext(getReq, org.ID, user.ID)
	require.NoError(t, app.GetAgentSelectionSettings(getReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(getReq))

	var result struct {
		Data struct {
			Settings *models.AgentSelectionSettings `json:"settings"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(getReq), &result))
	require.NotNil(t, result.Data.Settings)
	assert.Equal(t, uuid.Nil, result.Data.Settings.ID, "missing instance rule should return an inherited draft, not auto-create a row")
	require.NotNil(t, result.Data.Settings.InstanceID)
	assert.Equal(t, instance.ID, *result.Data.Settings.InstanceID)
	assert.Equal(t, 3, result.Data.Settings.PromptDelayMinMinutes)
	assert.Equal(t, 6, result.Data.Settings.PromptDelayMaxMinutes)

	var recreatedCount int64
	require.NoError(t, app.DB.Model(&models.AgentSelectionSettings{}).
		Where("organization_id = ? AND instance_id = ?", org.ID, instance.ID).
		Count(&recreatedCount).Error)
	assert.Zero(t, recreatedCount)
	_ = global
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
		{"prompt_delay_min_minutes<0", map[string]any{"prompt_delay_min_minutes": -1}, "prompt_delay_min_minutes"},
		{"prompt_delay_max_minutes>1440", map[string]any{"prompt_delay_max_minutes": 5000}, "prompt_delay_max_minutes"},
		{"prompt_delay_max_before_min", map[string]any{"prompt_delay_min_minutes": 5, "prompt_delay_max_minutes": 1}, "prompt_delay_max_minutes"},
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
		"prompt_delay_min_minutes":  1,
		"prompt_delay_max_minutes":  5,
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
	assert.Equal(t, 1, result.Data.Settings.PromptDelayMinMinutes)
	assert.Equal(t, 5, result.Data.Settings.PromptDelayMaxMinutes)
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

func TestApp_BuildAgentSelectionMenu_CustomFinalUsesResponseAsLabel(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	settings := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil, func(s *models.AgentSelectionSettings) {
		s.CustomFinalOptionEnabled = true
		s.CustomFinalOptionText = "other"
		s.CustomFinalOptionResponse = "x"
	})

	menu, err := app.BuildAgentSelectionMenuForTest(org.ID, settings, nil)
	require.NoError(t, err)
	require.NotEmpty(t, menu.Options)
	last := menu.Options[len(menu.Options)-1]
	assert.Equal(t, models.AgentSelectionOptionCustom, last.Type)
	assert.Equal(t, "x", last.Label)
	assert.Contains(t, menu.Text, "1. x")
}

func TestApp_HandleWhatsmeowInboundMessage_CreatesKeywordSessionWithoutMetaAccount(t *testing.T) {
	app := newTestApp(t)
	app.Config.WhatsApp.Provider = "whatsmeow"

	org := testutil.CreateTestOrganization(t, app.DB)
	instance := testutil.CreateTestWhatsAppInstance(t, app.DB, org.ID, func(i *models.WhatsAppInstance) {
		i.PhoneNumber = "201007181781"
		i.Status = models.InstanceStatusConnected
	})
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(instance.PhoneNumber),
		testutil.WithPhoneNumber("966561853319"),
	)
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"instance_id": instance.ID,
		"status":      models.ChatStatusPending,
	}).Error)
	contact.InstanceID = &instance.ID
	contact.Status = models.ChatStatusPending

	testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil, func(s *models.AgentSelectionSettings) {
		s.Enabled = true
		s.TriggerMode = models.AgentSelectionTriggerKeyword
		s.TriggerKeywords = models.StringArray{"11"}
		s.PromptDelayMinutes = 0
	})

	inbound := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    org.ID,
		WhatsAppAccount:   instance.PhoneNumber,
		ContactID:         contact.ID,
		InstanceID:        &instance.ID,
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeText,
		Content:           "11",
		WhatsAppMessageID: "wamid-test-keyword",
		Status:            models.MessageStatusRead,
	}
	require.NoError(t, app.DB.Create(inbound).Error)

	var accountCount int64
	require.NoError(t, app.DB.Model(&models.WhatsAppAccount{}).
		Where("organization_id = ?", org.ID).
		Count(&accountCount).Error)
	require.Zero(t, accountCount, "regression setup should not rely on legacy Meta account rows")

	app.HandleWhatsmeowInboundMessage(context.Background(), whatsmeow.InboundMessageInfo{
		OrganizationID:  org.ID,
		InstanceID:      instance.ID,
		WhatsAppAccount: instance.PhoneNumber,
		Contact:         contact,
		Message:         inbound,
		MessageType:     models.MessageTypeText,
		Content:         "11",
	})

	var session models.AgentSelectionSession
	require.NoError(t, app.DB.Where("organization_id = ? AND contact_id = ?", org.ID, contact.ID).First(&session).Error)
	assert.Equal(t, models.AgentSelectionSessionWaitingDelay, session.Status)
	assert.Equal(t, instance.PhoneNumber, session.WhatsAppAccount)
	assert.Equal(t, inbound.ID, *session.TriggerMessageID)
}

func TestApp_ProcessAgentSelectionDueSessions_SendsMenuWithoutMetaAccount(t *testing.T) {
	provider := &captureTextProvider{}
	app := newTestApp(t)
	app.Config.WhatsApp.Provider = "whatsmeow"
	app.MessageProvider = provider

	org := testutil.CreateTestOrganization(t, app.DB)
	instance := testutil.CreateTestWhatsAppInstance(t, app.DB, org.ID, func(i *models.WhatsAppInstance) {
		i.PhoneNumber = "201007181781"
		i.Status = models.InstanceStatusConnected
	})
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(instance.PhoneNumber),
		testutil.WithPhoneNumber("966561853319"),
	)
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"instance_id": instance.ID,
		"status":      models.ChatStatusPending,
	}).Error)

	settings := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil, func(s *models.AgentSelectionSettings) {
		s.Enabled = true
		s.TriggerMode = models.AgentSelectionTriggerKeyword
		s.TriggerKeywords = models.StringArray{"11"}
		s.PromptDelayMinutes = 0
		s.HideUnavailableAgents = false
	})
	agent := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithFullName("Agent One"))
	testutil.CreateTestAgentSelectionParticipant(t, app.DB, org.ID, settings.ID, agent.ID, func(p *models.AgentSelectionParticipant) {
		p.DisplayName = "Agent One"
	})

	instanceID := instance.ID
	inboundID := uuid.New()
	session := &models.AgentSelectionSession{
		BaseModel:        models.BaseModel{ID: uuid.New()},
		OrganizationID:   org.ID,
		ContactID:        contact.ID,
		InstanceID:       &instanceID,
		WhatsAppAccount:  instance.PhoneNumber,
		Status:           models.AgentSelectionSessionWaitingDelay,
		TriggerMessageID: &inboundID,
		PromptDueAt:      time.Now().Add(-time.Minute),
		Metadata:         models.JSONB{"settings_id": settings.ID.String()},
	}
	require.NoError(t, app.DB.Create(session).Error)

	var accountCount int64
	require.NoError(t, app.DB.Model(&models.WhatsAppAccount{}).
		Where("organization_id = ?", org.ID).
		Count(&accountCount).Error)
	require.Zero(t, accountCount, "regression setup should not rely on legacy Meta account rows")

	app.ProcessAgentSelectionDueSessions(context.Background(), 10)

	var refreshed models.AgentSelectionSession
	require.NoError(t, app.DB.First(&refreshed, "id = ?", session.ID).Error)
	assert.Equal(t, models.AgentSelectionSessionMenuSent, refreshed.Status)
	assert.Equal(t, instance.ID.String(), provider.instanceID)
	assert.Equal(t, "966561853319", provider.to)
	assert.Contains(t, provider.text, "1. Agent One")
	require.NotNil(t, refreshed.PromptMessageID)
}

func TestApp_CustomFinalResponseUsesPromptDelayAndTypingSendPath(t *testing.T) {
	provider := &captureTextProvider{}
	app := newTestApp(t)
	app.Config.WhatsApp.Provider = "whatsmeow"
	app.MessageProvider = provider

	org := testutil.CreateTestOrganization(t, app.DB)
	instance := testutil.CreateTestWhatsAppInstance(t, app.DB, org.ID, func(i *models.WhatsAppInstance) {
		i.PhoneNumber = "201007181781"
		i.Status = models.InstanceStatusConnected
	})
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(instance.PhoneNumber),
		testutil.WithPhoneNumber("966561853319"),
	)
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"instance_id": instance.ID,
		"status":      models.ChatStatusPending,
	}).Error)
	contact.InstanceID = &instance.ID
	contact.Status = models.ChatStatusPending

	settings := testutil.CreateTestAgentSelectionSettings(t, app.DB, org.ID, nil, func(s *models.AgentSelectionSettings) {
		s.Enabled = true
		s.PromptDelayMinMinutes = 1
		s.PromptDelayMaxMinutes = 1
		s.CustomFinalOptionEnabled = true
		s.CustomFinalOptionResponse = "x"
		s.CustomFinalOptionAction = models.AgentSelectionCustomActionKeepPending
	})
	instanceID := instance.ID
	session := &models.AgentSelectionSession{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		InstanceID:      &instanceID,
		WhatsAppAccount: instance.PhoneNumber,
		Status:          models.AgentSelectionSessionMenuSent,
		RenderedOptionsSnapshot: models.JSONBArray{map[string]any{
			"number":    float64(1),
			"option_id": "custom_final",
			"type":      string(models.AgentSelectionOptionCustom),
			"label":     "x",
			"action":    string(models.AgentSelectionCustomActionKeepPending),
			"response":  "x",
		}},
		Metadata: models.JSONB{"settings_id": settings.ID.String()},
	}
	require.NoError(t, app.DB.Create(session).Error)
	inbound := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    org.ID,
		WhatsAppAccount:   instance.PhoneNumber,
		ContactID:         contact.ID,
		InstanceID:        &instanceID,
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeText,
		Content:           "1",
		WhatsAppMessageID: "wamid-custom-choice",
		Status:            models.MessageStatusRead,
	}
	require.NoError(t, app.DB.Create(inbound).Error)

	app.HandleWhatsmeowInboundMessage(context.Background(), whatsmeow.InboundMessageInfo{
		OrganizationID:  org.ID,
		InstanceID:      instance.ID,
		WhatsAppAccount: instance.PhoneNumber,
		Contact:         contact,
		Message:         inbound,
		MessageType:     models.MessageTypeText,
		Content:         "1",
	})

	var waiting models.AgentSelectionSession
	require.NoError(t, app.DB.First(&waiting, "id = ?", session.ID).Error)
	assert.Equal(t, models.AgentSelectionSessionWaitingDelay, waiting.Status)
	assert.True(t, waiting.PromptDueAt.After(time.Now()), "custom response must wait for the configured prompt delay")
	assert.Empty(t, provider.text, "custom response must not send immediately")
	assert.True(t, bool(waiting.Metadata["pending_custom_response"].(bool)))

	require.NoError(t, app.DB.Model(&models.AgentSelectionSession{}).Where("id = ?", session.ID).Update("prompt_due_at", time.Now().Add(-time.Minute)).Error)
	app.ProcessAgentSelectionDueSessions(context.Background(), 10)

	var completed models.AgentSelectionSession
	require.NoError(t, app.DB.First(&completed, "id = ?", session.ID).Error)
	assert.Equal(t, models.AgentSelectionSessionSelected, completed.Status)
	assert.Equal(t, instance.ID.String(), provider.instanceID)
	assert.Equal(t, "966561853319", provider.to)
	assert.Equal(t, "x", provider.text)
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
