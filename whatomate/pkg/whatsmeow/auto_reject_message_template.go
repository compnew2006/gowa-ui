package whatsmeow

import "strings"

const (
	autoRejectDefaultCustomerName     = "Customer"
	autoRejectDefaultAgentName        = "Agent"
	autoRejectDefaultOrganizationName = "Organization"
)

type autoRejectMessagePlaceholderUsage struct {
	customerName     bool
	chatID           bool
	agentName        bool
	organizationName bool
	contactName      bool
	phoneNumber      bool
}

func (u autoRejectMessagePlaceholderUsage) any() bool {
	return u.customerName || u.chatID || u.agentName || u.organizationName || u.contactName || u.phoneNumber
}

type autoRejectMessageTemplateData struct {
	CustomerName     string
	ChatID           string
	AgentName        string
	OrganizationName string
	ContactName      string
	PhoneNumber      string
}

func detectAutoRejectMessagePlaceholderUsage(template string) autoRejectMessagePlaceholderUsage {
	return autoRejectMessagePlaceholderUsage{
		customerName:     containsAutoRejectPlaceholder(template, "customer_name"),
		chatID:           containsAutoRejectPlaceholder(template, "chat_id"),
		agentName:        containsAutoRejectPlaceholder(template, "agent_name"),
		organizationName: containsAutoRejectPlaceholder(template, "organization_name"),
		contactName:      containsAutoRejectPlaceholder(template, "contact_name"),
		phoneNumber:      containsAutoRejectPlaceholder(template, "phone_number"),
	}
}

func containsAutoRejectPlaceholder(template, name string) bool {
	return strings.Contains(template, "{"+name+"}") || strings.Contains(template, "{{"+name+"}}")
}

func renderAutoRejectMessageTemplate(template string, data autoRejectMessageTemplateData) string {
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
