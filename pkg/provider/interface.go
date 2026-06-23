package provider

import (
	"context"

	"github.com/google/uuid"
)

// MessageProvider defines the interface for sending messages and managing interactions
// across different WhatsApp providers (Meta Cloud API, whatsmeow, etc.)
type MessageProvider interface {
	// SendText sends a text message
	SendText(ctx context.Context, instanceID string, to string, text string) (string, error)

	// SendImage sends an image message
	SendImage(ctx context.Context, instanceID string, to string, imageURL string, caption string) (string, error)

	// SendDocument sends a document message
	SendDocument(ctx context.Context, instanceID string, to string, docURL string, filename string, caption string) (string, error)

	// SendVideo sends a video message
	SendVideo(ctx context.Context, instanceID string, to string, videoURL string, caption string) (string, error)

	// SendAudio sends an audio message
	SendAudio(ctx context.Context, instanceID string, to string, audioURL string) (string, error)

	// MarkRead marks a message as read
	MarkRead(ctx context.Context, instanceID string, messageID string) error

	// SendReaction sends an emoji reaction to a message
	SendReaction(ctx context.Context, instanceID string, messageID string, emoji string) error

	// RevokeMessage deletes an outgoing message from WhatsApp
	RevokeMessage(ctx context.Context, instanceID string, messageID string) error

	// GetMediaURL retrieves a temporary URL for a media ID (Meta specific usually, but useful abstraction)
	GetMediaURL(ctx context.Context, instanceID string, mediaID string) (string, error)

	// DownloadMedia downloads media bytes from a URL
	DownloadMedia(ctx context.Context, instanceID string, mediaURL string) ([]byte, error)

	// UploadMedia uploads media bytes and returns a handle/ID/URL
	UploadMedia(ctx context.Context, instanceID string, mediaType string, data []byte) (string, error)
}

// ReplyProvider is an optional extension to MessageProvider for adapters that
// support quoted replies (reply-to-message context). Callers should type-assert
// the MessageProvider to check if this is supported.
type ReplyProvider interface {
	// SendTextReply sends a text message as a reply to a specific message
	SendTextReply(ctx context.Context, instanceID string, to string, text string, replyToMsgID string) (string, error)
}

// PollProvider is an optional extension to MessageProvider for adapters that
// support sending native WhatsApp polls. Only whatsmeow implements this;
// Meta Cloud API does not. Callers should type-assert to check support.
type PollProvider interface {
	SendPoll(ctx context.Context, instanceID string, to string, question string, options []string, maxSelections int) (string, error)
}

// PollVoteTarget identifies the poll to vote on.
type PollVoteTarget struct {
	InstanceID             uuid.UUID
	OrgID                  uuid.UUID
	OriginalPollWhatsAppID string
}

// PollVoter is an optional extension for adapters that support voting on
// existing WhatsApp polls. Only whatsmeow implements this.
// Callers should type-assert to check support.
type PollVoter interface {
	SendPollVote(ctx context.Context, target PollVoteTarget, selectedOptions []string) (string, error)
}

// GroupInfo holds metadata for a WhatsApp group.
type GroupInfo struct {
	JID              string
	Name             string
	ParticipantCount int
}

// GroupProvider is an optional extension to MessageProvider for adapters that
// support WhatsApp group operations (e.g. whatsmeow). The Meta Cloud API does
// not implement this interface. Callers should type-assert to check support.
type GroupProvider interface {
	GetGroups(ctx context.Context, instanceID string) ([]GroupInfo, error)
	VerifyGroupMembership(ctx context.Context, instanceID string, groupJID string) (*GroupInfo, error)
}

// GroupParticipant holds info about a single group participant.
type GroupParticipant struct {
	JID          string
	PhoneNumber  string
	IsAdmin      bool
	IsSuperAdmin bool
}

// GroupParticipantProvider is an optional extension for adapters that support
// managing group participants (add/remove/promote/demote). Only whatsmeow
// implements this; Meta Cloud API does not.
type GroupParticipantProvider interface {
	// AddGroupParticipants adds one or more participants to a group.
	AddGroupParticipants(ctx context.Context, instanceID string, groupJID string, participantJIDs []string) ([]GroupParticipant, error)

	// RemoveGroupParticipants removes one or more participants from a group.
	RemoveGroupParticipants(ctx context.Context, instanceID string, groupJID string, participantJIDs []string) ([]GroupParticipant, error)

	// PromoteGroupParticipants promotes participants to group admin.
	PromoteGroupParticipants(ctx context.Context, instanceID string, groupJID string, participantJIDs []string) ([]GroupParticipant, error)

	// DemoteGroupParticipants demotes participants from group admin.
	DemoteGroupParticipants(ctx context.Context, instanceID string, groupJID string, participantJIDs []string) ([]GroupParticipant, error)

	// GetGroupParticipants returns all participants of a group.
	GetGroupParticipants(ctx context.Context, instanceID string, groupJID string) ([]GroupParticipant, error)
}

// JoinGroupProvider is an optional extension for adapters that support joining
// WhatsApp groups via invite link. Only the whatsmeow adapter implements this.
type JoinGroupProvider interface {
	// JoinGroupWithLink joins a WhatsApp group using an invite link/code.
	// Returns the group JID on success.
	JoinGroupWithLink(ctx context.Context, instanceID string, inviteLink string) (string, error)
}

// GroupInfoProvider is an optional extension for adapters that can fetch
// group metadata from an invite link without joining. Only whatsmeow implements this.
type GroupInfoProvider interface {
	// GetGroupInfoFromLink fetches group metadata from an invite link code.
	// Does NOT join the group — returns preview info only.
	GetGroupInfoFromLink(ctx context.Context, instanceID string, inviteLink string) (*GroupInfo, error)
}

// SessionResetter is an optional extension for adapters that can clear the
// Signal Protocol session state for a single recipient on demand. Only the
// whatsmeow adapter implements this; the Meta Cloud API does not.
// Callers should type-assert to check support.
//
// It exists so the send queue can recover from WhatsApp's "server returned
// error 400" stanza ack, which signals a desynced recipient session (usually
// a PN<->LID migration gap). Resetting the recipient's sessions forces a
// clean prekey rebuild on the next send attempt.
type SessionResetter interface {
	// ResetRecipientSession clears the recipient's Signal sessions for the
	// given instance and target (phone number or JID). It is scoped to that
	// recipient only and never affects other chats. Implementations must not
	// return fatal errors for missing clients or unresolvable mappings; a
	// reset is always best-effort.
	ResetRecipientSession(ctx context.Context, instanceID string, to string) error
}
