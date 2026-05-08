package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNow(t *testing.T) {
	t.Run("empty string returns current time", func(t *testing.T) {
		before := time.Now().UTC()
		result, err := parseNow("")
		after := time.Now().UTC()

		require.NoError(t, err)
		assert.True(t, !result.Before(before) && !result.After(after),
			"expected result between %v and %v, got %v", before, after, result)
	})

	t.Run("literal now is not a special case and returns error", func(t *testing.T) {
		_, err := parseNow("now")

		require.Error(t, err)
	})

	t.Run("RFC3339 datetime parses correctly", func(t *testing.T) {
		input := "2024-01-15T10:30:00Z"
		result, err := parseNow(input)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), result)
	})

	t.Run("date-only returns error", func(t *testing.T) {
		input := "2024-01-15"
		_, err := parseNow(input)

		require.Error(t, err)
	})

	t.Run("invalid string returns error", func(t *testing.T) {
		_, err := parseNow("not-a-time")

		require.Error(t, err)
	})

	t.Run("whitespace is trimmed", func(t *testing.T) {
		before := time.Now().UTC()
		result, err := parseNow("  ")
		after := time.Now().UTC()

		require.NoError(t, err)
		assert.True(t, !result.Before(before) && !result.After(after),
			"expected result between %v and %v, got %v", before, after, result)
	})
}

func TestParseTokenUnverified(t *testing.T) {
	t.Run("empty string returns error", func(t *testing.T) {
		claims, header, err := parseTokenUnverified("")

		require.Error(t, err)
		assert.Nil(t, claims)
		assert.Nil(t, header)
	})

	t.Run("invalid JWT returns error", func(t *testing.T) {
		claims, header, err := parseTokenUnverified("not-a-jwt")

		require.Error(t, err)
		assert.Nil(t, claims)
		assert.Nil(t, header)
	})

	t.Run("valid-looking JWT with fake signature returns parsed claims", func(t *testing.T) {
		claims := &license.LicenseClaims{
			LicenseID:       "lic-123",
			LicenseFamilyID: "fam-456",
			Revision:        1,
			Product:         license.ProductWhatomate,
			HWIDHash:        "hwid-abc",
			Tier:            "professional",
			LicenseKind:     license.KindPaid,
			TrialDays:       0,
			MaxOrganizations: 5,
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    license.TokenIssuer,
				Subject:   license.ProductWhatomate,
				Audience:  jwt.ClaimStrings{license.TokenAudience},
				ExpiresAt: jwt.NewNumericDate(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)),
				NotBefore: jwt.NewNumericDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				IssuedAt:  jwt.NewNumericDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				ID:        "lic-123",
			},
		}

		fakeSig := base64.RawURLEncoding.EncodeToString(make([]byte, 64))
		headerJSON, _ := json.Marshal(map[string]string{"alg": "EdDSA", "typ": license.TokenType, "kid": "vendor-1"})
		payloadJSON, _ := json.Marshal(claims)
		rawToken := strings.Join([]string{
			base64.RawURLEncoding.EncodeToString(headerJSON),
			base64.RawURLEncoding.EncodeToString(payloadJSON),
			fakeSig,
		}, ".")

		parsedClaims, parsedHeader, err := parseTokenUnverified(rawToken)

		require.NoError(t, err)
		require.NotNil(t, parsedClaims)
		require.NotNil(t, parsedHeader)

		assert.Equal(t, "lic-123", parsedClaims.LicenseID)
		assert.Equal(t, "fam-456", parsedClaims.LicenseFamilyID)
		assert.Equal(t, uint64(1), parsedClaims.Revision)
		assert.Equal(t, license.ProductWhatomate, parsedClaims.Product)
		assert.Equal(t, "hwid-abc", parsedClaims.HWIDHash)
		assert.Equal(t, "professional", parsedClaims.Tier)
		assert.Equal(t, license.KindPaid, parsedClaims.LicenseKind)
		assert.Equal(t, 0, parsedClaims.TrialDays)
		assert.Equal(t, 5, parsedClaims.MaxOrganizations)
		assert.Equal(t, license.TokenIssuer, parsedClaims.Issuer)
		assert.Equal(t, license.TokenAudience, parsedClaims.Audience[0])

		assert.Equal(t, "EdDSA", parsedHeader["alg"])
		assert.Equal(t, license.TokenType, parsedHeader["typ"])
		assert.Equal(t, "vendor-1", parsedHeader["kid"])
	})

	t.Run("garbage base64 segments returns error", func(t *testing.T) {
		_, _, err := parseTokenUnverified("a.b.c")

		require.Error(t, err)
	})
}
