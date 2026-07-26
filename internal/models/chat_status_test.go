package models

import (
	"testing"

	"github.com/google/uuid"
)

// TestEffectiveStatus_DefaultInferredFromAssignment locks the root-cause fix:
// when the chat_status metadata key is absent, the status is inferred from
// assignment (unassigned → pending, assigned → open) instead of defaulting to
// open. Defaulting an unassigned contact to open bypassed the claim gate.
func TestEffectiveStatus_DefaultInferredFromAssignment(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name     string
		metadata JSONB
		assigned *uuid.UUID
		want     ChatStatus
	}{
		{"nil metadata, unassigned → pending", nil, nil, ChatStatusPending},
		{"nil metadata, assigned → open", nil, &uid, ChatStatusOpen},
		{"no key, unassigned → pending", JSONB{"foo": "bar"}, nil, ChatStatusPending},
		{"no key, assigned → open", JSONB{"foo": "bar"}, &uid, ChatStatusOpen},
		{"explicit pending wins", JSONB{"chat_status": "pending"}, &uid, ChatStatusPending},
		{"explicit closed wins", JSONB{"chat_status": "closed"}, nil, ChatStatusClosed},
		{"explicit open wins", JSONB{"chat_status": "open"}, nil, ChatStatusOpen},
		{"unknown value → open", JSONB{"chat_status": "weird"}, &uid, ChatStatusOpen},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Contact{Metadata: tt.metadata, AssignedUserID: tt.assigned}
			if got := c.EffectiveStatus(); got != tt.want {
				t.Errorf("EffectiveStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}
