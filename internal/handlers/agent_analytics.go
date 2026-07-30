package handlers

import (
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// AgentAnalyticsSummary represents overall agent analytics
type AgentAnalyticsSummary struct {
	TotalBreakTimeMins float64 `json:"total_break_time_mins"`
	BreakCount         int64   `json:"break_count"`
	AvgRating          float64 `json:"avg_rating"`
	RatingsCount       int64   `json:"ratings_count"`
}

// AgentPerformanceStats represents performance metrics for an agent
type AgentPerformanceStats struct {
	AgentID            string  `json:"agent_id"`
	AgentName          string  `json:"agent_name"`
	MessagesSent       int64   `json:"messages_sent"`
	TotalBreakTimeMins float64 `json:"total_break_time_mins"`
	BreakCount         int64   `json:"break_count"`
	IsAvailable        bool    `json:"is_available"`
	CurrentBreakStart  *string `json:"current_break_start,omitempty"`
	AvgRating          float64 `json:"avg_rating"`
	RatingsCount       int64   `json:"ratings_count"`
}

// TrendPoint represents a data point for time-series charts (rated closures)
type TrendPoint struct {
	Date         string  `json:"date"`
	RatingsCount int64   `json:"ratings_count"`
	AvgRating    float64 `json:"avg_rating"`
}

// AgentAnalyticsResponse is the full API response
type AgentAnalyticsResponse struct {
	Summary    AgentAnalyticsSummary   `json:"summary"`
	AgentStats []AgentPerformanceStats `json:"agent_stats,omitempty"`
	TrendData  []TrendPoint            `json:"trend_data"`
	MyStats    *AgentPerformanceStats  `json:"my_stats,omitempty"`
}

// GetAgentAnalytics returns agent analytics for the organization
// Agents see only their own stats; Admin/Manager see all agents
func (a *App) GetAgentAnalytics(r *fastglue.Request) error {
	orgID, userID, err := a.requireOrgAndUserID(r)
	if err != nil {
		return nil
	}

	// Parse date range
	fromStr := string(r.RequestCtx.QueryArgs().Peek("from"))
	toStr := string(r.RequestCtx.QueryArgs().Peek("to"))
	groupBy := string(r.RequestCtx.QueryArgs().Peek("group_by"))
	agentIDStr := string(r.RequestCtx.QueryArgs().Peek("agent_id"))
	if groupBy == "" {
		groupBy = "day"
	}

	now := time.Now()
	var periodStart, periodEnd time.Time

	if fromStr != "" && toStr != "" {
		var errMsg string
		periodStart, periodEnd, errMsg = parseDateRange(fromStr, toStr)
		if errMsg != "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, errMsg, nil, "")
		}
	} else {
		// Default to current month
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		periodEnd = now
	}

	response := AgentAnalyticsResponse{
		TrendData: []TrendPoint{},
	}

	// Check if filtering by specific agent (requires analytics permission)
	var filterAgentID *uuid.UUID
	if a.HasPermission(userID, models.ResourceAnalytics, models.ActionRead, orgID) && agentIDStr != "" {
		parsedID, err := uuid.Parse(agentIDStr)
		if err == nil {
			filterAgentID = &parsedID
		}
	}

	if filterAgentID != nil {
		// User with analytics permission viewing specific agent
		agentStats := a.calculateAgentStats(orgID, *filterAgentID, periodStart, periodEnd)
		response.MyStats = &agentStats
		response.TrendData = a.calculateTrendData(orgID, periodStart, periodEnd, groupBy, filterAgentID)
		// Calculate summary for this specific agent
		a.calculateAgentSummaryStats(orgID, *filterAgentID, periodStart, periodEnd, &response.Summary)
	} else if !a.HasPermission(userID, models.ResourceAnalytics, models.ActionRead, orgID) {
		// Users without analytics permission only see their own stats
		myStats := a.calculateAgentStats(orgID, userID, periodStart, periodEnd)
		response.MyStats = &myStats
		response.TrendData = a.calculateTrendData(orgID, periodStart, periodEnd, groupBy, &userID)
		a.calculateAgentSummaryStats(orgID, userID, periodStart, periodEnd, &response.Summary)
	} else {
		// Users with analytics permission see all agents
		a.calculateSummaryStats(orgID, periodStart, periodEnd, &response.Summary)
		response.TrendData = a.calculateTrendData(orgID, periodStart, periodEnd, groupBy, nil)
		response.AgentStats = a.calculateAllAgentStats(orgID, periodStart, periodEnd)
		// Also include current user's stats (for their own break time tracking)
		myStats := a.calculateAgentStats(orgID, userID, periodStart, periodEnd)
		response.MyStats = &myStats
	}

	return r.SendEnvelope(response)
}

