package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditEvent is the canonical cross-cutting audit record. It records
// security-relevant and operational events across the platform.
//
// The shape mirrors the established ModuleEvent pattern (nullable
// OrganizationID, typed ActorUserID/ActorEmail, JSONB Details). Unlike
// ModuleEvent, AuditEvent is tenant-scoped via internal/tenant at write
// time and is the canonical log for all scope-B categories.
type AuditEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`

	// Tenancy: nullable for global/system events (e.g. server restart).
	OrganizationID *uuid.UUID `gorm:"type:uuid;index" json:"organization_id,omitempty"`

	// What happened — typed enum-backed strings for indexability + flexibility.
	Category string `gorm:"size:32;not null;index" json:"category"`
	Action   string `gorm:"size:64;not null;index" json:"action"`

	// Who/what initiated it. Source disambiguates system vs user attribution.
	Source      string     `gorm:"size:16;not null;index" json:"source"`
	ActorUserID *uuid.UUID `gorm:"type:uuid" json:"actor_user_id,omitempty"`
	ActorEmail  string     `gorm:"size:255" json:"actor_email,omitempty"`
	ActorRole   string     `gorm:"size:32" json:"actor_role,omitempty"`

	// Target of the action (nullable for events without a single target).
	// TargetID is stringly-typed to uniformly handle UUIDs, numeric IDs, and JIDs.
	TargetType string  `gorm:"size:32" json:"target_type,omitempty"`
	TargetID   *string `gorm:"size:64;index" json:"target_id,omitempty"`

	// Outcome of the action.
	Success bool   `gorm:"not null;default:true" json:"success"`
	Reason  string `gorm:"size:255" json:"reason,omitempty"`

	// Free-form structured detail (IP, user-agent, before/after, params).
	Details JSONB `gorm:"type:jsonb;default:'{}'" json:"details"`

	// Optional request origin for security forensics.
	IPAddress string `gorm:"size:45" json:"ip_address,omitempty"`
	UserAgent string `gorm:"size:255" json:"user_agent,omitempty"`
}

func (AuditEvent) TableName() string { return "audit_events" }
