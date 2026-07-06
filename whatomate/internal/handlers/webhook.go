package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/compnew2006/whatomate/internal/campaignstats"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// WebhookVerify handles Meta's webhook verification challenge
func (a *App) WebhookVerify(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	mode := string(r.RequestCtx.QueryArgs().Peek("hub.mode"))
	token := string(r.RequestCtx.QueryArgs().Peek("hub.verify_token"))
	challenge := string(r.RequestCtx.QueryArgs().Peek("hub.challenge"))

	if mode != "subscribe" {
		a.Log.Warn("Webhook verification failed - invalid mode", "mode", mode)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Verification failed", nil, "")
	}

	// First check against global config token
	if token == a.Config.WhatsApp.WebhookVerifyToken && token != "" {
		a.Log.Info("Webhook verified successfully (global token)")
		r.RequestCtx.SetStatusCode(fasthttp.StatusOK)
		r.RequestCtx.SetBodyString(challenge)
		return nil
	}

	// Then check against tokens stored in WhatsApp accounts
	var account models.WhatsAppAccount
	result := requestDB.Where("webhook_verify_token = ?", token).First(&account)
	if result.Error == nil {
		a.Log.Info("Webhook verified successfully (account token)", "account", account.Name)
		r.RequestCtx.SetStatusCode(fasthttp.StatusOK)
		r.RequestCtx.SetBodyString(challenge)
		return nil
	}

	a.Log.Warn("Webhook verification failed - token not found")
	return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Verification failed", nil, "")
}

// WebhookStatusError represents an error in a status update
type WebhookStatusError struct {
	Code      int    `json:"code"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	ErrorData struct {
		Details string `json:"details"`
	} `json:"error_data"`
}

// TemplateStatusUpdate represents a template status update from Meta webhook
type TemplateStatusUpdate struct {
	Event                   string `json:"event"`
	MessageTemplateID       int64  `json:"message_template_id"`
	MessageTemplateName     string `json:"message_template_name"`
	MessageTemplateLanguage string `json:"message_template_language"`
	Reason                  string `json:"reason,omitempty"`
}

// WebhookStatus represents a message status update from Meta
type WebhookStatus struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp"`
	RecipientID  string `json:"recipient_id"`
	Conversation *struct {
		ID string `json:"id"`
	} `json:"conversation,omitempty"`
	Pricing *struct {
		Billable     bool   `json:"billable"`
		PricingModel string `json:"pricing_model"`
		Category     string `json:"category"`
	} `json:"pricing,omitempty"`
	Errors []WebhookStatusError `json:"errors,omitempty"`
}

