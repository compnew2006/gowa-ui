package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestParseAnalyticsInstanceID_EmptyInput(t *testing.T) {
	t.Parallel()

	app := &App{}
	result, err := app.parseAnalyticsInstanceID(uuid.New(), "   ")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestParseAnalyticsInstanceID_InvalidUUID(t *testing.T) {
	t.Parallel()

	app := &App{}
	result, err := app.parseAnalyticsInstanceID(uuid.New(), "not-a-uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "valid UUID")
	assert.Nil(t, result)
}

func TestParseAnalyticsInstanceID_NilDBForValidUUID(t *testing.T) {
	t.Parallel()

	app := &App{}
	result, err := app.parseAnalyticsInstanceID(uuid.New(), uuid.NewString())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lookup is unavailable")
	assert.Nil(t, result)
}

func TestApplyTransferAnalyticsInstanceFilter_NilQuery(t *testing.T) {
	t.Parallel()

	instanceID := uuid.New()
	result := applyTransferAnalyticsInstanceFilter(nil, uuid.New(), &instanceID)
	assert.Nil(t, result)
}

func TestApplyRatingAnalyticsInstanceFilter_NilQuery(t *testing.T) {
	t.Parallel()

	instanceID := uuid.New()
	result := applyRatingAnalyticsInstanceFilter(nil, &instanceID, "contacts")
	assert.Nil(t, result)
}

func TestApplyRatingAnalyticsInstanceFilter_NilInstanceID(t *testing.T) {
	t.Parallel()

	result := applyRatingAnalyticsInstanceFilter(nil, nil, "contacts")
	assert.Nil(t, result)
}
