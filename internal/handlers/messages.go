package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/internal/sanitizer"
	"github.com/compnew2006/whatomate/internal/templateutil"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// ============================================================================
// Unified Message Sending
// ============================================================================

// OutgoingMessageRequest contains all parameters for sending any type of message
type OutgoingMessageRequest struct {
	// Required
	Account *models.WhatsAppAccount
	Contact *models.Contact
	// Optional when provider is whatsmeow
	InstanceID *uuid.UUID

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

	// Template messages
	Template   *models.Template
	BodyParams map[string]string // Parameter name -> value (supports both named and positional)

	// WhatsApp Flow messages
	FlowID          string // Meta Flow ID
	FlowHeader      string // Optional header text for flow
	FlowCTA         string // CTA button text (max 20 chars)
	FlowToken       string // Unique token for flow response tracking
	FlowFirstScreen string // First screen name to navigate to

	// Reply context
	ReplyToMessage *models.Message

	// IdempotencyKey prevents duplicate sends on frontend retry
	IdempotencyKey string
}

type MessageActorType string

const (
	MessageActorUser         MessageActorType = "user"
	MessageActorSystem       MessageActorType = "system"
	MessageActorWorker       MessageActorType = "worker"
	MessageActorAutoCampaign MessageActorType = "auto_campaign"
)

// MessageSendOptions configures optional behaviors for message sending
type MessageSendOptions struct {
	// BroadcastWebSocket enables WebSocket broadcast to org (default: true)
	BroadcastWebSocket bool

	// DispatchWebhook enables webhook dispatch for message.sent event (default: true)
	DispatchWebhook bool

	// TrackSLA enables SLA tracking for chatbot messages (default: false)
	TrackSLA bool

	// SentByUserID sets the user who sent the message (for agent messages)
	SentByUserID *uuid.UUID

	// ActorType identifies sender context explicitly (user|system|worker|auto_campaign).
	ActorType MessageActorType

	// Async if true, sends in background goroutine and returns immediately
	// Message is persisted before send, status updated after
	Async bool
}

func (o MessageSendOptions) resolvedActorType() MessageActorType {
	switch o.ActorType {
	case MessageActorUser, MessageActorSystem, MessageActorWorker, MessageActorAutoCampaign:
		return o.ActorType
	}
	if o.SentByUserID != nil {
		return MessageActorUser
	}
	return MessageActorSystem
}

// DefaultSendOptions returns options suitable for agent UI sends
func DefaultSendOptions() MessageSendOptions {
	return MessageSendOptions{
		BroadcastWebSocket: true,
		DispatchWebhook:    true,
		TrackSLA:           false,
		ActorType:          MessageActorUser,
		Async:              true,
	}
}

// ChatbotSendOptions returns options suitable for chatbot sends
func ChatbotSendOptions() MessageSendOptions {
	return MessageSendOptions{
		BroadcastWebSocket: true,
		DispatchWebhook:    false,
		TrackSLA:           true,
		ActorType:          MessageActorSystem,
		Async:              false,
	}
}

// APISendOptions returns options suitable for API/template sends
func APISendOptions() MessageSendOptions {
	return MessageSendOptions{
		BroadcastWebSocket: false,
		DispatchWebhook:    true,
		TrackSLA:           false,
		ActorType:          MessageActorSystem,
		Async:              true,
	}
}

// SLASendOptions returns options suitable for SLA system notifications
func SLASendOptions() MessageSendOptions {
	return MessageSendOptions{
		BroadcastWebSocket: true,
		DispatchWebhook:    false,
		TrackSLA:           false,
		ActorType:          MessageActorSystem,
		Async:              false, // Sync to ensure message is sent before continuing
	}
}

const (
	idempotencyKeyPrefix = "whatomate:msg:idem:"
	idempotencyKeyTTL    = 5 * time.Minute
)

