package handlers_test

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/gowa-ui/internal/models"
	"github.com/shridarpatil/gowa-ui/test/testutil"
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

	// Exactly two entries: the one image + _manifest.txt.
	names := zipEntryNames(zr)
	assert.Contains(t, names, "shared.jpg")
	assert.Contains(t, names, "_manifest.txt")
	assert.Len(t, names, 2, "orgB's message must be excluded")
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
	// and distinct.
	imgEntries := filterPrefix(names, "image.jpg")
	require.Len(t, imgEntries, 2, "expected two distinct entries for colliding names, got %v", names)
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

func TestServeMediaZip_Manifest(t *testing.T) {
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

	manifest := readZipEntry(t, zr, "_manifest.txt")
	assert.Contains(t, manifest, "message_id: "+msg.ID.String())
	assert.Contains(t, manifest, "direction:  incoming")
	assert.Contains(t, manifest, "type:       image")
	assert.Contains(t, manifest, "mime:       image/png")
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

func readZipEntry(t *testing.T, zr *zip.Reader, name string) string {
	t.Helper()
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		require.NoError(t, err)
		defer rc.Close()
		b, err := io.ReadAll(rc)
		require.NoError(t, err)
		return string(b)
	}
	require.Fail(t, "entry not found in zip: "+name)
	return ""
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