// WebhookPayload represents the incoming webhook from Meta
type WebhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID      string `json:"phone_number_id"`
				} `json:"metadata"`
				// Template status update fields (when field == "message_template_status_update")
				Event                   string `json:"event,omitempty"`
				MessageTemplateID       int64  `json:"message_template_id,omitempty"`
				MessageTemplateName     string `json:"message_template_name,omitempty"`
				MessageTemplateLanguage string `json:"message_template_language,omitempty"`
				Reason                  string `json:"reason,omitempty"`
				Contacts                []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaID string `json:"wa_id"`
				} `json:"contacts"`
				Messages []IncomingTextMessage `json:"messages,omitempty"`
				Statuses []WebhookStatus       `json:"statuses,omitempty"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}

// WebhookHandler processes incoming webhook events from Meta
func (a *App) WebhookHandler(r *fastglue.Request) error {
	body := r.RequestCtx.PostBody()
	signature := r.RequestCtx.Request.Header.Peek("X-Hub-Signature-256")

	if len(body) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid payload", nil, "")
	}
	if len(body) > maxWebhookBodyBytes {
		a.Log.Warn("Rejected oversized webhook body", "size_bytes", len(body), "max_bytes", maxWebhookBodyBytes)
		return r.SendErrorEnvelope(fasthttp.StatusRequestEntityTooLarge, "Webhook payload too large", nil, "")
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		a.Log.Error("Failed to parse webhook payload", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid payload", nil, "")
	}

	if payload.Object != "whatsapp_business_account" {
		a.Log.Warn("Rejected webhook with invalid object", "object", payload.Object)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid webhook object", nil, "")
	}

	if err := a.validateWebhookRequest(body, signature, &payload); err != nil {
		a.Log.Warn("Rejected webhook request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Invalid webhook signature", nil, "")
	}

	eventCount := countWebhookEvents(&payload)
	if eventCount > maxWebhookEventsPerRequest {
		a.Log.Warn("Rejected oversized webhook payload", "event_count", eventCount, "max", maxWebhookEventsPerRequest)
		return r.SendErrorEnvelope(fasthttp.StatusRequestEntityTooLarge, "Webhook payload too large", nil, "")
	}

	// Process each entry
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			// Handle template status updates
			if change.Field == "message_template_status_update" {
				a.Log.Info("Received template status update",
					"event", change.Value.Event,
					"template_name", change.Value.MessageTemplateName,
					"template_language", change.Value.MessageTemplateLanguage,
					"waba_id", entry.ID,
				)
				a.processTemplateStatusUpdate(entry.ID, change.Value.Event, change.Value.MessageTemplateName, change.Value.MessageTemplateLanguage, change.Value.Reason)
				continue
			}

			if change.Field != "messages" {
				continue
			}

			phoneNumberID := change.Value.Metadata.PhoneNumberID

			profileNamesByWaID := make(map[string]string, len(change.Value.Contacts))
			for _, contact := range change.Value.Contacts {
				profileNamesByWaID[contact.WaID] = contact.Profile.Name
			}

			existingMessageIDs := a.fetchExistingIncomingMessageIDs(change.Value.Messages)
			seenMessageIDs := make(map[string]struct{}, len(change.Value.Messages))

			// Process messages
			for _, msg := range change.Value.Messages {
				a.Log.Info("Received message",
					"from", msg.From,
					"type", msg.Type,
					"phone_number_id", phoneNumberID,
				)

				if msg.ID != "" {
					if _, exists := existingMessageIDs[msg.ID]; exists {
						a.Log.Debug("Duplicate message detected, skipping", "message_id", msg.ID)
						continue
					}
					if _, seen := seenMessageIDs[msg.ID]; seen {
						a.Log.Debug("Duplicate message detected in payload, skipping", "message_id", msg.ID)
						continue
					}
					seenMessageIDs[msg.ID] = struct{}{}
				}

				a.processIncomingMessageWithoutDuplicateCheck(phoneNumberID, msg, profileNamesByWaID[msg.From])
			}

			a.processStatusUpdatesBatch(phoneNumberID, change.Value.Statuses)
		}
	}

	// Always respond with 200 to acknowledge receipt
	return r.SendEnvelope(map[string]string{"status": "ok"})
}

func (a *App) processIncomingMessageWithoutDuplicateCheck(phoneNumberID string, msg IncomingTextMessage, profileName string) {
	a.processIncomingMessageFull(phoneNumberID, msg, profileName)
}

func (a *App) fetchExistingIncomingMessageIDs(messages []IncomingTextMessage) map[string]struct{} {
	existingMessageIDs := make(map[string]struct{})
	if len(messages) == 0 {
		return existingMessageIDs
	}

	uniqueMessageIDs := make([]string, 0, len(messages))
	seenMessageIDs := make(map[string]struct{}, len(messages))
	for _, message := range messages {
		if message.ID == "" {
			continue
		}
		if _, exists := seenMessageIDs[message.ID]; exists {
			continue
		}
		seenMessageIDs[message.ID] = struct{}{}
		uniqueMessageIDs = append(uniqueMessageIDs, message.ID)
	}

	if len(uniqueMessageIDs) == 0 {
		return existingMessageIDs
	}

	var existingRecords []struct {
		WhatsAppMessageID string `gorm:"column:whats_app_message_id"`
	}
	if err := a.DB.Model(&models.Message{}).
		Select("whats_app_message_id").
		Where("whats_app_message_id IN ?", uniqueMessageIDs).
		Find(&existingRecords).Error; err != nil {
		a.Log.Error("Failed to fetch existing incoming message IDs", "error", err, "message_count", len(uniqueMessageIDs))
		return existingMessageIDs
	}

	for _, record := range existingRecords {
		if record.WhatsAppMessageID == "" {
			continue
		}
		existingMessageIDs[record.WhatsAppMessageID] = struct{}{}
	}

	return existingMessageIDs
}

func (a *App) processStatusUpdate(phoneNumberID string, status WebhookStatus) {
	messageID := status.ID
	statusValue := status.Status

	a.Log.Info("Processing status update", "message_id", messageID, "status", statusValue, "phone_number_id", phoneNumberID)

	// Update messages table - this also handles campaign stats via incrementCampaignStat
	a.updateMessageStatus(messageID, statusValue, status.Errors)
}

func (a *App) processStatusUpdatesBatch(phoneNumberID string, statuses []WebhookStatus) {
	if len(statuses) == 0 {
		return
	}

	uniqueMessageIDs := make([]string, 0, len(statuses))
	seenMessageIDs := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		if status.ID == "" {
			continue
		}
		if _, exists := seenMessageIDs[status.ID]; exists {
			continue
		}
		seenMessageIDs[status.ID] = struct{}{}
		uniqueMessageIDs = append(uniqueMessageIDs, status.ID)
	}

	messagesByWhatsAppID := make(map[string]*models.Message, len(uniqueMessageIDs))
	if len(uniqueMessageIDs) > 0 {
		var messages []models.Message
		if err := a.DB.
			Where("whats_app_message_id IN ?", uniqueMessageIDs).
			Find(&messages).Error; err != nil {
			a.Log.Error("Failed to batch-fetch messages for status updates", "error", err, "message_count", len(uniqueMessageIDs))
			// Preserve previous behavior by falling back to per-status queries.
			for _, status := range statuses {
				a.processStatusUpdate(phoneNumberID, status)
			}
			return
		}

		for i := range messages {
			message := &messages[i]
			if message.WhatsAppMessageID == "" {
				continue
			}
			messagesByWhatsAppID[message.WhatsAppMessageID] = message
		}
	}

	for _, status := range statuses {
		messageID := status.ID
		statusValue := status.Status

		a.Log.Info("Processing status update", "message_id", messageID, "status", statusValue, "phone_number_id", phoneNumberID)

		message, exists := messagesByWhatsAppID[messageID]
		if !exists {
			a.Log.Debug("No message found for status update", "whats_app_message_id", messageID)
			continue
		}

		a.applyMessageStatusUpdate(message, statusValue, status.Errors)
	}
}

// statusPriority returns the priority of a status (higher = more progressed)
func statusPriority(status models.MessageStatus) int {
	switch status {
	case models.MessageStatusPending:
		return 0
	case models.MessageStatusSent:
		return 1
	case models.MessageStatusDelivered:
		return 2
	case models.MessageStatusRead:
		return 3
	case models.MessageStatusFailed:
		return 4 // Failed can override any status
	default:
		return -1
	}
}

// updateMessageStatus updates the status of a regular message in the messages table
func (a *App) updateMessageStatus(whatsappMsgID, statusValue string, errors []WebhookStatusError) {
	// Find the message by WhatsApp message ID
	var message models.Message
	result := a.DB.Where("whats_app_message_id = ?", whatsappMsgID).First(&message)
	if result.Error != nil {
		a.Log.Debug("No message found for status update", "whats_app_message_id", whatsappMsgID)
		return
	}

	a.applyMessageStatusUpdate(&message, statusValue, errors)
}

func (a *App) applyMessageStatusUpdate(message *models.Message, statusValue string, errors []WebhookStatusError) {
	newStatus := models.MessageStatus(statusValue)
	currentPriority := statusPriority(message.Status)
	newPriority := statusPriority(newStatus)

	// Only update if new status is a progression (higher priority) or if it's failed
	if newPriority <= currentPriority && newStatus != models.MessageStatusFailed {
		a.Log.Debug("Ignoring status update - not a progression",
			"message_id", message.ID,
			"current_status", message.Status,
			"new_status", statusValue)
		return
	}

	updates := map[string]interface{}{}

	switch newStatus {
	case models.MessageStatusSent:
		updates["status"] = models.MessageStatusSent
	case models.MessageStatusDelivered:
		updates["status"] = models.MessageStatusDelivered
	case models.MessageStatusRead:
		updates["status"] = models.MessageStatusRead
	case models.MessageStatusFailed:
		updates["status"] = models.MessageStatusFailed
		if len(errors) > 0 {
			// Prefer error_data.details (most descriptive), then Message, then Title.
			errText := errors[0].ErrorData.Details
			if errText == "" {
				errText = errors[0].Message
			}
			if errText == "" || errText == errors[0].Title {
				errText = errors[0].Title
			}

			updates["error_message"] = errText
		}
	default:
		a.Log.Debug("Ignoring message status update", "status", statusValue)
		return
	}

	if err := a.DB.Model(message).Updates(updates).Error; err != nil {
		a.Log.Error("Failed to update message status", "error", err, "message_id", message.ID)
		return
	}

	message.Status = newStatus
	if errorMessage, ok := updates["error_message"].(string); ok {
		message.ErrorMessage = errorMessage
	}

	a.Log.Info("Updated message status", "message_id", message.ID, "status", statusValue)

	var publisher *queue.Publisher
	if a.Redis != nil {
		publisher = queue.NewPublisher(a.Redis, a.Log)
	}
	campaignstats.ApplyMessageReceipt(context.Background(), a.DB, publisher, a.Log, message, newStatus)

	// Broadcast status update via WebSocket
	if a.WSHub != nil {
		wsPayload := map[string]any{
			"message_id": message.ID.String(),
			"status":     statusValue,
		}
		if errMsg, ok := updates["error_message"].(string); ok && errMsg != "" {
			wsPayload["error_message"] = errMsg
		}
		a.WSHub.BroadcastToOrg(message.OrganizationID, websocket.WSMessage{
			Type:    websocket.TypeStatusUpdate,
			Payload: wsPayload,
		})
	}
}

// processTemplateStatusUpdate updates template status when Meta sends a status update webhook
func (a *App) processTemplateStatusUpdate(wabaID, event, templateName, templateLanguage, reason string) {
	if templateName == "" {
		a.Log.Warn("Template status update missing template name")
		return
	}

	// Keep status uppercase to match existing template status format
	// Events: APPROVED, REJECTED, PENDING, DISABLED, PENDING_DELETION, DELETED, REINSTATED, FLAGGED
	status := strings.ToUpper(event)

	// Find WhatsApp accounts that use this WABA ID (business_id field)
	var accounts []models.WhatsAppAccount
	if err := a.DB.Where("business_id = ?", wabaID).Find(&accounts).Error; err != nil {
		a.Log.Error("Failed to find WhatsApp accounts for WABA", "error", err, "waba_id", wabaID)
		return
	}

	if len(accounts) == 0 {
		a.Log.Warn("No WhatsApp accounts found for WABA", "waba_id", wabaID)
		return
	}

	// Update template for each account that has it
	for _, account := range accounts {
		// Find and update the template
		result := a.DB.Model(&models.Template{}).
			Where("whats_app_account = ? AND name = ? AND language = ?", account.Name, templateName, templateLanguage).
			Update("status", status)

		if result.Error != nil {
			a.Log.Error("Failed to update template status",
				"error", result.Error,
				"account", account.Name,
				"template", templateName,
				"language", templateLanguage,
			)
			continue
		}

		if result.RowsAffected > 0 {
			a.Log.Info("Updated template status from webhook",
				"account", account.Name,
				"template", templateName,
				"language", templateLanguage,
				"status", status,
				"reason", reason,
			)
		}
	}
}

// verifyWebhookSignature verifies the X-Hub-Signature-256 header from Meta.
// The signature is HMAC-SHA256 of the request body using the App Secret.
func verifyWebhookSignature(body, signature, appSecret []byte) bool {
	// Signature format: "sha256=<hex_signature>"
	prefix := []byte("sha256=")
	if !bytes.HasPrefix(signature, prefix) {
		return false
	}

	expectedSig := bytes.TrimPrefix(signature, prefix)

	// Compute HMAC-SHA256
	mac := hmac.New(sha256.New, appSecret)
	mac.Write(body)
	computedSig := make([]byte, hex.EncodedLen(mac.Size()))
	hex.Encode(computedSig, mac.Sum(nil))

	// Constant-time comparison to prevent timing attacks
	return hmac.Equal(expectedSig, computedSig)
}
