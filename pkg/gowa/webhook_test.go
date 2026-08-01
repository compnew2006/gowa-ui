package gowa_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/shridarpatil/gowa-ui/pkg/gowa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hmacSHA256 computes the lowercase hex HMAC-SHA256 of body using secret.
func hmacSHA256(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestPhoneFromJID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		jid  string
		want string
	}{
		{"individual JID", "628123456789@s.whatsapp.net", "628123456789"},
		{"group JID", "120363402106XXXXX@g.us", "120363402106XXXXX"},
		{"LID", "251556368777322@lid", "251556368777322"},
		{"plain phone (no @)", "16505551234", "16505551234"},
		{"empty string", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, gowa.PhoneFromJID(tc.jid))
		})
	}
}

func TestResolveMediaField_StringForm(t *testing.T) {
	t.Parallel()
	// Auto-download ON, no caption → plain string (path)
	raw := json.RawMessage(`"statics/media/photo.jpeg"`)
	mf := gowa.ResolveMediaField(raw)
	assert.Equal(t, "statics/media/photo.jpeg", mf.URL)
	assert.Empty(t, mf.Caption)
	assert.Empty(t, mf.Filename)
}

func TestResolveMediaField_ObjectWithCaption(t *testing.T) {
	t.Parallel()
	// Auto-download ON with caption → object with path + caption
	raw := json.RawMessage(`{"path":"statics/media/photo.jpeg","caption":"Check this out!"}`)
	mf := gowa.ResolveMediaField(raw)
	assert.Equal(t, "statics/media/photo.jpeg", mf.URL)
	assert.Equal(t, "Check this out!", mf.Caption)
}

func TestResolveMediaField_ObjectWithURL(t *testing.T) {
	t.Parallel()
	// Auto-download OFF → object with url + caption
	raw := json.RawMessage(`{"url":"https://mmg.whatsapp.net/b64...","caption":"See this"}`)
	mf := gowa.ResolveMediaField(raw)
	assert.Equal(t, "https://mmg.whatsapp.net/b64...", mf.URL)
	assert.Equal(t, "See this", mf.Caption)
}

func TestResolveMediaField_DocumentWithFilename(t *testing.T) {
	t.Parallel()
	// Document with filename (auto-download OFF)
	raw := json.RawMessage(`{"url":"https://example.com/report.pdf","filename":"report.pdf"}`)
	mf := gowa.ResolveMediaField(raw)
	assert.Equal(t, "https://example.com/report.pdf", mf.URL)
	assert.Equal(t, "report.pdf", mf.Filename)
}

func TestResolveMediaField_EmptyInput(t *testing.T) {
	t.Parallel()
	mf := gowa.ResolveMediaField(nil)
	assert.Empty(t, mf.URL)
	assert.Empty(t, mf.Caption)
}

func TestResolveMediaField_InvalidJSON(t *testing.T) {
	t.Parallel()
	mf := gowa.ResolveMediaField(json.RawMessage(`{invalid`))
	assert.Empty(t, mf.URL)
}

func TestMessagePayload_IsGroup(t *testing.T) {
	t.Parallel()

	t.Run("group chat", func(t *testing.T) {
		mp := gowa.MessagePayload{ChatID: "120363402106@g.us"}
		assert.True(t, mp.IsGroup())
	})
	t.Run("individual chat", func(t *testing.T) {
		mp := gowa.MessagePayload{ChatID: "628123456789@s.whatsapp.net"}
		assert.False(t, mp.IsGroup())
	})
}

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	t.Parallel()
	body := []byte(`{"event":"message","device_id":"dev1"}`)
	secret := "mysecret"

	// Compute correct HMAC.
	mac := hmacSHA256(body, secret)
	header := "sha256=" + mac

	assert.True(t, gowa.VerifyWebhookSignature(body, header, secret))
}

func TestVerifyWebhookSignature_InvalidSecret(t *testing.T) {
	t.Parallel()
	body := []byte(`{"event":"message"}`)
	correctSig := "sha256=" + hmacSHA256(body, "correct-secret")
	assert.False(t, gowa.VerifyWebhookSignature(body, correctSig, "wrong-secret"))
}

func TestVerifyWebhookSignature_TamperedBody(t *testing.T) {
	t.Parallel()
	body := []byte(`{"event":"message","device_id":"dev1"}`)
	sig := "sha256=" + hmacSHA256(body, "secret")
	tamperedBody := []byte(`{"event":"message","device_id":"dev2"}`)
	assert.False(t, gowa.VerifyWebhookSignature(tamperedBody, sig, "secret"))
}

func TestVerifyWebhookSignature_EmptyHeader(t *testing.T) {
	t.Parallel()
	assert.False(t, gowa.VerifyWebhookSignature([]byte("body"), "", "secret"))
}

func TestVerifyWebhookSignature_EmptySecret(t *testing.T) {
	t.Parallel()
	assert.False(t, gowa.VerifyWebhookSignature([]byte("body"), "sha256=abc", ""))
}

