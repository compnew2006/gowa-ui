package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/crypto"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/internal/queue"
	"github.com/compnew2006/gowa-ui/internal/templateutil"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testWorker(t *testing.T) *Worker {
	t.Helper()
	db := testutil.SetupTestDB(t)
	log := testutil.NopLogger()

	w := &Worker{
		DB:         db,
		Log:        log,
		WARegistry: whatsapp.NewRegistry(log),
	}

	// Set up Publisher if Redis is available
	if rdb := testutil.SetupTestRedis(t); rdb != nil {
		w.Redis = rdb
		w.Publisher = queue.NewPublisher(rdb, log)
	}

	return w
}

// pointWorkerAtGowa registers a GOWA client factory and gives the worker a
// fresh registry, so accounts whose gowa_base_url points at a mock server
// resolve a client that talks to it. RegisterGowaFactory is process-global —
// this wrapper passes its own per-test closures + w.Log to
// NewRegistryWithFactory rather than sharing a hard-coded default.
func pointWorkerAtGowa(w *Worker) {
	w.WARegistry = whatsapp.NewRegistryWithFactory(
		w.Log,
		func(_ uuid.UUID, baseURL string) (string, string) { return "", "" },
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(baseURL, username, password)
		},
	)
}

// newGowaSendServer returns a mock GOWA server that captures the last request
// body and responds with a successful send result carrying msgID.
func newGowaSendServer(msgID string, capturedBody *map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if capturedBody != nil {
			_ = json.NewDecoder(r.Body).Decode(capturedBody)
		}
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(map[string]any{
			"code":    "SUCCESS",
			"message": "Success",
			"results": map[string]any{"message_id": msgID, "status": "ok"},
		})
	}))
}

// getOrCreateTestPermissions gets existing permissions or creates them for testing.
func getOrCreateTestPermissions(t *testing.T, w *Worker) []models.Permission {
	t.Helper()

	var existingPerms []models.Permission
	if err := w.DB.Order("resource, action").Find(&existingPerms).Error; err == nil && len(existingPerms) > 0 {
		return existingPerms
	}

	// Create all default permissions if none exist
	perms := models.DefaultPermissions()
	for i := range perms {
		perms[i].ID = uuid.New()
	}
	require.NoError(t, w.DB.Create(&perms).Error)
	return perms
}

// createTestRole creates an admin role with all permissions for testing.
func createTestRole(t *testing.T, w *Worker, orgID uuid.UUID) *models.CustomRole {
	t.Helper()

	// Get or create permissions
	perms := getOrCreateTestPermissions(t, w)

	role := &models.CustomRole{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           "admin_" + uuid.New().String()[:8],
		Description:    "Test admin role",
		IsSystem:       false,
		IsDefault:      false,
		Permissions:    perms,
	}
	require.NoError(t, w.DB.Create(role).Error)
	return role
}

func createTestCampaignData(t *testing.T, w *Worker) (*models.Organization, *models.WhatsAppAccount, *models.Template, *models.BulkMessageCampaign, *models.BulkMessageRecipient) {
	t.Helper()

	uniqueID := uuid.New().String()[:8]

	// Create organization
	org := &models.Organization{
		Name: "Test Org " + uniqueID,
		Slug: "test-org-" + uniqueID,
	}
	require.NoError(t, w.DB.Create(org).Error)

	// Create role for user
	role := createTestRole(t, w, org.ID)

	// Create user for CreatedBy foreign key
	user := &models.User{
		OrganizationID: org.ID,
		Email:          "test-" + uniqueID + "@example.com",
		PasswordHash:   "hashed",
		FullName:       "Test User",
		RoleID:         &role.ID,
		IsActive:       true,
	}
	require.NoError(t, w.DB.Create(user).Error)

	// Create WhatsApp account with unique name
	accountName := "test-account-" + uniqueID
	account := &models.WhatsAppAccount{
		OrganizationID: org.ID,
		Name:           accountName,
		GowaBaseURL:    "http://gowa.test:3000",
		GowaDeviceID:   "device-" + uniqueID,
	}
	require.NoError(t, w.DB.Create(account).Error)

	// Create template
	template := &models.Template{
		OrganizationID:  org.ID,
		WhatsAppAccount: accountName,
		Name:            "test_template_" + uniqueID,
		Language:        "en",
		Category:        "MARKETING",
		BodyContent:     "Hello {{1}}, your order {{2}} is ready!",
	}
	require.NoError(t, w.DB.Create(template).Error)

	// Create campaign with CreatedBy
	campaign := &models.BulkMessageCampaign{
		OrganizationID:  org.ID,
		Name:            "Test Campaign " + uniqueID,
		WhatsAppAccount: accountName,
		TemplateID:      template.ID,
		Status:          models.CampaignStatusProcessing,
		TotalRecipients: 1,
		CreatedBy:       user.ID,
	}
	require.NoError(t, w.DB.Create(campaign).Error)

	// Create recipient
	recipient := &models.BulkMessageRecipient{
		CampaignID:    campaign.ID,
		PhoneNumber:   "1112223333",
		RecipientName: "Test User",
		Status:        models.MessageStatusPending,
		TemplateParams: models.JSONB{
			"1": "John",
			"2": "ORD-123",
		},
	}
	require.NoError(t, w.DB.Create(recipient).Error)

	// Reload campaign with template
	require.NoError(t, w.DB.Preload("Template").First(campaign, campaign.ID).Error)

	return org, account, template, campaign, recipient
}

