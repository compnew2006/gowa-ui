package whatsmeow

import (
	"context"
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

func TestPauseActiveCampaignsForInstanceTemporaryBan(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)
	org, instanceID := seedCampaignPauseData(t, db)
	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")

	cm.pauseActiveCampaignsForInstance(context.Background(), org.ID, instanceID, "temporary_ban")

	assertCampaignStatuses(t, db, org.ID, instanceID, map[models.CampaignStatus]int{
		models.CampaignStatusPaused:    2,
		models.CampaignStatusCompleted: 1,
		models.CampaignStatusCancelled: 1,
	})
}

func TestPauseActiveCampaignsForInstanceLoggedOutScopesInstanceAndOrg(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.TruncateTables(db)
	org, instanceID := seedCampaignPauseData(t, db)
	otherOrg := testutil.CreateTestOrganization(t, db)
	otherInstanceID := uuid.New()
	seedCampaignPauseCampaign(t, db, otherOrg.ID, instanceID, models.CampaignStatusProcessing)
	seedCampaignPauseCampaign(t, db, org.ID, otherInstanceID, models.CampaignStatusProcessing)
	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")

	cm.pauseActiveCampaignsForInstance(context.Background(), org.ID, instanceID, "logged_out")

	assertCampaignStatuses(t, db, org.ID, instanceID, map[models.CampaignStatus]int{
		models.CampaignStatusPaused:    2,
		models.CampaignStatusCompleted: 1,
		models.CampaignStatusCancelled: 1,
	})
	var otherOrgProcessing int64
	require.NoError(t, db.Model(&models.BulkMessageCampaign{}).
		Where("organization_id = ? AND whats_app_account = ? AND status = ?", otherOrg.ID, instanceID.String(), models.CampaignStatusProcessing).
		Count(&otherOrgProcessing).Error)
	assert.Equal(t, int64(1), otherOrgProcessing)

	var otherInstanceProcessing int64
	require.NoError(t, db.Model(&models.BulkMessageCampaign{}).
		Where("organization_id = ? AND whats_app_account = ? AND status = ?", org.ID, otherInstanceID.String(), models.CampaignStatusProcessing).
		Count(&otherInstanceProcessing).Error)
	assert.Equal(t, int64(1), otherInstanceProcessing)
}

func seedCampaignPauseData(t *testing.T, db *gorm.DB) (*models.Organization, uuid.UUID) {
	t.Helper()
	org := testutil.CreateTestOrganization(t, db)
	instanceID := uuid.New()
	seedCampaignPauseCampaign(t, db, org.ID, instanceID, models.CampaignStatusProcessing)
	seedCampaignPauseCampaign(t, db, org.ID, instanceID, models.CampaignStatusQueued)
	seedCampaignPauseCampaign(t, db, org.ID, instanceID, models.CampaignStatusCompleted)
	seedCampaignPauseCampaign(t, db, org.ID, instanceID, models.CampaignStatusCancelled)
	return org, instanceID
}

func seedCampaignPauseCampaign(t *testing.T, db *gorm.DB, orgID, instanceID uuid.UUID, status models.CampaignStatus) {
	t.Helper()
	user := testutil.CreateTestUser(t, db, orgID, testutil.WithEmail(testutil.UniqueEmail("pause-user")))
	template := testutil.CreateTestTemplate(t, db, orgID, instanceID.String())
	campaign := models.BulkMessageCampaign{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		WhatsAppAccount: instanceID.String(),
		Name:            "Pause Campaign " + uuid.NewString(),
		TemplateID:      template.ID,
		Status:          status,
		CreatedBy:       user.ID,
		MinDelaySeconds: 10,
		MaxDelaySeconds: 10,
	}
	require.NoError(t, db.Create(&campaign).Error)
}

func assertCampaignStatuses(t *testing.T, db *gorm.DB, orgID, instanceID uuid.UUID, expected map[models.CampaignStatus]int) {
	t.Helper()
	for status, want := range expected {
		var got int64
		require.NoError(t, db.Model(&models.BulkMessageCampaign{}).
			Where("organization_id = ? AND whats_app_account = ? AND status = ?", orgID, instanceID.String(), status).
			Count(&got).Error)
		assert.Equal(t, int64(want), got, "status %s", status)
	}
}
