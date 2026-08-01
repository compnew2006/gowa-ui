package gowa_test

import (
	"context"
	"testing"

	"github.com/shridarpatil/gowa-ui/pkg/gowa"
	"github.com/shridarpatil/gowa-ui/pkg/whatsapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Capabilities(t *testing.T) {
	t.Parallel()
	c := gowa.New("http://localhost:3000", "", "")
	caps := c.Capabilities()

	// GOWA sends media inline (no two-step upload) and has no native
	// interactive buttons in v8.10.0.
	assert.False(t, caps.MediaUpload)
	assert.False(t, caps.Interactive)
}

func TestClient_InteractiveButtonsReturnErrNotSupported(t *testing.T) {
	t.Parallel()
	c := gowa.New("http://localhost:3000", "", "")
	ctx := context.Background()
	account := &whatsapp.Account{GowaDeviceID: "dev1"}

	_, err := c.SendInteractiveButtons(ctx, account, whatsapp.Recipient{}, "body", nil)
	assert.ErrorIs(t, err, whatsapp.ErrNotSupported)
}

func TestClient_SatisfiesProviderInterface(t *testing.T) {
	t.Parallel()
	// Compile-time check that *gowa.Client implements whatsapp.Provider.
	var _ whatsapp.Provider = (*gowa.Client)(nil)
}

func TestClient_UploadMediaCachesForInlineSend(t *testing.T) {
	t.Parallel()
	c := gowa.New("http://localhost:3000", "", "")
	ctx := context.Background()
	account := &whatsapp.Account{GowaDeviceID: "dev1"}

	data := []byte("fake-image-bytes")
	mediaID, err := c.UploadMedia(ctx, account, data, "image/jpeg", "photo.jpg")
	require.NoError(t, err)
	assert.NotEmpty(t, mediaID)

	// The media should be consumable by SendImageMessage via the cache.
	// We can't test the full send without a mock server here, but we
	// verify the cache key is generated.
	assert.Contains(t, mediaID, "gowa-media-")
}
