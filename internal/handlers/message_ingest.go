package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/contactutil"
	"github.com/compnew2006/gowa-ui/internal/models"
)

// IncomingTextMessage represents a text, interactive, or media message from the webhook
type IncomingTextMessage struct {
	From       string `json:"from"`
	FromUserID string `json:"from_user_id,omitempty"` // BSUID
	To         string `json:"to,omitempty"`
	ID         string `json:"id"`
	Timestamp  string `json:"timestamp"`
	Type       string `json:"type"`
	Text       *struct {
		Body string `json:"body"`
	} `json:"text,omitempty"`
	Interactive *struct {
		Type        string `json:"type"`
		ButtonReply *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"button_reply,omitempty"`
		ListReply *struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"list_reply,omitempty"`
	} `json:"interactive,omitempty"`
	Image *struct {
		ID       string `json:"id"`
		MimeType string `json:"mime_type"`
		SHA256   string `json:"sha256"`
		Caption  string `json:"caption,omitempty"`
	} `json:"image,omitempty"`
	Document *struct {
		ID       string `json:"id"`
		MimeType string `json:"mime_type"`
		SHA256   string `json:"sha256"`
		Filename string `json:"filename,omitempty"`
		Caption  string `json:"caption,omitempty"`
	} `json:"document,omitempty"`
	Audio *struct {
		ID       string `json:"id"`
		MimeType string `json:"mime_type"`
	} `json:"audio,omitempty"`
	Video *struct {
		ID       string `json:"id"`
		MimeType string `json:"mime_type"`
		SHA256   string `json:"sha256"`
		Caption  string `json:"caption,omitempty"`
	} `json:"video,omitempty"`
	Sticker *struct {
		ID       string `json:"id"`
		MimeType string `json:"mime_type"`
		SHA256   string `json:"sha256"`
		Animated bool   `json:"animated,omitempty"`
	} `json:"sticker,omitempty"`
	Context *struct {
		From string `json:"from"`
		ID   string `json:"id"` // WhatsApp message ID being replied to
	} `json:"context,omitempty"`
	Reaction *struct {
		MessageID string `json:"message_id"` // WhatsApp message ID being reacted to
		Emoji     string `json:"emoji"`      // The emoji reaction (empty string = remove reaction)
	} `json:"reaction,omitempty"`
	Location *struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Name      string  `json:"name,omitempty"`
		Address   string  `json:"address,omitempty"`
	} `json:"location,omitempty"`
	Button *struct {
		Text    string `json:"text"`
		Payload string `json:"payload"`
	} `json:"button,omitempty"`
	Contacts []struct {
		Name struct {
			FormattedName string `json:"formatted_name"`
			FirstName     string `json:"first_name,omitempty"`
			LastName      string `json:"last_name,omitempty"`
		} `json:"name"`
		Phones []struct {
			Phone string `json:"phone"`
			Type  string `json:"type,omitempty"`
		} `json:"phones,omitempty"`
	} `json:"contacts,omitempty"`
}

// Reaction represents a reaction on a message
type Reaction struct {
	Emoji     string `json:"emoji"`
	FromPhone string `json:"from_phone,omitempty"` // Phone number if from contact
	FromUser  string `json:"from_user,omitempty"`  // User ID if from agent
}

