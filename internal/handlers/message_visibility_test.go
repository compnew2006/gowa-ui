package handlers_test

import (
	"encoding/json"
	"strings"
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

// Message content is visible only to administrators and to the conversation's
// participants (assignee, collaborator, or active access grant). Managers and
// agents who are not part of a chat get no messages, even when the chat is in
// their account scope.
func TestGetMessages_ContentLimitedToAdministratorsAndParticipants(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	managerRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "manager",
		[]string{"chat:read", "contacts:read", "contacts:write"})
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)

	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	manager := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&managerRole.ID))
	assignee := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	outsider := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithAssignee(assignee.ID))
	require.NoError(t, app.DB.Create(&models.Message{
		OrganizationID: org.ID,
		ContactID:      contact.ID,
		Direction:      models.DirectionIncoming,
		MessageType:    models.MessageTypeText,
		Content:        "private hello",
		Status:         models.MessageStatusDelivered,
	}).Error)

	tests := []struct {
		name       string
		userID     uuid.UUID
		wantStatus int
		wantBody   string
	}{
		{name: "administrator reads any conversation", userID: admin.ID, wantStatus: fasthttp.StatusOK},
		{name: "assignee reads own conversation", userID: assignee.ID, wantStatus: fasthttp.StatusOK},
		{name: "manager not in the conversation is refused", userID: manager.ID,
			wantStatus: fasthttp.StatusForbidden, wantBody: "conversation_restricted"},
		{name: "agent not in the conversation is refused", userID: outsider.ID,
			wantStatus: fasthttp.StatusForbidden, wantBody: "conversation_restricted"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := testutil.NewGETRequest(t)
			testutil.SetAuthContext(req, org.ID, tc.userID)
			testutil.SetPathParam(req, "id", contact.ID.String())
			testutil.SetQueryParam(req, "limit", 50)

			require.NoError(t, app.GetMessages(req))
			assert.Equal(t, tc.wantStatus, testutil.GetResponseStatusCode(req))
			body := string(testutil.GetResponseBody(req))
			if tc.wantBody != "" {
				assert.True(t, strings.Contains(body, tc.wantBody), "body: %s", body)
				assert.False(t, strings.Contains(body, "private hello"), "restricted response leaked message text")
			}
		})
	}
}

// Ghost Mode: only administrators are hidden from the collaborators list. A
// manager who is a collaborator is visible to everyone, like any participant.
func TestGetContact_GhostHidesOnlyAdministratorCollaborators(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	managerRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "manager",
		[]string{"contacts:read", "contacts:write"})
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)

	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	manager := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&managerRole.ID))
	agent := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	collaborator := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithAssignee(agent.ID))
	contact.AddCollaborator(models.Collaborator{UserID: admin.ID.String(), Name: "Admin"})
	contact.AddCollaborator(models.Collaborator{UserID: manager.ID.String(), Name: "Manager"})
	contact.AddCollaborator(models.Collaborator{UserID: collaborator.ID.String(), Name: "Collaborator"})
	require.NoError(t, app.DB.Model(contact).Update("metadata", contact.Metadata).Error)

	collaboratorNames := func(viewer uuid.UUID) []string {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, viewer)
		testutil.SetPathParam(req, "id", contact.ID.String())
		require.NoError(t, app.GetContact(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Collaborators []models.Collaborator `json:"collaborators"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		names := make([]string, 0, len(resp.Data.Collaborators))
		for _, c := range resp.Data.Collaborators {
			names = append(names, c.Name)
		}
		return names
	}

	assert.ElementsMatch(t, []string{"Manager", "Collaborator"}, collaboratorNames(agent.ID),
		"agents must not see the administrator collaborator, but must see managers")
	assert.ElementsMatch(t, []string{"Manager", "Collaborator"}, collaboratorNames(manager.ID),
		"the administrator stays hidden from managers too; only administrators see administrators")
	assert.ElementsMatch(t, []string{"Admin", "Manager", "Collaborator"}, collaboratorNames(admin.ID),
		"administrators see everyone")
}

// Last-message previews are message content too: the contact list shows them
// only to administrators and to the conversation's participants.
func TestListContacts_PreviewLimitedToAdministratorsAndParticipants(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)

	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	assignee := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	outsider := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))

	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithAssignee(assignee.ID))
	require.NoError(t, app.DB.Model(contact).Update("last_message_preview", "private preview").Error)

	previewSeenBy := func(viewer uuid.UUID) string {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, viewer)
		require.NoError(t, app.ListContacts(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				Contacts []struct {
					ID                 string `json:"id"`
					LastMessagePreview string `json:"last_message_preview"`
				} `json:"contacts"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		for _, c := range resp.Data.Contacts {
			if c.ID == contact.ID.String() {
				return c.LastMessagePreview
			}
		}
		t.Fatalf("contact %s missing from list", contact.ID)
		return ""
	}

	assert.Equal(t, "private preview", previewSeenBy(admin.ID))
	assert.Equal(t, "private preview", previewSeenBy(assignee.ID))
	assert.Empty(t, previewSeenBy(outsider.ID), "a non-participant agent must not see the preview")
}

