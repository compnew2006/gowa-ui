package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// MarkMessageRead marks one or more messages as read via the configured provider.
// POST /api/messages/read
// Body: { "message_ids": ["uuid1", "uuid2"] }
func (a *App) MarkMessageRead(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var req struct {
		MessageIDs []string `json:"message_ids"`
	}
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	if len(req.MessageIDs) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "message_ids is required", nil, "")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var errs []string
	for _, idStr := range req.MessageIDs {
		msgID, err := uuid.Parse(idStr)
		if err != nil {
			errs = append(errs, fmt.Sprintf("invalid message ID %s", idStr))
			continue
		}

		var message models.Message
		if err := requestDB.Where("id = ? AND organization_id = ?", msgID, orgID).First(&message).Error; err != nil {
			errs = append(errs, fmt.Sprintf("message %s not found", idStr))
			continue
		}

		// Only mark incoming messages as read
		if message.Direction != models.DirectionIncoming {
			continue
		}

		if message.WhatsAppMessageID == "" {
			continue
		}

		// Route through the appropriate provider
		if a.isWhatsmeowProvider() && a.MessageProvider != nil && message.InstanceID != nil {
			if err := a.MessageProvider.MarkRead(ctx, message.InstanceID.String(), message.WhatsAppMessageID); err != nil {
				a.Log.Error("Failed to mark message read via provider", "error", err, "message_id", msgID)
				errs = append(errs, fmt.Sprintf("failed to mark %s as read: %s", idStr, err.Error()))
				continue
			}
		} else if a.WhatsApp != nil {
			// Meta path: resolve account and call MarkMessageRead
			var account models.WhatsAppAccount
			if err := requestDB.Where("name = ? AND organization_id = ?", message.WhatsAppAccount, orgID).First(&account).Error; err == nil {
				waAccount := a.toWhatsAppAccount(&account)
				if err := a.WhatsApp.MarkMessageRead(ctx, waAccount, message.WhatsAppMessageID); err != nil {
					a.Log.Error("Failed to mark message read via Meta", "error", err, "message_id", msgID)
					errs = append(errs, fmt.Sprintf("failed to mark %s as read: %s", idStr, err.Error()))
					continue
				}
			}
		}
		requestDB.

			// Update message status in DB
			Model(&models.Message{}).Where("id = ?", msgID).Update("status", models.MessageStatusRead)

		// Broadcast status update
		if a.WSHub != nil {
			a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
				Type: websocket.TypeStatusUpdate,
				Payload: websocket.StatusUpdatePayload{
					MessageID: msgID.String(),
					Status:    string(models.MessageStatusRead),
				},
			})
		}
	}

	if len(errs) > 0 {
		return r.SendEnvelope(map[string]any{
			"status": "partial",
			"errors": errs,
		})
	}

	return r.SendEnvelope(map[string]any{
		"status": "ok",
	})
}

// Analytics handlers
func (a *App) GetMessageAnalytics(r *fastglue.Request) error {
	return r.SendErrorEnvelope(fasthttp.StatusNotImplemented, "Not implemented yet", nil, "")
}

func (a *App) GetChatbotAnalytics(r *fastglue.Request) error {
	return r.SendErrorEnvelope(fasthttp.StatusNotImplemented, "Not implemented yet", nil, "")
}
