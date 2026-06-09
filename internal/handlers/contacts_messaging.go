package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	whatsmeowpkg "github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	waTypes "go.mau.fi/whatsmeow/types"
	"gorm.io/gorm"
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

	// Poll message fields (for type="poll")
	PollOptions       []string `json:"poll_options,omitempty"`
	PollMaxSelections int      `json:"poll_max_selections,omitempty"`
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
	requestDB := a.requestDB(r)
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

	// Agent-role users keep chat-scoped visibility even though they carry contacts:read.
	var contact models.Contact
	query := requestDB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if a.shouldRestrictChatVisibilityToAgentScope(userID, orgID) {
		query = applyAgentVisibleChatAccessFilter(query, userID)
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
		a.Log.Error("Failed to resolve WhatsApp account for sending", "error", err, "contact_id", contactID, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	// Handle reply context
	var replyToMessage *models.Message
	if req.ReplyToMessageID != "" {
		replyToID, err := uuid.Parse(req.ReplyToMessageID)
		if err == nil {
			var replyTo models.Message
			if err := a.DB.Where("id = ? AND contact_id = ? AND organization_id = ?", replyToID, contactID, orgID).First(&replyTo).Error; err == nil {
				replyToMessage = &replyTo
				a.Log.Info("Found reply message in DB", "req.ReplyToMessageID", req.ReplyToMessageID, "whats_app_message_id", replyToMessage.WhatsAppMessageID)
			} else {
				a.Log.Error("Reply message not found in DB", "req.ReplyToMessageID", req.ReplyToMessageID, "err", err)
			}
		} else {
			a.Log.Error("Invalid ReplyToMessageID UUID", "req.ReplyToMessageID", req.ReplyToMessageID, "err", err)
		}
	} else {
		a.Log.Info("No ReplyToMessageID provided in request")
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

	// Handle poll messages
	if req.Type == models.MessageTypePoll {
		msgReq.PollOptions = req.PollOptions
		msgReq.PollMaxSelections = req.PollMaxSelections
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
	requestDB := a.requestDB(r)
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
	query := requestDB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if a.shouldRestrictChatVisibilityToAgentScope(userID, orgID) {
		query = applyAgentVisibleChatAccessFilter(query, userID)
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
	requestDB := a.requestDB(r)
	account, err := findByIDAndOrg[models.WhatsAppAccount](requestDB, r, id, orgID, "Account")
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
	requestDB := a.requestDB(r)
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

	// Agent-role users keep chat-scoped visibility even though they carry contacts:read.
	var contact models.Contact
	query := requestDB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if a.shouldRestrictChatVisibilityToAgentScope(userID, orgID) {
		query = applyAgentVisibleChatAccessFilter(query, userID)
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
	if !a.checkQuotaWithDeltaOrRespond(r, license.ResourceStorage, orgID, int64(len(fileData))) {
		return nil
	}

	localPath, err := a.saveMediaLocally(orgID, fileData, effectiveMIMEType, fileHeader.Filename)
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
func (a *App) saveMediaLocally(orgID uuid.UUID, data []byte, mimeType, filename string) (string, error) {
	// Determine subdirectory based on MIME type
	var mediaTypeDir string
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		mediaTypeDir = "images"
	case strings.HasPrefix(mimeType, "video/"):
		mediaTypeDir = "videos"
	case strings.HasPrefix(mimeType, "audio/"):
		mediaTypeDir = "audio"
	default:
		mediaTypeDir = "documents"
	}
	subdir := organizationMediaSubdir(orgID, mediaTypeDir)

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
	relativePath := filepath.Join(subdir, newFilename)
	filePath := filepath.Join(a.getMediaStoragePath(), relativePath)

	// Save file
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to save media file: %w", err)
	}

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
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	messageID, err := parsePathUUID(r, "message_id", "message")
	if err != nil {
		return nil
	}

	// Parse request body
	var req SendReactionRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Agent-role users keep chat-scoped visibility even though they carry contacts:read.
	var contact models.Contact
	query := requestDB.Session(&gorm.Session{}).Where("id = ? AND organization_id = ?", contactID, orgID)
	if a.shouldRestrictChatVisibilityToAgentScope(userID, orgID) {
		query = applyAgentVisibleChatAccessFilter(query, userID)
	}
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Get message
	var message models.Message
	if err := requestDB.Session(&gorm.Session{}).Where("id = ? AND contact_id = ?", messageID, contactID).First(&message).Error; err != nil {
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
	newReactions := make([]Reaction, 0)
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
	if err := requestDB.Session(&gorm.Session{}).Model(&message).Update("metadata", metadata).Error; err != nil {
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
	requestDB := a.requestDB(r)
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
	contactQuery := requestDB.Where("id = ? AND organization_id = ?", contactID, orgID)
	if a.shouldRestrictChatVisibilityToAgentScope(userID, orgID) {
		contactQuery = applyAgentVisibleChatAccessFilter(contactQuery, userID)
	}
	if err := contactQuery.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	var message models.Message
	if err := requestDB.Where("id = ? AND contact_id = ? AND organization_id = ?", messageID, contactID, orgID).First(&message).Error; err != nil {
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

	if err := requestDB.Model(&models.Message{}).
		Where("id = ?", message.ID).
		Updates(map[string]any{
			"content":    content,
			"metadata":   metadata,
			"updated_at": updatedAt,
		}).Error; err != nil {
		a.Log.Error("Failed to persist revoked message state", "error", err, "message_id", message.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update message state", nil, "")
	}

	_ = requestDB.Model(&models.Contact{}).
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

// SendPollVote handles a request to vote on an existing WhatsApp poll.
// POST /api/messages/poll-vote  { message_id, selected_options }
func (a *App) SendPollVote(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var req struct {
		MessageID       string   `json:"message_id"`
		SelectedOptions []string `json:"selected_options"`
	}
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	if req.MessageID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "message_id is required", nil, "")
	}
	if req.SelectedOptions == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "selected_options is required", nil, "")
	}
	req.SelectedOptions = normalizePollVoteSelectedOptions(req.SelectedOptions)

	msgUUID, err := uuid.Parse(req.MessageID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid message_id format", nil, "")
	}

	requestDB := a.requestDB(r)
	var message models.Message
	if err := requestDB.Where("id = ? AND organization_id = ?", msgUUID, orgID).First(&message).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Message not found", nil, "")
	}

	if message.MessageType != models.MessageTypePoll {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Message is not a poll", nil, "")
	}
	if limit := pollVoteSelectionLimit(message.InteractiveData); len(req.SelectedOptions) > limit {
		return r.SendErrorEnvelope(
			fasthttp.StatusBadRequest,
			fmt.Sprintf("poll allows up to %d selected option", limit),
			nil,
			"selected_options",
		)
	}
	if message.InstanceID == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Message has no associated instance", nil, "")
	}
	if a.MessageProvider == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Message provider not available", nil, "")
	}

	pollVoter, ok := a.MessageProvider.(provider.PollVoter)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusNotAcceptable, "Poll voting not supported by current provider", nil, "")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	respID, err := pollVoter.SendPollVote(ctx, provider.PollVoteTarget{
		InstanceID:             *message.InstanceID,
		OrgID:                  orgID,
		OriginalPollWhatsAppID: message.WhatsAppMessageID,
	}, req.SelectedOptions)
	if err != nil {
		a.Log.Error("Failed to send poll vote", "error", err, "message_id", message.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to send poll vote", nil, "")
	}

	updatedInteractive := applyPollVoteSelectionToInteractive(message.InteractiveData, userID.String(), req.SelectedOptions)
	now := time.Now()
	if err := requestDB.Model(&models.Message{}).
		Where("id = ?", message.ID).
		Updates(map[string]any{
			"interactive_data": updatedInteractive,
			"updated_at":       now,
		}).Error; err != nil {
		a.Log.Error("Failed to update poll vote on original message", "error", err, "message_id", message.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update poll vote", nil, "")
	}

	message.InteractiveData = updatedInteractive
	message.UpdatedAt = now
	a.broadcastPollMessageUpdate(orgID, &message)

	return r.SendEnvelope(map[string]any{
		"wa_message_id":   respID,
		"poll_message_id": message.ID.String(),
	})
}

func normalizePollVoteSelectedOptions(selectedOptions []string) []string {
	seen := make(map[string]struct{}, len(selectedOptions))
	normalized := make([]string, 0, len(selectedOptions))
	for _, option := range selectedOptions {
		option = strings.TrimSpace(option)
		if option == "" {
			continue
		}
		if _, ok := seen[option]; ok {
			continue
		}
		seen[option] = struct{}{}
		normalized = append(normalized, option)
	}
	return normalized
}

func pollVoteSelectionLimit(interactive models.JSONB) int {
	maxSel, hasMax := interactive["max_selections"]
	selOpt, hasSelectable := interactive["selectable_options_count"]

	if !hasMax && !hasSelectable {
		return 1
	}

	limit := -1
	if hasMax {
		limit = pollVoteIntValue(maxSel)
	}
	if (limit <= 0 || !hasMax) && hasSelectable {
		limit = pollVoteIntValue(selOpt)
	}

	if limit == 0 {
		return 999
	}
	if limit < 0 {
		return 1
	}
	return limit
}

func pollVoteIntValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		parsed, _ := v.Int64()
		return int(parsed)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(v))
		return parsed
	default:
		return 0
	}
}

