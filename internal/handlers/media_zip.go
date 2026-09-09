package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// maxZipMessageIDs bounds how many messages a single burst download may
// contain. Bursts are small by design (a flurry of files in a live chat),
// but a hard cap protects against abuse.
const maxZipMessageIDs = 50

// maxZipTotalSize bounds the total uncompressed size of media files in a
// single ZIP archive (FR-015). Prevents memory exhaustion from large bursts.
const maxZipTotalSize = 250 * 1024 * 1024 // 250 MB

// ServeMediaZip streams a ZIP archive containing the media of the requested
// message IDs. Authorization mirrors ServeMedia exactly: the caller's org is
// enforced via the DB query, and per-user contact visibility is gated through
// scopeAssignedContact for agents without contacts:read. The client's ID list
// is never trusted — any ID that doesn't resolve to an org-owned,
// media-bearing, accessible message is silently dropped from the archive.
//
// Note: this is a chat-workflow convenience (collect a burst of incoming
// files), not a bulk data export — the same files are downloadable one by one
// via ServeMedia, so an export permission here would only block the ZIP
// button for agents without adding any real restriction.
//
// Example: GET /api/media/zip?ids=<uuid>,<uuid>,...
func (a *App) ServeMediaZip(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}

	// Parse the comma-separated message IDs from the query string.
	rawIDs := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("ids")))
	if rawIDs == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "ids query parameter is required", nil, "")
	}
	idStrs := strings.Split(rawIDs, ",")
	if len(idStrs) > maxZipMessageIDs {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			fmt.Sprintf("too many IDs (max %d)", maxZipMessageIDs), nil, "")
	}

	messageIDs := make([]uuid.UUID, 0, len(idStrs))
	for _, s := range idStrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := uuid.Parse(s)
		if err != nil {
			continue // malformed IDs are ignored, not fatal
		}
		messageIDs = append(messageIDs, id)
	}
	if len(messageIDs) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "no valid message IDs", nil, "")
	}

	// Org-scoped fetch — this is the core authorization filter. Messages from
	// another org or without media are simply absent from the result set.
	var messages []models.Message
	if err := a.DB.Where("id IN ? AND organization_id = ? AND media_url <> ''",
		messageIDs, orgID).Find(&messages).Error; err != nil {
		a.Log.Error("Failed to query messages for zip", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load messages", nil, "")
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

	// Build the archive in memory. Bursts are small by design (a flurry of
	// files in a live chat), so buffering matches every other binary response
	// in this codebase and avoids introducing an unproven streaming path.
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	usedNames := make(map[string]bool)

	wroteAtLeastOne := false
	var totalSize int64
	for i := range messages {
		msg := messages[i]

		// Path security — same guards as ServeMedia (media.go).
		fullPath, ok := resolveMediaPath(baseDir, msg.MediaURL)
		if !ok {
			a.Log.Warn("Skipping zip entry: invalid or missing path", "message_id", msg.ID, "media_url", msg.MediaURL)
			continue
		}

		// Total-size guard (FR-015): prevent memory exhaustion from large bursts.
		if fi, err := os.Stat(fullPath); err == nil {
			if totalSize+fi.Size() > maxZipTotalSize {
				a.Log.Warn("ZIP total size limit reached, stopping", "total_size", totalSize, "limit", maxZipTotalSize)
				break
			}
			totalSize += fi.Size()
		}

		entryName := uniqueZipName(defaultZipEntryName(&msg), usedNames)
		if err := addFileToZip(zw, entryName, fullPath); err != nil {
			a.Log.Error("Failed to add file to zip", "message_id", msg.ID, "error", err)
			continue
		}
		wroteAtLeastOne = true
	}

	if err := zw.Close(); err != nil {
		a.Log.Error("Failed to finalize zip", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to build archive", nil, "")
	}
	if !wroteAtLeastOne {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "No accessible media files found", nil, "")
	}

	zipName := fmt.Sprintf("files_%s.zip", time.Now().UTC().Format("20060102_150405"))
	r.RequestCtx.Response.Header.Set("Content-Type", "application/zip")
	r.RequestCtx.Response.Header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, zipName))
	r.RequestCtx.Response.Header.Set("Cache-Control", "private, no-store")
	r.RequestCtx.SetBody(buf.Bytes())

	return nil
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

// addFileToZip opens the source file and copies its bytes into a stored
// (uncompressed) zip entry. Stored mode avoids CPU cost on already-compressed
// media (jpg, mp4, pdf) and keeps memory bounded to the entry being written.
func addFileToZip(zw *zip.Writer, name, fullPath string) error {
	fh := &zip.FileHeader{Name: name, Method: zip.Store}
	fh.SetMode(0644)
	w, err := zw.CreateHeader(fh)
	if err != nil {
		return err
	}
	f, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}
