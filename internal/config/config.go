package config

import (
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config holds all configuration for the application
type Config struct {
	App           AppConfig          `koanf:"app"`
	Server        ServerConfig       `koanf:"server"`
	Database      DatabaseConfig     `koanf:"database"`
	Redis         RedisConfig        `koanf:"redis"`
	JWT           JWTConfig          `koanf:"jwt"`
	GOWA          GOWAConfig         `koanf:"gowa"`
	GOWAInstances []GOWAInstance     `koanf:"gowa_instances"`
	AI            AIConfig           `koanf:"ai"`
	Storage       StorageConfig      `koanf:"storage"`
	DefaultAdmin  DefaultAdminConfig `koanf:"default_admin"`
	RateLimit     RateLimitConfig    `koanf:"rate_limit"`
	Cookie        CookieConfig       `koanf:"cookie"`
}

type AppConfig struct {
	Name          string `koanf:"name"`
	Environment   string `koanf:"environment"` // development, staging, production
	Debug         bool   `koanf:"debug"`
	EncryptionKey string `koanf:"encryption_key"` // AES-256 key for encrypting secrets at rest
}

type ServerConfig struct {
	Host           string `koanf:"host"`
	Port           int    `koanf:"port"`
	ReadTimeout    int    `koanf:"read_timeout"`
	WriteTimeout   int    `koanf:"write_timeout"`
	BasePath       string `koanf:"base_path"`       // Base path for frontend (e.g., "/gowa-ui" for proxy pass)
	AllowedOrigins string `koanf:"allowed_origins"` // Comma-separated list of allowed CORS origins
}

type DatabaseConfig struct {
	Host            string `koanf:"host"`
	Port            int    `koanf:"port"`
	User            string `koanf:"user"`
	Password        string `koanf:"password"`
	Name            string `koanf:"name"`
	SSLMode         string `koanf:"ssl_mode"`
	MaxOpenConns    int    `koanf:"max_open_conns"`
	MaxIdleConns    int    `koanf:"max_idle_conns"`
	ConnMaxLifetime int    `koanf:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	DB       int    `koanf:"db"`
	TLS      bool   `koanf:"tls"`
}

type JWTConfig struct {
	Secret            string `koanf:"secret"`
	AccessExpiryMins  int    `koanf:"access_expiry_mins"`
	RefreshExpiryDays int    `koanf:"refresh_expiry_days"`
}

// GOWAConfig holds connection settings for GOWA (Go WhatsApp Web Multi-Device)
// instances. When [gowa].base_url is set, the application can manage GOWA-backed
// WhatsApp accounts.
type GOWAConfig struct {
	BaseURL     string `koanf:"base_url"`     // GOWA REST API base URL, e.g. "http://gowa:8080"
	WebhookPath string `koanf:"webhook_path"` // Path where GOWA sends webhook events, default "/api/gowa/webhook"
	Username    string `koanf:"username"`     // Basic Auth username for the GOWA REST API
	Password    string `koanf:"password"`     // Basic Auth password for the GOWA REST API

	// InternalBaseURL overrides the address server-side GOWA API calls dial,
	// regardless of the per-instance/account base_url (which stays the
	// browser-facing public URL for QR links and the media-origin allowlist).
	// Set it when gowa-ui and GOWA are co-located behind the same reverse
	// proxy, e.g. public "https://gowa.example.com" → internal
	// "http://127.0.0.1:3000", so API traffic skips the public hop. It is
	// global: with multiple GOWA instances at different hosts, leave it empty.
	InternalBaseURL string `koanf:"internal_base_url"`
}

// GOWAInstance represents a single GOWA instance configured in the app.
// Multiple instances can be configured via [[gowa_instances]] in the TOML config.
type GOWAInstance struct {
	Name          string   `koanf:"name"`          // Human-readable label for the instance (shown in UI dropdown)
	BaseURL       string   `koanf:"base_url"`      // GOWA REST API base URL
	Username      string   `koanf:"username"`      // Basic Auth username
	Password      string   `koanf:"password"`      // Basic Auth password
	WebhookURL    string   `koanf:"webhook_url"`   // Externally-reachable gowa-ui webhook URL (e.g. http://host.docker.internal:18080/api/gowa/webhook). If empty, derived from request.
	Organizations []string `koanf:"organizations"` // Org IDs allowed to use this instance. ["*"] or empty = all orgs (backward compat).
}

// FindGOWAInstance returns the GOWAInstance matching the given base URL,
// or nil if no instance is configured for that URL. When orgID is non-empty,
// only instances that allow the given organization are considered.
func (c *Config) FindGOWAInstance(baseURL string, orgID ...string) *GOWAInstance {
	org := ""
	if len(orgID) > 0 {
		org = orgID[0]
	}
	for i := range c.GOWAInstances {
		inst := &c.GOWAInstances[i]
		if inst.BaseURL != baseURL {
			continue
		}
		if !instanceAllowsOrg(inst, org) {
			continue
		}
		return inst
	}
	return nil
}

// FindGOWAInstancesForOrg returns all GOWA instances available to the given
// organization. Used by the GowaInstances handler to list org-scoped instances.
func (c *Config) FindGOWAInstancesForOrg(orgID string) []GOWAInstance {
	var result []GOWAInstance
	for i := range c.GOWAInstances {
		if instanceAllowsOrg(&c.GOWAInstances[i], orgID) {
			result = append(result, c.GOWAInstances[i])
		}
	}
	return result
}

// instanceAllowsOrg checks whether the instance's Organizations field allows
// the given org. Empty or ["*"] means all orgs are allowed (backward compat).
func instanceAllowsOrg(inst *GOWAInstance, orgID string) bool {
	if len(inst.Organizations) == 0 {
		return true // backward compat: no restriction
	}
	for _, allowed := range inst.Organizations {
		if allowed == "*" || allowed == orgID {
			return true
		}
	}
	return false
}

type AIConfig struct {
	OpenAIKey    string `koanf:"openai_key"`
	AnthropicKey string `koanf:"anthropic_key"`
	GoogleKey    string `koanf:"google_key"`
}

type StorageConfig struct {
	Type      string `koanf:"type"` // local
	LocalPath string `koanf:"local_path"`
	// EagerHistoryMedia controls whether the GOWA history-sync processor
	// downloads media bytes at sync time. Default false: media metadata is
	// stored and bytes are fetched lazily via ServeMedia's recovery path when
	// an agent first opens them. Set true only on a host with ample disk,
	// since eager download of large histories can fill the disk (each device
	// pulls up to 50 msgs × every chat).
	EagerHistoryMedia bool `koanf:"eager_history_media"`
}

type DefaultAdminConfig struct {
	Email    string `koanf:"email"`
	Password string `koanf:"password"`
	FullName string `koanf:"full_name"`
}

type CookieConfig struct {
	Domain string `koanf:"domain"` // Cookie domain (e.g., ".example.com"). Empty = current host.
	Secure bool   `koanf:"secure"` // Set Secure flag. Auto-set true when environment=production.
}

type RateLimitConfig struct {
	Enabled             bool `koanf:"enabled"`
	LoginMaxAttempts    int  `koanf:"login_max_attempts"`
	RegisterMaxAttempts int  `koanf:"register_max_attempts"`
	RefreshMaxAttempts  int  `koanf:"refresh_max_attempts"`
	SSOMaxAttempts      int  `koanf:"sso_max_attempts"`
	WindowSeconds       int  `koanf:"window_seconds"`
	TrustProxy          bool `koanf:"trust_proxy"`
	APIMaxRequests      int  `koanf:"api_max_requests"`
	APIWindowSeconds    int  `koanf:"api_window_seconds"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	k := koanf.New(".")

	// Load from config file if provided
	if configPath != "" {
		if err := k.Load(file.Provider(configPath), toml.Parser()); err != nil {
			return nil, err
		}
	}

	// Load from environment variables (gowa-ui_ prefix). A DOUBLE underscore
	// separates config levels; single underscores are preserved as part of the
	// key. This is required because both section and field names contain
	// underscores (e.g. default_admin, rate_limit) — collapsing every "_" to "."
	// would mangle them (default_admin.email -> default.admin.email), so those
	// keys could never be set via env.
	// e.g. gowa-ui_DATABASE__HOST -> database.host
	//      gowa-ui_DEFAULT_ADMIN__EMAIL -> default_admin.email
	if err := k.Load(env.Provider("gowa-ui_", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, "gowa-ui_")), "__", ".")
	}), nil); err != nil {
		return nil, err
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	setDefaults(&cfg)

	return &cfg, nil
}