// Helper functions

func (a *App) calculateSummaryStats(orgID uuid.UUID, start, end time.Time, summary *AgentAnalyticsSummary) {
	// Customer satisfaction from close-rating cycles (org-wide)
	summary.AvgRating, summary.RatingsCount = a.calculateClosureRatingStats(orgID, nil, start, end)
}

func (a *App) calculateAgentSummaryStats(orgID, agentID uuid.UUID, start, end time.Time, summary *AgentAnalyticsSummary) {
	// Calculate break time
	summary.TotalBreakTimeMins, summary.BreakCount = a.calculateBreakTime(agentID, start, end)

	// Customer satisfaction from close-rating cycles for this agent
	summary.AvgRating, summary.RatingsCount = a.calculateClosureRatingStats(orgID, &agentID, start, end)
}

func (a *App) calculateAgentStats(orgID, agentID uuid.UUID, start, end time.Time) AgentPerformanceStats {
	stats := AgentPerformanceStats{
		AgentID: agentID.String(),
	}

	// Get agent name and availability
	var agent models.User
	if a.DB.Where("id = ?", agentID).First(&agent).Error == nil {
		stats.AgentName = agent.FullName
		stats.IsAvailable = agent.IsAvailable
	}

	// Messages sent by this agent in the period
	a.DB.Model(&models.Message{}).
		Where("organization_id = ? AND direction = ? AND sent_by_user_id = ? AND created_at >= ? AND created_at <= ?",
			orgID, models.DirectionOutgoing, agentID, start, end).
		Count(&stats.MessagesSent)

	// Calculate break time from availability logs
	stats.TotalBreakTimeMins, stats.BreakCount = a.calculateBreakTime(agentID, start, end)

	// Customer satisfaction from close-rating cycles closed by this agent
	stats.AvgRating, stats.RatingsCount = a.calculateClosureRatingStats(orgID, &agentID, start, end)

	// Check if currently on break and get break start time
	if !stats.IsAvailable {
		var currentBreak models.UserAvailabilityLog
		if a.DB.Where("user_id = ? AND is_available = false AND ended_at IS NULL", agentID).
			Order("started_at DESC").First(&currentBreak).Error == nil {
			breakStart := currentBreak.StartedAt.Format(time.RFC3339)
			stats.CurrentBreakStart = &breakStart
		}
	}

	return stats
}

// calculateClosureRatingStats aggregates rated close-rating cycles. When
// agentID is nil the whole organization is aggregated; otherwise only cycles
// closed by that agent are counted.
func (a *App) calculateClosureRatingStats(orgID uuid.UUID, agentID *uuid.UUID, start, end time.Time) (avg float64, count int64) {
	type ratingResult struct {
		Avg   float64
		Count int64
	}
	var result ratingResult
	query := a.DB.Model(&models.ChatClosureRating{}).
		Select("COALESCE(AVG(rating), 0) as avg, COUNT(*) as count").
		Where("organization_id = ? AND status = ? AND rated_at >= ? AND rated_at <= ?",
			orgID, models.RatingStatusRated, start, end)
	if agentID != nil {
		query = query.Where("closed_by_user_id = ?", *agentID)
	}
	query.Scan(&result)
	return result.Avg, result.Count
}

