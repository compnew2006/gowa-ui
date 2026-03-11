package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateDefaultAdmin_NilConfig tests validation with nil config
func TestValidateDefaultAdmin_NilConfig(t *testing.T) {
	t.Parallel()

	err := ValidateDefaultAdmin(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is nil")
}

// TestValidateDefaultAdmin_EmptyCredentials tests validation with empty credentials
func TestValidateDefaultAdmin_EmptyCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{
			name:     "both empty",
			email:    "",
			password: "",
		},
		{
			name:     "both whitespace",
			email:    "   ",
			password: "   ",
		},
		{
			name:     "email whitespace, password empty",
			email:    "  ",
			password: "",
		},
		{
			name:     "email empty, password whitespace",
			email:    "",
			password: "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				DefaultAdmin: DefaultAdminConfig{
					Email:    tt.email,
					Password: tt.password,
				},
			}

			err := ValidateDefaultAdmin(cfg)
			assert.NoError(t, err, "Empty credentials should be valid")

			// Verify credentials are normalized to empty strings
			assert.Equal(t, "", cfg.DefaultAdmin.Email)
			assert.Equal(t, "", cfg.DefaultAdmin.Password)
		})
	}
}

// TestValidateDefaultAdmin_PartialCredentials tests validation with only email or password set
func TestValidateDefaultAdmin_PartialCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{
			name:     "only email set",
			email:    "admin@example.com",
			password: "",
		},
		{
			name:     "only password set",
			email:    "",
			password: "password123",
		},
		{
			name:     "email set, password whitespace",
			email:    "admin@example.com",
			password: "  ",
		},
		{
			name:     "email whitespace, password set",
			email:    "  ",
			password: "password123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				DefaultAdmin: DefaultAdminConfig{
					Email:    tt.email,
					Password: tt.password,
				},
			}

			err := ValidateDefaultAdmin(cfg)
			assert.Error(t, err, "Partial credentials should error")
			assert.Contains(t, err.Error(), "must be both set or both empty")
		})
	}
}

// TestValidateDefaultAdmin_ValidCredentials tests validation with valid credentials
func TestValidateDefaultAdmin_ValidCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		email          string
		password       string
		env            string
		expectErr      bool
		expectEmail    string
		expectPassword string
	}{
		{
			name:           "valid credentials development",
			email:          "admin@example.com",
			password:       "password123",
			env:            "development",
			expectErr:      false,
			expectEmail:    "admin@example.com",
			expectPassword: "password123",
		},
		{
			name:           "valid credentials staging",
			email:          "admin@staging.com",
			password:       "stagingpass123",
			env:            "staging",
			expectErr:      false,
			expectEmail:    "admin@staging.com",
			expectPassword: "stagingpass123",
		},
		{
			name:           "valid credentials with whitespace",
			email:          "  admin@example.com  ",
			password:       "  password123  ",
			env:            "development",
			expectErr:      false,
			expectEmail:    "admin@example.com",
			expectPassword: "password123",
		},
		{
			name:           "valid credentials mixed case",
			email:          "Admin@Example.COM",
			password:       "Password123",
			env:            "development",
			expectErr:      false,
			expectEmail:    "Admin@Example.COM",
			expectPassword: "Password123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				App: AppConfig{
					Environment: tt.env,
				},
				DefaultAdmin: DefaultAdminConfig{
					Email:    tt.email,
					Password: tt.password,
				},
			}

			err := ValidateDefaultAdmin(cfg)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify credentials are trimmed
				if tt.expectEmail != "" {
					assert.Equal(t, tt.expectEmail, cfg.DefaultAdmin.Email)
				} else {
					assert.Equal(t, "admin@example.com", cfg.DefaultAdmin.Email)
				}
				if tt.expectPassword != "" {
					assert.Equal(t, tt.expectPassword, cfg.DefaultAdmin.Password)
				} else {
					assert.Equal(t, "password123", cfg.DefaultAdmin.Password)
				}
			}
		})
	}
}

// TestValidateDefaultAdmin_ProductionInsecureEmail tests production validation with insecure emails
func TestValidateDefaultAdmin_ProductionInsecureEmail(t *testing.T) {
	t.Parallel()

	insecureEmails := []string{
		"admin@admin.com",
		"ADMIN@ADMIN.COM",
		"Admin@Admin.Com",
		"  admin@admin.com  ",
	}

	for _, email := range insecureEmails {
		t.Run(email, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				App: AppConfig{
					Environment: "production",
				},
				DefaultAdmin: DefaultAdminConfig{
					Email:    email,
					Password: "securepassword123456",
				},
			}

			err := ValidateDefaultAdmin(cfg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "insecure placeholder")
		})
	}
}

