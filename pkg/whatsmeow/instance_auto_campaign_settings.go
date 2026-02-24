package whatsmeow

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
)

const (
	// InstanceSettingAutoCampaign stores per-instance recurring campaign generation settings.
	InstanceSettingAutoCampaign = "auto_campaign"

	AutoCampaignTargetStatusDraft = "draft"
	AutoCampaignTargetStatusRun   = "run"
)

// AutoCampaignSettings controls periodic campaign creation for an instance.
type AutoCampaignSettings struct {
	Enabled         bool       `json:"enabled"`
	NamePrefix      string     `json:"name_prefix"`
	Message         string     `json:"message"`
	IntervalDays    int        `json:"interval_days"`
	MinDelayMinutes int        `json:"min_delay_minutes"`
	MaxDelayMinutes int        `json:"max_delay_minutes"`
	TargetStatus    string     `json:"target_status"` // draft | run
	MediaLocalPath  string     `json:"media_local_path"`
	MediaMimeType   string     `json:"media_mime_type"`
	MediaFilename   string     `json:"media_filename"`
	LastGeneratedAt *time.Time `json:"last_generated_at,omitempty"`
}

// DefaultAutoCampaignSettings returns sane defaults for auto campaign generation.
func DefaultAutoCampaignSettings() AutoCampaignSettings {
	return AutoCampaignSettings{
		Enabled:         false,
		NamePrefix:      "",
		Message:         "",
		IntervalDays:    7,
		MinDelayMinutes: 0,
		MaxDelayMinutes: 0,
		TargetStatus:    AutoCampaignTargetStatusDraft,
	}
}

func (s AutoCampaignSettings) ToJSONB() models.JSONB {
	payload := models.JSONB{
		"enabled":           s.Enabled,
		"name_prefix":       strings.TrimSpace(s.NamePrefix),
		"message":           strings.TrimSpace(s.Message),
		"interval_days":     s.IntervalDays,
		"min_delay_minutes": s.MinDelayMinutes,
		"max_delay_minutes": s.MaxDelayMinutes,
		"target_status":     strings.TrimSpace(s.TargetStatus),
		"media_local_path":  strings.TrimSpace(s.MediaLocalPath),
		"media_mime_type":   strings.TrimSpace(s.MediaMimeType),
		"media_filename":    strings.TrimSpace(s.MediaFilename),
	}
	if s.LastGeneratedAt != nil {
		payload["last_generated_at"] = s.LastGeneratedAt.UTC().Format(time.RFC3339)
	}
	return payload
}

// AutoCampaignSettingsFromSettings extracts normalized auto campaign settings from instance settings.
func AutoCampaignSettingsFromSettings(settings models.JSONB) AutoCampaignSettings {
	if settings == nil {
		return DefaultAutoCampaignSettings()
	}
	return NormalizeAutoCampaignSettings(settings[InstanceSettingAutoCampaign])
}

// NormalizeAutoCampaignSettings applies defaults and sanitizes values.
func NormalizeAutoCampaignSettings(raw any) AutoCampaignSettings {
	normalized := DefaultAutoCampaignSettings()
	rawMap, ok := mapFromAny(raw)
	if !ok || rawMap == nil {
		return normalized
	}

	normalized.Enabled = boolFromAny(rawMap["enabled"], normalized.Enabled)
	normalized.NamePrefix = strings.TrimSpace(stringFromAny(rawMap["name_prefix"]))
	normalized.Message = strings.TrimSpace(stringFromAny(rawMap["message"]))

	interval := intFromAny(rawMap["interval_days"], normalized.IntervalDays)
	if interval > 0 {
		normalized.IntervalDays = interval
	}
	normalized.MinDelayMinutes = intFromAny(rawMap["min_delay_minutes"], normalized.MinDelayMinutes)
	normalized.MaxDelayMinutes = intFromAny(rawMap["max_delay_minutes"], normalized.MaxDelayMinutes)
	if normalized.MinDelayMinutes < 0 {
		normalized.MinDelayMinutes = 0
	}
	if normalized.MaxDelayMinutes < 0 {
		normalized.MaxDelayMinutes = 0
	}
	if normalized.MaxDelayMinutes < normalized.MinDelayMinutes {
		normalized.MaxDelayMinutes = normalized.MinDelayMinutes
	}

	switch strings.ToLower(strings.TrimSpace(stringFromAny(rawMap["target_status"]))) {
	case AutoCampaignTargetStatusRun:
		normalized.TargetStatus = AutoCampaignTargetStatusRun
	case AutoCampaignTargetStatusDraft:
		normalized.TargetStatus = AutoCampaignTargetStatusDraft
	}

	normalized.MediaLocalPath = strings.TrimSpace(stringFromAny(rawMap["media_local_path"]))
	normalized.MediaMimeType = strings.TrimSpace(stringFromAny(rawMap["media_mime_type"]))
	normalized.MediaFilename = strings.TrimSpace(stringFromAny(rawMap["media_filename"]))
	if normalized.MediaLocalPath == "" {
		normalized.MediaMimeType = ""
		normalized.MediaFilename = ""
	}

	normalized.LastGeneratedAt = parseAutoCampaignTimestamp(rawMap["last_generated_at"])
	return normalized
}

