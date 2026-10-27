package handlers_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Send-account authorization: a user may send only from WhatsApp accounts
// they are assigned (or every account when they have no assignment, or are a
// super admin). allow_cross_account_send lets a current participant reply from
// an unassigned account already present in the conversation.

// sendAuthzFixture: the agent is assigned account A only and is the assignee
// of a conversation whose primary account is B. C has an incoming message in
// the conversation; D has none.
type sendAuthzFixture struct {
	app     *handlers.App
	mock    *mockGowaServer
	org     *models.Organization
	role    *models.CustomRole
	accA    *models.WhatsAppAccount
	accB    *models.WhatsAppAccount
	accC    *models.WhatsAppAccount
	accD    *models.WhatsAppAccount
	agent   *models.User
	contact *models.Contact
}

func newSendAuthzFixture(t *testing.T) *sendAuthzFixture {
	t.Helper()
	mock := newMockGowaServer()
	t.Cleanup(mock.close)
	app := newMsgTestApp(t, mock)
	org := testutil.CreateTestOrganization(t, app.DB)

	f := &sendAuthzFixture{app: app, mock: mock, org: org}
	f.accA = createTestAccount(t, app, org.ID)
	f.accB = createTestAccount(t, app, org.ID)
	f.accC = createTestAccount(t, app, org.ID)
	f.accD = createTestAccount(t, app, org.ID)

	f.role = testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "send-agent",
		[]string{"chat:read", "chat:write", "contacts:read", "chat.revoke:write", "campaigns:read", "campaigns:write", "campaigns:execute"})
	f.agent = testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&f.role.ID))
	testutil.AssignAccountToUser(t, app.DB, f.agent.ID, f.accA.ID)

	f.contact = testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(f.accB.Name), testutil.WithAssignee(f.agent.ID))
	f.addMessage(t, f.accC.Name, models.DirectionIncoming)
	return f
}

func (f *sendAuthzFixture) addMessage(t *testing.T, accountName string, direction models.Direction) *models.Message {
	t.Helper()
	msg := &models.Message{
		OrganizationID:    f.org.ID,
		WhatsAppAccount:   accountName,
		ContactID:         f.contact.ID,
		WhatsAppMessageID: "wamid-" + uuid.New().String()[:8],
		Direction:         direction,
		MessageType:       models.MessageTypeText,
		Content:           "fixture message",
		Status:            models.MessageStatusDelivered,
	}
	require.NoError(t, f.app.DB.Create(msg).Error)
	return msg
}

// assertNothingSent checks that no message beyond the fixture rows was
// created in the conversation. (The GOWA mock is not used for this: clients
// are cached per base URL, so parallel tests share request logs.)
func (f *sendAuthzFixture) assertNothingSent(t *testing.T) {
	t.Helper()
	var n int64
	require.NoError(t, f.app.DB.Model(&models.Message{}).
		Where("contact_id = ? AND content <> ?", f.contact.ID, "fixture message").Count(&n).Error)
	assert.Zero(t, n, "a denied write must not create a message")
}

func (f *sendAuthzFixture) setPolicy(t *testing.T, settings models.JSONB) {
	t.Helper()
	f.org.Settings = settings
	require.NoError(t, f.app.DB.Save(f.org).Error)
}

