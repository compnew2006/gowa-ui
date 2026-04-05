package whatsmeow

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
)

const (
	// InstanceSettingAssignedChatResetEnabled toggles the daily reset worker per instance.
	InstanceSettingAssignedChatResetEnabled = "assigned_chat_reset_enabled"
	// InstanceSettingAssignedChatResetMode controls when the daily reset executes.
	InstanceSettingAssignedChatResetMode = "assigned_chat_reset_mode"
	// InstanceSettingAssignedChatResetHour stores the local hour for custom schedules.
	InstanceSettingAssignedChatResetHour = "assigned_chat_reset_hour"
	// InstanceSettingAssignedChatResetLastDate stores the most recent execution date for one instance.
	InstanceSettingAssignedChatResetLastDate = "assigned_chat_reset_last_date"
)

// AssignedChatResetMode controls when assigned chats are reset back to pending.
type AssignedChatResetMode string

const (
	AssignedChatResetModeMidnight   AssignedChatResetMode = "midnight"
	AssignedChatResetModeCustomHour AssignedChatResetMode = "custom_hour"
)

// AssignedChatResetSettings contains per-instance schedule preferences.
type AssignedChatResetSettings struct {
	Enabled       bool
	Mode          AssignedChatResetMode
	Hour          int
	LastResetDate string
}

func DefaultAssignedChatResetSettings() AssignedChatResetSettings {
	return AssignedChatResetSettings{
		Enabled: true,
		Mode:    AssignedChatResetModeMidnight,
		Hour:    0,
	}
}

func IsValidAssignedChatResetMode(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(AssignedChatResetModeMidnight), string(AssignedChatResetModeCustomHour):
		return true
	default:
		return false
	}
}

func NormalizeAssignedChatResetMode(raw string) AssignedChatResetMode {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == string(AssignedChatResetModeCustomHour) {
		return AssignedChatResetModeCustomHour
	}
	return AssignedChatResetModeMidnight
}

func IsValidAssignedChatResetHour(hour int) bool {
	return hour >= 0 && hour <= 23
}

