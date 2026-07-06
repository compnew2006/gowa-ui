package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
)

func TestNormalizeWhatsAppMediaMIME(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "normalizes charset suffix", input: "image/jpeg; charset=UTF-8", expected: "image/jpeg"},
		{name: "normalizes casing and whitespace", input: "  APPLICATION/PDF  ", expected: "application/pdf"},
		{name: "returns empty for empty input", input: "", expected: ""},
		{name: "falls back for malformed media type", input: "audio/mpeg;bad", expected: "audio/mpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeWhatsAppMediaMIME(tt.input)
			if got != tt.expected {
				t.Fatalf("normalizeWhatsAppMediaMIME(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolveWhatsAppMediaMIME(t *testing.T) {
	jpegBytes := []byte{0xff, 0xd8, 0xff, 0xdb, 0x00, 0x43, 0x00}

	tests := []struct {
		name            string
		partContentType string
		filename        string
		fileData        []byte
		expected        string
	}{
		{
			name:            "prefers sniffed content type over multipart header",
			partContentType: "image/png; charset=binary",
			filename:        "voice.mp3",
			fileData:        jpegBytes,
			expected:        "image/jpeg",
		},
		{
			name:            "uses multipart content type when sniffing is unavailable",
			partContentType: "image/png; charset=binary",
			filename:        "voice.mp3",
			fileData:        nil,
			expected:        "image/png",
		},
		{
			name:            "falls back to extension when multipart type is generic",
			partContentType: "application/octet-stream",
			filename:        "photo.jpeg",
			fileData:        nil,
			expected:        "image/jpeg",
		},
		{
			name:            "falls back to sniffing when header and extension are unavailable",
			partContentType: "",
			filename:        "file.unknown",
			fileData:        jpegBytes,
			expected:        "image/jpeg",
		},
		{
			name:            "falls back to octet-stream when all sources unknown",
			partContentType: "",
			filename:        "file.unknown",
			fileData:        nil,
			expected:        "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveWhatsAppMediaMIME(tt.partContentType, tt.filename, tt.fileData)
			if got != tt.expected {
				t.Fatalf("resolveWhatsAppMediaMIME() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDeriveWhatsAppMediaMessageType(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected models.MessageType
	}{
		{name: "image mime", mimeType: "image/webp", expected: models.MessageTypeImage},
		{name: "video mime", mimeType: "video/mp4", expected: models.MessageTypeVideo},
		{name: "audio mime", mimeType: "audio/ogg", expected: models.MessageTypeAudio},
		{name: "unknown mime maps to document", mimeType: "application/zip", expected: models.MessageTypeDocument},
		{name: "empty mime maps to document", mimeType: "", expected: models.MessageTypeDocument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveWhatsAppMediaMessageType(tt.mimeType)
			if got != tt.expected {
				t.Fatalf("deriveWhatsAppMediaMessageType(%q) = %q, want %q", tt.mimeType, got, tt.expected)
			}
		})
	}
}

func TestWhatsAppMediaMaxSizeBytes(t *testing.T) {
	tests := []struct {
		name        string
		messageType models.MessageType
		expected    int64
	}{
		{name: "image limit", messageType: models.MessageTypeImage, expected: 5 * 1024 * 1024},
		{name: "video limit", messageType: models.MessageTypeVideo, expected: 16 * 1024 * 1024},
		{name: "audio limit", messageType: models.MessageTypeAudio, expected: 16 * 1024 * 1024},
		{name: "document limit", messageType: models.MessageTypeDocument, expected: 100 * 1024 * 1024},
		{name: "unknown type defaults to document limit", messageType: models.MessageTypeText, expected: 100 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := whatsappMediaMaxSizeBytes(tt.messageType)
			if got != tt.expected {
				t.Fatalf("whatsappMediaMaxSizeBytes(%q) = %d, want %d", tt.messageType, got, tt.expected)
			}
		})
	}
}
