package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/middleware"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestApp_Login_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("login-success")
	password := "validpassword123"
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword(password))

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": password,
	})

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Parse the response — tokens are in cookies, not body
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			ExpiresIn int `json:"expires_in"`
			User      struct {
				Email string `json:"email"`
				Role  string `json:"role"`
			} `json:"user"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, 15*60, resp.Data.ExpiresIn)
	assert.Equal(t, email, resp.Data.User.Email)

	// Tokens should be in Set-Cookie headers
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_access"))
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_refresh"))
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_csrf"))
}

func TestApp_Login_WrongPassword(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("wrong-pwd")
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword("correctpassword"))

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": "wrongpassword",
	})

	err := app.Login(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid credentials")
}

func TestApp_Login_UserNotFound(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    testutil.UniqueEmail("nonexistent"),
		"password": "anypassword",
	})

	err := app.Login(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid credentials")
}

func TestApp_Login_InactiveUser(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("inactive")
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword("validpassword123"), testutil.WithInactive())

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": "validpassword123",
	})

	err := app.Login(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Account is disabled")
}

func TestApp_Login_InvalidRequestBody(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.SetBody([]byte("invalid json"))
	req.RequestCtx.Request.Header.SetContentType("application/json")

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_Login_UserWithRole(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("role-test")
	password := "testpassword123"

	// Create an actual role first
	role := &models.CustomRole{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Test Role " + uuid.New().String()[:8],
		IsSystem:       false,
	}
	require.NoError(t, app.DB.Create(role).Error)

	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword(password), testutil.WithRoleID(&role.ID))

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": password,
	})

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
}

func TestApp_Register_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("register")

	// Create a default role for the org (Register looks for is_default=true, then falls back to name="agent" + is_system=true)
	defaultRole := testutil.CreateTestRoleExact(t, app.DB, org.ID, "agent", true, true, nil)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":           email,
		"password":        "securepassword123",
		"full_name":       "New User",
		"organization_id": org.ID.String(),
	})

	err := app.Register(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			ExpiresIn int `json:"expires_in"`
			User      struct {
				ID       string `json:"id"`
				Email    string `json:"email"`
				FullName string `json:"full_name"`
				RoleID   string `json:"role_id"`
				IsActive bool   `json:"is_active"`
			} `json:"user"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, email, resp.Data.User.Email)
	assert.Equal(t, "New User", resp.Data.User.FullName)
	assert.NotEmpty(t, resp.Data.User.RoleID, "User should have a role assigned")
	assert.True(t, resp.Data.User.IsActive)

	// Tokens should be in cookies
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_access"))
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_refresh"))

	// Verify the user has the default role in the database
	userID, err := uuid.Parse(resp.Data.User.ID)
	require.NoError(t, err)
	var user models.User
	require.NoError(t, app.DB.Preload("Role").Where("id = ?", userID).First(&user).Error)
	assert.NotNil(t, user.Role)
	assert.Equal(t, defaultRole.Name, user.Role.Name)
	assert.True(t, user.Role.IsSystem)

	// Verify user_organizations entry was created
	var userOrg models.UserOrganization
	require.NoError(t, app.DB.Where("user_id = ? AND organization_id = ?", userID, org.ID).First(&userOrg).Error)
	assert.True(t, userOrg.IsDefault)
}

func TestApp_Register_EmailAlreadyExists_WrongPassword(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("existing")
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword("password123"))

	// Create a second org to register into
	org2 := testutil.CreateTestOrganization(t, app.DB)
	testutil.CreateTestRoleExact(t, app.DB, org2.ID, "agent", true, true, nil)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":           email,
		"password":        "wrongpassword123",
		"full_name":       "Another User",
		"organization_id": org2.ID.String(),
	})

	err := app.Register(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusConflict, "An account with this email already exists")
}

func TestApp_Register_ExistingUser_JoinsNewOrg(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("multiorg")
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword("password123"))

	// Create a second org with a default role
	org2 := testutil.CreateTestOrganization(t, app.DB)
	testutil.CreateTestRoleExact(t, app.DB, org2.ID, "agent", true, true, nil)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":           email,
		"password":        "password123",
		"full_name":       "Same User",
		"organization_id": org2.ID.String(),
	})

	err := app.Register(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Tokens should be in cookies, not body
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_access"))
}

