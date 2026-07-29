package handlers

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"

	"github.com/shridarpatil/whatomate/internal/audit"
	"github.com/shridarpatil/whatomate/internal/models"
)

// errEnvelopeSent is a sentinel returned by helpers after they have already
// written an error envelope to the response. Callers should return nil to the framework.
var errEnvelopeSent = errors.New("error envelope sent")

// parsePathUUID extracts a UUID from a path parameter. On failure, it sends a
// 400 error envelope and returns uuid.Nil plus an error.
func parsePathUUID(r *fastglue.Request, param, label string) (uuid.UUID, error) {
	idStr, _ := r.RequestCtx.UserValue(param).(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid "+label+" ID", nil, "")
		return uuid.Nil, errEnvelopeSent
	}
	return id, nil
}

// requireOrgID resolves the caller's organization ID from the request context.
// On failure it sends a 401 Unauthorized error envelope and returns
// errEnvelopeSent, so callers can simply `return nil` to the framework. This
// collapses the org-preamble that was repeated across ~37 handlers.
func (a *App) requireOrgID(r *fastglue.Request) (uuid.UUID, error) {
	orgID, err := a.getOrgID(r)
	if err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
		return uuid.Nil, errEnvelopeSent
	}
	return orgID, nil
}

// requireOrgAndUserID resolves both the organization and user IDs from the
// request context. On failure it sends a 401 Unauthorized error envelope and
// returns errEnvelopeSent, so callers can `return nil`. Parallels requireOrgID
// for the many handlers that also need the acting user's ID.
func (a *App) requireOrgAndUserID(r *fastglue.Request) (orgID, userID uuid.UUID, err error) {
	orgID, userID, err = a.getOrgAndUserID(r)
	if err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
		return uuid.Nil, uuid.Nil, errEnvelopeSent
	}
	return orgID, userID, nil
}

// resolveOrgEntity combines the three-step request preamble used by most
// single-entity handlers: resolve the org, parse the path UUID, and load the
// record scoped by ID and organization. On any failure it has already written
// the appropriate error envelope and returns errEnvelopeSent, so callers can
// `return nil`. Use only for the simple id+organization_id lookup; handlers that
// need Preload or a custom query should call requireOrgID + findByIDAndOrg
// (or their own query) directly.
func resolveOrgEntity[T any](a *App, r *fastglue.Request, param, label string) (uuid.UUID, *T, error) {
	orgID, err := a.requireOrgID(r)
	if err != nil {
		return uuid.Nil, nil, err
	}
	id, err := parsePathUUID(r, param, label)
	if err != nil {
		return uuid.Nil, nil, err
	}
	entity, err := findByIDAndOrg[T](a.DB, r, id, orgID, label)
	if err != nil {
		return uuid.Nil, nil, err
	}
	return orgID, entity, nil
}

// Pagination holds parsed pagination parameters.
type Pagination struct {
	Page   int
	Limit  int
	Offset int
}

// Apply adds Offset and Limit to a GORM query.
func (pg Pagination) Apply(query *gorm.DB) *gorm.DB {
	return query.Offset(pg.Offset).Limit(pg.Limit)
}

// parsePagination extracts page-based pagination from query params with
// default limit=50 and max limit=100.
func parsePagination(r *fastglue.Request) Pagination {
	return parsePaginationWithDefaults(r, 50, 100)
}

// parsePaginationWithDefaults extracts page-based pagination with custom defaults.
func parsePaginationWithDefaults(r *fastglue.Request, defaultLimit, maxLimit int) Pagination {
	page, _ := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
	limit, _ := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("limit")))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > maxLimit {
		limit = defaultLimit
	}
	return Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

// parseDateParam parses a YYYY-MM-DD date from the named query parameter.
// Returns the parsed time and true on success, or zero time and false if the
// parameter is missing or malformed.
func parseDateParam(r *fastglue.Request, param string) (time.Time, bool) {
	s := string(r.RequestCtx.QueryArgs().Peek(param))
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// endOfDay returns the last nanosecond of the given day.
func endOfDay(t time.Time) time.Time {
	return t.Add(24*time.Hour - time.Nanosecond)
}

// findByIDAndOrg fetches a single record scoped by ID and organization.
// Sends a 404 error envelope on failure and returns the error.
func findByIDAndOrg[T any](db *gorm.DB, r *fastglue.Request, id, orgID uuid.UUID, label string) (*T, error) {
	var model T
	if err := db.Where("id = ? AND organization_id = ?", id, orgID).First(&model).Error; err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusNotFound, label+" not found", nil, "")
		return nil, errEnvelopeSent
	}
	return &model, nil
}

// logAudit records an audit-log entry for a resource mutation, resolving the
// actor's display name automatically. It wraps audit.LogAudit to remove the
// repeated a.DB + GetUserName boilerplate at call sites.
func (a *App) logAudit(orgID, userID uuid.UUID, resourceType string, resourceID uuid.UUID, action models.AuditAction, oldData, newData any, extraChanges ...map[string]any) {
	audit.LogAudit(a.DB, orgID, userID, audit.GetUserName(a.DB, userID), resourceType, resourceID, action, oldData, newData, extraChanges...)
}

// listEnvelope builds the standard paginated list response payload used across
// list handlers: {<key>: items, total, page, limit}.
func listEnvelope(key string, items, total any, pg Pagination) map[string]any {
	return map[string]any{
		key:     items,
		"total": total,
		"page":  pg.Page,
		"limit": pg.Limit,
	}
}

// parseDateRange parses start and end date strings in YYYY-MM-DD format.
// Applies end-of-day to the end date. Returns an error message suitable for
// display if parsing fails.
func parseDateRange(startStr, endStr string) (start, end time.Time, errMsg string) {
	var err error
	start, err = time.Parse("2006-01-02", startStr)
	if err != nil {
		return time.Time{}, time.Time{}, "Invalid start date format. Use YYYY-MM-DD"
	}
	end, err = time.Parse("2006-01-02", endStr)
	if err != nil {
		return time.Time{}, time.Time{}, "Invalid end date format. Use YYYY-MM-DD"
	}
	end = endOfDay(end)
	return start, end, ""
}

// phoneFromJID extracts the phone-number portion of a WhatsApp JID by stripping
// everything from the first "@" onward. For "628123456789@s.whatsapp.net" it
// returns "628123456789"; for a plain phone (no "@") it returns it unchanged.
// Mirrors gowa.PhoneFromJID for handler-layer use without importing the gowa pkg.
func phoneFromJID(jid string) string {
	if i := strings.IndexByte(jid, '@'); i >= 0 {
		return jid[:i]
	}
	return jid
}

// metadataString reads a string value from a JSONB (map[string]any) metadata
// field, returning "" when the key is missing or the value isn't a string.
// Centralizes the safe-access pattern so response builders don't each roll
// their own type-assertion boilerplate.
func metadataString(m models.JSONB, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