func (f *sendAuthzFixture) sendText(t *testing.T, userID uuid.UUID, account string) *fastglue.Request {
	t.Helper()
	body := map[string]any{"type": "text", "content": map[string]any{"body": "hello"}}
	if account != "" {
		body["whatsapp_account"] = account
	}
	req := testutil.NewJSONRequest(t, body)
	testutil.SetAuthContext(req, f.org.ID, userID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	require.NoError(t, f.app.SendMessage(req))
	return req
}

// sentAccountName returns the whatsapp_account of the message a successful
// SendMessage created.
func sentAccountName(t *testing.T, req *fastglue.Request) string {
	t.Helper()
	var resp struct {
		Data handlers.MessageResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	return resp.Data.WhatsAppAccount
}

func assertErrorCode(t *testing.T, req *fastglue.Request, status int, code string) {
	t.Helper()
	body := string(testutil.GetResponseBody(req))
	require.Equal(t, status, testutil.GetResponseStatusCode(req), "body: %s", body)
	assert.Contains(t, body, code)
}

// awaitSendAudit waits for the async audit row of a send-policy decision.
func awaitSendAudit(t *testing.T, app *handlers.App, orgID, userID uuid.UUID, action models.AuditAction) models.AuditLog {
	t.Helper()
	var entry models.AuditLog
	require.Eventually(t, func() bool {
		return app.DB.Where("organization_id = ? AND user_id = ? AND action = ? AND resource_type = ?",
			orgID, userID, action, "send_account").First(&entry).Error == nil
	}, 3*time.Second, 25*time.Millisecond, "expected %s audit entry", action)
	return entry
}

func TestSendMessage_SendAccountPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		settings   models.JSONB
		account    func(f *sendAuthzFixture) string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "assigned account",
			account:    func(f *sendAuthzFixture) string { return f.accA.Name },
			wantStatus: fasthttp.StatusOK,
		},
		{
			name:       "unassigned account with messages, cross-account off",
			account:    func(f *sendAuthzFixture) string { return f.accC.Name },
			wantStatus: fasthttp.StatusForbidden,
			wantCode:   "account_not_allowed",
		},
		{
			name:       "unassigned account with messages, cross-account on",
			settings:   models.JSONB{"allow_cross_account_send": true},
			account:    func(f *sendAuthzFixture) string { return f.accC.Name },
			wantStatus: fasthttp.StatusOK,
		},
		{
			name:       "conversation's own account, cross-account on",
			settings:   models.JSONB{"allow_cross_account_send": true},
			account:    func(f *sendAuthzFixture) string { return f.accB.Name },
			wantStatus: fasthttp.StatusOK,
		},
		{
			name:       "unassigned account without messages, cross-account on",
			settings:   models.JSONB{"allow_cross_account_send": true},
			account:    func(f *sendAuthzFixture) string { return f.accD.Name },
			wantStatus: fasthttp.StatusForbidden,
			wantCode:   "account_not_allowed",
		},
		{
			name:       "unknown account",
			account:    func(*sendAuthzFixture) string { return "no-such-account" },
			wantStatus: fasthttp.StatusNotFound,
			wantCode:   "account_not_found",
		},
		{
			name:       "audit-only mode lets a denied send through",
			settings:   models.JSONB{"enforce_send_account_policy": false},
			account:    func(f *sendAuthzFixture) string { return f.accD.Name },
			wantStatus: fasthttp.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newSendAuthzFixture(t)
			if tc.settings != nil {
				f.setPolicy(t, tc.settings)
			}
			account := tc.account(f)
			req := f.sendText(t, f.agent.ID, account)
			if tc.wantCode != "" {
				assertErrorCode(t, req, tc.wantStatus, tc.wantCode)
				return
			}
			require.Equal(t, tc.wantStatus, testutil.GetResponseStatusCode(req), string(testutil.GetResponseBody(req)))
			assert.Equal(t, account, sentAccountName(t, req))
		})
	}
}

func TestSendMessage_DeniedSendIsAuditedWithoutContent(t *testing.T) {
	t.Parallel()
	f := newSendAuthzFixture(t)

	req := f.sendText(t, f.agent.ID, f.accD.Name)
	assertErrorCode(t, req, fasthttp.StatusForbidden, "account_not_allowed")
	f.assertNothingSent(t)

	entry := awaitSendAudit(t, f.app, f.org.ID, f.agent.ID, models.AuditActionSendDenied)
	assert.Equal(t, f.accD.ID, entry.ResourceID)
	changes, err := json.Marshal(entry.Changes)
	require.NoError(t, err)
	assert.Contains(t, string(changes), "send_message")
	assert.Contains(t, string(changes), f.contact.ID.String())
	assert.NotContains(t, string(changes), "hello", "audit must not carry message text")
}

