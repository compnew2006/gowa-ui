package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/gowa-ui/internal/handlers"
	"github.com/shridarpatil/gowa-ui/internal/models"
	"github.com/shridarpatil/gowa-ui/internal/templateutil"
	"github.com/shridarpatil/gowa-ui/pkg/gowa"
	"github.com/shridarpatil/gowa-ui/pkg/whatsapp"
	"github.com/shridarpatil/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gowaRequest captures a single request received by the mock GOWA server.
type gowaRequest struct {
	method string
	path   string
	body   map[string]any    // decoded JSON body (JSON requests)
	form   map[string]string // form fields (multipart requests)
	files  []string          // file field names (multipart requests)
}

// mockGowaServer is a mock GOWA REST API server for message tests.
// It records every request and returns configurable send responses.
type mockGowaServer struct {
	server        *httptest.Server
	mu            sync.Mutex
	requests      []gowaRequest
	returnError   bool
	errorMessage  string
	nextMessageID string
}

func newMockGowaServer() *mockGowaServer {
	m := &mockGowaServer{
		nextMessageID: "gowa-msg-" + uuid.New().String()[:8],
	}

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := gowaRequest{method: r.Method, path: r.URL.Path}
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			if err := r.ParseMultipartForm(32 << 20); err == nil {
				req.form = make(map[string]string)
				for key, vals := range r.MultipartForm.Value {
					if len(vals) > 0 {
						req.form[key] = vals[0]
					}
				}
				for name := range r.MultipartForm.File {
					req.files = append(req.files, name)
				}
			}
		} else {
			_ = json.NewDecoder(r.Body).Decode(&req.body)
		}

		m.mu.Lock()
		m.requests = append(m.requests, req)
		returnError, errorMessage := m.returnError, m.errorMessage
		m.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if returnError {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    "BAD_REQUEST",
				"message": errorMessage,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    "SUCCESS",
			"message": "Success",
			"results": map[string]any{"message_id": m.nextMessageID, "status": "ok"},
		})
	}))

	return m
}

// setError makes the mock server fail every send with a GOWA error envelope.
func (m *mockGowaServer) setError(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.returnError = true
	m.errorMessage = msg
}

// sentRequests returns a snapshot of the recorded requests.
func (m *mockGowaServer) sentRequests() []gowaRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]gowaRequest(nil), m.requests...)
}

func (m *mockGowaServer) close() {
	m.server.Close()
}

// newMsgTestApp creates an App instance for message testing with a mock GOWA server.
func newMsgTestApp(t *testing.T, mockServer *mockGowaServer) *handlers.App {
	t.Helper()

	app := newTestApp(t)

	// Route every account through the mock server, regardless of its base URL.
	whatsapp.RegisterGowaFactory(
		func(baseURL string) (string, string) { return "", "" },
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(mockServer.server.URL, username, password)
		},
	)
	app.WARegistry = whatsapp.NewRegistry(testutil.NopLogger())

	return app
}

// createTestAccount creates a test WhatsApp account in the database.
func createTestAccount(t *testing.T, app *handlers.App, orgID uuid.UUID) *models.WhatsAppAccount {
	t.Helper()

	account := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           "test-account-" + uuid.New().String()[:8],
		GowaBaseURL:    "http://gowa.test:3000",
		GowaDeviceID:   "device-" + uuid.New().String()[:8],
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(account).Error)
	return account
}

// --- SendOutgoingMessage Tests ---

