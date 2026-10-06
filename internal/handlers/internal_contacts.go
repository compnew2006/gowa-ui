package handlers

import (
	"strings"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/google/uuid"
)

// Internal conversations are chats between two of the org's OWN connected
// WhatsApp numbers (e.g. the Saudi account messaging the Egypt account). Each
// side stores the other number as an ordinary contact, so without this marker
// they mix into the customer queue and trigger customer automations (away
// replies, close-rating prompts) against a colleague's number.

// accountPhoneFromJID extracts the bare phone from an account's connected
// JID, dropping any ":<device>" suffix ("9665...:12@s.whatsapp.net").
func accountPhoneFromJID(jid string) string {
	phone := gowa.PhoneFromJID(jid)
	if idx := strings.Index(phone, ":"); idx > 0 {
		phone = phone[:idx]
	}
	return phone
}

// orgAccountPhones maps each connected org account's phone number to its
// account Name. Accounts that never reported a JID are skipped.
func (a *App) orgAccountPhones(orgID uuid.UUID) map[string]string {
	var accounts []models.WhatsAppAccount
	if err := a.DB.Select("name", "gowa_jid").
		Where("organization_id = ? AND gowa_jid <> ''", orgID).
		Find(&accounts).Error; err != nil {
		a.Log.Error("Failed to load org account phones", "error", err, "org_id", orgID)
		return nil
	}
	phones := make(map[string]string, len(accounts))
	for _, acc := range accounts {
		if phone := accountPhoneFromJID(acc.GowaJID); phone != "" {
			phones[phone] = acc.Name
		}
	}
	return phones
}

// isInternalPhone reports whether phone is one of the org's own connected
// numbers, i.e. the conversation is with a colleague, not a customer.
func (a *App) isInternalPhone(orgID uuid.UUID, phone string) bool {
	if phone == "" {
		return false
	}
	_, ok := a.orgAccountPhones(orgID)[phone]
	return ok
}

// markInternalContact flags resp as an internal conversation when the
// contact's (unmasked) phone is one of the org's own numbers.
func markInternalContact(resp *ContactResponse, phone string, accountPhones map[string]string) {
	if name, ok := accountPhones[phone]; ok {
		resp.IsInternal = true
		resp.InternalAccountName = name
	}
}
