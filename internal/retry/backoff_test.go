package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithBackoffSuccessOnFirstAttempt(t *testing.T) {
	attempts := 0

	got, err := WithBackoff(context.Background(), 2, time.Millisecond, func(context.Context) (string, error) {
		attempts++
		return "ok", nil
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, "ok", got)
	assert.Equal(t, 1, attempts)
}

func TestWithBackoffSuccessAfterRetry(t *testing.T) {
	attempts := 0
	retryAttempts := []int{}

	got, err := WithBackoff(context.Background(), 2, time.Millisecond, func(context.Context) (string, error) {
		attempts++
		if attempts == 1 {
			return "", errors.New("try again")
		}
		return "ok", nil
	}, func(attempt int, _ error) {
		retryAttempts = append(retryAttempts, attempt)
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", got)
	assert.Equal(t, 2, attempts)
	assert.Equal(t, []int{1}, retryAttempts)
}

func TestWithBackoffRetriesExhaustedReturnsLastError(t *testing.T) {
	firstErr := errors.New("first")
	lastErr := errors.New("last")
	attempts := 0

	_, err := WithBackoff(context.Background(), 1, time.Millisecond, func(context.Context) (string, error) {
		attempts++
		if attempts == 1 {
			return "", firstErr
		}
		return "", lastErr
	}, nil)

	assert.ErrorIs(t, err, lastErr)
	assert.Equal(t, 2, attempts)
}

func TestWithBackoffContextCanceledBeforeRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attempts := 0
	_, err := WithBackoff(ctx, 1, time.Hour, func(context.Context) (string, error) {
		attempts++
		return "", errors.New("first failure")
	}, nil)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, attempts)
}

func TestWithBackoffContextCanceledDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	attempts := 0
	_, err := WithBackoff(ctx, 1, time.Hour, func(context.Context) (string, error) {
		attempts++
		cancel()
		return "", errors.New("first failure")
	}, nil)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, attempts)
}
