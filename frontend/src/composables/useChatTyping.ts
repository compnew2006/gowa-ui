import { ref, computed, onUnmounted } from 'vue'
import { messagesService } from '@/services/api'

export interface UseChatTypingOptions {
  /** Contacts store reactive surface. */
  contactsStore: {
    currentContact: { id: string } | null
    messages: Array<{ id: string; reactions?: Array<{ from_user?: string; emoji?: string }> }>
    updateMessageReactions: (...args: any[]) => void
  }
  /** Current user id — used to detect "I already reacted" for toggle-off. */
  getCurrentUserId: () => string | undefined
}

/**
 * Typing-indicator debounce, emoji reactions, and the emoji picker for the chat
 * composer. Owns the typing stop-timer and cleans it up on unmount.
 *
 * @example
 * ```ts
 * const typing = useChatTyping({ contactsStore, getCurrentUserId: () => authStore.user?.id })
 * ```
 */
export function useChatTyping(options: UseChatTypingOptions) {
  const { contactsStore, getCurrentUserId } = options

  // All accounts are GOWA providers; Meta-only restrictions (24h service
  // window) no longer apply. Kept as a computed for template bindings.
  const isCurrentAccountGowa = computed(() => true)

  // Reaction handling
  const reactionPickerMessageId = ref<string | null>(null)
  const quickReactionEmojis = ['👍', '❤️', '😂', '😮', '😢', '🙏']

  // Emoji picker state
  const emojiPickerOpen = ref(false)

  // Typing debounce state. On the first keystroke after idle we send "start";
  // after TYPING_STOP_DELAY ms of no input (or on send/blur) we send "stop".
  const TYPING_STOP_DELAY = 2000
  let typingStopTimer: ReturnType<typeof setTimeout> | null = null
  let typingIsActive = false

  function stopTypingIndicator() {
    if (typingStopTimer) {
      clearTimeout(typingStopTimer)
      typingStopTimer = null
    }
    if (!typingIsActive) return
    typingIsActive = false
    const contactId = contactsStore.currentContact?.id
    if (!contactId) return
    messagesService.sendTyping(contactId, 'stop').catch(() => {
      // Typing is best-effort; never surface an error toast (it would spam
      // the agent on every idle transition).
    })
  }

  function onTypingInput() {
    if (!isCurrentAccountGowa.value) return
    const contactId = contactsStore.currentContact?.id
    if (!contactId) return
    if (!typingIsActive) {
      typingIsActive = true
      messagesService.sendTyping(contactId, 'start').catch(() => { /* best-effort */ })
    }
    if (typingStopTimer) clearTimeout(typingStopTimer)
    typingStopTimer = setTimeout(stopTypingIndicator, TYPING_STOP_DELAY)
  }

  async function sendReaction(messageId: string, emoji: string) {
    if (!contactsStore.currentContact) return

    // Toggle-off: if the current user already reacted with this same emoji,
    // send an empty emoji to remove it (backend treats "" as "remove my reaction").
    const userId = getCurrentUserId()
    const message = contactsStore.messages.find(m => m.id === messageId)
    const myExisting = message?.reactions?.find(r => r.from_user === userId)
    const emojiToSend = myExisting && myExisting.emoji === emoji ? '' : emoji

    try {
      const response = await messagesService.sendReaction(
        contactsStore.currentContact.id,
        messageId,
        emojiToSend
      )
      // Update will come via WebSocket, but we can update locally for immediate feedback
      const data = (response.data as any).data || response.data
      contactsStore.updateMessageReactions(messageId, data.reactions)
    } catch {
      // Reactions are best-effort UX; ignore failures silently.
    }
    reactionPickerMessageId.value = null
  }

  function insertEmoji(emoji: string, messageInput: { value: string }) {
    messageInput.value += emoji
    emojiPickerOpen.value = false
  }

  onUnmounted(() => {
    stopTypingIndicator()
    if (typingStopTimer) clearTimeout(typingStopTimer)
  })

  return {
    // State
    isCurrentAccountGowa,
    reactionPickerMessageId,
    quickReactionEmojis,
    emojiPickerOpen,
    // Actions
    onTypingInput,
    stopTypingIndicator,
    sendReaction,
    insertEmoji,
  }
}
