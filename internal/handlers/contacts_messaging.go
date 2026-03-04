package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	whatsmeowpkg "github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	waTypes "go.mau.fi/whatsmeow/types"
)

// SendMessageRequest represents a send message request
type SendMessageRequest struct {
	Type    models.MessageType `json:"type"`
	Content struct {
		Body string `json:"body"`
	} `json:"content"`
	InstanceID       string `json:"instance_id,omitempty"`
	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
	WhatsAppAccount  string `json:"whatsapp_account,omitempty"`

	// Interactive message fields (for type="interactive")
	Interactive *InteractiveContent `json:"interactive,omitempty"`
}

// InteractiveContent holds interactive message data
type InteractiveContent struct {
	Type       string          `json:"type"`                  // "button", "list", "cta_url"
	Body       string          `json:"body"`                  // Body text
	Buttons    []ButtonContent `json:"buttons,omitempty"`     // For button type
	ButtonText string          `json:"button_text,omitempty"` // For cta_url type
	URL        string          `json:"url,omitempty"`         // For cta_url type
}

// ButtonContent represents a button in interactive messages
type ButtonContent struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type sendTypingPresenceRequest struct {
	State      string `json:"state"`
	InstanceID string `json:"instance_id,omitempty"`
}

// SendMessage sends a message to a contact
// Agents can only send messages to their assigned contacts
func (a *App) SendMessage(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	// Parse request body
	var req SendMessageRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Get contact (users without full read permission can only message their assigned contacts)
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if !a.canReadAllContacts(userID, orgID) {
		query = applyAssignedOrPublicContactAccessFilter(query, userID)
	}
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}
	status := normalizeContactStatus(&contact)
	if status == models.ChatStatusClosed {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Closed chats are read-only", nil, "")
	}
	if isChatRestrictedForMessageRead(contact) && !a.canSendRestrictedChatWithoutClaimForContact(contact, userID, orgID) {
		return r.SendErrorEnvelope(
			fasthttp.StatusForbidden,
			"This chat is currently unassigned. Claim it before sending messages.",
			nil,
			"",
		)
	}

	var (
		selectedInstanceID *uuid.UUID
		selectedInstance   *models.WhatsAppInstance
	)
	if a.isWhatsmeowProvider() {
		instance, resolveErr := a.resolveOutboundInstance(orgID, req.InstanceID, contact.InstanceID)
		if resolveErr != nil {
			if _, reasonCode, ok := asInstanceSelectionError(resolveErr); ok {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, resolveErr.Error(), reasonCodeDetails(reasonCode), "instance_id")
			}
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, resolveErr.Error(), nil, "instance_id")
		}
		selectedInstance = instance
		selectedInstanceID = &instance.ID
	}

	account, err := a.resolveOutboundMessageAccount(orgID, &contact, req.WhatsAppAccount, selectedInstance)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to resolve WhatsApp account", nil, "")
	}

	// Handle reply context
	var replyToMessage *models.Message
	if req.ReplyToMessageID != "" {
		replyToID, err := uuid.Parse(req.ReplyToMessageID)
		if err == nil {
			var replyTo models.Message
			if err := a.DB.Where("id = ? AND contact_id = ?", replyToID, contactID).First(&replyTo).Error; err == nil {
				replyToMessage = &replyTo
			}
		}
	}

	// Build request and send using unified sender
	msgReq := OutgoingMessageRequest{
		Account:        account,
		Contact:        &contact,
		InstanceID:     selectedInstanceID,
		Type:           req.Type,
		Content:        req.Content.Body,
		ReplyToMessage: replyToMessage,
	}

	// Handle interactive messages
	if req.Type == models.MessageTypeInteractive && req.Interactive != nil {
		msgReq.InteractiveType = req.Interactive.Type
		msgReq.BodyText = req.Interactive.Body
		msgReq.ButtonText = req.Interactive.ButtonText
		msgReq.URL = req.Interactive.URL

		// Convert buttons
		if len(req.Interactive.Buttons) > 0 {
			msgReq.Buttons = make([]whatsapp.Button, len(req.Interactive.Buttons))
			for i, btn := range req.Interactive.Buttons {
				msgReq.Buttons[i] = whatsapp.Button{
					ID:    btn.ID,
					Title: btn.Title,
				}
			}
		}
	}

	opts := DefaultSendOptions()
	opts.SentByUserID = &userID

	ctx := context.Background()
	message, err := a.SendOutgoingMessage(ctx, msgReq, opts)
	if err != nil {
		if restrictedMessage, reasonCode, ok := asRestrictedSendViolationWithReason(err); ok {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, restrictedMessage, reasonCodeDetails(reasonCode), "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to send message", nil, "")
	}

	// Build response
	response := MessageResponse{
		ID:              message.ID,
		ContactID:       message.ContactID,
		ConversationID:  message.ConversationID,
		IsGroupChat:     isGroupMessage(*message),
		Direction:       message.Direction,
		MessageType:     message.MessageType,
		Content:         map[string]string{"body": message.Content},
		InteractiveData: message.InteractiveData,
		Status:          message.Status,
		IsReply:         message.IsReply,
		WhatsAppAccount: message.WhatsAppAccount,
		CreatedAt:       message.CreatedAt,
		UpdatedAt:       message.UpdatedAt,
	}

	if message.InstanceID != nil {
		instanceIDStr := message.InstanceID.String()
		response.InstanceID = &instanceIDStr
	}

	// Add reply context to response
	if message.IsReply && message.ReplyToMessageID != nil && replyToMessage != nil {
		replyToID := message.ReplyToMessageID.String()
		response.ReplyToMessageID = &replyToID
		response.ReplyToMessage = &ReplyPreview{
			ID:            replyToMessage.ID.String(),
			Content:       map[string]string{"body": replyToMessage.Content},
			MessageType:   replyToMessage.MessageType,
			Direction:     replyToMessage.Direction,
			MediaURL:      replyToMessage.MediaURL,
			MediaMimeType: replyToMessage.MediaMimeType,
			MediaFilename: replyToMessage.MediaFilename,
		}
	}

	return r.SendEnvelope(response)
}

