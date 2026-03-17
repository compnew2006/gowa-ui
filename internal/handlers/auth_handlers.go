package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Login authenticates a user and returns tokens
func (a *App) Login(r *fastglue.Request) error {
	var req LoginRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	// Find user by email with role preloaded
	var user models.User
	if err := a.DB.Preload("Role").Where("email = ?", req.Email).First(&user).Error; err != nil {
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"), []byte(req.Password))
		a.LogAuthFailure(r, req.Email, nil, nil, "user_not_found")
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid credentials", nil, "")
	}

	// Load permissions from cache
	if user.Role != nil && user.RoleID != nil {
		cachedPerms, err := a.GetRolePermissionsCached(*user.RoleID)
		if err == nil {
			permissions := make([]models.Permission, 0, len(cachedPerms))
			for _, p := range cachedPerms {
				for i := len(p) - 1; i >= 0; i-- {
					if p[i] == ':' {
						permissions = append(permissions, models.Permission{
							Resource: p[:i],
							Action:   p[i+1:],
						})
						break
					}
				}
			}
			user.Role.Permissions = permissions
		}
	}

	// Check if user is active
	if !user.IsActive {
		userID := user.ID
		orgID := user.OrganizationID
		a.LogAuthFailure(r, req.Email, &userID, &orgID, "account_disabled")
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Account is disabled", nil, "")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		userID := user.ID
		orgID := user.OrganizationID
		a.LogAuthFailure(r, req.Email, &userID, &orgID, "invalid_password")
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid credentials", nil, "")
	}

	// Generate tokens
	accessToken, accessTokenExpiresAt, err := a.generateAccessToken(&user)
	if err != nil {
		a.Log.Error("Failed to generate access token", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate token", nil, "")
	}

	refreshToken, err := a.generateRefreshToken(&user)
	if err != nil {
		a.Log.Error("Failed to generate refresh token", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate token", nil, "")
	}

	if err := a.setAuthCookies(r, accessToken, accessTokenExpiresAt, refreshToken); err != nil {
		a.Log.Error("Failed to set auth cookies", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Internal Server Error", nil, "")
	}
	a.LogAuthSuccess(r, &user)

	now := time.Now()
	return r.SendEnvelope(CookieAuthResponse{
		ExpiresIn: accessTokenTTLSeconds(now, accessTokenExpiresAt),
		User:      user,
	})
}

// CreateRegisterInvite creates a new registration invitation link
func (a *App) CreateRegisterInvite(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceUsers, models.ActionWrite); err != nil {
		return nil
	}

	req := CreateRegisterInviteRequest{}
	if len(r.RequestCtx.Request.Body()) > 0 {
		if err := a.decodeRequest(r, &req); err != nil {
			return nil
		}
	}

	expiresInHours := 24
	if req.ExpiresInHours > 0 {
		expiresInHours = req.ExpiresInHours
	}
	if expiresInHours < 1 || expiresInHours > 168 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "expires_in_hours must be between 1 and 168", nil, "")
	}

	token, expiresAt, err := a.generateRegisterInviteToken(orgID, time.Duration(expiresInHours)*time.Hour)
	if err != nil {
		a.Log.Error("Failed to generate registration invite token", "error", err, "org_id", orgID, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate invite link", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"token":           token,
		"organization_id": orgID,
		"expires_at":      expiresAt.UTC().Format(time.RFC3339),
	})
}

