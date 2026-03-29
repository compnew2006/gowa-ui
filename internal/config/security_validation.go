package config

import (
	"fmt"
	"strings"
)

var insecureDatabaseUsers = map[string]struct{}{
	"admin":     {},
	"postgres":  {},
	"root":      {},
	"user":      {},
	"whatomate": {},
}

var insecureDatabasePasswords = map[string]struct{}{
	"admin":     {},
	"changeme":  {},
	"change-me": {},
	"default":   {},
	"password":  {},
	"postgres":  {},
	"root":      {},
	"secret":    {},
	"whatomate": {},
}

var insecureWebhookVerifyTokens = map[string]struct{}{
	"changeme":  {},
	"change-me": {},
	"default":   {},
	"secret":    {},
	"test":      {},
}

func isProductionEnv(cfg *Config) bool {
	if cfg == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(cfg.App.Environment), "production")
}

// ValidateDatabaseCredentials rejects insecure database credentials in production.
func ValidateDatabaseCredentials(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if !isProductionEnv(cfg) {
		return nil
	}

	user := strings.TrimSpace(cfg.Database.User)
	password := strings.TrimSpace(cfg.Database.Password)
	if _, insecure := insecureDatabaseUsers[strings.ToLower(user)]; insecure {
		return fmt.Errorf("database.user uses an insecure default value; set a unique database user")
	}
	if _, insecure := insecureDatabasePasswords[strings.ToLower(password)]; insecure {
		return fmt.Errorf("database.password uses an insecure default value; set a unique database password")
	}

	return nil
}

// ValidateWebhookVerifyToken rejects weak verify tokens in production.
func ValidateWebhookVerifyToken(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	token := strings.TrimSpace(cfg.WhatsApp.WebhookVerifyToken)
	if token == "" {
		if isProductionEnv(cfg) {
			return fmt.Errorf("whatsapp.webhook_verify_token must be set in production")
		}
		return nil
	}
	if isProductionEnv(cfg) {
		if _, insecure := insecureWebhookVerifyTokens[strings.ToLower(token)]; insecure {
			return fmt.Errorf("whatsapp.webhook_verify_token uses an insecure placeholder value; set a unique token")
		}
	}

	return nil
}
