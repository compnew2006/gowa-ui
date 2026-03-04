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
	}

	cfg.DefaultAdmin.Email = email
	cfg.DefaultAdmin.Password = password
	return nil
}