// TestValidateDefaultAdmin_ProductionInsecurePassword tests production validation with insecure passwords
func TestValidateDefaultAdmin_ProductionInsecurePassword(t *testing.T) {
	t.Parallel()

	insecurePasswords := []string{
		"admin",
		"changeme",
		"change-me",
		"default",
		"ADMIN",
		"CHANGEME",
		"CHANGE-ME",
		"DEFAULT",
		"  admin  ",
	}

	for _, password := range insecurePasswords {
		t.Run(password, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				App: AppConfig{
					Environment: "production",
				},
				DefaultAdmin: DefaultAdminConfig{
					Email:    "secure@example.com",
					Password: password,
				},
			}

			err := ValidateDefaultAdmin(cfg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "insecure placeholder")
		})
	}
}

// TestValidateDefaultAdmin_ProductionShortPassword tests production validation with short passwords
func TestValidateDefaultAdmin_ProductionShortPassword(t *testing.T) {
	t.Parallel()

	shortPasswords := []string{
		"short",
		"1234567890",  // 10 chars
		"short1",       // 6 chars
	}

	for _, password := range shortPasswords {
		t.Run(password, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				App: AppConfig{
					Environment: "production",
				},
				DefaultAdmin: DefaultAdminConfig{
					Email:    "admin@example.com",
					Password: password,
				},
			}

			err := ValidateDefaultAdmin(cfg)
			assert.Error(t, err)

			// Should get password length error, not insecure placeholder error
			assert.Contains(t, err.Error(), "at least 12 characters")
		})
	}
}

// TestValidateDefaultAdmin_ProductionValidPassword tests production validation with valid passwords
func TestValidateDefaultAdmin_ProductionValidPassword(t *testing.T) {
	t.Parallel()

	validPasswords := []string{
		"securePassword123",      // exactly 12 chars
		"verySecurePassword456!",  // more than 12 chars
		"admin12345678",           // 12 chars
		"p@ssw0rd12345",            // 12 chars with special char
	}

	for _, password := range validPasswords {
		t.Run(password, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				App: AppConfig{
					Environment: "production",
				},
				DefaultAdmin: DefaultAdminConfig{
					Email:    "admin@example.com",
					Password: password,
				},
			}

			err := ValidateDefaultAdmin(cfg)
			assert.NoError(t, err)
		})
	}
}

// TestValidateDefaultAdmin_NonProductionInsecureCredentials tests non-production environments allow insecure credentials
func TestValidateDefaultAdmin_NonProductionInsecureCredentials(t *testing.T) {
	t.Parallel()

	environments := []string{
		"development",
		"staging",
		"test",
		"local",
	}

	for _, env := range environments {
		t.Run(env, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				App: AppConfig{
					Environment: env,
				},
				DefaultAdmin: DefaultAdminConfig{
					Email:    "admin@admin.com",
					Password: "admin",
				},
			}

			err := ValidateDefaultAdmin(cfg)
			assert.NoError(t, err, "Non-production environments should allow insecure credentials")
		})
	}
}

// TestValidateDefaultAdmin_ProductionEnvironmentCaseInsensitive tests environment check is case-insensitive
func TestValidateDefaultAdmin_ProductionEnvironmentCaseInsensitive(t *testing.T) {
	t.Parallel()

	productionVariants := []string{
		"production",
		"PRODUCTION",
		"Production",
		"pRoDuCtIoN",
		"  production  ",
	}

	for _, env := range productionVariants {
		t.Run(env, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				App: AppConfig{
					Environment: env,
				},
				DefaultAdmin: DefaultAdminConfig{
					Email:    "admin@admin.com",
					Password: "securepassword123456",
				},
			}

			err := ValidateDefaultAdmin(cfg)
			assert.Error(t, err, "Production (any case) should reject insecure email")
			assert.Contains(t, err.Error(), "insecure placeholder")
		})
	}
}

// TestValidateDefaultAdmin_SecureEmailInProduction tests production accepts secure emails
func TestValidateDefaultAdmin_SecureEmailInProduction(t *testing.T) {
	t.Parallel()

	secureEmails := []string{
		"admin@mycompany.com",
		"superadmin@example.org",
		"root@internal.net",
		"unique-admin@production.io",
	}

	for _, email := range secureEmails {
		t.Run(email, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				App: AppConfig{
					Environment: "production",
				},
				DefaultAdmin: DefaultAdminConfig{
					Email:    email,
					Password: "securepassword123456",
				},
			}

			err := ValidateDefaultAdmin(cfg)
			assert.NoError(t, err)
		})
	}
}

