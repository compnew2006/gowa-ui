package handlers

import (
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
)

func messageHasVisibleMedia(message *models.Message) bool {
	return message != nil && message.MediaDeletedAt == nil && strings.TrimSpace(message.MediaURL) != ""
}
