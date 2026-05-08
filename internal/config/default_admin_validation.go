package config

import (
	"fmt"
	"strings"
)

var insecureDefaultAdminEmails = map[string]struct{}{
	"admin@admin.com": {},
}

var insecureDefaultAdminPasswords = map[string]struct{}{
	"admin":     {},
	"changeme":  {},
	"change-me": {},
	"default":   {},
}

// ValidateDefaultAdmin validates optional bootstrap admin credentials.
// Empty email+password means bootstrap creation is disabled.
func ValidateDefaultAdmin(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	email := strings.TrimSpace(cfg.DefaultAdmin.Email)
	password := strings.TrimSpace(cfg.DefaultAdmin.Password)
	isProduction := strings.EqualFold(strings.TrimSpace(cfg.App.Environment), "production")

	if email == "" && password == "" {
		cfg.DefaultAdmin.Email = ""
		cfg.DefaultAdmin.Password = ""
		return nil
	}
	if email == "" || password == "" {
		return fmt.Errorf("default_admin.email and default_admin.password must be both set or both empty")
	}

	if isProduction {
		if _, insecure := insecureDefaultAdminEmails[strings.ToLower(email)]; insecure {
			return fmt.Errorf("default_admin.email uses an insecure placeholder; set a unique bootstrap admin email or leave default_admin empty")
		}
		if _, insecure := insecureDefaultAdminPasswords[strings.ToLower(password)]; insecure {
			return fmt.Errorf("default_admin.password uses an insecure placeholder; set a strong bootstrap password or leave default_admin empty")
		}
		if len(password) < 12 {
			return fmt.Errorf("default_admin.password must be at least 12 characters in production")
		}
		if err := validatePasswordComplexity(password); err != nil {
			return fmt.Errorf("default_admin.password: %w", err)
		}
	}

	cfg.DefaultAdmin.Email = email
	cfg.DefaultAdmin.Password = password
	return nil
}

// validatePasswordComplexity checks that a password contains at least one
// uppercase letter, one lowercase letter, and one digit.
func validatePasswordComplexity(password string) error {
	var hasLower, hasUpper, hasDigit bool
	for _, ch := range password {
		switch {
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		}
	}
	if !hasLower || !hasUpper || !hasDigit {
		return fmt.Errorf("must include at least one uppercase letter, one lowercase letter, and one number")
	}
	return nil
}
