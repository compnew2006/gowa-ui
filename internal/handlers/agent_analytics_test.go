package handlers_test

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// --- Test Helpers ---

// createAgentAnalyticsTestApp creates a test app with the necessary setup for agent analytics tests.
func createAgentAnalyticsTestApp(t *testing.T) (*handlers.App, *models.Organization, *models.User, *models.User) {
	t.Helper()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Create a user with analytics permissions (admin)
	admin := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("admin")),
		testutil.WithPassword("password"),
	)

	// Grant analytics permission
	perms := testutil.GetOrCreateTestPermissions(t, app.DB)
	var analyticsPerm *models.Permission
	for _, p := range perms {
		if p.Resource == models.ResourceAnalytics && p.Action == models.ActionRead {
			analyticsPerm = &p
			break
		}
	}

	if analyticsPerm != nil {
		// Create admin role with analytics permission
		role := testutil.CreateTestRole(t, app.DB, org.ID, "Admin Role", []models.Permission{*analyticsPerm})

		// Update admin user with role
		admin.RoleID = &role.ID
		require.NoError(t, app.DB.Save(admin).Error)
	}

	// Create an agent user
	agent := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("agent")),
		testutil.WithPassword("password"),
	)

	// Create a team and add agent as member
	team := &models.Team{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Test Team",
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(team).Error)

	member := &models.TeamMember{
		BaseModel: models.BaseModel{ID: uuid.New()},
		TeamID:    team.ID,
		UserID:    agent.ID,
		Role:      models.TeamRoleAgent,
	}
	require.NoError(t, app.DB.Create(member).Error)

	return app, org, admin, agent
}

// createTestUserAvailabilityLog creates a user availability log for testing break time calculations.
func createTestUserAvailabilityLog(t *testing.T, app *handlers.App, userID, orgID uuid.UUID, isAvailable bool, startedAt time.Time, endedAt *time.Time) *models.UserAvailabilityLog {
	t.Helper()

	log := &models.UserAvailabilityLog{
		ID:             uuid.New(),
		UserID:         userID,
		OrganizationID: orgID,
		IsAvailable:    isAvailable,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
	}
	require.NoError(t, app.DB.Create(log).Error)
	return log
}

// createTestChatClosureRating creates a chat closure rating record for testing.
func createTestChatClosureRating(t *testing.T, app *handlers.App, orgID, contactID, closingAgentID uuid.UUID, agentID *uuid.UUID, rating int, ratedAt time.Time) *models.ChatClosureRating {
	t.Helper()

	cycle := &models.ChatClosureRating{
		BaseModel:        models.BaseModel{ID: uuid.New()},
		OrganizationID:   orgID,
		ContactID:        contactID,
		ChatID:           contactID,
		AgentUserID:      agentID,
		ClosingAgentID:   closingAgentID,
		ClosedAt:         ratedAt,
		State:            models.ChatClosureRatingStateRated,
		Rating:           &rating,
		RatedAt:          &ratedAt,
		RatingMessage:    "Great service",
		CloseMessage:     "Please rate 1-10",
		CloseMessageLanguage: "en",
	}
	require.NoError(t, app.DB.Create(cycle).Error)
	return cycle
}

// createTestWhatsAppInstance creates a WhatsApp instance for testing.
func createTestWhatsAppInstance(t *testing.T, app *handlers.App, orgID uuid.UUID) *models.WhatsAppInstance {
	t.Helper()

	instance := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           "test-instance-" + uuid.New().String()[:8],
		PhoneNumber:    "+1234567890",
		Status:         models.InstanceStatusConnected,
	}
	require.NoError(t, app.DB.Create(instance).Error)
	return instance
}

// withContactInstanceID sets the instance ID on the contact.
func withContactInstanceID(instanceID uuid.UUID) func(*models.Contact) {
	return func(c *models.Contact) {
		c.InstanceID = &instanceID
	}
}

// --- GetAgentAnalytics Handler Tests ---

