package whatsmeow

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/zerodha/logf"
)

func TestNewConnectionManager(t *testing.T) {
	logger := logf.New(logf.Opts{})
	cfg := &config.WhatsmeowConfig{
		MaxInstancesPerOrg: 5,
	}

	cm := NewConnectionManager(nil, nil, logger, cfg, nil, "./uploads")
	if cm == nil {
		t.Fatal("Expected ConnectionManager to be created")
	}

	if cm.cfg.MaxInstancesPerOrg != 5 {
		t.Errorf("Expected MaxInstancesPerOrg to be 5, got %d", cm.cfg.MaxInstancesPerOrg)
	}

	if cm.pool == nil {
		t.Fatal("Expected connection pool to be initialized")
	}
}

func TestIsGroupJID(t *testing.T) {
	tests := []struct {
		jid      string
		expected bool
	}{
		{"12345@s.whatsapp.net", false},
		{"120363123456789012@g.us", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		if result := isGroupJID(tt.jid); result != tt.expected {
			t.Errorf("isGroupJID(%q) = %v; want %v", tt.jid, result, tt.expected)
		}
	}
}
