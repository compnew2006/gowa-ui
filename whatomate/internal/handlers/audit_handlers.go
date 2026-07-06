package handlers

import (
	"strconv"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

const (
	auditDefaultPerPage = 50
	auditMaxPerPage     = 200
)

// ListAuditEvents GET /api/audit-events
//
// Admin-only (requires audit:read). Returns audit events with filtering and
// pagination. Super admins see events across all orgs (and global events);
// org admins see only their own org's events plus global (organization_id IS NULL)
// events. Non-admins are rejected by requirePermission before reaching the body.
//
// Filters: category, action, source, actor_user_id, target_id, target_type,
//
//	success, q (text on actor_email/reason), date_from, date_to,
//	organization_id (super-admin only)
//
// Pagination: page (1-based), per_page (default 50, max 200)
// Sort: created_at DESC (newest first)
func (a *App) ListAuditEvents(r *fastglue.Request) error {
	_, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAudit, models.ActionRead); err != nil {
		return nil // forbidden envelope already sent
	}

	q := r.RequestCtx.QueryArgs()

	// Base query: super-admin reads the global DB (optionally filtered by org);
	// org admin reads the tenant-scoped DB and additionally sees global events
	// (organization_id IS NULL), which carry no org-scoped actor actions.
	var db *gorm.DB
	isSuperAdmin := middleware.IsSuperAdmin(r)

	if isSuperAdmin {
		db = a.DB.Model(&models.AuditEvent{})
		if orgStr := string(q.Peek("organization_id")); orgStr != "" {
			db = db.Where("organization_id = ?", orgStr)
		}
	} else {
		orgID, _ := middleware.GetOrganizationID(r)
		db = a.requestDB(r).Model(&models.AuditEvent{}).
			Where("organization_id = ? OR organization_id IS NULL", orgID)
	}

	// Filters
	if v := string(q.Peek("category")); v != "" {
		db = db.Where("category = ?", v)
	}
	if v := string(q.Peek("action")); v != "" {
		db = db.Where("action = ?", v)
	}
	if v := string(q.Peek("source")); v != "" {
		db = db.Where("source = ?", v)
	}
	if v := string(q.Peek("actor_user_id")); v != "" {
		db = db.Where("actor_user_id = ?", v)
	}
	if v := string(q.Peek("target_id")); v != "" {
		db = db.Where("target_id = ?", v)
	}
	if v := string(q.Peek("target_type")); v != "" {
		db = db.Where("target_type = ?", v)
	}
	if v := string(q.Peek("success")); v != "" {
		if v == "true" || v == "1" {
			db = db.Where("success = ?", true)
		} else if v == "false" || v == "0" {
			db = db.Where("success = ?", false)
		}
	}
	if v := string(q.Peek("q")); v != "" {
		like := "%" + v + "%"
		db = db.Where("actor_email ILIKE ? OR reason ILIKE ?", like, like)
	}
	if v := string(q.Peek("date_from")); v != "" {
		db = db.Where("created_at >= ?", v)
	}
	if v := string(q.Peek("date_to")); v != "" {
		db = db.Where("created_at <= ?", v)
	}

	// Count (on a clone so pagination clauses don't affect it).
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		a.Log.Error("Failed to count audit events", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list audit events", nil, "")
	}

	// Pagination
	page, _ := strconv.Atoi(string(q.Peek("page")))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(string(q.Peek("per_page")))
	if perPage < 1 {
		perPage = auditDefaultPerPage
	}
	if perPage > auditMaxPerPage {
		perPage = auditMaxPerPage
	}
	offset := (page - 1) * perPage

	var events []models.AuditEvent
	if err := db.Order("created_at DESC").
		Offset(offset).Limit(perPage).
		Find(&events).Error; err != nil {
		a.Log.Error("Failed to list audit events", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list audit events", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"events":   events,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}