func TestWorker_HandleRecipientJob_CampaignPaused(t *testing.T) {
	w := testWorker(t)
	org, _, _, campaign, recipient := createTestCampaignData(t, w)

	// Pause the campaign
	require.NoError(t, w.DB.Model(campaign).Update("status", models.CampaignStatusPaused).Error)

	job := &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    recipient.PhoneNumber,
		RecipientName:  recipient.RecipientName,
	}

	err := w.HandleRecipientJob(context.Background(), job)
	require.NoError(t, err)

	// Recipient status should remain pending (job was skipped)
	var updatedRecipient models.BulkMessageRecipient
	require.NoError(t, w.DB.First(&updatedRecipient, recipient.ID).Error)
	assert.Equal(t, models.MessageStatusPending, updatedRecipient.Status)
}

func TestWorker_HandleRecipientJob_CampaignCancelled(t *testing.T) {
	w := testWorker(t)
	org, _, _, campaign, recipient := createTestCampaignData(t, w)

	// Cancel the campaign
	require.NoError(t, w.DB.Model(campaign).Update("status", models.CampaignStatusCancelled).Error)

	job := &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    recipient.PhoneNumber,
		RecipientName:  recipient.RecipientName,
	}

	err := w.HandleRecipientJob(context.Background(), job)
	require.NoError(t, err)

	// Recipient status should remain pending (job was skipped)
	var updatedRecipient models.BulkMessageRecipient
	require.NoError(t, w.DB.First(&updatedRecipient, recipient.ID).Error)
	assert.Equal(t, models.MessageStatusPending, updatedRecipient.Status)
}

func TestWorker_HandleRecipientJob_AccountNotFound(t *testing.T) {
	w := testWorker(t)
	org, _, _, campaign, recipient := createTestCampaignData(t, w)

	// Change campaign to use non-existent account
	campaign.WhatsAppAccount = "non-existent-account"
	require.NoError(t, w.DB.Save(campaign).Error)

	job := &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    recipient.PhoneNumber,
		RecipientName:  recipient.RecipientName,
	}

	err := w.HandleRecipientJob(context.Background(), job)
	require.NoError(t, err)

	// Verify recipient marked as failed
	var updatedRecipient models.BulkMessageRecipient
	require.NoError(t, w.DB.First(&updatedRecipient, recipient.ID).Error)
	assert.Equal(t, models.MessageStatusFailed, updatedRecipient.Status)
	assert.Contains(t, updatedRecipient.ErrorMessage, "WhatsApp account not found")
}

func TestWorker_HandleRecipientJob_CampaignNotFound(t *testing.T) {
	w := testWorker(t)

	job := &queue.RecipientJob{
		CampaignID:     uuid.New(), // Non-existent campaign
		RecipientID:    uuid.New(),
		OrganizationID: uuid.New(),
		PhoneNumber:    "1234567890",
		RecipientName:  "Test",
	}

	err := w.HandleRecipientJob(context.Background(), job)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load campaign")
}

