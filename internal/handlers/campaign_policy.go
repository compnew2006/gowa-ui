package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const strictCampaignDelayFloorSeconds = 10

type campaignPolicyViolationError struct {
	*reasonedError
}

func newCampaignPolicyViolationError(message, reasonCode string) *campaignPolicyViolationError {
	return &campaignPolicyViolationError{reasonedError: newReasonedError(message, reasonCode, "campaign policy violation")}
}

func (e *campaignPolicyViolationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.reasonedError
}

func asCampaignPolicyViolation(err error) (string, string, bool) {
	re, ok := asReasonedError(err)
	if !ok {
		return "", "", false
	}
	return re.Error(), strings.TrimSpace(re.reasonCode), true
}

func (a *App) campaignDelayFloorSeconds(orgID uuid.UUID) int {
	_ = orgID
	return strictCampaignDelayFloorSeconds
}

func validateCampaignDelayFloor(minDelaySeconds, maxDelaySeconds, floorSeconds int) error {
	if floorSeconds <= 0 {
		return nil
	}
	if minDelaySeconds < floorSeconds || maxDelaySeconds < floorSeconds {
		return fmt.Errorf("campaign delay must be at least %d seconds in strict inbound-only mode", floorSeconds)
	}
	return nil
}

func (a *App) enforceCampaignStartPolicy(orgID uuid.UUID, sender string) error {
	policy := a.loadOrganizationStrictPolicySettings(orgID)
	if policy.CampaignDraftOnly {
		return newCampaignPolicyViolationError("Direct campaign execution is disabled by organization policy. Keep the campaign as draft.", ReasonCodePolicyDraftOnly)
	}

	if !a.isWhatsmeowProvider() {
		return nil
	}

	instanceID, err := uuid.Parse(strings.TrimSpace(sender))
	if err != nil {
		return newCampaignPolicyViolationError("Campaign sender instance is invalid", ReasonCodeInstanceNotConn)
	}

	var instance models.WhatsAppInstance
	if err := a.DB.
		Select("id", "organization_id", "status", "send_blocked_until", "send_block_reason").
		Where("id = ? AND organization_id = ?", instanceID, orgID).
		First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return newCampaignPolicyViolationError("Campaign sender instance was not found", ReasonCodeInstanceNotConn)
		}
		return err
	}

	if instance.Status != models.InstanceStatusConnected {
		return newCampaignPolicyViolationError("Campaign sender instance is not connected", ReasonCodeInstanceNotConn)
	}

	if blockReason := instanceSendBlockReason(&instance); blockReason != "" {
		return newCampaignPolicyViolationError(blockReason, ReasonCodeInstanceBlocked)
	}

	return nil
}
