package gowa_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGowaServer creates a test HTTP server that captures the request and
// returns a standard GOWA send response.
type mockGowaServer struct {
	*httptest.Server
	lastMethod  string
	lastPath    string
	lastBody    []byte
	lastHeaders http.Header
}

func newMockGowaServer() *mockGowaServer {
	m := &mockGowaServer{}
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.lastMethod = r.Method
		m.lastPath = r.URL.Path
		m.lastHeaders = r.Header.Clone()
		m.lastBody, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    "SUCCESS",
			"message": "Success",
			"results": map[string]any{
				"message_id": "3EB0TESTMSGID",
				"status":     "ok",
			},
		})
	}))
	return m
}

func (m *mockGowaServer) close()      { m.Server.Close() }
func (m *mockGowaServer) url() string { return m.Server.URL }

func testAccount(deviceID string) *whatsapp.Account {
	return &whatsapp.Account{
		ProviderType: "gowa",
		GowaBaseURL:  "",
		GowaDeviceID: deviceID,
	}
}

func TestClient_SendTextMessage_Success(t *testing.T) {
	t.Parallel()
	mock := newMockGowaServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")
	ctx := context.Background()
	account := testAccount("dev1")

	msgID, err := c.SendTextMessage(ctx, account, whatsapp.Recipient{Phone: "16505551234"}, "Hello", "reply123")
	require.NoError(t, err)
	assert.Equal(t, "3EB0TESTMSGID", msgID)
	assert.Equal(t, "POST", mock.lastMethod)
	assert.Equal(t, "/send/message", mock.lastPath)
	assert.Equal(t, "dev1", mock.lastHeaders.Get("X-Device-Id"))

	// Verify body has phone as JID.
	var body map[string]any
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "16505551234@s.whatsapp.net", body["phone"])
	assert.Equal(t, "Hello", body["message"])
	assert.Equal(t, "reply123", body["reply_message_id"])
}

func TestClient_SendTextMessage_EmptyPhone(t *testing.T) {
	t.Parallel()
	mock := newMockGowaServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")
	ctx := context.Background()

	_, err := c.SendTextMessage(ctx, testAccount("dev1"), whatsapp.Recipient{Phone: ""}, "Hello")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "", body["phone"])
}

func TestClient_SendTextMessage_AlreadyJID(t *testing.T) {
	t.Parallel()
	mock := newMockGowaServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")

	jid := "16505551234@s.whatsapp.net"
	_, err := c.SendTextMessage(context.Background(), testAccount("dev1"), whatsapp.Recipient{Phone: jid}, "Hi")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, jid, body["phone"], "JID should be passed through unchanged")
}

func TestClient_SendImageMessage_WithCachedBytes(t *testing.T) {
	t.Parallel()
	mock := newMockGowaServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")
	ctx := context.Background()
	account := testAccount("dev1")

	// Upload media (caches bytes).
	imageData := []byte("fake-jpeg-data")
	mediaID, err := c.UploadMedia(ctx, account, imageData, "image/jpeg", "photo.jpg")
	require.NoError(t, err)

	// Send image using cached mediaID.
	msgID, err := c.SendImageMessage(ctx, account, whatsapp.Recipient{Phone: "16505551234"}, mediaID, "My caption", "")
	require.NoError(t, err)
	assert.Equal(t, "3EB0TESTMSGID", msgID)

	// Should have been sent as multipart, not JSON.
	assert.Contains(t, mock.lastHeaders.Get("Content-Type"), "multipart/form-data")
	assert.Equal(t, "/send/image", mock.lastPath)
}

func TestClient_SendImageMessage_WithURL(t *testing.T) {
	t.Parallel()
	mock := newMockGowaServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")

	// Pass a URL as mediaID (not in cache → treated as URL).
	url := "https://example.com/image.jpg"
	msgID, err := c.SendImageMessage(context.Background(), testAccount("dev1"), whatsapp.Recipient{Phone: "16505551234"}, url, "Caption", "")
	require.NoError(t, err)
	assert.Equal(t, "3EB0TESTMSGID", msgID)

	// Should have been sent as JSON with image_url field.
	assert.Contains(t, mock.lastHeaders.Get("Content-Type"), "application/json")

	var body map[string]any
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, url, body["image_url"])
	assert.Equal(t, "Caption", body["caption"])
	// reply_message_id must be omitted when not replying.
	_, hasReply := body["reply_message_id"]
	assert.False(t, hasReply, "reply_message_id must be omitted when empty")
}

