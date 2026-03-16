package handlers_test

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/stretchr/testify/assert"
)

// TestGenerateRandomString_ValidLength tests generateRandomString with valid length
func TestGenerateRandomString_ValidLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		length int
	}{
		{
			name:  "length 1",
			length: 1,
		},
		{
			name:  "length 8",
			length: 8,
		},
		{
			name:  "length 16",
			length: 16,
		},
		{
			name:  "length 32",
			length: 32,
		},
		{
			name:  "length 64",
			length: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := handlers.GenerateRandomString(tt.length)

			assert.NoError(t, err, "generateRandomString should succeed with valid length")
			assert.Len(t, result, tt.length, "generateRandomString should return string of requested length")

			// Verify it only contains base64url safe characters
			for _, c := range result {
				assert.True(t, isBase64URLChar(c), "generateRandomString should only contain base64url characters")
			}
		})
	}
}

// TestGenerateRandomString_UniqueValues tests that generateRandomString produces unique values
func TestGenerateRandomString_UniqueValues(t *testing.T) {
	t.Parallel()

	results := make(map[string]bool)
	for i := 0; i < 100; i++ {
		result, err := handlers.GenerateRandomString(16)
		assert.NoError(t, err, "generateRandomString should succeed")
		assert.False(t, results[result], "generateRandomString should produce unique values")
		results[result] = true
	}
}

// TestGenerateRandomString_ZeroLength tests generateRandomString with zero length
func TestGenerateRandomString_ZeroLength(t *testing.T) {
	t.Parallel()

	result, err := handlers.GenerateRandomString(0)

	assert.NoError(t, err, "generateRandomString should succeed with zero length")
	assert.Empty(t, result, "generateRandomString should return empty string for zero length")
}

// TestGenerateCSRFToken_ValidToken tests generateCSRFToken produces valid tokens
func TestGenerateCSRFToken_ValidToken(t *testing.T) {
	t.Parallel()

	result, err := handlers.GenerateCSRFToken()

	assert.NoError(t, err, "generateCSRFToken should succeed")
	assert.NotEmpty(t, result, "generateCSRFToken should return non-empty token")

	// RawURLEncoding of 32 bytes produces 43 characters
	assert.Len(t, result, 43, "generateCSRFToken should return 43 character string (32 bytes base64url encoded without padding)")

	// Verify it only contains base64url safe characters
	for _, c := range result {
		assert.True(t, isBase64URLChar(c), "generateCSRFToken should only contain base64url characters")
	}
}

// TestGenerateCSRFToken_UniqueTokens tests that generateCSRFToken produces unique tokens
func TestGenerateCSRFToken_UniqueTokens(t *testing.T) {
	t.Parallel()

	results := make(map[string]bool)
	for i := 0; i < 100; i++ {
		result, err := handlers.GenerateCSRFToken()
		assert.NoError(t, err, "generateCSRFToken should succeed")
		assert.False(t, results[result], "generateCSRFToken should produce unique tokens")
		results[result] = true
	}
}

// TestGenerateCSRFToken_NoSlashOrPlus tests that generateCSRFToken doesn't contain + or /
func TestGenerateCSRFToken_NoSlashOrPlus(t *testing.T) {
	t.Parallel()

	for i := 0; i < 10; i++ {
		result, err := handlers.GenerateCSRFToken()
		assert.NoError(t, err, "generateCSRFToken should succeed")
		assert.NotContains(t, result, "+", "generateCSRFToken should not contain + (not base64url safe)")
		assert.NotContains(t, result, "/", "generateCSRFToken should not contain / (not base64url safe)")
		assert.NotContains(t, result, "=", "generateCSRFToken should not contain = (no padding in raw encoding)")
	}
}

// TestGenerateRandomString_NoSlashOrPlus tests that generateRandomString doesn't contain + or /
func TestGenerateRandomString_NoSlashOrPlus(t *testing.T) {
	t.Parallel()

	for i := 0; i < 10; i++ {
		result, err := handlers.GenerateRandomString(32)
		assert.NoError(t, err, "generateRandomString should succeed")
		assert.NotContains(t, result, "+", "generateRandomString should not contain + (not base64url safe)")
		assert.NotContains(t, result, "/", "generateRandomString should not contain / (not base64url safe)")
	}
}

// Helper function to check if a character is valid base64url
func isBase64URLChar(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}
