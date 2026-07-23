package whatsapp_test

import (
	"testing"

	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
)

func TestClient_Name(t *testing.T) {
	t.Parallel()
	c := whatsapp.New(testutil.NopLogger())
	assert.Equal(t, "meta", c.Name())
}

func TestClient_Capabilities_AllTrue(t *testing.T) {
	t.Parallel()
	c := whatsapp.New(testutil.NopLogger())
	caps := c.Capabilities()

	// Meta supports every feature.
	assert.True(t, caps.Templates)
	assert.True(t, caps.Flows)
	assert.True(t, caps.Calls)
	assert.True(t, caps.Catalog)
	assert.True(t, caps.Analytics)
	assert.True(t, caps.BusinessProfile)
	assert.True(t, caps.MediaUpload)
	assert.True(t, caps.Interactive)
	assert.True(t, caps.AccountSetup)
}

func TestErrNotSupported(t *testing.T) {
	t.Parallel()
	// Verify the sentinel error exists and has the right message.
	assert.EqualError(t, whatsapp.ErrNotSupported, "operation not supported by this provider")
}
