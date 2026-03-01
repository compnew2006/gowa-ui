package whatsmeow

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTypingPresenceRecipient_NormalizesPhone(t *testing.T) {
	jid, err := parseTypingPresenceRecipient("+20 (101) 234-5678")
	require.NoError(t, err)
	assert.Equal(t, "201012345678@s.whatsapp.net", jid.String())
}

func TestParseTypingPresenceRecipient_ParsesExplicitJID(t *testing.T) {
	jid, err := parseTypingPresenceRecipient("15551234567@lid")
	require.NoError(t, err)
	assert.Equal(t, "15551234567@lid", jid.String())
}

func TestParseTypingPresenceRecipient_RejectsInvalidRecipient(t *testing.T) {
	_, err := parseTypingPresenceRecipient("   ")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTypingPresenceInvalidRecipient))
}

func TestNormalizeTypingPresencePhone(t *testing.T) {
	assert.Equal(t, "201012345678", normalizeTypingPresencePhone("+20 101-234-5678"))
	assert.Equal(t, "", normalizeTypingPresencePhone("(abc)"))
}
