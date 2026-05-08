package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestUploadsCleanupNormalizeRetentionDays(t *testing.T) {
	assert.Equal(t, 0, normalizeUploadsCleanupRetentionDays(-1))
	assert.Equal(t, 0, normalizeUploadsCleanupRetentionDays(0))
	assert.Equal(t, 30, normalizeUploadsCleanupRetentionDays(30))
	assert.Equal(t, 3650, normalizeUploadsCleanupRetentionDays(3650))
	assert.Equal(t, 3650, normalizeUploadsCleanupRetentionDays(9999))
}

func TestUploadsCleanupNormalizeScheduleHour(t *testing.T) {
	assert.Equal(t, 3, normalizeUploadsCleanupScheduleHour(-1))
	assert.Equal(t, 0, normalizeUploadsCleanupScheduleHour(0))
	assert.Equal(t, 12, normalizeUploadsCleanupScheduleHour(12))
	assert.Equal(t, 23, normalizeUploadsCleanupScheduleHour(23))
	assert.Equal(t, 23, normalizeUploadsCleanupScheduleHour(25))
}

func TestUploadsCleanupParseOrganizationIntSetting(t *testing.T) {
	assert.Equal(t, 5, parseOrganizationIntSetting(models.JSONB{"k": 5}, "k", 0))
	assert.Equal(t, 0, parseOrganizationIntSetting(nil, "k", 0))
	assert.Equal(t, 42, parseOrganizationIntSetting(models.JSONB{"k": int8(42)}, "k", 0))
	assert.Equal(t, 10, parseOrganizationIntSetting(models.JSONB{"k": int16(10)}, "k", 0))
	assert.Equal(t, 7, parseOrganizationIntSetting(models.JSONB{"k": float64(7.9)}, "k", 0))
	assert.Equal(t, 3, parseOrganizationIntSetting(models.JSONB{"k": "3"}, "k", 0))
	assert.Equal(t, 99, parseOrganizationIntSetting(models.JSONB{"k": []byte("99")}, "k", 0))
	assert.Equal(t, 0, parseOrganizationIntSetting(models.JSONB{"k": "not-a-number"}, "k", 0))
	assert.Equal(t, 0, parseOrganizationIntSetting(models.JSONB{"other": 5}, "k", 0))
	assert.Equal(t, 42, parseOrganizationIntSetting(models.JSONB{"k": nil}, "k", 42))
}

func TestUploadsCleanupParseLastRunDate(t *testing.T) {
	assert.Equal(t, "", parseUploadsCleanupLastRunDate(nil))
	assert.Equal(t, "", parseUploadsCleanupLastRunDate(models.JSONB{}))
	assert.Equal(t, "2025-01-01", parseUploadsCleanupLastRunDate(models.JSONB{
		organizationSettingUploadsCleanupLastRunDate: "2025-01-01",
	}))
	assert.Equal(t, "2025-01-01", parseUploadsCleanupLastRunDate(models.JSONB{
		organizationSettingUploadsCleanupLastRunDate: []byte("2025-01-01"),
	}))
	assert.Equal(t, "", parseUploadsCleanupLastRunDate(models.JSONB{
		organizationSettingUploadsCleanupLastRunDate: 123,
	}))
}

func TestUploadsCleanupParseRetentionDaysFromSettings(t *testing.T) {
	assert.Equal(t, 0, parseUploadsCleanupRetentionDays(nil))
	assert.Equal(t, 30, parseUploadsCleanupRetentionDays(models.JSONB{
		organizationSettingUploadsCleanupRetentionDays: 30,
	}))
}

func TestUploadsCleanupParseScheduleHourFromSettings(t *testing.T) {
	assert.Equal(t, 3, parseUploadsCleanupScheduleHour(nil))
	assert.Equal(t, 14, parseUploadsCleanupScheduleHour(models.JSONB{
		organizationSettingUploadsCleanupScheduleHour: 14,
	}))
}
