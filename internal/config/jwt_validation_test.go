package config

import (
	"strings"
	"testing"
)

func TestValidateJWTSecret(t *testing.T) {
	t.Parallel()

	validProdSecret := strings.Repeat("x", 32)

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "empty secret",
			cfg: &Config{
				App: AppConfig{Environment: "development"},
				JWT: JWTConfig{Secret: ""},
			},
			wantErr: true,
		},
		{
			name: "placeholder secret",
			cfg: &Config{
				App: AppConfig{Environment: "development"},
				JWT: JWTConfig{Secret: "your-jwt-secret-key"},
			},
			wantErr: true,
		},
		{
			name: "production short secret",
			cfg: &Config{
				App: AppConfig{Environment: "production"},
				JWT: JWTConfig{Secret: "short-secret"},
			},
			wantErr: true,
		},
		{
			name: "development custom secret",
			cfg: &Config{
				App: AppConfig{Environment: "development"},
				JWT: JWTConfig{Secret: "dev-local-secret"},
			},
			wantErr: false,
		},
		{
			name: "production strong secret",
			cfg: &Config{
				App: AppConfig{Environment: "production"},
				JWT: JWTConfig{Secret: validProdSecret},
			},
			wantErr: false,
		},
		{
			name: "secret gets trimmed",
			cfg: &Config{
				App: AppConfig{Environment: "development"},
				JWT: JWTConfig{Secret: "   dev-local-secret   "},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateJWTSecret(tc.cfg)
			if tc.wantErr && err == nil {
				t.Fatalf("expected validation error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			if !tc.wantErr && tc.cfg != nil && strings.TrimSpace(tc.cfg.JWT.Secret) != tc.cfg.JWT.Secret {
				t.Fatalf("expected secret to be trimmed, got %q", tc.cfg.JWT.Secret)
			}
		})
	}
}
