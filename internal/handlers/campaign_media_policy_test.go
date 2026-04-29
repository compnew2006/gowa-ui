package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveCampaignUploadMIMEAcceptsMagicBytesWithGenericHeader(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d}

	mimeType, ok := resolveCampaignUploadMIME("application/octet-stream", "upload.bin", png)

	assert.True(t, ok)
	assert.Equal(t, "image/png", mimeType)
}

func TestResolveCampaignUploadMIMERejectsSpoofedHeader(t *testing.T) {
	html := []byte("<html><body>not an image</body></html>")

	mimeType, ok := resolveCampaignUploadMIME("image/jpeg", "photo.jpg", html)

	assert.False(t, ok)
	assert.Equal(t, "text/html", mimeType)
}
