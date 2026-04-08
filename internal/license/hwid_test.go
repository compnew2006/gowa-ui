package license

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/zerodha/logf"
)

func TestBuildHWIDUsesConfiguredHostMachineIDPath(t *testing.T) {
	dir := t.TempDir()
	hostMachineID := filepath.Join(dir, "host-machine-id")
	productUUID := filepath.Join(dir, "product_uuid")
	if err := os.WriteFile(hostMachineID, []byte("host-machine-id\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(hostMachineID) error = %v", err)
	}
	if err := os.WriteFile(productUUID, []byte("product-uuid\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(productUUID) error = %v", err)
	}

	cfg := &config.LicenseConfig{
		HostMachineIDPath: hostMachineID,
		FingerprintSources: []string{
			productUUID,
		},
	}

	logger := logf.New(logf.Opts{Level: logf.InfoLevel})
	fullA, shortA, hashA, err := BuildHWID(cfg, logger)
	if err != nil {
		t.Fatalf("BuildHWID() error = %v", err)
	}
	fullB, shortB, hashB, err := BuildHWID(cfg, logger)
	if err != nil {
		t.Fatalf("BuildHWID() second error = %v", err)
	}

	if fullA == "" || shortA == "" || hashA == "" {
		t.Fatal("BuildHWID() returned empty values")
	}
	if fullA != hashA {
		t.Fatalf("BuildHWID() full = %q, want hash %q", fullA, hashA)
	}
	if fullA != fullB || shortA != shortB || hashA != hashB {
		t.Fatal("BuildHWID() returned non-deterministic values")
	}
}

func TestBuildHWIDFailsWhenConfiguredHostMachineIDPathIsMissing(t *testing.T) {
	cfg := &config.LicenseConfig{
		HostMachineIDPath: filepath.Join(t.TempDir(), "missing-machine-id"),
	}

	logger := logf.New(logf.Opts{Level: logf.InfoLevel})
	if _, _, _, err := BuildHWID(cfg, logger); err == nil {
		t.Fatal("BuildHWID() error = nil, want missing host machine-id error")
	}
}
