package whatsmeow

import (
	"context"

	"github.com/google/uuid"
	"github.com/compnew2006/whatomate/internal/models"
	waClient "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"gorm.io/gorm"
)

func (cm *ConnectionManager) lookupPNForLID(ctx context.Context, lidUser string) string {
	if lidUser == "" {
		return ""
	}
	var pn string
	err := cm.db.WithContext(ctx).
		Table("whatsmeow_lid_map").
		Select("pn").
		Where("lid = ?", lidUser).
		Limit(1).
		Scan(&pn).Error
	if err == nil && pn != "" {
		return pn
	}
	return ""
}

// resolveSenderPhone resolves message sender to a phone number, preferring PN identity.
func (cm *ConnectionManager) resolveSenderPhone(ctx context.Context, client *waClient.Client, info types.MessageInfo) string {
	sender := info.Sender.ToNonAD()
	senderAlt := info.SenderAlt.ToNonAD()

	if senderAlt.Server == types.DefaultUserServer && senderAlt.User != "" {
		return senderAlt.User
	}

	if senderAlt.Server == types.HiddenUserServer && senderAlt.User != "" {
		if pn := cm.lookupPNForLID(ctx, senderAlt.User); pn != "" {
			return pn
		}
	}

	if sender.Server == types.HiddenUserServer && client != nil && client.Store != nil && client.Store.LIDs != nil {
		pn, err := client.Store.LIDs.GetPNForLID(ctx, sender)
		if err != nil {
			cm.logger.Warn("Failed to resolve LID to phone number", "sender_lid", sender.String(), "error", err)
		} else if pn.Server == types.DefaultUserServer && pn.User != "" {
			return pn.User
		}
	}

	if sender.User != "" &&
		(sender.Server == types.HiddenUserServer ||
			info.AddressingMode == types.AddressingModeLID ||
			info.Chat.Server == types.HiddenUserServer) {
		if pn := cm.lookupPNForLID(ctx, sender.User); pn != "" {
			return pn
		}
	}

	if sender.Server == types.DefaultUserServer && sender.User != "" {
		return sender.User
	}
	if senderAlt.User != "" {
		return senderAlt.User
	}
	return sender.User
}

// migrateContactPhoneFromLID backfills an existing LID-based contact to PN once mapping is known.
// Contacts are scoped by instance to avoid cross-instance merges for the same phone number.
func (cm *ConnectionManager) migrateContactPhoneFromLID(ctx context.Context, orgID, instanceID uuid.UUID, lidPhone, pnPhone string) {
	if lidPhone == "" || pnPhone == "" || lidPhone == pnPhone {
		return
	}

	var lidContact models.Contact
	err := cm.db.WithContext(ctx).
		Where("organization_id = ? AND phone_number = ? AND instance_id = ?", orgID, lidPhone, instanceID).
		First(&lidContact).Error
	if err == gorm.ErrRecordNotFound {
		err = cm.db.WithContext(ctx).
			Where("organization_id = ? AND phone_number = ? AND instance_id IS NULL", orgID, lidPhone).
			First(&lidContact).Error
	}
	if err != nil {
		return
	}

	var pnContact models.Contact
	err = cm.db.WithContext(ctx).
		Where("organization_id = ? AND phone_number = ? AND instance_id = ?", orgID, pnPhone, instanceID).
		First(&pnContact).Error
	if err == gorm.ErrRecordNotFound {
		err = cm.db.WithContext(ctx).
			Where("organization_id = ? AND phone_number = ? AND instance_id IS NULL", orgID, pnPhone).
			First(&pnContact).Error
	}
	if err == nil {
		_ = cm.db.WithContext(ctx).
			Model(&models.Message{}).
			Where("contact_id = ?", lidContact.ID).
			Update("contact_id", pnContact.ID).Error

		updates := map[string]any{}
		if pnContact.ProfileName == "" && lidContact.ProfileName != "" {
			updates["profile_name"] = lidContact.ProfileName
		}
		if pnContact.InstanceID == nil && lidContact.InstanceID != nil {
			updates["instance_id"] = lidContact.InstanceID
		}
		if len(updates) > 0 {
			_ = cm.db.WithContext(ctx).Model(&pnContact).Updates(updates).Error
		}
		_ = cm.db.WithContext(ctx).Delete(&lidContact).Error
		return
	}
	if err == gorm.ErrRecordNotFound {
		_ = cm.db.WithContext(ctx).Model(&lidContact).Update("phone_number", pnPhone).Error
	}
}

// updateInstanceIdentity updates the JID and phone number of an instance.
func (cm *ConnectionManager) updateInstanceIdentity(ctx context.Context, instanceID uuid.UUID, jid, phoneNumber string) error {
	return cm.db.WithContext(ctx).Model(&models.WhatsAppInstance{}).
		Where("id = ?", instanceID).
		Updates(map[string]interface{}{
			"jid":          jid,
			"phone_number": phoneNumber,
		}).Error
}

// checkDuplicateJID checks if the JID is already used by another instance in the same organization.
func (cm *ConnectionManager) checkDuplicateJID(ctx context.Context, orgID, instanceID uuid.UUID, jid string) (bool, error) {
	var count int64
	err := cm.db.WithContext(ctx).Model(&models.WhatsAppInstance{}).
		Where("organization_id = ? AND jid = ? AND id != ?", orgID, jid, instanceID).
		Count(&count).Error
	return count > 0, err
}
