package handlers

import (
	"archive/zip"
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// maxZipMessageIDs bounds how many messages a single burst download may
// contain. WhatsApp shares up to 100 files at once and a burst can span
// several shares, so the cap sits well above a real album while still
// bounding abuse. (It was 50: a 55-file album could not be zipped at all.)
const maxZipMessageIDs = 500

// maxZipTotalSize bounds the total size of media files in a single ZIP
// archive (FR-015). The archive is streamed to the client (never buffered
// whole in memory), so this bounds disk reads and transfer size, not RAM.
const maxZipTotalSize = 1024 * 1024 * 1024 // 1 GiB

// zipRecoveryBudget bounds how long one ZIP request spends re-fetching
// missing files from GOWA before the archive starts streaming — kept under a
// reverse proxy's default 60s read timeout. Files not recovered in time are
// reported as skipped; a retry finds the already-recovered ones on disk.
const zipRecoveryBudget = 40 * time.Second

// Response headers reporting how many of the requested files made it into
// the archive, so the client can warn instead of silently delivering fewer
// files than the chat shows.
const (
	zipIncludedHeader = "X-Media-Zip-Included"
	zipSkippedHeader  = "X-Media-Zip-Skipped"
)

// mediaZipRequest is the POST body of /api/media/zip. A JSON body carries
// hundreds of IDs without hitting URL/header-size limits.
type mediaZipRequest struct {
	IDs []string `json:"ids"`
}

// zipEntry is one pre-validated archive member.
type zipEntry struct {
	name     string
	fullPath string
}

// ServeMediaZip streams a ZIP archive containing the media of the requested
// message IDs. Authorization mirrors ServeMedia exactly: the caller's org is
// enforced via the DB query, and per-user contact visibility is gated through
// scopeAssignedContact for agents without contacts:read. The client's ID list
// is never trusted — any ID that doesn't resolve to an org-owned,
// media-bearing, accessible message is dropped from the archive, and the
// X-Media-Zip-Included / X-Media-Zip-Skipped headers report the outcome.
//
// Files that were never downloaded (or are gone from disk) are lazily
// recovered from GOWA exactly like ServeMedia does, so the archive holds the
// same files the chat bubbles open one by one.
//
// Note: this is a chat-workflow convenience (collect a burst of incoming
// files), not a bulk data export — the same files are downloadable one by one
// via ServeMedia, so an export permission here would only block the ZIP
// button for agents without adding any real restriction.
//
// Example: POST /api/media/zip {"ids": ["<uuid>", ...]}
// (GET /api/media/zip?ids=<uuid>,<uuid>,... is kept for compatibility).
func (a *App) ServeMediaZip(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}

	idStrs, err := a.zipRequestIDs(r)
	if err != nil {
		return nil
	}
	if len(idStrs) > maxZipMessageIDs {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			fmt.Sprintf("too many IDs (max %d)", maxZipMessageIDs), nil, "")
	}

	messageIDs := make([]uuid.UUID, 0, len(idStrs))
	seen := make(map[uuid.UUID]bool, len(idStrs))
	for _, s := range idStrs {
		id, err := uuid.Parse(strings.TrimSpace(s))
		if err != nil || seen[id] {
			continue // malformed or repeated IDs are ignored, not fatal
		}
		seen[id] = true
		messageIDs = append(messageIDs, id)
	}
	if len(messageIDs) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "no valid message IDs", nil, "")
	}

	// Org-scoped fetch — this is the core authorization filter. Messages from
	// another org are simply absent from the result set. Oldest first, so
	// entries (and filename-collision suffixes) follow the chat order.
	var rows []models.Message
	if err := a.DB.Where("id IN ? AND organization_id = ?", messageIDs, orgID).
		Order("created_at ASC").Find(&rows).Error; err != nil {
		a.Log.Error("Failed to query messages for zip", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load messages", nil, "")
	}
	// Media-bearing only: a stored file, or a media message whose bytes were
	// never downloaded but can still be fetched from GOWA (same rule as
	// ServeMedia's lazy recovery).
	messages := rows[:0]
	for i := range rows {
		if rows[i].MediaURL != "" ||
			(rows[i].WhatsAppMessageID != "" && isRecoverableMediaType(rows[i].MessageType)) {
			messages = append(messages, rows[i])
		}
	}
	if len(messages) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "No accessible media found", nil, "")
	}

	// Every message's contact must pass the single contact-visibility gate —
	// contacts:read alone is NOT org-wide access (account subsets and
	// involvement still apply; mirrors ServeMedia). Messages from contacts
	// the caller can't reach are dropped from the archive.
	contactIDs := make([]uuid.UUID, 0, len(messages))
	for i := range messages {
		contactIDs = append(contactIDs, messages[i].ContactID)
	}
	var visibleIDs []uuid.UUID
	if err := a.scopeAssignedContact(
		a.DB.Model(&models.Contact{}).Select("id"), userID, orgID,
	).Where("id IN ?", contactIDs).Pluck("id", &visibleIDs).Error; err != nil {
		a.Log.Error("Failed to scope zip contacts", "error", err, "user_id", userID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Access denied", nil, "")
	}
	visible := make(map[uuid.UUID]bool, len(visibleIDs))
	for _, id := range visibleIDs {
		visible[id] = true
	}
	access := messages[:0]
	for i := range messages {
		if visible[messages[i].ContactID] {
			access = append(access, messages[i])
		}
	}
	messages = access
	if len(messages) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Access denied", nil, "")
	}

	// Resolve the storage base once; every file must live beneath it.
	baseDir, err := filepath.Abs(a.getMediaStoragePath())
	if err != nil {
		a.Log.Error("Storage configuration error", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Storage configuration error", nil, "")
	}

	// Pre-flight every member before the first byte goes out: once the
	// archive streams, the status can no longer change, so path checks,
	// lazy recovery and the size budget all happen here.
	recoverCtx, cancel := context.WithTimeout(context.Background(), zipRecoveryBudget)
	defer cancel()
	entries := make([]zipEntry, 0, len(messages))
	usedNames := make(map[string]bool)
	var totalSize int64
	for i := range messages {
		msg := &messages[i]
		fullPath, size, ok := a.zipEntryFile(recoverCtx, orgID, msg, baseDir)
		if !ok {
			continue
		}
		// Total-size guard (FR-015): files past the budget are skipped (and
		// reported), never silently cut.
		if totalSize+size > maxZipTotalSize {
			a.Log.Warn("ZIP total size limit reached, skipping entry",
				"message_id", msg.ID, "total_size", totalSize, "limit", maxZipTotalSize)
			continue
		}
		totalSize += size
		entries = append(entries, zipEntry{
			name:     uniqueZipName(defaultZipEntryName(msg), usedNames),
			fullPath: fullPath,
		})
	}
	if len(entries) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "No accessible media files found", nil, "")
	}

	zipName := fmt.Sprintf("files_%s.zip", time.Now().UTC().Format("20060102_150405"))
	h := &r.RequestCtx.Response.Header
	h.Set("Content-Type", "application/zip")
	h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, zipName))
	h.Set("Cache-Control", "private, no-store")
	h.Set(zipIncludedHeader, strconv.Itoa(len(entries)))
	h.Set(zipSkippedHeader, strconv.Itoa(len(messageIDs)-len(entries)))

	// Stream the archive: entries are copied file → response with a fixed
	// buffer, so memory stays flat no matter how many files the burst holds.
	log := a.Log
	r.RequestCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
		zw := zip.NewWriter(w)
		for _, e := range entries {
			if err := addFileToZip(zw, e.name, e.fullPath); err != nil {
				log.Error("Failed to add file to zip", "entry", e.name, "error", err)
				if errors.Is(err, errZipSourceUnavailable) {
					continue // nothing written for this entry; the archive stays valid
				}
				return // a half-written entry leaves the archive unusable
			}
			if err := w.Flush(); err != nil {
				return // client went away
			}
		}
		if err := zw.Close(); err != nil {
			log.Error("Failed to finalize zip", "error", err)
		}
	})

	return nil
}

