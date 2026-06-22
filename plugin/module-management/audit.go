package modulemanagement

import (
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// These audit event action constants keep the ModuleEvent.Action field stable
// for filtering and reporting.
const (
	ModuleActionEnable      = "enable"
	ModuleActionDisable     = "disable"
	ModuleActionLicenseDeny = "license_deny"
	ModuleActionConflict    = "conflict"
)

const (
	moduleScopeGlobal       = "global"
	moduleScopeOrganization = "organization"
)

// ModuleEvent is an append-only audit row recording every give/ungive action on
// a managed module, including license denials and dependency conflicts. The
// shape mirrors models.LicenseEvent and AgentSelectionAuditEvent precedent.
type ModuleEvent struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	OrganizationID *uuid.UUID     `gorm:"type:uuid;index" json:"organization_id,omitempty"`
	Scope          string         `gorm:"size:20;not null;index" json:"scope"`
	ModuleKey      string         `gorm:"size:128;not null;index" json:"module_key"`
	Action         string         `gorm:"size:32;not null;index" json:"action"`
	Enabled        *bool          `json:"enabled,omitempty"`
	ActorUserID    *uuid.UUID     `gorm:"type:uuid" json:"actor_user_id,omitempty"`
	ActorEmail     string         `gorm:"size:255" json:"actor_email,omitempty"`
	Reason         string         `gorm:"size:255" json:"reason,omitempty"`
	Details        models.JSONB   `gorm:"type:jsonb;default:'{}'" json:"details"`
}

func (ModuleEvent) TableName() string { return "module_events" }

// MigrateModuleEvents ensures the module_events audit table exists. Called by
// the plugin's Migrate(); deliberately kept out of internal/database/postgres.go
// per the plugin-owns-its-schemas invariant.
func MigrateModuleEvents(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(&ModuleEvent{})
}
