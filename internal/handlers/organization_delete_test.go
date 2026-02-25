package handlers_test

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestApp_DeleteOrganization_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	homeOrg := testutil.CreateTestOrganization(t, app.DB)
	targetOrg := testutil.CreateTestOrganization(t, app.DB)

	superAdmin := testutil.CreateTestUser(t, app.DB, homeOrg.ID,
		testutil.WithEmail(testutil.UniqueEmail("delete-org-admin")),
		testutil.WithSuperAdmin(),
	)
	targetNativeUser := testutil.CreateTestUser(t, app.DB, targetOrg.ID,
		testutil.WithEmail(testutil.UniqueEmail("delete-org-native")),
	)

	// Cross-org membership that should be removed with the organization.
	memberUser := testutil.CreateTestUser(t, app.DB, homeOrg.ID,
		testutil.WithEmail(testutil.UniqueEmail("delete-org-member")),
	)
	require.NoError(t, app.DB.Create(&models.UserOrganization{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		UserID:         memberUser.ID,
		OrganizationID: targetOrg.ID,
		IsDefault:      false,
	}).Error)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(req, homeOrg.ID, superAdmin.ID)
	testutil.SetPathParam(req, "id", targetOrg.ID.String())

	err := app.DeleteOrganization(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var org models.Organization
	assert.Error(t, app.DB.Where("id = ?", targetOrg.ID).First(&org).Error)

	var nativeUser models.User
	assert.Error(t, app.DB.Where("id = ?", targetNativeUser.ID).First(&nativeUser).Error)

	var orgMemberships int64
	app.DB.Model(&models.UserOrganization{}).Where("organization_id = ?", targetOrg.ID).Count(&orgMemberships)
	assert.Equal(t, int64(0), orgMemberships)
}

func TestApp_DeleteOrganization_CannotDeleteHomeOrganization(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	homeOrg := testutil.CreateTestOrganization(t, app.DB)
	_ = testutil.CreateTestOrganization(t, app.DB) // Keep org count > 1.
	superAdmin := testutil.CreateTestUser(t, app.DB, homeOrg.ID,
		testutil.WithEmail(testutil.UniqueEmail("delete-home-org-admin")),
		testutil.WithSuperAdmin(),
	)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(req, homeOrg.ID, superAdmin.ID)
	testutil.SetPathParam(req, "id", homeOrg.ID.String())

	err := app.DeleteOrganization(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_DeleteOrganization_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	targetOrg := testutil.CreateTestOrganization(t, app.DB)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetPathParam(req, "id", targetOrg.ID.String())

	err := app.DeleteOrganization(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}
