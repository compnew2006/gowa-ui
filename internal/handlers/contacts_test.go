package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestContactsConversationUnreadKeyWithNilInstance(t *testing.T) {
	result := conversationUnreadKey("conv-abc", nil)
	assert.Equal(t, "conv-abc|", result)
}

func TestContactsConversationUnreadKeyWithInstance(t *testing.T) {
	inst := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	result := conversationUnreadKey("conv-abc", &inst)
	assert.Contains(t, result, "conv-abc|")
	assert.Contains(t, result, "550e8400")
}

func TestContactsCloneJSONBIsolation(t *testing.T) {
	original := models.JSONB{"key": "value"}
	cloned := cloneJSONB(original)
	cloned["key"] = "modified"
	assert.Equal(t, "value", original["key"])
}
