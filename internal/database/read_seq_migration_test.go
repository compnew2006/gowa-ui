package database_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/database"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// The per-user unread migration converts the old conversation-wide read
// state (messages.status = read) into read_seq + read_watermark_seq without
// changing any badge: a conversation's unread count right after the
// migration must equal the old count, including unread messages older than
// a read one (history-synced or late-delivered). Runs in a private schema so
// it can start from a pre-migration table layout.
func TestReadSeqMigration_PreservesLegacyUnreadCounts(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping database test")
	}
	quiet := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	base, err := gorm.Open(postgres.Open(dsn), quiet)
	require.NoError(t, err)
	schema := "readseq_" + strings.ReplaceAll(uuid.New().String()[:8], "-", "")
	require.NoError(t, base.Exec("CREATE SCHEMA "+schema).Error)
	t.Cleanup(func() { base.Exec("DROP SCHEMA " + schema + " CASCADE") })

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := gorm.Open(postgres.Open(dsn+sep+"search_path="+schema), quiet)
	require.NoError(t, err)

	// Pre-migration layout: models only, no raw statements yet.
	for _, m := range database.GetMigrationModels() {
		require.NoError(t, db.AutoMigrate(m.Model), m.Name)
	}
	var hasCol bool
	require.NoError(t, db.Raw(`SELECT EXISTS (SELECT 1 FROM information_schema.columns
		WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'messages' AND column_name = 'read_seq')`).Scan(&hasCol).Error)
	require.False(t, hasCol, "read_seq must not be a model column")

	org := &models.Organization{Name: "Legacy Org", Slug: "legacy-" + uuid.New().String()[:8]}
	require.NoError(t, db.Create(org).Error)
	contact := func() uuid.UUID {
		c := &models.Contact{BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: org.ID,
			PhoneNumber: fmt.Sprintf("2010%07d", time.Now().UnixNano()%10000000)}
		require.NoError(t, db.Create(c).Error)
		return c.ID
	}
	t0 := time.Now().Add(-time.Hour)
	msg := func(contactID uuid.UUID, dir models.Direction, status models.MessageStatus, at time.Duration) {
		require.NoError(t, db.Create(&models.Message{
			BaseModel:      models.BaseModel{ID: uuid.New(), CreatedAt: t0.Add(at)},
			OrganizationID: org.ID, ContactID: contactID, Direction: dir,
			MessageType: models.MessageTypeText, Content: "x", Status: status,
		}).Error)
	}

	// An unread message stored BEFORE a message that was later read (a
	// "hole"), then a new unread message: legacy count 2. Insertion order
	// alone would hide the hole below the read message.
	withHole := contact()
	msg(withHole, models.DirectionIncoming, models.MessageStatusRead, 1*time.Minute)
	msg(withHole, models.DirectionIncoming, models.MessageStatusReceived, 2*time.Minute)
	msg(withHole, models.DirectionIncoming, models.MessageStatusRead, 3*time.Minute)
	msg(withHole, models.DirectionIncoming, models.MessageStatusDelivered, 4*time.Minute)
	msg(withHole, models.DirectionOutgoing, models.MessageStatusSent, 5*time.Minute)
	// Never opened: everything incoming is unread (2).
	neverOpened := contact()
	msg(neverOpened, models.DirectionIncoming, models.MessageStatusReceived, 1*time.Minute)
	msg(neverOpened, models.DirectionIncoming, models.MessageStatusDelivered, 2*time.Minute)
	// Fully read, plus terminal statuses that never count (0).
	allRead := contact()
	msg(allRead, models.DirectionIncoming, models.MessageStatusRead, 1*time.Minute)
	msg(allRead, models.DirectionIncoming, models.MessageStatusRevoked, 2*time.Minute)
	msg(allRead, models.DirectionIncoming, models.MessageStatusFailed, 3*time.Minute)

	unread := func(contactID uuid.UUID) int64 {
		var n int64
		require.NoError(t, db.Raw(`SELECT COUNT(*) FROM messages m JOIN contacts c ON c.id = m.contact_id
			WHERE m.contact_id = ? AND m.direction = 'incoming' AND m.status NOT IN ('revoked', 'failed')
			  AND m.deleted_at IS NULL AND m.read_seq > COALESCE(c.read_watermark_seq, 0)`, contactID).Scan(&n).Error)
		return n
	}
	expect := map[uuid.UUID]int64{withHole: 2, neverOpened: 2, allRead: 0}

	require.NoError(t, database.ApplyRawIndexes(db))
	for id, want := range expect {
		assert.Equal(t, want, unread(id), "badge must survive the migration unchanged")
	}

	// New rows are numbered after every migrated row, so they are unread.
	msg(allRead, models.DirectionIncoming, models.MessageStatusReceived, -30*time.Minute)
	expect[allRead] = 1
	assert.Equal(t, int64(1), unread(allRead), "a new message (even with an old timestamp) is unread")

	// Running the migration again is a no-op.
	require.NoError(t, database.ApplyRawIndexes(db))
	for id, want := range expect {
		assert.Equal(t, want, unread(id), "re-running the migration must not touch the read state")
	}
}
