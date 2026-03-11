package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeDeletedMessageBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "legacy deleted body case insensitive",
			input:    "This message was deleted",
			expected: deletedMessageBody,
		},
		{
			name:     "legacy deleted body uppercase",
			input:    "THIS MESSAGE WAS DELETED",
			expected: deletedMessageBody,
		},
		{
			name:     "legacy deleted body mixed case",
			input:    "ThIs MeSsAgE wAs DeLeTeD",
			expected: deletedMessageBody,
		},
		{
			name:     "legacy deleted body with leading/trailing spaces",
			input:    "  This message was deleted  ",
			expected: deletedMessageBody,
		},
		{
			name:     "already normalized deleted body",
			input:    deletedMessageBody,
			expected: deletedMessageBody,
		},
		{
			name:     "regular message unchanged",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "regular message with leading/trailing spaces",
			input:    "  Hello world  ",
			expected: "  Hello world  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeDeletedMessageBody(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAppendDeletedMessageCaption(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string returns deleted body",
			input:    "",
			expected: deletedMessageBody,
		},
		{
			name:     "deleted message body unchanged",
			input:    deletedMessageBody,
			expected: deletedMessageBody,
		},
		{
			name:     "legacy deleted body normalized",
			input:    legacyDeletedBody,
			expected: deletedMessageBody,
		},
		{
			name:     "unsupported message body replaced",
			input:    unsupportedMessageBody,
			expected: deletedMessageBody,
		},
		{
			name:     "content already containing deleted body unchanged",
			input:    deletedMessageBody + "\nSome text",
			expected: deletedMessageBody + "\nSome text",
		},
		{
			name:     "regular message gets deleted body appended",
			input:    "Hello world",
			expected: "Hello world\n" + deletedMessageBody,
		},
		{
			name:     "message with trailing newline",
			input:    "Hello world\n",
			expected: "Hello world\n" + deletedMessageBody,
		},
		{
			name:     "message with multiple trailing newlines",
			input:    "Hello world\n\n\n",
			expected: "Hello world\n" + deletedMessageBody,
		},
		{
			name:     "message with leading spaces",
			input:    "  Hello world",
			expected: "  Hello world\n" + deletedMessageBody,
		},
		{
			name:     "multiline message",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1\nLine 2\nLine 3\n" + deletedMessageBody,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendDeletedMessageCaption(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageMetadataBool(t *testing.T) {
	tests := []struct {
		name     string
		metadata models.JSONB
		key      string
		expected bool
	}{
		{
			name:     "nil metadata returns false",
			metadata: nil,
			key:      "test",
			expected: false,
		},
		{
			name:     "empty metadata returns false",
			metadata: models.JSONB{},
			key:      "test",
			expected: false,
		},
		{
			name:     "key not present returns false",
			metadata: models.JSONB{"other_key": true},
			key:      "test",
			expected: false,
		},
		{
			name:     "key present with true returns true",
			metadata: models.JSONB{"test": true},
			key:      "test",
			expected: true,
		},
		{
			name:     "key present with false returns false",
			metadata: models.JSONB{"test": false},
			key:      "test",
			expected: false,
		},
		{
			name:     "key present with non-boolean returns false",
			metadata: models.JSONB{"test": "true"},
			key:      "test",
			expected: false,
		},
		{
			name:     "key present with integer returns false",
			metadata: models.JSONB{"test": 1},
			key:      "test",
			expected: false,
		},
		{
			name:     "multiple keys, first one true",
			metadata: models.JSONB{"test": true, "other": false},
			key:      "test",
			expected: true,
		},
		{
			name:     "multiple keys, second one true",
			metadata: models.JSONB{"test": false, "other": true},
			key:      "other",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := messageMetadataBool(tt.metadata, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPlaceholderMessageBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "unsupported message body",
			input:    unsupportedMessageBody,
			expected: true,
		},
		{
			name:     "deleted message body",
			input:    deletedMessageBody,
			expected: true,
		},
		{
			name:     "legacy deleted body case insensitive",
			input:    "This message was deleted",
			expected: true,
		},
		{
			name:     "legacy deleted body uppercase",
			input:    "THIS MESSAGE WAS DELETED",
			expected: true,
		},
		{
			name:     "legacy deleted body with leading/trailing spaces",
			input:    "  This message was deleted  ",
			expected: true,
		},
		{
			name:     "regular text message",
			input:    "Hello world",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "regular message with leading/trailing spaces",
			input:    "  Hello world  ",
			expected: false,
		},
		{
			name:     "message similar but not exact legacy",
			input:    "This message was removed",
			expected: false,
		},
		{
			name:     "message with extra words",
			input:    "This message was deleted by user",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPlaceholderMessageBody(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPlaceholderTextMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  models.Message
		expected bool
	}{
		{
			name: "text message with placeholder content",
			message: models.Message{
				MessageType: models.MessageTypeText,
				Content:     unsupportedMessageBody,
			},
			expected: true,
		},
		{
			name: "text message with deleted content",
			message: models.Message{
				MessageType: models.MessageTypeText,
				Content:     deletedMessageBody,
			},
			expected: true,
		},
		{
			name: "text message with regular content",
			message: models.Message{
				MessageType: models.MessageTypeText,
				Content:     "Hello world",
			},
			expected: false,
		},
		{
			name: "image message with placeholder content",
			message: models.Message{
				MessageType: models.MessageTypeImage,
				Content:     unsupportedMessageBody,
			},
			expected: false,
		},
		{
			name: "image message with regular content",
			message: models.Message{
				MessageType: models.MessageTypeImage,
				Content:     "Some caption",
			},
			expected: false,
		},
		{
			name: "text message with empty content",
			message: models.Message{
				MessageType: models.MessageTypeText,
				Content:     "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPlaceholderTextMessage(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSyntheticPlaceholderMessage(t *testing.T) {
	tests := []struct {
		name                 string
		message              models.Message
		hasCompanionByWAMID  map[string]bool
		expected             bool
	}{
		{
			name: "placeholder message with companion WAMID",
			message: models.Message{
				MessageType:        models.MessageTypeText,
				Content:            unsupportedMessageBody,
				WhatsAppMessageID:  "msg123",
				Metadata:           models.JSONB{},
			},
			hasCompanionByWAMID: map[string]bool{"msg123": true},
			expected:            true,
		},
		{
			name: "placeholder message without companion WAMID",
			message: models.Message{
				MessageType:        models.MessageTypeText,
				Content:            unsupportedMessageBody,
				WhatsAppMessageID:  "msg123",
				Metadata:           models.JSONB{},
			},
			hasCompanionByWAMID: map[string]bool{"msg456": true},
			expected:            false,
		},
		{
			name: "placeholder message with empty WAMID",
			message: models.Message{
				MessageType:        models.MessageTypeText,
				Content:            unsupportedMessageBody,
				WhatsAppMessageID:  "",
				Metadata:           models.JSONB{},
			},
			hasCompanionByWAMID: map[string]bool{},
			expected:            false,
		},
		{
			name: "placeholder message with revoked metadata",
			message: models.Message{
				MessageType:        models.MessageTypeText,
				Content:            unsupportedMessageBody,
				WhatsAppMessageID:  "msg123",
				Metadata:           models.JSONB{"revoked": true},
			},
			hasCompanionByWAMID: map[string]bool{"msg123": true},
			expected:            false,
		},
		{
			name: "regular message with companion WAMID",
			message: models.Message{
				MessageType:        models.MessageTypeText,
				Content:            "Hello world",
				WhatsAppMessageID:  "msg123",
				Metadata:           models.JSONB{},
			},
			hasCompanionByWAMID: map[string]bool{"msg123": true},
			expected:            false,
		},
		{
			name: "placeholder message with nil metadata",
			message: models.Message{
				MessageType:        models.MessageTypeText,
				Content:            unsupportedMessageBody,
				WhatsAppMessageID:  "msg123",
				Metadata:           nil,
			},
			hasCompanionByWAMID: map[string]bool{"msg123": true},
			expected:            true,
		},
		{
			name: "placeholder message with whitespace-only WAMID",
			message: models.Message{
				MessageType:        models.MessageTypeText,
				Content:            unsupportedMessageBody,
				WhatsAppMessageID:  "  ",
				Metadata:           models.JSONB{},
			},
			hasCompanionByWAMID: map[string]bool{},
			expected:            false,
		},
		{
			name: "empty companion map",
			message: models.Message{
				MessageType:        models.MessageTypeText,
				Content:            unsupportedMessageBody,
				WhatsAppMessageID:  "msg123",
				Metadata:           models.JSONB{},
			},
			hasCompanionByWAMID: map[string]bool{},
			expected:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSyntheticPlaceholderMessage(tt.message, tt.hasCompanionByWAMID)
			assert.Equal(t, tt.expected, result)
		})
	}
}
