package gowa

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestChatInfo_UnmarshalJSON(t *testing.T) {
	// 1. JSON with empty time strings
	inputEmpty := `{
		"jid": "123@s.whatsapp.net",
		"name": "Test Chat",
		"last_message_time": "",
		"ephemeral_expiration": 0,
		"created_at": "",
		"updated_at": ""
	}`

	var chatEmpty ChatInfo
	err := json.Unmarshal([]byte(inputEmpty), &chatEmpty)
	assert.NoError(t, err)
	assert.Equal(t, "123@s.whatsapp.net", chatEmpty.JID)
	assert.Equal(t, "Test Chat", chatEmpty.Name)
	assert.Nil(t, chatEmpty.LastMessageTime)
	assert.True(t, chatEmpty.CreatedAt.IsZero())
	assert.True(t, chatEmpty.UpdatedAt.IsZero())

	// 2. JSON with valid times
	inputValid := `{
		"jid": "123@s.whatsapp.net",
		"name": "Test Chat",
		"last_message_time": "2026-07-05T15:00:00Z",
		"created_at": "2026-07-05T14:00:00Z",
		"updated_at": "2026-07-05T14:30:00Z"
	}`

	var chatValid ChatInfo
	err = json.Unmarshal([]byte(inputValid), &chatValid)
	assert.NoError(t, err)
	assert.NotNil(t, chatValid.LastMessageTime)
	assert.Equal(t, "2026-07-05T15:00:00Z", chatValid.LastMessageTime.Format(time.RFC3339))
	assert.Equal(t, "2026-07-05T14:00:00Z", chatValid.CreatedAt.Format(time.RFC3339))
	assert.Equal(t, "2026-07-05T14:30:00Z", chatValid.UpdatedAt.Format(time.RFC3339))
}

func TestMessageInfo_UnmarshalJSON(t *testing.T) {
	// 1. JSON with empty time strings
	inputEmpty := `{
		"id": "msg-123",
		"chat_jid": "123@s.whatsapp.net",
		"timestamp": "",
		"created_at": "",
		"updated_at": ""
	}`

	var msgEmpty MessageInfo
	err := json.Unmarshal([]byte(inputEmpty), &msgEmpty)
	assert.NoError(t, err)
	assert.Equal(t, "msg-123", msgEmpty.ID)
	assert.True(t, msgEmpty.Timestamp.IsZero())
	assert.True(t, msgEmpty.CreatedAt.IsZero())
	assert.True(t, msgEmpty.UpdatedAt.IsZero())

	// 2. JSON with valid times
	inputValid := `{
		"id": "msg-123",
		"chat_jid": "123@s.whatsapp.net",
		"timestamp": "2026-07-05T15:00:00Z",
		"created_at": "2026-07-05T14:00:00Z",
		"updated_at": "2026-07-05T14:30:00Z"
	}`

	var msgValid MessageInfo
	err = json.Unmarshal([]byte(inputValid), &msgValid)
	assert.NoError(t, err)
	assert.Equal(t, "2026-07-05T15:00:00Z", msgValid.Timestamp.Format(time.RFC3339))
	assert.Equal(t, "2026-07-05T14:00:00Z", msgValid.CreatedAt.Format(time.RFC3339))
	assert.Equal(t, "2026-07-05T14:30:00Z", msgValid.UpdatedAt.Format(time.RFC3339))
}
