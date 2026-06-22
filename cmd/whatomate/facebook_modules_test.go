package main

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompiledFacebookModulesResolveWithDependencies(t *testing.T) {
	registered := core.GetPlugins()
	pluginNames := make([]string, 0, len(registered))
	for _, plugin := range registered {
		pluginNames = append(pluginNames, plugin.Name())
	}
	assert.Contains(t, pluginNames, "module-management")

	_, manifests, err := core.ResolvePlugins(registered)
	require.NoError(t, err)

	keys := make([]string, 0, len(manifests))
	for _, manifest := range manifests {
		keys = append(keys, manifest.Key)
	}

	assert.Equal(t, []string{
		"facebook-core",
		"facebook-accounts",
		"facebook-comments",
		"facebook-oauth",
		"facebook-page-search",
		"facebook-people-search",
	}, keys)
}
