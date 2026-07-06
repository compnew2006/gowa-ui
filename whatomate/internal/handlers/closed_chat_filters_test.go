package handlers_test

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
)

// TestParseClosedChatFilters_NoFilters tests parsing with no query parameters
func TestParseClosedChatFilters_NoFilters(t *testing.T) {
	t.Parallel()

	req := testutil.NewRequest(t)

	filters, fieldErr, err := handlers.ParseClosedChatFilters(req)

	assert.Empty(t, filters.ClosedBy)
	assert.Nil(t, filters.ClosedFrom)
	assert.Nil(t, filters.ClosedTo)
	assert.Empty(t, fieldErr)
	assert.NoError(t, err)
}

// TestParseClosedChatFilters_OnlyClosedBy tests parsing with only closed_by parameter
func TestParseClosedChatFilters_OnlyClosedBy(t *testing.T) {
	t.Parallel()

	req := testutil.NewRequest(t)
	testutil.SetQueryParam(req, "closed_by", "john.doe")

	filters, _, err := handlers.ParseClosedChatFilters(req)

	assert.Equal(t, "john.doe", filters.ClosedBy)
	assert.Nil(t, filters.ClosedFrom)
	assert.Nil(t, filters.ClosedTo)
	assert.NoError(t, err)
}

// TestParseClosedChatFilters_ValidDateRanges tests parsing with valid date ranges
func TestParseClosedChatFilters_ValidDateRanges(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		closedFrom string
		closedTo   string
	}{
		{"Both dates", "2024-01-01", "2024-01-31"},
		{"Only from", "2024-01-01", ""},
		{"Only to", "", "2024-01-31"},
		{"Same date", "2024-01-15", "2024-01-15"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := testutil.NewRequest(t)
			if tc.closedFrom != "" {
				testutil.SetQueryParam(req, "closed_from", tc.closedFrom)
			}
			if tc.closedTo != "" {
				testutil.SetQueryParam(req, "closed_to", tc.closedTo)
			}

			filters, _, err := handlers.ParseClosedChatFilters(req)
			assert.NoError(t, err)

			if tc.closedFrom != "" {
				assert.NotNil(t, filters.ClosedFrom)
				expectedDate, _ := time.Parse("2006-01-02", tc.closedFrom)
				assert.Equal(t, expectedDate, *filters.ClosedFrom)
			}
			if tc.closedTo != "" {
				assert.NotNil(t, filters.ClosedTo)
				expectedDate, _ := time.Parse("2006-01-02", tc.closedTo)
				assert.Equal(t, expectedDate, *filters.ClosedTo)
			}
		})
	}
}

// TestParseClosedChatFilters_InvalidDateFormat tests error handling for invalid date formats
func TestParseClosedChatFilters_InvalidDateFormat(t *testing.T) {
	t.Parallel()

	req := testutil.NewRequest(t)
	testutil.SetQueryParam(req, "closed_from", "01/01/2024") // Wrong format

	filters, fieldErr, err := handlers.ParseClosedChatFilters(req)

	assert.Error(t, err)
	assert.Equal(t, "closed_from", fieldErr)
	assert.Contains(t, err.Error(), "invalid")
	_ = filters
}

// TestParseClosedChatFilters_DateRangeValidation tests that closed_to >= closed_from
func TestParseClosedChatFilters_DateRangeValidation(t *testing.T) {
	t.Parallel()

	req := testutil.NewRequest(t)
	testutil.SetQueryParam(req, "closed_from", "2024-01-31")
	testutil.SetQueryParam(req, "closed_to", "2024-01-01")

	filters, fieldErr, err := handlers.ParseClosedChatFilters(req)

	assert.Error(t, err)
	assert.Equal(t, "closed_to", fieldErr)
	assert.Contains(t, err.Error(), "on or after")
	_ = filters
}

// TestApplyClosedChatFilters_NoFilters tests applying filters without any filters set
func TestApplyClosedChatFilters_NoFilters(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	query := app.DB

	filters := handlers.ClosedChatFilters{}
	resultQuery := handlers.ApplyClosedChatFilters(query, "", filters)

	assert.NotNil(t, resultQuery)
}

// TestApplyClosedChatFilters_AllFilters tests applying all filters
func TestApplyClosedChatFilters_AllFilters(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	query := app.DB

	fromDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	filters := handlers.ClosedChatFilters{
		ClosedBy:   "john.doe",
		ClosedFrom: &fromDate,
		ClosedTo:   &toDate,
	}

	resultQuery := handlers.ApplyClosedChatFilters(query, "search", filters)
	assert.NotNil(t, resultQuery)
}