// SendOutgoingMessage is the unified method for sending all types of WhatsApp messages.
// It handles: text, media (image/video/audio/document), interactive (buttons/list/cta_url), and template messages.
// Routes through MessageProvider when configured for whatsmeow, otherwise uses the Meta client directly.
func (a *App) SendOutgoingMessage(ctx context.Context, req OutgoingMessageRequest, opts MessageSendOptions) (*models.Message, error) {
	if err := a.enforceStrictSendRestrictions(ctx, req, opts); err != nil {
		return nil, err
	}

	if existing, ok := a.checkIdempotencyKey(ctx, req.IdempotencyKey); ok {
		return existing, nil
	}

	a.applyAgentNamePrefixToTextMessage(&req, opts)

	// 1. Create message record
	msg := a.createOutgoingMessage(req, opts)

	// Save to database
	if err := a.DB.Create(msg).Error; err != nil {
		a.Log.Error("Failed to create message", "error", err)
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	a.recordIdempotencyKey(ctx, req.IdempotencyKey, msg.ID)

	// 2. Define the send function based on provider
	var sendFn func(sendCtx context.Context) (string, error)

	if a.isWhatsmeowProvider() && a.MessageProvider != nil {
		providerInstanceID, resolveErr := a.resolveProviderInstanceID(ctx, req, msg)
		if resolveErr != nil {
			a.finalizeMessageSend(msg, req, opts, "", resolveErr)
			return msg, resolveErr
		}
		// Route through MessageProvider (whatsmeow adapter)
		sendFn = func(sendCtx context.Context) (string, error) {
			return a.sendViaProvider(sendCtx, req, msg, providerInstanceID)
		}
	} else {
		// Route through Meta client (existing behavior)
		sendFn = func(sendCtx context.Context) (string, error) {
			waAccount := a.toWhatsAppAccount(req.Account)

			// Get reply-to message ID if this is a reply
			var replyToMsgID string
			if req.ReplyToMessage != nil && req.ReplyToMessage.WhatsAppMessageID != "" {
				replyToMsgID = req.ReplyToMessage.WhatsAppMessageID
			}

			switch req.Type {
			case models.MessageTypeText:
				return a.WhatsApp.SendTextMessage(sendCtx, waAccount, req.Contact.PhoneNumber, req.Content, replyToMsgID)

			case models.MessageTypeImage, models.MessageTypeVideo, models.MessageTypeAudio, models.MessageTypeDocument:
				// Upload media if MediaData is provided and MediaID is not set
				mediaID := req.MediaID
				if mediaID == "" && len(req.MediaData) > 0 {
					var err error
					mediaID, err = a.WhatsApp.UploadMedia(sendCtx, waAccount, req.MediaData, req.MediaMimeType, req.MediaFilename)
					if err != nil {
						return "", fmt.Errorf("failed to upload media: %w", err)
					}
				}
				// Send the appropriate media type
				switch req.Type {
				case models.MessageTypeImage:
					return a.WhatsApp.SendImageMessage(sendCtx, waAccount, req.Contact.PhoneNumber, mediaID, req.Caption)
				case models.MessageTypeVideo:
					return a.WhatsApp.SendVideoMessage(sendCtx, waAccount, req.Contact.PhoneNumber, mediaID, req.Caption)
				case models.MessageTypeAudio:
					return a.WhatsApp.SendAudioMessage(sendCtx, waAccount, req.Contact.PhoneNumber, mediaID)
				default: // document
					return a.WhatsApp.SendDocumentMessage(sendCtx, waAccount, req.Contact.PhoneNumber, mediaID, req.MediaFilename, req.Caption)
				}

			case models.MessageTypeInteractive:
				switch req.InteractiveType {
				case "cta_url":
					return a.WhatsApp.SendCTAURLButton(sendCtx, waAccount, req.Contact.PhoneNumber, req.BodyText, req.ButtonText, req.URL)
				default: // "button" or "list"
					return a.WhatsApp.SendInteractiveButtons(sendCtx, waAccount, req.Contact.PhoneNumber, req.BodyText, req.Buttons)
				}

			case models.MessageTypeTemplate:
				if req.Template == nil {
					return "", fmt.Errorf("template is required for template messages")
				}
				components := whatsapp.BodyParamsToComponents(req.BodyParams)
				return a.WhatsApp.SendTemplateMessage(sendCtx, waAccount, req.Contact.PhoneNumber, req.Template.Name, req.Template.Language, components)

			case models.MessageTypeFlow:
				if req.FlowID == "" {
					return "", fmt.Errorf("flow ID is required for flow messages")
				}
				return a.WhatsApp.SendFlowMessage(sendCtx, waAccount, req.Contact.PhoneNumber, req.FlowID, req.FlowHeader, req.BodyText, req.FlowCTA, req.FlowToken, req.FlowFirstScreen)

			default:
				return "", fmt.Errorf("unsupported message type: %s", req.Type)
			}
		}
	}

	// 3. Execute send — for whatsmeow, enqueue through rate-limited queue
	if a.isWhatsmeowProvider() && a.WhatsmeowQueue != nil && msg.InstanceID != nil {
		// Wrap sendFn as a queue Job
		msgID := msg.ID
		instanceIDStr := msg.InstanceID.String()
		err := a.WhatsmeowQueue.Enqueue(instanceIDStr, func(qCtx context.Context) error {
			wamid, sendErr := sendFn(qCtx)
			a.finalizeMessageSend(msg, req, opts, wamid, sendErr)
			if sendErr != nil {
				return sendErr
			}
			return nil
		})
		if err != nil {
			a.Log.Error("Failed to enqueue message", "error", err, "message_id", msgID)
			a.DB.Model(&models.Message{}).Where("id = ?", msgID).Updates(map[string]any{
				"status":        models.MessageStatusFailed,
				"error_message": "Queue full: " + err.Error(),
			})
			msg.Status = models.MessageStatusFailed
			msg.ErrorMessage = "Queue full: " + err.Error()
			if opts.BroadcastWebSocket && req.Contact != nil {
				a.broadcastNewMessage(req.Contact.OrganizationID, msg, req.Contact)
			}
		}
	} else if opts.Async {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			asyncCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			defer cancel()

			wamid, sendErr := sendFn(asyncCtx)
			a.finalizeMessageSend(msg, req, opts, wamid, sendErr)
		}()
	} else {
		wamid, err := sendFn(ctx)
		a.finalizeMessageSend(msg, req, opts, wamid, err)
	}

	// 4. Immediate actions (before send completes for async)
	if opts.TrackSLA {
		a.UpdateContactChatbotMessage(req.Contact.ID)
	}

	// Update contact's last message
	preview := a.getMessagePreview(req)
	a.updateContactLastMessage(req.Contact, preview)

	return msg, nil
}

