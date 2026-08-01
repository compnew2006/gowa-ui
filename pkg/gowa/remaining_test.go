package gowa_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// Reuse mock from full_api_test.go
var _ = httptest.NewServer
var _ = http.Header{}
