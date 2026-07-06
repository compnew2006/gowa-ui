package whatsmeow

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

// TestPersistStatusRecord_ConcurrentDuplicateIsNoOp reproduces the production
// race that produced 24 "duplicate key violates unique constraint
// idx_status_instance_wamid" errors: the same status broadcast reaches
// persistStatusRecord twice for the same (instance_id, wamid) within
// milliseconds (e.g. once via the low-priority shard and once via the legacy
// path). The old read-then-write implementation let both goroutines pass the
// existence check and then the second Create() failed.
//
// With the ON CONFLICT DO NOTHING fix, both calls must succeed and exactly one
// row must exist in the table.
func TestPersistStatusRecord_ConcurrentDuplicateIsNoOp(t *testing.T) {
	db := testutil.SetupTestDB(t) // skips if TEST_DATABASE_URL unset

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")

	orgID := uuid.New()
	instanceID := uuid.New()
	wamid := "3EB0F117A5C7A1B2C3D4"
	now := time.Now().UTC()

	// Two identical status records as if produced by two parallel event paths.
	buildStatus := func() *models.WhatsAppStatus {
		return &models.WhatsAppStatus{
			BaseModel: models.BaseModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			OrganizationID:    orgID,
			InstanceID:        instanceID,
			WhatsAppAccount:   "test-account",
			SenderJID:         "201001002003@s.whatsapp.net",
			SenderName:        "Tester",
			WhatsAppMessageID: wamid,
			StatusType:        models.WhatsAppStatusTypeText,
			Content:           "hello status",
			ExpiresAt:         now.Add(24 * time.Hour),
			Metadata:          models.JSONB{"from_me": false},
		}
	}

	var (
		wg       sync.WaitGroup
		errs     = make([]error, 2)
	)
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer wg.Done()
			errs[idx] = cm.persistStatusRecord(context.Background(), buildStatus())
		}(i)
	}
	wg.Wait()

	// Both calls must succeed — the loser is a silent no-op at the DB level,
	// never a duplicate-key error.
	for i, err := range errs {
		assert.NoError(t, err, "concurrent insert %d must not error", i)
	}

	// Exactly one row must survive regardless of insertion order.
	var count int64
	require.NoError(t, db.Model(&models.WhatsAppStatus{}).
		Where("instance_id = ? AND whats_app_message_id = ?", instanceID, wamid).
		Count(&count).Error)
	assert.Equal(t, int64(1), count, "exactly one status row should exist after the race")
}

// TestPersistStatusRecord_SequentialSameWamidIsIdempotent covers the simpler
// non-concurrent case: persisting the same wamid twice in sequence must be a
// no-op on the second call (previously it was caught by the read check, now it
// is handled by ON CONFLICT DO NOTHING).
func TestPersistStatusRecord_SequentialSameWamidIsIdempotent(t *testing.T) {
	db := testutil.SetupTestDB(t) // skips if TEST_DATABASE_URL unset

	cm := NewConnectionManager(db, nil, logf.New(logf.Opts{}), &config.WhatsmeowConfig{}, nil, "./uploads")

	orgID := uuid.New()
	instanceID := uuid.New()
	wamid := "3EB0SEQ1A5C7A1B2C3D5"
	now := time.Now().UTC()

	buildStatus := func() *models.WhatsAppStatus {
		return &models.WhatsAppStatus{
			OrganizationID:    orgID,
			InstanceID:        instanceID,
			SenderJID:         "201001002004@s.whatsapp.net",
			WhatsAppMessageID: wamid,
			StatusType:        models.WhatsAppStatusTypeText,
			ExpiresAt:         now.Add(24 * time.Hour),
			Metadata:          models.JSONB{},
		}
	}

	require.NoError(t, cm.persistStatusRecord(context.Background(), buildStatus()))
	require.NoError(t, cm.persistStatusRecord(context.Background(), buildStatus()))

	var count int64
	require.NoError(t, db.Model(&models.WhatsAppStatus{}).
		Where("instance_id = ? AND whats_app_message_id = ?", instanceID, wamid).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