// SendTypingPresence sends live typing presence (composing/paused) for chat compose UX.
// This endpoint is best-effort and returns success even when typing presence is skipped.
func (a *App) SendTypingPresence(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	if !a.isWhatsmeowProvider() || a.WhatsmeowManager == nil {
		return r.SendEnvelope(map[string]string{"status": "ignored"})
	}

	var req sendTypingPresenceRequest
	body := strings.TrimSpace(string(r.RequestCtx.PostBody()))
	if body != "" {
		if err := json.Unmarshal([]byte(body), &req); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
		}
	}

	state, err := parseTypingPresenceState(req.State)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "state")
	}

	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if !a.canReadAllContacts(userID, orgID) {
		query = applyAssignedOrPublicContactAccessFilter(query, userID)
	}
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}
	if isChannelOrGroupContact(contact) {
		return r.SendEnvelope(map[string]string{"status": "skipped"})
	}

	instance, resolveErr := a.resolveOutboundInstance(orgID, req.InstanceID, contact.InstanceID)
	if resolveErr != nil {
		if _, reasonCode, ok := asInstanceSelectionError(resolveErr); ok {
			return r.SendEnvelope(map[string]string{
				"status":      "skipped",
				"reason_code": reasonCode,
			})
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to resolve outbound instance", nil, "")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = a.WhatsmeowManager.SendTypingPresence(ctx, instance.ID, contact.PhoneNumber, state)
	if err != nil {
		switch {
		case errors.Is(err, whatsmeowpkg.ErrTypingPresenceUnsupportedChat):
			return r.SendEnvelope(map[string]string{"status": "skipped"})
		case errors.Is(err, whatsmeowpkg.ErrTypingPresenceInstanceUnavailable):
			return r.SendEnvelope(map[string]string{"status": "skipped", "reason_code": ReasonCodeInstanceNotConn})
		case errors.Is(err, whatsmeowpkg.ErrTypingPresenceInvalidRecipient):
			return r.SendEnvelope(map[string]string{"status": "skipped"})
		default:
			a.Log.Warn("Failed to send typing presence",
				"contact_id", contact.ID,
				"instance_id", instance.ID,
				"state", state,
				"error", err,
			)
			return r.SendEnvelope(map[string]string{"status": "skipped"})
		}
	}

	return r.SendEnvelope(map[string]string{"status": "ok"})
}

func parseTypingPresenceState(raw string) (waTypes.ChatPresence, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "", "composing":
		return waTypes.ChatPresenceComposing, nil
	case "paused":
		return waTypes.ChatPresencePaused, nil
	default:
		return "", fmt.Errorf("state must be one of: composing, paused")
	}
}