func TestGetAgentAnalytics_Unauthorized_Returns401(t *testing.T) {
	app := newTestApp(t)
	req := testutil.NewGETRequest(t)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestGetAgentAnalytics_AgentWithoutPermission_SeesOnlyOwnStats(t *testing.T) {
	app, org, _, agent := createAgentAnalyticsTestApp(t)

	// Create transfers handled by this agent
	now := time.Now().UTC()
	contact1 := testutil.CreateTestContact(t, app.DB, org.ID)
	contact2 := testutil.CreateTestContact(t, app.DB, org.ID)

	createTestAgentTransfer(t, app, org.ID, contact1.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-2*time.Hour), &[]time.Time{now.Add(-1*time.Hour)}[0])
	createTestAgentTransfer(t, app, org.ID, contact2.ID, &agent.ID, models.TransferStatusActive, models.TransferSourceFlow, now.Add(-30*time.Minute), nil)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req, &response)

	require.NotNil(t, response.MyStats)
	assert.Equal(t, agent.ID.String(), response.MyStats.AgentID)
	assert.Equal(t, int64(1), response.MyStats.TransfersHandled)
	assert.Equal(t, int64(1), response.MyStats.ActiveTransfers)
	assert.Nil(t, response.AgentStats)
}

func TestGetAgentAnalytics_AdminWithFullPermission_SeesAllAgents(t *testing.T) {
	app, org, admin, agent := createAgentAnalyticsTestApp(t)

	// Create another agent
	agent2 := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("agent2")),
		testutil.WithPassword("password"),
	)
	team := &models.Team{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Test Team 2",
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(team).Error)
	member := &models.TeamMember{
		BaseModel: models.BaseModel{ID: uuid.New()},
		TeamID:    team.ID,
		UserID:    agent2.ID,
		Role:      models.TeamRoleAgent,
	}
	require.NoError(t, app.DB.Create(member).Error)

	// Create transfers for both agents
	now := time.Now().UTC()
	contact1 := testutil.CreateTestContact(t, app.DB, org.ID)
	contact2 := testutil.CreateTestContact(t, app.DB, org.ID)

	createTestAgentTransfer(t, app, org.ID, contact1.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-2*time.Hour), &[]time.Time{now.Add(-1*time.Hour)}[0])
	createTestAgentTransfer(t, app, org.ID, contact2.ID, &agent2.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-3*time.Hour), &[]time.Time{now.Add(-2*time.Hour)}[0])

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req, &response)

	require.NotNil(t, response.AgentStats)
	assert.GreaterOrEqual(t, len(response.AgentStats), 2)
	require.NotNil(t, response.MyStats)
}

func TestGetAgentAnalytics_WithDateRange_FiltersCorrectly(t *testing.T) {
	app, org, admin, agent := createAgentAnalyticsTestApp(t)

	now := time.Now().UTC()
	startDate := now.Add(-7 * 24 * time.Hour).Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	// Create a transfer within the range
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	createTestAgentTransfer(t, app, org.ID, contact.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-24*time.Hour), &[]time.Time{now.Add(-23*time.Hour)}[0])

	// Create a transfer outside the range
	contact2 := testutil.CreateTestContact(t, app.DB, org.ID)
	createTestAgentTransfer(t, app, org.ID, contact2.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-10*24*time.Hour), &[]time.Time{now.Add(-9*24*time.Hour)}[0])

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("from", startDate)
	req.RequestCtx.QueryArgs().Add("to", endDate)
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req, &response)

	// Should only count the transfer within the date range
	assert.Equal(t, int64(1), response.Summary.TotalTransfersHandled)
}

func TestGetAgentAnalytics_WithInvalidDateRange_Returns400(t *testing.T) {
	app, org, admin, _ := createAgentAnalyticsTestApp(t)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("from", "invalid-date")
	req.RequestCtx.QueryArgs().Add("to", "2024-01-01")
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetAgentAnalytics_WithInstanceFilter_FiltersByInstance(t *testing.T) {
	app, org, admin, agent := createAgentAnalyticsTestApp(t)

	// Create two instances
	instance1 := createTestWhatsAppInstance(t, app, org.ID)
	instance2 := createTestWhatsAppInstance(t, app, org.ID)

	now := time.Now().UTC()

	// Create contacts for different instances
	contact1 := testutil.CreateTestContactWith(t, app.DB, org.ID, withContactInstanceID(instance1.ID))
	contact2 := testutil.CreateTestContactWith(t, app.DB, org.ID, withContactInstanceID(instance2.ID))

	createTestAgentTransfer(t, app, org.ID, contact1.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-2*time.Hour), &[]time.Time{now.Add(-1*time.Hour)}[0])
	createTestAgentTransfer(t, app, org.ID, contact2.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-3*time.Hour), &[]time.Time{now.Add(-2*time.Hour)}[0])

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("instance_id", instance1.ID.String())
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req, &response)

	// Should only see stats for instance1
	assert.Equal(t, int64(1), response.Summary.TotalTransfersHandled)
}

