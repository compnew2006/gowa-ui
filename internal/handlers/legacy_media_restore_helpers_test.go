package handlers

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeLegacyMediaMime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "standard", in: "image/jpeg", want: "image/jpeg"},
		{name: "with params", in: "image/webp; charset=utf-8", want: "image/webp"},
		{name: "uppercase", in: "Image/PNG", want: "image/png"},
		{name: "with spaces", in: "  image/gif  ", want: "image/gif"},
		{name: "empty", in: "", want: ""},
		{name: "whitespace only", in: "   ", want: ""},
		{name: "semicolon only", in: ";", want: ""},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, normalizeLegacyMediaMime(tc.in))
		})
	}
}

func TestLegacyMediaSameMIMEFamily(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{name: "same image types", a: "image/jpeg", b: "image/png", want: true},
		{name: "same video types", a: "video/mp4", b: "video/webm", want: true},
		{name: "identical", a: "image/jpeg", b: "image/jpeg", want: true},
		{name: "different families", a: "image/jpeg", b: "video/mp4", want: false},
		{name: "empty a", a: "", b: "image/jpeg", want: false},
		{name: "empty b", a: "image/jpeg", b: "", want: false},
		{name: "no slash a", a: "octet", b: "image/jpeg", want: false},
		{name: "no slash b", a: "image/jpeg", b: "octet", want: false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, legacyMediaSameMIMEFamily(tc.a, tc.b))
		})
	}
}

func TestLegacyMediaMimeMatchesMessageType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		messageType models.MessageType
		mimeType    string
		want        bool
	}{
		{name: "image type matches image mime", messageType: models.MessageTypeImage, mimeType: "image/jpeg", want: true},
		{name: "image type rejects video", messageType: models.MessageTypeImage, mimeType: "video/mp4", want: false},
		{name: "sticker matches image", messageType: models.MessageTypeSticker, mimeType: "image/webp", want: true},
		{name: "video matches video", messageType: models.MessageTypeVideo, mimeType: "video/mp4", want: true},
		{name: "video rejects image", messageType: models.MessageTypeVideo, mimeType: "image/png", want: false},
		{name: "audio matches audio", messageType: models.MessageTypeAudio, mimeType: "audio/ogg", want: true},
		{name: "document allows pdf", messageType: models.MessageTypeDocument, mimeType: "application/pdf", want: true},
		{name: "document rejects image", messageType: models.MessageTypeDocument, mimeType: "image/jpeg", want: false},
		{name: "unknown allows anything", messageType: "unknown_type", mimeType: "image/jpeg", want: true},
		{name: "empty mime for image", messageType: models.MessageTypeImage, mimeType: "", want: false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, legacyMediaMimeMatchesMessageType(tc.messageType, tc.mimeType))
		})
	}
}

func TestIsValidWEBP(t *testing.T) {
	t.Parallel()

	validWEBP := []byte("RIFF\x00\x00\x00\x00WEBP")
	shortData := []byte("RIFF")
	badMagic := []byte("GIF8\x00\x00\x00\x00\x00\x00\x00\x00WEBP")

	assert.True(t, isValidWEBP(validWEBP))
	assert.False(t, isValidWEBP(shortData))
	assert.False(t, isValidWEBP(badMagic))
	assert.False(t, isValidWEBP(nil))
	assert.False(t, isValidWEBP([]byte{}))
}

func TestLegacyMediaSubdirForMime(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "images", legacyMediaSubdirForMime("image/jpeg"))
	assert.Equal(t, "images", legacyMediaSubdirForMime("image/png"))
	assert.Equal(t, "videos", legacyMediaSubdirForMime("video/mp4"))
	assert.Equal(t, "audio", legacyMediaSubdirForMime("audio/ogg"))
	assert.Equal(t, "documents", legacyMediaSubdirForMime("application/pdf"))
	assert.Equal(t, "documents", legacyMediaSubdirForMime(""))
	assert.Equal(t, "documents", legacyMediaSubdirForMime("unknown/type"))
}

