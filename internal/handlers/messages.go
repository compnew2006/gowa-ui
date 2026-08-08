package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/internal/templateutil"
	"github.com/compnew2006/gowa-ui/internal/utils"
	"github.com/compnew2006/gowa-ui/internal/websocket"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// ============================================================================
// Unified Message Sending
// ============================================================================

// maxHeaderMediaBytes caps bytes read from a caller-supplied header_media_url
// (SSRF/DoS bound). Mirrors the 16MB media-upload ceiling (maxMediaSize) in
// campaigns.go for the same media domain.
const maxHeaderMediaBytes = 16 << 20 // 16MB

// OutgoingMessageRequest contains all parameters for sending any type of message
type OutgoingMessageRequest struct {
	// Required
	Account *models.WhatsAppAccount
	Contact *models.Contact

	// Message type determines which fields are used
	Type models.MessageType // text, image, video, audio, document, interactive, template

	// Text messages
	Content string

	// Media messages (image, video, audio, document)
	MediaID       string // WhatsApp media ID (if already uploaded)
	MediaData     []byte // Raw media data (if upload needed)
	MediaURL      string // Local media URL (for storage)
	MediaMimeType string
	MediaFilename string
	Caption       string

	// Interactive messages
	InteractiveType string            // "button", "list", "cta_url"
	BodyText        string            // Body text for interactive messages
	Buttons         []whatsapp.Button // For button/list messages
	ButtonText      string            // For CTA URL button
	URL             string            // For CTA URL button

	// Template messages (rendered locally and sent as text/media/interactive).
	// Header media for IMAGE/VIDEO/DOCUMENT headers is carried in the shared
	// MediaData/MediaURL/MediaMimeType/MediaFilename fields above.
	Template        *models.Template
	BodyParams      map[string]string // Parameter name -> value (supports both named and positional)
	HeaderParams    map[string]string // Header-only param values; falls back to BodyParams if empty (used for TEXT headers with a {{var}})
	ButtonURLParams map[string]string // Button index (as string) -> dynamic URL param value

	// Reply context
	ReplyToMessage *models.Message
}

// MessageSendOptions configures optional behaviors for message sending
type MessageSendOptions struct {
	// BroadcastWebSocket enables WebSocket broadcast to org (default: true)
	BroadcastWebSocket bool

	// DispatchWebhook enables webhook dispatch for message.sent event (default: true)
	DispatchWebhook bool

	// SentByUserID sets the user who sent the message (for agent messages)
	SentByUserID *uuid.UUID

	// Async if true, sends in background goroutine and returns immediately
	// Message is persisted before send, status updated after
	Async bool

	// MarkIncomingRead marks the contact's incoming messages as read after a
	// successful send. Used for automated replies so an auto-handled exchange
	// doesn't leave an "unread" badge in the agent's contact list.
	MarkIncomingRead bool
}

// DefaultSendOptions returns options suitable for agent UI sends
func DefaultSendOptions() MessageSendOptions {
	return MessageSendOptions{
		BroadcastWebSocket: true,
		DispatchWebhook:    true,
		Async:              true,
	}
}

// AutoReplySendOptions returns options suitable for automated replies
// (call auto-reject, CSAT close-rating prompts).
func AutoReplySendOptions() MessageSendOptions {
	return MessageSendOptions{
		BroadcastWebSocket: true,
		DispatchWebhook:    false,
		Async:              false,
		MarkIncomingRead:   true,
	}
}

// APISendOptions returns options suitable for API/template sends
func APISendOptions() MessageSendOptions {
	return MessageSendOptions{
		BroadcastWebSocket: false,
		DispatchWebhook:    true,
		Async:              true,
	}
}

// SendOutgoingMessage is the unified method for sending all types of WhatsApp messages.
// It handles: text, media (image/video/audio/document), interactive (buttons/list/cta_url), and template messages.
func (a *App) SendOutgoingMessage(ctx context.Context, req OutgoingMessageRequest, opts MessageSendOptions) (*models.Message, error) {
	// 1. Create message record
	msg := a.createOutgoingMessage(req, opts)

	// Save to database
	if err := a.DB.Create(msg).Error; err != nil {
		a.Log.Error("Failed to create message", "error", err)
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// 2. Define the send function based on message type.
	// The provider is resolved from the registry (GOWA).
	sendFn := func(sendCtx context.Context) (string, error) {
		waAccount := a.toWhatsAppAccount(req.Account)
		provider := a.resolveProvider(req.Account)
		rcpt := whatsapp.Recipient{Phone: req.Contact.PhoneNumber, BSUID: req.Contact.BSUID}

		// Get reply-to message ID if this is a reply
		var replyToMsgID string
		if req.ReplyToMessage != nil && req.ReplyToMessage.WhatsAppMessageID != "" {
			replyToMsgID = req.ReplyToMessage.WhatsAppMessageID
		}

		switch req.Type {
		case models.MessageTypeText:
			return provider.SendTextMessage(sendCtx, waAccount, rcpt, req.Content, replyToMsgID)

		case models.MessageTypeImage, models.MessageTypeVideo, models.MessageTypeAudio, models.MessageTypeDocument:
			// Upload media if MediaData is provided and MediaID is not set.
			// On retry/resend, MediaData is empty — fall back to reading the
			// local file from MediaURL so the media can be re-uploaded.
			mediaID := req.MediaID
			if mediaID == "" && len(req.MediaData) == 0 && req.MediaURL != "" {
				// Read the file from local storage for retry
				mediaPath := filepath.Join(a.getMediaStoragePath(), req.MediaURL)
				if fileData, readErr := os.ReadFile(mediaPath); readErr == nil && len(fileData) > 0 {
					req.MediaData = fileData
				}
			}
			if mediaID == "" && len(req.MediaData) > 0 {
				var err error
				mediaID, err = provider.UploadMedia(sendCtx, waAccount, req.MediaData, req.MediaMimeType, req.MediaFilename)
				if err != nil {
					return "", fmt.Errorf("failed to upload media: %w", err)
				}
			}
			// Send the appropriate media type
			switch req.Type {
			case models.MessageTypeImage:
				return provider.SendImageMessage(sendCtx, waAccount, rcpt, mediaID, req.Caption, replyToMsgID)
			case models.MessageTypeVideo:
				return provider.SendVideoMessage(sendCtx, waAccount, rcpt, mediaID, req.Caption, replyToMsgID)
			case models.MessageTypeAudio:
				return provider.SendAudioMessage(sendCtx, waAccount, rcpt, mediaID, replyToMsgID)
			default: // document
				return provider.SendDocumentMessage(sendCtx, waAccount, rcpt, mediaID, req.MediaFilename, req.Caption, replyToMsgID)
			}

		case models.MessageTypeInteractive:
			switch req.InteractiveType {
			case "cta_url":
				return provider.SendCTAURLButton(sendCtx, waAccount, rcpt, req.BodyText, req.ButtonText, req.URL)
			default: // "button" or "list"
				wamid, err := provider.SendInteractiveButtons(sendCtx, waAccount, rcpt, req.BodyText, req.Buttons)
				if err == nil || !errors.Is(err, whatsapp.ErrNotSupported) {
					return wamid, err
				}
				// Provider has no native buttons — render the options into
				// the text body so the recipient can still reply.
				return provider.SendTextMessage(sendCtx, waAccount, rcpt, renderButtonsAsText(req.BodyText, req.Buttons), replyToMsgID)
			}

		case models.MessageTypeTemplate:
			if req.Template == nil {
				return "", fmt.Errorf("template is required for template messages")
			}
			return a.sendRenderedTemplate(sendCtx, provider, waAccount, rcpt, req, replyToMsgID)

		default:
			return "", fmt.Errorf("unsupported message type: %s", req.Type)
		}
	}

	// 3. Execute send (async or sync)
	if opts.Async {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			asyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			wamid, sendErr := sendFn(asyncCtx)
			a.finalizeMessageSend(msg, req, opts, wamid, sendErr)
		}()
	} else {
		wamid, err := sendFn(ctx)
		a.finalizeMessageSend(msg, req, opts, wamid, err)
	}

	// 4. Immediate actions (before send completes for async)
	if opts.BroadcastWebSocket {
		a.broadcastNewMessage(req.Account.OrganizationID, msg, req.Contact)
	}

	// Update contact's last message
	preview := a.getMessagePreview(req)
	a.updateContactLastMessage(req.Contact, preview)

	return msg, nil
}

