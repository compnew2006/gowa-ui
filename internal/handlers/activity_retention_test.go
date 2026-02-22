package handlers_test

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestApp_PurgeOlderThan_RemovesOnlyExpiredRows(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	orgID := org.ID
	userID := user.ID

	oldLog := models.ActivityLog{
		OrganizationID: &orgID,
		UserID:         &userID,
		Category:       "system",
		EventType:      "system.api_interaction",
		Action:         "api_request",
		Status:         "success",
		Source:         "system",
	}
	newLog := models.ActivityLog{
		OrganizationID: &orgID,
		UserID:         &userID,
		Category:       "custom",
		EventType:      "ui.button_click",
		Action:         "export_contacts",
		Status:         "success",
		Source:         "custom",
	}

	require.NoError(t, app.DB.Create(&oldLog).Error)
	require.NoError(t, app.DB.Create(&newLog).Error)
	require.NoError(t, app.DB.Model(&models.ActivityLog{}).Where("id = ?", oldLog.ID).Update("created_at", time.Now().AddDate(0, 0, -120)).Error)
	require.NoError(t, app.DB.Model(&models.ActivityLog{}).Where("id = ?", newLog.ID).Update("created_at", time.Now()).Error)

	deleted, err := app.PurgeOlderThan(time.Now().AddDate(0, 0, -90))
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	var oldCheck models.ActivityLog
	err = app.DB.Unscoped().Where("id = ?", oldLog.ID).First(&oldCheck).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var newCheck models.ActivityLog
	require.NoError(t, app.DB.Where("id = ?", newLog.ID).First(&newCheck).Error)
	assert.Equal(t, newLog.ID, newCheck.ID)
}
