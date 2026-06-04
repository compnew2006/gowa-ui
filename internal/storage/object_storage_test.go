package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
)

type mockStorage struct {
	putFunc func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error
	calls   atomic.Int32
}

func (m *mockStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	m.calls.Add(1)
	if m.putFunc != nil {
		return m.putFunc(ctx, key, body, size, mimeType)
	}
	return nil
}

func (m *mockStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	return nil, ObjectInfo{}, nil
}

func (m *mockStorage) DeleteObject(ctx context.Context, key string) error {
	return nil
}

type nonSeekableReader struct {
	r io.Reader
}

func (n *nonSeekableReader) Read(p []byte) (int, error) {
	return n.r.Read(p)
}

func TestRetryableObjectStorage_PutObject_SuccessFirstAttempt(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:    mock,
		maxRetry: 3,
	}

	body := strings.NewReader("hello-world")
	err := retryStorage.PutObject(context.Background(), "test-key", body, int64(body.Len()), "text/plain")
	assert.NoError(t, err)
	assert.Equal(t, int32(1), mock.calls.Load())
}

func TestRetryableObjectStorage_PutObject_SuccessOnRetry(t *testing.T) {
	mock := &mockStorage{}
	// Fail the first 2 attempts with a transient error, succeed on the 3rd
	mock.putFunc = func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
		data, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		if string(data) != "hello-world" {
			return errors.New("unexpected body content")
		}

		if mock.calls.Load() < 3 {
			// Return a transient string error
			return errors.New("EOF during connection")
		}
		return nil
	}

	retryStorage := &retryableObjectStorage{
		inner:    mock,
		maxRetry: 3,
	}

	body := strings.NewReader("hello-world")
	err := retryStorage.PutObject(context.Background(), "test-key", body, int64(body.Len()), "text/plain")
	assert.NoError(t, err)
	assert.Equal(t, int32(3), mock.calls.Load())
}

func TestRetryableObjectStorage_PutObject_FailureMaxRetriesExhausted(t *testing.T) {
	mock := &mockStorage{}
	// Fail all attempts with transient error
	mock.putFunc = func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
		return errors.New("connection reset by peer")
	}

	retryStorage := &retryableObjectStorage{
		inner:    mock,
		maxRetry: 2, // 3 attempts total (0, 1, 2)
	}

	body := strings.NewReader("hello-world")
	err := retryStorage.PutObject(context.Background(), "test-key", body, int64(body.Len()), "text/plain")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PutObject failed after 3 attempts")
	assert.Equal(t, int32(3), mock.calls.Load())
}

func TestRetryableObjectStorage_NonSeekable_SizeZero(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:    mock,
		maxRetry: 3,
	}

	content := "hello-world-zero"
	body := &nonSeekableReader{r: strings.NewReader(content)}

	mock.putFunc = func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
		data, err := io.ReadAll(body)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
		assert.Equal(t, int64(len(content)), size)
		return nil
	}

	err := retryStorage.PutObject(context.Background(), "test-key", body, 0, "text/plain")
	assert.NoError(t, err)
	assert.Equal(t, int32(1), mock.calls.Load())
}

func TestRetryableObjectStorage_NonSeekable_SizeMinusOne(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:    mock,
		maxRetry: 3,
	}

	content := "hello-world-minus-one"
	body := &nonSeekableReader{r: strings.NewReader(content)}

	mock.putFunc = func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
		data, err := io.ReadAll(body)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
		assert.Equal(t, int64(len(content)), size)
		return nil
	}

	err := retryStorage.PutObject(context.Background(), "test-key", body, -1, "text/plain")
	assert.NoError(t, err)
	assert.Equal(t, int32(1), mock.calls.Load())
}

func TestRetryableObjectStorage_ShortBody_MemoryBuffer(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:    mock,
		maxRetry: 3,
	}

	body := &nonSeekableReader{r: strings.NewReader("short")}
	err := retryStorage.PutObject(context.Background(), "test-key", body, 100, "text/plain")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read body to memory")
	assert.Equal(t, int32(0), mock.calls.Load())
}

func TestRetryableObjectStorage_ShortBody_TempFile(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:    mock,
		maxRetry: 3,
	}

	body := &nonSeekableReader{r: strings.NewReader("short")}
	// Use size larger than 10MB to trigger temp file path
	err := retryStorage.PutObject(context.Background(), "test-key", body, 11*1024*1024, "text/plain")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "body size mismatch")
	assert.Equal(t, int32(0), mock.calls.Load())
}

