package handlers

import (
	"strconv"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
)

const (
	organizationSettingUploadsCleanupRetentionDays = "uploads_cleanup_retention_days"
	organizationSettingUploadsCleanupScheduleHour  = "uploads_cleanup_schedule_hour"
	organizationSettingUploadsCleanupLastRunDate   = "uploads_cleanup_last_run_date"
	defaultUploadsCleanupRetentionDays             = 0
	defaultUploadsCleanupScheduleHour              = 3
	maxUploadsCleanupRetentionDays                 = 3650
)

var uploadsCleanupTargetDirs = []string{
	"audio",
	"documents",
	"images",
	"stickers",
	"videos",
}

func parseUploadsCleanupRetentionDays(settings models.JSONB) int {
	return normalizeUploadsCleanupRetentionDays(
		parseOrganizationIntSetting(
			settings,
			organizationSettingUploadsCleanupRetentionDays,
			defaultUploadsCleanupRetentionDays,
		),
	)
}

func normalizeUploadsCleanupRetentionDays(days int) int {
	switch {
	case days < 0:
		return defaultUploadsCleanupRetentionDays
	case days > maxUploadsCleanupRetentionDays:
		return maxUploadsCleanupRetentionDays
	default:
		return days
	}
}

func parseUploadsCleanupScheduleHour(settings models.JSONB) int {
	return normalizeUploadsCleanupScheduleHour(
		parseOrganizationIntSetting(
			settings,
			organizationSettingUploadsCleanupScheduleHour,
			defaultUploadsCleanupScheduleHour,
		),
	)
}

func normalizeUploadsCleanupScheduleHour(hour int) int {
	switch {
	case hour < 0:
		return defaultUploadsCleanupScheduleHour
	case hour > 23:
		return 23
	default:
		return hour
	}
}

func parseUploadsCleanupLastRunDate(settings models.JSONB) string {
	if settings == nil {
		return ""
	}

	raw, ok := settings[organizationSettingUploadsCleanupLastRunDate]
	if !ok || raw == nil {
		return ""
	}

	switch typed := raw.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	default:
		return ""
	}
}

func parseOrganizationIntSetting(settings models.JSONB, key string, fallback int) int {
	if settings == nil {
		return fallback
	}

	raw, ok := settings[key]
	if !ok || raw == nil {
		return fallback
	}

	switch typed := raw.(type) {
	case int:
		return typed
	case int8:
		return int(typed)
	case int16:
		return int(typed)
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	case []byte:
		return parseOrganizationIntSetting(models.JSONB{
			key: string(typed),
		}, key, fallback)
	}

	return fallback
}
