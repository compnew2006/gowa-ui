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
	"github.com/zerodha/fastglue"
)

// =============================================================================
// LOGIN HANDLER TESTS
// =============================================================================

func TestLogin_Success_ValidCredentials(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("login-success")
	password := "validpassword123"
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword(password))

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": password,
	})

	err := app.Login(req)
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
				IsActive bool   `json:"is_active"`
			} `json:"user"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.Greater(t, resp.Data.ExpiresIn, 0)
	assert.LessOrEqual(t, resp.Data.ExpiresIn, 15*60)
	assert.Equal(t, email, resp.Data.User.Email)
	assert.Equal(t, user.ID.String(), resp.Data.User.ID)
	assert.True(t, resp.Data.User.IsActive)

	// Verify cookies are set
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_access"))
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_refresh"))
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_csrf"))
}

func TestLogin_Failure_InvalidEmail(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    "nonexistent@example.com",
		"password": "anypassword",
	})

	err := app.Login(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid credentials")
}

func TestLogin_Failure_InvalidPassword(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("wrong-password")
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword("correctpassword"))

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": "wrongpassword",
	})

	err := app.Login(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid credentials")
}

func TestLogin_Failure_InactiveUser(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("inactive-user")
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword("validpassword123"), testutil.WithInactive())

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": "validpassword123",
	})

	err := app.Login(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Account is disabled")
}

func TestLogin_Failure_RefreshTokenStorageUnavailable(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("login-no-refresh-storage")
	password := "validpassword123"
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword(password))
	app.Redis = nil

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": password,
	})

	err := app.Login(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusInternalServerError, "Failed to generate token")
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_refresh"))
}

func TestLogin_Failure_InvalidRequestBody(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.SetBody([]byte("invalid json"))
	req.RequestCtx.Request.Header.SetContentType("application/json")

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestLogin_Failure_MissingEmail(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]string{
		"password": "somepassword",
	})

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestLogin_Failure_MissingPassword(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]string{
		"email": "test@example.com",
	})

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestLogin_Success_WithRoleAndPermissions(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("role-perms")
	password := "testpassword123"

	// Create role with permissions
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "admin-role", []string{
		"users:write",
		"users:read",
		"campaigns:write",
	})

	// Cache the permissions for the role
	ctx := context.Background()
	permKeys := make([]string, 0, len(role.Permissions))
	for _, p := range role.Permissions {
		key := p.Resource + ":" + p.Action
		permKeys = append(permKeys, key)
	}
	cacheKey := fmt.Sprintf("role:perms:%d", role.ID)
	app.Redis.SAdd(ctx, cacheKey, permKeys)
	app.Redis.Expire(ctx, cacheKey, 24*time.Hour)

	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword(password), testutil.WithRoleID(&role.ID))

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": password,
	})

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			User struct {
				Role *struct {
					Permissions []models.Permission `json:"permissions"`
				} `json:"role"`
			} `json:"user"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	// Verify permissions are loaded
	require.NotNil(t, resp.Data.User.Role)
	assert.NotEmpty(t, resp.Data.User.Role.Permissions)
}

func TestLogin_LogsActivityOnSuccess(t *testing.T) {
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

	// Verify activity log was created
	var log models.ActivityLog
	err = app.DB.Where("user_id = ? AND event_type = ?", user.ID, "auth.login").
		Order("created_at DESC").
		First(&log).Error
	require.NoError(t, err)
	assert.Equal(t, "auth", log.Category)
	assert.Equal(t, "login", log.Action)
	assert.Equal(t, "success", log.Status)
}

func TestLogin_LogsActivityOnFailure(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithPassword("correctpassword"))

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    user.Email,
		"password": "wrongpassword",
	})

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))

	// Verify activity log was created
	var log models.ActivityLog
	err = app.DB.Where("user_id = ? AND event_type = ?", user.ID, "auth.login_failed").
		Order("created_at DESC").
		First(&log).Error
	require.NoError(t, err)
	assert.Equal(t, "auth", log.Category)
	assert.Equal(t, "login", log.Action)
	assert.Equal(t, "failure", log.Status)
}