func TestApp_SendOutgoingMessage_TextMessage_Success(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account: account,
		Contact: contact,
		Type:    models.MessageTypeText,
		Content: "Hello, World!",
	}

	// Use sync options to wait for result
	opts := handlers.AutoReplySendOptions()

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify message was saved to database
	assert.Equal(t, models.MessageTypeText, msg.MessageType)
	assert.Equal(t, "Hello, World!", msg.Content)
	assert.Equal(t, models.DirectionOutgoing, msg.Direction)
	assert.Equal(t, contact.ID, msg.ContactID)
	assert.Equal(t, org.ID, msg.OrganizationID)

	// Verify message was sent to the GOWA API
	reqs := mockServer.sentRequests()
	require.Len(t, reqs, 1)
	sent := reqs[0]
	assert.Equal(t, "/send/message", sent.path)
	assert.Equal(t, contact.PhoneNumber+"@s.whatsapp.net", sent.body["phone"])
	assert.Equal(t, "Hello, World!", sent.body["message"])

	// Verify message status was updated in DB
	var dbMsg models.Message
	require.NoError(t, app.DB.First(&dbMsg, msg.ID).Error)
	assert.Equal(t, models.MessageStatusSent, dbMsg.Status)
	assert.Equal(t, mockServer.nextMessageID, dbMsg.WhatsAppMessageID)
}

func TestApp_SendOutgoingMessage_TextMessage_APIError(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	mockServer.setError("Phone number is invalid")

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account: account,
		Contact: contact,
		Type:    models.MessageTypeText,
		Content: "Hello!",
	}

	opts := handlers.AutoReplySendOptions()

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	// Message is still returned (saved to DB) even if send fails
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify message status is failed in DB
	var dbMsg models.Message
	require.NoError(t, app.DB.First(&dbMsg, msg.ID).Error)
	assert.Equal(t, models.MessageStatusFailed, dbMsg.Status)
	assert.Contains(t, dbMsg.ErrorMessage, "Phone number is invalid")
}

func TestApp_SendOutgoingMessage_ImageMessage_WithMediaID(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account:       account,
		Contact:       contact,
		Type:          models.MessageTypeImage,
		MediaID:       "existing-media-id",
		MediaMimeType: "image/jpeg",
		Caption:       "Check this out!",
	}

	opts := handlers.AutoReplySendOptions()

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify image message was sent as a URL/ID pass-through (JSON, no upload)
	reqs := mockServer.sentRequests()
	require.Len(t, reqs, 1)
	sent := reqs[0]
	assert.Equal(t, "/send/image", sent.path)
	assert.Equal(t, "existing-media-id", sent.body["image_url"])
	assert.Equal(t, "Check this out!", sent.body["caption"])
	assert.Nil(t, sent.form)
}

func TestApp_SendOutgoingMessage_ImageMessage_WithMediaData(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account:       account,
		Contact:       contact,
		Type:          models.MessageTypeImage,
		MediaData:     []byte("fake image data"),
		MediaMimeType: "image/jpeg",
		MediaFilename: "photo.jpg",
		Caption:       "Photo caption",
	}

	opts := handlers.AutoReplySendOptions()

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// GOWA has no separate upload endpoint — the uploaded bytes are sent
	// inline as a single multipart request to /send/image.
	reqs := mockServer.sentRequests()
	require.Len(t, reqs, 1)
	sent := reqs[0]
	assert.Equal(t, "/send/image", sent.path)
	require.NotNil(t, sent.form)
	assert.Equal(t, contact.PhoneNumber+"@s.whatsapp.net", sent.form["phone"])
	assert.Equal(t, "Photo caption", sent.form["caption"])
	assert.Equal(t, []string{"image"}, sent.files)
}

func TestApp_SendOutgoingMessage_DocumentMessage(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account:       account,
		Contact:       contact,
		Type:          models.MessageTypeDocument,
		MediaID:       "doc-media-id",
		MediaMimeType: "application/pdf",
		MediaFilename: "report.pdf",
		Caption:       "Monthly report",
	}

	opts := handlers.AutoReplySendOptions()

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify document message was sent
	reqs := mockServer.sentRequests()
	require.Len(t, reqs, 1)
	sent := reqs[0]
	assert.Equal(t, "/send/file", sent.path)
	assert.Equal(t, "doc-media-id", sent.body["file_url"])
	assert.Equal(t, "Monthly report", sent.body["caption"])
}

