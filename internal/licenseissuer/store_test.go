package licenseissuer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
)

func TestRegistryStoreSaveAndPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	store := NewRegistryStore(path)
	store.now = func() time.Time {
		return time.Date(2026, time.April, 8, 9, 0, 0, 0, time.UTC)
	}

	entry, err := store.Save(RegistryEntry{
		ID:                         "lic-1",
		HWID:                       "hwid-1",
		Token:                      "token-1",
		LicenseID:                  "lic-1",
		LicenseFamilyID:            "fam-1",
		Tier:                       "starter",
		LicenseKind:                license.KindTrial,
		TrialDays:                  7,
		DurationPreset:             "7d",
		MaxOrganizations:           1,
		MaxUsersPerOrg:             5,
		MaxWhatsAppEndpointsPerOrg: 5,
		MaxWorkers:                 2,
		MaxWorkersPerOrg:           4,
		MaxStorageBytesPerOrg:      5 * 1024 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if entry.CreatedAt.IsZero() {
		t.Fatal("Save() did not set CreatedAt")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("registry permissions = %o, want 600", info.Mode().Perm())
	}

	items, err := store.List(RegistryFilter{HWID: "hwid-1", Kind: "trial"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("List() count = %d, want 1", len(items))
	}
	if items[0].MaxWorkersPerOrg != 4 {
		t.Fatalf("List() max workers per org = %d, want %d", items[0].MaxWorkersPerOrg, 4)
	}
	if items[0].MaxStorageBytesPerOrg != 5*1024*1024*1024 {
		t.Fatalf("List() max storage bytes per org = %d, want %d", items[0].MaxStorageBytesPerOrg, 5*1024*1024*1024)
	}
}

func TestKeyRingStoreUpsertAndPermissions(t *testing.T) {
	publicKey, _, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	decoded, err := license.DecodePublicKey(publicKey)
	if err != nil {
		t.Fatalf("DecodePublicKey() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "keyring.json")
	store := NewKeyRingStore(path)
	if err := store.UpsertPublicKey("vendor-1", decoded); err != nil {
		t.Fatalf("UpsertPublicKey() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("keyring permissions = %o, want 600", info.Mode().Perm())
	}

	kids, err := store.KnownKeyIDs()
	if err != nil {
		t.Fatalf("KnownKeyIDs() error = %v", err)
	}
	if len(kids) != 1 || kids[0] != "vendor-1" {
		t.Fatalf("KnownKeyIDs() = %v, want [vendor-1]", kids)
	}
}

func TestVerifyTokenStatuses(t *testing.T) {
	_, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	now := time.Date(2026, time.April, 8, 10, 0, 0, 0, time.UTC)
	issued, err := IssueLicenseFromPrivateKeyText(IssueOptions{
		HWID:                    strings.Repeat("a", 64),
		Trial:                   "7d",
		Organizations:           1,
		UsersPerOrg:             5,
		WhatsAppEndpointsPerOrg: 5,
		Workers:                 2,
	}, privateKey, now)
	if err != nil {
		t.Fatalf("IssueLicenseFromPrivateKeyText() error = %v", err)
	}

	tempDir := t.TempDir()
	registry := NewRegistryStore(filepath.Join(tempDir, "registry.json"))
	keyring := NewKeyRingStore(filepath.Join(tempDir, "keyring.json"))
	if err := keyring.UpsertPublicKey(issued.KeyID, issued.PublicKey); err != nil {
		t.Fatalf("UpsertPublicKey() error = %v", err)
	}
	if _, err := registry.Save(BuildRegistryEntry(issued)); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	tracked := VerifyToken(issued.Token, keyring, registry, now)
	if tracked.Status != StatusValidTracked {
		t.Fatalf("tracked status = %s, want %s", tracked.Status, StatusValidTracked)
	}

	untrackedIssued, err := IssueLicenseFromPrivateKeyText(IssueOptions{
		HWID:                    strings.Repeat("b", 64),
		Duration:                "365d",
		Tier:                    "starter",
		Organizations:           1,
		UsersPerOrg:             5,
		WhatsAppEndpointsPerOrg: 5,
		Workers:                 2,
	}, privateKey, now.Add(1*time.Minute))
	if err != nil {
		t.Fatalf("IssueLicenseFromPrivateKeyText() error = %v", err)
	}

	untracked := VerifyToken(untrackedIssued.Token, keyring, registry, now.Add(1*time.Minute))
	if untracked.Status != StatusValidUntracked {
		encoded, _ := json.Marshal(untracked)
		t.Fatalf("untracked status = %s, want %s (%s)", untracked.Status, StatusValidUntracked, encoded)
	}

	invalid := VerifyToken("bad-token", keyring, registry, now)
	if invalid.Status != StatusInvalid {
		t.Fatalf("invalid status = %s, want %s", invalid.Status, StatusInvalid)
	}
}
