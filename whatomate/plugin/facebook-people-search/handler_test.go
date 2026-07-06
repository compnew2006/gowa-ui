package facebookpeoplesearch

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

func TestPluginMigrationOwnsPeopleSearchTable(t *testing.T) {
	db := testutil.SetupTestDB(t)
	plugin := &Plugin{}
	require.NoError(t, plugin.Migrate(db))
	require.True(t, db.Migrator().HasTable(&PeopleSearch{}))
}

type peopleSearchTestApp struct {
	*Plugin
	DB *gorm.DB
}

func newTestApp(t *testing.T) *peopleSearchTestApp {
	t.Helper()
	db := testutil.SetupTestDB(t)
	app := &handlers.App{DB: db, Log: testutil.NopLogger()}
	plugin := &Plugin{}
	require.NoError(t, plugin.Init(app, db, nil, nil))
	return &peopleSearchTestApp{Plugin: plugin, DB: db}
}

type fbPeopleSearchEnvelope struct {
	Status string `json:"status"`
	Data   struct {
		Success    bool   `json:"success"`
		CampaignID string `json:"campaign_id"`
		Page       int    `json:"page"`
		PerPage    int    `json:"per_page"`
		Total      int    `json:"total"`
		TotalPages int    `json:"total_pages"`
		Data       []struct {
			Name           string `json:"name"`
			PageID         string `json:"page_id"`
			FollowersCount string `json:"followers_count"`
		} `json:"data"`
	} `json:"data"`
}

type addContactsEnvelope struct {
	Status string `json:"status"`
	Data   struct {
		Success bool `json:"success"`
		Created int  `json:"created"`
		Updated int  `json:"updated"`
		Total   int  `json:"total"`
	} `json:"data"`
}

func createFBPeopleSearchUser(t *testing.T, db *gorm.DB, orgID uuid.UUID) *models.User {
	t.Helper()
	role := testutil.CreateTestRoleWithKeys(t, db, orgID, "fb-people-search-reader", []string{
		"accounts:read",
		"contacts:write",
	})
	return testutil.CreateTestUser(t, db, orgID, testutil.WithRoleID(&role.ID))
}

func seedFBPeopleSearch(t *testing.T, db *gorm.DB, orgID uuid.UUID, campaignID, pageID, name, followers string) *PeopleSearch {
	t.Helper()
	row := &PeopleSearch{
		OrganizationID: orgID,
		CampaignID:     campaignID,
		PageID:         pageID,
		Name:           name,
		FollowersCount: followers,
	}
	require.NoError(t, db.Create(row).Error)
	return row
}

