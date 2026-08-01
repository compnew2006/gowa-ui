package handlers_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/gowa-ui/internal/handlers"
	"github.com/shridarpatil/gowa-ui/internal/models"
	"github.com/shridarpatil/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// scheduledMessageEnvelope decodes the standard {data: ...} envelope.
type scheduledMessageEnvelope struct {
	Data models.ScheduledMessage `json:"data"`
}

// createScheduledFixture inserts a scheduled message row directly for
// processor and update/cancel tests.
func createScheduledFixture(t *testing.T, app *handlers.App, orgID, contactID uuid.UUID, account string, userID uuid.UUID, status models.ScheduledMessageStatus, scheduledAt time.Time) *models.ScheduledMessage {
	t.Helper()

	sm := &models.ScheduledMessage{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		WhatsAppAccount: account,
		ContactID:       contactID,
		MessageType:     models.MessageTypeText,
		Content:         "scheduled hello",
		ScheduledAt:     scheduledAt,
		Status:          status,
		CreatedBy:       userID,
	}
	require.NoError(t, app.DB.Create(sm).Error)
	return sm
}

// --- Create ---

func TestApp_CreateScheduledMessage_Text(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-create")), testutil.WithPassword("password"), testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("sched-account"))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	scheduledAt := time.Now().Add(2 * time.Hour).UTC()
	req := testutil.NewJSONRequest(t, map[string]any{
		"type":             "text",
		"content":          map[string]any{"body": "see you tomorrow"},
		"whatsapp_account": account.Name,
		"scheduled_at":     scheduledAt.Format(time.RFC3339),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.CreateScheduledMessage(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp scheduledMessageEnvelope
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, models.ScheduledMessageStatusPending, resp.Data.Status)
	assert.Equal(t, "see you tomorrow", resp.Data.Content)
	assert.Equal(t, account.Name, resp.Data.WhatsAppAccount)
	assert.Equal(t, contact.ID, resp.Data.ContactID)
	assert.Equal(t, user.ID, resp.Data.CreatedBy)
	assert.WithinDuration(t, scheduledAt, resp.Data.ScheduledAt, time.Second)

	// Row persisted
	var count int64
	app.DB.Model(&models.ScheduledMessage{}).Where("contact_id = ?", contact.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestApp_CreateScheduledMessage_Validation(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-valid")), testutil.WithPassword("password"), testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("sched-valid-account"))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)

	tests := []struct {
		name string
		body map[string]any
	}{
		{
			name: "past scheduled_at",
			body: map[string]any{
				"type":             "text",
				"content":          map[string]any{"body": "late"},
				"whatsapp_account": account.Name,
				"scheduled_at":     time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
			},
		},
		{
			name: "missing scheduled_at",
			body: map[string]any{
				"type":             "text",
				"content":          map[string]any{"body": "no time"},
				"whatsapp_account": account.Name,
			},
		},
		{
			name: "empty text body",
			body: map[string]any{
				"type":             "text",
				"content":          map[string]any{"body": ""},
				"whatsapp_account": account.Name,
				"scheduled_at":     future,
			},
		},
		{
			name: "media without data",
			body: map[string]any{
				"type":             "image",
				"content":          map[string]any{"body": "caption"},
				"whatsapp_account": account.Name,
				"scheduled_at":     future,
			},
		},
		{
			name: "unsupported type",
			body: map[string]any{
				"type":             "interactive",
				"content":          map[string]any{"body": "x"},
				"whatsapp_account": account.Name,
				"scheduled_at":     future,
			},
		},
		{
			name: "unknown account",
			body: map[string]any{
				"type":             "text",
				"content":          map[string]any{"body": "hi"},
				"whatsapp_account": "no-such-account",
				"scheduled_at":     future,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.NewJSONRequest(t, tt.body)
			testutil.SetAuthContext(req, org.ID, user.ID)
			testutil.SetPathParam(req, "id", contact.ID.String())

			require.NoError(t, app.CreateScheduledMessage(req))
			assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
		})
	}
}

func TestApp_CreateScheduledMessage_Media(t *testing.T) {
	app := newTestApp(t)
	app.Config.Storage.LocalPath = t.TempDir() // keep media artifacts out of the repo
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-media")), testutil.WithPassword("password"), testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("sched-media-account"))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"type": "document",
		"content": map[string]any{
			"body":            "the report",
			"media_data":      base64.StdEncoding.EncodeToString([]byte("fake-pdf-bytes")),
			"media_mime_type": "application/pdf",
			"media_filename":  "report.pdf",
		},
		"whatsapp_account": account.Name,
		"scheduled_at":     time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.CreateScheduledMessage(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp scheduledMessageEnvelope
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, models.MessageTypeDocument, resp.Data.MessageType)
	assert.NotEmpty(t, resp.Data.MediaURL, "media must be persisted locally at schedule time")
	assert.Equal(t, "application/pdf", resp.Data.MediaMimeType)
	assert.Equal(t, "report.pdf", resp.Data.MediaFilename)
}

func TestApp_CreateScheduledMessage_AgentScoping(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	// Agent without any role/permissions — can only reach assigned contacts.
	agent := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-agent")), testutil.WithPassword("password"))
	account := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("sched-scope-account"))
	contact := testutil.CreateTestContact(t, app.DB, org.ID) // unassigned

	req := testutil.NewJSONRequest(t, map[string]any{
		"type":             "text",
		"content":          map[string]any{"body": "hi"},
		"whatsapp_account": account.Name,
		"scheduled_at":     time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
	})
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.CreateScheduledMessage(req))
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

// --- List ---

func TestApp_ListContactScheduledMessages(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-list")), testutil.WithPassword("password"), testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("sched-list-account"))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	other := testutil.CreateTestContact(t, app.DB, org.ID)

	createScheduledFixture(t, app, org.ID, contact.ID, account.Name, user.ID, models.ScheduledMessageStatusPending, time.Now().Add(time.Hour))
	createScheduledFixture(t, app, org.ID, contact.ID, account.Name, user.ID, models.ScheduledMessageStatusCancelled, time.Now().Add(2*time.Hour))
	createScheduledFixture(t, app, org.ID, other.ID, account.Name, user.ID, models.ScheduledMessageStatusPending, time.Now().Add(time.Hour))

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.ListContactScheduledMessages(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			ScheduledMessages []models.ScheduledMessage `json:"scheduled_messages"`
			Total             int64                     `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, int64(2), resp.Data.Total)

	// Status filter narrows to pending only.
	req = testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetQueryParam(req, "status", "pending")

	require.NoError(t, app.ListContactScheduledMessages(req))
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
}

// --- Update / Cancel ---

func TestApp_UpdateScheduledMessage(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-update")), testutil.WithPassword("password"), testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("sched-update-account"))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	sm := createScheduledFixture(t, app, org.ID, contact.ID, account.Name, user.ID, models.ScheduledMessageStatusPending, time.Now().Add(time.Hour))

	newTime := time.Now().Add(3 * time.Hour).UTC()
	req := testutil.NewJSONRequest(t, map[string]any{
		"content":      map[string]any{"body": "edited body"},
		"scheduled_at": newTime.Format(time.RFC3339),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", sm.ID.String())

	require.NoError(t, app.UpdateScheduledMessage(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp scheduledMessageEnvelope
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, "edited body", resp.Data.Content)
	assert.WithinDuration(t, newTime, resp.Data.ScheduledAt, time.Second)
}

func TestApp_UpdateScheduledMessage_NonPending(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-update-sent")), testutil.WithPassword("password"), testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("sched-upd-sent-acct"))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	sm := createScheduledFixture(t, app, org.ID, contact.ID, account.Name, user.ID, models.ScheduledMessageStatusSent, time.Now().Add(-time.Hour))

	req := testutil.NewJSONRequest(t, map[string]any{
		"content": map[string]any{"body": "too late"},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", sm.ID.String())

	require.NoError(t, app.UpdateScheduledMessage(req))
	assert.Equal(t, fasthttp.StatusConflict, testutil.GetResponseStatusCode(req))
}

func TestApp_CancelScheduledMessage(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-cancel")), testutil.WithPassword("password"), testutil.WithRoleID(&adminRole.ID))
	account := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("sched-cancel-account"))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	sm := createScheduledFixture(t, app, org.ID, contact.ID, account.Name, user.ID, models.ScheduledMessageStatusPending, time.Now().Add(time.Hour))

	req := testutil.NewRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", sm.ID.String())

	require.NoError(t, app.CancelScheduledMessage(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var stored models.ScheduledMessage
	require.NoError(t, app.DB.First(&stored, "id = ?", sm.ID).Error)
	assert.Equal(t, models.ScheduledMessageStatusCancelled, stored.Status)

	// Cancelling again conflicts.
	req = testutil.NewRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", sm.ID.String())

	require.NoError(t, app.CancelScheduledMessage(req))
	assert.Equal(t, fasthttp.StatusConflict, testutil.GetResponseStatusCode(req))
}

// --- Processor ---

func TestScheduledMessageProcessor_ProcessDue_SendsText(t *testing.T) {
	mock := newMockGowaServer()
	defer mock.close()
	app := newMsgTestApp(t, mock)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-proc")), testutil.WithPassword("password"))
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	sm := createScheduledFixture(t, app, org.ID, contact.ID, account.Name, user.ID, models.ScheduledMessageStatusPending, time.Now().Add(-time.Minute))

	proc := handlers.NewScheduledMessageProcessor(app, time.Minute)
	proc.ProcessDue(time.Now())

	var stored models.ScheduledMessage
	require.NoError(t, app.DB.First(&stored, "id = ?", sm.ID).Error)
	assert.Equal(t, models.ScheduledMessageStatusSent, stored.Status)
	require.NotNil(t, stored.SentMessageID)

	// The unified sender created a real Message row attributed to the creator.
	var msg models.Message
	require.NoError(t, app.DB.First(&msg, "id = ?", *stored.SentMessageID).Error)
	assert.Equal(t, models.MessageStatusSent, msg.Status)
	assert.Equal(t, models.DirectionOutgoing, msg.Direction)
	assert.Equal(t, "scheduled hello", msg.Content)
	require.NotNil(t, msg.SentByUserID)
	assert.Equal(t, user.ID, *msg.SentByUserID)

	// GOWA actually received the send.
	assert.NotEmpty(t, mock.sentRequests())
}

func TestScheduledMessageProcessor_ProcessDue_ProviderFailure(t *testing.T) {
	mock := newMockGowaServer()
	defer mock.close()
	mock.setError("device disconnected")
	app := newMsgTestApp(t, mock)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-proc-fail")), testutil.WithPassword("password"))
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	sm := createScheduledFixture(t, app, org.ID, contact.ID, account.Name, user.ID, models.ScheduledMessageStatusPending, time.Now().Add(-time.Minute))

	proc := handlers.NewScheduledMessageProcessor(app, time.Minute)
	proc.ProcessDue(time.Now())

	var stored models.ScheduledMessage
	require.NoError(t, app.DB.First(&stored, "id = ?", sm.ID).Error)
	assert.Equal(t, models.ScheduledMessageStatusFailed, stored.Status)
	assert.NotEmpty(t, stored.ErrorMessage)
}

func TestScheduledMessageProcessor_ProcessDue_SkipsFutureAndCancelled(t *testing.T) {
	mock := newMockGowaServer()
	defer mock.close()
	app := newMsgTestApp(t, mock)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-proc-skip")), testutil.WithPassword("password"))
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	future := createScheduledFixture(t, app, org.ID, contact.ID, account.Name, user.ID, models.ScheduledMessageStatusPending, time.Now().Add(time.Hour))
	cancelled := createScheduledFixture(t, app, org.ID, contact.ID, account.Name, user.ID, models.ScheduledMessageStatusCancelled, time.Now().Add(-time.Hour))

	proc := handlers.NewScheduledMessageProcessor(app, time.Minute)
	proc.ProcessDue(time.Now())

	// Use separate destination structs: GORM adds a populated primary key
	// on the dest to the WHERE clause, which would break the second read.
	var storedFuture models.ScheduledMessage
	require.NoError(t, app.DB.First(&storedFuture, "id = ?", future.ID).Error)
	assert.Equal(t, models.ScheduledMessageStatusPending, storedFuture.Status)

	var storedCancelled models.ScheduledMessage
	require.NoError(t, app.DB.First(&storedCancelled, "id = ?", cancelled.ID).Error)
	assert.Equal(t, models.ScheduledMessageStatusCancelled, storedCancelled.Status)
}

func TestScheduledMessageProcessor_ProcessDue_MissingContactFails(t *testing.T) {
	mock := newMockGowaServer()
	defer mock.close()
	app := newMsgTestApp(t, mock)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-proc-noc")), testutil.WithPassword("password"))
	account := createTestAccount(t, app, org.ID)

	// Contact deleted after scheduling: satisfies the FK but is invisible
	// to the processor's lookup.
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	sm := createScheduledFixture(t, app, org.ID, contact.ID, account.Name, user.ID, models.ScheduledMessageStatusPending, time.Now().Add(-time.Minute))
	require.NoError(t, app.DB.Delete(contact).Error)

	proc := handlers.NewScheduledMessageProcessor(app, time.Minute)
	proc.ProcessDue(time.Now())

	var stored models.ScheduledMessage
	require.NoError(t, app.DB.First(&stored, "id = ?", sm.ID).Error)
	assert.Equal(t, models.ScheduledMessageStatusFailed, stored.Status)
	assert.Contains(t, stored.ErrorMessage, "contact not found")
}

func TestScheduledMessageProcessor_RecoverStale(t *testing.T) {
	mock := newMockGowaServer()
	defer mock.close()
	app := newMsgTestApp(t, mock)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("sched-stale")), testutil.WithPassword("password"))
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	stale := createScheduledFixture(t, app, org.ID, contact.ID, account.Name, user.ID, models.ScheduledMessageStatusProcessing, time.Now().Add(-time.Hour))
	fresh := createScheduledFixture(t, app, org.ID, contact.ID, account.Name, user.ID, models.ScheduledMessageStatusProcessing, time.Now())

	// Backdate the stale row's updated_at beyond the recovery threshold.
	require.NoError(t, app.DB.Model(&models.ScheduledMessage{}).
		Where("id = ?", stale.ID).
		UpdateColumn("updated_at", time.Now().Add(-time.Hour)).Error)

	proc := handlers.NewScheduledMessageProcessor(app, time.Minute)
	proc.RecoverStale(time.Now())

	var storedStale models.ScheduledMessage
	require.NoError(t, app.DB.First(&storedStale, "id = ?", stale.ID).Error)
	assert.Equal(t, models.ScheduledMessageStatusPending, storedStale.Status, "stale processing row must return to pending")

	var storedFresh models.ScheduledMessage
	require.NoError(t, app.DB.First(&storedFresh, "id = ?", fresh.ID).Error)
	assert.Equal(t, models.ScheduledMessageStatusProcessing, storedFresh.Status, "recent processing row must be left alone")
}