func TestSendMessage_AuditOnlyModeRecordsWouldBeDenied(t *testing.T) {
	t.Parallel()
	f := newSendAuthzFixture(t)
	f.setPolicy(t, models.JSONB{"enforce_send_account_policy": false})

	req := f.sendText(t, f.agent.ID, f.accD.Name)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
	awaitSendAudit(t, f.app, f.org.ID, f.agent.ID, models.AuditActionSendWouldBeDenied)
}

func TestSendMessage_ImplicitAccountSelection(t *testing.T) {
	t.Parallel()

	t.Run("no allowed account is account_required", func(t *testing.T) {
		t.Parallel()
		f := newSendAuthzFixture(t)
		// Contact on B, messages on C, no default outgoing account: nothing the
		// agent (assigned A) may send from — never an arbitrary org account.
		req := f.sendText(t, f.agent.ID, "")
		assertErrorCode(t, req, fasthttp.StatusBadRequest, "account_required")
		f.assertNothingSent(t)
	})

	t.Run("picks an assigned account present in the conversation", func(t *testing.T) {
		t.Parallel()
		f := newSendAuthzFixture(t)
		f.addMessage(t, f.accA.Name, models.DirectionIncoming)
		req := f.sendText(t, f.agent.ID, "")
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req), string(testutil.GetResponseBody(req)))
		assert.Equal(t, f.accA.Name, sentAccountName(t, req))
	})

	t.Run("prefers an allowed default outgoing account", func(t *testing.T) {
		t.Parallel()
		f := newSendAuthzFixture(t)
		require.NoError(t, f.app.DB.Model(f.accA).Update("is_default_outgoing", true).Error)
		req := f.sendText(t, f.agent.ID, "")
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req), string(testutil.GetResponseBody(req)))
		assert.Equal(t, f.accA.Name, sentAccountName(t, req))
	})
}

func TestSendMessage_SendAccountPolicyUsers(t *testing.T) {
	t.Parallel()

	t.Run("super admin may send from any org account", func(t *testing.T) {
		t.Parallel()
		f := newSendAuthzFixture(t)
		admin := testutil.CreateTestUser(t, f.app.DB, f.org.ID, testutil.WithSuperAdmin(), testutil.WithRoleID(&f.role.ID))
		testutil.AssignAccountToUser(t, f.app.DB, admin.ID, f.accA.ID)
		require.NoError(t, f.app.DB.Model(f.contact).Update("assigned_user_id", admin.ID).Error)
		req := f.sendText(t, admin.ID, f.accD.Name)
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req), string(testutil.GetResponseBody(req)))
	})

	t.Run("user with no account assignment keeps the full-org fallback", func(t *testing.T) {
		t.Parallel()
		f := newSendAuthzFixture(t)
		unscoped := testutil.CreateTestUser(t, f.app.DB, f.org.ID, testutil.WithRoleID(&f.role.ID))
		require.NoError(t, f.app.DB.Model(f.contact).Update("assigned_user_id", unscoped.ID).Error)
		req := f.sendText(t, unscoped.ID, f.accD.Name)
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req), string(testutil.GetResponseBody(req)))
	})

	t.Run("non-participant outside the account scope cannot see the conversation", func(t *testing.T) {
		t.Parallel()
		f := newSendAuthzFixture(t)
		outsider := testutil.CreateTestUser(t, f.app.DB, f.org.ID, testutil.WithRoleID(&f.role.ID))
		testutil.AssignAccountToUser(t, f.app.DB, outsider.ID, f.accA.ID)
		req := f.sendText(t, outsider.ID, f.accA.Name)
		assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
		f.assertNothingSent(t)
	})

	t.Run("historical read-only access is refused before the account check", func(t *testing.T) {
		t.Parallel()
		f := newSendAuthzFixture(t)
		other := testutil.CreateTestUser(t, f.app.DB, f.org.ID, testutil.WithRoleID(&f.role.ID))
		require.NoError(t, f.app.DB.Model(f.contact).Update("assigned_user_id", other.ID).Error)
		require.NoError(t, f.app.DB.Create(&models.ContactAssignmentAccessGrant{
			OrganizationID: f.org.ID,
			ContactID:      f.contact.ID,
			UserID:         f.agent.ID,
			GrantedBy:      other.ID,
		}).Error)
		req := f.sendText(t, f.agent.ID, f.accA.Name)
		assertErrorCode(t, req, fasthttp.StatusForbidden, "historical_assignment_read_only")
	})

	t.Run("account of another organization is not found", func(t *testing.T) {
		t.Parallel()
		f := newSendAuthzFixture(t)
		otherOrg := testutil.CreateTestOrganization(t, f.app.DB)
		foreign := createTestAccount(t, f.app, otherOrg.ID)
		req := f.sendText(t, f.agent.ID, foreign.Name)
		assertErrorCode(t, req, fasthttp.StatusNotFound, "account_not_found")
	})

	t.Run("deleted account is not found", func(t *testing.T) {
		t.Parallel()
		f := newSendAuthzFixture(t)
		require.NoError(t, f.app.DB.Delete(f.accA).Error)
		req := f.sendText(t, f.agent.ID, f.accA.Name)
		assertErrorCode(t, req, fasthttp.StatusNotFound, "account_not_found")
	})
}

