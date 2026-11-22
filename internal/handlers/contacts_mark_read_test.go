package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// newMarkReadRequest builds a fastglue POST request for MarkContactRead tests.
func newMarkReadRequest(t *testing.T) *fastglue.Request {
	t.Helper()
	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&fasthttp.Request{}, nil, nil)
	ctx.Request.Header.SetMethod("POST")
	return &fastglue.Request{RequestCtx: ctx}
}

// createUnreadIncomingMessage inserts an incoming, not-yet-read message for
// the contact so the mark-read effect is observable.
func createUnreadIncomingMessage(t *testing.T, app *handlers.App, orgID, contactID uuid.UUID) *models.Message {
	t.Helper()
	msg := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		ContactID:      contactID,
		Direction:      models.DirectionIncoming,
		MessageType:    models.MessageTypeText,
		Content:        "hello",
		Status:         models.MessageStatusDelivered,
	}
	require.NoError(t, app.DB.Create(msg).Error)
	return msg
}

// readSeqOf returns the messages.read_seq assigned to msgID at insert time
// (INSERT order, not created_at).
func readSeqOf(t *testing.T, app *handlers.App, msgID uuid.UUID) int64 {
	t.Helper()
	var seq int64
	require.NoError(t, app.DB.Raw("SELECT read_seq FROM messages WHERE id = ?", msgID).Scan(&seq).Error)
	return seq
}

// setReadWatermark moves the conversation-wide read floor for contactID.
// The column is nullable with no default: a contact that never had automated
// handling has NULL here.
func setReadWatermark(t *testing.T, app *handlers.App, contactID uuid.UUID, seq int64) {
	t.Helper()
	require.NoError(t, app.DB.Exec("UPDATE contacts SET read_watermark_seq = ? WHERE id = ?", seq, contactID).Error)
}

// setViewerReadSeq records userID's own read position in the conversation.
func setViewerReadSeq(t *testing.T, app *handlers.App, orgID, contactID, userID uuid.UUID, seq int64) {
	t.Helper()
	require.NoError(t, app.DB.Create(&models.ContactReadState{
		ContactID:      contactID,
		UserID:         userID,
		OrganizationID: orgID,
		LastReadSeq:    seq,
	}).Error)
}

