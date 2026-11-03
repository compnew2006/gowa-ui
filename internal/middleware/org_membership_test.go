package middleware_test

import (
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/middleware"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/fastglue"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthRejectsTokenAfterOrganizationMembershipRemoval(t *testing.T) {
	db := testutil.SetupTestDB(t)
	home := testutil.CreateTestOrganization(t, db)
	target := testutil.CreateTestOrganization(t, db)
	user := testutil.CreateTestUser(t, db, home.ID)
	membership := models.UserOrganization{UserID: user.ID, OrganizationID: target.ID}
	require.NoError(t, db.Create(&membership).Error)
	token := generateTestToken(t, user.ID, target.ID, user.Email, nil, time.Hour)
	request := func() *fastglue.Request {
		req := newTestRequest()
		req.RequestCtx.Request.Header.Set("Authorization", "Bearer "+token)
		return req
	}
	require.NotNil(t, middleware.AuthWithDB(testJWTSecret, db)(request()))
	require.NoError(t, db.Delete(&membership).Error)
	require.Nil(t, middleware.AuthWithDB(testJWTSecret, db)(request()))
}

func TestAPIKeyRejectedAfterOrganizationMembershipRemoval(t *testing.T) {
	db := testutil.SetupTestDB(t)
	home := testutil.CreateTestOrganization(t, db)
	target := testutil.CreateTestOrganization(t, db)
	user := testutil.CreateTestUser(t, db, home.ID)
	membership := models.UserOrganization{UserID: user.ID, OrganizationID: target.ID}
	require.NoError(t, db.Create(&membership).Error)
	key := "whm_0123456789abcdef0123456789abcdef" // gitleaks:allow synthetic test key, not a real credential
	hash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.MinCost)
	require.NoError(t, err)
	apiKey := models.APIKey{OrganizationID: target.ID, UserID: user.ID, Name: "test", KeyPrefix: key[4:20], KeyHash: string(hash), IsActive: true}
	require.NoError(t, db.Create(&apiKey).Error)
	request := func() *fastglue.Request {
		req := newTestRequest()
		req.RequestCtx.Request.Header.Set("X-API-Key", key)
		return req
	}
	require.NotNil(t, middleware.AuthWithDB(testJWTSecret, db)(request()))
	require.NoError(t, db.Delete(&membership).Error)
	require.Nil(t, middleware.AuthWithDB(testJWTSecret, db)(request()))
}

func TestAuthAllowsSuperAdminWithoutTargetMembership(t *testing.T) {
	db := testutil.SetupTestDB(t)
	home := testutil.CreateTestOrganization(t, db)
	target := testutil.CreateTestOrganization(t, db)
	user := testutil.CreateTestUser(t, db, home.ID, testutil.WithSuperAdmin())
	token := generateTestToken(t, user.ID, target.ID, user.Email, nil, time.Hour)
	req := newTestRequest()
	req.RequestCtx.Request.Header.Set("Authorization", "Bearer "+token)
	require.NotNil(t, middleware.AuthWithDB(testJWTSecret, db)(req))
}
