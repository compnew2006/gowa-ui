package models

import "strings"

// ChatStatus describes the lifecycle state of a conversation/contact.
type ChatStatus string

const (
	ChatStatusPending ChatStatus = "pending"
	ChatStatusOpen    ChatStatus = "open"
	ChatStatusClosed  ChatStatus = "closed"
)

// NormalizeChatStatus coerces unknown/empty values to a safe default.
func NormalizeChatStatus(raw string) ChatStatus {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(ChatStatusOpen):
		return ChatStatusOpen
	case string(ChatStatusClosed):
		return ChatStatusClosed
	case string(ChatStatusPending):
		fallthrough
	default:
		return ChatStatusPending
	}
}

func (s ChatStatus) String() string {
	return string(NormalizeChatStatus(string(s)))
}

// EffectiveStatus keeps status/assignment backward compatible for legacy rows.
func (c Contact) EffectiveStatus() ChatStatus {
	status := NormalizeChatStatus(string(c.Status))
	if status == ChatStatusClosed {
		return ChatStatusClosed
	}
	if c.AssignedUserID != nil {
		return ChatStatusOpen
	}
	return ChatStatusPending
}
