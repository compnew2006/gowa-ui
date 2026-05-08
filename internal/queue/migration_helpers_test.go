package queue

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestCloneLegacyValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   map[string]interface{}
		len  int
		key  string
		val  interface{}
	}{
		{
			name: "nil returns empty map",
			in:   nil,
			len:  0,
		},
		{
			name: "empty returns empty map",
			in:   map[string]interface{}{},
			len:  0,
		},
		{
			name: "populated returns copy with same length",
			in: map[string]interface{}{
				"type":    "recipient",
				"payload": `{"organization_id":"550e8400-e29b-41d4-a716-446655440000"}`,
			},
			len: 2,
			key: "type",
			val: "recipient",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := cloneLegacyValues(tt.in)
			assert.Len(t, result, tt.len)
			if tt.key != "" {
				assert.Equal(t, tt.val, result[tt.key])
			}
		})
	}

	t.Run("modifying copy does not affect original", func(t *testing.T) {
		t.Parallel()
		original := map[string]interface{}{"key1": "val1", "key2": 42}
		cloned := cloneLegacyValues(original)
		cloned["key1"] = "modified"
		cloned["key3"] = "new"
		assert.Equal(t, "val1", original["key1"])
		assert.NotContains(t, original, "key3")
	})
}

func TestAppendMigrationSample(t *testing.T) {
	t.Parallel()

	t.Run("empty slice appends", func(t *testing.T) {
		t.Parallel()
		var dst []string
		appendMigrationSample(&dst, "msg1")
		assert.Equal(t, []string{"msg1"}, dst)
	})

	t.Run("below limit appends", func(t *testing.T) {
		t.Parallel()
		dst := make([]string, 0, maxMigrationSamples)
		for i := 0; i < maxMigrationSamples-1; i++ {
			appendMigrationSample(&dst, "msg")
		}
		appendMigrationSample(&dst, "last")
		assert.Len(t, dst, maxMigrationSamples)
		assert.Equal(t, "last", dst[maxMigrationSamples-1])
	})

	t.Run("at limit does not add", func(t *testing.T) {
		t.Parallel()
		dst := make([]string, maxMigrationSamples)
		initialLen := len(dst)
		appendMigrationSample(&dst, "overflow")
		assert.Len(t, dst, initialLen)
	})

	t.Run("above limit does not add", func(t *testing.T) {
		t.Parallel()
		dst := make([]string, maxMigrationSamples+5)
		initialLen := len(dst)
		appendMigrationSample(&dst, "overflow")
		assert.Len(t, dst, initialLen)
	})
}

func TestLegacyCampaignMessageOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	recipientPayload, _ := json.Marshal(RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: orgID,
		PhoneNumber:    "1234567890",
	})

	contactRepairPayload, _ := json.Marshal(ContactRepairJob{
		ContactID:      uuid.New(),
		OrganizationID: orgID,
		ConversationID: "1234567890@s.whatsapp.net",
	})

	nilOrgRecipientPayload, _ := json.Marshal(RecipientJob{
		CampaignID:     uuid.New(),
		RecipientID:    uuid.New(),
		OrganizationID: uuid.Nil,
		PhoneNumber:    "1234567890",
	})

	tests := []struct {
		name        string
		msg         redis.XMessage
		expectID    uuid.UUID
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing type field returns error",
			msg: redis.XMessage{
				Values: map[string]interface{}{
					"payload": string(recipientPayload),
				},
			},
			expectError: true,
			errorMsg:    "missing type",
		},
		{
			name: "missing payload returns error",
			msg: redis.XMessage{
				Values: map[string]interface{}{
					"type": string(JobTypeRecipient),
				},
			},
			expectError: true,
			errorMsg:    "missing payload",
		},
		{
			name: "invalid JSON returns error",
			msg: redis.XMessage{
				Values: map[string]interface{}{
					"type":    string(JobTypeRecipient),
					"payload": "not-json",
				},
			},
			expectError: true,
			errorMsg:    "decode legacy recipient job",
		},
		{
			name: "valid RecipientJob returns org ID",
			msg: redis.XMessage{
				Values: map[string]interface{}{
					"type":    string(JobTypeRecipient),
					"payload": string(recipientPayload),
				},
			},
			expectID: orgID,
		},
		{
			name: "valid ContactRepairJob returns org ID",
			msg: redis.XMessage{
				Values: map[string]interface{}{
					"type":    string(JobTypeContactRepair),
					"payload": string(contactRepairPayload),
				},
			},
			expectID: orgID,
		},
		{
			name: "nil org ID in RecipientJob returns error",
			msg: redis.XMessage{
				Values: map[string]interface{}{
					"type":    string(JobTypeRecipient),
					"payload": string(nilOrgRecipientPayload),
				},
			},
			expectError: true,
			errorMsg:    "missing organization_id",
		},
		{
			name: "unsupported job type returns error",
			msg: redis.XMessage{
				Values: map[string]interface{}{
					"type":    "unknown_type",
					"payload": string(recipientPayload),
				},
			},
			expectError: true,
			errorMsg:    "unsupported legacy job type",
		},
		{
			name: "empty type value returns error",
			msg: redis.XMessage{
				Values: map[string]interface{}{
					"type":    "",
					"payload": string(recipientPayload),
				},
			},
			expectError: true,
			errorMsg:    "missing type",
		},
		{
			name: "empty payload value returns error",
			msg: redis.XMessage{
				Values: map[string]interface{}{
					"type":    string(JobTypeRecipient),
					"payload": "",
				},
			},
			expectError: true,
			errorMsg:    "missing payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resultID, err := legacyCampaignMessageOrganizationID(tt.msg)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Equal(t, uuid.Nil, resultID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectID, resultID)
			}
		})
	}
}
