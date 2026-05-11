package handlers_test

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// listContactsResponse is the JSON shape returned by ListContacts.
type listContactsResponse struct {
	Contacts []contactResponseItem `json:"contacts"`
	Total    int64                 `json:"total"`
}

type contactResponseItem struct {
	ID              string  `json:"id"`
	InstanceID      *string `json:"instance_id"`
	WhatsAppAccount string  `json:"whatsapp_account"`
	AssignedUserID  *string `json:"assigned_user_id"`
	IsPublic        bool    `json:"is_public"`
	IsCollaborator  bool    `json:"is_collaborator"`
	Status          string  `json:"status"`
}

func setupAuthRequest(t *testing.T, orgID, userID uuid.UUID) *fastglue.Request {
	t.Helper()
	ctx := &fasthttp.RequestCtx{}
	ctx.SetUserValue("organization_id", orgID.String())
	ctx.SetUserValue("user_id", userID.String())
	return &fastglue.Request{RequestCtx: ctx}
}

// TestListContacts_AgentInstanceRedaction verifies that agent-role users have instance
// details (instance_id, whatsapp_account) redacted on chats assigned to other agents
// that are visible via the public chat filter, while keeping instance details visible
// on their own assignments, pending chats, and for admin users.
func TestListContacts_AgentInstanceRedaction(t *testing.T) {
	app := newTestApp(t)

	org := testutil.CreateTestOrganization(t, app.DB)

	// Create admin user (non-agent, should see everything).
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	adminUser := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&adminRole.ID))

	// Create two agent users.
	agentRole := testutil.CreateAgentRole(t, app.DB, org.ID)
	agentA := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	agentB := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))

	// Create a WhatsApp instance so contacts can reference a valid instance_id.
	instanceID := uuid.New()
	instanceName := "test-instance-leak"
	instance := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: instanceID},
		OrganizationID: org.ID,
		Name:           instanceName,
		Status:         models.InstanceStatusConnected,
	}
	require.NoError(t, app.DB.Create(instance).Error)

	// Contact 1: Assigned to Agent A (should NOT be redacted for Agent A).
	contactA := createContactWithInstance(t, app.DB, org, &instanceID, &agentA.ID, instanceName, false)

	// Contact 2: Assigned to Agent B, NOT public (Agent A cannot see it at all).
	contactBPrivate := createContactWithInstance(t, app.DB, org, &instanceID, &agentB.ID, instanceName, false)

	// Contact 3: Assigned to Agent B, IS public (Agent A CAN see it, but instance should be REDACTED).
	contactBPublic := createContactWithInstance(t, app.DB, org, &instanceID, &agentB.ID, instanceName, true)

	// Contact 4: Pending, unassigned (Agent A CAN see it, instance should NOT be redacted).
	contactPending := createContactWithInstance(t, app.DB, org, &instanceID, nil, instanceName, false)
	require.NoError(t, app.DB.Model(contactPending).Update("status", models.ChatStatusPending).Error)

	t.Run("agent_a_sees_own_instance_details", func(t *testing.T) {
		req := setupAuthRequest(t, org.ID, agentA.ID)
		require.NoError(t, app.ListContacts(req))

		var resp listContactsResponse
		testutil.ParseEnvelopeResponse(t, req, &resp)

		found := false
		for _, c := range resp.Contacts {
			if c.ID == contactA.ID.String() {
				found = true
				assert.NotNil(t, c.InstanceID, "agent should see instance_id on own assigned chat")
				assert.Equal(t, instanceName, c.WhatsAppAccount, "agent should see whatsapp_account on own assigned chat")
			}
		}
		assert.True(t, found, "agent A's own assigned contact should appear in list")
	})

	t.Run("agent_a_redacted_on_other_agent_public_chat", func(t *testing.T) {
		req := setupAuthRequest(t, org.ID, agentA.ID)
		require.NoError(t, app.ListContacts(req))

		var resp listContactsResponse
		testutil.ParseEnvelopeResponse(t, req, &resp)

		for _, c := range resp.Contacts {
			if c.ID == contactBPublic.ID.String() {
				assert.Nil(t, c.InstanceID, "instance_id should be redacted for other agent's public chat")
				assert.Empty(t, c.WhatsAppAccount, "whatsapp_account should be redacted for other agent's public chat")
			}
		}
	})

	t.Run("agent_a_sees_pending_instance_details", func(t *testing.T) {
		req := setupAuthRequest(t, org.ID, agentA.ID)
		require.NoError(t, app.ListContacts(req))

		var resp listContactsResponse
		testutil.ParseEnvelopeResponse(t, req, &resp)

		for _, c := range resp.Contacts {
			if c.ID == contactPending.ID.String() {
				assert.NotNil(t, c.InstanceID, "agent should see instance_id on pending/unassigned chat")
				assert.Equal(t, instanceName, c.WhatsAppAccount, "agent should see whatsapp_account on pending chat")
			}
		}
	})

	t.Run("agent_a_cannot_see_other_agent_private_chat", func(t *testing.T) {
		req := setupAuthRequest(t, org.ID, agentA.ID)
		require.NoError(t, app.ListContacts(req))

		var resp listContactsResponse
		testutil.ParseEnvelopeResponse(t, req, &resp)

		for _, c := range resp.Contacts {
			assert.NotEqual(t, contactBPrivate.ID.String(), c.ID, "agent should not see other agent's private chats")
		}
	})

	t.Run("admin_sees_all_instance_details", func(t *testing.T) {
		req := setupAuthRequest(t, org.ID, adminUser.ID)
		require.NoError(t, app.ListContacts(req))

		var resp listContactsResponse
		testutil.ParseEnvelopeResponse(t, req, &resp)

		// Admin is NOT an agent, so agentScope=false and no redaction occurs.
		// Admin should see all contacts with full instance details.
		foundIDs := map[string]bool{}
		for _, c := range resp.Contacts {
			foundIDs[c.ID] = true
			// Every contact the admin sees should have instance details.
			assert.NotNil(t, c.InstanceID, "admin should see instance_id on all contacts, got nil for %s", c.ID)
			assert.Equal(t, instanceName, c.WhatsAppAccount, "admin should see whatsapp_account on all contacts")
		}
		assert.True(t, foundIDs[contactA.ID.String()], "admin should see agent A's chat")
		assert.True(t, foundIDs[contactBPrivate.ID.String()], "admin should see agent B's private chat")
		assert.True(t, foundIDs[contactBPublic.ID.String()], "admin should see agent B's public chat")
		assert.True(t, foundIDs[contactPending.ID.String()], "admin should see pending chat")
	})
}

func createContactWithInstance(t *testing.T, db *gorm.DB, org *models.Organization, instanceID *uuid.UUID, assignedUserID *uuid.UUID, waAccount string, isPublic bool) *models.Contact {
	t.Helper()
	uniqueID := uuid.New().String()[:8]
	contact := &models.Contact{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		InstanceID:      instanceID,
		PhoneNumber:     "+1234567890" + uniqueID[:4],
		ProfileName:     "Test Contact " + uniqueID,
		WhatsAppAccount: waAccount,
		AssignedUserID:  assignedUserID,
		IsPublic:        isPublic,
		Status:          models.ChatStatusOpen,
	}
	if assignedUserID == nil {
		contact.Status = models.ChatStatusPending
	}
	require.NoError(t, db.Create(contact).Error)
	return contact
}
