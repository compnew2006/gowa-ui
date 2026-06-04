package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/compnew2006/whatomate/internal/models"
	objectstorage "github.com/compnew2006/whatomate/internal/storage"
	appwebsocket "github.com/compnew2006/whatomate/internal/websocket"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

type handlerFakeStorage struct {
	getCalls int
	getFunc  func(ctx context.Context, key string) (io.ReadCloser, objectstorage.ObjectInfo, error)
}

type legacyRestoreFixture struct {
	body        []byte
	contentType string
	delay       time.Duration
	release     <-chan struct{}
}

type legacyRestoreTestServer struct {
	server        *httptest.Server
	urlCalls      atomic.Int32
	downloadCalls atomic.Int32
}

func newLegacyRestoreTestServer(t *testing.T, fixtures map[string]legacyRestoreFixture) *legacyRestoreTestServer {
	t.Helper()

	testServer := &legacyRestoreTestServer{}
	testServer.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/v18.0/"):
			mediaID := strings.TrimPrefix(r.URL.Path, "/v18.0/")
			fixture, ok := fixtures[mediaID]
			if !ok {
				http.NotFound(w, r)
				return
			}
			testServer.urlCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{
				"url":       testServer.server.URL + "/download/" + mediaID,
				"mime_type": fixture.contentType,
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		case strings.HasPrefix(r.URL.Path, "/download/"):
			mediaID := strings.TrimPrefix(r.URL.Path, "/download/")
			fixture, ok := fixtures[mediaID]
			if !ok {
				http.NotFound(w, r)
				return
			}
			testServer.downloadCalls.Add(1)
			if fixture.release != nil {
				<-fixture.release
			}
			if fixture.delay > 0 {
				time.Sleep(fixture.delay)
			}
			if fixture.contentType != "" {
				w.Header().Set("Content-Type", fixture.contentType)
			}
			_, _ = w.Write(fixture.body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(testServer.server.Close)
	return testServer
}

func websocketClientSendChan(client *appwebsocket.Client) <-chan []byte {
	field := reflect.ValueOf(client).Elem().FieldByName("send")
	return *(*chan []byte)(unsafe.Pointer(field.UnsafeAddr()))
}

func waitForWSMessage(t *testing.T, client *appwebsocket.Client) appwebsocket.WSMessage {
	t.Helper()

	select {
	case raw := <-websocketClientSendChan(client):
		var msg appwebsocket.WSMessage
		require.NoError(t, json.Unmarshal(raw, &msg))
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for websocket message")
		return appwebsocket.WSMessage{}
	}
}

func waitForHubClientCount(t *testing.T, hub *appwebsocket.Hub, expected int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hub.GetClientCount() == expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for websocket hub client count %d", expected)
}

func countRegularFiles(t *testing.T, root string) int {
	t.Helper()

	count := 0
	require.NoError(t, filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info != nil && info.Mode().IsRegular() {
			count++
		}
		return nil
	}))
	return count
}

func (s *handlerFakeStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	_, err := io.Copy(io.Discard, body)
	return err
}

func (s *handlerFakeStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, objectstorage.ObjectInfo, error) {
	s.getCalls++
	if s.getFunc != nil {
		return s.getFunc(ctx, key)
	}
	return io.NopCloser(strings.NewReader("streamed-body")), objectstorage.ObjectInfo{
		Size:        int64(len("streamed-body")),
		ContentType: "application/pdf",
	}, nil
}

func (s *handlerFakeStorage) DeleteObject(ctx context.Context, key string) error {
	return nil
}

func TestServeMedia_StreamsFromObjectStorage(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	storage := &handlerFakeStorage{}
	app.ObjectStorage = storage

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	asset := models.MediaAsset{
		BaseModel: models.BaseModel{ID: uuid.New()},
		FileHash:  "feedface",
		S3Key:     "whatsmeow/media/fe/ed/feedface",
		MimeType:  "application/pdf",
		Size:      13,
	}
	require.NoError(t, app.DB.Create(&asset).Error)

	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: "whatsmeow",
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Document",
		MediaAssetID:    &asset.ID,
		MediaURL:        "/api/media/" + uuid.NewString(),
		MediaMimeType:   "application/pdf",
		MediaFilename:   "report.pdf",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&message).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "message_id", message.ID.String())

	err := app.ServeMedia(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	assert.Equal(t, "application/pdf", string(req.RequestCtx.Response.Header.ContentType()))
	assert.Equal(t, 1, storage.getCalls)
}