func TestApp_SendOutgoingMessage_VideoMessage(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account:       account,
		Contact:       contact,
		Type:          models.MessageTypeVideo,
		MediaID:       "video-media-id",
		MediaMimeType: "video/mp4",
		Caption:       "Watch this!",
	}

	opts := handlers.AutoReplySendOptions()

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	reqs := mockServer.sentRequests()
	require.Len(t, reqs, 1)
	sent := reqs[0]
	assert.Equal(t, "/send/video", sent.path)
	assert.Equal(t, "video-media-id", sent.body["video_url"])
	assert.Equal(t, "Watch this!", sent.body["caption"])
}

func TestApp_SendOutgoingMessage_AudioMessage(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account:       account,
		Contact:       contact,
		Type:          models.MessageTypeAudio,
		MediaID:       "audio-media-id",
		MediaMimeType: "audio/ogg",
	}

	opts := handlers.AutoReplySendOptions()

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	reqs := mockServer.sentRequests()
	require.Len(t, reqs, 1)
	sent := reqs[0]
	assert.Equal(t, "/send/audio", sent.path)
	assert.Equal(t, "audio-media-id", sent.body["audio_url"])
}

func TestApp_SendOutgoingMessage_InteractiveButtons(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account:         account,
		Contact:         contact,
		Type:            models.MessageTypeInteractive,
		InteractiveType: "button",
		BodyText:        "Choose an option:",
		Buttons: []whatsapp.Button{
			{ID: "btn_yes", Title: "Yes"},
			{ID: "btn_no", Title: "No"},
		},
	}

	opts := handlers.AutoReplySendOptions()

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify interactive message was saved
	assert.Equal(t, models.MessageTypeInteractive, msg.MessageType)
	assert.Equal(t, "Choose an option:", msg.Content)

	// GOWA has no native reply buttons — the message falls back to a plain
	// text send with the options rendered as a numbered list.
	reqs := mockServer.sentRequests()
	require.Len(t, reqs, 1)
	sent := reqs[0]
	assert.Equal(t, "/send/message", sent.path)
	assert.Equal(t, "Choose an option:\n\n1. Yes\n2. No", sent.body["message"])
}

func TestApp_SendOutgoingMessage_InteractiveCTAURL(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account:         account,
		Contact:         contact,
		Type:            models.MessageTypeInteractive,
		InteractiveType: "cta_url",
		BodyText:        "Visit our website",
		ButtonText:      "Visit Now",
		URL:             "https://example.com",
	}

	opts := handlers.AutoReplySendOptions()

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify message content and interactive data
	assert.Equal(t, "Visit our website", msg.Content)
	assert.NotNil(t, msg.InteractiveData)
	assert.Equal(t, "cta_url", msg.InteractiveData["type"])

	// GOWA approximates CTA URL buttons with a link-preview message.
	reqs := mockServer.sentRequests()
	require.Len(t, reqs, 1)
	sent := reqs[0]
	assert.Equal(t, "/send/link", sent.path)
	assert.Equal(t, "https://example.com", sent.body["link"])
	assert.Equal(t, "Visit our website", sent.body["caption"])
}