// newMediaSendRequest builds an authenticated multipart POST /messages/media.
func newMediaSendRequest(t *testing.T, orgID, userID, contactID uuid.UUID, account string) *fastglue.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	require.NoError(t, writer.WriteField("contact_id", contactID.String()))
	require.NoError(t, writer.WriteField("type", "image"))
	require.NoError(t, writer.WriteField("whatsapp_account", account))
	part, err := writer.CreateFormFile("file", "pic.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("not-really-a-png"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&fasthttp.Request{}, nil, nil)
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.Header.SetContentType(writer.FormDataContentType())
	ctx.Request.SetBody(buf.Bytes())
	req := &fastglue.Request{RequestCtx: ctx}
	testutil.SetAuthContext(req, orgID, userID)
	return req
}

// Every write path that acts as an account refuses an unassigned one.
func TestSendPaths_RefuseUnassignedAccount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(t *testing.T, f *sendAuthzFixture) *fastglue.Request
	}{
		{
			name: "media",
			call: func(t *testing.T, f *sendAuthzFixture) *fastglue.Request {
				f.app.Config.Storage.LocalPath = t.TempDir()
				req := newMediaSendRequest(t, f.org.ID, f.agent.ID, f.contact.ID, f.accD.Name)
				require.NoError(t, f.app.SendMediaMessage(req))
				return req
			},
		},
		{
			name: "template",
			call: func(t *testing.T, f *sendAuthzFixture) *fastglue.Request {
				template := testutil.CreateTestTemplate(t, f.app.DB, f.org.ID, f.accD.Name)
				req := testutil.NewJSONRequest(t, map[string]any{
					"contact_id":   f.contact.ID.String(),
					"template_id":  template.ID.String(),
					"account_name": f.accD.Name,
				})
				testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
				require.NoError(t, f.app.SendTemplateMessage(req))
				return req
			},
		},
		{
			name: "reaction on a message of an unassigned account",
			call: func(t *testing.T, f *sendAuthzFixture) *fastglue.Request {
				msg := f.addMessage(t, f.accD.Name, models.DirectionIncoming)
				req := testutil.NewJSONRequest(t, map[string]any{"emoji": "👍"})
				testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
				testutil.SetPathParam(req, "id", f.contact.ID.String())
				testutil.SetPathParam(req, "message_id", msg.ID.String())
				require.NoError(t, f.app.SendReaction(req))
				return req
			},
		},
		{
			name: "typing on an unassigned conversation account",
			call: func(t *testing.T, f *sendAuthzFixture) *fastglue.Request {
				req := testutil.NewJSONRequest(t, map[string]any{"action": "start"})
				testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
				testutil.SetPathParam(req, "id", f.contact.ID.String())
				require.NoError(t, f.app.SendTypingIndicator(req))
				return req
			},
		},
		{
			name: "revoke of a message sent from an unassigned account",
			call: func(t *testing.T, f *sendAuthzFixture) *fastglue.Request {
				msg := f.addMessage(t, f.accD.Name, models.DirectionOutgoing)
				req := testutil.NewJSONRequest(t, map[string]any{})
				testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
				testutil.SetPathParam(req, "id", f.contact.ID.String())
				testutil.SetPathParam(req, "message_id", msg.ID.String())
				require.NoError(t, f.app.RevokeMessage(req))
				return req
			},
		},
		{
			name: "scheduled message",
			call: func(t *testing.T, f *sendAuthzFixture) *fastglue.Request {
				req := testutil.NewJSONRequest(t, map[string]any{
					"type":             "text",
					"content":          map[string]any{"body": "later"},
					"whatsapp_account": f.accD.Name,
					"scheduled_at":     time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
				})
				testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
				testutil.SetPathParam(req, "id", f.contact.ID.String())
				require.NoError(t, f.app.CreateScheduledMessage(req))
				return req
			},
		},
		{
			name: "status",
			call: func(t *testing.T, f *sendAuthzFixture) *fastglue.Request {
				req := testutil.NewJSONRequest(t, map[string]any{
					"type": "text", "message": "status text", "whatsapp_account": f.accD.Name,
				})
				testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
				require.NoError(t, f.app.SendStatus(req))
				return req
			},
		},
		{
			name: "campaign create",
			call: func(t *testing.T, f *sendAuthzFixture) *fastglue.Request {
				template := testutil.CreateTestTemplate(t, f.app.DB, f.org.ID, f.accD.Name)
				req := testutil.NewJSONRequest(t, map[string]any{
					"name":             "blast",
					"whatsapp_account": f.accD.Name,
					"template_id":      template.ID.String(),
				})
				testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
				require.NoError(t, f.app.CreateCampaign(req))
				return req
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newSendAuthzFixture(t)
			req := tc.call(t, f)
			assertErrorCode(t, req, fasthttp.StatusForbidden, "account_not_allowed")
			f.assertNothingSent(t)
		})
	}
}

