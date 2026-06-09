package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sendStepTestApp(t *testing.T) (*App, *[]map[string]interface{}) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	log := testutil.NopLogger()

	var received []map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		received = append(received, map[string]interface{}{
			"path":   r.URL.Path,
			"method": r.Method,
			"body":   body,
		})
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"messages": []map[string]string{{"id": "wamid.step_test_" + uuid.NewString()[:8]}},
		})
	}))
	t.Cleanup(server.Close)

	app := &App{
		Config: &config.Config{
			App: config.AppConfig{EncryptionKey: testutil.TestEncryptionKey},
		},
		DB:       db,
		Log:      log,
		WhatsApp: whatsapp.NewWithBaseURL(log, server.URL),
	}
	if rdb := testutil.SetupTestRedis(t); rdb != nil {
		app.Redis = rdb
	}

	return app, &received
}

func makeStepTestFixtures(t *testing.T, app *App) (*models.Organization, *models.WhatsAppAccount, *models.Contact, *models.ChatbotSession) {
	t.Helper()

	org := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	session := &models.ChatbotSession{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: account.Name,
		PhoneNumber:     contact.PhoneNumber,
		Status:          models.SessionStatusActive,
		SessionData:     models.JSONB{},
		StartedAt:       time.Now(),
		LastActivityAt:  time.Now(),
	}
	require.NoError(t, app.DB.Create(session).Error)

	return org, account, contact, session
}

func TestSendStepMessage_TextStep_SendsProcessedText(t *testing.T) {
	app, sent := sendStepTestApp(t)
	_, account, contact, session := makeStepTestFixtures(t, app)

	session.SessionData = models.JSONB{"name": "Alice"}
	require.NoError(t, app.DB.Model(session).Update("session_data", session.SessionData).Error)

	step := &models.ChatbotFlowStep{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		StepName:    "greet",
		StepOrder:   1,
		Message:     "Hello {{name}}, welcome!",
		MessageType: models.FlowStepTypeText,
	}

	app.sendStepMessage(account, session, contact, step)

	require.Len(t, *sent, 1)
	body := (*sent)[0]["body"].(map[string]interface{})
	text := body["text"].(map[string]interface{})
	assert.Contains(t, text["body"], "Hello Alice, welcome!")

	var msgs []models.ChatbotSessionMessage
	require.NoError(t, app.DB.Where("session_id = ?", session.ID).Find(&msgs).Error)
	require.Len(t, msgs, 1)
	assert.Equal(t, models.DirectionOutgoing, msgs[0].Direction)
	assert.Contains(t, msgs[0].Message, "Hello Alice")
}

func TestSendStepMessage_TextStep_SendFailureStillLogsSessionMessage(t *testing.T) {
	db := testutil.SetupTestDB(t)
	log := testutil.NopLogger()

	waServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "send failed")
	}))
	t.Cleanup(waServer.Close)

	app := &App{
		Config:   &config.Config{App: config.AppConfig{EncryptionKey: testutil.TestEncryptionKey}},
		DB:       db,
		Log:      log,
		WhatsApp: whatsapp.NewWithBaseURL(log, waServer.URL),
	}
	if rdb := testutil.SetupTestRedis(t); rdb != nil {
		app.Redis = rdb
	}

	_, account, contact, session := makeStepTestFixtures(t, app)
	step := &models.ChatbotFlowStep{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		StepName:    "send_failure",
		StepOrder:   1,
		Message:     "This should still be logged",
		MessageType: models.FlowStepTypeText,
	}

	app.sendStepMessage(account, session, contact, step)

	var msgs []models.ChatbotSessionMessage
	require.NoError(t, app.DB.Where("session_id = ?", session.ID).Find(&msgs).Error)
	require.Len(t, msgs, 1)
	assert.Equal(t, "This should still be logged", msgs[0].Message)
}

func TestSendStepMessage_ButtonsStep_SendsInteractive(t *testing.T) {
	app, sent := sendStepTestApp(t)
	_, account, contact, session := makeStepTestFixtures(t, app)

	step := &models.ChatbotFlowStep{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		StepName:    "choose",
		StepOrder:   1,
		Message:     "Pick an option",
		MessageType: models.FlowStepTypeButtons,
		Buttons: models.JSONBArray{
			map[string]interface{}{"id": "btn_1", "title": "Option A"},
			map[string]interface{}{"id": "btn_2", "title": "Option B"},
		},
	}

	app.sendStepMessage(account, session, contact, step)

	require.Len(t, *sent, 1)
	body := (*sent)[0]["body"].(map[string]interface{})
	assert.NotNil(t, body["interactive"])
	interactive := body["interactive"].(map[string]interface{})
	assert.Equal(t, "button", interactive["type"])

	var msgs []models.ChatbotSessionMessage
	require.NoError(t, app.DB.Where("session_id = ?", session.ID).Find(&msgs).Error)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0].Message, "Pick an option")
}

