package whatsmeow

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/zerodha/logf"
	waCommon "go.mau.fi/whatsmeow/proto/waCommon"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func newTestConnectionManager(t *testing.T) *ConnectionManager {
	t.Helper()
	return &ConnectionManager{
		logger:           logf.New(logf.Opts{}),
		mediaStoragePath: t.TempDir(),
	}
}

func TestExtractMessageContentWithMedia_FileTypes_NoClient(t *testing.T) {
	cm := newTestConnectionManager(t)
	ctx := context.Background()

	tests := []struct {
		name         string
		msg          *waE2E.Message
		wantType     models.MessageType
		wantContent  string
		wantMime     string
		wantFilename string
	}{
		{
			name: "image",
			msg: &waE2E.Message{
				ImageMessage: &waE2E.ImageMessage{
					Caption:  proto.String("image caption"),
					Mimetype: proto.String("image/png"),
				},
			},
			wantType:     models.MessageTypeImage,
			wantContent:  "image caption",
			wantMime:     "image/png",
			wantFilename: "image.png",
		},
		{
			name: "video",
			msg: &waE2E.Message{
				VideoMessage: &waE2E.VideoMessage{
					Caption:  proto.String("video caption"),
					Mimetype: proto.String("video/mp4"),
				},
			},
			wantType:     models.MessageTypeVideo,
			wantContent:  "video caption",
			wantMime:     "video/mp4",
			wantFilename: "video.mp4",
		},
		{
			name: "audio",
			msg: &waE2E.Message{
				AudioMessage: &waE2E.AudioMessage{
					Mimetype: proto.String("audio/ogg; codecs=opus"),
				},
			},
			wantType:     models.MessageTypeAudio,
			wantContent:  "",
			wantMime:     "audio/ogg",
			wantFilename: "audio.ogg",
		},
		{
			name: "document",
			msg: &waE2E.Message{
				DocumentMessage: &waE2E.DocumentMessage{
					FileName: proto.String("report.xlsx"),
				},
			},
			wantType:     models.MessageTypeDocument,
			wantContent:  "",
			wantMime:     "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			wantFilename: "report.xlsx",
		},
		{
			name: "sticker",
			msg: &waE2E.Message{
				StickerMessage: &waE2E.StickerMessage{
					Mimetype: proto.String("image/webp"),
				},
			},
			wantType:     models.MessageTypeSticker,
			wantContent:  "",
			wantMime:     "image/webp",
			wantFilename: "sticker.webp",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgType, content, mediaURL, mimeType, filename := cm.extractMessageContentWithMedia(ctx, nil, tc.msg)
			if msgType != tc.wantType {
				t.Fatalf("type mismatch: got %q want %q", msgType, tc.wantType)
			}
			if content != tc.wantContent {
				t.Fatalf("content mismatch: got %q want %q", content, tc.wantContent)
			}
			if mimeType != tc.wantMime {
				t.Fatalf("mime mismatch: got %q want %q", mimeType, tc.wantMime)
			}
			if filename != tc.wantFilename {
				t.Fatalf("filename mismatch: got %q want %q", filename, tc.wantFilename)
			}
			if mediaURL != "" {
				t.Fatalf("expected empty media url when client=nil, got %q", mediaURL)
			}
		})
	}
}

