package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestConversationNoteTableName(t *testing.T) {
	note := ConversationNote{}
	if name := note.TableName(); name != "conversation_notes" {
		t.Errorf("Expected table name 'conversation_notes', got '%s'", name)
	}
}

func TestConversationNoteFields(t *testing.T) {
	orgID := uuid.New()
	contactID := uuid.New()
	userID := uuid.New()

	note := ConversationNote{
		OrganizationID: orgID,
		ContactID:      contactID,
		CreatedByID:    userID,
		Content:        "Test note content",
	}

	if note.OrganizationID != orgID {
		t.Error("OrganizationID not set correctly")
	}
	if note.ContactID != contactID {
		t.Error("ContactID not set correctly")
	}
	if note.CreatedByID != userID {
		t.Error("CreatedByID not set correctly")
	}
	if note.Content != "Test note content" {
		t.Error("Content not set correctly")
	}
}
