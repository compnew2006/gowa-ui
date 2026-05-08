package licensestudio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBrowserValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid https", "https://example.com", false},
		{"valid http", "http://example.com", false},
		{"valid with path", "https://example.com/license", false},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"no scheme", "example.com", true},
		{"ftp scheme", "ftp://example.com", true},
		{"missing host", "https://", true},
		{"trims whitespace", "  https://example.com  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateBrowserURL(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, got)
			}
		})
	}
}

func TestOpenBrowser(t *testing.T) {
	t.Run("invalid URL returns error", func(t *testing.T) {
		err := openBrowser("")
		assert.Error(t, err)
	})

	t.Run("invalid scheme returns error", func(t *testing.T) {
		err := openBrowser("ftp://example.com")
		assert.Error(t, err)
	})
}
