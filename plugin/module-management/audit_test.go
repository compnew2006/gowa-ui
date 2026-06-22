package modulemanagement

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, MigrateModuleEvents(db))
	return db
}

func TestMigrateModuleEventsCreatesTable(t *testing.T) {
	t.Parallel()
	db := newAuditTestDB(t)
	assert.True(t, db.Migrator().HasTable(&ModuleEvent{}))
}

func TestMigrateModuleEventsIsIdempotent(t *testing.T) {
	t.Parallel()
	db := newAuditTestDB(t)
	// Calling again must not error.
	require.NoError(t, MigrateModuleEvents(db))
	assert.True(t, db.Migrator().HasTable(&ModuleEvent{}))
}

func TestMigrateModuleEventsNilDBIsNoop(t *testing.T) {
	t.Parallel()
	require.NoError(t, MigrateModuleEvents(nil))
}

func TestModuleEventWriteReadRoundtrip(t *testing.T) {
	t.Parallel()
	db := newAuditTestDB(t)

	actor := uuid.New()
	orgID := uuid.New()
	enabled := true
	original := ModuleEvent{
		ID:             uuid.New(),
		OrganizationID: &orgID,
		Scope:          moduleScopeOrganization,
		ModuleKey:      "facebook-accounts",
		Action:         ModuleActionEnable,
		Enabled:        &enabled,
		ActorUserID:    &actor,
		ActorEmail:     "admin@example.com",
		Reason:         "tenant requested",
		Details:        models.JSONB{"tier": "starter"},
	}
	require.NoError(t, db.Create(&original).Error)

	var fetched ModuleEvent
	require.NoError(t, db.First(&fetched, "id = ?", original.ID).Error)
	assert.Equal(t, original.OrganizationID, fetched.OrganizationID)
	assert.Equal(t, moduleScopeOrganization, fetched.Scope)
	assert.Equal(t, "facebook-accounts", fetched.ModuleKey)
	assert.Equal(t, ModuleActionEnable, fetched.Action)
	require.NotNil(t, fetched.Enabled)
	assert.True(t, *fetched.Enabled)
	assert.Equal(t, &actor, fetched.ActorUserID)
	assert.Equal(t, "admin@example.com", fetched.ActorEmail)
	assert.Equal(t, "tenant requested", fetched.Reason)
	assert.Equal(t, "starter", fetched.Details["tier"])
}

func TestModuleEventGlobalScopeHasNilOrganization(t *testing.T) {
	t.Parallel()
	db := newAuditTestDB(t)

	event := ModuleEvent{
		ID:        uuid.New(),
		Scope:     moduleScopeGlobal,
		ModuleKey: "facebook-core",
		Action:    ModuleActionDisable,
		Details:   models.JSONB{},
	}
	require.NoError(t, db.Create(&event).Error)

	var fetched ModuleEvent
	require.NoError(t, db.First(&fetched, "id = ?", event.ID).Error)
	assert.Nil(t, fetched.OrganizationID)
	assert.Equal(t, moduleScopeGlobal, fetched.Scope)
}
