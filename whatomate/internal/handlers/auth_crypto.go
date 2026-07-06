package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// generateRandomString returns n random characters, base64url encoded.
func generateRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand.Read failed: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b)[:n], nil
}

// generateCSRFToken returns 32 random bytes, base64url encoded.
func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand.Read failed: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
