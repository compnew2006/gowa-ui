package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/gowa-ui/internal/handlers"
	"github.com/shridarpatil/gowa-ui/internal/models"
	"github.com/shridarpatil/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// --- GetCloseRatingSettings Tests ---

func TestApp_GetCloseRatingSettings_Defaults(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.GetCloseRatingSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.CloseRatingSettingsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	assert.False(t, resp.Data.Enabled)
	assert.Equal(t, 48, resp.Data.WindowHours)
	assert.NotEmpty(t, resp.Data.Prompt)
	assert.NotEmpty(t, resp.Data.Thanks)
	assert.NotNil(t, resp.Data.Lexicon)
}

func TestApp_GetCloseRatingSettings_AccountOverrides(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	settings := models.JSONB{
		"close_rating": map[string]any{
			"enabled":      true,
			"window_hours": float64(24),
			"prompt":       "Rate us!",
			"thanks":       "",
			"lexicon":      map[string]any{"يجنن": float64(5)},
		},
	}
	require.NoError(t, app.DB.Model(account).Update("settings", settings).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.GetCloseRatingSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.CloseRatingSettingsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	assert.True(t, resp.Data.Enabled)
	assert.Equal(t, 24, resp.Data.WindowHours)
	assert.Equal(t, "Rate us!", resp.Data.Prompt)
	assert.Empty(t, resp.Data.Thanks) // explicit empty disables the thank-you
	assert.Equal(t, 5, resp.Data.Lexicon["يجنن"])
}

func TestApp_GetCloseRatingSettings_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewGETRequest(t)
	// No auth context set

	err := app.GetCloseRatingSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_GetCloseRatingSettings_NotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.GetCloseRatingSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

// --- UpdateCloseRatingSettings Tests ---

func TestApp_UpdateCloseRatingSettings_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"enabled":      true,
		"window_hours": 72,
		"prompt":       "قيّمنا من فضلك",
		"thanks":       "شكراً",
		"lexicon":      map[string]int{"يجنن": 5, " ": 3},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.UpdateCloseRatingSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var updated models.WhatsAppAccount
	require.NoError(t, app.DB.Where("id = ?", account.ID).First(&updated).Error)

	block, ok := updated.Settings["close_rating"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, block["enabled"])
	assert.Equal(t, float64(72), block["window_hours"])
	assert.Equal(t, "قيّمنا من فضلك", block["prompt"])
	assert.Equal(t, "شكراً", block["thanks"])
	lexicon, ok := block["lexicon"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(5), lexicon["يجنن"])
	assert.Len(t, lexicon, 1) // blank word dropped

	// Settings on another account must stay untouched (per-account isolation).
	other := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	var otherReloaded models.WhatsAppAccount
	require.NoError(t, app.DB.Where("id = ?", other.ID).First(&otherReloaded).Error)
	_, has := otherReloaded.Settings["close_rating"]
	assert.False(t, has)
}

func TestApp_UpdateCloseRatingSettings_Validation(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	tests := []struct {
		name string
		body map[string]any
	}{
		{"window too small", map[string]any{"enabled": true, "window_hours": 0}},
		{"window too large", map[string]any{"enabled": true, "window_hours": 721}},
		{"lexicon rating out of range", map[string]any{
			"enabled": true, "window_hours": 48, "lexicon": map[string]int{"bad": 6},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.NewJSONRequest(t, tt.body)
			testutil.SetAuthContext(req, org.ID, user.ID)
			testutil.SetPathParam(req, "id", account.ID.String())

			err := app.UpdateCloseRatingSettings(req)
			require.NoError(t, err)
			assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
		})
	}
}

func TestApp_UpdateCloseRatingSettings_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]any{"enabled": true, "window_hours": 48})
	// No auth context set

	err := app.UpdateCloseRatingSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

// --- GetCloseRatingStats Tests ---

func TestApp_GetCloseRatingStats(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	otherAccount := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	// Seed cycles directly: 2 rated (5, 4), 1 pending, 1 expired for the
	// account; 1 rated cycle on a sibling account that must not leak in.
	five, four, three := 5, 4, 3
	now := time.Now()
	seed := []models.ChatClosureRating{
		{OrganizationID: org.ID, WhatsAppAccount: account.Name, ContactID: testutil.CreateTestContact(t, app.DB, org.ID).ID,
			Status: models.RatingStatusRated, Rating: &five, RatedAt: &now, ExpiresAt: now.Add(time.Hour)},
		{OrganizationID: org.ID, WhatsAppAccount: account.Name, ContactID: testutil.CreateTestContact(t, app.DB, org.ID).ID,
			Status: models.RatingStatusRated, Rating: &four, RatedAt: &now, ExpiresAt: now.Add(time.Hour)},
		{OrganizationID: org.ID, WhatsAppAccount: account.Name, ContactID: testutil.CreateTestContact(t, app.DB, org.ID).ID,
			Status: models.RatingStatusPending, ExpiresAt: now.Add(time.Hour)},
		{OrganizationID: org.ID, WhatsAppAccount: account.Name, ContactID: testutil.CreateTestContact(t, app.DB, org.ID).ID,
			Status: models.RatingStatusExpired, ExpiresAt: now.Add(-time.Hour)},
		{OrganizationID: org.ID, WhatsAppAccount: otherAccount.Name, ContactID: testutil.CreateTestContact(t, app.DB, org.ID).ID,
			Status: models.RatingStatusRated, Rating: &three, RatedAt: &now, ExpiresAt: now.Add(time.Hour)},
	}
	for i := range seed {
		require.NoError(t, app.DB.Create(&seed[i]).Error)
	}

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.GetCloseRatingStats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.CloseRatingStatsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	assert.Equal(t, int64(4), resp.Data.Total)
	assert.Equal(t, int64(2), resp.Data.Rated)
	assert.Equal(t, int64(1), resp.Data.Pending)
	assert.Equal(t, int64(1), resp.Data.Expired)
	assert.InDelta(t, 4.5, resp.Data.Average, 0.001)
	assert.InDelta(t, 50.0, resp.Data.ResponseRate, 0.001)
	assert.Equal(t, int64(1), resp.Data.Distribution["5"])
	assert.Equal(t, int64(1), resp.Data.Distribution["4"])
	assert.Equal(t, int64(0), resp.Data.Distribution["3"])
}

func TestApp_GetCloseRatingStats_Empty(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.GetCloseRatingStats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.CloseRatingStatsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	assert.Equal(t, int64(0), resp.Data.Total)
	assert.Equal(t, float64(0), resp.Data.Average)
	assert.Equal(t, float64(0), resp.Data.ResponseRate)
}