func TestServeMedia_StreamsLegacyLocalMedia(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	tempDir := t.TempDir()
	app.Config.Storage.LocalPath = tempDir

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	relativePath := filepath.Join("documents", "legacy-report.pdf")
	fullPath := filepath.Join(tempDir, relativePath)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o750))
	require.NoError(t, os.WriteFile(fullPath, []byte("legacy-pdf"), 0o600))

	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: "whatsmeow",
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Legacy document",
		MediaURL:        relativePath,
		MediaMimeType:   "application/pdf",
		MediaFilename:   "legacy-report.pdf",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&message).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "message_id", message.ID.String())

	err := app.ServeMedia(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	assert.Equal(t, "application/pdf", string(req.RequestCtx.Response.Header.ContentType()))
	assert.Equal(t, `inline; filename="legacy-report.pdf"`, string(req.RequestCtx.Response.Header.Peek("Content-Disposition")))
	assert.Equal(t, "legacy-pdf", string(req.RequestCtx.Response.Body()))
}

func TestRetryMediaDownload_ReturnsExistingMedia(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: "whatsmeow",
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Existing document",
		MediaURL:        "/api/media/" + uuid.NewString(),
		MediaMimeType:   "application/pdf",
		MediaFilename:   "existing.pdf",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&message).Error)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "message_id", message.ID.String())

	err := app.RetryMediaDownload(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var data struct {
		MediaURL      string `json:"media_url"`
		MediaFilename string `json:"media_filename"`
		MediaMimeType string `json:"media_mime_type"`
		MediaMimetype string `json:"media_mimetype"`
	}
	testutil.ParseEnvelopeResponse(t, req, &data)
	assert.Equal(t, message.MediaURL, data.MediaURL)
	assert.Equal(t, "existing.pdf", data.MediaFilename)
	assert.Equal(t, "application/pdf", data.MediaMimeType)
	assert.Equal(t, "application/pdf", data.MediaMimetype)
}

func TestRetryMediaDownload_ObjectBackedMissingMediaDoesNotReturnExistingMedia(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	app.ObjectStorage = &handlerFakeStorage{
		getFunc: func(ctx context.Context, key string) (io.ReadCloser, objectstorage.ObjectInfo, error) {
			return nil, objectstorage.ObjectInfo{}, objectstorage.ErrObjectNotFound
		},
	}

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	asset := models.MediaAsset{
		BaseModel: models.BaseModel{ID: uuid.New()},
		FileHash:  "missing-object-hash",
		S3Key:     "whatsmeow/media/mi/ss/missing-object-hash",
		MimeType:  "application/pdf",
		Size:      123,
	}
	require.NoError(t, app.DB.Create(&asset).Error)

	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: "whatsmeow",
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Object-backed document",
		MediaAssetID:    &asset.ID,
		MediaURL:        "/api/media/" + uuid.NewString(),
		MediaMimeType:   "application/pdf",
		MediaFilename:   "object-backed.pdf",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&message).Error)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "message_id", message.ID.String())

	err := app.RetryMediaDownload(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	assert.Contains(t, string(testutil.GetResponseBody(req)), "No recovery information available")
}