func TestGetAgentAnalytics_WithInvalidInstanceID_Returns400(t *testing.T) {
	app, org, admin, _ := createAgentAnalyticsTestApp(t)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("instance_id", "invalid-uuid")
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetAgentAnalytics_WithInstanceIDFromDifferentOrg_Returns400(t *testing.T) {
	app, org, admin, _ := createAgentAnalyticsTestApp(t)

	// Create instance in a different organization
	org2 := testutil.CreateTestOrganization(t, app.DB)
	instance := createTestWhatsAppInstance(t, app, org2.ID)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("instance_id", instance.ID.String())
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetAgentAnalytics_WithAgentFilter_AsAdmin_ShowsSpecificAgent(t *testing.T) {
	app, org, admin, agent := createAgentAnalyticsTestApp(t)

	now := time.Now().UTC()
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	createTestAgentTransfer(t, app, org.ID, contact.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-2*time.Hour), &[]time.Time{now.Add(-1*time.Hour)}[0])

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("agent_id", agent.ID.String())
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req, &response)

	require.NotNil(t, response.MyStats)
	assert.Equal(t, agent.ID.String(), response.MyStats.AgentID)
	assert.Nil(t, response.AgentStats)
}

func TestGetAgentAnalytics_WithInvalidAgentID_Returns400(t *testing.T) {
	app, org, admin, _ := createAgentAnalyticsTestApp(t)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("agent_id", "invalid-uuid")
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetAgentAnalytics_WithAgentIDFromDifferentOrg_Returns404(t *testing.T) {
	app, org, admin, _ := createAgentAnalyticsTestApp(t)

	org2 := testutil.CreateTestOrganization(t, app.DB)
	agent2 := testutil.CreateTestUser(t, app.DB, org2.ID)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("agent_id", agent2.ID.String())
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestGetAgentAnalytics_WithRatingFilter_FiltersRatings(t *testing.T) {
	app, org, admin, agent := createAgentAnalyticsTestApp(t)

	now := time.Now().UTC()
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Create ratings with different scores
	createTestChatClosureRating(t, app, org.ID, contact.ID, agent.ID, &agent.ID, 8, now.Add(-2*time.Hour))
	createTestChatClosureRating(t, app, org.ID, contact.ID, agent.ID, &agent.ID, 5, now.Add(-3*time.Hour))
	createTestChatClosureRating(t, app, org.ID, contact.ID, agent.ID, &agent.ID, 9, now.Add(-4*time.Hour))

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("min_rating", "8")
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req, &response)

	require.NotNil(t, response.RatingSummary)
	assert.Equal(t, int64(2), response.RatingSummary.TotalRatings)
	assert.GreaterOrEqual(t, response.RatingSummary.AverageRating, 8.0)
}

func TestGetAgentAnalytics_WithInvalidMinRating_Returns400(t *testing.T) {
	app, org, admin, _ := createAgentAnalyticsTestApp(t)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("min_rating", "invalid")
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetAgentAnalytics_WithMinRatingAbove10_Returns400(t *testing.T) {
	app, org, admin, _ := createAgentAnalyticsTestApp(t)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("min_rating", "11")
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetAgentAnalytics_WithMinRatingGreaterThanMax_Returns400(t *testing.T) {
	app, org, admin, _ := createAgentAnalyticsTestApp(t)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("min_rating", "8")
	req.RequestCtx.QueryArgs().Add("max_rating", "5")
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetAgentAnalytics_WithGroupByWeek_AggregatesByWeek(t *testing.T) {
	app, org, admin, agent := createAgentAnalyticsTestApp(t)

	now := time.Now().UTC()
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Create transfers across multiple days
	for i := 0; i < 7; i++ {
		createTestAgentTransfer(t, app, org.ID, contact.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-time.Duration(24*i)*time.Hour), &[]time.Time{now.Add(-time.Duration(24*i)*time.Hour + time.Hour)}[0])
	}

	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("group_by", "week")
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req, &response)

	assert.NotEmpty(t, response.TrendData)
}

func TestGetAgentAnalytics_DefaultsToCurrentMonth(t *testing.T) {
	app, org, admin, _ := createAgentAnalyticsTestApp(t)

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	createTestAgentTransfer(t, app, org.ID, contact.ID, nil, models.TransferStatusResumed, models.TransferSourceManual, monthStart.Add(24*time.Hour), &[]time.Time{monthStart.Add(25*time.Hour)}[0])

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req, &response)

	assert.GreaterOrEqual(t, response.Summary.TotalTransfersHandled, int64(1))
}

// --- calculateBreakTime Tests ---

func TestCalculateBreakTime_CompletedBreak_CalculatesCorrectly(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)

	now := time.Now().UTC()
	breakStart := now.Add(-2 * time.Hour)
	breakEnd := now.Add(-1 * time.Hour)

	createTestUserAvailabilityLog(t, app, agent.ID, org.ID, false, breakStart, &breakEnd)

	totalMins, count := app.CalculateBreakTime(org.ID, agent.ID, now.Add(-3*time.Hour), now)
	assert.Equal(t, float64(60), totalMins)
	assert.Equal(t, int64(1), count)
}

