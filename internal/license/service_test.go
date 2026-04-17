package license

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	appcrypto "github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/models"
)

func TestVerifyStoredActivationTokenRejectsTamperedCiphertext(t *testing.T) {
	t.Parallel()

	svc, token, _, _ := newTestLicenseService(t, time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC))
	record := &models.LicenseRecord{ActivationToken: token + "tampered"}

	if _, _, err := svc.verifyStoredActivationToken(record); err == nil {
		t.Fatal("verifyStoredActivationToken() error = nil, want decrypt or verify failure")
	}
}

func TestVerifyStoredActivationTokenAcceptsExpiredTokenForGraceEvaluation(t *testing.T) {
	t.Parallel()

	issuedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	expiresAt := issuedAt.AddDate(0, 0, 7)
	svc, token, claims, kid := newTestLicenseServiceWithIssueTime(t, issuedAt, &expiresAt)
	record := &models.LicenseRecord{ActivationToken: token}

	storedClaims, storedKID, err := svc.verifyStoredActivationToken(record)
	if err != nil {
		t.Fatalf("verifyStoredActivationToken() error = %v", err)
	}
	if storedKID != kid {
		t.Fatalf("verifyStoredActivationToken() kid = %q, want %q", storedKID, kid)
	}
	if storedClaims.LicenseID != claims.LicenseID {
		t.Fatalf("verifyStoredActivationToken() license_id = %q, want %q", storedClaims.LicenseID, claims.LicenseID)
	}
}

func TestApplySignedClaimsToRecordRestoresCanonicalEntitlements(t *testing.T) {
	t.Parallel()

	svc, token, claims, kid := newTestLicenseService(t, time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC))
	record := &models.LicenseRecord{
		ActivationToken:            token,
		LicenseFamilyID:            "forged-family",
		LicenseID:                  "forged-license",
		Revision:                   99,
		KeyID:                      "attacker",
		Issuer:                     "attacker",
		Audience:                   "attacker",
		Product:                    "attacker",
		HWIDFull:                   "forged-hwid",
		HWIDHash:                   "forged-hash",
		Tier:                       "enterprise",
		LicenseKind:                KindTrial,
		TrialDays:                  14,
		MaxOrganizations:           999,
		MaxUsersPerOrg:             999,
		MaxWhatsAppEndpointsPerOrg: 999,
		MaxWorkers:                 999,
		MaxWorkersPerOrg:           999,
		MaxStorageBytesPerOrg:      999,
	}

	updates := svc.applySignedClaimsToRecord(record, claims, kid)
	if len(updates) == 0 {
		t.Fatal("applySignedClaimsToRecord() updates = 0, want corrected fields")
	}
	if record.LicenseID != claims.LicenseID {
		t.Fatalf("record.LicenseID = %q, want %q", record.LicenseID, claims.LicenseID)
	}
	if record.LicenseFamilyID != claims.LicenseFamilyID {
		t.Fatalf("record.LicenseFamilyID = %q, want %q", record.LicenseFamilyID, claims.LicenseFamilyID)
	}
	if record.MaxUsersPerOrg != claims.MaxUsersPerOrg {
		t.Fatalf("record.MaxUsersPerOrg = %d, want %d", record.MaxUsersPerOrg, claims.MaxUsersPerOrg)
	}
	if record.KeyID != kid {
		t.Fatalf("record.KeyID = %q, want %q", record.KeyID, kid)
	}
	if record.HWIDHash != claims.HWIDHash {
		t.Fatalf("record.HWIDHash = %q, want %q", record.HWIDHash, claims.HWIDHash)
	}
	if record.MaxWorkersPerOrg != claims.MaxWorkersPerOrg {
		t.Fatalf("record.MaxWorkersPerOrg = %d, want %d", record.MaxWorkersPerOrg, claims.MaxWorkersPerOrg)
	}
	if record.MaxStorageBytesPerOrg != claims.MaxStorageBytesPerOrg {
		t.Fatalf("record.MaxStorageBytesPerOrg = %d, want %d", record.MaxStorageBytesPerOrg, claims.MaxStorageBytesPerOrg)
	}
}

func newTestLicenseService(t *testing.T, now time.Time) (*Service, string, *LicenseClaims, string) {
	t.Helper()
	expiresAt := now.AddDate(0, 0, 30)
	return newTestLicenseServiceWithIssueTime(t, now, &expiresAt)
}

func newTestLicenseServiceWithIssueTime(t *testing.T, issuedAt time.Time, expiresAt *time.Time) (*Service, string, *LicenseClaims, string) {
	t.Helper()

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

	cfg := &config.Config{
		App: config.AppConfig{
			EncryptionKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		License: config.LicenseConfig{
			Enabled:                  true,
			RollbackToleranceSeconds: 60,
			GracePeriodDays:          7,
		},
	}
	svc := &Service{
		cfg:       cfg,
		keyRing:   map[string]ed25519.PublicKey{"vendor-1": decodedPublicKey},
		hwidFull:  "hwid-full",
		hwidShort: "hwid-short",
		hwidHash:  "hwid-hash",
		now:       func() time.Time { return issuedAt.AddDate(0, 0, 20) },
	}

	token, err := IssueToken(IssueRequest{
		KeyID:                      "vendor-1",
		PrivateKey:                 decodedPrivateKey,
		LicenseID:                  "license-1",
		LicenseFamilyID:            "family-1",
		Revision:                   2,
		HWIDHash:                   svc.hwidHash,
		Tier:                       "starter",
		LicenseKind:                KindPaid,
		TrialDays:                  0,
		MaxOrganizations:           1,
		MaxUsersPerOrg:             5,
		MaxWhatsAppEndpointsPerOrg: 5,
		MaxWorkers:                 2,
		MaxWorkersPerOrg:           4,
		MaxStorageBytesPerOrg:      5 * 1024 * 1024 * 1024,
		IssuedAt:                   issuedAt,
		NotBefore:                  issuedAt,
		ExpiresAt:                  expiresAt,
	})
	if err != nil {
		t.Fatalf("IssueToken() error = %v", err)
	}

	claims, kid, err := VerifyTokenSignatureOnly(token, svc.keyRing)
	if err != nil {
		t.Fatalf("VerifyTokenSignatureOnly() error = %v", err)
	}

	encryptedToken, err := appcrypto.Encrypt(token, cfg.App.EncryptionKey)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	return svc, encryptedToken, claims, kid
}