func TestServeMedia_ReturnsNotFoundForMissingLegacyLocalMedia(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	app.Config.Storage.LocalPath = t.TempDir()

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: "whatsmeow",
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Missing legacy document",
		MediaURL:        filepath.Join("documents", "missing.pdf"),
		MediaMimeType:   "application/pdf",
		MediaFilename:   "missing.pdf",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&message).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "message_id", message.ID.String())

	err := app.ServeMedia(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestServeMedia_RestoresMissingLegacyLocalMedia(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	tempDir := t.TempDir()
	app.Config.Storage.LocalPath = tempDir

	pdfBody := []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n")
	restoreServer := newLegacyRestoreTestServer(t, map[string]legacyRestoreFixture{
		"media-restore-1": {
			body:        pdfBody,
			contentType: "application/pdf",
		},
	})
	app.WhatsApp = whatsapp.NewWithBaseURL(testutil.NopLogger(), restoreServer.server.URL)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	hub := appwebsocket.NewHub(logf.New(logf.Opts{}))
	go hub.Run()
	app.WSHub = hub
	client := appwebsocket.NewClient(hub, nil, user.ID, org.ID)
	hub.Register(client)
	waitForHubClientCount(t, hub, 1)

	oldPath := filepath.Join("documents", "missing-invoice.pdf")
	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: account.Name,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Missing invoice",
		MediaURL:        oldPath,
		MediaMimeType:   "application/pdf",
		MediaFilename:   "invoice.tmp",
		Status:          models.MessageStatusReceived,
		Metadata: models.JSONB{
			"legacy_media_recovery_provider":   "meta",
			"legacy_media_recovery_media_id":   "media-restore-1",
			"legacy_media_recovery_phone_id":   account.PhoneID,
			"legacy_media_recovery_expires_at": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339Nano),
		},
	}
	require.NoError(t, app.DB.Create(&message).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "message_id", message.ID.String())

	err := app.ServeMedia(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	assert.Equal(t, "application/pdf", string(req.RequestCtx.Response.Header.ContentType()))
	assert.Equal(t, string(pdfBody), string(req.RequestCtx.Response.Body()))

	var refreshed models.Message
	require.NoError(t, app.DB.First(&refreshed, "id = ?", message.ID).Error)
	assert.NotEqual(t, oldPath, refreshed.MediaURL)
	assert.Equal(t, "application/pdf", refreshed.MediaMimeType)
	assert.Equal(t, "invoice.pdf", refreshed.MediaFilename)
	_, hasRestoredAt := refreshed.Metadata["legacy_media_restored_at"]
	assert.True(t, hasRestoredAt)
	_, statErr := os.Stat(filepath.Join(tempDir, refreshed.MediaURL))
	assert.NoError(t, statErr)

	wsMsg := waitForWSMessage(t, client)
	assert.Equal(t, appwebsocket.TypeMessageMediaUpdated, wsMsg.Type)
	payload, ok := wsMsg.Payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, message.ID.String(), payload["id"])
	assert.Equal(t, refreshed.MediaURL, payload["media_url"])
	assert.Equal(t, "application/pdf", payload["media_mime_type"])
	assert.Equal(t, "invoice.pdf", payload["media_filename"])
}

func TestServeMedia_RejectsEmptyRestoredPayload(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	tempDir := t.TempDir()
	app.Config.Storage.LocalPath = tempDir

	restoreServer := newLegacyRestoreTestServer(t, map[string]legacyRestoreFixture{
		"media-empty": {
			body:        []byte{},
			contentType: "application/pdf",
		},
	})
	app.WhatsApp = whatsapp.NewWithBaseURL(testutil.NopLogger(), restoreServer.server.URL)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	oldPath := filepath.Join("documents", "empty.pdf")
	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: account.Name,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Missing empty invoice",
		MediaURL:        oldPath,
		MediaMimeType:   "application/pdf",
		MediaFilename:   "empty.pdf",
		Status:          models.MessageStatusReceived,
		Metadata: models.JSONB{
			"legacy_media_recovery_provider":   "meta",
			"legacy_media_recovery_media_id":   "media-empty",
			"legacy_media_recovery_phone_id":   account.PhoneID,
			"legacy_media_recovery_expires_at": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339Nano),
		},
	}
	require.NoError(t, app.DB.Create(&message).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "message_id", message.ID.String())

	err := app.ServeMedia(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	assert.Equal(t, 0, countRegularFiles(t, tempDir))

	var refreshed models.Message
	require.NoError(t, app.DB.First(&refreshed, "id = ?", message.ID).Error)
	assert.Equal(t, oldPath, refreshed.MediaURL)
}

func TestServeMedia_DoesNotRestoreExpiredLegacyMedia(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	tempDir := t.TempDir()
	app.Config.Storage.LocalPath = tempDir

	restoreServer := newLegacyRestoreTestServer(t, map[string]legacyRestoreFixture{
		"media-expired": {
			body:        []byte("%PDF-1.7\nexpired\n"),
			contentType: "application/pdf",
		},
	})
	app.WhatsApp = whatsapp.NewWithBaseURL(testutil.NopLogger(), restoreServer.server.URL)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: time.Now().UTC().Add(-(31 * 24 * time.Hour))},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: account.Name,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Expired restore",
		MediaURL:        filepath.Join("documents", "expired.pdf"),
		MediaMimeType:   "application/pdf",
		MediaFilename:   "expired.pdf",
		Status:          models.MessageStatusReceived,
		Metadata: models.JSONB{
			"legacy_media_recovery_provider": "meta",
			"legacy_media_recovery_media_id": "media-expired",
			"legacy_media_recovery_phone_id": account.PhoneID,
		},
	}
	require.NoError(t, app.DB.Create(&message).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "message_id", message.ID.String())

	err := app.ServeMedia(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	assert.Equal(t, int32(0), restoreServer.urlCalls.Load())
	assert.Equal(t, int32(0), restoreServer.downloadCalls.Load())
}