// =============================================================================
// CREATE REGISTER INVITE TESTS
// =============================================================================

func TestCreateRegisterInvite_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	inviter := createUsersWriteUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"expires_in_hours": 24,
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

	assert.NotEmpty(t, resp.Data.Token)
	assert.Equal(t, org.ID.String(), resp.Data.OrganizationID)
	assert.NotEmpty(t, resp.Data.ExpiresAt)

	// Verify token can be parsed
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

func TestCreateRegisterInvite_Success_DefaultExpiry(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	inviter := createUsersWriteUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, org.ID, inviter.ID)

	err := app.CreateRegisterInvite(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Token     string `json:"token"`
			ExpiresAt string `json:"expires_at"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	// Verify default 24 hour expiry
	expiresAt, err := time.Parse(time.RFC3339, resp.Data.ExpiresAt)
	require.NoError(t, err)
	expectedExpiry := time.Now().Add(24 * time.Hour)
	diff := expectedExpiry.Sub(expiresAt)
	assert.Less(t, diff, time.Minute) // Allow 1 minute variance
}

func TestCreateRegisterInvite_Failure_Unauthorized(t *testing.T) {
	app := newTestApp(t)
	_ = testutil.CreateTestOrganization(t, app.DB) // Not used but needed for DB setup

	req := testutil.NewJSONRequest(t, map[string]any{
		"expires_in_hours": 24,
	})
	// No auth context set

	err := app.CreateRegisterInvite(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Unauthorized")
}

func TestCreateRegisterInvite_Failure_InsufficientPermissions(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "read-only", []string{"users:read"})
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateRegisterInvite(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "Insufficient permissions")
}

func TestCreateRegisterInvite_Failure_InvalidExpiryTooShort(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	inviter := createUsersWriteUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"expires_in_hours": 0,
	})
	testutil.SetAuthContext(req, org.ID, inviter.ID)

	err := app.CreateRegisterInvite(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "expires_in_hours must be between 1 and 168")
}

func TestCreateRegisterInvite_Failure_InvalidExpiryTooLong(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	inviter := createUsersWriteUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"expires_in_hours": 169,
	})
	testutil.SetAuthContext(req, org.ID, inviter.ID)

	err := app.CreateRegisterInvite(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "expires_in_hours must be between 1 and 168")
}

// =============================================================================
// REGISTER HANDLER TESTS
// =============================================================================

func TestRegister_Success_NewUser(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("new-register")
	_ = testutil.CreateTestRoleExact(t, app.DB, org.ID, "agent", true, true, nil) // Creates default role
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

	// Verify user was created in database
	var user models.User
	err = app.DB.Where("email = ?", email).First(&user).Error
	require.NoError(t, err)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, "New User", user.FullName)
	assert.NotNil(t, user.RoleID)

	// Verify user_organizations entry
	var userOrg models.UserOrganization
	err = app.DB.Where("user_id = ? AND organization_id = ?", user.ID, org.ID).First(&userOrg).Error
	require.NoError(t, err)
	assert.True(t, userOrg.IsDefault)
}

func TestRegister_ExistingUserDoesNotJoinNewOrg(t *testing.T) {
	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("multi-org-register")
	password := "sharedpassword123"

	// Create user in org1
	testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithEmail(email), testutil.WithPassword(password))

	// Create invite for org2
	testutil.CreateTestRoleExact(t, app.DB, org2.ID, "agent", true, true, nil)
	inviteToken := createRegisterInviteToken(t, app, org2.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":            email,
		"password":         password,
		"full_name":        "Same User",
		"organization_id":  org2.ID.String(),
		"invitation_token": inviteToken,
	})

	err := app.Register(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Verify user still only belongs to original org
	var user models.User
	err = app.DB.Where("email = ?", email).First(&user).Error
	require.NoError(t, err)

	var userOrgs []models.UserOrganization
	err = app.DB.Where("user_id = ?", user.ID).Find(&userOrgs).Error
	require.NoError(t, err)
	assert.Len(t, userOrgs, 1)
}

func TestRegister_Failure_MissingInvitationToken(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":           testutil.UniqueEmail("no-invite"),
		"password":        "securepassword123",
		"full_name":       "No Invite",
		"organization_id": org.ID.String(),
	})

	err := app.Register(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "invitation_token is required")
}

func TestRegister_Failure_InvalidInvitationToken(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":            testutil.UniqueEmail("invalid-invite"),
		"password":         "securepassword123",
		"full_name":        "Invalid Invite",
		"organization_id":  org.ID.String(),
		"invitation_token": "invalid-token",
	})

	err := app.Register(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Invalid or expired invitation link")
}

func TestRegister_Failure_OrganizationMismatch(t *testing.T) {
	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	testutil.CreateTestRoleExact(t, app.DB, org1.ID, "agent", true, true, nil)
	testutil.CreateTestRoleExact(t, app.DB, org2.ID, "agent", true, true, nil)

	inviteToken := createRegisterInviteToken(t, app, org1.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":            testutil.UniqueEmail("org-mismatch"),
		"password":         "securepassword123",
		"full_name":        "Org Mismatch",
		"organization_id":  org2.ID.String(),
		"invitation_token": inviteToken,
	})

	err := app.Register(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Invitation link organization mismatch")
}

func TestRegister_ExistingUserWrongPassword_GenericResponse(t *testing.T) {
	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("wrong-pwd-existing")

	// Create user in org1
	testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithEmail(email), testutil.WithPassword("correctpassword"))

	// Create invite for org2
	testutil.CreateTestRoleExact(t, app.DB, org2.ID, "agent", true, true, nil)
	inviteToken := createRegisterInviteToken(t, app, org2.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":            email,
		"password":         "wrongpassword",
		"full_name":        "Wrong Password",
		"organization_id":  org2.ID.String(),
		"invitation_token": inviteToken,
	})

	err := app.Register(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
}

func TestRegister_ExistingUserAlreadyInOrg_GenericResponse(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("already-in-org")
	password := "password123"

	// Create user in org
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword(password))

	// Create another invite for same org
	testutil.CreateTestRoleExact(t, app.DB, org.ID, "agent", true, true, nil)
	inviteToken := createRegisterInviteToken(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":            email,
		"password":         password,
		"full_name":        "Already In Org",
		"organization_id":  org.ID.String(),
		"invitation_token": inviteToken,
	})

	err := app.Register(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
}

func TestRegister_Failure_WeakPassword(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	testutil.CreateTestRoleExact(t, app.DB, org.ID, "agent", true, true, nil)
	inviteToken := createRegisterInviteToken(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"email":            testutil.UniqueEmail("weak-pwd"),
		"password":         "weak",
		"full_name":        "Weak Password",
		"organization_id":  org.ID.String(),
		"invitation_token": inviteToken,
	})

	err := app.Register(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "password")
}

func TestRegister_Failure_InvalidRequestBody(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.SetBody([]byte("invalid json"))
	req.RequestCtx.Request.Header.SetContentType("application/json")

	err := app.Register(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

// =============================================================================
// REFRESH TOKEN HANDLER TESTS
// =============================================================================

func TestRefreshToken_Success_ViaRequestBody(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
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

	// Verify new tokens are set
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_access"))
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_refresh"))
}

func TestRefreshToken_Success_ViaCookie(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	refreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)
	seedRefreshTokenState(t, app, refreshToken, user.ID)

	req := testutil.NewJSONRequest(t, map[string]any{})
	req.RequestCtx.Request.Header.SetCookie("whm_refresh", refreshToken)

	err := app.RefreshToken(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Verify new tokens are set
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_access"))
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_refresh"))
}

func TestRefreshToken_Failure_MissingToken(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]any{})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Missing refresh token")
}

func TestRefreshToken_Failure_ExpiredToken(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	expiredToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, -time.Hour)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": expiredToken,
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid refresh token")
}

func TestRefreshToken_Failure_InvalidSignature(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	wrongSecretToken := testutil.GenerateTestRefreshToken(t, user, "wrong-secret-key-12345678901234567890", 7*24*time.Hour)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": wrongSecretToken,
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid refresh token")
}

func TestRefreshToken_Failure_RevokedToken(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	refreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)

	// Don't seed the token in Redis (simulating revocation)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": refreshToken,
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Refresh token has been revoked")
}

func TestRefreshToken_Failure_UserNoLongerInTokenOrganization(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	refreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)
	seedRefreshTokenState(t, app, refreshToken, user.ID)

	require.NoError(t, app.DB.Where("user_id = ? AND organization_id = ?", user.ID, org.ID).
		Delete(&models.UserOrganization{}).Error)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": refreshToken,
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid refresh token")
}

func TestRefreshToken_Failure_UserNotFound(t *testing.T) {
	app := newTestApp(t)
	fakeUser := &models.User{
		BaseModel:      models.BaseModel{ID: uuid.New()},
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

func TestRefreshToken_Failure_DisabledUser(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithInactive())
	token := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)
	seedRefreshTokenState(t, app, token, user.ID)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": token,
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Account is disabled")
}

func TestRefreshToken_Failure_MalformedToken(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": "not.a.valid.jwt",
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid refresh token")
}

func TestRefreshToken_Failure_StorageUnavailable(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	refreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)
	app.Redis = nil

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": refreshToken,
	})

	err := app.RefreshToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusInternalServerError, "Refresh token storage is unavailable")
}

func TestRefreshToken_TokenRotation(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	oldRefreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)

	// Seed old token
	token, err := jwt.ParseWithClaims(oldRefreshToken, &middleware.JWTClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(testutil.TestJWTSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	require.NoError(t, err)
	claims, ok := token.Claims.(*middleware.JWTClaims)
	require.True(t, ok)

	ctx := context.Background()
	app.Redis.Set(ctx, fmt.Sprintf("refresh:%s", claims.ID), user.ID.String(), 24*time.Hour)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": oldRefreshToken,
	})

	err = app.RefreshToken(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Verify old token was revoked
	exists, _ := app.Redis.Exists(ctx, fmt.Sprintf("refresh:%s", claims.ID)).Result()
	assert.Equal(t, int64(0), exists)

	// Verify new token was issued
	newRefreshToken := testutil.GetResponseCookie(req, "whm_refresh")
	assert.NotEmpty(t, newRefreshToken)
	assert.NotEqual(t, oldRefreshToken, newRefreshToken)

	// Verify new token is in Redis
	newToken, err := jwt.ParseWithClaims(newRefreshToken, &middleware.JWTClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(testutil.TestJWTSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	require.NoError(t, err)
	newClaims, ok := newToken.Claims.(*middleware.JWTClaims)
	require.True(t, ok)

	exists, _ = app.Redis.Exists(ctx, fmt.Sprintf("refresh:%s", newClaims.ID)).Result()
	assert.Equal(t, int64(1), exists)
}

// =============================================================================
// SWITCH ORGANIZATION HANDLER TESTS
// =============================================================================

func TestSwitchOrg_Success_MemberOfOrg(t *testing.T) {
	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org1.ID)

	// Add user to org2 with a different role
	role2 := testutil.CreateTestRoleWithKeys(t, app.DB, org2.ID, "org2-role", []string{"campaigns:write"})
	userOrg2 := &models.UserOrganization{
		UserID:         user.ID,
		OrganizationID: org2.ID,
		RoleID:         &role2.ID,
		IsDefault:      false,
	}
	require.NoError(t, app.DB.Create(userOrg2).Error)

	req := testutil.NewJSONRequest(t, map[string]any{
		"organization_id": org2.ID.String(),
	})
	testutil.SetAuthContext(req, org1.ID, user.ID)

	err := app.SwitchOrg(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			User struct {
				OrganizationID string `json:"organization_id"`
			} `json:"user"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, org2.ID.String(), resp.Data.User.OrganizationID)

	// Verify new tokens are set
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_access"))
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_refresh"))
}

