package whatsapp

import "github.com/google/uuid"

// Account represents WhatsApp Business Account credentials for a GOWA
// (Go WhatsApp Web Multi-Device) instance.
type Account struct {
	// OrganizationID owns this account. The provider registry keys cached
	// clients by (OrganizationID, GowaBaseURL) so two organizations that
	// register the same GOWA base URL with different Basic Auth credentials
	// get isolated clients — never each other's credentials. Zero value
	// (uuid.Nil) falls back to the legacy base-URL-only key for compatibility.
	OrganizationID uuid.UUID

	// GOWA credentials — used by the GOWA provider to route API calls.
	GowaBaseURL  string // GOWA REST API base URL
	GowaDeviceID string // GOWA device UUID
}

// Button represents an interactive button
type Button struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type,omitempty"` // "reply" (default) or "url"
	URL   string `json:"url,omitempty"`  // URL for type="url" buttons
}
