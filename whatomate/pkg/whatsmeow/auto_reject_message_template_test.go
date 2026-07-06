package whatsmeow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectAutoRejectMessagePlaceholderUsage(t *testing.T) {
	usage := detectAutoRejectMessagePlaceholderUsage("Hi {customer_name}, chat {chat_id} with {{agent_name}} at {organization_name} for {contact_name} ({phone_number})")

	assert.True(t, usage.customerName)
	assert.True(t, usage.chatID)
	assert.True(t, usage.agentName)
	assert.True(t, usage.organizationName)
	assert.True(t, usage.contactName)
	assert.True(t, usage.phoneNumber)
	assert.True(t, usage.any())
}

func TestRenderAutoRejectMessageTemplate_SingleBraces(t *testing.T) {
	result := renderAutoRejectMessageTemplate(
		"Hi {customer_name}, your chat {chat_id} with {agent_name} at {organization_name} was auto-rejected.",
		autoRejectMessageTemplateData{
			CustomerName:     "Alex",
			ChatID:           "chat-123",
			AgentName:        "Sam",
			OrganizationName: "Acme",
			ContactName:      "Alex",
			PhoneNumber:      "15551234567",
		},
	)

	assert.Equal(t, "Hi Alex, your chat chat-123 with Sam at Acme was auto-rejected.", result)
}

func TestRenderAutoRejectMessageTemplate_DoubleBraces(t *testing.T) {
	result := renderAutoRejectMessageTemplate(
		"Hi {{customer_name}}, your chat {{chat_id}} with {{agent_name}} at {{organization_name}} was auto-rejected.",
		autoRejectMessageTemplateData{
			CustomerName:     "Alex",
			ChatID:           "chat-123",
			AgentName:        "Sam",
			OrganizationName: "Acme",
			ContactName:      "Alex",
			PhoneNumber:      "15551234567",
		},
	)

	assert.Equal(t, "Hi Alex, your chat chat-123 with Sam at Acme was auto-rejected.", result)
}

func TestRenderAutoRejectMessageTemplate_ContactAndPhone(t *testing.T) {
	result := renderAutoRejectMessageTemplate(
		"Caller {contact_name} ({phone_number})",
		autoRejectMessageTemplateData{
			ContactName: "Alex",
			PhoneNumber: "15551234567",
		},
	)

	assert.Equal(t, "Caller Alex (15551234567)", result)
}
