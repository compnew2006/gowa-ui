package licenseissuer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
)

func TestIssueTokenFromOptionsTrial(t *testing.T) {
	_, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	privateKeyFile := filepath.Join(t.TempDir(), "private.key")
	if err := os.WriteFile(privateKeyFile, []byte(privateKey+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	token, err := IssueTokenFromOptions(IssueOptions{
		KeyID:                   DefaultKeyID,
		PrivateKeyFile:          privateKeyFile,
		HWID:                    strings.Repeat("a", 64),
		Trial:                   "7d",
		Organizations:           1,
		UsersPerOrg:             5,
		WhatsAppEndpointsPerOrg: 5,
		Workers:                 2,
	}, time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("IssueTokenFromOptions() error = %v", err)
	}
	if strings.TrimSpace(token) == "" {
		t.Fatal("IssueTokenFromOptions() returned empty token")
	}
}

func TestIssueTokenFromOptionsRejectsBadTrial(t *testing.T) {
	_, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	privateKeyFile := filepath.Join(t.TempDir(), "private.key")
	if err := os.WriteFile(privateKeyFile, []byte(privateKey+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err = IssueTokenFromOptions(IssueOptions{
		KeyID:                   DefaultKeyID,
		PrivateKeyFile:          privateKeyFile,
		HWID:                    strings.Repeat("b", 64),
		Trial:                   "21d",
		Organizations:           1,
		UsersPerOrg:             5,
		WhatsAppEndpointsPerOrg: 5,
		Workers:                 2,
	}, time.Now().UTC())
	if err == nil {
		t.Fatal("IssueTokenFromOptions() expected error for invalid trial")
	}
}

func TestIssueLicenseFromOptionsAcceptsCustomPaidDuration(t *testing.T) {
	_, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	privateKeyFile := filepath.Join(t.TempDir(), "private.key")
	if err := os.WriteFile(privateKeyFile, []byte(privateKey+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	issuedAt := time.Date(2026, time.April, 8, 12, 0, 0, 0, time.UTC)
	issued, err := IssueLicenseFromOptions(IssueOptions{
		KeyID:                   DefaultKeyID,
		PrivateKeyFile:          privateKeyFile,
		HWID:                    strings.Repeat("c", 64),
		Duration:                "55 days",
		Tier:                    "starter",
		Organizations:           1,
		UsersPerOrg:             5,
		WhatsAppEndpointsPerOrg: 5,
		Workers:                 2,
	}, issuedAt)
	if err != nil {
		t.Fatalf("IssueLicenseFromOptions() error = %v", err)
	}

	if issued.Duration != "55d" {
		t.Fatalf("issued.Duration = %q, want %q", issued.Duration, "55d")
	}
	if issued.ExpiresAt == nil {
		t.Fatal("issued.ExpiresAt is nil, want fixed expiry")
	}
	wantExpiry := issuedAt.AddDate(0, 0, 55)
	if !issued.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("issued.ExpiresAt = %s, want %s", issued.ExpiresAt.UTC(), wantExpiry.UTC())
	}
}

func TestIssueLicenseFromOptionsIncludesPerOrgEntitlements(t *testing.T) {
	t.Parallel()

	_, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	privateKeyFile := filepath.Join(t.TempDir(), "private.key")
	if err := os.WriteFile(privateKeyFile, []byte(privateKey+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	issued, err := IssueLicenseFromOptions(IssueOptions{
		KeyID:                   DefaultKeyID,
		PrivateKeyFile:          privateKeyFile,
		HWID:                    strings.Repeat("d", 64),
		Duration:                "365d",
		Tier:                    "business",
		Organizations:           5,
		UsersPerOrg:             25,
		WhatsAppEndpointsPerOrg: 25,
		Workers:                 100,
		WorkersPerOrg:           25,
		StorageBytesPerOrg:      5 * 1024 * 1024 * 1024,
	}, time.Date(2026, time.April, 8, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("IssueLicenseFromOptions() error = %v", err)
	}

	if issued.Claims.MaxWorkersPerOrg != 25 {
		t.Fatalf("issued.Claims.MaxWorkersPerOrg = %d, want %d", issued.Claims.MaxWorkersPerOrg, 25)
	}
	if issued.Claims.MaxStorageBytesPerOrg != 5*1024*1024*1024 {
		t.Fatalf("issued.Claims.MaxStorageBytesPerOrg = %d, want %d", issued.Claims.MaxStorageBytesPerOrg, 5*1024*1024*1024)
	}
}
