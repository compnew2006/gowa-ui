package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// newMessagesSyncMock returns a GOWA server that serves:
//   - GET /chats → one 1:1 chat
//   - GET /chat/{jid}/messages → one MEDIA message (image) carrying a GOWA
//     server URL as the media URL but NO actual downloadable bytes.
//
// This is the exact shape that exposed the bug: history sync returned a media
// message whose URL the importer wrote straight into messages.media_url,
// creating a lying row (claims local media, file never downloaded).
// newMessagesSyncMock returns a GOWA server that serves:
//   - GET /chats → one 1:1 chat
//   - GET /chat/{jid}/messages → one MEDIA message (image) carrying a GOWA
//     server URL as the media URL.
//   - GET /message/{id}/download?phone={jid} → JSON pointing at a file_path,
//     which DownloadMessageMedia then fetches.
//   - GET /statics/photo.jpg → the actual media bytes (a tiny fake JPEG).
//
// When downloadFails is true, the /message/{id}/download endpoint returns a
// 500 to simulate an expired/unreachable WhatsApp CDN link — used to verify
// the sync degrades gracefully and leaves media_url empty.
func newMessagesSyncMock(t *testing.T) *httptest.Server {
	t.Helper()
	return newMessagesSyncMockOpt(t, "HIST_IMG_MSG_001", false)
}

// newMessagesSyncMockOpt builds a GOWA mock. messageID must be unique per test
// to avoid the sync's existing-message dedup (which skips wamids already in
// the DB) cross-contaminating tests that share the same wamid.
func newMessagesSyncMockOpt(t *testing.T, messageID string, downloadFails bool) *httptest.Server {
	t.Helper()
	// Minimal valid JPEG bytes (SOI + EOI). http.DetectContentType must read
	// this as image/jpeg so the sync code's MIME sniff produces a real type.
	fakeJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0xFF, 0xD9}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, _ := url.PathUnescape(r.URL.Path)
		switch {
		case path == "/chats":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    "SUCCESS",
				"message": "Success get chat list",
				"results": map[string]any{
					"data": []map[string]any{
						{"jid": "16505551234@s.whatsapp.net", "name": "Alice One"},
					},
					"pagination": map[string]any{"total": 1, "limit": 100, "offset": 0},
				},
			})
		case strings.Contains(path, "/message/") && strings.HasSuffix(path, "/download"):
			// GET /message/{id}/download?phone={jid} — the message-ID download
			// endpoint. DownloadMessageMedia parses results.file_path and then
			// fetches it joined to the server base URL.
			if downloadFails {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    "SUCCESS",
				"message": "Success download media",
				"results": map[string]any{
					"message_id": messageID,
					"media_type": "image",
					"filename":   "photo.jpg",
					"file_path":  "/statics/photo.jpg",
				},
			})
		case path == "/statics/photo.jpg":
			// The actual media bytes, fetched after the download endpoint.
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(fakeJPEG)
		case strings.Contains(path, "/messages"):
			// GET /chat/{encoded-jid}/messages — match by suffix to ignore the
			// URL-encoded JID in the path.
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    "SUCCESS",
				"message": "Success get messages",
				"results": map[string]any{
					"data": []map[string]any{
						{
							"id":         messageID,
							"chat_jid":   "16505551234@s.whatsapp.net",
							"sender_jid": "16505551234@s.whatsapp.net",
							"content":    "",
							"timestamp":  "2026-07-20T10:00:00Z",
							"is_from_me": false,
							"media_type": "image",
							"filename":   "photo.jpg",
							// GOWA's history API returns a WhatsApp CDN URL here;
							// the sync code must NOT treat it as a local path but
							// instead download via the message-ID endpoint.
							"url": "https://mmg.whatsapp.net/fake/photo.enc",
						},
					},
					"pagination": map[string]any{"total": 1, "limit": 50, "offset": 0},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server
}

// TestSyncGowaInstanceMessages_MediaDownloadedAtSyncTime pins the fix for the
// empty-media_url problem on history-synced media:
//
// Before the fix, history-sync stored MediaURL="" intentionally (because
// GOWA's history URL is a WhatsApp CDN link, not a local path). The bytes were
// only ever fetched lazily via ServeMedia on first view — but by then the CDN
// link had usually expired (403), so the media was lost forever.
//
// Regression contract (post-fix): history sync MUST download the media bytes
// at sync time via the GOWA message-ID endpoint and write a real local path
// into media_url. The original WhatsApp CDN URL is still preserved in
// metadata.gowa_media_url for diagnostics.
func TestSyncGowaInstanceMessages_MediaDownloadedAtSyncTime(t *testing.T) {
	t.Parallel()

	mock := newMessagesSyncMock(t)
	defer mock.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)

	inst := &models.GowaInstance{
		OrganizationID: org.ID,
		Name:           "msg-sync-server-" + uuid.New().String()[:8],
		BaseURL:        mock.URL,
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(inst).Error)

	deviceID := "dev-msg-sync-" + uuid.New().String()[:8]
	accountName := "gowa-msg-sync-" + uuid.New().String()[:8]
	acc := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           accountName,
		GowaDeviceID:   deviceID,
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(acc).Error)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	testutil.SetPathParam(req, "deviceId", deviceID)

	require.NoError(t, app.SyncGowaInstanceMessages(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"want 200, body=%s", string(testutil.GetResponseBody(req)))

	// The image message from history must have been imported...
	var msg models.Message
	require.NoError(t, app.DB.Where("whats_app_message_id = ?", "HIST_IMG_MSG_001").First(&msg).Error,
		"history-sync image message must be imported")

	// ...and its media_url MUST be populated with a real local path (the fix).
	// The bytes were downloaded via the message-ID endpoint at sync time, before
	// the WhatsApp CDN link could expire.
	assert.NotEmpty(t, msg.MediaURL,
		"history-sync media_url must be populated at sync time — "+
			"bytes are downloaded via the GOWA message-ID endpoint, not left empty")
	// The MIME type is sniffed from the downloaded bytes (not GOWA's generic
	// "image" media_type), so it must be a real image/* type.
	assert.Contains(t, msg.MediaMimeType, "image/",
		"media_mime_type must be sniffed to a real image MIME type")

	// The original WhatsApp CDN URL is preserved in metadata for diagnostics.
	assert.Equal(t, "https://mmg.whatsapp.net/fake/photo.enc",
		msg.Metadata["gowa_media_url"],
		"GOWA's original media URL should be stashed in metadata.gowa_media_url")

	// The history flag is still set (used by the DB cleanup script + future
	// audits to identify history-synced rows).
	assert.Equal(t, true, msg.Metadata["synced_from_history"],
		"synced_from_history flag must be preserved")
}

