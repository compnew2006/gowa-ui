package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestResolveStatusMediaURL(t *testing.T) {
	statusID := uuid.New()

	t.Run("maps local relative path to api endpoint", func(t *testing.T) {
		status := models.WhatsAppStatus{
			BaseModel: models.BaseModel{ID: statusID},
			MediaURL:  "videos/sample.mp4",
		}
		assert.Equal(t, "/api/statuses/"+statusID.String()+"/media", resolveStatusMediaURL(status))
	})

	t.Run("maps local rooted path to api endpoint", func(t *testing.T) {
		status := models.WhatsAppStatus{
			BaseModel: models.BaseModel{ID: statusID},
			MediaURL:  "/videos/sample.mp4",
		}
		assert.Equal(t, "/api/statuses/"+statusID.String()+"/media", resolveStatusMediaURL(status))
	})

	t.Run("keeps external url", func(t *testing.T) {
		url := "https://cdn.example.com/status.mp4"
		status := models.WhatsAppStatus{
			BaseModel: models.BaseModel{ID: statusID},
			MediaURL:  url,
		}
		assert.Equal(t, url, resolveStatusMediaURL(status))
	})

	t.Run("keeps existing status media api url", func(t *testing.T) {
		url := "/api/statuses/" + statusID.String() + "/media"
		status := models.WhatsAppStatus{
			BaseModel: models.BaseModel{ID: statusID},
			MediaURL:  url,
		}
		assert.Equal(t, url, resolveStatusMediaURL(status))
	})

	t.Run("empty media url", func(t *testing.T) {
		status := models.WhatsAppStatus{
			BaseModel: models.BaseModel{ID: statusID},
		}
		assert.Equal(t, "", resolveStatusMediaURL(status))
	})
}

func TestResolveStatusReplyContactPhone(t *testing.T) {
	t.Run("normalizes default user jid to phone", func(t *testing.T) {
		assert.Equal(t, "15550001111", resolveStatusReplyContactPhone("15550001111@s.whatsapp.net"))
	})

	t.Run("keeps lid jid format", func(t *testing.T) {
		assert.Equal(t, "254884951650472@lid", resolveStatusReplyContactPhone("254884951650472@lid"))
	})

	t.Run("returns raw value when jid parse fails", func(t *testing.T) {
		assert.Equal(t, "invalid jid value", resolveStatusReplyContactPhone("invalid jid value"))
	})
}
