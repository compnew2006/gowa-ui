package handlers_test

import (
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRefreshTokenKey_ValidJTI tests refreshTokenKey with valid JTI
func TestRefreshTokenKey_ValidJTI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		jti  string
		want string
	}{
		{
			name: "valid UUID",
			jti:  "550e8400-e29b-41d4-a716-446655440000",
			want: "refresh:550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "short string",
			jti:  "abc123",
			want: "refresh:abc123",
		},
		{
			name: "empty string",
			jti:  "",
			want: "refresh:",
		},
		{
			name: "special characters",
			jti:  "token-with.special_chars",
			want: "refresh:token-with.special_chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := handlers.RefreshTokenKey(tt.jti)
			assert.Equal(t, tt.want, result, "refreshTokenKey should return correct format")
		})
	}
}

// TestGenerateSlug_ValidNames tests generateSlug with valid names
func TestGenerateSlug_ValidNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		prefixCheck string // Check that slug starts with this prefix
	}{
		{
			name:        "simple lowercase",
			input:       "myteam",
			prefixCheck: "myteam-",
		},
		{
			name:        "with uppercase",
			input:       "MyTeam",
			prefixCheck: "myteam-",
		},
		{
			name:        "with spaces",
			input:       "My Team",
			prefixCheck: "my-team-",
		},
		{
			name:        "with special chars",
			input:       "Team@#$%Name",
			prefixCheck: "teamname-",
		},
		{
			name:        "with numbers",
			input:       "Team123",
			prefixCheck: "team123-",
		},
		{
			name:        "mixed case and numbers",
			input:       "My Team 2024",
			prefixCheck: "my-team-2024-",
		},
		{
			name:        "hyphens preserved",
			input:       "my-team-name",
			prefixCheck: "my-team-name-",
		},
		{
			name:        "empty string",
			input:       "",
			prefixCheck: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := handlers.GenerateSlug(tt.input)

			// Check prefix
			assert.True(t, strings.HasPrefix(result, tt.prefixCheck), "Slug should start with expected prefix")

			// Check format: prefix + 8 char UUID suffix
			parts := strings.Split(result, "-")
			lastPart := parts[len(parts)-1]
			assert.Len(t, lastPart, 8, "Slug should end with 8 character UUID suffix")

			// Verify the entire slug structure
			assert.GreaterOrEqual(t, len(result), 9, "Slug should have at least prefix + hyphen + 8 chars")
		})
	}
}

// TestGenerateSlug_UniqueValues tests that generateSlug produces unique values
func TestGenerateSlug_UniqueValues(t *testing.T) {
	t.Parallel()

	input := "Test Team"
	results := make(map[string]bool)

	for i := 0; i < 100; i++ {
		result := handlers.GenerateSlug(input)
		assert.False(t, results[result], "generateSlug should produce unique values")
		results[result] = true
	}
}

// TestGenerateSlug_ConversionPreserved tests generateSlug character conversion
func TestGenerateSlug_ConversionPreserved(t *testing.T) {
	t.Parallel()

	// Test specific character conversions
	tests := []struct {
		name        string
		input       string
		contains    []string // Substrings that should be in the result (before UUID)
		notContains []string // Substrings that should NOT be in the result
	}{
		{
			name:        "uppercase converted",
			input:       "ABC",
			contains:    []string{"abc"},
			notContains: []string{"A", "B", "C"},
		},
		{
			name:        "spaces become hyphens",
			input:       "hello world",
			contains:    []string{"hello-world"},
			notContains: []string{" "},
		},
		{
			name:        "special chars removed",
			input:       "hello@world",
			contains:    []string{"helloworld"},
			notContains: []string{"@"},
		},
		{
			name:        "mixed case and spaces",
			input:       "Hello World Test",
			contains:    []string{"hello-world-test"},
			notContains: []string{"H", "W", "T", " "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := handlers.GenerateSlug(tt.input)

			// Remove the UUID suffix for checking
			parts := strings.Split(result, "-")
			parts = parts[:len(parts)-1] // Remove last part (UUID)
			slugOnly := strings.Join(parts, "-")

			for _, substr := range tt.contains {
				assert.Contains(t, slugOnly, substr, "Slug should contain expected substring")
			}
			for _, substr := range tt.notContains {
				assert.NotContains(t, slugOnly, substr, "Slug should not contain excluded substring")
			}
		})
	}
}

