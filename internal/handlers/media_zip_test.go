package handlers_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestServeMediaZip_NoIDs(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	// no ids param

	err := app.ServeMediaZip(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestServeMediaZip_OrgIsolation(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	mediaDir := t.TempDir()
	app.Config.Storage.LocalPath = mediaDir

	orgA := testutil.CreateTestOrganization(t, app.DB)
	orgB := testutil.CreateTestOrganization(t, app.DB)
	userA := createAdminUser(t, app, orgA.ID)
	contactA := testutil.CreateTestContact(t, app.DB, orgA.ID)
	contactB := testutil.CreateTestContact(t, app.DB, orgB.ID)

	// A media file that both orgs' messages will "point at" — but only orgA's
	// message should be returned, because the query is org-scoped.
	rel := filepath.Join("images", "shared.jpg")
	require.NoError(t, os.MkdirAll(filepath.Join(mediaDir, "images"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(mediaDir, rel), []byte("org-a-bytes"), 0644))

	msgA := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgA.ID,
		ContactID:      contactA.ID,
		Direction:      models.DirectionIncoming,
		MessageType:    models.MessageTypeImage,
		MediaURL:       rel,
		MediaFilename:  "shared.jpg",
		MediaMimeType:  "image/jpeg",
		Status:         models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msgA).Error)

	msgB := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgB.ID,
		ContactID:      contactB.ID,
		Direction:      models.DirectionIncoming,
		MessageType:    models.MessageTypeImage,
		MediaURL:       rel,
		MediaFilename:  "shared.jpg",
		MediaMimeType:  "image/jpeg",
		Status:         models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msgB).Error)

	// userA requests both IDs — only msgA may come back.
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgA.ID, userA.ID)
	testutil.SetQueryParam(req, "ids", msgA.ID.String()+","+msgB.ID.String())

	err := app.ServeMediaZip(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"body: %s", string(testutil.GetResponseBody(req)))

	zr, err := zip.NewReader(bytes.NewReader(testutil.GetResponseBody(req)), int64(len(testutil.GetResponseBody(req))))
	require.NoError(t, err)

	// Exactly one entry: the one image. No manifest, no orgB's message.
	names := zipEntryNames(zr)
	assert.Contains(t, names, "shared.jpg")
	assert.NotContains(t, names, "_manifest.txt")
	assert.Len(t, names, 1, "orgB's message must be excluded")
}

