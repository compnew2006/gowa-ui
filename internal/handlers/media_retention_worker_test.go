package handlers

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	objectstorage "github.com/compnew2006/whatomate/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type retentionFakeStorage struct {
	deletedKeys []string
}

func (s *retentionFakeStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	_, err := io.Copy(io.Discard, body)
	return err
}

func (s *retentionFakeStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, objectstorage.ObjectInfo, error) {
	return nil, objectstorage.ObjectInfo{}, objectstorage.ErrObjectNotFound
}

func (s *retentionFakeStorage) DeleteObject(ctx context.Context, key string) error {
	s.deletedKeys = append(s.deletedKeys, key)
	return nil
}

func newMediaRetentionTestApp(t *testing.T) (*App, *retentionFakeStorage) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "media-retention.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE organizations (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			settings BLOB
		)
	`).Error)
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
	require.NoError(t, db.Exec(`
		CREATE TABLE messages (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL,
			contact_id TEXT,
			instance_id TEXT,
			whats_app_account TEXT,
			direction TEXT,
			message_type TEXT,
			content TEXT,
			media_asset_id TEXT,
			media_url TEXT,
			media_mime_type TEXT,
			media_filename TEXT,
			media_deleted_at DATETIME,
			status TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)

	storage := &retentionFakeStorage{}
	app := &App{
		Config: &config.Config{
			WhatsApp: config.WhatsAppConfig{Provider: "whatsmeow"},
		},
		DB:            db,
		ObjectStorage: storage,
		Log:           logf.New(logf.Opts{}),
	}
	return app, storage
}

func createRetentionFixture(t *testing.T, app *App, orgSettings models.JSONB) (models.Organization, uuid.UUID, models.MediaAsset) {
	t.Helper()

	org := models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Retention Org",
		Slug:      "retention-" + uuid.NewString(),
		Settings:  orgSettings,
	}
	require.NoError(t, app.DB.Create(&org).Error)

	asset := models.MediaAsset{
		BaseModel: models.BaseModel{ID: uuid.New()},
		FileHash:  "abcd1234",
		S3Key:     "whatsmeow/media/ab/cd/abcd1234",
		MimeType:  "application/pdf",
		Size:      42,
	}
	require.NoError(t, app.DB.Create(&asset).Error)

	return org, uuid.New(), asset
}

