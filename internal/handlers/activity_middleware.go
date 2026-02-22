package handlers

import (
	"strings"

	"github.com/google/uuid"
	"github.com/zerodha/fastglue"
)

func parseContextUUID(value any) (uuid.UUID, bool) {
	switch v := value.(type) {
	case uuid.UUID:
		return v, true
	case string:
		id, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, false
		}
		return id, true
	default:
		return uuid.Nil, false
	}
}

// ActivityLogMiddleware captures authenticated API interactions.
func (a *App) ActivityLogMiddleware() fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		path := string(r.RequestCtx.Path())
		if !strings.HasPrefix(path, "/api") {
			return r
		}

		// Avoid recursive noise from reading/creating the activity logs themselves.
		if path == "/api/activity-logs" || strings.HasPrefix(path, "/api/activity-logs/") {
			return r
		}

		userID, ok := parseContextUUID(r.RequestCtx.UserValue("user_id"))
		if !ok || userID == uuid.Nil {
			return r
		}

		orgID, err := a.getOrgID(r)
		if err != nil || orgID == uuid.Nil {
			return r
		}

		statusCode := r.RequestCtx.Response.StatusCode()
		a.LogSystemInteraction(r, userID, orgID, statusCode)
		return r
	}
}
