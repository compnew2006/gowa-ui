package whatsmeow

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
	waClient "go.mau.fi/whatsmeow"
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

type stubInboundMediaDownloader struct {
	attempts int
	download func(attempt int) ([]byte, error)
}

func (s *stubInboundMediaDownloader) Download(_ context.Context, _ waClient.DownloadableMessage) ([]byte, error) {
	s.attempts++
	if s.download == nil {
		return nil, fmt.Errorf("download stub is not configured")
	}
	return s.download(s.attempts)
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

	t.Run("poll question uses poll type", func(t *testing.T) {
		msg := &waE2E.Message{
			PollCreationMessage: &waE2E.PollCreationMessage{
				Name: proto.String("Lunch option?"),
			},
		}

		msgType, content, _, _, _ := cm.extractMessageContentWithMedia(ctx, nil, msg)
		if msgType != models.MessageTypePoll {
			t.Fatalf("expected poll type for poll preview, got %q", msgType)
		}
		if content != "Lunch option?" {
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

	t.Run("protocol status mention renders readable placeholder", func(t *testing.T) {
		msg := &waE2E.Message{
			ProtocolMessage: &waE2E.ProtocolMessage{
				Type: waE2E.ProtocolMessage_STATUS_MENTION_MESSAGE.Enum(),
			},
		}

		msgType, content, mediaURL, mimeType, filename := cm.extractMessageContentWithMedia(ctx, nil, msg)
		if msgType != models.MessageTypeText {
			t.Fatalf("expected text type, got %q", msgType)
		}
		if content != statusMentionCaption {
			t.Fatalf("expected status mention placeholder, got %q", content)
		}
		if mediaURL != "" || mimeType != "" || filename != "" {
			t.Fatalf("expected no media metadata for protocol status mention, got mediaURL=%q mimeType=%q filename=%q", mediaURL, mimeType, filename)
		}
	})

	t.Run("protocol control messages are ignored", func(t *testing.T) {
		msg := &waE2E.Message{
			ProtocolMessage: &waE2E.ProtocolMessage{
				Type: waE2E.ProtocolMessage_APP_STATE_SYNC_KEY_REQUEST.Enum(),
			},
		}

		msgType, content, mediaURL, mimeType, filename := cm.extractMessageContentWithMedia(ctx, nil, msg)
		if msgType != models.MessageTypeIgnore {
			t.Fatalf("expected ignore type for protocol control message, got %q", msgType)
		}
		if content != "" || mediaURL != "" || mimeType != "" || filename != "" {
			t.Fatalf("expected empty payload for ignored control protocol message, got content=%q mediaURL=%q mimeType=%q filename=%q", content, mediaURL, mimeType, filename)
		}
	})

	t.Run("album wrapper message is ignored", func(t *testing.T) {
		msg := &waE2E.Message{
			AlbumMessage: &waE2E.AlbumMessage{
				ExpectedImageCount: proto.Uint32(2),
			},
		}

		msgType, content, mediaURL, mimeType, filename := cm.extractMessageContentWithMedia(ctx, nil, msg)
		if msgType != models.MessageTypeIgnore {
			t.Fatalf("expected ignore type for album wrapper message, got %q", msgType)
		}
		if content != "" || mediaURL != "" || mimeType != "" || filename != "" {
			t.Fatalf("expected empty payload for ignored album wrapper message, got content=%q mediaURL=%q mimeType=%q filename=%q", content, mediaURL, mimeType, filename)
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
		data     []byte
		prefix   string
		ext      string
	}{
		{models.MessageTypeImage, "image/png", "photo.png", []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, "images/", ".png"},
		{models.MessageTypeSticker, "image/webp", "sticker.webp", []byte("RIFF\x24\x00\x00\x00WEBPVP8 "), "stickers/", ".webp"},
		{models.MessageTypeVideo, "video/mp4", "clip.mp4", []byte("\x00\x00\x00\x18ftypmp42"), "videos/", ".mp4"},
		{models.MessageTypeAudio, "audio/ogg", "voice.ogg", []byte("OggS\x00\x02"), "audio/", ".ogg"},
		{models.MessageTypeDocument, "application/pdf", "doc.pdf", []byte("%PDF-1.7"), "documents/", ".pdf"},
	}

	for _, tc := range tests {
		t.Run(string(tc.msgType), func(t *testing.T) {
			relPath, err := cm.persistInboundMedia(tc.data, tc.msgType, tc.mimeType, tc.nameHint)
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

func TestSanitizeIncomingFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "strips traversal segments", input: "../../../etc/passwd", expected: "passwd"},
		{name: "handles windows separators", input: `..\..\secret\payload.pdf`, expected: "payload.pdf"},
		{name: "drops invalid values", input: "   ", expected: ""},
		{name: "drops dot segments", input: "..", expected: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeIncomingFilename(tc.input)
			if got != tc.expected {
				t.Fatalf("sanitizeIncomingFilename(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestResolveInboundMediaMimeType_PrefersDetectedContentType(t *testing.T) {
	htmlPayload := []byte("<!doctype html><html><body>owned</body></html>")
	mimeType := resolveInboundMediaMimeType(htmlPayload, "image/png", "invoice.png")
	if mimeType != "text/html" {
		t.Fatalf("expected content-sniffed mime type to win, got %q", mimeType)
	}
}

func TestNormalizeMediaFileExtension_RejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "accepts valid extension", input: ".pdf", expected: ".pdf"},
		{name: "normalizes missing dot", input: "PNG", expected: ".png"},
		{name: "rejects path separators", input: ".txt/../../etc/passwd", expected: ".bin"},
		{name: "rejects special characters", input: ".tar.gz", expected: ".bin"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeMediaFileExtension(tc.input)
			if got != tc.expected {
				t.Fatalf("normalizeMediaFileExtension(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestShouldRetryInboundMediaDownload(t *testing.T) {
	t.Run("nil error is not retryable", func(t *testing.T) {
		if shouldRetryInboundMediaDownload(nil) {
			t.Fatalf("expected nil error to be non-retryable")
		}
	})

	t.Run("context canceled is not retryable", func(t *testing.T) {
		if shouldRetryInboundMediaDownload(context.Canceled) {
			t.Fatalf("expected context.Canceled to be non-retryable")
		}
	})

	t.Run("deadline exceeded is not retryable", func(t *testing.T) {
		if shouldRetryInboundMediaDownload(fmt.Errorf("wrapped: %w", context.DeadlineExceeded)) {
			t.Fatalf("expected context.DeadlineExceeded to be non-retryable")
		}
	})

	t.Run("other errors are retryable", func(t *testing.T) {
		if !shouldRetryInboundMediaDownload(errors.New("hash of media ciphertext doesn't match")) {
			t.Fatalf("expected hash mismatch to be retryable")
		}
	})
}

func TestInboundMediaDownloadBackoff(t *testing.T) {
	base := 500 * time.Millisecond
	max := 2 * time.Second

	tests := []struct {
		name          string
		failedAttempt int
		want          time.Duration
	}{
		{name: "attempt 1", failedAttempt: 1, want: 500 * time.Millisecond},
		{name: "attempt 2", failedAttempt: 2, want: 1 * time.Second},
		{name: "attempt 3 capped", failedAttempt: 3, want: 2 * time.Second},
		{name: "attempt 4 still capped", failedAttempt: 4, want: 2 * time.Second},
		{name: "attempt below one clamps", failedAttempt: 0, want: 500 * time.Millisecond},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := inboundMediaDownloadBackoff(tc.failedAttempt, base, max)
			if got != tc.want {
				t.Fatalf("backoff mismatch: got %v want %v", got, tc.want)
			}
		})
	}
}

func TestInboundMediaDownloadRetrySettings(t *testing.T) {
	t.Run("uses defaults without config", func(t *testing.T) {
		cm := newTestConnectionManager(t)
		attempts, baseBackoff, maxBackoff := cm.inboundMediaDownloadRetrySettings()
		if attempts != defaultInboundMediaDownloadMaxAttempts {
			t.Fatalf("attempts mismatch: got %d want %d", attempts, defaultInboundMediaDownloadMaxAttempts)
		}
		if baseBackoff != defaultInboundMediaDownloadBaseBackoff {
			t.Fatalf("base backoff mismatch: got %v want %v", baseBackoff, defaultInboundMediaDownloadBaseBackoff)
		}
		if maxBackoff != defaultInboundMediaDownloadMaxBackoff {
			t.Fatalf("max backoff mismatch: got %v want %v", maxBackoff, defaultInboundMediaDownloadMaxBackoff)
		}
	})

	t.Run("uses explicit config", func(t *testing.T) {
		cm := newTestConnectionManager(t)
		cm.cfg = &config.WhatsmeowConfig{
			InboundMediaRetryCount:      5,
			InboundMediaRetryDelayMs:    250,
			InboundMediaRetryMaxDelayMs: 3000,
		}
		attempts, baseBackoff, maxBackoff := cm.inboundMediaDownloadRetrySettings()
		if attempts != 5 {
			t.Fatalf("attempts mismatch: got %d want 5", attempts)
		}
		if baseBackoff != 250*time.Millisecond {
			t.Fatalf("base backoff mismatch: got %v want %v", baseBackoff, 250*time.Millisecond)
		}
		if maxBackoff != 3*time.Second {
			t.Fatalf("max backoff mismatch: got %v want %v", maxBackoff, 3*time.Second)
		}
	})
}

func TestDownloadAndPersistIncomingMedia_BuildsArtifactAfterFinalInlineFailure(t *testing.T) {
	cm := newTestConnectionManager(t)
	cm.cfg = &config.WhatsmeowConfig{
		InboundMediaRetryCount:      2,
		InboundMediaRetryDelayMs:    0,
		InboundMediaRetryMaxDelayMs: 0,
	}

	media := &waE2E.DocumentMessage{
		FileName: proto.String("invoice.pdf"),
		Mimetype: proto.String("application/pdf"),
	}
	downloader := &stubInboundMediaDownloader{
		download: func(attempt int) ([]byte, error) {
			return nil, fmt.Errorf("download failed attempt %d", attempt)
		},
	}

	mediaURL, artifact := cm.downloadAndPersistIncomingMedia(
		context.Background(),
		downloader,
		media,
		models.MessageTypeDocument,
		"application/pdf",
		"invoice.pdf",
	)

	require.Equal(t, "", mediaURL)
	require.NotNil(t, artifact)
	assert.Equal(t, "document", artifact.MediaKind)
	assert.Equal(t, "application/pdf", artifact.MimeType)
	assert.Equal(t, "invoice.pdf", artifact.FallbackFilename)
	assert.Contains(t, artifact.LastError, "attempt 2")
	assert.Equal(t, 2, downloader.attempts)

	rawPayload, err := base64.StdEncoding.DecodeString(artifact.MediaPayloadBase64)
	require.NoError(t, err)
	var decoded waE2E.DocumentMessage
	require.NoError(t, proto.Unmarshal(rawPayload, &decoded))
	assert.Equal(t, "invoice.pdf", decoded.GetFileName())
}

func TestDownloadAndPersistIncomingMedia_NoArtifactOnInlineSuccess(t *testing.T) {
	cm := newTestConnectionManager(t)
	cm.cfg = &config.WhatsmeowConfig{
		InboundMediaRetryCount:      2,
		InboundMediaRetryDelayMs:    0,
		InboundMediaRetryMaxDelayMs: 0,
	}

	downloader := &stubInboundMediaDownloader{
		download: func(attempt int) ([]byte, error) {
			return []byte("ok"), nil
		},
	}

	mediaURL, artifact := cm.downloadAndPersistIncomingMedia(
		context.Background(),
		downloader,
		&waE2E.DocumentMessage{
			FileName: proto.String("report.pdf"),
			Mimetype: proto.String("application/pdf"),
		},
		models.MessageTypeDocument,
		"application/pdf",
		"report.pdf",
	)

	require.NotEmpty(t, mediaURL)
	require.Nil(t, artifact)
	assert.Equal(t, 1, downloader.attempts)

	absPath := filepath.Join(cm.mediaStoragePath, filepath.FromSlash(mediaURL))
	data, err := os.ReadFile(absPath)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(data))
}

func TestInboundMediaAsyncRetrySettings(t *testing.T) {
	t.Run("uses defaults without config", func(t *testing.T) {
		cm := newTestConnectionManager(t)
		attempts, baseBackoff, maxBackoff := cm.inboundMediaAsyncRetrySettings()
		assert.Equal(t, defaultInboundMediaAsyncMaxAttempts, attempts)
		assert.Equal(t, defaultInboundMediaAsyncBaseBackoff, baseBackoff)
		assert.Equal(t, defaultInboundMediaAsyncMaxBackoff, maxBackoff)
	})

	t.Run("uses configured values", func(t *testing.T) {
		cm := newTestConnectionManager(t)
		cm.cfg = &config.WhatsmeowConfig{
			InboundMediaAsyncRetryCount:      7,
			InboundMediaAsyncRetryDelayMs:    2500,
			InboundMediaAsyncRetryMaxDelayMs: 12000,
		}

		attempts, baseBackoff, maxBackoff := cm.inboundMediaAsyncRetrySettings()
		assert.Equal(t, 7, attempts)
		assert.Equal(t, 2500*time.Millisecond, baseBackoff)
		assert.Equal(t, 12*time.Second, maxBackoff)
	})
}
