package whatsmeow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zerodha/logf"
)

func newAdapterWithStorage(t *testing.T) (*WhatsmeowAdapter, string) {
	t.Helper()
	storage := t.TempDir()
	adapter := &WhatsmeowAdapter{
		manager: &ConnectionManager{mediaStoragePath: storage},
		logger:  logf.New(logf.Opts{}),
	}
	return adapter, storage
}

func TestDownloadMediaFromURL_LocalRelativePath(t *testing.T) {
	adapter, storage := newAdapterWithStorage(t)

	relPath := filepath.Join("documents", "sample.txt")
	fullPath := filepath.Join(storage, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("hello-local"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	data, mimeType, err := adapter.downloadMediaFromURL(relPath)
	if err != nil {
		t.Fatalf("downloadMediaFromURL failed: %v", err)
	}
	if string(data) != "hello-local" {
		t.Fatalf("data mismatch: got %q", string(data))
	}
	if mimeType == "" {
		t.Fatal("expected mime type to be detected")
	}
}

func TestDownloadMediaFromURL_LocalAbsolutePath(t *testing.T) {
	adapter, _ := newAdapterWithStorage(t)

	tmp := filepath.Join(t.TempDir(), "abs.bin")
	if err := os.WriteFile(tmp, []byte{0x01, 0x02, 0x03}, 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if _, _, err := adapter.downloadMediaFromURL(tmp); err == nil {
		t.Fatal("expected absolute path to be rejected")
	}
}

func TestDownloadMediaFromURL_RejectsTraversal(t *testing.T) {
	adapter, _ := newAdapterWithStorage(t)

	if _, _, err := adapter.downloadMediaFromURL("../secret.txt"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestDownloadMediaFromURL_RejectsSymlinkEscape(t *testing.T) {
	adapter, storage := newAdapterWithStorage(t)

	target := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(target, []byte("outside"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	linkPath := filepath.Join(storage, "documents", "escape.txt")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("symlink failed: %v", err)
	}

	if _, _, err := adapter.downloadMediaFromURL(filepath.Join("documents", "escape.txt")); err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}

func TestDownloadMediaFromURL_RejectsPrivateHTTPHost(t *testing.T) {
	adapter, _ := newAdapterWithStorage(t)

	if _, _, err := adapter.downloadMediaFromURL("http://127.0.0.1/media.txt"); err == nil {
		t.Fatal("expected private host URL to be rejected")
	}
}

func TestDownloadMediaFromURL_RejectsFileScheme(t *testing.T) {
	adapter, _ := newAdapterWithStorage(t)

	if _, _, err := adapter.downloadMediaFromURL("file:///etc/passwd"); err == nil {
		t.Fatal("expected file scheme URL to be rejected")
	}
}
