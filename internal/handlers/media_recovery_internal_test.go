package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

// TestRecoverMediaToDisk_StreamsToConfiguredStorage pins the streamed recovery
// contract: the file lands under the message type's subdir with the original
// filename's extension, the MIME is sniffed from the saved bytes, and the
// configurable cap (not the in-memory 50MiB constant) bounds the transfer.
func TestRecoverMediaToDisk_StreamsToConfiguredStorage(t *testing.T) {
	t.Parallel()
	storage := t.TempDir()
	app := &App{
		Config: &config.Config{},
		Log:    logf.New(logf.Opts{}),
	}
	app.Config.Storage.LocalPath = storage
	app.Config.Storage.MaxMediaDownloadMB = 1

	payload := []byte("%PDF-1.7 streamed recovery test body")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/message/W1/download":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": map[string]any{"file_path": "/statics/m.bin", "media_type": "document"},
			})
		case "/statics/m.bin":
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	msg := models.Message{
		WhatsAppMessageID: "W1",
		MediaFilename:     "Scan.pdf",
		MessageType:       models.MessageTypeDocument,
	}
	client := gowa.New(server.URL, "", "")
	waAccount := &whatsapp.Account{GowaDeviceID: "dev1"}

	rel, mime, err := app.recoverMediaToDisk(client, waAccount, msg, "628123@s.whatsapp.net")
	require.NoError(t, err)

	assert.Equal(t, "documents", filepath.Dir(rel), "document media belongs in documents/")
	assert.Equal(t, ".pdf", filepath.Ext(rel), "original filename extension must be preserved")
	assert.Equal(t, "application/pdf", mime, "MIME must be sniffed from the saved bytes")

	got, err := os.ReadFile(filepath.Join(storage, rel))
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

// TestRecoverMediaToDisk_CapAbortsCleanly pins the cap semantics: an
// over-budget stream must fail with the explicit size error and leave no
// file (nor .part) behind.
func TestRecoverMediaToDisk_CapAbortsCleanly(t *testing.T) {
	t.Parallel()
	storage := t.TempDir()
	app := &App{
		Config: &config.Config{},
		Log:    logf.New(logf.Opts{}),
	}
	app.Config.Storage.LocalPath = storage
	app.Config.Storage.MaxMediaDownloadMB = 1

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/message/W2/download":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": map[string]any{"file_path": "/statics/big.bin", "media_type": "document"},
			})
		case "/statics/big.bin":
			for {
				if _, err := w.Write(make([]byte, 64*1024)); err != nil {
					return
				}
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	msg := models.Message{
		WhatsAppMessageID: "W2",
		MediaFilename:     "Huge.pdf",
		MessageType:       models.MessageTypeDocument,
	}
	client := gowa.New(server.URL, "", "")

	_, _, err := app.recoverMediaToDisk(client, &whatsapp.Account{GowaDeviceID: "dev1"}, msg, "628123@s.whatsapp.net")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max download size")

	// Nothing may remain in storage after an aborted recovery.
	err = filepath.Walk(storage, func(p string, info os.FileInfo, werr error) error {
		if werr != nil || info.IsDir() {
			return werr
		}
		t.Errorf("stray file after aborted recovery: %s", p)
		return nil
	})
	require.NoError(t, err)
}
