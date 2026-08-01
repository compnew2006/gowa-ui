package gowa_test

import (
	"encoding/json"
	"testing"

	"github.com/shridarpatil/gowa-ui/pkg/gowa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionPayload_Unmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		wantEvent   string
		wantReason  string
		wantConnect bool
	}{
		{
			name:        "connected",
			body:        `{"event":"connected","is_connected":true}`,
			wantEvent:   gowa.ConnectionConnected,
			wantConnect: true,
		},
		{
			name:       "disconnected with reason",
			body:       `{"event":"disconnected","reason":"network_error","is_connected":false}`,
			wantEvent:  gowa.ConnectionDisconnected,
			wantReason: "network_error",
		},
		{
			name:      "connecting",
			body:      `{"event":"connecting","is_connected":false}`,
			wantEvent: gowa.ConnectionConnecting,
		},
		{
			name:      "logout",
			body:      `{"event":"logout","is_connected":false}`,
			wantEvent: gowa.ConnectionLogout,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var conn gowa.ConnectionPayload
			require.NoError(t, json.Unmarshal([]byte(tc.body), &conn))
			assert.Equal(t, tc.wantEvent, conn.Event)
			assert.Equal(t, tc.wantReason, conn.Reason)
			assert.Equal(t, tc.wantConnect, conn.IsConnected)
		})
	}
}

func TestWebhookPayload_ConnectionEvent(t *testing.T) {
	t.Parallel()
	body := `{
		"event": "connection",
		"device_id": "628123456789@s.whatsapp.net",
		"timestamp": "2026-07-11T12:00:00Z",
		"payload": {
			"event": "connected",
			"is_connected": true
		}
	}`

	var envelope gowa.WebhookPayload
	require.NoError(t, json.Unmarshal([]byte(body), &envelope))
	assert.Equal(t, "connection", envelope.Event)
	assert.Equal(t, "2026-07-11T12:00:00Z", envelope.Timestamp)

	var conn gowa.ConnectionPayload
	require.NoError(t, json.Unmarshal(envelope.Payload, &conn))
	assert.Equal(t, gowa.ConnectionConnected, conn.Event)
	assert.True(t, conn.IsConnected)
}
