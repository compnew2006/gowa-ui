package whatsmeow

import (
	"context"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

func (cm *ConnectionManager) pauseActiveCampaignsForInstance(ctx context.Context, orgID, instanceID uuid.UUID, trigger string) {
	if cm == nil || cm.db == nil || orgID == uuid.Nil || instanceID == uuid.Nil {
		return
	}

	result := cm.db.WithContext(ctx).
		Model(&models.BulkMessageCampaign{}).
		Where("organization_id = ? AND whats_app_account = ? AND status IN ?", orgID, instanceID.String(), []models.CampaignStatus{
			models.CampaignStatusProcessing,
			models.CampaignStatusQueued,
		}).
		Update("status", models.CampaignStatusPaused)
	if result.Error != nil {
		cm.logger.Warn("Failed to pause active campaigns for blocked instance", "error", result.Error, "instance_id", instanceID, "trigger", trigger)
		return
	}
	if result.RowsAffected > 0 {
		cm.logger.Warn(
			"Paused active campaigns after instance outbound block",
			"instance_id", instanceID,
			"organization_id", orgID,
			"trigger", trigger,
			"campaigns_paused", result.RowsAffected,
		)
	}
}
