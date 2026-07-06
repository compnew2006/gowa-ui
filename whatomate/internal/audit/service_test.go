package audit

import (
	"context"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecord_NilService_IsNoOp(t *testing.T) {
	var svc *Service // nil pointer
	// Must not panic.
	svc.Record(context.Background(), models.AuditEvent{Action: ActionLogout})
}

func TestRecord_NilDB_IsNoOp(t *testing.T) {
	svc := &Service{db: nil, log: testutil.NopLogger()}
	// Must not panic.
	svc.Record(context.Background(), models.AuditEvent{Action: ActionLogout})
}

func TestRecord_PersistsRow_GlobalEvent(t *testing.T) {
	db := testutil.SetupTestDB(t) // skips if TEST_DATABASE_URL unset
	svc := New(db, testutil.NopLogger())
	actorID := uuid.New()

	// OrganizationID nil → global event (assertable against the shared table).
	svc.Record(context.Background(), models.AuditEvent{
		Category:    CategoryAuth,
		Action:      ActionLoginSuccess,
		Source:      SourceUser,
		ActorUserID: &actorID,
		ActorEmail:  "admin@example.com",
		Success:     true,
		IPAddress:   "10.0.0.1",
	})

	var got models.AuditEvent
	require.NoError(t, db.Where("action = ?", ActionLoginSuccess).First(&got).Error)
	assert.NotEqual(t, uuid.Nil, got.ID)
	assert.False(t, got.CreatedAt.IsZero())
	assert.Equal(t, "admin@example.com", got.ActorEmail)
	assert.Equal(t, "10.0.0.1", got.IPAddress)
	assert.True(t, got.Success)
	assert.Nil(t, got.OrganizationID)
}

func TestRecord_AssignsIDAndCreatedAtWhenZero(t *testing.T) {
	db := testutil.SetupTestDB(t) // skips if TEST_DATABASE_URL unset
	svc := New(db, testutil.NopLogger())

	svc.Record(context.Background(), models.AuditEvent{
		Category: CategorySystem,
		Action:   ActionServerStarted,
		Source:   SourceSystem,
	})

	var got models.AuditEvent
	require.NoError(t, db.Where("action = ?", ActionServerStarted).First(&got).Error)
	assert.NotEqual(t, uuid.Nil, got.ID)
	assert.False(t, got.CreatedAt.IsZero())
}
