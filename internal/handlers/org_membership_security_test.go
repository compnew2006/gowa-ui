package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/middleware"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestGetOrgIDRejectsRemovedCrossOrganizationMember(t *testing.T) {
	db := testutil.SetupTestDB(t)
	app := &App{DB: db, Log: testutil.NopLogger()}
	home := testutil.CreateTestOrganization(t, db)
	target := testutil.CreateTestOrganization(t, db)
	user := testutil.CreateTestUser(t, db, home.ID)
	membership := models.UserOrganization{UserID: user.ID, OrganizationID: target.ID}
	require.NoError(t, db.Create(&membership).Error)

	req := testutil.NewRequest(t)
	testutil.SetAuthContext(req, target.ID, user.ID)
	got, err := app.getOrgID(req)
	require.NoError(t, err)
	require.Equal(t, target.ID, got)

	require.NoError(t, db.Delete(&membership).Error)
	_, err = app.getOrgID(req)
	require.Error(t, err, "a token for an organization must stop working after membership removal")

	homeReq := testutil.NewRequest(t)
	testutil.SetAuthContext(homeReq, home.ID, user.ID)
	got, err = app.getOrgID(homeReq)
	require.NoError(t, err)
	require.Equal(t, home.ID, got)
}

func TestWebSocketAuthenticationRejectsRemovedMember(t *testing.T) {
	db := testutil.SetupTestDB(t)
	app := &App{DB: db, Log: testutil.NopLogger(), Config: &config.Config{JWT: config.JWTConfig{Secret: testutil.TestJWTSecret}}}
	home := testutil.CreateTestOrganization(t, db)
	target := testutil.CreateTestOrganization(t, db)
	user := testutil.CreateTestUser(t, db, home.ID)
	membership := models.UserOrganization{UserID: user.ID, OrganizationID: target.ID}
	require.NoError(t, db.Create(&membership).Error)
	claims := middleware.JWTClaims{UserID: user.ID, OrganizationID: target.ID, TokenType: middleware.TokenTypeWS,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testutil.TestJWTSecret))
	require.NoError(t, err)
	_, _, err = app.validateWSTokenFn()(token)
	require.NoError(t, err)
	require.NoError(t, db.Delete(&membership).Error)
	_, _, err = app.validateWSTokenFn()(token)
	require.Error(t, err)
}

func TestGetWSTokenRejectsRemovedMember(t *testing.T) {
	db := testutil.SetupTestDB(t)
	app := &App{DB: db, Log: testutil.NopLogger(), Config: &config.Config{JWT: config.JWTConfig{Secret: testutil.TestJWTSecret}}}
	home := testutil.CreateTestOrganization(t, db)
	target := testutil.CreateTestOrganization(t, db)
	user := testutil.CreateTestUser(t, db, home.ID)
	membership := models.UserOrganization{UserID: user.ID, OrganizationID: target.ID}
	require.NoError(t, db.Create(&membership).Error)
	require.NoError(t, db.Delete(&membership).Error)
	req := testutil.NewRequest(t)
	testutil.SetAuthContext(req, target.ID, user.ID)
	require.NoError(t, app.GetWSToken(req))
	require.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestRemovedMemberCannotInheritHomeRoleInAnotherOrganization(t *testing.T) {
	db := testutil.SetupTestDB(t)
	app := &App{DB: db, Log: testutil.NopLogger()}
	home := testutil.CreateTestOrganization(t, db)
	target := testutil.CreateTestOrganization(t, db)
	perms := testutil.GetOrCreateTestPermissions(t, db)
	role := testutil.CreateTestRole(t, db, home.ID, "admin", perms)
	user := testutil.CreateTestUser(t, db, home.ID, testutil.WithRoleID(&role.ID))
	membership := models.UserOrganization{UserID: user.ID, OrganizationID: target.ID, RoleID: &role.ID}
	require.NoError(t, db.Create(&membership).Error)
	require.NoError(t, db.Delete(&membership).Error)

	require.False(t, app.HasPermission(user.ID, models.ResourceContacts, models.ActionRead, target.ID))
	require.True(t, app.HasPermission(user.ID, models.ResourceContacts, models.ActionRead, home.ID))
}

func TestCachedPermissionsDoNotSurviveMembershipRemoval(t *testing.T) {
	app := cacheTestApp(t)
	home := testutil.CreateTestOrganization(t, app.DB)
	target := testutil.CreateTestOrganization(t, app.DB)
	role := testutil.CreateTestRoleWithKeys(t, app.DB, target.ID, "member", []string{"contacts:read"})
	user := testutil.CreateTestUser(t, app.DB, home.ID)
	membership := models.UserOrganization{UserID: user.ID, OrganizationID: target.ID, RoleID: &role.ID}
	require.NoError(t, app.DB.Create(&membership).Error)
	require.True(t, app.HasPermission(user.ID, models.ResourceContacts, models.ActionRead, target.ID))
	require.NoError(t, app.DB.Delete(&membership).Error)
	require.False(t, app.HasPermission(user.ID, models.ResourceContacts, models.ActionRead, target.ID))
}