func TestSendStepMessage_ButtonsStep_NoButtons_FallsBackToText(t *testing.T) {
	app, sent := sendStepTestApp(t)
	_, account, contact, session := makeStepTestFixtures(t, app)

	step := &models.ChatbotFlowStep{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		StepName:    "empty_buttons",
		StepOrder:   1,
		Message:     "No buttons configured",
		MessageType: models.FlowStepTypeButtons,
		Buttons:     models.JSONBArray{},
	}

	app.sendStepMessage(account, session, contact, step)

	require.Len(t, *sent, 1)
	body := (*sent)[0]["body"].(map[string]interface{})
	assert.NotNil(t, body["text"])
}

func TestSendStepMessage_TransferStep_ExitsFlow(t *testing.T) {
	app, _ := sendStepTestApp(t)
	_, account, contact, session := makeStepTestFixtures(t, app)

	flow := &models.ChatbotFlow{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  account.OrganizationID,
		WhatsAppAccount: account.Name,
		Name:            "Transfer Flow",
		IsEnabled:       true,
		Steps:           []models.ChatbotFlowStep{},
	}
	require.NoError(t, app.DB.Create(flow).Error)
	session.CurrentFlowID = &flow.ID
	session.CurrentStep = "transfer_step"
	require.NoError(t, app.DB.Model(session).Updates(map[string]interface{}{
		"current_flow_id": flow.ID,
		"current_step":    "transfer_step",
	}).Error)

	step := &models.ChatbotFlowStep{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		StepName:    "transfer_step",
		StepOrder:   1,
		Message:     "Transferring you to an agent",
		MessageType: models.FlowStepTypeTransfer,
		TransferConfig: models.JSONB{
			"team_id": "_general",
			"notes":   "User requested transfer",
		},
	}

	app.sendStepMessage(account, session, contact, step)

	var updated models.ChatbotSession
	require.NoError(t, app.DB.First(&updated, session.ID).Error)
	assert.Equal(t, models.SessionStatusCompleted, updated.Status)
	assert.Empty(t, updated.CurrentStep)
}

func TestSendStepMessage_TransferStep_WithTeamID_CreatesTeamTransfer(t *testing.T) {
	app, _ := sendStepTestApp(t)
	org, account, contact, session := makeStepTestFixtures(t, app)

	teamID := uuid.New()
	team := &models.Team{
		BaseModel:          models.BaseModel{ID: teamID},
		OrganizationID:     org.ID,
		Name:               "Support Team",
		AssignmentStrategy: "manual",
	}
	require.NoError(t, app.DB.Create(team).Error)

	flow := &models.ChatbotFlow{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: account.Name,
		Name:            "Team Transfer Flow",
		IsEnabled:       true,
		Steps:           []models.ChatbotFlowStep{},
	}
	require.NoError(t, app.DB.Create(flow).Error)
	session.CurrentFlowID = &flow.ID
	session.CurrentStep = "team_transfer"
	require.NoError(t, app.DB.Model(session).Updates(map[string]interface{}{
		"current_flow_id": flow.ID,
		"current_step":    "team_transfer",
	}).Error)

	step := &models.ChatbotFlowStep{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		StepName:    "team_transfer",
		StepOrder:   1,
		Message:     "Connecting you to support",
		MessageType: models.FlowStepTypeTransfer,
		TransferConfig: models.JSONB{
			"team_id": teamID.String(),
			"notes":   "Needs help",
		},
	}

	app.sendStepMessage(account, session, contact, step)

	var updated models.ChatbotSession
	require.NoError(t, app.DB.First(&updated, session.ID).Error)
	assert.Equal(t, models.SessionStatusCompleted, updated.Status)

	var transfer models.AgentTransfer
	require.NoError(t, app.DB.Where("contact_id = ? AND organization_id = ?", contact.ID, org.ID).First(&transfer).Error)
	assert.Equal(t, models.TransferStatusActive, transfer.Status)
	require.NotNil(t, transfer.TeamID)
	assert.Equal(t, teamID, *transfer.TeamID)
	assert.Equal(t, models.TransferSourceFlow, transfer.Source)
}