func TestExtractMessageContentWithMedia_TextualVariants(t *testing.T) {
	cm := newTestConnectionManager(t)
	ctx := context.Background()

	t.Run("location payload", func(t *testing.T) {
		msg := &waE2E.Message{
			LocationMessage: &waE2E.LocationMessage{
				DegreesLatitude:  proto.Float64(37.4220),
				DegreesLongitude: proto.Float64(-122.0841),
				Name:             proto.String("Googleplex"),
				Address:          proto.String("Mountain View"),
			},
		}

		msgType, content, mediaURL, mimeType, filename := cm.extractMessageContentWithMedia(ctx, nil, msg)
		if msgType != models.MessageTypeLocation {
			t.Fatalf("expected location type, got %q", msgType)
		}
		if mediaURL != "" || mimeType != "" || filename != "" {
			t.Fatalf("expected no media metadata for location, got mediaURL=%q mimeType=%q filename=%q", mediaURL, mimeType, filename)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(content), &payload); err != nil {
			t.Fatalf("expected valid JSON location payload, got %q: %v", content, err)
		}
		if payload["name"] != "Googleplex" {
			t.Fatalf("expected location name Googleplex, got %#v", payload["name"])
		}
	})

	t.Run("single contact payload", func(t *testing.T) {
		msg := &waE2E.Message{
			ContactMessage: &waE2E.ContactMessage{
				DisplayName: proto.String("Alice"),
				Vcard:       proto.String("BEGIN:VCARD\nTEL;TYPE=CELL:+15551234567\nEND:VCARD"),
			},
		}

		msgType, content, _, _, _ := cm.extractMessageContentWithMedia(ctx, nil, msg)
		if msgType != models.MessageType("contacts") {
			t.Fatalf("expected contacts type, got %q", msgType)
		}
		var payload []map[string]any
		if err := json.Unmarshal([]byte(content), &payload); err != nil {
			t.Fatalf("expected valid JSON contacts payload, got %q: %v", content, err)
		}
		if len(payload) != 1 || payload[0]["name"] != "Alice" {
			t.Fatalf("unexpected contacts payload: %#v", payload)
		}
	})

	t.Run("interactive list response", func(t *testing.T) {
		msg := &waE2E.Message{
			ListResponseMessage: &waE2E.ListResponseMessage{
				Title: proto.String("Support"),
				SingleSelectReply: &waE2E.ListResponseMessage_SingleSelectReply{
					SelectedRowID: proto.String("billing"),
				},
			},
		}

		msgType, content, _, _, _ := cm.extractMessageContentWithMedia(ctx, nil, msg)
		if msgType != models.MessageTypeInteractive {
			t.Fatalf("expected interactive type, got %q", msgType)
		}
		if content != "Support" {
			t.Fatalf("expected list title to be used as content, got %q", content)
		}
	})

	t.Run("poll question as text", func(t *testing.T) {
		msg := &waE2E.Message{
			PollCreationMessage: &waE2E.PollCreationMessage{
				Name: proto.String("Lunch option?"),
			},
		}

		msgType, content, _, _, _ := cm.extractMessageContentWithMedia(ctx, nil, msg)
		if msgType != models.MessageTypeText {
			t.Fatalf("expected text type for poll preview, got %q", msgType)
		}
		if content != "[Poll] Lunch option?" {
			t.Fatalf("unexpected poll content: %q", content)
		}
	})

	t.Run("mentioned phone number keeps readable phone in text", func(t *testing.T) {
		msg := &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: proto.String("Hi @52488795361474, please review"),
				ContextInfo: &waE2E.ContextInfo{
					MentionedJID: []string{"966561853319@s.whatsapp.net"},
				},
			},
		}

		msgType, content, _, _, _ := cm.extractMessageContentWithMedia(ctx, nil, msg)
		if msgType != models.MessageTypeText {
			t.Fatalf("expected text type for mention, got %q", msgType)
		}
		if content != "Hi @966561853319, please review" {
			t.Fatalf("unexpected mention-normalized text: %q", content)
		}
	})

	t.Run("protocol message edit unwraps edited body", func(t *testing.T) {
		msg := &waE2E.Message{
			ProtocolMessage: &waE2E.ProtocolMessage{
				Type: waE2E.ProtocolMessage_MESSAGE_EDIT.Enum(),
				EditedMessage: &waE2E.Message{
					ExtendedTextMessage: &waE2E.ExtendedTextMessage{
						Text: proto.String("Edited text"),
					},
				},
			},
		}

		msgType, content, _, _, _ := cm.extractMessageContentWithMedia(ctx, nil, msg)
		if msgType != models.MessageTypeText {
			t.Fatalf("expected text type for edited message, got %q", msgType)
		}
		if content != "Edited text" {
			t.Fatalf("expected edited body, got %q", content)
		}
	})
}

func TestExtractMessageContentWithMedia_UnwrapsEphemeral(t *testing.T) {
	cm := newTestConnectionManager(t)
	ctx := context.Background()

	msg := &waE2E.Message{
		EphemeralMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				DocumentWithCaptionMessage: &waE2E.FutureProofMessage{
					Message: &waE2E.Message{
						DocumentMessage: &waE2E.DocumentMessage{
							FileName: proto.String("invoice.pdf"),
							Caption:  proto.String("invoice"),
							Mimetype: proto.String("application/pdf"),
						},
					},
				},
			},
		},
	}

	msgType, content, _, mimeType, filename := cm.extractMessageContentWithMedia(ctx, nil, msg)
	if msgType != models.MessageTypeDocument {
		t.Fatalf("expected document, got %q", msgType)
	}
	if content != "invoice" {
		t.Fatalf("expected caption invoice, got %q", content)
	}
	if mimeType != "application/pdf" {
		t.Fatalf("expected pdf mime, got %q", mimeType)
	}
	if filename != "invoice.pdf" {
		t.Fatalf("expected invoice.pdf filename, got %q", filename)
	}
}

