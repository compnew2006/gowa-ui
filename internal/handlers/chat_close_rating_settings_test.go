package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
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

// --- ListCloseRatingResponses Tests ---

// closeRatingResponsesBody decodes the list envelope: {"data": {"responses": ...}}.
type closeRatingResponsesBody struct {
	Data struct {
		Responses []handlers.CloseRatingResponseItem `json:"responses"`
		Total     int64                              `json:"total"`
		Page      int                                `json:"page"`
		Limit     int                                `json:"limit"`
	} `json:"data"`
}

// listCloseRatingResponses calls ListCloseRatingResponses as the given user.
// The body is decoded only for 200 responses.
func listCloseRatingResponses(t *testing.T, app *handlers.App, orgID, userID, accountID uuid.UUID, query map[string]string) (int, closeRatingResponsesBody) {
	t.Helper()

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgID, userID)
	testutil.SetPathParam(req, "id", accountID.String())
	for k, v := range query {
		testutil.SetQueryParam(req, k, v)
	}

	require.NoError(t, app.ListCloseRatingResponses(req))
	status := testutil.GetResponseStatusCode(req)

	var body closeRatingResponsesBody
	if status == fasthttp.StatusOK {
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &body))
	}
	return status, body
}

// seedRatingContact creates a contact with a fixed phone number and profile name.
func seedRatingContact(t *testing.T, app *handlers.App, orgID uuid.UUID, phone, name string) *models.Contact {
	t.Helper()

	contact := &models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		PhoneNumber:    phone,
		ProfileName:    name,
	}
	require.NoError(t, app.DB.Create(contact).Error)
	return contact
}

// createRatingCycle stores one CSAT cycle and returns it with its generated ID.
func createRatingCycle(t *testing.T, app *handlers.App, cycle models.ChatClosureRating) models.ChatClosureRating {
	t.Helper()

	if cycle.ExpiresAt.IsZero() {
		cycle.ExpiresAt = time.Now().Add(time.Hour)
	}
	require.NoError(t, app.DB.Create(&cycle).Error)
	return cycle
}

// responseIDs returns the cycle IDs in response order.
func responseIDs(items []handlers.CloseRatingResponseItem) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

// closeRatingFixture is one org whose account holds three rated cycles (sara
// newest, ahmed, omar with no rated_at), one pending and one expired cycle.
// Decoys on a sibling account and in another org must never appear in its list.
type closeRatingFixture struct {
	org     *models.Organization
	user    *models.User
	closer  *models.User
	account *models.WhatsAppAccount
	sara    models.ChatClosureRating
	ahmed   models.ChatClosureRating
	omar    models.ChatClosureRating
	pending models.ChatClosureRating
	expired models.ChatClosureRating
}

func seedCloseRatingFixture(t *testing.T, app *handlers.App) *closeRatingFixture {
	t.Helper()

	org := testutil.CreateTestOrganization(t, app.DB)
	otherOrg := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	sibling := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	f := &closeRatingFixture{
		org:     org,
		user:    createAdminUser(t, app, org.ID),
		closer:  testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithFullName("Mona Closer")),
		account: account,
	}

	five, four, three := 5, 4, 3
	now := time.Now()
	older, newer := now.Add(-2*time.Hour), now.Add(-time.Hour)

	ahmed := seedRatingContact(t, app, org.ID, "+201000000001", "Ahmed")
	sara := seedRatingContact(t, app, org.ID, "+201000000002", "Sara")
	omar := seedRatingContact(t, app, org.ID, "+201000000003", "Omar")
	pending := seedRatingContact(t, app, org.ID, "+201000000004", "Pending Customer")
	expired := seedRatingContact(t, app, org.ID, "+201000000005", "Expired Customer")
	siblingContact := seedRatingContact(t, app, org.ID, "+201000000006", "Sibling Customer")
	foreignContact := seedRatingContact(t, app, otherOrg.ID, "+201000000007", "Foreign Customer")

	f.ahmed = createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: ahmed.ID, WhatsAppAccount: account.Name,
		ClosedByUserID: &f.closer.ID, Status: models.RatingStatusRated, Rating: &four,
		RatingSource: models.RatingSourceExplicit, RawReply: "4", RatedAt: &older,
	})
	f.sara = createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: sara.ID, WhatsAppAccount: account.Name,
		Status: models.RatingStatusRated, Rating: &five,
		RatingSource: models.RatingSourceExplicit, RawReply: "ممتاز", RatedAt: &newer,
	})
	// A rated row without rated_at: NULLS LAST must keep it after the dated rows.
	f.omar = createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: omar.ID, WhatsAppAccount: account.Name,
		Status: models.RatingStatusRated, Rating: &three, RatingSource: models.RatingSourceExplicit,
	})
	f.pending = createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: pending.ID, WhatsAppAccount: account.Name,
		Status: models.RatingStatusPending,
	})
	f.expired = createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: expired.ID, WhatsAppAccount: account.Name,
		Status: models.RatingStatusExpired,
	})

	// Decoys: a sibling account's rated cycle and another org's cycle on the same name.
	createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: siblingContact.ID, WhatsAppAccount: sibling.Name,
		Status: models.RatingStatusRated, Rating: &five, RatedAt: &newer,
	})
	createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: otherOrg.ID, ContactID: foreignContact.ID, WhatsAppAccount: account.Name,
		Status: models.RatingStatusRated, Rating: &five, RatedAt: &newer,
	})

	return f
}

