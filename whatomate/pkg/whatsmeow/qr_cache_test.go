package whatsmeow

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestConnectionManagerQRCodeCache_Lifecycle(t *testing.T) {
	t.Parallel()

	cm := &ConnectionManager{
		qrCodes: make(map[uuid.UUID]cachedQRCode),
	}
	instanceID := uuid.New()

	_, ok := cm.GetCachedQRCode(instanceID)
	assert.False(t, ok)

	cm.CacheQRCode(instanceID, "qr-code-value", 2)

	snapshot, ok := cm.GetCachedQRCode(instanceID)
	assert.True(t, ok)
	assert.Equal(t, "qr-code-value", snapshot.Code)
	assert.Equal(t, 2, snapshot.TimeoutSec)
	assert.False(t, snapshot.ReceivedAt.IsZero())

	cm.ClearCachedQRCode(instanceID)
	_, ok = cm.GetCachedQRCode(instanceID)
	assert.False(t, ok)
}

func TestConnectionManagerQRCodeCache_Expires(t *testing.T) {
	t.Parallel()

	instanceID := uuid.New()
	cm := &ConnectionManager{
		qrCodes: map[uuid.UUID]cachedQRCode{
			instanceID: {
				code:       "stale",
				timeoutSec: 1,
				receivedAt: time.Now().UTC().Add(-3 * time.Second),
			},
		},
	}

	_, ok := cm.GetCachedQRCode(instanceID)
	assert.False(t, ok)
	assert.Empty(t, cm.qrCodes)
}
