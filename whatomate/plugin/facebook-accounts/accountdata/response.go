package accountdata

import (
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

type Response struct {
	ID             uuid.UUID                    `json:"id"`
	Name           string                       `json:"name"`
	AccountUID     string                       `json:"account_uid"`
	Platform       string                       `json:"platform"`
	Email          string                       `json:"email,omitempty"`
	AvatarURL      string                       `json:"avatar_url,omitempty"`
	Status         models.FacebookAccountStatus `json:"status"`
	Method         models.FacebookAccountMethod `json:"method"`
	Data           models.JSONB                 `json:"data"`
	HasCookies     bool                         `json:"has_cookies"`
	OAuthConnected bool                         `json:"oauth_connected"`
	TokenExpiresAt string                       `json:"token_expires_at,omitempty"`
	ConnectedAt    string                       `json:"connected_at,omitempty"`
	LastRenewedAt  string                       `json:"last_renewed_at,omitempty"`
	PageCount      int                          `json:"page_count"`
	CreatedAt      string                       `json:"created_at"`
	UpdatedAt      string                       `json:"updated_at"`
}

func ToResponse(account models.FacebookAccount) Response {
	return Response{
		ID:             account.ID,
		Name:           account.Name,
		AccountUID:     account.AccountUID,
		Platform:       account.Platform,
		Email:          account.Email,
		AvatarURL:      account.AvatarURL,
		Status:         account.Status,
		Method:         account.Method,
		Data:           account.Data,
		HasCookies:     account.CookiesText != "",
		OAuthConnected: account.AccessToken != "",
		TokenExpiresAt: formatOptionalTime(account.TokenExpiresAt),
		ConnectedAt:    formatOptionalTime(account.ConnectedAt),
		LastRenewedAt:  formatOptionalTime(account.LastRenewedAt),
		PageCount:      pageCount(account.Data),
		CreatedAt:      account.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      account.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02T15:04:05Z")
}

func pageCount(data models.JSONB) int {
	if data == nil {
		return 0
	}
	if value, ok := data["page_count"].(int); ok {
		return value
	}
	if value, ok := data["page_count"].(float64); ok {
		return int(value)
	}
	if pages, ok := data["pages"].([]interface{}); ok {
		return len(pages)
	}
	return 0
}
