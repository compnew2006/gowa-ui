package handlers

import (
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
)

// OAuth provider configurations (endpoints are hardcoded, only need client credentials)
var oauthProviders = map[string]struct {
	Endpoint    oauth2.Endpoint
	Scopes      []string
	UserInfoURL string
}{
	"google": {
		Endpoint:    google.Endpoint,
		Scopes:      []string{"openid", "email", "profile"},
		UserInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
	},
	"microsoft": {
		Endpoint:    microsoft.AzureADEndpoint("common"),
		Scopes:      []string{"openid", "email", "profile", "User.Read"},
		UserInfoURL: "https://graph.microsoft.com/v1.0/me",
	},
	"github": {
		Endpoint:    github.Endpoint,
		Scopes:      []string{"user:email", "read:user"},
		UserInfoURL: "https://api.github.com/user",
	},
	"facebook": {
		Endpoint:    facebook.Endpoint,
		Scopes:      []string{"email", "public_profile"},
		UserInfoURL: "https://graph.facebook.com/me?fields=id,email,name",
	},
}

// SSOState represents the state stored in Redis during OAuth flow
type SSOState struct {
	OrgID     string    `json:"org_id"`
	Provider  string    `json:"provider"`
	Nonce     string    `json:"nonce"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SSOProviderPublic represents public SSO provider info (no secrets)
type SSOProviderPublic struct {
	Provider         string `json:"provider"`
	Name             string `json:"name"`
	OrganizationSlug string `json:"organization_slug,omitempty"`
}

// SSOProviderRequest represents SSO provider config from admin
type SSOProviderRequest struct {
	ClientID        string `json:"client_id" validate:"required"`
	ClientSecret    string `json:"client_secret"`
	IsEnabled       bool   `json:"is_enabled"`
	AllowAutoCreate bool   `json:"allow_auto_create"`
	DefaultRole     string `json:"default_role"`
	AllowedDomains  string `json:"allowed_domains"`
	// Custom provider fields
	AuthURL     string `json:"auth_url"`
	TokenURL    string `json:"token_url"`
	UserInfoURL string `json:"user_info_url"`
}

// SSOProviderResponse represents SSO provider config response (masked secret)
type SSOProviderResponse struct {
	Provider        string `json:"provider"`
	ClientID        string `json:"client_id"`
	HasSecret       bool   `json:"has_secret"`
	IsEnabled       bool   `json:"is_enabled"`
	AllowAutoCreate bool   `json:"allow_auto_create"`
	DefaultRole     string `json:"default_role"`
	AllowedDomains  string `json:"allowed_domains"`
	AuthURL         string `json:"auth_url,omitempty"`
	TokenURL        string `json:"token_url,omitempty"`
	UserInfoURL     string `json:"user_info_url,omitempty"`
}

// providerDisplayNames maps provider keys to display names
var providerDisplayNames = map[string]string{
	"google":    "Google",
	"microsoft": "Microsoft",
	"github":    "GitHub",
	"facebook":  "Facebook",
	"custom":    "Custom SSO",
}
