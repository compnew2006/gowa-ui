package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ErrObjectNotFound is returned when the requested object does not exist.
var ErrObjectNotFound = errors.New("object not found")

// ObjectInfo describes a stored object returned by GetObject.
type ObjectInfo struct {
	Size        int64
	ContentType string
	ETag        string
}

// ObjectStorage abstracts the inbound-media object store.
type ObjectStorage interface {
	PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error
	GetObject(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)
	DeleteObject(ctx context.Context, key string) error
}

// NewObjectStorage creates a concrete object storage implementation for the current configuration.
func NewObjectStorage(cfg *config.StorageConfig) (ObjectStorage, error) {
	if cfg == nil {
		return nil, nil
	}

	var store ObjectStorage
	var err error
	switch strings.ToLower(strings.TrimSpace(cfg.Type)) {
	case "", "local":
		store, err = newFileSystemObjectStorage(cfg)
	case "s3":
		store, err = newMinIOObjectStorage(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage.type %q", cfg.Type)
	}
	if err != nil {
		return nil, err
	}

	return &retryableObjectStorage{
		inner:     store,
		maxRetry:  3,
		baseDelay: 500 * time.Millisecond,
	}, nil
}

type fileSystemObjectStorage struct {
	rootPath string
}

func newFileSystemObjectStorage(cfg *config.StorageConfig) (ObjectStorage, error) {
	root := strings.TrimSpace(cfg.LocalPath)
	if root == "" {
		root = "./uploads"
	}
	// Ensure root exists
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("create storage root %q: %w", root, err)
	}
	return &fileSystemObjectStorage{rootPath: root}, nil
}

func (s *fileSystemObjectStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	path := filepath.Join(s.rootPath, key)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	_, err = io.Copy(f, body)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func (s *fileSystemObjectStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	path := filepath.Join(s.rootPath, key)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ObjectInfo{}, ErrObjectNotFound
		}
		return nil, ObjectInfo{}, fmt.Errorf("open file: %w", err)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, ObjectInfo{}, fmt.Errorf("stat file: %w", err)
	}

	return f, ObjectInfo{
		Size:        stat.Size(),
		ContentType: "application/octet-stream", // Optional: improve mime detection if needed
	}, nil
}

func (s *fileSystemObjectStorage) DeleteObject(ctx context.Context, key string) error {
	path := filepath.Join(s.rootPath, key)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

type minIOObjectStorage struct {
	client *minio.Client
	bucket string
}

func newMinIOObjectStorage(cfg *config.StorageConfig) (ObjectStorage, error) {
	bucket := strings.TrimSpace(cfg.S3Bucket)
	if bucket == "" {
		return nil, fmt.Errorf("storage.s3_bucket is required when storage.type=s3")
	}

	endpoint, secure, err := resolveS3Endpoint(cfg)
	if err != nil {
		return nil, err
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(strings.TrimSpace(cfg.S3Key), strings.TrimSpace(cfg.S3Secret), ""),
		Secure: secure,
		Region: strings.TrimSpace(cfg.S3Region),
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &minIOObjectStorage{
		client: client,
		bucket: bucket,
	}, nil
}

func resolveS3Endpoint(cfg *config.StorageConfig) (string, bool, error) {
	if cfg == nil {
		return "", false, fmt.Errorf("storage config is nil")
	}

	rawEndpoint := strings.TrimSpace(cfg.S3Endpoint)
	if rawEndpoint != "" {
		if strings.Contains(rawEndpoint, "://") {
			parsed, err := url.Parse(rawEndpoint)
			if err != nil {
				return "", false, fmt.Errorf("invalid storage.s3_endpoint: %w", err)
			}
			if parsed.Host == "" {
				return "", false, fmt.Errorf("storage.s3_endpoint must include a host")
			}
			return parsed.Host, strings.EqualFold(parsed.Scheme, "https"), nil
		}
		return rawEndpoint, cfg.S3UseSSL, nil
	}

	region := strings.TrimSpace(cfg.S3Region)
	if region == "" {
		return "s3.amazonaws.com", true, nil
	}
	return fmt.Sprintf("s3.%s.amazonaws.com", region), true, nil
}

func (s *minIOObjectStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("object storage is not configured")
	}
	_, err := s.client.PutObject(ctx, s.bucket, strings.TrimSpace(key), body, size, minio.PutObjectOptions{
		ContentType: strings.TrimSpace(mimeType),
	})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

func (s *minIOObjectStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	if s == nil || s.client == nil {
		return nil, ObjectInfo{}, fmt.Errorf("object storage is not configured")
	}

	obj, err := s.client.GetObject(ctx, s.bucket, strings.TrimSpace(key), minio.GetObjectOptions{})
	if err != nil {
		if isObjectNotFound(err) {
			return nil, ObjectInfo{}, ErrObjectNotFound
		}
		return nil, ObjectInfo{}, fmt.Errorf("get object %q: %w", key, err)
	}

	info, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		if isObjectNotFound(err) {
			return nil, ObjectInfo{}, ErrObjectNotFound
		}
		return nil, ObjectInfo{}, fmt.Errorf("stat object %q: %w", key, err)
	}

	return obj, ObjectInfo{
		Size:        info.Size,
		ContentType: info.ContentType,
		ETag:        info.ETag,
	}, nil
}

func (s *minIOObjectStorage) DeleteObject(ctx context.Context, key string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("object storage is not configured")
	}

	err := s.client.RemoveObject(ctx, s.bucket, strings.TrimSpace(key), minio.RemoveObjectOptions{})
	if err != nil && !isObjectNotFound(err) {
		return fmt.Errorf("delete object %q: %w", key, err)
	}
	return nil
}

