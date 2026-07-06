package whatsmeow

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
)

const (
	// InstanceSettingAutoRejectCalls stores per-instance incoming call auto-reject behavior.
	InstanceSettingAutoRejectCalls = "auto_reject_calls"

	AutoRejectCallModeWithoutMessage = "without_message"
	AutoRejectCallModeWithMessage    = "with_message"

	AutoRejectScheduleAlways           = "always"
	AutoRejectScheduleCustomHours      = "custom_hours"
	AutoRejectScheduleWhileInOtherCall = "while_in_other_calls"
)

const defaultAutoRejectReplyMessage = "Call declined automatically. The recipient is unavailable right now."

type AutoRejectSchedule struct {
	Type     string `json:"type"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Days     []int  `json:"days"`
	Timezone string `json:"timezone"`
}

type AutoRejectCallSettings struct {
	Enabled               bool               `json:"enabled"`
	Mode                  string             `json:"mode"`
	Message               string             `json:"message"`
	RejectIndividualCalls bool               `json:"reject_individual_calls"`
	RejectGroupCalls      bool               `json:"reject_group_calls"`
	BypassContacts        []string           `json:"bypass_contacts"`
	Schedule              AutoRejectSchedule `json:"schedule"`
}

type AutoRejectCallDecision struct {
	ShouldReject bool
	Reason       string
	ReplyMessage string
}

func DefaultAutoRejectCallSettings() AutoRejectCallSettings {
	return AutoRejectCallSettings{
		Enabled:               false,
		Mode:                  AutoRejectCallModeWithoutMessage,
		Message:               "",
		RejectIndividualCalls: true,
		RejectGroupCalls:      true,
		BypassContacts:        []string{},
		Schedule: AutoRejectSchedule{
			Type:     AutoRejectScheduleAlways,
			Start:    "09:00",
			End:      "18:00",
			Days:     []int{1, 2, 3, 4, 5},
			Timezone: "UTC",
		},
	}
}

func (s AutoRejectCallSettings) ToJSONB() models.JSONB {
	return models.JSONB{
		"enabled":                 s.Enabled,
		"mode":                    s.Mode,
		"message":                 s.Message,
		"reject_individual_calls": s.RejectIndividualCalls,
		"reject_group_calls":      s.RejectGroupCalls,
		"bypass_contacts":         append([]string(nil), s.BypassContacts...),
		"schedule": models.JSONB{
			"type":     s.Schedule.Type,
			"start":    s.Schedule.Start,
			"end":      s.Schedule.End,
			"days":     append([]int(nil), s.Schedule.Days...),
			"timezone": s.Schedule.Timezone,
		},
	}
}

func AutoRejectCallSettingsFromSettings(settings models.JSONB) AutoRejectCallSettings {
	if settings == nil {
		return DefaultAutoRejectCallSettings()
	}
	return NormalizeAutoRejectCallSettings(settings[InstanceSettingAutoRejectCalls])
}

func mapFromAny(raw any) (map[string]any, bool) {
	rawMap, ok := raw.(map[string]any)
	if ok {
		return rawMap, true
	}
	if typed, typedOK := raw.(models.JSONB); typedOK {
		return map[string]any(typed), true
	}
	return nil, false
}

func NormalizeAutoRejectCallSettings(raw any) AutoRejectCallSettings {
	normalized := DefaultAutoRejectCallSettings()

	rawMap, ok := mapFromAny(raw)
	if !ok || rawMap == nil {
		return normalized
	}

	normalized.Enabled = boolFromAny(rawMap["enabled"], normalized.Enabled)
	normalized.RejectIndividualCalls = boolFromAny(rawMap["reject_individual_calls"], normalized.RejectIndividualCalls)
	normalized.RejectGroupCalls = boolFromAny(rawMap["reject_group_calls"], normalized.RejectGroupCalls)

	mode := strings.TrimSpace(stringFromAny(rawMap["mode"]))
	if mode == AutoRejectCallModeWithMessage {
		normalized.Mode = AutoRejectCallModeWithMessage
	} else {
		normalized.Mode = AutoRejectCallModeWithoutMessage
	}

	normalized.Message = strings.TrimSpace(stringFromAny(rawMap["message"]))
	normalized.BypassContacts = normalizeBypassContacts(rawMap["bypass_contacts"])

	normalized.Schedule = normalizeAutoRejectSchedule(rawMap["schedule"], normalized.Schedule)
	return normalized
}

func ValidateAutoRejectCallSettings(raw any) error {
	rawMap, _ := mapFromAny(raw)
	normalized := NormalizeAutoRejectCallSettings(raw)

	if rawMap != nil {
		rawMode := strings.TrimSpace(stringFromAny(rawMap["mode"]))
		if rawMode != "" && rawMode != AutoRejectCallModeWithoutMessage && rawMode != AutoRejectCallModeWithMessage {
			return fmt.Errorf("auto reject mode must be %q or %q", AutoRejectCallModeWithoutMessage, AutoRejectCallModeWithMessage)
		}
	}

	if normalized.Mode != AutoRejectCallModeWithoutMessage && normalized.Mode != AutoRejectCallModeWithMessage {
		return fmt.Errorf("auto reject mode must be %q or %q", AutoRejectCallModeWithoutMessage, AutoRejectCallModeWithMessage)
	}

	if rawMap != nil {
		if rawSchedule, ok := mapFromAny(rawMap["schedule"]); ok {
			rawScheduleType := strings.TrimSpace(stringFromAny(rawSchedule["type"]))
			if rawScheduleType != "" &&
				rawScheduleType != AutoRejectScheduleAlways &&
				rawScheduleType != AutoRejectScheduleCustomHours &&
				rawScheduleType != AutoRejectScheduleWhileInOtherCall {
				return fmt.Errorf("auto reject schedule type is invalid")
			}
		}
	}

	if normalized.Schedule.Type != AutoRejectScheduleAlways &&
		normalized.Schedule.Type != AutoRejectScheduleCustomHours &&
		normalized.Schedule.Type != AutoRejectScheduleWhileInOtherCall {
		return fmt.Errorf("auto reject schedule type is invalid")
	}
	if normalized.Schedule.Type == AutoRejectScheduleCustomHours {
		if rawMap != nil {
			if rawSchedule, ok := mapFromAny(rawMap["schedule"]); ok {
				if rawStart, exists := rawSchedule["start"]; exists {
					start := strings.TrimSpace(stringFromAny(rawStart))
					if _, valid := parseHHMM(start); !valid {
						return fmt.Errorf("auto reject custom schedule start must be HH:MM")
					}
				}
				if rawEnd, exists := rawSchedule["end"]; exists {
					end := strings.TrimSpace(stringFromAny(rawEnd))
					if _, valid := parseHHMM(end); !valid {
						return fmt.Errorf("auto reject custom schedule end must be HH:MM")
					}
				}
				if rawDays, exists := rawSchedule["days"]; exists {
					if len(normalizeDays(rawDays)) == 0 {
						return fmt.Errorf("auto reject custom schedule days cannot be empty")
					}
				}
				if rawTimezone, exists := rawSchedule["timezone"]; exists {
					timezone := strings.TrimSpace(stringFromAny(rawTimezone))
					if timezone == "" {
						return fmt.Errorf("auto reject schedule timezone is invalid")
					}
					if _, err := time.LoadLocation(timezone); err != nil {
						return fmt.Errorf("auto reject schedule timezone is invalid")
					}
				}
			}
		}

		if _, ok := parseHHMM(normalized.Schedule.Start); !ok {
			return fmt.Errorf("auto reject custom schedule start must be HH:MM")
		}
		if _, ok := parseHHMM(normalized.Schedule.End); !ok {
			return fmt.Errorf("auto reject custom schedule end must be HH:MM")
		}
		if len(normalized.Schedule.Days) == 0 {
			return fmt.Errorf("auto reject custom schedule days cannot be empty")
		}
		for _, day := range normalized.Schedule.Days {
			if day < 0 || day > 6 {
				return fmt.Errorf("auto reject custom schedule days must be between 0 and 6")
			}
		}
		if _, err := time.LoadLocation(normalized.Schedule.Timezone); err != nil {
			return fmt.Errorf("auto reject schedule timezone is invalid")
		}
	}

	if normalized.Mode == AutoRejectCallModeWithMessage && strings.TrimSpace(normalized.Message) == "" {
		return fmt.Errorf("auto reject message is required in with_message mode")
	}
	return nil
}

func EvaluateAutoRejectCall(settings AutoRejectCallSettings, now time.Time, isGroupCall bool, activeCallCount int, callerPhone string) AutoRejectCallDecision {
	if !settings.Enabled {
		return AutoRejectCallDecision{ShouldReject: false, Reason: "disabled"}
	}

	if isGroupCall && !settings.RejectGroupCalls {
		return AutoRejectCallDecision{ShouldReject: false, Reason: "group_calls_disabled"}
	}
	if !isGroupCall && !settings.RejectIndividualCalls {
		return AutoRejectCallDecision{ShouldReject: false, Reason: "individual_calls_disabled"}
	}

	if isBypassContact(settings, callerPhone) {
		return AutoRejectCallDecision{ShouldReject: false, Reason: "bypass_contact"}
	}

	if !isScheduleActive(settings.Schedule, now, activeCallCount) {
		return AutoRejectCallDecision{ShouldReject: false, Reason: "outside_schedule"}
	}

	decision := AutoRejectCallDecision{ShouldReject: true, Reason: "matched"}
	if settings.Mode == AutoRejectCallModeWithMessage {
		decision.ReplyMessage = strings.TrimSpace(settings.Message)
		if decision.ReplyMessage == "" {
			decision.ReplyMessage = defaultAutoRejectReplyMessage
		}
	}

	return decision
}

func normalizeAutoRejectSchedule(raw any, fallback AutoRejectSchedule) AutoRejectSchedule {
	normalized := fallback

	rawMap, ok := mapFromAny(raw)
	if !ok || rawMap == nil {
		return normalized
	}

	scheduleType := strings.TrimSpace(stringFromAny(rawMap["type"]))
	switch scheduleType {
	case AutoRejectScheduleAlways, AutoRejectScheduleCustomHours, AutoRejectScheduleWhileInOtherCall:
		normalized.Type = scheduleType
	}

	start := strings.TrimSpace(stringFromAny(rawMap["start"]))
	if _, ok := parseHHMM(start); ok {
		normalized.Start = start
	}

	end := strings.TrimSpace(stringFromAny(rawMap["end"]))
	if _, ok := parseHHMM(end); ok {
		normalized.End = end
	}

	days := normalizeDays(rawMap["days"])
	if len(days) > 0 {
		normalized.Days = days
	}

	timezone := strings.TrimSpace(stringFromAny(rawMap["timezone"]))
	if timezone != "" {
		if _, err := time.LoadLocation(timezone); err == nil {
			normalized.Timezone = timezone
		}
	}

	return normalized
}

func normalizeBypassContacts(raw any) []string {
	if raw == nil {
		return []string{}
	}

	add := func(value string, seen map[string]struct{}, out *[]string) {
		normalized := normalizeContactNumber(value)
		if normalized == "" {
			return
		}
		if _, exists := seen[normalized]; exists {
			return
		}
		seen[normalized] = struct{}{}
		*out = append(*out, normalized)
	}

	seen := map[string]struct{}{}
	results := make([]string, 0)
	switch value := raw.(type) {
	case []string:
		for _, entry := range value {
			add(entry, seen, &results)
		}
	case []any:
		for _, entry := range value {
			add(stringFromAny(entry), seen, &results)
		}
	case string:
		for _, part := range splitByCommaOrLine(value) {
			add(part, seen, &results)
		}
	}

	sort.Strings(results)
	return results
}

func normalizeDays(raw any) []int {
	resultSet := map[int]struct{}{}
	appendDay := func(value int) {
		if value < 0 || value > 6 {
			return
		}
		resultSet[value] = struct{}{}
	}

	switch days := raw.(type) {
	case []int:
		for _, day := range days {
			appendDay(day)
		}
	case []any:
		for _, day := range days {
			appendDay(intFromAny(day, -1))
		}
	}

	if len(resultSet) == 0 {
		return nil
	}
	result := make([]int, 0, len(resultSet))
	for day := range resultSet {
		result = append(result, day)
	}
	sort.Ints(result)
	return result
}

func isScheduleActive(schedule AutoRejectSchedule, now time.Time, activeCallCount int) bool {
	switch schedule.Type {
	case AutoRejectScheduleWhileInOtherCall:
		return activeCallCount > 0
	case AutoRejectScheduleCustomHours:
		return isWithinCustomHours(schedule, now)
	default:
		return true
	}
}

func isWithinCustomHours(schedule AutoRejectSchedule, now time.Time) bool {
	location := time.UTC
	if schedule.Timezone != "" {
		if parsedLocation, err := time.LoadLocation(schedule.Timezone); err == nil {
			location = parsedLocation
		}
	}
	localized := now.In(location)

	if len(schedule.Days) > 0 {
		day := int(localized.Weekday())
		allowed := false
		for _, entry := range schedule.Days {
			if entry == day {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	start, okStart := parseHHMM(schedule.Start)
	end, okEnd := parseHHMM(schedule.End)
	if !okStart || !okEnd {
		return false
	}
	if start == end {
		return true
	}

	minutes := localized.Hour()*60 + localized.Minute()
	if start < end {
		return minutes >= start && minutes < end
	}
	return minutes >= start || minutes < end
}

func isBypassContact(settings AutoRejectCallSettings, callerPhone string) bool {
	caller := normalizeContactNumber(callerPhone)
	if caller == "" {
		return false
	}
	for _, entry := range settings.BypassContacts {
		if caller == entry {
			return true
		}
	}
	return false
}

func parseHHMM(value string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, false
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, false
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, false
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func normalizeContactNumber(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func splitByCommaOrLine(value string) []string {
	value = strings.ReplaceAll(value, "\n", ",")
	value = strings.ReplaceAll(value, "\r", ",")
	parts := strings.Split(value, ",")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}

func boolFromAny(raw any, fallback bool) bool {
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

func stringFromAny(raw any) string {
	switch value := raw.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return ""
	}
}

func intFromAny(raw any, fallback int) int {
	switch value := raw.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	}
	return fallback
}
