package handlers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	appcrypto "github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"golang.org/x/oauth2"
)

// GetPublicSSOProviders returns enabled SSO providers for login page (public, no auth)
func (a *App) GetPublicSSOProviders(r *fastglue.Request) error {
	requestDB :=
		// Get all enabled SSO providers.
		a.requestDB(r)

	var providers []models.SSOProvider
	if err := requestDB.Preload("Organization").Where("is_enabled = ?", true).Find(&providers).Error; err != nil {
		a.Log.Error("Failed to fetch SSO providers", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch providers", nil, "")
	}

	providerCounts := make(map[string]int, len(providers))
	for _, provider := range providers {
		providerCounts[provider.Provider]++
	}

	result := make([]SSOProviderPublic, 0)
	for _, p := range providers {
		name := providerDisplayNames[p.Provider]
		if name == "" {
			name = p.Provider
		}
		orgSlug := ""
		if p.Organization != nil {
			orgSlug = strings.TrimSpace(p.Organization.Slug)
		}
		if providerCounts[p.Provider] > 1 && p.Organization != nil && strings.TrimSpace(p.Organization.Name) != "" {
			name = fmt.Sprintf("%s (%s)", name, strings.TrimSpace(p.Organization.Name))
		}
		result = append(result, SSOProviderPublic{
			Provider:         p.Provider,
			Name:             name,
			OrganizationSlug: orgSlug,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Provider == result[j].Provider {
			return result[i].Name < result[j].Name
		}
		return result[i].Provider < result[j].Provider
	})

	return r.SendEnvelope(result)
}

// InitSSO initiates OAuth flow for a provider
func (a *App) InitSSO(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	provider := r.RequestCtx.UserValue("provider").(string)
	orgSlug := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("org")))

	// Validate provider
	if provider != "custom" {
		if _, ok := oauthProviders[provider]; !ok {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid SSO provider", nil, "")
		}
	}

	var ssoConfig models.SSOProvider
	if orgSlug != "" {
		var org models.Organization
		if err := requestDB.Select("id").Where("slug = ?", orgSlug).First(&org).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "SSO provider not configured or disabled", nil, "")
		}
		if err := requestDB.Where("provider = ? AND is_enabled = ? AND organization_id = ?", provider, true, org.ID).First(&ssoConfig).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "SSO provider not configured or disabled", nil, "")
		}
	} else {
		var configs []models.SSOProvider
		if err := requestDB.Where("provider = ? AND is_enabled = ?", provider, true).Limit(2).Find(&configs).Error; err != nil {
			a.Log.Error("Failed to fetch SSO provider", "error", err, "provider", provider)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initiate SSO", nil, "")
		}
		switch len(configs) {
		case 0:
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "SSO provider not configured or disabled", nil, "")
		case 1:
			ssoConfig = configs[0]
		default:
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Organization selection is required for this SSO provider", nil, "org")
		}
	}

	// Generate state token
	nonce, err := generateRandomString(32)
	if err != nil {
		a.Log.Error("Failed to generate SSO nonce", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initiate SSO", nil, "")
	}

	browserToken, err := generateRandomString(32)
	if err != nil {
		a.Log.Error("Failed to generate SSO browser token", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initiate SSO", nil, "")
	}

	pkceVerifier, err := generatePKCEVerifier()
	if err != nil {
		a.Log.Error("Failed to generate SSO PKCE verifier", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initiate SSO", nil, "")
	}

	// Build OAuth config before persisting state so invalid custom provider URLs or secrets
	// do not leave behind a stale Redis entry.
	oauthConfig := a.buildOAuthConfig(provider, &ssoConfig, r)
	if oauthConfig == nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initiate SSO", nil, "")
	}

	state := SSOState{
		OrgID:        ssoConfig.OrganizationID.String(),
		Provider:     provider,
		Nonce:        nonce,
		BrowserToken: browserToken,
		PKCEVerifier: pkceVerifier,
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}

	stateJSON, _ := json.Marshal(state)
	stateKey := "sso:state:" + nonce

	// Store state in Redis (5 min TTL)
	if err := a.Redis.Set(r.RequestCtx, stateKey, stateJSON, 5*time.Minute).Err(); err != nil {
		a.Log.Error("Failed to store SSO state", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initiate SSO", nil, "")
	}

	a.setSSOStateCookie(r, browserToken)

	// Redirect to provider
	authURL := oauthConfig.AuthCodeURL(
		nonce,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge(pkceVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	r.RequestCtx.Redirect(authURL, fasthttp.StatusTemporaryRedirect)
	return nil
}

// CallbackSSO handles OAuth callback
func (a *App) CallbackSSO(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	provider := r.RequestCtx.UserValue("provider").(string)
	code := string(r.RequestCtx.QueryArgs().Peek("code"))
	stateNonce := string(r.RequestCtx.QueryArgs().Peek("state"))
	errorParam := string(r.RequestCtx.QueryArgs().Peek("error"))

	// Check for OAuth error
	if errorParam != "" {
		errorDesc := string(r.RequestCtx.QueryArgs().Peek("error_description"))
		a.redirectWithError(r, "SSO failed: "+errorDesc)
		return nil
	}

	if code == "" || stateNonce == "" {
		a.redirectWithError(r, "Invalid callback parameters")
		return nil
	}
	defer a.clearSSOStateCookie(r)

	// Retrieve and validate state from Redis
	stateKey := "sso:state:" + stateNonce
	stateJSON, err := a.Redis.Get(r.RequestCtx, stateKey).Bytes()
	if err != nil {
		a.redirectWithError(r, "Invalid or expired state")
		return nil
	}

	// Delete state immediately to prevent replay
	a.Redis.Del(r.RequestCtx, stateKey)

	var state SSOState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		a.redirectWithError(r, "Invalid state")
		return nil
	}

	// Validate state
	if state.Provider != provider || state.BrowserToken == "" || state.PKCEVerifier == "" || time.Now().After(state.ExpiresAt) {
		a.redirectWithError(r, "Invalid or expired state")
		return nil
	}
	if a.ssoStateCookieValue(r) != state.BrowserToken {
		a.redirectWithError(r, "Invalid or expired state")
		return nil
	}

	// Parse org ID from state
	orgID, err := uuid.Parse(state.OrgID)
	if err != nil {
		a.redirectWithError(r, "Invalid organization")
		return nil
	}

	// Get SSO provider config
	var ssoConfig models.SSOProvider
	if err := requestDB.Where("organization_id = ? AND provider = ? AND is_enabled = ?", orgID, provider, true).First(&ssoConfig).Error; err != nil {
		a.redirectWithError(r, "SSO provider not configured")
		return nil
	}

	// Build OAuth config and exchange code for token
	oauthConfig := a.buildOAuthConfig(provider, &ssoConfig, r)
	if oauthConfig == nil {
		a.redirectWithError(r, "SSO provider is misconfigured")
		return nil
	}
	token, err := oauthConfig.Exchange(a.oauthContext(), code, oauth2.SetAuthURLParam("code_verifier", state.PKCEVerifier))
	if err != nil {
		a.Log.Error("Failed to exchange OAuth code", "error", err, "provider", provider)
		a.redirectWithError(r, "Failed to authenticate with provider")
		return nil
	}

	// Fetch user info from provider
	userInfo, err := a.fetchUserInfo(provider, &ssoConfig, token)
	if err != nil {
		a.Log.Error("Failed to fetch user info", "error", err, "provider", provider)
		a.redirectWithError(r, "Failed to get user information")
		return nil
	}
	userInfo.ID = strings.TrimSpace(userInfo.ID)
	userInfo.Email = strings.ToLower(strings.TrimSpace(userInfo.Email))
	userInfo.Name = strings.TrimSpace(userInfo.Name)
	if userInfo.ID == "" {
		a.redirectWithError(r, "Invalid account identifier from provider")
		return nil
	}

	// Validate email domain if configured
	if ssoConfig.AllowedDomains != "" {
		domains := strings.Split(ssoConfig.AllowedDomains, ",")
		emailParts := strings.Split(userInfo.Email, "@")
		if len(emailParts) != 2 {
			a.redirectWithError(r, "Invalid email from provider")
			return nil
		}
		emailDomain := strings.ToLower(strings.TrimSpace(emailParts[1]))
		allowed := false
		for _, d := range domains {
			if strings.ToLower(strings.TrimSpace(d)) == emailDomain {
				allowed = true
				break
			}
		}
		if !allowed {
			a.redirectWithError(r, "Email domain not allowed for this organization")
			return nil
		}
	}

	// Find user by email (across all orgs, like regular login)
	var user models.User
	if err := requestDB.Where("email = ?", userInfo.Email).First(&user).Error; err != nil {
		// User doesn't exist - check if auto-create is enabled
		if !ssoConfig.AllowAutoCreate {
			a.redirectWithError(r, "User not found. Contact your administrator.")
			return nil
		}

		// Auto-create user in the SSO config's organization
		roleName := ssoConfig.DefaultRoleName
		if roleName == "" {
			roleName = "agent"
		}

		// Look up the CustomRole by name for this organization
		var customRole models.CustomRole
		if err := requestDB.Where("organization_id = ? AND name = ?", orgID, roleName).First(&customRole).Error; err != nil {
			a.Log.Error("Failed to find role for SSO user", "error", err, "role_name", roleName)
			a.redirectWithError(r, "Failed to create user account: role not found")
			return nil
		}

		user = models.User{
			OrganizationID: orgID,
			Email:          userInfo.Email,
			FullName:       userInfo.Name,
			RoleID:         &customRole.ID,
			IsActive:       true,
			IsAvailable:    true,
			SSOProvider:    provider,
			SSOProviderID:  userInfo.ID,
		}
		if !a.checkQuotaOrRespond(r, license.ResourceUsers, orgID) {
			a.redirectWithError(r, "Licensed user quota exceeded for this organization")
			return nil
		}

		if err := requestDB.Create(&user).Error; err != nil {
			a.Log.Error("Failed to create SSO user", "error", err, "email", userInfo.Email)
			a.redirectWithError(r, "Failed to create user account")
			return nil
		}

		// Create UserOrganization entry
		userOrg := models.UserOrganization{
			UserID:         user.ID,
			OrganizationID: orgID,
			RoleID:         &customRole.ID,
			IsDefault:      true,
		}
		if err := requestDB.Create(&userOrg).Error; err != nil {
			a.Log.Error("Failed to create user organization entry for SSO user", "error", err)
			// Non-fatal: user was already created
		}

		a.Log.Info("Created SSO user", "user_id", user.ID, "email", user.Email, "provider", provider)
	} else {
		if user.OrganizationID != orgID {
			a.redirectWithError(r, "User account is not authorized for this organization")
			return nil
		}

		// Check if user is active
		if !user.IsActive {
			a.redirectWithError(r, "Account is disabled")
			return nil
		}

		needsSave := false
		switch {
		case user.SSOProvider == "" && user.SSOProviderID == "":
			if provider == "custom" {
				a.redirectWithError(r, "SSO account is not linked. Contact your administrator.")
				return nil
			}
			user.SSOProvider = provider
			user.SSOProviderID = userInfo.ID
			needsSave = true
		case user.SSOProvider != provider:
			a.redirectWithError(r, "SSO account is linked to a different provider")
			return nil
		case user.SSOProviderID == "":
			if provider == "custom" {
				a.redirectWithError(r, "SSO account is not linked. Contact your administrator.")
				return nil
			}
			user.SSOProviderID = userInfo.ID
			needsSave = true
		case user.SSOProviderID != userInfo.ID:
			a.redirectWithError(r, "SSO account identity mismatch")
			return nil
		}

		if needsSave {
			if err := requestDB.Save(&user).Error; err != nil {
				a.Log.Error("Failed to update SSO user binding", "error", err, "user_id", user.ID)
				a.redirectWithError(r, "Failed to complete authentication")
				return nil
			}
		}
	}

	// Generate JWT tokens
	accessToken, accessTokenExpiresAt, err := a.generateAccessToken(&user)
	if err != nil {
		a.Log.Error("Failed to generate access token", "error", err)
		a.redirectWithError(r, "Failed to complete authentication")
		return nil
	}

	refreshToken, err := a.generateRefreshToken(&user)
	if err != nil {
		a.Log.Error("Failed to generate refresh token", "error", err)
		a.redirectWithError(r, "Failed to complete authentication")
		return nil
	}

	// Set auth cookies (tokens no longer exposed in URL)
	if err := a.setAuthCookies(r, accessToken, accessTokenExpiresAt, refreshToken); err != nil {
		a.Log.Error("Failed to set auth cookies", "error", err)
		a.redirectWithError(r, "Failed to complete authentication")
		return nil
	}

	// Redirect to frontend SSO callback page (cookies already set)
	basePath := sanitizeRedirectPath(a.Config.Server.BasePath)
	redirectURL := fmt.Sprintf("%s/auth/sso/callback", basePath)

	r.RequestCtx.Redirect(redirectURL, fasthttp.StatusTemporaryRedirect)
	return nil
}

// GetSSOSettings returns all SSO provider configs for the organization (admin only)
func (a *App) GetSSOSettings(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceSettingsSSO, models.ActionRead); err != nil {
		return nil
	}

	var providers []models.SSOProvider
	if err := requestDB.Where("organization_id = ?", orgID).Find(&providers).Error; err != nil {
		a.Log.Error("Failed to fetch SSO providers", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch SSO settings", nil, "")
	}

	// Map to response (hide secrets)
	result := make([]SSOProviderResponse, 0, len(providers))
	for _, p := range providers {
		result = append(result, SSOProviderResponse{
			Provider:        p.Provider,
			ClientID:        p.ClientID,
			HasSecret:       p.ClientSecret != "",
			IsEnabled:       p.IsEnabled,
			AllowAutoCreate: p.AllowAutoCreate,
			DefaultRole:     p.DefaultRoleName,
			AllowedDomains:  p.AllowedDomains,
			AuthURL:         p.AuthURL,
			TokenURL:        p.TokenURL,
			UserInfoURL:     p.UserInfoURL,
		})
	}

	return r.SendEnvelope(result)
}

// UpdateSSOProvider creates or updates an SSO provider config (admin only)
func (a *App) UpdateSSOProvider(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceSettingsSSO, models.ActionWrite); err != nil {
		return nil
	}

	provider := r.RequestCtx.UserValue("provider").(string)

	// Validate provider
	validProviders := []string{"google", "microsoft", "github", "facebook", "custom"}
	isValid := false
	for _, p := range validProviders {
		if p == provider {
			isValid = true
			break
		}
	}
	if !isValid {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid provider", nil, "")
	}

	var req SSOProviderRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Validate custom provider fields
	if provider == "custom" {
		if req.AuthURL == "" || req.TokenURL == "" || req.UserInfoURL == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Custom provider requires auth_url, token_url, and user_info_url", nil, "")
		}
		if req.AuthURL, err = a.validateCustomSSOEndpoint(req.AuthURL); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, fmt.Sprintf("Invalid auth_url: %v", err), nil, "auth_url")
		}
		if req.TokenURL, err = a.validateCustomSSOEndpoint(req.TokenURL); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, fmt.Sprintf("Invalid token_url: %v", err), nil, "token_url")
		}
		if req.UserInfoURL, err = a.validateCustomSSOEndpoint(req.UserInfoURL); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, fmt.Sprintf("Invalid user_info_url: %v", err), nil, "user_info_url")
		}
	}

	// Find or create SSO provider config
	var ssoConfig models.SSOProvider
	err = requestDB.Where("organization_id = ? AND provider = ?", orgID, provider).First(&ssoConfig).Error

	if err != nil {
		// Create new
		ssoConfig = models.SSOProvider{
			OrganizationID: orgID,
			Provider:       provider,
		}
	}

	// Update fields
	ssoConfig.ClientID = req.ClientID
	if req.ClientSecret != "" {
		enc, err := appcrypto.Encrypt(req.ClientSecret, a.Config.App.EncryptionKey)
		if err != nil {
			a.Log.Error("Failed to encrypt SSO client secret", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save SSO configuration", nil, "")
		}
		ssoConfig.ClientSecret = enc
	}
	ssoConfig.IsEnabled = req.IsEnabled
	ssoConfig.AllowAutoCreate = req.AllowAutoCreate
	ssoConfig.DefaultRoleName = req.DefaultRole
	if ssoConfig.DefaultRoleName == "" {
		ssoConfig.DefaultRoleName = "agent"
	}
	ssoConfig.AllowedDomains = req.AllowedDomains
	ssoConfig.AuthURL = req.AuthURL
	ssoConfig.TokenURL = req.TokenURL
	ssoConfig.UserInfoURL = req.UserInfoURL

	if err := requestDB.Save(&ssoConfig).Error; err != nil {
		a.Log.Error("Failed to save SSO provider", "error", err, "provider", provider)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save SSO settings", nil, "")
	}

	return r.SendEnvelope(SSOProviderResponse{
		Provider:        ssoConfig.Provider,
		ClientID:        ssoConfig.ClientID,
		HasSecret:       ssoConfig.ClientSecret != "",
		IsEnabled:       ssoConfig.IsEnabled,
		AllowAutoCreate: ssoConfig.AllowAutoCreate,
		DefaultRole:     ssoConfig.DefaultRoleName,
		AllowedDomains:  ssoConfig.AllowedDomains,
		AuthURL:         ssoConfig.AuthURL,
		TokenURL:        ssoConfig.TokenURL,
		UserInfoURL:     ssoConfig.UserInfoURL,
	})
}

// DeleteSSOProvider removes an SSO provider config (admin only)
func (a *App) DeleteSSOProvider(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceSettingsSSO, models.ActionWrite); err != nil {
		return nil
	}

	provider := r.RequestCtx.UserValue("provider").(string)

	result := requestDB.Where("organization_id = ? AND provider = ?", orgID, provider).Delete(&models.SSOProvider{})
	if result.Error != nil {
		a.Log.Error("Failed to delete SSO provider", "error", result.Error, "provider", provider)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete SSO provider", nil, "")
	}

	if result.RowsAffected == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "SSO provider not found", nil, "")
	}

	return r.SendEnvelope(map[string]string{"message": "SSO provider deleted"})
}
