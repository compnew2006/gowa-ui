package handlers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCampaignStartErrorNil(t *testing.T) {
	var e *campaignStartError
	assert.Equal(t, "", e.Error())
	assert.Nil(t, e.Unwrap())
}

func TestCampaignStartErrorMessage(t *testing.T) {
	e := &campaignStartError{message: "test error"}
	assert.Equal(t, "test error", e.Error())
}

func TestCampaignStartErrorUnwrap(t *testing.T) {
	inner := errors.New("inner")
	e := &campaignStartError{err: inner}
	assert.Equal(t, inner, e.Unwrap())
	assert.Equal(t, "inner", e.Error())
}

func TestCampaignStartErrorDefaults(t *testing.T) {
	e := &campaignStartError{}
	assert.Equal(t, "failed to start campaign", e.Error())
	assert.Nil(t, e.Unwrap())
}

func TestCampaignStartErrorKinds(t *testing.T) {
	assert.Equal(t, campaignStartErrorKind("bad_request"), campaignStartBadRequest)
	assert.Equal(t, campaignStartErrorKind("forbidden"), campaignStartForbidden)
	assert.Equal(t, campaignStartErrorKind("conflict"), campaignStartConflict)
	assert.Equal(t, campaignStartErrorKind("internal"), campaignStartInternal)
}

func TestCampaignStartResult(t *testing.T) {
	r := &campaignStartResult{status: "processing", enqueuedCount: 5}
	assert.Equal(t, 5, r.enqueuedCount)
}