func TestServeMediaZip_FilenameCollision(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	mediaDir := t.TempDir()
	app.Config.Storage.LocalPath = mediaDir

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Two distinct files on disk, both reported as "image.jpg".
	require.NoError(t, os.MkdirAll(filepath.Join(mediaDir, "images"), 0755))
	rel1 := filepath.Join("images", "img1.jpg")
	rel2 := filepath.Join("images", "img2.jpg")
	require.NoError(t, os.WriteFile(filepath.Join(mediaDir, rel1), []byte("first"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(mediaDir, rel2), []byte("second"), 0644))

	m1 := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID, ContactID: contact.ID,
		Direction: models.DirectionIncoming, MessageType: models.MessageTypeImage,
		MediaURL: rel1, MediaFilename: "image.jpg", MediaMimeType: "image/jpeg",
		Status: models.MessageStatusDelivered,
	}
	m2 := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID, ContactID: contact.ID,
		Direction: models.DirectionIncoming, MessageType: models.MessageTypeImage,
		MediaURL: rel2, MediaFilename: "image.jpg", MediaMimeType: "image/jpeg",
		Status: models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(m1).Error)
	require.NoError(t, app.DB.Create(m2).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "ids", m1.ID.String()+","+m2.ID.String())

	require.NoError(t, app.ServeMediaZip(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	zr, err := zip.NewReader(bytes.NewReader(testutil.GetResponseBody(req)), int64(len(testutil.GetResponseBody(req))))
	require.NoError(t, err)

	names := zipEntryNames(zr)
	// One keeps the original name, the other is suffixed — both must be present
	// and distinct. uniqueZipName suffixes as "<base>_<id>.<ext>", so filter on
	// the stem, not the full original filename.
	imgEntries := filterPrefix(names, "image")
	require.Len(t, imgEntries, 2, "expected two distinct entries for colliding names, got %v", names)
	assert.Contains(t, imgEntries, "image.jpg")
	assert.NotEqual(t, imgEntries[0], imgEntries[1])
}

func TestServeMediaZip_PathTraversalSkipped(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	mediaDir := t.TempDir()
	app.Config.Storage.LocalPath = mediaDir

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// A legitimate file...
	require.NoError(t, os.MkdirAll(filepath.Join(mediaDir, "documents"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(mediaDir, "documents", "ok.pdf"), []byte("ok"), 0644))
	// ...and a traversal-escaping path that must be silently skipped.
	secretPath := filepath.Join(t.TempDir(), "secret.txt")
	require.NoError(t, os.WriteFile(secretPath, []byte("topsecret"), 0644))

	good := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID, ContactID: contact.ID,
		Direction: models.DirectionIncoming, MessageType: models.MessageTypeDocument,
		MediaURL: filepath.Join("documents", "ok.pdf"), MediaFilename: "ok.pdf", MediaMimeType: "application/pdf",
		Status: models.MessageStatusDelivered,
	}
	// Resolve the relative traversal segment that points outside mediaDir.
	evilRel := traversalRel(t, mediaDir, secretPath)
	evil := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID, ContactID: contact.ID,
		Direction: models.DirectionIncoming, MessageType: models.MessageTypeDocument,
		MediaURL: evilRel, MediaFilename: "secret.txt",
		Status: models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(good).Error)
	require.NoError(t, app.DB.Create(evil).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "ids", good.ID.String()+","+evil.ID.String())

	require.NoError(t, app.ServeMediaZip(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	zr, err := zip.NewReader(bytes.NewReader(testutil.GetResponseBody(req)), int64(len(testutil.GetResponseBody(req))))
	require.NoError(t, err)

	names := zipEntryNames(zr)
	assert.Contains(t, names, "ok.pdf")
	for _, n := range names {
		assert.NotContains(t, n, "secret", "traversal target must not appear in archive")
	}
}

func TestServeMediaZip_NoManifestEntry(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	mediaDir := t.TempDir()
	app.Config.Storage.LocalPath = mediaDir

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	require.NoError(t, os.MkdirAll(filepath.Join(mediaDir, "images"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(mediaDir, "images", "a.png"), []byte("png-bytes"), 0644))

	msg := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID, ContactID: contact.ID,
		Direction: models.DirectionIncoming, MessageType: models.MessageTypeImage,
		MediaURL: filepath.Join("images", "a.png"), MediaFilename: "a.png", MediaMimeType: "image/png",
		Status: models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msg).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "ids", msg.ID.String())

	require.NoError(t, app.ServeMediaZip(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	zr, err := zip.NewReader(bytes.NewReader(testutil.GetResponseBody(req)), int64(len(testutil.GetResponseBody(req))))
	require.NoError(t, err)

	// The archive must contain only the media file — no _manifest.txt entry.
	names := zipEntryNames(zr)
	assert.Contains(t, names, "a.png")
	assert.NotContains(t, names, "_manifest.txt", "manifest entry was removed — zips carry media only")
	assert.Len(t, names, 1)
}

// An agent (contacts:read, no contacts:export) must be able to collect a
// burst of incoming files as a ZIP — the archive bundles the same files the
// per-message ServeMedia endpoint already serves them, so gating it on an
// export permission only blocked the feature for its target users.
func TestServeMediaZip_AgentWithoutExport(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	mediaDir := t.TempDir()
	app.Config.Storage.LocalPath = mediaDir

	org := testutil.CreateTestOrganization(t, app.DB)
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)
	agent := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithRoleID(&agentRole.ID))
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithAssignee(agent.ID))

	require.NoError(t, os.MkdirAll(filepath.Join(mediaDir, "images"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(mediaDir, "images", "doc.png"), []byte("png"), 0644))

	msg := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID, ContactID: contact.ID,
		Direction: models.DirectionIncoming, MessageType: models.MessageTypeImage,
		MediaURL: filepath.Join("images", "doc.png"), MediaFilename: "doc.png", MediaMimeType: "image/png",
		Status: models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msg).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetQueryParam(req, "ids", msg.ID.String())

	err := app.ServeMediaZip(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"agent must download the burst ZIP without contacts:export; body: %s", string(testutil.GetResponseBody(req)))

	zr, err := zip.NewReader(bytes.NewReader(testutil.GetResponseBody(req)), int64(len(testutil.GetResponseBody(req))))
	require.NoError(t, err)
	assert.Contains(t, zipEntryNames(zr), "doc.png")
}

