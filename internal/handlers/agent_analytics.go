package handlers

import (
	"errors"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// AgentAnalyticsSummary represents overall agent analytics
type AgentAnalyticsSummary struct {
	TotalTransfersHandled int64            `json:"total_transfers_handled"`
	ActiveTransfers       int64            `json:"active_transfers"`
	AvgQueueTimeMins      float64          `json:"avg_queue_time_mins"`
	AvgFirstResponseMins  float64          `json:"avg_first_response_mins"`
	AvgResolutionMins     float64          `json:"avg_resolution_mins"`
	TransfersBySource     map[string]int64 `json:"transfers_by_source"`
	TotalBreakTimeMins    float64          `json:"total_break_time_mins"`
	BreakCount            int64            `json:"break_count"`
}

// AgentPerformanceStats represents performance metrics for an agent
type AgentPerformanceStats struct {
	AgentID              string  `json:"agent_id"`
	AgentName            string  `json:"agent_name"`
	AvgFirstResponseMins float64 `json:"avg_first_response_mins"`
	AvgResolutionMins    float64 `json:"avg_resolution_mins"`
	TransfersHandled     int64   `json:"transfers_handled"`
	ActiveTransfers      int64   `json:"active_transfers"`
	MessagesSent         int64   `json:"messages_sent"`
	TotalBreakTimeMins   float64 `json:"total_break_time_mins"`
	BreakCount           int64   `json:"break_count"`
	IsAvailable          bool    `json:"is_available"`
	CurrentBreakStart    *string `json:"current_break_start,omitempty"`
}

// TrendPoint represents a data point for time-series charts
type TrendPoint struct {
	Date             string  `json:"date"`
	TransfersHandled int64   `json:"transfers_handled"`
	AvgResponseMins  float64 `json:"avg_response_mins"`
}

// AgentAnalyticsResponse is the full API response
type AgentAnalyticsResponse struct {
	Summary       AgentAnalyticsSummary   `json:"summary"`
	AgentStats    []AgentPerformanceStats `json:"agent_stats,omitempty"`
	TrendData     []TrendPoint            `json:"trend_data"`
	MyStats       *AgentPerformanceStats  `json:"my_stats,omitempty"`
	RatingSummary *AgentRatingSummary     `json:"rating_summary,omitempty"`
	RatingRecords []AgentRatingRecord     `json:"rating_records,omitempty"`
}

// GetAgentAnalytics returns agent analytics for the organization
// Agents see only their own stats; Admin/Manager see all agents
func (a *App) GetAgentAnalytics(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Parse date range
	fromStr := string(r.RequestCtx.QueryArgs().Peek("from"))
	toStr := string(r.RequestCtx.QueryArgs().Peek("to"))
	groupBy := string(r.RequestCtx.QueryArgs().Peek("group_by"))
	agentIDStr := string(r.RequestCtx.QueryArgs().Peek("agent_id"))
	instanceIDStr := string(r.RequestCtx.QueryArgs().Peek("instance_id"))
	minRating, minRatingErr := parseRatingFilterBound(string(r.RequestCtx.QueryArgs().Peek("min_rating")))
	if minRatingErr != "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, minRatingErr, nil, "")
	}
	maxRating, maxRatingErr := parseRatingFilterBound(string(r.RequestCtx.QueryArgs().Peek("max_rating")))
	if maxRatingErr != "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, maxRatingErr, nil, "")
	}
	if minRating != nil && maxRating != nil && *minRating > *maxRating {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "min_rating cannot be greater than max_rating", nil, "")
	}
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
		Summary: AgentAnalyticsSummary{
			TransfersBySource: make(map[string]int64),
		},
		TrendData: []TrendPoint{},
	}

	filterInstanceID, instanceErr := a.parseAnalyticsInstanceID(orgID, instanceIDStr)
	if instanceErr != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, instanceErr.Error(), nil, "instance_id")
	}

	// Check if filtering by specific agent (requires analytics permission)
	var filterAgentID *uuid.UUID
	if a.HasPermission(userID, models.ResourceAnalytics, models.ActionRead, orgID) && agentIDStr != "" {
		parsedID, parseErr := uuid.Parse(agentIDStr)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid agent_id", nil, "agent_id")
		}
		inOrg, scopeErr := a.analyticsAgentBelongsToOrg(orgID, parsedID)
		if scopeErr != nil {
			a.Log.Error("Failed to validate analytics agent scope", "error", scopeErr, "organization_id", orgID, "agent_id", parsedID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch analytics", nil, "")
		}
		if !inOrg {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Agent not found", nil, "")
		}
		filterAgentID = &parsedID
	}

	if filterAgentID != nil {
		// User with analytics permission viewing specific agent
		agentStats := a.calculateAgentStats(orgID, *filterAgentID, periodStart, periodEnd, filterInstanceID)
		response.MyStats = &agentStats
		response.TrendData = a.calculateTrendData(orgID, periodStart, periodEnd, groupBy, filterAgentID, filterInstanceID)
		a.calculateAgentSummaryStats(orgID, *filterAgentID, periodStart, periodEnd, &response.Summary, filterInstanceID)

		ratingSummary, summaryErr := a.calculateAgentRatingSummary(orgID, periodStart, periodEnd, filterAgentID, filterInstanceID, minRating, maxRating)
		if summaryErr != nil {
			a.Log.Error("Failed to calculate agent rating summary", "error", summaryErr, "organization_id", orgID)
		} else {
			response.RatingSummary = &ratingSummary
		}

		ratingRecords, recordsErr := a.listAgentRatingRecords(orgID, periodStart, periodEnd, filterAgentID, filterInstanceID, minRating, maxRating, 200)
		if recordsErr != nil {
			a.Log.Error("Failed to list agent rating records", "error", recordsErr, "organization_id", orgID)
		} else {
			response.RatingRecords = ratingRecords
		}
	} else if !a.HasPermission(userID, models.ResourceAnalytics, models.ActionRead, orgID) {
		// Users without analytics permission only see their own stats
		myStats := a.calculateAgentStats(orgID, userID, periodStart, periodEnd, filterInstanceID)
		response.MyStats = &myStats
		response.TrendData = a.calculateTrendData(orgID, periodStart, periodEnd, groupBy, &userID, filterInstanceID)
		a.calculateAgentSummaryStats(orgID, userID, periodStart, periodEnd, &response.Summary, filterInstanceID)

		ratingSummary, summaryErr := a.calculateAgentRatingSummary(orgID, periodStart, periodEnd, &userID, filterInstanceID, minRating, maxRating)
		if summaryErr != nil {
			a.Log.Error("Failed to calculate agent self rating summary", "error", summaryErr, "organization_id", orgID, "agent_id", userID)
		} else {
			response.RatingSummary = &ratingSummary
		}
	} else {
		// Users with analytics permission see all agents
		a.calculateSummaryStats(orgID, periodStart, periodEnd, &response.Summary, filterInstanceID)
		response.TrendData = a.calculateTrendData(orgID, periodStart, periodEnd, groupBy, nil, filterInstanceID)
		response.AgentStats = a.calculateAllAgentStats(orgID, periodStart, periodEnd, filterInstanceID)
		myStats := a.calculateAgentStats(orgID, userID, periodStart, periodEnd, filterInstanceID)
		response.MyStats = &myStats

		ratingSummary, summaryErr := a.calculateAgentRatingSummary(orgID, periodStart, periodEnd, nil, filterInstanceID, minRating, maxRating)
		if summaryErr != nil {
			a.Log.Error("Failed to calculate rating summary", "error", summaryErr, "organization_id", orgID)
		} else {
			response.RatingSummary = &ratingSummary
		}

		ratingRecords, recordsErr := a.listAgentRatingRecords(orgID, periodStart, periodEnd, nil, filterInstanceID, minRating, maxRating, 200)
		if recordsErr != nil {
			a.Log.Error("Failed to list rating records", "error", recordsErr, "organization_id", orgID)
		} else {
			response.RatingRecords = ratingRecords
		}
	}

	return r.SendEnvelope(response)
}

