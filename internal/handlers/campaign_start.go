package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type campaignStartErrorKind string

const (
	campaignStartBadRequest campaignStartErrorKind = "bad_request"
	campaignStartForbidden  campaignStartErrorKind = "forbidden"
	campaignStartConflict   campaignStartErrorKind = "conflict"
	campaignStartInternal   campaignStartErrorKind = "internal"
)

type campaignStartError struct {
	kind       campaignStartErrorKind
	message    string
	reasonCode string
	err        error
}

func (e *campaignStartError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.message) != "" {
		return e.message
	}
	if e.err != nil {
		return e.err.Error()
	}
	return "failed to start campaign"
}

func (e *campaignStartError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

type campaignStartResult struct {
	status        models.CampaignStatus
	enqueuedCount int
}

// StartCampaignByID starts a campaign without depending on HTTP request objects.
func (a *App) StartCampaignByID(ctx context.Context, db *gorm.DB, orgID, campaignID uuid.UUID) (*campaignStartResult, error) {
	if db == nil {
		db = a.DB
	}
	if a == nil || db == nil || a.Queue == nil {
		return nil, &campaignStartError{kind: campaignStartInternal, message: "Campaign queue is not initialized"}
	}

	var campaign models.BulkMessageCampaign
	if err := db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", campaignID, orgID).
		First(&campaign).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &campaignStartError{kind: campaignStartBadRequest, message: "Campaign not found", err: err}
		}
		return nil, &campaignStartError{kind: campaignStartInternal, message: "Failed to load campaign", err: err}
	}

	originalStatus := campaign.Status
	if originalStatus != models.CampaignStatusDraft && originalStatus != models.CampaignStatusScheduled && originalStatus != models.CampaignStatusPaused {
		return nil, &campaignStartError{kind: campaignStartBadRequest, message: "Campaign cannot be started in current state"}
	}
	if err := a.enforceCampaignStartPolicy(orgID, campaign.WhatsAppAccount); err != nil {
		if message, reasonCode, ok := asCampaignPolicyViolation(err); ok {
			return nil, &campaignStartError{kind: campaignStartForbidden, message: message, reasonCode: reasonCode, err: err}
		}
		a.Log.Error("Failed to validate campaign start policy", "error", err, "campaign_id", campaignID, "organization_id", orgID)
		return nil, &campaignStartError{kind: campaignStartInternal, message: "Failed to validate campaign policy", err: err}
	}
	if err := validateCampaignDelayFloor(campaign.MinDelaySeconds, campaign.MaxDelaySeconds, a.campaignDelayFloorSeconds(orgID)); err != nil {
		return nil, &campaignStartError{kind: campaignStartBadRequest, message: err.Error(), err: err}
	}

	var recipients []models.BulkMessageRecipient
	if err := db.WithContext(ctx).
		Where("campaign_id = ? AND status = ?", campaignID, models.MessageStatusPending).
		Find(&recipients).Error; err != nil {
		a.Log.Error("Failed to load recipients", "error", err, "campaign_id", campaignID)
		return nil, &campaignStartError{kind: campaignStartInternal, message: "Failed to load recipients", err: err}
	}
	if len(recipients) == 0 {
		return nil, &campaignStartError{kind: campaignStartBadRequest, message: "Campaign has no pending recipients"}
	}
	if blockedCount, err := a.countInboundPolicyViolationsForRecipients(orgID, recipients); err != nil {
		a.Log.Error("Failed to validate campaign recipients against strict inbound policy", "campaign_id", campaignID, "error", err)
		return nil, &campaignStartError{kind: campaignStartInternal, message: "Failed to validate campaign recipients", err: err}
	} else if blockedCount > 0 && a.shouldEnforceInboundOnlyForSystemSends(orgID) {
		return nil, &campaignStartError{
			kind:       campaignStartForbidden,
			message:    fmt.Sprintf("Campaign contains %d recipient(s) without inbound history in strict inbound-only mode", blockedCount),
			reasonCode: ReasonCodePolicyNoInbound,
		}
	}

	if campaign.TemplateID != uuid.Nil {
		var template models.Template
		if err := db.WithContext(ctx).
			Select("id").
			Where("id = ? AND organization_id = ?", campaign.TemplateID, orgID).
			First(&template).Error; err != nil {
			return nil, &campaignStartError{kind: campaignStartBadRequest, message: "Campaign template no longer exists", err: err}
		}
	}

	now := time.Now().UTC()
	result := db.WithContext(ctx).
		Model(&models.BulkMessageCampaign{}).
		Where("id = ? AND organization_id = ? AND status = ?", campaignID, orgID, originalStatus).
		Updates(map[string]any{
			"status":     models.CampaignStatusProcessing,
			"started_at": now,
		})
	if result.Error != nil {
		a.Log.Error("Failed to start campaign", "error", result.Error, "campaign_id", campaignID)
		return nil, &campaignStartError{kind: campaignStartInternal, message: "Failed to start campaign", err: result.Error}
	}
	if result.RowsAffected == 0 {
		return nil, &campaignStartError{kind: campaignStartConflict, message: "Campaign is already being started"}
	}

	jobs := make([]*queue.RecipientJob, len(recipients))
	for i, recipient := range recipients {
		jobs[i] = &queue.RecipientJob{
			CampaignID:     campaignID,
			RecipientID:    recipient.ID,
			OrganizationID: orgID,
			PhoneNumber:    recipient.PhoneNumber,
			RecipientName:  recipient.RecipientName,
			TemplateParams: recipient.TemplateParams,
		}
	}
	if err := a.Queue.EnqueueRecipients(ctx, jobs); err != nil {
		a.Log.Error("Failed to enqueue campaign recipients", "error", err, "campaign_id", campaignID, "organization_id", orgID)
		rollback := db.WithContext(ctx).
			Model(&models.BulkMessageCampaign{}).
			Where("id = ? AND organization_id = ? AND status = ?", campaignID, orgID, models.CampaignStatusProcessing).
			Updates(map[string]any{
				"status":     originalStatus,
				"started_at": nil,
			})
		if rollback.Error != nil {
			a.Log.Error("Failed to rollback campaign status after enqueue failure", "error", rollback.Error, "campaign_id", campaignID)
		}
		return nil, &campaignStartError{kind: campaignStartInternal, message: "Failed to queue recipients", err: err}
	}

	return &campaignStartResult{
		status:        models.CampaignStatusProcessing,
		enqueuedCount: len(jobs),
	}, nil
}
