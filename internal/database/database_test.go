package database_test

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/database"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// cleanAll truncates every table so each test starts with a blank slate.
func cleanAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	testutil.TruncateTables(db)
}

// --- SeedPermissionsAndRoles ---

func TestSeedPermissionsAndRoles_CreatesAllDefaultPermissions(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	err := database.SeedPermissionsAndRoles(db)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Permission{}).Count(&count)

	expected := len(models.DefaultPermissions())
	assert.Equal(t, int64(expected), count, "all default permissions should be created")
}

func TestSeedPermissionsAndRoles_Idempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	// Seed twice
	require.NoError(t, database.SeedPermissionsAndRoles(db))
	require.NoError(t, database.SeedPermissionsAndRoles(db))

	var count int64
	db.Model(&models.Permission{}).Count(&count)

	expected := len(models.DefaultPermissions())
	assert.Equal(t, int64(expected), count, "idempotent: count should remain the same after two seeds")
}

func TestSeedPermissionsAndRoles_PermissionsHaveResourceAndAction(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	require.NoError(t, database.SeedPermissionsAndRoles(db))

	var perms []models.Permission
	db.Find(&perms)

	for _, p := range perms {
		assert.NotEmpty(t, p.Resource, "permission resource must not be empty")
		assert.NotEmpty(t, p.Action, "permission action must not be empty")
		assert.NotEqual(t, uuid.Nil, p.ID, "permission ID must be set")
	}
}

// --- SeedSystemRolesForOrg ---

func TestSeedSystemRolesForOrg_CreatesThreeSystemRoles(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	// Need permissions first
	require.NoError(t, database.SeedPermissionsAndRoles(db))

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Test Org",
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	err := database.SeedSystemRolesForOrg(db, org.ID)
	require.NoError(t, err)

	var roles []models.CustomRole
	db.Where("organization_id = ? AND is_system = ?", org.ID, true).Find(&roles)
	assert.Len(t, roles, 3, "should create admin, manager, agent roles")

	names := make(map[string]bool)
	for _, r := range roles {
		names[r.Name] = true
	}
	assert.True(t, names["admin"], "admin role should exist")
	assert.True(t, names["manager"], "manager role should exist")
	assert.True(t, names["agent"], "agent role should exist")
}

func TestSeedSystemRolesForOrg_Idempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	require.NoError(t, database.SeedPermissionsAndRoles(db))

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Idempotent Org",
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	require.NoError(t, database.SeedSystemRolesForOrg(db, org.ID))
	require.NoError(t, database.SeedSystemRolesForOrg(db, org.ID))

	var count int64
	db.Model(&models.CustomRole{}).Where("organization_id = ? AND is_system = ?", org.ID, true).Count(&count)
	assert.Equal(t, int64(3), count, "idempotent: still exactly 3 system roles")
}

func TestSeedSystemRolesForOrg_AgentIsDefault(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	require.NoError(t, database.SeedPermissionsAndRoles(db))

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Default Role Org",
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)
	require.NoError(t, database.SeedSystemRolesForOrg(db, org.ID))

	var agentRole models.CustomRole
	err := db.Where("organization_id = ? AND name = ? AND is_system = ?", org.ID, "agent", true).First(&agentRole).Error
	require.NoError(t, err)
	assert.True(t, agentRole.IsDefault, "agent role should be the default role")
}

func TestSeedSystemRolesForOrg_AdminRoleHasAllPermissions(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	require.NoError(t, database.SeedPermissionsAndRoles(db))

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Admin Perms Org",
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)
	require.NoError(t, database.SeedSystemRolesForOrg(db, org.ID))

	var adminRole models.CustomRole
	err := db.Where("organization_id = ? AND name = ? AND is_system = ?", org.ID, "admin", true).First(&adminRole).Error
	require.NoError(t, err)

	// Load permissions through the association
	var perms []models.Permission
	err = db.Model(&adminRole).Association("Permissions").Find(&perms)
	require.NoError(t, err)

	totalPerms := len(models.DefaultPermissions())
	assert.Equal(t, totalPerms, len(perms), "admin role should have all permissions")
}