func (a *App) calculateAllAgentStats(orgID uuid.UUID, start, end time.Time) []AgentPerformanceStats {
	// List every user who acts as an agent in this org: team members with the
	// agent role, plus anyone who closed rated conversations — so orgs that
	// don't use teams still see their staff here.
	var agents []models.User
	if err := a.DB.
		Where("users.organization_id = ?", orgID).
		Where(`users.id IN (SELECT team_members.user_id FROM team_members
				JOIN teams ON teams.id = team_members.team_id
				WHERE teams.organization_id = ? AND team_members.role = ?)
			OR users.id IN (SELECT closed_by_user_id FROM chat_closure_ratings
				WHERE organization_id = ? AND closed_by_user_id IS NOT NULL)`,
			orgID, models.TeamRoleAgent, orgID).
		Find(&agents).Error; err != nil {
		a.Log.Error("Failed to fetch agents for analytics", "error", err, "org_id", orgID)
		return []AgentPerformanceStats{}
	}

	stats := make([]AgentPerformanceStats, 0, len(agents))
	for _, agent := range agents {
		agentStats := a.calculateAgentStats(orgID, agent.ID, start, end)
		stats = append(stats, agentStats)
	}

	return stats
}

// calculateBreakTime calculates total break time and count for an agent within a time period
func (a *App) calculateBreakTime(agentID uuid.UUID, start, end time.Time) (totalMins float64, count int64) {
	// Get all "away" periods that overlap with the time range
	var logs []models.UserAvailabilityLog
	if err := a.DB.Where("user_id = ? AND is_available = false AND started_at <= ? AND (ended_at >= ? OR ended_at IS NULL)",
		agentID, end, start).
		Find(&logs).Error; err != nil {
		a.Log.Error("Failed to fetch availability logs for break time calculation", "error", err, "agent_id", agentID)
		return 0, 0
	}

	for _, log := range logs {
		// Calculate the overlap with our time range
		logStart := log.StartedAt
		if logStart.Before(start) {
			logStart = start
		}

		var logEnd time.Time
		if log.EndedAt != nil {
			logEnd = *log.EndedAt
		} else {
			// Still on break, use current time but cap at end of period
			logEnd = time.Now()
		}
		if logEnd.After(end) {
			logEnd = end
		}

		// Add duration in minutes
		if logEnd.After(logStart) {
			duration := logEnd.Sub(logStart).Minutes()
			totalMins += duration
			count++
		}
	}

	return totalMins, count
}

// calculateTrendData builds a time series of rated close-rating cycles.
func (a *App) calculateTrendData(orgID uuid.UUID, start, end time.Time, groupBy string, agentID *uuid.UUID) []TrendPoint {
	var dateFormat string
	var dateTrunc string

	switch groupBy {
	case "week":
		dateFormat = "2006-01-02"
		dateTrunc = "week"
	default: // day
		dateFormat = "2006-01-02"
		dateTrunc = "day"
	}

	type TrendResult struct {
		Date  time.Time
		Count int64
		Avg   float64
	}

	query := a.DB.Model(&models.ChatClosureRating{}).
		Select("DATE_TRUNC('"+dateTrunc+"', rated_at) as date, COUNT(*) as count, COALESCE(AVG(rating), 0) as avg").
		Where("organization_id = ? AND status = ? AND rated_at >= ? AND rated_at <= ?",
			orgID, models.RatingStatusRated, start, end)

	if agentID != nil {
		query = query.Where("closed_by_user_id = ?", *agentID)
	}

	var results []TrendResult
	query.Group("DATE_TRUNC('" + dateTrunc + "', rated_at)").
		Order("date ASC").
		Scan(&results)

	trendData := make([]TrendPoint, len(results))
	for i, r := range results {
		trendData[i] = TrendPoint{
			Date:         r.Date.Format(dateFormat),
			RatingsCount: r.Count,
			AvgRating:    r.Avg,
		}
	}

	return trendData
}
