package worker

import (
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
)

const organizationWorkerScalerSettingsKey = "worker_scaler"

// OrganizationWorkerConfig controls tenant-scoped worker allocation.
type OrganizationWorkerConfig struct {
	MinWorkers               int
	MaxWorkers               int
	JobsPerWorker            int
	ScaleUpCooldownSeconds   int
	ScaleDownCooldownSeconds int
	PauseOnDisconnect        bool
}

// DefaultOrganizationWorkerConfig returns the baseline scaler settings.
func DefaultOrganizationWorkerConfig() OrganizationWorkerConfig {
	return OrganizationWorkerConfig{
		MinWorkers:               0,
		MaxWorkers:               4,
		JobsPerWorker:            25,
		ScaleUpCooldownSeconds:   30,
		ScaleDownCooldownSeconds: 60,
		PauseOnDisconnect:        true,
	}
}

// Normalize coerces invalid or conflicting settings back to safe defaults.
func (c OrganizationWorkerConfig) Normalize() OrganizationWorkerConfig {
	defaults := DefaultOrganizationWorkerConfig()

	if c.MinWorkers < 0 {
		c.MinWorkers = defaults.MinWorkers
	}
	if c.MaxWorkers <= 0 {
		c.MaxWorkers = defaults.MaxWorkers
	}
	if c.MinWorkers > c.MaxWorkers {
		c.MinWorkers = c.MaxWorkers
	}
	if c.JobsPerWorker <= 0 {
		c.JobsPerWorker = defaults.JobsPerWorker
	}
	if c.ScaleUpCooldownSeconds < 0 {
		c.ScaleUpCooldownSeconds = defaults.ScaleUpCooldownSeconds
	}
	if c.ScaleDownCooldownSeconds < 0 {
		c.ScaleDownCooldownSeconds = defaults.ScaleDownCooldownSeconds
	}
	return c
}

// LoadOrganizationWorkerConfig parses worker scaler settings from organization JSONB settings.
func LoadOrganizationWorkerConfig(settings models.JSONB) OrganizationWorkerConfig {
	config := DefaultOrganizationWorkerConfig()
	if settings == nil {
		return config
	}

	raw, ok := settings[organizationWorkerScalerSettingsKey]
	if !ok || raw == nil {
		return config
	}

	values := normalizeWorkerSettingsMap(raw)
	if values == nil {
		return config
	}

	config.MinWorkers = readWorkerSettingsInt(values, "min_workers", config.MinWorkers)
	config.MaxWorkers = readWorkerSettingsInt(values, "max_workers", config.MaxWorkers)
	config.JobsPerWorker = readWorkerSettingsInt(values, "jobs_per_worker", config.JobsPerWorker)
	config.ScaleUpCooldownSeconds = readWorkerSettingsInt(values, "scale_up_cooldown_seconds", config.ScaleUpCooldownSeconds)
	config.ScaleDownCooldownSeconds = readWorkerSettingsInt(values, "scale_down_cooldown_seconds", config.ScaleDownCooldownSeconds)
	config.PauseOnDisconnect = readWorkerSettingsBool(values, "pause_on_disconnect", config.PauseOnDisconnect)
	return config.Normalize()
}

func applyLicensedWorkerCap(config OrganizationWorkerConfig, state license.State) OrganizationWorkerConfig {
	if state.MaxWorkersPerOrg <= 0 {
		return config.Normalize()
	}
	if config.MaxWorkers > state.MaxWorkersPerOrg {
		config.MaxWorkers = state.MaxWorkersPerOrg
	}
	if config.MinWorkers > state.MaxWorkersPerOrg {
		config.MinWorkers = state.MaxWorkersPerOrg
	}
	return config.Normalize()
}

func normalizeWorkerSettingsMap(raw any) map[string]any {
	switch typed := raw.(type) {
	case models.JSONB:
		return map[string]any(typed)
	case map[string]any:
		return typed
	default:
		return nil
	}
}

func readWorkerSettingsInt(values map[string]any, key string, fallback int) int {
	raw, ok := values[key]
	if !ok || raw == nil {
		return fallback
	}

	switch typed := raw.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}

func readWorkerSettingsBool(values map[string]any, key string, fallback bool) bool {
	raw, ok := values[key]
	if !ok || raw == nil {
		return fallback
	}

	switch typed := raw.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return fallback
}
