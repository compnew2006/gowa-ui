package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
)

type mockStorage struct {
	putFunc    func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error
	getFunc    func(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)
	deleteFunc func(ctx context.Context, key string) error
	putCalls   atomic.Int32
	getCalls   atomic.Int32
	delCalls   atomic.Int32
}

func (m *mockStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	m.putCalls.Add(1)
	if m.putFunc != nil {
		return m.putFunc(ctx, key, body, size, mimeType)
	}
	return nil
}

func (m *mockStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	m.getCalls.Add(1)
	if m.getFunc != nil {
		return m.getFunc(ctx, key)
	}
	return nil, ObjectInfo{}, nil
}

func (m *mockStorage) DeleteObject(ctx context.Context, key string) error {
	m.delCalls.Add(1)
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, key)
	}
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
	assert.Equal(t, int32(1), mock.putCalls.Load())
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

		if mock.putCalls.Load() < 3 {
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
	assert.Equal(t, int32(3), mock.putCalls.Load())
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
	assert.Equal(t, int32(3), mock.putCalls.Load())
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
	assert.Equal(t, int32(1), mock.putCalls.Load())
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
	assert.Equal(t, int32(1), mock.putCalls.Load())
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
	assert.Equal(t, int32(0), mock.putCalls.Load())
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
	assert.Equal(t, int32(0), mock.putCalls.Load())
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
	assert.Equal(t, int32(1), mock.putCalls.Load())
}

func TestRetryableObjectStorage_TransientError_Succeeds(t *testing.T) {
	mock := &mockStorage{}
	transientErr := minio.ErrorResponse{
		StatusCode: 503,
		Code:       "ServiceUnavailable",
		Message:    "Slow down",
	}

	mock.putFunc = func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
		if mock.putCalls.Load() < 2 {
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
	assert.Equal(t, int32(2), mock.putCalls.Load())
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

		if mock.putCalls.Load() < 2 {
			return minio.ErrorResponse{
				StatusCode: 503,
				Code:       "ServiceUnavailable",
			}
		}
		return nil
	}

	err = retryStorage.PutObject(context.Background(), "test-key", body, expectedSize, "text/plain")
	assert.NoError(t, err)
	assert.Equal(t, int32(2), mock.putCalls.Load())
}

func TestRetryableObjectStorage_GetObject_SuccessFirstAttempt(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:     mock,
		maxRetry:  2,
		baseDelay: 0,
		getBreaker: newCircuitBreaker(5, 30*time.Second),
	}

	_, _, err := retryStorage.GetObject(context.Background(), "test-key")
	assert.NoError(t, err)
	assert.Equal(t, int32(1), mock.getCalls.Load())

	cb := retryStorage.getBreaker
	assert.Equal(t, circuitClosed, cb.state)
	assert.Equal(t, 0, cb.failures)
}

func TestRetryableObjectStorage_GetObject_SuccessOnRetry(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:     mock,
		maxRetry:  2,
		baseDelay: 0,
		getBreaker: newCircuitBreaker(5, 30*time.Second),
	}

	mock.getFunc = func(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
		if mock.getCalls.Load() < 2 {
			return nil, ObjectInfo{}, minio.ErrorResponse{StatusCode: 503, Code: "ServiceUnavailable"}
		}
		return io.NopCloser(strings.NewReader("data")), ObjectInfo{Size: 4, ContentType: "text/plain"}, nil
	}

	rc, _, err := retryStorage.GetObject(context.Background(), "test-key")
	assert.NoError(t, err)
	assert.NotNil(t, rc)
	assert.Equal(t, int32(2), mock.getCalls.Load())
	_ = rc.Close()
}