func TestSwitchOrg_Success_SuperAdmin(t *testing.T) {
	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithSuperAdmin())

	// Super admin not explicitly in org2
	req := testutil.NewJSONRequest(t, map[string]any{
		"organization_id": org2.ID.String(),
	})
	testutil.SetAuthContext(req, org1.ID, user.ID)

	err := app.SwitchOrg(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			User struct {
				OrganizationID string `json:"organization_id"`
			} `json:"user"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, org2.ID.String(), resp.Data.User.OrganizationID)
}

func TestSwitchOrg_Failure_NotMemberOfOrg(t *testing.T) {
	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org1.ID)

	// User not in org2
	req := testutil.NewJSONRequest(t, map[string]any{
		"organization_id": org2.ID.String(),
	})
	testutil.SetAuthContext(req, org1.ID, user.ID)

	err := app.SwitchOrg(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "You are not a member of this organization")
}

func TestSwitchOrg_Failure_OrganizationNotFound(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	fakeOrgID := uuid.New()

	req := testutil.NewJSONRequest(t, map[string]any{
		"organization_id": fakeOrgID.String(),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.SwitchOrg(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "Organization not found")
}

func TestSwitchOrg_Failure_MissingOrganizationID(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.SwitchOrg(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "organization_id is required")
}

func TestSwitchOrg_Failure_Unauthorized(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	req := testutil.NewJSONRequest(t, map[string]any{
		"organization_id": org.ID.String(),
	})
	// No auth context

	err := app.SwitchOrg(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Unauthorized")
}

func TestSwitchOrg_Success_LoadsRolePermissions(t *testing.T) {
	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org1.ID)

	// Create role with permissions in org2
	role2 := testutil.CreateTestRoleWithKeys(t, app.DB, org2.ID, "org2-admin", []string{
		"users:write",
		"campaigns:write",
	})
	userOrg2 := &models.UserOrganization{
		UserID:         user.ID,
		OrganizationID: org2.ID,
		RoleID:         &role2.ID,
		IsDefault:      false,
	}
	require.NoError(t, app.DB.Create(userOrg2).Error)

	// Cache permissions
	ctx := context.Background()
	permKeys := make([]string, 0, len(role2.Permissions))
	for _, p := range role2.Permissions {
		key := p.Resource + ":" + p.Action
		permKeys = append(permKeys, key)
	}
	cacheKey := fmt.Sprintf("role:perms:%d", role2.ID)
	app.Redis.SAdd(ctx, cacheKey, permKeys)
	app.Redis.Expire(ctx, cacheKey, 24*time.Hour)

	req := testutil.NewJSONRequest(t, map[string]any{
		"organization_id": org2.ID.String(),
	})
	testutil.SetAuthContext(req, org1.ID, user.ID)

	err := app.SwitchOrg(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			User struct {
				Role *struct {
					Permissions []models.Permission `json:"permissions"`
				} `json:"role"`
			} `json:"user"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	// Verify permissions are loaded
	require.NotNil(t, resp.Data.User.Role)
	assert.NotEmpty(t, resp.Data.User.Role.Permissions)
}