func TestApp_SendOutgoingMessage_TemplateMessage(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	// Create a test template
	template := &models.Template{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: account.Name,
		Name:            "hello_world",
		DisplayName:     "Hello World Template",
		Category:        "MARKETING",
		Language:        "en",
		BodyContent:     "Hello {{1}}! Your order {{2}} is ready.",
	}
	require.NoError(t, app.DB.Create(template).Error)

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account:    account,
		Contact:    contact,
		Type:       models.MessageTypeTemplate,
		Template:   template,
		BodyParams: map[string]string{"1": "John", "2": "ORD-123"},
	}

	opts := handlers.AutoReplySendOptions()

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify template message was saved with rendered content
	assert.Equal(t, models.MessageTypeTemplate, msg.MessageType)
	assert.Equal(t, "Hello John! Your order ORD-123 is ready.", msg.Content)

	// Verify template metadata
	assert.NotNil(t, msg.Metadata)
	assert.Equal(t, "hello_world", msg.Metadata["template_name"])

	// Verify the template was rendered locally and sent as plain text
	reqs := mockServer.sentRequests()
	require.Len(t, reqs, 1)
	sent := reqs[0]
	assert.Equal(t, "/send/message", sent.path)
	assert.Equal(t, "Hello John! Your order ORD-123 is ready.", sent.body["message"])
}

func TestApp_SendOutgoingMessage_TemplateMessage_MissingTemplate(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account:    account,
		Contact:    contact,
		Type:       models.MessageTypeTemplate,
		Template:   nil, // Missing template
		BodyParams: map[string]string{"1": "param1"},
	}

	opts := handlers.AutoReplySendOptions()

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	// Message is created but send fails
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify message status is failed
	var dbMsg models.Message
	require.NoError(t, app.DB.First(&dbMsg, msg.ID).Error)
	assert.Equal(t, models.MessageStatusFailed, dbMsg.Status)
	assert.Contains(t, dbMsg.ErrorMessage, "template is required")
}

func TestApp_SendOutgoingMessage_AsyncOption(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account: account,
		Contact: contact,
		Type:    models.MessageTypeText,
		Content: "Async message",
	}

	// Use async options
	opts := handlers.DefaultSendOptions()
	assert.True(t, opts.Async)

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Wait for async send to complete
	app.WaitForBackgroundTasks()

	// Now verify message was sent and status updated in DB
	var dbMsg models.Message
	require.NoError(t, app.DB.First(&dbMsg, msg.ID).Error)
	assert.Equal(t, models.MessageStatusSent, dbMsg.Status)
	assert.NotEmpty(t, dbMsg.WhatsAppMessageID)
}

func TestApp_SendOutgoingMessage_SyncOption(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account: account,
		Contact: contact,
		Type:    models.MessageTypeText,
		Content: "Sync message",
	}

	// Use sync options (AutoReplySendOptions has Async: false)
	opts := handlers.AutoReplySendOptions()
	assert.False(t, opts.Async)

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Message status should be updated immediately (sync)
	var dbMsg models.Message
	require.NoError(t, app.DB.First(&dbMsg, msg.ID).Error)
	assert.Equal(t, models.MessageStatusSent, dbMsg.Status)
}

func TestApp_SendOutgoingMessage_WithSentByUser(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	// Create a test user (required due to foreign key constraint)
	user := &models.User{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Email:          "agent-" + uuid.New().String()[:8] + "@test.com",
		FullName:       "Test Agent",
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(user).Error)
	userID := user.ID

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account: account,
		Contact: contact,
		Type:    models.MessageTypeText,
		Content: "Message from agent",
	}

	opts := handlers.DefaultSendOptions()
	opts.SentByUserID = &userID

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Wait for async send
	app.WaitForBackgroundTasks()

	// Verify sent by user is recorded
	var dbMsg models.Message
	require.NoError(t, app.DB.First(&dbMsg, msg.ID).Error)
	require.NotNil(t, dbMsg.SentByUserID)
	assert.Equal(t, userID, *dbMsg.SentByUserID)
}

func TestApp_SendOutgoingMessage_UnsupportedType(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account: account,
		Contact: contact,
		Type:    "unknown_type",
		Content: "Some content",
	}

	opts := handlers.AutoReplySendOptions()

	msg, err := app.SendOutgoingMessage(ctx, req, opts)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify message status is failed due to unsupported type
	var dbMsg models.Message
	require.NoError(t, app.DB.First(&dbMsg, msg.ID).Error)
	assert.Equal(t, models.MessageStatusFailed, dbMsg.Status)
	assert.Contains(t, dbMsg.ErrorMessage, "unsupported message type")
}

