package whatsmeow

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldRetrySendError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
		expected  sendErrorClass
	}{
		{
			name:      "nil error is retryable",
			err:       nil,
			retryable: true,
			expected:  sendErrorRetryable,
		},
		{
			name:      "context canceled is retryable",
			err:       context.Canceled,
			retryable: true,
			expected:  sendErrorRetryable,
		},
		{
			name:      "context deadline is retryable",
			err:       context.DeadlineExceeded,
			retryable: true,
			expected:  sendErrorRetryable,
		},
		{
			name:      "policy no inbound is permanent",
			err:       errors.New("POLICY_NO_INBOUND"),
			retryable: false,
			expected:  sendErrorPermanent,
		},
		{
			name:      "instance blocked is permanent",
			err:       errors.New("instance_blocked"),
			retryable: false,
			expected:  sendErrorPermanent,
		},
		{
			name:      "instance disconnected is permanent",
			err:       errors.New("instance is not connected"),
			retryable: false,
			expected:  sendErrorPermanent,
		},
		{
			name:      "invalid recipient is permanent",
			err:       errors.New("invalid recipient jid"),
			retryable: false,
			expected:  sendErrorPermanent,
		},
		{
			name:      "unauthorized is permanent",
			err:       errors.New("unauthorized request"),
			retryable: false,
			expected:  sendErrorPermanent,
		},
		{
			name:      "transient network error is retryable",
			err:       errors.New("dial tcp timeout"),
			retryable: true,
			expected:  sendErrorRetryable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, classifySendError(tc.err))
			assert.Equal(t, tc.retryable, shouldRetrySendError(tc.err))
		})
	}
}
