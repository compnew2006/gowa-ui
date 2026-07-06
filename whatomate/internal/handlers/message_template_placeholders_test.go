package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDetectMessageTemplatePlaceholderUsage(t *testing.T) {
	usage := detectMessageTemplatePlaceholderUsage(
		"Hi {customer_name} ({contact_name}) {phone_number} chat {chat_id} with {agent_name} at {organization_name}",
	)

	assert.True(t, usage.customerName)
	assert.True(t, usage.contactName)
	assert.True(t, usage.phoneNumber)
	assert.True(t, usage.chatID)
	assert.True(t, usage.agentName)
	assert.True(t, usage.organizationName)
	assert.True(t, usage.any())
}

func TestRenderMessageTemplate(t *testing.T) {
	message := "Hi {customer_name} ({contact_name}) {phone_number} chat {chat_id} with {agent_name} at {organization_name}"
	rendered := renderMessageTemplate(message, messageTemplatePlaceholderData{
		CustomerName:     "Alex",
		ContactName:      "Alex",
		PhoneNumber:      "15551234567",
		ChatID:           "chat-123",
		AgentName:        "Sam",
		OrganizationName: "Acme",
	})

	assert.Equal(t, "Hi Alex (Alex) 15551234567 chat chat-123 with Sam at Acme", rendered)
}

func TestResolveTemplateValuesFromContact(t *testing.T) {
	contactID := uuid.New()
	contact := &models.Contact{
		BaseModel:   models.BaseModel{ID: contactID},
		ProfileName: "Alex",
		PhoneNumber: "15551234567",
	}

	assert.Equal(t, "Alex", resolveTemplateCustomerName(contact))
	assert.Equal(t, "Alex", resolveTemplateContactName(contact))
	assert.Equal(t, "15551234567", resolveTemplatePhoneNumber(contact))
	assert.Equal(t, contactID.String(), resolveTemplateChatID(contact))
}
