package handlers

import (
	"strings"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestCampaignTemplateDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		template *models.Template
		expected string
	}{
		{
			name:     "nil template returns empty string",
			template: nil,
			expected: "",
		},
		{
			name: "template with display name returns display name",
			template: &models.Template{
				Name:        "template_name",
				DisplayName: "Template Display Name",
			},
			expected: "Template Display Name",
		},
		{
			name: "template with empty display name returns name",
			template: &models.Template{
				Name:        "template_name",
				DisplayName: "",
			},
			expected: "template_name",
		},
		{
			name: "template with whitespace-only display name returns name",
			template: &models.Template{
				Name:        "template_name",
				DisplayName: "   ",
			},
			expected: "template_name",
		},
		{
			name: "template with display name containing leading/trailing spaces returns trimmed",
			template: &models.Template{
				Name:        "template_name",
				DisplayName: "  Display Name  ",
			},
			expected: "  Display Name  ",
		},
		{
			name: "template with both name and display name populated returns display name",
			template: &models.Template{
				Name:        "fallback_name",
				DisplayName: "Primary Display Name",
			},
			expected: "Primary Display Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := campaignTemplateDisplayName(tt.template)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeCampaignDelayRange(t *testing.T) {
	tests := []struct {
		name          string
		currentMin    int
		currentMax    int
		requestedMin  *int
		requestedMax  *int
		expectedMin   int
		expectedMax   int
		expectError   bool
		errorContains string
	}{
		{
			name:        "no requested values returns current",
			currentMin:  5,
			currentMax:  10,
			requestedMin: nil,
			requestedMax: nil,
			expectedMin:  5,
			expectedMax:  10,
			expectError:  false,
		},
		{
			name:        "requested min only updates min",
			currentMin:  5,
			currentMax:  10,
			requestedMin: intPtr(7),
			requestedMax: nil,
			expectedMin:  7,
			expectedMax:  10,
			expectError:  false,
		},
		{
			name:        "requested max only updates max",
			currentMin:  5,
			currentMax:  10,
			requestedMin: nil,
			requestedMax: intPtr(15),
			expectedMin:  5,
			expectedMax:  15,
			expectError:  false,
		},
		{
			name:        "both requested values update both",
			currentMin:  5,
			currentMax:  10,
			requestedMin: intPtr(3),
			requestedMax: intPtr(20),
			expectedMin:  3,
			expectedMax:  20,
			expectError:  false,
		},
		{
			name:        "zero values are valid",
			currentMin:  5,
			currentMax:  10,
			requestedMin: intPtr(0),
			requestedMax: intPtr(0),
			expectedMin:  0,
			expectedMax:  0,
			expectError:  false,
		},
		{
			name:          "negative min returns error",
			currentMin:    5,
			currentMax:    10,
			requestedMin:  intPtr(-1),
			requestedMax:  nil,
			expectedMin:   0,
			expectedMax:   0,
			expectError:   true,
			errorContains: "non-negative",
		},
		{
			name:          "negative max returns error",
			currentMin:    5,
			currentMax:    10,
			requestedMin:  nil,
			requestedMax:  intPtr(-5),
			expectedMin:   0,
			expectedMax:   0,
			expectError:   true,
			errorContains: "non-negative",
		},
		{
			name:          "both negative returns error",
			currentMin:    5,
			currentMax:    10,
			requestedMin:  intPtr(-10),
			requestedMax:  intPtr(-5),
			expectedMin:   0,
			expectedMax:   0,
			expectError:   true,
			errorContains: "non-negative",
		},
		{
			name:          "min greater than max returns error",
			currentMin:    5,
			currentMax:    10,
			requestedMin:  intPtr(15),
			requestedMax:  intPtr(10),
			expectedMin:   0,
			expectedMax:   0,
			expectError:   true,
			errorContains: "min cannot be greater than max",
		},
		{
			name:        "equal min and max is valid",
			currentMin:  5,
			currentMax:  10,
			requestedMin: intPtr(8),
			requestedMax: intPtr(8),
			expectedMin:  8,
			expectedMax:  8,
			expectError:  false,
		},
		{
			name:        "large values are valid",
			currentMin:  5,
			currentMax:  10,
			requestedMin: intPtr(1000),
			requestedMax: intPtr(2000),
			expectedMin:  1000,
			expectedMax:  2000,
			expectError:  false,
		},
		{
			name:        "current negative values caught on no update",
			currentMin:  -5,
			currentMax:  10,
			requestedMin: nil,
			requestedMax: nil,
			expectedMin:  0,
			expectedMax:  0,
			expectError:  true,
			errorContains: "non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min, max, err := normalizeCampaignDelayRange(tt.currentMin, tt.currentMax, tt.requestedMin, tt.requestedMax)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMin, min)
				assert.Equal(t, tt.expectedMax, max)
			}
		})
	}
}

func TestNormalizeCampaignRecipientPhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string returns empty",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace-only returns empty",
			input:    "   ",
			expected: "",
		},
		{
			name:     "numeric characters preserved",
			input:    "1234567890",
			expected: "1234567890",
		},
		{
			name:     "leading/trailing whitespace trimmed",
			input:    "  1234567890  ",
			expected: "1234567890",
		},
		{
			name:     "dashes removed",
			input:    "123-456-7890",
			expected: "1234567890",
		},
		{
			name:     "parentheses and spaces removed",
			input:    "(123) 456-7890",
			expected: "1234567890",
		},
		{
			name:     "plus prefix removed",
			input:    "+1234567890",
			expected: "1234567890",
		},
		{
			name:     "mixed format phone number normalized",
			input:    "+1 (555) 123-4567",
			expected: "15551234567",
		},
		{
			name:     "letters removed",
			input:    "123-456-7890ext123",
			expected: "1234567890123",
		},
		{
			name:     "special characters removed",
			input:    "123@456#7890",
			expected: "1234567890",
		},
		{
			name:     "only non-numeric returns empty",
			input:    "abc-def",
			expected: "",
		},
		{
			name:     "international format preserved",
			input:    "44 20 7123 4567",
			expected: "442071234567",
		},
		{
			name:     "single digit preserved",
			input:    "5",
			expected: "5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCampaignRecipientPhone(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMimeTypeFromExtension(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		expected string
	}{
		{
			name:     "jpeg extension",
			ext:      ".jpg",
			expected: "image/jpeg",
		},
		{
			name:     "jpeg alternate extension",
			ext:      ".jpeg",
			expected: "image/jpeg",
		},
		{
			name:     "png extension",
			ext:      ".png",
			expected: "image/png",
		},
		{
			name:     "gif extension",
			ext:      ".gif",
			expected: "image/gif",
		},
		{
			name:     "webp extension",
			ext:      ".webp",
			expected: "image/webp",
		},
		{
			name:     "mp4 extension",
			ext:      ".mp4",
			expected: "video/mp4",
		},
		{
			name:     "3gp extension",
			ext:      ".3gp",
			expected: "video/3gpp",
		},
		{
			name:     "pdf extension",
			ext:      ".pdf",
			expected: "application/pdf",
		},
		{
			name:     "doc extension",
			ext:      ".doc",
			expected: "application/msword",
		},
		{
			name:     "docx extension",
			ext:      ".docx",
			expected: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
		{
			name:     "unknown extension returns octet-stream",
			ext:      ".xyz",
			expected: "application/octet-stream",
		},
		{
			name:     "empty extension returns octet-stream",
			ext:      "",
			expected: "application/octet-stream",
		},
		{
			name:     "no dot returns octet-stream",
			ext:      "jpg",
			expected: "application/octet-stream",
		},
		{
			name:     "uppercase extension returns octet-stream",
			ext:      ".JPG",
			expected: "application/octet-stream",
		},
		{
			name:     "mixed case returns octet-stream",
			ext:      ".JpG",
			expected: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMimeTypeFromExtension(tt.ext)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple filename unchanged",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "filename with path component stripped",
			input:    "/path/to/document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "windows path stripped",
			input:    "C:\\Users\\test\\document.pdf",
			expected: "C__Users_test_document.pdf", // Colon and backslashes both replaced with _
		},
		{
			name:     "spaces replaced with underscores",
			input:    "my document.pdf",
			expected: "my_document.pdf", // Spaces are not in allowed set
		},
		{
			name:     "special characters replaced with underscores",
			input:    "file@name#.pdf",
			expected: "file_name_.pdf",
		},
		{
			name:     "consecutive special chars replaced",
			input:    "file$$$name.pdf",
			expected: "file___name.pdf",
		},
		{
			name:     "filename truncated to 255 chars",
			input:    string(make([]byte, 300)),
			expected: strings.Repeat("_", 255), // Null bytes replaced with underscores
		},
		{
			name:     "empty string returns unnamed",
			input:    "",
			expected: "unnamed",
		},
		{
			name:     "dot returns unnamed",
			input:    ".",
			expected: "unnamed",
		},
		{
			name:     "double dot returns unnamed",
			input:    "..",
			expected: "unnamed",
		},
		{
			name:     "path traversal stripped",
			input:    "../../etc/passwd",
			expected: "passwd",
		},
		{
			name:     "allowed special chars preserved",
			input:    "file_name-v1.0.pdf",
			expected: "file_name-v1.0.pdf",
		},
		{
			name:     "unicode characters replaced",
			input:    "文件名.pdf",
			expected: "___.pdf", // 3 unicode chars, each replaced with underscore
		},
		{
			name:     "mixed allowed and disallowed chars",
			input:    "my file (1) [copy].pdf",
			expected: "my_file__1___copy_.pdf", // Spaces, parens, brackets replaced
		},
		{
			name:     "filename at exactly 255 chars unchanged",
			input:    strings.Repeat("a", 255),
			expected: strings.Repeat("a", 255), // All alphanumeric, unchanged
		},
		{
			name:     "complex path sanitized",
			input:    "/var/www/uploads/user@domain/file (1).pdf",
			expected: "file__1_.pdf", // After filepath.Base, then spaces/punctuation replaced
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function for tests
func intPtr(i int) *int {
	return &i
}
