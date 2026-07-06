package database

import (
	"fmt"

	"github.com/compnew2006/whatomate/internal/models"
	waManager "github.com/compnew2006/whatomate/pkg/whatsmeow"
	"gorm.io/gorm"
)

// BackfillInstanceAssignedChatResetSettings migrates legacy organization-level
// assigned chat reset settings into per-instance settings.
func BackfillInstanceAssignedChatResetSettings(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if !db.Migrator().HasTable(&models.Organization{}) || !db.Migrator().HasTable(&models.WhatsAppInstance{}) {
		return nil
	}

	var organizations []models.Organization
	if err := db.Select("id", "settings").Find(&organizations).Error; err != nil {
		return fmt.Errorf("failed to load organizations for assigned chat reset backfill: %w", err)
	}

	for _, org := range organizations {
		if !waManager.HasAssignedChatResetSettings(org.Settings) {
			continue
		}

		legacy := waManager.AssignedChatResetSettingsFromSettings(org.Settings)

		var instances []models.WhatsAppInstance
		if err := db.Select("id", "organization_id", "settings").Where("organization_id = ?", org.ID).Find(&instances).Error; err != nil {
			return fmt.Errorf("failed to load instances for organization %s: %w", org.ID, err)
		}

		for _, instance := range instances {
			if waManager.HasAssignedChatResetSettings(instance.Settings) {
				continue
			}

			nextSettings := cloneJSONB(instance.Settings)
			nextSettings[waManager.InstanceSettingAssignedChatResetEnabled] = legacy.Enabled
			nextSettings[waManager.InstanceSettingAssignedChatResetMode] = string(legacy.Mode)
			nextSettings[waManager.InstanceSettingAssignedChatResetHour] = legacy.Hour
			if legacy.LastResetDate != "" {
				nextSettings[waManager.InstanceSettingAssignedChatResetLastDate] = legacy.LastResetDate
			}

			if err := db.Model(&models.WhatsAppInstance{}).
				Where("id = ?", instance.ID).
				Update("settings", nextSettings).
				Error; err != nil {
				return fmt.Errorf("failed to backfill assigned chat reset settings for instance %s: %w", instance.ID, err)
			}
		}

		nextOrgSettings := cloneJSONB(org.Settings)
		delete(nextOrgSettings, waManager.InstanceSettingAssignedChatResetEnabled)
		delete(nextOrgSettings, waManager.InstanceSettingAssignedChatResetMode)
		delete(nextOrgSettings, waManager.InstanceSettingAssignedChatResetHour)
		delete(nextOrgSettings, waManager.InstanceSettingAssignedChatResetLastDate)

		if err := db.Model(&models.Organization{}).
			Where("id = ?", org.ID).
			Update("settings", nextOrgSettings).
			Error; err != nil {
			return fmt.Errorf("failed to clear legacy assigned chat reset settings for organization %s: %w", org.ID, err)
		}
	}

	return nil
}

func cloneJSONB(settings models.JSONB) models.JSONB {
	if settings == nil {
		return models.JSONB{}
	}

	cloned := make(models.JSONB, len(settings))
	for key, value := range settings {
		cloned[key] = value
	}
	return cloned
}
