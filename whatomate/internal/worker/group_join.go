package worker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HandleGroupJoinJob processes a single group join invite link from a campaign.
func (w *Worker) HandleGroupJoinJob(ctx context.Context, job *queue.GroupJoinJob) error {
	if job == nil {
		return queue.NewPermanentError(fmt.Errorf("group join job is nil"))
	}
	if job.CampaignID == uuid.Nil {
		return queue.NewPermanentError(fmt.Errorf("group join job missing campaign_id"))
	}
	if job.RecipientID == uuid.Nil {
		return queue.NewPermanentError(fmt.Errorf("group join job missing recipient_id"))
	}
	if job.OrganizationID == uuid.Nil {
		return queue.NewPermanentError(fmt.Errorf("group join job missing organization_id"))
	}
	if strings.TrimSpace(job.InviteLink) == "" {
		return queue.NewPermanentError(fmt.Errorf("group join job missing invite_link"))
	}
	if strings.TrimSpace(job.InstanceID) == "" {
		return queue.NewPermanentError(fmt.Errorf("group join job missing instance_id"))
	}

	// Load the recipient record.
	var recipient models.GroupJoinRecipient
	if err := w.DB.Where("id = ? AND campaign_id = ?", job.RecipientID, job.CampaignID).First(&recipient).Error; err != nil {
		return fmt.Errorf("failed to load group join recipient: %w", err)
	}

	// Skip if already processed.
	if recipient.Status != models.GroupJoinRecipientPending {
		w.Log.Info("Skipping already-processed group join recipient",
			"recipient_id", job.RecipientID,
			"campaign_id", job.CampaignID,
			"status", recipient.Status,
		)
		return nil
	}

	// Load campaign to check status.
	var campaign models.GroupJoinCampaign
	if err := w.DB.Where("id = ?", job.CampaignID).First(&campaign).Error; err != nil {
		return fmt.Errorf("failed to load group join campaign: %w", err)
	}

	// Skip if campaign is paused or cancelled.
	if campaign.Status == models.GroupJoinStatusPaused || campaign.Status == models.GroupJoinStatusCancelled {
		w.Log.Info("Group join campaign not active, skipping recipient",
			"campaign_id", job.CampaignID,
			"status", campaign.Status,
			"recipient_id", job.RecipientID,
		)
		return nil
	}

	// Validate the instance is connected.
	if failureReason, validationErr := w.validateWhatsmeowCampaignInstance(job.OrganizationID, job.InstanceID); validationErr != nil {
		return fmt.Errorf("failed to validate instance: %w", validationErr)
	} else if failureReason != "" {
		w.updateGroupJoinRecipientStatus(job.RecipientID, models.GroupJoinRecipientFailed, "", failureReason)
		w.incrementGroupJoinCampaignCount(job.CampaignID, "failed_count")
		return nil
	}

	// Attempt to join the group.
	joinProvider, ok := w.MessageProvider.(provider.JoinGroupProvider)
	if !ok {
		reason := "Message provider does not support group joining (requires whatsmeow)"
		w.updateGroupJoinRecipientStatus(job.RecipientID, models.GroupJoinRecipientFailed, "", reason)
		w.incrementGroupJoinCampaignCount(job.CampaignID, "failed_count")
		return nil
	}

	// Apply delay based on speed mode.
	delay := w.groupJoinDelay(campaign.Speed)
	if delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	groupJID, err := joinProvider.JoinGroupWithLink(ctx, job.InstanceID, job.InviteLink)
	if err != nil {
		errMsg := err.Error()
		w.Log.Error("Failed to join group", "error", errMsg, "invite_link", job.InviteLink, "campaign_id", job.CampaignID)

		// Check for common permanent errors.
		if isPermanentJoinError(errMsg) {
			w.updateGroupJoinRecipientStatus(job.RecipientID, models.GroupJoinRecipientFailed, "", errMsg)
			w.incrementGroupJoinCampaignCount(job.CampaignID, "failed_count")
			return nil
		}

		// Retryable error.
		return fmt.Errorf("join group: %w", err)
	}

	// Success — update recipient.
	now := time.Now()
	if err := w.DB.Model(&models.GroupJoinRecipient{}).Where("id = ?", job.RecipientID).Updates(map[string]interface{}{
		"status":      models.GroupJoinRecipientJoined,
		"group_jid":   groupJID,
		"processed_at": now,
	}).Error; err != nil {
		w.Log.Error("Failed to update group join recipient status", "error", err, "recipient_id", job.RecipientID)
	}

	// Increment joined count.
	w.incrementGroupJoinCampaignCount(job.CampaignID, "joined_count")

	w.Log.Info("Group joined successfully",
		"group_jid", groupJID,
		"invite_link", job.InviteLink,
		"campaign_id", job.CampaignID,
	)

	// Check if campaign is complete.
	w.checkGroupJoinCampaignCompletion(ctx, job.CampaignID)

	return nil
}

