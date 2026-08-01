package handlers

// Contact avatar (profile picture) handling: fetch from GOWA, cache the bytes
// locally, and serve them via stable cookie-authenticated routes so plain
// <img> tags can render them (WhatsApp CDN URLs are signed and expire).
//
// Split out of contacts.go so the CRUD handlers stay focused on contact
// records. These functions are methods/receivers in the same `handlers`
// package, so routing in main.go (app.RefreshContactAvatar /
// app.ServeContactAvatar) is unchanged.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// RefreshContactAvatar re-fetches a contact's WhatsApp profile picture from the
// provider and returns the stable serve-route URL.
//
// GET /api/contacts/{id}/avatar
func (a *App) RefreshContactAvatar(r *fastglue.Request) error {
	orgID, _, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	// The avatar endpoint returns a (cached) profile picture URL — not private
	// conversation content — so it is scoped to the organization only. Applying
	// scopeAssignedContact here caused a 404 for any contact an agent can see
	// in their sidebar but isn't formally assigned/collaborating on (e.g. a
	// pending chat before claim, or a newsletter the admin surfaced). The
	// agent still sees the cached avatar; only the live GOWA re-fetch below is
	// gated by having a resolvable owning account.
	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Fetch + cache the picture (force so a manual refresh always re-checks the
	// provider), then return the stable serve-route URL. resolveAndRefreshAvatar
	// is a no-op when the owning account can't be resolved, so the response
	// still returns whatever is already cached.
	a.resolveAndRefreshAvatar(&contact, orgID, true)

	return r.SendEnvelope(map[string]any{"avatar_url": contactAvatarURL(&contact)})
}

// ServeContactAvatar streams the locally-cached copy of a contact's WhatsApp
// profile picture. WhatsApp CDN URLs are signed and expire, so the bytes are
// downloaded once (see refreshContactAvatar) and served from this stable,
// cookie-authenticated route that plain <img> tags can load directly. When the
// cache is empty (or the file has vanished) it attempts a best-effort lazy
// fetch before giving up with a 404, at which point the frontend shows colored
// initials.
//
// GET /api/contacts/{id}/avatar/image
func (a *App) ServeContactAvatar(r *fastglue.Request) error {
	orgID, _, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}
	contactID, err := parsePathUUID(r, "id", "contact")
	if err != nil {
		return nil
	}

	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
	}

	// Populate the cache lazily the first time this contact's picture is
	// requested (contacts created by an inbound message before a GOWA sync).
	if contact.AvatarLocalPath == "" {
		a.resolveAndRefreshAvatar(&contact, orgID, false)
	}

	data, ok := a.readLocalMedia(contact.AvatarLocalPath)
	if !ok {
		// The file is missing from disk (e.g. cache cleared or a stale row) —
		// try one forced re-fetch before giving up.
		a.resolveAndRefreshAvatar(&contact, orgID, true)
		if data, ok = a.readLocalMedia(contact.AvatarLocalPath); !ok {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "No avatar", nil, "")
		}
	}

	sniffLen := len(data)
	if sniffLen > 512 {
		sniffLen = 512
	}
	r.RequestCtx.Response.Header.Set("Content-Type", http.DetectContentType(data[:sniffLen]))
	r.RequestCtx.Response.Header.Set("Cache-Control", "private, max-age=3600")
	r.RequestCtx.SetBody(data)
	return nil
}

// resolveAndRefreshAvatar resolves the contact's owning GOWA account and does a
// best-effort profile-picture fetch+cache via refreshContactAvatar. It is a
// no-op (and logs at warn) when the account can't be resolved, so callers can
// invoke it unconditionally. force=true always re-checks the provider; false
// skips contacts that already have a locally-cached avatar.
func (a *App) resolveAndRefreshAvatar(contact *models.Contact, orgID uuid.UUID, force bool) {
	if contact == nil || contact.WhatsAppAccount == "" {
		return
	}
	var account models.WhatsAppAccount
	if err := a.DB.Where("name = ? AND organization_id = ?", contact.WhatsAppAccount, orgID).First(&account).Error; err != nil {
		a.Log.Warn("Contact avatar refresh: account not found", "account", contact.WhatsAppAccount, "contact", contact.ID)
		return
	}
	a.decryptAccountSecrets(&account)
	provider := a.resolveProvider(&account)
	gowaClient, _ := provider.(*gowa.Client)
	a.refreshContactAvatar(gowaClient, &account, contact, account.GowaDeviceID, force)
}

// contactAvatarURL returns the stable backend route that serves the contact's
// locally-cached profile picture, or an empty string when nothing is cached
// yet (the frontend then renders colored initials). A cache-busting token
// derived from the stored filename (a fresh UUID on every re-download) makes
// the browser refetch when the picture changes even though the path is stable.
func contactAvatarURL(contact *models.Contact) string {
	if contact == nil || contact.AvatarLocalPath == "" {
		return ""
	}
	base := filepath.Base(contact.AvatarLocalPath)
	token := strings.TrimSuffix(base, filepath.Ext(base))
	return fmt.Sprintf("/api/contacts/%s/avatar/image?v=%s", contact.ID, token)
}