func TestScheduledMessageProcessor_FailsWhenCreatorLostAccount(t *testing.T) {
	t.Parallel()
	f := newSendAuthzFixture(t)

	// Scheduled while assigned to D; the assignment is then revoked.
	testutil.AssignAccountToUser(t, f.app.DB, f.agent.ID, f.accD.ID)
	sm := createScheduledFixture(t, f.app, f.org.ID, f.contact.ID, f.accD.Name, f.agent.ID,
		models.ScheduledMessageStatusPending, time.Now().Add(-time.Minute))
	require.NoError(t, f.app.DB.Where("user_id = ? AND whats_app_account_id = ?", f.agent.ID, f.accD.ID).
		Delete(&models.UserWhatsAppAccount{}).Error)

	handlers.NewScheduledMessageProcessor(f.app, time.Minute).ProcessDue(time.Now())

	var stored models.ScheduledMessage
	require.NoError(t, f.app.DB.First(&stored, "id = ?", sm.ID).Error)
	assert.Equal(t, models.ScheduledMessageStatusFailed, stored.Status)
	assert.Contains(t, stored.ErrorMessage, "not allowed")
	assert.Nil(t, stored.SentMessageID, "no send after the creator lost the account")
	f.assertNothingSent(t)
}

func TestCampaignScheduler_FailsWhenCreatorLostAccount(t *testing.T) {
	t.Parallel()
	f := newSendAuthzFixture(t)
	template := testutil.CreateTestTemplate(t, f.app.DB, f.org.ID, f.accD.Name)
	due := time.Now().Add(-time.Minute)
	campaign := &models.BulkMessageCampaign{
		OrganizationID:  f.org.ID,
		Name:            "scheduled blast",
		WhatsAppAccount: f.accD.Name,
		TemplateID:      template.ID,
		Status:          models.CampaignStatusScheduled,
		ScheduledAt:     &due,
		CreatedBy:       f.agent.ID,
	}
	require.NoError(t, f.app.DB.Create(campaign).Error)

	handlers.NewCampaignSchedulerProcessor(f.app, time.Minute).ProcessDue(time.Now())

	var stored models.BulkMessageCampaign
	require.NoError(t, f.app.DB.First(&stored, "id = ?", campaign.ID).Error)
	assert.Equal(t, models.CampaignStatusFailed, stored.Status)
	awaitSendAudit(t, f.app, f.org.ID, f.agent.ID, models.AuditActionSendDenied)
}

