package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// redownloadTimeout caps how long a single media re-fetch may take. GOWA's
// /message/{id}/download resolves a URL then fetches the bytes, so allow room.
const redownloadTimeout = 60 * time.Second

// redownloadCooldown is the per-message cooldown (FR-014, FR-024). Any user's
// re-download of message X activates the cooldown for ALL users requesting X
// within this window. This prevents coordinated abuse across accounts.
const redownloadCooldown = 60 * time.Second

// RedownloadMedia re-fetches a message's media from its provider and updates
// the stored MediaURL. It exists for the case where the original download at
// receipt time failed (e.g. a transient GOWA error), leaving the message with
// a MediaURL that points at no file on disk. Auth mirrors ServeMedia: the
// caller must own the message's contact (or hold contacts:read).
//
// Only GOWA accounts are supported today — Meta Cloud media expires on
// Meta's servers and can't be reliably re-fetched by media ID later.
//
//	POST /api/media/{message_id}/redownload
func (a *App) RedownloadMedia(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}

	messageIDStr := r.RequestCtx.UserValue("message_id").(string)
	_ = messageIDStr // kept for log/debug parity; parsePathUUID does the real parse
	msgUUID, perr := parsePathUUID(r, "message_id", "message")
	if perr != nil {
		return nil
	}

	// Load the message scoped to the org.
	var message models.Message
	if err := a.DB.Where("id = ? AND organization_id = ?", msgUUID, orgID).First(&message).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Message not found", nil, "")
	}

	// Ownership gate — same logic as ServeMedia (media.go:212-228).
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionRead, orgID) {
		if !a.canAccessContactMedia(userID, orgID, message.ContactID) {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Access denied", nil, "")
		}
	}

	// Must be a media message with a provider message ID to re-fetch.
	if message.WhatsAppMessageID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"This message has no provider message ID to re-download from", nil, "")
	}

	// The contact's phone is the chat JID for the GOWA download call.
	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", message.ContactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Resolve the account. A message may reference an account that was renamed
	// or deleted (common with legacy GOWA history-sync rows); fall back to the
	// contact's current account before giving up. Mirrors ServeMedia's logic.
	var account models.WhatsAppAccount
	acctName := message.WhatsAppAccount
	if err := a.DB.Where("name = ? AND organization_id = ?", acctName, orgID).First(&account).Error; err != nil {
		if contact.WhatsAppAccount != "" && contact.WhatsAppAccount != acctName {
			a.Log.Warn("Re-download: message references a non-existent account; falling back to the contact's current account",
				"message_id", message.ID, "msg_account", acctName, "contact_account", contact.WhatsAppAccount)
			err = a.DB.Where("name = ? AND organization_id = ?", contact.WhatsAppAccount, orgID).First(&account).Error
		}
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Account not found", nil, "")
		}
	}
	a.decryptAccountSecrets(&account)

	provider := a.resolveProvider(&account)
	gowaClient, ok := provider.(*gowa.Client)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Re-download is only supported for GOWA accounts", nil, "")
	}

	// Build the chat JID (handles group @g.us vs 1:1 suffix).
	chatJID := gowaChatJID(&contact)

	// Per-message cooldown (FR-014, FR-024, FR-025). The key is per-MESSAGE
	// (not per-user) so any user's re-download blocks all others for the same
	// message within the cooldown window. If Redis is unavailable, fail-open
	// (allow the re-download) and log a warning — the cooldown is an
	// abuse-control optimization, not a security boundary.
	cooldownKey := fmt.Sprintf("media:redownload:%s", message.ID)
	set, err := a.Redis.SetNX(r.RequestCtx, cooldownKey, "1", redownloadCooldown).Result()
	if err != nil {
		a.Log.Warn("Redis unavailable for re-download cooldown, failing open", "message_id", message.ID, "error", err)
		// Continue — fail-open per FR-025
	} else if !set {
		return r.SendErrorEnvelope(fasthttp.StatusTooManyRequests,
			"Re-download recently performed for this message. Please wait and try again.", nil, "")
	}

	ctx, cancel := context.WithTimeout(r.RequestCtx, redownloadTimeout)
	defer cancel()

	waAccount := account.ToWAAccount()
	data, mediaType, derr := gowaClient.DownloadMessageMedia(ctx, waAccount, message.WhatsAppMessageID, chatJID)
	if derr != nil {
		a.Log.Error("Re-download failed", "message_id", message.ID, "gowa_msg_id", message.WhatsAppMessageID, "error", derr)
		// Surface a plain, user-facing message. Map the two common failure
		// modes (media purged from provider, vs. a transient provider error)
		// to stable codes the frontend can translate.
		code, msg := classifyRedownloadError(derr)
		return r.SendErrorEnvelope(code, msg, nil, "")
	}
	if len(data) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound,
			"This media is no longer available on the provider", nil, "")
	}

	// Save the bytes using the same logic as the original download path.
	relativePath, serr := a.saveMediaBytes(data, mediaType)
	if serr != nil {
		a.Log.Error("Failed to save re-downloaded media", "message_id", message.ID, "error", serr)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Downloaded the file but failed to save it", nil, "")
	}

	// Sniff the real MIME type from the bytes. GOWA returns a generic
	// media_type ("image"/"audio"/"video") which is NOT a valid MIME type and
	// breaks the frontend (which checks startsWith("image/")). The sniffed
	// value (e.g. "image/jpeg") is authoritative.
	sniffLen := 512
	if len(data) < sniffLen {
		sniffLen = len(data)
	}
	sniffedType := http.DetectContentType(data[:sniffLen])

	// Update the message in place.
	updates := map[string]any{
		"media_url": relativePath,
	}
	// Store the sniffed MIME type so the frontend can render the bubble.
	if sniffedType != "" {
		updates["media_mime_type"] = sniffedType
	}
	if err := a.DB.Model(&models.Message{}).Where("id = ?", message.ID).Updates(updates).Error; err != nil {
		a.Log.Error("Failed to update message media_url after re-download", "message_id", message.ID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Saved but failed to update message", nil, "")
	}

	a.Log.Info("Media re-downloaded", "message_id", message.ID, "path", relativePath, "size", len(data))

	return r.SendEnvelope(map[string]any{
		"message_id":  message.ID,
		"media_url":   relativePath,
		"media_type":  mediaType,
		"size":        len(data),
		"revalidated": true,
	})
}

// classifyRedownloadError inspects a provider error and returns a stable,
// user-facing message. The detailed provider payload is logged server-side;
// the user only ever sees a short, translatable sentence.
//
// Two buckets:
//   - mediaGone: the provider no longer has this media (404, "does not
//     belong", purge, not-found) → the file is permanently unrecoverable.
//   - otherwise: a transient/auth/provider issue → worth retrying later.
func classifyRedownloadError(err error) (int, string) {
	msg := err.Error()
	low := strings.ToLower(msg)

	// Signals that the media is gone from the provider for good.
	mediaGone := strings.Contains(low, "does not belong") ||
		strings.Contains(low, "not found") ||
		strings.Contains(low, "no longer available") ||
		strings.Contains(low, "expired") ||
		strings.Contains(low, "deleted") ||
		strings.Contains(low, "status 404")

	if mediaGone {
		return fasthttp.StatusNotFound,
			"This file is no longer available on the provider and cannot be recovered"
	}
	return fasthttp.StatusBadGateway,
		"Could not fetch this media from the provider. Please try again later"
}