func TestBackfillAdminChatDeletePermission_AddsMissingPermission(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	require.NoError(t, database.SeedPermissionsAndRoles(db))

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Backfill Org",
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	var allPerms []models.Permission
	require.NoError(t, db.Find(&allPerms).Error)

	filteredPerms := make([]models.Permission, 0, len(allPerms))
	for _, perm := range allPerms {
		if perm.Resource == models.ResourceChat && perm.Action == models.ActionDelete {
			continue
		}
		filteredPerms = append(filteredPerms, perm)
	}

	role := models.CustomRole{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "admin",
		Description:    "System admin",
		IsSystem:       true,
	}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Model(&role).Association("Permissions").Replace(filteredPerms))

	require.NoError(t, database.BackfillAdminChatDeletePermission(db))

	var refreshed models.CustomRole
	require.NoError(t, db.Preload("Permissions").First(&refreshed, "id = ?", role.ID).Error)

	hasChatDelete := false
	for _, perm := range refreshed.Permissions {
		if perm.Resource == models.ResourceChat && perm.Action == models.ActionDelete {
			hasChatDelete = true
			break
		}
	}
	assert.True(t, hasChatDelete, "backfill should add chat:delete permission to admin role")
}

func TestBackfillAdminChatDeletePermission_Idempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	require.NoError(t, database.SeedPermissionsAndRoles(db))

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Backfill Idempotent Org",
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	role := models.CustomRole{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "admin",
		Description:    "System admin",
		IsSystem:       true,
	}
	require.NoError(t, db.Create(&role).Error)

	require.NoError(t, database.BackfillAdminChatDeletePermission(db))
	require.NoError(t, database.BackfillAdminChatDeletePermission(db))

	var chatDeletePermission models.Permission
	require.NoError(t, db.Where("resource = ? AND action = ?", models.ResourceChat, models.ActionDelete).
		First(&chatDeletePermission).Error)

	var count int64
	require.NoError(t, db.Table("role_permissions").
		Where("custom_role_id = ? AND permission_id = ?", role.ID, chatDeletePermission.ID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count, "backfill should not create duplicate role_permissions rows")
}

func TestBackfillSystemChatPrefixPermission_AddsMissingPermission(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	require.NoError(t, database.SeedPermissionsAndRoles(db))

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Prefix Backfill Org",
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	roleNames := []string{"admin", "manager", "agent"}
	for _, roleName := range roleNames {
		role := models.CustomRole{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: org.ID,
			Name:           roleName,
			Description:    "System role",
			IsSystem:       true,
		}
		require.NoError(t, db.Create(&role).Error)
	}

	require.NoError(t, database.BackfillSystemChatPrefixPermission(db))

	var chatPrefixPermission models.Permission
	require.NoError(t, db.Where("resource = ? AND action = ?", models.ResourceChat, models.ActionPrefix).
		First(&chatPrefixPermission).Error)

	for _, roleName := range roleNames {
		var role models.CustomRole
		require.NoError(t, db.Where("organization_id = ? AND name = ? AND is_system = ?", org.ID, roleName, true).First(&role).Error)

		var count int64
		require.NoError(t, db.Table("role_permissions").
			Where("custom_role_id = ? AND permission_id = ?", role.ID, chatPrefixPermission.ID).
			Count(&count).Error)
		assert.Equal(t, int64(1), count, "backfill should add chat:prefix to %s role", roleName)
	}
}

func TestBackfillSystemChatPrefixPermission_Idempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	require.NoError(t, database.SeedPermissionsAndRoles(db))

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Prefix Backfill Idempotent Org",
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	role := models.CustomRole{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "agent",
		Description:    "System role",
		IsSystem:       true,
	}
	require.NoError(t, db.Create(&role).Error)

	require.NoError(t, database.BackfillSystemChatPrefixPermission(db))
	require.NoError(t, database.BackfillSystemChatPrefixPermission(db))

	var chatPrefixPermission models.Permission
	require.NoError(t, db.Where("resource = ? AND action = ?", models.ResourceChat, models.ActionPrefix).
		First(&chatPrefixPermission).Error)

	var count int64
	require.NoError(t, db.Table("role_permissions").
		Where("custom_role_id = ? AND permission_id = ?", role.ID, chatPrefixPermission.ID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count, "backfill should not create duplicate role_permissions rows")
}