// Writing into a conversation needs the same participant rule as reading it: a
// non-participant agent is refused even when the chat is in their account scope.
func TestSendMessage_NonParticipantRefused(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	outsider := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithAssignee(assignee.ID))

	req := testutil.NewJSONRequest(t, map[string]any{
		"type":    "text",
		"content": map[string]any{"body": "hello"},
	})
	testutil.SetAuthContext(req, org.ID, outsider.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.SendMessage(req))
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
	assert.Contains(t, string(testutil.GetResponseBody(req)), "conversation_restricted")
}

// Conversation notes are conversation content: only administrators and the
// conversation's participants may list them.
func TestConversationNotes_NonParticipantRefused(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	outsider := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithAssignee(assignee.ID))

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, outsider.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.ListConversationNotes(req))
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
	assert.Contains(t, string(testutil.GetResponseBody(req)), "conversation_restricted")
}

// A viewer who is only account-covered gets the restricted access mode: no reply
// and no close flags, and the participant gets the assignee mode.
func TestGetContact_AccessModeRestrictedForNonParticipant(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)
	assignee := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	outsider := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithAssignee(assignee.ID))

	read := func(viewer uuid.UUID) (string, bool) {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, viewer)
		testutil.SetPathParam(req, "id", contact.ID.String())
		require.NoError(t, app.GetContact(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
		var resp struct {
			Data struct {
				AccessMode string `json:"access_mode"`
				CanReply   bool   `json:"can_reply"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		return resp.Data.AccessMode, resp.Data.CanReply
	}

	mode, canReply := read(outsider.ID)
	assert.Equal(t, handlers.AccessModeRestricted, mode)
	assert.False(t, canReply)

	mode, canReply = read(assignee.ID)
	assert.Equal(t, handlers.AccessModeCurrentAssignee, mode)
	assert.True(t, canReply)
}

// Close-rating replies are the customer's own words: only administrators and the
// conversation's participants see the raw reply text.
func TestCloseRatingResponses_RawReplyLimitedToParticipants(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	readerRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "rating-reader",
		[]string{"accounts:read", "contacts:read"})
	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	reader := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&readerRole.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	now := time.Now()
	rating := 5
	require.NoError(t, app.DB.Create(&models.ChatClosureRating{
		OrganizationID:  org.ID,
		ContactID:       contact.ID,
		WhatsAppAccount: account.Name,
		Status:          "rated",
		Rating:          &rating,
		RawReply:        "great service",
		RatedAt:         &now,
		ExpiresAt:       now.Add(time.Hour),
	}).Error)

	list := func(viewer uuid.UUID) string {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, viewer)
		testutil.SetPathParam(req, "id", account.ID.String())
		require.NoError(t, app.ListCloseRatingResponses(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
		return string(testutil.GetResponseBody(req))
	}

	assert.Contains(t, list(admin.ID), "great service")
	assert.NotContains(t, list(reader.ID), "great service")
}

// Scheduled message text is conversation content: the org-wide list shows it only
// for conversations the viewer takes part in (administrators see everything).
func TestListScheduledMessages_TextLimitedToParticipants(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)
	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	assignee := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	outsider := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))

	mine := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithAssignee(assignee.ID))
	theirs := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithAssignee(admin.ID))
	schedule := func(contactID uuid.UUID, text string) {
		require.NoError(t, app.DB.Create(&models.ScheduledMessage{
			OrganizationID:  org.ID,
			WhatsAppAccount: account.Name,
			ContactID:       contactID,
			MessageType:     models.MessageTypeText,
			Content:         text,
			ScheduledAt:     time.Now().Add(time.Hour),
			CreatedBy:       admin.ID,
		}).Error)
	}
	schedule(mine.ID, "reminder for my customer")
	schedule(theirs.ID, "reminder for someone else")

	listed := func(viewer uuid.UUID) string {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, viewer)
		require.NoError(t, app.ListScheduledMessages(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
		return string(testutil.GetResponseBody(req))
	}

	assert.Contains(t, listed(admin.ID), "reminder for someone else")
	assert.Contains(t, listed(assignee.ID), "reminder for my customer")
	assert.NotContains(t, listed(assignee.ID), "reminder for someone else")
	assert.NotContains(t, listed(outsider.ID), "reminder for my customer")
	assert.NotContains(t, listed(outsider.ID), "reminder for someone else")
}

// The messages table widget shows a snippet of each message. Administrators see it;
// other viewers of a shared widget see the row without the text.
func TestWidgetMessagesTable_SnippetLimitedToAdministrators(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	analystRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "analyst", []string{"analytics:read"})
	admin := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))
	analyst := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&analystRole.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	require.NoError(t, app.DB.Create(&models.Message{
		OrganizationID: org.ID,
		ContactID:      contact.ID,
		Direction:      models.DirectionIncoming,
		MessageType:    models.MessageTypeText,
		Content:        "widget secret text",
		Status:         models.MessageStatusDelivered,
	}).Error)

	creator := admin.ID
	widget := &models.Widget{
		OrganizationID: org.ID,
		UserID:         &creator,
		Name:           "Recent messages",
		DataSource:     "messages",
		Metric:         "count",
		DisplayType:    "table",
		IsShared:       true,
	}
	require.NoError(t, app.DB.Create(widget).Error)

	snippets := func(viewer uuid.UUID) []string {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, viewer)
		testutil.SetPathParam(req, "id", widget.ID.String())
		require.NoError(t, app.GetWidgetData(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
		var resp struct {
			Data struct {
				TableRows []struct {
					SubLabel string `json:"sub_label"`
				} `json:"table_rows"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		out := make([]string, 0, len(resp.Data.TableRows))
		for _, r := range resp.Data.TableRows {
			out = append(out, r.SubLabel)
		}
		return out
	}

	assert.Contains(t, snippets(admin.ID), "widget secret text")
	assert.NotContains(t, snippets(analyst.ID), "widget secret text")
}

// Claiming is how a non-participant becomes a participant, so it must still work
// for an agent who is not yet part of a pending chat; afterwards the claimant reads it.
func TestClaimChat_NonParticipantAgentClaimsThenReads(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)
	agent := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	contact.SetStatus(models.ChatStatusPending)
	require.NoError(t, app.DB.Model(contact).Update("metadata", contact.Metadata).Error)

	claim := testutil.NewRequest(t)
	testutil.SetAuthContext(claim, org.ID, agent.ID)
	testutil.SetPathParam(claim, "id", contact.ID.String())
	require.NoError(t, app.ClaimChat(claim))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(claim), string(testutil.GetResponseBody(claim)))

	read := testutil.NewGETRequest(t)
	testutil.SetAuthContext(read, org.ID, agent.ID)
	testutil.SetPathParam(read, "id", contact.ID.String())
	testutil.SetQueryParam(read, "limit", 50)
	require.NoError(t, app.GetMessages(read))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(read))
}

