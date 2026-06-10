package perinstanceuploadscleanup

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxRetentionDays = 3650

type RetentionSnapshot struct {
	Inherit       bool
	RetentionDays *int
}

type effectiveRetention struct {
	Days   int
	Source string
}

type service struct {
	db           *gorm.DB
	log          *slog.Logger
	instanceRunMu sync.Mutex
}

func newService(db *gorm.DB, log *slog.Logger) *service {
	return &service{db: db, log: log}
}

func (s *service) ResolveEffectiveRetention(ctx context.Context, orgID, instanceID uuid.UUID, now time.Time) (int, string, error) {
	var instance models.WhatsAppInstance
	scopedDB := tenant.ScopedDB(s.db, orgID)
	if err := scopedDB.Where("id = ?", instanceID).First(&instance).Error; err != nil {
		return 0, "", fmt.Errorf("resolve effective retention: %w", err)
	}

	workspaceDefault, _, err := s.resolveWorkspaceDefault(orgID)
	if err != nil {
		return 0, "", err
	}

	days, source := handlers.ResolveInstanceRetention(instance.Settings, workspaceDefault)
	if days > maxRetentionDays {
		days = maxRetentionDays
	}
	return days, source, nil
}

func (s *service) resolveWorkspaceDefault(orgID uuid.UUID) (int, string, error) {
	var org models.Organization
	if err := s.db.Where("id = ?", orgID).First(&org).Error; err != nil {
		return 0, "", fmt.Errorf("resolve workspace default: %w", err)
	}
	orgSettings := org.Settings
	if orgSettings == nil {
		return 0, "disabled", nil
	}
	rd, ok := orgSettings["uploads_cleanup_retention_days"].(float64)
	if !ok || int(rd) == 0 {
		return 0, "disabled", nil
	}
	days := int(rd)
	if days > maxRetentionDays {
		days = maxRetentionDays
	}
	return days, "default", nil
}

func (s *service) WriteAuditRow(ctx context.Context, orgID, instanceID uuid.UUID, actorUserID *uuid.UUID, actorEmail *string, old, new RetentionSnapshot, reason *string) error {
	audit := InstanceUploadsCleanupAudit{
		OrganizationID:   orgID,
		InstanceID:       instanceID,
		ActorUserID:      actorUserID,
		ActorEmail:       actorEmail,
		OldInherit:       &old.Inherit,
		NewInherit:       new.Inherit,
		OldRetentionDays: old.RetentionDays,
		NewRetentionDays: new.RetentionDays,
		Reason:           reason,
	}
	return s.db.WithContext(ctx).Create(&audit).Error
}

func (s *service) tryAcquireInstanceRun() (release func(), ok bool) {
	s.instanceRunMu.Lock()
	return s.instanceRunMu.Unlock, true
}