// GetAgentDetails returns detailed analytics for a specific agent
func (a *App) GetAgentDetails(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceAnalytics, models.ActionRead); err != nil {
		return nil
	}

	agentID, err := parsePathUUID(r, "id", "agent")
	if err != nil {
		return nil
	}

	// Parse date range
	fromStr := string(r.RequestCtx.QueryArgs().Peek("from"))
	toStr := string(r.RequestCtx.QueryArgs().Peek("to"))
	groupBy := string(r.RequestCtx.QueryArgs().Peek("group_by"))
	if groupBy == "" {
		groupBy = "day"
	}

	now := time.Now()
	var periodStart, periodEnd time.Time

	if fromStr != "" && toStr != "" {
		var errMsg string
		periodStart, periodEnd, errMsg = parseDateRange(fromStr, toStr)
		if errMsg != "" {
			// Fall back to current month on parse error
			periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			periodEnd = now
		}
	} else {
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		periodEnd = now
	}

	// Verify agent exists (checks both native and cross-org members via user_organizations)
	var agent models.User
	if err := a.DB.Where("id = ? AND deleted_at IS NULL", agentID).First(&agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Agent not found", nil, "")
		}
		a.Log.Error("GetAgentAnalytics: Failed to fetch agent", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch agent", nil, "")
	}
	if !a.userBelongsToOrg(a.DB, agentID, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Agent not found", nil, "")
	}

	filterInstanceID, instanceErr := a.parseAnalyticsInstanceID(orgID, string(r.RequestCtx.QueryArgs().Peek("instance_id")))
	if instanceErr != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, instanceErr.Error(), nil, "instance_id")
	}

	stats := a.calculateAgentStats(orgID, agentID, periodStart, periodEnd, filterInstanceID)
	trendData := a.calculateTrendData(orgID, periodStart, periodEnd, groupBy, &agentID, filterInstanceID)

	return r.SendEnvelope(map[string]any{
		"agent":      stats,
		"trend_data": trendData,
	})
}

