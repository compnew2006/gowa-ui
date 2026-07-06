package whatsmeow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/types"
)

func TestIsStatusMessageInfo(t *testing.T) {
	t.Run("matches status chat jid", func(t *testing.T) {
		info := types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat: types.StatusBroadcastJID,
			},
		}
		assert.True(t, isStatusMessageInfo(info))
	})

	t.Run("matches recipient alt jid", func(t *testing.T) {
		info := types.MessageInfo{
			MessageSource: types.MessageSource{
				RecipientAlt: types.StatusBroadcastJID,
			},
		}
		assert.True(t, isStatusMessageInfo(info))
	})

	t.Run("matches broadcast owner jid", func(t *testing.T) {
		info := types.MessageInfo{
			MessageSource: types.MessageSource{
				BroadcastListOwner: types.StatusBroadcastJID,
			},
		}
		assert.True(t, isStatusMessageInfo(info))
	})

	t.Run("matches device sent destination jid", func(t *testing.T) {
		info := types.MessageInfo{
			DeviceSentMeta: &types.DeviceSentMeta{
				DestinationJID: "status@broadcast",
			},
		}
		assert.True(t, isStatusMessageInfo(info))
	})

	t.Run("matches status category", func(t *testing.T) {
		info := types.MessageInfo{
			Category: "status",
		}
		assert.True(t, isStatusMessageInfo(info))
	})

	t.Run("non status message", func(t *testing.T) {
		chat, err := types.ParseJID("201000000000@s.whatsapp.net")
		assert.NoError(t, err)

		info := types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat: chat,
			},
		}
		assert.False(t, isStatusMessageInfo(info))
	})
}

func TestIsStatusMessageSource(t *testing.T) {
	t.Run("matches status chat jid", func(t *testing.T) {
		source := types.MessageSource{
			Chat: types.StatusBroadcastJID,
		}
		assert.True(t, isStatusMessageSource(source))
	})

	t.Run("matches recipient alt jid", func(t *testing.T) {
		source := types.MessageSource{
			RecipientAlt: types.StatusBroadcastJID,
		}
		assert.True(t, isStatusMessageSource(source))
	})

	t.Run("non status source", func(t *testing.T) {
		chat, err := types.ParseJID("201000000000@s.whatsapp.net")
		assert.NoError(t, err)

		source := types.MessageSource{
			Chat: chat,
		}
		assert.False(t, isStatusMessageSource(source))
	})
}
