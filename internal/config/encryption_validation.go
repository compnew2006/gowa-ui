package config

import (
	"fmt"
	"strings"
)

const minProductionEncryptionKeyLength = 32

var insecureEncryptionKeyValues = map[string]struct{}{
	"change-me":     {},
	"changeme":      {},
	"default":       {},
	"your-key":      {},
	"your-secret":   {},
	"encryptionkey": {},
}

// ValidateEncryptionKey validates at-rest encryption configuration and prevents no-op secret storage.
func ValidateEncryptionKey(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	isProduction := strings.EqualFold(strings.TrimSpace(cfg.App.Environment), "production")
	key := strings.TrimSpace(cfg.App.EncryptionKey)
	if key == "" {
		if isProduction {
			return fmt.Errorf("app.encryption_key must be set and non-empty (generate one with: openssl rand -hex 32)")
		}
		cfg.App.EncryptionKey = ""
		return nil
	}

	if _, insecure := insecureEncryptionKeyValues[strings.ToLower(key)]; insecure {
		return fmt.Errorf("app.encryption_key uses an insecure placeholder value; set a unique key in config.toml or WHATOMATE_APP_ENCRYPTION_KEY")
	}

	if isProduction && len(key) < minProductionEncryptionKeyLength {
		return fmt.Errorf("app.encryption_key must be at least %d characters in production", minProductionEncryptionKeyLength)
	}

	// Persist trimmed value so encryption/decryption behavior does not depend on accidental whitespace.
	cfg.App.EncryptionKey = key

	return nil
}