func TestCalculateBreakTime_ActiveBreak_CalculatesUpToNow(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)

	now := time.Now().UTC()
	breakStart := now.Add(-30 * time.Minute)

	createTestUserAvailabilityLog(t, app, agent.ID, org.ID, false, breakStart, nil)

	totalMins, count := app.CalculateBreakTime(org.ID, agent.ID, now.Add(-1*time.Hour), now)

	assert.GreaterOrEqual(t, totalMins, float64(29))
	assert.LessOrEqual(t, totalMins, float64(31))
	assert.Equal(t, int64(1), count)
}

func TestCalculateBreakTime_MultipleBreaks_SumsCorrectly(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)

	now := time.Now().UTC()
	break1Start := now.Add(-3 * time.Hour)
	break1End := now.Add(-2*time.Hour).Add(30*time.Minute)
	break2Start := now.Add(-2 * time.Hour)
	break2End := now.Add(-1*time.Hour).Add(30*time.Minute)

	createTestUserAvailabilityLog(t, app, agent.ID, org.ID, false, break1Start, &break1End)
	createTestUserAvailabilityLog(t, app, agent.ID, org.ID, false, break2Start, &break2End)

	totalMins, count := app.CalculateBreakTime(org.ID, agent.ID, now.Add(-4*time.Hour), now)

	assert.GreaterOrEqual(t, totalMins, float64(119))
	assert.LessOrEqual(t, totalMins, float64(121))
	assert.Equal(t, int64(2), count)
}

func TestCalculateBreakTime_BreakOverlappingRange_ClipsCorrectly(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)

	now := time.Now().UTC()
	rangeStart := now.Add(-1 * time.Hour)
	rangeEnd := now
	breakStart := now.Add(-2 * time.Hour)
	breakEnd := now.Add(-30 * time.Minute)

	createTestUserAvailabilityLog(t, app, agent.ID, org.ID, false, breakStart, &breakEnd)

	totalMins, count := app.CalculateBreakTime(org.ID, agent.ID, rangeStart, rangeEnd)

	assert.GreaterOrEqual(t, totalMins, float64(29))
	assert.LessOrEqual(t, totalMins, float64(31))
	assert.Equal(t, int64(1), count)
}

func TestCalculateBreakTime_NoBreaks_ReturnsZero(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)

	now := time.Now().UTC()

	totalMins, count := app.CalculateBreakTime(org.ID, agent.ID, now.Add(-1*time.Hour), now)

	assert.Equal(t, float64(0), totalMins)
	assert.Equal(t, int64(0), count)
}

// --- analyticsAgentBelongsToOrg Tests ---

func TestAnalyticsAgentBelongsToOrg_AgentInOrg_ReturnsTrue(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)

	belongs, err := app.AnalyticsAgentBelongsToOrg(org.ID, agent.ID)

	require.NoError(t, err)
	assert.True(t, belongs)
}

func TestAnalyticsAgentBelongsToOrg_AgentNotInOrg_ReturnsFalse(t *testing.T) {
	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	agent := testutil.CreateTestUser(t, app.DB, org1.ID)

	belongs, err := app.AnalyticsAgentBelongsToOrg(org2.ID, agent.ID)

	require.NoError(t, err)
	assert.False(t, belongs)
}

func TestAnalyticsAgentBelongsToOrg_NonExistentAgent_ReturnsFalse(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	nonExistentAgent := uuid.New()

	belongs, err := app.AnalyticsAgentBelongsToOrg(org.ID, nonExistentAgent)

	require.NoError(t, err)
	assert.False(t, belongs)
}

// --- calculateTrendData Tests ---

