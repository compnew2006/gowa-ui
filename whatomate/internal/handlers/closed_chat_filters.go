package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

type closedChatFilters struct {
	ClosedBy   string
	ClosedFrom *time.Time
	ClosedTo   *time.Time
}

func parseClosedChatFilters(r *fastglue.Request) (closedChatFilters, string, error) {
	filters := closedChatFilters{
		ClosedBy: strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("closed_by"))),
	}

	if raw := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("closed_from"))); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return filters, "closed_from", fmt.Errorf("invalid closed_from format. Use YYYY-MM-DD")
		}
		filters.ClosedFrom = &parsed
	}

	if raw := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("closed_to"))); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return filters, "closed_to", fmt.Errorf("invalid closed_to format. Use YYYY-MM-DD")
		}
		filters.ClosedTo = &parsed
	}

	if filters.ClosedFrom != nil && filters.ClosedTo != nil && filters.ClosedFrom.After(*filters.ClosedTo) {
		return filters, "closed_to", fmt.Errorf("closed_to must be on or after closed_from")
	}

	return filters, "", nil
}

func applyClosedChatFilters(query *gorm.DB, search string, filters closedChatFilters) *gorm.DB {
	trimmedSearch := strings.TrimSpace(search)
	trimmedClosedBy := strings.TrimSpace(filters.ClosedBy)

	if trimmedSearch != "" || trimmedClosedBy != "" {
		query = query.
			Joins("LEFT JOIN users closed_by_users ON closed_by_users.id = contacts.closed_by_user_id").
			Joins("LEFT JOIN users assigned_users ON assigned_users.id = contacts.assigned_user_id")
	}

	if trimmedSearch != "" {
		searchPattern := "%" + trimmedSearch + "%"
		query = query.Where(
			"contacts.phone_number LIKE ? OR contacts.profile_name ILIKE ? OR closed_by_users.full_name ILIKE ? OR assigned_users.full_name ILIKE ?",
			searchPattern,
			searchPattern,
			searchPattern,
			searchPattern,
		)
	}

	if trimmedClosedBy != "" {
		closedByPattern := "%" + trimmedClosedBy + "%"
		query = query.Where(
			"contacts.closed_by_user_id::text = ? OR contacts.assigned_user_id::text = ? OR closed_by_users.full_name ILIKE ? OR assigned_users.full_name ILIKE ?",
			trimmedClosedBy,
			trimmedClosedBy,
			closedByPattern,
			closedByPattern,
		)
	}

	if filters.ClosedFrom != nil {
		query = query.Where("COALESCE(contacts.closed_at, contacts.updated_at) >= ?", *filters.ClosedFrom)
	}
	if filters.ClosedTo != nil {
		query = query.Where("COALESCE(contacts.closed_at, contacts.updated_at) <= ?", endOfDay(*filters.ClosedTo))
	}

	return query
}
