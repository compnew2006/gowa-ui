package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	appcrypto "github.com/compnew2006/whatomate/internal/crypto"
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

// Helper function to create a test SSO provider
func createTestSSOProvider(t *testing.T, db *gorm.DB, orgID uuid.UUID, provider string, enabled bool) *models.SSOProvider {
	t.Helper()

	ssoProvider := &models.SSOProvider{
		OrganizationID:  orgID,
		Provider:        provider,
		ClientID:        "test-client-id",
		ClientSecret:    "test-client-secret",
		IsEnabled:       enabled,
		AllowAutoCreate: true,
		DefaultRoleName: "agent",
		AllowedDomains:  "example.com,test.com",
	}

	if provider == "custom" {
		ssoProvider.AuthURL = "https://custom.example.com/oauth/authorize"
		ssoProvider.TokenURL = "https://custom.example.com/oauth/token"
		ssoProvider.UserInfoURL = "https://custom.example.com/oauth/userinfo"
	}

	// Encrypt the secret
	encSecret, err := appcrypto.Encrypt(ssoProvider.ClientSecret, "test-encryption-key")
	require.NoError(t, err)
	ssoProvider.ClientSecret = encSecret

	require.NoError(t, db.Create(ssoProvider).Error)
	return ssoProvider
}

// Helper function to create a test role with SSO settings permissions
func createSSOAdminRole(t *testing.T, app *handlers.App, orgID uuid.UUID) *models.CustomRole {
	t.Helper()

	return testutil.CreateTestRoleWithKeys(t, app.DB, orgID, "sso-admin", []string{
		"settings:sso:read",
		"settings:sso:write",
	})
}

// Helper function to create an authenticated request for SSO operations
func createSSOAdminRequest(t *testing.T, app *handlers.App, orgID uuid.UUID) (*fastglue.Request, *models.User) {
	t.Helper()

	role := createSSOAdminRole(t, app, orgID)
	user := testutil.CreateTestUser(t, app.DB, orgID, testutil.WithRoleID(&role.ID))

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, orgID, user.ID)

	return req, user
}

// Mock HTTP server for OAuth provider
func createMockOAuthServer(t *testing.T, userInfoHandler http.HandlerFunc) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// Token endpoint
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"refresh_token": "test-refresh-token",
		})
	})

	// User info endpoint
	mux.HandleFunc("/oauth/userinfo", userInfoHandler)

	return httptest.NewServer(mux)
}

// Test GetPublicSSOProviders
func TestApp_GetPublicSSOProviders_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create multiple enabled SSO providers
	createTestSSOProvider(t, app.DB, org.ID, "google", true)
	createTestSSOProvider(t, app.DB, org.ID, "microsoft", true)
	createTestSSOProvider(t, app.DB, org.ID, "github", true)

	// Create a disabled provider (should not appear)
	createTestSSOProvider(t, app.DB, org.ID, "facebook", false)

	req := testutil.NewJSONRequest(t, nil)

	err := app.GetPublicSSOProviders(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string                       `json:"status"`
		Data   []handlers.SSOProviderPublic `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.Len(t, resp.Data, 3)

	// Check that providers are deduplicated and have display names
	providers := make(map[string]handlers.SSOProviderPublic)
	for _, p := range resp.Data {
		providers[p.Provider] = p
	}

	assert.Contains(t, providers, "google")
	assert.Equal(t, "Google", providers["google"].Name)
	assert.Contains(t, providers, "microsoft")
	assert.Equal(t, "Microsoft", providers["microsoft"].Name)
	assert.Contains(t, providers, "github")
	assert.Equal(t, "GitHub", providers["github"].Name)
	assert.NotContains(t, providers, "facebook")
}

func TestApp_GetPublicSSOProviders_NoProviders(t *testing.T) {
	app := newTestApp(t)
	_ = testutil.CreateTestOrganization(t, app.DB) // Org created for DB setup

	// No providers configured
	req := testutil.NewJSONRequest(t, nil)

	err := app.GetPublicSSOProviders(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string                       `json:"status"`
		Data   []handlers.SSOProviderPublic `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.Empty(t, resp.Data)
}

