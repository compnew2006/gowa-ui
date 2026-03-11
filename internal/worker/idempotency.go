package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	recipientLockKeyPrefix = "whatomate:campaign:recipient:lock:"
	recipientLockTTL       = 2 * time.Minute
)

func recipientLockKey(recipientID uuid.UUID) string {
	return recipientLockKeyPrefix + recipientID.String()
}

func (w *Worker) acquireRecipientLock(ctx context.Context, recipientID uuid.UUID) (bool, error) {
	if w.Redis == nil {
		return true, nil
	}
	key := recipientLockKey(recipientID)
	// Try to acquire the lock using SET NX
	acquired, err := w.Redis.SetNX(ctx, key, "1", recipientLockTTL).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire recipient lock: %w", err)
	}
	return acquired, nil
}

func (w *Worker) releaseRecipientLock(ctx context.Context, recipientID uuid.UUID) {
	if w.Redis == nil {
		return
	}
	if err := w.Redis.Del(ctx, recipientLockKey(recipientID)).Err(); err != nil {
		w.Log.Warn("Failed to release recipient lock", "recipient_id", recipientID, "error", err)
	}
}
