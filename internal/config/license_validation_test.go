package config

import "testing"

func TestValidateLicenseConfigRejectsProductionPublicKeyOverride(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		App: AppConfig{
			Environment: "production",
		},
		License: LicenseConfig{
			Enabled:                      true,
			PublicKey:                    "test-public-key",
			PublicKeyKID:                 "vendor-1",
			AllowUnsafePublicKeyOverride: true,
			RollbackToleranceSeconds:     60,
			GracePeriodDays:              7,
		},
	}

	if err := ValidateLicenseConfig(cfg); err == nil {
		t.Fatal("ValidateLicenseConfig() error = nil, want production override rejection")
	}
}

func TestValidateLicenseConfigAllowsDevelopmentPublicKeyOverrideWithOptIn(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		App: AppConfig{
			Environment: "development",
		},
		License: LicenseConfig{
			Enabled:                      true,
			PublicKey:                    "test-public-key",
			PublicKeyKID:                 "vendor-1",
			AllowUnsafePublicKeyOverride: true,
			RollbackToleranceSeconds:     60,
			GracePeriodDays:              7,
		},
	}

	if err := ValidateLicenseConfig(cfg); err != nil {
		t.Fatalf("ValidateLicenseConfig() error = %v", err)
	}
}

func TestValidateLicenseConfigRequiresOptInOutsideProduction(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		App: AppConfig{
			Environment: "staging",
		},
		License: LicenseConfig{
			Enabled:                  true,
			PublicKey:                "test-public-key",
			PublicKeyKID:             "vendor-1",
			RollbackToleranceSeconds: 60,
			GracePeriodDays:          7,
		},
	}

	if err := ValidateLicenseConfig(cfg); err == nil {
		t.Fatal("ValidateLicenseConfig() error = nil, want non-production opt-in enforcement")
	}
}
