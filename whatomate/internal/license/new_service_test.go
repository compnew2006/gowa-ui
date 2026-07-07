package license

import (
	"strings"
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/zerodha/logf"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewService(t *testing.T) {
	logger := logf.New(logf.Opts{})

	t.Run("nil config", func(t *testing.T) {
		svc, err := NewService(nil, nil, nil, logger)
		if err == nil || err.Error() != "config is nil" {
			t.Errorf("expected error 'config is nil', got %v", err)
		}
		if svc != nil {
			t.Errorf("expected nil service, got %v", svc)
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		cfg := &config.Config{
			License: config.LicenseConfig{
				Enabled:                  true,
				RollbackToleranceSeconds: 10,
			},
		}
		_, err := NewService(cfg, nil, nil, logger)
		if err == nil {
			t.Error("expected config validation error, got nil")
		}
	})

	t.Run("invalid public key", func(t *testing.T) {
		cfg := &config.Config{
			App: config.AppConfig{
				Environment: "development",
			},
			License: config.LicenseConfig{
				Enabled:                      true,
				RollbackToleranceSeconds:     60,
				AllowUnsafePublicKeyOverride: true,
				PublicKey:                    "invalid_base64_or_key",
			},
		}
		_, err := NewService(cfg, nil, nil, logger)
		if err == nil {
			t.Error("expected public key parsing error, got nil")
		}
	})

	t.Run("disabled license returns disabled state", func(t *testing.T) {
		cfg := &config.Config{
			License: config.LicenseConfig{
				Enabled: false,
			},
		}
		svc, err := NewService(cfg, nil, nil, logger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svc == nil {
			t.Fatal("expected non-nil service")
		}
		state := svc.CurrentState()
		if state.Enabled {
			t.Error("expected state.Enabled to be false")
		}
		if state.Status != StatusDisabled {
			t.Errorf("expected StatusDisabled, got %s", state.Status)
		}
	})

	t.Run("enabled but no usable public keys", func(t *testing.T) {
		originalEmbedded := EmbeddedPublicKeyRingBase64
		EmbeddedPublicKeyRingBase64 = ""
		defer func() { EmbeddedPublicKeyRingBase64 = originalEmbedded }()

		cfg := &config.Config{
			License: config.LicenseConfig{
				Enabled:                  true,
				RollbackToleranceSeconds: 60,
			},
		}
		_, err := NewService(cfg, nil, nil, logger)
		if err == nil || err.Error() != "license is enabled but no usable public keys are configured or embedded" {
			t.Errorf("expected no usable public keys error, got %v", err)
		}
	})

	t.Run("enabled but database is nil", func(t *testing.T) {
		cfg := &config.Config{
			App: config.AppConfig{
				Environment: "development",
			},
			License: config.LicenseConfig{
				Enabled:                      true,
				RollbackToleranceSeconds:     60,
				AllowUnsafePublicKeyOverride: true,
			},
		}
		pub, _, _ := GenerateKeyPair()
		cfg.License.PublicKey = pub

		_, err := NewService(cfg, nil, nil, logger)
		if err == nil || err.Error() != "license is enabled but database is nil" {
			t.Errorf("expected database is nil error, got %v", err)
		}
	})

	t.Run("refresh state failure due to unmigrated db", func(t *testing.T) {
		cfg := &config.Config{
			App: config.AppConfig{
				Environment: "development",
			},
			License: config.LicenseConfig{
				Enabled:                      true,
				RollbackToleranceSeconds:     60,
				AllowUnsafePublicKeyOverride: true,
			},
		}
		pub, _, _ := GenerateKeyPair()
		cfg.License.PublicKey = pub

		db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to open in-memory sqlite: %v", err)
		}

		// Because we don't AutoMigrate the license record table,
		// loadRecord will fail with a "no such table" error, which propagates.
		_, err = NewService(cfg, db, nil, logger)
		if err == nil {
			t.Error("expected refresh state error due to unmigrated db, got nil")
		}
	})

	t.Run("successful initialization", func(t *testing.T) {
		cfg := &config.Config{
			App: config.AppConfig{
				Environment: "development",
			},
			License: config.LicenseConfig{
				Enabled:                      true,
				RollbackToleranceSeconds:     60,
				AllowUnsafePublicKeyOverride: true,
			},
		}
		pub, _, _ := GenerateKeyPair()
		cfg.License.PublicKey = pub

		db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to open in-memory sqlite: %v", err)
		}

		// Create the required tables so it doesn't fail on missing tables.
		if err := db.AutoMigrate(&models.LicenseRecord{}); err != nil {
			t.Fatalf("failed to auto migrate: %v", err)
		}

		svc, err := NewService(cfg, db, nil, logger)
		if err != nil {
			t.Fatalf("expected successful initialization, got %v", err)
		}
		if svc == nil {
			t.Fatal("expected non-nil service")
		}

		state := svc.CurrentState()
		if !state.Enabled {
			t.Error("expected state.Enabled to be true")
		}
		if state.Status != StatusUnlicensed {
			t.Errorf("expected StatusUnlicensed, got %s", state.Status)
		}
	})

	t.Run("hwid build error", func(t *testing.T) {
		cfg := &config.Config{
			App: config.AppConfig{
				Environment: "development",
			},
			License: config.LicenseConfig{
				Enabled:                  true,
				RollbackToleranceSeconds: 60,
				HostMachineIDPath:        "/path/does/not/exist/ever/12345",
			},
		}

		_, err := NewService(cfg, nil, nil, logger)
		if err == nil {
			t.Error("expected hwid build error, got nil")
		} else if !strings.Contains(err.Error(), "missing or unreadable") {
			t.Errorf("expected missing or unreadable error, got %v", err)
		}
	})
}