// createMinimalCampaignData creates the minimum data needed for campaign tests
// Returns org, user, template, and campaign
func createMinimalCampaignData(t *testing.T, w *Worker, status models.CampaignStatus) (*models.Organization, *models.User, *models.Template, *models.BulkMessageCampaign) {
	t.Helper()
	uniqueID := uuid.New().String()[:8]

	org := &models.Organization{
		Name: "Test Org " + uniqueID,
		Slug: "test-org-" + uniqueID,
	}
	require.NoError(t, w.DB.Create(org).Error)

	// Create role for user
	role := createTestRole(t, w, org.ID)

	user := &models.User{
		OrganizationID: org.ID,
		Email:          "test-" + uniqueID + "@example.com",
		PasswordHash:   "hashed",
		FullName:       "Test User",
		RoleID:         &role.ID,
		IsActive:       true,
	}
	require.NoError(t, w.DB.Create(user).Error)

	accountName := "test-account-" + uniqueID
	account := &models.WhatsAppAccount{
		OrganizationID: org.ID,
		Name:           accountName,
		GowaBaseURL:    "http://gowa.test:3000",
		GowaDeviceID:   "device-" + uniqueID,
	}
	require.NoError(t, w.DB.Create(account).Error)

	template := &models.Template{
		OrganizationID:  org.ID,
		WhatsAppAccount: accountName,
		Name:            "test_template_" + uniqueID,
		Language:        "en",
		Category:        "MARKETING",
		BodyContent:     "Hello {{1}}!",
	}
	require.NoError(t, w.DB.Create(template).Error)

	campaign := &models.BulkMessageCampaign{
		OrganizationID:  org.ID,
		Name:            "Test Campaign " + uniqueID,
		WhatsAppAccount: accountName,
		TemplateID:      template.ID,
		Status:          status,
		CreatedBy:       user.ID,
	}
	require.NoError(t, w.DB.Create(campaign).Error)

	return org, user, template, campaign
}

func TestWorker_updateRecipientStatus_Sent(t *testing.T) {
	w := testWorker(t)

	// Create campaign data with proper foreign keys
	_, _, _, campaign := createMinimalCampaignData(t, w, models.CampaignStatusProcessing)

	recipient := &models.BulkMessageRecipient{
		CampaignID:  campaign.ID,
		PhoneNumber: "1234567890",
		Status:      models.MessageStatusPending,
	}
	require.NoError(t, w.DB.Create(recipient).Error)

	// Test updating to sent status
	w.updateRecipientStatus(recipient.ID, models.MessageStatusSent, "wamid.123", "")

	var updated models.BulkMessageRecipient
	require.NoError(t, w.DB.First(&updated, recipient.ID).Error)
	assert.Equal(t, models.MessageStatusSent, updated.Status)
	assert.Equal(t, "wamid.123", updated.WhatsAppMessageID)
	assert.NotNil(t, updated.SentAt)
}

func TestWorker_updateRecipientStatus_Failed(t *testing.T) {
	w := testWorker(t)

	// Create campaign data with proper foreign keys
	_, _, _, campaign := createMinimalCampaignData(t, w, models.CampaignStatusProcessing)

	recipient := &models.BulkMessageRecipient{
		CampaignID:  campaign.ID,
		PhoneNumber: "9876543210",
		Status:      models.MessageStatusPending,
	}
	require.NoError(t, w.DB.Create(recipient).Error)

	w.updateRecipientStatus(recipient.ID, models.MessageStatusFailed, "", "API error")

	var updated models.BulkMessageRecipient
	require.NoError(t, w.DB.First(&updated, recipient.ID).Error)
	assert.Equal(t, models.MessageStatusFailed, updated.Status)
	assert.Equal(t, "API error", updated.ErrorMessage)
}

func TestWorker_incrementCampaignCount(t *testing.T) {
	w := testWorker(t)

	// Create campaign data with proper foreign keys
	_, _, _, campaign := createMinimalCampaignData(t, w, models.CampaignStatusProcessing)

	// Increment sent count multiple times
	w.incrementCampaignCount(campaign.ID, "sent_count")
	w.incrementCampaignCount(campaign.ID, "sent_count")
	w.incrementCampaignCount(campaign.ID, "failed_count")

	var updated models.BulkMessageCampaign
	require.NoError(t, w.DB.First(&updated, campaign.ID).Error)
	assert.Equal(t, 2, updated.SentCount)
	assert.Equal(t, 1, updated.FailedCount)
}