func TestExtractMessageContentWithMedia_UnwrapsCommentMessage(t *testing.T) {
	cm := newTestConnectionManager(t)
	ctx := context.Background()

	msg := &waE2E.Message{
		CommentMessage: &waE2E.CommentMessage{
			Message: &waE2E.Message{
				ExtendedTextMessage: &waE2E.ExtendedTextMessage{
					Text: proto.String("Mechanical"),
				},
			},
		},
	}

	msgType, content, mediaURL, mimeType, filename := cm.extractMessageContentWithMedia(ctx, nil, msg)
	if msgType != models.MessageTypeText {
		t.Fatalf("expected text type, got %q", msgType)
	}
	if content != "Mechanical" {
		t.Fatalf("expected Mechanical, got %q", content)
	}
	if mediaURL != "" || mimeType != "" || filename != "" {
		t.Fatalf("expected no media metadata for comment message, got mediaURL=%q mimeType=%q filename=%q", mediaURL, mimeType, filename)
	}
}

func TestExtractMessageContentWithMedia_UnwrapsDeviceSentCommentMessage(t *testing.T) {
	cm := newTestConnectionManager(t)
	ctx := context.Background()

	msg := &waE2E.Message{
		DeviceSentMessage: &waE2E.DeviceSentMessage{
			Message: &waE2E.Message{
				CommentMessage: &waE2E.CommentMessage{
					Message: &waE2E.Message{
						ExtendedTextMessage: &waE2E.ExtendedTextMessage{
							Text: proto.String("Mechanical wrapped"),
						},
					},
				},
			},
		},
	}

	msgType, content, mediaURL, mimeType, filename := cm.extractMessageContentWithMedia(ctx, nil, msg)
	if msgType != models.MessageTypeText {
		t.Fatalf("expected text type, got %q", msgType)
	}
	if content != "Mechanical wrapped" {
		t.Fatalf("expected Mechanical wrapped, got %q", content)
	}
	if mediaURL != "" || mimeType != "" || filename != "" {
		t.Fatalf("expected no media metadata for wrapped comment message, got mediaURL=%q mimeType=%q filename=%q", mediaURL, mimeType, filename)
	}
}

func TestExtractMessageContentWithMedia_ProtocolRevoke(t *testing.T) {
	cm := newTestConnectionManager(t)
	ctx := context.Background()

	msg := &waE2E.Message{
		ProtocolMessage: &waE2E.ProtocolMessage{
			Type: waE2E.ProtocolMessage_REVOKE.Enum(),
		},
	}

	msgType, content, mediaURL, mimeType, filename := cm.extractMessageContentWithMedia(ctx, nil, msg)
	if msgType != models.MessageTypeText {
		t.Fatalf("expected text type, got %q", msgType)
	}
	if content != "(This message was deleted)" {
		t.Fatalf("expected deleted caption, got %q", content)
	}
	if mediaURL != "" || mimeType != "" || filename != "" {
		t.Fatalf("expected no media metadata for revoke message, got mediaURL=%q mimeType=%q filename=%q", mediaURL, mimeType, filename)
	}
}

func TestExtractMessageContentWithMedia_ProtocolRevoke_UnwrapsEphemeral(t *testing.T) {
	cm := newTestConnectionManager(t)
	ctx := context.Background()

	msg := &waE2E.Message{
		EphemeralMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				ProtocolMessage: &waE2E.ProtocolMessage{
					Type: waE2E.ProtocolMessage_REVOKE.Enum(),
				},
			},
		},
	}

	msgType, content, mediaURL, mimeType, filename := cm.extractMessageContentWithMedia(ctx, nil, msg)
	if msgType != models.MessageTypeText {
		t.Fatalf("expected text type, got %q", msgType)
	}
	if content != "(This message was deleted)" {
		t.Fatalf("expected deleted caption, got %q", content)
	}
	if mediaURL != "" || mimeType != "" || filename != "" {
		t.Fatalf("expected no media metadata for revoke message, got mediaURL=%q mimeType=%q filename=%q", mediaURL, mimeType, filename)
	}
}

func TestExtractMessageContentWithMedia_ProtocolRevoke_UnwrapsDeviceSent(t *testing.T) {
	cm := newTestConnectionManager(t)
	ctx := context.Background()

	msg := &waE2E.Message{
		DeviceSentMessage: &waE2E.DeviceSentMessage{
			Message: &waE2E.Message{
				ProtocolMessage: &waE2E.ProtocolMessage{
					Type: waE2E.ProtocolMessage_REVOKE.Enum(),
				},
			},
		},
	}

	msgType, content, mediaURL, mimeType, filename := cm.extractMessageContentWithMedia(ctx, nil, msg)
	if msgType != models.MessageTypeText {
		t.Fatalf("expected text type, got %q", msgType)
	}
	if content != "(This message was deleted)" {
		t.Fatalf("expected deleted caption, got %q", content)
	}
	if mediaURL != "" || mimeType != "" || filename != "" {
		t.Fatalf("expected no media metadata for revoke message, got mediaURL=%q mimeType=%q filename=%q", mediaURL, mimeType, filename)
	}
}