func TestSendStepMessage_APIFetchSuccess_SendsProcessedMessage(t *testing.T) {
	db := testutil.SetupTestDB(t)
	log := testutil.NopLogger()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"message": "Your order #12345 is shipped",
			"buttons": []map[string]interface{}{
				{"id": "track", "value": "Track Order"},
			},
		})
	}))
	t.Cleanup(apiServer.Close)

	waServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"messages": []map[string]string{{"id": "wamid.api_test"}},
		})
	}))
	t.Cleanup(waServer.Close)

	app := &App{
		Config:     &config.Config{App: config.AppConfig{EncryptionKey: testutil.TestEncryptionKey}},
		DB:         db,
		Log:        log,
		WhatsApp:   whatsapp.NewWithBaseURL(log, waServer.URL),
		HTTPClient: apiServer.Client(),
	}
	if rdb := testutil.SetupTestRedis(t); rdb != nil {
		app.Redis = rdb
	}

	_, account, contact, session := makeStepTestFixtures(t, app)

	step := &models.ChatbotFlowStep{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		StepName:    "check_order",
		StepOrder:   1,
		Message:     "Order status: {{api_status}}",
		MessageType: models.FlowStepTypeAPIFetch,
		ApiConfig: models.JSONB{
			"url":    apiServer.URL + "/orders/12345",
			"method": "GET",
			"response_mapping": map[string]interface{}{
				"api_status": "status",
			},
		},
	}

	app.sendStepMessage(account, session, contact, step)

	var updated models.ChatbotSession
	require.NoError(t, app.DB.First(&updated, session.ID).Error)
	assert.Equal(t, "ok", updated.SessionData["api_status"])

	var msgs []models.ChatbotSessionMessage
	require.NoError(t, app.DB.Where("session_id = ?", session.ID).Find(&msgs).Error)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0].Message, "ok")
}

func TestSendStepMessage_APIFetchFailure_UsesFallbackMessage(t *testing.T) {
	db := testutil.SetupTestDB(t)
	log := testutil.NopLogger()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "internal server error")
	}))
	t.Cleanup(apiServer.Close)

	waServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"messages": []map[string]string{{"id": "wamid.api_fail_test"}},
		})
	}))
	t.Cleanup(waServer.Close)

	app := &App{
		Config:     &config.Config{App: config.AppConfig{EncryptionKey: testutil.TestEncryptionKey}},
		DB:         db,
		Log:        log,
		WhatsApp:   whatsapp.NewWithBaseURL(log, waServer.URL),
		HTTPClient: apiServer.Client(),
	}
	if rdb := testutil.SetupTestRedis(t); rdb != nil {
		app.Redis = rdb
	}

	_, account, contact, session := makeStepTestFixtures(t, app)

	step := &models.ChatbotFlowStep{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		StepName:    "failing_api",
		StepOrder:   1,
		Message:     "Default message",
		MessageType: models.FlowStepTypeAPIFetch,
		ApiConfig: models.JSONB{
			"url":              apiServer.URL + "/fail",
			"fallback_message": "Service temporarily unavailable",
		},
	}

	app.sendStepMessage(account, session, contact, step)

	var msgs []models.ChatbotSessionMessage
	require.NoError(t, app.DB.Where("session_id = ?", session.ID).Find(&msgs).Error)
	require.Len(t, msgs, 1)
	assert.Equal(t, "Service temporarily unavailable", msgs[0].Message)
}

func TestSendStepMessage_APIFetchFailure_NoFallback_UsesStepMessage(t *testing.T) {
	db := testutil.SetupTestDB(t)
	log := testutil.NopLogger()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(apiServer.Close)

	waServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"messages": []map[string]string{{"id": "wamid.api_nofb_test"}},
		})
	}))
	t.Cleanup(waServer.Close)

	app := &App{
		Config:     &config.Config{App: config.AppConfig{EncryptionKey: testutil.TestEncryptionKey}},
		DB:         db,
		Log:        log,
		WhatsApp:   whatsapp.NewWithBaseURL(log, waServer.URL),
		HTTPClient: apiServer.Client(),
	}
	if rdb := testutil.SetupTestRedis(t); rdb != nil {
		app.Redis = rdb
	}

	_, account, contact, session := makeStepTestFixtures(t, app)

	step := &models.ChatbotFlowStep{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		StepName:    "no_fallback",
		StepOrder:   1,
		Message:     "Sorry, something went wrong.",
		MessageType: models.FlowStepTypeAPIFetch,
		ApiConfig: models.JSONB{
			"url": apiServer.URL + "/fail",
		},
	}

	app.sendStepMessage(account, session, contact, step)

	var msgs []models.ChatbotSessionMessage
	require.NoError(t, app.DB.Where("session_id = ?", session.ID).Find(&msgs).Error)
	require.Len(t, msgs, 1)
	assert.Equal(t, "Sorry, something went wrong.", msgs[0].Message)
}

func TestSendStepMessage_WhatsAppFlowStep_MissingFlowID_FallsBackToText(t *testing.T) {
	app, sent := sendStepTestApp(t)
	_, account, contact, session := makeStepTestFixtures(t, app)

	step := &models.ChatbotFlowStep{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		StepName:    "flow_no_id",
		StepOrder:   1,
		Message:     "Fallback text",
		MessageType: models.FlowStepTypeWhatsAppFlow,
		InputConfig: models.JSONB{},
	}

	app.sendStepMessage(account, session, contact, step)

	require.Len(t, *sent, 1)
	body := (*sent)[0]["body"].(map[string]interface{})
	assert.NotNil(t, body["text"])
}