func TestApp_ListCloseRatingResponses_RatedNewestFirst(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	f := seedCloseRatingFixture(t, app)

	status, body := listCloseRatingResponses(t, app, f.org.ID, f.user.ID, f.account.ID, nil)
	require.Equal(t, fasthttp.StatusOK, status)

	assert.Equal(t, int64(3), body.Data.Total)
	assert.Equal(t, 1, body.Data.Page)
	assert.Equal(t, 50, body.Data.Limit)
	require.Len(t, body.Data.Responses, 3)
	assert.Equal(t, []uuid.UUID{f.sara.ID, f.ahmed.ID, f.omar.ID}, responseIDs(body.Data.Responses))

	sara := body.Data.Responses[0]
	assert.Equal(t, "Sara", sara.ContactName)
	assert.Equal(t, "+201000000002", sara.PhoneNumber)
	assert.Equal(t, f.account.Name, sara.WhatsAppAccount)
	assert.Equal(t, models.RatingStatusRated, sara.Status)
	require.NotNil(t, sara.Rating)
	assert.Equal(t, 5, *sara.Rating)
	assert.Equal(t, models.RatingSourceExplicit, sara.RatingSource)
	assert.Equal(t, "ممتاز", sara.RawReply)
	assert.Nil(t, sara.ClosedByUserID)
	assert.Empty(t, sara.ClosedByUserName)

	ahmed := body.Data.Responses[1]
	assert.Equal(t, "Ahmed", ahmed.ContactName)
	assert.Equal(t, "4", ahmed.RawReply)
	require.NotNil(t, ahmed.Rating)
	assert.Equal(t, 4, *ahmed.Rating)
	require.NotNil(t, ahmed.ClosedByUserID)
	assert.Equal(t, f.closer.ID, *ahmed.ClosedByUserID)
	assert.Equal(t, "Mona Closer", ahmed.ClosedByUserName)

	omar := body.Data.Responses[2]
	assert.Nil(t, omar.RatedAt)
	require.NotNil(t, omar.Rating)
	assert.Equal(t, 3, *omar.Rating)
}

func TestApp_ListCloseRatingResponses_Pagination(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	f := seedCloseRatingFixture(t, app)

	tests := []struct {
		name    string
		query   map[string]string
		wantIDs []uuid.UUID
	}{
		{"first page of two", map[string]string{"page": "1", "limit": "2"}, []uuid.UUID{f.sara.ID, f.ahmed.ID}},
		{"second page of two", map[string]string{"page": "2", "limit": "2"}, []uuid.UUID{f.omar.ID}},
		{"one per page, second", map[string]string{"page": "2", "limit": "1"}, []uuid.UUID{f.ahmed.ID}},
		{"page past the end", map[string]string{"page": "9", "limit": "1"}, []uuid.UUID{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := listCloseRatingResponses(t, app, f.org.ID, f.user.ID, f.account.ID, tt.query)
			require.Equal(t, fasthttp.StatusOK, status)

			assert.Equal(t, int64(3), body.Data.Total, "total ignores pagination")
			assert.Equal(t, tt.wantIDs, responseIDs(body.Data.Responses))
		})
	}
}