// =============================================================================
// LOGOUT HANDLER TESTS
// =============================================================================

func TestLogout_Success_ViaRequestBody(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	refreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)
	seedRefreshTokenState(t, app, refreshToken, user.ID)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": refreshToken,
	})

	err := app.Logout(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data map[string]string `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "logged_out", resp.Data["status"])

	// Verify cookies are cleared
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_access"))
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_refresh"))
}

func TestLogout_Success_ViaCookie(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	refreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)
	seedRefreshTokenState(t, app, refreshToken, user.ID)

	req := testutil.NewJSONRequest(t, map[string]any{})
	req.RequestCtx.Request.Header.SetCookie("whm_refresh", refreshToken)

	err := app.Logout(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Verify token was revoked from Redis
	token, err := jwt.ParseWithClaims(refreshToken, &middleware.JWTClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(testutil.TestJWTSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	require.NoError(t, err)
	claims, ok := token.Claims.(*middleware.JWTClaims)
	require.True(t, ok)

	ctx := context.Background()
	exists, _ := app.Redis.Exists(ctx, fmt.Sprintf("refresh:%s", claims.ID)).Result()
	assert.Equal(t, int64(0), exists)
}

func TestLogout_Success_NoToken(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]any{})

	err := app.Logout(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data map[string]string `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "logged_out", resp.Data["status"])
}

