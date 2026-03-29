package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func seedRefreshTokenState(t *testing.T, app *handlers.App, refreshToken string, userID uuid.UUID) {
	t.Helper()

	token, err := jwt.ParseWithClaims(refreshToken, &middleware.JWTClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(testutil.TestJWTSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(*middleware.JWTClaims)
	require.True(t, ok)
	require.NotEmpty(t, claims.ID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, app.Redis.Set(ctx, fmt.Sprintf("refresh:%s", claims.ID), userID.String(), 24*time.Hour).Err())
}

func createUsersWriteUser(t *testing.T, app *handlers.App, orgID uuid.UUID) *models.User {
	t.Helper()

	role := testutil.CreateTestRoleWithKeys(t, app.DB, orgID, "users-writer", []string{
		"users:write",
	})
	return testutil.CreateTestUser(t, app.DB, orgID, testutil.WithRoleID(&role.ID))
}

func createRegisterInviteToken(t *testing.T, app *handlers.App, orgID uuid.UUID) string {
	t.Helper()

	inviter := createUsersWriteUser(t, app, orgID)

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, orgID, inviter.ID)

	err := app.CreateRegisterInvite(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Data.Token)

	return resp.Data.Token
}

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
	assert.Greater(t, resp.Data.ExpiresIn, 0)
	assert.LessOrEqual(t, resp.Data.ExpiresIn, 15*60)
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
	inviteToken := createRegisterInviteToken(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":            email,
		"password":         "securepassword123",
		"full_name":        "New User",
		"organization_id":  org.ID.String(),
		"invitation_token": inviteToken,
	})

	err := app.Register(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.NotEmpty(t, resp.Data.Message)

	// Tokens should NOT be in cookies
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_access"))
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_refresh"))

	// Verify the user has the default role in the database
	var user models.User
	require.NoError(t, app.DB.Preload("Role").Where("email = ?", email).First(&user).Error)
	assert.NotNil(t, user.Role)
	assert.Equal(t, defaultRole.Name, user.Role.Name)
	assert.True(t, user.Role.IsSystem)

	// Verify user_organizations entry was created
	var userOrg models.UserOrganization
	require.NoError(t, app.DB.Where("user_id = ? AND organization_id = ?", user.ID, org.ID).First(&userOrg).Error)
	assert.True(t, userOrg.IsDefault)
}

func TestApp_Register_EmailAlreadyExists_WrongPassword(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("existing")
	existingUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword("password123"))

	// Create a second org to register into
	org2 := testutil.CreateTestOrganization(t, app.DB)
	testutil.CreateTestRoleExact(t, app.DB, org2.ID, "agent", true, true, nil)
	inviteToken := createRegisterInviteToken(t, app, org2.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":            email,
		"password":         "wrongpassword123",
		"full_name":        "Another User",
		"organization_id":  org2.ID.String(),
		"invitation_token": inviteToken,
	})

	err := app.Register(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	assert.NotEmpty(t, resp.Data.Message)

	// Tokens should NOT be in cookies
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_access"))
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_refresh"))

	// Ensure user was not added to org2
	var userOrg models.UserOrganization
	err = app.DB.Where("user_id = ? AND organization_id = ?", existingUser.ID, org2.ID).First(&userOrg).Error
	assert.Error(t, err)
}

func TestApp_Register_ExistingUser_DoesNotJoinNewOrg(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("multiorg")
	existingUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword("password123"))

	// Create a second org with a default role
	org2 := testutil.CreateTestOrganization(t, app.DB)
	testutil.CreateTestRoleExact(t, app.DB, org2.ID, "agent", true, true, nil)
	inviteToken := createRegisterInviteToken(t, app, org2.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":            email,
		"password":         "password123",
		"full_name":        "Same User",
		"organization_id":  org2.ID.String(),
		"invitation_token": inviteToken,
	})

	err := app.Register(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	assert.NotEmpty(t, resp.Data.Message)

	// Tokens should NOT be in cookies
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_access"))
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_refresh"))

	// Ensure user was not added to org2
	var userOrg models.UserOrganization
	err = app.DB.Where("user_id = ? AND organization_id = ?", existingUser.ID, org2.ID).First(&userOrg).Error
	assert.Error(t, err)
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

func TestApp_Register_MissingInvitationToken(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	testutil.CreateTestRoleExact(t, app.DB, org.ID, "agent", true, true, nil)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":           testutil.UniqueEmail("missing-invite"),
		"password":        "securepassword123",
		"full_name":       "No Invite",
		"organization_id": org.ID.String(),
	})

	err := app.Register(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "invitation_token is required")
}