func TestCalculateTrendData_AggregatesByDay(t *testing.T) {
	app, org, _, agent := createAgentAnalyticsTestApp(t)

	now := time.Now().UTC()
	startDate := now.Add(-7 * 24 * time.Hour)

	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Create transfers across 3 days
	createTestAgentTransfer(t, app, org.ID, contact.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, startDate.Add(24*time.Hour), &[]time.Time{startDate.Add(25*time.Hour)}[0])
	createTestAgentTransfer(t, app, org.ID, contact.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, startDate.Add(48*time.Hour), &[]time.Time{startDate.Add(49*time.Hour)}[0])
	createTestAgentTransfer(t, app, org.ID, contact.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, startDate.Add(72*time.Hour), &[]time.Time{startDate.Add(73*time.Hour)}[0])

	trendData := app.CalculateTrendData(org.ID, startDate, now, "day", &agent.ID, nil)

	assert.GreaterOrEqual(t, len(trendData), 3)
	for _, point := range trendData {
		assert.NotEmpty(t, point.Date)
		assert.GreaterOrEqual(t, point.TransfersHandled, int64(0))
	}
}

func TestCalculateTrendData_WithInstanceFilter_FiltersCorrectly(t *testing.T) {
	app, org, _, agent := createAgentAnalyticsTestApp(t)

	instance1 := createTestWhatsAppInstance(t, app, org.ID)
	instance2 := createTestWhatsAppInstance(t, app, org.ID)

	now := time.Now().UTC()
	startDate := now.Add(-24 * time.Hour)

	contact1 := testutil.CreateTestContactWith(t, app.DB, org.ID, withContactInstanceID(instance1.ID))
	contact2 := testutil.CreateTestContactWith(t, app.DB, org.ID, withContactInstanceID(instance2.ID))

	createTestAgentTransfer(t, app, org.ID, contact1.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, startDate.Add(2*time.Hour), &[]time.Time{startDate.Add(3*time.Hour)}[0])
	createTestAgentTransfer(t, app, org.ID, contact2.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, startDate.Add(4*time.Hour), &[]time.Time{startDate.Add(5*time.Hour)}[0])

	trendData := app.CalculateTrendData(org.ID, startDate, now, "day", &agent.ID, &instance1.ID)

	// Should only count transfers from instance1
	totalTransfers := int64(0)
	for _, point := range trendData {
		totalTransfers += point.TransfersHandled
	}
	assert.Equal(t, int64(1), totalTransfers)
}

// --- GetAgentDetails Tests ---

