package handlers

import (
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=12"`
}

// RegisterRequest represents registration data
type RegisterRequest struct {
	Email           string    `json:"email" validate:"required,email"`
	Password        string    `json:"password" validate:"required,min=12"`
	FullName        string    `json:"full_name" validate:"required"`
	OrganizationID  uuid.UUID `json:"organization_id,omitempty"` // Optional legacy field; validated against invitation token when provided.
	InvitationToken string    `json:"invitation_token" validate:"required"`
}

// CookieAuthResponse represents authentication response when tokens are in cookies.
// No tokens in the body — only the expiry hint and user object.
type CookieAuthResponse struct {
	ExpiresIn int         `json:"expires_in"`
	User      models.User `json:"user"`
}

// RefreshRequest represents token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

const registerInvitePurpose = "register_invite"

// RegisterInviteClaims is the signed claim set for public registration links.
// The token binds registration to a single organization and expires quickly.
type RegisterInviteClaims struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	Purpose        string    `json:"purpose"`
	jwt.RegisteredClaims
}

type CreateRegisterInviteRequest struct {
	ExpiresInHours int `json:"expires_in_hours,omitempty"`
}

// SwitchOrgRequest represents request to change organization
type SwitchOrgRequest struct {
	OrganizationID uuid.UUID `json:"organization_id" validate:"required"`
}

// LogoutRequest represents an explicit logout request
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}
