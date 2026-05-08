package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestLegacyMediaRestoreMIMETypeMapping(t *testing.T) {
	assert.Equal(t, models.MessageTypeImage, legacyMediaMessageTypeFromMIME("image/png"))
	assert.Equal(t, models.MessageTypeVideo, legacyMediaMessageTypeFromMIME("video/mp4"))
	assert.Equal(t, models.MessageTypeAudio, legacyMediaMessageTypeFromMIME("audio/ogg"))
	assert.Equal(t, models.MessageTypeDocument, legacyMediaMessageTypeFromMIME("application/pdf"))
	assert.Equal(t, models.MessageTypeDocument, legacyMediaMessageTypeFromMIME("unknown/type"))
}

func TestLegacyMediaRestoreFilenameNormalization(t *testing.T) {
	result := normalizeLegacyMediaFilename("photo.txt", "image/jpeg")
	assert.Equal(t, "photo.jpg", result)

	result = normalizeLegacyMediaFilename("photo.png", "image/png")
	assert.Equal(t, "photo.png", result)

	result = normalizeLegacyMediaFilename("photo", "image/png")
	assert.Equal(t, "photo.png", result)
}
