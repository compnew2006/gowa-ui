package whatsmeow

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/types"
)

const (
	avatarURLMetadataKey      = "avatar_url"
	avatarSyncedAtMetadataKey = "avatar_synced_at"
)

var avatarRefreshCooldown = 6 * time.Hour
var errInvalidProfilePhotoJID = errors.New("invalid profile photo jid")

func (cm *ConnectionManager) scheduleContactAvatarRefresh(instanceID uuid.UUID, contact *models.Contact) {
	if cm == nil || cm.db == nil || contact == nil {
		return
	}

	now := time.Now().UTC()
	if !shouldRefreshAvatar(contact.Metadata, now) {
		return
	}

	if _, inFlight := cm.avatarSync.LoadOrStore(contact.ID, struct{}{}); inFlight {
		return
	}

	contactID := contact.ID
	go func() {
		defer cm.avatarSync.Delete(contactID)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cm.refreshContactAvatar(ctx, instanceID, contactID)
	}()
}

// ScheduleContactAvatarRefresh queues a best-effort background avatar sync for a contact.
func (cm *ConnectionManager) ScheduleContactAvatarRefresh(instanceID uuid.UUID, contact *models.Contact) {
	cm.scheduleContactAvatarRefresh(instanceID, contact)
}

func shouldRefreshAvatar(metadata models.JSONB, now time.Time) bool {
	raw := metadataString(metadata, avatarSyncedAtMetadataKey)
	if raw == "" {
		return true
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return true
	}
	return now.Sub(parsed) >= avatarRefreshCooldown
}

func (cm *ConnectionManager) refreshContactAvatar(ctx context.Context, instanceID, contactID uuid.UUID) {
	client := cm.GetClient(instanceID)
	if client == nil {
		return
	}

	var contact models.Contact
	if err := cm.db.WithContext(ctx).
		Select("id", "organization_id", "phone_number", "metadata").
		Where("id = ?", contactID).
		First(&contact).Error; err != nil {
		return
	}

	targetJID, err := profilePhotoTargetJID(&contact)
	if err != nil {
		return
	}

	avatarURL := ""
	previousAvatarURL := strings.TrimSpace(metadataString(contact.Metadata, avatarURLMetadataKey))
	info, err := client.GetProfilePictureInfo(ctx, targetJID, nil)
	if err != nil {
		cm.logger.Debug("Profile photo fetch skipped", "instance_id", instanceID, "contact_id", contact.ID, "error", err)
	} else if info != nil {
		avatarURL = strings.TrimSpace(info.URL)
	}

	metadata := cloneProfileMetadata(contact.Metadata)
	metadata = buildAvatarMetadataForPersistence(metadata, avatarURL, time.Now().UTC())

	if err := cm.db.WithContext(ctx).
		Model(&models.Contact{}).
		Where("id = ?", contact.ID).
		Update("metadata", metadata).Error; err != nil {
		cm.logger.Debug("Failed to persist profile photo metadata", "instance_id", instanceID, "contact_id", contact.ID, "error", err)
		return
	}

	if cm.hub != nil && avatarURL != previousAvatarURL {
		cm.hub.BroadcastToOrg(contact.OrganizationID, websocket.WSMessage{
			Type: websocket.TypeContactUpdate,
			Payload: map[string]any{
				"id":         contact.ID.String(),
				"avatar_url": avatarURL,
			},
		})
	}
}

func profilePhotoTargetJID(contact *models.Contact) (types.JID, error) {
	if contact == nil {
		return types.JID{}, errInvalidProfilePhotoJID
	}

	raw := strings.TrimSpace(metadataString(contact.Metadata, "group_jid"))
	if raw == "" {
		raw = strings.TrimSpace(metadataString(contact.Metadata, "channel_jid"))
	}
	if raw == "" {
		raw = strings.TrimSpace(contact.PhoneNumber)
	}
	if raw == "" {
		return types.JID{}, errInvalidProfilePhotoJID
	}

	if strings.Contains(raw, "@") {
		return types.ParseJID(raw)
	}

	normalizedUser := normalizeProfilePhotoUser(raw)
	if normalizedUser == "" {
		return types.JID{}, errInvalidProfilePhotoJID
	}

	return types.ParseJID(normalizedUser + "@s.whatsapp.net")
}

func cloneProfileMetadata(metadata models.JSONB) models.JSONB {
	if metadata == nil {
		return nil
	}
	cloned := make(models.JSONB, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func buildAvatarMetadataForPersistence(metadata models.JSONB, avatarURL string, syncedAt time.Time) models.JSONB {
	if metadata == nil {
		metadata = models.JSONB{}
	}

	metadata[avatarSyncedAtMetadataKey] = syncedAt.UTC().Format(time.RFC3339)
	trimmedURL := strings.TrimSpace(avatarURL)
	if trimmedURL == "" {
		delete(metadata, avatarURLMetadataKey)
		return metadata
	}

	metadata[avatarURLMetadataKey] = trimmedURL
	return metadata
}

func normalizeProfilePhotoUser(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	// WhatsApp user part is numeric for individual accounts.
	normalized := make([]rune, 0, len(trimmed))
	for _, ch := range trimmed {
		if ch >= '0' && ch <= '9' {
			normalized = append(normalized, ch)
		}
	}
	return string(normalized)
}
