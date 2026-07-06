package handlers

import (
	"testing"
)

func TestGetExtensionFromMimeType(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected string
	}{
		{"JPEG image", "image/jpeg", ".jpg"},
		{"PNG image", "image/png", ".png"},
		{"GIF image", "image/gif", ".gif"},
		{"WebP image", "image/webp", ".webp"},
		{"MP4 video", "video/mp4", ".mp4"},
		{"PDF document", "application/pdf", ".pdf"},
		{"Plain text", "text/plain", ".txt"},
		{"Unknown mime type", "application/unknown", ""},
		{"Empty mime type", "", ""},
		{"Mime type with charset", "text/plain; charset=utf-8", ".txt"},
		{"Fully uppercase", "IMAGE/JPEG", ""},
		{"Mixed case", "Image/Png", ""},
		{"Trailing whitespace", "image/jpeg ", ".jpg"},
		{"Leading whitespace", " image/png", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getExtensionFromMimeType(tt.mimeType)
			if got != tt.expected {
				t.Errorf("getExtensionFromMimeType(%q) = %q; want %q", tt.mimeType, got, tt.expected)
			}
		})
	}
}