// unreadCountViaGetContact reads the sidebar badge for contactID as seen by
// userID through the exported endpoint (buildContactResponse →
// unreadCountsForUser).
func unreadCountViaGetContact(t *testing.T, app *handlers.App, orgID, userID, contactID uuid.UUID) int {
	t.Helper()
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgID, userID)
	testutil.SetPathParam(req, "id", contactID.String())
	require.NoError(t, app.GetContact(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	var resp struct {
		Data handlers.ContactResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	return resp.Data.UnreadCount
}

// unreadFixture is one contact with three incoming messages, inserted in
// order so their read_seq values ascend. seqs[i] is message i's read_seq.
type unreadFixture struct {
	app         *handlers.App
	orgID       uuid.UUID
	viewer      uuid.UUID
	adminRoleID uuid.UUID
	contact     *models.Contact
	seqs        []int64
}

func newUnreadFixture(t *testing.T) unreadFixture {
	t.Helper()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	viewer := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	seqs := make([]int64, 0, 3)
	for i := 0; i < 3; i++ {
		msg := createUnreadIncomingMessage(t, app, org.ID, contact.ID)
		seqs = append(seqs, readSeqOf(t, app, msg.ID))
	}
	return unreadFixture{app: app, orgID: org.ID, viewer: viewer.ID, adminRoleID: adminRole.ID, contact: contact, seqs: seqs}
}

// TestApp_UnreadCount_ReadThreshold locks the agreed unread definition
// (counting-definitions §6): a message counts when its read_seq is above
// BOTH the viewer's read position and the conversation watermark; a missing
// value counts as 0 inside the max — it does not discard the other one.
// PostgreSQL GREATEST ignores NULL arguments (NULL only when every argument
// is NULL), so the COALESCE(GREATEST(...), 0) threshold already implements
// this; the one-sided-missing cases below pin it, including the two shapes
// the old (wrong) NULL-collapse reading predicted would count the whole
// inbound history.
func TestApp_UnreadCount_ReadThreshold(t *testing.T) {
	t.Run("watermark only, viewer never opened", func(t *testing.T) {
		f := newUnreadFixture(t)
		setReadWatermark(t, f.app, f.contact.ID, f.seqs[0])
		assert.Equal(t, 2, unreadCountViaGetContact(t, f.app, f.orgID, f.viewer, f.contact.ID),
			"messages above the watermark count even with no read-state row")
	})

	t.Run("read position only, no watermark", func(t *testing.T) {
		f := newUnreadFixture(t)
		setViewerReadSeq(t, f.app, f.orgID, f.contact.ID, f.viewer, f.seqs[1])
		assert.Equal(t, 1, unreadCountViaGetContact(t, f.app, f.orgID, f.viewer, f.contact.ID),
			"messages above the viewer's read position count even with no watermark")
	})

	t.Run("both set, the greater wins", func(t *testing.T) {
		f := newUnreadFixture(t)
		setReadWatermark(t, f.app, f.contact.ID, f.seqs[0])
		setViewerReadSeq(t, f.app, f.orgID, f.contact.ID, f.viewer, f.seqs[1])
		assert.Equal(t, 1, unreadCountViaGetContact(t, f.app, f.orgID, f.viewer, f.contact.ID))
	})

	t.Run("watermark above the read position", func(t *testing.T) {
		f := newUnreadFixture(t)
		setReadWatermark(t, f.app, f.contact.ID, f.seqs[1])
		setViewerReadSeq(t, f.app, f.orgID, f.contact.ID, f.viewer, f.seqs[0])
		assert.Equal(t, 1, unreadCountViaGetContact(t, f.app, f.orgID, f.viewer, f.contact.ID),
			"the watermark also clears the badge for every viewer")
	})

	t.Run("neither set counts the whole inbound history", func(t *testing.T) {
		f := newUnreadFixture(t)
		assert.Equal(t, 3, unreadCountViaGetContact(t, f.app, f.orgID, f.viewer, f.contact.ID))
	})

	t.Run("outgoing, revoked and failed above the threshold do not count", func(t *testing.T) {
		f := newUnreadFixture(t)
		setReadWatermark(t, f.app, f.contact.ID, f.seqs[0])
		// The org's own number sending (outgoing copy), and terminal
		// incoming messages: none of them is an unread customer message.
		for _, m := range []models.Message{
			{Direction: models.DirectionOutgoing, Status: models.MessageStatusSent},
			{Direction: models.DirectionIncoming, Status: models.MessageStatusRevoked},
			{Direction: models.DirectionIncoming, Status: models.MessageStatusFailed},
		} {
			m.BaseModel = models.BaseModel{ID: uuid.New()}
			m.OrganizationID = f.orgID
			m.ContactID = f.contact.ID
			m.MessageType = models.MessageTypeText
			m.Content = "not unread"
			require.NoError(t, f.app.DB.Create(&m).Error)
		}
		assert.Equal(t, 2, unreadCountViaGetContact(t, f.app, f.orgID, f.viewer, f.contact.ID))
	})

	t.Run("read state is per viewer", func(t *testing.T) {
		f := newUnreadFixture(t)
		setReadWatermark(t, f.app, f.contact.ID, f.seqs[0])
		// Viewer A read everything; viewer B never opened the conversation.
		setViewerReadSeq(t, f.app, f.orgID, f.contact.ID, f.viewer, f.seqs[2])
		other := testutil.CreateTestUser(t, f.app.DB, f.orgID, testutil.WithRoleID(&f.adminRoleID))
		assert.Equal(t, 0, unreadCountViaGetContact(t, f.app, f.orgID, f.viewer, f.contact.ID),
			"reading clears the badge for that viewer only")
		assert.Equal(t, 2, unreadCountViaGetContact(t, f.app, f.orgID, other.ID, f.contact.ID),
			"the other viewer still sees the messages above the watermark")
	})
}

// TestApp_MarkContactRead_PendingUnclaimed_Forbidden locks the claim-gate
// guard: an agent who cannot view a pending unclaimed chat's content must not
// be able to mark it read either — that would clear the unread badge and send
// read receipts for messages nobody has seen.
func TestApp_MarkContactRead_PendingUnclaimed_Forbidden(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	// Mirrors the real "agent" role: contacts:read but neither contacts:write
	// nor chat.collaborate:write.
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "read-agent",
		[]string{"contacts:read", "chat:read", "chat.assign:write"})
	agent := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	contact.SetStatus(models.ChatStatusPending)
	require.NoError(t, app.DB.Model(contact).Update("metadata", contact.Metadata).Error)
	msg := createUnreadIncomingMessage(t, app, org.ID, contact.ID)

	req := newMarkReadRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	_ = app.MarkContactRead(req)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "Claim this chat")

	// The message must remain unread.
	var updated models.Message
	require.NoError(t, app.DB.First(&updated, "id = ?", msg.ID).Error)
	assert.Equal(t, models.MessageStatusDelivered, updated.Status,
		"message must stay unread behind the claim gate")
}

// TestApp_MarkContactRead_Assignee_Success verifies the happy path is intact:
// the assigned agent of an open chat can mark it read.
func TestApp_MarkContactRead_Assignee_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "read-agent",
		[]string{"contacts:read", "chat:read", "chat.assign:write"})
	agent := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	contact.AssignedUserID = &agent.ID
	contact.SetStatus(models.ChatStatusOpen)
	require.NoError(t, app.DB.Model(contact).Updates(map[string]any{
		"assigned_user_id": agent.ID,
		"metadata":         contact.Metadata,
	}).Error)
	msg := createUnreadIncomingMessage(t, app, org.ID, contact.ID)

	req := newMarkReadRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.MarkContactRead(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var updated models.Message
	require.NoError(t, app.DB.First(&updated, "id = ?", msg.ID).Error)
	assert.Equal(t, models.MessageStatusRead, updated.Status)
}