func TestRetryableObjectStorage_PermanentError(t *testing.T) {
	mock := &mockStorage{}
	permanentErr := minio.ErrorResponse{
		StatusCode: 403,
		Code:       "AccessDenied",
		Message:    "Access Denied",
	}

	mock.putFunc = func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
		return permanentErr
	}

	retryStorage := &retryableObjectStorage{
		inner:    mock,
		maxRetry: 3,
	}

	body := strings.NewReader("hello-world")
	err := retryStorage.PutObject(context.Background(), "test-key", body, int64(body.Len()), "text/plain")
	assert.Error(t, err)
	assert.Equal(t, permanentErr, err)
	assert.Equal(t, int32(1), mock.calls.Load())
}

func TestRetryableObjectStorage_TransientError_Succeeds(t *testing.T) {
	mock := &mockStorage{}
	transientErr := minio.ErrorResponse{
		StatusCode: 503,
		Code:       "ServiceUnavailable",
		Message:    "Slow down",
	}

	mock.putFunc = func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
		if mock.calls.Load() < 2 {
			return transientErr
		}
		return nil
	}

	retryStorage := &retryableObjectStorage{
		inner:    mock,
		maxRetry: 1, // Only 1 retry (2 attempts total)
	}

	body := strings.NewReader("hello")
	err := retryStorage.PutObject(context.Background(), "test-key", body, int64(body.Len()), "text/plain")
	assert.NoError(t, err)
	assert.Equal(t, int32(2), mock.calls.Load())
}

func TestIsTransientError(t *testing.T) {
	// Standard nil error
	assert.False(t, isTransientError(nil))

	// Context errors
	assert.False(t, isTransientError(context.Canceled))
	assert.False(t, isTransientError(context.DeadlineExceeded))

	// Non-transient MinIO S3 status codes
	assert.False(t, isTransientError(minio.ErrorResponse{StatusCode: 403, Code: "AccessDenied"}))
	assert.False(t, isTransientError(minio.ErrorResponse{StatusCode: 404, Code: "NoSuchBucket"}))
	assert.False(t, isTransientError(minio.ErrorResponse{StatusCode: 400, Code: "BadRequest"}))

	// Transient MinIO S3 status codes / codes
	assert.True(t, isTransientError(minio.ErrorResponse{StatusCode: 429, Code: "SlowDown"}))
	assert.True(t, isTransientError(minio.ErrorResponse{StatusCode: 503, Code: "ServiceUnavailable"}))
	assert.True(t, isTransientError(minio.ErrorResponse{StatusCode: 500, Code: "InternalError"}))
	assert.True(t, isTransientError(minio.ErrorResponse{Code: "RequestTimeout"}))

	// Fallback temporary network errors
	assert.True(t, isTransientError(errors.New("connection reset by peer")))
	assert.True(t, isTransientError(errors.New("broken pipe")))
	assert.True(t, isTransientError(errors.New("EOF during read")))
	assert.True(t, isTransientError(errors.New("connection timeout exceeded")))

	// Arbitrary custom errors are not transient
	assert.False(t, isTransientError(errors.New("some custom db error")))
}

func TestRetryableObjectStorage_PartiallyReadSeeker_RetryOffset(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:     mock,
		maxRetry:  1,
		baseDelay: 0, // Instant retry
	}

	content := "unrelated-prefix-header-actual-content"
	prefixLen := len("unrelated-prefix-header-")
	body := strings.NewReader(content)

	// Read the prefix first so it is "partially read"
	_, err := body.Seek(int64(prefixLen), io.SeekStart)
	assert.NoError(t, err)

	expectedUploadContent := content[prefixLen:]
	expectedSize := int64(len(expectedUploadContent))

	mock.putFunc = func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
		data, err := io.ReadAll(body)
		assert.NoError(t, err)
		assert.Equal(t, expectedUploadContent, string(data))
		assert.Equal(t, expectedSize, size)

		if mock.calls.Load() < 2 {
			return minio.ErrorResponse{
				StatusCode: 503,
				Code:       "ServiceUnavailable",
			}
		}
		return nil
	}

	err = retryStorage.PutObject(context.Background(), "test-key", body, expectedSize, "text/plain")
	assert.NoError(t, err)
	assert.Equal(t, int32(2), mock.calls.Load())
}