func TestGetAgentDetails_Unauthorized_Returns401(t *testing.T) {
	app := newTestApp(t)
	req := testutil.NewGETRequest(t)
	req.RequestCtx.SetUserValue("id", uuid.New().String())

	err := app.GetAgentDetails(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestGetAgentDetails_WithPermission_ReturnsAgentStats(t *testing.T) {
	app, org, admin, agent := createAgentAnalyticsTestApp(t)

	now := time.Now().UTC()
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	createTestAgentTransfer(t, app, org.ID, contact.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-2*time.Hour), &[]time.Time{now.Add(-1*time.Hour)}[0])

	req := testutil.NewGETRequest(t)
	req.RequestCtx.SetUserValue("id", agent.ID.String())
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentDetails(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response map[string]any
	testutil.ParseJSONResponse(t, req, &response)

	agentData, ok := response["agent"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, agent.ID.String(), agentData["agent_id"])
	assert.NotNil(t, response["trend_data"])
}

func TestGetAgentDetails_InvalidAgentID_Returns400(t *testing.T) {
	app, org, admin, _ := createAgentAnalyticsTestApp(t)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.SetUserValue("id", "invalid-uuid")
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentDetails(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetAgentDetails_AgentNotInOrg_Returns404(t *testing.T) {
	app, org, admin, _ := createAgentAnalyticsTestApp(t)

	org2 := testutil.CreateTestOrganization(t, app.DB)
	agent2 := testutil.CreateTestUser(t, app.DB, org2.ID)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.SetUserValue("id", agent2.ID.String())
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentDetails(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

// --- GetAgentComparison Tests ---

func TestGetAgentComparison_Unauthorized_Returns401(t *testing.T) {
	app := newTestApp(t)
	req := testutil.NewGETRequest(t)

	err := app.GetAgentComparison(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestGetAgentComparison_WithPermission_ReturnsAllAgents(t *testing.T) {
	app, org, admin, _ := createAgentAnalyticsTestApp(t)

	agent2 := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("agent2")),
		testutil.WithPassword("password"),
	)
	team := &models.Team{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Test Team 2",
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(team).Error)
	member := &models.TeamMember{
		BaseModel: models.BaseModel{ID: uuid.New()},
		TeamID:    team.ID,
		UserID:    agent2.ID,
		Role:      models.TeamRoleAgent,
	}
	require.NoError(t, app.DB.Create(member).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentComparison(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response map[string]any
	testutil.ParseJSONResponse(t, req, &response)

	agents, ok := response["agents"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(agents), 2)
}

// --- calculateAgentStats Tests ---

func TestCalculateAgentStats_IncludesBreakTime(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)

	now := time.Now().UTC()
	breakStart := now.Add(-2 * time.Hour)
	breakEnd := now.Add(-1 * time.Hour)

	createTestUserAvailabilityLog(t, app, agent.ID, org.ID, false, breakStart, &breakEnd)

	stats := app.CalculateAgentStats(org.ID, agent.ID, now.Add(-3*time.Hour), now, nil)

	assert.GreaterOrEqual(t, stats.TotalBreakTimeMins, float64(59))
	assert.LessOrEqual(t, stats.TotalBreakTimeMins, float64(61))
	assert.Equal(t, int64(1), stats.BreakCount)
}

func TestCalculateAgentStats_AgentUnavailable_IncludesCurrentBreakStart(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := testutil.CreateTestUser(t, app.DB, org.ID)

	now := time.Now().UTC()
	breakStart := now.Add(-30 * time.Minute)

	// Set agent as unavailable
	require.NoError(t, app.DB.Model(&models.User{}).Where("id = ?", agent.ID).Update("is_available", false).Error)

	createTestUserAvailabilityLog(t, app, agent.ID, org.ID, false, breakStart, nil)

	stats := app.CalculateAgentStats(org.ID, agent.ID, now.Add(-1*time.Hour), now, nil)

	assert.False(t, stats.IsAvailable)
	require.NotNil(t, stats.CurrentBreakStart)
	assert.Contains(t, *stats.CurrentBreakStart, breakStart.Format("2006-01-02T15:04"))
}

// --- calculateSummaryStats Tests ---

func TestCalculateSummaryStats_CalculatesAllMetrics(t *testing.T) {
	app, org, _, agent := createAgentAnalyticsTestApp(t)

	now := time.Now().UTC()
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Create resumed transfers
	createTestAgentTransfer(t, app, org.ID, contact.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-2*time.Hour), &[]time.Time{now.Add(-1*time.Hour)}[0])
	createTestAgentTransfer(t, app, org.ID, contact.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceFlow, now.Add(-4*time.Hour), &[]time.Time{now.Add(-3*time.Hour)}[0])

	// Create active transfer
	contact2 := testutil.CreateTestContact(t, app.DB, org.ID)
	createTestAgentTransfer(t, app, org.ID, contact2.ID, &agent.ID, models.TransferStatusActive, models.TransferSourceManual, now.Add(-30*time.Minute), nil)

	var summary handlers.AgentAnalyticsSummary
	summary.TransfersBySource = make(map[string]int64)
	app.CalculateSummaryStats(org.ID, now.Add(-5*time.Hour), now, &summary, nil)

	assert.Equal(t, int64(2), summary.TotalTransfersHandled)
	assert.Equal(t, int64(1), summary.ActiveTransfers)
	assert.Greater(t, summary.AvgResolutionMins, float64(0))
	assert.NotEmpty(t, summary.TransfersBySource)
}

func TestCalculateSummaryStats_WithInstanceFilter_FiltersCorrectly(t *testing.T) {
	app, org, _, agent := createAgentAnalyticsTestApp(t)

	instance1 := createTestWhatsAppInstance(t, app, org.ID)
	instance2 := createTestWhatsAppInstance(t, app, org.ID)

	now := time.Now().UTC()
	contact1 := testutil.CreateTestContactWith(t, app.DB, org.ID, withContactInstanceID(instance1.ID))
	contact2 := testutil.CreateTestContactWith(t, app.DB, org.ID, withContactInstanceID(instance2.ID))

	createTestAgentTransfer(t, app, org.ID, contact1.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-2*time.Hour), &[]time.Time{now.Add(-1*time.Hour)}[0])
	createTestAgentTransfer(t, app, org.ID, contact2.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-3*time.Hour), &[]time.Time{now.Add(-2*time.Hour)}[0])

	var summary handlers.AgentAnalyticsSummary
	app.CalculateSummaryStats(org.ID, now.Add(-5*time.Hour), now, &summary, &instance1.ID)

	assert.Equal(t, int64(1), summary.TotalTransfersHandled)
}

// --- calculateAllAgentStats Tests ---

func TestCalculateAllAgentStats_ReturnsAllAgents(t *testing.T) {
	app, org, _, agent := createAgentAnalyticsTestApp(t)

	agent2 := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("agent2")),
		testutil.WithPassword("password"),
	)
	team := &models.Team{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Test Team 2",
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(team).Error)
	member := &models.TeamMember{
		BaseModel: models.BaseModel{ID: uuid.New()},
		TeamID:    team.ID,
		UserID:    agent2.ID,
		Role:      models.TeamRoleAgent,
	}
	require.NoError(t, app.DB.Create(member).Error)

	now := time.Now().UTC()
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	createTestAgentTransfer(t, app, org.ID, contact.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-2*time.Hour), &[]time.Time{now.Add(-1*time.Hour)}[0])

	stats := app.CalculateAllAgentStats(org.ID, now.Add(-5*time.Hour), now, nil)

	assert.GreaterOrEqual(t, len(stats), 2)
}

// --- Integration Tests ---

func TestAgentAnalytics_EndToEnd_FullWorkflow(t *testing.T) {
	app, org, admin, agent := createAgentAnalyticsTestApp(t)

	agent2 := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("agent2")),
		testutil.WithPassword("password"),
	)
	team := &models.Team{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "Test Team 2",
		IsActive:       true,
	}
	require.NoError(t, app.DB.Create(team).Error)
	member := &models.TeamMember{
		BaseModel: models.BaseModel{ID: uuid.New()},
		TeamID:    team.ID,
		UserID:    agent2.ID,
		Role:      models.TeamRoleAgent,
	}
	require.NoError(t, app.DB.Create(member).Error)

	instance1 := createTestWhatsAppInstance(t, app, org.ID)
	instance2 := createTestWhatsAppInstance(t, app, org.ID)

	now := time.Now().UTC()

	// Create comprehensive test data
	contacts := make([]*models.Contact, 5)
	for i := 0; i < 5; i++ {
		instanceID := instance1.ID
		if i%2 == 0 {
			instanceID = instance2.ID
		}
		contacts[i] = testutil.CreateTestContactWith(t, app.DB, org.ID, withContactInstanceID(instanceID))
	}

	// Create transfers with various statuses and sources
	createTestAgentTransfer(t, app, org.ID, contacts[0].ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-48*time.Hour), &[]time.Time{now.Add(-47*time.Hour)}[0])
	createTestAgentTransfer(t, app, org.ID, contacts[1].ID, &agent.ID, models.TransferStatusActive, models.TransferSourceFlow, now.Add(-24*time.Hour), nil)
	createTestAgentTransfer(t, app, org.ID, contacts[2].ID, &agent2.ID, models.TransferStatusResumed, models.TransferSourceManual, now.Add(-12*time.Hour), &[]time.Time{now.Add(-11*time.Hour)}[0])
	createTestAgentTransfer(t, app, org.ID, contacts[3].ID, &agent2.ID, models.TransferStatusResumed, models.TransferSourceKeyword, now.Add(-6*time.Hour), &[]time.Time{now.Add(-5*time.Hour)}[0])
	createTestAgentTransfer(t, app, org.ID, contacts[4].ID, nil, models.TransferStatusActive, models.TransferSourceManual, now.Add(-1*time.Hour), nil)

	// Create availability logs
	createTestUserAvailabilityLog(t, app, agent.ID, org.ID, false, now.Add(-4*time.Hour), &[]time.Time{now.Add(-3*time.Hour)}[0])
	createTestUserAvailabilityLog(t, app, agent2.ID, org.ID, false, now.Add(-2*time.Hour), nil)

	// Create ratings
	createTestChatClosureRating(t, app, org.ID, contacts[0].ID, agent.ID, &agent.ID, 9, now.Add(-46*time.Hour))
	createTestChatClosureRating(t, app, org.ID, contacts[2].ID, agent2.ID, &agent2.ID, 7, now.Add(-10*time.Hour))

	// Test full analytics endpoint
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var response handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req, &response)

	// Verify summary
	assert.GreaterOrEqual(t, response.Summary.TotalTransfersHandled, int64(3))
	assert.GreaterOrEqual(t, response.Summary.ActiveTransfers, int64(2))

	// Verify agent stats
	require.NotNil(t, response.AgentStats)
	assert.GreaterOrEqual(t, len(response.AgentStats), 2)

	// Verify trend data
	assert.NotEmpty(t, response.TrendData)

	// Verify rating summary
	require.NotNil(t, response.RatingSummary)
	assert.Equal(t, int64(2), response.RatingSummary.TotalRatings)
	assert.Greater(t, response.RatingSummary.AverageRating, float64(0))

	// Verify rating records
	assert.NotEmpty(t, response.RatingRecords)

	// Test with filters
	req2 := testutil.NewGETRequest(t)
	req2.RequestCtx.QueryArgs().Add("instance_id", instance1.ID.String())
	req2.RequestCtx.QueryArgs().Add("min_rating", "8")
	testutil.SetAuthContext(req2, org.ID, admin.ID)

	err2 := app.GetAgentAnalytics(req2)
	require.NoError(t, err2)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req2))

	var response2 handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req2, &response2)

	// Should have filtered results
	require.NotNil(t, response2.RatingSummary)
	assert.Equal(t, int64(1), response2.RatingSummary.TotalRatings)
}