// ValidateAutoCampaignSettings validates auto campaign settings payload.
func ValidateAutoCampaignSettings(raw any) error {
	rawMap, _ := mapFromAny(raw)
	normalized := NormalizeAutoCampaignSettings(raw)

	if rawMap != nil {
		var minDelayRawValue *int
		var maxDelayRawValue *int

		if rawInterval, exists := rawMap["interval_days"]; exists {
			interval, ok := parsePositiveInt(rawInterval)
			if !ok || interval < 1 {
				return fmt.Errorf("auto campaign interval_days must be at least 1")
			}
		}

		if rawTarget, exists := rawMap["target_status"]; exists {
			target := strings.ToLower(strings.TrimSpace(stringFromAny(rawTarget)))
			if target != "" && target != AutoCampaignTargetStatusDraft && target != AutoCampaignTargetStatusRun {
				return fmt.Errorf("auto campaign target_status must be %q or %q", AutoCampaignTargetStatusDraft, AutoCampaignTargetStatusRun)
			}
		}

		if rawMinDelay, exists := rawMap["min_delay_minutes"]; exists {
			minDelay, ok := parsePositiveInt(rawMinDelay)
			if !ok || minDelay < 0 {
				return fmt.Errorf("auto campaign min_delay_minutes must be non-negative")
			}
			minDelayRawValue = &minDelay
		}

		if rawMaxDelay, exists := rawMap["max_delay_minutes"]; exists {
			maxDelay, ok := parsePositiveInt(rawMaxDelay)
			if !ok || maxDelay < 0 {
				return fmt.Errorf("auto campaign max_delay_minutes must be non-negative")
			}
			maxDelayRawValue = &maxDelay
		}

		if minDelayRawValue != nil && maxDelayRawValue != nil && *minDelayRawValue > *maxDelayRawValue {
			return fmt.Errorf("auto campaign min_delay_minutes cannot be greater than max_delay_minutes")
		}

		if rawTimestamp, exists := rawMap["last_generated_at"]; exists {
			if _, ok := parseOptionalTimestamp(rawTimestamp); !ok {
				return fmt.Errorf("auto campaign last_generated_at is invalid")
			}
		}
	}

	if normalized.IntervalDays < 1 {
		return fmt.Errorf("auto campaign interval_days must be at least 1")
	}
	if normalized.IntervalDays > 365 {
		return fmt.Errorf("auto campaign interval_days cannot exceed 365")
	}
	if normalized.MinDelayMinutes < 0 {
		return fmt.Errorf("auto campaign min_delay_minutes must be non-negative")
	}
	if normalized.MaxDelayMinutes < 0 {
		return fmt.Errorf("auto campaign max_delay_minutes must be non-negative")
	}
	if normalized.MaxDelayMinutes < normalized.MinDelayMinutes {
		return fmt.Errorf("auto campaign min_delay_minutes cannot be greater than max_delay_minutes")
	}

	if normalized.TargetStatus != AutoCampaignTargetStatusDraft && normalized.TargetStatus != AutoCampaignTargetStatusRun {
		return fmt.Errorf("auto campaign target_status must be %q or %q", AutoCampaignTargetStatusDraft, AutoCampaignTargetStatusRun)
	}

	if normalized.Enabled && strings.TrimSpace(normalized.Message) == "" {
		return fmt.Errorf("auto campaign message is required when enabled")
	}

	if normalized.MediaLocalPath != "" {
		cleaned := filepath.Clean(normalized.MediaLocalPath)
		if filepath.IsAbs(cleaned) || cleaned == "." || strings.HasPrefix(cleaned, "..") {
			return fmt.Errorf("auto campaign media_local_path is invalid")
		}
	}

	return nil
}

func parseAutoCampaignTimestamp(raw any) *time.Time {
	parsed, ok := parseOptionalTimestamp(raw)
	if !ok {
		return nil
	}
	return parsed
}

func parseOptionalTimestamp(raw any) (*time.Time, bool) {
	if raw == nil {
		return nil, true
	}

	switch value := raw.(type) {
	case time.Time:
		t := value.UTC()
		return &t, true
	case string:
		text := strings.TrimSpace(value)
		if text == "" {
			return nil, true
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
			if parsed, err := time.Parse(layout, text); err == nil {
				t := parsed.UTC()
				return &t, true
			}
		}
		return nil, false
	default:
		return nil, false
	}
}

func parsePositiveInt(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), value == float64(int(value))
	case string:
		parsed := intFromAny(value, -1)
		return parsed, parsed >= 0
	default:
		return 0, false
	}
}