func isChannelOrGroupContact(contact models.Contact) bool {
	phone := strings.ToLower(strings.TrimSpace(contact.PhoneNumber))
	if strings.HasSuffix(phone, "@g.us") || strings.HasSuffix(phone, "@newsletter") {
		return true
	}
	if contact.Metadata == nil {
		return false
	}
	if isGroup, ok := contact.Metadata["is_group_chat"].(bool); ok && isGroup {
		return true
	}
	if isChannel, ok := contact.Metadata["is_channel_chat"].(bool); ok && isChannel {
		return true
	}
	return false
}

// resolveWhatsAppAccount gets the WhatsApp account for sending messages
func (a *App) resolveWhatsAppAccount(orgID uuid.UUID, accountName string) (*models.WhatsAppAccount, error) {
	var account models.WhatsAppAccount

	if accountName != "" {
		if err := a.DB.Where("name = ? AND organization_id = ?", accountName, orgID).First(&account).Error; err != nil {
			return nil, fmt.Errorf("WhatsApp account not found")
		}
		if err := a.decryptAccountSecrets(&account); err != nil {
			return nil, err
		}
		return &account, nil
	}

	// Get default outgoing account
	if err := a.DB.Where("organization_id = ? AND is_default_outgoing = ?", orgID, true).First(&account).Error; err != nil {
		// Fall back to any account
		if err := a.DB.Where("organization_id = ?", orgID).First(&account).Error; err != nil {
			return nil, fmt.Errorf("no WhatsApp account configured")
		}
	}
	if err := a.decryptAccountSecrets(&account); err != nil {
		return nil, err
	}
	return &account, nil
}

// resolveWhatsAppAccountByID fetches a WhatsApp account by UUID and org and decrypts secrets.
func (a *App) resolveWhatsAppAccountByID(
	r *fastglue.Request,
	id uuid.UUID,
	orgID uuid.UUID,
) (*models.WhatsAppAccount, error) {
	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil, err
	}
	if err := a.decryptAccountSecrets(account); err != nil {
		a.Log.Error("Failed to decrypt account secrets", "error", err, "account_id", account.ID)
		return nil, fmt.Errorf("failed to decrypt account secrets")
	}
	return account, nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// SendMediaMessage sends a media message (image, document, video, audio) to a contact
