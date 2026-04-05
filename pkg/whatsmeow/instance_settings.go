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
	// InstanceSettingChatCloseRatingEnabled toggles the close rating flow for one instance.
	InstanceSettingChatCloseRatingEnabled = "chat_close_rating_enabled"
	// InstanceSettingChatCloseRatingFollowupWindowMinutes controls the reply capture window per instance.
	InstanceSettingChatCloseRatingFollowupWindowMinutes = "chat_close_rating_followup_window_minutes"
	// InstanceSettingChatCloseRatingTemplates stores per-language close rating prompt templates for one instance.
	InstanceSettingChatCloseRatingTemplates = "chat_close_rating_templates"
)

// EnsureInstanceSettingsDefaults injects default settings for unset keys.
func EnsureInstanceSettingsDefaults(settings models.JSONB) models.JSONB {
	normalized := cloneSettings(settings)
	if _, ok := normalized[InstanceSettingAutoSyncHistory]; !ok {
		normalized[InstanceSettingAutoSyncHistory] = true
	}
	normalized = injectAutoDownloadIncomingMediaDefault(normalized)
	normalized[InstanceSettingAutoRejectCalls] = NormalizeAutoRejectCallSettings(
		normalized[InstanceSettingAutoRejectCalls],
	).ToJSONB()
	normalized[InstanceSettingAutoCampaign] = NormalizeAutoCampaignSettings(
		normalized[InstanceSettingAutoCampaign],
	).ToJSONB()
	normalized = injectChatCloseRatingDefaults(normalized)
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

func injectChatCloseRatingDefaults(settings models.JSONB) models.JSONB {
	normalized := cloneSettings(settings)
	if _, ok := normalized[InstanceSettingChatCloseRatingEnabled]; !ok {
		normalized[InstanceSettingChatCloseRatingEnabled] = true
	}
	if _, ok := normalized[InstanceSettingChatCloseRatingFollowupWindowMinutes]; !ok {
		normalized[InstanceSettingChatCloseRatingFollowupWindowMinutes] = defaultChatCloseRatingFollowupWindowMinutes
	}
	return normalized
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

func validateInstanceChatCloseRatingEnabledSetting(raw any) error {
	switch raw.(type) {
	case bool, string, int, int64, float64:
		return nil
	default:
		return fmt.Errorf("must be a boolean-like value")
	}
}

func parseIntLikeSetting(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed, true
		}
	}

	return 0, false
}

func validateInstanceChatCloseRatingFollowupWindowSetting(raw any) error {
	parsed, ok := parseIntLikeSetting(raw)
	if !ok {
		return fmt.Errorf("must be an integer between 1 and %d", maxChatCloseRatingFollowupWindowMinutes)
	}
	if parsed < 1 || parsed > maxChatCloseRatingFollowupWindowMinutes {
		return fmt.Errorf("must be between 1 and %d", maxChatCloseRatingFollowupWindowMinutes)
	}
	return nil
}

func validateInstanceChatCloseRatingTemplatesSetting(raw any) error {
	switch typed := raw.(type) {
	case map[string]string:
		return nil
	case map[string]any:
		for _, value := range typed {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("must contain only string template values")
			}
		}
		return nil
	case models.JSONB:
		for _, value := range typed {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("must contain only string template values")
			}
		}
		return nil
	default:
		return fmt.Errorf("must be an object keyed by language")
	}
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
	if err := ValidateAutoDownloadIncomingMediaSetting(settings[InstanceSettingAutoDownloadIncomingMedia]); err != nil {
		return fmt.Errorf("invalid %s: %w", InstanceSettingAutoDownloadIncomingMedia, err)
	}
	if raw, ok := settings[InstanceSettingChatCloseRatingEnabled]; ok {
		if err := validateInstanceChatCloseRatingEnabledSetting(raw); err != nil {
			return fmt.Errorf("invalid %s: %w", InstanceSettingChatCloseRatingEnabled, err)
		}
	}
	if raw, ok := settings[InstanceSettingChatCloseRatingFollowupWindowMinutes]; ok {
		if err := validateInstanceChatCloseRatingFollowupWindowSetting(raw); err != nil {
			return fmt.Errorf("invalid %s: %w", InstanceSettingChatCloseRatingFollowupWindowMinutes, err)
		}
	}
	if raw, ok := settings[InstanceSettingChatCloseRatingTemplates]; ok {
		if err := validateInstanceChatCloseRatingTemplatesSetting(raw); err != nil {
			return fmt.Errorf("invalid %s: %w", InstanceSettingChatCloseRatingTemplates, err)
		}
	}

	return nil
}
