package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// errEnvelopeSent is a sentinel returned by helpers after they have already
// written an error envelope to the response. Callers should return nil to the framework.
var errEnvelopeSent = errors.New("error envelope sent")

// parsePathUUID extracts a UUID from a path parameter.
//
// On success, returns the parsed UUID and nil error.
//
// On failure (invalid UUID format):
//   - Sends HTTP 400 error envelope to response: "Invalid <label> ID"
//   - Returns uuid.Nil and errEnvelopeSent (sentinel error)
//
// The errEnvelopeSent sentinel indicates that an error response has already
// been written to the HTTP response. Callers should return nil to the framework
// without sending another error response.
//
// Parameters:
//   - r: The HTTP request containing the path parameter
//   - param: The request context key for the path parameter (e.g., "id", "user_id")
//   - label: Human-readable label for the parameter (e.g., "user", "organization")
//
// Returns:
//   - uuid.UUID: The parsed UUID, or uuid.Nil on error
//   - error: nil on success, errEnvelopeSent on failure (error already sent to client)
func parsePathUUID(r *fastglue.Request, param, label string) (uuid.UUID, error) {
	idStr, _ := r.RequestCtx.UserValue(param).(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid "+label+" ID", nil, "")
		return uuid.Nil, errEnvelopeSent
	}
	return id, nil
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
//
// On success, returns a pointer to the fetched model and nil error.
//
// On failure (record not found or database error):
//   - Sends HTTP 404 error envelope to response: "<label> not found"
//   - Returns nil pointer and errEnvelopeSent (sentinel error)
//
// The errEnvelopeSent sentinel indicates that an error response has already
// been written to the HTTP response. Callers should return nil to the framework
// without sending another error response.
//
// Type Parameters:
//   - T: The model type to fetch (must have ID and OrganizationID fields)
//
// Parameters:
//   - db: GORM database instance
//   - r: The HTTP request (for error response)
//   - id: The UUID of the record to fetch
//   - orgID: The organization ID to scope the query
//   - label: Human-readable label for the resource (e.g., "User", "Organization")
//
// Returns:
//   - *T: Pointer to the fetched model on success, nil on failure
//   - error: nil on success, errEnvelopeSent on failure (error already sent to client)
func findByIDAndOrg[T any](db *gorm.DB, r *fastglue.Request, id, orgID uuid.UUID, label string) (*T, error) {
	var model T
	if err := db.Where("id = ? AND organization_id = ?", id, orgID).First(&model).Error; err != nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusNotFound, label+" not found", nil, "")
		return nil, errEnvelopeSent
	}
	return &model, nil
}

// parseDateRange parses start and end date strings in YYYY-MM-DD format.
// Applies end-of-day to the end date.
//
// Returns:
//   - start, end: Parsed dates
//   - errMsg: Empty string on success, error message on failure (e.g., "Invalid start date format. Use YYYY-MM-DD")
//
// On error (parsing failure), both start and end will be zero values and errMsg
// will contain a user-friendly error message suitable for API responses.
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

// validateExportColumns validates requested columns against allowed columns.
// Returns error if any requested column is not in the allowed set.
//
// This is a pure validation function extracted from ExportData to improve
// testability. It validates that all requested columns are present in the
// allowed set, returning both the validated list and an error if validation fails.
//
// Parameters:
//   - requested: List of column names to export
//   - allowed: List of allowed column names from config
//
// Returns:
//   - validated: List of validated columns (subset of requested, preserving order)
//   - error: Non-nil if any column is not allowed, with descriptive error message
//
// Error returns:
//   - error: Contains message like "column 'name' is not allowed for export"
func validateExportColumns(requested, allowed []string) (validated []string, err error) {
	// Build allowed set for O(1) lookup
	allowedSet := make(map[string]bool, len(allowed))
	for _, col := range allowed {
		allowedSet[col] = true
	}

	// Validate each requested column
	validated = make([]string, 0, len(requested))
	for _, col := range requested {
		if !allowedSet[col] {
			return nil, fmt.Errorf("column '%s' is not allowed for export", col)
		}
		validated = append(validated, col)
	}
	return validated, nil
}

// validateRequiredColumns validates that all required columns exist in the provided column index.
//
// This is a pure validation function extracted from ImportData to improve testability.
// It performs case-insensitive matching and handles underscore/space variations (e.g.,
// "phone_number" matches "phone number" and vice versa).
//
// Parameters:
//   - colIndex: Map of column names (lowercase or mapped) to their CSV indices
//   - required: List of required column names that must be present
//
// Returns:
//   - error: Non-nil if any required column is missing, with descriptive error message
//
// Error returns:
//   - error: Contains message like "Required column 'phone_number' not found in CSV"
func validateRequiredColumns(colIndex map[string]int, required []string) error {
	for _, reqCol := range required {
		found := false
		for col := range colIndex {
			// Case-insensitive match with underscore/space normalization (both directions)
			// Example: "phone_number" matches "phone number" and vice versa
			if strings.EqualFold(col, reqCol) ||
				strings.EqualFold(col, strings.ReplaceAll(reqCol, "_", " ")) ||
				strings.EqualFold(col, strings.ReplaceAll(reqCol, " ", "_")) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Required column '%s' not found in CSV", reqCol)
		}
	}
	return nil
}
