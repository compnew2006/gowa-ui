package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrateDBMigrationOptionsDefaults(t *testing.T) {
	opts := MigrationOptions{}
	assert.False(t, opts.DryRun)
	assert.Equal(t, 0, opts.BatchSize)
	assert.False(t, opts.IncludeEnc2)
}

func TestMigrateDBMigrationTargets(t *testing.T) {
	assert.Len(t, migrationTargets, 4)

	tables := make(map[string]int)
	for _, target := range migrationTargets {
		tables[target.Table]++
	}
	assert.Contains(t, tables, "whatsapp_accounts")
	assert.Contains(t, tables, "chatbot_settings")
	assert.Contains(t, tables, "sso_providers")
	assert.Equal(t, 2, tables["whatsapp_accounts"])
}

func TestMigrateDBUpgradeCiphertextSkipsEmpty(t *testing.T) {
	result, changed, err := UpgradeCiphertext("", "key", false)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, "", result)
}

func TestMigrateDBUpgradeCiphertextSkipsV3(t *testing.T) {
	v3 := "enc3:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	result, changed, err := UpgradeCiphertext(v3, "anykey", false)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, v3, result)
}

func TestMigrateDBUpgradeCiphertextSkipsNonLegacy(t *testing.T) {
	result, changed, err := UpgradeCiphertext("random_string_not_legacy", "key", false)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, "random_string_not_legacy", result)
}

func TestMigrateDBUpgradeCiphertextSkipsWhitespace(t *testing.T) {
	result, changed, err := UpgradeCiphertext("   ", "key", false)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, "   ", result)
}
