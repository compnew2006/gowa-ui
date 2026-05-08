package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserDisplayNameNormalizeBasic(t *testing.T) {
	assert.Equal(t, "John Doe", NormalizeDisplayText("John Doe", 80))
}

func TestUserDisplayNameNormalizeCollapsesWhitespace(t *testing.T) {
	assert.Equal(t, "a b c", NormalizeDisplayText("  a   b    c  ", 80))
}

func TestUserDisplayNameNormalizeTrims(t *testing.T) {
	assert.Equal(t, "hello", NormalizeDisplayText("  hello  ", 80))
}

func TestUserDisplayNameNormalizeEmpty(t *testing.T) {
	assert.Equal(t, "", NormalizeDisplayText("", 80))
	assert.Equal(t, "", NormalizeDisplayText("   ", 80))
}

func TestUserDisplayNameTruncation(t *testing.T) {
	result := NormalizeDisplayText("abcdefghij", 5)
	assert.Equal(t, "abcde...", result)
	assert.Len(t, result, 8)
}

func TestUserDisplayNameNoLimit(t *testing.T) {
	long := "a" + "b" + "c" + "d" + "e" + "f" + "g" + "h" + "i" + "j"
	assert.Equal(t, long, NormalizeDisplayText(long, 0))
}

func TestUserDisplayNameExactLimit(t *testing.T) {
	assert.Equal(t, "abc", NormalizeDisplayText("abc", 3))
}

func TestUserDisplayNameResolveNilApp(t *testing.T) {
	var a *App
	assert.Equal(t, "", a.ResolveUserDisplayName(uuid.Nil))
}