// TestValidateDefaultAdmin_CredentialNormalization tests that credentials are properly normalized
func TestValidateDefaultAdmin_CredentialNormalization(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		App: AppConfig{
			Environment: "development",
		},
		DefaultAdmin: DefaultAdminConfig{
			Email:    "  Admin@Example.COM  ",
			Password: "  Password123  ",
		},
	}

	err := ValidateDefaultAdmin(cfg)
	require.NoError(t, err)

	// Verify normalization happened
	assert.Equal(t, "Admin@Example.COM", cfg.DefaultAdmin.Email, "Email should be trimmed but not lowercased")
	assert.Equal(t, "Password123", cfg.DefaultAdmin.Password, "Password should be trimmed")
}

// TestValidateDefaultAdmin_EmailCaseSensitivity tests that email case is preserved
func TestValidateDefaultAdmin_EmailCaseSensitivity(t *testing.T) {
	t.Parallel()

	// In non-production, case should be preserved
	cfg := &Config{
		App: AppConfig{
			Environment: "development",
		},
		DefaultAdmin: DefaultAdminConfig{
			Email:    "Admin@Example.COM",
			Password: "password123",
		},
	}

	err := ValidateDefaultAdmin(cfg)
	require.NoError(t, err)

	assert.Equal(t, "Admin@Example.COM", cfg.DefaultAdmin.Email, "Email case should be preserved")
}

// TestValidateDefaultAdmin_ConfigMutation tests that the config is mutated
func TestValidateDefaultAdmin_ConfigMutation(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		App: AppConfig{
			Environment: "development",
		},
		DefaultAdmin: DefaultAdminConfig{
			Email:    "  admin@example.com  ",
			Password: "  password123  ",
		},
	}

	err := ValidateDefaultAdmin(cfg)
	require.NoError(t, err)

	// Config should be mutated with trimmed values
	assert.Equal(t, "admin@example.com", cfg.DefaultAdmin.Email, "Email should be trimmed in-place")
	assert.Equal(t, "password123", cfg.DefaultAdmin.Password, "Password should be trimmed in-place")
}

// TestValidateDefaultAdmin_EmptyConfigMutation tests that empty credentials are cleared
func TestValidateDefaultAdmin_EmptyConfigMutation(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		App: AppConfig{
			Environment: "development",
		},
		DefaultAdmin: DefaultAdminConfig{
			Email:    "  ",
			Password: "  ",
		},
	}

	err := ValidateDefaultAdmin(cfg)
	require.NoError(t, err)

	// Config should be mutated to empty strings
	assert.Equal(t, "", cfg.DefaultAdmin.Email, "Email should be cleared")
	assert.Equal(t, "", cfg.DefaultAdmin.Password, "Password should be cleared")
}

// TestValidateDefaultAdmin_MinimumPasswordLengthProduction tests exact 12 character password in production
func TestValidateDefaultAdmin_MinimumPasswordLengthProduction(t *testing.T) {
	t.Parallel()

	exactly12Chars := "123456789012" // exactly 12 characters

	cfg := &Config{
		App: AppConfig{
			Environment: "production",
		},
		DefaultAdmin: DefaultAdminConfig{
			Email:    "admin@example.com",
			Password: exactly12Chars,
		},
	}

	err := ValidateDefaultAdmin(cfg)
	assert.NoError(t, err, "Exactly 12 character password should be accepted in production")
}

// TestValidateDefaultAdmin_AllInsecureDefaultAdminEmails tests all insecure default emails are detected
func TestValidateDefaultAdmin_AllInsecureDefaultAdminEmails(t *testing.T) {
	t.Parallel()

	// Test that all emails in the insecure list are detected
	for email := range insecureDefaultAdminEmails {
		cfg := &Config{
			App: AppConfig{
				Environment: "production",
			},
			DefaultAdmin: DefaultAdminConfig{
				Email:    email,
				Password: "securepassword123456",
			},
		}

		err := ValidateDefaultAdmin(cfg)
		assert.Error(t, err, "Email %s should be detected as insecure", email)
	}
}

// TestValidateDefaultAdmin_AllInsecureDefaultAdminPasswords tests all insecure default passwords are detected
func TestValidateDefaultAdmin_AllInsecureDefaultAdminPasswords(t *testing.T) {
	t.Parallel()

	// Test that all passwords in the insecure list are detected
	for password := range insecureDefaultAdminPasswords {
		cfg := &Config{
			App: AppConfig{
				Environment: "production",
			},
			DefaultAdmin: DefaultAdminConfig{
				Email:    "secure@example.com",
				Password: password,
			},
		}

		err := ValidateDefaultAdmin(cfg)
		assert.Error(t, err, "Password %s should be detected as insecure", password)
	}
}