func TestIncomingRevokeTargetID(t *testing.T) {
	t.Run("direct protocol revoke", func(t *testing.T) {
		msg := &waE2E.Message{
			ProtocolMessage: &waE2E.ProtocolMessage{
				Type: waE2E.ProtocolMessage_REVOKE.Enum(),
				Key:  &waCommon.MessageKey{ID: proto.String("wamid.direct")},
			},
		}
		targetID, isRevoke := incomingRevokeTargetID(msg)
		if !isRevoke {
			t.Fatalf("expected revoke message")
		}
		if targetID != "wamid.direct" {
			t.Fatalf("expected target wamid.direct, got %q", targetID)
		}
	})

	t.Run("wrapped device-sent protocol revoke", func(t *testing.T) {
		msg := &waE2E.Message{
			DeviceSentMessage: &waE2E.DeviceSentMessage{
				Message: &waE2E.Message{
					ProtocolMessage: &waE2E.ProtocolMessage{
						Type: waE2E.ProtocolMessage_REVOKE.Enum(),
						Key:  &waCommon.MessageKey{ID: proto.String("wamid.device")},
					},
				},
			},
		}
		targetID, isRevoke := incomingRevokeTargetID(msg)
		if !isRevoke {
			t.Fatalf("expected revoke message")
		}
		if targetID != "wamid.device" {
			t.Fatalf("expected target wamid.device, got %q", targetID)
		}
	})

	t.Run("non-revoke returns false", func(t *testing.T) {
		msg := &waE2E.Message{
			Conversation: proto.String("hello"),
		}
		targetID, isRevoke := incomingRevokeTargetID(msg)
		if isRevoke {
			t.Fatalf("expected non-revoke message")
		}
		if targetID != "" {
			t.Fatalf("expected empty target ID for non-revoke message, got %q", targetID)
		}
	})
}

func TestPersistInboundMedia_StoresByType(t *testing.T) {
	cm := newTestConnectionManager(t)

	tests := []struct {
		msgType  models.MessageType
		mimeType string
		nameHint string
		prefix   string
		ext      string
	}{
		{models.MessageTypeImage, "image/png", "photo.png", "images/", ".png"},
		{models.MessageTypeSticker, "image/webp", "sticker.webp", "stickers/", ".webp"},
		{models.MessageTypeVideo, "video/mp4", "clip.mp4", "videos/", ".mp4"},
		{models.MessageTypeAudio, "audio/ogg", "voice.ogg", "audio/", ".ogg"},
		{models.MessageTypeDocument, "application/pdf", "doc.pdf", "documents/", ".pdf"},
	}

	for _, tc := range tests {
		t.Run(string(tc.msgType), func(t *testing.T) {
			relPath, err := cm.persistInboundMedia([]byte("test-data"), tc.msgType, tc.mimeType, tc.nameHint)
			if err != nil {
				t.Fatalf("persistInboundMedia failed: %v", err)
			}

			if !strings.HasPrefix(relPath, tc.prefix) {
				t.Fatalf("path prefix mismatch: got %q want prefix %q", relPath, tc.prefix)
			}
			if filepath.Ext(relPath) != tc.ext {
				t.Fatalf("extension mismatch: got %q want %q", filepath.Ext(relPath), tc.ext)
			}

			fullPath := filepath.Join(cm.mediaStoragePath, relPath)
			if _, statErr := os.Stat(fullPath); statErr != nil {
				t.Fatalf("expected persisted file at %s: %v", fullPath, statErr)
			}
		})
	}
}

func TestPersistInboundMedia_GeneratesUniqueNames(t *testing.T) {
	cm := newTestConnectionManager(t)

	first, err := cm.persistInboundMedia([]byte("a"), models.MessageTypeDocument, "application/pdf", "doc.pdf")
	if err != nil {
		t.Fatalf("first persist failed: %v", err)
	}
	second, err := cm.persistInboundMedia([]byte("b"), models.MessageTypeDocument, "application/pdf", "doc.pdf")
	if err != nil {
		t.Fatalf("second persist failed: %v", err)
	}
	if first == second {
		t.Fatalf("expected unique filenames, got identical paths %q", first)
	}

	if _, parseErr := uuid.Parse(strings.TrimSuffix(filepath.Base(first), filepath.Ext(first))); parseErr != nil {
		t.Fatalf("expected UUID filename, got %q", filepath.Base(first))
	}
}
