package handlers

import (
	"strings"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDeleteOrganizationQueries_UseFreshScopedSessions(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestDB(t)
	currentOrgID := uuid.New()
	targetOrgID := uuid.New()
	userID := uuid.New()
	requestDB := tenant.ScopedDB(db.Session(&gorm.Session{DryRun: true}), currentOrgID)

	var org models.Organization
	lookupQuery := requestDB.Session(&gorm.Session{}).
		Where("id = ?", targetOrgID)
	require.NoError(t, lookupQuery.First(&org).Error)

	var orgCount int64
	countQuery := requestDB.Session(&gorm.Session{}).
		Model(&models.Organization{})
	require.NoError(t, countQuery.Count(&orgCount).Error)

	var currentUser models.User
	currentUserQuery := requestDB.Session(&gorm.Session{}).
		Select("id", "organization_id").
		Where("id = ?", userID)
	require.NoError(t, currentUserQuery.First(&currentUser).Error)

	lookupSQL := strings.ToLower(lookupQuery.Statement.SQL.String())
	countSQL := strings.ToLower(countQuery.Statement.SQL.String())
	currentUserSQL := strings.ToLower(currentUserQuery.Statement.SQL.String())

	require.Contains(t, lookupSQL, "from `organizations`")
	require.Contains(t, lookupSQL, "`organizations`.`id`")
	require.Contains(t, countSQL, "from `organizations`")
	require.NotContains(t, countSQL, "`organizations`.`id`")
	require.Contains(t, currentUserSQL, "from `users`")
	require.NotContains(t, currentUserSQL, "from `organizations`")
}

func TestInstanceUpdates_UseFreshScopedSessions(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestDB(t)
	orgID := uuid.New()
	instanceID := uuid.New()

	org := models.Organization{
		BaseModel: models.BaseModel{ID: orgID},
		Name:      "Regression Org",
		Slug:      "regression-org-instance-update",
	}
	require.NoError(t, db.Create(&org).Error)

	instance := models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: instanceID},
		OrganizationID: orgID,
		Name:           "Regression Instance",
		Status:         models.InstanceStatusDisconnected,
		Settings: models.JSONB{
			"auto_sync_history": true,
		},
	}
	require.NoError(t, db.Create(&instance).Error)

	requestDB := tenant.ScopedDB(db, orgID)
	var loaded models.WhatsAppInstance
	require.NoError(t, requestDB.
		Where("id = ? AND organization_id = ?", instanceID, orgID).
		First(&loaded).Error)

	updates := map[string]any{
		"settings": models.JSONB{
			"auto_sync_history":            true,
			"auto_download_incoming_media": true,
		},
	}

	err := tenant.ScopedDB(db.Session(&gorm.Session{}), orgID).
		Model(&models.WhatsAppInstance{}).
		Where("id = ?", instanceID).
		Updates(updates).Error
	require.NoError(t, err)

	var refreshed models.WhatsAppInstance
	require.NoError(t, db.Where("id = ?", instanceID).First(&refreshed).Error)
	require.Equal(t, true, refreshed.Settings["auto_download_incoming_media"])
	require.Equal(t, true, refreshed.Settings["auto_sync_history"])
}
