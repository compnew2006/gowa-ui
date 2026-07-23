package gowa_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck_PostsToHealthEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.HealthCheck(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/health", mock.lastPath)
}

func TestLoginWithCode_ReturnsPairCode(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"pair_code":"ABCD-1234"}}`
	c := gowa.New(mock.url(), "", "")

	resp, err := c.LoginWithCode(context.Background(), "dev1", "628123456789")
	require.NoError(t, err)
	assert.Equal(t, "ABCD-1234", resp.PairCode)
	assert.Equal(t, "/app/login-with-code", mock.lastPath)
	// The phone is sent as a query param, not in the path.
	// The mock captures r.URL.Path only (no query string), so we verify
	// the body wasn't sent (it's a GET with query params).
	assert.Equal(t, "GET", mock.lastMethod)
}

func TestGetPasskeyStatus_ParsesStatusField(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"device_id":"dev1","status":"none"}}`
	c := gowa.New(mock.url(), "", "")

	status, err := c.GetPasskeyStatus(context.Background(), "dev1")
	require.NoError(t, err)
	assert.Equal(t, "none", status.Status)
}

func TestSubmitPasskeyResponse_PostsAssertion(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	assertion := gowa.WebAuthnAssertion{ID: "cred1", RawID: "cred1", Type: "public-key"}
	err := c.SubmitPasskeyResponse(context.Background(), "dev1", assertion)
	require.NoError(t, err)
	assert.Equal(t, "/app/passkey/response", mock.lastPath)
}

func TestConfirmPasskey_PostsToConfirmEndpoint(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.ConfirmPasskey(context.Background(), "dev1")
	require.NoError(t, err)
	assert.Equal(t, "/app/passkey/confirm", mock.lastPath)
}

func TestAppLogout_SendsGetToLogoutPath(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.AppLogout(context.Background(), "dev1")
	require.NoError(t, err)
	assert.Equal(t, "GET", mock.lastMethod)
	assert.Equal(t, "/app/logout", mock.lastPath)
}

func TestAppReconnect_SendsGetToReconnectPath(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.AppReconnect(context.Background(), "dev1")
	require.NoError(t, err)
	assert.Equal(t, "GET", mock.lastMethod)
	assert.Equal(t, "/app/reconnect", mock.lastPath)
}

func TestAppListDevices_ParsesDeviceArray(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":[{"name":"Alice","device":"dev1","jid":"alice@s.whatsapp.net"}]}`
	c := gowa.New(mock.url(), "", "")

	devices, err := c.AppListDevices(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "Alice", devices[0].Name)
	assert.Equal(t, "dev1", devices[0].Device)
}

func TestGetAppStatus_ParsesJIDAndConnectionState(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"is_connected":true,"is_logged_in":true,"device_id":"dev1","jid":"628123@s.whatsapp.net"}}`
	c := gowa.New(mock.url(), "", "")

	status, err := c.GetAppStatus(context.Background(), "dev1")
	require.NoError(t, err)
	assert.True(t, status.IsConnected)
	assert.Equal(t, "628123@s.whatsapp.net", status.JID)
}

func TestListGroupParticipants_ParsesAdminFlag(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"group_id":"grp@g.us","name":"My Group","participants":[{"jid":"alice@s.whatsapp.net","is_admin":true}]}}`
	c := gowa.New(mock.url(), "", "")

	result, err := c.ListGroupParticipants(context.Background(), "dev1", "grp@g.us")
	require.NoError(t, err)
	assert.Equal(t, "My Group", result.Name)
	require.Len(t, result.Participants, 1)
	assert.True(t, result.Participants[0].IsAdmin)
}

func TestExportGroupParticipants_ParsesCSVOutput(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `jid,name,admin
alice@s.whatsapp.net,Alice,true`
	c := gowa.New(mock.url(), "", "")

	csvData, err := c.ExportGroupParticipants(context.Background(), "dev1", "grp@g.us")
	require.NoError(t, err)
	assert.Contains(t, string(csvData), "alice@s.whatsapp.net")
	assert.Contains(t, mock.lastPath, "/group/participants/export")
}

func TestSetGroupPhoto_SendsMultipartUpload(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		photoData       []byte
		filename        string
		expectMultipart bool
	}{
		{
			name:            "sends multipart with photo data",
			photoData:       []byte{0xFF, 0xD8, 0xFF, 0xE0}, // JPEG header
			filename:        "photo.jpg",
			expectMultipart: true,
		},
		{
			name:            "empty data removes photo",
			photoData:       nil,
			filename:        "",
			expectMultipart: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mock := newMockAPIServer()
			defer mock.close()
			c := gowa.New(mock.url(), "", "")

			_, err := c.SetGroupPhoto(context.Background(), "dev1", "grp@g.us", tc.photoData, tc.filename)
			require.NoError(t, err)
			assert.Equal(t, "/group/photo", mock.lastPath)
			if tc.expectMultipart {
				assert.Contains(t, mock.lastHeaders.Get("Content-Type"), "multipart")
			}
		})
	}
}

func TestUnfollowNewsletter_SendsNewsletterID(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	err := c.UnfollowNewsletter(context.Background(), "dev1", "120363@newsletter")
	require.NoError(t, err)
	assert.Equal(t, "/newsletter/unfollow", mock.lastPath)

	var body map[string]string
	require.NoError(t, json.Unmarshal(mock.lastBody, &body))
	assert.Equal(t, "120363@newsletter", body["newsletter_id"])
}

func TestGetNewsletterMessages_ParsesMessageText(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	mock.respBody = `{"results":{"data":[{"server_id":1,"message_id":"MSG1","type":"text","text":"Hello"}]}}`
	c := gowa.New(mock.url(), "", "")

	msgs, err := c.GetNewsletterMessages(context.Background(), "dev1", "120363@newsletter", 10)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "Hello", msgs[0].Text)
	assert.Equal(t, "/newsletter/messages", mock.lastPath)
	assert.Equal(t, "GET", mock.lastMethod)
}

func TestSetUserAvatar_SendsMultipartUpload(t *testing.T) {
	t.Parallel()
	mock := newMockAPIServer()
	defer mock.close()
	c := gowa.New(mock.url(), "", "")

	photoData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	err := c.SetUserAvatar(context.Background(), "dev1", photoData, "me.jpg")
	require.NoError(t, err)
	assert.Equal(t, "/user/avatar", mock.lastPath)
	assert.Contains(t, mock.lastHeaders.Get("Content-Type"), "multipart")
}

// Reuse mock from full_api_test.go
var _ = httptest.NewServer
var _ = http.Header{}
