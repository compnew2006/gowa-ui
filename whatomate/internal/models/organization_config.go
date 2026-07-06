package models

import "github.com/google/uuid"

// OrganizationConfig stores per-organization operational limits.
type OrganizationConfig struct {
	BaseModel
	OrganizationID       uuid.UUID     `gorm:"column:organization_id;type:uuid;not null;uniqueIndex" json:"organization_id"`
	WorkerCount          int           `gorm:"column:worker_count;not null;default:0" json:"worker_count"`
	MaxQueueSize         int           `gorm:"column:max_queue_size;not null;default:0" json:"max_queue_size"`
	MaxWhatsAppInstances int           `gorm:"column:max_whatsapp_instances;not null;default:0" json:"max_whatsapp_instances"`
	Organization         *Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"organization,omitempty"`
}

func (OrganizationConfig) TableName() string {
	return "organization_configs"
}
