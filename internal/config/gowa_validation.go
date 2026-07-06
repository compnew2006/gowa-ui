package config

import (
	"fmt"
	"strings"
)

const minGowaWebhookSecretLength = 16

var insecureGowaWebhookSecrets = map[string]struct{}{
	"secret":        {},
	"changeme":      {},
	"change-me":     {},
	"default":       {},
	"password":      {},
	"test":          {},
	"your-secret":   {},
	"webhooksecret": {},
}

// ValidateGowa validates the GOWA HTTP backend configuration. It only enforces
// requirements when the GOWA provider is selected, so deployments that use Meta
// or whatsmeow are not blocked by unrelated GOWA settings.
//
// Hard requirements when cfg.WhatsApp.Provider == "gowa":
//   - GowaConfig.BaseURL must be a non-empty http(s) URL
//   - GowaConfig.WebhookSecret must be set, >= 16 bytes, and not a known insecure value
//   - GowaConfig.WebhookCallbackURL must be a non-empty http(s) URL
//   - RequestTimeoutSeconds and PollingIntervalSeconds must be positive
//
// Basic auth credentials are optional (GOWA can run without basic auth), but
// if a user is set, a password must also be set, and vice versa.
func ValidateGowa(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.WhatsApp.Provider != "gowa" {
		return nil
	}

	baseURL := strings.TrimSpace(cfg.Gowa.BaseURL)
	if baseURL == "" {
		return fmt.Errorf("gowa.base_url must be set when whatsapp.provider=gowa")
	}
	if !isHTTPURL(baseURL) {
		return fmt.Errorf("gowa.base_url must start with http:// or https:// (got %q)", baseURL)
	}
	cfg.Gowa.BaseURL = strings.TrimSuffix(baseURL, "/")

	callbackURL := strings.TrimSpace(cfg.Gowa.WebhookCallbackURL)
	if callbackURL == "" {
		return fmt.Errorf("gowa.webhook_callback_url must be set when whatsapp.provider=gowa (the public URL where GOWA can reach whatomate's /api/gowa/webhook)")
	}
	if !isHTTPURL(callbackURL) {
		return fmt.Errorf("gowa.webhook_callback_url must start with http:// or https:// (got %q)", callbackURL)
	}
	cfg.Gowa.WebhookCallbackURL = strings.TrimSuffix(callbackURL, "/")

	secret := cfg.Gowa.WebhookSecret
	if secret == "" {
		return fmt.Errorf("gowa.webhook_secret must be set (generate one with: openssl rand -hex 32)")
	}
	if _, insecure := insecureGowaWebhookSecrets[strings.ToLower(strings.TrimSpace(secret))]; insecure {
		return fmt.Errorf("gowa.webhook_secret uses an insecure placeholder value; set a unique secret in config.toml or WHATOMATE_GOWA_WEBHOOK_SECRET")
	}
	if len(secret) < minGowaWebhookSecretLength {
		return fmt.Errorf("gowa.webhook_secret must be at least %d characters", minGowaWebhookSecretLength)
	}

	if (cfg.Gowa.BasicAuthUser == "") != (cfg.Gowa.BasicAuthPassword == "") {
		return fmt.Errorf("gowa.basic_auth_user and gowa.basic_auth_password must be set together (either both set or both empty)")
	}

	if cfg.Gowa.RequestTimeoutSeconds <= 0 {
		return fmt.Errorf("gowa.request_timeout_seconds must be > 0 (got %d)", cfg.Gowa.RequestTimeoutSeconds)
	}
	if cfg.Gowa.PollingIntervalSeconds <= 0 {
		return fmt.Errorf("gowa.polling_interval_seconds must be > 0 (got %d)", cfg.Gowa.PollingIntervalSeconds)
	}
	if cfg.Gowa.MaxRetries < 0 {
		return fmt.Errorf("gowa.max_retries must be >= 0 (got %d)", cfg.Gowa.MaxRetries)
	}

	return nil
}

func isHTTPURL(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}
