package whatsmeow

import (
	"context"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
)

func TestExtractMessageContentWithMedia_ReproGroupIssue(t *testing.T) {
	cm := newTestConnectionManager(t)
	ctx := context.Background()

	t.Run("sender key distribution message returns unsupported", func(t *testing.T) {
		msg := &waE2E.Message{
			SenderKeyDistributionMessage: &waE2E.SenderKeyDistributionMessage{
				GroupID:                             nil,
				AxolotlSenderKeyDistributionMessage: []byte("fake-key-data"),
			},
		}

		msgType, content, _, _, _ := cm.extractMessageContentWithMedia(ctx, nil, msg)

		// We expect this to be identified as Ignore, fixing the bug.
		if msgType != models.MessageTypeIgnore {
			t.Fatalf("expected ignore type for sender key distribution message, got %q", msgType)
		}
		if content != "" {
			t.Fatalf("expected empty content, got %q", content)
		}
	})
}
