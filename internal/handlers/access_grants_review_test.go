package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// unmarshalEnvelope decodes the standard {status, data} envelope into out.
func unmarshalEnvelope(req *fastglue.Request, out any) error {
	return json.Unmarshal(testutil.GetResponseBody(req), out)
}

// Explicit regression tests for the review findings on the assignment-grant
// feature: after the assignment moves on, the historical READ-ONLY holder is
// refused on EVERY mutation path — including the bypasses that existed before
// the hardening pass (template send, scheduled edit/cancel, assignment,
// grant management, media files, bulk release) — and a Release must end
// access even for the agent who closed the conversation.

// promoteToHistorical moves the fixture contact's assignment to another user,
// leaving `agent` with only the (active) grant — the canonical historical
// read-only state.
func promoteToHistorical(t *testing.T, f *grantFixture, newAssignee *models.User) {
	t.Helper()
	f.assignViaAPI(t, newAssignee)
}

// TestReview_ReleaseEndsAccessEvenForCloser (finding 1): after Release, the
// released user must lose visibility EVEN IF they were the one who closed the
// conversation (closed_by is audit-only). Order: assign → close → reassign →
// release grant → invisible.
func TestReview_ReleaseEndsAccessEvenForCloser(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)

	// Agent closes (allowed as current assignee).
	closeReq := newPUTRequest(t)
	testutil.SetAuthContext(closeReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(closeReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.CloseChat(closeReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(closeReq))

	// Release the (closed) assignee's grant directly — closed stays closed.
	del := testutil.NewRequest(t)
	del.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(del, f.org.ID, f.admin.ID)
	testutil.SetPathParam(del, "id", f.contact.ID.String())
	testutil.SetPathParam(del, "target_user_id", f.agent.ID.String())
	require.NoError(t, f.app.RevokeAccessGrant(del))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(del))

	visible := visibleContactIDs(t, f.app, f.org.ID, f.agent.ID)
	assert.False(t, visible[f.contact.ID],
		"after Release, closed_by must not keep the conversation visible")
	assert.EqualValues(t, 0, f.activeGrantCount(t, f.agent.ID))

	// Audit stamp preserved (closed_by still recorded for audit).
	gone := f.loadFresh(t)
	assert.Equal(t, f.agent.ID.String(), gone.ClosedByUserID(),
		"closed_by remains as an audit record after access ends")
}

// TestReview_TemplateSendForbiddenForHistorical (finding 2): a historical
// holder cannot send a template by contact_id — the scoped load + read-only
// guard in SendTemplateMessage refuses before any send is attempted.
func TestReview_TemplateSendForbiddenForHistorical(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)
	promoteToHistorical(t, f, f.other)

	template := testutil.CreateTestTemplate(t, f.app.DB, f.org.ID, f.accB.Name)

	req := testutil.NewJSONRequest(t, map[string]any{
		"contact_id":      f.contact.ID.String(),
		"template_id":     template.ID.String(),
		"template_params": map[string]string{},
	})
	testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
	require.NoError(t, f.app.SendTemplateMessage(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "Read-only access")
}

// TestReview_TemplateSendPhoneBypassForbidden: the phone-number branch must
// not become a scope bypass either — a scoped agent cannot template-send to
// another account's contact by phone.
func TestReview_TemplateSendPhoneBypassForbidden(t *testing.T) {
	f := newGrantFixture(t) // agent scoped to account A only
	template := testutil.CreateTestTemplate(t, f.app.DB, f.org.ID, f.accB.Name)

	req := testutil.NewJSONRequest(t, map[string]any{
		"phone_number":    f.contact.PhoneNumber,
		"template_id":     template.ID.String(),
		"template_params": map[string]string{},
	})
	testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
	require.NoError(t, f.app.SendTemplateMessage(req))
	// Scoped load misses → 404 (no existence leak, no send).
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "Contact not found")
}

