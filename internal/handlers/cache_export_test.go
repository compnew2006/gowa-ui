package handlers

import (
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// Exported functions for testing cache.go
// These functions expose private cache functions for testing purposes

// Exported functions for testing cache.go
// These functions expose private cache functions for testing purposes

// GetChatbotSettingsCached exports getChatbotSettingsCached for testing
func (a *App) GetChatbotSettingsCached(orgID uuid.UUID, whatsAppAccount string) (*models.ChatbotSettings, error) {
	return a.getChatbotSettingsCached(orgID, whatsAppAccount)
}

// GetChatbotFlowsCached exports getChatbotFlowsCached for testing
func (a *App) GetChatbotFlowsCached(orgID uuid.UUID) ([]models.ChatbotFlow, error) {
	return a.getChatbotFlowsCached(orgID)
}

// GetChatbotFlowByIDCached exports getChatbotFlowByIDCached for testing
func (a *App) GetChatbotFlowByIDCached(orgID uuid.UUID, flowID uuid.UUID) (*models.ChatbotFlow, error) {
	return a.getChatbotFlowByIDCached(orgID, flowID)
}

// GetKeywordRulesCached exports getKeywordRulesCached for testing
func (a *App) GetKeywordRulesCached(orgID uuid.UUID, whatsAppAccount string) ([]models.KeywordRule, error) {
	return a.getKeywordRulesCached(orgID, whatsAppAccount)
}

// GetWhatsAppAccountCached exports getWhatsAppAccountCached for testing
func (a *App) GetWhatsAppAccountCached(phoneID string) (*models.WhatsAppAccount, error) {
	return a.getWhatsAppAccountCached(phoneID)
}

// GetWebhooksCached exports getWebhooksCached for testing
func (a *App) GetWebhooksCached(orgID uuid.UUID) ([]models.Webhook, error) {
	return a.getWebhooksCached(orgID)
}

// GetSLAEnabledSettingsCached exports getSLAEnabledSettingsCached for testing
func (a *App) GetSLAEnabledSettingsCached() ([]models.ChatbotSettings, error) {
	return a.getSLAEnabledSettingsCached()
}

// GetAIContextsCached exports getAIContextsCached for testing
func (a *App) GetAIContextsCached(orgID uuid.UUID, whatsAppAccount string) ([]models.AIContext, error) {
	return a.getAIContextsCached(orgID, whatsAppAccount)
}

// GetUserPermissionsCached exports getUserPermissionsCached for testing
func (a *App) GetUserPermissionsCached(userID uuid.UUID, orgIDs ...uuid.UUID) (*UserPermissions, error) {
	return a.getUserPermissionsCached(userID, orgIDs...)
}

// GetTagsCached exports getTagsCached for testing
func (a *App) GetTagsCached(orgID uuid.UUID) ([]models.Tag, error) {
	return a.getTagsCached(orgID)
}

// ParseContextUUID exports parseContextUUID for testing
func ParseContextUUID(value any) (uuid.UUID, bool) {
	return parseContextUUID(value)
}

// ParseAnalyticsInstanceID exports parseAnalyticsInstanceID for testing
func (a *App) ParseAnalyticsInstanceID(orgID uuid.UUID, raw string) (*uuid.UUID, error) {
	return a.parseAnalyticsInstanceID(orgID, raw)
}

// ApplyTransferAnalyticsInstanceFilter exports applyTransferAnalyticsInstanceFilter for testing
func ApplyTransferAnalyticsInstanceFilter(query *gorm.DB, orgID uuid.UUID, instanceID *uuid.UUID) *gorm.DB {
	return applyTransferAnalyticsInstanceFilter(query, orgID, instanceID)
}

// ApplyRatingAnalyticsInstanceFilter exports applyRatingAnalyticsInstanceFilter for testing
func ApplyRatingAnalyticsInstanceFilter(query *gorm.DB, instanceID *uuid.UUID, contactAlias string) *gorm.DB {
	return applyRatingAnalyticsInstanceFilter(query, instanceID, contactAlias)
}

// CalculateBreakTime exports calculateBreakTime for testing
func (a *App) CalculateBreakTime(orgID, agentID uuid.UUID, start, end time.Time) (totalMins float64, count int64) {
	return a.calculateBreakTime(orgID, agentID, start, end)
}

// AnalyticsAgentBelongsToOrg exports analyticsAgentBelongsToOrg for testing
func (a *App) AnalyticsAgentBelongsToOrg(orgID, agentID uuid.UUID) (bool, error) {
	return a.analyticsAgentBelongsToOrg(orgID, agentID)
}

// CalculateTrendData exports calculateTrendData for testing
func (a *App) CalculateTrendData(orgID uuid.UUID, start, end time.Time, groupBy string, agentID *uuid.UUID, instanceID *uuid.UUID) []TrendPoint {
	return a.calculateTrendData(orgID, start, end, groupBy, agentID, instanceID)
}

// CalculateSummaryStats exports calculateSummaryStats for testing
func (a *App) CalculateSummaryStats(orgID uuid.UUID, start, end time.Time, summary *AgentAnalyticsSummary, instanceID *uuid.UUID) {
	a.calculateSummaryStats(orgID, start, end, summary, instanceID)
}

// CalculateAgentSummaryStats exports calculateAgentSummaryStats for testing
func (a *App) CalculateAgentSummaryStats(orgID, agentID uuid.UUID, start, end time.Time, summary *AgentAnalyticsSummary, instanceID *uuid.UUID) {
	a.calculateAgentSummaryStats(orgID, agentID, start, end, summary, instanceID)
}

// CalculateAgentStats exports calculateAgentStats for testing
func (a *App) CalculateAgentStats(orgID, agentID uuid.UUID, start, end time.Time, instanceID *uuid.UUID) AgentPerformanceStats {
	return a.calculateAgentStats(orgID, agentID, start, end, instanceID)
}

// CalculateAllAgentStats exports calculateAllAgentStats for testing
func (a *App) CalculateAllAgentStats(orgID uuid.UUID, start, end time.Time, instanceID *uuid.UUID) []AgentPerformanceStats {
	return a.calculateAllAgentStats(orgID, start, end, instanceID)
}

// GetOrgID exports getOrgID for testing
func (a *App) GetOrgID(r *fastglue.Request) (uuid.UUID, error) {
	return a.getOrgID(r)
}

// GetOrgAndUserID exports getOrgAndUserID for testing
func (a *App) GetOrgAndUserID(r *fastglue.Request) (orgID, userID uuid.UUID, err error) {
	return a.getOrgAndUserID(r)
}

// RequirePermission exports requirePermission for testing
func (a *App) RequirePermission(r *fastglue.Request, userID uuid.UUID, resource, action string) error {
	return a.requirePermission(r, userID, resource, action)
}

// DecodeRequest exports decodeRequest for testing
func (a *App) DecodeRequest(r *fastglue.Request, v interface{}) error {
	return a.decodeRequest(r, v)
}

// GenerateRandomString exports generateRandomString for testing
func GenerateRandomString(n int) (string, error) {
	return generateRandomString(n)
}

// GenerateCSRFToken exports generateCSRFToken for testing
func GenerateCSRFToken() (string, error) {
	return generateCSRFToken()
}

// RefreshTokenKey exports refreshTokenKey for testing
func RefreshTokenKey(jti string) string {
	return refreshTokenKey(jti)
}

// GenerateSlug exports generateSlug for testing
func GenerateSlug(name string) string {
	return generateSlug(name)
}

// GenerateAccessToken exports generateAccessToken for testing
func (a *App) GenerateAccessToken(user *models.User) (string, time.Time, error) {
	return a.generateAccessToken(user)
}

// GenerateRefreshToken exports generateRefreshToken for testing
func (a *App) GenerateRefreshToken(user *models.User) (string, error) {
	return a.generateRefreshToken(user)
}

// GenerateRegisterInviteToken exports generateRegisterInviteToken for testing
func (a *App) GenerateRegisterInviteToken(orgID uuid.UUID, ttl time.Duration) (string, time.Time, error) {
	return a.generateRegisterInviteToken(orgID, ttl)
}

// ValidateRegisterInviteToken exports validateRegisterInviteToken for testing
func (a *App) ValidateRegisterInviteToken(tokenString string) (uuid.UUID, error) {
	return a.validateRegisterInviteToken(tokenString)
}

// Exported functions for testing closed_chat_filters.go

// ParseClosedChatFilters exports parseClosedChatFilters for testing
func ParseClosedChatFilters(r *fastglue.Request) (ClosedChatFilters, string, error) {
	return parseClosedChatFilters(r)
}

// ApplyClosedChatFilters exports applyClosedChatFilters for testing
func ApplyClosedChatFilters(query *gorm.DB, search string, filters ClosedChatFilters) *gorm.DB {
	return applyClosedChatFilters(query, search, filters)
}

// ClosedChatFilters exposes the closedChatFilters struct for testing
type ClosedChatFilters = closedChatFilters

// Exported functions for testing canned_response_media.go

// IsMultipartFormRequest exports isMultipartFormRequest for testing
func IsMultipartFormRequest(r *fastglue.Request) bool {
	return isMultipartFormRequest(r)
}

// FirstFormValue exports firstFormValue for testing
func FirstFormValue(values map[string][]string, key string) string {
	return firstFormValue(values, key)
}

// ParseOptionalBool exports parseOptionalBool for testing
func ParseOptionalBool(raw string) (*bool, error) {
	return parseOptionalBool(raw)
}

// NormalizeCannedAttachmentType exports normalizeCannedAttachmentType for testing
func NormalizeCannedAttachmentType(mimeType string) (string, bool) {
	return normalizeCannedAttachmentType(mimeType)
}

// ResolveMediaFilePath exports resolveMediaFilePath for testing
func (a *App) ResolveMediaFilePath(relativePath string) (string, error) {
	return a.resolveMediaFilePath(relativePath)
}

// ReadCannedAttachmentData exports readCannedAttachmentData for testing
func (a *App) ReadCannedAttachmentData(attachment models.CannedResponseAttachment) ([]byte, error) {
	return a.readCannedAttachmentData(attachment)
}

// CleanupCannedResponseAttachments exports cleanupCannedResponseAttachments for testing
func (a *App) CleanupCannedResponseAttachments(attachments models.CannedResponseAttachments) {
	a.cleanupCannedResponseAttachments(attachments)
}

// Exported functions for testing chat_assignment_reset_settings.go

// DefaultChatAssignmentResetSettings exports defaultChatAssignmentResetSettings for testing
func DefaultChatAssignmentResetSettings() ChatAssignmentResetSettings {
	return defaultChatAssignmentResetSettings()
}

// IsValidChatAssignmentResetMode exports isValidChatAssignmentResetMode for testing
func IsValidChatAssignmentResetMode(raw string) bool {
	return isValidChatAssignmentResetMode(raw)
}

// NormalizeChatAssignmentResetMode exports normalizeChatAssignmentResetMode for testing
func NormalizeChatAssignmentResetMode(raw string) ChatAssignmentResetMode {
	return normalizeChatAssignmentResetMode(raw)
}

// IsValidChatAssignmentResetHour exports isValidChatAssignmentResetHour for testing
func IsValidChatAssignmentResetHour(hour int) bool {
	return isValidChatAssignmentResetHour(hour)
}

// ParseChatAssignmentResetHour exports parseChatAssignmentResetHour for testing
func ParseChatAssignmentResetHour(raw any) (int, bool) {
	return parseChatAssignmentResetHour(raw)
}

// ParseJSONBBool exports parseJSONBBool for testing
func ParseJSONBBool(raw any) (bool, bool) {
	return parseJSONBBool(raw)
}

// ParseOrganizationTimezone exports parseOrganizationTimezone for testing
func ParseOrganizationTimezone(settings models.JSONB) string {
	return parseOrganizationTimezone(settings)
}

// ValidateChatAssignmentResetInputs exports validateChatAssignmentResetInputs for testing
func ValidateChatAssignmentResetInputs(mode *string, hour *int) error {
	return validateChatAssignmentResetInputs(mode, hour)
}

// ReadChatAssignmentResetSettings exports readChatAssignmentResetSettings for testing
func ReadChatAssignmentResetSettings(settings models.JSONB) ChatAssignmentResetSettings {
	return readChatAssignmentResetSettings(settings)
}
