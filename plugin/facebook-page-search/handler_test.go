package facebookpagesearch

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

func TestPluginMigrationOwnsPageSearchTable(t *testing.T) {
	db := testutil.SetupTestDB(t)
	plugin := &Plugin{}
	require.NoError(t, plugin.Migrate(db))
	require.True(t, db.Migrator().HasTable(&PageSearch{}))
}

type pageSearchTestApp struct {
	*Plugin
	DB *gorm.DB
}

func newTestApp(t *testing.T) *pageSearchTestApp {
	t.Helper()
	db := testutil.SetupTestDB(t)
	app := &handlers.App{DB: db, Log: testutil.NopLogger()}
	plugin := &Plugin{}
	require.NoError(t, plugin.Init(app, db, nil, nil))
	return &pageSearchTestApp{Plugin: plugin, DB: db}
}

type fbPageSearchEnvelope struct {
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

func createFBPageSearchUser(t *testing.T, db *gorm.DB, orgID uuid.UUID) *models.User {
	t.Helper()
	role := testutil.CreateTestRoleWithKeys(t, db, orgID, "fb-page-search-reader", []string{
		"accounts:read",
	})
	return testutil.CreateTestUser(t, db, orgID, testutil.WithRoleID(&role.ID))
}

func seedFBPageSearch(t *testing.T, db *gorm.DB, orgID uuid.UUID, campaignID, pageID, name, followers string) *PageSearch {
	t.Helper()
	row := &PageSearch{
		OrganizationID: orgID,
		CampaignID:     campaignID,
		PageID:         pageID,
		Name:           name,
		FollowersCount: followers,
	}
	require.NoError(t, db.Create(row).Error)
	return row
}

func decodeFBPageSearch(t *testing.T, req *fastglue.Request) fbPageSearchEnvelope {
	t.Helper()
	var resp fbPageSearchEnvelope
	body := testutil.GetResponseBody(req)
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func TestApp_SearchFBPages_RequiresCampaignID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createFBPageSearchUser(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.SearchFBPages(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_SearchFBPages_RequiresAuth(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, uuid.Nil)

	err := app.SearchFBPages(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_SearchFBPages_EmptyResult(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createFBPageSearchUser(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.QueryArgs().Add("campaign_id", "camp-empty")

	err := app.SearchFBPages(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	resp := decodeFBPageSearch(t, req)
	assert.Equal(t, "success", resp.Status)
	assert.True(t, resp.Data.Success)
	assert.Equal(t, "camp-empty", resp.Data.CampaignID)
	assert.Equal(t, 1, resp.Data.Page)
	assert.Equal(t, 25, resp.Data.PerPage)
	assert.Equal(t, 0, resp.Data.Total)
	assert.Equal(t, 0, resp.Data.TotalPages)
	assert.Empty(t, resp.Data.Data)
}

func TestApp_SearchFBPages_ReturnsAllRowsForCampaign(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createFBPageSearchUser(t, app.DB, org.ID)

	seedFBPageSearch(t, app.DB, org.ID, "camp-1", "111", "Alpha Page", "1000")
	seedFBPageSearch(t, app.DB, org.ID, "camp-1", "222", "Beta Page", "2000")
	seedFBPageSearch(t, app.DB, org.ID, "camp-1", "333", "Gamma Page", "3000")
	seedFBPageSearch(t, app.DB, org.ID, "camp-2", "444", "Delta Page", "4000")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.QueryArgs().Add("campaign_id", "camp-1")

	err := app.SearchFBPages(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	resp := decodeFBPageSearch(t, req)
	assert.True(t, resp.Data.Success)
	assert.Equal(t, "camp-1", resp.Data.CampaignID)
	assert.Equal(t, 3, resp.Data.Total)
	assert.Equal(t, 1, resp.Data.TotalPages)
	require.Len(t, resp.Data.Data, 3)

	// Ordered by id DESC => newest first: Gamma, Beta, Alpha
	assert.Equal(t, "Gamma Page", resp.Data.Data[0].Name)
	assert.Equal(t, "333", resp.Data.Data[0].PageID)
	assert.Equal(t, "3000", resp.Data.Data[0].FollowersCount)
	assert.Equal(t, "Beta Page", resp.Data.Data[1].Name)
	assert.Equal(t, "Alpha Page", resp.Data.Data[2].Name)
}

func TestApp_SearchFBPages_QueryFilterMatchesName(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createFBPageSearchUser(t, app.DB, org.ID)

	seedFBPageSearch(t, app.DB, org.ID, "camp-q", "111", "Acme Coffee", "100")
	seedFBPageSearch(t, app.DB, org.ID, "camp-q", "222", "Beta Books", "200")
	seedFBPageSearch(t, app.DB, org.ID, "camp-q", "333", "Gamma Gifts", "300")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.QueryArgs().Add("campaign_id", "camp-q")
	req.RequestCtx.QueryArgs().Add("q", "coffee")

	err := app.SearchFBPages(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	resp := decodeFBPageSearch(t, req)
	assert.Equal(t, 1, resp.Data.Total)
	require.Len(t, resp.Data.Data, 1)
	assert.Equal(t, "Acme Coffee", resp.Data.Data[0].Name)
}

func TestApp_SearchFBPages_QueryFilterMatchesPageID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createFBPageSearchUser(t, app.DB, org.ID)

	seedFBPageSearch(t, app.DB, org.ID, "camp-qid", "111111", "Foo", "1")
	seedFBPageSearch(t, app.DB, org.ID, "camp-qid", "222222", "Bar", "2")
	seedFBPageSearch(t, app.DB, org.ID, "camp-qid", "333333", "Baz", "3")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.QueryArgs().Add("campaign_id", "camp-qid")
	req.RequestCtx.QueryArgs().Add("q", "222")

	err := app.SearchFBPages(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	resp := decodeFBPageSearch(t, req)
	assert.Equal(t, 1, resp.Data.Total)
	require.Len(t, resp.Data.Data, 1)
	assert.Equal(t, "222222", resp.Data.Data[0].PageID)
}

func TestApp_SearchFBPages_Pagination(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createFBPageSearchUser(t, app.DB, org.ID)

	for i := 0; i < 7; i++ {
		seedFBPageSearch(t, app.DB, org.ID, "camp-page", "id", "Page", "0")
	}

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.QueryArgs().Add("campaign_id", "camp-page")
	req.RequestCtx.QueryArgs().Add("page", "2")
	req.RequestCtx.QueryArgs().Add("per_page", "3")

	err := app.SearchFBPages(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	resp := decodeFBPageSearch(t, req)
	assert.Equal(t, 7, resp.Data.Total)
	assert.Equal(t, 2, resp.Data.Page)
	assert.Equal(t, 3, resp.Data.PerPage)
	assert.Equal(t, 3, resp.Data.TotalPages)
	require.Len(t, resp.Data.Data, 3)
}

func TestApp_SearchFBPages_TenantIsolation(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	orgB := testutil.CreateTestOrganization(t, app.DB)
	userA := createFBPageSearchUser(t, app.DB, orgA.ID)

	// orgA has 1 row, orgB has 2 rows for the same campaign_id
	seedFBPageSearch(t, app.DB, orgA.ID, "shared-camp", "1", "A only", "10")
	seedFBPageSearch(t, app.DB, orgB.ID, "shared-camp", "2", "B-1", "20")
	seedFBPageSearch(t, app.DB, orgB.ID, "shared-camp", "3", "B-2", "30")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgA.ID, userA.ID)
	req.RequestCtx.QueryArgs().Add("campaign_id", "shared-camp")

	err := app.SearchFBPages(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	resp := decodeFBPageSearch(t, req)
	assert.Equal(t, 1, resp.Data.Total, "orgA must not see orgB rows")
	require.Len(t, resp.Data.Data, 1)
	assert.Equal(t, "A only", resp.Data.Data[0].Name)
}

func TestApp_SearchFBPages_RejectsForbiddenUser(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	// Create user with NO permissions
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.QueryArgs().Add("campaign_id", "camp-1")

	err := app.SearchFBPages(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

// Reference unused imports to keep them if test fixtures change
var _ = handlers.App{}