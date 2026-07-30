package contactutil

import (
	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"gorm.io/gorm"
)

// GetOrCreateContact finds or creates a contact for the given phone number.
// Merges behaviors from both handler and worker implementations:
//   - Normalizes phone (strips leading "+")
//   - Tries both normalized and +prefix forms
//   - Updates profile name if changed
//   - Handles race conditions on create by re-fetching
//   - Restores soft-deleted contacts if found
//
// Returns the contact, whether it was newly created, and any error.
func GetOrCreateContact(db *gorm.DB, orgID uuid.UUID, phoneNumber, profileName string) (*models.Contact, bool, error) {
	// Normalize phone number (remove + prefix if present)
	normalizedPhone := phoneNumber
	if len(normalizedPhone) > 0 && normalizedPhone[0] == '+' {
		normalizedPhone = normalizedPhone[1:]
	}

	// Try to find existing contact with normalized phone (including soft-deleted)
	var contact models.Contact
	if err := db.Unscoped().Where("organization_id = ? AND phone_number = ?", orgID, normalizedPhone).First(&contact).Error; err == nil {
		// Restore if soft-deleted
		if contact.DeletedAt.Valid {
			db.Unscoped().Model(&contact).Update("deleted_at", nil)
			contact.DeletedAt.Valid = false
		}
		// Update profile name if changed
		if profileName != "" && contact.ProfileName != profileName {
			db.Model(&contact).Update("profile_name", profileName)
		}
		return &contact, false, nil
	}

	// Also try with + prefix (contacts may have been stored with it)
	if err := db.Unscoped().Where("organization_id = ? AND phone_number = ?", orgID, "+"+normalizedPhone).First(&contact).Error; err == nil {
		// Restore if soft-deleted
		if contact.DeletedAt.Valid {
			db.Unscoped().Model(&contact).Update("deleted_at", nil)
			contact.DeletedAt.Valid = false
		}
		if profileName != "" && contact.ProfileName != profileName {
			db.Model(&contact).Update("profile_name", profileName)
		}
		return &contact, false, nil
	}

	// Create new contact
	contact = models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		PhoneNumber:    normalizedPhone,
		ProfileName:    profileName,
	}
	if err := db.Create(&contact).Error; err != nil {
		// Race condition: another goroutine may have created the contact
		if err2 := db.Unscoped().Where("organization_id = ? AND phone_number = ?", orgID, normalizedPhone).First(&contact).Error; err2 == nil {
			// Restore if soft-deleted
			if contact.DeletedAt.Valid {
				db.Unscoped().Model(&contact).Update("deleted_at", nil)
				contact.DeletedAt.Valid = false
			}
			return &contact, false, nil
		}
		return nil, false, err
	}
	return &contact, true, nil
}

// StampChatCategory marks the contact as a group chat (is_group_chat) or
// newsletter (is_newsletter) in its metadata. The two categories are mutually
// exclusive — a @newsletter JID is NOT a group — so setting one clears the
// other, letting legacy contacts that carry both flags self-heal on the next
// stamp. No-op for plain 1:1 chats or when the flags are already correct.
// Shared by the webhook, chatbot-processor, and GOWA sync paths so the badge
// convention lives in one place.
func StampChatCategory(db *gorm.DB, contact *models.Contact, isGroup, isNewsletter bool) error {
	metaKey := ""
	if isGroup {
		metaKey = "is_group_chat"
	} else if isNewsletter {
		metaKey = "is_newsletter"
	}
	if metaKey == "" {
		return nil
	}
	if contact.Metadata == nil {
		contact.Metadata = models.JSONB{}
	}
	otherKey := "is_newsletter"
	if metaKey == "is_newsletter" {
		otherKey = "is_group_chat"
	}
	_, hasOther := contact.Metadata[otherKey]
	if contact.Metadata[metaKey] == true && !hasOther {
		return nil
	}
	contact.Metadata[metaKey] = true
	delete(contact.Metadata, otherKey)
	return db.Model(contact).Update("metadata", contact.Metadata).Error
}

// StampAccountName overwrites the contact's owning WhatsApp account name so
// an empty or stale value self-heals (webhook-created rows can carry one that
// would otherwise break the Contacts UI's account filter). NOTE: the DB
// column is whats_app_account — GORM's mapping of the WhatsAppAccount field,
// not whatsapp_account — so the raw Update must use that name. No-op when the
// contact already carries the account.
func StampAccountName(db *gorm.DB, contact *models.Contact, accountName string) error {
	if contact.WhatsAppAccount == accountName {
		return nil
	}
	if err := db.Model(contact).Update("whats_app_account", accountName).Error; err != nil {
		return err
	}
	contact.WhatsAppAccount = accountName
	return nil
}
