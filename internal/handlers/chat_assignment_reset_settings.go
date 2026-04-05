package handlers

import (
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	waManager "github.com/compnew2006/whatomate/pkg/whatsmeow"
)

const (
	organizationSettingAssignedChatResetEnabled  = waManager.InstanceSettingAssignedChatResetEnabled
	organizationSettingAssignedChatResetMode     = waManager.InstanceSettingAssignedChatResetMode
	organizationSettingAssignedChatResetHour     = waManager.InstanceSettingAssignedChatResetHour
	organizationSettingAssignedChatResetLastDate = waManager.InstanceSettingAssignedChatResetLastDate
)

// ChatAssignmentResetMode controls when assigned chats are reset back to pending.
type ChatAssignmentResetMode = waManager.AssignedChatResetMode

const (
	ChatAssignmentResetModeMidnight   = waManager.AssignedChatResetModeMidnight
	ChatAssignmentResetModeCustomHour = waManager.AssignedChatResetModeCustomHour
)

// ChatAssignmentResetSettings contains organization-level schedule preferences.
type ChatAssignmentResetSettings = waManager.AssignedChatResetSettings

func defaultChatAssignmentResetSettings() ChatAssignmentResetSettings {
	return waManager.DefaultAssignedChatResetSettings()
}

func isValidChatAssignmentResetMode(raw string) bool {
	return waManager.IsValidAssignedChatResetMode(raw)
}

func normalizeChatAssignmentResetMode(raw string) ChatAssignmentResetMode {
	return waManager.NormalizeAssignedChatResetMode(raw)
}

func isValidChatAssignmentResetHour(hour int) bool {
	return waManager.IsValidAssignedChatResetHour(hour)
}

func parseChatAssignmentResetHour(raw any) (int, bool) {
	return waManager.ParseAssignedChatResetHour(raw)
}

func readChatAssignmentResetSettings(settings models.JSONB) ChatAssignmentResetSettings {
	return waManager.AssignedChatResetSettingsFromSettings(settings)
}

func parseJSONBBool(raw any) (bool, bool) {
	return waManager.ParseBoolLike(raw)
}

func parseOrganizationTimezone(settings models.JSONB) string {
	if settings == nil {
		return "UTC"
	}
	if rawTZ, ok := settings["timezone"].(string); ok {
		tz := strings.TrimSpace(rawTZ)
		if tz != "" {
			return tz
		}
	}
	return "UTC"
}

func validateChatAssignmentResetInputs(mode *string, hour *int) error {
	return waManager.ValidateAssignedChatResetInputs(mode, hour)
}
