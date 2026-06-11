package handlers

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	filepath.Join("whatsmeow", "media"),
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

func (a *App) isReady() bool {
	return a != nil && a.DB != nil && a.Config != nil
}

func (a *App) resolveUploadsRootPath() (string, error) {
	return filepath.Abs(a.getMediaStoragePath())
}

func uploadsCutoffTime(now time.Time, retentionDays int) time.Time {
	return now.Add(-time.Duration(retentionDays) * 24 * time.Hour)
}

// ParseUploadsCleanupRetentionDays extracts retention days from org-level settings.
func ParseUploadsCleanupRetentionDays(settings models.JSONB) int {
	return parseUploadsCleanupRetentionDays(settings)
}

// ResolveInstanceRetention returns the effective retention days for a specific instance.
// If the instance has uploads_cleanup.inherit=true or no uploads_cleanup config, it falls
// back to workspaceDefault. Returns (days, source) where source is "custom", "default", or "disabled".
func ResolveInstanceRetention(instanceSettings models.JSONB, workspaceDefault int) (int, string) {
	uc, ok := instanceSettings["uploads_cleanup"].(map[string]interface{})
	if !ok {
		if workspaceDefault > 0 {
			return workspaceDefault, "default"
		}
		return 0, "disabled"
	}
	inherit, _ := uc["inherit"].(bool)
	if inherit {
		if workspaceDefault > 0 {
			return workspaceDefault, "default"
		}
		return 0, "disabled"
	}
	rd, ok := uc["retention_days"].(float64)
	if !ok || int(rd) <= 0 {
		return 0, "disabled"
	}
	return int(rd), "custom"
}