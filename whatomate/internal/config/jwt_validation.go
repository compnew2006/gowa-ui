package config

import (
	"fmt"
	"strings"
)

const minJWTSecretLength = 32

var insecureJWTSecretValues = map[string]struct{}{
	"your-super-secret-jwt-key-change-in-production": {},
	"your-jwt-secret-key":                            {},
	"change-me":                                      {},
	"changeme":                                       {},
	"default":                                        {},
}

// ValidateJWTSecret validates JWT signing secret configuration and fails fast on unsafe values.
func ValidateJWTSecret(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	secret := strings.TrimSpace(cfg.JWT.Secret)
	if secret == "" {
		return fmt.Errorf("jwt.secret must be set and non-empty (generate one with: openssl rand -hex 32)")
	}

	if _, insecure := insecureJWTSecretValues[strings.ToLower(secret)]; insecure {
		return fmt.Errorf("jwt.secret uses an insecure placeholder value; set a unique secret in config.toml or WHATOMATE_JWT_SECRET")
	}

	if len(secret) < minJWTSecretLength {
		return fmt.Errorf("jwt.secret must be at least %d characters", minJWTSecretLength)
	}

	// Persist trimmed value so signing/verification do not depend on accidental whitespace.
	cfg.JWT.Secret = secret

	return nil
}
