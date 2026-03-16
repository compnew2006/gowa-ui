package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

func TestMapStatusTypeToMessageType(t *testing.T) {
	tests := []struct {
		name       string
		statusType models.WhatsAppStatusType
		want       models.MessageType
	}{
		{
			name:       "image status",
			statusType: models.WhatsAppStatusTypeImage,
			want:       models.MessageTypeImage,
		},
		{
			name:       "video status",
			statusType: models.WhatsAppStatusTypeVideo,
			want:       models.MessageTypeVideo,
		},
		{
			name:       "text status (default)",
			statusType: models.WhatsAppStatusTypeText,
			want:       models.MessageTypeText,
		},
		{
			name:       "unknown status type",
			statusType: models.WhatsAppStatusType("unknown"),
			want:       models.MessageTypeText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapStatusTypeToMessageType(tt.statusType); got != tt.want {
				t.Errorf("mapStatusTypeToMessageType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindStatusByWAMIDNilDB(t *testing.T) {
	result := findStatusByWAMID(nil, uuid.UUID{}, nil, "test_wamid")
	if result != nil {
		t.Error("Expected nil for nil DB, got non-nil result")
	}
}

func TestFindStatusByWAMIDEmptyWAMID(t *testing.T) {
	// Note: This test requires a mock DB, for now we test the nil/empty cases
	result := findStatusByWAMID(nil, uuid.UUID{}, nil, "")
	if result != nil {
		t.Error("Expected nil for empty WAMID, got non-nil result")
	}
}

func TestBuildReplyPreviewFromNilMetadata(t *testing.T) {
	result := buildReplyPreviewFromMetadata(nil, uuid.UUID{}, nil, nil)
	if result != nil {
		t.Error("Expected nil for nil metadata, got non-nil result")
	}
}

func TestBuildReplyPreviewFromEmptyMetadata(t *testing.T) {
	metadata := make(models.JSONB)
	result := buildReplyPreviewFromMetadata(nil, uuid.UUID{}, nil, metadata)
	if result != nil {
		t.Error("Expected nil for empty metadata, got non-nil result")
	}
}