// handleIncomingReaction handles incoming reaction messages from WhatsApp
func (a *App) handleIncomingReaction(account *models.WhatsAppAccount, fromPhone, messageWAMID, emoji, profileName string) {
	a.Log.Info("Handling incoming reaction",
		"from", fromPhone,
		"message_wamid", messageWAMID,
		"emoji", emoji,
	)

	// Find the message being reacted to
	// Meta encodes phone numbers in the WAMID prefix, so the same message
	// has different WAMIDs from sender vs recipient perspective. We match on
	// the suffix after "FQIA" + 4 chars (type indicator like "ERgS" or "EhgU").
	// GOWA message IDs are plain IDs without FQIA — they match directly. The
	// lookup is scoped to the reacting device's account so the reaction lands
	// on that account's copy when two org accounts share a wamid.
	var message models.Message
	if err := a.DB.Where("whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
		messageWAMID, account.OrganizationID, account.Name).First(&message).Error; err != nil {
		// Try matching on WAMID suffix (the unique message ID part)
		if idx := strings.Index(messageWAMID, "FQIA"); idx != -1 {
			// Extract suffix after "FQIA" + 4 char type indicator (e.g., "ERgS", "EhgU")
			suffixStart := idx + 8
			if suffixStart < len(messageWAMID) {
				suffix := messageWAMID[suffixStart:]
				if err := a.DB.Where("whats_app_message_id LIKE ?", "%"+suffix).First(&message).Error; err != nil {
					a.Log.Warn("Message not found for reaction", "wamid", messageWAMID, "suffix", suffix)
					return
				}
			} else {
				a.Log.Warn("Message not found for reaction - invalid WAMID format", "wamid", messageWAMID)
				return
			}
		} else {
			// Non-FQIA ID (e.g., GOWA message ID) — try a LIKE match as fallback,
			// scoped to the owning org and account to prevent cross-tenant and
			// cross-account matching.
			if err := a.DB.Where("whats_app_message_id LIKE ? AND organization_id = ? AND whats_app_account = ?",
				"%"+messageWAMID+"%", account.OrganizationID, account.Name).First(&message).Error; err != nil {
				a.Log.Warn("Message not found for reaction", "wamid", messageWAMID)
				return
			}
		}
	}

	// Get or create contact
	contact, _, _ := contactutil.GetOrCreateContact(a.DB, account.OrganizationID, fromPhone, profileName)

	// Parse existing reactions from Metadata
	var metadata map[string]any
	if message.Metadata != nil {
		metadata = message.Metadata
	} else {
		metadata = make(map[string]any)
	}

	// Get or initialize reactions array
	var reactions []Reaction
	if reactionsRaw, ok := metadata["reactions"]; ok {
		if reactionsArray, ok := reactionsRaw.([]any); ok {
			for _, r := range reactionsArray {
				if rMap, ok := r.(map[string]any); ok {
					emoji, _ := rMap["emoji"].(string)
					reactions = append(reactions, Reaction{
						Emoji:     emoji,
						FromPhone: getStringFromMap(rMap, "from_phone"),
						FromUser:  getStringFromMap(rMap, "from_user"),
					})
				}
			}
		}
	}

	// Remove existing reaction from this contact (each contact can only have one reaction)
	var newReactions []Reaction
	for _, r := range reactions {
		if r.FromPhone != fromPhone {
			newReactions = append(newReactions, r)
		}
	}

	// Add new reaction if emoji is not empty (empty = remove reaction)
	if emoji != "" {
		newReactions = append(newReactions, Reaction{
			Emoji:     emoji,
			FromPhone: fromPhone,
		})
	}

	// Update metadata
	metadata["reactions"] = newReactions

	// Save to database
	if err := a.DB.Model(&message).Update("metadata", metadata).Error; err != nil {
		a.Log.Error("Failed to update message reactions", "error", err)
		return
	}

	a.Log.Info("Updated message reaction", "message_id", message.ID, "reactions_count", len(newReactions))

	// Broadcast via WebSocket
	a.broadcastReactionUpdate(account.OrganizationID, message.ID, contact.ID, newReactions)
}