// A real WhatsApp burst can hold more than 50 files (the user's case: 55
// PDFs). The old hard cap answered 400 for any album above 50, so such an
// album could never be zipped. 60 files must all land in the archive.
func TestServeMediaZip_MoreThanFiftyFiles(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	mediaDir := t.TempDir()
	app.Config.Storage.LocalPath = mediaDir

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	require.NoError(t, os.MkdirAll(filepath.Join(mediaDir, "documents"), 0755))

	ids := make([]string, 0, 60)
	for i := 0; i < 60; i++ {
		name := fmt.Sprintf("scan-%02d.pdf", i)
		rel := filepath.Join("documents", name)
		require.NoError(t, os.WriteFile(filepath.Join(mediaDir, rel), []byte(name), 0644))
		msg := &models.Message{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: org.ID, ContactID: contact.ID,
			Direction: models.DirectionIncoming, MessageType: models.MessageTypeDocument,
			MediaURL: rel, MediaFilename: name, MediaMimeType: "application/pdf",
			Status: models.MessageStatusDelivered,
		}
		require.NoError(t, app.DB.Create(msg).Error)
		ids = append(ids, msg.ID.String())
	}

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "ids", strings.Join(ids, ","))

	require.NoError(t, app.ServeMediaZip(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"body: %s", string(testutil.GetResponseBody(req)))
	assert.Equal(t, "60", string(req.RequestCtx.Response.Header.Peek("X-Media-Zip-Included")))
	assert.Equal(t, "0", string(req.RequestCtx.Response.Header.Peek("X-Media-Zip-Skipped")))

	body := testutil.GetResponseBody(req)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	assert.Len(t, zipEntryNames(zr), 60)
}

// POST carries the IDs as a JSON body, so large albums never hit URL or
// header-size limits; duplicates are counted once.
func TestServeMediaZip_PostJSONBody(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	mediaDir := t.TempDir()
	app.Config.Storage.LocalPath = mediaDir

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	require.NoError(t, os.MkdirAll(filepath.Join(mediaDir, "images"), 0755))

	var ids []string
	for _, name := range []string{"a.jpg", "b.jpg"} {
		rel := filepath.Join("images", name)
		require.NoError(t, os.WriteFile(filepath.Join(mediaDir, rel), []byte(name), 0644))
		msg := &models.Message{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: org.ID, ContactID: contact.ID,
			Direction: models.DirectionIncoming, MessageType: models.MessageTypeImage,
			MediaURL: rel, MediaFilename: name, MediaMimeType: "image/jpeg",
			Status: models.MessageStatusDelivered,
		}
		require.NoError(t, app.DB.Create(msg).Error)
		ids = append(ids, msg.ID.String())
	}

	req := testutil.NewJSONRequest(t, map[string]any{"ids": append(ids, ids[0])})
	testutil.SetAuthContext(req, org.ID, user.ID)

	require.NoError(t, app.ServeMediaZip(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"body: %s", string(testutil.GetResponseBody(req)))
	assert.Equal(t, "2", string(req.RequestCtx.Response.Header.Peek("X-Media-Zip-Included")))
	assert.Equal(t, "0", string(req.RequestCtx.Response.Header.Peek("X-Media-Zip-Skipped")))

	body := testutil.GetResponseBody(req)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"a.jpg", "b.jpg"}, zipEntryNames(zr))
}

func TestServeMediaZip_PostValidation(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)

	tooMany := make([]string, 501)
	for i := range tooMany {
		tooMany[i] = uuid.New().String()
	}
	cases := map[string]any{
		"empty ids":    map[string]any{"ids": []string{}},
		"missing ids":  map[string]any{},
		"above cap":    map[string]any{"ids": tooMany},
		"only garbage": map[string]any{"ids": []string{"not-a-uuid"}},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			req := testutil.NewJSONRequest(t, body)
			testutil.SetAuthContext(req, org.ID, user.ID)
			require.NoError(t, app.ServeMediaZip(req))
			assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
		})
	}
}

