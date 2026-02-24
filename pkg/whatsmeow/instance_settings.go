package whatsmeow

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

const (
	// InstanceSettingAutoSyncHistory toggles WhatsApp history ingestion on connect.
	InstanceSettingAutoSyncHistory = "auto_sync_history"
)

// EnsureInstanceSettingsDefaults injects default settings for unset keys.
func EnsureInstanceSettingsDefaults(settings models.JSONB) models.JSONB {
	normalized := cloneSettings(settings)
	if _, ok := normalized[InstanceSettingAutoSyncHistory]; !ok {
		normalized[InstanceSettingAutoSyncHistory] = true
	}
	normalized[InstanceSettingAutoRejectCalls] = NormalizeAutoRejectCallSettings(
		normalized[InstanceSettingAutoRejectCalls],
	).ToJSONB()
	normalized[InstanceSettingAutoCampaign] = NormalizeAutoCampaignSettings(
		normalized[InstanceSettingAutoCampaign],
	).ToJSONB()
	return normalized
}

// IsAutoSyncHistoryEnabled returns whether history sync is enabled in settings.
// Missing/invalid values default to true.
func IsAutoSyncHistoryEnabled(settings models.JSONB) bool {
	return boolSetting(settings, InstanceSettingAutoSyncHistory, true)
}

func cloneSettings(settings models.JSONB) models.JSONB {
	if settings == nil {
		return models.JSONB{}
	}
	cloned := make(models.JSONB, len(settings))
	for key, value := range settings {
		cloned[key] = value
	}
	return cloned
}

func boolSetting(settings models.JSONB, key string, fallback bool) bool {
	if settings == nil {
		return fallback
	}
	raw, ok := settings[key]
	if !ok || raw == nil {
		return fallback
	}

	switch value := raw.(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	case int:
		return value != 0
	case int64:
		return value != 0
	case float64:
		return value != 0
	}

	return fallback
}

func (cm *ConnectionManager) isHistorySyncEnabled(ctx context.Context, instanceID uuid.UUID) bool {
	var instance models.WhatsAppInstance
	if err := cm.db.WithContext(ctx).
		Select("id", "settings").
		Where("id = ?", instanceID).
		First(&instance).Error; err != nil {
		cm.logger.Warn("Failed to load instance settings for history sync; using default", "instance_id", instanceID, "error", err)
		return true
	}

	return IsAutoSyncHistoryEnabled(instance.Settings)
}

func (cm *ConnectionManager) loadAutoRejectCallSettings(ctx context.Context, instanceID uuid.UUID) (AutoRejectCallSettings, error) {
	settings := DefaultAutoRejectCallSettings()
	if cm == nil || cm.db == nil {
		return settings, fmt.Errorf("database is not initialized")
	}

	var instance models.WhatsAppInstance
	if err := cm.db.WithContext(ctx).
		Select("id", "settings").
		Where("id = ?", instanceID).
		First(&instance).Error; err != nil {
		return settings, err
	}

	return AutoRejectCallSettingsFromSettings(instance.Settings), nil
}

// ValidateInstanceSettings validates known instance setting domains.
func ValidateInstanceSettings(settings models.JSONB) error {
	if settings == nil {
		return nil
	}

	if err := ValidateAutoRejectCallSettings(settings[InstanceSettingAutoRejectCalls]); err != nil {
		return fmt.Errorf("invalid %s: %w", InstanceSettingAutoRejectCalls, err)
	}
	if err := ValidateAutoCampaignSettings(settings[InstanceSettingAutoCampaign]); err != nil {
		return fmt.Errorf("invalid %s: %w", InstanceSettingAutoCampaign, err)
	}

	return nil
}
