package handlers_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
		Category:        "MARKETING",
		Language:        "en",
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

	// The table lists holders of the system agent role (Decision 10): the
	// team membership above is not what puts anyone in the table.
	createSystemAgent(t, app, org.ID, "Table Agent")

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

// createTestClosureRating creates a chat closure rating cycle in the database.
func createTestClosureRating(t *testing.T, app *handlers.App, orgID, contactID uuid.UUID, closedBy *uuid.UUID, status string, rating *int, ratedAt *time.Time) *models.ChatClosureRating {
	t.Helper()

	cycle := &models.ChatClosureRating{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		ContactID:       contactID,
		WhatsAppAccount: "test-account",
		ClosedByUserID:  closedBy,
		Status:          status,
		Rating:          rating,
		RatedAt:         ratedAt,
		ExpiresAt:       time.Now().UTC().Add(48 * time.Hour),
	}
	require.NoError(t, app.DB.Create(cycle).Error)
	return cycle
}

func TestApp_GetAgentAnalytics_ClosureRatings(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	// The closing user holds the system agent role (Decision 10) plus
	// analytics:read so they both view the table and appear in it.
	role := systemAgentRole(t, app, org.ID, "analytics:read")
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("agent-ratings")),
		testutil.WithPassword("password"),
		testutil.WithRoleID(&role.ID),
	)

	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	now := time.Now().UTC()
	ratedAt := now.Add(-1 * time.Hour)

	r4, r5 := 4, 5
	createTestClosureRating(t, app, org.ID, contact.ID, &user.ID, models.RatingStatusRated, &r4, &ratedAt)
	createTestClosureRating(t, app, org.ID, contact.ID, &user.ID, models.RatingStatusRated, &r5, &ratedAt)
	// Pending and expired cycles must NOT count toward the average.
	createTestClosureRating(t, app, org.ID, contact.ID, &user.ID, models.RatingStatusPending, nil, nil)
	createTestClosureRating(t, app, org.ID, contact.ID, &user.ID, models.RatingStatusExpired, nil, nil)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	// Explicit range so the default month window can't clip ratedAt on the 1st.
	testutil.SetQueryParam(req, "from", now.Add(-48*time.Hour).Format("2006-01-02"))
	testutil.SetQueryParam(req, "to", now.Format("2006-01-02"))

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AgentAnalyticsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	// Org-wide summary aggregates only rated cycles.
	assert.Equal(t, int64(2), resp.Data.Summary.RatingsCount)
	assert.InDelta(t, 4.5, resp.Data.Summary.AvgRating, 0.001)

	// The closing user appears in agent stats without any team membership,
	// carrying their name and rating aggregate.
	require.Len(t, resp.Data.AgentStats, 1)
	agentStat := resp.Data.AgentStats[0]
	assert.Equal(t, user.ID.String(), agentStat.AgentID)
	assert.Equal(t, user.FullName, agentStat.AgentName)
	assert.Equal(t, int64(2), agentStat.RatingsCount)
	assert.InDelta(t, 4.5, agentStat.AvgRating, 0.001)

	// my_stats carries the same rating aggregate for the requesting user.
	require.NotNil(t, resp.Data.MyStats)
	assert.Equal(t, int64(2), resp.Data.MyStats.RatingsCount)
	assert.InDelta(t, 4.5, resp.Data.MyStats.AvgRating, 0.001)
}

func TestApp_GetAgentAnalytics_NoRatings(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	// The user holds the system agent role (Decision 10) plus analytics:read.
	role := systemAgentRole(t, app, org.ID, "analytics:read")
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("agent-noratings")),
		testutil.WithPassword("password"),
		testutil.WithRoleID(&role.ID),
	)
	createTestTeamWithAgent(t, app, org.ID, user.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.GetAgentAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AgentAnalyticsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	assert.Equal(t, int64(0), resp.Data.Summary.RatingsCount)
	assert.Equal(t, float64(0), resp.Data.Summary.AvgRating)
	require.Len(t, resp.Data.AgentStats, 1)
	assert.Equal(t, int64(0), resp.Data.AgentStats[0].RatingsCount)
	assert.Equal(t, float64(0), resp.Data.AgentStats[0].AvgRating)
}

// --- Task 4: break time org scope + org-total summary, agent population (Decisions 9, 10) ---

