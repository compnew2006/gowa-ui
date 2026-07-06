package whatsmeow

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/zerodha/logf"
	"go.mau.fi/whatsmeow"
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

	_, err := adapter.getClient(context.Background(), "invalid-uuid")
	if err == nil {
		t.Error("Expected error for invalid UUID, got nil")
	}
	if err != nil && !containsString(err.Error(), "invalid instance ID") {
		t.Errorf("Expected 'invalid instance ID' error, got: %v", err)
	}
}

func TestWhatsmeowAdapterGetClientNilManager(t *testing.T) {
	adapter := &WhatsmeowAdapter{logger: newTestLogger()}

	_, err := adapter.getClient(context.Background(), uuid.NewString())
	if err == nil {
		t.Fatal("Expected error for missing manager, got nil")
	}
	if !containsString(err.Error(), "instance not connected") {
		t.Fatalf("Expected not connected error, got: %v", err)
	}
}

func TestWhatsmeowAdapterGetClientAttemptsReconnectWhenRuntimeMissing(t *testing.T) {
	instanceID := uuid.New()
	reconnectCalls := 0
	adapter := &WhatsmeowAdapter{
		logger: newTestLogger(),
		getRuntimeClient: func(id uuid.UUID) *whatsmeow.Client {
			if id != instanceID {
				t.Fatalf("unexpected instance ID: %s", id)
			}
			return nil
		},
		connectRuntime: func(ctx context.Context, id uuid.UUID) error {
			reconnectCalls++
			if id != instanceID {
				t.Fatalf("unexpected instance ID: %s", id)
			}
			return nil
		},
	}

	_, err := adapter.getClient(context.Background(), instanceID.String())
	if err == nil {
		t.Fatal("Expected error when reconnect does not hydrate a client")
	}
	if reconnectCalls != 1 {
		t.Fatalf("Expected one reconnect attempt, got %d", reconnectCalls)
	}
	if !containsString(err.Error(), "instance not connected") {
		t.Fatalf("Expected not connected error, got: %v", err)
	}
}

func TestWhatsmeowAdapterGetClientAttemptsReconnectWhenClientDisconnected(t *testing.T) {
	instanceID := uuid.New()
	reconnectCalls := 0
	adapter := &WhatsmeowAdapter{
		logger: newTestLogger(),
		getRuntimeClient: func(id uuid.UUID) *whatsmeow.Client {
			if id != instanceID {
				t.Fatalf("unexpected instance ID: %s", id)
			}
			return &whatsmeow.Client{}
		},
		connectRuntime: func(ctx context.Context, id uuid.UUID) error {
			reconnectCalls++
			if id != instanceID {
				t.Fatalf("unexpected instance ID: %s", id)
			}
			return nil
		},
	}

	_, err := adapter.getClient(context.Background(), instanceID.String())
	if err == nil {
		t.Fatal("Expected error when reconnect leaves client disconnected")
	}
	if reconnectCalls != 1 {
		t.Fatalf("Expected one reconnect attempt, got %d", reconnectCalls)
	}
	if !containsString(err.Error(), "instance disconnected") {
		t.Fatalf("Expected disconnected error, got: %v", err)
	}
}

func newTestLogger() logf.Logger {
	return logf.New(logf.Opts{
		Writer: io.Discard,
		Level:  logf.DebugLevel,
	})
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
