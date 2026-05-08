package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestMediaVisibilityHasVisibleMedia(t *testing.T) {
	t.Run("nil message returns false", func(t *testing.T) {
		assert.False(t, messageHasVisibleMedia(nil))
	})

	t.Run("message with media URL returns true", func(t *testing.T) {
		msg := &models.Message{MediaURL: "https://example.com/media.jpg"}
		assert.True(t, messageHasVisibleMedia(msg))
	})

	t.Run("message with empty media URL returns false", func(t *testing.T) {
		msg := &models.Message{MediaURL: ""}
		assert.False(t, messageHasVisibleMedia(msg))
	})

	t.Run("message with whitespace media URL returns false", func(t *testing.T) {
		msg := &models.Message{MediaURL: "   "}
		assert.False(t, messageHasVisibleMedia(msg))
	})

	t.Run("message with deleted media returns false", func(t *testing.T) {
		now := time.Now().UTC()
		msg := &models.Message{MediaURL: "https://example.com/media.jpg", MediaDeletedAt: &now}
		assert.False(t, messageHasVisibleMedia(msg))
	})
}