func TestApp_GetAgentAnalytics_BreakTimeOrgTotal(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	otherOrg := testutil.CreateTestOrganization(t, app.DB)

	// Viewer holds the system agent role plus analytics:read; two agents each
	// take a 30-minute break inside the current month.
	viewer := createSystemAgent(t, app, org.ID, "Break Viewer", "analytics:read")
	agentA := createSystemAgent(t, app, org.ID, "Break Agent A")
	agentB := createSystemAgent(t, app, org.ID, "Break Agent B")

	now := time.Now().UTC()
	createAvailabilityLog(t, app, org.ID, agentA.ID, false, now.Add(-60*time.Minute), now.Add(-30*time.Minute))
	createAvailabilityLog(t, app, org.ID, agentB.ID, false, now.Add(-60*time.Minute), now.Add(-30*time.Minute))
	// Same user, different org: must NOT leak into this org's numbers.
	createAvailabilityLog(t, app, otherOrg.ID, agentA.ID, false, now.Add(-60*time.Minute), now.Add(-30*time.Minute))

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, viewer.ID)

	require.NoError(t, app.GetAgentAnalytics(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AgentAnalyticsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	// All-agents summary is the ORG TOTAL, not the viewer's own break time
	// (the viewer took no break) and not inflated by the other org's log.
	assert.InDelta(t, 60, resp.Data.Summary.TotalBreakTimeMins, 1.0)
	assert.Equal(t, int64(2), resp.Data.Summary.BreakCount)

	// Per-agent rows carry each agent's own break time.
	require.Len(t, resp.Data.AgentStats, 3) // viewer + A + B, no duplicates
	byID := make(map[string]handlers.AgentPerformanceStats, len(resp.Data.AgentStats))
	for _, s := range resp.Data.AgentStats {
		byID[s.AgentID] = s
	}
	require.Contains(t, byID, agentA.ID.String())
	require.Contains(t, byID, agentB.ID.String())
	assert.InDelta(t, 30, byID[agentA.ID.String()].TotalBreakTimeMins, 1.0)
	assert.InDelta(t, 30, byID[agentB.ID.String()].TotalBreakTimeMins, 1.0)
}

func TestApp_GetAgentAnalytics_BreakTimeOwnStatsOrgScoped(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	otherOrg := testutil.CreateTestOrganization(t, app.DB)

	// Agent without analytics permission sees only their own stats.
	agent := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("break-own")),
		testutil.WithFullName("Break Own"),
	)

	now := time.Now().UTC()
	createAvailabilityLog(t, app, org.ID, agent.ID, false, now.Add(-60*time.Minute), now.Add(-30*time.Minute))
	// Same user, different org: must NOT count here.
	createAvailabilityLog(t, app, otherOrg.ID, agent.ID, false, now.Add(-90*time.Minute), now.Add(-45*time.Minute))

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)

	require.NoError(t, app.GetAgentAnalytics(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AgentAnalyticsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	require.NotNil(t, resp.Data.MyStats)
	assert.InDelta(t, 30, resp.Data.MyStats.TotalBreakTimeMins, 1.0)
	assert.Equal(t, int64(1), resp.Data.MyStats.BreakCount)
}

func TestApp_GetAgentAnalytics_AgentPopulationIsSystemRoleHolders(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Viewer with analytics:read (not an agent).
	perms := getAnalyticsPermissions(t, app)
	viewerRole := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Analytics Viewer", false, false, perms)
	viewer := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("analytics-viewer")),
		testutil.WithRoleID(&viewerRole.ID),
	)

	// Agent-role holders: one with zero activity of any kind, one team member.
	zeroAgent := createSystemAgent(t, app, org.ID, "Zero Activity Agent")
	teamAgent := createSystemAgent(t, app, org.ID, "Team Agent")
	createTestTeamWithAgent(t, app, org.ID, teamAgent.ID)

	// Team member with team-role agent but NO agent system role: must NOT appear.
	adminRole := testutil.CreateAdminRole(t, app.DB, org.ID)
	teamOnly := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("team-only")),
		testutil.WithFullName("Team Only User"),
		testutil.WithRoleID(&adminRole.ID),
	)
	createTestTeamWithAgent(t, app, org.ID, teamOnly.ID)

	// Rated closer with no agent system role and no team: must NOT appear.
	closer := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("rated-closer")),
		testutil.WithFullName("Rated Closer"),
	)
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	rating := 5
	ratedAt := time.Now().UTC().Add(-1 * time.Hour)
	createTestClosureRating(t, app, org.ID, contact.ID, &closer.ID, models.RatingStatusRated, &rating, &ratedAt)

	// NOTE (fix round 1): no duplicate-membership row is constructed here
	// because none is constructible. The partial index
	// idx_user_org_unique on user_organizations(user_id, organization_id)
	// WHERE deleted_at IS NULL forbids two ACTIVE rows, and the model tag's
	// non-partial uniqueIndex idx_user_org on the same columns (applied by
	// AutoMigrate, verified present in the test DB alongside the partial
	// one) forbids even a soft-deleted second row — an insert attempt fails
	// with a 23505 violation on idx_user_org. So DISTINCT in
	// listAnalyticsAgents is defense-in-depth only, and is pinned at the
	// query level instead: see TestApp_GetAgentAnalytics_ListQueryIsDistinct.

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, viewer.ID)

	require.NoError(t, app.GetAgentAnalytics(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AgentAnalyticsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	// Table = agent system-role holders, including zero-activity ones.
	require.Len(t, resp.Data.AgentStats, 2)
	ids := []string{resp.Data.AgentStats[0].AgentID, resp.Data.AgentStats[1].AgentID}
	assert.Contains(t, ids, zeroAgent.ID.String())
	assert.Contains(t, ids, teamAgent.ID.String())
	for _, s := range resp.Data.AgentStats {
		if s.AgentID == zeroAgent.ID.String() {
			assert.Equal(t, int64(0), s.MessagesSent)
			assert.Equal(t, int64(0), s.RatingsCount)
		}
	}
}

// TestApp_GetAgentAnalytics_DeletedAgentRoleHoldersExcluded pins that the
// custom_roles join in listAnalyticsAgents excludes soft-deleted roles:
// a user whose only agent role was soft-deleted must not appear in the
// table (and must not inflate the org-total break time).
func TestApp_GetAgentAnalytics_DeletedAgentRoleHoldersExcluded(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	// Viewer with analytics:read (not an agent).
	perms := getAnalyticsPermissions(t, app)
	viewerRole := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Analytics Viewer", false, false, perms)
	viewer := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("analytics-viewer")),
		testutil.WithRoleID(&viewerRole.ID),
	)

	// Agent-role holder, then soft-delete the org's agent role itself.
	agent := createSystemAgent(t, app, org.ID, "Deleted Role Agent")
	role := systemAgentRole(t, app, org.ID)
	require.NoError(t, app.DB.Delete(role).Error)

	// The agent took a break before the role was deleted: it must not leak
	// into the org total via a stale holder row.
	now := time.Now().UTC()
	createAvailabilityLog(t, app, org.ID, agent.ID, false, now.Add(-60*time.Minute), now.Add(-30*time.Minute))

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, viewer.ID)

	require.NoError(t, app.GetAgentAnalytics(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.AgentAnalyticsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))

	require.Len(t, resp.Data.AgentStats, 0)
	assert.InDelta(t, 0, resp.Data.Summary.TotalBreakTimeMins, 0.0)
	assert.Equal(t, int64(0), resp.Data.Summary.BreakCount)
}

