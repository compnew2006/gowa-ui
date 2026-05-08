package handlers

import (
	"errors"
	"strings"
)

type reasonedError struct {
	message    string
	reasonCode string
	fallback   string
}

func (e *reasonedError) Error() string {
	if e == nil {
		return e.fallback
	}
	if strings.TrimSpace(e.message) == "" {
		return e.fallback
	}
	return e.message
}

func newReasonedError(message, reasonCode, fallback string) *reasonedError {
	return &reasonedError{
		message:    message,
		reasonCode: reasonCode,
		fallback:   fallback,
	}
}

func asReasonedError(err error) (*reasonedError, bool) {
	if err == nil {
		return nil, false
	}
	var target *reasonedError
	if !errors.As(err, &target) {
		return nil, false
	}
	return target, true
}
