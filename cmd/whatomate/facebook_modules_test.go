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

	// Use ElementsMatch (set comparison) rather than an exact ordered slice:
	// ResolvePlugins returns manifests in topologically-sorted dependency order,
	// and the precise ordering is incidental to this test's intent — that every
	// compiled Facebook module resolves, all dependencies are satisfied, and no
	// dependency cycle exists. The managed Facebook module set is enumerated
	// exhaustively here so adding a new module without registering it fails
	// loudly.
	assert.ElementsMatch(t, []string{
		"facebook-accounts",
		"facebook-auto-share",
		"facebook-comments",
		"facebook-core",
		"facebook-extract-data",
		"facebook-extract-likes",
		"facebook-group-search",
		"facebook-oauth",
		"facebook-page-messengers",
		"facebook-page-search",
		"facebook-people-search",
		"facebook-retargeting",
	}, keys)
}