func TestRetryableObjectStorage_GetObject_NotFound_NoRetry(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:     mock,
		maxRetry:  3,
		baseDelay: 0,
		getBreaker: newCircuitBreaker(5, 30*time.Second),
	}

	mock.getFunc = func(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
		return nil, ObjectInfo{}, ErrObjectNotFound
	}

	_, _, err := retryStorage.GetObject(context.Background(), "missing-key")
	assert.ErrorIs(t, err, ErrObjectNotFound)
	assert.Equal(t, int32(1), mock.getCalls.Load())
}

func TestRetryableObjectStorage_GetObject_MaxRetriesExhausted(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:     mock,
		maxRetry:  2,
		baseDelay: 0,
		getBreaker: newCircuitBreaker(5, 30*time.Second),
	}

	mock.getFunc = func(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
		return nil, ObjectInfo{}, minio.ErrorResponse{StatusCode: 500, Code: "InternalError"}
	}

	_, _, err := retryStorage.GetObject(context.Background(), "test-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GetObject failed after 3 attempts")
	assert.Equal(t, int32(3), mock.getCalls.Load())

	cb := retryStorage.getBreaker
	assert.Equal(t, 1, cb.failures)
}

func TestRetryableObjectStorage_DeleteObject_SuccessFirstAttempt(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:         mock,
		maxRetry:      2,
		baseDelay:     0,
		deleteBreaker: newCircuitBreaker(5, 30*time.Second),
	}

	err := retryStorage.DeleteObject(context.Background(), "test-key")
	assert.NoError(t, err)
	assert.Equal(t, int32(1), mock.delCalls.Load())
}

func TestRetryableObjectStorage_DeleteObject_SuccessOnRetry(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:         mock,
		maxRetry:      2,
		baseDelay:     0,
		deleteBreaker: newCircuitBreaker(5, 30*time.Second),
	}

	mock.deleteFunc = func(ctx context.Context, key string) error {
		if mock.delCalls.Load() < 2 {
			return minio.ErrorResponse{StatusCode: 503, Code: "ServiceUnavailable"}
		}
		return nil
	}

	err := retryStorage.DeleteObject(context.Background(), "test-key")
	assert.NoError(t, err)
	assert.Equal(t, int32(2), mock.delCalls.Load())
}

func TestRetryableObjectStorage_DeleteObject_MaxRetriesExhausted(t *testing.T) {
	mock := &mockStorage{}
	retryStorage := &retryableObjectStorage{
		inner:         mock,
		maxRetry:      2,
		baseDelay:     0,
		deleteBreaker: newCircuitBreaker(5, 30*time.Second),
	}

	mock.deleteFunc = func(ctx context.Context, key string) error {
		return minio.ErrorResponse{StatusCode: 500, Code: "InternalError"}
	}

	err := retryStorage.DeleteObject(context.Background(), "test-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DeleteObject failed after 3 attempts")
	assert.Equal(t, int32(3), mock.delCalls.Load())
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	mock := &mockStorage{}
	breaker := newCircuitBreaker(3, 100*time.Millisecond)
	retryStorage := &retryableObjectStorage{
		inner:      mock,
		maxRetry:   0,
		baseDelay:  0,
		putBreaker: breaker,
	}

	mock.putFunc = func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
		return minio.ErrorResponse{StatusCode: 500, Code: "InternalError"}
	}

	for i := 0; i < 3; i++ {
		_ = retryStorage.PutObject(context.Background(), "key", strings.NewReader("data"), 4, "text/plain")
	}

	assert.Equal(t, circuitOpen, breaker.state)

	err := retryStorage.PutObject(context.Background(), "key", strings.NewReader("data"), 4, "text/plain")
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestCircuitBreaker_HalfOpenAfterResetTimeout(t *testing.T) {
	breaker := newCircuitBreaker(1, 50*time.Millisecond)

	breaker.recordFailure()
	assert.Equal(t, circuitOpen, breaker.state)

	time.Sleep(80 * time.Millisecond)

	assert.True(t, breaker.allow())
	assert.Equal(t, circuitHalfOpen, breaker.state)
}

