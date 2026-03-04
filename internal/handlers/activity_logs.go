package handlers

import (
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// CreateActivityLogRequest defines the request body for custom activity events.
type CreateActivityLogRequest struct {
	Category  string       `json:"category"`
	EventType string       `json:"event_type"`
	Action    string       `json:"action"`
	ContactID string       `json:"contact_id,omitempty"`
	MessageID string       `json:"message_id,omitempty"`
	Metadata  models.JSONB `json:"metadata"`
}

func (a *App) canAccessActivityLogs(userID, orgID uuid.UUID) bool {
	var user models.User
	if err := a.DB.Select("is_super_admin").Where("id = ?", userID).First(&user).Error; err == nil && user.IsSuperAdmin {
		return true
	}

	perms, err := a.getUserPermissionsCached(userID, orgID)
	if err != nil {
		return false
	}
	return strings.EqualFold(perms.RoleName, "admin") || strings.EqualFold(perms.RoleName, "manager")
}

// CreateActivityLog stores a custom activity event for the authenticated user.
func (a *App) CreateActivityLog(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if !a.canAccessActivityLogs(userID, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Access denied", nil, "")
	}

	var req CreateActivityLogRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	req.Category = strings.TrimSpace(req.Category)
	if req.Category == "" {
		req.Category = "custom"
	}
	if req.Category != "custom" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "category must be custom", nil, "")
	}

	req.EventType = strings.TrimSpace(req.EventType)
	req.Action = strings.TrimSpace(req.Action)
	if req.EventType == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "event_type is required", nil, "")
	}
	if req.Action == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "action is required", nil, "")
	}

	var contactID *uuid.UUID
	if req.ContactID != "" {
		parsed, parseErr := uuid.Parse(req.ContactID)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid contact ID", nil, "")
		}
		contactID = &parsed
	}

	var messageID *uuid.UUID
	if req.MessageID != "" {
		parsed, parseErr := uuid.Parse(req.MessageID)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid message ID", nil, "")
		}
		messageID = &parsed
	}

	entry, logErr := a.LogCustomEvent(
		r,
		userID,
		orgID,
		req.Category,
		req.EventType,
		req.Action,
		contactID,
		messageID,
		req.Metadata,
	)
	if logErr != nil {
		a.Log.Error("Failed to create custom activity log", "error", logErr, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create activity log", nil, "")
	}

	return r.SendEnvelope(entry)
}

// ListActivityLogs returns paginated activity events for the authenticated user.
func (a *App) ListActivityLogs(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if !a.canAccessActivityLogs(userID, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Access denied", nil, "")
	}

	pg := parsePagination(r)
	filter := ActivityListFilter{
		Pagination: pg,
		Category:   strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("category"))),
		EventType:  strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("event_type"))),
		Source:     strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("source"))),
		Status:     strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("status"))),
	}

	if startDate, ok := parseDateParam(r, "start_date"); ok {
		filter.StartDate = &startDate
	}

	if endDate, ok := parseDateParam(r, "end_date"); ok {
		end := endOfDay(endDate)
		filter.EndDate = &end
	}

	logs, total, listErr := a.ListOwnEvents(userID, orgID, filter)
	if listErr != nil {
		a.Log.Error("Failed to list activity logs", "error", listErr, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list activity logs", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"logs":  logs,
		"total": total,
		"page":  pg.Page,
		"limit": pg.Limit,
	})
}
