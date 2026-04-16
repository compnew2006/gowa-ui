package config

import (
	"fmt"
	"strings"
)

// ValidateLicenseConfig checks licensing-specific configuration constraints.
func ValidateLicenseConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if !cfg.License.Enabled {
		return nil
	}

	cfg.License.PublicKey = strings.TrimSpace(cfg.License.PublicKey)
	cfg.License.PublicKeyKID = strings.TrimSpace(cfg.License.PublicKeyKID)
	cfg.License.HostMachineIDPath = strings.TrimSpace(cfg.License.HostMachineIDPath)

	if cfg.License.RollbackToleranceSeconds < 60 {
		return fmt.Errorf("license.rollback_tolerance_seconds must be at least 60")
	}
	if cfg.License.GracePeriodDays < 0 {
		return fmt.Errorf("license.grace_period_days cannot be negative")
	}

	if cfg.License.PublicKey != "" && !cfg.License.AllowUnsafePublicKeyOverride {
		if isProductionEnv(cfg) {
			return fmt.Errorf("license.public_key override is disabled in production unless license.allow_unsafe_public_key_override=true")
		}
		return fmt.Errorf("license.public_key override requires license.allow_unsafe_public_key_override=true outside production")
	}

	return nil
}