func TestLogout_LogsActivity(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	refreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)
	seedRefreshTokenState(t, app, refreshToken, user.ID)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": refreshToken,
	})

	err := app.Logout(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Verify activity log was created
	var log models.ActivityLog
	err = app.DB.Where("user_id = ? AND event_type = ?", user.ID, "auth.logout").
		Order("created_at DESC").
		First(&log).Error
	require.NoError(t, err)
	assert.Equal(t, "auth", log.Category)
	assert.Equal(t, "logout", log.Action)
	assert.Equal(t, "success", log.Status)
}

func TestLogout_ClearsAllAuthCookies(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	refreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)
	seedRefreshTokenState(t, app, refreshToken, user.ID)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": refreshToken,
	})

	err := app.Logout(req)
	require.NoError(t, err)

	// Verify all auth cookies are cleared
	cookies := []string{"whm_access", "whm_refresh", "whm_csrf"}
	for _, cookieName := range cookies {
		cookie := testutil.GetResponseCookie(req, cookieName)
		assert.Empty(t, cookie, fmt.Sprintf("Cookie %s should be cleared", cookieName))
	}
}

// =============================================================================
// GET WS TOKEN HANDLER TESTS
// =============================================================================

func TestGetWSToken_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetWSToken(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.Data.Token)

	// Verify token can be parsed
	token, err := jwt.ParseWithClaims(resp.Data.Token, &middleware.JWTClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(testutil.TestJWTSecret), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(*middleware.JWTClaims)
	require.True(t, ok)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, org.ID, claims.OrganizationID)
	assert.Equal(t, "ws_token", claims.Subject)

	// Verify token expires in ~30 seconds
	expiresAt := claims.ExpiresAt.Time
	expectedExpiry := time.Now().Add(30 * time.Second)
	diff := expectedExpiry.Sub(expiresAt)
	assert.Less(t, diff, 5*time.Second) // Allow 5 second variance
}

