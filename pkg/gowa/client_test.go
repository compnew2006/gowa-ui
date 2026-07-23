package gowa_test

import (
	"context"
	"testing"

	"github.com/shridarpatil/whatomate/pkg/gowa"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Capabilities(t *testing.T) {
	t.Parallel()
	c := gowa.New("http://localhost:3000", "", "")
	caps := c.Capabilities()

	// GOWA does NOT support any Meta-only features.
	assert.False(t, caps.Templates)
	assert.False(t, caps.Flows)
	assert.False(t, caps.Calls)
	assert.False(t, caps.Catalog)
	assert.False(t, caps.BusinessProfile)
	assert.False(t, caps.MediaUpload)
	assert.False(t, caps.Interactive)
	assert.False(t, caps.AccountSetup)
}

func TestClient_MetaOnlyMethodsReturnErrNotSupported(t *testing.T) {
	t.Parallel()
	c := gowa.New("http://localhost:3000", "", "")
	ctx := context.Background()
	account := &whatsapp.Account{ProviderType: "gowa", GowaDeviceID: "dev1"}

	t.Run("SendTemplateMessage", func(t *testing.T) {
		_, err := c.SendTemplateMessage(ctx, account, whatsapp.Recipient{}, "tmpl", "en", nil)
		assert.ErrorIs(t, err, whatsapp.ErrNotSupported)
	})
	t.Run("SendFlowMessage", func(t *testing.T) {
		_, err := c.SendFlowMessage(ctx, account, whatsapp.Recipient{}, "f1", "h", "b", "c", "t", "s")
		assert.ErrorIs(t, err, whatsapp.ErrNotSupported)
	})
	t.Run("SubmitTemplate", func(t *testing.T) {
		_, err := c.SubmitTemplate(ctx, account, &whatsapp.TemplateSubmission{})
		assert.ErrorIs(t, err, whatsapp.ErrNotSupported)
	})
	t.Run("FetchTemplates", func(t *testing.T) {
		_, err := c.FetchTemplates(ctx, account)
		assert.ErrorIs(t, err, whatsapp.ErrNotSupported)
	})
	t.Run("CreateCatalog", func(t *testing.T) {
		_, err := c.CreateCatalog(ctx, account, "catalog")
		assert.ErrorIs(t, err, whatsapp.ErrNotSupported)
	})
	t.Run("GetBusinessProfile", func(t *testing.T) {
		_, err := c.GetBusinessProfile(ctx, account)
		assert.ErrorIs(t, err, whatsapp.ErrNotSupported)
	})
	t.Run("ValidateCredentials", func(t *testing.T) {
		_, err := c.ValidateCredentials(ctx, "p", "b", "t", "v21.0")
		assert.ErrorIs(t, err, whatsapp.ErrNotSupported)
	})
	t.Run("SubscribeApp", func(t *testing.T) {
		err := c.SubscribeApp(ctx, account)
		assert.ErrorIs(t, err, whatsapp.ErrNotSupported)
	})
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
	account := &whatsapp.Account{ProviderType: "gowa", GowaDeviceID: "dev1"}

	data := []byte("fake-image-bytes")
	mediaID, err := c.UploadMedia(ctx, account, data, "image/jpeg", "photo.jpg")
	require.NoError(t, err)
	assert.NotEmpty(t, mediaID)

	// The media should be consumable by SendImageMessage via the cache.
	// We can't test the full send without a mock server here, but we
	// verify the cache key is generated.
	assert.Contains(t, mediaID, "gowa-media-")
}