func TestBackfillAdminUploadsCleanupPermissions_AddsMissingPermissions(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	require.NoError(t, database.SeedPermissionsAndRoles(db))

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Uploads Cleanup Backfill Org",
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	adminRole := models.CustomRole{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "admin",
		Description:    "System admin",
		IsSystem:       true,
	}
	managerRole := models.CustomRole{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "manager",
		Description:    "System manager",
		IsSystem:       true,
	}
	require.NoError(t, db.Create(&adminRole).Error)
	require.NoError(t, db.Create(&managerRole).Error)

	require.NoError(t, database.BackfillAdminUploadsCleanupPermissions(db))

	requiredPermissions := []struct {
		resource string
		action   string
	}{
		{resource: models.ResourceSettingsUploadsCleanup, action: models.ActionRead},
		{resource: models.ResourceSettingsUploadsCleanup, action: models.ActionWrite},
		{resource: models.ResourceSettingsUploadsCleanup, action: models.ActionExecute},
	}

	for _, required := range requiredPermissions {
		var permission models.Permission
		require.NoError(t, db.Where("resource = ? AND action = ?", required.resource, required.action).
			First(&permission).Error)

		var adminCount int64
		require.NoError(t, db.Table("role_permissions").
			Where("custom_role_id = ? AND permission_id = ?", adminRole.ID, permission.ID).
			Count(&adminCount).Error)
		assert.Equal(t, int64(1), adminCount, "admin role should receive %s:%s", required.resource, required.action)

		var managerCount int64
		require.NoError(t, db.Table("role_permissions").
			Where("custom_role_id = ? AND permission_id = ?", managerRole.ID, permission.ID).
			Count(&managerCount).Error)
		assert.Equal(t, int64(0), managerCount, "manager role should not receive %s:%s", required.resource, required.action)
	}
}

func TestBackfillAdminUploadsCleanupPermissions_Idempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	require.NoError(t, database.SeedPermissionsAndRoles(db))

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Uploads Cleanup Backfill Idempotent Org",
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	adminRole := models.CustomRole{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "admin",
		Description:    "System admin",
		IsSystem:       true,
	}
	require.NoError(t, db.Create(&adminRole).Error)

	require.NoError(t, database.BackfillAdminUploadsCleanupPermissions(db))
	require.NoError(t, database.BackfillAdminUploadsCleanupPermissions(db))

	var executePermission models.Permission
	require.NoError(t, db.Where("resource = ? AND action = ?", models.ResourceSettingsUploadsCleanup, models.ActionExecute).
		First(&executePermission).Error)

	var count int64
	require.NoError(t, db.Table("role_permissions").
		Where("custom_role_id = ? AND permission_id = ?", adminRole.ID, executePermission.ID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count, "backfill should not create duplicate role_permissions rows")
}

// --- CreateDefaultAdmin ---

func TestCreateDefaultAdmin_CreatesOrgAndUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	cfg := &config.DefaultAdminConfig{
		Email:    "test-admin@example.com",
		Password: "testpassword123",
		FullName: "Test Admin",
	}

	err := database.CreateDefaultAdmin(db, cfg)
	require.NoError(t, err)

	// Verify user was created
	var user models.User
	err = db.Where("email = ?", cfg.Email).First(&user).Error
	require.NoError(t, err)
	assert.Equal(t, cfg.FullName, user.FullName)
	assert.True(t, user.IsActive)
	assert.True(t, user.IsSuperAdmin)
	assert.NotEmpty(t, user.PasswordHash)

	// Verify an organization was created
	var org models.Organization
	err = db.First(&org).Error
	require.NoError(t, err)
	assert.Equal(t, "Default Organization", org.Name)

	// Verify the user belongs to the organization
	assert.Equal(t, org.ID, user.OrganizationID)
}

