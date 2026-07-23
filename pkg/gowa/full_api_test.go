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
	return &whatsapp.Account{ProviderType: "gowa", GowaDeviceID: "dev1"}
}

// --- Send extensions ---

func TestSendSticker_PostsToStickerEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	_, err := c.SendSticker(context.Background(), testAcct(), whatsapp.Recipient{Phone: "123"}, "https://example.com/s.webp")
	require.NoError(t, err)
	assert.Equal(t, "/send/sticker", mock.lastPath)
}

func TestSendContact_SendsNameAndPhoneInBody(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	_, err := c.SendContact(context.Background(), testAcct(), whatsapp.Recipient{Phone: "123"}, "John", "5551234")
	require.NoError(t, err)
	assert.Equal(t, "/send/contact", mock.lastPath)

	var body map[string]string
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "John", body["contact_name"])
	assert.Equal(t, "5551234", body["contact_phone"])
}

func TestSendLocation_SendsCoordinatesInBody(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	_, err := c.SendLocation(context.Background(), testAcct(), whatsapp.Recipient{Phone: "123"}, "37.7749", "-122.4194")
	require.NoError(t, err)
	assert.Equal(t, "/send/location", mock.lastPath)

	var body map[string]string
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "37.7749", body["latitude"])
}

func TestSendPoll_SendsQuestionAndMaxAnswer(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	_, err := c.SendPoll(context.Background(), testAcct(), whatsapp.Recipient{Phone: "123"}, "Pick one?", []string{"A", "B", "C"}, 1)
	require.NoError(t, err)
	assert.Equal(t, "/send/poll", mock.lastPath)

	var body map[string]any
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "Pick one?", body["question"])
	assert.Equal(t, float64(1), body["max_answer"])
}

func TestSendPresence_PostsToPresenceEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.SendPresence(context.Background(), testAcct(), "available")
	require.NoError(t, err)
	assert.Equal(t, "/send/presence", mock.lastPath)
}

func TestSendChatPresence_PostsToChatPresenceEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.SendChatPresence(context.Background(), testAcct(), "123@s.whatsapp.net", "start")
	require.NoError(t, err)
	assert.Equal(t, "/send/chat-presence", mock.lastPath)
}

func TestUpdateMessage_SendsEditedTextToCorrectEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.UpdateMessage(context.Background(), testAcct(), "MSG123", "123@s.whatsapp.net", "edited text")
	require.NoError(t, err)
	assert.Equal(t, "/message/MSG123/update", mock.lastPath)

	var body map[string]string
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "edited text", body["message"])
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

func TestGetUserInfo_ParsesVerifiedName(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"verified_name":"My Biz","status":"Online","picture_id":"p123"}}`
	c := gowa.New(mock.url(), "", "")

	info, err := c.GetUserInfo(context.Background(), "dev1")
	require.NoError(t, err)
	assert.Equal(t, "My Biz", info.VerifiedName)
}

func TestCheckUser_ParsesIsOnWhatsApp(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"is_on_whatsapp":true}}`
	c := gowa.New(mock.url(), "", "")

	onWA, err := c.CheckUser(context.Background(), "dev1", "16505551234")
	require.NoError(t, err)
	assert.True(t, onWA)
}

func TestSetPushName_PostsToPushnameEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.SetPushName(context.Background(), "dev1", "New Name")
	require.NoError(t, err)
	assert.Equal(t, "/user/pushname", mock.lastPath)
}

func TestGetMyContacts_ParsesContactList(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"data":[{"jid":"123@s.whatsapp.net","name":"Alice"}]}}`
	c := gowa.New(mock.url(), "", "")

	contacts, err := c.GetMyContacts(context.Background(), "dev1")
	require.NoError(t, err)
	require.Len(t, contacts, 1)
	assert.Equal(t, "Alice", contacts[0].Name)
}

func TestGetMyGroups_ParsesGroupList(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"data":[{"JID":"group@g.us","Name":"My Group"}]}}`
	c := gowa.New(mock.url(), "", "")

	groups, err := c.GetMyGroups(context.Background(), "dev1")
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, "My Group", groups[0].Name)
}

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