// After an outsider claims a pending chat they become its participant, so the
// contact read used to refresh the conversation reports the assignee mode with
// reply enabled, not the restricted viewer state from before the claim.
func TestClaimChat_RefreshedContactReportsAssigneeAccess(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)
	agent := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	contact.SetStatus(models.ChatStatusPending)
	require.NoError(t, app.DB.Model(contact).Update("metadata", contact.Metadata).Error)

	read := func() (string, bool) {
		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, agent.ID)
		testutil.SetPathParam(req, "id", contact.ID.String())
		require.NoError(t, app.GetContact(req))
		require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))
		var resp struct {
			Data struct {
				AccessMode string `json:"access_mode"`
				CanReply   bool   `json:"can_reply"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
		return resp.Data.AccessMode, resp.Data.CanReply
	}

	mode, canReply := read()
	assert.NotEqual(t, handlers.AccessModeCurrentAssignee, mode)
	assert.False(t, canReply)

	claim := testutil.NewRequest(t)
	testutil.SetAuthContext(claim, org.ID, agent.ID)
	testutil.SetPathParam(claim, "id", contact.ID.String())
	require.NoError(t, app.ClaimChat(claim))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(claim))

	mode, canReply = read()
	assert.Equal(t, handlers.AccessModeCurrentAssignee, mode)
	assert.True(t, canReply)
}
