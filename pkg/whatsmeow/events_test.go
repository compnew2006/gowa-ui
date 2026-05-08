package whatsmeow

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestStatusesAtOrAbove_Read(t *testing.T) {
	result := statusesAtOrAbove(models.MessageStatusRead)
	assert.Equal(t, []models.MessageStatus{models.MessageStatusRead}, result)
}

func TestStatusesAtOrAbove_Delivered(t *testing.T) {
	result := statusesAtOrAbove(models.MessageStatusDelivered)
	assert.Equal(t, []models.MessageStatus{models.MessageStatusDelivered, models.MessageStatusRead}, result)
}

func TestStatusesAtOrAbove_Sent(t *testing.T) {
	result := statusesAtOrAbove(models.MessageStatusSent)
	assert.Equal(t, []models.MessageStatus{
		models.MessageStatusSent,
		models.MessageStatusDelivered,
		models.MessageStatusRead,
	}, result)
}

func TestStatusesAtOrAbove_UnknownStatus(t *testing.T) {
	result := statusesAtOrAbove(models.MessageStatus("unknown"))
	assert.Nil(t, result)
}

func TestStatusesAtOrAbove_Pending(t *testing.T) {
	result := statusesAtOrAbove(models.MessageStatusPending)
	assert.Nil(t, result)
}

func TestStatusesAtOrAbove_Failed(t *testing.T) {
	result := statusesAtOrAbove(models.MessageStatusFailed)
	assert.Nil(t, result)
}
