package whatsmeow

import (
	"context"
	"fmt"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"go.mau.fi/whatsmeow"
	waCommon "go.mau.fi/whatsmeow/proto/waCommon"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// MarkRead marks a message as read.
func (a *WhatsmeowAdapter) MarkRead(ctx context.Context, instanceID string, messageID string) error {
	client, err := a.getClient(instanceID)
	if err != nil {
		return err
	}

	var message models.Message
	if err := a.db.WithContext(ctx).Where("whats_app_message_id = ?", messageID).Preload("Contact").First(&message).Error; err != nil {
		return fmt.Errorf("failed to find message %s: %w", messageID, err)
	}

	isGroupConversation := message.ConversationID != "" && isGroupJID(message.ConversationID)
	if message.Contact == nil && !isGroupConversation {
		return fmt.Errorf("message %s has no associated contact", messageID)
	}

	var chatJID, senderJID types.JID

	if isGroupConversation {
		chatJID, err = a.parseJID(message.ConversationID)
		if err != nil {
			return fmt.Errorf("invalid group JID from conversation %s: %w", message.ConversationID, err)
		}
		senderPhone := metadataString(message.Metadata, "sender_phone")
		if senderPhone == "" && message.Contact != nil {
			senderPhone = message.Contact.PhoneNumber
		}
		senderJID, err = a.parseJID(senderPhone)
		if err != nil {
			return fmt.Errorf("invalid sender JID for group message %s: %w", messageID, err)
		}
	} else {
		chatJID, err = a.parseJID(message.Contact.PhoneNumber)
		if err != nil {
			return fmt.Errorf("invalid chat JID from contact %s: %w", message.Contact.PhoneNumber, err)
		}
		senderJID = chatJID
	}

	return client.MarkRead(ctx, []types.MessageID{types.MessageID(messageID)}, time.Now(), chatJID, senderJID)
}

// SendReaction sends an emoji reaction.
func (a *WhatsmeowAdapter) SendReaction(ctx context.Context, instanceID string, messageID string, emoji string) error {
	client, err := a.getClient(instanceID)
	if err != nil {
		return err
	}

	var message models.Message
	if err := a.db.WithContext(ctx).Where("whats_app_message_id = ?", messageID).Preload("Contact").First(&message).Error; err != nil {
		return fmt.Errorf("failed to find message %s: %w", messageID, err)
	}

	var chatJID types.JID
	if message.ConversationID != "" && isGroupJID(message.ConversationID) {
		chatJID, err = a.parseJID(message.ConversationID)
		if err != nil {
			return fmt.Errorf("invalid group JID from conversation %s: %w", message.ConversationID, err)
		}
	} else {
		if message.Contact == nil {
			return fmt.Errorf("message %s has no associated contact", messageID)
		}
		chatJID, err = a.parseJID(message.Contact.PhoneNumber)
		if err != nil {
			return fmt.Errorf("invalid chat JID from contact %s: %w", message.Contact.PhoneNumber, err)
		}
	}

	fromMe := message.Direction == "outbound"

	msg := &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Key: &waCommon.MessageKey{
				RemoteJID: proto.String(chatJID.String()),
				FromMe:    proto.Bool(fromMe),
				ID:        proto.String(messageID),
			},
			Text:              proto.String(emoji),
			SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
		},
	}

	_, err = client.SendMessage(ctx, chatJID, msg)
	if err != nil {
		return fmt.Errorf("failed to send reaction: %w", err)
	}

	return nil
}

// RevokeMessage deletes an outgoing message from WhatsApp.
func (a *WhatsmeowAdapter) RevokeMessage(ctx context.Context, instanceID string, messageID string) error {
	client, err := a.getClient(instanceID)
	if err != nil {
		return err
	}

	var message models.Message
	if err := a.db.WithContext(ctx).Where("whats_app_message_id = ?", messageID).Preload("Contact").First(&message).Error; err != nil {
		return fmt.Errorf("failed to find message %s: %w", messageID, err)
	}

	var chatJID types.JID
	if message.ConversationID != "" {
		chatJID, err = a.parseJID(message.ConversationID)
		if err != nil {
			return fmt.Errorf("invalid conversation JID %s: %w", message.ConversationID, err)
		}
	} else {
		if message.Contact == nil {
			return fmt.Errorf("message %s has no associated contact", messageID)
		}
		chatJID, err = a.parseJID(message.Contact.PhoneNumber)
		if err != nil {
			return fmt.Errorf("invalid chat JID from contact %s: %w", message.Contact.PhoneNumber, err)
		}
	}

	//nolint:staticcheck // RevokeMessage is deprecated but BuildRevoke is complex to use without types.NewMessageID
	if _, err := client.RevokeMessage(ctx, chatJID, types.MessageID(messageID)); err != nil {
		return fmt.Errorf("failed to revoke message: %w", err)
	}

	return nil
}

// GetMediaURL retrieves a media URL.
func (a *WhatsmeowAdapter) GetMediaURL(ctx context.Context, instanceID string, mediaID string) (string, error) {
	return "", fmt.Errorf("not applicable for whatsmeow")
}

// DownloadMedia downloads media.
func (a *WhatsmeowAdapter) DownloadMedia(ctx context.Context, instanceID string, mediaURL string) ([]byte, error) {
	data, _, err := a.downloadMediaFromURL(mediaURL)
	return data, err
}

// UploadMedia uploads media.
func (a *WhatsmeowAdapter) UploadMedia(ctx context.Context, instanceID string, mediaType string, data []byte) (string, error) {
	client, err := a.getClient(instanceID)
	if err != nil {
		return "", err
	}

	var appType whatsmeow.MediaType
	switch mediaType {
	case "image":
		appType = whatsmeow.MediaImage
	case "video":
		appType = whatsmeow.MediaVideo
	case "audio":
		appType = whatsmeow.MediaAudio
	case "document":
		appType = whatsmeow.MediaDocument
	default:
		appType = whatsmeow.MediaDocument
	}

	resp, err := client.Upload(ctx, data, appType)
	if err != nil {
		return "", err
	}

	return resp.URL, nil
}
