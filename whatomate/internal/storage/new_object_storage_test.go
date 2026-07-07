package storage

import (
	"os"
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewObjectStorage(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		store, err := NewObjectStorage(nil)
		assert.NoError(t, err)
		assert.Nil(t, store)
	})

	t.Run("local type (explicit)", func(t *testing.T) {
		tempDir := t.TempDir()
		cfg := &config.StorageConfig{
			Type:      "local",
			LocalPath: tempDir,
		}
		store, err := NewObjectStorage(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, store)
		assert.IsType(t, &retryableObjectStorage{}, store)

		retryableStore := store.(*retryableObjectStorage)
		assert.IsType(t, &fileSystemObjectStorage{}, retryableStore.inner)

		fsStore := retryableStore.inner.(*fileSystemObjectStorage)
		assert.Equal(t, tempDir, fsStore.rootPath)
	})

	t.Run("empty type defaults to local", func(t *testing.T) {
		tempDir := t.TempDir()
		cfg := &config.StorageConfig{
			Type:      "",
			LocalPath: tempDir,
		}
		store, err := NewObjectStorage(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, store)

		retryableStore := store.(*retryableObjectStorage)
		assert.IsType(t, &fileSystemObjectStorage{}, retryableStore.inner)
	})

	t.Run("local path defaults to ./uploads", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Type:      "local",
			LocalPath: "",
		}
		store, err := NewObjectStorage(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, store)

		defer os.RemoveAll("./uploads") // cleanup

		retryableStore := store.(*retryableObjectStorage)
		fsStore := retryableStore.inner.(*fileSystemObjectStorage)
		assert.Equal(t, "./uploads", fsStore.rootPath)
	})

	t.Run("unsupported type", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Type: "unknown-type",
		}
		store, err := NewObjectStorage(cfg)
		assert.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "unsupported storage.type")
	})

	t.Run("s3 type valid", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Type:       "s3",
			S3Bucket:   "my-bucket",
			S3Endpoint: "s3.amazonaws.com",
			S3Region:   "us-east-1",
		}
		store, err := NewObjectStorage(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, store)

		retryableStore := store.(*retryableObjectStorage)
		assert.IsType(t, &minIOObjectStorage{}, retryableStore.inner)
	})

	t.Run("s3 type missing bucket", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Type: "s3",
		}
		store, err := NewObjectStorage(cfg)
		assert.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "storage.s3_bucket is required")
	})

	t.Run("s3 type invalid endpoint", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Type: "s3",
			S3Bucket: "bucket",
			S3Endpoint: "http://[::1]:namedport",
		}
		store, err := NewObjectStorage(cfg)
		assert.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "invalid port")
	})
}
