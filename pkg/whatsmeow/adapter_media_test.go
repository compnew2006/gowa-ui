package whatsmeow

import (
	"net/http"
	"net/http/httptest"
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

	data, _, err := adapter.downloadMediaFromURL(tmp)
	if err != nil {
		t.Fatalf("downloadMediaFromURL failed: %v", err)
	}
	if len(data) != 3 {
		t.Fatalf("expected 3 bytes, got %d", len(data))
	}
}

func TestDownloadMediaFromURL_RejectsTraversal(t *testing.T) {
	adapter, _ := newAdapterWithStorage(t)

	if _, _, err := adapter.downloadMediaFromURL("../secret.txt"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestDownloadMediaFromURL_HTTPURL(t *testing.T) {
	adapter, _ := newAdapterWithStorage(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("http-media"))
	}))
	defer srv.Close()

	data, mimeType, err := adapter.downloadMediaFromURL(srv.URL)
	if err != nil {
		t.Fatalf("downloadMediaFromURL failed: %v", err)
	}
	if string(data) != "http-media" {
		t.Fatalf("data mismatch: got %q", string(data))
	}
	if mimeType != "text/plain" {
		t.Fatalf("mime mismatch: got %q want text/plain", mimeType)
	}
}
