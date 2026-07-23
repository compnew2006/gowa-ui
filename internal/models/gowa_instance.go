package models

import (
	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/crypto"
)

// GowaInstance is a DB-managed GOWA server (base_url + Basic Auth credentials)
// scoped to an organization. Username and Password are encrypted at rest and
// never serialized to JSON responses (use ToResponse to expose has_credentials).
type GowaInstance struct {
	BaseModel
	OrganizationID uuid.UUID `gorm:"type:uuid;index;not null" json:"organization_id"`
	Name           string    `gorm:"size:100;not null" json:"name"`
	BaseURL        string    `gorm:"size:255;not null" json:"base_url"`
	Username       string    `gorm:"size:255" json:"-"` // encrypted, never serialized
	Password       string    `gorm:"size:255" json:"-"` // encrypted, never serialized
	WebhookURL     string    `gorm:"size:255" json:"webhook_url,omitempty"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`

	// Relations
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

func (GowaInstance) TableName() string { return "gowa_instances" }

// EncryptCredentials encrypts Username and Password in place using the given
// key. Fields already carrying the "enc:" prefix are skipped (idempotent).
func (g *GowaInstance) EncryptCredentials(key string) error {
	return crypto.EncryptFields(key, &g.Username, &g.Password)
}

// DecryptCredentials decrypts Username and Password in place. Safe to call on
// legacy/unencrypted values.
func (g *GowaInstance) DecryptCredentials(key string) {
	crypto.DecryptFields(key, &g.Username, &g.Password)
}

// HasCredentials reports whether both Username and Password are populated.
func (g *GowaInstance) HasCredentials() bool {
	return g.Username != "" && g.Password != ""
}

// GowaInstanceResponse is the credentials-safe projection returned by handlers.
type GowaInstanceResponse struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	BaseURL        string    `json:"base_url"`
	WebhookURL     string    `json:"webhook_url"`
	IsActive       bool      `json:"is_active"`
	HasCredentials bool      `json:"has_credentials"`
	CreatedAt      string    `json:"created_at"`
	UpdatedAt      string    `json:"updated_at"`
}

// ToResponse builds the credentials-safe projection. Call DecryptCredentials
// first if you need the raw credentials (not exposed here).
func (g *GowaInstance) ToResponse() GowaInstanceResponse {
	return GowaInstanceResponse{
		ID:             g.ID,
		Name:           g.Name,
		BaseURL:        g.BaseURL,
		WebhookURL:     g.WebhookURL,
		IsActive:       g.IsActive,
		HasCredentials: g.HasCredentials(),
		CreatedAt:      g.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      g.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
