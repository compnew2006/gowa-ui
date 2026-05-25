package whatsmeow

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	objectstorage "github.com/compnew2006/whatomate/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
	waClient "go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeObjectStorage struct {
	putFunc    func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error
	getFunc    func(ctx context.Context, key string) (io.ReadCloser, objectstorage.ObjectInfo, error)
	deleteFunc func(ctx context.Context, key string) error
}

func (f *fakeObjectStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	if f.putFunc != nil {
		return f.putFunc(ctx, key, body, size, mimeType)
	}
	_, err := io.Copy(io.Discard, body)
	return err
}

func (f *fakeObjectStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, objectstorage.ObjectInfo, error) {
	if f.getFunc != nil {
		return f.getFunc(ctx, key)
	}
	return nil, objectstorage.ObjectInfo{}, objectstorage.ErrObjectNotFound
}

func (f *fakeObjectStorage) DeleteObject(ctx context.Context, key string) error {
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, key)
	}
	return nil
}

func newMediaServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "media-service.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE media_assets (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			file_hash TEXT NOT NULL UNIQUE,
			s3_key TEXT NOT NULL,
			mime_type TEXT NOT NULL,
			size INTEGER NOT NULL
		)
	`).Error)
	return db
}

func newMediaServiceTestEvent(fileHash []byte, fileLength uint64) *events.Message {
	return &events.Message{
		Message: &waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				Mimetype:      proto.String("application/pdf"),
				FileName:      proto.String("report.pdf"),
				FileSHA256:    fileHash,
				FileLength:    proto.Uint64(fileLength),
				FileEncSHA256: []byte("encrypted-sha-placeholder"),
				MediaKey:      []byte("media-key-placeholder"),
				DirectPath:    proto.String("/mms/document/test"),
			},
		},
	}
}

func TestMediaService_HandleIncomingMedia_DedupHitSkipsUpload(t *testing.T) {
	t.Parallel()

	db := newMediaServiceTestDB(t)
	fileHash := stringsRepeatByte(0x11, 32)
	hashHex, err := nativeMediaFileHash(newMediaServiceTestEvent(fileHash, 5).Message.GetDocumentMessage())
	require.NoError(t, err)

	existingAsset := models.MediaAsset{
		BaseModel: models.BaseModel{ID: uuid.New()},
		FileHash:  hashHex,
		S3Key:     buildMediaObjectKey(hashHex),
		MimeType:  "application/pdf",
		Size:      5,
	}
	require.NoError(t, db.Create(&existingAsset).Error)

	var streamCalls atomic.Int32
	var uploadCalls atomic.Int32
	service := NewMediaService(db, &fakeObjectStorage{
		getFunc: func(ctx context.Context, key string) (io.ReadCloser, objectstorage.ObjectInfo, error) {
			return io.NopCloser(bytes.NewReader([]byte("stored"))), objectstorage.ObjectInfo{
				Size:        existingAsset.Size,
				ContentType: existingAsset.MimeType,
			}, nil
		},
		putFunc: func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
			uploadCalls.Add(1)
			return nil
		},
	}, logf.New(logf.Opts{}), func(uuid.UUID) *waClient.Client { return &waClient.Client{} })
	service.streamDownload = func(ctx context.Context, client *waClient.Client, media waClient.DownloadableMessage, dst io.Writer) (int64, error) {
		streamCalls.Add(1)
		return 0, nil
	}

	result, err := service.HandleIncomingMedia(WithMediaInstanceID(context.Background(), uuid.New()), newMediaServiceTestEvent(fileHash, 5))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.WasDedupHit)
	assert.Equal(t, existingAsset.ID, result.MediaAssetID)
	assert.Equal(t, int32(0), streamCalls.Load())
	assert.Equal(t, int32(0), uploadCalls.Load())
}

func TestMediaService_HandleIncomingMedia_RestoresMissingDedupAsset(t *testing.T) {
	t.Parallel()

	db := newMediaServiceTestDB(t)
	fileHash := stringsRepeatByte(0x12, 32)
	payload := []byte("restored-pdf")
	hashHex, err := nativeMediaFileHash(newMediaServiceTestEvent(fileHash, uint64(len(payload))).Message.GetDocumentMessage())
	require.NoError(t, err)

	existingAsset := models.MediaAsset{
		BaseModel: models.BaseModel{ID: uuid.New()},
		FileHash:  hashHex,
		S3Key:     buildMediaObjectKey(hashHex),
		MimeType:  "application/pdf",
		Size:      99,
	}
	require.NoError(t, db.Create(&existingAsset).Error)

	var uploadedKey string
	var uploadedData []byte
	var streamCalls atomic.Int32
	var uploadCalls atomic.Int32
	service := NewMediaService(db, &fakeObjectStorage{
		getFunc: func(ctx context.Context, key string) (io.ReadCloser, objectstorage.ObjectInfo, error) {
			return nil, objectstorage.ObjectInfo{}, objectstorage.ErrObjectNotFound
		},
		putFunc: func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
			uploadCalls.Add(1)
			uploadedKey = key
			var err error
			uploadedData, err = io.ReadAll(body)
			return err
		},
	}, logf.New(logf.Opts{}), func(uuid.UUID) *waClient.Client { return &waClient.Client{} })
	service.streamDownload = func(ctx context.Context, client *waClient.Client, media waClient.DownloadableMessage, dst io.Writer) (int64, error) {
		streamCalls.Add(1)
		n, err := dst.Write(payload)
		return int64(n), err
	}

	result, err := service.HandleIncomingMedia(WithMediaInstanceID(context.Background(), uuid.New()), newMediaServiceTestEvent(fileHash, uint64(len(payload))))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.WasDedupHit)
	assert.Equal(t, existingAsset.ID, result.MediaAssetID)
	assert.Equal(t, int32(1), streamCalls.Load())
	assert.Equal(t, int32(1), uploadCalls.Load())
	assert.Equal(t, buildMediaObjectKey(hashHex), uploadedKey)
	assert.Equal(t, payload, uploadedData)

	var refreshed models.MediaAsset
	require.NoError(t, db.First(&refreshed, "id = ?", existingAsset.ID).Error)
	assert.Equal(t, int64(len(payload)), refreshed.Size)
	assert.Equal(t, buildMediaObjectKey(hashHex), refreshed.S3Key)
}

func TestMediaService_HandleIncomingMedia_StoresNewAssetWithDeterministicKey(t *testing.T) {
	t.Parallel()

	db := newMediaServiceTestDB(t)
	fileHash := stringsRepeatByte(0x22, 32)
	payload := []byte("hello")

	var uploadedKey string
	var uploadedMIME string
	var uploadedData []byte
	service := NewMediaService(db, &fakeObjectStorage{
		putFunc: func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
			uploadedKey = key
			uploadedMIME = mimeType
			data, err := io.ReadAll(body)
			if err != nil {
				return err
			}
			uploadedData = data
			assert.Equal(t, int64(len(payload)), size)
			return nil
		},
	}, logf.New(logf.Opts{}), func(uuid.UUID) *waClient.Client { return &waClient.Client{} })
	service.streamDownload = func(ctx context.Context, client *waClient.Client, media waClient.DownloadableMessage, dst io.Writer) (int64, error) {
		n, err := dst.Write(payload)
		return int64(n), err
	}

	result, err := service.HandleIncomingMedia(WithMediaInstanceID(context.Background(), uuid.New()), newMediaServiceTestEvent(fileHash, uint64(len(payload))))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.WasDedupHit)
	assert.Equal(t, int64(len(payload)), result.Size)
	assert.Equal(t, payload, uploadedData)

	hashHex, err := nativeMediaFileHash(newMediaServiceTestEvent(fileHash, uint64(len(payload))).Message.GetDocumentMessage())
	require.NoError(t, err)
	assert.Equal(t, buildMediaObjectKey(hashHex), uploadedKey)
	assert.Equal(t, "application/pdf", uploadedMIME)

	var stored models.MediaAsset
	require.NoError(t, db.Where("file_hash = ?", hashHex).First(&stored).Error)
	assert.Equal(t, uploadedKey, stored.S3Key)
	assert.Equal(t, int64(len(payload)), stored.Size)
	assert.Equal(t, result.MediaAssetID, stored.ID)
}

func TestMediaService_HandleIncomingMedia_StreamFailureClosesPipe(t *testing.T) {
	t.Parallel()

	db := newMediaServiceTestDB(t)
	fileHash := stringsRepeatByte(0x33, 32)
	expectedErr := errors.New("download failed")

	service := NewMediaService(db, &fakeObjectStorage{
		putFunc: func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
			_, err := io.Copy(io.Discard, body)
			return err
		},
	}, logf.New(logf.Opts{}), func(uuid.UUID) *waClient.Client { return &waClient.Client{} })
	service.streamDownload = func(ctx context.Context, client *waClient.Client, media waClient.DownloadableMessage, dst io.Writer) (int64, error) {
		return 0, expectedErr
	}

	_, err := service.HandleIncomingMedia(WithMediaInstanceID(context.Background(), uuid.New()), newMediaServiceTestEvent(fileHash, 5))
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestMediaService_HandleIncomingMedia_UploadFailureCancelsStreamer(t *testing.T) {
	t.Parallel()

	db := newMediaServiceTestDB(t)
	fileHash := stringsRepeatByte(0x44, 32)
	expectedErr := errors.New("s3 unavailable")
	streamExited := make(chan struct{})

	service := NewMediaService(db, &fakeObjectStorage{
		putFunc: func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
			return expectedErr
		},
	}, logf.New(logf.Opts{}), func(uuid.UUID) *waClient.Client { return &waClient.Client{} })
	service.streamDownload = func(ctx context.Context, client *waClient.Client, media waClient.DownloadableMessage, dst io.Writer) (int64, error) {
		defer close(streamExited)
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(2 * time.Second):
			return 0, errors.New("streamer was not cancelled")
		}
	}

	_, err := service.HandleIncomingMedia(WithMediaInstanceID(context.Background(), uuid.New()), newMediaServiceTestEvent(fileHash, 5))
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)

	select {
	case <-streamExited:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("streamer goroutine did not exit after upload failure")
	}
}

func TestMediaService_HandleIncomingMedia_ConcurrentFirstWriteSharesOneAsset(t *testing.T) {
	t.Parallel()

	db := newMediaServiceTestDB(t)
	fileHash := stringsRepeatByte(0x55, 32)
	payload := []byte("shared-payload")
	service := NewMediaService(db, &fakeObjectStorage{
		putFunc: func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
			_, err := io.Copy(io.Discard, body)
			return err
		},
	}, logf.New(logf.Opts{}), func(uuid.UUID) *waClient.Client { return &waClient.Client{} })
	service.streamDownload = func(ctx context.Context, client *waClient.Client, media waClient.DownloadableMessage, dst io.Writer) (int64, error) {
		n, err := dst.Write(payload)
		return int64(n), err
	}

	event := newMediaServiceTestEvent(fileHash, uint64(len(payload)))
	results := make([]*HandledMedia, 2)
	errs := make([]error, 2)

	var wg sync.WaitGroup
	for idx := 0; idx < 2; idx++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = service.HandleIncomingMedia(WithMediaInstanceID(context.Background(), uuid.New()), event)
		}(idx)
	}
	wg.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	require.NotNil(t, results[0])
	require.NotNil(t, results[1])
	assert.Equal(t, results[0].MediaAssetID, results[1].MediaAssetID)

	hashHex, err := nativeMediaFileHash(event.Message.GetDocumentMessage())
	require.NoError(t, err)
	var count int64
	require.NoError(t, db.Model(&models.MediaAsset{}).Where("file_hash = ?", hashHex).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func stringsRepeatByte(value byte, count int) []byte {
	buf := make([]byte, count)
	for i := range buf {
		buf[i] = value
	}
	return buf
}
