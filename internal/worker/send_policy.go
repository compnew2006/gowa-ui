package worker

import (
	"errors"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	organizationSettingStrictSendingRestrictionsEnabled = "strict_sending_restrictions_enabled"
	organizationSettingOutboundMode                     = "outbound_mode"
	organizationSettingStrictSendingApplyToSystem       = "strict_sending_apply_to_system"
	organizationSettingCampaignDraftOnly                = "campaign_draft_only"
	organizationOutboundModeInboundOnly                 = "inbound_only"
)

type organizationSendPolicy struct {
	StrictEnabled     bool
	OutboundMode      string
	ApplyToSystem     bool
	CampaignDraftOnly bool
}

func (p organizationSendPolicy) ShouldEnforceInboundOnlyForSystemSends() bool {
	if !p.StrictEnabled || !p.ApplyToSystem {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(p.OutboundMode), organizationOutboundModeInboundOnly)
}

func (w *Worker) loadOrganizationSendPolicy(orgID uuid.UUID) (organizationSendPolicy, error) {
	policy := organizationSendPolicy{
		StrictEnabled:     false,
		OutboundMode:      "mixed",
		ApplyToSystem:     true,
		CampaignDraftOnly: false,
	}
	if w == nil || w.DB == nil || orgID == uuid.Nil {
		return policy, nil
	}

	var org models.Organization
	if err := w.DB.Select("id", "settings").Where("id = ?", orgID).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return policy, nil
		}
		return policy, err
	}

	policy.StrictEnabled = org.Settings.Bool(organizationSettingStrictSendingRestrictionsEnabled, false)
	policy.OutboundMode = org.Settings.String(organizationSettingOutboundMode, policy.OutboundMode)
	policy.ApplyToSystem = org.Settings.Bool(organizationSettingStrictSendingApplyToSystem, true)
	policy.CampaignDraftOnly = org.Settings.Bool(organizationSettingCampaignDraftOnly, false)
	return policy, nil
}

func (w *Worker) validateWhatsmeowCampaignInstance(orgID uuid.UUID, rawInstanceID string) (string, error) {
	instanceIDRaw := strings.TrimSpace(rawInstanceID)
	if instanceIDRaw == "" {
		return "Campaign sender instance is missing", nil
	}

	instanceID, err := uuid.Parse(instanceIDRaw)
	if err != nil {
		return "Campaign sender instance is invalid", nil
	}

	var instance models.WhatsAppInstance
	if err := w.DB.
		Select("id", "organization_id", "status", "send_blocked_until", "send_block_reason").
		Where("id = ? AND organization_id = ?", instanceID, orgID).
		First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "Campaign sender instance was not found", nil
		}
		return "", err
	}

	if instance.Status != models.InstanceStatusConnected {
		return "Campaign sender instance is not connected", nil
	}

	if instance.SendBlockedUntil != nil && time.Now().UTC().Before(instance.SendBlockedUntil.UTC()) {
		reason := strings.TrimSpace(instance.SendBlockReason)
		if reason == "" {
			reason = "Campaign sender instance is temporarily blocked"
		}
		return reason, nil
	}

	return "", nil
}