// --- Options Preset Tests ---

func TestDefaultSendOptions(t *testing.T) {
	opts := handlers.DefaultSendOptions()

	assert.True(t, opts.BroadcastWebSocket)
	assert.True(t, opts.DispatchWebhook)
	assert.True(t, opts.Async)
	assert.Nil(t, opts.SentByUserID)
}

func TestAutoReplySendOptions(t *testing.T) {
	opts := handlers.AutoReplySendOptions()

	assert.True(t, opts.BroadcastWebSocket)
	assert.False(t, opts.DispatchWebhook)
	assert.False(t, opts.Async)
	assert.Nil(t, opts.SentByUserID)
}

func TestAPISendOptions(t *testing.T) {
	opts := handlers.APISendOptions()

	assert.False(t, opts.BroadcastWebSocket)
	assert.True(t, opts.DispatchWebhook)
	assert.True(t, opts.Async)
	assert.Nil(t, opts.SentByUserID)
}

// --- Message Preview Tests ---

func TestApp_SendOutgoingMessage_ContactLastMessageUpdated(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account: account,
		Contact: contact,
		Type:    models.MessageTypeText,
		Content: "This is a test message for preview",
	}

	opts := handlers.AutoReplySendOptions()

	_, err := app.SendOutgoingMessage(ctx, req, opts)
	require.NoError(t, err)

	// Verify contact's last message was updated
	var updatedContact models.Contact
	require.NoError(t, app.DB.First(&updatedContact, contact.ID).Error)
	assert.NotNil(t, updatedContact.LastMessageAt)
	assert.Equal(t, "This is a test message for preview", updatedContact.LastMessagePreview)
}

func TestApp_SendOutgoingMessage_MediaPreview(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	// Test image without caption
	req := handlers.OutgoingMessageRequest{
		Account:       account,
		Contact:       contact,
		Type:          models.MessageTypeImage,
		MediaID:       "media-123",
		MediaMimeType: "image/jpeg",
	}

	opts := handlers.AutoReplySendOptions()

	_, err := app.SendOutgoingMessage(ctx, req, opts)
	require.NoError(t, err)

	var updatedContact models.Contact
	require.NoError(t, app.DB.First(&updatedContact, contact.ID).Error)
	assert.Equal(t, "[Image]", updatedContact.LastMessagePreview)
}

func TestApp_SendOutgoingMessage_DocumentPreview(t *testing.T) {
	mockServer := newMockGowaServer()
	defer mockServer.close()

	app := newMsgTestApp(t, mockServer)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	ctx := testutil.TestContext(t)

	req := handlers.OutgoingMessageRequest{
		Account:       account,
		Contact:       contact,
		Type:          models.MessageTypeDocument,
		MediaID:       "media-123",
		MediaFilename: "report.pdf",
	}

	opts := handlers.AutoReplySendOptions()

	_, err := app.SendOutgoingMessage(ctx, req, opts)
	require.NoError(t, err)

	var updatedContact models.Contact
	require.NoError(t, app.DB.First(&updatedContact, contact.ID).Error)
	assert.Equal(t, "[Document: report.pdf]", updatedContact.LastMessagePreview)
}

// --- Template Parameter Tests ---

func TestExtractParamNamesFromContent_Positional(t *testing.T) {
	content := "Hello {{1}}! Your order {{2}} is ready for pickup at {{3}}."
	names := templateutil.ExtParamNames(content)

	require.Len(t, names, 3)
	assert.Equal(t, "1", names[0])
	assert.Equal(t, "2", names[1])
	assert.Equal(t, "3", names[2])
}