// GetAgentComparison returns comparison data for multiple agents
func (a *App) GetAgentComparison(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if !a.HasPermission(userID, models.ResourceAnalytics, models.ActionRead, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Access denied", nil, "")
	}

	// Parse date range
	fromStr := string(r.RequestCtx.QueryArgs().Peek("from"))
	toStr := string(r.RequestCtx.QueryArgs().Peek("to"))

	now := time.Now()
	var periodStart, periodEnd time.Time

	if fromStr != "" && toStr != "" {
		var errMsg string
		periodStart, periodEnd, errMsg = parseDateRange(fromStr, toStr)
		if errMsg != "" {
			// Fall back to current month on parse error
			periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			periodEnd = now
		}
	} else {
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		periodEnd = now
	}

	filterInstanceID, instanceErr := a.parseAnalyticsInstanceID(orgID, string(r.RequestCtx.QueryArgs().Peek("instance_id")))
	if instanceErr != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, instanceErr.Error(), nil, "instance_id")
	}

	agentStats := a.calculateAllAgentStats(orgID, periodStart, periodEnd, filterInstanceID)

	return r.SendEnvelope(map[string]any{
		"agents": agentStats,
	})
}

// Helper functions

func (a *App) calculateSummaryStats(orgID uuid.UUID, start, end time.Time, summary *AgentAnalyticsSummary, instanceID *uuid.UUID) {
	// Total transfers handled (resumed)
	totalTransfersQuery := a.DB.Model(&models.AgentTransfer{})
	totalTransfersQuery = applyTransferAnalyticsInstanceFilter(totalTransfersQuery, orgID, instanceID)
	totalTransfersQuery.
		Where("organization_id = ? AND status = ? AND transferred_at >= ? AND transferred_at <= ?",
			orgID, models.TransferStatusResumed, start, end).
		Count(&summary.TotalTransfersHandled)

	// Active transfers
	activeTransfersQuery := a.DB.Model(&models.AgentTransfer{})
	activeTransfersQuery = applyTransferAnalyticsInstanceFilter(activeTransfersQuery, orgID, instanceID)
	activeTransfersQuery.
		Where("organization_id = ? AND status = ?", orgID, models.TransferStatusActive).
		Count(&summary.ActiveTransfers)

	// Average queue time (time from transfer to assignment for assigned transfers)
	type AvgResult struct {
		Avg float64
	}
	var queueTimeResult AvgResult
	queueTimeQuery := a.DB.Model(&models.AgentTransfer{})
	queueTimeQuery = applyTransferAnalyticsInstanceFilter(queueTimeQuery, orgID, instanceID)
	queueTimeQuery.
		Select("AVG(EXTRACT(EPOCH FROM (updated_at - transferred_at))/60) as avg").
		Where("organization_id = ? AND agent_id IS NOT NULL AND transferred_at >= ? AND transferred_at <= ?",
			orgID, start, end).
		Scan(&queueTimeResult)
	summary.AvgQueueTimeMins = queueTimeResult.Avg

	// Average resolution time (time from transfer to resume)
	var resolutionTimeResult AvgResult
	resolutionTimeQuery := a.DB.Model(&models.AgentTransfer{})
	resolutionTimeQuery = applyTransferAnalyticsInstanceFilter(resolutionTimeQuery, orgID, instanceID)
	resolutionTimeQuery.
		Select("AVG(EXTRACT(EPOCH FROM (resumed_at - transferred_at))/60) as avg").
		Where("organization_id = ? AND status = ? AND resumed_at IS NOT NULL AND transferred_at >= ? AND transferred_at <= ?",
			orgID, models.TransferStatusResumed, start, end).
		Scan(&resolutionTimeResult)
	summary.AvgResolutionMins = resolutionTimeResult.Avg

	// Transfers by source
	type SourceCount struct {
		Source string
		Count  int64
	}
	var sourceCounts []SourceCount
	sourceCountsQuery := a.DB.Model(&models.AgentTransfer{})
	sourceCountsQuery = applyTransferAnalyticsInstanceFilter(sourceCountsQuery, orgID, instanceID)
	sourceCountsQuery.
		Select("source, COUNT(*) as count").
		Where("organization_id = ? AND transferred_at >= ? AND transferred_at <= ?", orgID, start, end).
		Group("source").
		Scan(&sourceCounts)

	if summary.TransfersBySource == nil {
		summary.TransfersBySource = make(map[string]int64)
	}
	for _, sc := range sourceCounts {
		summary.TransfersBySource[sc.Source] = sc.Count
	}
}