// TestClient_SendImageMessage_WithReply verifies that the quoted-context
// (reply_message_id) is threaded through the JSON body for media replies —
// the silent-data-loss bug where media replies dropped the reply arg.
func TestClient_SendImageMessage_WithReply(t *testing.T) {
	t.Parallel()
	mock := newMockGowaServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")

	url := "https://example.com/image.jpg"
	msgID, err := c.SendImageMessage(context.Background(), testAccount("dev1"), whatsapp.Recipient{Phone: "16505551234"}, url, "Caption", "3EB0QUOTEDMSG")
	require.NoError(t, err)
	assert.Equal(t, "3EB0TESTMSGID", msgID)

	var body map[string]any
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "3EB0QUOTEDMSG", body["reply_message_id"])
}

func TestClient_SendDocumentMessage(t *testing.T) {
	t.Parallel()
	mock := newMockGowaServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")

	msgID, err := c.SendDocumentMessage(context.Background(), testAccount("dev1"), whatsapp.Recipient{Phone: "16505551234"}, "https://example.com/doc.pdf", "report.pdf", "See attached", "")
	require.NoError(t, err)
	assert.Equal(t, "3EB0TESTMSGID", msgID)
	assert.Equal(t, "/send/file", mock.lastPath)
}

func TestClient_SendAudioMessage(t *testing.T) {
	t.Parallel()
	mock := newMockGowaServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")

	msgID, err := c.SendAudioMessage(context.Background(), testAccount("dev1"), whatsapp.Recipient{Phone: "16505551234"}, "https://example.com/audio.ogg", "")
	require.NoError(t, err)
	assert.Equal(t, "3EB0TESTMSGID", msgID)
	assert.Equal(t, "/send/audio", mock.lastPath)
}

func TestClient_SendVideoMessage(t *testing.T) {
	t.Parallel()
	mock := newMockGowaServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")

	msgID, err := c.SendVideoMessage(context.Background(), testAccount("dev1"), whatsapp.Recipient{Phone: "16505551234"}, "https://example.com/video.mp4", "Video caption", "")
	require.NoError(t, err)
	assert.Equal(t, "3EB0TESTMSGID", msgID)
	assert.Equal(t, "/send/video", mock.lastPath)
}

func TestClient_SendInteractiveButtons_NotSupported(t *testing.T) {
	t.Parallel()
	c := gowa.New("http://unused", "", "")

	_, err := c.SendInteractiveButtons(context.Background(), testAccount("dev1"), whatsapp.Recipient{Phone: "16505551234"}, "body", []whatsapp.Button{{ID: "1", Title: "A"}})
	assert.ErrorIs(t, err, whatsapp.ErrNotSupported)
}

func TestClient_SendCTAURLButton_SendsLink(t *testing.T) {
	t.Parallel()
	mock := newMockGowaServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")

	msgID, err := c.SendCTAURLButton(context.Background(), testAccount("dev1"), whatsapp.Recipient{Phone: "16505551234"}, "Click here", "Visit", "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "3EB0TESTMSGID", msgID)
	assert.Equal(t, "/send/link", mock.lastPath)

	var body map[string]any
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "https://example.com", body["link"])
}

func TestClient_SendResponse_ErrorMessage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    400,
			"message": "Phone number is invalid",
			"results": nil,
		})
	}))
	defer server.Close()

	c := gowa.New(server.URL, "", "")

	_, err := c.SendTextMessage(context.Background(), testAccount("dev1"), whatsapp.Recipient{Phone: "bad"}, "Hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Phone number is invalid")
}

func TestClient_SendResponse_NoMessageID(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// results present but no message_id
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    "SUCCESS",
			"message": "ok",
			"results": map[string]any{},
		})
	}))
	defer server.Close()

	c := gowa.New(server.URL, "", "")

	_, err := c.SendTextMessage(context.Background(), testAccount("dev1"), whatsapp.Recipient{Phone: "16505551234"}, "Hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no message ID")
}

func TestClient_BasicAuth(t *testing.T) {
	t.Parallel()
	mock := newMockGowaServer()
	defer mock.close()

	c := gowa.New(mock.url(), "admin", "secretpass")
	_, err := c.SendTextMessage(context.Background(), testAccount("dev1"), whatsapp.Recipient{Phone: "16505551234"}, "Hello")
	require.NoError(t, err)

	// Parse Basic Auth from the Authorization header.
	authHeader := mock.lastHeaders.Get("Authorization")
	require.NotEmpty(t, authHeader, "Authorization header should be set")
	assert.Contains(t, authHeader, "Basic ", "should use Basic auth scheme")

	// Decode base64 credentials to verify username:password.
	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	assert.Equal(t, "admin:secretpass", string(decoded))
}
