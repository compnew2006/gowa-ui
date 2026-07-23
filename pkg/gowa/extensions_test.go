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

func TestSendReaction_SendsEmojiAndPhone(t *testing.T) {
	t.Parallel()
	mock := newGowaExtMockServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")
	account := &whatsapp.Account{ProviderType: "gowa", GowaDeviceID: "dev1"}

	err := c.SendReaction(context.Background(), account, "MSG123", "628123@s.whatsapp.net", "👍")
	require.NoError(t, err)
	assert.Equal(t, "POST", mock.lastMethod)
	assert.Equal(t, "/message/MSG123/reaction", mock.lastPath)

	var body map[string]string
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "628123@s.whatsapp.net", body["phone"])
	assert.Equal(t, "👍", body["emoji"])
}

func TestRevokeMessage_PostsToRevokeEndpoint(t *testing.T) {
	t.Parallel()
	mock := newGowaExtMockServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")
	account := &whatsapp.Account{ProviderType: "gowa", GowaDeviceID: "dev1"}

	err := c.RevokeMessage(context.Background(), account, "MSG456", "628123@s.whatsapp.net")
	require.NoError(t, err)
	assert.Equal(t, "/message/MSG456/revoke", mock.lastPath)
}

func TestDeleteMessage_AppendsJIDSuffixToPlainPhone(t *testing.T) {
	t.Parallel()
	mock := newGowaExtMockServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")
	account := &whatsapp.Account{ProviderType: "gowa", GowaDeviceID: "dev1"}

	err := c.DeleteMessage(context.Background(), account, "MSG789", "628123")
	require.NoError(t, err)
	assert.Equal(t, "/message/MSG789/delete", mock.lastPath)

	var body map[string]string
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "628123@s.whatsapp.net", body["phone"], "plain phone gets JID suffix")
}

func TestStarMessage_PostsToStarEndpoint(t *testing.T) {
	t.Parallel()
	mock := newGowaExtMockServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")
	account := &whatsapp.Account{ProviderType: "gowa", GowaDeviceID: "dev1"}

	err := c.StarMessage(context.Background(), account, "MSG001", "628123@s.whatsapp.net")
	require.NoError(t, err)
	assert.Equal(t, "/message/MSG001/star", mock.lastPath)
}

func TestUnstarMessage_PostsToUnstarEndpoint(t *testing.T) {
	t.Parallel()
	mock := newGowaExtMockServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")
	account := &whatsapp.Account{ProviderType: "gowa", GowaDeviceID: "dev1"}

	err := c.UnstarMessage(context.Background(), account, "MSG001", "628123@s.whatsapp.net")
	require.NoError(t, err)
	assert.Equal(t, "/message/MSG001/unstar", mock.lastPath)
}

func TestForwardMessage_SendsTargetPhone(t *testing.T) {
	t.Parallel()
	mock := newGowaExtMockServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")
	account := &whatsapp.Account{ProviderType: "gowa", GowaDeviceID: "dev1"}

	err := c.ForwardMessage(context.Background(), account, "MSG999", "628999@s.whatsapp.net")
	require.NoError(t, err)
	assert.Equal(t, "/message/MSG999/forward", mock.lastPath)

	var body map[string]string
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "628999@s.whatsapp.net", body["phone"])
}

func TestMarkMessageRead_PassesJIDAsPhoneField(t *testing.T) {
	t.Parallel()
	mock := newGowaExtMockServer()
	defer mock.close()

	c := gowa.New(mock.url(), "", "")
	account := &whatsapp.Account{ProviderType: "gowa", GowaDeviceID: "dev1"}

	err := c.MarkMessageReadWithJID(context.Background(), account, "MSG111", "628123@s.whatsapp.net")
	require.NoError(t, err)
	assert.Equal(t, "/message/MSG111/read", mock.lastPath)

	var body map[string]string
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "628123@s.whatsapp.net", body["phone"], "JID must be passed as phone")
}

func TestSendReaction_PropagatesAPIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    400,
			"message": "Message not found",
			"results": nil,
		})
	}))
	defer server.Close()

	c := gowa.New(server.URL, "", "")
	account := &whatsapp.Account{ProviderType: "gowa", GowaDeviceID: "dev1"}

	err := c.SendReaction(context.Background(), account, "BADMSG", "628123@s.whatsapp.net", "❤️")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Message not found")
}

func TestWebhookPayload_ReactionEvent(t *testing.T) {
	t.Parallel()
	body := `{
		"event": "message.reaction",
		"device_id": "628987@s.whatsapp.net",
		"timestamp": "2026-07-11T12:00:00Z",
		"payload": {
			"reaction": "🙏",
			"reacted_message_id": "MSG_ORIGINAL_001",
			"from": "628123@s.whatsapp.net",
			"chat_id": "628123@s.whatsapp.net"
		}
	}`

	var envelope gowa.WebhookPayload
	require.NoError(t, json.Unmarshal([]byte(body), &envelope))
	assert.Equal(t, "message.reaction", envelope.Event)

	var react gowa.ReactionPayload
	require.NoError(t, json.Unmarshal(envelope.Payload, &react))
	assert.Equal(t, "🙏", react.Reaction)
	assert.Equal(t, "MSG_ORIGINAL_001", react.ReactedMessageID)
}

func TestWebhookPayload_RevokedEvent(t *testing.T) {
	t.Parallel()
	body := `{
		"event": "message.revoked",
		"device_id": "628987@s.whatsapp.net",
		"timestamp": "2026-07-11T12:01:00Z",
		"payload": {
			"revoked_message_id": "MSG_REVOKE_001"
		}
	}`

	var envelope gowa.WebhookPayload
	require.NoError(t, json.Unmarshal([]byte(body), &envelope))
	assert.Equal(t, "message.revoked", envelope.Event)

	var revoked gowa.RevokedPayload
	require.NoError(t, json.Unmarshal(envelope.Payload, &revoked))
	assert.Equal(t, "MSG_REVOKE_001", revoked.RevokedMessageID)
}

// --- Mock ---

type gowaExtMock struct {
	*httptest.Server
	lastMethod  string
	lastPath    string
	lastBody    []byte
	lastHeaders http.Header
}

func newGowaExtMock() *gowaExtMock {
	m := &gowaExtMock{}
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
			"results": map[string]any{"message_id": "OK", "status": "ok"},
		})
	}))
	return m
}

func (m *gowaExtMock) close()      { m.Server.Close() }
func (m *gowaExtMock) url() string { return m.Server.URL }

// Alias for test naming consistency.
func newGowaExtMockServer() *gowaExtMock { return newGowaExtMock() }
