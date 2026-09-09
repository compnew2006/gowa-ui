package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

// cdnMediaServer mimics a GOWA v9.3+ instance: the webhook carries an absolute
// WhatsApp CDN URL (which must never be fetched), while the engine's
// /message/{id}/download endpoint answers with the decrypted file.
func cdnMediaServer(t *testing.T, payload []byte, hits *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/message/MSG-CDN-1/download":
			*hits++
			assert.Equal(t, "966501234567@s.whatsapp.net", r.URL.Query().Get("phone"),
				"bare phone from the webhook must be normalized to a full JID")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": map[string]any{"file_path": "/statics/cdn.bin", "media_type": "image"},
			})
		case "/statics/cdn.bin":
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
}

// cdnTestApp builds an App whose GOWA registry resolves every account to a
// client pointed at the given test server (no credentials — the mock ignores
// auth, matching how the production factory resolves Basic Auth per org).
func cdnTestApp(storage string, serverURL string) *App {
	app := &App{Config: &config.Config{}, Log: logf.New(logf.Opts{})}
	app.Config.Storage.LocalPath = storage
	app.WARegistry = whatsapp.NewRegistryWithFactory(
		logf.New(logf.Opts{}),
		func(uuid.UUID, string) (string, string) { return "", "" },
		func(baseURL, _, _ string) whatsapp.Provider {
			return gowa.New(serverURL, "", "")
		},
	)
	return app
}

// TestDownloadAndSaveMediaForMessage_CDNURLFallsBackToMessageDownload pins the
// GOWA v9.3+ compatibility path: the webhook media field carries an absolute
// mmg.whatsapp.net URL, which the SSRF gate rightly refuses — the download must
// fall back to the engine's decrypting /message/{id}/download endpoint and
// still persist the bytes locally.
func TestDownloadAndSaveMediaForMessage_CDNURLFallsBackToMessageDownload(t *testing.T) {
	t.Parallel()
	payload := []byte("\xff\xd8\xff\xe0 fake jpeg bytes for the CDN fallback test")
	hits := 0
	server := cdnMediaServer(t, payload, &hits)
	defer server.Close()

	app := cdnTestApp(t.TempDir(), server.URL)
	cdnURL := "https://mmg.whatsapp.net/v/t62.7118-24/606653704_1079796527927832_n.enc?ccb=11-4"

	rel, err := app.DownloadAndSaveMediaForMessage(t.Context(), cdnURL, "image/jpeg",
		&whatsapp.Account{GowaBaseURL: server.URL, GowaDeviceID: "dev1"},
		"MSG-CDN-1", "966501234567")
	require.NoError(t, err)
	assert.Equal(t, 1, hits, "engine download endpoint must be used exactly once")
	assert.Equal(t, "images", filepath.Dir(rel), "image media belongs in images/")

	got, err := os.ReadFile(filepath.Join(app.Config.Storage.LocalPath, rel))
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

// TestDownloadAndSaveMedia_CDNURLWithoutMessageIDStillRefused pins that the
// fallback never widens the SSRF gate: an off-host absolute URL with no message
// identity to retry with stays refused.
func TestDownloadAndSaveMedia_CDNURLWithoutMessageIDStillRefused(t *testing.T) {
	t.Parallel()
	hits := 0
	server := cdnMediaServer(t, []byte("x"), &hits)
	defer server.Close()

	app := cdnTestApp(t.TempDir(), server.URL)
	cdnURL := "https://mmg.whatsapp.net/v/t62.7118-24/whatever_n.enc?ccb=11-4"

	_, err := app.DownloadAndSaveMedia(t.Context(), cdnURL, "image/jpeg",
		&whatsapp.Account{GowaBaseURL: server.URL, GowaDeviceID: "dev1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refusing media URL not on the GOWA base host")
	assert.Zero(t, hits, "no engine download may happen without a message id")
}

// TestNormalizeChatJID pins the bare-identifier normalization used by the
// CDN fallback (phones → @s.whatsapp.net, group ids → @g.us, JIDs untouched).
func TestNormalizeChatJID(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"966501234567":                   "966501234567@s.whatsapp.net",
		"12036299abcdef":                 "12036299abcdef@g.us",
		"12036301122":                    "12036301122@g.us",
		"966501234567:27@s.whatsapp.net": "966501234567:27@s.whatsapp.net",
		"":                               "",
	}
	for in, want := range cases {
		assert.Equal(t, want, normalizeChatJID(in), "input %q", in)
	}
}
