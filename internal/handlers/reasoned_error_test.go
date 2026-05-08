package handlers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReasonedErrorMessage(t *testing.T) {
	e := newReasonedError("something failed", "E001", "fallback msg")
	assert.Equal(t, "something failed", e.Error())
	assert.Equal(t, "E001", e.reasonCode)
}

func TestReasonedErrorEmptyMessageUsesFallback(t *testing.T) {
	e := newReasonedError("", "E002", "fallback msg")
	assert.Equal(t, "fallback msg", e.Error())
}

func TestReasonedErrorWhitespaceMessageUsesFallback(t *testing.T) {
	e := newReasonedError("   ", "E003", "fallback msg")
	assert.Equal(t, "fallback msg", e.Error())
}

func TestReasonedErrorAsSuccess(t *testing.T) {
	e := newReasonedError("bad", "E004", "fb")
	got, ok := asReasonedError(e)
	assert.True(t, ok)
	assert.Equal(t, "bad", got.message)
	assert.Equal(t, "E004", got.reasonCode)
}

func TestReasonedErrorAsNil(t *testing.T) {
	got, ok := asReasonedError(nil)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestReasonedErrorAsNonReasoned(t *testing.T) {
	got, ok := asReasonedError(errors.New("plain error"))
	assert.False(t, ok)
	assert.Nil(t, got)
}