// avatarFetchTimeout caps how long a single profile-picture lookup may take.
// GOWA's /user/avatar is usually fast, but a hung device shouldn't stall a
// whole contact sync or the lazy avatar endpoint.
const avatarFetchTimeout = 8 * time.Second

// refreshContactAvatar fetches the contact's WhatsApp profile picture (or group
// icon) via the GOWA client and persists it onto the contact row. It is
// best-effort: any error (no GOWA provider, network failure, the contact has
// hidden their picture, etc.) is logged and ignored so callers can keep
// processing other contacts. The fetched URL is cached on the contact's
// avatar_url column; callers pass force=false to skip contacts that already
// have an avatar (avoiding a GOWA round-trip on every sync).
//
// client may be nil (the caller decides whether to resolve one); in that
// case this is a no-op.
func (a *App) refreshContactAvatar(client *gowa.Client, account *models.WhatsAppAccount, contact *models.Contact, deviceID string, force bool) bool {
	if client == nil || contact == nil || account == nil {
		return false
	}
	// Skip the round-trip when we already have a locally-cached picture and the
	// caller didn't ask for a forced refresh (e.g. user clicked "refresh
	// avatar"). We key the skip on AvatarLocalPath (not AvatarURL) so contacts
	// carrying only a legacy/expired CDN URL still get their bytes cached.
	if !force && contact.AvatarLocalPath != "" {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), avatarFetchTimeout)
	defer cancel()

	// GOWA's /user/avatar?phone= accepts either a bare phone or a full JID.
	// For groups/newsletters the JID (…@g.us / …@newsletter) is what resolves
	// the group icon, so we reuse the same JID builder as send/reaction paths.
	phone := contact.PhoneNumber
	if isGroupContact(contact) {
		phone = gowaChatJID(contact)
	}
	if phone == "" {
		return false
	}

	avatar, err := client.GetUserAvatar(ctx, deviceID, phone)
	if err != nil {
		a.Log.Debug("Could not fetch WhatsApp avatar",
			"contact_id", contact.ID, "phone", phone, "error", err)
		return false
	}
	if avatar == nil || avatar.URL == "" {
		return false
	}
	// Nothing to do when the provider handed back the same URL we already
	// downloaded and the bytes are still on disk.
	if avatar.URL == contact.AvatarURL && contact.AvatarLocalPath != "" {
		return false
	}

	// Download the picture bytes and cache them on disk. WhatsApp CDN URLs are
	// signed and expire, so we serve our own stable copy instead of hot-linking
	// the ephemeral URL. Download via the shared SSRF-safe HTTPClient (the GOWA
	// client uses Basic Auth, which is wrong for the public pps.whatsapp.net CDN).
	data, err := a.downloadAvatarImage(ctx, avatar.URL)
	if err != nil {
		a.Log.Debug("Could not download WhatsApp avatar bytes",
			"contact_id", contact.ID, "url", avatar.URL, "error", err)
		return false
	}

	relPath, err := a.saveMediaBytes(data, "image/jpeg")
	if err != nil {
		a.Log.Error("Failed to cache contact avatar",
			"contact_id", contact.ID, "error", err)
		return false
	}

	oldPath := contact.AvatarLocalPath
	if err := a.DB.Model(contact).Updates(map[string]any{
		"avatar_url":        avatar.URL,
		"avatar_local_path": relPath,
	}).Error; err != nil {
		a.Log.Error("Failed to persist contact avatar",
			"contact_id", contact.ID, "error", err)
		a.removeLocalMedia(relPath) // don't leak the just-written file
		return false
	}
	contact.AvatarURL = avatar.URL
	contact.AvatarLocalPath = relPath
	// Best-effort cleanup of the superseded cached copy.
	if oldPath != "" && oldPath != relPath {
		a.removeLocalMedia(oldPath)
	}
	return true
}

// maxAvatarBytes caps how many bytes we read from the profile-picture CDN so a
// hostile/misbehaving URL can't exhaust memory. Profile pictures are small
// (typically well under 200 KiB); 5 MiB is a generous ceiling.
const maxAvatarBytes = 5 << 20

// downloadAvatarImage fetches the (public) WhatsApp CDN profile-picture URL via
// the shared SSRF-safe HTTPClient and returns the response body, capped at
// maxAvatarBytes. Callers pass a ctx carrying the avatar fetch timeout.
func (a *App) downloadAvatarImage(ctx context.Context, url string) ([]byte, error) {
	if a.HTTPClient == nil {
		return nil, fmt.Errorf("no HTTP client configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAvatarBytes))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty avatar body")
	}
	return data, nil
}

// isGroupContact reports whether the contact represents a WhatsApp group or
// newsletter (its phone_number carries the @g.us/@newsletter JID suffix or the
// 120362/120363 group-ID prefix, or it was flagged via metadata).
func isGroupContact(contact *models.Contact) bool {
	if contact == nil {
		return false
	}
	if contact.Metadata != nil && (contact.Metadata["is_group_chat"] == true || contact.Metadata["is_newsletter"] == true) {
		return true
	}
	p := contact.PhoneNumber
	if strings.Contains(p, "@g.us") || strings.Contains(p, "@newsletter") {
		return true
	}
	return strings.HasPrefix(p, "120362") || strings.HasPrefix(p, "120363")
}
