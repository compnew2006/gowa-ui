package gowa

import (
	"errors"
	"fmt"
)

// Error is the typed error returned by the GOWA client for non-2xx responses.
// It captures the upstream HTTP status, error code, and message so callers can
// classify and retry appropriately.
type Error struct {
	// StatusCode is the HTTP status code returned by GOWA (e.g. 503).
	StatusCode int
	// Code is GOWA's internal error code from the envelope (e.g. "DEVICE_NOT_FOUND").
	Code string
	// Message is the human-readable error message from GOWA.
	Message string
	// Cause is the wrapped transport error, if any (e.g. context.DeadlineExceeded).
	Cause error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("gowa: status=%d code=%s: %s: %v", e.StatusCode, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("gowa: status=%d code=%s: %s", e.StatusCode, e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

// Retryable reports whether the failure is worth a retry attempt. The
// classification mirrors GOWA's own client.go: HTTP 429 (rate limited) and 5xx
// (transient server fault) are retryable; everything else (4xx) is final.
// Network/transport errors (e.g. connection refused) are also retryable.
func (e *Error) Retryable() bool {
	if e.StatusCode == 0 && e.Cause != nil {
		// Transport-level failure (DNS, connection refused, deadline).
		return true
	}
	if e.StatusCode == 429 {
		return true
	}
	if e.StatusCode >= 500 && e.StatusCode < 600 {
		return true
	}
	return false
}

// IsRetryable is a convenience for callers that hold a generic error.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	var ge *Error
	if errors.As(err, &ge) {
		return ge.Retryable()
	}
	// Transport errors that don't get wrapped in *Error are retryable.
	return true
}
