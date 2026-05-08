package campaignstats

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestCampaignStatsRecipientStatusesAtOrAbove(t *testing.T) {
	t.Run("delivered includes delivered and read", func(t *testing.T) {
		result := recipientStatusesAtOrAbove(models.MessageStatusDelivered)
		assert.Contains(t, result, models.MessageStatusDelivered)
		assert.Contains(t, result, models.MessageStatusRead)
		assert.Len(t, result, 2)
	})

	t.Run("read includes only read", func(t *testing.T) {
		result := recipientStatusesAtOrAbove(models.MessageStatusRead)
		assert.Contains(t, result, models.MessageStatusRead)
		assert.Len(t, result, 1)
	})

	t.Run("failed includes only failed", func(t *testing.T) {
		result := recipientStatusesAtOrAbove(models.MessageStatusFailed)
		assert.Contains(t, result, models.MessageStatusFailed)
		assert.Len(t, result, 1)
	})

	t.Run("unknown status returns nil", func(t *testing.T) {
		result := recipientStatusesAtOrAbove(models.MessageStatusPending)
		assert.Nil(t, result)
	})
}

func TestCampaignStatsCounterColumn(t *testing.T) {
	t.Run("delivered returns delivered_count", func(t *testing.T) {
		assert.Equal(t, "delivered_count", counterColumn(models.MessageStatusDelivered))
	})

	t.Run("read returns read_count", func(t *testing.T) {
		assert.Equal(t, "read_count", counterColumn(models.MessageStatusRead))
	})

	t.Run("failed returns failed_count", func(t *testing.T) {
		assert.Equal(t, "failed_count", counterColumn(models.MessageStatusFailed))
	})

	t.Run("unknown status returns empty string", func(t *testing.T) {
		assert.Equal(t, "", counterColumn(models.MessageStatusPending))
		assert.Equal(t, "", counterColumn(models.MessageStatus("unknown")))
	})
}
