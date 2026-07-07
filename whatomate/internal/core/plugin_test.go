package core

import (
	"context"
	"log/slog"
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/fastglue"
	"gorm.io/driver/sqlite"
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

// permissionTestPlugin implements both ManagedPlugin and PermissionProvidingPlugin
// to exercise the plugin-permission collector path.
type permissionTestPlugin struct {
	testPlugin
	perms []PluginPermission
}

func (p *permissionTestPlugin) Permissions() []PluginPermission { return p.perms }

func TestResolvePluginsCollectsPluginPermissions(t *testing.T) {
	previousPerms := pluginPermissions
	t.Cleanup(func() { pluginPermissions = previousPerms })

	_, _, err := ResolvePlugins([]Plugin{
		&permissionTestPlugin{
			testPlugin: testPlugin{
				name: "facebook-accounts",
				manifest: ModuleManifest{
					Key: "facebook-accounts", DisplayName: "Facebook Accounts",
					Version: "1.0.0", SchemaVersion: 1, DefaultEnabled: true,
				},
			},
			perms: []PluginPermission{
				{Resource: "plugin.facebook.accounts", Action: "pages_manage", Description: "Manage pages"},
			},
		},
		managedTestPlugin("facebook-core"), // no perms
	})
	require.NoError(t, err)

	collected := PluginPermissions()
	assert.Len(t, collected, 1)
	assert.Equal(t, "plugin.facebook.accounts", collected[0].Resource)
	assert.Equal(t, "pages_manage", collected[0].Action)
}

func TestPluginPermissionsReturnsEmptyWhenUnset(t *testing.T) {
	previousPerms := pluginPermissions
	t.Cleanup(func() { pluginPermissions = previousPerms })
	pluginPermissions = nil
	assert.Empty(t, PluginPermissions())
}

func TestPluginPermissionsReturnsDefensiveCopy(t *testing.T) {
	previousPerms := pluginPermissions
	t.Cleanup(func() { pluginPermissions = previousPerms })
	pluginPermissions = []PluginPermission{{Resource: "r", Action: "a"}}
	got := PluginPermissions()
	got[0].Resource = "mutated"
	// Mutation of the returned slice must not bleed into the package state.
	assert.Equal(t, "r", pluginPermissions[0].Resource)
}

func newPluginPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	// Raw DDL: models.BaseModel.ID uses default:gen_random_uuid() which is
	// Postgres-only; SQLite rejects it during AutoMigrate. Mirror the columns
	// and unique index that matter to SyncPluginPermissions.
	require.NoError(t, db.Exec(`CREATE TABLE permissions (
		id text PRIMARY KEY,
		created_at datetime,
		updated_at datetime,
		deleted_at datetime,
		resource text NOT NULL,
		action text NOT NULL,
		description text
	)`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX idx_permission_resource_action ON permissions(resource, action)`).Error)
	return db
}

func TestSyncPluginPermissionsCreatesMissingRows(t *testing.T) {
	previousPerms := pluginPermissions
	t.Cleanup(func() { pluginPermissions = previousPerms })
	pluginPermissions = []PluginPermission{
		{Resource: "plugin.facebook.accounts", Action: "pages_manage", Description: "Manage pages"},
		{Resource: "plugin.instagram.direct", Action: "write", Description: "Send Instagram DMs"},
	}
	db := newPluginPermissionTestDB(t)

	require.NoError(t, SyncPluginPermissions(context.Background(), db))

	var count int64
	require.NoError(t, db.Model(&models.Permission{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)

	var perm models.Permission
	require.NoError(t, db.Where("resource = ? AND action = ?", "plugin.facebook.accounts", "pages_manage").First(&perm).Error)
	assert.Equal(t, "Manage pages", perm.Description)
}

func TestSyncPluginPermissionsIsIdempotent(t *testing.T) {
	previousPerms := pluginPermissions
	t.Cleanup(func() { pluginPermissions = previousPerms })
	pluginPermissions = []PluginPermission{
		{Resource: "plugin.facebook.accounts", Action: "pages_manage", Description: "Manage pages"},
	}
	db := newPluginPermissionTestDB(t)

	require.NoError(t, SyncPluginPermissions(context.Background(), db))
	// Second call must not create duplicates or error.
	require.NoError(t, SyncPluginPermissions(context.Background(), db))

	var count int64
	require.NoError(t, db.Model(&models.Permission{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "second sync must not duplicate")
}

func TestSyncPluginPermissionsRejectsEmptyEntries(t *testing.T) {
	previousPerms := pluginPermissions
	t.Cleanup(func() { pluginPermissions = previousPerms })
	pluginPermissions = []PluginPermission{
		{Resource: "", Action: "write", Description: "broken"},
	}
	db := newPluginPermissionTestDB(t)

	err := SyncPluginPermissions(context.Background(), db)
	require.Error(t, err)
	assert.ErrorContains(t, err, "empty resource or action")
}

func TestSyncPluginPermissionsNoopWithoutContributions(t *testing.T) {
	previousPerms := pluginPermissions
	t.Cleanup(func() { pluginPermissions = previousPerms })
	pluginPermissions = nil
	db := newPluginPermissionTestDB(t)

	require.NoError(t, SyncPluginPermissions(context.Background(), db))

	var count int64
	require.NoError(t, db.Model(&models.Permission{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestSyncPluginPermissionsRejectsNilDB(t *testing.T) {
	err := SyncPluginPermissions(context.Background(), nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "database is nil")
}

// Compile-time assertion that permissionTestPlugin satisfies both interfaces.
var (
	_ ManagedPlugin             = (*permissionTestPlugin)(nil)
	_ PermissionProvidingPlugin = (*permissionTestPlugin)(nil)
)

// dummyGatingModulePlugin is used to test GatingModule embedding.
type dummyGatingModulePlugin struct {
	GatingModule
}

func (p *dummyGatingModulePlugin) Name() string {
	return "dummy-gating-module"
}

func (p *dummyGatingModulePlugin) Manifest() ModuleManifest {
	return ModuleManifest{
		Key:            "dummy-gating-module",
		DisplayName:    "Dummy Gating Module",
		Version:        "1.0.0",
		SchemaVersion:  1,
		DefaultEnabled: true,
	}
}

func TestNewGatingModule(t *testing.T) {
	gm := NewGatingModule()

	// 1. Verify it can initialize embedded PluginBase fields
	var app handlers.App
	db := newPluginPermissionTestDB(t)
	rdb := redis.NewClient(&redis.Options{})
	logger := slog.Default()

	err := gm.Init(&app, db, rdb, logger)
	require.NoError(t, err)

	assert.Same(t, &app, gm.App)
	assert.Same(t, db, gm.DB)
	assert.Same(t, rdb, gm.RDB)
	assert.Same(t, logger, gm.Log)

	// 2. Verify Migrate returns nil
	err = gm.Migrate(db)
	assert.NoError(t, err)

	// 3. Verify Routes does not panic
	require.NotPanics(t, func() {
		gm.Routes(fastglue.New())
	})

	// 4. Verify structural inheritance fulfills ManagedPlugin
	plugin := &dummyGatingModulePlugin{GatingModule: gm}
	var _ ManagedPlugin = plugin
	var _ Plugin = plugin

	assert.Equal(t, "dummy-gating-module", plugin.Name())
	assert.Equal(t, "dummy-gating-module", plugin.Manifest().Key)
}