// TestGenerateAccessToken_Success tests generateAccessToken with valid user
func TestGenerateAccessToken_Success(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{
			JWT: config.JWTConfig{
				Secret:           "test-secret-key-for-jwt-signing",
				AccessExpiryMins: 60,
			},
		},
	}

	user := &models.User{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		Email:          "test@example.com",
		OrganizationID: uuid.New(),
		IsSuperAdmin:   false,
	}

	token, expiresAt, err := app.GenerateAccessToken(user)

	require.NoError(t, err, "generateAccessToken should succeed with valid config")
	assert.NotEmpty(t, token, "Token should not be empty")
	assert.False(t, expiresAt.IsZero(), "Expiry time should be set")
	assert.True(t, expiresAt.After(time.Now()), "Expiry should be in the future")
}

// TestGenerateAccessToken_NilConfig tests generateAccessToken with nil config
func TestGenerateAccessToken_NilConfig(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: nil,
	}

	user := &models.User{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		Email:          "test@example.com",
		OrganizationID: uuid.New(),
	}

	token, expiresAt, err := app.GenerateAccessToken(user)

	assert.Error(t, err, "generateAccessToken should error with nil config")
	assert.Empty(t, token, "Token should be empty on error")
	assert.True(t, expiresAt.IsZero(), "Expiry should be zero on error")
}

// TestGenerateAccessToken_CustomExpiry tests generateAccessToken with custom expiry
func TestGenerateAccessToken_CustomExpiry(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{
			JWT: config.JWTConfig{
				Secret:           "test-secret-key-for-jwt-signing",
				AccessExpiryMins: 120, // 2 hours
			},
		},
	}

	user := &models.User{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		Email:          "test@example.com",
		OrganizationID: uuid.New(),
	}

	_, expiresAt, err := app.GenerateAccessToken(user)

	require.NoError(t, err, "generateAccessToken should succeed")

	// Check expiry is approximately 2 hours from now
	expectedExpiry := time.Now().Add(120 * time.Minute)
	diff := expectedExpiry.Sub(expiresAt)
	assert.Less(t, diff.Abs(), 10*time.Second, "Expiry should be approximately 2 hours from now")
}

// TestGenerateRefreshToken_Success tests generateRefreshToken with valid setup
func TestGenerateRefreshToken_Success(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = redisClient.Close() }()

	app := &handlers.App{
		Config: &config.Config{
			JWT: config.JWTConfig{
				Secret:            "test-secret-key-for-jwt-signing",
				RefreshExpiryDays: 7,
			},
		},
		Redis: redisClient,
		Log:   createTestLogger(),
	}

	user := &models.User{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		Email:          "test@example.com",
		OrganizationID: uuid.New(),
	}

	token, err := app.GenerateRefreshToken(user)

	assert.NoError(t, err, "generateRefreshToken should succeed with valid setup")
	assert.NotEmpty(t, token, "Token should not be empty")
	keys := mr.Keys()
	require.Len(t, keys, 1, "refresh token JTI should be stored in Redis")
	assert.True(t, strings.HasPrefix(keys[0], "refresh:"), "stored Redis key should use refresh prefix")
	storedUserID, getErr := mr.Get(keys[0])
	require.NoError(t, getErr)
	assert.Equal(t, user.ID.String(), storedUserID, "stored token mapping should point to the user ID")
}

func TestGenerateRefreshToken_NilRedis(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{
			JWT: config.JWTConfig{
				Secret:            "test-secret-key-for-jwt-signing",
				RefreshExpiryDays: 7,
			},
		},
		Redis: nil,
		Log:   createTestLogger(),
	}

	user := &models.User{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		Email:          "test@example.com",
		OrganizationID: uuid.New(),
	}

	token, err := app.GenerateRefreshToken(user)
	assert.Error(t, err, "generateRefreshToken should fail when refresh-token storage is unavailable")
	assert.ErrorContains(t, err, "refresh token storage is unavailable")
	assert.Empty(t, token, "Token should be empty on storage failure")
}

func TestGenerateRefreshToken_RedisWriteFailure(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: addr})
	defer func() { _ = redisClient.Close() }()

	app := &handlers.App{
		Config: &config.Config{
			JWT: config.JWTConfig{
				Secret:            "test-secret-key-for-jwt-signing",
				RefreshExpiryDays: 7,
			},
		},
		Redis: redisClient,
		Log:   createTestLogger(),
	}

	user := &models.User{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		Email:          "test@example.com",
		OrganizationID: uuid.New(),
	}

	token, err := app.GenerateRefreshToken(user)
	assert.Error(t, err, "generateRefreshToken should fail when Redis write fails")
	assert.ErrorContains(t, err, "refresh token storage is unavailable")
	assert.Empty(t, token, "Token should be empty on storage write failure")
}

