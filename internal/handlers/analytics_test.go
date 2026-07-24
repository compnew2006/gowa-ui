package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/handlers"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// --- Helper functions ---

// createTestMessage creates a message in the database for analytics testing.
func createTestMessage(t *testing.T, app *handlers.App, orgID, contactID uuid.UUID, direction models.Direction, createdAt time.Time) *models.Message {
	t.Helper()

	msg := &models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: createdAt},
		OrganizationID:  orgID,
		ContactID:       contactID,
		WhatsAppAccount: "test-account",
		Direction:       direction,
		MessageType:     models.MessageTypeText,
		Content:         "Test message",
		Status:          models.MessageStatusSent,
	}
	require.NoError(t, app.DB.Create(msg).Error)
	return msg
}

// createTestChatbotSession creates a chatbot session in the database.
func createTestChatbotSession(t *testing.T, app *handlers.App, orgID, contactID uuid.UUID, createdAt time.Time) *models.ChatbotSession {
	t.Helper()

	session := &models.ChatbotSession{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: createdAt},
		OrganizationID:  orgID,
		ContactID:       contactID,
		WhatsAppAccount: "test-account",
		PhoneNumber:     "+1234567890",
		Status:          "active",
		StartedAt:       createdAt,
		LastActivityAt:  createdAt,
	}
	require.NoError(t, app.DB.Create(session).Error)
	return session
}

// createAnalyticsTestCampaign creates a bulk message campaign with a specific creation time.
func createAnalyticsTestCampaign(t *testing.T, app *handlers.App, orgID, createdBy uuid.UUID, status string, createdAt time.Time) *models.BulkMessageCampaign {
	t.Helper()

	templateID := uuid.New()
	// Create a minimal template for the foreign key
	tmpl := &models.Template{
		BaseModel:       models.BaseModel{ID: templateID},
		OrganizationID:  orgID,
		WhatsAppAccount: "test-account",
		Name:            "campaign-template-" + uuid.New().String()[:8],
		MetaTemplateID:  "meta-" + uuid.New().String()[:8],
		Category:        "MARKETING",
		Language:        "en",
		Status:          string(models.TemplateStatusApproved),
		BodyContent:     "Hello",
	}
	require.NoError(t, app.DB.Create(tmpl).Error)

	campaign := &models.BulkMessageCampaign{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: createdAt},
		OrganizationID:  orgID,
		WhatsAppAccount: "test-account",
		Name:            "Test Campaign " + uuid.New().String()[:8],
		TemplateID:      templateID,
		Status:          models.CampaignStatus(status),
		CreatedBy:       createdBy,
	}
	require.NoError(t, app.DB.Create(campaign).Error)
	return campaign
}

// createTestAgentTransfer creates an agent transfer in the database.
func createTestAgentTransfer(t *testing.T, app *handlers.App, orgID, contactID uuid.UUID, agentID *uuid.UUID, status models.TransferStatus, source models.TransferSource, transferredAt time.Time, resumedAt *time.Time) *models.AgentTransfer {
	t.Helper()

	transfer := &models.AgentTransfer{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		ContactID:       contactID,
		WhatsAppAccount: "test-account",
		PhoneNumber:     "+1234567890",
		Status:          status,
		Source:          source,
		AgentID:         agentID,
		TransferredAt:   transferredAt,
		ResumedAt:       resumedAt,
	}
	require.NoError(t, app.DB.Create(transfer).Error)
	return transfer
}

// createTestTeamWithAgent creates a team and adds the user as an agent member.
func createTestTeamWithAgent(t *testing.T, app *handlers.App, orgID, userID uuid.UUID) (*models.Team, *models.TeamMember) {
	t.Helper()

	team := &models.Team{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           "Test Team " + uuid.New().String()[:8],
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(team).Error)

	member := &models.TeamMember{
		BaseModel: models.BaseModel{ID: uuid.New()},
		TeamID:    team.ID,
		UserID:    userID,
		Role:      models.TeamRoleAgent,
	}
	require.NoError(t, app.DB.Create(member).Error)

	return team, member
}

// --- GetDashboardStats Tests ---

// --- GetAgentAnalytics Tests ---

func TestApp_GetAgentAnalytics_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	perms := getAnalyticsPermissions(t, app)
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Analytics Agent", false, false, perms)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("agent-analytics")),
		testutil.WithPassword("password"),
		testutil.WithRoleID(&role.ID),
	)

	// Create a team and add user as agent so calculateAllAgentStats finds them
	createTestTeamWithAgent(t, app, org.ID, user.ID)

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	now := time.Now().UTC()
	resumedAt := now.Add(-30 * time.Minute)

	createTestAgentTransfer(t, app, org.ID, contact.ID, &user.ID,
		models.TransferStatusResumed, models.TransferSourceManual,
		now.Add(-2*time.Hour), &resumedAt)
	createTestAgentTransfer(t, app, org.ID, contact.ID, &user.ID,
		models.TransferStatusActive, models.TransferSourceFlow,
		now.Add(-1*time.Hour), nil)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AgentAnalyticsResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	// User has analytics permission so sees summary + all agent stats + my_stats
	assert.NotNil(t, resp.Data.MyStats)
	assert.NotNil(t, resp.Data.AgentStats)
}

func TestApp_GetAgentAnalytics_EmptyData(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	perms := getAnalyticsPermissions(t, app)
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Analytics Empty", false, false, perms)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("agent-empty")),
		testutil.WithPassword("password"),
		testutil.WithRoleID(&role.ID),
	)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AgentAnalyticsResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	assert.Equal(t, int64(0), resp.Data.Summary.TotalTransfersHandled)
	assert.Equal(t, int64(0), resp.Data.Summary.ActiveTransfers)
	assert.NotNil(t, resp.Data.TrendData)
	assert.Empty(t, resp.Data.TrendData)
}

func TestApp_GetAgentAnalytics_Unauthorized(t *testing.T) {
	app := newTestApp(t)

	req := testutil.NewGETRequest(t)
	// No auth context

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_GetAgentAnalytics_AgentSeesOwnStats(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create a user without analytics permission (agent-level user)
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("agent-own")),
		testutil.WithPassword("password"),
	)

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	now := time.Now().UTC()
	resumedAt := now.Add(-10 * time.Minute)

	createTestAgentTransfer(t, app, org.ID, contact.ID, &user.ID,
		models.TransferStatusResumed, models.TransferSourceManual,
		now.Add(-1*time.Hour), &resumedAt)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AgentAnalyticsResponse `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)

	// User without analytics permission sees only their own stats
	assert.NotNil(t, resp.Data.MyStats)
	assert.Nil(t, resp.Data.AgentStats)
	assert.Equal(t, user.ID.String(), resp.Data.MyStats.AgentID)
}

// --- GetAgentDetails Tests ---

// --- GetAgentComparison Tests ---