func TestLegacyMediaMessageTypeFromMIME(t *testing.T) {
	t.Parallel()

	assert.Equal(t, models.MessageTypeImage, legacyMediaMessageTypeFromMIME("image/jpeg"))
	assert.Equal(t, models.MessageTypeImage, legacyMediaMessageTypeFromMIME("image/png"))
	assert.Equal(t, models.MessageTypeVideo, legacyMediaMessageTypeFromMIME("video/mp4"))
	assert.Equal(t, models.MessageTypeAudio, legacyMediaMessageTypeFromMIME("audio/ogg"))
	assert.Equal(t, models.MessageTypeDocument, legacyMediaMessageTypeFromMIME("application/pdf"))
	assert.Equal(t, models.MessageTypeDocument, legacyMediaMessageTypeFromMIME("unknown"))
}

func TestMinInt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 3, minInt(3, 5))
	assert.Equal(t, 3, minInt(5, 3))
	assert.Equal(t, 0, minInt(0, 10))
	assert.Equal(t, -5, minInt(-5, -3))
}

func TestCoalesceLegacyMediaValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "first", coalesceLegacyMediaValue("first", "second"))
	assert.Equal(t, "first", coalesceLegacyMediaValue("  first  ", "second"))
	assert.Equal(t, "second", coalesceLegacyMediaValue("", "second"))
	assert.Equal(t, "second", coalesceLegacyMediaValue("  ", "second"))
	assert.Equal(t, "only", coalesceLegacyMediaValue("only"))
	assert.Equal(t, "", coalesceLegacyMediaValue())
	assert.Equal(t, "", coalesceLegacyMediaValue("", "  ", "  "))
}

func TestCloneMessageMetadata(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, cloneMessageMetadata(nil))
	})

	t.Run("returns independent copy", func(t *testing.T) {
		t.Parallel()
		orig := models.JSONB{"key": "value", "nested": map[string]any{"a": 1}}
		cloned := cloneMessageMetadata(orig)
		assert.Equal(t, orig, cloned)
		cloned["key"] = "changed"
		assert.Equal(t, "value", orig["key"])
	})
}

func TestLegacyMediaFilenameHint(t *testing.T) {
	t.Parallel()

	t.Run("nil message", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", legacyMediaFilenameHint(nil))
	})

	t.Run("uses media filename", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{MediaFilename: "photo.jpg"}
		assert.Equal(t, "photo.jpg", legacyMediaFilenameHint(msg))
	})

	t.Run("falls back to media url", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{MediaURL: "https://example.com/path/photo.png"}
		assert.Equal(t, "photo.png", legacyMediaFilenameHint(msg))
	})

	t.Run("extracts basename", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{MediaFilename: "/some/deep/path/file.webp"}
		assert.Equal(t, "file.webp", legacyMediaFilenameHint(msg))
	})

	t.Run("empty fields", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{MediaFilename: "  ", MediaURL: "  "}
		assert.Equal(t, "", legacyMediaFilenameHint(msg))
	})
}

func TestLegacyMediaRestoreLockKey(t *testing.T) {
	t.Parallel()

	t.Run("deterministic for same UUID", func(t *testing.T) {
		t.Parallel()
		id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		key1 := legacyMediaRestoreLockKey(id)
		key2 := legacyMediaRestoreLockKey(id)
		assert.Equal(t, key1, key2)
	})

	t.Run("different UUIDs produce different keys", func(t *testing.T) {
		t.Parallel()
		key1 := legacyMediaRestoreLockKey(uuid.New())
		key2 := legacyMediaRestoreLockKey(uuid.New())
		assert.NotEqual(t, key1, key2)
	})
}

