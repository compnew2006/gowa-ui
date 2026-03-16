package handlers

import (
	"context"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

const (
	defaultTemplateCustomerName     = "Customer"
	defaultTemplateAgentName        = "Agent"
	defaultTemplateOrganizationName = "Organization"
)

type messageTemplatePlaceholderUsage struct {
	customerName     bool
	chatID           bool
	agentName        bool
	organizationName bool
	contactName      bool
	phoneNumber      bool
}

func (u messageTemplatePlaceholderUsage) any() bool {
	return u.customerName || u.chatID || u.agentName || u.organizationName || u.contactName || u.phoneNumber
}

type messageTemplatePlaceholderData struct {
	CustomerName     string
	ChatID           string
	AgentName        string
	OrganizationName string
	ContactName      string
	PhoneNumber      string
}

func (a *App) renderMessageTemplatePlaceholders(ctx context.Context, orgID uuid.UUID, contact *models.Contact, message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}

	usage := detectMessageTemplatePlaceholderUsage(trimmed)
	if !usage.any() {
		return trimmed
	}

	data := messageTemplatePlaceholderData{
		CustomerName: resolveTemplateCustomerName(contact),
		ContactName:  resolveTemplateContactName(contact),
		PhoneNumber:  resolveTemplatePhoneNumber(contact),
		ChatID:       resolveTemplateChatID(contact),
		AgentName:    defaultTemplateAgentName,
	}

	if usage.agentName {
		data.AgentName = a.resolveTemplateAgentName(ctx, contact)
	}
	if usage.organizationName {
		data.OrganizationName = a.resolveTemplateOrganizationName(ctx, orgID)
	} else {
		data.OrganizationName = defaultTemplateOrganizationName
	}

	return strings.TrimSpace(renderMessageTemplate(trimmed, data))
}

func detectMessageTemplatePlaceholderUsage(template string) messageTemplatePlaceholderUsage {
	return messageTemplatePlaceholderUsage{
		customerName:     containsMessageTemplatePlaceholder(template, "customer_name"),
		chatID:           containsMessageTemplatePlaceholder(template, "chat_id"),
		agentName:        containsMessageTemplatePlaceholder(template, "agent_name"),
		organizationName: containsMessageTemplatePlaceholder(template, "organization_name"),
		contactName:      containsMessageTemplatePlaceholder(template, "contact_name"),
		phoneNumber:      containsMessageTemplatePlaceholder(template, "phone_number"),
	}
}

func containsMessageTemplatePlaceholder(template, name string) bool {
	return strings.Contains(template, "{"+name+"}") || strings.Contains(template, "{{"+name+"}}")
}

func renderMessageTemplate(template string, data messageTemplatePlaceholderData) string {
	return strings.NewReplacer(
		"{{customer_name}}", data.CustomerName,
		"{{chat_id}}", data.ChatID,
		"{{agent_name}}", data.AgentName,
		"{{organization_name}}", data.OrganizationName,
		"{{contact_name}}", data.ContactName,
		"{{phone_number}}", data.PhoneNumber,
		"{customer_name}", data.CustomerName,
		"{chat_id}", data.ChatID,
		"{agent_name}", data.AgentName,
		"{organization_name}", data.OrganizationName,
		"{contact_name}", data.ContactName,
		"{phone_number}", data.PhoneNumber,
	).Replace(template)
}

func resolveTemplateCustomerName(contact *models.Contact) string {
	if contact == nil {
		return defaultTemplateCustomerName
	}
	if name := strings.TrimSpace(contact.ProfileName); name != "" {
		return name
	}
	if number := strings.TrimSpace(contact.PhoneNumber); number != "" {
		return number
	}
	return defaultTemplateCustomerName
}

func resolveTemplateContactName(contact *models.Contact) string {
	return resolveTemplateCustomerName(contact)
}

func resolveTemplatePhoneNumber(contact *models.Contact) string {
	if contact == nil {
		return ""
	}
	return strings.TrimSpace(contact.PhoneNumber)
}

func resolveTemplateChatID(contact *models.Contact) string {
	if contact == nil || contact.ID == uuid.Nil {
		return ""
	}
	return contact.ID.String()
}

func (a *App) resolveTemplateAgentName(ctx context.Context, contact *models.Contact) string {
	if a == nil || a.DB == nil || contact == nil || contact.AssignedUserID == nil || *contact.AssignedUserID == uuid.Nil {
		return defaultTemplateAgentName
	}

	if resolved := strings.TrimSpace(a.ResolveActivityActorName(*contact.AssignedUserID)); resolved != "" {
		return resolved
	}
	return defaultTemplateAgentName
}

func (a *App) resolveTemplateOrganizationName(ctx context.Context, orgID uuid.UUID) string {
	if a == nil || a.DB == nil || orgID == uuid.Nil {
		return defaultTemplateOrganizationName
	}

	var org models.Organization
	if err := a.DB.WithContext(ctx).Select("id", "name").Where("id = ?", orgID).First(&org).Error; err != nil {
		return defaultTemplateOrganizationName
	}
	if name := strings.TrimSpace(org.Name); name != "" {
		return name
	}
	return defaultTemplateOrganizationName
}