// sendAndSaveTextMessage sends a text message and saves it to the database.
// Uses the unified SendOutgoingMessage for consistent behavior. Shared by the
// call auto-reject and CSAT close-rating flows.
func (a *App) sendAndSaveTextMessage(account *models.WhatsAppAccount, contact *models.Contact, message string) error {
	ctx := context.Background()
	_, err := a.SendOutgoingMessage(ctx, OutgoingMessageRequest{
		Account: account,
		Contact: contact,
		Type:    models.MessageTypeText,
		Content: message,
	}, AutoReplySendOptions())
	return err
}

// ============================================================================
// Internal Helpers
// ============================================================================

// toWhatsAppAccount converts models.WhatsAppAccount to whatsapp.Account
func (a *App) toWhatsAppAccount(account *models.WhatsAppAccount) *whatsapp.Account {
	return account.ToWAAccount()
}

// renderButtonsAsText renders an interactive-buttons message as plain text
// (body followed by a numbered option list) for providers without native
// reply buttons.
func renderButtonsAsText(bodyText string, buttons []whatsapp.Button) string {
	if len(buttons) == 0 {
		return bodyText
	}
	lines := make([]string, len(buttons))
	for i, btn := range buttons {
		lines[i] = fmt.Sprintf("%d. %s", i+1, btn.Title)
	}
	if bodyText == "" {
		return strings.Join(lines, "\n")
	}
	return bodyText + "\n\n" + strings.Join(lines, "\n")
}

// sendRenderedTemplate renders a local template blueprint and sends it through
// the provider as a plain text, media-with-caption, or interactive message.
// Templates are no longer submitted to a remote API — the body/header/footer
// are resolved locally from the stored template content and parameters.
func (a *App) sendRenderedTemplate(ctx context.Context, provider whatsapp.Provider, waAccount *whatsapp.Account, rcpt whatsapp.Recipient, req OutgoingMessageRequest, replyToMsgID string) (string, error) {
	tpl := req.Template

	// Render body text from params
	body := templateutil.ReplaceWithStringParams(tpl.BodyContent, req.BodyParams)

	// TEXT headers render above the body; header params fall back to body params.
	var parts []string
	if tpl.HeaderType == "TEXT" && tpl.HeaderContent != "" {
		headerParams := req.HeaderParams
		if len(headerParams) == 0 {
			headerParams = req.BodyParams
		}
		if header := templateutil.ReplaceWithStringParams(tpl.HeaderContent, headerParams); header != "" {
			parts = append(parts, "*"+header+"*")
		}
	}
	if body != "" {
		parts = append(parts, body)
	}
	if tpl.FooterContent != "" {
		parts = append(parts, tpl.FooterContent)
	}

	// Split template buttons: QUICK_REPLY becomes native reply buttons; URL
	// buttons can't be rendered natively, so append them as links in the text.
	var quickReplies []whatsapp.Button
	for i, raw := range tpl.Buttons {
		btn, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		btnType, _ := btn["type"].(string)
		label, _ := btn["text"].(string)
		switch strings.ToUpper(btnType) {
		case "QUICK_REPLY":
			quickReplies = append(quickReplies, whatsapp.Button{ID: fmt.Sprintf("btn_%d", i), Title: label})
		case "URL":
			urlStr, _ := btn["url"].(string)
			if val, exists := req.ButtonURLParams[fmt.Sprintf("%d", i)]; exists {
				urlStr = templateutil.ParameterPattern.ReplaceAllString(urlStr, val)
			}
			if urlStr != "" {
				if label != "" {
					parts = append(parts, label+": "+urlStr)
				} else {
					parts = append(parts, urlStr)
				}
			}
		}
	}

	text := strings.Join(parts, "\n\n")

	// Media header: send as media with the rendered text as caption.
	if tpl.HeaderType == "IMAGE" || tpl.HeaderType == "VIDEO" || tpl.HeaderType == "DOCUMENT" {
		mediaID := req.MediaID
		mediaData := req.MediaData
		if mediaID == "" && len(mediaData) == 0 && req.MediaURL != "" {
			// Retry/resend: read the header media back from local storage
			mediaPath := filepath.Join(a.getMediaStoragePath(), req.MediaURL)
			if fileData, readErr := os.ReadFile(mediaPath); readErr == nil && len(fileData) > 0 {
				mediaData = fileData
			}
		}
		if mediaID == "" && len(mediaData) > 0 {
			var err error
			mediaID, err = provider.UploadMedia(ctx, waAccount, mediaData, req.MediaMimeType, req.MediaFilename)
			if err != nil {
				return "", fmt.Errorf("failed to upload template header media: %w", err)
			}
		}
		if mediaID != "" {
			switch tpl.HeaderType {
			case "IMAGE":
				return provider.SendImageMessage(ctx, waAccount, rcpt, mediaID, text, replyToMsgID)
			case "VIDEO":
				return provider.SendVideoMessage(ctx, waAccount, rcpt, mediaID, text, replyToMsgID)
			default: // DOCUMENT
				return provider.SendDocumentMessage(ctx, waAccount, rcpt, mediaID, req.MediaFilename, text, replyToMsgID)
			}
		}
		// No media supplied — fall through to a plain text send.
	}

	// Quick-reply buttons via interactive send; fall back to plain text when
	// the provider doesn't support native reply buttons.
	if len(quickReplies) > 0 {
		wamid, err := provider.SendInteractiveButtons(ctx, waAccount, rcpt, text, quickReplies)
		if err == nil || !errors.Is(err, whatsapp.ErrNotSupported) {
			return wamid, err
		}
	}

	return provider.SendTextMessage(ctx, waAccount, rcpt, text, replyToMsgID)
}

// createOutgoingMessage creates a Message model from the request
func (a *App) createOutgoingMessage(req OutgoingMessageRequest, opts MessageSendOptions) *models.Message {
	msg := &models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  req.Account.OrganizationID,
		WhatsAppAccount: req.Account.Name,
		ContactID:       req.Contact.ID,
		Direction:       models.DirectionOutgoing,
		MessageType:     req.Type,
		Status:          models.MessageStatusPending,
		SentByUserID:    opts.SentByUserID,
	}

	// Set content based on message type
	switch req.Type {
	case models.MessageTypeText:
		msg.Content = req.Content

	case models.MessageTypeImage, models.MessageTypeVideo, models.MessageTypeAudio, models.MessageTypeDocument:
		msg.Content = req.Caption
		msg.MediaURL = req.MediaURL
		msg.MediaMimeType = req.MediaMimeType
		msg.MediaFilename = req.MediaFilename

	case models.MessageTypeInteractive:
		msg.Content = req.BodyText
		msg.InteractiveData = a.buildInteractiveData(req)

	case models.MessageTypeTemplate:
		if req.Template != nil {
			// Store actual rendered content instead of just template name
			content := templateutil.ReplaceWithStringParams(req.Template.BodyContent, req.BodyParams)
			if content == "" {
				content = fmt.Sprintf("[Template: %s]", req.Template.DisplayName)
			}
			msg.Content = content
			msg.TemplateName = req.Template.Name
			msg.Metadata = models.JSONB{
				"template_name": req.Template.Name,
				"template_id":   req.Template.ID.String(),
			}
			// Store header media so it renders in the chat bubble
			if req.MediaURL != "" {
				msg.MediaURL = req.MediaURL
				msg.MediaMimeType = req.MediaMimeType
			}
			// Store template buttons so they render in the chat bubble
			if len(req.Template.Buttons) > 0 {
				msg.InteractiveData = a.buildInteractiveData(req)
			}
		}
	}

	// Handle reply context
	if req.ReplyToMessage != nil {
		msg.IsReply = true
		replyID := req.ReplyToMessage.ID
		msg.ReplyToMessageID = &replyID
	}

	return msg
}