func TestCreateGroup_ReturnsGroupID(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"group_id":"newgroup@g.us"}}`
	c := gowa.New(mock.url(), "", "")

	groupID, err := c.CreateGroup(context.Background(), "dev1", "Test Group", []string{"123@s.whatsapp.net"})
	require.NoError(t, err)
	assert.Equal(t, "newgroup@g.us", groupID)
	assert.Equal(t, "/group", mock.lastPath)
}

func TestAddParticipants_ParsesParticipantStatus(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":[{"participant":"123@s.whatsapp.net","status":"success","message":"Added"}]}`
	c := gowa.New(mock.url(), "", "")

	results, err := c.AddParticipants(context.Background(), "dev1", "grp@g.us", []string{"123@s.whatsapp.net"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "success", results[0].Status)
}

func TestLeaveGroup_PostsToLeaveEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.LeaveGroup(context.Background(), "dev1", "grp@g.us")
	require.NoError(t, err)
	assert.Equal(t, "/group/leave", mock.lastPath)
}

func TestSetGroupName_PostsToNameEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.SetGroupName(context.Background(), "dev1", "grp@g.us", "New Name")
	require.NoError(t, err)
	assert.Equal(t, "/group/name", mock.lastPath)
}

func TestGetGroupInviteLink_ParsesInviteURL(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"invite_link":"https://chat.whatsapp.com/ABC","group_id":"grp@g.us"}}`
	c := gowa.New(mock.url(), "", "")

	link, err := c.GetGroupInviteLink(context.Background(), "dev1", "grp@g.us", false)
	require.NoError(t, err)
	assert.Equal(t, "https://chat.whatsapp.com/ABC", link.InviteLink)
}

func TestJoinGroupWithLink_PostsToJoinEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.JoinGroupWithLink(context.Background(), "dev1", "https://chat.whatsapp.com/ABC")
	require.NoError(t, err)
	assert.Equal(t, "/group/join-with-link", mock.lastPath)
}

// --- Chat operations ---

func TestListChats_ParsesChatsAndPagination(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"data":[{"jid":"123@s.whatsapp.net","name":"Alice"}],"pagination":{"limit":25,"offset":0,"total":1}}}`
	c := gowa.New(mock.url(), "", "")

	chats, page, err := c.ListChats(context.Background(), "dev1", gowa.ChatListParams{Limit: 25})
	require.NoError(t, err)
	require.Len(t, chats, 1)
	assert.Equal(t, "Alice", chats[0].Name)
	assert.Equal(t, 1, page.Total)
}

func TestGetChatHistory_ParsesMessageContent(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"data":[{"id":"MSG1","content":"Hello","is_from_me":false}],"pagination":{"limit":50,"offset":0,"total":1}}}`
	c := gowa.New(mock.url(), "", "")

	msgs, _, err := c.GetChatHistory(context.Background(), "dev1", "123@s.whatsapp.net", gowa.ChatHistoryParams{Limit: 50})
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "Hello", msgs[0].Content)
}

func TestPinChat_PostsToPinEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.PinChat(context.Background(), "dev1", "123@s.whatsapp.net", true)
	require.NoError(t, err)
	assert.Contains(t, mock.lastPath, "/pin")
}

func TestArchiveChat_PostsToArchiveEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.ArchiveChat(context.Background(), "dev1", "123@s.whatsapp.net", true)
	require.NoError(t, err)
	assert.Contains(t, mock.lastPath, "/archive")
}

func TestSetDisappearingTimer_PostsToDisappearingEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.SetDisappearingTimer(context.Background(), "dev1", "123@s.whatsapp.net", 86400)
	require.NoError(t, err)
	assert.Contains(t, mock.lastPath, "/disappearing")
}

// --- Call ---

func TestRejectCall_SendsCallerJIDAndCallID(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
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