func applyPollVoteSelectionToInteractive(existing models.JSONB, voter string, selectedOptions []string) models.JSONB {
	updated := make(models.JSONB, len(existing)+5)
	for key, value := range existing {
		updated[key] = value
	}
	if _, ok := updated["type"].(string); !ok {
		updated["type"] = "poll"
	}

	voters := pollVoteSelectionVoters(updated["voters"])
	if len(selectedOptions) == 0 {
		delete(voters, voter)
	} else {
		voters[voter] = append([]string(nil), selectedOptions...)
	}
	updated["voters"] = voters
	updated["votes"] = pollVoteSelectionCounts(voters)
	updated["total_votes"] = len(voters)
	updated["last_selected_options"] = append([]string(nil), selectedOptions...)
	updated["last_voter"] = voter
	return updated
}

func pollVoteSelectionVoters(raw interface{}) map[string][]string {
	voters := map[string][]string{}
	switch values := raw.(type) {
	case map[string][]string:
		for voter, selected := range values {
			voters[voter] = append([]string(nil), selected...)
		}
	case map[string]interface{}:
		for voter, selected := range values {
			voters[voter] = pollVoteSelectionStrings(selected)
		}
	}
	return voters
}

func pollVoteSelectionStrings(raw interface{}) []string {
	switch values := raw.(type) {
	case []string:
		return append([]string(nil), values...)
	case []interface{}:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func pollVoteSelectionCounts(voters map[string][]string) map[string]int {
	counts := map[string]int{}
	for _, selected := range voters {
		for _, option := range selected {
			option = strings.TrimSpace(option)
			if option != "" {
				counts[option]++
			}
		}
	}
	return counts
}

func (a *App) broadcastPollMessageUpdate(orgID uuid.UUID, message *models.Message) {
	if a.WSHub == nil || message == nil {
		return
	}
	a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type: websocket.TypeMessageMediaUpdated,
		Payload: map[string]any{
			"id":               message.ID.String(),
			"contact_id":       message.ContactID.String(),
			"message_type":     message.MessageType,
			"content":          map[string]string{"body": message.Content},
			"interactive_data": message.InteractiveData,
			"updated_at":       message.UpdatedAt,
		},
	})
}
