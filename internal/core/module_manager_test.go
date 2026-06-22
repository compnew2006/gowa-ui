package core

import (
	"context"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newModuleManagerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE organizations (id text PRIMARY KEY, name text NOT NULL, slug text NOT NULL, deleted_at datetime)`).Error)
	return db
}

func createModuleManagerTestOrganization(t *testing.T, db *gorm.DB, name string) models.Organization {
	t.Helper()
	org := models.Organization{Name: name, Slug: name + "-" + uuid.NewString()}
	org.ID = uuid.New()
	require.NoError(t, db.Exec(
		"INSERT INTO organizations (id, name, slug) VALUES (?, ?, ?)",
		org.ID,
		org.Name,
		org.Slug,
	).Error)
	return org
}

func moduleManagerTestManifests() []ModuleManifest {
	return []ModuleManifest{
		{
			Key:            "facebook-core",
			DisplayName:    "Facebook Core",
			Version:        "1.0.0",
			SchemaVersion:  1,
			DefaultEnabled: true,
			Technical:      true,
		},
		{
			Key:            "facebook-accounts",
			DisplayName:    "Facebook Accounts",
			Version:        "1.0.0",
			SchemaVersion:  1,
			Dependencies:   []string{"facebook-core"},
			DefaultEnabled: true,
		},
	}
}

func TestSyncManagedModulesCreatesControlPlaneTables(t *testing.T) {
	db := newModuleManagerTestDB(t)
	previousManager := moduleManager
	previousManifests := moduleManifests
	t.Cleanup(func() {
		moduleManager = previousManager
		moduleManifests = previousManifests
	})

	moduleManifests = moduleManagerTestManifests()
	moduleManager = NewModuleManager(db, moduleManifests)

	require.NoError(t, SyncManagedModules(context.Background()))
	assert.True(t, db.Migrator().HasTable(&ModuleCatalog{}))
	assert.True(t, db.Migrator().HasTable(&OrganizationModule{}))
	assert.True(t, db.Migrator().HasTable(&ModuleSchemaVersion{}))

	var schemaVersion ModuleSchemaVersion
	require.NoError(t, db.First(&schemaVersion, "module_key = ?", "facebook-core").Error)
	assert.Equal(t, uint(1), schemaVersion.Version)
}

func TestModuleManagerSyncSeedsExistingOrganizationsEnabled(t *testing.T) {
	db := newModuleManagerTestDB(t)
	org := createModuleManagerTestOrganization(t, db, "Existing")

	manager := NewModuleManager(db, moduleManagerTestManifests())
	require.NoError(t, manager.Migrate(context.Background()))
	require.NoError(t, manager.Sync(context.Background()))

	modules, err := manager.ListEffective(context.Background(), org.ID)
	require.NoError(t, err)
	require.Len(t, modules, 2)
	assert.True(t, modules[0].EffectiveEnabled)
	assert.True(t, modules[1].EffectiveEnabled)
	assert.Equal(t, uint(1), modules[0].InstalledSchemaVersion)
}

func TestModuleManagerEffectiveStateHonorsGlobalAndOrganizationControls(t *testing.T) {
	db := newModuleManagerTestDB(t)
	org := createModuleManagerTestOrganization(t, db, "Tenant")

	manager := NewModuleManager(db, moduleManagerTestManifests())
	require.NoError(t, manager.Migrate(context.Background()))
	require.NoError(t, manager.Sync(context.Background()))

	require.NoError(t, manager.SetOrganizationEnabled(context.Background(), org.ID, "facebook-accounts", false))
	enabled, err := manager.IsEnabled(context.Background(), org.ID, "facebook-accounts")
	require.NoError(t, err)
	assert.False(t, enabled)

	require.NoError(t, manager.SetGlobalEnabled(context.Background(), "facebook-accounts", false))
	require.NoError(t, manager.SetOrganizationEnabled(context.Background(), org.ID, "facebook-accounts", true))
	enabled, err = manager.IsEnabled(context.Background(), org.ID, "facebook-accounts")
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestModuleManagerDependencyChangesAreSafeAndTransactional(t *testing.T) {
	db := newModuleManagerTestDB(t)
	org := createModuleManagerTestOrganization(t, db, "Tenant")

	manager := NewModuleManager(db, moduleManagerTestManifests())
	require.NoError(t, manager.Migrate(context.Background()))
	require.NoError(t, manager.Sync(context.Background()))

	require.NoError(t, manager.SetOrganizationEnabled(context.Background(), org.ID, "facebook-accounts", false))
	require.NoError(t, manager.SetOrganizationEnabled(context.Background(), org.ID, "facebook-core", false))
	require.NoError(t, manager.SetOrganizationEnabled(context.Background(), org.ID, "facebook-accounts", true))

	coreEnabled, err := manager.IsEnabled(context.Background(), org.ID, "facebook-core")
	require.NoError(t, err)
	assert.True(t, coreEnabled)

	err = manager.SetOrganizationEnabled(context.Background(), org.ID, "facebook-core", false)
	require.ErrorIs(t, err, ErrModuleHasEnabledDependents)
}

func TestGateModuleBlocksDisabledTenantWithoutCallingHandler(t *testing.T) {
	db := newModuleManagerTestDB(t)
	org := createModuleManagerTestOrganization(t, db, "Gate")
	manager := NewModuleManager(db, moduleManagerTestManifests())
	require.NoError(t, manager.Migrate(context.Background()))
	require.NoError(t, manager.Sync(context.Background()))

	previousManager := moduleManager
	moduleManager = manager
	t.Cleanup(func() { moduleManager = previousManager })

	called := false
	handler := GateModule("facebook-accounts", func(r *fastglue.Request) error {
		called = true
		return r.SendEnvelope(map[string]bool{"called": true})
	})

	require.NoError(t, manager.SetOrganizationEnabled(context.Background(), org.ID, "facebook-accounts", false))
	request := testutil.NewRequest(t)
	request.RequestCtx.SetUserValue(middleware.ContextKeyOrganizationID, org.ID)
	require.NoError(t, handler(request))
	assert.Equal(t, fasthttp.StatusNotFound, request.RequestCtx.Response.StatusCode())
	assert.False(t, called)

	require.NoError(t, manager.SetOrganizationEnabled(context.Background(), org.ID, "facebook-accounts", true))
	request = testutil.NewRequest(t)
	request.RequestCtx.SetUserValue(middleware.ContextKeyOrganizationID, org.ID)
	require.NoError(t, handler(request))
	assert.Equal(t, fasthttp.StatusOK, request.RequestCtx.Response.StatusCode())
	assert.True(t, called)
}
