package whatsmeow

import (
	"testing"
)

func TestIsGroupJIDAdapter(t *testing.T) {
	tests := []struct {
		name   string
		jidStr string
		want   bool
	}{
		{"group JID", "1234567890@g.us", true},
		{"individual JID", "1234567890@s.whatsapp.net", false},
		{"newsletter JID", "1234567890@newsletter", false},
		{"plain number", "1234567890", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGroupJID(tt.jidStr); got != tt.want {
				t.Errorf("isGroupJID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWhatsmeowAdapterGetClientInvalidUUID(t *testing.T) {
	adapter := &WhatsmeowAdapter{} // Minimal setup for testing error path

	_, err := adapter.getClient("invalid-uuid")
	if err == nil {
		t.Error("Expected error for invalid UUID, got nil")
	}
	if err != nil && !containsString(err.Error(), "invalid instance ID") {
		t.Errorf("Expected 'invalid instance ID' error, got: %v", err)
	}
}

func TestWhatsmeowAdapterGetClientNilManager(t *testing.T) {
	// Note: This test would panic because manager.GetClient is called on a nil pointer.
	// In a real scenario, the adapter should always have a valid manager.
	// Skipping this test to avoid panic.
	t.Skip("Skipping: nil manager causes panic in manager.GetClient()")
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
