package whatsapp

// Recipient identifies a WhatsApp user by phone number and/or BSUID.
type Recipient struct {
	Phone string // Phone number (e.g., "16505551234")
	BSUID string // Business-Scoped User ID (e.g., "US.13491208655302741918")
}
