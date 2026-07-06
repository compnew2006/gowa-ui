package whatsmeow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	waClient "go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm/clause"
)

// StatusTextStyle defines optional text-style metadata for text statuses.
type StatusTextStyle struct {
	TextARGB       *uint32
	BackgroundARGB *uint32
	Font           *waE2E.ExtendedTextMessage_FontType
}

func (cm *ConnectionManager) handleStatusMessage(ctx context.Context, evt *events.Message, instanceID, orgID uuid.UUID) {
	if cm == nil || evt == nil || evt.Message == nil {
		return
	}

	client := cm.GetClient(instanceID)
	if client == nil || client.Store == nil {
		cm.logger.Warn("Skipping status event: instance client unavailable", "instance_id", instanceID, "wa_message_id", evt.Info.ID)
		return
	}

	normalizedEvt := cm.normalizeIncomingEventMessage(ctx, client, evt, instanceID)
	if normalizedEvt == nil || normalizedEvt.Message == nil {
		return
	}

	statusType, content, mediaURL, mimeType, filename, _ := cm.extractMessageContentWithMediaRetryArtifact(ctx, client, normalizedEvt.Message)
	modelStatusType, ok := toWhatsAppStatusType(statusType)
	if !ok {
		return
	}

	createdAt := normalizedEvt.Info.Timestamp
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	senderJID := resolveStatusSenderJID(client, normalizedEvt, instanceID)
	if senderJID == "" {
		cm.logger.Warn("Skipping status event: missing sender jid", "instance_id", instanceID, "wa_message_id", normalizedEvt.Info.ID)
		return
	}

	textARGB, bgARGB, font := extractStatusTextStyle(normalizedEvt.Message)
	status := &models.WhatsAppStatus{
		BaseModel: models.BaseModel{
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		},
		OrganizationID:    orgID,
		InstanceID:        instanceID,
		WhatsAppAccount:   resolveWhatsmeowAccountID(client),
		SenderJID:         senderJID,
		SenderName:        strings.TrimSpace(normalizedEvt.Info.PushName),
		WhatsAppMessageID: strings.TrimSpace(normalizedEvt.Info.ID),
		StatusType:        modelStatusType,
		Content:           strings.TrimSpace(content),
		MediaURL:          strings.TrimSpace(mediaURL),
		MediaMimeType:     strings.TrimSpace(mimeType),
		MediaFilename:     strings.TrimSpace(filename),
		TextARGB:          textARGB,
		BackgroundARGB:    bgARGB,
		Font:              font,
		ExpiresAt:         createdAt.Add(24 * time.Hour),
		Metadata: models.JSONB{
			"from_me":          normalizedEvt.Info.IsFromMe,
			"status_chat_jid":  normalizedEvt.Info.Chat.String(),
			"sender_push_name": strings.TrimSpace(normalizedEvt.Info.PushName),
		},
	}

	if err := cm.persistStatusRecord(ctx, status); err != nil {
		cm.logger.Error("Failed to persist WhatsApp status", "error", err, "instance_id", instanceID, "wa_message_id", status.WhatsAppMessageID)
	}
}

func toWhatsAppStatusType(msgType models.MessageType) (models.WhatsAppStatusType, bool) {
	switch msgType {
	case models.MessageTypeText:
		return models.WhatsAppStatusTypeText, true
	case models.MessageTypeImage, models.MessageTypeSticker:
		return models.WhatsAppStatusTypeImage, true
	case models.MessageTypeVideo:
		return models.WhatsAppStatusTypeVideo, true
	default:
		return "", false
	}
}

func resolveStatusSenderJID(client *waClient.Client, evt *events.Message, instanceID uuid.UUID) string {
	if evt == nil {
		return ""
	}
	if evt.Info.IsFromMe {
		if client != nil && client.Store != nil && client.Store.ID != nil {
			return client.Store.ID.ToNonAD().String()
		}
		return ""
	}

	sender := evt.Info.Sender.ToNonAD()
	if sender.User != "" && sender.Server != "" {
		return sender.String()
	}

	senderAlt := evt.Info.SenderAlt.ToNonAD()
	if senderAlt.User != "" && senderAlt.Server != "" {
		return senderAlt.String()
	}

	recipientAlt := evt.Info.RecipientAlt.ToNonAD()
	if recipientAlt.User != "" && recipientAlt.Server != "" {
		return recipientAlt.String()
	}

	_ = instanceID
	return ""
}

