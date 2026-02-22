package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
)

const (
	organizationSettingAssignedChatResetMode     = "assigned_chat_reset_mode"
	organizationSettingAssignedChatResetHour     = "assigned_chat_reset_hour"
	organizationSettingAssignedChatResetLastDate = "assigned_chat_reset_last_date"
)

// ChatAssignmentResetMode controls when assigned chats are reset back to pending.
type ChatAssignmentResetMode string

const (
	ChatAssignmentResetModeMidnight   ChatAssignmentResetMode = "midnight"
	ChatAssignmentResetModeCustomHour ChatAssignmentResetMode = "custom_hour"
)

// ChatAssignmentResetSettings contains organization-level schedule preferences.
type ChatAssignmentResetSettings struct {
	Mode          ChatAssignmentResetMode
	Hour          int
	LastResetDate string
}

func defaultChatAssignmentResetSettings() ChatAssignmentResetSettings {
	return ChatAssignmentResetSettings{
		Mode: ChatAssignmentResetModeMidnight,
		Hour: 0,
	}
}

func isValidChatAssignmentResetMode(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(ChatAssignmentResetModeMidnight), string(ChatAssignmentResetModeCustomHour):
		return true
	default:
		return false
	}
}

func normalizeChatAssignmentResetMode(raw string) ChatAssignmentResetMode {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == string(ChatAssignmentResetModeCustomHour) {
		return ChatAssignmentResetModeCustomHour
	}
	return ChatAssignmentResetModeMidnight
}

func isValidChatAssignmentResetHour(hour int) bool {
	return hour >= 0 && hour <= 23
}

func parseChatAssignmentResetHour(raw any) (int, bool) {
	switch v := raw.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func readChatAssignmentResetSettings(settings models.JSONB) ChatAssignmentResetSettings {
	config := defaultChatAssignmentResetSettings()
	if settings == nil {
		return config
	}

	if rawMode, ok := settings[organizationSettingAssignedChatResetMode].(string); ok && strings.TrimSpace(rawMode) != "" {
		config.Mode = normalizeChatAssignmentResetMode(rawMode)
	}

	if rawHour, ok := settings[organizationSettingAssignedChatResetHour]; ok {
		if parsedHour, parsed := parseChatAssignmentResetHour(rawHour); parsed && isValidChatAssignmentResetHour(parsedHour) {
			config.Hour = parsedHour
		}
	}

	if config.Mode == ChatAssignmentResetModeMidnight {
		config.Hour = 0
	}

	if rawLastDate, ok := settings[organizationSettingAssignedChatResetLastDate].(string); ok {
		config.LastResetDate = strings.TrimSpace(rawLastDate)
	}

	return config
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
	if mode != nil && !isValidChatAssignmentResetMode(*mode) {
		return fmt.Errorf("assigned_chat_reset_mode must be one of: midnight, custom_hour")
	}

	if hour != nil && !isValidChatAssignmentResetHour(*hour) {
		return fmt.Errorf("assigned_chat_reset_hour must be between 0 and 23")
	}

	return nil
}