func createRetentionMessage(
	t *testing.T,
	app *App,
	org models.Organization,
	contactID uuid.UUID,
	asset models.MediaAsset,
	createdAt time.Time,
	content string,
) models.Message {
	t.Helper()

	instanceID := uuid.New()
	message := models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: createdAt, UpdatedAt: createdAt},
		OrganizationID:  org.ID,
		InstanceID:      &instanceID,
		ContactID:       contactID,
		WhatsAppAccount: "whatsmeow",
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeDocument,
		Content:         content,
		MediaAssetID:    &asset.ID,
		MediaURL:        "/api/media/" + uuid.NewString(),
		MediaMimeType:   asset.MimeType,
		MediaFilename:   "report.pdf",
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Exec(`
		INSERT INTO messages (
			id, organization_id, contact_id, instance_id, whats_app_account, direction, message_type,
			content, media_asset_id, media_url, media_mime_type, media_filename, media_deleted_at,
			status, created_at, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		message.ID,
		message.OrganizationID,
		message.ContactID,
		instanceID,
		message.WhatsAppAccount,
		message.Direction,
		message.MessageType,
		message.Content,
		asset.ID,
		message.MediaURL,
		message.MediaMimeType,
		message.MediaFilename,
		nil,
		message.Status,
		message.CreatedAt,
		message.UpdatedAt,
		nil,
	).Error)
	return message
}

func TestMediaRetentionWorker_SweepExpiredMedia_PartialExpiryKeepsSharedAsset(t *testing.T) {
	t.Parallel()

	app, storage := newMediaRetentionTestApp(t)
	worker := NewMediaRetentionWorker(app, 24*time.Hour)
	org, contactID, asset := createRetentionFixture(t, app, models.JSONB{"media_retention_tier": "free"})
	now := time.Now().UTC()

	expired := createRetentionMessage(t, app, org, contactID, asset, now.Add(-40*24*time.Hour), "old message")
	active := createRetentionMessage(t, app, org, contactID, asset, now.Add(-5*24*time.Hour), "new message")

	require.NoError(t, worker.sweepExpiredMedia(context.Background(), now))

	var expiredMessage models.Message
	require.NoError(t, app.DB.Select("id", "content", "media_asset_id", "media_deleted_at").First(&expiredMessage, "id = ?", expired.ID).Error)
	require.NotNil(t, expiredMessage.MediaDeletedAt)
	assert.Contains(t, expiredMessage.Content, mediaRetentionDeletedNote)
	require.NotNil(t, expiredMessage.MediaAssetID)
	assert.Equal(t, asset.ID, *expiredMessage.MediaAssetID)

	var activeMessage models.Message
	require.NoError(t, app.DB.Select("id", "content", "media_asset_id", "media_deleted_at").First(&activeMessage, "id = ?", active.ID).Error)
	assert.Nil(t, activeMessage.MediaDeletedAt)
	require.NotNil(t, activeMessage.MediaAssetID)
	assert.Equal(t, asset.ID, *activeMessage.MediaAssetID)

	var storedAsset models.MediaAsset
	require.NoError(t, app.DB.First(&storedAsset, "id = ?", asset.ID).Error)
	assert.Empty(t, storage.deletedKeys)
}

func TestMediaRetentionWorker_SweepExpiredMedia_FullExpiryDeletesAssetOnceAndNullsRefs(t *testing.T) {
	t.Parallel()

	app, storage := newMediaRetentionTestApp(t)
	worker := NewMediaRetentionWorker(app, 24*time.Hour)
	org, contactID, asset := createRetentionFixture(t, app, models.JSONB{"media_retention_tier": "free"})
	now := time.Now().UTC()

	first := createRetentionMessage(t, app, org, contactID, asset, now.Add(-45*24*time.Hour), "first old")
	second := createRetentionMessage(t, app, org, contactID, asset, now.Add(-35*24*time.Hour), "second old")

	require.NoError(t, worker.sweepExpiredMedia(context.Background(), now))
	require.NoError(t, worker.sweepExpiredMedia(context.Background(), now.Add(time.Hour)))

	var firstMessage models.Message
	require.NoError(t, app.DB.Select("id", "content", "media_asset_id", "media_deleted_at").First(&firstMessage, "id = ?", first.ID).Error)
	require.NotNil(t, firstMessage.MediaDeletedAt)
	assert.Nil(t, firstMessage.MediaAssetID)
	assert.Equal(t, 1, countRetentionNote(firstMessage.Content))

	var secondMessage models.Message
	require.NoError(t, app.DB.Select("id", "content", "media_asset_id", "media_deleted_at").First(&secondMessage, "id = ?", second.ID).Error)
	require.NotNil(t, secondMessage.MediaDeletedAt)
	assert.Nil(t, secondMessage.MediaAssetID)
	assert.Equal(t, 1, countRetentionNote(secondMessage.Content))

	var assetCount int64
	require.NoError(t, app.DB.Model(&models.MediaAsset{}).Where("id = ?", asset.ID).Count(&assetCount).Error)
	assert.Equal(t, int64(0), assetCount)

	var deletedAsset models.MediaAsset
	require.NoError(t, app.DB.Unscoped().Where("id = ?", asset.ID).First(&deletedAsset).Error)
	assert.True(t, deletedAsset.DeletedAt.Valid)
	require.Len(t, storage.deletedKeys, 1)
	assert.Equal(t, asset.S3Key, storage.deletedKeys[0])
}

func countRetentionNote(content string) int {
	return strings.Count(content, mediaRetentionDeletedNote)
}
