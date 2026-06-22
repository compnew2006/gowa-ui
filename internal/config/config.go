package config

import (
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var envTopLevelSections = []string{
	"default_admin",
	"rate_limit",
	"whatsmeow",
	"whatsapp",
	"observability",
	"license",
	"facebook_oauth",
	"database",
	"storage",
	"server",
	"cookie",
	"redis",
	"app",
	"jwt",
	"ai",
}

// Config holds all configuration for the application
type Config struct {
	App           AppConfig           `koanf:"app"`
	Server        ServerConfig        `koanf:"server"`
	Database      DatabaseConfig      `koanf:"database"`
	Redis         RedisConfig         `koanf:"redis"`
	JWT           JWTConfig           `koanf:"jwt"`
	WhatsApp      WhatsAppConfig      `koanf:"whatsapp"`
	Whatsmeow     WhatsmeowConfig     `koanf:"whatsmeow"`
	Observability ObservabilityConfig `koanf:"observability"`
	AI            AIConfig            `koanf:"ai"`
	Storage       StorageConfig       `koanf:"storage"`
	DefaultAdmin  DefaultAdminConfig  `koanf:"default_admin"`
	RateLimit     RateLimitConfig     `koanf:"rate_limit"`
	Campaigns     CampaignsConfig     `koanf:"campaigns"`
	Cookie        CookieConfig        `koanf:"cookie"`
	License       LicenseConfig       `koanf:"license"`
	FacebookOAuth FacebookOAuthConfig `koanf:"facebook_oauth"`
	Facebook      FacebookConfig      `koanf:"facebook"`
}

type FacebookConfig struct {
	AccessToken string `koanf:"access_token"` // Optional: server-side default access token used by /api/facebook/page-search
	APIVersion  string `koanf:"api_version"`
	BaseURL     string `koanf:"base_url"`
}

type AppConfig struct {
	Name                           string `koanf:"name"`
	Environment                    string `koanf:"environment"` // development, staging, production
	Debug                          bool   `koanf:"debug"`
	EncryptionKey                  string `koanf:"encryption_key"` // AES-256 key for encrypting secrets at rest
	AllowLegacyEncryption          *bool  `koanf:"allow_legacy_encryption"`
	SandboxMode                    bool   `koanf:"sandbox_mode"`                      // Shared-data sandbox guardrail that disables startup mutations and background automation.
	SandboxAllowWhatsmeowReconnect bool   `koanf:"sandbox_allow_whatsmeow_reconnect"` // Narrow sandbox override for receiving WhatsApp Web messages without enabling jobs.
}

type ServerConfig struct {
	Host                 string `koanf:"host"`
	Port                 int    `koanf:"port"`
	ReadTimeout          int    `koanf:"read_timeout"`
	WriteTimeout         int    `koanf:"write_timeout"`
	MaxRequestBodySizeMB int    `koanf:"max_request_body_size_mb"`
	BasePath             string `koanf:"base_path"`       // Base path for frontend (e.g., "/whatomate" for proxy pass)
	AllowedOrigins       string `koanf:"allowed_origins"` // Comma-separated list of allowed CORS origins
}

type DatabaseConfig struct {
	Host            string `koanf:"host"`
	Port            int    `koanf:"port"`
	User            string `koanf:"user"`
	Password        string `koanf:"password"`
	Name            string `koanf:"name"`
	SSLMode         string `koanf:"ssl_mode"`
	LogSQL          bool   `koanf:"log_sql"`
	MaxOpenConns    int    `koanf:"max_open_conns"`
	MaxIdleConns    int    `koanf:"max_idle_conns"`
	ConnMaxLifetime int    `koanf:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Password string `koanf:"password"`
	DB       int    `koanf:"db"`
}

type JWTConfig struct {
	Secret            string `koanf:"secret"`
	AccessExpiryMins  int    `koanf:"access_expiry_mins"`
	RefreshExpiryDays int    `koanf:"refresh_expiry_days"`
}

type WhatsAppConfig struct {
	WebhookVerifyToken string `koanf:"webhook_verify_token"`
	APIVersion         string `koanf:"api_version"`
	BaseURL            string `koanf:"base_url"` // Meta Graph API base URL
	Provider           string `koanf:"provider"` // meta, whatsmeow
}

type WhatsmeowConfig struct {
	RateLimitMinDelayMs              int    `koanf:"rate_limit_min_delay_ms"`
	RateLimitMaxDelayMs              int    `koanf:"rate_limit_max_delay_ms"`
	QueueTimeoutSeconds              int    `koanf:"queue_timeout_seconds"`
	MaxInstancesPerOrg               int    `koanf:"max_instances_per_org"`
	UploadRetryCount                 int    `koanf:"upload_retry_count"`
	UploadRetryDelaySec              int    `koanf:"upload_retry_delay_sec"`
	InboundMediaRetryCount           int    `koanf:"inbound_media_retry_count"`
	InboundMediaRetryDelayMs         int    `koanf:"inbound_media_retry_delay_ms"`
	InboundMediaRetryMaxDelayMs      int    `koanf:"inbound_media_retry_max_delay_ms"`
	InboundMediaAsyncRetryCount      int    `koanf:"inbound_media_async_retry_count"`
	InboundMediaAsyncRetryDelayMs    int    `koanf:"inbound_media_async_retry_delay_ms"`
	InboundMediaAsyncRetryMaxDelayMs int    `koanf:"inbound_media_async_retry_max_delay_ms"`
	DeferInboundMedia                *bool  `koanf:"defer_inbound_media"`
	InboundMediaWorkerConcurrency    int    `koanf:"inbound_media_worker_concurrency"`
	InboundMediaQueueNamespace       string `koanf:"inbound_media_queue_namespace"`
	EventBufferSize                  int    `koanf:"event_buffer_size"`
	EventDispatchEnabled             *bool  `koanf:"event_dispatch_enabled"`
	Identity                         string `koanf:"identity"` // Optional prefix for linked device label (e.g. "whats")
	HealthMonitorIntervalSeconds     int    `koanf:"health_monitor_interval_seconds"`
	ReconnectTimeoutSeconds          int    `koanf:"reconnect_timeout_seconds"`
	// ForceIPv4 pins all whatsmeow HTTP clients (websocket, media, pre-login)
	// to IPv4-only dialing. nil/false preserves the default dual-stack behaviour.
	// Enable on deployments where the IPv6 peering path to WhatsApp/Meta is
	// flaky (e.g. observed TCP resets on Hostinger -> face:b00c). The resolver
	// still returns A and AAAA records; the dialer filters to "tcp4" only.
	ForceIPv4                        *bool  `koanf:"force_ipv4"`
	TypingIndicatorEnabled           bool   `koanf:"typing_indicator_enabled"`
	TypingMinDelayMs                 int    `koanf:"typing_min_delay_ms"`
	TypingMaxDelayMs                 int    `koanf:"typing_max_delay_ms"`
	TypingCharDelayMs                int    `koanf:"typing_char_delay_ms"`
	TypingCooldownMs                 int    `koanf:"typing_cooldown_ms"`

	// Priority event queue fields. Only used when PriorityQueuesEnabled=true.
	PriorityQueuesEnabled          *bool `koanf:"priority_queues_enabled"`
	EventMsgQueueSize              int   `koanf:"event_msg_queue_size"`
	EventLowQueueSize              int   `koanf:"event_low_queue_size"`
	EventMsgShards                 int   `koanf:"event_msg_shards"`
	EventLowWorkers                int   `koanf:"event_low_workers"`
	EventHighEnqueueTimeoutMs      int   `koanf:"event_high_enqueue_timeout_ms"`
	EventShutdownDrainTimeoutSeconds int `koanf:"event_shutdown_drain_timeout_seconds"`
	EventCircuitBreakerRatePerMinute       int `koanf:"event_circuit_breaker_rate_per_minute"`
	EventCircuitBreakerConsecutiveWindows int `koanf:"event_circuit_breaker_consecutive_windows"`
	EventCircuitBreakerCooldownSeconds    int `koanf:"event_circuit_breaker_cooldown_seconds"`
}

type ObservabilityConfig struct {
	EnableMetrics bool   `koanf:"enable_metrics"`
	EnablePprof   bool   `koanf:"enable_pprof"`
	AccessToken   string `koanf:"access_token"`
}

type AIConfig struct {
	OpenAIKey    string `koanf:"openai_key"`
	AnthropicKey string `koanf:"anthropic_key"`
	GoogleKey    string `koanf:"google_key"`
}

type StorageConfig struct {
	Type       string `koanf:"type"` // local, s3
	LocalPath  string `koanf:"local_path"`
	S3Bucket   string `koanf:"s3_bucket"`
	S3Region   string `koanf:"s3_region"`
	S3Key      string `koanf:"s3_key"`
	S3Secret   string `koanf:"s3_secret"`
	S3Endpoint string `koanf:"s3_endpoint"`
	S3UseSSL   bool   `koanf:"s3_use_ssl"`
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
	Enabled                     bool `koanf:"enabled"`
	LoginMaxAttempts            int  `koanf:"login_max_attempts"`
	RegisterMaxAttempts         int  `koanf:"register_max_attempts"`
	RefreshMaxAttempts          int  `koanf:"refresh_max_attempts"`
	SSOMaxAttempts              int  `koanf:"sso_max_attempts"`
	WebhookMaxAttempts          int  `koanf:"webhook_max_attempts"`
	WindowSeconds               int  `koanf:"window_seconds"`
	TrustProxy                  bool `koanf:"trust_proxy"`
	OutboundPerUserPS           int  `koanf:"outbound_per_user_per_second"`
	OutboundPerIPPS             int  `koanf:"outbound_per_ip_per_second"`
	CampaignMutatingMaxAttempts int  `koanf:"campaign_mutating_max_attempts"`
}

type CampaignsConfig struct {
	MaxImportRecipients int `koanf:"max_import_recipients"`
}

type LicenseConfig struct {
	Enabled                      bool     `koanf:"enabled"`
	PublicKey                    string   `koanf:"public_key"`
	PublicKeyKID                 string   `koanf:"public_key_kid"`
	AllowUnsafePublicKeyOverride bool     `koanf:"allow_unsafe_public_key_override"`
	FingerprintSources           []string `koanf:"fingerprint_sources"`
	HostMachineIDPath            string   `koanf:"host_machine_id_path"`
	RollbackToleranceSeconds     int      `koanf:"rollback_tolerance_seconds"`
	GracePeriodDays              int      `koanf:"grace_period_days"`
	EnforceOnWorkers             *bool    `koanf:"enforce_on_workers"`
}

type FacebookOAuthConfig struct {
	AppID              string `koanf:"app_id"`
	AppSecret          string `koanf:"app_secret"`
	APIVersion         string `koanf:"api_version"`
	RedirectURI        string `koanf:"redirect_uri"`
	BaseURL            string `koanf:"base_url"`
	WebhookVerifyToken string `koanf:"webhook_verify_token"`
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

	// Load from environment variables (WHATOMATE_ prefix)
	// e.g., WHATOMATE_DATABASE_HOST -> database.host
	if err := k.Load(env.Provider("WHATOMATE_", ".", func(s string) string {
		return envKeyToKoanfPath(s)
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

func envKeyToKoanfPath(raw string) string {
	key := strings.ToLower(strings.TrimPrefix(raw, "WHATOMATE_"))
	if key == "" {
		return ""
	}

	for _, section := range envTopLevelSections {
		if key == section {
			return section
		}
		prefix := section + "_"
		if strings.HasPrefix(key, prefix) {
			remainder := strings.TrimPrefix(key, prefix)
			if remainder == "" {
				return section
			}
			return section + "." + remainder
		}
	}

	// Fallback for unknown keys to preserve legacy behavior.
	return strings.ReplaceAll(key, "_", ".")
}

func setDefaults(cfg *Config) {
	if cfg.App.Name == "" {
		cfg.App.Name = "Whatomate"
	}
	if cfg.App.Environment == "" {
		cfg.App.Environment = "development"
	}
	if cfg.App.AllowLegacyEncryption == nil {
		allowLegacy := true
		cfg.App.AllowLegacyEncryption = &allowLegacy
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
	if cfg.Server.MaxRequestBodySizeMB == 0 {
		cfg.Server.MaxRequestBodySizeMB = 110
	}
	cfg.Server.BasePath = normalizeBasePath(cfg.Server.BasePath)
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
	if cfg.WhatsApp.APIVersion == "" {
		cfg.WhatsApp.APIVersion = "v18.0"
	}
	if cfg.WhatsApp.BaseURL == "" {
		cfg.WhatsApp.BaseURL = "https://graph.facebook.com"
	}
	if cfg.WhatsApp.Provider == "" {
		cfg.WhatsApp.Provider = "meta"
	}
	if cfg.FacebookOAuth.APIVersion == "" {
		cfg.FacebookOAuth.APIVersion = "v20.0"
	}
	if cfg.FacebookOAuth.BaseURL == "" {
		cfg.FacebookOAuth.BaseURL = "https://graph.facebook.com"
	}
	if cfg.Whatsmeow.RateLimitMinDelayMs == 0 {
		cfg.Whatsmeow.RateLimitMinDelayMs = 1000
	}
	if cfg.Whatsmeow.RateLimitMaxDelayMs == 0 {
		cfg.Whatsmeow.RateLimitMaxDelayMs = 3000
	}
	if cfg.Whatsmeow.QueueTimeoutSeconds == 0 {
		cfg.Whatsmeow.QueueTimeoutSeconds = 300
	}
	if cfg.Whatsmeow.MaxInstancesPerOrg == 0 {
		cfg.Whatsmeow.MaxInstancesPerOrg = 5
	}
	if cfg.Whatsmeow.UploadRetryCount == 0 {
		cfg.Whatsmeow.UploadRetryCount = 1
	}
	if cfg.Whatsmeow.UploadRetryDelaySec == 0 {
		cfg.Whatsmeow.UploadRetryDelaySec = 2
	}
	if cfg.Whatsmeow.InboundMediaRetryCount == 0 {
		cfg.Whatsmeow.InboundMediaRetryCount = 3
	}
	if cfg.Whatsmeow.InboundMediaRetryDelayMs == 0 {
		cfg.Whatsmeow.InboundMediaRetryDelayMs = 500
	}
	if cfg.Whatsmeow.InboundMediaRetryMaxDelayMs == 0 {
		cfg.Whatsmeow.InboundMediaRetryMaxDelayMs = 2000
	}
	if cfg.Whatsmeow.InboundMediaAsyncRetryCount == 0 {
		cfg.Whatsmeow.InboundMediaAsyncRetryCount = 4
	}
	if cfg.Whatsmeow.InboundMediaAsyncRetryDelayMs == 0 {
		cfg.Whatsmeow.InboundMediaAsyncRetryDelayMs = 5000
	}
	if cfg.Whatsmeow.InboundMediaAsyncRetryMaxDelayMs == 0 {
		cfg.Whatsmeow.InboundMediaAsyncRetryMaxDelayMs = 60000
	}
	if cfg.Whatsmeow.DeferInboundMedia == nil {
		deferMedia := true
		cfg.Whatsmeow.DeferInboundMedia = &deferMedia
	}
	if cfg.Whatsmeow.InboundMediaWorkerConcurrency == 0 {
		cfg.Whatsmeow.InboundMediaWorkerConcurrency = 4
	}
	if cfg.Whatsmeow.EventBufferSize == 0 {
		cfg.Whatsmeow.EventBufferSize = 4096
	}
	if cfg.Whatsmeow.EventDispatchEnabled == nil {
		enabled := true
		cfg.Whatsmeow.EventDispatchEnabled = &enabled
	}
	if cfg.Whatsmeow.HealthMonitorIntervalSeconds == 0 {
		cfg.Whatsmeow.HealthMonitorIntervalSeconds = 30
	}
	if cfg.Whatsmeow.ReconnectTimeoutSeconds == 0 {
		cfg.Whatsmeow.ReconnectTimeoutSeconds = 45
	}
	if !cfg.Whatsmeow.TypingIndicatorEnabled {
		// Default to enabled to improve human-like direct chat sends unless explicitly disabled in config.
		cfg.Whatsmeow.TypingIndicatorEnabled = true
	}
	if cfg.Whatsmeow.TypingMinDelayMs == 0 {
		cfg.Whatsmeow.TypingMinDelayMs = 700
	}
	if cfg.Whatsmeow.TypingMaxDelayMs == 0 {
		cfg.Whatsmeow.TypingMaxDelayMs = 3000
	}
	if cfg.Whatsmeow.TypingCharDelayMs == 0 {
		cfg.Whatsmeow.TypingCharDelayMs = 35
	}
	if cfg.Whatsmeow.TypingCooldownMs == 0 {
		cfg.Whatsmeow.TypingCooldownMs = 4000
	}
	// Priority event queue defaults
	if cfg.Whatsmeow.EventMsgQueueSize == 0 {
		cfg.Whatsmeow.EventMsgQueueSize = 2048
	}
	if cfg.Whatsmeow.EventLowQueueSize == 0 {
		cfg.Whatsmeow.EventLowQueueSize = 512
	}
	if cfg.Whatsmeow.EventMsgShards == 0 {
		cfg.Whatsmeow.EventMsgShards = 4
	}
	if cfg.Whatsmeow.EventLowWorkers == 0 {
		cfg.Whatsmeow.EventLowWorkers = 2
	}
	if cfg.Whatsmeow.EventHighEnqueueTimeoutMs == 0 {
		cfg.Whatsmeow.EventHighEnqueueTimeoutMs = 10
	}
	if cfg.Whatsmeow.EventShutdownDrainTimeoutSeconds == 0 {
		cfg.Whatsmeow.EventShutdownDrainTimeoutSeconds = 5
	}
	if cfg.Whatsmeow.EventCircuitBreakerRatePerMinute == 0 {
		cfg.Whatsmeow.EventCircuitBreakerRatePerMinute = 60
	}
	if cfg.Whatsmeow.EventCircuitBreakerConsecutiveWindows == 0 {
		cfg.Whatsmeow.EventCircuitBreakerConsecutiveWindows = 2
	}
	if cfg.Whatsmeow.EventCircuitBreakerCooldownSeconds == 0 {
		cfg.Whatsmeow.EventCircuitBreakerCooldownSeconds = 300
	}
	if cfg.Storage.Type == "" {
		cfg.Storage.Type = "local"
	}
	if cfg.Storage.LocalPath == "" {
		cfg.Storage.LocalPath = "./uploads"
	}
	// Default admin bootstrap metadata (credentials must be explicitly configured).
	if cfg.DefaultAdmin.FullName == "" {
		cfg.DefaultAdmin.FullName = "Admin"
	}
	// Cookie defaults
	if strings.EqualFold(strings.TrimSpace(cfg.App.Environment), "production") {
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
	if cfg.RateLimit.WebhookMaxAttempts == 0 {
		cfg.RateLimit.WebhookMaxAttempts = 300
	}
	if cfg.RateLimit.WindowSeconds == 0 {
		cfg.RateLimit.WindowSeconds = 60
	}
	if cfg.RateLimit.OutboundPerUserPS == 0 {
		cfg.RateLimit.OutboundPerUserPS = 5
	}
	if cfg.RateLimit.OutboundPerIPPS == 0 {
		cfg.RateLimit.OutboundPerIPPS = 15
	}
	if cfg.RateLimit.CampaignMutatingMaxAttempts == 0 {
		cfg.RateLimit.CampaignMutatingMaxAttempts = 60
	}
	if cfg.Campaigns.MaxImportRecipients == 0 {
		cfg.Campaigns.MaxImportRecipients = 10000
	}
	if cfg.License.RollbackToleranceSeconds == 0 {
		cfg.License.RollbackToleranceSeconds = 900
	}
	if cfg.License.GracePeriodDays == 0 {
		cfg.License.GracePeriodDays = 7
	}
	if len(cfg.License.FingerprintSources) == 0 {
		cfg.License.FingerprintSources = []string{
			"/etc/machine-id",
			"/sys/class/dmi/id/product_uuid",
		}
	}
	if cfg.License.EnforceOnWorkers == nil {
		enforceOnWorkers := true
		cfg.License.EnforceOnWorkers = &enforceOnWorkers
	}
}

func normalizeBasePath(basePath string) string {
	trimmed := strings.TrimSpace(basePath)
	if trimmed == "" || trimmed == "." || trimmed == "./" || trimmed == "/" {
		return ""
	}

	trimmed = strings.TrimSuffix(trimmed, "/")
	if trimmed == "" || trimmed == "." {
		return ""
	}

	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}

	return "/" + strings.TrimPrefix(trimmed, "./")
}