func (a *App) calculateAgentSummaryStats(orgID, agentID uuid.UUID, start, end time.Time, summary *AgentAnalyticsSummary, instanceID *uuid.UUID) {
	// Total transfers handled by this agent (resumed)
	totalTransfersQuery := a.DB.Model(&models.AgentTransfer{})
	totalTransfersQuery = applyTransferAnalyticsInstanceFilter(totalTransfersQuery, orgID, instanceID)
	totalTransfersQuery.
		Where("organization_id = ? AND agent_id = ? AND status = ? AND transferred_at >= ? AND transferred_at <= ?",
			orgID, agentID, models.TransferStatusResumed, start, end).
		Count(&summary.TotalTransfersHandled)

	// Active transfers for this agent
	activeTransfersQuery := a.DB.Model(&models.AgentTransfer{})
	activeTransfersQuery = applyTransferAnalyticsInstanceFilter(activeTransfersQuery, orgID, instanceID)
	activeTransfersQuery.
		Where("organization_id = ? AND agent_id = ? AND status = ?", orgID, agentID, models.TransferStatusActive).
		Count(&summary.ActiveTransfers)

	// Average resolution time for this agent
	type AvgResult struct {
		Avg float64
	}
	var resolutionTimeResult AvgResult
	resolutionQuery := a.DB.Model(&models.AgentTransfer{})
	resolutionQuery = applyTransferAnalyticsInstanceFilter(resolutionQuery, orgID, instanceID)
	resolutionQuery.
		Select("AVG(EXTRACT(EPOCH FROM (resumed_at - transferred_at))/60) as avg").
		Where("organization_id = ? AND agent_id = ? AND status = ? AND resumed_at IS NOT NULL AND transferred_at >= ? AND transferred_at <= ?",
			orgID, agentID, models.TransferStatusResumed, start, end).
		Scan(&resolutionTimeResult)
	summary.AvgResolutionMins = resolutionTimeResult.Avg

	// Transfers by source for this agent
	type SourceCount struct {
		Source string
		Count  int64
	}
	var sourceCounts []SourceCount
	sourceCountsQuery := a.DB.Model(&models.AgentTransfer{})
	sourceCountsQuery = applyTransferAnalyticsInstanceFilter(sourceCountsQuery, orgID, instanceID)
	sourceCountsQuery.
		Select("source, COUNT(*) as count").
		Where("organization_id = ? AND agent_id = ? AND transferred_at >= ? AND transferred_at <= ?", orgID, agentID, start, end).
		Group("source").
		Scan(&sourceCounts)

	if summary.TransfersBySource == nil {
		summary.TransfersBySource = make(map[string]int64)
	}
	for _, sc := range sourceCounts {
		summary.TransfersBySource[sc.Source] = sc.Count
	}

	// Calculate break time
	summary.TotalBreakTimeMins, summary.BreakCount = a.calculateBreakTime(orgID, agentID, start, end)
}