// queryCaptureLogger records every SQL statement executed through a GORM
// session. Used to pin the shape of a query the handler actually runs,
// where the behaviour the query guards against cannot be constructed
// (see the note in TestApp_GetAgentAnalytics_AgentPopulationIsSystemRoleHolders).
type queryCaptureLogger struct {
	mu      sync.Mutex
	queries []string
}

func (l *queryCaptureLogger) LogMode(logger.LogLevel) logger.Interface      { return l }
func (l *queryCaptureLogger) Info(context.Context, string, ...interface{})  {}
func (l *queryCaptureLogger) Warn(context.Context, string, ...interface{})  {}
func (l *queryCaptureLogger) Error(context.Context, string, ...interface{}) {}
func (l *queryCaptureLogger) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	l.mu.Lock()
	l.queries = append(l.queries, sql)
	l.mu.Unlock()
}

// TestApp_GetAgentAnalytics_ListQueryIsDistinct pins the hardening in
// listAnalyticsAgents at the query level: the agent-population query the
// endpoint actually runs must SELECT DISTINCT over users and must exclude
// soft-deleted rows on both joins. A duplicate membership row cannot be
// constructed (see the note in the population test), so no behavioural
// test can exercise the DISTINCT — this assertion is the pin.
func TestApp_GetAgentAnalytics_ListQueryIsDistinct(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)

	perms := getAnalyticsPermissions(t, app)
	viewerRole := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Analytics Viewer", false, false, perms)
	viewer := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("analytics-viewer")),
		testutil.WithRoleID(&viewerRole.ID),
	)
	createSystemAgent(t, app, org.ID, "Distinct Agent")

	capture := &queryCaptureLogger{}
	origDB := app.DB
	app.DB = app.DB.Session(&gorm.Session{Logger: capture})
	t.Cleanup(func() { app.DB = origDB })

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, viewer.ID)
	require.NoError(t, app.GetAgentAnalytics(req))
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var found string
	for _, sql := range capture.queries {
		if strings.Contains(sql, "user_organizations") && strings.Contains(sql, "custom_roles") {
			found = sql
			break
		}
	}
	require.NotEmpty(t, found, "expected the agent-population query to run; captured: %v", capture.queries)
	assert.Contains(t, found, "DISTINCT")
	assert.Contains(t, found, "custom_roles.deleted_at IS NULL")
	assert.Contains(t, found, "user_organizations.deleted_at IS NULL")
}
