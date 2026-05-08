package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestContactRepairNilDB(t *testing.T) {
	contact := &models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: uuid.New(),
	}
	err := RepairDirectContactPhoneFromConversation(nil, contact, "1234567890@s.whatsapp.net")
	assert.NoError(t, err)
}

func TestContactRepairNilContact(t *testing.T) {
	err := RepairDirectContactPhoneFromConversation(nil, nil, "1234567890@s.whatsapp.net")
	assert.NoError(t, err)
}

func TestContactRepairDirectUserFromConversationID(t *testing.T) {
	tests := []struct {
		input string
		exp   string
	}{
		{"1234567890@s.whatsapp.net", "1234567890"},
		{"", ""},
		{"invalid", ""},
		{"@s.whatsapp.net", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.exp, directUserFromConversationID(tt.input))
		})
	}
}

func TestContactRepairIsGroupContact(t *testing.T) {
	assert.False(t, isGroupContact(nil))
	assert.False(t, isGroupContact(&models.Contact{}))
}

func TestContactRepairIsChannelContact(t *testing.T) {
	assert.False(t, isChannelContact(nil))
	assert.False(t, isChannelContact(&models.Contact{}))
}
