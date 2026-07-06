package facebookcore

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestPluginManifestDefinesTechnicalFacebookFoundation(t *testing.T) {
	plugin := &Plugin{}
	manifest := plugin.Manifest()

	assert.Equal(t, "facebook-core", plugin.Name())
	assert.Equal(t, "facebook-core", manifest.Key)
	assert.Equal(t, "Facebook Core", manifest.DisplayName)
	assert.Equal(t, "1.0.0", manifest.Version)
	assert.Equal(t, uint(1), manifest.SchemaVersion)
	assert.True(t, manifest.DefaultEnabled)
	assert.True(t, manifest.Technical)
	assert.Empty(t, manifest.Dependencies)
}

var _ core.ManagedPlugin = (*Plugin)(nil)
