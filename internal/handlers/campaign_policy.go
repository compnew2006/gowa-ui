package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const strictCampaignDelayFloorSeconds = 3

type campaignPolicyViolationError struct {
	message    string
	reasonCode string
}

func (e *campaignPolicyViolationError) Error() string {
	if e == nil {
		return "campaign policy violation"
	}
	if strings.TrimSpace(e.message) == "" {
		return "campaign policy violation"
	}
	return e.message
}

func asCampaignPolicyViolation(err error) (string, string, bool) {
	if err == nil {
		return "", "", false
	}
	var violation *campaignPolicyViolationError
	if !errors.As(err, &violation) {
		return "", "", false
	}
	return violation.Error(), strings.TrimSpace(violation.reasonCode), true
}

func (a *App) campaignDelayFloorSeconds(orgID uuid.UUID) int {
	policy := a.loadOrganizationStrictPolicySettings(orgID)
	if !policy.StrictEnabled {
		return 0
	}
	if normalizeOutboundMode(policy.OutboundMode) != organizationOutboundModeInboundOnly {
		return 0
	}
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
		return &campaignPolicyViolationError{
			message:    "Direct campaign execution is disabled by organization policy. Keep the campaign as draft.",
			reasonCode: ReasonCodePolicyDraftOnly,
		}
	}

	if !a.isWhatsmeowProvider() {
		return nil
	}

	instanceID, err := uuid.Parse(strings.TrimSpace(sender))
	if err != nil {
		return &campaignPolicyViolationError{
			message:    "Campaign sender instance is invalid",
			reasonCode: ReasonCodeInstanceNotConn,
		}
	}

	var instance models.WhatsAppInstance
	if err := a.DB.
		Select("id", "organization_id", "status", "send_blocked_until", "send_block_reason").
		Where("id = ? AND organization_id = ?", instanceID, orgID).
		First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &campaignPolicyViolationError{
				message:    "Campaign sender instance was not found",
				reasonCode: ReasonCodeInstanceNotConn,
			}
		}
		return err
	}

	if instance.Status != models.InstanceStatusConnected {
		return &campaignPolicyViolationError{
			message:    "Campaign sender instance is not connected",
			reasonCode: ReasonCodeInstanceNotConn,
		}
	}

	if blockReason := instanceSendBlockReason(&instance); blockReason != "" {
		return &campaignPolicyViolationError{
			message:    blockReason,
			reasonCode: ReasonCodeInstanceBlocked,
		}
	}

	return nil
}
