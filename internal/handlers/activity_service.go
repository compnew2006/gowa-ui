package handlers

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/zerodha/fastglue"
)

// ActivityListFilter defines filters for listing activity logs.
type ActivityListFilter struct {
	Pagination
	Category  string
	EventType string
	Source    string
	Status    string
	StartDate *time.Time
	EndDate   *time.Time
}

func (a *App) trustProxyEnabled() bool {
	return a != nil && a.Config != nil && a.Config.RateLimit.TrustProxy
}

func (a *App) insertActivity(entry *models.ActivityLog) error {
	if entry == nil {
		return fmt.Errorf("activity entry is nil")
	}
	if entry.Metadata == nil {
		entry.Metadata = models.JSONB{}
	}
	return a.DB.Create(entry).Error
}

func requestPath(r *fastglue.Request) string {
	if r == nil || r.RequestCtx == nil {
		return ""
	}
	return string(r.RequestCtx.Path())
}

func requestMethod(r *fastglue.Request) string {
	if r == nil || r.RequestCtx == nil {
		return ""
	}
	return string(r.RequestCtx.Method())
}

func requestUserAgent(r *fastglue.Request) string {
	if r == nil || r.RequestCtx == nil {
		return ""
	}
	return string(r.RequestCtx.Request.Header.Peek("User-Agent"))
}