func TestVerifyWebhookSignature_WrongPrefix(t *testing.T) {
	t.Parallel()
	assert.False(t, gowa.VerifyWebhookSignature([]byte("body"), "sha1=abc", "secret"))
}

func TestParseTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("valid RFC3339", func(t *testing.T) {
		ts := gowa.ParseTimestamp("2023-10-15T10:30:00Z")
		assert.False(t, ts.IsZero())
	})
	t.Run("invalid", func(t *testing.T) {
		ts := gowa.ParseTimestamp("invalid-timestamp")
		assert.True(t, ts.IsZero())
	})
	t.Run("empty", func(t *testing.T) {
		ts := gowa.ParseTimestamp("")
		assert.True(t, ts.IsZero())
	})
}

func TestWebhookPayload_Unmarshal(t *testing.T) {
	t.Parallel()
	body := `{
		"event": "message",
		"device_id": "628123456789@s.whatsapp.net",
		"session_id": "org_2",
		"payload": {
			"id": "3EB0C127",
			"chat_id": "628123456789@s.whatsapp.net",
			"from": "628987654321@s.whatsapp.net",
			"from_name": "John Doe",
			"timestamp": "2023-10-15T10:30:00Z",
			"is_from_me": false,
			"body": "Hello"
		}
	}`

	var envelope gowa.WebhookPayload
	require.NoError(t, json.Unmarshal([]byte(body), &envelope))

	assert.Equal(t, "message", envelope.Event)
	assert.Equal(t, "628123456789@s.whatsapp.net", envelope.DeviceID)
	assert.Equal(t, "org_2", envelope.SessionID)

	var msg gowa.MessagePayload
	require.NoError(t, json.Unmarshal(envelope.Payload, &msg))
	assert.Equal(t, "3EB0C127", msg.ID)
	assert.Equal(t, "Hello", msg.Body)
	assert.Equal(t, "John Doe", msg.FromName)
	assert.False(t, msg.IsFromMe)
}

func TestWebhookPayload_AckTimestampAtTopLevel(t *testing.T) {
	t.Parallel()
	body := `{
		"event": "message.ack",
		"device_id": "628123456789@s.whatsapp.net",
		"timestamp": "2025-07-18T22:44:20Z",
		"payload": {
			"ids": ["3EB00106E8BE0F407E88EC"],
			"chat_id": "120363402106@g.us",
			"receipt_type": "delivered"
		}
	}`

	var envelope gowa.WebhookPayload
	require.NoError(t, json.Unmarshal([]byte(body), &envelope))

	// timestamp is at top level for ack events
	assert.Equal(t, "2025-07-18T22:44:20Z", envelope.Timestamp)

	var ack gowa.AckPayload
	require.NoError(t, json.Unmarshal(envelope.Payload, &ack))
	assert.Equal(t, "delivered", ack.ReceiptType)
	require.Len(t, ack.IDs, 1)
	assert.Equal(t, "3EB00106E8BE0F407E88EC", ack.IDs[0])
}

func TestCheckReplay(t *testing.T) {
	t.Parallel()
	maxAge := 5 * time.Minute

	tests := []struct {
		name      string
		timestamp string
		expect    bool
	}{
		{
			name:      "fresh RFC3339 timestamp (now)",
			timestamp: time.Now().UTC().Format(time.RFC3339),
			expect:    true,
		},
		{
			name:      "fresh epoch timestamp (now)",
			timestamp: fmt.Sprintf("%d", time.Now().Unix()),
			expect:    true,
		},
		{
			name:      "stale RFC3339 (6 minutes ago — beyond 5 min window)",
			timestamp: time.Now().Add(-6 * time.Minute).UTC().Format(time.RFC3339),
			expect:    false,
		},
		{
			name:      "stale epoch (6 minutes ago)",
			timestamp: fmt.Sprintf("%d", time.Now().Add(-6*time.Minute).Unix()),
			expect:    false,
		},
		{
			name:      "future timestamp within drift tolerance (2 min ahead)",
			timestamp: time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339),
			expect:    true,
		},
		{
			name:      "future timestamp beyond drift tolerance (6 min ahead)",
			timestamp: time.Now().Add(6 * time.Minute).UTC().Format(time.RFC3339),
			expect:    false,
		},
		{
			name:      "empty timestamp (missing)",
			timestamp: "",
			expect:    false,
		},
		{
			name:      "zero epoch timestamp",
			timestamp: "0",
			expect:    false,
		},
		{
			name:      "unparseable timestamp",
			timestamp: "not-a-timestamp",
			expect:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gowa.CheckReplay(tt.timestamp, maxAge)
			assert.Equal(t, tt.expect, result, "CheckReplay(%q, %v) = %v, want %v", tt.timestamp, maxAge, result, tt.expect)
		})
	}
}