func ParseAssignedChatResetHour(raw any) (int, bool) {
	const (
		maxInt = int(^uint(0) >> 1)
		minInt = -maxInt - 1
	)

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
		if v < int64(minInt) || v > int64(maxInt) {
			return 0, false
		}
		return int(v), true
	case uint:
		if uint64(v) > uint64(maxInt) {
			return 0, false
		}
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		if uint64(v) > uint64(maxInt) {
			return 0, false
		}
		return int(v), true
	case uint64:
		if v > uint64(maxInt) {
			return 0, false
		}
		return int(v), true
	case float64:
		if v < float64(minInt) || v > float64(maxInt) || math.Trunc(v) != v {
			return 0, false
		}
		return int(v), true
	case float32:
		floatV := float64(v)
		if floatV < float64(minInt) || floatV > float64(maxInt) || math.Trunc(floatV) != floatV {
			return 0, false
		}
		return int(v), true
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return 0, false
		}
		if parsed < int64(minInt) || parsed > int64(maxInt) {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func ParseBoolLike(raw any) (bool, bool) {
	switch v := raw.(type) {
	case bool:
		return v, true
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(v))
		switch trimmed {
		case "true", "1", "yes":
			return true, true
		case "false", "0", "no":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func ValidateAssignedChatResetInputs(mode *string, hour *int) error {
	if mode != nil && !IsValidAssignedChatResetMode(*mode) {
		return fmt.Errorf("assigned_chat_reset_mode must be one of: midnight, custom_hour")
	}
	if hour != nil && !IsValidAssignedChatResetHour(*hour) {
		return fmt.Errorf("assigned_chat_reset_hour must be between 0 and 23")
	}
	return nil
}

func AssignedChatResetSettingsFromSettings(settings models.JSONB) AssignedChatResetSettings {
	config := DefaultAssignedChatResetSettings()
	if settings == nil {
		return config
	}

	if rawEnabled, ok := settings[InstanceSettingAssignedChatResetEnabled]; ok {
		if parsedEnabled, parsed := ParseBoolLike(rawEnabled); parsed {
			config.Enabled = parsedEnabled
		}
	}

	if rawMode, ok := settings[InstanceSettingAssignedChatResetMode].(string); ok && strings.TrimSpace(rawMode) != "" {
		config.Mode = NormalizeAssignedChatResetMode(rawMode)
	}

	if rawHour, ok := settings[InstanceSettingAssignedChatResetHour]; ok {
		if parsedHour, parsed := ParseAssignedChatResetHour(rawHour); parsed && IsValidAssignedChatResetHour(parsedHour) {
			config.Hour = parsedHour
		}
	}

	if config.Mode == AssignedChatResetModeMidnight {
		config.Hour = 0
	}

	if rawLastDate, ok := settings[InstanceSettingAssignedChatResetLastDate].(string); ok {
		config.LastResetDate = strings.TrimSpace(rawLastDate)
	}

	return config
}

func HasAssignedChatResetSettings(settings models.JSONB) bool {
	if settings == nil {
		return false
	}

	for _, key := range []string{
		InstanceSettingAssignedChatResetEnabled,
		InstanceSettingAssignedChatResetMode,
		InstanceSettingAssignedChatResetHour,
		InstanceSettingAssignedChatResetLastDate,
	} {
		if _, ok := settings[key]; ok {
			return true
		}
	}

	return false
}

func injectAssignedChatResetDefaults(settings models.JSONB) models.JSONB {
	normalized := cloneSettings(settings)
	defaults := DefaultAssignedChatResetSettings()

	if _, ok := normalized[InstanceSettingAssignedChatResetEnabled]; !ok {
		normalized[InstanceSettingAssignedChatResetEnabled] = defaults.Enabled
	}
	if _, ok := normalized[InstanceSettingAssignedChatResetMode]; !ok {
		normalized[InstanceSettingAssignedChatResetMode] = string(defaults.Mode)
	}
	if _, ok := normalized[InstanceSettingAssignedChatResetHour]; !ok {
		normalized[InstanceSettingAssignedChatResetHour] = defaults.Hour
	}

	if NormalizeAssignedChatResetMode(stringFromAny(normalized[InstanceSettingAssignedChatResetMode])) == AssignedChatResetModeMidnight {
		normalized[InstanceSettingAssignedChatResetHour] = 0
	}

	return normalized
}

func validateAssignedChatResetSettings(settings models.JSONB) error {
	if settings == nil {
		return nil
	}

	var mode *string
	if raw, ok := settings[InstanceSettingAssignedChatResetMode]; ok {
		value, ok := raw.(string)
		if !ok {
			return fmt.Errorf("invalid %s: must be a string", InstanceSettingAssignedChatResetMode)
		}
		mode = &value
	}

	var hour *int
	if raw, ok := settings[InstanceSettingAssignedChatResetHour]; ok {
		parsedHour, parsed := ParseAssignedChatResetHour(raw)
		if !parsed {
			return fmt.Errorf("invalid %s: must be an integer between 0 and 23", InstanceSettingAssignedChatResetHour)
		}
		hour = &parsedHour
	}

	if err := ValidateAssignedChatResetInputs(mode, hour); err != nil {
		return err
	}

	if raw, ok := settings[InstanceSettingAssignedChatResetEnabled]; ok {
		if _, parsed := ParseBoolLike(raw); !parsed {
			return fmt.Errorf("invalid %s: must be a boolean-like value", InstanceSettingAssignedChatResetEnabled)
		}
	}

	if raw, ok := settings[InstanceSettingAssignedChatResetLastDate]; ok {
		if _, ok := raw.(string); !ok {
			return fmt.Errorf("invalid %s: must be a string", InstanceSettingAssignedChatResetLastDate)
		}
	}

	return nil
}