// Register creates a new user in an existing organization
func (a *App) Register(r *fastglue.Request) error {
	var req RegisterRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	if req.InvitationToken == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invitation_token is required", nil, "")
	}

	inviteOrgID, err := a.validateRegisterInviteToken(req.InvitationToken)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid or expired invitation link", nil, "")
	}

	if req.OrganizationID != uuid.Nil && req.OrganizationID != inviteOrgID {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invitation link organization mismatch", nil, "")
	}

	// Validate the organization exists
	var org models.Organization
	if err := a.DB.Where("id = ?", inviteOrgID).First(&org).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Organization not found", nil, "")
	}

	// Get the org's default role
	var defaultRole models.CustomRole
	if err := a.DB.Where("organization_id = ? AND is_default = ?", inviteOrgID, true).First(&defaultRole).Error; err != nil {
		if err := a.DB.Where("organization_id = ? AND name = ? AND is_system = ?", inviteOrgID, "agent", true).First(&defaultRole).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to find default role", nil, "")
		}
	}

	// Check if email already exists
	var existingUser models.User
	if err := a.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		if err := bcrypt.CompareHashAndPassword([]byte(existingUser.PasswordHash), []byte(req.Password)); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusConflict, "An account with this email already exists", nil, "")
		}

		if !existingUser.IsActive {
			return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Account is disabled", nil, "")
		}

		var count int64
		a.DB.Model(&models.UserOrganization{}).
			Where("user_id = ? AND organization_id = ?", existingUser.ID, inviteOrgID).
			Count(&count)
		if count > 0 {
			return r.SendErrorEnvelope(fasthttp.StatusConflict, "You are already a member of this organization", nil, "")
		}

		userOrg := models.UserOrganization{
			UserID:         existingUser.ID,
			OrganizationID: inviteOrgID,
			RoleID:         &defaultRole.ID,
			IsDefault:      false,
		}
		if err := a.DB.Create(&userOrg).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to join organization", nil, "")
		}

		existingUser.OrganizationID = inviteOrgID
		existingUser.Role = &defaultRole
		existingUser.RoleID = &defaultRole.ID

		accessToken, accessTokenExpiresAt, err := a.generateAccessToken(&existingUser)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate token", nil, "")
		}
		refreshToken, err := a.generateRefreshToken(&existingUser)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate token", nil, "")
		}

		if err := a.setAuthCookies(r, accessToken, accessTokenExpiresAt, refreshToken); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Internal Server Error", nil, "")
		}

		now := time.Now()
		return r.SendEnvelope(CookieAuthResponse{
			ExpiresIn: accessTokenTTLSeconds(now, accessTokenExpiresAt),
			User:      existingUser,
		})
	}

	if err := validatePasswordStrength(req.Password); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "password")
	}

	hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if hashErr != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create account", nil, "")
	}
	tx := a.DB.Begin()

	user := models.User{
		OrganizationID: inviteOrgID,
		Email:          req.Email,
		PasswordHash:   string(hashedPassword),
		FullName:       req.FullName,
		RoleID:         &defaultRole.ID,
		IsActive:       true,
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create account", nil, "")
	}

	userOrg := models.UserOrganization{
		UserID:         user.ID,
		OrganizationID: inviteOrgID,
		RoleID:         &defaultRole.ID,
		IsDefault:      true,
	}
	if err := tx.Create(&userOrg).Error; err != nil {
		tx.Rollback()
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create account", nil, "")
	}

	if err := tx.Commit().Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create account", nil, "")
	}

	user.Role = &defaultRole
	accessToken, accessTokenExpiresAt, err := a.generateAccessToken(&user)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate token", nil, "")
	}
	refreshToken, err := a.generateRefreshToken(&user)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate token", nil, "")
	}

	if err := a.setAuthCookies(r, accessToken, accessTokenExpiresAt, refreshToken); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Internal Server Error", nil, "")
	}

	now := time.Now()
	return r.SendEnvelope(CookieAuthResponse{
		ExpiresIn: accessTokenTTLSeconds(now, accessTokenExpiresAt),
		User:      user,
	})
}