func TestCircuitBreaker_ClosesAfterHalfOpenRecovery(t *testing.T) {
	breaker := newCircuitBreaker(1, 50*time.Millisecond)

	breaker.recordFailure()
	assert.Equal(t, circuitOpen, breaker.state)

	time.Sleep(80 * time.Millisecond)

	assert.True(t, breaker.allow())
	breaker.recordSuccess()
	breaker.recordSuccess()
	assert.Equal(t, circuitClosed, breaker.state)
}

func TestCircuitBreaker_ReOpensOnHalfOpenFailure(t *testing.T) {
	breaker := newCircuitBreaker(1, 50*time.Millisecond)

	breaker.recordFailure()
	time.Sleep(80 * time.Millisecond)

	assert.True(t, breaker.allow())
	breaker.recordFailure()
	assert.Equal(t, circuitOpen, breaker.state)
}

func TestCircuitBreaker_GetObjectBlockedWhenOpen(t *testing.T) {
	mock := &mockStorage{}
	breaker := newCircuitBreaker(1, 1*time.Hour)
	breaker.recordFailure()

	retryStorage := &retryableObjectStorage{
		inner:      mock,
		maxRetry:   2,
		baseDelay:  0,
		getBreaker: breaker,
	}

	_, _, err := retryStorage.GetObject(context.Background(), "key")
	assert.ErrorIs(t, err, ErrCircuitGetOpen)
	assert.Equal(t, int32(0), mock.getCalls.Load())
}

func TestCircuitBreaker_DeleteObjectBlockedWhenOpen(t *testing.T) {
	mock := &mockStorage{}
	breaker := newCircuitBreaker(1, 1*time.Hour)
	breaker.recordFailure()

	retryStorage := &retryableObjectStorage{
		inner:         mock,
		maxRetry:      2,
		baseDelay:     0,
		deleteBreaker: breaker,
	}

	err := retryStorage.DeleteObject(context.Background(), "key")
	assert.ErrorIs(t, err, ErrCircuitDeleteOpen)
	assert.Equal(t, int32(0), mock.delCalls.Load())
}

func TestCircuitBreaker_PermanentError_CountsAsFailure(t *testing.T) {
	mock := &mockStorage{}
	breaker := newCircuitBreaker(2, 30*time.Second)
	retryStorage := &retryableObjectStorage{
		inner:      mock,
		maxRetry:   0,
		baseDelay:  0,
		putBreaker: breaker,
	}

	mock.putFunc = func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
		return errors.New("permanent error: access denied")
	}

	_ = retryStorage.PutObject(context.Background(), "key", strings.NewReader("data"), 4, "text/plain")
	assert.Equal(t, 1, breaker.failures)

	_ = retryStorage.PutObject(context.Background(), "key", strings.NewReader("data"), 4, "text/plain")
	assert.Equal(t, circuitOpen, breaker.state)
}

func TestCircuitBreaker_ContextErrorsDoNotCountAsFailure(t *testing.T) {
	mock := &mockStorage{}
	breaker := newCircuitBreaker(2, 30*time.Second)
	retryStorage := &retryableObjectStorage{
		inner:      mock,
		maxRetry:   0,
		baseDelay:  0,
		putBreaker: breaker,
	}

	mock.putFunc = func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
		return context.Canceled
	}

	_ = retryStorage.PutObject(context.Background(), "key", strings.NewReader("data"), 4, "text/plain")
	assert.Equal(t, 0, breaker.failures, "context.Canceled should not increment failures")
	assert.Equal(t, circuitClosed, breaker.state, "circuit should remain closed after context errors")

	mock.putFunc = func(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
		return context.DeadlineExceeded
	}

	_ = retryStorage.PutObject(context.Background(), "key", strings.NewReader("data"), 4, "text/plain")
	assert.Equal(t, 0, breaker.failures, "context.DeadlineExceeded should not increment failures")
	assert.Equal(t, circuitClosed, breaker.state)
}

