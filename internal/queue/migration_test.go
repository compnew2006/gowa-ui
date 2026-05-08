package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrationAppendSampleRespectsLimit(t *testing.T) {
	var samples []string
	for i := 0; i < 15; i++ {
		appendMigrationSample(&samples, "msg")
	}
	assert.Len(t, samples, maxMigrationSamples)
}

func TestMigrationCloneLegacyValuesNilSafe(t *testing.T) {
	result := cloneLegacyValues(nil)
	assert.NotNil(t, result)
	assert.Empty(t, result)

	result = cloneLegacyValues(map[string]any{})
	assert.NotNil(t, result)
	assert.Empty(t, result)
}
