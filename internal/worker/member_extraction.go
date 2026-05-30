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

func (w *Worker) HandleMemberExtractionJob(ctx context.Context, job *queue.MemberExtractionJob) error {
	if job.CampaignID == uuid.Nil {
		return fmt.Errorf("member extraction job missing campaign_id")
	}
	if job.OrganizationID == uuid.Nil {
		return fmt.Errorf("member extraction job missing organization_id")
	}
	if job.GroupJID == "" {
		return fmt.Errorf("member extraction job missing group_jid")
	}

	var campaign models.MemberExtractionCampaign
	if err := w.DB.Where("id = ? AND organization_id = ?", job.CampaignID, job.OrganizationID).First(&campaign).Error; err != nil {
		return fmt.Errorf("failed to load member extraction campaign: %w", err)
	}

	if campaign.Status == models.MemberExtractionStatusPaused || campaign.Status == models.MemberExtractionStatusCancelled {
		w.Log.Info("Member extraction campaign no longer active, skipping", "campaign_id", campaign.ID, "status", campaign.Status)
		return nil
	}

	if w.MessageProvider == nil {
		return w.failMemberExtractionCampaign(campaign.ID, "Message provider not initialized")
	}

	participantProvider, ok := w.MessageProvider.(provider.GroupParticipantProvider)
	if !ok {
		return w.failMemberExtractionCampaign(campaign.ID, "Provider does not support group participant operations")
	}

	w.Log.Info("Starting member extraction", "campaign_id", campaign.ID, "instance_id", job.InstanceID, "group_jid", job.GroupJID)

	participants, err := participantProvider.GetGroupParticipants(ctx, job.InstanceID.String(), job.GroupJID)
	if err != nil {
		return w.failMemberExtractionCampaign(campaign.ID, fmt.Sprintf("Failed to get group participants: %v", err))
	}

	results := make([]models.MemberExtractionResult, 0, len(participants))
	for _, p := range participants {
		results = append(results, models.MemberExtractionResult{
			CampaignID:     campaign.ID,
			ParticipantJID: p.JID,
			PhoneNumber:    p.PhoneNumber,
			IsAdmin:        p.IsAdmin,
			IsSuperAdmin:   p.IsSuperAdmin,
			Status:         models.MemberExtractionResultExtracted,
		})
	}

	if len(results) > 0 {
		if err := w.DB.CreateInBatches(&results, 100).Error; err != nil {
			return w.failMemberExtractionCampaign(campaign.ID, fmt.Sprintf("Failed to save results: %v", err))
		}
	}

	now := time.Now()
	w.DB.Model(&models.MemberExtractionCampaign{}).Where("id = ?", campaign.ID).Updates(map[string]interface{}{
		"status":          models.MemberExtractionStatusCompleted,
		"total_members":   len(participants),
		"extracted_count": len(results),
		"completed_at":    now,
	})

	w.Log.Info("Member extraction completed", "campaign_id", campaign.ID, "total_members", len(participants))
	return nil
}

func (w *Worker) failMemberExtractionCampaign(campaignID uuid.UUID, errMsg string) error {
	w.DB.Model(&models.MemberExtractionCampaign{}).Where("id = ?", campaignID).Updates(map[string]interface{}{
		"status": models.MemberExtractionStatusFailed,
	})
	w.Log.Error("Member extraction campaign failed", "campaign_id", campaignID, "error", errMsg)
	return fmt.Errorf("member extraction campaign %s failed: %s", campaignID, errMsg)
}
