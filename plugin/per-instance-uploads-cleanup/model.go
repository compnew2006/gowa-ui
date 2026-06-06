package perinstanceuploadscleanup

import (
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

type InstanceUploadsCleanupAudit struct {
	models.BaseModel
	OrganizationID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"organization_id"`
	InstanceID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_iuca_org_instance_created,priority:2" json:"instance_id"`
	ActorUserID      *uuid.UUID `gorm:"type:uuid;index" json:"actor_user_id,omitempty"`
	ActorEmail       *string    `gorm:"type:varchar(255)" json:"actor_email,omitempty"`
	OldInherit       *bool      `json:"old_inherit,omitempty"`
	NewInherit       bool       `gorm:"not null" json:"new_inherit"`
	OldRetentionDays *int       `json:"old_retention_days,omitempty"`
	NewRetentionDays *int       `json:"new_retention_days,omitempty"`
	Reason           *string    `gorm:"type:varchar(500)" json:"reason,omitempty"`
}

func (InstanceUploadsCleanupAudit) TableName() string {
	return "instance_uploads_cleanup_audits"
}
