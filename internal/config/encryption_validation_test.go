package config

import (
	"strings"
	"testing"
)

func TestValidateEncryptionKey(t *testing.T) {
	t.Parallel()

	validProdKey := strings.Repeat("k", 32)
	validDevKey := strings.Repeat("d", 32)

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
			name: "development empty key rejected",
			cfg: &Config{
				App: AppConfig{Environment: "development", EncryptionKey: ""},
			},
			wantErr: true,
		},
		{
			name: "placeholder key",
			cfg: &Config{
				App: AppConfig{Environment: "development", EncryptionKey: "change-me"},
			},
			wantErr: true,
		},
		{
			name: "production short key",
			cfg: &Config{
				App: AppConfig{Environment: "production", EncryptionKey: "short-key"},
			},
			wantErr: true,
		},
		{
			name: "development key allowed",
			cfg: &Config{
				App: AppConfig{Environment: "development", EncryptionKey: validDevKey},
			},
			wantErr: false,
		},
		{
			name: "production key allowed",
			cfg: &Config{
				App: AppConfig{Environment: "production", EncryptionKey: validProdKey},
			},
			wantErr: false,
		},
		{
			name: "key is trimmed",
			cfg: &Config{
				App: AppConfig{Environment: "development", EncryptionKey: "   " + validDevKey + "   "},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateEncryptionKey(tc.cfg)
			if tc.wantErr && err == nil {
				t.Fatalf("expected validation error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			if !tc.wantErr && tc.cfg != nil && strings.TrimSpace(tc.cfg.App.EncryptionKey) != tc.cfg.App.EncryptionKey {
				t.Fatalf("expected key to be trimmed, got %q", tc.cfg.App.EncryptionKey)
			}
		})
	}
}