// isWhatsmeowProvider returns true if the configured provider is whatsmeow
func (a *App) isWhatsmeowProvider() bool {
	return a.Config != nil && a.Config.WhatsApp.Provider == "whatsmeow"
}

func (a *App) applyAgentNamePrefixToTextMessage(req *OutgoingMessageRequest, opts MessageSendOptions) {
	if a == nil || req == nil || req.Type != models.MessageTypeText || opts.SentByUserID == nil {
		return
	}
	if !a.canPrefixTextWithAgentName(req, *opts.SentByUserID) {
		return
	}

	agentName := a.resolveAgentMessagePrefixName(*opts.SentByUserID)
	req.Content = formatAgentMessageContent(agentName, req.Content)
}

func (a *App) canPrefixTextWithAgentName(req *OutgoingMessageRequest, userID uuid.UUID) bool {
	if a == nil || userID == uuid.Nil {
		return false
	}

	orgID := uuid.Nil
	if req != nil {
		if req.Contact != nil {
			orgID = req.Contact.OrganizationID
		}
		if orgID == uuid.Nil && req.Account != nil {
			orgID = req.Account.OrganizationID
		}
	}

	return a.shouldPrefixAgentNameForUser(orgID, userID)
}

func (a *App) resolveAgentMessagePrefixName(userID uuid.UUID) string {
	if a == nil || userID == uuid.Nil {
		return ""
	}

	name := strings.TrimSpace(a.ResolveUserDisplayName(userID))
	if name == "" {
		return ""
	}

	if at := strings.Index(name, "@"); at > 0 {
		return strings.TrimSpace(name[:at])
	}

	return name
}

func formatAgentMessageContent(agentName, content string) string {
	trimmedContent := strings.TrimSpace(content)
	trimmedName := strings.TrimSpace(agentName)
	if trimmedContent == "" || trimmedName == "" {
		return trimmedContent
	}

	if contentHasAgentPrefix(trimmedContent, trimmedName) {
		return trimmedContent
	}

	return fmt.Sprintf("%s : %s", trimmedName, trimmedContent)
}