func TestAgentAnalytics_RatingFilters_WorkCorrectly(t *testing.T) {
	app, org, admin, agent := createAgentAnalyticsTestApp(t)

	now := time.Now().UTC()
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Create ratings with scores 1-10
	for score := 1; score <= 10; score++ {
		createTestChatClosureRating(t, app, org.ID, contact.ID, agent.ID, &agent.ID, score, now.Add(-time.Duration(score)*time.Hour))
	}

	// Test min_rating filter
	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("min_rating", "7")
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)

	var response handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req, &response)

	require.NotNil(t, response.RatingSummary)
	assert.Equal(t, int64(4), response.RatingSummary.TotalRatings) // 7, 8, 9, 10

	// Test max_rating filter
	req2 := testutil.NewGETRequest(t)
	req2.RequestCtx.QueryArgs().Add("max_rating", "4")
	testutil.SetAuthContext(req2, org.ID, admin.ID)

	err2 := app.GetAgentAnalytics(req2)
	require.NoError(t, err2)

	var response2 handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req2, &response2)

	require.NotNil(t, response2.RatingSummary)
	assert.Equal(t, int64(4), response2.RatingSummary.TotalRatings) // 1, 2, 3, 4

	// Test range filter
	req3 := testutil.NewGETRequest(t)
	req3.RequestCtx.QueryArgs().Add("min_rating", "5")
	req3.RequestCtx.QueryArgs().Add("max_rating", "7")
	testutil.SetAuthContext(req3, org.ID, admin.ID)

	err3 := app.GetAgentAnalytics(req3)
	require.NoError(t, err3)

	var response3 handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req3, &response3)

	require.NotNil(t, response3.RatingSummary)
	assert.Equal(t, int64(3), response3.RatingSummary.TotalRatings) // 5, 6, 7
}

