package config

import (
	"fmt"
	"strings"
)

const minEncryptionKeyLength = 32

var insecureEncryptionKeyValues = map[string]struct{}{
	"change-me":     {},
	"changeme":      {},
	"default":       {},
	"dev":           {},
	"development":   {},
	"local":         {},
	"password":      {},
	"secret":        {},
	"test":          {},
	"your-key":      {},
	"your-secret":   {},
	"encryptionkey": {},
}

// ValidateEncryptionKey validates at-rest encryption configuration and prevents no-op secret storage.
func ValidateEncryptionKey(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	key := strings.TrimSpace(cfg.App.EncryptionKey)
	if key == "" {
		return fmt.Errorf("app.encryption_key must be set and non-empty (generate one with: openssl rand -hex 32)")
	}

	if _, insecure := insecureEncryptionKeyValues[strings.ToLower(key)]; insecure {
		return fmt.Errorf("app.encryption_key uses an insecure placeholder value; set a unique key in config.toml or WHATOMATE_APP_ENCRYPTION_KEY")
	}

	if len(key) < minEncryptionKeyLength {
		return fmt.Errorf("app.encryption_key must be at least %d characters", minEncryptionKeyLength)
	}

	// Persist trimmed value so encryption/decryption behavior does not depend on accidental whitespace.
	cfg.App.EncryptionKey = key

	return nil
}