// Helper function to safely get string from map
func getStringFromMap(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ExtractedMessage holds the derived content fields of a message.
type ExtractedMessage struct {
	Text  string
	Type  string // may differ from msg.Type, e.g. "button_reply"
	Media *MediaInfo
}

// extractMessageContent walks an IncomingTextMessage and returns the derived
// fields. Used by both the inbound and echo paths.
func (a *App) extractMessageContent(ctx context.Context, msg IncomingTextMessage, account *models.WhatsAppAccount) ExtractedMessage {
	extracted := ExtractedMessage{
		Type: msg.Type,
	}

	if msg.Type == "text" && msg.Text != nil {
		extracted.Text = msg.Text.Body
	} else if msg.Type == "button" && msg.Button != nil {
		// Template quick_reply button click — WhatsApp sends type "button"
		extracted.Text = msg.Button.Text
		extracted.Type = "button_reply"
	} else if msg.Type == "interactive" && msg.Interactive != nil {
		// Handle button reply
		if msg.Interactive.ButtonReply != nil {
			extracted.Text = msg.Interactive.ButtonReply.Title
			extracted.Type = "button_reply"
		}
		// Handle list reply
		if msg.Interactive.ListReply != nil {
			extracted.Text = msg.Interactive.ListReply.Title
			extracted.Type = "button_reply"
		}
	} else if msg.Type == "image" && msg.Image != nil {
		// Handle image message
		extracted.Text = msg.Image.Caption
		extracted.Media = &MediaInfo{
			MediaMimeType: msg.Image.MimeType,
		}
		// Download and save media locally
		waAccount := a.toWhatsAppAccount(account)
		if localPath, err := a.DownloadAndSaveMedia(ctx, msg.Image.ID, msg.Image.MimeType, waAccount); err != nil {
			a.Log.Error("Failed to download image", "error", err, "media_id", msg.Image.ID)
		} else {
			extracted.Media.MediaURL = localPath
		}
	} else if msg.Type == "document" && msg.Document != nil {
		// Handle document message
		extracted.Text = msg.Document.Caption
		extracted.Media = &MediaInfo{
			MediaMimeType: msg.Document.MimeType,
			MediaFilename: msg.Document.Filename,
		}
		// Download and save media locally
		waAccount := a.toWhatsAppAccount(account)
		if localPath, err := a.DownloadAndSaveMedia(ctx, msg.Document.ID, msg.Document.MimeType, waAccount); err != nil {
			a.Log.Error("Failed to download document", "error", err, "media_id", msg.Document.ID)
		} else {
			extracted.Media.MediaURL = localPath
		}
	} else if msg.Type == "video" && msg.Video != nil {
		// Handle video message
		extracted.Text = msg.Video.Caption
		extracted.Media = &MediaInfo{
			MediaMimeType: msg.Video.MimeType,
		}
		// Download and save media locally
		waAccount := a.toWhatsAppAccount(account)
		if localPath, err := a.DownloadAndSaveMedia(ctx, msg.Video.ID, msg.Video.MimeType, waAccount); err != nil {
			a.Log.Error("Failed to download video", "error", err, "media_id", msg.Video.ID)
		} else {
			extracted.Media.MediaURL = localPath
		}
	} else if msg.Type == "audio" && msg.Audio != nil {
		// Handle audio message
		extracted.Media = &MediaInfo{
			MediaMimeType: msg.Audio.MimeType,
		}
		// Download and save media locally
		waAccount := a.toWhatsAppAccount(account)
		if localPath, err := a.DownloadAndSaveMedia(ctx, msg.Audio.ID, msg.Audio.MimeType, waAccount); err != nil {
			a.Log.Error("Failed to download audio", "error", err, "media_id", msg.Audio.ID)
		} else {
			extracted.Media.MediaURL = localPath
		}
	} else if msg.Type == "sticker" && msg.Sticker != nil {
		// Handle sticker message (treat like image)
		extracted.Media = &MediaInfo{
			MediaMimeType: msg.Sticker.MimeType,
		}
		// Download and save media locally
		waAccount := a.toWhatsAppAccount(account)
		if localPath, err := a.DownloadAndSaveMedia(ctx, msg.Sticker.ID, msg.Sticker.MimeType, waAccount); err != nil {
			a.Log.Error("Failed to download sticker", "error", err, "media_id", msg.Sticker.ID)
		} else {
			extracted.Media.MediaURL = localPath
		}
	} else if msg.Type == "location" && msg.Location != nil {
		// Handle location message - store as JSON in content
		locationData := map[string]any{
			"latitude":  msg.Location.Latitude,
			"longitude": msg.Location.Longitude,
		}
		if msg.Location.Name != "" {
			locationData["name"] = msg.Location.Name
		}
		if msg.Location.Address != "" {
			locationData["address"] = msg.Location.Address
		}
		if jsonBytes, err := json.Marshal(locationData); err == nil {
			extracted.Text = string(jsonBytes)
		}
	} else if msg.Type == "contacts" && len(msg.Contacts) > 0 {
		// Handle contacts message - store as JSON in content
		contactsData := make([]map[string]any, 0, len(msg.Contacts))
		for _, c := range msg.Contacts {
			contactVal := map[string]any{
				"name": c.Name.FormattedName,
			}
			if len(c.Phones) > 0 {
				phones := make([]string, 0, len(c.Phones))
				for _, p := range c.Phones {
					phones = append(phones, p.Phone)
				}
				contactVal["phones"] = phones
			}
			contactsData = append(contactsData, contactVal)
		}
		if jsonBytes, err := json.Marshal(contactsData); err == nil {
			extracted.Text = string(jsonBytes)
		}
	}

	return extracted
}

// MediaInfo holds media-related information for an incoming message
type MediaInfo struct {
	MediaURL      string
	MediaMimeType string
	MediaFilename string
}

// saveIncomingMessage saves an incoming message to the messages table.
// senderName/senderJID carry the per-message sender for group messages
// (empty for 1:1), stored in the message's Metadata JSONB.
func (a *App) saveIncomingMessage(account *models.WhatsAppAccount, contact *models.Contact, whatsappMsgID, msgType, content string, mediaInfo *MediaInfo, replyToWAMID, senderName, senderJID string) {
	now := time.Now()

	message := models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    account.OrganizationID,
		WhatsAppAccount:   account.Name,
		ContactID:         contact.ID,
		WhatsAppMessageID: whatsappMsgID,
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageType(msgType),
		Content:           content,
		Status:            models.MessageStatusReceived,
	}

	// Record the per-message sender for group messages in the existing
	// Metadata JSONB (no schema change). Empty for 1:1 conversations.
	if senderName != "" || senderJID != "" {
		if message.Metadata == nil {
			message.Metadata = models.JSONB{}
		}
		if senderName != "" {
			message.Metadata["sender_push_name"] = senderName
		}
		if senderJID != "" {
			message.Metadata["sender_phone"] = phoneFromJID(senderJID)
		}
	}

	// Handle reply context - look up the original message by WhatsApp message
	// ID, scoped to this account so the quote resolves to this account's copy
	// (two org accounts messaging each other share wamids across their copies).
	if replyToWAMID != "" {
		var replyToMsg models.Message
		if err := a.DB.Where("whats_app_message_id = ? AND organization_id = ? AND whats_app_account = ?",
			replyToWAMID, account.OrganizationID, account.Name).First(&replyToMsg).Error; err == nil {
			message.IsReply = true
			message.ReplyToMessageID = &replyToMsg.ID
		} else {
			a.Log.Warn("Reply-to message not found", "reply_to_wamid", replyToWAMID)
		}
	}

	// Add media fields if present
	if mediaInfo != nil {
		message.MediaURL = mediaInfo.MediaURL
		message.MediaMimeType = mediaInfo.MediaMimeType
		message.MediaFilename = mediaInfo.MediaFilename
	}

	if err := a.DB.Create(&message).Error; err != nil {
		a.Log.Error("Failed to save incoming message", "error", err)
		return
	}

	// Update contact's last message info
	preview := content
	if len(preview) > 100 {
		preview = preview[:97] + "..."
	}
	if msgType != "text" && msgType != "button_reply" {
		preview = "[" + msgType + "]"
	}

	a.DB.Model(contact).Updates(map[string]any{
		"last_message_at":      now,
		"last_message_preview": preview,
		"is_read":              false,
		"whats_app_account":    account.Name,
		"last_inbound_at":      now,
	})

	a.Log.Info("Saved incoming message", "message_id", message.ID, "contact_id", contact.ID, "media_url", message.MediaURL)

	// Broadcast new message via WebSocket
	a.broadcastNewMessage(account.OrganizationID, &message, contact)

	// Dispatch webhook for incoming message
	a.DispatchWebhook(account.OrganizationID, models.WebhookEventMessageIncoming, MessageEventData{
		MessageID:       message.ID.String(),
		ContactID:       contact.ID.String(),
		ContactPhone:    contact.PhoneNumber,
		ContactName:     contact.ProfileName,
		MessageType:     models.MessageType(msgType),
		Content:         content,
		WhatsAppAccount: account.Name,
		Direction:       models.DirectionIncoming,
	})
}