func contentHasAgentPrefix(content, agentName string) bool {
	if content == "" || agentName == "" {
		return false
	}

	if len(content) <= len(agentName) {
		return false
	}

	if !strings.EqualFold(content[:len(agentName)], agentName) {
		return false
	}

	remaining := strings.TrimLeft(content[len(agentName):], " \t")
	return strings.HasPrefix(remaining, ":")
}

func (a *App) resolveProviderInstanceID(ctx context.Context, req OutgoingMessageRequest, msg *models.Message) (string, error) {
	if msg == nil {
		return "", fmt.Errorf("message is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if msg.InstanceID != nil {
		return msg.InstanceID.String(), nil
	}
	if req.Contact != nil && req.Contact.InstanceID != nil {
		instanceID := *req.Contact.InstanceID
		msg.InstanceID = &instanceID
		if err := a.DB.WithContext(ctx).Model(&models.Message{}).
			Where("id = ?", msg.ID).
			Update("instance_id", msg.InstanceID).Error; err != nil {
			a.Log.Warn("failed to persist contact instance ID for provider send", "message_id", msg.ID, "error", err)
		}
		return msg.InstanceID.String(), nil
	}

	orgID := uuid.Nil
	if req.Account != nil {
		orgID = req.Account.OrganizationID
	} else if req.Contact != nil {
		orgID = req.Contact.OrganizationID
	}
	if orgID == uuid.Nil {
		return "", fmt.Errorf("cannot resolve organization for provider send")
	}

	ctxDB := a.DB.WithContext(ctx)
	var instance models.WhatsAppInstance
	if err := ctxDB.Where("organization_id = ? AND is_default = ? AND status = ?",
		orgID, true, models.InstanceStatusConnected).
		First(&instance).Error; err != nil {
		if err := ctxDB.Where("organization_id = ? AND status = ?",
			orgID, models.InstanceStatusConnected).
			First(&instance).Error; err != nil {
			return "", fmt.Errorf("no connected WhatsApp instance found")
		}
	}

	instanceID := instance.ID
	msg.InstanceID = &instanceID
	if err := ctxDB.Model(&models.Message{}).Where("id = ?", msg.ID).
		Update("instance_id", msg.InstanceID).Error; err != nil {
		a.Log.Warn("failed to persist resolved instance ID for provider send", "message_id", msg.ID, "error", err)
	}

	return instanceID.String(), nil
}

// sendViaProvider routes the message through the MessageProvider interface.
// This is used when the provider is whatsmeow (or any future MessageProvider).
func (a *App) sendViaProvider(ctx context.Context, req OutgoingMessageRequest, msg *models.Message, instanceID string) (string, error) {
	if instanceID == "" {
		return "", fmt.Errorf("missing instance ID for provider send")
	}

	to := req.Contact.PhoneNumber
	if canonicalTo := a.resolveDirectRecipientFromConversation(ctx, req.Contact); canonicalTo != "" {
		to = canonicalTo
	}

	switch req.Type {
	case models.MessageTypeText:
		// Check if this is a reply and the provider supports it
		if req.ReplyToMessage != nil && req.ReplyToMessage.WhatsAppMessageID != "" {
			if rp, ok := a.MessageProvider.(provider.ReplyProvider); ok {
				return rp.SendTextReply(ctx, instanceID, to, req.Content, req.ReplyToMessage.WhatsAppMessageID)
			}
		}
		return a.MessageProvider.SendText(ctx, instanceID, to, req.Content)

	case models.MessageTypeImage:
		mediaRef := resolveProviderMediaRef(req)
		return a.MessageProvider.SendImage(ctx, instanceID, to, mediaRef, req.Caption)

	case models.MessageTypeVideo:
		mediaRef := resolveProviderMediaRef(req)
		return a.MessageProvider.SendVideo(ctx, instanceID, to, mediaRef, req.Caption)

	case models.MessageTypeAudio:
		mediaRef := resolveProviderMediaRef(req)
		return a.MessageProvider.SendAudio(ctx, instanceID, to, mediaRef)

	case models.MessageTypeDocument:
		mediaRef := resolveProviderMediaRef(req)
		return a.MessageProvider.SendDocument(ctx, instanceID, to, mediaRef, req.MediaFilename, req.Caption)

	case models.MessageTypeInteractive:
		// Interactive messages are Meta-specific; for whatsmeow, send as text fallback
		fallbackText := req.BodyText
		if fallbackText == "" {
			fallbackText = req.Content
		}
		return a.MessageProvider.SendText(ctx, instanceID, to, fallbackText)

	case models.MessageTypeTemplate:
		// Templates are Meta-specific; for whatsmeow, render template text and send as plain text
		if req.Template == nil {
			return "", fmt.Errorf("template is required for template messages")
		}
		renderedContent := templateutil.ReplaceWithStringParams(req.Template.BodyContent, req.BodyParams)
		if renderedContent == "" {
			renderedContent = fmt.Sprintf("[Template: %s]", req.Template.DisplayName)
		}
		return a.MessageProvider.SendText(ctx, instanceID, to, renderedContent)

	default:
		return "", fmt.Errorf("unsupported message type for whatsmeow: %s", req.Type)
	}
}

// resolveProviderMediaRef standardizes how provider sends resolve media input.
// MediaURL (e.g. signed/local URL) takes precedence; fallback is existing MediaID.
func resolveProviderMediaRef(req OutgoingMessageRequest) string {
	if req.MediaURL != "" {
		return req.MediaURL
	}
	return req.MediaID
}

func (a *App) resolveDirectRecipientFromConversation(ctx context.Context, contact *models.Contact) string {
	if a == nil || contact == nil {
		return ""
	}
	if isGroupConversationID(contact.PhoneNumber) || isChannelConversationID(contact.PhoneNumber) {
		return ""
	}

	var latestMessage models.Message
	if err := a.DB.WithContext(ctx).
		Select("conversation_id").
		Where("organization_id = ? AND contact_id = ? AND conversation_id <> ''", contact.OrganizationID, contact.ID).
		Order("created_at DESC").
		First(&latestMessage).Error; err != nil {
		return ""
	}

	canonicalPhone := strings.TrimSpace(directUserFromConversationID(latestMessage.ConversationID))
	if canonicalPhone == "" {
		return ""
	}

	a.repairDirectContactPhoneFromConversation(contact, latestMessage.ConversationID)
	return canonicalPhone
}

// ============================================================================
// Internal Helpers
// ============================================================================

// toWhatsAppAccount converts models.WhatsAppAccount to whatsapp.Account
func (a *App) toWhatsAppAccount(account *models.WhatsAppAccount) *whatsapp.Account {
	return &whatsapp.Account{
		PhoneID:     account.PhoneID,
		BusinessID:  account.BusinessID,
		AppID:       account.AppID,
		APIVersion:  account.APIVersion,
		AccessToken: account.AccessToken,
	}
}

// createOutgoingMessage creates a Message model from the request
func (a *App) createOutgoingMessage(req OutgoingMessageRequest, opts MessageSendOptions) *models.Message {
	instanceID := req.InstanceID
	if instanceID == nil && req.Contact != nil {
		instanceID = req.Contact.InstanceID
	}

	orgID := uuid.Nil
	whatsAppAccount := ""
	if req.Account != nil {
		orgID = req.Account.OrganizationID
		whatsAppAccount = req.Account.Name
	} else if req.Contact != nil {
		orgID = req.Contact.OrganizationID
	}

	msg := &models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		InstanceID:      instanceID,
		WhatsAppAccount: whatsAppAccount,
		ContactID:       req.Contact.ID,
		Direction:       models.DirectionOutgoing,
		MessageType:     req.Type,
		Status:          models.MessageStatusPending,
		SentByUserID:    opts.SentByUserID,
	}

	// Set content based on message type
	switch req.Type {
	case models.MessageTypeText:
		msg.Content = sanitizer.SanitizeMessageContent(req.Content)

	case models.MessageTypeImage, models.MessageTypeVideo, models.MessageTypeAudio, models.MessageTypeDocument:
		msg.Content = sanitizer.SanitizeMessageContent(req.Caption)
		msg.MediaURL = req.MediaURL
		msg.MediaMimeType = req.MediaMimeType
		msg.MediaFilename = req.MediaFilename

	case models.MessageTypeInteractive:
		msg.Content = sanitizer.SanitizeMessageContent(req.BodyText)
		msg.InteractiveData = a.buildInteractiveData(req)

	case models.MessageTypeTemplate:
		if req.Template != nil {
			content := templateutil.ReplaceWithStringParams(req.Template.BodyContent, req.BodyParams)
			if content == "" {
				content = fmt.Sprintf("[Template: %s]", req.Template.DisplayName)
			}
			msg.Content = sanitizer.SanitizeMessageContent(content)
			msg.TemplateName = req.Template.Name
			msg.Metadata = models.JSONB{
				"template_name": req.Template.Name,
				"template_id":   req.Template.ID.String(),
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

	if req.Contact != nil && isGroupConversationID(req.Contact.PhoneNumber) {
		msg.ConversationID = req.Contact.PhoneNumber
		if msg.Metadata == nil {
			msg.Metadata = models.JSONB{}
		}
		msg.Metadata["is_group_chat"] = true
		msg.Metadata["group_jid"] = req.Contact.PhoneNumber
	}

	return msg
}

// buildInteractiveData creates the InteractiveData JSONB for interactive and template messages
func (a *App) buildInteractiveData(req OutgoingMessageRequest) models.JSONB {
	// Template buttons: stored as JSONBArray on Template.Buttons
	if req.Template != nil && len(req.Template.Buttons) > 0 {
		return models.JSONB{
			"type":    "button",
			"buttons": req.Template.Buttons,
		}
	}

	switch req.InteractiveType {
	case "cta_url":
		return models.JSONB{
			"type":        "cta_url",
			"body":        sanitizer.SanitizeMessageContent(req.BodyText),
			"button_text": sanitizer.SanitizeMessageContent(req.ButtonText),
			"url":         req.URL,
		}
	case "list":
		rows := make([]interface{}, len(req.Buttons))
		for i, btn := range req.Buttons {
			rows[i] = map[string]string{
				"id":    btn.ID,
				"title": sanitizer.SanitizeMessageContent(btn.Title),
			}
		}
		return models.JSONB{
			"type": "list",
			"body": sanitizer.SanitizeMessageContent(req.BodyText),
			"rows": rows,
		}
	default: // "button"
		buttons := make([]interface{}, len(req.Buttons))
		for i, btn := range req.Buttons {
			buttons[i] = map[string]string{
				"id":    btn.ID,
				"title": sanitizer.SanitizeMessageContent(btn.Title),
			}
		}
		return models.JSONB{
			"type":    "button",
			"body":    sanitizer.SanitizeMessageContent(req.BodyText),
			"buttons": buttons,
		}
	}
}

// finalizeMessageSend updates message status and triggers post-send actions
func (a *App) finalizeMessageSend(msg *models.Message, req OutgoingMessageRequest, opts MessageSendOptions, wamid string, err error) {
	if err != nil {
		errMsg := err.Error()

		a.DB.Model(&models.Message{}).Where("id = ?", msg.ID).Updates(map[string]any{
			"status":        models.MessageStatusFailed,
			"error_message": errMsg,
		})
		msg.Status = models.MessageStatusFailed
		msg.ErrorMessage = errMsg
		if msg.InstanceID != nil && a.WhatsmeowManager != nil {
			a.WhatsmeowManager.MarkMessageFailed(*msg.InstanceID)
		}
		a.Log.Error("Failed to send message", "error", err, "message_id", msg.ID, "type", msg.MessageType)

		// Push to retry queue for exponential backoff retry
		a.pushToOutgoingRetryQueue(msg)

		// Broadcast failure as new message via WebSocket (single authoritative event)
		if opts.BroadcastWebSocket && req.Contact != nil {
			a.broadcastNewMessage(req.Contact.OrganizationID, msg, req.Contact)
		}
		return
	}

	a.DB.Model(&models.Message{}).Where("id = ?", msg.ID).Updates(map[string]any{
		"status":               models.MessageStatusSent,
		"whats_app_message_id": wamid,
		"error_message":        "",
	})
	msg.Status = models.MessageStatusSent
	msg.WhatsAppMessageID = wamid
	if msg.InstanceID != nil && a.WhatsmeowManager != nil {
		a.WhatsmeowManager.MarkMessageSent(*msg.InstanceID)
	}
	a.Log.Info("Message sent", "message_id", msg.ID, "wa_message_id", wamid, "type", msg.MessageType)

	// Dispatch webhook for successful send
	if opts.DispatchWebhook && req.Account != nil {
		a.dispatchMessageSentWebhook(req.Account, req.Contact, msg)
	}

	// Broadcast as new message via WebSocket (single authoritative event after send completes)
	if opts.BroadcastWebSocket && req.Contact != nil {
		a.broadcastNewMessage(req.Contact.OrganizationID, msg, req.Contact)
	}
}

// pushToOutgoingRetryQueue enqueues a failed outgoing message for retry.
func (a *App) pushToOutgoingRetryQueue(msg *models.Message) {
	if a.OutgoingRetryQueue == nil {
		return
	}

	entry := &queue.OutgoingRetryEntry{
		MessageID: msg.ID.String(),
		OrgID:     msg.OrganizationID.String(),
	}

	if err := a.OutgoingRetryQueue.Push(context.Background(), entry); err != nil {
		a.Log.Error("Failed to push to outgoing retry queue",
			"error", err,
			"message_id", msg.ID,
		)
	}
}

// broadcastNewMessage broadcasts a new message via WebSocket
func (a *App) broadcastNewMessage(orgID uuid.UUID, msg *models.Message, contact *models.Contact) {
	if a.WSHub == nil {
		return
	}

	contentBody := msg.Content
	if a.ShouldMaskPhoneNumbers(orgID) {
		contentBody = MaskPhoneNumbersInText(contentBody)
	}

	payload := map[string]any{
		"id":               msg.ID,
		"contact_id":       contact.ID.String(),
		"direction":        msg.Direction,
		"message_type":     msg.MessageType,
		"content":          map[string]string{"body": contentBody},
		"status":           msg.Status,
		"whatsapp_account": msg.WhatsAppAccount,
		"created_at":       msg.CreatedAt,
		"updated_at":       msg.UpdatedAt,
	}

	if msg.ConversationID != "" {
		payload["conversation_id"] = msg.ConversationID
	}
	if isGroupMessage(*msg) || isGroupContact(contact) {
		payload["is_group_chat"] = true
	}
	if msg.Metadata != nil {
		payload["metadata"] = msg.Metadata
	}
	if senderPhone := extractMessageSenderPhone(msg.Metadata); senderPhone != "" {
		payload["sender_phone"] = senderPhone
	}
	if senderPushName := extractMessageSenderPushName(msg.Metadata); senderPushName != "" {
		payload["sender_push_name"] = senderPushName
	}

	// Add assignment and lifecycle info
	assignedUserID := ""
	if contact.AssignedUserID != nil {
		assignedUserID = contact.AssignedUserID.String()
	}
	payload["assigned_user_id"] = assignedUserID
	payload["contact_status"] = contact.EffectiveStatus().String()

	profileName := contact.ProfileName
	if a.ShouldMaskPhoneNumbers(orgID) {
		profileName = MaskIfPhoneNumber(profileName)
	}
	payload["profile_name"] = profileName

	// Add instance ID
	if msg.InstanceID != nil {
		payload["instance_id"] = msg.InstanceID.String()
	}

	// Add media fields
	if messageHasVisibleMedia(msg) {
		payload["media_url"] = msg.MediaURL
		payload["media_mime_type"] = msg.MediaMimeType
		payload["media_filename"] = msg.MediaFilename
	}

	// Add interactive data
	if msg.InteractiveData != nil {
		payload["interactive_data"] = msg.InteractiveData
	}

	// Add reply context
	if msg.IsReply && msg.ReplyToMessageID != nil {
		payload["is_reply"] = true
		payload["reply_to_message_id"] = msg.ReplyToMessageID.String()

		// Include reply preview for UI
		var replyToMsg models.Message
		if err := a.DB.First(&replyToMsg, msg.ReplyToMessageID).Error; err == nil {
			replyContent := replyToMsg.Content
			if a.ShouldMaskPhoneNumbers(orgID) {
				replyContent = MaskPhoneNumbersInText(replyContent)
			}
			replyPayload := map[string]any{
				"id":           replyToMsg.ID.String(),
				"content":      replyContent,
				"message_type": replyToMsg.MessageType,
				"direction":    replyToMsg.Direction,
			}
			if senderPhone := extractMessageSenderPhone(replyToMsg.Metadata); senderPhone != "" {
				replyPayload["sender_phone"] = senderPhone
			}
			if messageHasVisibleMedia(&replyToMsg) {
				replyPayload["media_url"] = replyToMsg.MediaURL
				replyPayload["media_mime_type"] = replyToMsg.MediaMimeType
				replyPayload["media_filename"] = replyToMsg.MediaFilename
			}
			payload["reply_to_message"] = replyPayload
		}
	}

	a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type:    websocket.TypeNewMessage,
		Payload: payload,
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

// checkIdempotencyKey returns the existing message if the key was already processed.
func (a *App) checkIdempotencyKey(ctx context.Context, key string) (*models.Message, bool) {
	if key == "" || a.Redis == nil {
		return nil, false
	}

	redisKey := idempotencyKeyPrefix + key
	msgIDStr, err := a.Redis.Get(ctx, redisKey).Result()
	if err != nil {
		return nil, false
	}

	msgID, err := uuid.Parse(msgIDStr)
	if err != nil {
		return nil, false
	}

	var msg models.Message
	if err := a.DB.WithContext(ctx).First(&msg, msgID).Error; err != nil {
		return nil, false
	}

	a.Log.Info("Idempotency hit: returning existing message", "key", key, "message_id", msgID)
	return &msg, true
}

// recordIdempotencyKey stores the message ID against the idempotency key in Redis.
func (a *App) recordIdempotencyKey(ctx context.Context, key string, msgID uuid.UUID) {
	if key == "" || a.Redis == nil {
		return
	}

	redisKey := idempotencyKeyPrefix + key
	if err := a.Redis.Set(ctx, redisKey, msgID.String(), idempotencyKeyTTL).Err(); err != nil {
		a.Log.Warn("Failed to record idempotency key", "key", key, "error", err)
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
	AccountName    string            `json:"account_name"`    // Optional: specific WhatsApp account
}

// SendTemplateMessage sends a template message to a contact or phone number
func (a *App) SendTemplateMessage(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var req SendTemplateMessageRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
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
		t, err := findByIDAndOrg[models.Template](requestDB, r, templateID, orgID, "Template")
		if err != nil {
			return nil
		}
		template = *t
	} else {
		if err := requestDB.Where("name = ? AND organization_id = ?", req.TemplateName, orgID).First(&template).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Template not found", nil, "")
		}
	}

	// Check template is approved
	if template.Status != "APPROVED" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, fmt.Sprintf("Template is not approved (status: %s)", template.Status), nil, "")
	}

	// Get contact or use phone number directly
	var contact *models.Contact

	if req.ContactID != "" {
		cID, err := uuid.Parse(req.ContactID)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid contact_id", nil, "")
		}
		c, err := findByIDAndOrg[models.Contact](requestDB, r, cID, orgID, "Contact")
		if err != nil {
			return nil
		}
		contact = c
	} else {
		// Find or create contact from phone number
		phoneNumber := req.PhoneNumber
		var c models.Contact
		err := requestDB.Where("phone_number = ? AND organization_id = ?", phoneNumber, orgID).First(&c).Error
		if err != nil {
			// Contact not found, create new one
			c = models.Contact{
				BaseModel:      models.BaseModel{ID: uuid.New()},
				OrganizationID: orgID,
				PhoneNumber:    phoneNumber,
			}
			if err := requestDB.Create(&c).Error; err != nil {
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

	// Send using unified message sender
	msgReq := OutgoingMessageRequest{
		Account:    account,
		Contact:    contact,
		Type:       models.MessageTypeTemplate,
		Template:   &template,
		BodyParams: req.TemplateParams,
	}

	opts := DefaultSendOptions()
	opts.SentByUserID = &userID

	ctx := r.RequestCtx
	message, err := a.SendOutgoingMessage(ctx, msgReq, opts)
	if err != nil {
		if restrictedMessage, reasonCode, ok := asRestrictedSendViolationWithReason(err); ok {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, restrictedMessage, reasonCodeDetails(reasonCode), "")
		}
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
