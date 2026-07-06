package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	waTypes "go.mau.fi/whatsmeow/types"
)

func TestParseTypingPresenceState(t *testing.T) {
	state, err := parseTypingPresenceState("")
	require.NoError(t, err)
	assert.Equal(t, waTypes.ChatPresenceComposing, state)

	state, err = parseTypingPresenceState("composing")
	require.NoError(t, err)
	assert.Equal(t, waTypes.ChatPresenceComposing, state)

	state, err = parseTypingPresenceState("paused")
	require.NoError(t, err)
	assert.Equal(t, waTypes.ChatPresencePaused, state)

	_, err = parseTypingPresenceState("invalid")
	require.Error(t, err)
}

func TestIsChannelOrGroupContact(t *testing.T) {
	assert.True(t, isChannelOrGroupContact(models.Contact{PhoneNumber: "120363@g.us"}))
	assert.True(t, isChannelOrGroupContact(models.Contact{PhoneNumber: "12345@newsletter"}))
	assert.True(t, isChannelOrGroupContact(models.Contact{Metadata: models.JSONB{"is_group_chat": true}}))
	assert.True(t, isChannelOrGroupContact(models.Contact{Metadata: models.JSONB{"is_channel_chat": true}}))
	assert.False(t, isChannelOrGroupContact(models.Contact{PhoneNumber: "+201001234567"}))
}
