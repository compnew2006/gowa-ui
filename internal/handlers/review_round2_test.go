package handlers_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// Explicit prevention tests for the SECOND review round (2026-09-09) on the
// assignment-access-grant hardening: WebSocket content scoping, media
// ZIP/redownload account scoping, current-collaborator write access, the
// UpdateContact assignment path minting grants, and the org-wide scheduled
// messages list.

// mustZipReader wraps a zip response body in a zip.Reader (reuses the
// zipEntryNames helper from media_zip_test.go).
func mustZipReader(t *testing.T, body []byte) *zip.Reader {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	return zr
}

// addMediaMessage inserts a media message pointing at a file under the media
// storage root.
func addMediaMessage(t *testing.T, app *handlers.App, orgID, contactID uuid.UUID, accountName, rel, filename string) *models.Message {
	t.Helper()
	full := filepath.Join(app.Config.Storage.LocalPath, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte("bytes-of-"+filename), 0o644))

	msg := &models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		ContactID:       contactID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeImage,
		MediaURL:        rel,
		MediaFilename:   filename,
		MediaMimeType:   "image/jpeg",
		WhatsAppAccount: accountName,
	}
	require.NoError(t, app.DB.Create(msg).Error)
	return msg
}

// TestReview2_ZipDropsOutOfScopeMedia (finding: ZIP bypassed account
// scoping for contacts:read holders): an agent assigned account A only must
// find account-B media absent from the archive — even though they hold
// contacts:read.
func TestReview2_ZipDropsOutOfScopeMedia(t *testing.T) {
	f := newGrantFixture(t)
	f.app.Config.Storage.LocalPath = t.TempDir()

	contactA := testutil.CreateTestContactWith(t, f.app.DB, f.org.ID, testutil.WithContactAccount(f.accA.Name))
	visible := addMediaMessage(t, f.app, f.org.ID, contactA.ID, f.accA.Name,
		filepath.Join("images", "visible.jpg"), "visible.jpg")
	hidden := addMediaMessage(t, f.app, f.org.ID, f.contact.ID, f.accB.Name,
		filepath.Join("images", "hidden.jpg"), "hidden.jpg")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
	req.RequestCtx.QueryArgs().Set("ids", visible.ID.String()+","+hidden.ID.String())
	require.NoError(t, f.app.ServeMediaZip(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"zip must succeed with the in-scope entries, got: %s", string(testutil.GetResponseBody(req)))

	names := zipEntryNames(mustZipReader(t, testutil.GetResponseBody(req)))
	assert.Contains(t, names, "visible.jpg", "in-scope media stays in the archive")
	assert.NotContains(t, names, "hidden.jpg",
		"account-B media must be dropped for an account-A-only agent")
}

// TestReview2_RedownloadForbiddenForOutOfScopeContact (finding: media
// redownload treated contacts:read as org-wide): the gate must refuse before
// any provider call.
func TestReview2_RedownloadForbiddenForOutOfScopeContact(t *testing.T) {
	f := newGrantFixture(t) // agent scoped to account A; contact under account B

	msg := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    f.org.ID,
		ContactID:         f.contact.ID,
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeImage,
		MediaURL:          "images/gone.jpg",
		WhatsAppMessageID: "wamid.REDL1",
		WhatsAppAccount:   f.accB.Name,
	}
	require.NoError(t, f.app.DB.Create(msg).Error)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod("POST")
	testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
	testutil.SetPathParam(req, "message_id", msg.ID.String())
	require.NoError(t, f.app.RedownloadMedia(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "Access denied")
}

// TestReview2_CurrentCollaboratorFullAccess (finding: collaborators were
// collapsed into the historical read-only bucket): an ACTIVE collaborator on
// a cross-account conversation gets a writable access mode and passes the
// read-only gate on writes.
func TestReview2_CurrentCollaboratorFullAccess(t *testing.T) {
	f := newGrantFixture(t)

	// Make the agent an active collaborator on the account-B contact
	// (directly in metadata — the same state InviteCollaborator produces).
	f.contact.Metadata = models.JSONB{"collaborators": []any{
		map[string]any{"user_id": f.agent.ID.String(), "name": "Scoped Agent",
			"role": "agent", "joined_at": time.Now()},
	}}
	require.NoError(t, f.app.DB.Model(&models.Contact{}).Where("id = ?", f.contact.ID).
		Update("metadata", f.contact.Metadata).Error)

	// Single-contact read exposes the collaborator mode with full flags.
	getReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(getReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(getReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.GetContact(getReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(getReq))

	var resp struct {
		Data struct {
			AccessMode string `json:"access_mode"`
			CanReply   bool   `json:"can_reply"`
			CanClose   bool   `json:"can_close"`
		} `json:"data"`
	}
	require.NoError(t, unmarshalEnvelope(getReq, &resp))
	assert.Equal(t, "collaborator", resp.Data.AccessMode, "active collaborator is a current participant, not historical")
	assert.True(t, resp.Data.CanReply)
	assert.True(t, resp.Data.CanClose)

	// Write probe: the tags write must NOT be refused as read-only.
	tagReq := testutil.NewJSONRequest(t, map[string]any{"tags": []string{"vip"}})
	tagReq.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(tagReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(tagReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.UpdateContactTags(tagReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(tagReq),
		"collaborator write must pass the read-only gate, got: %s", string(testutil.GetResponseBody(tagReq)))
}

// TestReview2_UpdateContactAssignMintsGrant (finding: UpdateContact wrote
// assigned_user_id directly, creating assignments with no grant): the update
// path must go through ChatLifecycle.Assign so the durable grant is minted,
// and clear_assigned_agent must release through the lifecycle.
func TestReview2_UpdateContactAssignMintsGrant(t *testing.T) {
	f := newGrantFixture(t)

	req := testutil.NewJSONRequest(t, map[string]any{"assigned_user_id": f.agent.ID.String()})
	req.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(req, f.org.ID, f.admin.ID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	require.NoError(t, f.app.UpdateContact(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req),
		"update-assign must succeed, got: %s", string(testutil.GetResponseBody(req)))

	assert.EqualValues(t, 1, f.activeGrantCount(t, f.agent.ID),
		"assignment via UpdateContact must mint the access grant")
	assigned := f.loadFresh(t)
	require.NotNil(t, assigned.AssignedUserID)
	assert.Equal(t, f.agent.ID, *assigned.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, assigned.EffectiveStatus())

	// Clearing releases through the lifecycle: assignee gone, chat pending.
	clearReq := testutil.NewJSONRequest(t, map[string]any{"clear_assigned_agent": true})
	clearReq.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(clearReq, f.org.ID, f.admin.ID)
	testutil.SetPathParam(clearReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.UpdateContact(clearReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(clearReq))

	released := f.loadFresh(t)
	assert.Nil(t, released.AssignedUserID, "clear_assigned_agent releases")
	assert.Equal(t, models.ChatStatusPending, released.EffectiveStatus())
}

// TestReview2_ScheduledListScopedByAccountSubset (finding: the org-wide
// scheduled list ignored account scoping for contacts:read holders): a
// contacts:read user assigned account A lists only account-A rows.
func TestReview2_ScheduledListScopedByAccountSubset(t *testing.T) {
	f := newGrantFixture(t)

	contactA := testutil.CreateTestContactWith(t, f.app.DB, f.org.ID, testutil.WithContactAccount(f.accA.Name))
	newSched := func(contact *models.Contact, account *models.WhatsAppAccount, body string) *models.ScheduledMessage {
		sm := &models.ScheduledMessage{
			OrganizationID:  f.org.ID,
			WhatsAppAccount: account.Name,
			ContactID:       contact.ID,
			MessageType:     models.MessageTypeText,
			Content:         body,
			ScheduledAt:     time.Now().Add(time.Hour),
			CreatedBy:       f.admin.ID,
		}
		require.NoError(t, f.app.DB.Create(sm).Error)
		return sm
	}
	inScope := newSched(contactA, f.accA, "account-a")
	outScope := newSched(f.contact, f.accB, "account-b")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
	require.NoError(t, f.app.ListScheduledMessages(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			ScheduledMessages []models.ScheduledMessage `json:"scheduled_messages"`
		} `json:"data"`
	}
	require.NoError(t, unmarshalEnvelope(req, &resp))

	ids := map[uuid.UUID]bool{}
	for _, sm := range resp.Data.ScheduledMessages {
		ids[sm.ID] = true
	}
	assert.True(t, ids[inScope.ID], "in-scope scheduled message is listed")
	assert.False(t, ids[outScope.ID],
		"account-B scheduled messages must be hidden from an account-A-only contacts:read user")
}
