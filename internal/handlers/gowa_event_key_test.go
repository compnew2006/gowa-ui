package handlers

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/stretchr/testify/assert"
)

// TestDeriveGowaEventKey locks the inbox idempotency key (gap #8): a redelivery
// of the SAME logical event must derive the SAME key (so the partial unique
// index collapses it), while a genuinely different event derives a different
// one. Pure function — no DB.
func TestDeriveGowaEventKey(t *testing.T) {
	mk := func(event string, payload any) *gowa.WebhookPayload {
		b, _ := json.Marshal(payload)
		return &gowa.WebhookPayload{Event: event, Payload: b}
	}

	t.Run("message by wamid", func(t *testing.T) {
		assert.Equal(t, "wamid:MSG1",
			deriveGowaEventKey(mk("message", map[string]any{"id": "MSG1"})))
	})

	t.Run("call.offer by call_id", func(t *testing.T) {
		assert.Equal(t, "call:CALL1",
			deriveGowaEventKey(mk("call.offer", map[string]any{"call_id": "CALL1"})))
	})

	t.Run("revoked by target wamid", func(t *testing.T) {
		assert.Equal(t, "revoke:W1",
			deriveGowaEventKey(mk("message.revoked", map[string]any{"revoked_message_id": "W1"})))
	})

	t.Run("reaction by target wamid + emoji", func(t *testing.T) {
		assert.Equal(t, "react:W1:👍",
			deriveGowaEventKey(mk("message.reaction",
				map[string]any{"reacted_message_id": "W1", "reaction": "👍"})))
	})

	t.Run("ack by receipt_type + first id", func(t *testing.T) {
		assert.Equal(t, "ack:read:M1",
			deriveGowaEventKey(mk("message.ack",
				map[string]any{"ids": []string{"M1", "M2"}, "receipt_type": "read"})))
	})

	t.Run("edited includes body hash", func(t *testing.T) {
		e1 := deriveGowaEventKey(mk("message.edited",
			map[string]any{"original_message_id": "W1", "body": "hi"}))
		e2 := deriveGowaEventKey(mk("message.edited",
			map[string]any{"original_message_id": "W1", "body": "hi"}))
		e3 := deriveGowaEventKey(mk("message.edited",
			map[string]any{"original_message_id": "W1", "body": "bye"}))
		// Same edit → same key (redelivery dedupes).
		assert.Equal(t, e1, e2)
		// Same target, different body → different key (a real second edit).
		assert.NotEqual(t, e1, e3)
		// Namespace prefix carries the target so it can't collide with other types.
		assert.Equal(t, "edit:W1:", e1[:len("edit:W1:")])
	})

	t.Run("unknown event hashes payload stably", func(t *testing.T) {
		h1 := deriveGowaEventKey(&gowa.WebhookPayload{
			Event: "message.deleted", Payload: json.RawMessage(`{"id":"D1"}`)})
		h2 := deriveGowaEventKey(&gowa.WebhookPayload{
			Event: "message.deleted", Payload: json.RawMessage(`{"id":"D1"}`)})
		assert.Equal(t, h1, h2, "same payload → same hash (redelivery dedupes)")
		assert.True(t, len(h1) > len("hash:"), "hash key has a non-empty digest")
	})

	t.Run("empty payload collapses to sentinel", func(t *testing.T) {
		assert.Equal(t, "empty",
			deriveGowaEventKey(&gowa.WebhookPayload{Event: "unknown", Payload: nil}))
	})
}