// RefreshToken refreshes access token using refresh token with rotation.
func (a *App) RefreshToken(r *fastglue.Request) error {
	refreshTokenStr := string(r.RequestCtx.Request.Header.Cookie(cookieRefreshName))
	if refreshTokenStr == "" {
		var req RefreshRequest
		_ = r.Decode(&req, "json")
		refreshTokenStr = req.RefreshToken
	}
	if refreshTokenStr == "" {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Missing refresh token", nil, "")
	}

	token, err := jwt.ParseWithClaims(refreshTokenStr, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return a.jwtSecretBytes()
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil || !token.Valid {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid refresh token", nil, "")
	}

	claims, ok := token.Claims.(*middleware.JWTClaims)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid token claims", nil, "")
	}
	if claims.Subject != refreshTokenSubject || claims.ID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid refresh token", nil, "")
	}
	if a.Redis == nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Refresh token storage is unavailable", nil, "")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	deleted, err := a.Redis.Del(ctx, refreshTokenKey(claims.ID)).Result()
	if err != nil || deleted == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Refresh token has been revoked", nil, "")
	}

	var user models.User
	if err := a.DB.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "User not found", nil, "")
	}

	if !user.IsActive {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Account is disabled", nil, "")
	}

	if err := a.applyRefreshTokenOrgContext(&user, claims.OrganizationID); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid refresh token", nil, "")
	}
	if err := a.populateUserRolePermissions(&user); err != nil {
		a.Log.Error("Failed to hydrate role permissions during refresh", "error", err, "user_id", user.ID, "organization_id", user.OrganizationID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate token", nil, "")
	}

	accessToken, accessTokenExpiresAt, err := a.generateAccessToken(&user)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate token", nil, "")
	}
	newRefreshToken, err := a.generateRefreshToken(&user)
	if err != nil {
		if errors.Is(err, errRefreshTokenStorageUnavailable) {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Refresh token storage is unavailable", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate token", nil, "")
	}

	if err := a.setAuthCookies(r, accessToken, accessTokenExpiresAt, newRefreshToken); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Internal Server Error", nil, "")
	}

	now := time.Now()
	return r.SendEnvelope(CookieAuthResponse{
		ExpiresIn: accessTokenTTLSeconds(now, accessTokenExpiresAt),
		User:      user,
	})
}

func (a *App) applyRefreshTokenOrgContext(user *models.User, tokenOrgID uuid.UUID) error {
	if user == nil {
		return errors.New("user is nil")
	}

	targetOrgID := tokenOrgID
	if targetOrgID == uuid.Nil {
		targetOrgID = user.OrganizationID
	}
	if targetOrgID == uuid.Nil {
		return errors.New("missing organization")
	}

	if user.IsSuperAdmin {
		var org models.Organization
		if err := a.DB.Select("id").Where("id = ?", targetOrgID).First(&org).Error; err != nil {
			return err
		}
		user.OrganizationID = targetOrgID
		return nil
	}

	var membership models.UserOrganization
	if err := a.DB.Where("user_id = ? AND organization_id = ?", user.ID, targetOrgID).First(&membership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return err
	}

	user.OrganizationID = targetOrgID
	user.RoleID = membership.RoleID
	return nil
}

func (a *App) populateUserRolePermissions(user *models.User) error {
	if user == nil {
		return errors.New("user is nil")
	}
	if user.RoleID == nil {
		user.Role = nil
		return nil
	}

	var role models.CustomRole
	if err := a.DB.Where("id = ?", *user.RoleID).First(&role).Error; err != nil {
		return err
	}

	cachedPerms, err := a.GetRolePermissionsCached(*user.RoleID)
	if err == nil {
		permissions := make([]models.Permission, 0, len(cachedPerms))
		for _, p := range cachedPerms {
			parts := splitPermission(p)
			if len(parts) == 2 {
				permissions = append(permissions, models.Permission{
					Resource: parts[0],
					Action:   parts[1],
				})
			}
		}
		role.Permissions = permissions
	}

	user.Role = &role
	return nil
}

