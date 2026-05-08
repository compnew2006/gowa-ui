package worker

import (
	"context"
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/templateutil"
	"github.com/google/uuid"
)

const (
	defaultCampaignCustomerName     = "Customer"
	defaultCampaignAgentName        = "Agent"
	defaultCampaignOrganizationName = "Organization"
)

type campaignTemplatePlaceholderUsage struct {
	customerName     bool
	chatID           bool
	agentName        bool
	organizationName bool
	contactName      bool
	phoneNumber      bool
}

func (u campaignTemplatePlaceholderUsage) any() bool {
	return u.customerName || u.chatID || u.agentName || u.organizationName || u.contactName || u.phoneNumber
}

func (w *Worker) resolveCampaignTemplateParams(
	ctx context.Context,
	orgID uuid.UUID,
	contact *models.Contact,
	fallbackPhone string,
	fallbackRecipientName string,
	templateBody string,
	existing map[string]any,
) models.JSONB {
	base := make(models.JSONB, len(existing))
	for key, value := range existing {
		base[key] = value
	}

	usage := detectCampaignTemplatePlaceholderUsage(templateBody)
	if !usage.any() {
		return base
	}

	defaults := models.JSONB{
		"customer_name": resolveCampaignCustomerName(contact, fallbackRecipientName),
		"contact_name":  resolveCampaignContactName(contact, fallbackRecipientName),
		"phone_number":  resolveCampaignPhoneNumber(contact, fallbackPhone),
		"chat_id":       resolveCampaignChatID(contact),
		"agent_name":    defaultCampaignAgentName,
	}
	if usage.agentName {
		defaults["agent_name"] = w.resolveCampaignAgentName(ctx, contact)
	}
	if usage.organizationName {
		defaults["organization_name"] = w.resolveCampaignOrganizationName(ctx, orgID)
	} else {
		defaults["organization_name"] = defaultCampaignOrganizationName
	}

	for key, value := range defaults {
		base[key] = value
	}
	for key, value := range existing {
		base[key] = value
	}

	return base
}

func detectCampaignTemplatePlaceholderUsage(template string) campaignTemplatePlaceholderUsage {
	return campaignTemplatePlaceholderUsage{
		customerName:     containsCampaignTemplatePlaceholder(template, "customer_name"),
		chatID:           containsCampaignTemplatePlaceholder(template, "chat_id"),
		agentName:        containsCampaignTemplatePlaceholder(template, "agent_name"),
		organizationName: containsCampaignTemplatePlaceholder(template, "organization_name"),
		contactName:      containsCampaignTemplatePlaceholder(template, "contact_name"),
		phoneNumber:      containsCampaignTemplatePlaceholder(template, "phone_number"),
	}
}

func containsCampaignTemplatePlaceholder(template, name string) bool {
	return strings.Contains(template, "{"+name+"}") || strings.Contains(template, "{{"+name+"}}")
}

func resolveCampaignCustomerName(contact *models.Contact, fallbackRecipientName string) string {
	if contact != nil {
		if name := strings.TrimSpace(contact.ProfileName); name != "" {
			return name
		}
		if phone := strings.TrimSpace(contact.PhoneNumber); phone != "" {
			return phone
		}
	}
	if fallback := strings.TrimSpace(fallbackRecipientName); fallback != "" {
		return fallback
	}
	return defaultCampaignCustomerName
}

func resolveCampaignContactName(contact *models.Contact, fallbackRecipientName string) string {
	return resolveCampaignCustomerName(contact, fallbackRecipientName)
}

func resolveCampaignPhoneNumber(contact *models.Contact, fallbackPhone string) string {
	if contact != nil {
		if phone := strings.TrimSpace(contact.PhoneNumber); phone != "" {
			return phone
		}
	}
	return strings.TrimSpace(fallbackPhone)
}

func resolveCampaignChatID(contact *models.Contact) string {
	if contact == nil || contact.ID == uuid.Nil {
		return ""
	}
	return contact.ID.String()
}

func (w *Worker) resolveCampaignAgentName(ctx context.Context, contact *models.Contact) string {
	if w == nil || w.DB == nil || contact == nil || contact.AssignedUserID == nil || *contact.AssignedUserID == uuid.Nil {
		return defaultCampaignAgentName
	}

	var user struct {
		FullName string `gorm:"column:full_name"`
		Email    string `gorm:"column:email"`
	}
	if err := w.DB.WithContext(ctx).
		Model(&models.User{}).
		Select("full_name", "email").
		Where("id = ?", *contact.AssignedUserID).
		Take(&user).Error; err != nil {
		return defaultCampaignAgentName
	}
	if name := strings.TrimSpace(user.FullName); name != "" {
		return name
	}
	if email := strings.TrimSpace(user.Email); email != "" {
		return email
	}
	return defaultCampaignAgentName
}

func (w *Worker) resolveCampaignOrganizationName(ctx context.Context, orgID uuid.UUID) string {
	if w == nil || w.DB == nil || orgID == uuid.Nil {
		return defaultCampaignOrganizationName
	}

	var org models.Organization
	if err := w.DB.WithContext(ctx).
		Select("id", "name").
		Where("id = ?", orgID).
		First(&org).Error; err != nil {
		return defaultCampaignOrganizationName
	}
	if name := strings.TrimSpace(org.Name); name != "" {
		return name
	}
	return defaultCampaignOrganizationName
}

func renderCampaignTemplateBody(templateBody string, params map[string]any) string {
	rendered := templateutil.ReplaceWithJSONBParams(templateBody, templateBody, params)
	return strings.NewReplacer(
		"{{customer_name}}", campaignTemplateValue(params, "customer_name"),
		"{{chat_id}}", campaignTemplateValue(params, "chat_id"),
		"{{agent_name}}", campaignTemplateValue(params, "agent_name"),
		"{{organization_name}}", campaignTemplateValue(params, "organization_name"),
		"{{contact_name}}", campaignTemplateValue(params, "contact_name"),
		"{{phone_number}}", campaignTemplateValue(params, "phone_number"),
		"{customer_name}", campaignTemplateValue(params, "customer_name"),
		"{chat_id}", campaignTemplateValue(params, "chat_id"),
		"{agent_name}", campaignTemplateValue(params, "agent_name"),
		"{organization_name}", campaignTemplateValue(params, "organization_name"),
		"{contact_name}", campaignTemplateValue(params, "contact_name"),
		"{phone_number}", campaignTemplateValue(params, "phone_number"),
	).Replace(rendered)
}

func campaignTemplateValue(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	value, ok := params[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}
