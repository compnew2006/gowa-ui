package audit

import (
	"context"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// Service is the central audit recorder. Safe for concurrent use.
// One instance lives on *handlers.App.
type Service struct {
	db  *gorm.DB
	log logf.Logger
}

// New returns a recorder backed by the global (unscoped) DB. Tenant scope is
// carried per-row by AuditEvent.OrganizationID; this codebase has no per-tenant
// physical DB or RLS (internal/tenant.ScopedDB only adds a WHERE scope that is
// a no-op for inserts), so writes go directly to the shared table and the
// organization_id column is the tenant boundary on the read side.
func New(db *gorm.DB, log logf.Logger) *Service {
	return &Service{db: db, log: log}
}

// Record persists one event. Best-effort: logs on failure, never panics,
// and never surfaces an error to the caller. The AuditEvent.OrganizationID
// field carries tenant scope into the row; system/global events leave it nil.
func (s *Service) Record(ctx context.Context, evt models.AuditEvent) {
	if s == nil || s.db == nil {
		return
	}
	evt.ID = uuid.New()
	if evt.CreatedAt.IsZero() {
		evt.CreatedAt = time.Now().UTC()
	}

	if err := s.db.WithContext(ctx).Create(&evt).Error; err != nil {
		s.log.Error("audit write failed",
			"error", err,
			"category", evt.Category,
			"action", evt.Action,
			"source", evt.Source)
	}
}
