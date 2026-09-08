package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

// newBackfillTestApp builds a minimal App with a real test DB (no Redis /
// WARegistry needed — the units under test are pendingBackfillQuery,
// backfillWithClient, and backfillMessageMedia's skip/mark logic).
func newBackfillTestApp(t *testing.T) *App {
	t.Helper()
	app := &App{
		Config: &config.Config{
			App: config.AppConfig{EncryptionKey: "test-encryption-key-for-handlers-longer-than-32-chars"},
			JWT: config.JWTConfig{Secret: testutil.TestJWTSecret},
		},
		DB:  testutil.SetupTestDB(t),
		Log: logf.New(logf.Opts{}),
	}
	app.Config.Storage.LocalPath = t.TempDir()
	app.Config.Storage.MaxMediaDownloadMB = 5
	return app
}

// backfillTestOrg seeds an org + a regular phone contact.
func backfillTestOrg(t *testing.T, app *App) (uuid.UUID, *models.Contact) {
	t.Helper()
	org := testutil.CreateTestOrganization(t, app.DB)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	contact.PhoneNumber = "966501234567@s.whatsapp.net"
	require.NoError(t, app.DB.Save(contact).Error)
	return org.ID, contact
}

// fakeGowaMediaServer serves the /message/{id}/download → {file_path} → bytes
// handshake for one successful media id ("W_OK"); every other id 404s
// (media gone).
func fakeGowaMediaServer(t *testing.T, payload []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/message/W_OK/download":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": map[string]any{"file_path": "/statics/m.bin", "media_type": "image"},
			})
		case r.URL.Path == "/statics/m.bin":
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
}

// TestBackfillWithClient_RecoverAndClaim pins the happy path: pending media
// is streamed to disk, the row is claimed (media_url + sniffed mime + account
// re-link), and the file really exists in storage.
func TestBackfillWithClient_RecoverAndClaim(t *testing.T) {
	app := newBackfillTestApp(t)
	_, contact := backfillTestOrg(t, app)

	payload := []byte("\xFF\xD8\xFF\xE0fake-jpeg-backfill")
	server := fakeGowaMediaServer(t, payload)
	defer server.Close()

	msg := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    contact.OrganizationID,
		ContactID:         contact.ID,
		WhatsAppAccount:   "acct-a",
		WhatsAppMessageID: "W_OK",
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeImage,
		MediaURL:          "",
		Status:            models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msg).Error)

	res := app.backfillWithClient(
		gowa.New(server.URL, "", ""),
		&whatsapp.Account{GowaDeviceID: "dev1"},
		"acct-live", msg, contact)
	require.Equal(t, gowaBackfillRecovered, res)

	var after models.Message
	require.NoError(t, app.DB.Where("id = ?", msg.ID).First(&after).Error)
	assert.NotEmpty(t, after.MediaURL, "media_url must be claimed")
	assert.Equal(t, "image/jpeg", after.MediaMimeType, "mime sniffed from saved bytes")
	assert.Equal(t, "acct-live", after.WhatsAppAccount, "row re-linked to the recovering account")

	_, err := os.Stat(filepath.Join(app.Config.Storage.LocalPath, after.MediaURL))
	require.NoError(t, err, "file must exist on disk")
}

// TestBackfillWithClient_MediaGoneMarksPermanent pins the expiry path: a 404
// from the provider marks the row permanent so the scan stops selecting it.
func TestBackfillWithClient_MediaGoneMarksPermanent(t *testing.T) {
	app := newBackfillTestApp(t)
	_, contact := backfillTestOrg(t, app)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r) // every download 404s: media expired
	}))
	defer server.Close()

	msg := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    contact.OrganizationID,
		ContactID:         contact.ID,
		WhatsAppMessageID: "W_GONE",
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeImage,
		MediaURL:          "",
		Status:            models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msg).Error)

	res := app.backfillWithClient(
		gowa.New(server.URL, "", ""),
		&whatsapp.Account{GowaDeviceID: "dev1"},
		"acct-live", msg, contact)
	require.Equal(t, gowaBackfillPermanent, res)

	var after models.Message
	require.NoError(t, app.DB.Where("id = ?", msg.ID).First(&after).Error)
	require.NotNil(t, after.Metadata)
	assert.Equal(t, true, after.Metadata[gowaBackfillPermanentKey])
	assert.Empty(t, after.MediaURL)

	var count int64
	app.pendingBackfillQuery().Where("id = ?", msg.ID).Count(&count)
	assert.Zero(t, count, "permanently-failed row must be excluded from future scans")
}

