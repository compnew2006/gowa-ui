package whatsmeow

import (
	"context"
	"testing"

    waCommon "go.mau.fi/whatsmeow/proto/waCommon"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestExtractMessageContent_Investigation_Advanced(t *testing.T) {
	cm := newTestConnectionManager(t)
	ctx := context.Background()

	tests := []struct {
		name          string
		msg           *waE2E.Message
		expectContent string
	}{
		{
			name: "ProtocolMessage (Revoke)",
			msg: &waE2E.Message{
				ProtocolMessage: &waE2E.ProtocolMessage{
					Type: waE2E.ProtocolMessage_REVOKE.Enum(),
				},
			},
		},
        {
			name: "ReactionMessage",
			msg: &waE2E.Message{
				ReactionMessage: &waE2E.ReactionMessage{
					Key: &waCommon.MessageKey{ID: proto.String("123")},
                    Text: proto.String("👍"),
				},
			},
		},
        {
            name: "PollCreationMessage",
            msg: &waE2E.Message{
                PollCreationMessage: &waE2E.PollCreationMessage{
                    Name: proto.String("Mechanical Poll"),
                    Options: []*waE2E.PollCreationMessage_Option{
                        {OptionName: proto.String("Yes")},
                    },
                },
            },
        },
        {
            name: "PollUpdateMessage",
            msg: &waE2E.Message{
                PollUpdateMessage: &waE2E.PollUpdateMessage{
                    PollCreationMessageKey: &waCommon.MessageKey{ID: proto.String("123")},
                    Vote: &waE2E.PollEncValue{EncPayload: []byte("vote")},
                },
            },
        },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgType, content, _, _, _ := cm.extractMessageContentWithMedia(ctx, nil, tt.msg)
			t.Logf("%s -> Type: %s, Content: %s", tt.name, msgType, content)
			
			if content == "[Unsupported message type]" {
				t.Logf("HIT: %s returned unsupported/unknown message type", tt.name)
            } else {
                t.Logf("PASS: %s -> %s", tt.name, content)
            }
		})
	}
}
