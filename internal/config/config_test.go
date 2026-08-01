package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestLoad_AppliesDefaultsForMissingFields(t *testing.T) {
	cfg, err := config.Load(writeConfig(t, ""))
	require.NoError(t, err)

	assert.Equal(t, "gowa-ui", cfg.App.Name)
	assert.Equal(t, "development", cfg.App.Environment)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 30, cfg.Server.ReadTimeout)
	assert.Equal(t, 30, cfg.Server.WriteTimeout)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	assert.Equal(t, 300, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.Equal(t, 15, cfg.JWT.AccessExpiryMins)
	assert.Equal(t, 1, cfg.JWT.RefreshExpiryDays)
	assert.Equal(t, "local", cfg.Storage.Type)
	assert.Equal(t, "./uploads", cfg.Storage.LocalPath)
	assert.Equal(t, "admin@admin.com", cfg.DefaultAdmin.Email)
	assert.Equal(t, "admin", cfg.DefaultAdmin.Password)
}

func TestLoad_FileValuesOverrideDefaults(t *testing.T) {
	cfg, err := config.Load(writeConfig(t, `
[app]
name = "MyApp"
environment = "production"

[server]
port = 9090

[database]
host = "db.example.com"
port = 5433
user = "u"
password = "p"
name = "n"

[storage]
local_path = "/data/media"
`))
	require.NoError(t, err)

	assert.Equal(t, "MyApp", cfg.App.Name)
	assert.Equal(t, "production", cfg.App.Environment)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "/data/media", cfg.Storage.LocalPath)
}

func TestLoad_ProductionEnvironmentForcesSecureCookie(t *testing.T) {
	cfg, err := config.Load(writeConfig(t, `
[app]
environment = "production"

[cookie]
secure = false
`))
	require.NoError(t, err)
	assert.True(t, cfg.Cookie.Secure, "production environment must force Cookie.Secure=true")
}

func TestLoad_DevelopmentDoesNotForceSecureCookie(t *testing.T) {
	cfg, err := config.Load(writeConfig(t, `
[app]
environment = "development"

[cookie]
secure = false
`))
	require.NoError(t, err)
	assert.False(t, cfg.Cookie.Secure)
}

func TestLoad_EnvVarsOverrideFile(t *testing.T) {
	t.Setenv("gowa-ui_DATABASE__HOST", "from-env")
	t.Setenv("gowa-ui_SERVER__PORT", "1234")

	cfg, err := config.Load(writeConfig(t, `
[database]
host = "from-file"

[server]
port = 8080
`))
	require.NoError(t, err)
	assert.Equal(t, "from-env", cfg.Database.Host, "gowa-ui_DATABASE__HOST must override file")
	assert.Equal(t, 1234, cfg.Server.Port, "gowa-ui_SERVER__PORT must override file")
}

func TestLoad_EmptyConfigPathStillLoadsDefaults(t *testing.T) {
	cfg, err := config.Load("")
	require.NoError(t, err)
	assert.Equal(t, "gowa-ui", cfg.App.Name)
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestLoad_MissingFileReturnsError(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.toml")
	require.Error(t, err)
}

func TestLoad_RateLimitDefaults(t *testing.T) {
	cfg, err := config.Load(writeConfig(t, ""))
	require.NoError(t, err)
	assert.Equal(t, 10, cfg.RateLimit.LoginMaxAttempts)
	assert.Equal(t, 10, cfg.RateLimit.RegisterMaxAttempts)
	assert.Equal(t, 30, cfg.RateLimit.RefreshMaxAttempts)
	assert.Equal(t, 10, cfg.RateLimit.SSOMaxAttempts)
}

// TestLoad_EnvMapsMultiWordKeys is the regression for the embedded-signup bug
// (gowa-ui#476): env vars for keys whose section OR field name contains an
// underscore must map correctly. Levels are separated by "__"; single
// underscores stay part of the key. This exercises config.Load()'s env path,
// which the handler-level tests bypass by setting the struct directly.
func TestLoad_EnvMapsMultiWordKeys(t *testing.T) {
	t.Setenv("gowa-ui_DEFAULT_ADMIN__EMAIL", "admin@example.com")
	t.Setenv("gowa-ui_DATABASE__HOST", "db.internal")
	t.Setenv("gowa-ui_STORAGE__LOCAL_PATH", "/env/media")

	cfg, err := config.Load("") // no file; env-only
	require.NoError(t, err)

	assert.Equal(t, "admin@example.com", cfg.DefaultAdmin.Email)
	assert.Equal(t, "db.internal", cfg.Database.Host)
	assert.Equal(t, "/env/media", cfg.Storage.LocalPath)
}

// TestLoad_GOWABasicAuthFields verifies the GOWA REST API Basic Auth
// credentials (added so main.go can seed the GOWA factory from config)
// load from both the TOML [gowa] section and the gowa-ui_GOWA__ env vars.
func TestLoad_GOWABasicAuthFields(t *testing.T) {
	t.Run("toml", func(t *testing.T) {
		cfg, err := config.Load(writeConfig(t, `
[gowa]
username = "gowa-user"
password = "gowa-pass"
`))
		require.NoError(t, err)
		assert.Equal(t, "gowa-user", cfg.GOWA.Username)
		assert.Equal(t, "gowa-pass", cfg.GOWA.Password)
	})

	t.Run("env", func(t *testing.T) {
		t.Setenv("gowa-ui_GOWA__USERNAME", "env-gowa-user")
		t.Setenv("gowa-ui_GOWA__PASSWORD", "env-gowa-pass")

		cfg, err := config.Load("")
		require.NoError(t, err)
		assert.Equal(t, "env-gowa-user", cfg.GOWA.Username)
		assert.Equal(t, "env-gowa-pass", cfg.GOWA.Password)
	})
}
