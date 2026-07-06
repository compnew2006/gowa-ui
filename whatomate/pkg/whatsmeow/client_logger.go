package whatsmeow

import (
	"fmt"
	"strings"

	waLog "go.mau.fi/whatsmeow/util/log"
)

const statusRetryReceiptErrorPrefix = "Failed to handle retry receipt for status@broadcast/"
const socketEOFErrorPrefix = "Error reading from websocket:"

type filteredClientLogger struct {
	inner waLog.Logger
}

func newClientLogger(inner waLog.Logger) waLog.Logger {
	if inner == nil {
		inner = waLog.Noop
	}
	return &filteredClientLogger{inner: inner}
}

func shouldSuppressClientError(msg string, args ...interface{}) bool {
	if !strings.Contains(msg, "Failed to handle retry receipt for") {
		return false
	}
	rendered := fmt.Sprintf(msg, args...)
	if !strings.Contains(rendered, statusRetryReceiptErrorPrefix) {
		return false
	}
	return strings.Contains(rendered, "couldn't find message")
}

func shouldDemoteClientError(msg string, args ...interface{}) bool {
	if !strings.Contains(msg, socketEOFErrorPrefix) {
		return false
	}
	rendered := fmt.Sprintf(msg, args...)
	return strings.Contains(rendered, "failed to get reader") &&
		strings.Contains(rendered, "failed to read frame header: EOF")
}

func (l *filteredClientLogger) Warnf(msg string, args ...interface{}) {
	l.inner.Warnf(msg, args...)
}

func (l *filteredClientLogger) Errorf(msg string, args ...interface{}) {
	if shouldSuppressClientError(msg, args...) {
		return
	}
	if shouldDemoteClientError(msg, args...) {
		l.inner.Warnf(msg, args...)
		return
	}
	l.inner.Errorf(msg, args...)
}

func (l *filteredClientLogger) Infof(msg string, args ...interface{}) {
	l.inner.Infof(msg, args...)
}

func (l *filteredClientLogger) Debugf(msg string, args ...interface{}) {
	l.inner.Debugf(msg, args...)
}

func (l *filteredClientLogger) Sub(module string) waLog.Logger {
	return &filteredClientLogger{inner: l.inner.Sub(module)}
}