func TestGetWSToken_Failure_Unauthorized(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewGETRequest(t)
	// No auth context

	err := app.GetWSToken(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Unauthorized")
}

func TestGetWSToken_Success_SuperAdmin(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithSuperAdmin())

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetWSToken(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.Data.Token)
}

// =============================================================================
// TOKEN VALIDATION TESTS
// =============================================================================

func TestAccessToken_Claims(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("token-claims")
	password := "password123"
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword(password))

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": password,
	})

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	accessToken := testutil.GetResponseCookie(req, "whm_access")
	require.NotEmpty(t, accessToken)

	// Parse and verify access token
	token, err := jwt.ParseWithClaims(accessToken, &middleware.JWTClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(testutil.TestJWTSecret), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(*middleware.JWTClaims)
	require.True(t, ok)

	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, org.ID, claims.OrganizationID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, user.RoleID, claims.RoleID)
	assert.Equal(t, "whatomate", claims.Issuer)
	assert.Equal(t, "access", claims.Subject)

	// Verify expiration
	expiresAt := claims.ExpiresAt.Time
	expectedExpiry := time.Now().Add(15 * time.Minute)
	diff := expectedExpiry.Sub(expiresAt)
	assert.Less(t, diff, time.Minute) // Allow 1 minute variance
}