// TestReview_ScheduledMutationsForbiddenForHistorical (finding 3): a
// historical holder cannot edit or cancel scheduled messages on the
// conversation.
func TestReview_ScheduledMutationsForbiddenForHistorical(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)

	// Agent schedules a message (allowed as current assignee).
	createReq := testutil.NewJSONRequest(t, map[string]any{
		"type":         "text",
		"content":      map[string]any{"body": "later"},
		"scheduled_at": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	})
	testutil.SetAuthContext(createReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(createReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.CreateScheduledMessage(createReq))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(createReq),
		"current assignee may schedule")

	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, unmarshalEnvelope(createReq, &created))
	smID := created.Data.ID
	require.NotEmpty(t, smID)

	// Assignment moves on → historical read-only.
	promoteToHistorical(t, f, f.other)

	// Edit → refused.
	editReq := testutil.NewJSONRequest(t, map[string]any{
		"type":         "text",
		"content":      map[string]any{"body": "edited"},
		"scheduled_at": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	})
	editReq.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(editReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(editReq, "id", smID)
	require.NoError(t, f.app.UpdateScheduledMessage(editReq))
	testutil.AssertErrorResponse(t, editReq, fasthttp.StatusForbidden, "Read-only access")

	// Cancel → refused.
	cancelReq := testutil.NewRequest(t)
	cancelReq.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(cancelReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(cancelReq, "id", smID)
	require.NoError(t, f.app.CancelScheduledMessage(cancelReq))
	testutil.AssertErrorResponse(t, cancelReq, fasthttp.StatusForbidden, "Read-only access")

	// Untouched in DB.
	var sm models.ScheduledMessage
	require.NoError(t, f.app.DB.First(&sm, "id = ?", smID).Error)
	assert.Equal(t, models.ScheduledMessageStatusPending, sm.Status)
}

// TestReview_AssignAndGrantMgmtForbiddenForHistorical (finding 4): a
// historical holder can neither reassign the conversation (to anyone) nor
// manage its grants.
func TestReview_AssignAndGrantMgmtForbiddenForHistorical(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)
	promoteToHistorical(t, f, f.other)

	// Assign to a THIRD user → refused.
	third := scopedUser(t, f.app, f.org, "Third Agent", []string{"contacts:read"}, f.accA.ID)
	assignReq := testutil.NewJSONRequest(t, map[string]any{"user_id": third.ID.String()})
	assignReq.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(assignReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(assignReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.AssignContact(assignReq))
	testutil.AssertErrorResponse(t, assignReq, fasthttp.StatusForbidden, "Read-only access")

	// List grants → refused.
	listReq := testutil.NewGETRequest(t)
	testutil.SetAuthContext(listReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(listReq, "id", f.contact.ID.String())
	require.NoError(t, f.app.ListAccessGrants(listReq))
	testutil.AssertErrorResponse(t, listReq, fasthttp.StatusForbidden, "Read-only access")

	// Release another user's grant → refused.
	revokeReq := testutil.NewRequest(t)
	revokeReq.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(revokeReq, f.org.ID, f.agent.ID)
	testutil.SetPathParam(revokeReq, "id", f.contact.ID.String())
	testutil.SetPathParam(revokeReq, "target_user_id", f.other.ID.String())
	require.NoError(t, f.app.RevokeAccessGrant(revokeReq))
	testutil.AssertErrorResponse(t, revokeReq, fasthttp.StatusForbidden, "Read-only access")

	// The other user's grant is untouched.
	assert.EqualValues(t, 1, f.activeGrantCount(t, f.other.ID))
}

// TestReview_MediaFileScopedByAccount (finding 5): media file serving follows
// chat visibility for EVERY caller — an account-scoped agent cannot read
// another account's media by message ID; a grant holder can.
func TestReview_MediaFileScopedByAccount(t *testing.T) {
	f := newGrantFixture(t)

	// A media message on the foreign account's contact.
	msg := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: f.org.ID,
		ContactID:      f.contact.ID,
		Direction:      models.DirectionIncoming,
		MessageType:    models.MessageTypeImage,
		MediaURL:       "images/review-test.jpg",
	}
	require.NoError(t, f.app.DB.Create(msg).Error)

	mediaReq := func(user *models.User) int {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, f.org.ID, user.ID)
		testutil.SetPathParam(req, "message_id", msg.ID.String())
		require.NoError(t, f.app.ServeMedia(req))
		return testutil.GetResponseStatusCode(req)
	}

	// Account-A agent (no involvement) → refused by the scope gate.
	assert.Equal(t, fasthttp.StatusForbidden, mediaReq(f.agent),
		"account-scoped agent must not read another account's media")

	// Grant holder passes the scope gate (then hits plain file-not-found).
	f.assignViaAPI(t, f.agent)
	promoteToHistorical(t, f, f.other)
	assert.NotEqual(t, fasthttp.StatusForbidden, mediaReq(f.agent),
		"historical grant holder keeps media READ access")
}

// TestReview_BulkReleaseRefusesHistoricalContacts: bulk release must not let
// a historical holder release conversations they only hold read-only grants
// on — they surface as failed entries and the row is untouched.
func TestReview_BulkReleaseRefusesHistoricalContacts(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)
	promoteToHistorical(t, f, f.other)

	req := testutil.NewJSONRequest(t, map[string]any{
		"contact_ids": []string{f.contact.ID.String()},
	})
	testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
	require.NoError(t, f.app.BulkReleaseChats(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			ReleasedIDs []string         `json:"released_ids"`
			Failed      []map[string]any `json:"failed"`
		} `json:"data"`
	}
	require.NoError(t, unmarshalEnvelope(req, &resp))
	assert.Empty(t, resp.Data.ReleasedIDs, "nothing released")
	require.Len(t, resp.Data.Failed, 1)
	assert.Equal(t, "read-only", resp.Data.Failed[0]["reason"])

	fresh := f.loadFresh(t)
	require.NotNil(t, fresh.AssignedUserID)
	assert.Equal(t, f.other.ID, *fresh.AssignedUserID, "assignment untouched")
}

// TestReview_CurrentAssigneeStillReleasableByAdmin (finding 6 companion):
// the atomic Release path still releases an open assignee exactly as before —
// regression guard for the transaction refactor.
func TestReview_CurrentAssigneeStillReleasableByAdmin(t *testing.T) {
	f := newGrantFixture(t)
	f.assignViaAPI(t, f.agent)

	del := testutil.NewRequest(t)
	del.RequestCtx.Request.Header.SetMethod("DELETE")
	testutil.SetAuthContext(del, f.org.ID, f.admin.ID)
	testutil.SetPathParam(del, "id", f.contact.ID.String())
	testutil.SetPathParam(del, "target_user_id", f.agent.ID.String())
	require.NoError(t, f.app.RevokeAccessGrant(del))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(del))

	fresh := f.loadFresh(t)
	assert.Nil(t, fresh.AssignedUserID)
	assert.Equal(t, models.ChatStatusPending, fresh.EffectiveStatus())
	assert.EqualValues(t, 0, f.activeGrantCount(t, f.agent.ID))
}
