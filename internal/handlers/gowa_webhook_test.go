package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

// nopLogger returns a no-op logger for tests.
func nopLogger() logf.Logger {
	return logf.New(logf.Opts{Level: logf.ErrorLevel})
}

// --- Provider routing: GOWA account routes to GOWA client ---

func TestE2E_GOWAAccountResolvesGOWAProvider(t *testing.T) {
	t.Parallel()
	mock := newMockGowaAPI()
	defer mock.close()

	whatsapp.RegisterGowaFactory(
		func(baseURL string) (string, string) { return "", "" },
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(baseURL, username, password)
		},
	)

	reg := whatsapp.NewRegistry(nopLogger())

	account := &whatsapp.Account{
		GowaBaseURL:  mock.url(),
		GowaDeviceID: "628123456789@s.whatsapp.net",
	}

	provider := reg.Get(account)
	_, ok := provider.(*gowa.Client)
	assert.True(t, ok, "GOWA account must route to GOWA provider")
	assert.False(t, provider.Capabilities().MediaUpload)
}

// --- Provider fallback: GOWA down does NOT silently fall back to Meta ---

func TestE2E_Registry_GOWADownReturnsErrorNotMetaFallback(t *testing.T) {
	t.Parallel()
	whatsapp.RegisterGowaFactory(
		func(baseURL string) (string, string) { return "", "" },
		func(baseURL, username, password string) whatsapp.Provider {
			return gowa.New(baseURL, username, password)
		},
	)

	reg := whatsapp.NewRegistry(nopLogger())

	account := &whatsapp.Account{
		GowaBaseURL:  "http://127.0.0.1:1", // port 1 = connection refused
		GowaDeviceID: "dev1",
	}

	provider := reg.Get(account)
	_, ok := provider.(*gowa.Client)
	require.True(t, ok, "must resolve to GOWA provider even when server is down")

	_, err := provider.SendTextMessage(
		context.Background(),
		account,
		whatsapp.Recipient{Phone: "16505551234"},
		"Hello",
	)
	require.Error(t, err, "GOWA send must fail when server is unreachable")
	assert.NotContains(t, err.Error(), "graph.facebook.com",
		"error must originate from GOWA, not a silent Meta fallback")
}

// --- E2E: send text via GOWA → verify HTTP call format ---

