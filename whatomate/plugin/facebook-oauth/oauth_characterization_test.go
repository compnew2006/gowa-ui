package facebookoauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/crypto"
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

type oauthTestApp struct {
	*Plugin
	DB     *gorm.DB
	Config *config.Config
}

func newOAuthTestApp(t *testing.T, client *http.Client) *oauthTestApp {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		App: config.AppConfig{
			EncryptionKey: testutil.TestEncryptionKey,
			Environment:   "test",
		},
	}
	app := &handlers.App{
		Config:     cfg,
		DB:         db,
		Log:        testutil.NopLogger(),
		HTTPClient: client,
	}
	plugin := &Plugin{}
	require.NoError(t, plugin.Init(app, db, nil, nil))
	return &oauthTestApp{Plugin: plugin, DB: db, Config: cfg}
}

func createAccountsAuthorizedUser(t *testing.T, app *oauthTestApp, organizationID uuid.UUID) *models.User {
	t.Helper()
	role := testutil.CreateTestRoleWithKeys(
		t,
		app.DB,
		organizationID,
		"facebook-oauth-accounts",
		[]string{"accounts:read", "accounts:write", "accounts:delete"},
	)
	return testutil.CreateTestUser(t, app.DB, organizationID, testutil.WithRoleID(&role.ID))
}

