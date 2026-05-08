package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
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
		name                string
		message             models.Message
		hasCompanionByWAMID map[string]bool
		expected            bool
	}{
		{
			name: "placeholder message with companion WAMID",
			message: models.Message{
				MessageType:       models.MessageTypeText,
				Content:           unsupportedMessageBody,
				WhatsAppMessageID: "msg123",
				Metadata:          models.JSONB{},
			},
			hasCompanionByWAMID: map[string]bool{"msg123": true},
			expected:            true,
		},
		{
			name: "placeholder message without companion WAMID",
			message: models.Message{
				MessageType:       models.MessageTypeText,
				Content:           unsupportedMessageBody,
				WhatsAppMessageID: "msg123",
				Metadata:          models.JSONB{},
			},
			hasCompanionByWAMID: map[string]bool{"msg456": true},
			expected:            false,
		},
		{
			name: "placeholder message with empty WAMID",
			message: models.Message{
				MessageType:       models.MessageTypeText,
				Content:           unsupportedMessageBody,
				WhatsAppMessageID: "",
				Metadata:          models.JSONB{},
			},
			hasCompanionByWAMID: map[string]bool{},
			expected:            false,
		},
		{
			name: "placeholder message with revoked metadata",
			message: models.Message{
				MessageType:       models.MessageTypeText,
				Content:           unsupportedMessageBody,
				WhatsAppMessageID: "msg123",
				Metadata:          models.JSONB{"revoked": true},
			},
			hasCompanionByWAMID: map[string]bool{"msg123": true},
			expected:            false,
		},
		{
			name: "regular message with companion WAMID",
			message: models.Message{
				MessageType:       models.MessageTypeText,
				Content:           "Hello world",
				WhatsAppMessageID: "msg123",
				Metadata:          models.JSONB{},
			},
			hasCompanionByWAMID: map[string]bool{"msg123": true},
			expected:            false,
		},
		{
			name: "placeholder message with nil metadata",
			message: models.Message{
				MessageType:       models.MessageTypeText,
				Content:           unsupportedMessageBody,
				WhatsAppMessageID: "msg123",
				Metadata:          nil,
			},
			hasCompanionByWAMID: map[string]bool{"msg123": true},
			expected:            true,
		},
		{
			name: "placeholder message with whitespace-only WAMID",
			message: models.Message{
				MessageType:       models.MessageTypeText,
				Content:           unsupportedMessageBody,
				WhatsAppMessageID: "  ",
				Metadata:          models.JSONB{},
			},
			hasCompanionByWAMID: map[string]bool{},
			expected:            false,
		},
		{
			name: "empty companion map",
			message: models.Message{
				MessageType:       models.MessageTypeText,
				Content:           unsupportedMessageBody,
				WhatsAppMessageID: "msg123",
				Metadata:          models.JSONB{},
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

func TestStringifyInstanceID(t *testing.T) {
	uuid1 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name     string
		input    *uuid.UUID
		expected *string
	}{
		{
			name:     "nil returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "valid UUID returns pointer to string representation",
			input:    &uuid1,
			expected: strPtr(uuid1.String()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringifyInstanceID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConversationUnreadKey(t *testing.T) {
	uuid1 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name         string
		conversation string
		instanceID   *uuid.UUID
		expected     string
	}{
		{
			name:         "nil instanceID appends pipe",
			conversation: "conv-123",
			instanceID:   nil,
			expected:     "conv-123|",
		},
		{
			name:         "with instanceID appends pipe uuid",
			conversation: "conv-456",
			instanceID:   &uuid1,
			expected:     "conv-456|" + uuid1.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, conversationUnreadKey(tt.conversation, tt.instanceID))
		})
	}
}

func TestCloneJSONB(t *testing.T) {
	t.Run("nil returns empty not nil", func(t *testing.T) {
		result := cloneJSONB(nil)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("populated returns copy", func(t *testing.T) {
		original := models.JSONB{"key1": "val1", "key2": 42}
		result := cloneJSONB(original)
		if len(result) != len(original) {
			t.Fatalf("length mismatch: got %d, want %d", len(result), len(original))
		}
		for k, v := range original {
			if result[k] != v {
				t.Fatalf("key %q: got %v, want %v", k, result[k], v)
			}
		}
		_ = result
		_ = original
	})

	t.Run("modifying copy does not affect original", func(t *testing.T) {
		original := models.JSONB{"key1": "val1"}
		copy := cloneJSONB(original)
		copy["key1"] = "modified"
		copy["key2"] = "new"
		assert.Equal(t, "val1", original["key1"])
		assert.NotContains(t, original, "key2")
	})
}

func TestContactAvatarURL(t *testing.T) {
	tests := []struct {
		name     string
		metadata models.JSONB
		expected string
	}{
		{
			name:     "nil returns empty",
			metadata: nil,
			expected: "",
		},
		{
			name:     "missing key returns empty",
			metadata: models.JSONB{"other": "val"},
			expected: "",
		},
		{
			name:     "non-string value returns empty",
			metadata: models.JSONB{"avatar_url": 123},
			expected: "",
		},
		{
			name:     "valid URL returns trimmed value",
			metadata: models.JSONB{"avatar_url": "https://example.com/avatar.jpg"},
			expected: "https://example.com/avatar.jpg",
		},
		{
			name:     "whitespace-padded URL trimmed",
			metadata: models.JSONB{"avatar_url": "  https://example.com/avatar.jpg  "},
			expected: "https://example.com/avatar.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, contactAvatarURL(tt.metadata))
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestApplyDirectContactPhoneFromConversation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		contact        *models.Contact
		conversationID string
		assertPhone    string
	}{
		{
			name:           "nil contact does not panic",
			contact:        nil,
			conversationID: "1234567890@s.whatsapp.net",
			assertPhone:    "",
		},
		{
			name: "group contact does not change phone",
			contact: &models.Contact{
				PhoneNumber: "123456789@g.us",
				Metadata:    models.JSONB{"is_group_chat": true},
			},
			conversationID: "1234567890@s.whatsapp.net",
			assertPhone:    "123456789@g.us",
		},
		{
			name: "channel contact does not change phone",
			contact: &models.Contact{
				PhoneNumber: "123456789@newsletter",
				Metadata:    models.JSONB{"is_channel": true},
			},
			conversationID: "1234567890@s.whatsapp.net",
			assertPhone:    "123456789@newsletter",
		},
		{
			name: "direct conversation sets phone number",
			contact: &models.Contact{
				PhoneNumber: "",
			},
			conversationID: "1234567890@s.whatsapp.net",
			assertPhone:    "1234567890",
		},
		{
			name: "direct conversation overwrites existing phone",
			contact: &models.Contact{
				PhoneNumber: "old_phone",
			},
			conversationID: "9876543210@s.whatsapp.net",
			assertPhone:    "9876543210",
		},
		{
			name: "conversation ID without suffix does not change phone",
			contact: &models.Contact{
				PhoneNumber: "existing",
			},
			conversationID: "plain_text_id",
			assertPhone:    "existing",
		},
		{
			name: "empty conversation ID does not change phone",
			contact: &models.Contact{
				PhoneNumber: "existing",
			},
			conversationID: "",
			assertPhone:    "existing",
		},
		{
			name: "whitespace conversation ID does not change phone",
			contact: &models.Contact{
				PhoneNumber: "existing",
			},
			conversationID: "   ",
			assertPhone:    "existing",
		},
		{
			name: "group JID suffix on phone is treated as group contact",
			contact: &models.Contact{
				PhoneNumber: "123456789@g.us",
			},
			conversationID: "1234567890@s.whatsapp.net",
			assertPhone:    "123456789@g.us",
		},
		{
			name: "channel JID suffix on phone is treated as channel contact",
			contact: &models.Contact{
				PhoneNumber: "123456789@newsletter",
			},
			conversationID: "1234567890@s.whatsapp.net",
			assertPhone:    "123456789@newsletter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			applyDirectContactPhoneFromConversation(tt.contact, tt.conversationID)
			if tt.contact == nil {
				return
			}
			assert.Equal(t, tt.assertPhone, tt.contact.PhoneNumber)
		})
	}
}