func TestInspectLegacyMediaRecovery(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)

	t.Run("nil message", func(t *testing.T) {
		t.Parallel()
		_, ok, reason := inspectLegacyMediaRecovery(nil, now)
		assert.False(t, ok)
		assert.Equal(t, "missing_metadata", reason)
	})

	t.Run("nil metadata", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{}
		_, ok, reason := inspectLegacyMediaRecovery(msg, now)
		assert.False(t, ok)
		assert.Equal(t, "missing_metadata", reason)
	})

	t.Run("missing provider key", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{Metadata: models.JSONB{"other": "value"}}
		_, ok, reason := inspectLegacyMediaRecovery(msg, now)
		assert.False(t, ok)
		assert.Equal(t, "missing_metadata", reason)
	})

	t.Run("wrong provider", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{Metadata: models.JSONB{
			legacyMediaRecoveryProviderKey: "whatsmeow",
			legacyMediaRecoveryMediaIDKey:  "abc123",
		}}
		_, ok, reason := inspectLegacyMediaRecovery(msg, now)
		assert.False(t, ok)
		assert.Equal(t, "missing_metadata", reason)
	})

	t.Run("empty media id", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{Metadata: models.JSONB{
			legacyMediaRecoveryProviderKey: "meta",
			legacyMediaRecoveryMediaIDKey:  "",
		}}
		_, ok, reason := inspectLegacyMediaRecovery(msg, now)
		assert.False(t, ok)
		assert.Equal(t, "missing_metadata", reason)
	})

	t.Run("eligible", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{
			BaseModel: models.BaseModel{CreatedAt: now},
			Metadata: models.JSONB{
				legacyMediaRecoveryProviderKey: "meta",
				legacyMediaRecoveryMediaIDKey:  "media-123",
				legacyMediaRecoveryPhoneIDKey: "phone-456",
			},
		}
		info, ok, reason := inspectLegacyMediaRecovery(msg, now)
		assert.True(t, ok)
		assert.Equal(t, "eligible", reason)
		assert.Equal(t, "meta", info.Provider)
		assert.Equal(t, "media-123", info.MediaID)
		assert.Equal(t, "phone-456", info.PhoneID)
	})

	t.Run("ttl expired", func(t *testing.T) {
		t.Parallel()
		expiredTime := now.Add(-31 * 24 * time.Hour)
		msg := &models.Message{
			BaseModel: models.BaseModel{CreatedAt: expiredTime},
			Metadata: models.JSONB{
				legacyMediaRecoveryProviderKey:  "meta",
				legacyMediaRecoveryMediaIDKey:   "media-123",
				legacyMediaRecoveryExpiresAtKey: expiredTime.Add(29 * 24 * time.Hour).Format(time.RFC3339),
			},
		}
		_, ok, reason := inspectLegacyMediaRecovery(msg, now)
		assert.False(t, ok)
		assert.Equal(t, "ttl_expired", reason)
	})
}

func TestLegacyMediaRecoveryExpiresAt(t *testing.T) {
	t.Parallel()

	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	explicitExpiry := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	t.Run("nil message", func(t *testing.T) {
		t.Parallel()
		assert.True(t, legacyMediaRecoveryExpiresAt(nil).IsZero())
	})

	t.Run("nil metadata", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{}
		assert.True(t, legacyMediaRecoveryExpiresAt(msg).IsZero())
	})

	t.Run("explicit RFC3339 expiry", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{Metadata: models.JSONB{
			legacyMediaRecoveryExpiresAtKey: explicitExpiry.Format(time.RFC3339),
		}}
		result := legacyMediaRecoveryExpiresAt(msg)
		assert.Equal(t, explicitExpiry.UTC(), result)
	})

	t.Run("explicit time.Time value", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{Metadata: models.JSONB{
			legacyMediaRecoveryExpiresAtKey: explicitExpiry,
		}}
		result := legacyMediaRecoveryExpiresAt(msg)
		assert.Equal(t, explicitExpiry, result)
	})

	t.Run("fallback to created + 30 days for valid meta candidate", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{
			BaseModel: models.BaseModel{CreatedAt: created},
			Metadata: models.JSONB{
				legacyMediaRecoveryProviderKey: "meta",
				legacyMediaRecoveryMediaIDKey:  "media-123",
			},
		}
		result := legacyMediaRecoveryExpiresAt(msg)
		assert.Equal(t, created.Add(legacyMediaRecoveryTTL), result)
	})

	t.Run("no fallback for non-meta candidate", func(t *testing.T) {
		t.Parallel()
		msg := &models.Message{
			BaseModel: models.BaseModel{CreatedAt: created},
			Metadata: models.JSONB{
				legacyMediaRecoveryProviderKey: "whatsmeow",
			},
		}
		assert.True(t, legacyMediaRecoveryExpiresAt(msg).IsZero())
	})
}