// updateGroupJoinRecipientStatus updates a group join recipient's status.
func (w *Worker) updateGroupJoinRecipientStatus(recipientID interface{}, status models.GroupJoinRecipientStatus, groupJID, errorMsg string) {
	updates := map[string]interface{}{
		"status":        status,
		"error_message": errorMsg,
	}
	if groupJID != "" {
		updates["group_jid"] = groupJID
	}
	if status == models.GroupJoinRecipientJoined || status == models.GroupJoinRecipientFailed || status == models.GroupJoinRecipientSkipped {
		updates["processed_at"] = time.Now()
	}
	if err := w.DB.Model(&models.GroupJoinRecipient{}).Where("id = ?", recipientID).Updates(updates).Error; err != nil {
		w.Log.Error("Failed to update group join recipient status", "error", err, "recipient_id", recipientID, "status", status)
	}
}

// incrementGroupJoinCampaignCount increments a group join campaign counter atomically.
func (w *Worker) incrementGroupJoinCampaignCount(campaignID interface{}, column string) {
	if err := w.DB.Model(&models.GroupJoinCampaign{}).
		Where("id = ?", campaignID).
		Update(column, gorm.Expr(column+" + 1")).Error; err != nil {
		w.Log.Error("Failed to increment group join campaign count", "error", err, "campaign_id", campaignID, "column", column)
	}
}

// checkGroupJoinCampaignCompletion checks if all recipients are processed.
func (w *Worker) checkGroupJoinCampaignCompletion(ctx context.Context, campaignID interface{}) {
	var pendingCount int64
	if err := w.DB.Model(&models.GroupJoinRecipient{}).
		Where("campaign_id = ? AND status = ?", campaignID, models.GroupJoinRecipientPending).
		Count(&pendingCount).Error; err != nil {
		w.Log.Error("Failed to count pending group join recipients", "error", err, "campaign_id", campaignID)
		return
	}

	if pendingCount == 0 {
		now := time.Now()
		result := w.DB.Model(&models.GroupJoinCampaign{}).
			Where("id = ? AND status = ?", campaignID, models.GroupJoinStatusProcessing).
			Updates(map[string]interface{}{
				"status":       models.GroupJoinStatusCompleted,
				"completed_at": now,
			})
		if result.Error != nil {
			w.Log.Error("Failed to mark group join campaign as completed", "error", result.Error, "campaign_id", campaignID)
			return
		}
		if result.RowsAffected == 0 {
			return
		}

		var campaign models.GroupJoinCampaign
		if err := w.DB.Where("id = ?", campaignID).First(&campaign).Error; err != nil {
			w.Log.Error("Failed to load completed group join campaign", "error", err, "campaign_id", campaignID)
			return
		}

		w.Log.Info("Group join campaign completed",
			"campaign_id", campaignID,
			"joined", campaign.JoinedCount,
			"failed", campaign.FailedCount,
			"skipped", campaign.SkippedCount,
		)
	}
}

// groupJoinDelay returns the delay between join attempts based on speed mode.
func (w *Worker) groupJoinDelay(speed models.GroupJoinSpeed) time.Duration {
	switch speed {
	case models.GroupJoinSpeedFast:
		return 5 * time.Second
	case models.GroupJoinSpeedSlow:
		return 30 * time.Second
	default:
		return 30 * time.Second
	}
}

// isPermanentJoinError checks if a join error is permanent (non-retryable).
func isPermanentJoinError(errMsg string) bool {
	errMsg = strings.ToLower(errMsg)
	permanentPhrases := []string{
		"invalid invite",
		"invite link expired",
		"invite link not found",
		"group does not exist",
		"you are already in this group",
		"banned",
		"not allowed",
		"invalid code",
	}
	for _, phrase := range permanentPhrases {
		if strings.Contains(errMsg, phrase) {
			return true
		}
	}
	return false
}
