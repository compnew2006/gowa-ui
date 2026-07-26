package whatsapp

// Account represents WhatsApp Business Account credentials for a GOWA
// (Go WhatsApp Web Multi-Device) instance.
type Account struct {
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
