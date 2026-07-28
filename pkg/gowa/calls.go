package gowa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// RejectCall rejects an incoming WhatsApp call.
// The callerJID and callID come from the webhook event payload
// (event: "call.offer", payload.from and payload.call_id).
// GOWA endpoint: POST /call/reject
//
// Uses doJSONRaw because /call/reject answers with a GenericResponse that
// has no message_id — doJSON's parseSendResponse would misreport the
// successful rejection as an error.
func (c *Client) RejectCall(ctx context.Context, deviceID, callerJID, callID string) error {
	body := map[string]any{
		"caller_jid": callerJID,
		"call_id":    callID,
	}
	_, err := c.doJSONRaw(ctx, "POST", "/call/reject", deviceID, body)
	return err
}

// userInfoResponse is the GOWA /user/info response shape.
type userInfoResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Results struct {
		ResolvedPhone string `json:"resolved_phone"`
		ResolvedLID   string `json:"resolved_lid"`
		Data          []struct {
			Devices []struct {
				User string `json:"User"`
			} `json:"Devices"`
		} `json:"data"`
	} `json:"results"`
}

// ResolveLID resolves a WhatsApp LID (Linked ID) to a real phone number using
// GOWA's /user/info endpoint. WhatsApp calls use LIDs for privacy instead of
// phone numbers, but /send/message only accepts phone-number JIDs.
//
// lid can be either a bare number ("149641526026409") or a full LID JID
// ("149641526026409@lid"). Returns the phone digits (e.g. "966561853319")
// and nil if resolved, or "" and nil if no mapping exists.
func (c *Client) ResolveLID(ctx context.Context, deviceID, lid string) (string, error) {
	// Normalize to the @lid form that GOWA expects.
	bare := strings.TrimSuffix(lid, "@lid")
	if idx := strings.Index(bare, "@"); idx > 0 {
		bare = bare[:idx]
	}
	query := url.Values{}
	query.Set("phone", bare+"@lid")

	respBody, err := c.doRaw(ctx, "GET", "/user/info?"+query.Encode(), deviceID)
	if err != nil {
		return "", fmt.Errorf("resolve LID %s: %w", lid, err)
	}

	var info userInfoResponse
	if err := json.Unmarshal(respBody, &info); err != nil {
		return "", fmt.Errorf("parse /user/info response: %w", err)
	}

	if info.Results.ResolvedPhone != "" {
		return info.Results.ResolvedPhone, nil
	}
	// Fall back to the first device's phone if resolved_phone is absent.
	for _, d := range info.Results.Data {
		for _, dev := range d.Devices {
			if dev.User != "" {
				return dev.User, nil
			}
		}
	}
	return "", nil
}
