package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/internal/contactutil"
	"github.com/compnew2006/whatomate/internal/models"
)

type IncomingTextMessage struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      *struct {
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
		NFMReply *struct {
			ResponseJSON string `json:"response_json"`
			Body         string `json:"body"`
			Name         string `json:"name"`
		} `json:"nfm_reply,omitempty"`
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
		ID   string `json:"id"`
	} `json:"context,omitempty"`
	Reaction *struct {
		MessageID string `json:"message_id"`
		Emoji     string `json:"emoji"`
	} `json:"reaction,omitempty"`
	Location *struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Name      string  `json:"name,omitempty"`
		Address   string  `json:"address,omitempty"`
	} `json:"location,omitempty"`
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

type incomingMessagePayload struct {
	MessageText      string
	MessageType      string
	ButtonID         string
	MediaInfo        *MediaInfo
	FlowResponseData map[string]any
	ReplyToWAMID     string
}

func (a *App) processIncomingMessageFull(phoneNumberID string, msg IncomingTextMessage, profileName string) {
	a.Log.Info("Processing incoming message",
		"phone_number_id", phoneNumberID,
		"from", msg.From,
		"type", msg.Type,
		"profile_name", profileName,
	)

	account, err := a.getWhatsAppAccountCached(phoneNumberID)
	if err != nil {
		a.Log.Error("WhatsApp account not found", "phone_id", phoneNumberID, "error", err)
		return
	}

	if msg.Type == "reaction" && msg.Reaction != nil {
		a.handleIncomingReaction(account, msg.From, msg.Reaction.MessageID, msg.Reaction.Emoji, profileName)
		return
	}

	contact, isNewContact, _ := contactutil.GetOrCreateContact(a.DB, account.OrganizationID, msg.From, profileName)

	payload := a.parseIncomingMessagePayload(account, msg)

	savedIncomingMessage := a.saveIncomingMessage(account, contact, msg.ID, payload.MessageType, payload.MessageText, payload.MediaInfo, payload.ReplyToWAMID)
	if savedIncomingMessage == nil {
		return
	}

	if a.licenseBlocksValueDelivery() {
		a.Log.Info("License is locked; suppressing outbound webhook/chatbot processing for inbound message",
			"contact_id", contact.ID,
			"organization_id", account.OrganizationID,
		)
		return
	}

	if isNewContact {
		a.DispatchWebhook(account.OrganizationID, models.WebhookEventContactCreated, ContactEventData{
			ContactID:       contact.ID.String(),
			ContactPhone:    contact.PhoneNumber,
			ContactName:     contact.ProfileName,
			WhatsAppAccount: account.Name,
		})
	}

	a.ClearContactChatbotTracking(contact.ID)

	if a.maybeCaptureChatCloseRating(account.OrganizationID, contact, payload, savedIncomingMessage) {
		return
	}

	if a.hasActiveAgentTransfer(account.OrganizationID, contact.ID) {
		a.Log.Info("Contact has active agent transfer, skipping chatbot processing",
			"contact_id", contact.ID,
			"phone_number", contact.PhoneNumber)
		return
	}

	settings, err := a.getChatbotSettingsCached(account.OrganizationID, account.Name)
	if err != nil {
		a.Log.Error("Failed to load chatbot settings", "error", err, "account", account.Name, "org_id", account.OrganizationID)
		return
	}
	if !settings.IsEnabled {
		a.Log.Debug("Chatbot not enabled for this account, creating transfer for agent queue", "account", account.Name, "settings_id", settings.ID)
		a.createTransferToQueue(account, contact, models.TransferSourceChatbotDisabled)
		return
	}
	a.Log.Info("Chatbot settings loaded", "settings_id", settings.ID, "is_enabled", settings.IsEnabled, "ai_enabled", settings.AI.Enabled, "ai_provider", settings.AI.Provider, "default_response", settings.DefaultResponse)

	if settings.BusinessHours.Enabled && len(settings.BusinessHours.Hours) > 0 {
		if !a.isWithinBusinessHours(settings.BusinessHours.Hours) {
			if !settings.BusinessHours.AllowAutomatedOutside {
				a.Log.Info("Outside business hours, sending out of hours message")
				a.sendOutOfHoursMessage(account, contact, settings)
				return
			}
			a.Log.Info("Outside business hours but automated responses allowed, continuing")
		}
	}

	if payload.MessageText == "" {
		a.Log.Debug("Skipping message with no text content for chatbot", "type", msg.Type)
		return
	}

	a.handleChatbotConversation(account, contact, msg, settings, payload)
}

