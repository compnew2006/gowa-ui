package whatsmeow

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAvatarMetadataForPersistence_SetsAvatarURLAndSyncTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)
	metadata := models.JSONB{
		"existing": "value",
	}

	updated := buildAvatarMetadataForPersistence(metadata, "  https://pps.whatsapp.net/avatar.jpg  ", now)

	require.NotNil(t, updated)
	assert.Equal(t, "value", updated["existing"])
	assert.Equal(t, "https://pps.whatsapp.net/avatar.jpg", updated[avatarURLMetadataKey])
	assert.Equal(t, now.Format(time.RFC3339), updated[avatarSyncedAtMetadataKey])
}

func TestBuildAvatarMetadataForPersistence_ClearsStaleAvatarURLOnEmptyFetch(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)
	metadata := models.JSONB{
		avatarURLMetadataKey:      "https://pps.whatsapp.net/stale.jpg",
		avatarSyncedAtMetadataKey: "2026-03-10T08:00:00Z",
		"existing":                "value",
	}

	updated := buildAvatarMetadataForPersistence(metadata, "   ", now)

	require.NotNil(t, updated)
	_, hasAvatarURL := updated[avatarURLMetadataKey]
	assert.False(t, hasAvatarURL, "stale avatar_url should be removed when refresh yields empty URL")
	assert.Equal(t, now.Format(time.RFC3339), updated[avatarSyncedAtMetadataKey])
	assert.Equal(t, "value", updated["existing"])
}