// Files that can't be delivered (gone from disk with nothing to recover from,
// retention-purged) used to vanish from the archive silently. They are now
// counted in X-Media-Zip-Skipped so the client can warn, and a purged file is
// never resurrected through the ZIP.
func TestServeMediaZip_ReportsSkippedFiles(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	mediaDir := t.TempDir()
	app.Config.Storage.LocalPath = mediaDir

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	require.NoError(t, os.MkdirAll(filepath.Join(mediaDir, "documents"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(mediaDir, "documents", "ok.pdf"), []byte("ok"), 0644))

	ok := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID, ContactID: contact.ID,
		Direction: models.DirectionIncoming, MessageType: models.MessageTypeDocument,
		MediaURL: filepath.Join("documents", "ok.pdf"), MediaFilename: "ok.pdf",
		Status: models.MessageStatusDelivered,
	}
	gone := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID, ContactID: contact.ID,
		Direction: models.DirectionIncoming, MessageType: models.MessageTypeDocument,
		MediaURL: filepath.Join("documents", "deleted.pdf"), MediaFilename: "deleted.pdf",
		Status: models.MessageStatusDelivered,
	}
	purged := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID, ContactID: contact.ID,
		Direction: models.DirectionIncoming, MessageType: models.MessageTypeDocument,
		MediaURL: filepath.Join("documents", "ok.pdf"), MediaFilename: "purged.pdf",
		WhatsAppMessageID: "W_PURGED",
		Metadata:          models.JSONB{"retention_state": "purged"},
		Status:            models.MessageStatusDelivered,
	}
	for _, m := range []*models.Message{ok, gone, purged} {
		require.NoError(t, app.DB.Create(m).Error)
	}

	req := testutil.NewJSONRequest(t, map[string]any{"ids": []string{ok.ID.String(), gone.ID.String(), purged.ID.String()}})
	testutil.SetAuthContext(req, org.ID, user.ID)

	require.NoError(t, app.ServeMediaZip(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	assert.Equal(t, "1", string(req.RequestCtx.Response.Header.Peek("X-Media-Zip-Included")))
	assert.Equal(t, "2", string(req.RequestCtx.Response.Header.Peek("X-Media-Zip-Skipped")))

	body := testutil.GetResponseBody(req)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	assert.Equal(t, []string{"ok.pdf"}, zipEntryNames(zr))
}

// A file whose download failed at ingest (media_url = ”) is still visible in
// the chat (ServeMedia recovers it lazily) — the old ZIP filtered on
// media_url <> ” and dropped it. The ZIP must recover it from GOWA the same
// way, include it, and re-link the row.
func TestServeMediaZip_RecoversUndownloadedMedia(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	mediaDir := t.TempDir()
	app.Config.Storage.LocalPath = mediaDir
	app.Config.Storage.MaxMediaDownloadMB = 5

	payload := []byte("%PDF-1.4 recovered bytes")
	gowaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/message/W_MISSING/download":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": map[string]any{"file_path": "/statics/r.bin", "media_type": "document"},
			})
		case "/statics/r.bin":
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(gowaSrv.Close)
	app.WARegistry = whatsapp.NewRegistryWithFactory(
		app.Log,
		func(uuid.UUID, string) (string, string) { return "", "" },
		func(_, _, _ string) whatsapp.Provider { return gowa.New(gowaSrv.URL, "", "") },
	)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	msg := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID, ContactID: contact.ID,
		WhatsAppAccount:   account.Name,
		WhatsAppMessageID: "W_MISSING",
		Direction:         models.DirectionIncoming, MessageType: models.MessageTypeDocument,
		MediaURL: "", MediaFilename: "invoice.pdf",
		Status: models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msg).Error)

	req := testutil.NewJSONRequest(t, map[string]any{"ids": []string{msg.ID.String()}})
	testutil.SetAuthContext(req, org.ID, user.ID)

	require.NoError(t, app.ServeMediaZip(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"body: %s", string(testutil.GetResponseBody(req)))
	assert.Equal(t, "1", string(req.RequestCtx.Response.Header.Peek("X-Media-Zip-Included")))

	body := testutil.GetResponseBody(req)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	require.Equal(t, []string{"invoice.pdf"}, zipEntryNames(zr))
	f, err := zr.File[0].Open()
	require.NoError(t, err)
	got, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	var after models.Message
	require.NoError(t, app.DB.Where("id = ?", msg.ID).First(&after).Error)
	assert.NotEmpty(t, after.MediaURL, "recovered file must be linked to the message row")
}

// --- helpers ---

func zipEntryNames(zr *zip.Reader) []string {
	out := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		out = append(out, f.Name)
	}
	return out
}

func filterPrefix(in []string, prefix string) []string {
	var out []string
	for _, s := range in {
		if strings.HasPrefix(s, prefix) {
			out = append(out, s)
		}
	}
	return out
}

// traversalRel returns a media_url value that, when joined to mediaDir, would
// resolve to targetPath (escaping the storage root). Used to confirm the
// handler rejects such paths.
func traversalRel(t *testing.T, mediaDir, targetPath string) string {
	t.Helper()
	rel, err := filepath.Rel(mediaDir, targetPath)
	require.NoError(t, err)
	return rel
}