func TestRefreshToken_Claims(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    user.Email,
		"password": "password123",
	})

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	refreshToken := testutil.GetResponseCookie(req, "whm_refresh")
	require.NotEmpty(t, refreshToken)

	// Parse and verify refresh token
	token, err := jwt.ParseWithClaims(refreshToken, &middleware.JWTClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(testutil.TestJWTSecret), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(*middleware.JWTClaims)
	require.True(t, ok)

	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, org.ID, claims.OrganizationID)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, user.RoleID, claims.RoleID)
	assert.Equal(t, "whatomate", claims.Issuer)
	assert.Equal(t, "refresh", claims.Subject)
	assert.NotEmpty(t, claims.ID) // JWT ID for revocation

	// Verify expiration (7 days)
	expiresAt := claims.ExpiresAt.Time
	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	diff := expectedExpiry.Sub(expiresAt)
	assert.Less(t, diff, time.Minute) // Allow 1 minute variance
}

// =============================================================================
// COOKIE SECURITY TESTS
// =============================================================================

func TestAuthCookies_SecurityAttributes(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("cookie-security")
	password := "password123"
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword(password))

	req := testutil.NewJSONRequest(t, map[string]string{
		"email":    email,
		"password": password,
	})

	err := app.Login(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Check access cookie
	accessCookie := getCookieDetails(req, "whm_access")
	require.NotNil(t, accessCookie)
	assert.Equal(t, "whm_access", accessCookie.Key())
	assert.True(t, accessCookie.HTTPOnly(), "Access cookie should be HTTPOnly")

	// Check refresh cookie
	refreshCookie := getCookieDetails(req, "whm_refresh")
	require.NotNil(t, refreshCookie)
	assert.Equal(t, "whm_refresh", refreshCookie.Key())
	assert.True(t, refreshCookie.HTTPOnly(), "Refresh cookie should be HTTPOnly")
}

func TestLogout_CookiesCleared(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	refreshToken := testutil.GenerateTestRefreshToken(t, user, testutil.TestJWTSecret, 7*24*time.Hour)
	seedRefreshTokenState(t, app, refreshToken, user.ID)

	req := testutil.NewJSONRequest(t, map[string]string{
		"refresh_token": refreshToken,
	})

	err := app.Logout(req)
	require.NoError(t, err)

	// Check that cookies are cleared (max-age=0 or empty)
	accessCookie := getCookieDetails(req, "whm_access")
	if accessCookie != nil {
		// Cookie should be expired or empty
		assert.True(t, accessCookie.Expire().Unix() <= time.Now().Add(-24*time.Hour).Unix() ||
			len(accessCookie.Value()) == 0)
	}

	refreshCookie := getCookieDetails(req, "whm_refresh")
	if refreshCookie != nil {
		assert.True(t, refreshCookie.Expire().Unix() <= time.Now().Add(-24*time.Hour).Unix() ||
			len(refreshCookie.Value()) == 0)
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// getCookieDetails extracts cookie details from response
func getCookieDetails(req *fastglue.Request, name string) *fasthttp.Cookie {
	var found *fasthttp.Cookie
	req.RequestCtx.Response.Header.VisitAllCookie(func(key, val []byte) {
		c := fasthttp.AcquireCookie()
		defer fasthttp.ReleaseCookie(c)
		if err := c.ParseBytes(val); err == nil && string(c.Key()) == name {
			// Copy the cookie before returning
			foundCopy := fasthttp.AcquireCookie()
			foundCopy.SetKeyBytes(c.Key())
			foundCopy.SetValueBytes(c.Value())
			foundCopy.SetHTTPOnly(c.HTTPOnly())
			foundCopy.SetSecure(c.Secure())
			foundCopy.SetPathBytes(c.Path())
			foundCopy.SetDomainBytes(c.Domain())
			foundCopy.SetExpire(c.Expire())
			found = foundCopy
		}
	})
	return found
}
