package handlers

import (
	"context"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// SendCannedResponseRequest represents the payload for dispatching a canned response to a contact.
type SendCannedResponseRequest struct {
	ContactID        string `json:"contact_id"`
	Content          string `json:"content,omitempty"`
	InstanceID       string `json:"instance_id,omitempty"`
	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
	WhatsAppAccount  string `json:"whatsapp_account,omitempty"`
}

// SendCannedResponse sends a canned response as one or more outbound messages (text + attachments).
func (a *App) SendCannedResponse(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	cannedResponseID, err := parsePathUUID(r, "id", "canned response")
	if err != nil {
		return nil
	}

	var req SendCannedResponseRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	contactID, err := uuid.Parse(strings.TrimSpace(req.ContactID))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "contact_id is required", nil, "")
	}

	var cannedResponse models.CannedResponse
	if err := a.DB.Where("id = ? AND organization_id = ?", cannedResponseID, orgID).
		First(&cannedResponse).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Canned response not found", nil, "")
	}
	if !cannedResponse.IsActive {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Canned response is inactive", nil, "")
	}

	// Get contact (users without full read permission can only message their assigned contacts)
	var contact models.Contact
	contactQuery := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionRead, orgID) {
		contactQuery = contactQuery.Where("assigned_user_id = ?", userID)
	}
	if err := contactQuery.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	status := normalizeContactStatus(&contact)
	if status == models.ChatStatusClosed {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Closed chats are read-only", nil, "")
	}
	if isChatRestrictedForMessageRead(contact) && !a.canBypassPendingChatRestriction(userID, orgID) {
		return r.SendErrorEnvelope(
			fasthttp.StatusForbidden,
			"This chat is currently unassigned. Claim it before sending messages.",
			nil,
			"",
		)
	}

	var selectedInstanceID *uuid.UUID
	if a.isWhatsmeowProvider() {
		instance, resolveErr := a.resolveOutboundInstance(orgID, req.InstanceID, contact.InstanceID)
		if resolveErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, resolveErr.Error(), nil, "instance_id")
		}
		selectedInstanceID = &instance.ID
	}

	// Resolve WhatsApp account only for Meta provider.
	var account *models.WhatsAppAccount
	if !a.isWhatsmeowProvider() {
		accountName := contact.WhatsAppAccount
		if req.WhatsAppAccount != "" {
			accountName = req.WhatsAppAccount
		}
		account, err = a.resolveWhatsAppAccount(orgID, accountName)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to resolve WhatsApp account", nil, "")
		}
	}

	var replyToMessage *models.Message
	if req.ReplyToMessageID != "" {
		replyToID, parseErr := uuid.Parse(req.ReplyToMessageID)
		if parseErr == nil {
			var replyTo models.Message
			if err := a.DB.Where("id = ? AND contact_id = ?", replyToID, contactID).First(&replyTo).Error; err == nil {
				replyToMessage = &replyTo
			}
		}
	}

	messageText := req.Content
	if strings.TrimSpace(messageText) == "" {
		messageText = cannedResponse.Content
	}

	opts := DefaultSendOptions()
	opts.SentByUserID = &userID

	sentMessages := make([]models.Message, 0, 1+len(cannedResponse.Attachments))
	sendCtx := context.Background()

	if strings.TrimSpace(messageText) != "" {
		textRequest := OutgoingMessageRequest{
			Account:        account,
			Contact:        &contact,
			InstanceID:     selectedInstanceID,
			Type:           models.MessageTypeText,
			Content:        messageText,
			ReplyToMessage: replyToMessage,
		}
		message, sendErr := a.SendOutgoingMessage(sendCtx, textRequest, opts)
		if sendErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to send canned response text", nil, "")
		}
		sentMessages = append(sentMessages, *message)
	}

	for _, attachment := range cannedResponse.Attachments {
		attachmentData, readErr := a.readCannedAttachmentData(attachment)
		if readErr != nil {
			a.Log.Error("Failed to read canned response attachment", "error", readErr, "canned_response_id", cannedResponse.ID, "attachment_id", attachment.ID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load canned response attachment", nil, "")
		}

		messageType := models.MessageTypeImage
		if attachment.Type == models.CannedResponseAttachmentTypeVideo {
			messageType = models.MessageTypeVideo
		}

		mediaRequest := OutgoingMessageRequest{
			Account:       account,
			Contact:       &contact,
			InstanceID:    selectedInstanceID,
			Type:          messageType,
			MediaData:     attachmentData,
			MediaURL:      attachment.FilePath,
			MediaMimeType: attachment.MimeType,
			MediaFilename: attachment.FileName,
		}
		message, sendErr := a.SendOutgoingMessage(sendCtx, mediaRequest, opts)
		if sendErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to send canned response attachment", nil, "")
		}
		sentMessages = append(sentMessages, *message)
	}

	if len(sentMessages) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Canned response has no text or attachments to send", nil, "")
	}

	shouldMaskPhoneNumbers := a.ShouldMaskPhoneNumbers(orgID)
	response := a.buildMessagesResponse(sentMessages, shouldMaskPhoneNumbers)
	return r.SendEnvelope(map[string]any{
		"messages": response,
	})
}
