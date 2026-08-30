package handlers

import (
	"testing"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestStatusPriority_RevokedIsTerminal pins the revoke-vs-receipt guard.
// "revoked" must outrank every delivery-path status (sent/delivered/read AND
// failed): GOWA replays receipt batches on reconnect, so a late ack arriving
// after a sender unsent a message would otherwise flip the row back via
// updateMessageStatus's progression check, and a page refresh would render the
// deleted message with its original content again.
func TestStatusPriority_RevokedIsTerminal(t *testing.T) {
	tests := []struct {
		status models.MessageStatus
		want   int
	}{
		{models.MessageStatusPending, 0},
		{models.MessageStatusSent, 1},
		{models.MessageStatusDelivered, 2},
		{models.MessageStatusRead, 3},
		{models.MessageStatusFailed, 4},
		{models.MessageStatusRevoked, 5},
		{models.MessageStatus("unknown"), -1},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, statusPriority(tt.status), "status %q", tt.status)
	}

	// No delivery-path status may progress past revoked — including failed.
	for _, later := range []models.MessageStatus{
		models.MessageStatusSent,
		models.MessageStatusDelivered,
		models.MessageStatusRead,
		models.MessageStatusFailed,
	} {
		assert.LessOrEqual(t, statusPriority(later), statusPriority(models.MessageStatusRevoked),
			"%q must not override revoked", later)
	}
}