func (a *App) SendMediaMessage(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Parse multipart form
	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid multipart form", nil, "")
	}

	// Get contact ID from form
	contactIDValues := form.Value["contact_id"]
	if len(contactIDValues) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "contact_id is required", nil, "")
	}
	contactID, err := uuid.Parse(contactIDValues[0])
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid contact ID", nil, "")
	}

	// Get caption (optional)
	caption := ""
	if captionValues := form.Value["caption"]; len(captionValues) > 0 {
		caption = captionValues[0]
	}

	// Get WhatsApp account override (optional)
	formWhatsAppAccount := ""
	if accountValues := form.Value["whatsapp_account"]; len(accountValues) > 0 {
		formWhatsAppAccount = accountValues[0]
	}

	// Get uploaded file
	files := form.File["file"]
	if len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "file is required", nil, "")
	}
	fileHeader := files[0]

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to read file", nil, "")
	}
	defer func() { _ = file.Close() }()

	// Read at most 100MB + 1 byte to enforce bounded memory usage and policy limits.
	fileData, err := io.ReadAll(io.LimitReader(file, whatsappDocumentMaxBytes+1))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file data", nil, "")
	}

	effectiveMIMEType := resolveWhatsAppMediaMIME(
		fileHeader.Header.Get("Content-Type"),
		fileHeader.Filename,
		fileData,
	)
	mediaType := deriveWhatsAppMediaMessageType(effectiveMIMEType)
	maxAllowedBytes := whatsappMediaMaxSizeBytes(mediaType)
	if int64(len(fileData)) > maxAllowedBytes {
		return r.SendErrorEnvelope(
			fasthttp.StatusBadRequest,
			fmt.Sprintf("%s file is too large (max %dMB)", mediaType, whatsappMediaMaxSizeMB(mediaType)),
			nil,
			"file",
		)
	}

	// Get contact (users without full read permission can only message their assigned contacts)
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if !a.canReadAllContacts(userID, orgID) {
		query = applyAssignedOrPublicContactAccessFilter(query, userID)
	}
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}
	status := normalizeContactStatus(&contact)
	if status == models.ChatStatusClosed {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Closed chats are read-only", nil, "")
	}
	if isChatRestrictedForMessageRead(contact) && !a.canSendRestrictedChatWithoutClaimForContact(contact, userID, orgID) {
		return r.SendErrorEnvelope(
			fasthttp.StatusForbidden,
			"This chat is currently unassigned. Claim it before sending messages.",
			nil,
			"",
		)
	}

	requestedInstanceID := ""
	if instanceValues := form.Value["instance_id"]; len(instanceValues) > 0 {
		requestedInstanceID = instanceValues[0]
	}
	var (
		selectedInstanceID *uuid.UUID
		selectedInstance   *models.WhatsAppInstance
	)
	if a.isWhatsmeowProvider() {
		instance, resolveErr := a.resolveOutboundInstance(orgID, requestedInstanceID, contact.InstanceID)
		if resolveErr != nil {
			if _, reasonCode, ok := asInstanceSelectionError(resolveErr); ok {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, resolveErr.Error(), reasonCodeDetails(reasonCode), "instance_id")
			}
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, resolveErr.Error(), nil, "instance_id")
		}
		selectedInstance = instance
		selectedInstanceID = &instance.ID
	}

	account, err := a.resolveOutboundMessageAccount(orgID, &contact, formWhatsAppAccount, selectedInstance)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	// Save file locally first
	localPath, err := a.saveMediaLocally(fileData, effectiveMIMEType, fileHeader.Filename)
	if err != nil {
		a.Log.Error("Failed to save media locally", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save media", nil, "")
	}

	// Build and send via unified message sender
	msgReq := OutgoingMessageRequest{
		Account:       account,
		Contact:       &contact,
		InstanceID:    selectedInstanceID,
		Type:          mediaType,
		MediaData:     fileData,
		MediaURL:      localPath,
		MediaMimeType: effectiveMIMEType,
		MediaFilename: fileHeader.Filename,
		Caption:       caption,
	}

	opts := DefaultSendOptions()
	opts.SentByUserID = &userID

	ctx := context.Background()
	message, err := a.SendOutgoingMessage(ctx, msgReq, opts)
	if err != nil {
		if restrictedMessage, reasonCode, ok := asRestrictedSendViolationWithReason(err); ok {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, restrictedMessage, reasonCodeDetails(reasonCode), "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to send message", nil, "")
	}

	response := MessageResponse{
		ID:              message.ID,
		ContactID:       message.ContactID,
		ConversationID:  message.ConversationID,
		IsGroupChat:     isGroupMessage(*message),
		Direction:       message.Direction,
		MessageType:     message.MessageType,
		Content:         map[string]string{"body": message.Content},
		MediaURL:        message.MediaURL,
		MediaMimeType:   message.MediaMimeType,
		MediaFilename:   message.MediaFilename,
		Status:          message.Status,
		WhatsAppAccount: message.WhatsAppAccount,
		CreatedAt:       message.CreatedAt,
		UpdatedAt:       message.UpdatedAt,
	}
	if message.InstanceID != nil {
		instanceIDStr := message.InstanceID.String()
		response.InstanceID = &instanceIDStr
	}

	return r.SendEnvelope(response)
}