func extractStatusTextStyle(msg *waE2E.Message) (*int64, *int64, string) {
	if msg == nil {
		return nil, nil, ""
	}

	extended := msg.GetExtendedTextMessage()
	if extended == nil {
		return nil, nil, ""
	}

	var textARGB *int64
	if extended.TextArgb != nil {
		value := int64(extended.GetTextArgb())
		textARGB = &value
	}

	var bgARGB *int64
	if extended.BackgroundArgb != nil {
		value := int64(extended.GetBackgroundArgb())
		bgARGB = &value
	}

	font := ""
	if extended.Font != nil {
		font = extended.GetFont().String()
	}

	return textARGB, bgARGB, font
}

// persistStatusRecord inserts a status row atomically.
//
// Status broadcast events can reach this code path twice for the same
// (instance_id, whats_app_message_id) within milliseconds — e.g. once via the
// low-priority shard (status broadcasts are demoted there) and once via the
// legacy/normal ingestion path. A naive read-then-write check is not atomic
// (TOCTOU): both goroutines see "not found" and both attempt Create(), and the
// second hits the partial unique index idx_status_instance_wamid. We instead
// rely on INSERT ... ON CONFLICT DO NOTHING so the loser becomes a silent
// no-op at the DB level instead of a logged error + transaction rollback.
//
// On conflict the passed status.ID is left as the client-generated value; we do
// not reload the existing row because callers only need to know the insert was
// safe, not the survivor's DB-assigned identity.
func (cm *ConnectionManager) persistStatusRecord(ctx context.Context, status *models.WhatsAppStatus) error {
	if cm == nil || status == nil {
		return fmt.Errorf("status is nil")
	}

	return cm.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(status).Error
}

// SendTextStatus sends a text status update to status@broadcast for a specific instance.
func (cm *ConnectionManager) SendTextStatus(ctx context.Context, instanceID uuid.UUID, text string, style StatusTextStyle) (string, error) {
	client := cm.GetClient(instanceID)
	if client == nil || !client.IsConnected() {
		return "", fmt.Errorf("instance not connected")
	}

	trimmedText := strings.TrimSpace(text)
	if trimmedText == "" {
		return "", fmt.Errorf("status text is required")
	}

	extended := &waE2E.ExtendedTextMessage{
		Text: proto.String(trimmedText),
	}
	if style.TextARGB != nil {
		extended.TextArgb = proto.Uint32(*style.TextARGB)
	}
	if style.BackgroundARGB != nil {
		extended.BackgroundArgb = proto.Uint32(*style.BackgroundARGB)
	}
	if style.Font != nil {
		extended.Font = style.Font
	}

	resp, err := client.SendMessage(ctx, types.StatusBroadcastJID, &waE2E.Message{
		ExtendedTextMessage: extended,
	})
	if err != nil {
		return "", fmt.Errorf("failed to send text status: %w", err)
	}

	return resp.ID, nil
}

// SendStatusReadReceipt sends a read receipt for a status message.
func (cm *ConnectionManager) SendStatusReadReceipt(ctx context.Context, instanceID uuid.UUID, senderJID, messageID string) error {
	client := cm.GetClient(instanceID)
	if client == nil || !client.IsConnected() {
		return fmt.Errorf("instance not connected")
	}

	trimmedSender := strings.TrimSpace(senderJID)
	if trimmedSender == "" {
		return fmt.Errorf("sender_jid is required")
	}
	trimmedMessageID := strings.TrimSpace(messageID)
	if trimmedMessageID == "" {
		return fmt.Errorf("message_id is required")
	}

	parsedSender, err := types.ParseJID(trimmedSender)
	if err != nil {
		if !strings.Contains(trimmedSender, "@") {
			parsedSender, err = types.ParseJID(trimmedSender + "@s.whatsapp.net")
		}
		if err != nil {
			return fmt.Errorf("invalid sender jid: %w", err)
		}
	}

	return client.MarkRead(
		ctx,
		[]types.MessageID{types.MessageID(trimmedMessageID)},
		time.Now(),
		types.StatusBroadcastJID,
		parsedSender.ToNonAD(),
	)
}
