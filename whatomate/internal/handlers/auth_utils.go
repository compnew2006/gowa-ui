package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	accessTokenSubject  = "access"
	refreshTokenSubject = "refresh"
	wsTokenSubject      = "ws"
)

var errRefreshTokenStorageUnavailable = errors.New("refresh token storage is unavailable")

func (a *App) generateAccessToken(user *models.User) (string, time.Time, error) {
	now := time.Now()
	ttlMinutes := defaultAccessTokenExpiryMinutes
	if a != nil && a.Config != nil && a.Config.JWT.AccessExpiryMins > 0 {
		ttlMinutes = a.Config.JWT.AccessExpiryMins
	}
	expiresAt := nextAccessTokenExpiry(now, ttlMinutes)

	claims := middleware.JWTClaims{
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		Email:          user.Email,
		RoleID:         user.RoleID,
		IsSuperAdmin:   user.IsSuperAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "whatomate",
			Subject:   accessTokenSubject,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signingKey, err := a.jwtSecretBytes()
	if err != nil {
		return "", time.Time{}, err
	}
	signed, err := token.SignedString(signingKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

func (a *App) generateRefreshToken(user *models.User) (string, error) {
	if a == nil || a.Config == nil {
		return "", fmt.Errorf("app or config is nil")
	}
	if a.Redis == nil {
		return "", errRefreshTokenStorageUnavailable
	}

	jti := uuid.New().String()
	expiry := time.Duration(a.Config.JWT.RefreshExpiryDays) * 24 * time.Hour

	claims := middleware.JWTClaims{
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		Email:          user.Email,
		RoleID:         user.RoleID,
		IsSuperAdmin:   user.IsSuperAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "whatomate",
			Subject:   refreshTokenSubject,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signingKey, err := a.jwtSecretBytes()
	if err != nil {
		return "", err
	}

	signed, err := token.SignedString(signingKey)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Redis.Set(ctx, refreshTokenKey(jti), user.ID.String(), expiry).Err(); err != nil {
		a.Log.Error("Failed to store refresh token in Redis", "error", err)
		return "", fmt.Errorf("%w: %v", errRefreshTokenStorageUnavailable, err)
	}

	return signed, nil
}

func (a *App) generateRegisterInviteToken(orgID uuid.UUID, ttl time.Duration) (string, time.Time, error) {
	if a == nil || a.Config == nil {
		return "", time.Time{}, fmt.Errorf("app or config is nil")
	}

	expiresAt := time.Now().Add(ttl)
	claims := RegisterInviteClaims{
		OrganizationID: orgID,
		Purpose:        registerInvitePurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "whatomate",
			Subject:   "org_registration_invite",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signingKey, err := a.jwtSecretBytes()
	if err != nil {
		return "", time.Time{}, err
	}

	signed, err := token.SignedString(signingKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

func (a *App) validateRegisterInviteToken(tokenString string) (uuid.UUID, error) {
	if a == nil || a.Config == nil {
		return uuid.Nil, fmt.Errorf("app or config is nil")
	}

	token, err := jwt.ParseWithClaims(tokenString, &RegisterInviteClaims{}, func(token *jwt.Token) (interface{}, error) {
		signingMethod, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok || signingMethod.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected JWT signing method: %s", token.Method.Alg())
		}
		return a.jwtSecretBytes()
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid invite token")
	}

	claims, ok := token.Claims.(*RegisterInviteClaims)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid invite claims")
	}
	if claims.OrganizationID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("missing organization id in invite")
	}
	if claims.Purpose != registerInvitePurpose {
		return uuid.Nil, fmt.Errorf("invalid invite purpose")
	}

	return claims.OrganizationID, nil
}

func refreshTokenKey(jti string) string {
	return fmt.Sprintf("refresh:%s", jti)
}

func generateSlug(name string) string {
	slug := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			slug += string(c)
		} else if c >= 'A' && c <= 'Z' {
			slug += string(c + 32)
		} else if c == ' ' || c == '-' {
			slug += "-"
		}
	}
	u := uuid.New().String()
	if len(u) > 8 {
		return slug + "-" + u[:8]
	}
	return slug + "-" + u
}
