package whatsmeow

import (
	"testing"

	"github.com/compnew2006/whatomate/pkg/provider"
)

func TestGroupProvider_Interface(t *testing.T) {
	var _ provider.GroupProvider = (*WhatsmeowAdapter)(nil)
}

// Test that the GroupInfo struct is correctly shaped.
func TestGroupInfo_Struct(t *testing.T) {
	info := provider.GroupInfo{
		JID:              "1203631234567-1601234567@g.us",
		Name:             "Test Group",
		ParticipantCount: 42,
	}
	if info.JID == "" {
		t.Error("expected non-empty JID")
	}
	if info.ParticipantCount != 42 {
		t.Errorf("ParticipantCount = %d, want 42", info.ParticipantCount)
	}
}