func isObjectNotFound(err error) bool {
	if err == nil {
		return false
	}
	resp := minio.ToErrorResponse(err)
	return resp.Code == "NoSuchKey" || resp.Code == "NoSuchObject" || resp.Code == "NoSuchBucket"
}

type retryableObjectStorage struct {
	inner     ObjectStorage
	maxRetry  int
	baseDelay time.Duration
}

func (r *retryableObjectStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	seekableBody, actualSize, startOffset, cleanup, err := toSeekableReader(body, size)
	if err != nil {
		return fmt.Errorf("buffer storage body: %w", err)
	}
	defer cleanup()

	var lastErr error
	baseDelay := r.baseDelay
	for attempt := 0; attempt <= r.maxRetry; attempt++ {
		if attempt > 0 {
			if baseDelay > 0 {
				// Exponential backoff: baseDelay * 2^(attempt-1)
				delay := baseDelay * time.Duration(1<<(attempt-1))
				var jitter time.Duration
				if delay > 2 {
					// #nosec G404 - math/rand is safe for retry backoff jitter
					jitter = time.Duration(rand.Int63n(int64(delay) / 2))
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay + jitter):
				}
			} else {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
			}

			// Seek back to startOffset for the next attempt
			if _, err := seekableBody.Seek(startOffset, io.SeekStart); err != nil {
				return fmt.Errorf("seek body for retry: %w", err)
			}
		}

		lastErr = r.inner.PutObject(ctx, key, seekableBody, actualSize, mimeType)
		if lastErr == nil {
			return nil
		}

		if !isTransientError(lastErr) {
			return lastErr
		}
	}
	return fmt.Errorf("PutObject failed after %d attempts: %w", r.maxRetry+1, lastErr)
}

func (r *retryableObjectStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	return r.inner.GetObject(ctx, key)
}

func (r *retryableObjectStorage) DeleteObject(ctx context.Context, key string) error {
	return r.inner.DeleteObject(ctx, key)
}

func toSeekableReader(body io.Reader, size int64) (io.ReadSeeker, int64, int64, func(), error) {
	if rs, ok := body.(io.ReadSeeker); ok {
		current, err := rs.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, 0, 0, func() {}, fmt.Errorf("seek current: %w", err)
		}
		if size <= 0 {
			end, err := rs.Seek(0, io.SeekEnd)
			if err != nil {
				return nil, 0, 0, func() {}, fmt.Errorf("seek end: %w", err)
			}
			_, err = rs.Seek(current, io.SeekStart)
			if err != nil {
				return nil, 0, 0, func() {}, fmt.Errorf("seek start: %w", err)
			}
			return rs, end - current, current, func() {}, nil
		}
		return rs, size, current, func() {}, nil
	}

	// Buffer small payloads (up to 10MB) in memory if size is positive
	const maxMemoryBuffer = 10 * 1024 * 1024
	if size > 0 && size <= maxMemoryBuffer {
		buf := make([]byte, size)
		_, err := io.ReadFull(body, buf)
		if err != nil {
			return nil, 0, 0, func() {}, fmt.Errorf("read body to memory: %w", err)
		}
		return bytes.NewReader(buf), size, 0, func() {}, nil
	}

	// Buffer larger or unknown-sized payloads in a temporary file
	tmpFile, err := os.CreateTemp("", "whatomate-storage-retry-*")
	if err != nil {
		return nil, 0, 0, func() {}, fmt.Errorf("create temp file: %w", err)
	}

	cleanup := func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}

	n, err := io.Copy(tmpFile, body)
	if err != nil {
		cleanup()
		return nil, 0, 0, func() {}, fmt.Errorf("copy body to temp file: %w", err)
	}

	if size > 0 && n != size {
		cleanup()
		return nil, 0, 0, func() {}, fmt.Errorf("body size mismatch: read %d bytes, expected %d", n, size)
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		cleanup()
		return nil, 0, 0, func() {}, fmt.Errorf("seek temp file: %w", err)
	}

	return tmpFile, n, 0, cleanup, nil
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// 1. Check if the error implements net.Error and is temporary or timeout.
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() || netErr.Temporary() {
			return true
		}
	}

	// 2. Check for MinIO S3 API error responses.
	resp := minio.ToErrorResponse(err)
	if resp.StatusCode > 0 {
		if resp.StatusCode == 429 {
			return true
		}
		if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			return true
		}
		switch resp.Code {
		case "SlowDown", "RequestTimeout", "InternalError", "ServiceUnavailable":
			return true
		}
		return false
	}

	// 3. Fallback string matching for common temporary network / connection issues.
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "eof") ||
		strings.Contains(errStr, "timeout") {
		return true
	}

	return false
}