func decodeFBPeopleSearch(t *testing.T, req *fastglue.Request) fbPeopleSearchEnvelope {
	t.Helper()
	var resp fbPeopleSearchEnvelope
	body := testutil.GetResponseBody(req)
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func decodeAddContactsResponse(t *testing.T, req *fastglue.Request) addContactsEnvelope {
	t.Helper()
	var resp addContactsEnvelope
	body := testutil.GetResponseBody(req)
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func TestApp_SearchFBPeople_RequiresCampaignID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createFBPeopleSearchUser(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.SearchFBPeople(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_SearchFBPeople_EmptyResult(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createFBPeopleSearchUser(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.QueryArgs().Add("campaign_id", "camp-empty")

	err := app.SearchFBPeople(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	resp := decodeFBPeopleSearch(t, req)
	assert.Equal(t, "success", resp.Status)
	assert.True(t, resp.Data.Success)
	assert.Equal(t, "camp-empty", resp.Data.CampaignID)
	assert.Equal(t, 1, resp.Data.Page)
	assert.Equal(t, 25, resp.Data.PerPage)
	assert.Equal(t, 0, resp.Data.Total)
	assert.Empty(t, resp.Data.Data)
}

func TestApp_SearchFBPeople_ReturnsRows(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createFBPeopleSearchUser(t, app.DB, org.ID)

	seedFBPeopleSearch(t, app.DB, org.ID, "camp-people-1", "p111", "John Doe", "1500")
	seedFBPeopleSearch(t, app.DB, org.ID, "camp-people-1", "p222", "Jane Smith", "2500")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.QueryArgs().Add("campaign_id", "camp-people-1")

	err := app.SearchFBPeople(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	resp := decodeFBPeopleSearch(t, req)
	assert.True(t, resp.Data.Success)
	assert.Equal(t, 2, resp.Data.Total)
	require.Len(t, resp.Data.Data, 2)
	assert.Equal(t, "Jane Smith", resp.Data.Data[0].Name)
	assert.Equal(t, "p222", resp.Data.Data[0].PageID)
}

func TestApp_AddFBPeopleContacts_BulkCreate(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createFBPeopleSearchUser(t, app.DB, org.ID)

	payload := map[string]interface{}{
		"name": "My VIP Contacts List",
		"data": []map[string]string{
			{"identifier": "201002003004", "name": "FB Person 1"},
			{"identifier": "201002003005", "name": "FB Person 2"},
			{"identifier": "invalid-phone", "name": "Skipped User"},
		},
	}

	req := testutil.NewJSONRequest(t, payload)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.AddFBPeopleContacts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	resp := decodeAddContactsResponse(t, req)
	assert.Equal(t, "success", resp.Status)
	assert.True(t, resp.Data.Success)
	assert.Equal(t, 2, resp.Data.Created)
	assert.Equal(t, 0, resp.Data.Updated)

	// Verify DB contacts
	var dbContacts []models.Contact
	require.NoError(t, app.DB.Where("organization_id = ?", org.ID).Find(&dbContacts).Error)
	require.Len(t, dbContacts, 2)

	assert.Equal(t, "201002003004", dbContacts[0].PhoneNumber)
	assert.Equal(t, "FB Person 1", dbContacts[0].ProfileName)
	assert.Contains(t, dbContacts[0].Tags, "My VIP Contacts List")
}

func TestApp_AddFBPeopleContacts_UpdatesExisting(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createFBPeopleSearchUser(t, app.DB, org.ID)

	// Pre-create contact
	existingContact := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    "201002003006",
		ProfileName:    "Old Name",
		Tags:           models.JSONBArray{"Existing Tag"},
	}
	require.NoError(t, app.DB.Create(&existingContact).Error)

	payload := map[string]interface{}{
		"name": "Update List",
		"data": []map[string]string{
			{"identifier": "201002003006", "name": "Updated Name"},
		},
	}

	req := testutil.NewJSONRequest(t, payload)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.AddFBPeopleContacts(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	resp := decodeAddContactsResponse(t, req)
	assert.Equal(t, 1, resp.Data.Updated)

	var check models.Contact
	require.NoError(t, app.DB.First(&check, existingContact.ID).Error)
	// ProfileName should NOT be updated if not empty
	assert.Equal(t, "Old Name", check.ProfileName)
	// Tag should be appended
	assert.Contains(t, check.Tags, "Existing Tag")
	assert.Contains(t, check.Tags, "Update List")
}

func TestApp_ListFBPeopleCampaigns_ReturnsUniqueCampaigns(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createFBPeopleSearchUser(t, app.DB, org.ID)

	seedFBPeopleSearch(t, app.DB, org.ID, "camp-alpha", "1", "A", "1")
	seedFBPeopleSearch(t, app.DB, org.ID, "camp-alpha", "2", "B", "2")
	seedFBPeopleSearch(t, app.DB, org.ID, "camp-beta", "3", "C", "3")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListFBPeopleCampaigns(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Success   bool     `json:"success"`
			Campaigns []string `json:"campaigns"`
		} `json:"data"`
	}
	body := testutil.GetResponseBody(req)
	require.NoError(t, json.Unmarshal(body, &resp))

	assert.True(t, resp.Data.Success)
	assert.Len(t, resp.Data.Campaigns, 2)
	assert.Contains(t, resp.Data.Campaigns, "camp-alpha")
	assert.Contains(t, resp.Data.Campaigns, "camp-beta")
}

// Reference unused imports to keep them if test fixtures change
var _ = handlers.App{}