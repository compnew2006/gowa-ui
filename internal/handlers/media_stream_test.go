package handlers_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	objectstorage "github.com/compnew2006/whatomate/internal/storage"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

type handlerFakeStorage struct {
	getCalls int
}

func (s *handlerFakeStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	_, err := io.Copy(io.Discard, body)
	return err
}

func (s *handlerFakeStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, objectstorage.ObjectInfo, error) {
	s.getCalls++
	return io.NopCloser(strings.NewReader("streamed-body")), objectstorage.ObjectInfo{
		Size:        int64(len("streamed-body")),
		ContentType: "application/pdf",
	}, nil
}

func (s *handlerFakeStorage) DeleteObject(ctx context.Context, key string) error {
	return nil
}

func TestServeMedia_StreamsFromObjectStorage(t *testing.T) {
	app := newTestApp(t)
	testutil.TruncateTables(app.DB)

	storage := &handlerFakeStorage{}
	app.ObjectStorage = storage

	org := testutil.CreateTestOrganization(t, app.DB)
	user := createUserWithPermissionKeys(t, app, org.ID, "chat-reader", []string{"chat:read"})
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	asset := models.MediaAsset{
		BaseModel: models.BaseModel{ID: uuid.New()},
		FileHash:  "feedface",
		S3Key:     "whatsmeow/media/fe/ed/feedface",
		MimeType:  "application/pdf",
		Size:      13,
	}
	require.NoError(t, app.DB.Create(&asset).Error)

	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: "whatsmeow",
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         "Document",
		MediaAssetID:    &asset.ID,
		MediaURL:        "/api/media/" + uuid.NewString(),
		MediaMimeType:   "application/pdf",
		MediaFilename:   "report.pdf",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(&message).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "message_id", message.ID.String())

	err := app.ServeMedia(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	assert.Equal(t, "application/pdf", string(req.RequestCtx.Response.Header.ContentType()))
	assert.Equal(t, 1, storage.getCalls)
}