func TestApp_InitFacebookOAuth_AddsRerequestAuthType(t *testing.T) {
	app := newOAuthTestApp(t, nil)
	app.Config.FacebookOAuth = config.FacebookOAuthConfig{
		AppID:       "test-app-id",
		AppSecret:   "test-app-secret",
		APIVersion:  "v20.0",
		RedirectURI: "https://example.com/api/facebook/oauth/callback",
	}
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.InitFacebookOAuth(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			AuthURL string `json:"auth_url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	authURL, err := url.Parse(resp.Data.AuthURL)
	require.NoError(t, err)
	query := authURL.Query()

	assert.Equal(t, "rerequest", query.Get("auth_type"))
	assert.Equal(t, "code", query.Get("response_type"))
	assert.Equal(t, "test-app-id", query.Get("client_id"))
	assert.NotEmpty(t, query.Get("state"))
	assert.Contains(t, strings.Split(query.Get("scope"), ","), "pages_show_list")
}

func TestApp_CallbackFacebookOAuth_TokenExchangeErrorDoesNotSaveAccount(t *testing.T) {
	app, server := newFacebookOAuthCallbackTestApp(t, facebookOAuthGraphScenario{
		CodeExchangeStatus: http.StatusOK,
		CodeExchangeBody: map[string]any{
			"error": map[string]any{
				"message": "Invalid verification code",
				"type":    "OAuthException",
				"code":    100,
			},
		},
	})
	defer server.Close()

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	stateToken := storeFacebookOAuthState(t, app, org.ID, user.ID, "token-error-state")
	req := newFacebookOAuthCallbackRequest(t, stateToken)

	err := app.CallbackFacebookOAuth(req)
	require.NoError(t, err)

	assertFacebookOAuthRedirect(t, req, "error", "")
	assertFacebookAccountCount(t, app, 0)
}

func TestApp_CallbackFacebookOAuth_RejectsPageTokenDebugType(t *testing.T) {
	app, server := newFacebookOAuthCallbackTestApp(t, facebookOAuthGraphScenario{
		DebugTokenType: "PAGE",
	})
	defer server.Close()

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	stateToken := storeFacebookOAuthState(t, app, org.ID, user.ID, "page-token-state")
	req := newFacebookOAuthCallbackRequest(t, stateToken)

	err := app.CallbackFacebookOAuth(req)
	require.NoError(t, err)

	assertFacebookOAuthRedirect(t, req, "error", "Facebook returned a PAGE token")
	assertFacebookAccountCount(t, app, 0)
}

func TestApp_CallbackFacebookOAuth_SavesVerifiedUserTokenAndPageTokens(t *testing.T) {
	app, server := newFacebookOAuthCallbackTestApp(t, facebookOAuthGraphScenario{
		DebugTokenType: "USER",
	})
	defer server.Close()

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	stateToken := storeFacebookOAuthState(t, app, org.ID, user.ID, "user-token-state")
	req := newFacebookOAuthCallbackRequest(t, stateToken)

	err := app.CallbackFacebookOAuth(req)
	require.NoError(t, err)

	assertFacebookOAuthRedirect(t, req, "connected", "")

	var account models.FacebookAccount
	require.NoError(t, app.DB.First(&account, "organization_id = ? AND account_uid = ?", org.ID, "user-123").Error)
	decryptedUserToken, err := crypto.Decrypt(account.AccessToken, app.Config.App.EncryptionKey)
	require.NoError(t, err)
	assert.Equal(t, "long-user-token", decryptedUserToken)

	decryptedPageTokens, err := crypto.Decrypt(account.PageTokens, app.Config.App.EncryptionKey)
	require.NoError(t, err)
	var pageTokens map[string]string
	require.NoError(t, json.Unmarshal([]byte(decryptedPageTokens), &pageTokens))
	assert.Equal(t, map[string]string{"page-1": "page-token-1"}, pageTokens)
	assert.Equal(t, float64(1), account.Data["page_count"])
}

func TestApp_CallbackFacebookOAuth_PageFetchFailureDoesNotSaveAccount(t *testing.T) {
	app, server := newFacebookOAuthCallbackTestApp(t, facebookOAuthGraphScenario{
		DebugTokenType:       "USER",
		AccountsStatus:       http.StatusBadRequest,
		AccountsErrorMessage: "Tried accessing nonexisting field",
	})
	defer server.Close()

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAccountsAuthorizedUser(t, app, org.ID)
	stateToken := storeFacebookOAuthState(t, app, org.ID, user.ID, "page-fetch-failure-state")
	req := newFacebookOAuthCallbackRequest(t, stateToken)

	err := app.CallbackFacebookOAuth(req)
	require.NoError(t, err)

	assertFacebookOAuthRedirect(t, req, "error", "failed to fetch managed Facebook pages")
	assertFacebookAccountCount(t, app, 0)
}

type facebookOAuthGraphScenario struct {
	CodeExchangeStatus   int
	CodeExchangeBody     map[string]any
	LongExchangeStatus   int
	LongExchangeBody     map[string]any
	DebugTokenType       string
	DebugTokenStatus     int
	AccountsStatus       int
	AccountsErrorMessage string
}

func newFacebookOAuthCallbackTestApp(t *testing.T, scenario facebookOAuthGraphScenario) (*oauthTestApp, *httptest.Server) {
	t.Helper()
	_ = testutil.SetupTestDB(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v20.0/oauth/access_token" && r.URL.Query().Get("code") != "":
			writeFacebookOAuthJSON(t, w, defaultStatus(scenario.CodeExchangeStatus), defaultMap(scenario.CodeExchangeBody, map[string]any{
				"access_token": "short-user-token",
				"token_type":   "bearer",
				"expires_in":   3600,
			}))
		case r.URL.Path == "/v20.0/oauth/access_token" && r.URL.Query().Get("grant_type") == "fb_exchange_token":
			writeFacebookOAuthJSON(t, w, defaultStatus(scenario.LongExchangeStatus), defaultMap(scenario.LongExchangeBody, map[string]any{
				"access_token": "long-user-token",
				"token_type":   "bearer",
				"expires_in":   5184000,
			}))
		case r.URL.Path == "/v20.0/debug_token":
			writeFacebookOAuthJSON(t, w, defaultStatus(scenario.DebugTokenStatus), map[string]any{
				"data": map[string]any{
					"app_id":     "test-app-id",
					"type":       defaultString(scenario.DebugTokenType, "USER"),
					"is_valid":   true,
					"expires_at": 1893456000,
					"scopes":     []string{"public_profile", "pages_show_list"},
				},
			})
		case r.URL.Path == "/v20.0/me":
			writeFacebookOAuthJSON(t, w, http.StatusOK, map[string]any{
				"id":    "user-123",
				"name":  "OAuth User",
				"email": "oauth-user@example.com",
				"picture": map[string]any{"data": map[string]any{
					"url": "https://example.com/avatar.jpg",
				}},
			})
		case r.URL.Path == "/v20.0/me/accounts":
			if scenario.AccountsStatus >= 400 {
				writeFacebookOAuthJSON(t, w, scenario.AccountsStatus, map[string]any{
					"error": map[string]any{
						"message": scenario.AccountsErrorMessage,
						"type":    "OAuthException",
						"code":    100,
					},
				})
				return
			}
			writeFacebookOAuthJSON(t, w, http.StatusOK, map[string]any{
				"data": []map[string]any{{
					"id":           "page-1",
					"name":         "Managed Page",
					"access_token": "page-token-1",
					"category":     "Business",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))

	app := newOAuthTestApp(t, server.Client())
	app.Config.FacebookOAuth = config.FacebookOAuthConfig{
		AppID:       "test-app-id",
		AppSecret:   "test-app-secret",
		APIVersion:  "v20.0",
		RedirectURI: "https://example.com/api/facebook/oauth/callback",
		BaseURL:     server.URL,
	}
	return app, server
}

func storeFacebookOAuthState(t *testing.T, app *oauthTestApp, orgID, userID uuid.UUID, stateToken string) string {
	t.Helper()
	state := OAuthState{
		OrganizationID: orgID,
		UserID:         userID,
		StateToken:     stateToken,
		Action:         "connect",
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	}
	require.NoError(t, app.DB.Create(&state).Error)
	return stateToken
}

func newFacebookOAuthCallbackRequest(t *testing.T, stateToken string) *fastglue.Request {
	t.Helper()
	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("code", "test-auth-code")
	req.RequestCtx.QueryArgs().Add("state", stateToken)
	return req
}

func assertFacebookOAuthRedirect(t *testing.T, req *fastglue.Request, expectedStatus, expectedMessagePart string) {
	t.Helper()
	assert.Equal(t, fasthttp.StatusFound, testutil.GetResponseStatusCode(req))
	location := string(req.RequestCtx.Response.Header.Peek("Location"))
	redirectURL, err := url.Parse(location)
	require.NoError(t, err)
	query := redirectURL.Query()
	assert.Equal(t, expectedStatus, query.Get("facebook_oauth"))
	if expectedMessagePart != "" {
		assert.Contains(t, query.Get("message"), expectedMessagePart)
	}
}

func assertFacebookAccountCount(t *testing.T, app *oauthTestApp, expected int64) {
	t.Helper()
	var count int64
	require.NoError(t, app.DB.Model(&models.FacebookAccount{}).Count(&count).Error)
	assert.Equal(t, expected, count)
}

func createOAuthFacebookAccount(t *testing.T, app *oauthTestApp, orgID, userID uuid.UUID, pages []map[string]any, pageTokens map[string]string) models.FacebookAccount {
	t.Helper()
	accessToken, err := crypto.Encrypt("long-user-token", app.Config.App.EncryptionKey)
	require.NoError(t, err)

	pageTokensJSON, err := json.Marshal(pageTokens)
	require.NoError(t, err)
	encryptedPageTokens, err := crypto.Encrypt(string(pageTokensJSON), app.Config.App.EncryptionKey)
	require.NoError(t, err)

	account := models.FacebookAccount{
		OrganizationID: orgID,
		UserID:         userID,
		Platform:       "facebook",
		Name:           "OAuth User",
		AccountUID:     "user-123",
		Status:         models.FBAccountStatusActive,
		Method:         models.FBAccountMethodOAuth,
		AccessToken:    accessToken,
		PageTokens:     encryptedPageTokens,
		Data: models.JSONB{
			"pages":      pages,
			"page_count": len(pages),
		},
	}
	require.NoError(t, app.DB.Create(&account).Error)
	return account
}

func reloadFacebookOAuthAccount(t *testing.T, app *oauthTestApp, id uuid.UUID) models.FacebookAccount {
	t.Helper()
	var account models.FacebookAccount
	require.NoError(t, app.DB.First(&account, "id = ?", id).Error)
	return account
}

func facebookAccountPagesFromData(t *testing.T, account models.FacebookAccount) []map[string]any {
	t.Helper()
	rawPages, ok := account.Data["pages"]
	require.True(t, ok)

	encoded, err := json.Marshal(rawPages)
	require.NoError(t, err)
	var pages []map[string]any
	require.NoError(t, json.Unmarshal(encoded, &pages))
	return pages
}

func decryptFacebookPageTokens(t *testing.T, app *oauthTestApp, encrypted string) map[string]string {
	t.Helper()
	decrypted, err := crypto.Decrypt(encrypted, app.Config.App.EncryptionKey)
	require.NoError(t, err)
	if decrypted == "" {
		return map[string]string{}
	}
	var pageTokens map[string]string
	require.NoError(t, json.Unmarshal([]byte(decrypted), &pageTokens))
	if pageTokens == nil {
		return map[string]string{}
	}
	return pageTokens
}

func newFacebookPageManagementRequest(t *testing.T, orgID, userID, accountID uuid.UUID, pageID string) *fastglue.Request {
	t.Helper()
	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, orgID, userID)
	testutil.SetPathParam(req, "id", accountID.String())
	if pageID != "" {
		testutil.SetPathParam(req, "page_id", pageID)
	}
	return req
}

func writeFacebookOAuthJSON(t *testing.T, w http.ResponseWriter, status int, payload map[string]any) {
	t.Helper()
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}

func defaultStatus(status int) int {
	if status == 0 {
		return http.StatusOK
	}
	return status
}

func defaultMap(value, fallback map[string]any) map[string]any {
	if value == nil {
		return fallback
	}
	return value
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