func requestClientIP(r *fastglue.Request, trustProxy bool) string {
	if r == nil || r.RequestCtx == nil {
		return ""
	}

	if trustProxy {
		if xff := string(r.RequestCtx.Request.Header.Peek("X-Forwarded-For")); xff != "" {
			parts := strings.SplitN(xff, ",", 2)
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
		if realIP := string(r.RequestCtx.Request.Header.Peek("X-Real-IP")); realIP != "" {
			return strings.TrimSpace(realIP)
		}
	}

	addr := r.RequestCtx.RemoteAddr().String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func normalizeActivityText(value string, limit int) string {
	cleaned := strings.TrimSpace(strings.Join(strings.Fields(value), " "))
	if cleaned == "" {
		return ""
	}
	if limit > 0 && len(cleaned) > limit {
		return cleaned[:limit] + "..."
	}
	return cleaned
}

func (a *App) resolveActivityActorName(userID uuid.UUID) string {
	if userID == uuid.Nil {
		return ""
	}

	var actor struct {
		FullName string `gorm:"column:full_name"`
		Email    string `gorm:"column:email"`
	}
	if err := a.DB.Model(&models.User{}).
		Select("full_name", "email").
		Where("id = ?", userID).
		Take(&actor).Error; err != nil {
		return ""
	}

	if name := normalizeActivityText(actor.FullName, 80); name != "" {
		return name
	}
	return normalizeActivityText(actor.Email, 120)
}

// LogAuthSuccess records successful user authentication.
func (a *App) LogAuthSuccess(r *fastglue.Request, user *models.User) {
	if user == nil {
		return
	}

	orgID := user.OrganizationID
	userID := user.ID
	entry := &models.ActivityLog{
		OrganizationID: &orgID,
		UserID:         &userID,
		Category:       "auth",
		EventType:      "auth.login",
		Action:         "login",
		Status:         "success",
		Source:         "auth",
		Method:         requestMethod(r),
		Path:           requestPath(r),
		IPAddress:      requestClientIP(r, a.trustProxyEnabled()),
		UserAgent:      requestUserAgent(r),
		Metadata: models.JSONB{
			"email": user.Email,
		},
	}

	if err := a.insertActivity(entry); err != nil {
		a.Log.Error("Failed to log auth success", "error", err, "user_id", user.ID)
	}
}

// LogAuthFailure records failed authentication attempts.
func (a *App) LogAuthFailure(r *fastglue.Request, email string, userID, orgID *uuid.UUID, reason string) {
	entry := &models.ActivityLog{
		OrganizationID: orgID,
		UserID:         userID,
		Category:       "auth",
		EventType:      "auth.login_failed",
		Action:         "login",
		Status:         "failure",
		Source:         "auth",
		Method:         requestMethod(r),
		Path:           requestPath(r),
		IPAddress:      requestClientIP(r, a.trustProxyEnabled()),
		UserAgent:      requestUserAgent(r),
		Metadata: models.JSONB{
			"email":  email,
			"reason": reason,
		},
	}

	if err := a.insertActivity(entry); err != nil {
		a.Log.Error("Failed to log auth failure", "error", err, "email", email)
	}
}

// LogLogout records successful logout actions.
func (a *App) LogLogout(r *fastglue.Request, userID, orgID *uuid.UUID) {
	entry := &models.ActivityLog{
		OrganizationID: orgID,
		UserID:         userID,
		Category:       "auth",
		EventType:      "auth.logout",
		Action:         "logout",
		Status:         "success",
		Source:         "auth",
		Method:         requestMethod(r),
		Path:           requestPath(r),
		IPAddress:      requestClientIP(r, a.trustProxyEnabled()),
		UserAgent:      requestUserAgent(r),
	}

	if err := a.insertActivity(entry); err != nil {
		a.Log.Error("Failed to log logout event", "error", err)
	}
}

// LogConversationResponse records outbound user responses in a conversation.
func (a *App) LogConversationResponse(
	userID, orgID, contactID, messageID uuid.UUID,
	messageType models.MessageType,
	messageContent, chatName, chatPhone string,
) {
	metadata := models.JSONB{
		"message_type": messageType,
	}
	if content := normalizeActivityText(messageContent, 240); content != "" {
		metadata["message_content"] = content
	}
	if name := normalizeActivityText(chatName, 120); name != "" {
		metadata["chat_name"] = name
	}
	if phone := normalizeActivityText(chatPhone, 80); phone != "" {
		metadata["chat_phone"] = phone
	}
	if actor := a.resolveActivityActorName(userID); actor != "" {
		metadata["actor_name"] = actor
	}

	entry := &models.ActivityLog{
		OrganizationID: &orgID,
		UserID:         &userID,
		Category:       "engagement",
		EventType:      "engagement.conversation_response",
		Action:         "send_message",
		Status:         "success",
		Source:         "engagement",
		ContactID:      &contactID,
		MessageID:      &messageID,
		Metadata:       metadata,
	}

	if err := a.insertActivity(entry); err != nil {
		a.Log.Error("Failed to log conversation response", "error", err, "message_id", messageID)
	}
}

// LogSystemInteraction records authenticated API interactions.
func (a *App) LogSystemInteraction(r *fastglue.Request, userID, orgID uuid.UUID, statusCode int) {
	status := "success"
	if statusCode >= 400 {
		status = "failure"
	}

	entry := &models.ActivityLog{
		OrganizationID: &orgID,
		UserID:         &userID,
		Category:       "system",
		EventType:      "system.api_interaction",
		Action:         "api_request",
		Status:         status,
		Source:         "system",
		Method:         requestMethod(r),
		Path:           requestPath(r),
		IPAddress:      requestClientIP(r, a.trustProxyEnabled()),
		UserAgent:      requestUserAgent(r),
		Metadata: models.JSONB{
			"http_status": statusCode,
		},
	}

	if err := a.insertActivity(entry); err != nil {
		a.Log.Error("Failed to log system interaction", "error", err, "user_id", userID)
	}
}

// LogCustomEvent stores a custom activity event posted by an authenticated user.
func (a *App) LogCustomEvent(
	r *fastglue.Request,
	userID, orgID uuid.UUID,
	category, eventType, action string,
	contactID, messageID *uuid.UUID,
	metadata models.JSONB,
) (*models.ActivityLog, error) {
	entry := &models.ActivityLog{
		OrganizationID: &orgID,
		UserID:         &userID,
		Category:       category,
		EventType:      eventType,
		Action:         action,
		Status:         "success",
		Source:         "custom",
		ContactID:      contactID,
		MessageID:      messageID,
		Method:         requestMethod(r),
		Path:           requestPath(r),
		IPAddress:      requestClientIP(r, a.trustProxyEnabled()),
		UserAgent:      requestUserAgent(r),
		Metadata:       metadata,
	}

	if err := a.insertActivity(entry); err != nil {
		return nil, err
	}

	return entry, nil
}

// ListOwnEvents returns the authenticated user's own events with filters and pagination.
func (a *App) ListOwnEvents(userID, orgID uuid.UUID, filter ActivityListFilter) ([]models.ActivityLog, int64, error) {
	query := a.DB.Model(&models.ActivityLog{}).
		Where("user_id = ?", userID).
		Where("(organization_id = ? OR organization_id IS NULL)", orgID)

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.EventType != "" {
		query = query.Where("event_type = ?", filter.EventType)
	}
	if filter.Source != "" {
		query = query.Where("source = ?", filter.Source)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", *filter.EndDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []models.ActivityLog
	if err := filter.Pagination.Apply(query.Order("created_at DESC")).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// PurgeOlderThan permanently removes activity logs older than cutoff.
func (a *App) PurgeOlderThan(cutoff time.Time) (int64, error) {
	result := a.DB.Unscoped().Where("created_at < ?", cutoff).Delete(&models.ActivityLog{})
	return result.RowsAffected, result.Error
}