func TestCircuitBreaker_HalfOpenLimitsConcurrentProbes(t *testing.T) {
	breaker := newCircuitBreaker(1, 50*time.Millisecond)

	breaker.recordFailure()
	assert.Equal(t, circuitOpen, breaker.state)

	time.Sleep(80 * time.Millisecond)

	assert.True(t, breaker.allow(), "first probe in half-open should be allowed")
	assert.Equal(t, circuitHalfOpen, breaker.state)
	assert.True(t, breaker.allow(), "second probe (halfOpenRequired=2) should be allowed")
	assert.False(t, breaker.allow(), "third probe should be rejected (in-flight limit reached)")
}

func TestCircuitBreaker_HalfOpenInFlightDecrementsOnSuccess(t *testing.T) {
	breaker := newCircuitBreaker(1, 50*time.Millisecond)

	breaker.recordFailure()
	time.Sleep(80 * time.Millisecond)

	assert.True(t, breaker.allow())
	assert.True(t, breaker.allow())
	assert.False(t, breaker.allow())

	breaker.recordSuccess()
	assert.True(t, breaker.allow(), "after one success, one slot freed")
}

func TestCircuitBreaker_HalfOpenInFlightResetsOnFailure(t *testing.T) {
	breaker := newCircuitBreaker(1, 50*time.Millisecond)

	breaker.recordFailure()
	time.Sleep(80 * time.Millisecond)

	assert.True(t, breaker.allow())
	breaker.recordFailure()
	assert.Equal(t, circuitOpen, breaker.state)
	assert.Equal(t, 0, breaker.halfOpenInFlight)
}

func TestCircuitBreaker_PutOpenDoesNotBlockGet(t *testing.T) {
	mock := &mockStorage{}
	putBreaker := newCircuitBreaker(1, 1*time.Hour)
	putBreaker.recordFailure()

	retryStorage := &retryableObjectStorage{
		inner:         mock,
		maxRetry:      0,
		baseDelay:     0,
		putBreaker:    putBreaker,
		getBreaker:    newCircuitBreaker(5, 30*time.Second),
		deleteBreaker: newCircuitBreaker(5, 30*time.Second),
	}

	putErr := retryStorage.PutObject(context.Background(), "k", strings.NewReader("x"), 1, "text/plain")
	assert.ErrorIs(t, putErr, ErrCircuitOpen)
	assert.Equal(t, int32(0), mock.putCalls.Load())

	_, _, getErr := retryStorage.GetObject(context.Background(), "k")
	assert.NoError(t, getErr, "GetObject must succeed even when PutObject breaker is open")
	assert.Equal(t, int32(1), mock.getCalls.Load())

	delErr := retryStorage.DeleteObject(context.Background(), "k")
	assert.NoError(t, delErr, "DeleteObject must succeed even when PutObject breaker is open")
	assert.Equal(t, int32(1), mock.delCalls.Load())
}

func TestCircuitBreaker_GetOpenDoesNotBlockPut(t *testing.T) {
	mock := &mockStorage{}
	getBreaker := newCircuitBreaker(1, 1*time.Hour)
	getBreaker.recordFailure()

	retryStorage := &retryableObjectStorage{
		inner:         mock,
		maxRetry:      0,
		baseDelay:     0,
		putBreaker:    newCircuitBreaker(5, 30*time.Second),
		getBreaker:    getBreaker,
		deleteBreaker: newCircuitBreaker(5, 30*time.Second),
	}

	_, _, getErr := retryStorage.GetObject(context.Background(), "k")
	assert.ErrorIs(t, getErr, ErrCircuitGetOpen)

	putErr := retryStorage.PutObject(context.Background(), "k", strings.NewReader("x"), 1, "text/plain")
	assert.NoError(t, putErr, "PutObject must succeed even when GetObject breaker is open")
	assert.Equal(t, int32(1), mock.putCalls.Load())
}
