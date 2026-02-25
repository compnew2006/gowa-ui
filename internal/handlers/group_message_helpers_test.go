package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestIsGroupConversationID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		conversationID string
		want           bool
	}{
		{
			name:           "group jid suffix",
			conversationID: "12345@g.us",
			want:           true,
		},
		{
			name:           "group jid suffix with whitespace",
			conversationID: " 12345@g.us ",
			want:           true,
		},
		{
			name:           "non group direct jid",
			conversationID: "12345@s.whatsapp.net",
			want:           false,
		},
		{
			name:           "empty conversation id",
			conversationID: "",
			want:           false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isGroupConversationID(tt.conversationID))
		})
	}
}

func TestIsGroupMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message models.Message
		want    bool
	}{
		{
			name: "group conversation ID suffix",
			message: models.Message{
				ConversationID: "123456@g.us",
			},
			want: true,
		},
		{
			name: "group conversation ID suffix with surrounding whitespace",
			message: models.Message{
				ConversationID: "  123456@g.us  ",
				Metadata: models.JSONB{
					"is_group_chat": false,
					"is_group":      false,
				},
			},
			want: true,
		},
		{
			name: "non-group conversation with nil metadata",
			message: models.Message{
				ConversationID: "123456@s.whatsapp.net",
			},
			want: false,
		},
		{
			name: "metadata is_group_chat true",
			message: models.Message{
				ConversationID: "123456@s.whatsapp.net",
				Metadata: models.JSONB{
					"is_group_chat": true,
				},
			},
			want: true,
		},
		{
			name: "metadata is_group true",
			message: models.Message{
				ConversationID: "123456@s.whatsapp.net",
				Metadata: models.JSONB{
					"is_group": true,
				},
			},
			want: true,
		},
		{
			name: "metadata booleans false",
			message: models.Message{
				ConversationID: "123456@s.whatsapp.net",
				Metadata: models.JSONB{
					"is_group_chat": false,
					"is_group":      false,
				},
			},
			want: false,
		},
		{
			name: "metadata wrong types are ignored",
			message: models.Message{
				ConversationID: "123456@s.whatsapp.net",
				Metadata: models.JSONB{
					"is_group_chat": "true",
					"is_group":      1,
				},
			},
			want: false,
		},
		{
			name: "group jid takes precedence even when metadata keys are false",
			message: models.Message{
				ConversationID: "123456@g.us",
				Metadata: models.JSONB{
					"is_group_chat": false,
					"is_group":      false,
				},
			},
			want: true,
		},
		{
			name: "metadata first key wrong type but second key true",
			message: models.Message{
				ConversationID: "123456@s.whatsapp.net",
				Metadata: models.JSONB{
					"is_group_chat": "true",
					"is_group":      true,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isGroupMessage(tt.message))
		})
	}
}

func TestIsDirectIdentityValue(t *testing.T) {
	t.Parallel()

	assert.True(t, isDirectIdentityValue("15551234567@s.whatsapp.net"))
	assert.True(t, isDirectIdentityValue("149641526026409@lid"))
	assert.False(t, isDirectIdentityValue("John Doe"))
	assert.False(t, isDirectIdentityValue(""))
}

func TestFallbackDirectContactDisplayName(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"15551234567",
		fallbackDirectContactDisplayName("149641526026409@lid", "15551234567@s.whatsapp.net"),
	)
	assert.Equal(
		t,
		"966561853319",
		fallbackDirectContactDisplayName("966561853319", ""),
	)
	assert.Equal(
		t,
		"149641526026409",
		fallbackDirectContactDisplayName("149641526026409@lid", ""),
	)
}