func TestWorker_checkCampaignCompletion_CompletesWhenAllProcessed(t *testing.T) {
	w := testWorker(t)

	// Create campaign data with proper foreign keys
	org, _, _, campaign := createMinimalCampaignData(t, w, models.CampaignStatusProcessing)

	// Update campaign counts for this test
	require.NoError(t, w.DB.Model(campaign).Updates(map[string]any{
		"total_recipients": 2,
		"sent_count":       2,
	}).Error)

	// Create recipients that are already processed (not pending)
	recipient1 := &models.BulkMessageRecipient{
		CampaignID:  campaign.ID,
		PhoneNumber: "1111111111",
		Status:      models.MessageStatusSent,
	}
	recipient2 := &models.BulkMessageRecipient{
		CampaignID:  campaign.ID,
		PhoneNumber: "2222222222",
		Status:      models.MessageStatusSent,
	}
	require.NoError(t, w.DB.Create(recipient1).Error)
	require.NoError(t, w.DB.Create(recipient2).Error)

	// Check completion - should complete since no pending recipients
	w.checkCampaignCompletion(context.Background(), campaign.ID, org.ID)

	var updated models.BulkMessageCampaign
	require.NoError(t, w.DB.First(&updated, campaign.ID).Error)
	assert.Equal(t, models.CampaignStatusCompleted, updated.Status)
	assert.NotNil(t, updated.CompletedAt)
}

func TestWorker_checkCampaignCompletion_DoesNotCompleteWithPending(t *testing.T) {
	w := testWorker(t)

	// Create campaign data with proper foreign keys
	org, _, _, campaign := createMinimalCampaignData(t, w, models.CampaignStatusProcessing)

	// Update campaign counts for this test
	require.NoError(t, w.DB.Model(campaign).Updates(map[string]any{
		"total_recipients": 2,
		"sent_count":       1,
	}).Error)

	// Create one processed and one pending recipient
	recipient1 := &models.BulkMessageRecipient{
		CampaignID:  campaign.ID,
		PhoneNumber: "1111111111",
		Status:      models.MessageStatusSent,
	}
	recipient2 := &models.BulkMessageRecipient{
		CampaignID:  campaign.ID,
		PhoneNumber: "2222222222",
		Status:      models.MessageStatusPending,
	}
	require.NoError(t, w.DB.Create(recipient1).Error)
	require.NoError(t, w.DB.Create(recipient2).Error)

	// Check completion - should NOT complete since there's a pending recipient
	w.checkCampaignCompletion(context.Background(), campaign.ID, org.ID)

	var updated models.BulkMessageCampaign
	require.NoError(t, w.DB.First(&updated, campaign.ID).Error)
	assert.Equal(t, models.CampaignStatusProcessing, updated.Status)
	assert.Nil(t, updated.CompletedAt)
}

func TestWorker_checkCampaignCompletion_NotProcessingStatus(t *testing.T) {
	w := testWorker(t)

	// Create campaign data with proper foreign keys - status is paused
	org, _, _, campaign := createMinimalCampaignData(t, w, models.CampaignStatusPaused)

	// Should not change status since it's not models.CampaignStatusProcessing
	w.checkCampaignCompletion(context.Background(), campaign.ID, org.ID)

	var updated models.BulkMessageCampaign
	require.NoError(t, w.DB.First(&updated, campaign.ID).Error)
	assert.Equal(t, models.CampaignStatusPaused, updated.Status)
}

func TestWorker_sendTemplateMessage_RendersBodyLocally(t *testing.T) {
	w := testWorker(t)

	var capturedBody map[string]any
	server := newGowaSendServer("3EB0WORKERTEST1", &capturedBody)
	defer server.Close()

	pointWorkerAtGowa(w)

	account := &models.WhatsAppAccount{
		GowaBaseURL:  server.URL,
		GowaDeviceID: "dev1",
	}

	template := &models.Template{
		Name:        "test_template",
		Language:    "en",
		BodyContent: "Hello {{1}}, welcome to {{2}}!",
	}

	recipient := &models.BulkMessageRecipient{
		PhoneNumber: "1234567890",
		TemplateParams: models.JSONB{
			"1": "Hello",
			"2": "World",
		},
	}

	msgID, err := w.sendTemplateMessage(context.Background(), account, template, recipient, &models.BulkMessageCampaign{})
	require.NoError(t, err)
	assert.Equal(t, "3EB0WORKERTEST1", msgID)

	// Verify the body was rendered locally and sent as plain text
	assert.Equal(t, "1234567890@s.whatsapp.net", capturedBody["phone"])
	assert.Equal(t, "Hello Hello, welcome to World!", capturedBody["message"])
}

