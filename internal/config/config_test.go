package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	// Ensure env vars don't interfere
	os.Clearenv()

	cfg, err := Load("")
	require.NoError(t, err)

	assert.Equal(t, "Whatomate", cfg.App.Name)
	assert.Equal(t, "development", cfg.App.Environment)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 30, cfg.Server.ReadTimeout)
	assert.Equal(t, 30, cfg.Server.WriteTimeout)
	assert.Equal(t, 110, cfg.Server.MaxRequestBodySizeMB)

	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.False(t, cfg.Database.LogSQL)
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	assert.Equal(t, 300, cfg.Database.ConnMaxLifetime)

	assert.Equal(t, 6379, cfg.Redis.Port)

	assert.Equal(t, 15, cfg.JWT.AccessExpiryMins)
	assert.Equal(t, 1, cfg.JWT.RefreshExpiryDays)

	assert.Equal(t, "v18.0", cfg.WhatsApp.APIVersion)
	assert.Equal(t, "https://graph.facebook.com", cfg.WhatsApp.BaseURL)
	assert.Equal(t, "meta", cfg.WhatsApp.Provider)

	assert.Equal(t, 1000, cfg.Whatsmeow.RateLimitMinDelayMs)
	assert.Equal(t, 3000, cfg.Whatsmeow.RateLimitMaxDelayMs)
	assert.Equal(t, 300, cfg.Whatsmeow.QueueTimeoutSeconds)
	assert.Equal(t, 5, cfg.Whatsmeow.MaxInstancesPerOrg)
	assert.Equal(t, 1, cfg.Whatsmeow.UploadRetryCount)
	assert.Equal(t, 2, cfg.Whatsmeow.UploadRetryDelaySec)
	assert.Equal(t, 4, cfg.Whatsmeow.InboundMediaAsyncRetryCount)
	assert.Equal(t, 5000, cfg.Whatsmeow.InboundMediaAsyncRetryDelayMs)
	assert.Equal(t, 60000, cfg.Whatsmeow.InboundMediaAsyncRetryMaxDelayMs)
	assert.True(t, cfg.Whatsmeow.TypingIndicatorEnabled)
	assert.Equal(t, 700, cfg.Whatsmeow.TypingMinDelayMs)
	assert.Equal(t, 3000, cfg.Whatsmeow.TypingMaxDelayMs)
	assert.Equal(t, 35, cfg.Whatsmeow.TypingCharDelayMs)
	assert.Equal(t, 4000, cfg.Whatsmeow.TypingCooldownMs)

	assert.Equal(t, "local", cfg.Storage.Type)
	assert.Equal(t, "./uploads", cfg.Storage.LocalPath)

	assert.Equal(t, "", cfg.DefaultAdmin.Email)
	assert.Equal(t, "", cfg.DefaultAdmin.Password)
	assert.Equal(t, "Admin", cfg.DefaultAdmin.FullName)

	assert.False(t, cfg.Cookie.Secure)

	assert.Equal(t, 10, cfg.RateLimit.LoginMaxAttempts)
	assert.Equal(t, 10, cfg.RateLimit.RegisterMaxAttempts)
	assert.Equal(t, 30, cfg.RateLimit.RefreshMaxAttempts)
	assert.Equal(t, 10, cfg.RateLimit.SSOMaxAttempts)
	assert.Equal(t, 300, cfg.RateLimit.WebhookMaxAttempts)
	assert.Equal(t, 60, cfg.RateLimit.WindowSeconds)
	assert.Equal(t, 5, cfg.RateLimit.OutboundPerUserPS)
	assert.Equal(t, 15, cfg.RateLimit.OutboundPerIPPS)
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	os.Clearenv()
	t.Setenv("WHATOMATE_APP_NAME", "TestApp")
	t.Setenv("WHATOMATE_SERVER_PORT", "9090")
	t.Setenv("WHATOMATE_DATABASE_USER", "dbuser")
	t.Setenv("WHATOMATE_WHATSMEOW_TYPING_INDICATOR_ENABLED", "false")
	t.Setenv("WHATOMATE_WHATSMEOW_INBOUND_MEDIA_ASYNC_RETRY_COUNT", "7")
	t.Setenv("WHATOMATE_APP_ENVIRONMENT", "production")

	cfg, err := Load("")
	require.NoError(t, err)

	assert.Equal(t, "TestApp", cfg.App.Name)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "dbuser", cfg.Database.User)
	assert.Equal(t, "production", cfg.App.Environment)
	assert.Equal(t, 7, cfg.Whatsmeow.InboundMediaAsyncRetryCount)
	assert.True(t, cfg.Cookie.Secure) // Auto-set to true when environment=production
}

func TestLoad_ConfigFile(t *testing.T) {
	os.Clearenv()

	content := `
[app]
name = "TomlApp"
environment = "staging"

[server]
port = 8081

[database]
host = "localhost"
port = 5433
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "TomlApp", cfg.App.Name)
	assert.Equal(t, "staging", cfg.App.Environment)
	assert.Equal(t, 8081, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)

	// Check a default value
	assert.Equal(t, 110, cfg.Server.MaxRequestBodySizeMB)
}

func TestLoad_InvalidFile(t *testing.T) {
	_, err := Load("non_existent_file.toml")
	assert.Error(t, err)
}

func TestNormalizeBasePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "dot",
			input:    ".",
			expected: "",
		},
		{
			name:     "dot slash",
			input:    "./",
			expected: "",
		},
		{
			name:     "root slash",
			input:    "/",
			expected: "",
		},
		{
			name:     "relative subpath",
			input:    "whatomate",
			expected: "/whatomate",
		},
		{
			name:     "relative subpath with slash",
			input:    "whatomate/",
			expected: "/whatomate",
		},
		{
			name:     "relative dot subpath",
			input:    "./whatomate/",
			expected: "/whatomate",
		},
		{
			name:     "absolute subpath",
			input:    "/whatomate/",
			expected: "/whatomate",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, normalizeBasePath(tc.input))
		})
	}
}
