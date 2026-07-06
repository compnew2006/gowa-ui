package handlers

import (
	"context"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// GOWA read-side proxy endpoints.
//
// These endpoints are registered ONLY when cfg.WhatsApp.Provider == "gowa".
// They proxy chat/message reads to the GOWA server, scoped to a whatomate
// instance (which maps 1:1 to a GOWA device). They do NOT touch the local
// PostgreSQL Contact/Message tables because GOWA is the source of truth for
// chat content in gowa mode; whatomate only persists org/user/RBAC/instance
// metadata locally.
//
// Path convention: /api/gowa/instances/{instanceID}/... so the routes are
// unambiguously gowa-scoped and never collide with /api/contacts/* (which
// stays the Meta/whatsmeow contact API).

// GowaListChats proxies GET /chats on GOWA for a given instance.
//
// Query params (mirrored to GOWA):
//   - limit (default 25, max 100)
//   - offset (default 0)
//   - search (substring match on chat name)
//   - has_media (bool)
//   - archived (bool, omit to get both)
//
// GET /api/gowa/instances/{id}/chats
func (a *App) GowaListChats(r *fastglue.Request) error {
	if !a.isGowaProvider() || a.GowaClient == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not active", nil, "")
	}
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceReadPermission(r, userID); err != nil {
		return nil
	}
	instance, glucErr := a.fetchOwnedInstance(r, orgID, userID)
	if glucErr != nil {
		return glucErr
	}

	filter := parseListChatsFilter(r)
	ctx, cancel := context.WithTimeout(r.RequestCtx, 30*time.Second)
	defer cancel()
	resp, err := a.GowaClient.ListChats(ctx, gowaDeviceID(instance), filter)
	if err != nil {
		return gowaSendError(r, err, "Failed to list chats from GOWA")
	}
	return r.SendEnvelope(resp)
}

// GowaGetChatMessages proxies GET /chat/:chat_jid/messages on GOWA.
//
// Path params:
//   - id       = whatomate instance UUID
//   - chat_jid = WhatsApp JID (e.g. "12025550100@s.whatsapp.net" or a group JID)
//
// Query params (mirrored to GOWA):
//   - limit (default 50, max 200)
//   - offset (default 0)
//   - search
//   - start_time (RFC3339)
//   - end_time (RFC3339)
//   - media_only (bool)
//   - is_from_me (bool)
//
// GET /api/gowa/instances/{id}/chats/{chat_jid}/messages
func (a *App) GowaGetChatMessages(r *fastglue.Request) error {
	if !a.isGowaProvider() || a.GowaClient == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not active", nil, "")
	}
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceReadPermission(r, userID); err != nil {
		return nil
	}
	instance, glucErr := a.fetchOwnedInstance(r, orgID, userID)
	if glucErr != nil {
		return glucErr
	}

	chatJID, ok := r.RequestCtx.UserValue("chat_jid").(string)
	if !ok || chatJID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Missing chat_jid in path", nil, "chat_jid")
	}
	if !isValidWhatsAppJID(chatJID) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid chat_jid (expected <number>@s.whatsapp.net or group JID)", nil, "chat_jid")
	}

	filter := parseGetMessagesFilter(r)
	ctx, cancel := context.WithTimeout(r.RequestCtx, 30*time.Second)
	defer cancel()
	resp, err := a.GowaClient.GetChatMessages(ctx, gowaDeviceID(instance), chatJID, filter)
	if err != nil {
		return gowaSendError(r, err, "Failed to fetch messages from GOWA")
	}
	return r.SendEnvelope(resp)
}

// GowaDownloadMedia proxies GET /message/:message_id/download on GOWA.
//
// Returns GOWA's media metadata (file_url, filename, mime, size). The caller
// can then fetch the bytes via the file_url (which points at GOWA's static
// server) — whatomate does not relay the bytes themselves in this endpoint.
//
// GET /api/gowa/instances/{id}/messages/{message_id}/media
func (a *App) GowaDownloadMedia(r *fastglue.Request) error {
	if !a.isGowaProvider() || a.GowaClient == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "GOWA provider not active", nil, "")
	}
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireInstanceReadPermission(r, userID); err != nil {
		return nil
	}
	instance, glucErr := a.fetchOwnedInstance(r, orgID, userID)
	if glucErr != nil {
		return glucErr
	}

	messageID, ok := r.RequestCtx.UserValue("message_id").(string)
	if !ok || messageID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Missing message_id in path", nil, "message_id")
	}
	phone := string(r.RequestCtx.QueryArgs().Peek("phone"))

	ctx, cancel := context.WithTimeout(r.RequestCtx, 30*time.Second)
	defer cancel()
	resp, err := a.GowaClient.DownloadMedia(ctx, gowaDeviceID(instance), messageID, phone)
	if err != nil {
		return gowaSendError(r, err, "Failed to download media from GOWA")
	}
	return r.SendEnvelope(resp)
}