func TestCreateDefaultAdmin_Idempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	cfg := &config.DefaultAdminConfig{
		Email:    "idempotent-admin@example.com",
		Password: "pass123",
		FullName: "Idempotent Admin",
	}

	require.NoError(t, database.CreateDefaultAdmin(db, cfg))
	require.NoError(t, database.CreateDefaultAdmin(db, cfg))

	var count int64
	db.Model(&models.User{}).Where("email = ?", cfg.Email).Count(&count)
	assert.Equal(t, int64(1), count, "should not create duplicate admin")
}

func TestCreateDefaultAdmin_UsesExistingOrg(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	// Pre-create an organization
	existingOrg := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Pre-existing Org",
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&existingOrg).Error)

	cfg := &config.DefaultAdminConfig{
		Email:    "admin-existing-org@example.com",
		Password: "password",
		FullName: "Admin",
	}

	err := database.CreateDefaultAdmin(db, cfg)
	require.NoError(t, err)

	var user models.User
	err = db.Where("email = ?", cfg.Email).First(&user).Error
	require.NoError(t, err)
	assert.Equal(t, existingOrg.ID, user.OrganizationID, "admin should belong to existing org")

	// Should not have created a new org
	var orgCount int64
	db.Model(&models.Organization{}).Count(&orgCount)
	assert.Equal(t, int64(1), orgCount, "should reuse existing organization")
}

func TestCreateIndexes_AllowsMultipleUnpairedInstancesWithEmptyJID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	require.NoError(t, database.CreateIndexes(db))

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Instance Index Org",
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	first := models.WhatsAppInstance{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		Name:            "First",
		JID:             "",
		Status:          models.InstanceStatusDisconnected,
		Settings:        models.JSONB{},
		AutoReadReceipt: false,
	}
	second := models.WhatsAppInstance{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		Name:            "Second",
		JID:             "",
		Status:          models.InstanceStatusDisconnected,
		Settings:        models.JSONB{},
		AutoReadReceipt: false,
	}

	require.NoError(t, db.Create(&first).Error)
	require.NoError(t, db.Create(&second).Error)
}

func TestCreateIndexes_ContactsAreScopedPerInstance(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cleanAll(t, db)

	require.NoError(t, database.CreateIndexes(db))

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Contacts Per Instance Org",
		Slug:      "contacts-per-instance-" + uuid.NewString(),
		Settings:  models.JSONB{},
	}
	require.NoError(t, db.Create(&org).Error)

	instanceA := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Instance A",
		Status:         models.InstanceStatusDisconnected,
		Settings:       models.JSONB{},
	}
	instanceB := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Instance B",
		Status:         models.InstanceStatusDisconnected,
		Settings:       models.JSONB{},
	}
	require.NoError(t, db.Create(&instanceA).Error)
	require.NoError(t, db.Create(&instanceB).Error)

	contactA := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instanceA.ID,
		PhoneNumber:    "15550009999",
		ProfileName:    "Contact A",
		Metadata:       models.JSONB{},
	}
	contactB := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instanceB.ID,
		PhoneNumber:    "15550009999",
		ProfileName:    "Contact B",
		Metadata:       models.JSONB{},
	}
	require.NoError(t, db.Create(&contactA).Error)
	require.NoError(t, db.Create(&contactB).Error)

	duplicateSameInstance := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		InstanceID:     &instanceA.ID,
		PhoneNumber:    "15550009999",
		ProfileName:    "Duplicate",
		Metadata:       models.JSONB{},
	}
	require.Error(t, db.Create(&duplicateSameInstance).Error)

	legacyNoInstance := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    "15550008888",
		ProfileName:    "Legacy",
		Metadata:       models.JSONB{},
	}
	require.NoError(t, db.Create(&legacyNoInstance).Error)

	legacyNoInstanceDuplicate := models.Contact{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		PhoneNumber:    "15550008888",
		ProfileName:    "Legacy Duplicate",
		Metadata:       models.JSONB{},
	}
	require.Error(t, db.Create(&legacyNoInstanceDuplicate).Error)
}
