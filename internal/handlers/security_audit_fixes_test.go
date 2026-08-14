package handlers_test

import (
	"context"
	"encoding/json"
	"strings"
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
	"golang.org/x/crypto/bcrypt"
)

// --- JWT token-type confusion: RefreshToken must reject access tokens ---

// generateTypedToken creates a JWT with the given token_type / JTI for
// confusion tests (mirrors generateRefreshTokenWithJTI in auth_gaps_test.go).
func generateTypedToken(t *testing.T, secret string, user *models.User, tokenType, jti string, expiry time.Duration) string {
	t.Helper()
	claims := middleware.JWTClaims{
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		Email:          user.Email,
		RoleID:         user.RoleID,
		TokenType:      tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "gowa-ui",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

func TestApp_RefreshToken_RejectsAccessToken(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	tests := []struct {
		name  string
		token func() string
	}{
		{
			name: "access token with token_type claim",
			token: func() string {
				return generateTypedToken(t, testutil.TestJWTSecret, user, middleware.TokenTypeAccess, "", time.Hour)
			},
		},
		{
			name: "legacy access token without JTI or type",
			token: func() string {
				return generateTypedToken(t, testutil.TestJWTSecret, user, "", "", time.Hour)
			},
		},
		{
			name: "ws token",
			token: func() string {
				return generateTypedToken(t, testutil.TestJWTSecret, user, middleware.TokenTypeWS, "", time.Hour)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.NewJSONRequest(t, map[string]string{"refresh_token": tt.token()})
			require.NoError(t, app.RefreshToken(req))
			testutil.AssertErrorResponse(t, req, fasthttp.StatusUnauthorized, "Invalid refresh token")
		})
	}
}

func TestApp_RefreshToken_AcceptsNewStyleRefreshToken(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	jti := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, app.Redis.Set(ctx, "refresh:"+jti, user.ID.String(), time.Hour).Err())

	token := generateTypedToken(t, testutil.TestJWTSecret, user, middleware.TokenTypeRefresh, jti, time.Hour)
	req := testutil.NewJSONRequest(t, map[string]string{"refresh_token": token})
	require.NoError(t, app.RefreshToken(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"typed refresh token with live JTI must rotate")
}

// --- SSO cross-org takeover ---

// callSSOCallback drives CallbackSSO through a full fake-IdP round trip for a
// custom provider owned by ssoOrgID, asserting the IdP identity (email/sub).
// It returns the request so the caller can inspect status/cookies.
func callSSOCallback(t *testing.T, app *handlers.App, ssoOrgID uuid.UUID, email, sub string) *fastglue.Request {
	t.Helper()
	fake := newFakeOAuth(t)
	fake.UserEmail = email
	fake.UserID = sub
	createCustomSSOProvider(t, app, ssoOrgID, fake, func(p *models.SSOProvider) {
		p.AllowAutoCreate = false
	})

	nonce := "sec-nonce-" + uuid.New().String()
	stateJSON, err := json.Marshal(handlers.SSOState{
		OrgID:     ssoOrgID.String(),
		Provider:  "custom",
		Nonce:     nonce,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, app.Redis.Set(ctx, "sso:state:"+nonce, stateJSON, 5*time.Minute).Err())

	req := testutil.NewGETRequest(t)
	testutil.SetPathParam(req, "provider", "custom")
	testutil.SetQueryParam(req, "code", "fake-code")
	testutil.SetQueryParam(req, "state", nonce)
	require.NoError(t, app.CallbackSSO(req))
	return req
}

func TestApp_CallbackSSO_CustomIdPCannotTakeOverCrossOrgAccount(t *testing.T) {
	app := newSSOApp(t)
	attackerOrg := testutil.CreateTestOrganization(t, app.DB)
	victimOrg := testutil.CreateTestOrganization(t, app.DB)

	// Victim is a password-only user of victimOrg with no link to attackerOrg.
	victim := testutil.CreateTestUser(t, app.DB, victimOrg.ID)

	req := callSSOCallback(t, app, attackerOrg.ID, victim.Email, "attacker-sub")

	// Must NOT mint a session: redirect to /login with an error and no
	// access cookie.
	assert.Equal(t, fasthttp.StatusTemporaryRedirect, testutil.GetResponseStatusCode(req))
	location := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, location, "/login?sso_error=", "expected rejection redirect")
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_access"), "no session may be minted")

	// Account must remain unlinked.
	var dbUser models.User
	require.NoError(t, app.DB.Where("id = ?", victim.ID).First(&dbUser).Error)
	assert.Empty(t, dbUser.SSOProvider, "password-only account must not be auto-linked via custom IdP")
}

func TestApp_CallbackSSO_CustomIdPCannotTakeOverSameOrgPasswordAccount(t *testing.T) {
	app := newSSOApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Victim belongs to the SSO org itself but has never linked SSO — a
	// malicious org admin's custom IdP still must not take the account over.
	victim := testutil.CreateTestUser(t, app.DB, org.ID)

	req := callSSOCallback(t, app, org.ID, victim.Email, "attacker-sub")

	assert.Equal(t, fasthttp.StatusTemporaryRedirect, testutil.GetResponseStatusCode(req))
	assert.Contains(t, string(req.RequestCtx.Response.Header.Peek("Location")), "/login?sso_error=")
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_access"))

	var dbUser models.User
	require.NoError(t, app.DB.Where("id = ?", victim.ID).First(&dbUser).Error)
	assert.Empty(t, dbUser.SSOProvider)
}

func TestApp_CallbackSSO_ProviderMismatchRejected(t *testing.T) {
	app := newSSOApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	user := testutil.CreateTestUser(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(user).Updates(map[string]any{
		"sso_provider":    "google",
		"sso_provider_id": "google-sub-1",
	}).Error)

	req := callSSOCallback(t, app, org.ID, user.Email, "custom-sub-1")

	assert.Equal(t, fasthttp.StatusTemporaryRedirect, testutil.GetResponseStatusCode(req))
	assert.Contains(t, string(req.RequestCtx.Response.Header.Peek("Location")), "/login?sso_error=")
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_access"))
}

func TestApp_CallbackSSO_SubjectMismatchRejected(t *testing.T) {
	app := newSSOApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	user := testutil.CreateTestUser(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(user).Updates(map[string]any{
		"sso_provider":    "custom",
		"sso_provider_id": "legit-sub",
	}).Error)

	// Same provider, different subject at the IdP → must be rejected.
	req := callSSOCallback(t, app, org.ID, user.Email, "impostor-sub")

	assert.Equal(t, fasthttp.StatusTemporaryRedirect, testutil.GetResponseStatusCode(req))
	assert.Contains(t, string(req.RequestCtx.Response.Header.Peek("Location")), "/login?sso_error=")
	assert.Empty(t, testutil.GetResponseCookie(req, "whm_access"))
}

func TestApp_CallbackSSO_LinkedMemberStillSucceeds(t *testing.T) {
	app := newSSOApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	user := testutil.CreateTestUser(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(user).Updates(map[string]any{
		"sso_provider":    "custom",
		"sso_provider_id": "legit-sub",
	}).Error)

	req := callSSOCallback(t, app, org.ID, user.Email, "legit-sub")

	assert.Equal(t, fasthttp.StatusTemporaryRedirect, testutil.GetResponseStatusCode(req))
	location := string(req.RequestCtx.Response.Header.Peek("Location"))
	assert.Contains(t, location, "/auth/sso/callback")
	assert.NotContains(t, location, "sso_error")
	assert.NotEmpty(t, testutil.GetResponseCookie(req, "whm_access"), "legit SSO login must mint a session")
}

// --- Contact write endpoints enforce account-assignment scoping ---

func TestApp_ContactWriteEndpoints_EnforceAccountScoping(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "manager", []string{
		"contacts:read", "contacts:write", "contacts:delete",
	})
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	accA := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("scoped-acc-a"))
	accB := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("scoped-acc-b"))
	testutil.AssignAccountToUser(t, app.DB, user.ID, accA.ID)

	contactA := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accA.Name))
	contactB := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accB.Name))

	updateContact := func(contactID uuid.UUID) int {
		req := testutil.NewJSONRequest(t, map[string]any{"profile_name": "Renamed"})
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contactID.String())
		require.NoError(t, app.UpdateContact(req))
		return testutil.GetResponseStatusCode(req)
	}

	t.Run("assigned account contact is writable", func(t *testing.T) {
		assert.Equal(t, fasthttp.StatusOK, updateContact(contactA.ID))
	})

	t.Run("other account contact is 404", func(t *testing.T) {
		assert.Equal(t, fasthttp.StatusNotFound, updateContact(contactB.ID),
			"scoped user must not modify contacts outside assigned accounts")
	})

	t.Run("delete other account contact is 404", func(t *testing.T) {
		req := testutil.NewRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", contactB.ID.String())
		require.NoError(t, app.DeleteContact(req))
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
	})

	t.Run("unassigned user keeps full org visibility", func(t *testing.T) {
		unscoped := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
		req := testutil.NewJSONRequest(t, map[string]any{"profile_name": "Renamed 2"})
		testutil.SetAuthContext(req, org.ID, unscoped.ID)
		testutil.SetPathParam(req, "id", contactB.ID.String())
		require.NoError(t, app.UpdateContact(req))
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	})
}

