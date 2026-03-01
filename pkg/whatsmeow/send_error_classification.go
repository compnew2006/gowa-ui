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

	return sendErrorRetryable
}

func shouldRetrySendError(err error) bool {
	return classifySendError(err) == sendErrorRetryable
}
