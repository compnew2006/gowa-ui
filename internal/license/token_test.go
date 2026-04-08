package license

import (
	"crypto/ed25519"
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

	original := EmbeddedPublicKeyRingJSON
	t.Cleanup(func() {
		EmbeddedPublicKeyRingJSON = original
	})
	EmbeddedPublicKeyRingJSON = `[{"kid":"vendor-1","public_key":"` + publicKey + `"}]`

	keyRing, err := ParseEmbeddedKeyRing()
	if err != nil {
		t.Fatalf("ParseEmbeddedKeyRing() error = %v", err)
	}
	if _, ok := keyRing["vendor-1"]; !ok {
		t.Fatal("ParseEmbeddedKeyRing() missing vendor-1 key")
	}
}

func TestBuildKeyRingFromConfigVerifiesIssuedToken(t *testing.T) {
	cfg, err := config.Load("../../config.toml")
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	keyRing, err := buildKeyRing(cfg)
	if err != nil {
		t.Fatalf("buildKeyRing() error = %v", err)
	}

	token := "eyJhbGciOiJFZERTQSIsImtpZCI6InZlbmRvci0xIiwidHlwIjoiV0hNLUxJQ0VOU0UifQ.eyJsaWNlbnNlX2lkIjoiNTNiY2RmNjgtNTFhMy00YTAyLTkxODQtOWJiZTEyNWJmMDAyIiwibGljZW5zZV9mYW1pbHlfaWQiOiI1M2JjZGY2OC01MWEzLTRhMDItOTE4NC05YmJlMTI1YmYwMDIiLCJyZXZpc2lvbiI6MSwicHJvZHVjdCI6IndoYXRvbWF0ZSIsImh3aWRfaGFzaCI6IjRlN2Y1NDhjZmViZjc3NzgxN2YxMzVkYzY2NjQwMTUzOTFlMGRmNTFlZDRhYjcwOGZmNTkyYWE3MTVlNGNkOGYiLCJ0aWVyIjoic3RhcnRlciIsImxpY2Vuc2Vfa2luZCI6InBhaWQiLCJ0cmlhbF9kYXlzIjowLCJtYXhfb3JnYW5pemF0aW9ucyI6MSwibWF4X3VzZXJzX3Blcl9vcmciOjEsIm1heF93aGF0c2FwcF9lbmRwb2ludHNfcGVyX29yZyI6MSwibWF4X3dvcmtlcnMiOjEsImlzcyI6IndoYXRvbWF0ZS1saWNlbnNlLXZlbmRvciIsInN1YiI6IndoYXRvbWF0ZSIsImF1ZCI6WyJ3aGF0b21hdGUtc2VydmVyIl0sImV4cCI6MTc4MDQ0MDA3MSwibmJmIjoxNzc1NjAxNjcxLCJpYXQiOjE3NzU2MDE2NzEsImp0aSI6IjUzYmNkZjY4LTUxYTMtNGEwMi05MTg0LTliYmUxMjViZjAwMiJ9.kihvlrOD5amf4uWIjoegPmoorgbxQ1Q3fGME91Z3ScWeMSuyEZUOGb8hycbYeqSCVbrTak0OUcVrD93YSsL4Cw"

	if _, _, err := VerifyToken(token, keyRing, time.Date(2026, 4, 7, 22, 44, 3, 0, time.UTC)); err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}
}
