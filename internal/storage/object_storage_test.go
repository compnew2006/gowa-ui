package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectStorageResolveS3Endpoint(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.StorageConfig
		wantHost      string
		wantSecure    bool
		wantErr       bool
	}{
		{
			name:       "nil config",
			cfg:        nil,
			wantErr:    true,
		},
		{
			name:       "no endpoint no region defaults to s3.amazonaws.com",
			cfg:        &config.StorageConfig{},
			wantHost:   "s3.amazonaws.com",
			wantSecure: true,
		},
		{
			name: "region without endpoint",
			cfg:  &config.StorageConfig{S3Region: "eu-west-1"},
			wantHost: "s3.eu-west-1.amazonaws.com",
			wantSecure: true,
		},
		{
			name: "plain endpoint without scheme uses S3UseSSL",
			cfg:  &config.StorageConfig{S3Endpoint: "localhost:9000", S3UseSSL: false},
			wantHost: "localhost:9000",
			wantSecure: false,
		},
		{
			name: "https endpoint",
			cfg:  &config.StorageConfig{S3Endpoint: "https://s3.example.com"},
			wantHost: "s3.example.com",
			wantSecure: true,
		},
		{
			name: "http endpoint",
			cfg:  &config.StorageConfig{S3Endpoint: "http://minio.local:9000"},
			wantHost: "minio.local:9000",
			wantSecure: false,
		},
		{
			name: "endpoint with trailing whitespace",
			cfg:  &config.StorageConfig{S3Endpoint: "  https://s3.example.com  "},
			wantHost: "s3.example.com",
			wantSecure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, secure, err := resolveS3Endpoint(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, host)
			assert.Equal(t, tt.wantSecure, secure)
		})
	}
}

func TestObjectStorageNew(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		store, err := NewObjectStorage(nil)
		assert.NoError(t, err)
		assert.Nil(t, store)
	})

	t.Run("unsupported type returns error", func(t *testing.T) {
		store, err := NewObjectStorage(&config.StorageConfig{Type: "gcs"})
		assert.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "unsupported storage.type")
	})

	t.Run("empty type defaults to local", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewObjectStorage(&config.StorageConfig{Type: "", LocalPath: tmpDir})
		require.NoError(t, err)
		assert.NotNil(t, store)
	})

	t.Run("s3 type requires bucket", func(t *testing.T) {
		store, err := NewObjectStorage(&config.StorageConfig{Type: "s3"})
		assert.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "s3_bucket is required")
	})
}

func TestObjectStorageFileSystemPutGetDelete(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := newFileSystemObjectStorage(&config.StorageConfig{LocalPath: tmpDir})
	require.NoError(t, err)
	require.NotNil(t, store)

	ctx := context.Background()
	content := "hello world"

	t.Run("put then get", func(t *testing.T) {
		err := store.PutObject(ctx, "subdir/test.txt", strings.NewReader(content), int64(len(content)), "text/plain")
		require.NoError(t, err)

		reader, info, err := store.GetObject(ctx, "subdir/test.txt")
		require.NoError(t, err)
		defer reader.Close()
		assert.Equal(t, int64(len(content)), info.Size)

		buf := make([]byte, len(content))
		n, err := reader.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, content, string(buf[:n]))
	})

	t.Run("get nonexistent returns ErrObjectNotFound", func(t *testing.T) {
		_, _, err := store.GetObject(ctx, "nope.txt")
		assert.ErrorIs(t, err, ErrObjectNotFound)
	})

	t.Run("delete existing", func(t *testing.T) {
		err := store.DeleteObject(ctx, "subdir/test.txt")
		require.NoError(t, err)

		_, _, err = store.GetObject(ctx, "subdir/test.txt")
		assert.ErrorIs(t, err, ErrObjectNotFound)
	})

	t.Run("delete nonexistent is no-op", func(t *testing.T) {
		err := store.DeleteObject(ctx, "nope.txt")
		assert.NoError(t, err)
	})
}

func TestObjectStorageFileSystemSafePath(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := newFileSystemObjectStorage(&config.StorageConfig{LocalPath: tmpDir})
	require.NoError(t, err)

	t.Run("path traversal rejected", func(t *testing.T) {
		_, _, err := store.GetObject(context.Background(), "../etc/passwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "escapes storage root")
	})

	t.Run("absolute path key rejected", func(t *testing.T) {
		_, _, err := store.GetObject(context.Background(), "/etc/passwd")
		assert.Error(t, err)
	})
}
