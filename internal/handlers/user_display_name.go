package handlers

import (
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

func NormalizeDisplayText(value string, limit int) string {
	cleaned := strings.TrimSpace(strings.Join(strings.Fields(value), " "))
	if cleaned == "" {
		return ""
	}
	if limit > 0 && len(cleaned) > limit {
		return cleaned[:limit] + "..."
	}
	return cleaned
}

func (a *App) ResolveUserDisplayName(userID uuid.UUID) string {
	if a == nil || a.DB == nil || userID == uuid.Nil {
		return ""
	}

	var user struct {
		FullName string `gorm:"column:full_name"`
		Email    string `gorm:"column:email"`
	}
	if err := a.DB.Model(&models.User{}).
		Select("full_name", "email").
		Where("id = ?", userID).
		Take(&user).Error; err != nil {
		return ""
	}

	if name := NormalizeDisplayText(user.FullName, 80); name != "" {
		return name
	}
	return NormalizeDisplayText(user.Email, 120)
}