func setDefaults(cfg *Config) {
	if cfg.App.Name == "" {
		cfg.App.Name = "gowa-ui"
	}
	if cfg.App.Environment == "" {
		cfg.App.Environment = "development"
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 25
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 5
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		cfg.Database.ConnMaxLifetime = 300
	}
	if cfg.Redis.Port == 0 {
		cfg.Redis.Port = 6379
	}
	if cfg.JWT.AccessExpiryMins == 0 {
		cfg.JWT.AccessExpiryMins = 15
	}
	if cfg.JWT.RefreshExpiryDays == 0 {
		cfg.JWT.RefreshExpiryDays = 1
	}
	if cfg.GOWA.WebhookPath == "" {
		cfg.GOWA.WebhookPath = "/api/gowa/webhook"
	}
	// If the legacy [gowa] section has a base_url, add it as the first instance
	// so single-instance configs keep working without [[gowa_instances]].
	if cfg.GOWA.BaseURL != "" {
		hasDefault := false
		for _, inst := range cfg.GOWAInstances {
			if inst.BaseURL == cfg.GOWA.BaseURL {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			cfg.GOWAInstances = append([]GOWAInstance{{
				Name:     "Default",
				BaseURL:  cfg.GOWA.BaseURL,
				Username: cfg.GOWA.Username,
				Password: cfg.GOWA.Password,
			}}, cfg.GOWAInstances...)
		}
	}
	if cfg.Storage.Type == "" {
		cfg.Storage.Type = "local"
	}
	if cfg.Storage.LocalPath == "" {
		cfg.Storage.LocalPath = "./uploads"
	}
	// Default admin credentials (only used during initial setup)
	if cfg.DefaultAdmin.Email == "" {
		cfg.DefaultAdmin.Email = "admin@admin.com"
	}
	if cfg.DefaultAdmin.Password == "" {
		cfg.DefaultAdmin.Password = "admin"
	}
	if cfg.DefaultAdmin.FullName == "" {
		cfg.DefaultAdmin.FullName = "Admin"
	}
	// Cookie defaults
	if cfg.App.Environment == "production" {
		cfg.Cookie.Secure = true
	}
	// Rate limiting defaults
	if cfg.RateLimit.LoginMaxAttempts == 0 {
		cfg.RateLimit.LoginMaxAttempts = 10
	}
	if cfg.RateLimit.RegisterMaxAttempts == 0 {
		cfg.RateLimit.RegisterMaxAttempts = 10
	}
	if cfg.RateLimit.RefreshMaxAttempts == 0 {
		cfg.RateLimit.RefreshMaxAttempts = 30
	}
	if cfg.RateLimit.SSOMaxAttempts == 0 {
		cfg.RateLimit.SSOMaxAttempts = 10
	}
	if cfg.RateLimit.WindowSeconds == 0 {
		cfg.RateLimit.WindowSeconds = 60
	}
}