// buildInteractiveData creates the InteractiveData JSONB for interactive and template messages
func (a *App) buildInteractiveData(req OutgoingMessageRequest) models.JSONB {
	// Template buttons: stored as JSONBArray on Template.Buttons
	// Resolve dynamic URL params (e.g., {{1}}) before storing
	if req.Template != nil && len(req.Template.Buttons) > 0 {
		buttons := make([]any, len(req.Template.Buttons))
		for i, btn := range req.Template.Buttons {
			btnMap, ok := btn.(map[string]any)
			if !ok {
				buttons[i] = btn
				continue
			}
			resolved := make(map[string]any, len(btnMap))
			maps.Copy(resolved, btnMap)
			if resolved["type"] == "URL" {
				if urlStr, ok := resolved["url"].(string); ok {
					idx := fmt.Sprintf("%d", i)
					if val, exists := req.ButtonURLParams[idx]; exists {
						resolved["url"] = templateutil.ParameterPattern.ReplaceAllString(urlStr, val)
					}
				}
			}
			buttons[i] = resolved
		}
		return models.JSONB{
			"type":    "button",
			"buttons": buttons,
		}
	}

	switch req.InteractiveType {
	case "cta_url":
		return models.JSONB{
			"type":        "cta_url",
			"body":        req.BodyText,
			"button_text": req.ButtonText,
			"url":         req.URL,
		}
	case "list":
		rows := make([]any, len(req.Buttons))
		for i, btn := range req.Buttons {
			rows[i] = map[string]string{"id": btn.ID, "title": btn.Title}
		}
		return models.JSONB{
			"type": "list",
			"body": req.BodyText,
			"rows": rows,
		}
	default: // "button"
		buttons := make([]any, len(req.Buttons))
		for i, btn := range req.Buttons {
			buttons[i] = map[string]string{"id": btn.ID, "title": btn.Title}
		}
		return models.JSONB{
			"type":    "button",
			"body":    req.BodyText,
			"buttons": buttons,
		}
	}
}

// finalizeMessageSend updates message status and triggers post-send actions
func (a *App) finalizeMessageSend(msg *models.Message, req OutgoingMessageRequest, opts MessageSendOptions, wamid string, err error) {
	// Use Where instead of Model(msg) to avoid mutating the shared msg struct,
	// which may be read concurrently by the caller when sending is async.
	if err != nil {
		errMsg := err.Error()

		a.DB.Model(&models.Message{}).Where("id = ?", msg.ID).Updates(map[string]any{
			"status":        models.MessageStatusFailed,
			"error_message": errMsg,
		})
		a.Log.Error("Failed to send message", "error", err, "message_id", msg.ID, "type", msg.MessageType)

		// Broadcast failure status via WebSocket so frontend updates immediately
		if opts.BroadcastWebSocket && a.WSHub != nil {
			a.WSHub.BroadcastToOrg(req.Account.OrganizationID, websocket.WSMessage{
				Type: websocket.TypeStatusUpdate,
				Payload: map[string]any{
					"message_id":    msg.ID,
					"contact_id":    req.Contact.ID,
					"status":        models.MessageStatusFailed,
					"error_message": errMsg,
				},
			})
		}
		return
	}

	a.DB.Model(&models.Message{}).Where("id = ?", msg.ID).Updates(map[string]any{
		"status":               models.MessageStatusSent,
		"whats_app_message_id": wamid,
	})
	a.Log.Info("Message sent", "message_id", msg.ID, "wa_message_id", wamid, "type", msg.MessageType)

	// Dispatch webhook for successful send
	if opts.DispatchWebhook {
		a.dispatchMessageSentWebhook(req.Account, req.Contact, msg)
	}

	// Broadcast status update via WebSocket
	if opts.BroadcastWebSocket && a.WSHub != nil {
		a.WSHub.BroadcastToOrg(req.Account.OrganizationID, websocket.WSMessage{
			Type: websocket.TypeStatusUpdate,
			Payload: map[string]any{
				"message_id": msg.ID,
				"contact_id": req.Contact.ID,
				"status":     models.MessageStatusSent,
				"wamid":      wamid,
			},
		})
	}

	// Mark the contact's incoming messages as read once an automated reply has
	// gone out. Keeps the agent's contact-list unread count clean for
	// conversations handled automatically. See issue #280.
	if opts.MarkIncomingRead {
		a.markMessagesAsRead(req.Account.OrganizationID, req.Contact.ID, req.Contact)
	}
}

// broadcastNewMessage broadcasts a new message via WebSocket
func (a *App) broadcastNewMessage(orgID uuid.UUID, msg *models.Message, contact *models.Contact) {
	if a.WSHub == nil {
		return
	}

	var assignedUserIDStr string
	if contact.AssignedUserID != nil {
		assignedUserIDStr = contact.AssignedUserID.String()
	}
	profileName := contact.ProfileName
	if a.ShouldMaskPhoneNumbers(orgID) {
		profileName = utils.MaskIfPhoneNumber(profileName)
	}

	payload := map[string]any{
		"id":               msg.ID.String(),
		"contact_id":       contact.ID.String(),
		"assigned_user_id": assignedUserIDStr,
		"profile_name":     profileName,
		"direction":        msg.Direction,
		"message_type":     msg.MessageType,
		"content":          map[string]string{"body": msg.Content},
		"media_url":        msg.MediaURL,
		"media_mime_type":  msg.MediaMimeType,
		"media_filename":   msg.MediaFilename,
		"interactive_data": msg.InteractiveData,
		"status":           msg.Status,
		"wamid":            msg.WhatsAppMessageID,
		// The account the message belongs to — the frontend's live-append
		// guard needs it to keep messages out of the wrong account tab.
		"whatsapp_account": msg.WhatsAppAccount,
		"created_at":       msg.CreatedAt,
		"updated_at":       msg.UpdatedAt,
		"is_reply":         msg.IsReply,
		"is_group_chat":    contact.Metadata != nil && contact.Metadata["is_group_chat"] == true,
		"is_newsletter":    contact.Metadata != nil && contact.Metadata["is_newsletter"] == true,
	}

	// Per-message sender for group conversations (empty for 1:1).
	if msg.Metadata != nil {
		payload["metadata"] = msg.Metadata
		if v, ok := msg.Metadata["sender_phone"]; ok {
			payload["sender_phone"] = v
		}
		if v, ok := msg.Metadata["sender_push_name"]; ok {
			payload["sender_push_name"] = v
		}
	}

	// Add interactive data
	if msg.InteractiveData != nil {
		payload["interactive_data"] = msg.InteractiveData
	}

	// Add reply context
	if msg.IsReply && msg.ReplyToMessageID != nil {
		payload["reply_to_message_id"] = msg.ReplyToMessageID.String()

		// Include reply preview for UI
		var replyToMsg models.Message
		if err := a.DB.First(&replyToMsg, msg.ReplyToMessageID).Error; err == nil {
			payload["reply_to_message"] = map[string]any{
				"id":           replyToMsg.ID.String(),
				"content":      map[string]string{"body": replyToMsg.Content},
				"message_type": replyToMsg.MessageType,
				"direction":    replyToMsg.Direction,
			}
		}
	}

	a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type:    websocket.TypeNewMessage,
		Payload: payload,
	})
}

// broadcastReactionUpdate broadcasts a reaction update via WebSocket
func (a *App) broadcastReactionUpdate(orgID uuid.UUID, messageID, contactID uuid.UUID, reactions any) {
	if a.WSHub == nil {
		return
	}
	a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type: "reaction_update",
		Payload: map[string]any{
			"message_id": messageID.String(),
			"contact_id": contactID.String(),
			"reactions":  reactions,
		},
	})
}

// dispatchMessageSentWebhook dispatches webhook for message.sent event
func (a *App) dispatchMessageSentWebhook(account *models.WhatsAppAccount, contact *models.Contact, msg *models.Message) {
	var sentByUserID string
	if msg.SentByUserID != nil {
		sentByUserID = msg.SentByUserID.String()
	}

	a.DispatchWebhook(account.OrganizationID, models.WebhookEventMessageSent, MessageEventData{
		MessageID:       msg.ID.String(),
		ContactID:       contact.ID.String(),
		ContactPhone:    contact.PhoneNumber,
		ContactName:     contact.ProfileName,
		MessageType:     msg.MessageType,
		Content:         msg.Content,
		WhatsAppAccount: account.Name,
		Direction:       models.DirectionOutgoing,
		SentByUserID:    sentByUserID,
	})
}

// updateContactLastMessage updates contact's last_message_at and preview
func (a *App) updateContactLastMessage(contact *models.Contact, preview string) {
	a.DB.Model(contact).Updates(map[string]any{
		"last_message_at":      time.Now(),
		"last_message_preview": preview,
	})
}