func TestServeMedia_RestoresMissingLegacyLocalMediaOnlyOnceForConcurrentRequests(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	tempDir := t.TempDir()
	app.Config.Storage.LocalPath = tempDir

	restoreServer := newLegacyRestoreTestServer(t, map[string]legacyRestoreFixture{
		"media-concurrent": {
			body:        []byte("%PDF-1.7\nshared\n"),
			contentType: "application/pdf",
			delay:       250 * time.Millisecond,
		},
	})
	app.WhatsApp = whatsapp.NewWithBaseURL(testutil.NopLogger(), restoreServer.server.URL)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: account.Name,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Concurrent restore",
		MediaURL:        filepath.Join("documents", "concurrent.pdf"),
		MediaMimeType:   "application/pdf",
		MediaFilename:   "concurrent.pdf",
		Status:          models.MessageStatusReceived,
		Metadata: models.JSONB{
			"legacy_media_recovery_provider":   "meta",
			"legacy_media_recovery_media_id":   "media-concurrent",
			"legacy_media_recovery_phone_id":   account.PhoneID,
			"legacy_media_recovery_expires_at": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339Nano),
		},
	}
	require.NoError(t, app.DB.Create(&message).Error)

	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := testutil.NewGETRequest(t)
			testutil.SetAuthContext(req, org.ID, user.ID)
			testutil.SetPathParam(req, "message_id", message.ID.String())
			errs <- app.ServeMedia(req)
			statuses <- testutil.GetResponseStatusCode(req)
		}()
	}
	wg.Wait()
	close(errs)
	close(statuses)

	for err := range errs {
		require.NoError(t, err)
	}
	for status := range statuses {
		assert.Equal(t, fasthttp.StatusOK, status)
	}
	assert.Equal(t, int32(1), restoreServer.urlCalls.Load())
	assert.Equal(t, int32(1), restoreServer.downloadCalls.Load())
}

func TestServeMedia_ReturnsNotFoundWhenLegacyRestoreBudgetIsExhausted(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	tempDir := t.TempDir()
	app.Config.Storage.LocalPath = tempDir

	release := make(chan struct{})
	fixtures := map[string]legacyRestoreFixture{}
	for i := 0; i < 5; i++ {
		fixtures[fmt.Sprintf("media-budget-%d", i)] = legacyRestoreFixture{
			body:        []byte("%PDF-1.7\nbudget\n"),
			contentType: "application/pdf",
			release:     release,
		}
	}
	restoreServer := newLegacyRestoreTestServer(t, fixtures)
	app.WhatsApp = whatsapp.NewWithBaseURL(testutil.NopLogger(), restoreServer.server.URL)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	messageIDs := make([]uuid.UUID, 0, 5)
	for i := 0; i < 5; i++ {
		message := models.Message{
			BaseModel:       models.BaseModel{ID: uuid.New()},
			OrganizationID:  org.ID,
			ContactID:       contact.ID,
			WhatsAppAccount: account.Name,
			Direction:       models.DirectionIncoming,
			MessageType:     models.MessageTypeDocument,
			Content:         fmt.Sprintf("Budget restore %d", i),
			MediaURL:        filepath.Join("documents", fmt.Sprintf("budget-%d.pdf", i)),
			MediaMimeType:   "application/pdf",
			MediaFilename:   fmt.Sprintf("budget-%d.pdf", i),
			Status:          models.MessageStatusReceived,
			Metadata: models.JSONB{
				"legacy_media_recovery_provider":   "meta",
				"legacy_media_recovery_media_id":   fmt.Sprintf("media-budget-%d", i),
				"legacy_media_recovery_phone_id":   account.PhoneID,
				"legacy_media_recovery_expires_at": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339Nano),
			},
		}
		require.NoError(t, app.DB.Create(&message).Error)
		messageIDs = append(messageIDs, message.ID)
	}

	var wg sync.WaitGroup
	firstStatuses := make(chan int, 4)
	firstErrs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(messageID uuid.UUID) {
			defer wg.Done()
			req := testutil.NewGETRequest(t)
			testutil.SetAuthContext(req, org.ID, user.ID)
			testutil.SetPathParam(req, "message_id", messageID.String())
			firstErrs <- app.ServeMedia(req)
			firstStatuses <- testutil.GetResponseStatusCode(req)
		}(messageIDs[i])
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if restoreServer.downloadCalls.Load() == 4 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.Equal(t, int32(4), restoreServer.downloadCalls.Load())

	fifthReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(fifthReq, org.ID, user.ID)
	testutil.SetPathParam(fifthReq, "message_id", messageIDs[4].String())
	err := app.ServeMedia(fifthReq)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(fifthReq))

	close(release)
	wg.Wait()
	close(firstErrs)
	close(firstStatuses)

	for restoreErr := range firstErrs {
		require.NoError(t, restoreErr)
	}
	for status := range firstStatuses {
		assert.Equal(t, fasthttp.StatusOK, status)
	}
	assert.Equal(t, int32(4), restoreServer.urlCalls.Load())
	assert.Equal(t, int32(4), restoreServer.downloadCalls.Load())
}

