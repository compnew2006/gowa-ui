package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityValidationNilConfig(t *testing.T) {
	t.Run("ValidateDatabaseCredentials rejects nil config", func(t *testing.T) {
		err := ValidateDatabaseCredentials(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("ValidateWebhookVerifyToken rejects nil config", func(t *testing.T) {
		err := ValidateWebhookVerifyToken(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("isProductionEnv returns false for nil", func(t *testing.T) {
		assert.False(t, isProductionEnv(nil))
	})
}

func TestSecurityValidationNonProductionPasses(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Environment: "development"},
		Database: DatabaseConfig{User: "admin", Password: "password"},
		WhatsApp: WhatsAppConfig{WebhookVerifyToken: "changeme"},
	}

	assert.NoError(t, ValidateDatabaseCredentials(cfg))
	assert.NoError(t, ValidateWebhookVerifyToken(cfg))
}

func TestSecurityValidationProductionInsecureDB(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Environment: "production"},
		Database: DatabaseConfig{User: "admin", Password: "changeme"},
	}

	err := ValidateDatabaseCredentials(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database.user")
}

func TestSecurityValidationProductionSecureDB(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Environment: "production"},
		Database: DatabaseConfig{User: "secure_user", Password: "uniquePassword123"},
	}
	assert.NoError(t, ValidateDatabaseCredentials(cfg))
}

func TestSecurityValidationProductionEmptyToken(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Environment: "production"},
		WhatsApp: WhatsAppConfig{WebhookVerifyToken: ""},
	}
	err := ValidateWebhookVerifyToken(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook_verify_token must be set")
}

func TestSecurityValidationProductionInsecureToken(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Environment: "production"},
		WhatsApp: WhatsAppConfig{WebhookVerifyToken: "changeme"},
	}
	err := ValidateWebhookVerifyToken(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insecure")
}

func TestSecurityValidationProductionSecureToken(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Environment: "production"},
		WhatsApp: WhatsAppConfig{WebhookVerifyToken: "a_very_unique_and_long_token_xyz"},
	}
	assert.NoError(t, ValidateWebhookVerifyToken(cfg))
}

func TestIsProductionEnv(t *testing.T) {
	tests := []struct {
		env string
		exp bool
	}{
		{"production", true},
		{"Production", true},
		{"PRODUCTION", true},
		{"  production  ", true},
		{"staging", false},
		{"development", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{App: AppConfig{Environment: tt.env}}
			assert.Equal(t, tt.exp, isProductionEnv(cfg))
		})
	}
}
