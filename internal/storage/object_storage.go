package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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

	switch strings.ToLower(strings.TrimSpace(cfg.Type)) {
	case "", "local":
		return newFileSystemObjectStorage(cfg)
	case "s3":
		return newMinIOObjectStorage(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage.type %q", cfg.Type)
	}
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

// safePath resolves key against root and enforces the resolved path stays
// within the storage root. Returns an error if the key would escape the root
// (path traversal) or if the final path cannot be resolved.
func (s *fileSystemObjectStorage) safePath(key string) (string, error) {
	cleanKey := filepath.Clean(filepath.FromSlash(key))
	if cleanKey == ".." || strings.HasPrefix(cleanKey, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("object key %q escapes storage root", key)
	}

	absRoot, err := filepath.Abs(s.rootPath)
	if err != nil {
		return "", fmt.Errorf("resolve storage root: %w", err)
	}
	resolvedRoot := absRoot
	if r, err := filepath.EvalSymlinks(absRoot); err == nil {
		resolvedRoot = r
	}

	fullPath, err := filepath.Abs(filepath.Join(resolvedRoot, cleanKey))
	if err != nil {
		return "", fmt.Errorf("resolve object path: %w", err)
	}

	if !strings.HasPrefix(fullPath, resolvedRoot+string(os.PathSeparator)) && fullPath != resolvedRoot {
		return "", fmt.Errorf("object key %q escapes storage root", key)
	}

	return fullPath, nil
}

func (s *fileSystemObjectStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	path, err := s.safePath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
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
	path, err := s.safePath(key)
	if err != nil {
		return nil, ObjectInfo{}, err
	}
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
		ContentType: "application/octet-stream",
	}, nil
}

func (s *fileSystemObjectStorage) DeleteObject(ctx context.Context, key string) error {
	path, err := s.safePath(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
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
