package whatsmeow

import (
	"context"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

func (cm *ConnectionManager) renderAutoRejectReplyMessage(ctx context.Context, orgID uuid.UUID, contact *models.Contact, conversationID, message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}

	usage := detectAutoRejectMessagePlaceholderUsage(trimmed)
	if !usage.any() {
		return trimmed
	}

	data := autoRejectMessageTemplateData{
		CustomerName: resolveAutoRejectCustomerName(contact),
		ContactName:  resolveAutoRejectContactName(contact),
		PhoneNumber:  resolveAutoRejectPhoneNumber(contact),
		ChatID:       resolveAutoRejectChatID(contact, conversationID),
		AgentName:    autoRejectDefaultAgentName,
	}
	if usage.organizationName {
		data.OrganizationName = cm.resolveAutoRejectOrganizationName(ctx, orgID)
	} else {
		data.OrganizationName = autoRejectDefaultOrganizationName
	}
	if usage.agentName {
		var assignedUserID *uuid.UUID
		if contact != nil {
			assignedUserID = contact.AssignedUserID
		}
		data.AgentName = cm.resolveAutoRejectAgentName(ctx, assignedUserID)
	}

	return strings.TrimSpace(renderAutoRejectMessageTemplate(trimmed, data))
}

func resolveAutoRejectCustomerName(contact *models.Contact) string {
	if contact == nil {
		return autoRejectDefaultCustomerName
	}
	if name := strings.TrimSpace(contact.ProfileName); name != "" {
		return name
	}
	if number := strings.TrimSpace(contact.PhoneNumber); number != "" {
		return number
	}
	return autoRejectDefaultCustomerName
}

func resolveAutoRejectContactName(contact *models.Contact) string {
	return resolveAutoRejectCustomerName(contact)
}

func resolveAutoRejectPhoneNumber(contact *models.Contact) string {
	if contact == nil {
		return ""
	}
	return strings.TrimSpace(contact.PhoneNumber)
}

func resolveAutoRejectChatID(contact *models.Contact, conversationID string) string {
	if contact != nil && contact.ID != uuid.Nil {
		return contact.ID.String()
	}
	if id := strings.TrimSpace(conversationID); id != "" {
		return id
	}
	return ""
}

func (cm *ConnectionManager) resolveAutoRejectAgentName(ctx context.Context, assignedUserID *uuid.UUID) string {
	if cm == nil || cm.db == nil || assignedUserID == nil || *assignedUserID == uuid.Nil {
		return autoRejectDefaultAgentName
	}

	var user models.User
	if err := cm.db.WithContext(ctx).Select("id", "full_name").Where("id = ?", *assignedUserID).First(&user).Error; err != nil {
		return autoRejectDefaultAgentName
	}
	if name := strings.TrimSpace(user.FullName); name != "" {
		return name
	}
	return autoRejectDefaultAgentName
}

func (cm *ConnectionManager) resolveAutoRejectOrganizationName(ctx context.Context, orgID uuid.UUID) string {
	if cm == nil || cm.db == nil || orgID == uuid.Nil {
		return autoRejectDefaultOrganizationName
	}

	var org models.Organization
	if err := cm.db.WithContext(ctx).Select("id", "name").Where("id = ?", orgID).First(&org).Error; err != nil {
		return autoRejectDefaultOrganizationName
	}
	if name := strings.TrimSpace(org.Name); name != "" {
		return name
	}
	return autoRejectDefaultOrganizationName
}
