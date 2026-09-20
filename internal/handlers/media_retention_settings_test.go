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

// media-retention settings: the block validates its day range only when
// retention is ENABLED (a disabled block may keep its last days value so a
// toggle off/on round-trip doesn't lose the configuration), and the GET
// reports `effective` — the same condition the retention processor scans on.

type mediaRetentionResponse struct {
	Data struct {
		Enabled       bool `json:"enabled"`
		RetentionDays int  `json:"retention_days"`
		Effective     bool `json:"effective"`
	} `json:"data"`
}

func getMediaRetention(t *testing.T, app *handlers.App, orgID, userID uuid.UUID, accountID string) mediaRetentionResponse {
	t.Helper()
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgID, userID)
	testutil.SetPathParam(req, "id", accountID)
	require.NoError(t, app.GetMediaRetentionSettings(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	var resp mediaRetentionResponse
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	return resp
}

func putMediaRetention(t *testing.T, app *handlers.App, orgID, userID uuid.UUID, accountID string, body map[string]any) int {
	t.Helper()
	req := testutil.NewJSONRequest(t, body)
	testutil.SetAuthContext(req, orgID, userID)
	testutil.SetPathParam(req, "id", accountID)
	require.NoError(t, app.UpdateMediaRetentionSettings(req))
	return testutil.GetResponseStatusCode(req)
}

func TestApp_MediaRetentionSettings_DefaultsToDisabled(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	resp := getMediaRetention(t, app, org.ID, user.ID, account.ID.String())
	assert.False(t, resp.Data.Enabled)
	assert.Zero(t, resp.Data.RetentionDays)
	assert.False(t, resp.Data.Effective, "no block must mean no retention")
}

func TestApp_MediaRetentionSettings_RoundTripAndEffective(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	require.Equal(t, fasthttp.StatusOK,
		putMediaRetention(t, app, org.ID, user.ID, account.ID.String(), map[string]any{
			"enabled": true, "retention_days": 30,
		}))

	resp := getMediaRetention(t, app, org.ID, user.ID, account.ID.String())
	assert.True(t, resp.Data.Enabled)
	assert.Equal(t, 30, resp.Data.RetentionDays)
	assert.True(t, resp.Data.Effective)

	// Disabling keeps the stored days (so re-enabling doesn't lose them) but
	// is not effective.
	require.Equal(t, fasthttp.StatusOK,
		putMediaRetention(t, app, org.ID, user.ID, account.ID.String(), map[string]any{
			"enabled": false, "retention_days": 30,
		}))
	resp = getMediaRetention(t, app, org.ID, user.ID, account.ID.String())
	assert.False(t, resp.Data.Enabled)
	assert.Equal(t, 30, resp.Data.RetentionDays)
	assert.False(t, resp.Data.Effective)
}

func TestApp_MediaRetentionSettings_Validation(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"enabled with zero days", map[string]any{"enabled": true, "retention_days": 0}, fasthttp.StatusBadRequest},
		{"enabled with negative days", map[string]any{"enabled": true, "retention_days": -5}, fasthttp.StatusBadRequest},
		{"enabled with days over max", map[string]any{"enabled": true, "retention_days": 3651}, fasthttp.StatusBadRequest},
		{"enabled at lower bound", map[string]any{"enabled": true, "retention_days": 1}, fasthttp.StatusOK},
		{"enabled at upper bound", map[string]any{"enabled": true, "retention_days": 3650}, fasthttp.StatusOK},
		{"disabled with zero days", map[string]any{"enabled": false, "retention_days": 0}, fasthttp.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := putMediaRetention(t, app, org.ID, user.ID, account.ID.String(), tc.body)
			assert.Equal(t, tc.want, got)
		})
	}
}
