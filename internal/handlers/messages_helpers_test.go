package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestResolvedActorType(t *testing.T) {
	userID := uuid.New()
	systemID := uuid.New()

	tests := []struct {
		name     string
		opts     MessageSendOptions
		expected MessageActorType
	}{
		{
			name: "explicit user actor type",
			opts: MessageSendOptions{
				ActorType: MessageActorUser,
			},
			expected: MessageActorUser,
		},
		{
			name: "explicit system actor type",
			opts: MessageSendOptions{
				ActorType: MessageActorSystem,
			},
			expected: MessageActorSystem,
		},
		{
			name: "explicit worker actor type",
			opts: MessageSendOptions{
				ActorType: MessageActorWorker,
			},
			expected: MessageActorWorker,
		},
		{
			name: "explicit auto campaign actor type",
			opts: MessageSendOptions{
				ActorType: MessageActorAutoCampaign,
			},
			expected: MessageActorAutoCampaign,
		},
		{
			name: "unknown actor type with user ID returns user",
			opts: MessageSendOptions{
				ActorType:    MessageActorType("unknown"),
				SentByUserID: &userID,
			},
			expected: MessageActorUser,
		},
		{
			name: "unknown actor type without user ID returns system",
			opts: MessageSendOptions{
				ActorType: MessageActorType("unknown"),
			},
			expected: MessageActorSystem,
		},
		{
			name: "empty actor type with user ID returns user",
			opts: MessageSendOptions{
				ActorType:    MessageActorType(""),
				SentByUserID: &userID,
			},
			expected: MessageActorUser,
		},
		{
			name: "empty actor type without user ID returns system",
			opts: MessageSendOptions{
				ActorType: MessageActorType(""),
			},
			expected: MessageActorSystem,
		},
		{
			name: "nil user ID with system actor type returns system",
			opts: MessageSendOptions{
				ActorType:    MessageActorSystem,
				SentByUserID: nil,
			},
			expected: MessageActorSystem,
		},
		{
			name: "user actor type overrides user ID presence",
			opts: MessageSendOptions{
				ActorType:    MessageActorUser,
				SentByUserID: &systemID,
			},
			expected: MessageActorUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.opts.resolvedActorType()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatAgentMessageContent(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		content   string
		expected  string
	}{
		{
			name:      "both empty returns empty",
			agentName: "",
			content:   "",
			expected:  "",
		},
		{
			name:      "empty agent name returns content",
			agentName: "",
			content:   "Hello world",
			expected:  "Hello world",
		},
		{
			name:      "empty content returns empty",
			agentName: "Agent",
			content:   "",
			expected:  "",
		},
		{
			name:      "whitespace-only agent name returns content",
			agentName: "   ",
			content:   "Hello world",
			expected:  "Hello world",
		},
		{
			name:      "whitespace-only content returns empty",
			agentName: "Agent",
			content:   "   ",
			expected:  "",
		},
		{
			name:      "both trimmed and formatted",
			agentName: "Agent",
			content:   "Hello world",
			expected:  "Agent : Hello world",
		},
		{
			name:      "agent name with extra spaces trimmed",
			agentName: "  Agent  ",
			content:   "Hello world",
			expected:  "Agent : Hello world",
		},
		{
			name:      "content with extra spaces trimmed",
			agentName: "Agent",
			content:   "  Hello world  ",
			expected:  "Agent : Hello world",
		},
		{
			name:      "content with existing prefix unchanged",
			agentName: "Agent",
			content:   "Agent : Hello world",
			expected:  "Agent : Hello world",
		},
		{
			name:      "content with case-insensitive prefix unchanged",
			agentName: "Agent",
			content:   "AGENT : Hello world",
			expected:  "AGENT : Hello world",
		},
		{
			name:      "content with prefix and extra spaces unchanged",
			agentName: "Agent",
			content:   "Agent :   Hello world",
			expected:  "Agent :   Hello world",
		},
		{
			name:      "multi-word agent name formatted",
			agentName: "Support Agent",
			content:   "Hello",
			expected:  "Support Agent : Hello",
		},
		{
			name:      "special characters in agent name preserved",
			agentName: "Agent-Smith",
			content:   "Hello",
			expected:  "Agent-Smith : Hello",
		},
		{
			name:      "content with leading colon gets prefix",
			agentName: "Agent",
			content:   ": Test message",
			expected:  "Agent : : Test message",
		},
		{
			name:      "content with colon but no prefix gets prefix",
			agentName: "Agent",
			content:   "Different : Test",
			expected:  "Agent : Different : Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAgentMessageContent(tt.agentName, tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContentHasAgentPrefix(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		agentName string
		expected  bool
	}{
		{
			name:      "empty content returns false",
			content:   "",
			agentName: "Agent",
			expected:  false,
		},
		{
			name:      "empty agent name returns false",
			content:   "Agent : Hello",
			agentName: "",
			expected:  false,
		},
		{
			name:      "both empty returns false",
			content:   "",
			agentName: "",
			expected:  false,
		},
		{
			name:      "content shorter than agent name returns false",
			content:   "Ag",
			agentName: "Agent",
			expected:  false,
		},
		{
			name:      "content same length as agent name returns false",
			content:   "Agent",
			agentName: "Agent",
			expected:  false,
		},
		{
			name:      "matching prefix with colon returns true",
			content:   "Agent : Hello",
			agentName: "Agent",
			expected:  true,
		},
		{
			name:      "matching prefix with colon and spaces returns true",
			content:   "Agent   :   Hello",
			agentName: "Agent",
			expected:  true,
		},
		{
			name:      "matching prefix with tab and colon returns true",
			content:   "Agent\t: Hello",
			agentName: "Agent",
			expected:  true,
		},
		{
			name:      "case-insensitive match returns true",
			content:   "AGENT : Hello",
			agentName: "Agent",
			expected:  true,
		},
		{
			name:      "no match returns false",
			content:   "Different : Hello",
			agentName: "Agent",
			expected:  false,
		},
		{
			name:      "matching prefix without colon returns false",
			content:   "Agent Hello",
			agentName: "Agent",
			expected:  false,
		},
		{
			name:      "matching prefix with different separator returns false",
			content:   "Agent - Hello",
			agentName: "Agent",
			expected:  false,
		},
		{
			name:      "colon at beginning only returns false",
			content:   "Agent:Hello",
			agentName: "Agent",
			expected:  true, // No space after agent name, but colon is there
		},
		{
			name:      "multi-word agent name with colon returns true",
			content:   "Support Agent : Hello",
			agentName: "Support Agent",
			expected:  true,
		},
		{
			name:      "special characters in agent name match",
			content:   "Agent-Smith : Hello",
			agentName: "Agent-Smith",
			expected:  true,
		},
		{
			name:      "partial prefix match returns false",
			content:   "AgentX : Hello",
			agentName: "Agent",
			expected:  false,
		},
		{
			name:      "content with only agent name returns false",
			content:   "Agent ",
			agentName: "Agent",
			expected:  false, // No colon after prefix
		},
		{
			name:      "unicode characters in agent name match",
			content:   "代理 : Hello",
			agentName: "代理",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contentHasAgentPrefix(tt.content, tt.agentName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveProviderMediaRef(t *testing.T) {
	tests := []struct {
		name     string
		req      OutgoingMessageRequest
		expected string
	}{
		{
			name: "media URL present returns media URL",
			req: OutgoingMessageRequest{
				MediaURL: "https://example.com/image.jpg",
				MediaID:  "media123",
			},
			expected: "https://example.com/image.jpg",
		},
		{
			name: "media URL empty returns media ID",
			req: OutgoingMessageRequest{
				MediaURL: "",
				MediaID:  "media123",
			},
			expected: "media123",
		},
		{
			name: "both media URL and ID present prefers URL",
			req: OutgoingMessageRequest{
				MediaURL: "https://example.com/image.jpg",
				MediaID:  "media123",
			},
			expected: "https://example.com/image.jpg",
		},
		{
			name: "both empty returns empty",
			req: OutgoingMessageRequest{
				MediaURL: "",
				MediaID:  "",
			},
			expected: "",
		},
		{
			name: "whitespace media URL treated as present",
			req: OutgoingMessageRequest{
				MediaURL: "  ",
				MediaID:  "media123",
			},
			expected: "  ",
		},
		{
			name: "media URL with special characters preserved",
			req: OutgoingMessageRequest{
				MediaURL: "https://example.com/image.jpg?token=abc123&size=large",
				MediaID:  "media123",
			},
			expected: "https://example.com/image.jpg?token=abc123&size=large",
		},
		{
			name: "relative media URL returned as-is",
			req: OutgoingMessageRequest{
				MediaURL: "/uploads/image.jpg",
				MediaID:  "media123",
			},
			expected: "/uploads/image.jpg",
		},
		{
			name: "S3 URL returned",
			req: OutgoingMessageRequest{
				MediaURL: "s3://bucket/path/to/image.jpg",
				MediaID:  "media123",
			},
			expected: "s3://bucket/path/to/image.jpg",
		},
		{
			name: "data URI returned",
			req: OutgoingMessageRequest{
				MediaURL: "data:image/jpeg;base64,/9j/4AAQSkZJRg...",
				MediaID:  "media123",
			},
			expected: "data:image/jpeg;base64,/9j/4AAQSkZJRg...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveProviderMediaRef(tt.req)
			assert.Equal(t, tt.expected, result)
		})
	}
}
