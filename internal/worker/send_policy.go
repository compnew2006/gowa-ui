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

	settings := org.Settings
	policy.StrictEnabled = readOrganizationBool(settings, organizationSettingStrictSendingRestrictionsEnabled, false)
	policy.OutboundMode = readOrganizationString(settings, organizationSettingOutboundMode, policy.OutboundMode)
	policy.ApplyToSystem = readOrganizationBool(settings, organizationSettingStrictSendingApplyToSystem, true)
	policy.CampaignDraftOnly = readOrganizationBool(settings, organizationSettingCampaignDraftOnly, false)
	return policy, nil
}

func (w *Worker) contactHasIncomingHistory(orgID, contactID uuid.UUID) (bool, error) {
	var count int64
	if err := w.DB.Model(&models.Message{}).
		Where("organization_id = ? AND contact_id = ? AND direction = ?", orgID, contactID, models.DirectionIncoming).
		Limit(1).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
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

func readOrganizationBool(settings models.JSONB, key string, fallback bool) bool {
	if settings == nil {
		return fallback
	}
	value, ok := settings[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		default:
			return fallback
		}
	default:
		return fallback
	}
}

func readOrganizationString(settings models.JSONB, key, fallback string) string {
	if settings == nil {
		return fallback
	}
	value, ok := settings[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return fallback
		}
		return trimmed
	default:
		return fallback
	}
}