func TestAgentAnalytics_TrendData_DifferentGroupings(t *testing.T) {
	app, org, admin, agent := createAgentAnalyticsTestApp(t)

	now := time.Now().UTC()
	startDate := now.Add(-14 * 24 * time.Hour)

	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	// Create transfers across multiple days
	for i := 0; i < 14; i++ {
		createTestAgentTransfer(t, app, org.ID, contact.ID, &agent.ID, models.TransferStatusResumed, models.TransferSourceManual, startDate.Add(time.Duration(i)*24*time.Hour), &[]time.Time{startDate.Add(time.Duration(i)*24*time.Hour + time.Hour)}[0])
	}

	// Test day grouping
	req := testutil.NewGETRequest(t)
	req.RequestCtx.QueryArgs().Add("from", startDate.Format("2006-01-02"))
	req.RequestCtx.QueryArgs().Add("to", now.Format("2006-01-02"))
	req.RequestCtx.QueryArgs().Add("group_by", "day")
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)

	var response handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req, &response)

	assert.GreaterOrEqual(t, len(response.TrendData), 14)

	// Test week grouping
	req2 := testutil.NewGETRequest(t)
	req2.RequestCtx.QueryArgs().Add("from", startDate.Format("2006-01-02"))
	req2.RequestCtx.QueryArgs().Add("to", now.Format("2006-01-02"))
	req2.RequestCtx.QueryArgs().Add("group_by", "week")
	testutil.SetAuthContext(req2, org.ID, admin.ID)

	err2 := app.GetAgentAnalytics(req2)
	require.NoError(t, err2)

	var response2 handlers.AgentAnalyticsResponse
	testutil.ParseJSONResponse(t, req2, &response2)

	assert.GreaterOrEqual(t, len(response2.TrendData), 2)
	assert.LessOrEqual(t, len(response2.TrendData), 3)
}
