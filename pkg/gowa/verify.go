package gowa

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// GenerateWebhookSecret generates a random 32-byte hex webhook secret.
func GenerateWebhookSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateDeviceID creates a slug-style device ID from a name plus random suffix.
func GenerateDeviceID(name string) string {
	slug := strings.ToLower(name)
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, slug)
	// Collapse consecutive dashes
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "device"
	}
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	return slug + "-" + hex.EncodeToString(suffix)
}

// VerifyWebhookSignature verifies the X-Hub-Signature-256 header against the
// raw request body. The header format is "sha256={hex_lower_hmac}".
//
// IMPORTANT: rawBody must be the exact bytes received on the wire — not
// re-serialized JSON — because the HMAC is computed over the original body.
func VerifyWebhookSignature(rawBody []byte, signatureHeader, secret string) bool {
	if signatureHeader == "" || secret == "" {
		return false
	}

	// Strip the "sha256=" prefix.
	const prefix = "sha256="
	if !strings.HasPrefix(signatureHeader, prefix) {
		return false
	}
	receivedSig, err := hex.DecodeString(signatureHeader[len(prefix):])
	if err != nil {
		return false
	}

	// Compute expected HMAC.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	expectedSig := mac.Sum(nil)

	// Constant-time comparison to prevent timing attacks.
	return hmac.Equal(receivedSig, expectedSig)
}