// SwitchOrg generates new tokens for a different organization the user belongs to
func (a *App) SwitchOrg(r *fastglue.Request) error {
	userID, ok := r.RequestCtx.UserValue("user_id").(uuid.UUID)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var req SwitchOrgRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	if req.OrganizationID == uuid.Nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "organization_id is required", nil, "")
	}

	var org models.Organization
	if err := a.DB.Where("id = ?", req.OrganizationID).First(&org).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Organization not found", nil, "")
	}

	var user models.User
	if err := a.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
	}

	if !user.IsSuperAdmin {
		var userOrg models.UserOrganization
		if err := a.DB.Where("user_id = ? AND organization_id = ?", userID, req.OrganizationID).First(&userOrg).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You are not a member of this organization", nil, "")
		}
		if userOrg.RoleID != nil {
			user.RoleID = userOrg.RoleID
		}
	}

	user.OrganizationID = req.OrganizationID

	if user.RoleID != nil {
		var role models.CustomRole
		if err := a.DB.Where("id = ?", *user.RoleID).First(&role).Error; err == nil {
			user.Role = &role
			cachedPerms, err := a.GetRolePermissionsCached(*user.RoleID)
			if err == nil {
				permissions := make([]models.Permission, 0, len(cachedPerms))
				for _, p := range cachedPerms {
					parts := splitPermission(p)
					if len(parts) == 2 {
						permissions = append(permissions, models.Permission{
							Resource: parts[0],
							Action:   parts[1],
						})
					}
				}
				user.Role.Permissions = permissions
			}
		}
	}

	accessToken, accessTokenExpiresAt, err := a.generateAccessToken(&user)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate token", nil, "")
	}

	refreshToken, err := a.generateRefreshToken(&user)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate token", nil, "")
	}

	if err := a.setAuthCookies(r, accessToken, accessTokenExpiresAt, refreshToken); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Internal Server Error", nil, "")
	}

	now := time.Now()
	return r.SendEnvelope(CookieAuthResponse{
		ExpiresIn: accessTokenTTLSeconds(now, accessTokenExpiresAt),
		User:      user,
	})
}

// Logout invalidates the user's refresh token
func (a *App) Logout(r *fastglue.Request) error {
	var loggedUserID *uuid.UUID
	var loggedOrgID *uuid.UUID

	refreshTokenStr := string(r.RequestCtx.Request.Header.Cookie(cookieRefreshName))
	if refreshTokenStr == "" {
		var req LogoutRequest
		_ = r.Decode(&req, "json")
		refreshTokenStr = req.RefreshToken
	}

	if refreshTokenStr != "" {
		token, _ := jwt.ParseWithClaims(refreshTokenStr, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return a.jwtSecretBytes()
		})
		if token != nil {
			if claims, ok := token.Claims.(*middleware.JWTClaims); ok {
				if claims.UserID != uuid.Nil {
					u := claims.UserID
					loggedUserID = &u
				}
				if claims.OrganizationID != uuid.Nil {
					o := claims.OrganizationID
					loggedOrgID = &o
				}
				if claims.ID != "" {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					a.Redis.Del(ctx, refreshTokenKey(claims.ID))
				}
			}
		}
	}

	a.clearAuthCookies(r)
	a.LogLogout(r, loggedUserID, loggedOrgID)

	return r.SendEnvelope(map[string]string{"status": "logged_out"})
}

// GetWSToken returns a short-lived single-use JWT for WebSocket authentication.
func (a *App) GetWSToken(r *fastglue.Request) error {
	userID, ok := r.RequestCtx.UserValue("user_id").(uuid.UUID)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	orgID, _ := r.RequestCtx.UserValue("organization_id").(uuid.UUID)

	claims := middleware.JWTClaims{
		UserID:         userID,
		OrganizationID: orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "whatomate",
			Subject:   wsTokenSubject,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signingKey, err := a.jwtSecretBytes()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Authentication misconfigured", nil, "")
	}

	signed, err := token.SignedString(signingKey)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate token", nil, "")
	}

	return r.SendEnvelope(map[string]string{"token": signed})
}