// TestGenerateRefreshToken_NilConfig tests generateRefreshToken with nil config
func TestGenerateRefreshToken_NilConfig(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: nil,
		Redis:  nil,
		Log:    createTestLogger(),
	}

	user := &models.User{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		Email:          "test@example.com",
		OrganizationID: uuid.New(),
	}

	token, err := app.GenerateRefreshToken(user)

	// Should error when trying to get JWT secret from nil config
	assert.Error(t, err, "generateRefreshToken should error with nil config")
	assert.Empty(t, token, "Token should be empty on error")
}

// TestGenerateRegisterInviteToken_Success tests generateRegisterInviteToken
func TestGenerateRegisterInviteToken_Success(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{
			JWT: config.JWTConfig{
				Secret: "test-secret-key-for-jwt-signing",
			},
		},
	}

	orgID := uuid.New()
	ttl := 24 * time.Hour

	token, expiresAt, err := app.GenerateRegisterInviteToken(orgID, ttl)

	require.NoError(t, err, "generateRegisterInviteToken should succeed with valid config")
	assert.NotEmpty(t, token, "Token should not be empty")
	assert.False(t, expiresAt.IsZero(), "Expiry time should be set")
	assert.True(t, expiresAt.After(time.Now()), "Expiry should be in the future")
}

// TestGenerateRegisterInviteToken_NilConfig tests generateRegisterInviteToken with nil config
func TestGenerateRegisterInviteToken_NilConfig(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: nil,
	}

	orgID := uuid.New()
	ttl := 24 * time.Hour

	token, expiresAt, err := app.GenerateRegisterInviteToken(orgID, ttl)

	assert.Error(t, err, "generateRegisterInviteToken should error with nil config")
	assert.Empty(t, token, "Token should be empty on error")
	assert.True(t, expiresAt.IsZero(), "Expiry should be zero on error")
}

// TestValidateRegisterInviteToken_ValidToken tests validateRegisterInviteToken with valid token
func TestValidateRegisterInviteToken_ValidToken(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{
			JWT: config.JWTConfig{
				Secret: "test-secret-key-for-jwt-signing",
			},
		},
	}

	orgID := uuid.New()
	ttl := 24 * time.Hour

	// First generate a valid token
	tokenString, _, err := app.GenerateRegisterInviteToken(orgID, ttl)
	require.NoError(t, err, "Should generate token successfully")

	// Now validate it
	validatedOrgID, err := app.ValidateRegisterInviteToken(tokenString)

	assert.NoError(t, err, "validateRegisterInviteToken should succeed with valid token")
	assert.Equal(t, orgID, validatedOrgID, "Validated org ID should match original")
}

// TestValidateRegisterInviteToken_InvalidToken tests validateRegisterInviteToken with invalid token
func TestValidateRegisterInviteToken_InvalidToken(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{
			JWT: config.JWTConfig{
				Secret: "test-secret-key-for-jwt-signing",
			},
		},
	}

	invalidToken := "invalid.token.string"

	validatedOrgID, err := app.ValidateRegisterInviteToken(invalidToken)

	assert.Error(t, err, "validateRegisterInviteToken should error with invalid token")
	assert.Equal(t, uuid.Nil, validatedOrgID, "Validated org ID should be Nil on error")
}

// TestValidateRegisterInviteToken_WrongSecret tests validateRegisterInviteToken with wrong secret
func TestValidateRegisterInviteToken_WrongSecret(t *testing.T) {
	t.Parallel()

	// Generate token with one secret
	app1 := &handlers.App{
		Config: &config.Config{
			JWT: config.JWTConfig{
				Secret: "test-secret-key-for-jwt-signing",
			},
		},
	}

	orgID := uuid.New()
	ttl := 24 * time.Hour
	tokenString, _, err := app1.GenerateRegisterInviteToken(orgID, ttl)
	require.NoError(t, err, "Should generate token successfully")

	// Try to validate with different secret
	app2 := &handlers.App{
		Config: &config.Config{
			JWT: config.JWTConfig{
				Secret: "different-secret-key",
			},
		},
	}

	validatedOrgID, err := app2.ValidateRegisterInviteToken(tokenString)

	assert.Error(t, err, "validateRegisterInviteToken should error with wrong secret")
	assert.Equal(t, uuid.Nil, validatedOrgID, "Validated org ID should be Nil on error")
}