// saveMediaLocally saves media data to local storage and returns the relative path
func (a *App) saveMediaLocally(data []byte, mimeType, filename string) (string, error) {
	// Determine subdirectory based on MIME type
	var subdir string
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		subdir = "images"
	case strings.HasPrefix(mimeType, "video/"):
		subdir = "videos"
	case strings.HasPrefix(mimeType, "audio/"):
		subdir = "audio"
	default:
		subdir = "documents"
	}

	// Ensure directory exists
	if err := a.ensureMediaDir(subdir); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
	}

	// Get extension from MIME type or filename
	ext := getExtensionFromMimeType(mimeType)
	if ext == "" {
		ext = safeUploadFileExtension(filename)
	}
	if ext == "" {
		ext = ".bin"
	}

	// Generate unique filename
	newFilename := uuid.New().String() + ext
	filePath := filepath.Join(a.getMediaStoragePath(), subdir, newFilename)

	// Save file
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to save media file: %w", err)
	}

	// Return relative path
	relativePath := filepath.Join(subdir, newFilename)
	a.Log.Info("Media saved locally", "path", relativePath, "size", len(data))

	return relativePath, nil
}

func safeUploadFileExtension(filename string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(filename), "\\", "/")
	base := filepath.Base(normalized)
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(base)))
	if ext == "" || len(ext) > 17 {
		return ""
	}
	for _, r := range ext[1:] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return ""
		}
	}
	return ext
}

// SendReactionRequest represents a request to send a reaction
type SendReactionRequest struct {
	Emoji string `json:"emoji"` // Empty string to remove reaction
}

// SendReaction sends a reaction to a message
func (a *App) SendReaction(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	messageIDStr := r.RequestCtx.UserValue("message_id").(string)

	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid message ID", nil, "")
	}

	// Parse request body
	var req SendReactionRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Get contact (users without full read permission can only react to messages in their assigned contacts)
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if !a.canReadAllContacts(userID, orgID) {
		query = applyAssignedOrPublicContactAccessFilter(query, userID)
	}
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Get message
	var message models.Message
	if err := a.DB.Where("id = ? AND contact_id = ?", messageID, contactID).First(&message).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Message not found", nil, "")
	}

	// Resolve WhatsApp account only for Meta provider.
	var account *models.WhatsAppAccount
	if !a.isWhatsmeowProvider() {
		reactionAccountName := message.WhatsAppAccount
		if reactionAccountName == "" {
			reactionAccountName = contact.WhatsAppAccount
		}
		account, err = a.resolveWhatsAppAccount(orgID, reactionAccountName)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
		}
	}

	// Parse existing reactions from Metadata
	var metadata map[string]interface{}
	if message.Metadata != nil {
		metadata = message.Metadata
	} else {
		metadata = make(map[string]interface{})
	}

	// Get or initialize reactions array
	type Reaction struct {
		Emoji     string `json:"emoji"`
		FromPhone string `json:"from_phone,omitempty"`
		FromUser  string `json:"from_user,omitempty"`
	}
	var reactions []Reaction
	if reactionsRaw, ok := metadata["reactions"]; ok {
		if reactionsArray, ok := reactionsRaw.([]interface{}); ok {
			for _, r := range reactionsArray {
				if rMap, ok := r.(map[string]interface{}); ok {
					emoji, _ := rMap["emoji"].(string)
					fromPhone, _ := rMap["from_phone"].(string)
					fromUser, _ := rMap["from_user"].(string)
					reactions = append(reactions, Reaction{
						Emoji:     emoji,
						FromPhone: fromPhone,
						FromUser:  fromUser,
					})
				}
			}
		}
	}

	// Remove existing reaction from this user (each user can only have one reaction)
	userIDStr := userID.String()
	var newReactions []Reaction
	for _, r := range reactions {
		if r.FromUser != userIDStr {
			newReactions = append(newReactions, r)
		}
	}

	// Add new reaction if emoji is not empty
	if req.Emoji != "" {
		newReactions = append(newReactions, Reaction{
			Emoji:    req.Emoji,
			FromUser: userIDStr,
		})
	}

	// Update metadata
	metadata["reactions"] = newReactions
	if err := a.DB.Model(&message).Update("metadata", metadata).Error; err != nil {
		a.Log.Error("Failed to update message reactions", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update reaction", nil, "")
	}

	// Send reaction to WhatsApp API
	go a.sendWhatsAppReaction(account, &contact, &message, req.Emoji)

	// Broadcast via WebSocket
	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: "reaction_update",
			Payload: map[string]any{
				"message_id": message.ID.String(),
				"contact_id": contact.ID.String(),
				"reactions":  newReactions,
			},
		})
	}

	return r.SendEnvelope(map[string]any{
		"message_id": message.ID.String(),
		"reactions":  newReactions,
	})
}

