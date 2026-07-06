package handlers

import (
	"errors"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"gorm.io/gorm"
)

// RepairDirectContactPhoneFromConversation keeps private-chat contacts on canonical PN identity.
// It uses the latest direct conversation JID (`<pn>@s.whatsapp.net`) as source of truth.
func RepairDirectContactPhoneFromConversation(db *gorm.DB, contact *models.Contact, conversationID string) error {
	if db == nil || contact == nil {
		return nil
	}
	if isGroupContact(contact) || isChannelContact(contact) {
		return nil
	}

	canonicalPhone := strings.TrimSpace(directUserFromConversationID(conversationID))
	if canonicalPhone == "" || canonicalPhone == strings.TrimSpace(contact.PhoneNumber) {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		target := models.Contact{}
		targetQuery := tx.Where("organization_id = ? AND phone_number = ?", contact.OrganizationID, canonicalPhone)
		if contact.InstanceID != nil {
			targetQuery = targetQuery.Where("instance_id = ?", *contact.InstanceID)
		} else {
			targetQuery = targetQuery.Where("instance_id IS NULL")
		}

		if err := targetQuery.First(&target).Error; err == nil {
			if target.ID == contact.ID {
				contact.PhoneNumber = canonicalPhone
				return nil
			}

			if err := tx.Model(&models.Message{}).
				Where("organization_id = ? AND contact_id = ?", contact.OrganizationID, contact.ID).
				Update("contact_id", target.ID).Error; err != nil {
				return err
			}

			profileName := strings.TrimSpace(contact.ProfileName)
			if (target.ProfileName == "" || target.ProfileName == target.PhoneNumber) && profileName != "" && profileName != canonicalPhone {
				if err := tx.Model(&models.Contact{}).Where("id = ?", target.ID).Update("profile_name", profileName).Error; err != nil {
					return err
				}
				target.ProfileName = profileName
			}

			if err := tx.Delete(&models.Contact{}, "id = ?", contact.ID).Error; err != nil {
				return err
			}

			*contact = target
			contact.PhoneNumber = canonicalPhone
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Model(&models.Contact{}).Where("id = ?", contact.ID).Update("phone_number", canonicalPhone).Error; err != nil {
			return err
		}
		contact.PhoneNumber = canonicalPhone
		return nil
	})
}