func (a *App) parseIncomingMessagePayload(account *models.WhatsAppAccount, msg IncomingTextMessage) incomingMessagePayload {
	payload := incomingMessagePayload{
		MessageType: msg.Type,
	}

	if msg.Context != nil && msg.Context.ID != "" {
		payload.ReplyToWAMID = msg.Context.ID
	}

	switch msg.Type {
	case "text":
		if msg.Text != nil {
			payload.MessageText = msg.Text.Body
		}
	case "interactive":
		if msg.Interactive == nil {
			break
		}
		if msg.Interactive.ButtonReply != nil {
			payload.MessageText = msg.Interactive.ButtonReply.Title
			payload.ButtonID = msg.Interactive.ButtonReply.ID
			payload.MessageType = "button_reply"
		}
		if msg.Interactive.ListReply != nil {
			payload.MessageText = msg.Interactive.ListReply.Title
			payload.ButtonID = msg.Interactive.ListReply.ID
			payload.MessageType = "button_reply"
		}
		if msg.Interactive.NFMReply != nil {
			payload.MessageText = msg.Interactive.NFMReply.Body
			payload.MessageType = "nfm_reply"
			if msg.Interactive.NFMReply.ResponseJSON != "" {
				var responseData map[string]any
				if err := json.Unmarshal([]byte(msg.Interactive.NFMReply.ResponseJSON), &responseData); err != nil {
					a.Log.Error("Failed to parse flow response JSON", "error", err, "response_json", msg.Interactive.NFMReply.ResponseJSON)
				} else {
					payload.FlowResponseData = responseData
					a.Log.Info("Parsed WhatsApp Flow response", "data", payload.FlowResponseData)
				}
			}
		}
	case "image":
		if msg.Image != nil {
			a.populateMediaPayload(account, &payload, msg.Image.ID, msg.Image.MimeType, msg.Image.Caption, "", "image")
		}
	case "document":
		if msg.Document != nil {
			a.populateMediaPayload(account, &payload, msg.Document.ID, msg.Document.MimeType, msg.Document.Caption, msg.Document.Filename, "document")
		}
	case "video":
		if msg.Video != nil {
			a.populateMediaPayload(account, &payload, msg.Video.ID, msg.Video.MimeType, msg.Video.Caption, "", "video")
		}
	case "audio":
		if msg.Audio != nil {
			a.populateMediaPayload(account, &payload, msg.Audio.ID, msg.Audio.MimeType, "", "", "audio")
		}
	case "sticker":
		if msg.Sticker != nil {
			a.populateMediaPayload(account, &payload, msg.Sticker.ID, msg.Sticker.MimeType, "", "", "sticker")
		}
	case "location":
		if msg.Location != nil {
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
				payload.MessageText = string(jsonBytes)
			}
		}
	case "contacts":
		if len(msg.Contacts) > 0 {
			contactsData := make([]map[string]any, 0, len(msg.Contacts))
			for _, c := range msg.Contacts {
				contactData := map[string]any{
					"name": c.Name.FormattedName,
				}
				if len(c.Phones) > 0 {
					phones := make([]string, 0, len(c.Phones))
					for _, p := range c.Phones {
						phones = append(phones, p.Phone)
					}
					contactData["phones"] = phones
				}
				contactsData = append(contactsData, contactData)
			}
			if jsonBytes, err := json.Marshal(contactsData); err == nil {
				payload.MessageText = string(jsonBytes)
			}
		}
	}

	return payload
}

func (a *App) populateMediaPayload(
	account *models.WhatsAppAccount,
	payload *incomingMessagePayload,
	mediaID string,
	mimeType string,
	caption string,
	filename string,
	mediaType string,
) {
	payload.MessageText = caption
	payload.MediaInfo = &MediaInfo{
		MediaMimeType:    mimeType,
		MediaFilename:    filename,
		RecoveryProvider: legacyMediaRecoveryProviderMeta,
		RecoveryMediaID:  strings.TrimSpace(mediaID),
		RecoveryPhoneID:  strings.TrimSpace(account.PhoneID),
	}

	waAccount := a.toWhatsAppAccount(account)
	if savedFile, err := a.downloadAndSaveLegacyMedia(
		context.Background(),
		mediaID,
		mimeType,
		models.MessageType(mediaType),
		filename,
		waAccount,
	); err != nil {
		a.Log.Error(fmt.Sprintf("Failed to download %s", mediaType), "error", err, "media_id", mediaID)
	} else {
		payload.MediaInfo.MediaURL = savedFile.RelativePath
		payload.MediaInfo.MediaMimeType = savedFile.MIMEType
		if strings.TrimSpace(savedFile.Filename) != "" {
			payload.MediaInfo.MediaFilename = savedFile.Filename
		}
	}
}