func TestResolveLegacyMediaMimeType(t *testing.T) {
	t.Parallel()

	jpegData := []byte("\xff\xd8\xff\xe0\x00\x10JFIF")

	t.Run("meta mime preferred when valid and same family", func(t *testing.T) {
		t.Parallel()
		result := resolveLegacyMediaMimeType("image/jpeg", jpegData)
		assert.Equal(t, "image/jpeg", result)
	})

	t.Run("falls back to sniffed when meta is octet-stream", func(t *testing.T) {
		t.Parallel()
		result := resolveLegacyMediaMimeType("application/octet-stream", jpegData)
		assert.Equal(t, "image/jpeg", result)
	})

	t.Run("empty data falls back to sniffed text/plain", func(t *testing.T) {
		t.Parallel()
		result := resolveLegacyMediaMimeType("application/pdf", []byte{})
		assert.Equal(t, "text/plain", result)
		result2 := resolveLegacyMediaMimeType("", []byte{})
		assert.Equal(t, "text/plain", result2)
	})
}

func createTestPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 1, 1)))
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestValidateDownloadedLegacyMedia(t *testing.T) {
	t.Parallel()

	pngData := createTestPNG(t)
	validWEBP := append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), make([]byte, 100)...)
	pdfData := []byte("%PDF-1.4 test content")

	t.Run("empty data rejects", func(t *testing.T) {
		t.Parallel()
		err := validateDownloadedLegacyMedia([]byte{}, "image/jpeg", models.MessageTypeImage)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("valid image", func(t *testing.T) {
		t.Parallel()
		err := validateDownloadedLegacyMedia(pngData, "image/png", models.MessageTypeImage)
		assert.NoError(t, err)
	})

	t.Run("invalid image rejects", func(t *testing.T) {
		t.Parallel()
		err := validateDownloadedLegacyMedia([]byte("not an image"), "image/jpeg", models.MessageTypeImage)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid image")
	})

	t.Run("sticker requires valid webp", func(t *testing.T) {
		t.Parallel()
		err := validateDownloadedLegacyMedia(validWEBP, "image/webp", models.MessageTypeSticker)
		assert.NoError(t, err)
	})

	t.Run("sticker rejects invalid webp", func(t *testing.T) {
		t.Parallel()
		err := validateDownloadedLegacyMedia([]byte("not webp"), "image/webp", models.MessageTypeSticker)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid WEBP")
	})

	t.Run("valid pdf", func(t *testing.T) {
		t.Parallel()
		err := validateDownloadedLegacyMedia(pdfData, "application/pdf", models.MessageTypeDocument)
		assert.NoError(t, err)
	})

	t.Run("invalid pdf rejects", func(t *testing.T) {
		t.Parallel()
		err := validateDownloadedLegacyMedia([]byte("not pdf"), "application/pdf", models.MessageTypeDocument)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid PDF")
	})

	t.Run("non-media types pass through", func(t *testing.T) {
		t.Parallel()
		err := validateDownloadedLegacyMedia([]byte("any data"), "application/octet-stream", models.MessageTypeDocument)
		assert.NoError(t, err)
	})
}

func TestNormalizeLegacyMediaFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		hint string
		mime string
		want string
	}{
		{name: "correct extension kept", hint: "photo.jpg", mime: "image/jpeg", want: "photo.jpg"},
		{name: "missing extension appended", hint: "photo", mime: "image/jpeg", want: "photo.jpg"},
		{name: "wrong extension replaced", hint: "photo.txt", mime: "image/png", want: "photo.png"},
		{name: "empty hint becomes file.ext", hint: "", mime: "image/jpeg", want: "file.jpg"},
		{name: "path stripped to basename", hint: "/path/to/photo.jpg", mime: "image/jpeg", want: "photo.jpg"},
		{name: "empty basename with extension", hint: ".txt", mime: "image/jpeg", want: "file.jpg"},
		{name: "case insensitive extension", hint: "photo.JPG", mime: "image/jpeg", want: "photo.JPG"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, normalizeLegacyMediaFilename(tc.hint, tc.mime))
		})
	}
}

func TestResolveLegacyMediaPath(t *testing.T) {
	t.Parallel()

	t.Run("empty path rejected", func(t *testing.T) {
		t.Parallel()
		_, err := resolveLegacyMediaPath("/storage", "")
		assert.Error(t, err)
	})

	t.Run("dot path rejected", func(t *testing.T) {
		t.Parallel()
		_, err := resolveLegacyMediaPath("/storage", ".")
		assert.Error(t, err)
	})

	t.Run("parent traversal rejected", func(t *testing.T) {
		t.Parallel()
		_, err := resolveLegacyMediaPath("/storage", "../etc/passwd")
		assert.Error(t, err)
	})
}
