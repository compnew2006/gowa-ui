package gowa_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestURLMatchesBase locks the SSRF gate used before fetching a webhook-supplied
// media URL (gap #7): only URLs on the GOWA instance's own host:port are
// allowed, so an attacker-controlled URL in a signed webhook can neither leak
// Basic Auth nor be fetched as an SSRF vector.
func TestURLMatchesBase(t *testing.T) {
	const base = "http://gowa.corp:3000"

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"same host:port", "http://gowa.corp:3000/statics/media/x.jpg", true},
		// Host match is origin-scoped (host:port); a path-only difference is fine.
		{"same host different path", "http://gowa.corp:3000/other/y.png", true},
		// Different port → different origin (would hit a different service).
		{"different port", "http://gowa.corp:8080/x.jpg", false},
		// External host → blocked (the core SSRF case).
		{"external host", "http://attacker.example/x.jpg", false},
		// Loopback / private IPs are not the configured base host.
		{"localhost", "http://127.0.0.1/x.jpg", false},
		{"metadata service", "http://169.254.169.254/latest/meta-data", false},
		// Relative / non-absolute URLs are not fetchable as-is → false (callers
		// must resolve them against baseURL themselves).
		{"relative path", "/statics/media/x.jpg", false},
		{"bare path", "statics/media/x.jpg", false},
		// Subdomain of the base host is a DIFFERENT host → blocked.
		{"subdomain", "http://evil.gowa.corp:3000/x.jpg", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, gowa.URLMatchesBase(tc.url, base),
				"URLMatchesBase(%q, %q)", tc.url, base)
		})
	}
}

// TestDownloadMedia_EnforcesSizeCap pins the in-memory download cap: a body
// larger than MaxMediaDownloadSize must be rejected with the explicit limit
// error (never buffered). The test server streams from an infinite source so
// neither side allocates the full payload in memory.
func TestDownloadMedia_EnforcesSizeCap(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Stream one byte more than the cap.
		_, _ = io.Copy(w, io.LimitReader(zeroReader{}, gowa.MaxMediaDownloadSize+1))
	}))
	defer server.Close()

	c := gowa.New(server.URL, "", "")
	_, err := c.DownloadMedia(context.Background(), server.URL+"/big.bin", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max download size")
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// newStreamMock serves GOWA's two-step download: /message/{id}/download
// returns a JSON pointing at a statics path, and the statics path serves
// the payload (finite for success cases, infinite for cap-abort cases).
func newStreamMock(t *testing.T, payload []byte, infinite bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/message/MSG1/download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": map[string]any{
				"file_path":  "/statics/f.bin",
				"media_type": "document",
			},
		})
	})
	mux.HandleFunc("/statics/f.bin", func(w http.ResponseWriter, r *http.Request) {
		if infinite {
			_, _ = io.Copy(w, zeroReader{})
			return
		}
		_, _ = w.Write(payload)
	})
	return httptest.NewServer(mux)
}

func TestDownloadMessageMediaToPath_StreamsFileToDisk(t *testing.T) {
	t.Parallel()
	payload := []byte("%PDF-1.7 fake pdf body for streaming test")
	server := newStreamMock(t, payload, false)
	defer server.Close()

	c := gowa.New(server.URL, "", "")
	account := &whatsapp.Account{GowaDeviceID: "dev1"}
	dest := filepath.Join(t.TempDir(), "out.pdf")

	mediaType, written, err := c.DownloadMessageMediaToPath(context.Background(), account, "MSG1", "628@s.whatsapp.net", dest, 1<<20)
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), written)
	assert.Equal(t, "document", mediaType)
	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
	_, statErr := os.Stat(dest + ".part")
	assert.True(t, os.IsNotExist(statErr), ".part must be renamed away on success")
}

func TestDownloadMessageMediaToPath_AbortsWhenCapExceeded(t *testing.T) {
	t.Parallel()
	server := newStreamMock(t, nil, true)
	defer server.Close()

	c := gowa.New(server.URL, "", "")
	account := &whatsapp.Account{GowaDeviceID: "dev1"}
	dest := filepath.Join(t.TempDir(), "out.pdf")

	_, _, err := c.DownloadMessageMediaToPath(context.Background(), account, "MSG1", "628@s.whatsapp.net", dest, 4096)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max download size")
	_, statErr := os.Stat(dest)
	assert.True(t, os.IsNotExist(statErr), "oversized file must not land at dest")
	_, partErr := os.Stat(dest + ".part")
	assert.True(t, os.IsNotExist(partErr), "partial file must be cleaned up")
}