// When the recipient has explicit HeaderParams, the worker must use them
// for the TEXT header instead of falling back to TemplateParams.
// This protects positional templates where header {{1}} and body {{1}}
// would otherwise share the same value.
func TestWorker_sendTemplateMessage_HeaderParamsTakePrecedence(t *testing.T) {
	w := testWorker(t)

	var capturedBody map[string]any
	server := newGowaSendServer("3EB0HPEXPLICIT", &capturedBody)
	defer server.Close()

	pointWorkerAtGowa(w)

	account := &models.WhatsAppAccount{GowaBaseURL: server.URL, GowaDeviceID: "dev1"}
	template := &models.Template{
		Name:          "tpl_explicit_hp",
		Language:      "en",
		HeaderType:    "TEXT",
		HeaderContent: "Code {{1}}",
		BodyContent:   "Use {{1}} to redeem",
	}
	recipient := &models.BulkMessageRecipient{
		PhoneNumber:    "1234567890",
		HeaderParams:   models.JSONB{"1": "HEADER-VAL"},
		TemplateParams: models.JSONB{"1": "BODY-VAL"},
	}

	_, err := w.sendTemplateMessage(context.Background(), account, template, recipient, &models.BulkMessageCampaign{})
	require.NoError(t, err)

	assert.Equal(t, "*Code HEADER-VAL*\n\nUse BODY-VAL to redeem", capturedBody["message"],
		"body {{1}} must keep its own value when HeaderParams is set")
}

// Legacy recipient rows persisted before HeaderParams existed only have
// TemplateParams. For NAMED templates the var name is unique across
// components, so the worker falls back to TemplateParams[name].
func TestWorker_sendTemplateMessage_HeaderParamsFallbackToTemplateParams(t *testing.T) {
	w := testWorker(t)

	var capturedBody map[string]any
	server := newGowaSendServer("3EB0HPFALLBACK", &capturedBody)
	defer server.Close()

	pointWorkerAtGowa(w)

	account := &models.WhatsAppAccount{GowaBaseURL: server.URL, GowaDeviceID: "dev1"}
	template := &models.Template{
		Name:          "tpl_fallback_hp",
		Language:      "en",
		HeaderType:    "TEXT",
		HeaderContent: "Our {{season}} sale",
		BodyContent:   "Hi {{name}}",
	}
	recipient := &models.BulkMessageRecipient{
		PhoneNumber: "1234567890",
		// No HeaderParams — legacy row. season lives in TemplateParams.
		TemplateParams: models.JSONB{"season": "Summer", "name": "Alex"},
	}

	_, err := w.sendTemplateMessage(context.Background(), account, template, recipient, &models.BulkMessageCampaign{})
	require.NoError(t, err)

	assert.Equal(t, "*Our Summer sale*\n\nHi Alex", capturedBody["message"],
		"header must still render via the TemplateParams fallback")
}

func TestWorker_sendTemplateMessage_NoParams(t *testing.T) {
	w := testWorker(t)

	var capturedBody map[string]any
	server := newGowaSendServer("3EB0WORKERTEST2", &capturedBody)
	defer server.Close()

	pointWorkerAtGowa(w)

	account := &models.WhatsAppAccount{
		GowaBaseURL:  server.URL,
		GowaDeviceID: "dev1",
	}

	template := &models.Template{
		Name:        "simple_template",
		Language:    "en",
		BodyContent: "Static message body",
	}

	recipient := &models.BulkMessageRecipient{
		PhoneNumber:    "1234567890",
		TemplateParams: nil, // No params
	}

	msgID, err := w.sendTemplateMessage(context.Background(), account, template, recipient, &models.BulkMessageCampaign{})
	require.NoError(t, err)
	assert.Equal(t, "3EB0WORKERTEST2", msgID)

	// Body without placeholders is sent verbatim
	assert.Equal(t, "Static message body", capturedBody["message"])
}

func TestWorker_Close_NilConsumer(t *testing.T) {
	w := &Worker{
		Consumer: nil, // No consumer
	}

	err := w.Close()
	assert.NoError(t, err)
}