func TestApp_ConversationNotes_EnforceAccountScoping(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "agent", []string{
		"chat:read", "chat:write", "contacts:read",
	})
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	accA := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("notes-acc-a"))
	accB := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("notes-acc-b"))
	testutil.AssignAccountToUser(t, app.DB, user.ID, accA.ID)

	contactB := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(accB.Name))

	// Note CRUD on a contact outside the assigned accounts must 404.
	req := testutil.NewJSONRequest(t, map[string]string{"content": "should not be created"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contactB.ID.String())
	require.NoError(t, app.CreateConversationNote(req))
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))

	listReq := testutil.NewRequest(t)
	testutil.SetAuthContext(listReq, org.ID, user.ID)
	testutil.SetPathParam(listReq, "id", contactB.ID.String())
	require.NoError(t, app.ListConversationNotes(listReq))
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(listReq))
}

// --- Deactivation / deletion revoke sessions and API keys ---

func TestApp_UpdateUser_DeactivateRevokesAPIKeysAndTokens(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	target := testutil.CreateTestUser(t, app.DB, org.ID)

	// Seed an active API key for the target.
	hash, err := bcrypt.GenerateFromPassword([]byte("whm_"+strings.Repeat("a", 32)), bcrypt.MinCost)
	require.NoError(t, err)
	apiKey := models.APIKey{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		UserID:         target.ID,
		Name:           "target key",
		KeyPrefix:      strings.Repeat("a", 16),
		KeyHash:        string(hash),
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(&apiKey).Error)

	req := testutil.NewJSONRequest(t, map[string]any{"is_active": false})
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", target.ID.String())
	require.NoError(t, app.UpdateUser(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var dbKey models.APIKey
	require.NoError(t, app.DB.Where("id = ?", apiKey.ID).First(&dbKey).Error)
	assert.False(t, dbKey.IsActive, "deactivation must deactivate the user's API keys")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	version, err := app.Redis.Get(ctx, middleware.TokenVersionKey(target.ID)).Int()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, version, 1, "deactivation must bump the token-revocation version")
}

func TestApp_DeleteUser_RevokesAPIKeysAndTokens(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	target := testutil.CreateTestUser(t, app.DB, org.ID)

	hash, err := bcrypt.GenerateFromPassword([]byte("whm_"+strings.Repeat("b", 32)), bcrypt.MinCost)
	require.NoError(t, err)
	apiKey := models.APIKey{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		UserID:         target.ID,
		Name:           "target key",
		KeyPrefix:      strings.Repeat("b", 16),
		KeyHash:        string(hash),
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(&apiKey).Error)

	req := testutil.NewRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", target.ID.String())
	require.NoError(t, app.DeleteUser(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var dbKey models.APIKey
	require.NoError(t, app.DB.Where("id = ?", apiKey.ID).First(&dbKey).Error)
	assert.False(t, dbKey.IsActive, "deletion must deactivate the user's API keys")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	version, err := app.Redis.Get(ctx, middleware.TokenVersionKey(target.ID)).Int()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, version, 1, "deletion must bump the token-revocation version")
}

// --- Self password change via UpdateUser requires the current password ---

func TestApp_UpdateUser_SelfPasswordChangeRequiresCurrentPassword(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithPassword("original-password-123"))

	newReq := func(body map[string]any) *fastglue.Request {
		req := testutil.NewJSONRequest(t, body)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", user.ID.String())
		return req
	}

	t.Run("missing current password is rejected", func(t *testing.T) {
		req := newReq(map[string]any{"password": "brand-new-password-456"})
		require.NoError(t, app.UpdateUser(req))
		testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "current_password is required to change your own password")
	})

	t.Run("wrong current password is rejected", func(t *testing.T) {
		req := newReq(map[string]any{"password": "brand-new-password-456", "current_password": "wrong-password-999"})
		require.NoError(t, app.UpdateUser(req))
		testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Current password is incorrect")
	})

	t.Run("correct current password changes it", func(t *testing.T) {
		req := newReq(map[string]any{"password": "brand-new-password-456", "current_password": "original-password-123"})
		require.NoError(t, app.UpdateUser(req))
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var dbUser models.User
		require.NoError(t, app.DB.Where("id = ?", user.ID).First(&dbUser).Error)
		require.NoError(t, bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte("brand-new-password-456")))
	})

	t.Run("short new password is rejected", func(t *testing.T) {
		// Subtest 3 above changed the password — verify against the new one.
		req := newReq(map[string]any{"password": "short", "current_password": "brand-new-password-456"})
		require.NoError(t, app.UpdateUser(req))
		testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Password must be at least 12 characters")
	})
}
