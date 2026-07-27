package gowa_test

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
)

// mockAPIServer is a reusable mock GOWA API server for all tests.
type mockAPIServer struct {
	*httptest.Server
	lastMethod  string
	lastPath    string
	lastBody    []byte
	lastHeaders http.Header
	respBody    string // customizable response body
}

func newMockAPIServer() *mockAPIServer {
	m := &mockAPIServer{}
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
		resp := m.respBody
		if resp == "" {
			resp = `{"code":"SUCCESS","message":"ok","results":{"message_id":"OK","status":"ok"}}`
		}
		w.Write([]byte(resp))
	}))
	return m
}

func (m *mockAPIServer) close()      { m.Server.Close() }
func (m *mockAPIServer) url() string { return m.Server.URL }

func testAcct() *whatsapp.Account {
	return &whatsapp.Account{GowaDeviceID: "dev1"}
}

// --- Send extensions ---

func TestSendChatPresence_PostsToChatPresenceEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.SendChatPresence(context.Background(), testAcct(), "123@s.whatsapp.net", "start")
	require.NoError(t, err)
	assert.Equal(t, "/send/chat-presence", mock.lastPath)
}

// --- Device management ---

func TestListDevices_ParsesDeviceArrayFromResults(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":[{"id":"dev1","display_name":"Phone","state":"connected","jid":"123@s.whatsapp.net"}]}`
	c := gowa.New(mock.url(), "", "")

	devices, err := c.ListDevices(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "dev1", devices[0].ID)
	assert.Equal(t, "connected", devices[0].State)
}

func TestCreateDevice_ReturnsNewDeviceFromResults(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"id":"dev2","display_name":"New","state":"connecting","jid":""}}`
	c := gowa.New(mock.url(), "", "")

	dev, err := c.CreateDevice(context.Background(), "test-device-id", gowa.WebhookConfig{WebhookURL: "http://callback"})
	require.NoError(t, err)
	assert.Equal(t, "dev2", dev.ID)
}

func TestDeleteDevice_SendsDeleteToCorrectPath(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.DeleteDevice(context.Background(), "dev2")
	require.NoError(t, err)
	assert.Equal(t, "DELETE", mock.lastMethod)
	assert.Equal(t, "/devices/dev2", mock.lastPath)
}

func TestGetDeviceStatus_ParsesConnectedState(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"device_id":"dev1","is_connected":true,"is_logged_in":true}}`
	c := gowa.New(mock.url(), "", "")

	status, err := c.GetDeviceStatus(context.Background(), "dev1")
	require.NoError(t, err)
	assert.True(t, status.IsConnected)
	assert.True(t, status.IsLoggedIn)
}

func TestLogoutDevice_PostsToLogoutPath(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.LogoutDevice(context.Background(), "dev1")
	require.NoError(t, err)
	assert.Equal(t, "/devices/dev1/logout", mock.lastPath)
}

func TestReconnectDevice_PostsToReconnectPath(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.ReconnectDevice(context.Background(), "dev1")
	require.NoError(t, err)
	assert.Equal(t, "/devices/dev1/reconnect", mock.lastPath)
}

func TestGetLoginQR_ParsesQRLink(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"qr_duration":60,"qr_link":"data:image/png;base64,ABC"}}`
	c := gowa.New(mock.url(), "", "")

	login, err := c.GetLoginQR(context.Background(), "dev1")
	require.NoError(t, err)
	assert.Equal(t, "data:image/png;base64,ABC", login.QRLink)
}

// --- User operations ---

func TestGetPrivacySettings_ParsesGroupAddSetting(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"group_add":"all","last_seen":"contacts","status":"all","profile":"contacts","read_receipts":"all"}}`
	c := gowa.New(mock.url(), "", "")

	privacy, err := c.GetPrivacySettings(context.Background(), "dev1")
	require.NoError(t, err)
	assert.Equal(t, "all", privacy.GroupAdd)
}

// --- Group management ---

func TestJoinGroupWithLink_PostsToJoinEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.JoinGroupWithLink(context.Background(), "dev1", "https://chat.whatsapp.com/ABC")
	require.NoError(t, err)
	assert.Equal(t, "/group/join-with-link", mock.lastPath)
}

// --- Call ---

func TestRejectCall_SendsCallerJIDAndCallID(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	// Real GOWA answers /call/reject with a GenericResponse — no message_id.
	mock.respBody = `{"code":"SUCCESS","message":"Success","results":null}`
	c := gowa.New(mock.url(), "", "")

	err := c.RejectCall(context.Background(), "dev1", "caller@s.whatsapp.net", "CALL_123")
	require.NoError(t, err)
	assert.Equal(t, "/call/reject", mock.lastPath)

	var body map[string]string
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "caller@s.whatsapp.net", body["caller_jid"])
	assert.Equal(t, "CALL_123", body["call_id"])
}

// --- Webhook config ---

func TestSetDeviceWebhook_SendsPatchWithWebhookURL(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"webhook_url":"http://cb","webhook_events":"message"}}`
	c := gowa.New(mock.url(), "", "")

	cfg, err := c.SetDeviceWebhook(context.Background(), "dev1", gowa.WebhookConfig{
		WebhookURL:    "http://cb",
		WebhookEvents: "message",
	})
	require.NoError(t, err)
	assert.Equal(t, "http://cb", cfg.WebhookURL)
	assert.Equal(t, "PATCH", mock.lastMethod)
}
