package gowa

import "context"

// RejectCall rejects an incoming WhatsApp call.
// The callerJID and callID come from the webhook event payload
// (event: "call.offer", payload.from and payload.call_id).
// GOWA endpoint: POST /call/reject
func (c *Client) RejectCall(ctx context.Context, deviceID, callerJID, callID string) error {
	body := map[string]any{
		"caller_jid": callerJID,
		"call_id":    callID,
	}
	_, err := c.doJSON(ctx, "POST", "/call/reject", deviceID, body)
	return err
}
