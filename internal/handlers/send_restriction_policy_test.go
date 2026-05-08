package handlers

import (
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSendRestrictionPolicyStringifyUUIDs(t *testing.T) {
	ids := []uuid.UUID{
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"),
	}
	result := stringifyUUIDs(ids)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "550e8400-e29b-41d4-a716-446655440000")
}

func TestSendRestrictionPolicyContainsRestrictedNumber(t *testing.T) {
	numbers := []string{"+0987654321", "+1234567890"}
	sort.Strings(numbers)
	assert.True(t, containsRestrictedNumber(numbers, "+1234567890"))
	assert.True(t, containsRestrictedNumber(numbers, "+0987654321"))
	assert.False(t, containsRestrictedNumber(numbers, "+1111111111"))
	assert.False(t, containsRestrictedNumber(nil, "+1234567890"))
	assert.False(t, containsRestrictedNumber([]string{}, "+1234567890"))
}