func TestWorker_HandleRecipientJob_Success(t *testing.T) {
	w := testWorker(t)
	org, account, template, campaign, recipient := createTestCampaignData(t, w)

	// Create mock GOWA server
	server := newGowaSendServer("wamid.success123", nil)
	defer server.Close()

	// Point the account at the mock server
	require.NoError(t, w.DB.Model(account).Update("gowa_base_url", server.URL).Error)
	pointWorkerAtGowa(w)

	job := &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    recipient.PhoneNumber,
		RecipientName:  recipient.RecipientName,
		TemplateParams: recipient.TemplateParams,
	}

	err := w.HandleRecipientJob(context.Background(), job)
	require.NoError(t, err)

	// Verify recipient status updated
	var updatedRecipient models.BulkMessageRecipient
	require.NoError(t, w.DB.First(&updatedRecipient, recipient.ID).Error)
	assert.Equal(t, models.MessageStatusSent, updatedRecipient.Status)
	assert.Equal(t, "wamid.success123", updatedRecipient.WhatsAppMessageID)

	// Verify campaign count incremented
	var updatedCampaign models.BulkMessageCampaign
	require.NoError(t, w.DB.First(&updatedCampaign, campaign.ID).Error)
	assert.Equal(t, 1, updatedCampaign.SentCount)

	// Verify message record created
	var message models.Message
	require.NoError(t, w.DB.Where("template_name = ?", template.Name).First(&message).Error)
	assert.Equal(t, models.MessageStatusSent, message.Status)
	assert.Equal(t, models.DirectionOutgoing, message.Direction)
	assert.Equal(t, models.MessageTypeTemplate, message.MessageType)
}

func TestWorker_HandleRecipientJob_WhatsAppError(t *testing.T) {
	w := testWorker(t)
	org, account, _, campaign, recipient := createTestCampaignData(t, w)

	// Create mock server that returns a GOWA error
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(rw).Encode(map[string]any{
			"code":    "BAD_REQUEST",
			"message": "Invalid phone number",
		})
	}))
	defer server.Close()

	require.NoError(t, w.DB.Model(account).Update("gowa_base_url", server.URL).Error)
	pointWorkerAtGowa(w)

	job := &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    recipient.PhoneNumber,
		RecipientName:  recipient.RecipientName,
		TemplateParams: recipient.TemplateParams,
	}

	err := w.HandleRecipientJob(context.Background(), job)
	require.NoError(t, err) // Job handler returns nil to not retry

	// Verify recipient marked as failed
	var updatedRecipient models.BulkMessageRecipient
	require.NoError(t, w.DB.First(&updatedRecipient, recipient.ID).Error)
	assert.Equal(t, models.MessageStatusFailed, updatedRecipient.Status)
	assert.NotEmpty(t, updatedRecipient.ErrorMessage)

	// Verify campaign failed count incremented
	var updatedCampaign models.BulkMessageCampaign
	require.NoError(t, w.DB.First(&updatedCampaign, campaign.ID).Error)
	assert.Equal(t, 1, updatedCampaign.FailedCount)
}

func TestWorker_HandleRecipientJob_CreatesContact(t *testing.T) {
	w := testWorker(t)
	org, account, _, campaign, recipient := createTestCampaignData(t, w)

	// Create mock GOWA server
	server := newGowaSendServer("wamid.contact123", nil)
	defer server.Close()

	require.NoError(t, w.DB.Model(account).Update("gowa_base_url", server.URL).Error)
	pointWorkerAtGowa(w)

	// Use a new phone number that doesn't have a contact
	newPhone := "9998887777"
	job := &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    newPhone,
		RecipientName:  "New Contact",
		TemplateParams: recipient.TemplateParams,
	}

	err := w.HandleRecipientJob(context.Background(), job)
	require.NoError(t, err)

	// Verify contact was created
	var contact models.Contact
	require.NoError(t, w.DB.Where("organization_id = ? AND phone_number = ?", org.ID, newPhone).First(&contact).Error)
	assert.Equal(t, "New Contact", contact.ProfileName)
}