func TestApp_ListCloseRatingResponses_Filters(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	f := seedCloseRatingFixture(t, app)

	tests := []struct {
		name    string
		query   map[string]string
		wantIDs []uuid.UUID
	}{
		{"pending", map[string]string{"status": "pending"}, []uuid.UUID{f.pending.ID}},
		{"expired", map[string]string{"status": "expired"}, []uuid.UUID{f.expired.ID}},
		{"rated, five stars", map[string]string{"status": "rated", "rating": "5"}, []uuid.UUID{f.sara.ID}},
		{"rating alone implies rated", map[string]string{"rating": "4"}, []uuid.UUID{f.ahmed.ID}},
		{"rated, three stars", map[string]string{"rating": "3"}, []uuid.UUID{f.omar.ID}},
		{"pending has no rating to match", map[string]string{"status": "pending", "rating": "5"}, []uuid.UUID{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := listCloseRatingResponses(t, app, f.org.ID, f.user.ID, f.account.ID, tt.query)
			require.Equal(t, fasthttp.StatusOK, status)

			assert.Equal(t, int64(len(tt.wantIDs)), body.Data.Total)
			assert.Equal(t, tt.wantIDs, responseIDs(body.Data.Responses))
		})
	}
}

func TestApp_ListCloseRatingResponses_InvalidFilters(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	f := seedCloseRatingFixture(t, app)

	tests := []struct {
		name  string
		query map[string]string
	}{
		{"unknown status", map[string]string{"status": "bogus"}},
		{"uppercase status", map[string]string{"status": "RATED"}},
		{"rating above range", map[string]string{"rating": "9"}},
		{"rating zero", map[string]string{"rating": "0"}},
		{"rating not an integer", map[string]string{"rating": "abc"}},
		{"rating fractional", map[string]string{"rating": "3.5"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _ := listCloseRatingResponses(t, app, f.org.ID, f.user.ID, f.account.ID, tt.query)
			assert.Equal(t, fasthttp.StatusBadRequest, status)
		})
	}
}

func TestApp_ListCloseRatingResponses_AccountNotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	f := seedCloseRatingFixture(t, app)
	otherOrg := testutil.CreateTestOrganization(t, app.DB)
	otherAccount := testutil.CreateTestWhatsAppAccount(t, app.DB, otherOrg.ID)

	tests := []struct {
		name      string
		accountID uuid.UUID
	}{
		{"unknown account", uuid.New()},
		{"account in another org", otherAccount.ID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _ := listCloseRatingResponses(t, app, f.org.ID, f.user.ID, tt.accountID, nil)
			assert.Equal(t, fasthttp.StatusNotFound, status)
		})
	}
}

func TestApp_ListCloseRatingResponses_MasksPhonesWhenEnabled(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	org.Settings = models.JSONB{"mask_phone_numbers": true}
	require.NoError(t, app.DB.Save(org).Error)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	five := 5
	now := time.Now()
	older := now.Add(-time.Hour)
	// Phone-like profile name: masked as a phone. Plain name: kept, phone still masked.
	phoneNamed := seedRatingContact(t, app, org.ID, "+201111222333", "+201111222333")
	named := seedRatingContact(t, app, org.ID, "+201999888777", "Sara")
	first := createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: phoneNamed.ID, WhatsAppAccount: account.Name,
		Status: models.RatingStatusRated, Rating: &five, RatedAt: &now,
	})
	second := createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: named.ID, WhatsAppAccount: account.Name,
		Status: models.RatingStatusRated, Rating: &five, RatedAt: &older,
	})

	status, body := listCloseRatingResponses(t, app, org.ID, user.ID, account.ID, nil)
	require.Equal(t, fasthttp.StatusOK, status)
	require.Len(t, body.Data.Responses, 2)
	assert.Equal(t, []uuid.UUID{first.ID, second.ID}, responseIDs(body.Data.Responses))

	assert.Equal(t, "*********2333", body.Data.Responses[0].PhoneNumber)
	assert.Equal(t, "*********2333", body.Data.Responses[0].ContactName)
	assert.Equal(t, "*********8777", body.Data.Responses[1].PhoneNumber)
	assert.Equal(t, "Sara", body.Data.Responses[1].ContactName)
}