func TestServeMedia_RollsBackRestoredFileWhenMessageUpdateFails(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	tempDir := t.TempDir()
	app.Config.Storage.LocalPath = tempDir

	restoreServer := newLegacyRestoreTestServer(t, map[string]legacyRestoreFixture{
		"media-rollback": {
			body:        []byte("%PDF-1.7\nrollback\n"),
			contentType: "application/pdf",
		},
	})
	app.WhatsApp = whatsapp.NewWithBaseURL(testutil.NopLogger(), restoreServer.server.URL)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: account.Name,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Rollback restore",
		MediaURL:        filepath.Join("documents", "rollback.pdf"),
		MediaMimeType:   "application/pdf",
		MediaFilename:   "rollback.pdf",
		Status:          models.MessageStatusReceived,
		Metadata: models.JSONB{
			"legacy_media_recovery_provider":   "meta",
			"legacy_media_recovery_media_id":   "media-rollback",
			"legacy_media_recovery_phone_id":   account.PhoneID,
			"legacy_media_recovery_expires_at": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339Nano),
		},
	}
	require.NoError(t, app.DB.Create(&message).Error)

	callbackName := "test_fail_legacy_media_restore_update"
	require.NoError(t, app.DB.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Schema == nil || tx.Statement.Schema.Table != "messages" {
			return
		}
		_ = tx.AddError(errors.New("forced restore update failure"))
	}))
	defer func() {
		require.NoError(t, app.DB.Callback().Update().Remove(callbackName))
	}()

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "message_id", message.ID.String())

	err := app.ServeMedia(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	assert.Equal(t, 0, countRegularFiles(t, tempDir))

	var refreshed models.Message
	require.NoError(t, app.DB.First(&refreshed, "id = ?", message.ID).Error)
	assert.Equal(t, message.MediaURL, refreshed.MediaURL)
}

func TestServeMedia_SelfHealingOnMissingObject(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	// Stub the ObjectStorage to return storage.ErrObjectNotFound
	storageMock := &handlerFakeStorage{
		getFunc: func(ctx context.Context, key string) (io.ReadCloser, objectstorage.ObjectInfo, error) {
			return nil, objectstorage.ObjectInfo{}, objectstorage.ErrObjectNotFound
		},
	}
	app.ObjectStorage = storageMock

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	asset := models.MediaAsset{
		BaseModel: models.BaseModel{ID: uuid.New()},
		FileHash:  "missing-self-heal-hash",
		S3Key:     "whatsmeow/media/mi/ss/missing-self-heal-hash",
		MimeType:  "application/pdf",
		Size:      123,
	}
	require.NoError(t, app.DB.Create(&asset).Error)

	message1 := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: "whatsmeow",
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Missing PDF Document 1",
		MediaAssetID:    &asset.ID,
		MediaURL:        "/api/media/" + uuid.NewString(),
		MediaMimeType:   "application/pdf",
		MediaFilename:   "missing-report-1.pdf",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&message1).Error)

	message2 := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: "whatsmeow",
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Missing PDF Document 2",
		MediaAssetID:    &asset.ID,
		MediaURL:        "/api/media/" + uuid.NewString(),
		MediaMimeType:   "application/pdf",
		MediaFilename:   "missing-report-2.pdf",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&message2).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "message_id", message1.ID.String())

	err := app.ServeMedia(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))

	// Verify that media_deleted_at was updated to a non-nil timestamp for BOTH messages in the DB
	var updatedMessage1 models.Message
	require.NoError(t, app.DB.First(&updatedMessage1, "id = ?", message1.ID).Error)
	assert.NotNil(t, updatedMessage1.MediaDeletedAt)

	var updatedMessage2 models.Message
	require.NoError(t, app.DB.First(&updatedMessage2, "id = ?", message2.ID).Error)
	assert.NotNil(t, updatedMessage2.MediaDeletedAt)

	// Verify that the MediaAsset is soft-deleted
	var count int64
	require.NoError(t, app.DB.Model(&models.MediaAsset{}).Where("id = ?", asset.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}