func (a *App) calculateAgentStats(orgID, agentID uuid.UUID, start, end time.Time, instanceID *uuid.UUID) AgentPerformanceStats {
	stats := AgentPerformanceStats{
		AgentID: agentID.String(),
	}

	// Get agent name and availability
	var agent models.User
	if a.DB.
		Select("users.full_name", "users.is_available").
		Joins("JOIN user_organizations ON user_organizations.user_id = users.id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID).
		Where("users.id = ? AND users.deleted_at IS NULL", agentID).
		First(&agent).Error == nil {
		stats.AgentName = agent.FullName
		stats.IsAvailable = agent.IsAvailable
	}

	// Transfers handled (resumed)
	handledTransfersQuery := a.DB.Model(&models.AgentTransfer{})
	handledTransfersQuery = applyTransferAnalyticsInstanceFilter(handledTransfersQuery, orgID, instanceID)
	handledTransfersQuery.
		Where("organization_id = ? AND agent_id = ? AND status = ? AND transferred_at >= ? AND transferred_at <= ?",
			orgID, agentID, models.TransferStatusResumed, start, end).
		Count(&stats.TransfersHandled)

	// Active transfers
	activeTransfersQuery := a.DB.Model(&models.AgentTransfer{})
	activeTransfersQuery = applyTransferAnalyticsInstanceFilter(activeTransfersQuery, orgID, instanceID)
	activeTransfersQuery.
		Where("organization_id = ? AND agent_id = ? AND status = ?", orgID, agentID, models.TransferStatusActive).
		Count(&stats.ActiveTransfers)

	// Messages sent - count outgoing messages to contacts during agent's active transfers
	// This captures all messages sent while the agent was handling the conversation
	messagesQuery := a.DB.Model(&models.Message{})
	if instanceID != nil {
		messagesQuery = messagesQuery.Where("instance_id = ?", *instanceID)
	}
	messagesQuery.
		Where("organization_id = ? AND direction = ? AND created_at >= ? AND created_at <= ?", orgID, models.DirectionOutgoing, start, end).
		Where("contact_id IN (SELECT contact_id FROM agent_transfers WHERE agent_id = ? AND organization_id = ?)", agentID, orgID).
		Count(&stats.MessagesSent)

	// Average resolution time
	type AvgResult struct {
		Avg float64
	}
	var resolutionTimeResult AvgResult
	resolutionQuery := a.DB.Model(&models.AgentTransfer{})
	resolutionQuery = applyTransferAnalyticsInstanceFilter(resolutionQuery, orgID, instanceID)
	resolutionQuery.
		Select("AVG(EXTRACT(EPOCH FROM (resumed_at - transferred_at))/60) as avg").
		Where("organization_id = ? AND agent_id = ? AND status = ? AND resumed_at IS NOT NULL AND transferred_at >= ? AND transferred_at <= ?",
			orgID, agentID, models.TransferStatusResumed, start, end).
		Scan(&resolutionTimeResult)
	stats.AvgResolutionMins = resolutionTimeResult.Avg

	// Calculate break time from availability logs
	stats.TotalBreakTimeMins, stats.BreakCount = a.calculateBreakTime(orgID, agentID, start, end)

	// Check if currently on break and get break start time
	if !stats.IsAvailable {
		var currentBreak models.UserAvailabilityLog
		if a.DB.
			Joins("JOIN user_organizations ON user_organizations.user_id = user_availability_logs.user_id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID).
			Where("user_availability_logs.user_id = ? AND user_availability_logs.is_available = false AND user_availability_logs.ended_at IS NULL", agentID).
			Order("started_at DESC").First(&currentBreak).Error == nil {
			breakStart := currentBreak.StartedAt.Format(time.RFC3339)
			stats.CurrentBreakStart = &breakStart
		}
	}

	return stats
}

func (a *App) calculateAllAgentStats(orgID uuid.UUID, start, end time.Time, instanceID *uuid.UUID) []AgentPerformanceStats {
	// Get all agents in the organization through team membership
	var agents []models.User
	if err := a.DB.
		Joins("JOIN team_members ON team_members.user_id = users.id").
		Joins("JOIN teams ON teams.id = team_members.team_id").
		Where("users.organization_id = ? AND team_members.role = ?", orgID, models.TeamRoleAgent).
		Distinct().
		Find(&agents).Error; err != nil {
		a.Log.Error("Failed to fetch agents for analytics", "error", err, "org_id", orgID)
		return []AgentPerformanceStats{}
	}

	stats := make([]AgentPerformanceStats, 0, len(agents))
	for _, agent := range agents {
		agentStats := a.calculateAgentStats(orgID, agent.ID, start, end, instanceID)
		stats = append(stats, agentStats)
	}

	return stats
}

// calculateBreakTime calculates total break time and count for an agent within a time period
func (a *App) calculateBreakTime(orgID, agentID uuid.UUID, start, end time.Time) (totalMins float64, count int64) {
	// Get all "away" periods that overlap with the time range
	var logs []models.UserAvailabilityLog
	if err := a.DB.
		Joins("JOIN user_organizations ON user_organizations.user_id = user_availability_logs.user_id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID).
		Where("user_availability_logs.user_id = ? AND user_availability_logs.is_available = false AND user_availability_logs.started_at <= ? AND (user_availability_logs.ended_at >= ? OR user_availability_logs.ended_at IS NULL)",
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

func (a *App) analyticsAgentBelongsToOrg(orgID, agentID uuid.UUID) (bool, error) {
	var count int64
	err := a.DB.Model(&models.User{}).
		Joins("JOIN user_organizations ON user_organizations.user_id = users.id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID).
		Where("users.id = ? AND users.deleted_at IS NULL", agentID).
		Limit(1).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (a *App) calculateTrendData(orgID uuid.UUID, start, end time.Time, groupBy string, agentID *uuid.UUID, instanceID *uuid.UUID) []TrendPoint {
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
	}

	query := a.DB.Model(&models.AgentTransfer{}).
		Select("DATE_TRUNC('"+dateTrunc+"', transferred_at) as date, COUNT(*) as count").
		Where("organization_id = ? AND status = ? AND transferred_at >= ? AND transferred_at <= ?",
			orgID, models.TransferStatusResumed, start, end)
	query = applyTransferAnalyticsInstanceFilter(query, orgID, instanceID)

	if agentID != nil {
		query = query.Where("agent_id = ?", *agentID)
	}

	var results []TrendResult
	query.Group("DATE_TRUNC('" + dateTrunc + "', transferred_at)").
		Order("date ASC").
		Scan(&results)

	trendData := make([]TrendPoint, len(results))
	for i, r := range results {
		trendData[i] = TrendPoint{
			Date:             r.Date.Format(dateFormat),
			TransfersHandled: r.Count,
		}
	}

	return trendData
}
