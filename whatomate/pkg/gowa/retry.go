package gowa

import (
	"context"
	"math"
	"time"
)

// ----- Retry plumbing -----

// WithMaxRetries sets the retry count applied to retryable GOWA failures
// (HTTP 429, 5xx, and transport errors). Defaults to 0 (no retries) when
// unset. Wired from cfg.Gowa.MaxRetries in main.go.
func (a *GowaAdapter) WithMaxRetries(n int) *GowaAdapter {
	if n < 0 {
		n = 0
	}
	a.maxRetries = n
	return a
}

// withRetry wraps a GOWA call with bounded exponential backoff for retryable
// failures (HTTP 429, 5xx, transport errors). Non-retryable errors are
// returned immediately. The original context's deadline is always honoured.
func (a *GowaAdapter) withRetry(ctx context.Context, fn func(ctx context.Context) (string, error)) (string, error) {
	var last error
	for attempt := 0; attempt <= a.maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}
		last = err
		if !IsRetryable(err) {
			return "", err
		}
		if attempt < a.maxRetries {
			backoff := backoffFor(attempt)
			a.log.Warn("GOWA retryable failure, backing off", "attempt", attempt+1, "backoff_ms", backoff.Milliseconds(), "error", err)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
	if last == nil {
		last = ErrGowaNotConnected
	}
	return "", last
}

// withRetryErr is the error-only variant of withRetry.
func (a *GowaAdapter) withRetryErr(ctx context.Context, fn func(ctx context.Context) error) error {
	_, err := a.withRetry(ctx, func(ctx context.Context) (string, error) {
		if err := fn(ctx); err != nil {
			return "", err
		}
		return "", nil
	})
	return err
}

// backoffFor returns an exponential backoff for the given attempt index,
// capped at 16s. Sequence: 1s, 2s, 4s, 8s, 16s, 16s, ...
func backoffFor(attempt int) time.Duration {
	ms := math.Pow(2, float64(attempt)) * 1000 // 1, 2, 4, 8, 16, 32...
	if ms > 16000 {
		ms = 16000
	}
	return time.Duration(ms) * time.Millisecond
}