// getMessagePreview returns a preview string for the message
func (a *App) getMessagePreview(req OutgoingMessageRequest) string {
	switch req.Type {
	case models.MessageTypeText:
		return truncateString(req.Content, 100)
	case models.MessageTypeImage:
		if req.Caption != "" {
			return truncateString(req.Caption, 100)
		}
		return "[Image]"
	case models.MessageTypeVideo:
		if req.Caption != "" {
			return truncateString(req.Caption, 100)
		}
		return "[Video]"
	case models.MessageTypeAudio:
		return "[Audio]"
	case models.MessageTypeDocument:
		if req.MediaFilename != "" {
			return "[Document: " + req.MediaFilename + "]"
		}
		return "[Document]"
	case models.MessageTypeInteractive:
		return truncateString(req.BodyText, 100)
	case models.MessageTypeTemplate:
		if req.Template != nil {
			return fmt.Sprintf("[Template: %s]", req.Template.DisplayName)
		}
		return "[Template]"
	default:
		return "[Message]"
	}
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// SendTemplateMessageRequest represents the request to send a template message
type SendTemplateMessageRequest struct {
	ContactID      string            `json:"contact_id"`
	PhoneNumber    string            `json:"phone_number"`    // Alternative to contact_id - send to phone directly
	TemplateName   string            `json:"template_name"`   // Template name
	TemplateID     string            `json:"template_id"`     // Alternative: template UUID
	TemplateParams map[string]string `json:"template_params"` // Named or positional params
	ButtonParams   map[string]string `json:"button_params"`   // Button index -> dynamic URL param value
	AccountName    string            `json:"account_name"`    // Optional: specific WhatsApp account

	// Header media for templates with IMAGE/VIDEO/DOCUMENT headers.
	// Two options (in priority order):
	//   1. header_media_url — URL to fetch the media from (server downloads it)
	//   2. multipart header_file — raw file upload via multipart/form-data
	HeaderMediaURL      string `json:"header_media_url"`      // URL to download media from
	HeaderMediaFilename string `json:"header_media_filename"` // Filename for DOCUMENT headers

	// Header text parameter values for TEXT headers that contain a {{var}}.
	// Only one variable is permitted in a TEXT header. Keyed by the variable's
	// name (named templates) or by "1" (positional). Optional — if absent, the
	// value is looked up in TemplateParams as a fallback.
	HeaderParams map[string]string `json:"header_params"`
}

// SendTemplateMessage sends a template message to a contact or phone number.
// Accepts either JSON body or multipart/form-data (when a header media file is included).
func (a *App) SendTemplateMessage(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChat, models.ActionWrite)
	if err != nil {
		return nil
	}

	var req SendTemplateMessageRequest
	var headerFileData []byte
	var headerFileMimeType string
	var headerFileFilename string

	contentType := string(r.RequestCtx.Request.Header.ContentType())
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Parse multipart form — used when template has a media header
		form, err := r.RequestCtx.MultipartForm()
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid multipart form", nil, "")
		}
		if v := form.Value["contact_id"]; len(v) > 0 {
			req.ContactID = v[0]
		}
		if v := form.Value["phone_number"]; len(v) > 0 {
			req.PhoneNumber = v[0]
		}
		if v := form.Value["template_name"]; len(v) > 0 {
			req.TemplateName = v[0]
		}
		if v := form.Value["template_id"]; len(v) > 0 {
			req.TemplateID = v[0]
		}
		if v := form.Value["account_name"]; len(v) > 0 {
			req.AccountName = v[0]
		}
		// Parse template_params from JSON string
		if v := form.Value["template_params"]; len(v) > 0 && v[0] != "" {
			if err := json.Unmarshal([]byte(v[0]), &req.TemplateParams); err != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid template_params JSON", nil, "")
			}
		}
		// Parse button_params from JSON string
		if v := form.Value["button_params"]; len(v) > 0 && v[0] != "" {
			if err := json.Unmarshal([]byte(v[0]), &req.ButtonParams); err != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid button_params JSON", nil, "")
			}
		}
		// Parse header_params from JSON string
		if v := form.Value["header_params"]; len(v) > 0 && v[0] != "" {
			if err := json.Unmarshal([]byte(v[0]), &req.HeaderParams); err != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid header_params JSON", nil, "")
			}
		}
		// Read header media file
		if files := form.File["header_file"]; len(files) > 0 {
			fh := files[0]
			f, err := fh.Open()
			if err != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to read header file", nil, "")
			}
			defer f.Close() //nolint:errcheck
			headerFileData, err = io.ReadAll(f)
			if err != nil {
				a.Log.Error("Failed to read header file", "error", err)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read header file", nil, "")
			}
			headerFileMimeType = fh.Header.Get("Content-Type")
			if headerFileMimeType == "" {
				headerFileMimeType = "application/octet-stream"
			}
			headerFileFilename = fh.Filename
		}
		if v := form.Value["header_media_filename"]; len(v) > 0 {
			req.HeaderMediaFilename = v[0]
		}
	} else {
		if err := a.decodeRequest(r, &req); err != nil {
			return nil
		}
	}

	// Must have either contact_id or phone_number
	if req.ContactID == "" && req.PhoneNumber == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Either contact_id or phone_number is required", nil, "")
	}

	// Must have either template_name or template_id
	if req.TemplateName == "" && req.TemplateID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Either template_name or template_id is required", nil, "")
	}

	// Get template
	var template models.Template
	if req.TemplateID != "" {
		templateID, err := uuid.Parse(req.TemplateID)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid template_id", nil, "")
		}
		t, err := findByIDAndOrg[models.Template](a.DB, r, templateID, orgID, "Template")
		if err != nil {
			return nil
		}
		template = *t
	} else {
		if err := a.DB.Where("name = ? AND organization_id = ?", req.TemplateName, orgID).First(&template).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Template not found", nil, "")
		}
	}

	// Get contact or use phone number directly
	var contact *models.Contact

	if req.ContactID != "" {
		cID, err := uuid.Parse(req.ContactID)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid contact_id", nil, "")
		}
		c, err := findByIDAndOrg[models.Contact](a.DB, r, cID, orgID, "Contact")
		if err != nil {
			return nil
		}
		contact = c
	} else {
		// Find or create contact from phone number
		phoneNumber := req.PhoneNumber
		var c models.Contact
		err := a.DB.Where("phone_number = ? AND organization_id = ?", phoneNumber, orgID).First(&c).Error
		if err != nil {
			// Contact not found, create new one
			c = models.Contact{
				BaseModel:      models.BaseModel{ID: uuid.New()},
				OrganizationID: orgID,
				PhoneNumber:    phoneNumber,
			}
			if err := a.DB.Create(&c).Error; err != nil {
				a.Log.Error("Failed to create contact", "error", err, "phone", phoneNumber)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create contact", nil, "")
			}
			a.Log.Info("Contact created from API", "contact_id", c.ID, "phone", phoneNumber)
		}
		contact = &c
	}

	// Determine which WhatsApp account to use (explicit > template > contact > default)
	accountName := req.AccountName
	if accountName == "" {
		accountName = template.WhatsAppAccount
	}
	if accountName == "" && contact != nil {
		accountName = contact.WhatsAppAccount
	}

	account, err := a.resolveWhatsAppAccount(orgID, accountName)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	// Extract parameter names and resolve values
	paramNames := templateutil.ExtParamNames(template.BodyContent)
	bodyParams := templateutil.ResolveParamsFromMap(paramNames, req.TemplateParams)

	// Validate that all required parameters are provided
	if len(paramNames) > 0 {
		var missingParams []string
		for i, name := range paramNames {
			if i >= len(bodyParams) || bodyParams[i] == "" {
				missingParams = append(missingParams, name)
			}
		}
		if len(missingParams) > 0 {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
				fmt.Sprintf("Missing template parameters: %s. Expected parameters: %v", strings.Join(missingParams, ", "), paramNames),
				nil, "")
		}
	}

	// Validate the header variable (TEXT headers only). At most one variable is
	// allowed in a TEXT header — surface a clean 400 if it's missing.
	if template.HeaderType == "TEXT" {
		headerNames := templateutil.ExtParamNames(template.HeaderContent)
		if len(headerNames) > 1 {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
				fmt.Sprintf("Template header text contains %d variables; at most 1 is allowed", len(headerNames)),
				nil, "")
		}
		if len(headerNames) == 1 {
			name := headerNames[0]
			if req.HeaderParams[name] == "" && req.TemplateParams[name] == "" {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
					fmt.Sprintf("Missing header parameter %q. Pass it in header_params or template_params.", name),
					nil, "")
			}
		}
	}

	// Resolve header media for templates with IMAGE/VIDEO/DOCUMENT headers.
	// Priority: header_media_url > multipart header_file. The raw bytes are
	// passed through to the unified sender, which uploads them via the provider.
	var headerMediaData []byte
	var headerMimeType string
	if template.HeaderType == "IMAGE" || template.HeaderType == "VIDEO" || template.HeaderType == "DOCUMENT" {
		if req.HeaderMediaURL != "" {
			// SSRF guard (H1): validate URL shape + block internal IPs/hosts before
			// fetch, then go through the shared SSRF-safe client (re-resolves DNS at
			// dial time to defeat rebinding). Mirrors webhook/custom-action fetches.
			if err := validateWebhookURL(req.HeaderMediaURL); err != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
					fmt.Sprintf("Invalid header media URL: %v", err), nil, "")
			}

			httpReq, err := http.NewRequestWithContext(r.RequestCtx, http.MethodGet, req.HeaderMediaURL, nil)
			if err != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid header media URL", nil, "")
			}
			resp, err := a.HTTPClient.Do(httpReq)
			if err != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to download header media from URL", nil, "")
			}
			defer resp.Body.Close() //nolint:errcheck
			if resp.StatusCode != http.StatusOK {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, fmt.Sprintf("Header media URL returned status %d", resp.StatusCode), nil, "")
			}
			// Cap the download (H1 + DoS): same 16MB ceiling used by campaign media uploads.
			headerMediaData, err = io.ReadAll(io.LimitReader(resp.Body, maxHeaderMediaBytes))
			if err != nil {
				a.Log.Error("Failed to read header media from URL", "error", err)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read header media from URL", nil, "")
			}
			headerMimeType = resp.Header.Get("Content-Type")
			if headerMimeType == "" {
				headerMimeType = "application/octet-stream"
			}
		} else if len(headerFileData) > 0 {
			// Option 2: Multipart file upload
			headerMediaData = headerFileData
			headerMimeType = headerFileMimeType
		}
	}

	// Save header media locally so it can be served for chat preview
	var headerLocalPath string
	if len(headerMediaData) > 0 {
		localPath, err := a.saveMediaLocally(headerMediaData, headerMimeType, "header")
		if err != nil {
			a.Log.Error("Failed to save template header media locally", "error", err)
			// Non-fatal — message will still send, just won't show preview
		} else {
			headerLocalPath = localPath
		}
	}

	// Check marketing opt-out
	if contact.MarketingOptOut && strings.EqualFold(template.Category, "MARKETING") {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Contact has opted out of marketing messages", nil, "")
	}

	// Resolve filename for DOCUMENT headers — caller-supplied wins, then the
	// multipart filename.
	headerMediaFilename := req.HeaderMediaFilename
	if headerMediaFilename == "" {
		headerMediaFilename = headerFileFilename
	}

	// Send using unified message sender
	msgReq := OutgoingMessageRequest{
		Account:         account,
		Contact:         contact,
		Type:            models.MessageTypeTemplate,
		Template:        &template,
		BodyParams:      req.TemplateParams,
		HeaderParams:    req.HeaderParams,
		MediaData:       headerMediaData,
		MediaURL:        headerLocalPath,
		MediaMimeType:   headerMimeType,
		MediaFilename:   headerMediaFilename,
		ButtonURLParams: req.ButtonParams,
	}

	opts := DefaultSendOptions()
	opts.SentByUserID = &userID

	ctx := context.Background()
	message, err := a.SendOutgoingMessage(ctx, msgReq, opts)
	if err != nil {
		a.Log.Error("Failed to send template message", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to send template message", nil, "")
	}

	// Build full message response (same shape as SendMessage)
	response := MessageResponse{
		ID:              message.ID,
		ContactID:       message.ContactID,
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
	return r.SendEnvelope(response)
}


// ============================================================================
// Contact message handlers (split out of contacts.go): list / send / react /
// typing / revoke / read-state. Same package (handlers), so routing in
// main.go (app.GetMessages / app.SendMessage / ...) is unchanged.
// ============================================================================

type MessageResponse struct {
	ID               uuid.UUID            `json:"id"`
	ContactID        uuid.UUID            `json:"contact_id"`
	Direction        models.Direction     `json:"direction"`
	MessageType      models.MessageType   `json:"message_type"`
	Content          any                  `json:"content"`
	MediaURL         string               `json:"media_url,omitempty"`
	MediaMimeType    string               `json:"media_mime_type,omitempty"`
	MediaFilename    string               `json:"media_filename,omitempty"`
	InteractiveData  models.JSONB         `json:"interactive_data,omitempty"`
	Status           models.MessageStatus `json:"status"`
	WAMID            string               `json:"wamid"`
	Error            string               `json:"error_message"`
	IsReply          bool                 `json:"is_reply"`
	ReplyToMessageID *string              `json:"reply_to_message_id,omitempty"`
	ReplyToMessage   *ReplyPreview        `json:"reply_to_message,omitempty"`
	Reactions        []ReactionInfo       `json:"reactions,omitempty"`
	WhatsAppAccount  string               `json:"whatsapp_account,omitempty"`
	IsGroupChat      bool                 `json:"is_group_chat"`
	IsNewsletter     bool                 `json:"is_newsletter"`
	SenderPhone      string               `json:"sender_phone,omitempty"`
	SenderPushName   string               `json:"sender_push_name,omitempty"`
	Metadata         models.JSONB         `json:"metadata,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

type ReplyPreview struct {
	ID          string             `json:"id"`
	Content     any                `json:"content"`
	MessageType models.MessageType `json:"message_type"`
	Direction   models.Direction   `json:"direction"`
}

func (a *App) GetMessages(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChat, models.ActionRead)
	if err != nil {
		return nil
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	// Verify contact belongs to org (and to user if no contacts:read permission)
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Managers/admins (contacts:write) can see any chat — no restrictions.
	// Agents can only see messages if: they own it, are a collaborator, or have collaborate permission.
	// Closed conversations are readable by everyone (read-only).
	hasContactsWritePermission := a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID)
	hasCollaboratePermission := a.HasPermission(userID, models.ResourceChatCollaborate, models.ActionWrite, orgID)
	isAssigned := contact.AssignedUserID != nil && *contact.AssignedUserID == userID
	isCollaborator := contact.IsCollaborator(userID.String())
	canViewContent := hasContactsWritePermission || isAssigned || isCollaborator || hasCollaboratePermission

	if !canViewContent && contact.EffectiveStatus() == models.ChatStatusPending {
		var pendingCount int64
		a.DB.Model(&models.Message{}).
			Where("contact_id = ? AND direction = ? AND status != ?",
				contactID, models.DirectionIncoming, models.MessageStatusRead).
			Count(&pendingCount)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden,
			"Claim this chat to view messages",
			map[string]any{"pending_message_count": pendingCount},
			"chat_not_claimed")
	}
	// ─── End privacy guard ───

	// Pagination parameters
	limit, _ := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("limit")))
	beforeIDStr := string(r.RequestCtx.QueryArgs().Peek("before_id"))

	if limit < 1 || limit > 100 {
		limit = 50
	}

	// Build base query
	msgQuery := a.DB.Where("contact_id = ?", contactID)

	// Filter by WhatsApp account if specified. System messages (claim/close/
	// release/reopen notifications) are conversation-level events written
	// without a WhatsApp account, so they must be exempt from the account
	// filter — otherwise they vanish on refresh whenever an account is
	// selected (multi-account org, switchAccount, or older-message paging).
	accountFilter := string(r.RequestCtx.QueryArgs().Peek("account"))
	if accountFilter != "" {
		if mirrorIDs := a.crossAccountMirrorContactIDs(orgID, accountFilter, &contact); len(mirrorIDs) > 0 {
			// The selected tab's account IS this contact (the page shows one of
			// the org's own numbers). That account's copies of the cross-account
			// conversation are stored under the counterpart contacts of the
			// *other* org accounts, so surface those instead of the always-empty
			// self view — while keeping this page's own system messages.
			msgQuery = a.DB.Where(
				"(contact_id IN ? AND whats_app_account = ?) OR (contact_id = ? AND metadata->>'is_system_message' = 'true')",
				mirrorIDs, accountFilter, contactID)
		} else {
			msgQuery = msgQuery.Where(
				"whats_app_account = ? OR metadata->>'is_system_message' = 'true'",
				accountFilter)
		}
	}

	// Count total messages
	var total int64
	msgQuery.Model(&models.Message{}).Count(&total)

	// Cursor-based pagination: load messages before a specific ID
	if beforeIDStr != "" {
		beforeID, err := uuid.Parse(beforeIDStr)
		if err == nil {
			// Get the created_at of the before_id message
			var beforeMsg models.Message
			if err := a.DB.Where("id = ?", beforeID).First(&beforeMsg).Error; err == nil {
				msgQuery = msgQuery.Where("created_at < ?", beforeMsg.CreatedAt)
			}
		}
		// For loading older messages, order DESC and limit, then reverse
		var messages []models.Message
		if err := msgQuery.Preload("ReplyToMessage").Order("created_at DESC").Limit(limit).Find(&messages).Error; err != nil {
			a.Log.Error("Failed to list messages", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list messages", nil, "")
		}
		// Reverse to get chronological order
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}

		response := a.buildMessagesResponse(messages)
		return r.SendEnvelope(map[string]any{
			"messages": response,
			"total":    total,
			"has_more": len(messages) == limit,
		})
	}

	// Default: load most recent messages (page 1)
	page, _ := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
	if page < 1 {
		page = 1
	}

	// For chat, we want the most recent messages
	// Calculate offset from the end for pagination
	// Preserve the original limit for the response; adjust a query-specific limit
	// when the remaining messages are fewer than the requested page size.
	responseLimit := limit
	queryLimit := limit
	offset := int(total) - (page * limit)
	if offset < 0 {
		queryLimit = limit + offset // Adjust limit if we're on the last page
		offset = 0
	}

	var messages []models.Message
	if err := msgQuery.Preload("ReplyToMessage").Order("created_at ASC").Offset(offset).Limit(queryLimit).Find(&messages).Error; err != nil {
		a.Log.Error("Failed to list messages", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list messages", nil, "")
	}

	// Mark messages as read
	a.markMessagesAsRead(orgID, contactID, &contact)

	response := a.buildMessagesResponse(messages)
	return r.SendEnvelope(map[string]any{
		"messages": response,
		"total":    total,
		"page":     page,
		"limit":    responseLimit,
		"has_more": offset > 0,
	})
}

// crossAccountMirrorContactIDs resolves the "mirror" view for cross-account
// conversations: when the requested account tab's own connected number equals
// the page contact's phone (two org accounts messaging each other), that
// account's copies of the thread are stored under the counterpart contacts of
// the other org accounts — not under this contact. It returns those
// counterpart contact IDs, or nil when this is a normal (non-self) tab.
func (a *App) crossAccountMirrorContactIDs(orgID uuid.UUID, accountName string, contact *models.Contact) []uuid.UUID {
	var account models.WhatsAppAccount
	if err := a.DB.Where("organization_id = ? AND name = ?", orgID, accountName).
		First(&account).Error; err != nil {
		return nil
	}
	if account.GowaJID == "" || gowa.PhoneFromJID(account.GowaJID) != contact.PhoneNumber {
		return nil
	}

	// The peers are the org's other connected numbers.
	var others []models.WhatsAppAccount
	if err := a.DB.Where("organization_id = ? AND name != ? AND gowa_jid != ''", orgID, accountName).
		Find(&others).Error; err != nil || len(others) == 0 {
		return nil
	}
	phones := make([]string, 0, len(others))
	for _, o := range others {
		phones = append(phones, gowa.PhoneFromJID(o.GowaJID))
	}

	var ids []uuid.UUID
	if err := a.DB.Model(&models.Contact{}).
		Where("organization_id = ? AND phone_number IN ?", orgID, phones).
		Pluck("id", &ids).Error; err != nil {
		return nil
	}
	return ids
}

// buildMessagesResponse converts messages to response format
func (a *App) buildMessagesResponse(messages []models.Message) []MessageResponse {
	response := make([]MessageResponse, len(messages))
	for i, m := range messages {
		var content any
		if m.MessageType == models.MessageTypeText {
			content = map[string]string{"body": m.Content}
		} else {
			content = map[string]string{"body": m.Content}
		}

		msgResp := MessageResponse{
			ID:              m.ID,
			ContactID:       m.ContactID,
			Direction:       m.Direction,
			MessageType:     m.MessageType,
			Content:         content,
			MediaURL:        m.MediaURL,
			MediaMimeType:   m.MediaMimeType,
			MediaFilename:   m.MediaFilename,
			InteractiveData: m.InteractiveData,
			Status:          m.Status,
			WAMID:           m.WhatsAppMessageID,
			Error:           m.ErrorMessage,
			IsReply:         m.IsReply,
			WhatsAppAccount: m.WhatsAppAccount,
			IsGroupChat:     m.Metadata != nil && m.Metadata["is_group_chat"] == true,
			IsNewsletter:    m.Metadata != nil && m.Metadata["is_newsletter"] == true,
			SenderPhone:     metadataString(m.Metadata, "sender_phone"),
			SenderPushName:  metadataString(m.Metadata, "sender_push_name"),
			Metadata:        m.Metadata,
			CreatedAt:       m.CreatedAt,
			UpdatedAt:       m.UpdatedAt,
		}

		if m.IsReply && m.ReplyToMessageID != nil {
			replyToID := m.ReplyToMessageID.String()
			msgResp.ReplyToMessageID = &replyToID
			if m.ReplyToMessage != nil {
				msgResp.ReplyToMessage = &ReplyPreview{
					ID:          m.ReplyToMessage.ID.String(),
					Content:     map[string]string{"body": m.ReplyToMessage.Content},
					MessageType: m.ReplyToMessage.MessageType,
					Direction:   m.ReplyToMessage.Direction,
				}
			}
		}

		if m.Metadata != nil {
			if reactionsRaw, ok := m.Metadata["reactions"]; ok {
				if reactionsArray, ok := reactionsRaw.([]any); ok {
					for _, r := range reactionsArray {
						if rMap, ok := r.(map[string]any); ok {
							emoji, _ := rMap["emoji"].(string)
							fromPhone, _ := rMap["from_phone"].(string)
							fromUser, _ := rMap["from_user"].(string)
							msgResp.Reactions = append(msgResp.Reactions, ReactionInfo{
								Emoji:     emoji,
								FromPhone: fromPhone,
								FromUser:  fromUser,
							})
						}
					}
				}
			}
		}

		response[i] = msgResp
	}
	return response
}

// MarkContactRead marks all incoming messages from a contact as read.
// Called from the frontend when a new message arrives for the chat the
// user is currently viewing, so the sidebar unread badge stays at zero.
func (a *App) MarkContactRead(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChat, models.ActionRead)
	if err != nil {
		return nil
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Mirror the GetMessages privacy guard: a caller who can't view the
	// content of a pending unclaimed chat must not mark it read either —
	// doing so would clear the unread badge and fire read receipts (blue
	// ticks) for messages nobody has actually seen.
	hasContactsWritePermission := a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID)
	hasCollaboratePermission := a.HasPermission(userID, models.ResourceChatCollaborate, models.ActionWrite, orgID)
	isAssigned := contact.AssignedUserID != nil && *contact.AssignedUserID == userID
	isCollaborator := contact.IsCollaborator(userID.String())
	canViewContent := hasContactsWritePermission || isAssigned || isCollaborator || hasCollaboratePermission

	if !canViewContent && contact.EffectiveStatus() == models.ChatStatusPending {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden,
			"Claim this chat to view messages", nil, "chat_not_claimed")
	}

	a.markMessagesAsRead(orgID, contactID, &contact)
	return r.SendEnvelope(map[string]any{"status": "ok"})
}

// markMessagesAsRead marks messages as read and sends read receipts
func (a *App) markMessagesAsRead(orgID uuid.UUID, contactID uuid.UUID, contact *models.Contact) {
	var unreadMessages []models.Message
	a.DB.Where("contact_id = ? AND direction = ? AND status != ?", contactID, models.DirectionIncoming, models.MessageStatusRead).
		Find(&unreadMessages)

	a.DB.Model(&models.Message{}).
		Where("contact_id = ? AND direction = ?", contactID, models.DirectionIncoming).
		Update("status", models.MessageStatusRead)

	a.DB.Model(contact).Update("is_read", true)

	if len(unreadMessages) > 0 && contact.WhatsAppAccount != "" {
		if account, err := a.resolveWhatsAppAccount(orgID, contact.WhatsAppAccount); err == nil {
			if account.AutoReadReceipt {
				a.wg.Add(1)
				go func() {
					defer a.wg.Done()
					// Use timeout context for external API calls
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					waAccount := a.toWhatsAppAccount(account)
					for _, msg := range unreadMessages {
						// Check if context was cancelled
						if ctx.Err() != nil {
							a.Log.Warn("Read receipt sending cancelled", "reason", ctx.Err())
							return
						}
						if msg.WhatsAppMessageID != "" {
							provider := a.resolveProvider(account)
							// GOWA requires the chat JID for read receipts;
							// the Provider interface's MarkMessageRead lacks it.
							if gc, ok := provider.(*gowa.Client); ok {
								// Build the chat JID (handles group @g.us vs 1:1 suffix).
								chatJID := gowaChatJID(contact)
								if err := gc.MarkMessageReadWithJID(ctx, waAccount, msg.WhatsAppMessageID, chatJID); err != nil {
									a.Log.Error("Failed to send GOWA read receipt", "error", err, "message_id", msg.WhatsAppMessageID)
								}
							}
						}
					}
				}()
			}
		}
	}
}

// SendMessageRequest represents a send message request
type SendMessageRequest struct {
	Type    models.MessageType `json:"type"`
	Content struct {
		Body string `json:"body"`
		// Media fields (used on retry/resend of media messages)
		MediaData     string `json:"media_data,omitempty"`
		MediaMimeType string `json:"media_mime_type,omitempty"`
		MediaFilename string `json:"media_filename,omitempty"`
		MediaURL      string `json:"media_url,omitempty"`
	} `json:"content"`
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
	ButtonText string          `json:"button_text,omitempty"` // CTA label for cta_url
	URL        string          `json:"url,omitempty"`         // For cta_url type
}

// ButtonContent represents a button in interactive messages
type ButtonContent struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// SendMessage sends a message to a contact
// Agents can only send messages to their assigned contacts
func (a *App) SendMessage(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChat, models.ActionWrite)
	if err != nil {
		return nil
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
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Get WhatsApp account - prefer request-specified account over contact default
	accountName := contact.WhatsAppAccount
	if req.WhatsAppAccount != "" {
		accountName = req.WhatsAppAccount
	}
	account, err := a.resolveWhatsAppAccount(orgID, accountName)
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
		Type:           req.Type,
		Content:        req.Content.Body,
		ReplyToMessage: replyToMessage,
	}

	// Wire media fields for image/video/audio/document sends and retries.
	if req.Content.MediaData != "" {
		if decoded, decErr := base64.StdEncoding.DecodeString(req.Content.MediaData); decErr == nil {
			msgReq.MediaData = decoded
		}
	}
	msgReq.MediaMimeType = req.Content.MediaMimeType
	msgReq.MediaFilename = req.Content.MediaFilename
	msgReq.MediaURL = req.Content.MediaURL

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
		a.Log.Error("Failed to send message", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to send message", nil, "")
	}

	// Build response
	response := MessageResponse{
		ID:              message.ID,
		ContactID:       message.ContactID,
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

	// Add reply context to response
	if message.IsReply && message.ReplyToMessageID != nil && replyToMessage != nil {
		replyToID := message.ReplyToMessageID.String()
		response.ReplyToMessageID = &replyToID
		response.ReplyToMessage = &ReplyPreview{
			ID:          replyToMessage.ID.String(),
			Content:     map[string]string{"body": replyToMessage.Content},
			MessageType: replyToMessage.MessageType,
			Direction:   replyToMessage.Direction,
		}
	}

	return r.SendEnvelope(response)
}

// resolveWhatsAppAccount gets the WhatsApp account for sending messages
func (a *App) resolveWhatsAppAccount(orgID uuid.UUID, accountName string) (*models.WhatsAppAccount, error) {
	var account models.WhatsAppAccount

	if accountName != "" {
		if err := a.DB.Where("name = ? AND organization_id = ?", accountName, orgID).First(&account).Error; err != nil {
			return nil, fmt.Errorf("WhatsApp account not found")
		}
		a.decryptAccountSecrets(&account)
		return &account, nil
	}

	// Get default outgoing account
	if err := a.DB.Where("organization_id = ? AND is_default_outgoing = ?", orgID, true).First(&account).Error; err != nil {
		// Fall back to any account
		if err := a.DB.Where("organization_id = ?", orgID).First(&account).Error; err != nil {
			return nil, fmt.Errorf("no WhatsApp account configured")
		}
	}
	a.decryptAccountSecrets(&account)
	return &account, nil
}

// resolveProvider returns the WhatsApp provider (GOWA) for the given account.
func (a *App) resolveProvider(account *models.WhatsAppAccount) whatsapp.Provider {
	if a.WARegistry != nil && account != nil {
		return a.WARegistry.Get(account.ToWAAccount())
	}
	return nil
}

// resolveWhatsAppAccountByID fetches a WhatsApp account by UUID and org, decrypts secrets.
func (a *App) resolveWhatsAppAccountByID(r *fastglue.Request, id, orgID uuid.UUID) (*models.WhatsAppAccount, error) {
	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil, err
	}
	a.decryptAccountSecrets(account)
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
	orgID, userID, err := a.requireAuth(r, models.ResourceChat, models.ActionWrite)
	if err != nil {
		return nil
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

	// Get media type (image, document, video, audio)
	mediaType := "image"
	if typeValues := form.Value["type"]; len(typeValues) > 0 {
		mediaType = typeValues[0]
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

	// Read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		a.Log.Error("Failed to read file data", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file data", nil, "")
	}

	// Get MIME type
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Get contact (users without full read permission can only message their assigned contacts)
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Get WhatsApp account - prefer form-specified account over contact default
	mediaAccountName := contact.WhatsAppAccount
	if formWhatsAppAccount != "" {
		mediaAccountName = formWhatsAppAccount
	}
	account, err := a.resolveWhatsAppAccount(orgID, mediaAccountName)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	// Save file locally first
	localPath, err := a.saveMediaLocally(fileData, mimeType, fileHeader.Filename)
	if err != nil {
		a.Log.Error("Failed to save media locally", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save media", nil, "")
	}

	// Build and send via unified message sender
	msgReq := OutgoingMessageRequest{
		Account:       account,
		Contact:       &contact,
		Type:          models.MessageType(mediaType),
		MediaData:     fileData,
		MediaURL:      localPath,
		MediaMimeType: mimeType,
		MediaFilename: fileHeader.Filename,
		Caption:       caption,
	}

	opts := DefaultSendOptions()
	opts.SentByUserID = &userID

	ctx := context.Background()
	message, err := a.SendOutgoingMessage(ctx, msgReq, opts)
	if err != nil {
		a.Log.Error("Failed to send message", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to send message", nil, "")
	}

	response := MessageResponse{
		ID:              message.ID,
		ContactID:       message.ContactID,
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

	return r.SendEnvelope(response)
}

// saveMediaLocally saves media data to local storage and returns the relative path
func (a *App) saveMediaLocally(data []byte, mimeType, filename string) (string, error) {
	// Get extension from MIME type or filename
	ext := getExtensionFromMimeType(mimeType)
	if ext == "" {
		// Try to get from filename
		if dotIdx := strings.LastIndex(filename, "."); dotIdx >= 0 {
			ext = filename[dotIdx:]
		} else {
			ext = ".bin"
		}
	}

	relativePath, err := a.writeMediaFile(data, mimeType, ext)
	if err != nil {
		return "", err
	}
	a.Log.Info("Media saved locally", "path", relativePath, "size", len(data))

	return relativePath, nil
}

// SendReactionRequest represents a request to send a reaction
type SendReactionRequest struct {
	Emoji string `json:"emoji"` // Empty string to remove reaction
}

// SendReaction sends a reaction to a message
func (a *App) SendReaction(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChat, models.ActionWrite)
	if err != nil {
		return nil
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
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Get message
	var message models.Message
	if err := a.DB.Where("id = ? AND contact_id = ?", messageID, contactID).First(&message).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Message not found", nil, "")
	}

	// Get WhatsApp account from the message being reacted to (not from contact, which may be stale)
	reactionAccountName := message.WhatsAppAccount
	if reactionAccountName == "" {
		reactionAccountName = contact.WhatsAppAccount
	}
	account, err := a.resolveWhatsAppAccount(orgID, reactionAccountName)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	// Parse existing reactions from Metadata
	var metadata map[string]any
	if message.Metadata != nil {
		metadata = message.Metadata
	} else {
		metadata = make(map[string]any)
	}

	// Get or initialize reactions array
	type Reaction struct {
		Emoji     string `json:"emoji"`
		FromPhone string `json:"from_phone,omitempty"`
		FromUser  string `json:"from_user,omitempty"`
	}
	var reactions []Reaction
	if reactionsRaw, ok := metadata["reactions"]; ok {
		if reactionsArray, ok := reactionsRaw.([]any); ok {
			for _, r := range reactionsArray {
				if rMap, ok := r.(map[string]any); ok {
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
	a.broadcastReactionUpdate(orgID, message.ID, contact.ID, newReactions)

	return r.SendEnvelope(map[string]any{
		"message_id": message.ID.String(),
		"reactions":  newReactions,
	})
}

// sendWhatsAppReaction sends a reaction to WhatsApp via the GOWA client.
func (a *App) sendWhatsAppReaction(account *models.WhatsAppAccount, contact *models.Contact, message *models.Message, emoji string) {
	if message.WhatsAppMessageID == "" {
		a.Log.Warn("Cannot send reaction - message has no WhatsApp ID", "message_id", message.ID)
		return
	}

	provider := a.resolveProvider(account)
	gowaClient, ok := provider.(*gowa.Client)
	if !ok {
		a.Log.Error("GOWA provider not available for reaction", "account", account.Name)
		return
	}
	chatJID := gowaChatJID(contact)
	if err := gowaClient.SendReaction(context.Background(), account.ToWAAccount(), message.WhatsAppMessageID, chatJID, emoji); err != nil {
		a.Log.Error("GOWA reaction error", "error", err, "account", account.Name)
		return
	}

	a.Log.Info("Reaction sent successfully", "message_id", message.WhatsAppMessageID, "emoji", emoji)
}

// TypingRequest is the body for the typing-indicator endpoint.
// gowaChatJID builds the WhatsApp JID for a GOWA API call from a contact.
// Group contacts (metadata is_group_chat, or phone_number starting with the
// WhatsApp group-ID prefix 120362/120363) need the "@g.us" suffix; newsletter
// contacts (metadata is_newsletter) need the "@newsletter" suffix; 1:1 chats
// use "@s.whatsapp.net". If the phone already contains "@" it is returned
// unchanged. Centralizing this fixes group/newsletter send/revoke/typing/
// reaction/read — previously each call site hardcoded "@s.whatsapp.net" and
// GOWA rejected group JIDs with "is not on whatsapp".
func gowaChatJID(contact *models.Contact) string {
	if contact == nil {
		return ""
	}
	phone := contact.PhoneNumber
	if phone == "" || strings.Contains(phone, "@") {
		return phone
	}
	isNewsletter := contact.Metadata != nil && contact.Metadata["is_newsletter"] == true
	if isNewsletter {
		return phone + "@newsletter"
	}
	isGroup := contact.Metadata != nil && contact.Metadata["is_group_chat"] == true
	if !isGroup && (strings.HasPrefix(phone, "120362") || strings.HasPrefix(phone, "120363")) {
		isGroup = true
	}
	if isGroup {
		return phone + "@g.us"
	}
	return phone + "@s.whatsapp.net"
}

// action is "start" or "stop".
type TypingRequest struct {
	Action string `json:"action"`
}

// SendTypingIndicator forwards a typing ("composing") presence to the chat's
// recipient via the GOWA send/chat-presence endpoint. This is GOWA-only: Meta
// Cloud API has no equivalent, so non-GOWA accounts get a clean 400.
// The indicator is outbound-only (it shows on the recipient's WhatsApp), so
// no WebSocket event is broadcast back to the GOWA UI.
func (a *App) SendTypingIndicator(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChat, models.ActionWrite)
	if err != nil {
		return nil
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var req TypingRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	// Normalize the action and reject anything that is not start/stop.
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "start" && action != "stop" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, `action must be "start" or "stop"`, nil, "")
	}

	// Resolve contact (honoring per-user assignment scoping).
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	account, err := a.resolveWhatsAppAccount(orgID, contact.WhatsAppAccount)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	provider := a.resolveProvider(account)
	gowaClient, ok := provider.(*gowa.Client)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not available", nil, "")
	}

	// Derive the chat JID (handles group @g.us vs 1:1 @s.whatsapp.net).
	chatJID := gowaChatJID(&contact)

	if err := gowaClient.SendChatPresence(context.Background(), account.ToWAAccount(), chatJID, action); err != nil {
		a.Log.Error("GOWA typing indicator error", "error", err, "account", account.Name, "action", action)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to send typing indicator", nil, "")
	}

	return r.SendEnvelope(map[string]any{"status": "ok", "action": action})
}

// RevokeMessageRequest is the (empty) body for the revoke endpoint. It exists
// so future fields can be added without changing the handler signature; the
// chat JID is derived server-side from the contact, never trusted from input.
type RevokeMessageRequest struct{}

// RevokeMessage unsends a message for everyone in the chat (GOWA-only) and
// marks the local message row as revoked so the UI shows a "[message revoked]"
// placeholder. It broadcasts a status_update over WebSocket so every open
// client reflects the revoked state in real time. The status and content set
// here mirror the inbound message.revoked webhook handler so the two paths
// stay consistent.
func (a *App) RevokeMessage(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceChatRevoke, models.ActionWrite)
	if err != nil {
		return nil
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

	// Resolve the contact (assignment scoping applied).
	var contact models.Contact
	query := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID)
	query = a.scopeAssignedContact(query, userID, orgID)
	if err := query.First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Load the message scoped to the contact (mirrors SendReaction).
	var message models.Message
	if err := a.DB.Where("id = ? AND contact_id = ?", messageID, contactID).First(&message).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Message not found", nil, "")
	}

	// Only outgoing messages can be revoked by the connected account.
	if message.Direction != models.DirectionOutgoing {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Only outgoing messages can be revoked", nil, "")
	}
	if message.WhatsAppMessageID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Message has no WhatsApp ID; cannot revoke", nil, "")
	}

	// Resolve the account from the message (authoritative) falling back to the
	// contact's account, exactly like SendReaction.
	revokeAccountName := message.WhatsAppAccount
	if revokeAccountName == "" {
		revokeAccountName = contact.WhatsAppAccount
	}
	account, err := a.resolveWhatsAppAccount(orgID, revokeAccountName)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	provider := a.resolveProvider(account)
	gowaClient, ok := provider.(*gowa.Client)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not available", nil, "")
	}

	chatJID := gowaChatJID(&contact)

	if err := gowaClient.RevokeMessage(context.Background(), account.ToWAAccount(), message.WhatsAppMessageID, chatJID); err != nil {
		a.Log.Error("GOWA revoke error", "error", err, "message_id", message.ID, "account", account.Name)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to revoke message", nil, "")
	}

	// Persist the revoked status locally. We flip ONLY status — the original
	// content/media stays in the DB so the UI can render it under a "deleted"
	// overlay (matching WhatsApp's "This message was deleted" behaviour where
	// the sender still sees what they sent, dimmed). The frontend keys the
	// revoked render entirely off status === "revoked", so content is never
	// interpreted as live text after this point. Outbound and inbound paths
	// stay consistent.
	if err := a.DB.Model(&models.Message{}).Where("id = ?", message.ID).Updates(map[string]any{
		"status": models.MessageStatusRevoked,
	}).Error; err != nil {
		a.Log.Error("Failed to mark message as revoked after outbound revoke", "error", err, "message_id", message.ID)
	}

	// Broadcast a status_update so every open client swaps the bubble for the
	// revoked placeholder in real time.
	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type: websocket.TypeStatusUpdate,
			Payload: map[string]any{
				"message_id": message.ID,
				"contact_id": message.ContactID,
				"status":     models.MessageStatusRevoked,
			},
		})
	}

	return r.SendEnvelope(map[string]any{
		"message_id": message.ID.String(),
		"status":     models.MessageStatusRevoked,
	})
}