func TestExtractParamNamesFromContent_Named(t *testing.T) {
	content := "Hi {{customer_name}}, your order {{order_id}} will arrive on {{delivery_date}}."
	names := templateutil.ExtParamNames(content)

	require.Len(t, names, 3)
	assert.Equal(t, "customer_name", names[0])
	assert.Equal(t, "order_id", names[1])
	assert.Equal(t, "delivery_date", names[2])
}

func TestExtractParamNamesFromContent_Mixed(t *testing.T) {
	content := "Hello {{name}}, your code is {{1}}."
	names := templateutil.ExtParamNames(content)

	require.Len(t, names, 2)
	assert.Equal(t, "name", names[0])
	assert.Equal(t, "1", names[1])
}

func TestExtractParamNamesFromContent_NoParams(t *testing.T) {
	content := "This is a static message with no parameters."
	names := templateutil.ExtParamNames(content)

	assert.Nil(t, names)
}

func TestExtractParamNamesFromContent_DuplicateParams(t *testing.T) {
	content := "Hello {{name}}, {{name}} is a great name!"
	names := templateutil.ExtParamNames(content)

	// Should deduplicate
	require.Len(t, names, 1)
	assert.Equal(t, "name", names[0])
}

func TestResolveParams_NamedMatch(t *testing.T) {
	paramNames := []string{"customer_name", "order_id"}
	params := map[string]string{
		"customer_name": "John",
		"order_id":      "ORD-123",
	}

	result := templateutil.ResolveParamsFromMap(paramNames, params)

	require.Len(t, result, 2)
	assert.Equal(t, "John", result[0])
	assert.Equal(t, "ORD-123", result[1])
}

func TestResolveParams_PositionalMatch(t *testing.T) {
	paramNames := []string{"1", "2"}
	params := map[string]string{
		"1": "First",
		"2": "Second",
	}

	result := templateutil.ResolveParamsFromMap(paramNames, params)

	require.Len(t, result, 2)
	assert.Equal(t, "First", result[0])
	assert.Equal(t, "Second", result[1])
}

func TestResolveParams_FallbackToPositional(t *testing.T) {
	// Template has named params, but user sends positional
	paramNames := []string{"name", "code"}
	params := map[string]string{
		"1": "John",
		"2": "ABC123",
	}

	result := templateutil.ResolveParamsFromMap(paramNames, params)

	require.Len(t, result, 2)
	assert.Equal(t, "John", result[0])
	assert.Equal(t, "ABC123", result[1])
}

func TestResolveParams_MissingParams(t *testing.T) {
	paramNames := []string{"name", "order_id", "date"}
	params := map[string]string{
		"name": "John",
		// order_id and date are missing
	}

	result := templateutil.ResolveParamsFromMap(paramNames, params)

	require.Len(t, result, 3)
	assert.Equal(t, "John", result[0])
	assert.Equal(t, "", result[1]) // Missing - defaults to empty
	assert.Equal(t, "", result[2]) // Missing - defaults to empty
}

func TestResolveParams_EmptyInputs(t *testing.T) {
	// Empty param names
	result1 := templateutil.ResolveParamsFromMap([]string{}, map[string]string{"a": "b"})
	assert.Nil(t, result1)

	// Empty params map
	result2 := templateutil.ResolveParamsFromMap([]string{"a"}, map[string]string{})
	assert.Nil(t, result2)

	// Both empty
	result3 := templateutil.ResolveParamsFromMap([]string{}, map[string]string{})
	assert.Nil(t, result3)
}

func TestResolveParams_WrongParamNames(t *testing.T) {
	// User sends different param names than template expects
	paramNames := []string{"1", "2"} // Template expects positional
	params := map[string]string{
		"lastrec":  "Nifty",     // Wrong name
		"misstime": "banknifty", // Wrong name
	}

	result := templateutil.ResolveParamsFromMap(paramNames, params)

	require.Len(t, result, 2)
	// Both should be empty since names don't match
	assert.Equal(t, "", result[0])
	assert.Equal(t, "", result[1])
}
