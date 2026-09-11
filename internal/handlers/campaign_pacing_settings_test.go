package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// The send-pacing GET must report the STORED account value separately from
// the EFFECTIVE one (stored → config default → 0), plus which source won.
// The UI builds its toggle from `stored` — a server default must never look
// like an account override (that used to materialize the default as an
// explicit per-account value on first save).

type sendPacingResponse struct {
	Data struct {
		MessagesPerMinute int    `json:"messages_per_minute"`
		Stored            int    `json:"stored_messages_per_minute"`
		Effective         int    `json:"effective_messages_per_minute"`
		Source            string `json:"source"`
	} `json:"data"`
}

func getSendPacing(t *testing.T, app *handlers.App, orgID, userID uuid.UUID, accountID string) sendPacingResponse {
	t.Helper()
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgID, userID)
	testutil.SetPathParam(req, "id", accountID)
	require.NoError(t, app.GetSendPacingSettings(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	var resp sendPacingResponse
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	return resp
}

func putSendPacing(t *testing.T, app *handlers.App, orgID, userID uuid.UUID, accountID string, perMinute int) {
	t.Helper()
	req := testutil.NewJSONRequest(t, map[string]any{"messages_per_minute": perMinute})
	testutil.SetAuthContext(req, orgID, userID)
	testutil.SetPathParam(req, "id", accountID)
	require.NoError(t, app.UpdateSendPacingSettings(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
}

func TestApp_GetSendPacingSettings_ReportsStoredAndSource(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	// 1) No block, no config default → unlimited.
	resp := getSendPacing(t, app, org.ID, user.ID, account.ID.String())
	assert.Equal(t, "unlimited", resp.Data.Source)
	assert.Zero(t, resp.Data.Stored)
	assert.Zero(t, resp.Data.Effective)
	assert.Zero(t, resp.Data.MessagesPerMinute)

	// 2) No block, config default set → source "server", stored stays 0.
	app.Config.Campaigns.DefaultPacingPerMinute = 45
	resp = getSendPacing(t, app, org.ID, user.ID, account.ID.String())
	assert.Equal(t, "server", resp.Data.Source)
	assert.Zero(t, resp.Data.Stored, "a server default must not be reported as stored on the account")
	assert.Equal(t, 45, resp.Data.Effective)
	assert.Equal(t, 45, resp.Data.MessagesPerMinute, "legacy field keeps carrying the effective value")

	// 3) Explicit account override → source "account".
	putSendPacing(t, app, org.ID, user.ID, account.ID.String(), 30)
	resp = getSendPacing(t, app, org.ID, user.ID, account.ID.String())
	assert.Equal(t, "account", resp.Data.Source)
	assert.Equal(t, 30, resp.Data.Stored)
	assert.Equal(t, 30, resp.Data.Effective)
	assert.Equal(t, 30, resp.Data.MessagesPerMinute)

	// 4) Override removed (0) → back to the server default, stored 0.
	putSendPacing(t, app, org.ID, user.ID, account.ID.String(), 0)
	resp = getSendPacing(t, app, org.ID, user.ID, account.ID.String())
	assert.Equal(t, "server", resp.Data.Source)
	assert.Zero(t, resp.Data.Stored)
	assert.Equal(t, 45, resp.Data.Effective)
}
