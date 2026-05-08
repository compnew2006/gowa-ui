package worker

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDefaultOrganizationWorkerConfig(t *testing.T) {
	cfg := DefaultOrganizationWorkerConfig()
	assert.Equal(t, 0, cfg.MinWorkers)
	assert.Equal(t, 4, cfg.MaxWorkers)
	assert.Equal(t, 25, cfg.JobsPerWorker)
	assert.Equal(t, 30, cfg.ScaleUpCooldownSeconds)
	assert.Equal(t, 60, cfg.ScaleDownCooldownSeconds)
	assert.True(t, cfg.PauseOnDisconnect)
}

func TestOrganizationWorkerConfigNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input OrganizationWorkerConfig
		want  OrganizationWorkerConfig
	}{
		{
			name:  "zero values get defaults for numeric fields",
			input: OrganizationWorkerConfig{},
			want:  OrganizationWorkerConfig{MinWorkers: 0, MaxWorkers: 4, JobsPerWorker: 25, ScaleUpCooldownSeconds: 0, ScaleDownCooldownSeconds: 0, PauseOnDisconnect: false},
		},
		{
			name: "negative MinWorkers clamped to zero",
			input: OrganizationWorkerConfig{MinWorkers: -5, MaxWorkers: 10},
			want:  OrganizationWorkerConfig{MinWorkers: 0, MaxWorkers: 10, JobsPerWorker: 25, ScaleUpCooldownSeconds: 0, ScaleDownCooldownSeconds: 0, PauseOnDisconnect: false},
		},
		{
			name: "MinWorkers capped to MaxWorkers",
			input: OrganizationWorkerConfig{MinWorkers: 20, MaxWorkers: 10},
			want:  OrganizationWorkerConfig{MinWorkers: 10, MaxWorkers: 10, JobsPerWorker: 25, ScaleUpCooldownSeconds: 0, ScaleDownCooldownSeconds: 0, PauseOnDisconnect: false},
		},
		{
			name: "valid config unchanged",
			input: OrganizationWorkerConfig{MinWorkers: 2, MaxWorkers: 8, JobsPerWorker: 50, ScaleUpCooldownSeconds: 15, ScaleDownCooldownSeconds: 45, PauseOnDisconnect: false},
			want:  OrganizationWorkerConfig{MinWorkers: 2, MaxWorkers: 8, JobsPerWorker: 50, ScaleUpCooldownSeconds: 15, ScaleDownCooldownSeconds: 45, PauseOnDisconnect: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.input.Normalize())
		})
	}
}

func TestLoadOrganizationWorkerConfig(t *testing.T) {
	t.Run("nil settings returns defaults", func(t *testing.T) {
		cfg := LoadOrganizationWorkerConfig(nil)
		assert.Equal(t, DefaultOrganizationWorkerConfig(), cfg)
	})

	t.Run("missing key returns defaults", func(t *testing.T) {
		cfg := LoadOrganizationWorkerConfig(models.JSONB{"other": "data"})
		assert.Equal(t, DefaultOrganizationWorkerConfig(), cfg)
	})

	t.Run("valid settings parsed", func(t *testing.T) {
		settings := models.JSONB{
			"worker_scaler": map[string]any{
				"min_workers":              2,
				"max_workers":              8,
				"jobs_per_worker":           50,
				"scale_up_cooldown_seconds": 15,
				"scale_down_cooldown_seconds": 45,
				"pause_on_disconnect":       false,
			},
		}
		cfg := LoadOrganizationWorkerConfig(settings)
		assert.Equal(t, 2, cfg.MinWorkers)
		assert.Equal(t, 8, cfg.MaxWorkers)
		assert.Equal(t, 50, cfg.JobsPerWorker)
		assert.Equal(t, 15, cfg.ScaleUpCooldownSeconds)
		assert.Equal(t, 45, cfg.ScaleDownCooldownSeconds)
		assert.False(t, cfg.PauseOnDisconnect)
	})

	t.Run("string values parsed", func(t *testing.T) {
		settings := models.JSONB{
			"worker_scaler": map[string]any{
				"min_workers":        "3",
				"pause_on_disconnect": "false",
			},
		}
		cfg := LoadOrganizationWorkerConfig(settings)
		assert.Equal(t, 3, cfg.MinWorkers)
		assert.False(t, cfg.PauseOnDisconnect)
	})
}

func TestApplyLicensedWorkerCapConfig(t *testing.T) {
	t.Run("zero cap returns normalized", func(t *testing.T) {
		input := OrganizationWorkerConfig{MaxWorkers: 10, JobsPerWorker: 25}
		got := applyLicensedWorkerCap(input, license.State{})
		assert.Equal(t, 10, got.MaxWorkers)
		assert.Equal(t, 25, got.JobsPerWorker)
	})

	t.Run("cap reduces max workers", func(t *testing.T) {
		input := OrganizationWorkerConfig{MinWorkers: 1, MaxWorkers: 10}
		got := applyLicensedWorkerCap(input, license.State{MaxWorkersPerOrg: 5})
		assert.Equal(t, 5, got.MaxWorkers)
		assert.Equal(t, 1, got.MinWorkers)
	})

	t.Run("cap reduces min workers too", func(t *testing.T) {
		input := OrganizationWorkerConfig{MinWorkers: 8, MaxWorkers: 10}
		got := applyLicensedWorkerCap(input, license.State{MaxWorkersPerOrg: 5})
		assert.Equal(t, 5, got.MaxWorkers)
		assert.Equal(t, 5, got.MinWorkers)
	})
}

func TestReadWorkerSettingsInt(t *testing.T) {
	t.Run("missing key returns fallback", func(t *testing.T) {
		assert.Equal(t, 42, readWorkerSettingsInt(map[string]any{}, "x", 42))
	})

	t.Run("int value", func(t *testing.T) {
		assert.Equal(t, 7, readWorkerSettingsInt(map[string]any{"x": 7}, "x", 0))
	})

	t.Run("float64 value", func(t *testing.T) {
		assert.Equal(t, 3, readWorkerSettingsInt(map[string]any{"x": float64(3.9)}, "x", 0))
	})

	t.Run("string value", func(t *testing.T) {
		assert.Equal(t, 10, readWorkerSettingsInt(map[string]any{"x": "10"}, "x", 0))
	})

	t.Run("invalid string returns fallback", func(t *testing.T) {
		assert.Equal(t, 99, readWorkerSettingsInt(map[string]any{"x": "abc"}, "x", 99))
	})
}

func TestReadWorkerSettingsBool(t *testing.T) {
	t.Run("missing key returns fallback", func(t *testing.T) {
		assert.True(t, readWorkerSettingsBool(map[string]any{}, "x", true))
	})

	t.Run("bool true", func(t *testing.T) {
		assert.True(t, readWorkerSettingsBool(map[string]any{"x": true}, "x", false))
	})

	t.Run("string true variants", func(t *testing.T) {
		for _, v := range []string{"true", "1", "yes", "on"} {
			assert.True(t, readWorkerSettingsBool(map[string]any{"x": v}, "x", false), v)
		}
	})

	t.Run("string false variants", func(t *testing.T) {
		for _, v := range []string{"false", "0", "no", "off"} {
			assert.False(t, readWorkerSettingsBool(map[string]any{"x": v}, "x", true), v)
		}
	})
}