func TestE2E_GOWASendText_VerifiesHTTPFormat(t *testing.T) {
	t.Parallel()
	mock := newMockGowaAPI()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")
	account := &whatsapp.Account{
		GowaDeviceID: "628123456789@s.whatsapp.net",
	}

	msgID, err := c.SendTextMessage(
		context.Background(), account,
		whatsapp.Recipient{Phone: "16505551234"}, "Hello from GOWA", "reply123",
	)
	require.NoError(t, err)
	assert.Equal(t, "GOWA_MSG_001", msgID)
	assert.Equal(t, "POST", mock.lastMethod)
	assert.Equal(t, "/send/message", mock.lastPath)
	assert.Equal(t, "628123456789@s.whatsapp.net", mock.lastHeaders.Get("X-Device-Id"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "16505551234@s.whatsapp.net", body["phone"])
	assert.Equal(t, "Hello from GOWA", body["message"])
	assert.Equal(t, "reply123", body["reply_message_id"])
}

// --- E2E: send image via GOWA → verify multipart format (no two-step upload) ---

func TestE2E_GOWASendImage_MultipartNotTwoStep(t *testing.T) {
	t.Parallel()
	mock := newMockGowaAPI()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")
	account := &whatsapp.Account{
		GowaDeviceID: "dev1",
	}

	imageData := []byte{0x89, 0x50, 0x4E, 0x47}
	mediaID, err := c.UploadMedia(context.Background(), account, imageData, "image/png", "photo.png")
	require.NoError(t, err)

	msgID, err := c.SendImageMessage(
		context.Background(), account,
		whatsapp.Recipient{Phone: "16505551234"}, mediaID, "Image caption", "",
	)
	require.NoError(t, err)
	assert.Equal(t, "GOWA_MSG_001", msgID)
	assert.Contains(t, mock.lastHeaders.Get("Content-Type"), "multipart/form-data",
		"cached media bytes must be sent as multipart")
	assert.Equal(t, "/send/image", mock.lastPath)
}

// --- E2E: webhook text message normalization ---

func TestE2E_GOWAWebhook_TextMessageNormalization(t *testing.T) {
	t.Parallel()
	body := `{"event":"message","device_id":"628987654321@s.whatsapp.net","payload":{"id":"MSG_001","chat_id":"628987654321@s.whatsapp.net","from":"628123456789@s.whatsapp.net","from_name":"John Doe","timestamp":"2026-07-11T10:30:00Z","is_from_me":false,"body":"Hello from GOWA client"}}`

	var envelope gowa.WebhookPayload
	require.NoError(t, json.Unmarshal([]byte(body), &envelope))
	require.Equal(t, "message", envelope.Event)

	var msg gowa.MessagePayload
	require.NoError(t, json.Unmarshal(envelope.Payload, &msg))

	assert.Equal(t, "MSG_001", msg.ID)
	assert.Equal(t, "628123456789", gowa.PhoneFromJID(msg.From))
	assert.Equal(t, "John Doe", msg.FromName)
	assert.Equal(t, "Hello from GOWA client", msg.Body)
	assert.False(t, msg.IsFromMe)
	assert.False(t, msg.IsGroup())
}

// --- E2E: webhook image message normalization (polymorphic media) ---

func TestE2E_GOWAWebhook_ImageMessageNormalization(t *testing.T) {
	t.Parallel()
	body := `{"event":"message","device_id":"628987654321@s.whatsapp.net","payload":{"id":"MSG_002","chat_id":"628987654321@s.whatsapp.net","from":"628123456789@s.whatsapp.net","timestamp":"2026-07-11T10:31:00Z","body":"Check this image","image":{"path":"statics/media/photo.jpeg","caption":"Check this image"}}}`

	var envelope gowa.WebhookPayload
	require.NoError(t, json.Unmarshal([]byte(body), &envelope))

	var msg gowa.MessagePayload
	require.NoError(t, json.Unmarshal(envelope.Payload, &msg))

	mf := gowa.ResolveMediaField(msg.Image)
	assert.Equal(t, "statics/media/photo.jpeg", mf.URL)
	assert.Equal(t, "Check this image", mf.Caption)
	assert.Equal(t, msg.Body, mf.Caption)
}

// --- E2E: webhook ack event (batch of message IDs) ---

func TestE2E_GOWAWebhook_AckBatchIDs(t *testing.T) {
	t.Parallel()
	body := `{"event":"message.ack","device_id":"628987654321@s.whatsapp.net","timestamp":"2026-07-11T10:35:00Z","payload":{"ids":["MSG_001","MSG_002","MSG_003"],"chat_id":"628987654321@s.whatsapp.net","receipt_type":"read"}}`

	var envelope gowa.WebhookPayload
	require.NoError(t, json.Unmarshal([]byte(body), &envelope))

	var ack gowa.AckPayload
	require.NoError(t, json.Unmarshal(envelope.Payload, &ack))
	require.Len(t, ack.IDs, 3)
	assert.Equal(t, "read", ack.ReceiptType)
}

// --- E2E: webhook group message ---

func TestE2E_GOWAWebhook_GroupMessage(t *testing.T) {
	t.Parallel()
	body := `{"event":"message","device_id":"628987654321@s.whatsapp.net","payload":{"id":"GRP_MSG_001","chat_id":"120363402106AAAAA@g.us","from":"628123456789@s.whatsapp.net","from_name":"Group Member","timestamp":"2026-07-11T10:45:00Z","body":"Hello group"}}`

	var envelope gowa.WebhookPayload
	require.NoError(t, json.Unmarshal([]byte(body), &envelope))

	var msg gowa.MessagePayload
	require.NoError(t, json.Unmarshal(envelope.Payload, &msg))
	assert.True(t, msg.IsGroup())
	assert.NotEqual(t, msg.From, msg.ChatID)
}

// --- E2E: webhook connection event ---

func TestE2E_GOWAWebhook_ConnectionEvent(t *testing.T) {
	t.Parallel()
	body := `{"event":"connection","device_id":"628987654321@s.whatsapp.net","timestamp":"2026-07-11T10:40:00Z","payload":{"event":"connected","is_connected":true}}`

	var envelope gowa.WebhookPayload
	require.NoError(t, json.Unmarshal([]byte(body), &envelope))

	var conn gowa.ConnectionPayload
	require.NoError(t, json.Unmarshal(envelope.Payload, &conn))
	assert.Equal(t, gowa.ConnectionConnected, conn.Event)
	assert.True(t, conn.IsConnected)
}

// --- Mock GOWA API server ---

type mockGowaAPI struct {
	*httptest.Server
	lastMethod  string
	lastPath    string
	lastBody    []byte
	lastHeaders http.Header
}

func newMockGowaAPI() *mockGowaAPI {
	m := &mockGowaAPI{}
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.lastMethod = r.Method
		m.lastPath = r.URL.Path
		m.lastHeaders = r.Header.Clone()

		buf := make([]byte, 0, r.ContentLength)
		if r.Body != nil && r.ContentLength > 0 {
			tmp := make([]byte, r.ContentLength)
			n, _ := r.Body.Read(tmp)
			buf = tmp[:n]
		}
		m.lastBody = buf

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    "SUCCESS",
			"message": "Success",
			"results": map[string]any{
				"message_id": "GOWA_MSG_001",
				"status":     "ok",
			},
		})
	}))
	return m
}

func (m *mockGowaAPI) close()      { m.Server.Close() }
func (m *mockGowaAPI) url() string { return m.Server.URL }
