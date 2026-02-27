package whatsmeow

import (
	"time"

	"github.com/google/uuid"
)

// QRCodeSnapshot is the latest known QR code for an instance.
type QRCodeSnapshot struct {
	Code       string
	TimeoutSec int
	ReceivedAt time.Time
}

// CacheQRCode stores the latest QR code for an instance with a TTL window.
func (cm *ConnectionManager) CacheQRCode(instanceID uuid.UUID, code string, timeoutSec int) {
	if cm == nil || instanceID == uuid.Nil || code == "" {
		return
	}
	if timeoutSec <= 0 {
		timeoutSec = 20
	}

	cm.qrCodesMu.Lock()
	defer cm.qrCodesMu.Unlock()
	cm.qrCodes[instanceID] = cachedQRCode{
		code:       code,
		timeoutSec: timeoutSec,
		receivedAt: time.Now().UTC(),
	}
}

// ClearCachedQRCode removes any cached QR code for the instance.
func (cm *ConnectionManager) ClearCachedQRCode(instanceID uuid.UUID) {
	if cm == nil || instanceID == uuid.Nil {
		return
	}
	cm.qrCodesMu.Lock()
	defer cm.qrCodesMu.Unlock()
	delete(cm.qrCodes, instanceID)
}

// GetCachedQRCode returns a non-expired QR snapshot if available.
func (cm *ConnectionManager) GetCachedQRCode(instanceID uuid.UUID) (QRCodeSnapshot, bool) {
	if cm == nil || instanceID == uuid.Nil {
		return QRCodeSnapshot{}, false
	}

	cm.qrCodesMu.RLock()
	entry, ok := cm.qrCodes[instanceID]
	cm.qrCodesMu.RUnlock()
	if !ok || entry.code == "" {
		return QRCodeSnapshot{}, false
	}

	expiresAt := entry.receivedAt.Add(time.Duration(entry.timeoutSec) * time.Second)
	if time.Now().UTC().After(expiresAt) {
		cm.ClearCachedQRCode(instanceID)
		return QRCodeSnapshot{}, false
	}

	return QRCodeSnapshot{
		Code:       entry.code,
		TimeoutSec: entry.timeoutSec,
		ReceivedAt: entry.receivedAt,
	}, true
}
