package whatsmeow

import (
	"context"
	"errors"
	"strings"
)

type sendErrorClass string

const (
	sendErrorRetryable sendErrorClass = "retryable"
	sendErrorPermanent sendErrorClass = "permanent"
)

func classifySendError(err error) sendErrorClass {
	if err == nil {
		return sendErrorRetryable
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return sendErrorRetryable
	}

	normalized := strings.ToLower(strings.TrimSpace(err.Error()))
	if normalized == "" {
		return sendErrorRetryable
	}

	permanentMarkers := []string{
		"policy_no_inbound",
		"policy_no_instance",
		"policy_draft_only",
		"instance_blocked",
		"instance_banned",
		"instance_not_connected",
		"instance_logged_out",
		"logged out",
		"instance is not connected",
		"instance not connected",
		"instance disconnected",
		"invalid jid",
		"invalid instance",
		"invalid phone",
		"invalid recipient",
		"unauthorized",
		"forbidden",
		"not allowed",
		"not found",
		"unsupported",
	}
	for _, marker := range permanentMarkers {
		if strings.Contains(normalized, marker) {
			return sendErrorPermanent
		}
	}

	// WhatsApp "server returned error 400" typically signals a transient
	// Signal/PN-LID session desync rather than a hard permanent failure.
	// whatsmeow logs "No sessions or sender keys found to migrate from <PN> to <LID>"
	// immediately before the 400 ack. The session is usually rebuilt within
	// seconds via an incoming message or app-state sync, so classifying it as
	// retryable gives the existing queue retry loop (1s/2s/4s backoff) a chance
	// to succeed instead of marking the message as permanently failed.
	if strings.Contains(normalized, "server returned error 400") ||
		strings.Contains(normalized, "400 bad request") {
		return sendErrorRetryable
	}

	return sendErrorRetryable
}

func shouldRetrySendError(err error) bool {
	return classifySendError(err) == sendErrorRetryable
}