func TestWorker_HandleRecipientJob_CampaignCompletion(t *testing.T) {
	w := testWorker(t)
	org, account, _, campaign, recipient := createTestCampaignData(t, w)

	// Create mock GOWA server
	server := newGowaSendServer("wamid.complete123", nil)
	defer server.Close()

	require.NoError(t, w.DB.Model(account).Update("gowa_base_url", server.URL).Error)
	pointWorkerAtGowa(w)

	job := &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    recipient.PhoneNumber,
		RecipientName:  recipient.RecipientName,
		TemplateParams: recipient.TemplateParams,
	}

	err := w.HandleRecipientJob(context.Background(), job)
	require.NoError(t, err)

	// Verify campaign is marked as completed (all recipients processed)
	var updatedCampaign models.BulkMessageCampaign
	require.NoError(t, w.DB.First(&updatedCampaign, campaign.ID).Error)
	assert.Equal(t, models.CampaignStatusCompleted, updatedCampaign.Status)
	assert.NotNil(t, updatedCampaign.CompletedAt)
}

func TestWorker_HandleRecipientJob_TemplateParamSubstitution(t *testing.T) {
	w := testWorker(t)
	org, account, template, campaign, recipient := createTestCampaignData(t, w)

	// Create mock GOWA server
	server := newGowaSendServer("wamid.subst123", nil)
	defer server.Close()

	require.NoError(t, w.DB.Model(account).Update("gowa_base_url", server.URL).Error)
	pointWorkerAtGowa(w)

	job := &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    recipient.PhoneNumber,
		RecipientName:  recipient.RecipientName,
		TemplateParams: models.JSONB{
			"1": "Alice",
			"2": "ORD-456",
		},
	}

	err := w.HandleRecipientJob(context.Background(), job)
	require.NoError(t, err)

	// Verify message content has substituted params
	var message models.Message
	require.NoError(t, w.DB.Where("template_name = ?", template.Name).Order("created_at desc").First(&message).Error)
	assert.Contains(t, message.Content, "Alice")
	assert.Contains(t, message.Content, "ORD-456")
	assert.NotContains(t, message.Content, "{{1}}")
	assert.NotContains(t, message.Content, "{{2}}")
}

func TestWorker_DecryptAccountSecrets_WithEncryptionKey(t *testing.T) {
	w := &Worker{
		Config: &config.Config{
			App: config.AppConfig{EncryptionKey: "test-secret-key-for-aes256"},
		},
	}

	// Encrypt the GOWA webhook secret
	plainSecret := "gowa-webhook-secret-123"
	encSecret, err := crypto.Encrypt(plainSecret, w.Config.App.EncryptionKey)
	require.NoError(t, err)

	// Verify it is actually encrypted
	assert.True(t, crypto.IsEncrypted(encSecret))

	account := &models.WhatsAppAccount{
		GowaWebhookSecret: encSecret,
	}

	w.decryptAccountSecrets(account)

	assert.Equal(t, plainSecret, account.GowaWebhookSecret)
}

func TestWorker_DecryptAccountSecrets_NilConfig(t *testing.T) {
	w := &Worker{}

	account := &models.WhatsAppAccount{
		GowaWebhookSecret: "plain-secret",
	}

	w.decryptAccountSecrets(account)

	// Should remain unchanged (no-op)
	assert.Equal(t, "plain-secret", account.GowaWebhookSecret)
}

// Unit tests for parameter resolution functions (no database required)

func TestResolveTemplateParams_NamedParams(t *testing.T) {
	bodyContent := "Hello {{name}}, your order {{order_id}} is ready!"
	params := models.JSONB{
		"name":     "John",
		"order_id": "ORD-123",
	}

	result := templateutil.ResolveParams(bodyContent, params)

	assert.Equal(t, []string{"John", "ORD-123"}, result)
}

func TestResolveTemplateParams_PositionalParams(t *testing.T) {
	bodyContent := "Hello {{1}}, your order {{2}} is ready!"
	params := models.JSONB{
		"1": "John",
		"2": "ORD-123",
	}

	result := templateutil.ResolveParams(bodyContent, params)

	assert.Equal(t, []string{"John", "ORD-123"}, result)
}

func TestResolveTemplateParams_FallbackToPositional(t *testing.T) {
	// Named params in template, but user provides positional params
	bodyContent := "Hello {{name}}, your order {{order_id}} is ready!"
	params := models.JSONB{
		"1": "John",
		"2": "ORD-123",
	}

	result := templateutil.ResolveParams(bodyContent, params)

	assert.Equal(t, []string{"John", "ORD-123"}, result)
}

func TestResolveTemplateParams_MixedParams(t *testing.T) {
	// User provides some named, some positional
	bodyContent := "Hello {{name}}, your order {{order_id}} is ready!"
	params := models.JSONB{
		"name": "John",
		"2":    "ORD-123", // Positional fallback for second param
	}

	result := templateutil.ResolveParams(bodyContent, params)

	assert.Equal(t, []string{"John", "ORD-123"}, result)
}