func TestApp_ListCloseRatingResponses_AgentSeesOnlyInvolvedContacts(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	// accounts:read lets the agent call the endpoint; no contacts:read means
	// visibility falls back to conversations the agent is involved in.
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "rating-viewer-"+uuid.New().String()[:8], []string{"accounts:read"})
	agent := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	five := 5
	now := time.Now()
	mine := seedRatingContact(t, app, org.ID, "+201000000011", "Mine")
	theirs := seedRatingContact(t, app, org.ID, "+201000000012", "Theirs")
	mineCycle := createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: mine.ID, WhatsAppAccount: account.Name,
		Status: models.RatingStatusRated, Rating: &five, RatedAt: &now,
	})
	createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: theirs.ID, WhatsAppAccount: account.Name,
		Status: models.RatingStatusRated, Rating: &five, RatedAt: &now,
	})

	// Not involved in either contact: an empty list, not an error.
	status, body := listCloseRatingResponses(t, app, org.ID, agent.ID, account.ID, nil)
	require.Equal(t, fasthttp.StatusOK, status)
	assert.Equal(t, int64(0), body.Data.Total)
	assert.Empty(t, body.Data.Responses)

	// Assigning one contact to the agent exposes only that contact's cycle.
	require.NoError(t, app.DB.Model(mine).Update("assigned_user_id", agent.ID).Error)

	status, body = listCloseRatingResponses(t, app, org.ID, agent.ID, account.ID, nil)
	require.Equal(t, fasthttp.StatusOK, status)
	assert.Equal(t, int64(1), body.Data.Total)
	assert.Equal(t, []uuid.UUID{mineCycle.ID}, responseIDs(body.Data.Responses))
}

func TestApp_ListCloseRatingResponses_AccountAssignedReaderScopedByAccount(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	assigned := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	other := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	// contacts:read plus an account assignment: the account-name branch of
	// scopeAssignedContact applies, so only the assigned account's contacts show.
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "branch-reader-"+uuid.New().String()[:8],
		[]string{"accounts:read", "contacts:read"})
	reader := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	testutil.AssignAccountToUser(t, app.DB, reader.ID, assigned.ID)

	five := 5
	now := time.Now()
	inAssigned := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(assigned.Name))
	inOther := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(other.Name))
	assignedCycle := createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: inAssigned.ID, WhatsAppAccount: assigned.Name,
		Status: models.RatingStatusRated, Rating: &five, RatedAt: &now,
	})
	createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: inOther.ID, WhatsAppAccount: other.Name,
		Status: models.RatingStatusRated, Rating: &five, RatedAt: &now,
	})

	status, body := listCloseRatingResponses(t, app, org.ID, reader.ID, assigned.ID, nil)
	require.Equal(t, fasthttp.StatusOK, status)
	assert.Equal(t, int64(1), body.Data.Total)
	assert.Equal(t, []uuid.UUID{assignedCycle.ID}, responseIDs(body.Data.Responses))

	// The unassigned account's conversations are outside this reader's scope.
	status, body = listCloseRatingResponses(t, app, org.ID, reader.ID, other.ID, nil)
	require.Equal(t, fasthttp.StatusOK, status)
	assert.Equal(t, int64(0), body.Data.Total)
	assert.Empty(t, body.Data.Responses)
}

func TestApp_ListCloseRatingResponses_ExcludesDeletedContacts(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	five := 5
	now := time.Now()
	gone := seedRatingContact(t, app, org.ID, "+201000000021", "Deleted Customer")
	createRatingCycle(t, app, models.ChatClosureRating{
		OrganizationID: org.ID, ContactID: gone.ID, WhatsAppAccount: account.Name,
		Status: models.RatingStatusRated, Rating: &five, RatedAt: &now,
	})
	require.NoError(t, app.DB.Delete(gone).Error)

	status, body := listCloseRatingResponses(t, app, org.ID, user.ID, account.ID, nil)
	require.Equal(t, fasthttp.StatusOK, status)
	assert.Equal(t, int64(0), body.Data.Total)
	assert.Empty(t, body.Data.Responses)
}

func TestApp_ListCloseRatingResponses_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewGETRequest(t)
	// No auth context set

	require.NoError(t, app.ListCloseRatingResponses(req))
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}
