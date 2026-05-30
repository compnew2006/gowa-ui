package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/google/uuid"
)

func (w *Worker) HandleMessageExtractionJob(ctx context.Context, job *queue.MessageExtractionJob) error {
	if job.CampaignID == uuid.Nil {
		return fmt.Errorf("message extraction job missing campaign_id")
	}
	if job.OrganizationID == uuid.Nil {
		return fmt.Errorf("message extraction job missing organization_id")
	}

	var campaign models.MessageExtractionCampaign
	if err := w.DB.Where("id = ? AND organization_id = ?", job.CampaignID, job.OrganizationID).First(&campaign).Error; err != nil {
		return fmt.Errorf("failed to load message extraction campaign: %w", err)
	}

	if campaign.Status == models.MsgExtractionStatusPaused || campaign.Status == models.MsgExtractionStatusCancelled {
		w.Log.Info("Message extraction campaign no longer active, skipping", "campaign_id", campaign.ID, "status", campaign.Status)
		return nil
	}

	if w.whatsmeowMgr == nil {
		return w.failMessageExtractionCampaign(campaign.ID, "WhatsApp manager not initialized")
	}

	client := w.whatsmeowMgr.GetClient(job.InstanceID)
	if client == nil {
		return w.failMessageExtractionCampaign(campaign.ID, "WhatsApp instance is not connected")
	}

	w.Log.Info("Starting message extraction", "campaign_id", campaign.ID, "instance_id", job.InstanceID)

	msg := client.BuildHistorySyncRequest(nil, 100)
	if _, err := client.SendPeerMessage(ctx, msg); err != nil {
		return w.failMessageExtractionCampaign(campaign.ID, fmt.Sprintf("Failed to trigger history sync: %v", err))
	}

	time.Sleep(5 * time.Second)

	var contacts []models.Contact
	if err := w.DB.Where("organization_id = ? AND instance_id = ?", job.OrganizationID, job.InstanceID).
		Find(&contacts).Error; err != nil {
		return w.failMessageExtractionCampaign(campaign.ID, fmt.Sprintf("Failed to load contacts: %v", err))
	}

	var messages []models.Message
	if err := w.DB.Where("organization_id = ? AND instance_id = ?", job.OrganizationID, job.InstanceID).
		Order("created_at DESC").
		Find(&messages).Error; err != nil {
		return w.failMessageExtractionCampaign(campaign.ID, fmt.Sprintf("Failed to load messages: %v", err))
	}

	chatMap := make(map[string]*models.MessageExtractionResult)
	for _, msg := range messages {
		jid := msg.WhatsAppMessageID
		if msg.ContactID != uuid.Nil {
			jid = msg.ContactID.String()
		}
		if jid == "" {
			continue
		}

		if existing, ok := chatMap[jid]; ok {
			existing.UnreadCount++
			continue
		}

		result := models.MessageExtractionResult{
			CampaignID:    campaign.ID,
			ChatJID:       jid,
			PhoneNumber:   "",
			IsMe:          msg.Direction == "outgoing",
			LastMessageAt: &msg.CreatedAt,
			Status:        models.MsgExtractionResultPending,
		}

		chatMap[jid] = &result
	}

	var contactMap map[string]models.Contact
	contactMap = make(map[string]models.Contact)
	for _, c := range contacts {
		if c.InstanceID != nil && *c.InstanceID == job.InstanceID {
			contactMap[c.ID.String()] = c
		}
	}

	for jid, result := range chatMap {
		if c, ok := contactMap[jid]; ok {
			result.PhoneNumber = c.PhoneNumber
			result.ProfileName = c.ProfileName
		}
		result.Status = models.MsgExtractionResultExtracted
	}

	results := make([]models.MessageExtractionResult, 0, len(chatMap))
	for _, result := range chatMap {
		results = append(results, *result)
	}

	if len(results) > 0 {
		if err := w.DB.CreateInBatches(&results, 100).Error; err != nil {
			return w.failMessageExtractionCampaign(campaign.ID, fmt.Sprintf("Failed to save results: %v", err))
		}
	}

	now := time.Now()
	w.DB.Model(&models.MessageExtractionCampaign{}).Where("id = ?", campaign.ID).Updates(map[string]interface{}{
		"status":          models.MsgExtractionStatusCompleted,
		"total_chats":     len(results),
		"extracted_count": len(results),
		"completed_at":    now,
	})

	w.Log.Info("Message extraction completed", "campaign_id", campaign.ID, "total_chats", len(results))
	return nil
}

func (w *Worker) failMessageExtractionCampaign(campaignID uuid.UUID, errMsg string) error {
	w.DB.Model(&models.MessageExtractionCampaign{}).Where("id = ?", campaignID).Updates(map[string]interface{}{
		"status":       models.MsgExtractionStatusFailed,
		"failed_count": 0,
	})
	w.Log.Error("Message extraction campaign failed", "campaign_id", campaignID, "error", errMsg)
	return fmt.Errorf("message extraction campaign %s failed: %s", campaignID, errMsg)
}