func TestApp_Register_InvalidRequestBody(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.SetBody([]byte("invalid json"))
	req.RequestCtx.Request.Header.SetContentType("application/json")

	err := app.Register(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_RefreshToken_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("refresh")), testutil.WithPassword("password123"))
	refreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": refreshToken,
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			ExpiresIn int `json:"expires_in"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, 15*60, resp.Data.ExpiresIn)

	// Tokens should be in cookies
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_access"))
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_refresh"))
}

func TestApp_RefreshToken_Expired(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("expired")), testutil.WithPassword("password123"))
	expiredToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, -time.Hour)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": expiredToken,
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid refresh token")
}

func TestApp_RefreshToken_InvalidSignature(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("invalid-sig")), testutil.WithPassword("password123"))
	wrongSecretToken := testutil.GenerateTestRefreshToken(t, user, "wrong-secret-key-that-is-long", 7*24*time.Hour)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": wrongSecretToken,
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid refresh token")
}

func TestApp_RefreshToken_UserNotFound(t *testing.T) {
	app := newTestApp(t)
	fakeUser := &models.User{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		OrganizationID: uuid.New(),
		Email:          "fake@example.com",
	}
	token := testutil.GenerateTestRefreshToken(t, fakeUser, testutil.TestJWTSecret, 7*24*time.Hour)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": token,
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "User not found")
}

func TestApp_RefreshToken_DisabledUser(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("disabled")), testutil.WithPassword("password123"), testutil.WithInactive())
	token := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": token,
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Account is disabled")
}

func TestApp_RefreshToken_MalformedToken(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": "not.a.valid.jwt.token",
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid refresh token")
}

func TestApp_RefreshToken_InvalidRequestBody(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.SetBody([]byte("invalid json"))
	req.RequestCtx.Request.Header.SetContentType("application/json")

	err := app.RefreshToken(req)
	require.NoError(t, err)
	// No cookie and no valid body → 401 "Missing refresh token"
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Missing refresh token")
}

func TestApp_GeneratedTokensAreValid(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("tokentest")
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword("password123"))

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": "password123",
	})

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Read tokens from cookies
	accessTokenStr := testutil.GetResponseCookie(req, "whm_access")
	refreshTokenStr := testutil.GetResponseCookie(req, "whm_refresh")
	require.NotEmpty(t, accessTokenStr)
	require.NotEmpty(t, refreshTokenStr)

	// Verify access token can be parsed
	accessToken, err := jwt.ParseWithClaims(accessTokenStr, &middleware.JWTClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(testutil.TestJWTSecret), nil
	})
	require.NoError(t, err)
	require.True(t, accessToken.Valid)

	accessClaims, ok := accessToken.Claims.(*middleware.JWTClaims)
	require.True(t, ok)
	assert.Equal(t, user.ID, accessClaims.UserID)
	assert.Equal(t, org.ID, accessClaims.OrganizationID)
	assert.Equal(t, user.Email, accessClaims.Email)
	assert.Equal(t, user.RoleID, accessClaims.RoleID)
	assert.Equal(t, "gowa-ui", accessClaims.Issuer)

	// Verify refresh token can be parsed
	refreshToken, err := jwt.ParseWithClaims(refreshTokenStr, &middleware.JWTClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(testutil.TestJWTSecret), nil
	})
	require.NoError(t, err)
	require.True(t, refreshToken.Valid)

	refreshClaims, ok := refreshToken.Claims.(*middleware.JWTClaims)
	require.True(t, ok)
	assert.Equal(t, user.ID, refreshClaims.UserID)
}

// authMiddlewareForTest wires AuthWithDBAndRedis exactly as routes.go does,
// so H3 revocation is exercised through the production middleware path.
func authMiddlewareForTest(app *handlers.App) fastglue.FastMiddleware {
	return middleware.AuthWithDBAndRedis(testutil.TestJWTSecret, app.DB, app.Redis, testutil.NopLogger())
}

// TestLogout_BumpsTokenVersion (H3 end-to-end) proves that calling Logout
// immediately invalidates the caller's outstanding access token via the
// per-user token_version bump, not just at natural JWT expiry.
//
// Flow: Login → access token validates against AuthWithDBAndRedis →
// Logout (presenting the refresh token) → the SAME access token is now
// rejected with 401. Skips when Redis is unavailable (the bump lives in
// Redis; without it the test cannot observe revocation).
func TestLogout_BumpsTokenVersion(t *testing.T) {
	app := newTestApp(t)
	if app.Redis == nil {
		t.Skip("TEST_REDIS_URL not set, skipping H3 revocation test")
	}
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("logout-revoke")
	password := "validpassword123"
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword(password))

	// 1. Login to obtain real access + refresh tokens (production stamping path).
	loginReq := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, app.Login(loginReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(loginReq))

	accessToken := testutil.GetResponseCookie(loginReq, "whm_access")
	refreshToken := testutil.GetResponseCookie(loginReq, "whm_refresh")
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	// Sanity: the freshly-issued access token must pass the Redis-backed
	// middleware before logout (version matches).
	preReq := testutil.NewGETRequest(t)
	testutil.SetAuthHeader(preReq, accessToken)
	require.NotNil(t, authMiddlewareForTest(app)(preReq), "fresh access token must validate before logout")

	// 2. Logout, presenting the refresh token via the whm_refresh cookie
	//    (mirrors the browser flow). This must bump token_version:<user>.
	logoutReq := testutil.NewRequest(t)
	logoutReq.RequestCtx.Request.Header.SetCookie("whm_refresh", refreshToken)
	logoutReq.RequestCtx.Request.Header.SetContentType("application/json")
	require.NoError(t, app.Logout(logoutReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(logoutReq))

	// 3. The SAME access token must now be rejected (version is now stale).
	postReq := testutil.NewGETRequest(t)
	testutil.SetAuthHeader(postReq, accessToken)
	result := authMiddlewareForTest(app)(postReq)
	assert.Nil(t, result, "access token must be rejected after logout (H3 revocation)")
	assert.Equal(t, fasthttp.StatusUnauthorized, postReq.RequestCtx.Response.StatusCode())
}

// TestLogout_BumpsTokenVersionViaAccessTokenCookie (H3, Reviewer note 1)
// covers the fallback path: when no refresh token is presented, Logout
// recovers the user identity from the whm_access cookie and still bumps the
// version — so a client that only sends the access token is also revoked.
func TestLogout_BumpsTokenVersionViaAccessTokenCookie(t *testing.T) {
	app := newTestApp(t)
	if app.Redis == nil {
		t.Skip("TEST_REDIS_URL not set, skipping H3 revocation test")
	}
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("logout-access-revoke")
	password := "validpassword123"
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword(password))

	loginReq := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, app.Login(loginReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(loginReq))

	accessToken := testutil.GetResponseCookie(loginReq, "whm_access")
	require.NotEmpty(t, accessToken)

	preReq := testutil.NewGETRequest(t)
	testutil.SetAuthHeader(preReq, accessToken)
	require.NotNil(t, authMiddlewareForTest(app)(preReq), "fresh access token must validate before logout")

	// Logout with ONLY the access-token cookie (no refresh token).
	logoutReq := testutil.NewRequest(t)
	logoutReq.RequestCtx.Request.Header.SetCookie("whm_access", accessToken)
	logoutReq.RequestCtx.Request.Header.SetContentType("application/json")
	require.NoError(t, app.Logout(logoutReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(logoutReq))

	postReq := testutil.NewGETRequest(t)
	testutil.SetAuthHeader(postReq, accessToken)
	result := authMiddlewareForTest(app)(postReq)
	assert.Nil(t, result, "access token must be rejected after access-cookie-only logout")
	assert.Equal(t, fasthttp.StatusUnauthorized, postReq.RequestCtx.Response.StatusCode())
}

// TestChangePassword_BumpsTokenVersion (H3) proves that a successful password
// change invalidates the user's outstanding access token immediately, forcing
// re-auth on all sessions.
func TestChangePassword_BumpsTokenVersion(t *testing.T) {
	app := newTestApp(t)
	if app.Redis == nil {
		t.Skip("TEST_REDIS_URL not set, skipping H3 revocation test")
	}
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("changepass-revoke")
	password := "validpassword123"
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword(password))

	// Mint an access token via Login (real stamping).
	loginReq := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, app.Login(loginReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(loginReq))
	accessToken := testutil.GetResponseCookie(loginReq, "whm_access")
	require.NotEmpty(t, accessToken)

	// Token valid before password change.
	preReq := testutil.NewGETRequest(t)
	testutil.SetAuthHeader(preReq, accessToken)
	require.NotNil(t, authMiddlewareForTest(app)(preReq), "access token must validate before password change")

	// Change the password (ChangePassword reads user_id from context).
	cpReq := testutil.NewJSONRequest(t, map[string]string{
		"current_password": password,
		"new_password":     "brandnewpassword456",
	})
	testutil.SetAuthContext(cpReq, org.ID, user.ID)
	require.NoError(t, app.ChangePassword(cpReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(cpReq))

	// The pre-change access token must now be rejected.
	postReq := testutil.NewGETRequest(t)
	testutil.SetAuthHeader(postReq, accessToken)
	result := authMiddlewareForTest(app)(postReq)
	assert.Nil(t, result, "access token must be rejected after password change (H3)")
	assert.Equal(t, fasthttp.StatusUnauthorized, postReq.RequestCtx.Response.StatusCode())
}
