package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/google/uuid"
)

func (w *Worker) HandleGroupExtractionJob(ctx context.Context, job *queue.GroupExtractionJob) error {
	if job.CampaignID == uuid.Nil {
		return fmt.Errorf("group extraction job missing campaign_id")
	}
	if job.OrganizationID == uuid.Nil {
		return fmt.Errorf("group extraction job missing organization_id")
	}

	var campaign models.GroupExtractionCampaign
	if err := w.DB.Where("id = ? AND organization_id = ?", job.CampaignID, job.OrganizationID).First(&campaign).Error; err != nil {
		return fmt.Errorf("failed to load group extraction campaign: %w", err)
	}

	if campaign.Status == models.GroupExtractionStatusPaused || campaign.Status == models.GroupExtractionStatusCancelled {
		w.Log.Info("Group extraction campaign no longer active, skipping", "campaign_id", campaign.ID, "status", campaign.Status)
		return nil
	}

	if w.MessageProvider == nil {
		return w.failGroupExtractionCampaign(campaign.ID, "Message provider not initialized")
	}

	groupProvider, ok := w.MessageProvider.(provider.GroupProvider)
	if !ok {
		return w.failGroupExtractionCampaign(campaign.ID, "Provider does not support group operations")
	}

	w.Log.Info("Starting group extraction", "campaign_id", campaign.ID, "instance_id", job.InstanceID)

	groups, err := groupProvider.GetGroups(ctx, job.InstanceID.String())
	if err != nil {
		return w.failGroupExtractionCampaign(campaign.ID, fmt.Sprintf("Failed to get groups: %v", err))
	}

	results := make([]models.GroupExtractionResult, 0, len(groups))
	for _, g := range groups {
		results = append(results, models.GroupExtractionResult{
			CampaignID:       campaign.ID,
			GroupJID:         g.JID,
			GroupName:        g.Name,
			ParticipantCount: g.ParticipantCount,
			Status:           models.GroupExtractionResultExtracted,
		})
	}

	if len(results) > 0 {
		if err := w.DB.CreateInBatches(&results, 100).Error; err != nil {
			return w.failGroupExtractionCampaign(campaign.ID, fmt.Sprintf("Failed to save results: %v", err))
		}
	}

	now := time.Now()
	w.DB.Model(&models.GroupExtractionCampaign{}).Where("id = ?", campaign.ID).Updates(map[string]interface{}{
		"status":          models.GroupExtractionStatusCompleted,
		"total_groups":    len(groups),
		"extracted_count": len(results),
		"completed_at":    now,
	})

	w.Log.Info("Group extraction completed", "campaign_id", campaign.ID, "total_groups", len(groups))
	return nil
}

func (w *Worker) failGroupExtractionCampaign(campaignID uuid.UUID, errMsg string) error {
	w.DB.Model(&models.GroupExtractionCampaign{}).Where("id = ?", campaignID).Updates(map[string]interface{}{
		"status": models.GroupExtractionStatusFailed,
	})
	w.Log.Error("Group extraction campaign failed", "campaign_id", campaignID, "error", errMsg)
	return fmt.Errorf("group extraction campaign %s failed: %s", campaignID, errMsg)
}
