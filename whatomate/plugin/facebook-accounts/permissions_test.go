package facebookaccounts

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPluginImplementsPermissionProvider is a compile-time + runtime contract
// check: facebook-accounts must implement core.PermissionProvidingPlugin so its
// plugin-namespaced permissions are seeded and enforced. This is the new
// contract introduced for plugin RBAC (requirement D).
func TestPluginImplementsPermissionProvider(t *testing.T) {
	t.Parallel()
	var p core.Plugin = &Plugin{}
	_, ok := p.(core.PermissionProvidingPlugin)
	assert.True(t, ok, "Plugin must implement core.PermissionProvidingPlugin")
}

func TestPluginDeclaresPagesManagePermission(t *testing.T) {
	t.Parallel()
	plugin := &Plugin{}
	perms := plugin.Permissions()

	require.Len(t, perms, 1)
	assert.Equal(t, "plugin.facebook.accounts", perms[0].Resource)
	assert.Equal(t, "pages_manage", perms[0].Action)
	assert.NotEmpty(t, perms[0].Description, "description is required for the permissions catalog")
}
