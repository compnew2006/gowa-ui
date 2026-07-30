package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/test/testutil"
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
func newMessagesSyncMock(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/chats":
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
		default:
			// GET /chat/{encoded-jid}/messages — match by suffix to ignore the
			// URL-encoded JID in the path.
			path, _ := url.PathUnescape(r.URL.Path)
			if !strings.Contains(path, "/messages") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    "SUCCESS",
				"message": "Success get messages",
				"results": map[string]any{
					"data": []map[string]any{
						{
							"id":         "HIST_IMG_MSG_001",
							"chat_jid":   "16505551234@s.whatsapp.net",
							"sender_jid": "16505551234@s.whatsapp.net",
							"content":    "",
							"timestamp":  "2026-07-20T10:00:00Z",
							"is_from_me": false,
							"media_type": "image",
							"filename":   "photo.jpg",
							// GOWA returns its OWN server URL for the media. This is NOT
							// a local file path — bytes were never downloaded.
							"url": "http://gowa-mock/statics/photo.jpg",
						},
					},
					"pagination": map[string]any{"total": 1, "limit": 50, "offset": 0},
				},
			})
		}
	}))
	return server
}

// TestSyncGowaInstanceMessages_MediaUrlNotWrittenForUndownloadedBytes pins
// the fix for the lying-media_url bug:
// the fix for the lying-media_url bug:
//
// Before the fix, history-sync wrote GOWA's server URL straight into
// messages.media_url, creating rows that claimed local media existed when no
// bytes had ever been downloaded. ServeMedia then 404'd on every view.
//
// Regression contract: after a history sync that includes a media message,
// the imported message row must have an EMPTY media_url (bytes were not
// downloaded). The GOWA URL is preserved in metadata.gowa_media_url for the
// lazy-recovery path in ServeMedia to use on first view.
func TestSyncGowaInstanceMessages_MediaUrlNotWrittenForUndownloadedBytes(t *testing.T) {
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

	// ...and its media_url MUST be empty (the fix). Before the fix it was set
	// to GOWA's server URL, creating a lying row that ServeMedia could never
	// satisfy.
	assert.Empty(t, msg.MediaURL,
		"history-sync media_url must be empty when bytes were not downloaded — "+
			"ServeMedia's lazy recovery will fetch on first view via WhatsAppMessageID")

	// The original GOWA URL is preserved in metadata for the recovery path.
	assert.Equal(t, "http://gowa-mock/statics/photo.jpg",
		msg.Metadata["gowa_media_url"],
		"GOWA's original media URL should be stashed in metadata.gowa_media_url "+
			"as a fallback hint for lazy recovery")

	// The history flag is still set (used by the DB cleanup script + future
	// audits to identify history-synced rows).
	assert.Equal(t, true, msg.Metadata["synced_from_history"],
		"synced_from_history flag must be preserved")
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
