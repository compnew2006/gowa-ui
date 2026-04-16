package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
)

func TestIssueAndVerifyTokenPaid(t *testing.T) {
	publicKey, privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	decodedPublicKey, err := DecodePublicKey(publicKey)
	if err != nil {
		t.Fatalf("DecodePublicKey() error = %v", err)
	}
	decodedPrivateKey, err := DecodePrivateKey(privateKey)
	if err != nil {
		t.Fatalf("DecodePrivateKey() error = %v", err)
	}

	now := time.Date(2026, 4, 7, 20, 0, 0, 0, time.UTC)
	expiresAt := now.AddDate(0, 0, 365)

	token, err := IssueToken(IssueRequest{
		KeyID:                      "vendor-1",
		PrivateKey:                 decodedPrivateKey,
		LicenseID:                  "license-1",
		LicenseFamilyID:            "family-1",
		Revision:                   2,
		HWIDHash:                   "hwid-hash",
		Tier:                       "starter",
		LicenseKind:                KindPaid,
		TrialDays:                  0,
		MaxOrganizations:           1,
		MaxUsersPerOrg:             5,
		MaxWhatsAppEndpointsPerOrg: 5,
		MaxWorkers:                 2,
		IssuedAt:                   now,
		NotBefore:                  now,
		ExpiresAt:                  &expiresAt,
	})
	if err != nil {
		t.Fatalf("IssueToken() error = %v", err)
	}

	claims, kid, err := VerifyToken(token, map[string]ed25519.PublicKey{
		"vendor-1": decodedPublicKey,
	}, now)
	if err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}
	if kid != "vendor-1" {
		t.Fatalf("VerifyToken() kid = %q, want %q", kid, "vendor-1")
	}
	if claims.LicenseKind != KindPaid {
		t.Fatalf("VerifyToken() license kind = %q, want %q", claims.LicenseKind, KindPaid)
	}
	if claims.MaxUsersPerOrg != 5 {
		t.Fatalf("VerifyToken() max users = %d, want %d", claims.MaxUsersPerOrg, 5)
	}
}

func TestIssueTokenRejectsInvalidTrialPreset(t *testing.T) {
	_, privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	decodedPrivateKey, err := DecodePrivateKey(privateKey)
	if err != nil {
		t.Fatalf("DecodePrivateKey() error = %v", err)
	}

	_, err = IssueToken(IssueRequest{
		KeyID:                      "vendor-1",
		PrivateKey:                 decodedPrivateKey,
		HWIDHash:                   "hwid-hash",
		Tier:                       "trial",
		LicenseKind:                KindTrial,
		TrialDays:                  30,
		MaxOrganizations:           1,
		MaxUsersPerOrg:             5,
		MaxWhatsAppEndpointsPerOrg: 5,
		MaxWorkers:                 1,
		IssuedAt:                   time.Now().UTC(),
		NotBefore:                  time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("IssueToken() error = nil, want validation error")
	}
}

func TestParseEmbeddedKeyRing(t *testing.T) {
	publicKey, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	originalJSON := EmbeddedPublicKeyRingJSON
	originalBase64 := EmbeddedPublicKeyRingBase64
	t.Cleanup(func() {
		EmbeddedPublicKeyRingJSON = originalJSON
		EmbeddedPublicKeyRingBase64 = originalBase64
	})
	EmbeddedPublicKeyRingBase64 = ""
	EmbeddedPublicKeyRingJSON = `[{"kid":"vendor-1","public_key":"` + publicKey + `"}]`

	keyRing, err := ParseEmbeddedKeyRing()
	if err != nil {
		t.Fatalf("ParseEmbeddedKeyRing() error = %v", err)
	}
	if _, ok := keyRing["vendor-1"]; !ok {
		t.Fatal("ParseEmbeddedKeyRing() missing vendor-1 key")
	}
}

func TestParseEmbeddedKeyRingBase64(t *testing.T) {
	publicKey, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	originalJSON := EmbeddedPublicKeyRingJSON
	originalBase64 := EmbeddedPublicKeyRingBase64
	t.Cleanup(func() {
		EmbeddedPublicKeyRingJSON = originalJSON
		EmbeddedPublicKeyRingBase64 = originalBase64
	})
	EmbeddedPublicKeyRingJSON = "[]"
	EmbeddedPublicKeyRingBase64 = base64.StdEncoding.EncodeToString([]byte(`[{"kid":"vendor-1","public_key":"` + publicKey + `"}]`))

	keyRing, err := ParseEmbeddedKeyRing()
	if err != nil {
		t.Fatalf("ParseEmbeddedKeyRing() error = %v", err)
	}
	if _, ok := keyRing["vendor-1"]; !ok {
		t.Fatal("ParseEmbeddedKeyRing() missing vendor-1 key from base64 payload")
	}
}

func TestBuildKeyRingFromDevelopmentConfigVerifiesIssuedToken(t *testing.T) {
	publicKey, privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	decodedPrivateKey, err := DecodePrivateKey(privateKey)
	if err != nil {
		t.Fatalf("DecodePrivateKey() error = %v", err)
	}
	cfg := &config.Config{
		App: config.AppConfig{
			Environment: "development",
		},
		License: config.LicenseConfig{
			PublicKey:                    publicKey,
			PublicKeyKID:                 "vendor-1",
			AllowUnsafePublicKeyOverride: true,
		},
	}

	keyRing, err := buildKeyRing(cfg)
	if err != nil {
		t.Fatalf("buildKeyRing() error = %v", err)
	}

	now := time.Date(2026, 4, 7, 22, 44, 3, 0, time.UTC)
	expiresAt := now.AddDate(0, 0, 365)
	token, err := IssueToken(IssueRequest{
		KeyID:                      "vendor-1",
		PrivateKey:                 decodedPrivateKey,
		LicenseID:                  "license-1",
		LicenseFamilyID:            "family-1",
		Revision:                   1,
		HWIDHash:                   "hwid-hash",
		Tier:                       "starter",
		LicenseKind:                KindPaid,
		MaxOrganizations:           1,
		MaxUsersPerOrg:             1,
		MaxWhatsAppEndpointsPerOrg: 1,
		MaxWorkers:                 1,
		IssuedAt:                   now,
		NotBefore:                  now,
		ExpiresAt:                  &expiresAt,
	})
	if err != nil {
		t.Fatalf("IssueToken() error = %v", err)
	}

	if _, _, err := VerifyToken(token, keyRing, now); err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}
}

func TestBuildKeyRingAllowsProductionConfigOverrideWithExplicitOptIn(t *testing.T) {
	publicKey, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	keyRing, err := buildKeyRing(&config.Config{
		App: config.AppConfig{
			Environment: "production",
		},
		License: config.LicenseConfig{
			PublicKey:                    publicKey,
			PublicKeyKID:                 "vendor-1",
			AllowUnsafePublicKeyOverride: true,
		},
	})
	if err != nil {
		t.Fatalf("buildKeyRing() error = %v", err)
	}
	if _, ok := keyRing["vendor-1"]; !ok {
		t.Fatal("buildKeyRing() missing production override key")
	}
}

func TestBuildKeyRingRejectsProductionConfigOverrideWithoutOptIn(t *testing.T) {
	publicKey, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	_, err = buildKeyRing(&config.Config{
		App: config.AppConfig{
			Environment: "production",
		},
		License: config.LicenseConfig{
			PublicKey:    publicKey,
			PublicKeyKID: "vendor-1",
		},
	})
	if err == nil {
		t.Fatal("buildKeyRing() error = nil, want production opt-in enforcement")
	}
}