func TestApp_Register_InvalidInvitationToken(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	testutil.CreateTestRoleExact(t, app.DB, org.ID, "agent", true, true, nil)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":            testutil.UniqueEmail("invalid-invite"),
		"password":         "securepassword123",
		"full_name":        "Invalid Invite",
		"organization_id":  org.ID.String(),
		"invitation_token": "not-a-valid-token",
	})

	err := app.Register(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Invalid or expired invitation link")
}

func TestApp_Register_InvitationOrganizationMismatch(t *testing.T) {
	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	testutil.CreateTestRoleExact(t, app.DB, org1.ID, "agent", true, true, nil)
	testutil.CreateTestRoleExact(t, app.DB, org2.ID, "agent", true, true, nil)

	inviteToken := createRegisterInviteToken(t, app, org1.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":            testutil.UniqueEmail("invite-mismatch"),
		"password":         "securepassword123",
		"full_name":        "Invite Mismatch",
		"organization_id":  org2.ID.String(),
		"invitation_token": inviteToken,
	})

	err := app.Register(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Invitation link organization mismatch")
}

func TestApp_CreateRegisterInvite_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	inviter := createUsersWriteUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"expires_in_hours": 2,
	})
	testutil.SetAuthContext(req, org.ID, inviter.ID)

	err := app.CreateRegisterInvite(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Token          string `json:"token"`
			OrganizationID string `json:"organization_id"`
			ExpiresAt      string `json:"expires_at"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Data.Token)
	require.Equal(t, org.ID.String(), resp.Data.OrganizationID)
	require.NotEmpty(t, resp.Data.ExpiresAt)

	token, err := jwt.ParseWithClaims(resp.Data.Token, &handlers.RegisterInviteClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(testutil.TestJWTSecret), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(*handlers.RegisterInviteClaims)
	require.True(t, ok)
	assert.Equal(t, org.ID, claims.OrganizationID)
	assert.Equal(t, "register_invite", claims.Purpose)
}

func TestApp_CreateRegisterInvite_ForbiddenWithoutPermission(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "users-read-only", []string{"users:read"})
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateRegisterInvite(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "Insufficient permissions")
}

func TestApp_RefreshToken_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("refresh")), testutil.WithPassword("password123"))
	refreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)
	seedRefreshTokenState(t, app, refreshToken, user.ID)

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
	assert.Greater(t, resp.Data.ExpiresIn, 0)
	assert.LessOrEqual(t, resp.Data.ExpiresIn, 15*60)

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
	seedRefreshTokenState(t, app, token, fakeUser.ID)

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
	seedRefreshTokenState(t, app, token, user.ID)

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
	assert.Equal(t, "whatomate", accessClaims.Issuer)

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

func TestApp_Login_CreatesActivityLogSuccess(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	password := "validpassword123"
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithPassword(password))

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    user.Email,
		"password": password,
	})

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var log models.ActivityLog
	require.NoError(t, app.DB.Where("user_id = ? AND event_type = ?", user.ID, "auth.login").Order("created_at DESC").First(&log).Error)
	assert.Equal(t, "auth", log.Category)
	assert.Equal(t, "login", log.Action)
	assert.Equal(t, "success", log.Status)
	assert.Equal(t, "auth", log.Source)
}

func TestApp_Login_CreatesActivityLogFailure(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithPassword("correctpassword123"))

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    user.Email,
		"password": "wrongpassword123",
	})

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))

	var log models.ActivityLog
	require.NoError(t, app.DB.Where("user_id = ? AND event_type = ?", user.ID, "auth.login_failed").Order("created_at DESC").First(&log).Error)
	assert.Equal(t, "auth", log.Category)
	assert.Equal(t, "login", log.Action)
	assert.Equal(t, "failure", log.Status)
	assert.Equal(t, "auth", log.Source)
}

func TestApp_Logout_CreatesActivityLog(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	refreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 24*time.Hour)
	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": refreshToken,
	})

	err := app.Logout(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var log models.ActivityLog
	require.NoError(t, app.DB.Where("user_id = ? AND event_type = ?", user.ID, "auth.logout").Order("created_at DESC").First(&log).Error)
	assert.Equal(t, "auth", log.Category)
	assert.Equal(t, "logout", log.Action)
	assert.Equal(t, "success", log.Status)
	assert.Equal(t, "auth", log.Source)
}