// TestPendingBackfillQuery_Eligibility exercises the scan's selection rules
// directly: only media types, only empty media_url with a wamid, never
// exhausted or permanently-failed rows.
func TestPendingBackfillQuery_Eligibility(t *testing.T) {
	app := newBackfillTestApp(t)
	org, contact := backfillTestOrg(t, app)

	mk := func(msgType models.MessageType, mediaURL, wamid string, meta models.JSONB) *models.Message {
		m := &models.Message{
			BaseModel:         models.BaseModel{ID: uuid.New()},
			OrganizationID:    org,
			ContactID:         contact.ID,
			MessageType:       msgType,
			MediaURL:          mediaURL,
			WhatsAppMessageID: wamid,
			Metadata:          meta,
			Status:            models.MessageStatusDelivered,
		}
		require.NoError(t, app.DB.Create(m).Error)
		return m
	}

	pending := mk(models.MessageTypeImage, "", "W1", nil) // eligible
	mk(models.MessageTypeText, "", "W2", nil)             // not a media type
	mk(models.MessageTypeImage, "images/x.jpg", "", nil)  // already local / no wamid
	mk(models.MessageTypeDocument, "", "W4", models.JSONB{
		gowaBackfillAttemptsKey: float64(gowaBackfillMaxAttempts), // attempts exhausted
	})
	mk(models.MessageTypeImage, "", "W5", models.JSONB{
		gowaBackfillPermanentKey: true, // permanently gone
	})

	var ids []uuid.UUID
	require.NoError(t, app.pendingBackfillQuery().Pluck("id", &ids).Error)
	assert.Equal(t, []uuid.UUID{pending.ID}, ids,
		"only the untouched pending media row is eligible")
}

// TestBackfillMessageMedia_StatusLikeContactsMarkedPermanent pins that
// status/newsletter contacts are marked permanent instead of retried forever
// (GOWA's chat download endpoint can never serve their media).
func TestBackfillMessageMedia_StatusLikeContactsMarkedPermanent(t *testing.T) {
	app := newBackfillTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	contact.PhoneNumber = "1234@newsletter"
	require.NoError(t, app.DB.Save(contact).Error)

	msg := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    org.ID,
		ContactID:         contact.ID,
		WhatsAppMessageID: "W_NL",
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeImage,
		MediaURL:          "",
		Status:            models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msg).Error)

	assert.Equal(t, gowaBackfillPermanent, app.backfillMessageMedia(msg))

	var after models.Message
	require.NoError(t, app.DB.Where("id = ?", msg.ID).First(&after).Error)
	require.NotNil(t, after.Metadata)
	assert.Equal(t, true, after.Metadata[gowaBackfillPermanentKey])
}

// TestBackfillProcessor_RunPassSkipsWithoutAccount runs one full pass with no
// GOWA account in the org: the pass must tolerate a provider-less environment
// (skip, no crash) and leave the row pending and eligible for a later pass —
// an account may be added afterwards.
func TestBackfillProcessor_RunPassSkipsWithoutAccount(t *testing.T) {
	app := newBackfillTestApp(t)
	_, contact := backfillTestOrg(t, app)

	msg := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    contact.OrganizationID,
		ContactID:         contact.ID,
		WhatsAppMessageID: "W1",
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeImage,
		MediaURL:          "",
		Status:            models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msg).Error)

	p := NewGowaMediaBackfillProcessor(app, time.Minute)
	p.runPass(context.Background())

	var after models.Message
	require.NoError(t, app.DB.Where("id = ?", msg.ID).First(&after).Error)
	assert.Empty(t, after.MediaURL, "no account → nothing recovered")

	var count int64
	app.pendingBackfillQuery().Where("id = ?", msg.ID).Count(&count)
	assert.Equal(t, int64(1), count, "skipped rows must remain eligible")
}

// TestBackfillWithClient_TransientFailureCountsAttempts pins that a
// non-gone provider error (e.g. 500) increments mbf_attempts without marking
// permanent, and the row stays scannable until the cap.
func TestBackfillWithClient_TransientFailureCountsAttempts(t *testing.T) {
	app := newBackfillTestApp(t)
	_, contact := backfillTestOrg(t, app)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // transient
	}))
	defer server.Close()

	msg := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    contact.OrganizationID,
		ContactID:         contact.ID,
		WhatsAppMessageID: "W_500",
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeImage,
		MediaURL:          "",
		Status:            models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msg).Error)

	res := app.backfillWithClient(
		gowa.New(server.URL, "", ""),
		&whatsapp.Account{GowaDeviceID: "dev1"},
		"acct-live", msg, contact)
	require.Equal(t, gowaBackfillTransient, res)

	var after models.Message
	require.NoError(t, app.DB.Where("id = ?", msg.ID).First(&after).Error)
	require.NotNil(t, after.Metadata)
	assert.Equal(t, float64(1), after.Metadata[gowaBackfillAttemptsKey])
	assert.Nil(t, after.Metadata[gowaBackfillPermanentKey], "transient failures stay retryable")

	var count int64
	app.pendingBackfillQuery().Where("id = ?", msg.ID).Count(&count)
	assert.Equal(t, int64(1), count, "row stays eligible after a transient failure")
}
