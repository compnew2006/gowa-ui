package whatsmeow

import (
	"context"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// handleEvent processes incoming events from whatsmeow client.
func (cm *ConnectionManager) handleEvent(evt interface{}, instanceID, orgID uuid.UUID) {
	switch v := evt.(type) {
	case *events.Message:
		if cm.handleAnyReaction(context.Background(), v, instanceID, orgID) {
			return
		}
		cm.handleMessage(context.Background(), v, instanceID, orgID)

	case *events.Receipt:
		cm.handleReceipt(context.Background(), v, instanceID, orgID)

	case *events.HistorySync:
		cm.handleHistorySync(context.Background(), v, instanceID, orgID)

	case *events.CallOffer:
		cm.handleCallOffer(context.Background(), v, instanceID, orgID)

	case *events.CallOfferNotice:
		cm.handleCallOfferNotice(context.Background(), v, instanceID, orgID)

	case *events.CallPreAccept:
		cm.markCallActive(instanceID, v.CallID)

	case *events.CallAccept:
		cm.markCallActive(instanceID, v.CallID)

	case *events.CallTransport:
		cm.markCallActive(instanceID, v.CallID)

	case *events.CallTerminate:
		cm.markCallEnded(instanceID, v.CallID)

	case *events.CallReject:
		cm.markCallEnded(instanceID, v.CallID)

	case *events.QR:
		code := ""
		if len(v.Codes) > 0 {
			code = v.Codes[0]
		}
		cm.CacheQRCode(instanceID, code, 20)

		if cm.hub != nil {
			cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
				Type: websocket.TypeInstanceQRCode,
				Payload: websocket.QRCodePayload{
					InstanceID: instanceID.String(),
					QRCode:     code,
					TimeoutSec: 20,
				},
			})
		}
		cm.logger.Info("QR code received", "component", "whatsmeow", "event", "qr_code", "instance_id", instanceID)

	case *events.PairSuccess:
		cm.ClearCachedQRCode(instanceID)
		jid := v.ID.String()
		phoneNumber := v.ID.User

		exists, err := cm.checkDuplicateJID(context.Background(), orgID, instanceID, jid)
		if err != nil {
			cm.logger.Error("Failed to check duplicate JID", "component", "whatsmeow", "event", "duplicate_jid_check_error", "error", err)
			cm.MarkError(instanceID)
		} else if exists {
			cm.logger.Warn("Duplicate JID detected, disconnecting", "component", "whatsmeow", "event", "duplicate_jid", "instance_id", instanceID, "jid", jid)
			if cm.hub != nil {
				cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
					Type: websocket.TypeInstanceReconnectFailed,
					Payload: websocket.InstanceReconnectFailedPayload{
						InstanceID: instanceID.String(),
						Reason:     "duplicate_jid",
						Message:    "This WhatsApp account is already connected to another instance.",
					},
				})
			}
			if err := cm.Disconnect(context.Background(), instanceID); err != nil {
				cm.logger.Error("Failed to disconnect duplicate instance", "error", err, "instance_id", instanceID)
			}
			return
		}

		if err := cm.updateInstanceIdentity(context.Background(), instanceID, jid, phoneNumber); err != nil {
			cm.logger.Error("Failed to update instance identity on pair success", "error", err)
			cm.MarkError(instanceID)
		}

		if err := cm.updateInstanceStatus(context.Background(), instanceID, models.InstanceStatusConnected); err != nil {
			cm.logger.Error("Failed to update status on pair success", "error", err)
			cm.MarkError(instanceID)
		}
		if err := cm.clearInstanceSendBlock(context.Background(), instanceID); err != nil {
			cm.logger.Warn("Failed to clear send block on pair success", "error", err, "instance_id", instanceID)
		}
		cm.MarkConnected(instanceID)
		cm.broadcastInstanceConnected(orgID, instanceID, phoneNumber)
		cm.logger.Info("Instance paired successfully", "component", "whatsmeow", "event", "pair_success", "instance_id", instanceID, "jid", jid)

	case *events.Connected:
		cm.ClearCachedQRCode(instanceID)
		phoneNumber := ""
		var instance models.WhatsAppInstance
		if err := cm.db.WithContext(context.Background()).
			Select("phone_number").
			Where("id = ?", instanceID).
			First(&instance).Error; err == nil {
			phoneNumber = instance.PhoneNumber
		} else {
			cm.logger.Debug("Failed to resolve phone number on connect event", "component", "whatsmeow", "event", "connect_phone_lookup_failed", "instance_id", instanceID, "error", err)
		}

		if err := cm.updateInstanceStatus(context.Background(), instanceID, models.InstanceStatusConnected); err != nil {
			cm.logger.Error("Failed to update status on connect", "error", err)
			cm.MarkError(instanceID)
		}
		if err := cm.clearInstanceSendBlock(context.Background(), instanceID); err != nil {
			cm.logger.Warn("Failed to clear send block on connect", "error", err, "instance_id", instanceID)
		}
		cm.MarkConnected(instanceID)
		cm.broadcastInstanceConnected(orgID, instanceID, phoneNumber)
		cm.logger.Info("Instance connected", "component", "whatsmeow", "event", "connected", "instance_id", instanceID)

	case *events.Disconnected:
		cm.ClearCachedQRCode(instanceID)
		cm.clearActiveCalls(instanceID)
		if err := cm.updateInstanceStatus(context.Background(), instanceID, models.InstanceStatusDisconnected); err != nil {
			cm.logger.Error("Failed to update status on disconnect", "error", err)
			cm.MarkError(instanceID)
		}
		cm.MarkDisconnected(instanceID)

		if cm.hub != nil {
			cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
				Type: websocket.TypeInstanceDisconnected,
				Payload: websocket.InstancePayload{
					InstanceID: instanceID.String(),
					Status:     string(models.InstanceStatusDisconnected),
				},
			})
		}
		cm.logger.Info("Instance disconnected", "component", "whatsmeow", "event", "disconnected", "instance_id", instanceID)

	case *events.TemporaryBan:
		cm.ClearCachedQRCode(instanceID)
		cm.clearActiveCalls(instanceID)
		reason := strings.TrimSpace(v.String())
		if reason == "" {
			reason = "WhatsApp temporary ban detected"
		}
		blockedUntil := time.Now().UTC().Add(24 * time.Hour)
		if err := cm.updateInstanceSendBlock(context.Background(), instanceID, &blockedUntil, reason); err != nil {
			cm.logger.Warn("Failed to persist temporary send block", "error", err, "instance_id", instanceID)
		}
		cm.pauseActiveCampaignsForInstance(context.Background(), orgID, instanceID, "temporary_ban")
		if err := cm.updateInstanceStatus(context.Background(), instanceID, models.InstanceStatusBanned); err != nil {
			cm.logger.Error("Failed to update status on ban", "error", err)
			cm.MarkError(instanceID)
		}
		cm.MarkDisconnected(instanceID)

		notification, err := cm.createInstanceNotification(
			context.Background(),
			orgID,
			instanceID,
			"banned",
			reason,
		)
		if err != nil {
			cm.logger.Error("Failed to create banned notification", "error", err, "instance_id", instanceID)
			cm.MarkError(instanceID)
		}

		if cm.hub != nil {
			cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
				Type: websocket.TypeInstanceBanned,
				Payload: websocket.InstancePayload{
					InstanceID: instanceID.String(),
					Status:     string(models.InstanceStatusBanned),
				},
			})
		}
		cm.broadcastInstanceNotification(orgID, notification)
		cm.logger.Warn("Instance temporarily banned", "component", "whatsmeow", "event", "banned", "instance_id", instanceID, "reason", reason, "blocked_until", blockedUntil)

	case *events.LoggedOut:
		cm.ClearCachedQRCode(instanceID)
		cm.clearActiveCalls(instanceID)
		logoutReason := "WhatsApp session was logged out. Reconnect and scan QR code again."
		blockedUntil := time.Now().UTC().Add(24 * time.Hour)
		if err := cm.updateInstanceSendBlock(context.Background(), instanceID, &blockedUntil, logoutReason); err != nil {
			cm.logger.Warn("Failed to persist logged-out send block", "error", err, "instance_id", instanceID)
		}
		cm.pauseActiveCampaignsForInstance(context.Background(), orgID, instanceID, "logged_out")
		if err := cm.updateInstanceIdentity(context.Background(), instanceID, "", ""); err != nil {
			cm.logger.Error("Failed to clear instance identity on logout", "error", err)
			cm.MarkError(instanceID)
		}

		if err := cm.updateInstanceStatus(context.Background(), instanceID, models.InstanceStatusLoggedOut); err != nil {
			cm.logger.Error("Failed to update status on logout", "error", err)
			cm.MarkError(instanceID)
		}
		cm.MarkDisconnected(instanceID)

		notification, err := cm.createInstanceNotification(
			context.Background(),
			orgID,
			instanceID,
			"logged_out",
			logoutReason,
		)
		if err != nil {
			cm.logger.Error("Failed to create logged out notification", "error", err, "instance_id", instanceID)
			cm.MarkError(instanceID)
		}

		if cm.hub != nil {
			cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
				Type: websocket.TypeInstanceLoggedOut,
				Payload: websocket.InstancePayload{
					InstanceID: instanceID.String(),
					Status:     string(models.InstanceStatusLoggedOut),
				},
			})
		}
		cm.broadcastInstanceNotification(orgID, notification)
		cm.logger.Info("Instance logged out", "component", "whatsmeow", "event", "logged_out", "instance_id", instanceID)
	}
}