func TestResolveTemplateParams_NoParams(t *testing.T) {
	// Template without any parameters
	bodyContent := "Hello, your order is ready!"
	params := models.JSONB{
		"1": "John",
		"2": "ORD-123",
	}

	result := templateutil.ResolveParams(bodyContent, params)

	assert.Nil(t, result)
}

func TestResolveTemplateParams_EmptyParams(t *testing.T) {
	bodyContent := "Hello {{name}}!"
	params := models.JSONB{}

	result := templateutil.ResolveParams(bodyContent, params)

	assert.Nil(t, result)
}

func TestReplaceTemplateContent_NamedParams(t *testing.T) {
	bodyContent := "Hello {{name}}, your order {{order_id}} is ready!"
	content := "Hello {{name}}, your order {{order_id}} is ready!"
	params := models.JSONB{
		"name":     "John",
		"order_id": "ORD-123",
	}

	result := templateutil.ReplaceWithJSONBParams(bodyContent, content, params)

	assert.Equal(t, "Hello John, your order ORD-123 is ready!", result)
}

func TestReplaceTemplateContent_PositionalParams(t *testing.T) {
	bodyContent := "Hello {{1}}, your order {{2}} is ready!"
	content := "Hello {{1}}, your order {{2}} is ready!"
	params := models.JSONB{
		"1": "John",
		"2": "ORD-123",
	}

	result := templateutil.ReplaceWithJSONBParams(bodyContent, content, params)

	assert.Equal(t, "Hello John, your order ORD-123 is ready!", result)
}

func TestReplaceTemplateContent_NamedParamsWithPositionalInput(t *testing.T) {
	// Template has named placeholders but user provides positional params
	bodyContent := "Hello {{name}}, your order {{order_id}} is ready!"
	content := "Hello {{name}}, your order {{order_id}} is ready!"
	params := models.JSONB{
		"1": "John",
		"2": "ORD-123",
	}

	result := templateutil.ReplaceWithJSONBParams(bodyContent, content, params)

	assert.Equal(t, "Hello John, your order ORD-123 is ready!", result)
}

func TestReplaceTemplateContent_NoParams(t *testing.T) {
	// Template without any parameters
	bodyContent := "Hello, your order is ready!"
	content := "Hello, your order is ready!"
	params := models.JSONB{
		"1": "John",
		"2": "ORD-123",
	}

	result := templateutil.ReplaceWithJSONBParams(bodyContent, content, params)

	assert.Equal(t, "Hello, your order is ready!", result)
}

// A duplicate job — what a pause→restart re-enqueue produces, with the
// original job still unread in the Redis Stream — must skip a recipient
// another job already claimed or finished. Without the atomic claim both
// jobs would send.
func TestWorker_HandleRecipientJob_DuplicateJobSkipsClaimedRecipient(t *testing.T) {
	w := testWorker(t)
	org, account, _, campaign, recipient := createTestCampaignData(t, w)

	// The original job already claimed and sent this recipient.
	require.NoError(t, w.DB.Model(recipient).Update("status", models.MessageStatusSent).Error)

	server := newGowaSendServer("wamid.duplicate", nil)
	defer server.Close()
	require.NoError(t, w.DB.Model(account).Update("gowa_base_url", server.URL).Error)
	pointWorkerAtGowa(w)

	job := &queue.RecipientJob{
		CampaignID:     campaign.ID,
		RecipientID:    recipient.ID,
		OrganizationID: org.ID,
		PhoneNumber:    recipient.PhoneNumber,
		RecipientName:  recipient.RecipientName,
		TemplateParams: recipient.TemplateParams,
	}

	require.NoError(t, w.HandleRecipientJob(context.Background(), job))

	var updatedRecipient models.BulkMessageRecipient
	require.NoError(t, w.DB.First(&updatedRecipient, recipient.ID).Error)
	assert.Equal(t, models.MessageStatusSent, updatedRecipient.Status, "duplicate job must not touch a finished recipient")

	var updatedCampaign models.BulkMessageCampaign
	require.NoError(t, w.DB.First(&updatedCampaign, campaign.ID).Error)
	assert.Zero(t, updatedCampaign.SentCount, "duplicate job must not double-count")
}