func TestListContactSendAccounts(t *testing.T) {
	t.Parallel()
	f := newSendAuthzFixture(t)
	f.addMessage(t, f.accA.Name, models.DirectionIncoming)

	list := func(settings models.JSONB) map[string]handlers.SendAccountOption {
		f.setPolicy(t, settings)
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, f.org.ID, f.agent.ID)
		testutil.SetPathParam(req, "id", f.contact.ID.String())
		require.NoError(t, f.app.ListContactSendAccounts(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req), string(testutil.GetResponseBody(req)))
		body := string(testutil.GetResponseBody(req))
		for _, secret := range []string{"gowa_base_url", "gowa_device_id", "password", "token"} {
			assert.NotContains(t, body, secret)
		}
		var resp struct {
			Data struct {
				Accounts []handlers.SendAccountOption `json:"accounts"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		byName := map[string]handlers.SendAccountOption{}
		for _, a := range resp.Data.Accounts {
			byName[a.Name] = a
		}
		return byName
	}

	got := list(models.JSONB{})
	require.Len(t, got, 3, "contact account B plus message accounts C and A")
	assert.True(t, got[f.accA.Name].CanSend)
	assert.Nil(t, got[f.accA.Name].Reason)
	for _, name := range []string{f.accB.Name, f.accC.Name} {
		assert.False(t, got[name].CanSend, name)
		require.NotNil(t, got[name].Reason, name)
		assert.Equal(t, "account_not_allowed", *got[name].Reason)
	}

	got = list(models.JSONB{"allow_cross_account_send": true})
	assert.True(t, got[f.accB.Name].CanSend, "conversation account with cross-account on")
	assert.True(t, got[f.accC.Name].CanSend, "account with messages with cross-account on")
}

func TestListContactSendAccounts_RefusesNonParticipant(t *testing.T) {
	t.Parallel()
	f := newSendAuthzFixture(t)
	viewer := testutil.CreateTestUser(t, f.app.DB, f.org.ID, testutil.WithRoleID(&f.role.ID))
	testutil.AssignAccountToUser(t, f.app.DB, viewer.ID, f.accB.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, f.org.ID, viewer.ID)
	testutil.SetPathParam(req, "id", f.contact.ID.String())
	require.NoError(t, f.app.ListContactSendAccounts(req))
	assertErrorCode(t, req, fasthttp.StatusForbidden, "conversation_restricted")
}

func TestOrganizationSettings_SendAccountPolicy(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	get := func() handlers.OrganizationSettings {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, admin.ID)
		require.NoError(t, app.GetOrganizationSettings(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
		var resp struct {
			Data struct {
				Settings handlers.OrganizationSettings `json:"settings"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		return resp.Data.Settings
	}

	s := get()
	assert.False(t, s.AllowCrossAccountSend, "cross-account send is off by default")
	assert.True(t, s.EnforceSendAccountPolicy, "the policy is enforced by default")

	req := testutil.NewJSONRequest(t, map[string]any{"allow_cross_account_send": true, "enforce_send_account_policy": false})
	req.RequestCtx.Request.Header.SetMethod("PUT")
	testutil.SetAuthContext(req, org.ID, admin.ID)
	require.NoError(t, app.UpdateOrganizationSettings(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	s = get()
	assert.True(t, s.AllowCrossAccountSend)
	assert.False(t, s.EnforceSendAccountPolicy)
}