// handleReceipt processes incoming read/delivered/sent receipts from WhatsApp.
func (cm *ConnectionManager) handleReceipt(ctx context.Context, evt *events.Receipt, instanceID, orgID uuid.UUID) {
	if evt == nil {
		return
	}
	if isStatusMessageSource(evt.MessageSource) {
		return
	}

	var newStatus models.MessageStatus
	switch evt.Type {
	case types.ReceiptTypeRead, types.ReceiptTypeReadSelf:
		newStatus = models.MessageStatusRead
	case types.ReceiptTypeDelivered:
		newStatus = models.MessageStatusDelivered
	default:
		return
	}

	for _, msgID := range evt.MessageIDs {
		trimmedMessageID := strings.TrimSpace(string(msgID))
		if trimmedMessageID == "" {
			continue
		}
		if cm.isStatusReceiptMessageID(ctx, orgID, instanceID, trimmedMessageID) {
			continue
		}

		result := cm.db.WithContext(ctx).
			Model(&models.Message{}).
			Where("whats_app_message_id = ? AND instance_id = ? AND status NOT IN ?",
				trimmedMessageID, instanceID, statusesAtOrAbove(newStatus)).
			Update("status", newStatus)

		if result.Error != nil {
			cm.logger.Error("Failed to update message receipt status",
				"error", result.Error,
				"message_id", trimmedMessageID,
				"new_status", newStatus)
			cm.MarkError(instanceID)
			continue
		}

		if result.RowsAffected == 0 {
			continue
		}

		var message models.Message
		if err := cm.db.WithContext(ctx).
			Where("whats_app_message_id = ? AND instance_id = ?", trimmedMessageID, instanceID).
			First(&message).Error; err != nil {
			continue
		}

		if cm.hub != nil {
			cm.hub.BroadcastToOrg(orgID, websocket.WSMessage{
				Type: websocket.TypeStatusUpdate,
				Payload: websocket.StatusUpdatePayload{
					MessageID: message.ID.String(),
					Status:    string(newStatus),
				},
			})
		}

		cm.logger.Debug("Receipt processed",
			"wa_message_id", trimmedMessageID,
			"status", newStatus,
			"contact_id", message.ContactID)
	}
}