// RevokeMessage revokes an outgoing message from WhatsApp and marks it deleted locally.
func (a *App) RevokeMessage(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if !a.HasPermission(userID, models.ResourceChat, models.ActionDelete, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to revoke messages", nil, "")
	}

	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}
	messageID, err := parsePathUUID(r, "message_id", "message")
	if err != nil {
		return nil
	}

	var contact models.Contact
	contactQuery := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if !a.canReadAllContacts(userID, orgID) {
		contactQuery = applyAssignedOrPublicContactAccessFilter(contactQuery, userID)
	}
	if err := contactQuery.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	var message models.Message
	if err := a.DB.Where("id = ? AND contact_id = ? AND organization_id = ?", messageID, contactID, orgID).First(&message).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Message not found", nil, "")
	}

	if message.Direction != models.DirectionOutgoing {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Only outgoing messages can be revoked", nil, "")
	}
	if message.InstanceID == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Message is not linked to an instance", nil, "")
	}
	waMessageID := strings.TrimSpace(message.WhatsAppMessageID)
	if waMessageID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Message does not have a WhatsApp ID", nil, "")
	}

	if a.MessageProvider == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Message provider not configured", nil, "")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.MessageProvider.RevokeMessage(ctx, message.InstanceID.String(), waMessageID); err != nil {
		a.Log.Error("Failed to revoke message via provider", "error", err, "message_id", message.ID, "wa_message_id", waMessageID)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to revoke message on WhatsApp", nil, "")
	}

	metadata := cloneJSONB(message.Metadata)
	metadata["revoked"] = true
	metadata["revoked_at"] = time.Now().UTC().Format(time.RFC3339)
	content := appendDeletedMessageCaption(message.Content)
	updatedAt := time.Now()

	if err := a.DB.Model(&models.Message{}).
		Where("id = ?", message.ID).
		Updates(map[string]any{
			"content":    content,
			"metadata":   metadata,
			"updated_at": updatedAt,
		}).Error; err != nil {
		a.Log.Error("Failed to persist revoked message state", "error", err, "message_id", message.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update message state", nil, "")
	}

	_ = a.DB.Model(&models.Contact{}).
		Where("id = ? AND organization_id = ? AND (last_message_at IS NULL OR last_message_at <= ?)", contact.ID, orgID, message.CreatedAt).
		Update("last_message_preview", content).Error

	message.Content = content
	message.Metadata = metadata
	message.UpdatedAt = updatedAt
	a.broadcastNewMessage(orgID, &message, &contact)

	shouldMask := a.ShouldMaskPhoneNumbers(orgID)
	responseMessages := a.buildMessagesResponse([]models.Message{message}, shouldMask)
	if len(responseMessages) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to serialize message", nil, "")
	}

	return r.SendEnvelope(responseMessages[0])
}

// sendWhatsAppReaction sends a reaction to WhatsApp via the configured provider
func (a *App) sendWhatsAppReaction(account *models.WhatsAppAccount, contact *models.Contact, message *models.Message, emoji string) {
	if message.WhatsAppMessageID == "" {
		a.Log.Warn("Cannot send reaction - message has no WhatsApp ID", "message_id", message.ID)
		return
	}

	if a.MessageProvider == nil {
		a.Log.Warn("Message provider not configured - cannot send reaction", "message_id", message.ID)
		return
	}

	// Use timeout context for external API calls
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use unified MessageProvider. For Meta, it will resolve the account and contact again.
	// For whatsmeow, it will use the client.
	instanceID := ""
	if message.InstanceID != nil {
		instanceID = message.InstanceID.String()
	} else if contact.InstanceID != nil {
		instanceID = contact.InstanceID.String()
	}

	if err := a.MessageProvider.SendReaction(ctx, instanceID, message.WhatsAppMessageID, emoji); err != nil {
		a.Log.Error("Failed to send reaction via provider", "error", err, "message_id", message.ID)
	} else {
		a.Log.Info("Reaction sent successfully via provider", "message_id", message.WhatsAppMessageID, "emoji", emoji)
	}
}
