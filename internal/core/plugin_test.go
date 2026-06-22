package core

import (
	"log/slog"
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

type legacyTestPlugin struct {
	name   string
	onInit func()
}

func (p *legacyTestPlugin) Name() string { return p.name }

func (p *legacyTestPlugin) Init(*handlers.App, *gorm.DB, *redis.Client, *slog.Logger) error {
	if p.onInit != nil {
		p.onInit()
	}
	return nil
}

func (p *legacyTestPlugin) Routes(*fastglue.Fastglue) {}

func (p *legacyTestPlugin) Migrate(*gorm.DB) error { return nil }

type testPlugin struct {
	name     string
	manifest ModuleManifest
}

func (p *testPlugin) Name() string { return p.name }

func (p *testPlugin) Init(*handlers.App, *gorm.DB, *redis.Client, *slog.Logger) error {
	return nil
}

func (p *testPlugin) Routes(*fastglue.Fastglue) {}

func (p *testPlugin) Migrate(*gorm.DB) error { return nil }

func (p *testPlugin) Manifest() ModuleManifest { return p.manifest }

func managedTestPlugin(key string, dependencies ...string) Plugin {
	return &testPlugin{
		name: key,
		manifest: ModuleManifest{
			Key:            key,
			DisplayName:    key,
			Version:        "1.0.0",
			SchemaVersion:  1,
			Dependencies:   dependencies,
			DefaultEnabled: true,
		},
	}
}

func TestResolvePluginsOrdersManagedDependenciesAndPreservesLegacyOrder(t *testing.T) {
	legacyFirst := &legacyTestPlugin{name: "legacy-first"}
	legacySecond := &legacyTestPlugin{name: "legacy-second"}

	resolved, manifests, err := ResolvePlugins([]Plugin{
		legacyFirst,
		managedTestPlugin("facebook-comments", "facebook-accounts"),
		managedTestPlugin("facebook-core"),
		managedTestPlugin("facebook-accounts", "facebook-core"),
		legacySecond,
	})
	require.NoError(t, err)

	require.Len(t, resolved, 5)
	assert.Same(t, legacyFirst, resolved[0])
	assert.Same(t, legacySecond, resolved[1])
	assert.Equal(t, "facebook-core", resolved[2].Name())
	assert.Equal(t, "facebook-accounts", resolved[3].Name())
	assert.Equal(t, "facebook-comments", resolved[4].Name())

	require.Len(t, manifests, 3)
	assert.Equal(t, "facebook-core", manifests[0].Key)
	assert.Equal(t, "facebook-accounts", manifests[1].Key)
	assert.Equal(t, "facebook-comments", manifests[2].Key)
}

func TestResolvePluginsRejectsInvalidManagedGraph(t *testing.T) {
	tests := []struct {
		name    string
		plugins []Plugin
		wantErr string
	}{
		{
			name: "duplicate module key",
			plugins: []Plugin{
				managedTestPlugin("facebook-core"),
				managedTestPlugin("facebook-core"),
			},
			wantErr: "duplicate managed module key",
		},
		{
			name: "missing dependency",
			plugins: []Plugin{
				managedTestPlugin("facebook-comments", "facebook-accounts"),
			},
			wantErr: "missing dependency",
		},
		{
			name: "dependency cycle",
			plugins: []Plugin{
				managedTestPlugin("facebook-core", "facebook-comments"),
				managedTestPlugin("facebook-comments", "facebook-core"),
			},
			wantErr: "dependency cycle",
		},
		{
			name: "empty key",
			plugins: []Plugin{
				&testPlugin{
					name: "broken",
					manifest: ModuleManifest{
						DisplayName:    "Broken",
						Version:        "1.0.0",
						SchemaVersion:  1,
						DefaultEnabled: true,
					},
				},
			},
			wantErr: "module key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ResolvePlugins(tt.plugins)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestInitPluginsConfiguresManagedModuleRuntime(t *testing.T) {
	previousPlugins := plugins
	previousInitialized := initializedPlugins
	previousManifests := moduleManifests
	previousManager := GetModuleManager()
	t.Cleanup(func() {
		plugins = previousPlugins
		initializedPlugins = previousInitialized
		moduleManifests = previousManifests
		moduleManager = previousManager
	})

	managerAvailableDuringInit := false
	plugins = []Plugin{
		&legacyTestPlugin{name: "legacy", onInit: func() {
			managerAvailableDuringInit = GetModuleManager() != nil
		}},
		managedTestPlugin("facebook-core"),
	}
	initializedPlugins = nil
	moduleManifests = nil
	moduleManager = nil

	require.NoError(t, InitPlugins(nil, nil, nil, slog.Default()))
	assert.True(t, managerAvailableDuringInit)
	require.NotNil(t, GetModuleManager())
	assert.Equal(t, []ModuleManifest{managedTestPlugin("facebook-core").(ManagedPlugin).Manifest()}, GetModuleManifests())
}

var _ ManagedPlugin = (*testPlugin)(nil)
