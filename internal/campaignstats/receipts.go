package campaignstats

import (
	"context"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/google/uuid"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ApplyMessageReceipt updates campaign recipient state and aggregate counters for
// an already-progressed outbound campaign message.
func ApplyMessageReceipt(ctx context.Context, db *gorm.DB, publisher *queue.Publisher, log logf.Logger, message *models.Message, newStatus models.MessageStatus) {
	if db == nil || message == nil || message.Metadata == nil {
		return
	}
	if newStatus != models.MessageStatusDelivered && newStatus != models.MessageStatusRead && newStatus != models.MessageStatusFailed {
		return
	}

	campaignIDRaw, ok := message.Metadata["campaign_id"].(string)
	if !ok || strings.TrimSpace(campaignIDRaw) == "" {
		return
	}
	campaignID, err := uuid.Parse(strings.TrimSpace(campaignIDRaw))
	if err != nil {
		log.Error("Invalid campaign ID for receipt update", "campaign_id", campaignIDRaw, "message_id", message.ID)
		return
	}

	var campaign models.BulkMessageCampaign
	if err := db.WithContext(ctx).
		Select("id", "organization_id").
		Where("id = ? AND organization_id = ?", campaignID, message.OrganizationID).
		First(&campaign).Error; err != nil {
		log.Debug("Campaign not found for receipt update", "campaign_id", campaignID, "organization_id", message.OrganizationID, "error", err)
		return
	}

	recipientQuery := db.WithContext(ctx).
		Model(&models.BulkMessageRecipient{}).
		Where("campaign_id = ? AND status NOT IN ?", campaignID, recipientStatusesAtOrAbove(newStatus)).
		Where("whats_app_message_id = ?", strings.TrimSpace(message.WhatsAppMessageID))
	if message.ID != uuid.Nil {
		recipientQuery = db.WithContext(ctx).
			Model(&models.BulkMessageRecipient{}).
			Where("campaign_id = ? AND status NOT IN ?", campaignID, recipientStatusesAtOrAbove(newStatus)).
			Where("whats_app_message_id = ? OR message_id = ?", strings.TrimSpace(message.WhatsAppMessageID), message.ID)
	}

	now := time.Now().UTC()
	recipientUpdates := map[string]interface{}{
		"status": newStatus,
	}
	switch newStatus {
	case models.MessageStatusDelivered:
		recipientUpdates["delivered_at"] = now
	case models.MessageStatusRead:
		recipientUpdates["read_at"] = now
	case models.MessageStatusFailed:
		if strings.TrimSpace(message.ErrorMessage) != "" {
			recipientUpdates["error_message"] = strings.TrimSpace(message.ErrorMessage)
		}
	}

	result := recipientQuery.Updates(recipientUpdates)
	if result.Error != nil {
		log.Error("Failed to update campaign recipient receipt status", "error", result.Error, "campaign_id", campaignID, "message_id", message.ID, "status", newStatus)
		return
	}
	if result.RowsAffected == 0 {
		return
	}

	column := counterColumn(newStatus)
	if column == "" {
		return
	}

	var updatedCampaign models.BulkMessageCampaign
	updatedCampaign.ID = campaignID
	updateResult := db.WithContext(ctx).
		Model(&updatedCampaign).
		Clauses(clause.Returning{}).
		Where("organization_id = ?", message.OrganizationID).
		Update(column, gorm.Expr(column+" + ?", result.RowsAffected))
	if updateResult.Error != nil {
		log.Error("Failed to increment campaign receipt stat", "error", updateResult.Error, "campaign_id", campaignID, "column", column)
		return
	}
	if updateResult.RowsAffected == 0 || publisher == nil {
		return
	}

	_ = publisher.PublishCampaignStats(ctx, &queue.CampaignStatsUpdate{
		CampaignID:     campaignID.String(),
		OrganizationID: message.OrganizationID,
		Status:         updatedCampaign.Status,
		SentCount:      updatedCampaign.SentCount,
		DeliveredCount: updatedCampaign.DeliveredCount,
		ReadCount:      updatedCampaign.ReadCount,
		FailedCount:    updatedCampaign.FailedCount,
	})
}

func recipientStatusesAtOrAbove(status models.MessageStatus) []models.MessageStatus {
	switch status {
	case models.MessageStatusRead:
		return []models.MessageStatus{models.MessageStatusRead}
	case models.MessageStatusDelivered:
		return []models.MessageStatus{models.MessageStatusDelivered, models.MessageStatusRead}
	case models.MessageStatusFailed:
		return []models.MessageStatus{models.MessageStatusFailed}
	default:
		return nil
	}
}

func counterColumn(status models.MessageStatus) string {
	switch status {
	case models.MessageStatusDelivered:
		return "delivered_count"
	case models.MessageStatusRead:
		return "read_count"
	case models.MessageStatusFailed:
		return "failed_count"
	default:
		return ""
	}
}
