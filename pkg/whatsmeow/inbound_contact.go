package whatsmeow

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	waClient "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type inboundContactDetails struct {
	PhoneNumber         string
	ProfileName         string
	GroupName           string
	GroupTopic          string
	ChannelName         string
	ChannelDescription  string
	ChannelConversation string
}

func (cm *ConnectionManager) resolveInboundContactDetails(
	ctx context.Context,
	client *waClient.Client,
	chatJID types.JID,
	senderPhone string,
	senderPushName string,
	isGroup bool,
) inboundContactDetails {
	details := inboundContactDetails{
		PhoneNumber: senderPhone,
		ProfileName: senderPushName,
	}
	isChannel := chatJID.Server == types.NewsletterServer

	if !isGroup && !isChannel {
		if details.PhoneNumber == "" && chatJID.Server == types.DefaultUserServer && chatJID.User != "" {
			details.PhoneNumber = chatJID.User
		}
		if strings.TrimSpace(details.ProfileName) == "" {
			details.ProfileName = cm.resolveStoredContactName(ctx, client, chatJID, details.PhoneNumber)
		}
		if strings.TrimSpace(details.ProfileName) == "" {
			details.ProfileName = details.PhoneNumber
		}
		return details
	}

	if isChannel {
		details.ChannelConversation = chatJID.String()
		if details.PhoneNumber == "" {
			details.PhoneNumber = chatJID.User
		}

		channelName := strings.TrimSpace(senderPushName)
		channelDescription := ""
		if client != nil {
			infoCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			newsletterInfo, err := client.GetNewsletterInfo(infoCtx, chatJID)
			if err == nil && newsletterInfo != nil {
				if resolvedName := strings.TrimSpace(newsletterInfo.ThreadMeta.Name.Text); resolvedName != "" {
					channelName = resolvedName
				}
				channelDescription = strings.TrimSpace(newsletterInfo.ThreadMeta.Description.Text)
			}
		}

		if channelName == "" {
			channelName = details.PhoneNumber
		}

		details.ChannelName = channelName
		details.ChannelDescription = channelDescription
		details.ProfileName = channelName
		return details
	}

	details.PhoneNumber = chatJID.String()
	details.GroupName = chatJID.User

	if client != nil {
		groupInfo, err := client.GetGroupInfo(ctx, chatJID)
		if err == nil && groupInfo != nil {
			if trimmedName := strings.TrimSpace(groupInfo.Name); trimmedName != "" {
				details.GroupName = trimmedName
			}
			details.GroupTopic = strings.TrimSpace(groupInfo.Topic)
		}
	}

	if strings.TrimSpace(details.GroupName) == "" {
		details.GroupName = details.PhoneNumber
	}
	details.ProfileName = details.GroupName

	return details
}

func groupContactMetadata(chatJID types.JID, details inboundContactDetails) models.JSONB {
	metadata := models.JSONB{
		"is_group_chat": true,
		"group_jid":     chatJID.String(),
	}
	if details.GroupName != "" {
		metadata["group_name"] = details.GroupName
	}
	if details.GroupTopic != "" {
		metadata["group_topic"] = details.GroupTopic
	}
	return metadata
}

func channelContactMetadata(chatJID types.JID, details inboundContactDetails) models.JSONB {
	metadata := models.JSONB{
		"is_channel_chat": true,
		"channel_jid":     chatJID.String(),
	}
	if details.ChannelName != "" {
		metadata["channel_name"] = details.ChannelName
	}
	if details.ChannelDescription != "" {
		metadata["channel_description"] = details.ChannelDescription
	}
	return metadata
}

func mergeJSONB(existing, updates models.JSONB) (models.JSONB, bool) {
	if len(updates) == 0 {
		return existing, false
	}

	merged := make(models.JSONB, len(existing)+len(updates))
	for key, value := range existing {
		merged[key] = value
	}

	changed := false
	for key, value := range updates {
		current, ok := merged[key]
		if !ok || !reflect.DeepEqual(current, value) {
			merged[key] = value
			changed = true
		}
	}

	return merged, changed
}

func isGroupContactMetadata(metadata models.JSONB) bool {
	if metadata == nil {
		return false
	}
	isGroup, ok := metadata["is_group_chat"].(bool)
	return ok && isGroup
}

func isChannelContactMetadata(metadata models.JSONB) bool {
	if metadata == nil {
		return false
	}
	isChannel, ok := metadata["is_channel_chat"].(bool)
	return ok && isChannel
}

func metadataString(metadata models.JSONB, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	parsed, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(parsed)
}

func (cm *ConnectionManager) resolveStoredContactName(
	ctx context.Context,
	client *waClient.Client,
	chatJID types.JID,
	phoneNumber string,
) string {
	if client == nil || client.Store == nil || client.Store.Contacts == nil {
		return ""
	}

	candidates := make([]types.JID, 0, 2)
	if !chatJID.IsEmpty() {
		candidates = append(candidates, chatJID.ToNonAD())
	}
	if strings.TrimSpace(phoneNumber) != "" {
		if parsed, err := types.ParseJID(strings.TrimSpace(phoneNumber) + "@s.whatsapp.net"); err == nil {
			candidates = append(candidates, parsed.ToNonAD())
		}
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		key := candidate.String()
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		contactInfo, err := client.Store.Contacts.GetContact(ctx, candidate)
		if err != nil || !contactInfo.Found {
			continue
		}
		if name := strings.TrimSpace(firstNonEmptyString(
			contactInfo.BusinessName,
			contactInfo.FullName,
			contactInfo.PushName,
			contactInfo.FirstName,
		)); name != "" {
			return name
		}
	}

	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
