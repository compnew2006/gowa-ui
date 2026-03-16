package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomString_LengthAndCharset(t *testing.T) {
	t.Parallel()

	result, err := generateRandomString(32)
	assert.NoError(t, err)
	assert.Len(t, result, 32)
	for _, c := range result {
		assert.True(t,
			(c >= 'A' && c <= 'Z') ||
				(c >= 'a' && c <= 'z') ||
				(c >= '0' && c <= '9') ||
				c == '-' || c == '_',
		)
	}
}

func TestGenerateRandomString_ZeroLength(t *testing.T) {
	t.Parallel()

	result, err := generateRandomString(0)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestGenerateCSRFToken_LengthAndCharset(t *testing.T) {
	t.Parallel()

	result, err := generateCSRFToken()
	assert.NoError(t, err)
	assert.Len(t, result, 43)
	for _, c := range result {
		assert.True(t,
			(c >= 'A' && c <= 'Z') ||
				(c >= 'a' && c <= 'z') ||
				(c >= '0' && c <= '9') ||
				c == '-' || c == '_',
		)
	}
}
