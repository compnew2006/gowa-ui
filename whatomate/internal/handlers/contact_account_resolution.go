package handlers

import (
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

// resolveContactMessageAccount returns the best available WhatsApp account for
// system-generated chat messages and backfills the contact row when possible.
func (a *App) resolveContactMessageAccount(contact *models.Contact) string {
	if a == nil || contact == nil {
		return ""
	}

	if account := strings.TrimSpace(contact.WhatsAppAccount); account != "" {
		return account
	}

	resolvedAccount := a.lookupLatestContactMessageAccount(contact.OrganizationID, contact.ID)
	if resolvedAccount == "" && contact.InstanceID != nil {
		var instance models.WhatsAppInstance
		if err := a.DB.
			Select("phone_number").
			Where("id = ? AND organization_id = ?", *contact.InstanceID, contact.OrganizationID).
			First(&instance).Error; err == nil {
			resolvedAccount = strings.TrimSpace(instance.PhoneNumber)
		}
	}
	if resolvedAccount == "" {
		return ""
	}

	if err := a.DB.Model(&models.Contact{}).
		Where("id = ?", contact.ID).
		Update("whats_app_account", resolvedAccount).Error; err != nil {
		a.Log.Warn("Failed to persist resolved contact account", "contact_id", contact.ID, "error", err)
	} else {
		contact.WhatsAppAccount = resolvedAccount
	}

	return resolvedAccount
}

func (a *App) resolveOutboundMessageAccount(
	orgID uuid.UUID,
	contact *models.Contact,
	requestedAccount string,
	selectedInstance *models.WhatsAppInstance,
) (*models.WhatsAppAccount, error) {
	accountName := strings.TrimSpace(requestedAccount)

	if a.isWhatsmeowProvider() || a.isGowaProvider() {
		if accountName == "" && selectedInstance != nil {
			accountName = strings.TrimSpace(selectedInstance.PhoneNumber)
		}
		if accountName == "" {
			accountName = a.resolveContactMessageAccount(contact)
		}

		return &models.WhatsAppAccount{
			OrganizationID: orgID,
			Name:           accountName,
		}, nil
	}

	if accountName == "" && contact != nil {
		accountName = strings.TrimSpace(contact.WhatsAppAccount)
	}

	return a.resolveWhatsAppAccount(orgID, accountName)
}

func (a *App) lookupLatestContactMessageAccount(orgID, contactID uuid.UUID) string {
	if a == nil {
		return ""
	}

	var latestMessage models.Message
	if err := a.DB.
		Select("whats_app_account").
		Where("organization_id = ? AND contact_id = ? AND COALESCE(whats_app_account, '') <> ''", orgID, contactID).
		Order("created_at DESC").
		First(&latestMessage).Error; err != nil {
		return ""
	}

	return strings.TrimSpace(latestMessage.WhatsAppAccount)
}