func (cm *ConnectionManager) isStatusReceiptMessageID(ctx context.Context, orgID, instanceID uuid.UUID, waMessageID string) bool {
	if cm == nil || cm.db == nil {
		return false
	}
	trimmedMessageID := strings.TrimSpace(waMessageID)
	if trimmedMessageID == "" {
		return false
	}

	var count int64
	if err := cm.db.WithContext(ctx).
		Model(&models.WhatsAppStatus{}).
		Where("organization_id = ? AND instance_id = ? AND whats_app_message_id = ?",
			orgID, instanceID, trimmedMessageID).
		Count(&count).Error; err != nil {
		cm.logger.Debug("Failed to check status receipt message id", "message_id", trimmedMessageID, "error", err)
		return false
	}

	return count > 0
}

// statusesAtOrAbove returns statuses that are at or above the given status.
func statusesAtOrAbove(status models.MessageStatus) []models.MessageStatus {
	switch status {
	case models.MessageStatusRead:
		return []models.MessageStatus{models.MessageStatusRead}
	case models.MessageStatusDelivered:
		return []models.MessageStatus{models.MessageStatusDelivered, models.MessageStatusRead}
	case models.MessageStatusSent:
		return []models.MessageStatus{models.MessageStatusSent, models.MessageStatusDelivered, models.MessageStatusRead}
	default:
		return nil
	}
}