// TestSyncGowaInstanceMessages_MediaDownloadFailureLeavesEmptyUrl verifies that
// when the GOWA media download fails at sync time (e.g. expired WhatsApp CDN
// link → 500), the sync degrades gracefully:
//   - the message row is still created (sync is not aborted),
//   - media_url stays empty (no lying row),
//   - ServeMedia's lazy recovery path remains as the fallback.
//
// This pins the non-fatal contract: a single media download failure must not
// break the whole history backfill.
func TestSyncGowaInstanceMessages_MediaDownloadFailureLeavesEmptyUrl(t *testing.T) {
	t.Parallel()

	// Unique wamid so the sync's existing-message dedup doesn't skip this row
	// based on a wamid left over from TestSyncGowaInstanceMessages_MediaDownloadedAtSyncTime.
	failMsgID := "HIST_IMG_FAIL_001"
	mock := newMessagesSyncMockOpt(t, failMsgID, true) // download endpoint returns 500
	defer mock.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)

	inst := &models.GowaInstance{
		OrganizationID: org.ID,
		Name:           "msg-sync-server-" + uuid.New().String()[:8],
		BaseURL:        mock.URL,
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(inst).Error)

	deviceID := "dev-msg-sync-" + uuid.New().String()[:8]
	accountName := "gowa-msg-sync-" + uuid.New().String()[:8]
	acc := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           accountName,
		GowaDeviceID:   deviceID,
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(acc).Error)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", inst.ID.String())
	testutil.SetPathParam(req, "deviceId", deviceID)

	// The sync must still succeed — the download failure is non-fatal.
	require.NoError(t, app.SyncGowaInstanceMessages(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"history sync must not fail when a single media download fails")

	// The message row is still created...
	var msg models.Message
	require.NoError(t, app.DB.Where("whats_app_message_id = ?", failMsgID).First(&msg).Error,
		"message row must be created even when media download fails")

	// ...but media_url stays empty (graceful degradation).
	assert.Empty(t, msg.MediaURL,
		"media_url must stay empty when the download fails at sync time — "+
			"ServeMedia's lazy recovery remains as the fallback")
}

// TestAutoSyncGowaHistory covers the automatic (no-button) backfill path:
// AutoSyncGowaHistory must import history on first run, then honor the
// per-account cooldown on immediate re-runs, and no-op for accounts without
// a GOWA device.
func TestAutoSyncGowaHistory(t *testing.T) {
	t.Parallel()

	mock := newMessagesSyncMock(t)
	defer mock.Close()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	deviceID := "dev-auto-sync-" + uuid.New().String()[:8]
	accountName := "gowa-auto-sync-" + uuid.New().String()[:8]
	acc := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           accountName,
		GowaDeviceID:   deviceID,
		GowaBaseURL:    mock.URL,
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(acc).Error)

	// First run: history is imported without any user action.
	app.AutoSyncGowaHistory(acc)

	var msg models.Message
	require.NoError(t, app.DB.
		Where("whats_app_message_id = ? AND whats_app_account = ?", "HIST_IMG_MSG_001", accountName).
		First(&msg).Error,
		"auto-sync must import history messages")
	assert.Equal(t, true, msg.Metadata["synced_from_history"])

	// Second run within the cooldown window: must be skipped entirely. Delete
	// the imported row so a (wrong) re-sync would visibly re-create it.
	require.NoError(t, app.DB.Unscoped().Delete(&msg).Error)
	app.AutoSyncGowaHistory(acc)

	var count int64
	app.DB.Model(&models.Message{}).
		Where("whats_app_message_id = ? AND whats_app_account = ?", "HIST_IMG_MSG_001", accountName).
		Count(&count)
	assert.Equal(t, int64(0), count,
		"a second auto-sync within the cooldown window must be skipped")

	// Accounts without a GOWA device are ignored (no panic, no import).
	noDevice := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "no-device-" + uuid.New().String()[:8],
		GowaBaseURL:    mock.URL,
		Status:         "active",
	}
	require.NoError(t, app.DB.Create(noDevice).Error)
	app.AutoSyncGowaHistory(noDevice)

	app.DB.Model(&models.Message{}).
		Where("whats_app_account = ?", noDevice.Name).
		Count(&count)
	assert.Equal(t, int64(0), count,
		"accounts without a GOWA device must not be synced")
}
