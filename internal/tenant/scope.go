package tenant

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ContextKeyScopedDB       = "tenant_db"
	contextKeyUserID         = "user_id"
	contextKeyOrganizationID = "organization_id"
	contextKeyIsSuperAdmin   = "is_super_admin"
	headerOrganizationID     = "X-Organization-ID"
)

// ScopedDB returns a request-scoped GORM clone with tenant filtering enabled.
func ScopedDB(db *gorm.DB, orgID uuid.UUID) *gorm.DB {
	if db == nil || orgID == uuid.Nil {
		return db
	}
	return db.Session(&gorm.Session{}).Scopes(func(tx *gorm.DB) *gorm.DB {
		if tx == nil || tx.Statement == nil {
			return tx
		}

		if tx.Statement.Schema == nil {
			switch {
			case tx.Statement.Model != nil:
				_ = tx.Statement.Parse(tx.Statement.Model)
			case tx.Statement.Dest != nil:
				_ = tx.Statement.Parse(tx.Statement.Dest)
			}
		}

		if tx.Statement.Schema == nil {
			return tx
		}

		field := tx.Statement.Schema.LookUpField("OrganizationID")
		if field == nil {
			return tx
		}

		return tx.Where(clause.Eq{
			Column: clause.Column{Table: clause.CurrentTable, Name: field.DBName},
			Value:  orgID,
		})
	})
}

func SetScopedDB(r *fastglue.Request, db *gorm.DB) {
	if r == nil || db == nil {
		return
	}
	r.RequestCtx.SetUserValue(ContextKeyScopedDB, db)
}

func GetScopedDB(r *fastglue.Request) (*gorm.DB, bool) {
	if r == nil {
		return nil, false
	}
	db, ok := r.RequestCtx.UserValue(ContextKeyScopedDB).(*gorm.DB)
	return db, ok
}

// ResolveOrganizationID calculates the effective organization for a request.
func ResolveOrganizationID(r *fastglue.Request, db *gorm.DB) (uuid.UUID, error) {
	if r == nil {
		return uuid.Nil, errors.New("request is nil")
	}

	orgIDVal := r.RequestCtx.UserValue(contextKeyOrganizationID)
	if orgIDVal == nil {
		return uuid.Nil, errors.New("organization_id not found in context")
	}
	defaultOrgID, ok := parseContextUUID(orgIDVal)
	if !ok {
		return uuid.Nil, errors.New("organization_id is not a valid UUID")
	}

	overrideOrgID := strings.TrimSpace(string(r.RequestCtx.Request.Header.Peek(headerOrganizationID)))
	if overrideOrgID == "" || db == nil {
		return defaultOrgID, nil
	}

	parsedOrgID, err := uuid.Parse(overrideOrgID)
	if err != nil || parsedOrgID == defaultOrgID {
		return defaultOrgID, nil
	}

	userID, ok := parseContextUUID(r.RequestCtx.UserValue(contextKeyUserID))
	if !ok {
		return defaultOrgID, nil
	}

	isSuperAdmin, _ := r.RequestCtx.UserValue(contextKeyIsSuperAdmin).(bool)
	var count int64
	if isSuperAdmin {
		if err := db.Table("organizations").Where("id = ? AND deleted_at IS NULL", parsedOrgID).Count(&count).Error; err == nil && count > 0 {
			return parsedOrgID, nil
		}
		return defaultOrgID, nil
	}

	if err := db.Table("user_organizations").
		Where("user_id = ? AND organization_id = ? AND deleted_at IS NULL", userID, parsedOrgID).
		Count(&count).Error; err == nil && count > 0 {
		return parsedOrgID, nil
	}

	return defaultOrgID, nil
}

func parseContextUUID(value any) (uuid.UUID, bool) {
	switch typed := value.(type) {
	case uuid.UUID:
		if typed == uuid.Nil {
			return uuid.Nil, false
		}
		return typed, true
	case string:
		parsed, err := uuid.Parse(strings.TrimSpace(typed))
		if err != nil || parsed == uuid.Nil {
			return uuid.Nil, false
		}
		return parsed, true
	default:
		return uuid.Nil, false
	}
}
