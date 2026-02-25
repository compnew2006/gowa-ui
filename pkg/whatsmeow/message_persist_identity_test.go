package whatsmeow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/types"
)

func TestNormalizeDirectSenderIdentity_PrefersPhoneJID(t *testing.T) {
	chat := types.NewJID("15550001234", types.DefaultUserServer)
	sender := types.NewJID("269638281724102", types.HiddenUserServer)

	normalized := normalizeDirectSenderIdentity("269638281724102", chat, sender)
	assert.Equal(t, "15550001234", normalized)
}

func TestNormalizeDirectSenderIdentity_ConvertsHiddenNumericIDToJID(t *testing.T) {
	chat := types.NewJID("269638281724102", types.HiddenUserServer)
	sender := types.NewJID("269638281724102", types.HiddenUserServer)

	normalized := normalizeDirectSenderIdentity("269638281724102", chat, sender)
	assert.Equal(t, "269638281724102@"+string(types.HiddenUserServer), normalized)
}

func TestShouldMigrateLIDContact(t *testing.T) {
	assert.True(t, shouldMigrateLIDContact("15550001234"))
	assert.False(t, shouldMigrateLIDContact("269638281724102@"+string(types.HiddenUserServer)))
	assert.False(t, shouldMigrateLIDContact(""))
}

func TestInferPhoneFromWAMID(t *testing.T) {
	wamid := "wamid.HBgMNTU1NTEyMzQ1NjcVAgASGBQzRUE5QTY4N0Q4Q0Y2Q0E3QjQ2AA=="
	assert.Equal(t, "55551234567", inferPhoneFromWAMID(wamid))
	assert.Empty(t, inferPhoneFromWAMID("wamid.invalid"))
	assert.Empty(t, inferPhoneFromWAMID(""))
}

func TestDirectConversationID(t *testing.T) {
	chat := types.NewJID("269638281724102", types.HiddenUserServer)
	assert.Equal(t, "201007181781@s.whatsapp.net", directConversationID(chat, "201007181781"))
	assert.Equal(t, "269638281724102@"+string(types.HiddenUserServer), directConversationID(chat, "269638281724102@"+string(types.HiddenUserServer)))
	assert.Equal(t, chat.String(), directConversationID(chat, ""))
}

func TestFallbackContactProfileName(t *testing.T) {
	assert.Equal(t, "", fallbackContactProfileName("269638281724102@"+string(types.HiddenUserServer)))
	assert.Equal(t, "15550001234", fallbackContactProfileName("15550001234@"+string(types.DefaultUserServer)))
	assert.Equal(t, "15550001234", fallbackContactProfileName("15550001234"))
}

func TestIsPlaceholderProfileName(t *testing.T) {
	assert.True(t, isPlaceholderProfileName("", "15550001234"))
	assert.True(t, isPlaceholderProfileName("15550001234", "15550001234"))
	assert.True(t, isPlaceholderProfileName("269638281724102@"+string(types.HiddenUserServer), "15550001234"))
	assert.False(t, isPlaceholderProfileName("Customer Name", "15550001234"))
}