// zipRequestIDs reads the requested message IDs: a JSON body on POST, the
// comma-separated `ids` query parameter otherwise. On a malformed or empty
// request the 400 envelope is already sent and a non-nil error returned.
func (a *App) zipRequestIDs(r *fastglue.Request) ([]string, error) {
	if string(r.RequestCtx.Method()) == fasthttp.MethodPost {
		var req mediaZipRequest
		if err := a.decodeRequest(r, &req); err != nil {
			return nil, err
		}
		if len(req.IDs) == 0 {
			_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "ids are required", nil, "")
			return nil, errEnvelopeSent
		}
		return req.IDs, nil
	}
	rawIDs := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("ids")))
	if rawIDs == "" {
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "ids query parameter is required", nil, "")
		return nil, errEnvelopeSent
	}
	return strings.Split(rawIDs, ","), nil
}

// zipEntryFile resolves the on-disk file for one archive member and its size.
// A file that was never downloaded or is gone from disk is lazily recovered
// from GOWA within ctx (same path as ServeMedia). Retention-purged media is
// never resurrected here — ServeMedia answers 410 for it, and only an
// explicit redownload may bring it back. Invalid paths and symlinks are
// skipped (same guards as ServeMedia).
func (a *App) zipEntryFile(ctx context.Context, orgID uuid.UUID, msg *models.Message, baseDir string) (string, int64, bool) {
	if messageRetentionState(msg) != "" {
		return "", 0, false
	}
	var missingErr error = os.ErrNotExist
	if msg.MediaURL != "" {
		fullPath, ok := resolveWithinBase(baseDir, msg.MediaURL)
		if !ok {
			a.Log.Warn("Skipping zip entry: invalid path", "message_id", msg.ID, "media_url", msg.MediaURL)
			return "", 0, false
		}
		info, err := os.Lstat(fullPath)
		if err == nil {
			if !info.Mode().IsRegular() {
				a.Log.Warn("Skipping zip entry: not a regular file", "message_id", msg.ID)
				return "", 0, false
			}
			return fullPath, info.Size(), true
		}
		missingErr = err
	}
	if msg.WhatsAppMessageID == "" || !isRecoverableMediaType(msg.MessageType) || ctx.Err() != nil {
		a.Log.Warn("Skipping zip entry: media missing and not recoverable now", "message_id", msg.ID, "error", missingErr)
		return "", 0, false
	}
	fullPath, info, err := a.recoverMissingMedia(ctx, orgID, msg, baseDir, missingErr)
	if err != nil || !info.Mode().IsRegular() {
		return "", 0, false
	}
	return fullPath, info.Size(), true
}