func TestApp_GetPublicSSOProviders_Deduplication(t *testing.T) {
	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)

	// Create same provider in multiple organizations
	createTestSSOProvider(t, app.DB, org1.ID, "google", true)
	createTestSSOProvider(t, app.DB, org2.ID, "google", true)

	req := testutil.NewJSONRequest(t, nil)

	err := app.GetPublicSSOProviders(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string                       `json:"status"`
		Data   []handlers.SSOProviderPublic `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	// Should only return one Google provider despite being in multiple orgs
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, "google", resp.Data[0].Provider)
}

// Test InitSSO
func TestApp_InitSSO_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	createTestSSOProvider(t, app.DB, org.ID, "google", true)

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "google")

	err := app.InitSSO(req)
	require.NoError(t, err)

	// Should redirect to OAuth provider
	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, statusCode)

	// Check redirect URL contains provider's auth URL
	redirectURL := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, redirectURL, "accounts.google.com")
	assert.Contains(t, redirectURL, "client_id=test-client-id")

	// Verify state was stored in Redis
	stateNonce := extractStateFromAuthURL(t, redirectURL)
	stateKey := "sso:state:" + stateNonce

	var state handlers.SSOState
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stateJSON, err := app.Redis.Get(ctx, stateKey).Bytes()
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(stateJSON, &state))

	assert.Equal(t, "google", state.Provider)
	assert.Equal(t, org.ID.String(), state.OrgID)
	assert.False(t, state.ExpiresAt.IsZero())
}

func TestApp_InitSSO_InvalidProvider(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "invalid-provider")

	err := app.InitSSO(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Invalid SSO provider")
}

func TestApp_InitSSO_ProviderNotConfigured(t *testing.T) {
	app := newTestApp(t)
	_ = testutil.CreateTestOrganization(t, app.DB) // Org created for DB setup

	// Don't create any SSO provider configuration
	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "google")

	err := app.InitSSO(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "SSO provider not configured or disabled")
}

func TestApp_InitSSO_ProviderDisabled(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	createTestSSOProvider(t, app.DB, org.ID, "google", false) // Disabled

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "google")

	err := app.InitSSO(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "SSO provider not configured or disabled")
}

func TestApp_InitSSO_CustomProvider(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	createTestSSOProvider(t, app.DB, org.ID, "custom", true)

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "custom")

	err := app.InitSSO(req)
	require.NoError(t, err)

	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, statusCode)

	redirectURL := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, redirectURL, "custom.example.com")
}

// Test CallbackSSO
func TestApp_CallbackSSO_Success(t *testing.T) {
	// Create mock OAuth server
	mockServer := createMockOAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "test-user-id",
			"email": "test@example.com",
			"name":  "Test User",
		})
	})
	defer mockServer.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create custom SSO provider pointing to mock server
	ssoProvider := createTestSSOProvider(t, app.DB, org.ID, "custom", true)
	ssoProvider.AuthURL = mockServer.URL + "/oauth/authorize"
	ssoProvider.TokenURL = mockServer.URL + "/oauth/token"
	ssoProvider.UserInfoURL = mockServer.URL + "/oauth/userinfo"
	app.DB.Save(ssoProvider)

	// Create test role for auto-create user
	_ = testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "agent", nil)

	// Set up state in Redis
	stateNonce := "test-state-nonce"
	state := handlers.SSOState{
		OrgID:     org.ID.String(),
		Provider:  "custom",
		Nonce:     stateNonce,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	stateJSON, _ := json.Marshal(state)
	stateKey := "sso:state:" + stateNonce

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, app.Redis.Set(ctx, stateKey, stateJSON, 5*time.Minute).Err())

	// Create callback request
	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "custom")
	req.RequestCtx.QueryArgs().Add("code", "test-auth-code")
	req.RequestCtx.QueryArgs().Add("state", stateNonce)

	err := app.CallbackSSO(req)
	require.NoError(t, err)

	// Should redirect to frontend callback page
	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, statusCode)

	redirectURL := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, redirectURL, "/auth/sso/callback")

	// Verify user was created
	var user models.User
	err = app.DB.Where("email = ?", "test@example.com").First(&user).Error
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.FullName)
	assert.Equal(t, "custom", user.SSOProvider)
	assert.Equal(t, "test-user-id", user.SSOProviderID)
	assert.True(t, user.IsActive)

	// Verify auth cookies were set
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_access"))
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_refresh"))
}

func TestApp_CallbackSSO_OAuthError(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "google")
	req.RequestCtx.QueryArgs().Add("error", "access_denied")
	req.RequestCtx.QueryArgs().Add("error_description", "User denied access")

	err := app.CallbackSSO(req)
	require.NoError(t, err)

	// Should redirect to login with error
	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, statusCode)

	redirectURL := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, redirectURL, "/login")
	assert.Contains(t, redirectURL, "sso_error")
}

func TestApp_CallbackSSO_MissingParameters(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "google")
	// Missing code and state

	err := app.CallbackSSO(req)
	require.NoError(t, err)

	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, statusCode)

	redirectURL := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, redirectURL, "Invalid callback parameters")
}

func TestApp_CallbackSSO_InvalidState(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "google")
	req.RequestCtx.QueryArgs().Add("code", "test-code")
	req.RequestCtx.QueryArgs().Add("state", "invalid-state")

	err := app.CallbackSSO(req)
	require.NoError(t, err)

	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, statusCode)

	redirectURL := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, redirectURL, "Invalid or expired state")
}

func TestApp_CallbackSSO_ExpiredState(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create expired state
	stateNonce := "test-state-nonce"
	state := handlers.SSOState{
		OrgID:     org.ID.String(),
		Provider:  "google",
		Nonce:     stateNonce,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	stateJSON, _ := json.Marshal(state)
	stateKey := "sso:state:" + stateNonce

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, app.Redis.Set(ctx, stateKey, stateJSON, 5*time.Minute).Err())

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "google")
	req.RequestCtx.QueryArgs().Add("code", "test-code")
	req.RequestCtx.QueryArgs().Add("state", stateNonce)

	err := app.CallbackSSO(req)
	require.NoError(t, err)

	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, statusCode)

	redirectURL := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, redirectURL, "Invalid or expired state")
}

func TestApp_CallbackSSO_EmailDomainRestriction(t *testing.T) {
	mockServer := createMockOAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "test-user-id",
			"email": "user@restricted.com", // Not in allowed domains
			"name":  "Test User",
		})
	})
	defer mockServer.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create SSO provider with domain restriction
	ssoProvider := createTestSSOProvider(t, app.DB, org.ID, "custom", true)
	ssoProvider.AllowedDomains = "example.com,allowed.com"
	ssoProvider.AuthURL = mockServer.URL + "/oauth/authorize"
	ssoProvider.TokenURL = mockServer.URL + "/oauth/token"
	ssoProvider.UserInfoURL = mockServer.URL + "/oauth/userinfo"
	app.DB.Save(ssoProvider)

	// Set up state
	stateNonce := "test-state-nonce"
	state := handlers.SSOState{
		OrgID:     org.ID.String(),
		Provider:  "custom",
		Nonce:     stateNonce,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	stateJSON, _ := json.Marshal(state)
	stateKey := "sso:state:" + stateNonce

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, app.Redis.Set(ctx, stateKey, stateJSON, 5*time.Minute).Err())

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "custom")
	req.RequestCtx.QueryArgs().Add("code", "test-code")
	req.RequestCtx.QueryArgs().Add("state", stateNonce)

	err := app.CallbackSSO(req)
	require.NoError(t, err)

	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, statusCode)

	redirectURL := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, redirectURL, "Email domain not allowed")
}

func TestApp_CallbackSSO_ExistingUserUpdatesSSOInfo(t *testing.T) {
	mockServer := createMockOAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "test-user-id",
			"email": "existing@example.com",
			"name":  "Existing User",
		})
	})
	defer mockServer.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create existing user without SSO info
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "agent", nil)
	existingUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail("existing@example.com"), testutil.WithRoleID(&role.ID))

	// Create SSO provider
	ssoProvider := createTestSSOProvider(t, app.DB, org.ID, "custom", true)
	ssoProvider.AuthURL = mockServer.URL + "/oauth/authorize"
	ssoProvider.TokenURL = mockServer.URL + "/oauth/token"
	ssoProvider.UserInfoURL = mockServer.URL + "/oauth/userinfo"
	app.DB.Save(ssoProvider)

	// Set up state
	stateNonce := "test-state-nonce"
	state := handlers.SSOState{
		OrgID:     org.ID.String(),
		Provider:  "custom",
		Nonce:     stateNonce,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	stateJSON, _ := json.Marshal(state)
	stateKey := "sso:state:" + stateNonce

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, app.Redis.Set(ctx, stateKey, stateJSON, 5*time.Minute).Err())

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "custom")
	req.RequestCtx.QueryArgs().Add("code", "test-code")
	req.RequestCtx.QueryArgs().Add("state", stateNonce)

	err := app.CallbackSSO(req)
	require.NoError(t, err)

	// Verify user's SSO info was updated
	var updatedUser models.User
	err = app.DB.First(&updatedUser, existingUser.ID).Error
	require.NoError(t, err)
	assert.Equal(t, "custom", updatedUser.SSOProvider)
	assert.Equal(t, "test-user-id", updatedUser.SSOProviderID)
}

func TestApp_CallbackSSO_InactiveUser(t *testing.T) {
	mockServer := createMockOAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "test-user-id",
			"email": "inactive@example.com",
			"name":  "Inactive User",
		})
	})
	defer mockServer.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create inactive user
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "agent", nil)
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail("inactive@example.com"), testutil.WithRoleID(&role.ID), testutil.WithInactive())

	// Create SSO provider
	ssoProvider := createTestSSOProvider(t, app.DB, org.ID, "custom", true)
	ssoProvider.AuthURL = mockServer.URL + "/oauth/authorize"
	ssoProvider.TokenURL = mockServer.URL + "/oauth/token"
	ssoProvider.UserInfoURL = mockServer.URL + "/oauth/userinfo"
	app.DB.Save(ssoProvider)

	// Set up state
	stateNonce := "test-state-nonce"
	state := handlers.SSOState{
		OrgID:     org.ID.String(),
		Provider:  "custom",
		Nonce:     stateNonce,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	stateJSON, _ := json.Marshal(state)
	stateKey := "sso:state:" + stateNonce

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, app.Redis.Set(ctx, stateKey, stateJSON, 5*time.Minute).Err())

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "custom")
	req.RequestCtx.QueryArgs().Add("code", "test-code")
	req.RequestCtx.QueryArgs().Add("state", stateNonce)

	err := app.CallbackSSO(req)
	require.NoError(t, err)

	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, statusCode)

	redirectURL := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, redirectURL, "Account is disabled")
}

// Test GetSSOSettings
func TestApp_GetSSOSettings_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	req, _ := createSSOAdminRequest(t, app, org.ID)

	// Create SSO providers
	createTestSSOProvider(t, app.DB, org.ID, "google", true)
	createTestSSOProvider(t, app.DB, org.ID, "microsoft", false)

	err := app.GetSSOSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string                         `json:"status"`
		Data   []handlers.SSOProviderResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.Len(t, resp.Data, 2)

	// Check that secrets are masked
	for _, provider := range resp.Data {
		assert.NotEmpty(t, provider.ClientID)
		assert.True(t, provider.HasSecret)
		// ClientSecret is not exposed in API response for security
	}
}

func TestApp_GetSSOSettings_Unauthorized(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create user without SSO permissions
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "user", nil)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetSSOSettings(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission_denied")
}

func TestApp_GetSSOSettings_NoProviders(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	req, _ := createSSOAdminRequest(t, app, org.ID)

	err := app.GetSSOSettings(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string                         `json:"status"`
		Data   []handlers.SSOProviderResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Empty(t, resp.Data)
}

// Test UpdateSSOProvider
func TestApp_UpdateSSOProvider_CreateNew(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	req, _ := createSSOAdminRequest(t, app, org.ID)
	req.RequestCtx.SetUserValue("provider", "google")

	body := map[string]any{
		"client_id":        "new-client-id",
		"client_secret":    "new-client-secret",
		"is_enabled":       true,
		"allow_auto_create": true,
		"default_role":     "agent",
		"allowed_domains":  "example.com",
	}
	jsonBody, _ := json.Marshal(body); req.RequestCtx.Request.SetBody(jsonBody)

	err := app.UpdateSSOProvider(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string                       `json:"status"`
		Data   handlers.SSOProviderResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, "google", resp.Data.Provider)
	assert.Equal(t, "new-client-id", resp.Data.ClientID)
	assert.True(t, resp.Data.IsEnabled)
	assert.True(t, resp.Data.AllowAutoCreate)
	assert.Equal(t, "agent", resp.Data.DefaultRole)
	assert.Equal(t, "example.com", resp.Data.AllowedDomains)

	// Verify in database
	var provider models.SSOProvider
	err = app.DB.Where("organization_id = ? AND provider = ?", org.ID, "google").First(&provider).Error
	require.NoError(t, err)
	assert.Equal(t, "new-client-id", provider.ClientID)
	assert.NotEmpty(t, provider.ClientSecret) // Should be encrypted
}

func TestApp_UpdateSSOProvider_UpdateExisting(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create existing provider
	existingProvider := createTestSSOProvider(t, app.DB, org.ID, "google", true)

	req, _ := createSSOAdminRequest(t, app, org.ID)
	req.RequestCtx.SetUserValue("provider", "google")

	body := map[string]any{
		"client_id":        "updated-client-id",
		"client_secret":    "", // Don't update secret
		"is_enabled":       false,
		"allow_auto_create": false,
		"default_role":     "admin",
		"allowed_domains":  "updated.com",
	}
	jsonBody, _ := json.Marshal(body); req.RequestCtx.Request.SetBody(jsonBody)

	err := app.UpdateSSOProvider(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string                       `json:"status"`
		Data   handlers.SSOProviderResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "updated-client-id", resp.Data.ClientID)
	assert.False(t, resp.Data.IsEnabled)

	// Verify in database - secret should remain unchanged
	var provider models.SSOProvider
	err = app.DB.First(&provider, existingProvider.ID).Error
	require.NoError(t, err)
	assert.Equal(t, "updated-client-id", provider.ClientID)
	assert.Equal(t, existingProvider.ClientSecret, provider.ClientSecret) // Unchanged
}

func TestApp_UpdateSSOProvider_InvalidProvider(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	req, _ := createSSOAdminRequest(t, app, org.ID)
	req.RequestCtx.SetUserValue("provider", "invalid-provider")

	body := map[string]any{
		"client_id":     "test-client-id",
		"client_secret": "test-client-secret",
	}
	jsonBody, _ := json.Marshal(body); req.RequestCtx.Request.SetBody(jsonBody)

	err := app.UpdateSSOProvider(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Invalid provider")
}

func TestApp_UpdateSSOProvider_CustomProviderMissingURLs(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	req, _ := createSSOAdminRequest(t, app, org.ID)
	req.RequestCtx.SetUserValue("provider", "custom")

	body := map[string]any{
		"client_id":     "test-client-id",
		"client_secret": "test-client-secret",
		// Missing auth_url, token_url, user_info_url
	}
	jsonBody, _ := json.Marshal(body); req.RequestCtx.Request.SetBody(jsonBody)

	err := app.UpdateSSOProvider(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Custom provider requires auth_url, token_url, and user_info_url")
}

func TestApp_UpdateSSOProvider_CustomProviderSuccess(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	req, _ := createSSOAdminRequest(t, app, org.ID)
	req.RequestCtx.SetUserValue("provider", "custom")

	body := map[string]any{
		"client_id":        "test-client-id",
		"client_secret":    "test-client-secret",
		"is_enabled":       true,
		"auth_url":         "https://custom.example.com/oauth/authorize",
		"token_url":        "https://custom.example.com/oauth/token",
		"user_info_url":    "https://custom.example.com/oauth/userinfo",
		"allow_auto_create": true,
		"default_role":     "agent",
	}
	jsonBody, _ := json.Marshal(body); req.RequestCtx.Request.SetBody(jsonBody)

	err := app.UpdateSSOProvider(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Status string                       `json:"status"`
		Data   handlers.SSOProviderResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, "custom", resp.Data.Provider)
	assert.Equal(t, "https://custom.example.com/oauth/authorize", resp.Data.AuthURL)
	assert.Equal(t, "https://custom.example.com/oauth/token", resp.Data.TokenURL)
	assert.Equal(t, "https://custom.example.com/oauth/userinfo", resp.Data.UserInfoURL)
}

func TestApp_UpdateSSOProvider_Unauthorized(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create user without SSO write permissions
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "user", []string{"settings:sso:read"})
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("provider", "google")

	body := map[string]any{
		"client_id":     "test-client-id",
		"client_secret": "test-client-secret",
	}
	jsonBody, _ := json.Marshal(body); req.RequestCtx.Request.SetBody(jsonBody)

	err := app.UpdateSSOProvider(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission_denied")
}

// Test DeleteSSOProvider
func TestApp_DeleteSSOProvider_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create provider
	provider := createTestSSOProvider(t, app.DB, org.ID, "google", true)

	req, _ := createSSOAdminRequest(t, app, org.ID)
	req.RequestCtx.SetUserValue("provider", "google")

	err := app.DeleteSSOProvider(req)
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
	assert.Equal(t, "SSO provider deleted", resp.Data.Message)

	// Verify deleted from database
	var deletedProvider models.SSOProvider
	err = app.DB.First(&deletedProvider, provider.ID).Error
	assert.Error(t, err)
}

func TestApp_DeleteSSOProvider_NotFound(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	req, _ := createSSOAdminRequest(t, app, org.ID)
	req.RequestCtx.SetUserValue("provider", "google")

	err := app.DeleteSSOProvider(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "SSO provider not found")
}

func TestApp_DeleteSSOProvider_Unauthorized(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create user without SSO write permissions
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "user", []string{"settings:sso:read"})
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("provider", "google")

	err := app.DeleteSSOProvider(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "permission_denied")
}

// Test CallbackSSO user auto-creation disabled
func TestApp_CallbackSSO_AutoCreateDisabled(t *testing.T) {
	mockServer := createMockOAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "new-user-id",
			"email": "newuser@example.com",
			"name":  "New User",
		})
	})
	defer mockServer.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create SSO provider with auto-create disabled
	ssoProvider := createTestSSOProvider(t, app.DB, org.ID, "custom", true)
	ssoProvider.AllowAutoCreate = false
	ssoProvider.AuthURL = mockServer.URL + "/oauth/authorize"
	ssoProvider.TokenURL = mockServer.URL + "/oauth/token"
	ssoProvider.UserInfoURL = mockServer.URL + "/oauth/userinfo"
	app.DB.Save(ssoProvider)

	// Set up state
	stateNonce := "test-state-nonce"
	state := handlers.SSOState{
		OrgID:     org.ID.String(),
		Provider:  "custom",
		Nonce:     stateNonce,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	stateJSON, _ := json.Marshal(state)
	stateKey := "sso:state:" + stateNonce

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, app.Redis.Set(ctx, stateKey, stateJSON, 5*time.Minute).Err())

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "custom")
	req.RequestCtx.QueryArgs().Add("code", "test-code")
	req.RequestCtx.QueryArgs().Add("state", stateNonce)

	err := app.CallbackSSO(req)
	require.NoError(t, err)

	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, statusCode)

	redirectURL := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, redirectURL, "User not found")
}

// Test CallbackSSO with invalid email from provider
func TestApp_CallbackSSO_InvalidEmail(t *testing.T) {
	mockServer := createMockOAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "test-user-id",
			"email": "invalid-email-format", // Invalid email
			"name":  "Test User",
		})
	})
	defer mockServer.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create SSO provider with domain restriction
	ssoProvider := createTestSSOProvider(t, app.DB, org.ID, "custom", true)
	ssoProvider.AllowedDomains = "example.com"
	ssoProvider.AuthURL = mockServer.URL + "/oauth/authorize"
	ssoProvider.TokenURL = mockServer.URL + "/oauth/token"
	ssoProvider.UserInfoURL = mockServer.URL + "/oauth/userinfo"
	app.DB.Save(ssoProvider)

	// Set up state
	stateNonce := "test-state-nonce"
	state := handlers.SSOState{
		OrgID:     org.ID.String(),
		Provider:  "custom",
		Nonce:     stateNonce,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	stateJSON, _ := json.Marshal(state)
	stateKey := "sso:state:" + stateNonce

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, app.Redis.Set(ctx, stateKey, stateJSON, 5*time.Minute).Err())

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "custom")
	req.RequestCtx.QueryArgs().Add("code", "test-code")
	req.RequestCtx.QueryArgs().Add("state", stateNonce)

	err := app.CallbackSSO(req)
	require.NoError(t, err)

	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, statusCode)

	redirectURL := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, redirectURL, "Invalid email from provider")
}

// Test CallbackSSO with missing email from provider
func TestApp_CallbackSSO_MissingEmail(t *testing.T) {
	mockServer := createMockOAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "test-user-id",
			"name": "Test User",
			// No email field
		})
	})
	defer mockServer.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create SSO provider
	ssoProvider := createTestSSOProvider(t, app.DB, org.ID, "custom", true)
	ssoProvider.AuthURL = mockServer.URL + "/oauth/authorize"
	ssoProvider.TokenURL = mockServer.URL + "/oauth/token"
	ssoProvider.UserInfoURL = mockServer.URL + "/oauth/userinfo"
	app.DB.Save(ssoProvider)

	// Set up state
	stateNonce := "test-state-nonce"
	state := handlers.SSOState{
		OrgID:     org.ID.String(),
		Provider:  "custom",
		Nonce:     stateNonce,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	stateJSON, _ := json.Marshal(state)
	stateKey := "sso:state:" + stateNonce

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, app.Redis.Set(ctx, stateKey, stateJSON, 5*time.Minute).Err())

	req := testutil.NewJSONRequest(t, nil)
	req.RequestCtx.SetUserValue("provider", "custom")
	req.RequestCtx.QueryArgs().Add("code", "test-code")
	req.RequestCtx.QueryArgs().Add("state", stateNonce)

	err := app.CallbackSSO(req)
	require.NoError(t, err)

	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, statusCode)

	redirectURL := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, redirectURL, "Failed to get user information")
}

// Helper function to extract state from OAuth URL
func extractStateFromAuthURL(t *testing.T, authURL string) string {
	t.Helper()

	parsedURL, err := url.Parse(authURL)
	require.NoError(t, err)

	queryParams := parsedURL.Query()
	state := queryParams.Get("state")
	require.NotEmpty(t, state, "State parameter not found in auth URL")

	return state
}