// defaultZipEntryName picks a sensible filename for a zip entry from the
// message, falling back to the message type + id when no original name is
// stored.
func defaultZipEntryName(msg *models.Message) string {
	if name := strings.TrimSpace(msg.MediaFilename); name != "" {
		return filepath.Base(name)
	}
	mimeType := string(msg.MediaMimeType)
	ext := getExtensionFromMimeType(mimeType)
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("%s_%s%s", msg.MessageType, msg.ID.String()[:8], ext)
}

// uniqueZipName ensures no two entries in the same archive share a name by
// suffixing a short id fragment on collision.
func uniqueZipName(name string, used map[string]bool) string {
	if !used[name] {
		used[name] = true
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	shortID := uuid.New().String()[:8]
	for i := 2; ; i++ {
		var candidate string
		if i == 2 {
			candidate = fmt.Sprintf("%s_%s%s", base, shortID, ext)
		} else {
			candidate = fmt.Sprintf("%s_%s_%d%s", base, shortID, i, ext)
		}
		if !used[candidate] {
			used[candidate] = true
			return candidate
		}
	}
}

// errZipSourceUnavailable marks an archive member whose source file could not
// be opened — nothing was written for it, so the archive itself stays valid.
var errZipSourceUnavailable = errors.New("zip source file unavailable")

// addFileToZip opens the source file and copies its bytes into a stored
// (uncompressed) zip entry. Stored mode avoids CPU cost on already-compressed
// media (jpg, mp4, pdf) and keeps memory bounded to the copy buffer. The
// source is opened BEFORE the entry header is written, so a vanished file
// never leaves an empty entry behind.
func addFileToZip(zw *zip.Writer, name, fullPath string) error {
	f, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("%w: %v", errZipSourceUnavailable, err)
	}
	defer f.Close()
	fh := &zip.FileHeader{Name: name, Method: zip.Store}
	fh.SetMode(0644)
	w, err := zw.CreateHeader(fh)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}
