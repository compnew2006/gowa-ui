import { ref, nextTick } from 'vue'
import type { Ref } from 'vue'
import { toast } from 'vue-sonner'
import { statusService, messagesService } from '@/services/api'
import { getErrorMessage } from '@/lib/api-utils'
import { STATUS_CONTACT_ID, isStatusContact } from '@/lib/status'
import type { Message } from '@/stores/contacts'

export interface UseChatMessagingOptions {
  /** i18n translator. */
  t: (key: string, params?: Record<string, unknown>) => string
  /** Contacts store reactive surface. */
  contactsStore: {
    currentContact: { id: string } | null
    replyingTo: { id: string } | null
    setReplyingTo: (message: Message | null) => void
    clearReplyingTo: () => void
    // Store sendMessage is variadic/loosely typed; accept the call shape we use.
    sendMessage: (...args: any[]) => Promise<void>
    addStatusMessage: (m: Message) => void
    updateMessageStatus: (id: string, status: string) => void
    messages: Message[]
  }
  /** Selected account ref (multi-account). */
  selectedAccount: { value: string | null }
  /** True when the current provider is GOWA (gates revoke). */
  isCurrentAccountGowa: { value: boolean }
  /** Stop the typing indicator (e.g. right after sending). */
  stopTypingIndicator: () => void
  /** Called after a message lands so the room scrolls to the new bubble. */
  scrollToBottom: (instant?: boolean) => void
  /** Media helper: maps a mime type to 'image'|'video'|'audio'|'document'. */
  getMediaType: (mimeType: string) => string
  /** Close the media upload dialog after a status media send. */
  closeMediaDialog: () => void
  /** Read the shared isUploadingMedia flag (toggled by the status-media path). */
  getIsUploadingMedia: () => boolean
  /** Write the shared isUploadingMedia flag (toggled by the status-media path). */
  setIsUploadingMedia: (v: boolean) => void
  /** Composer textarea template ref. Owned by the view (template ref). */
  messageInputRef: Ref<HTMLTextAreaElement | null>
}

/**
 * Send / retry / revoke / reply text messaging for the chat composer, plus the
 * WhatsApp Status send paths (text + media). Owns the composer input refs and
 * the retry/revoke busy flags.
 *
 * @example
 * ```ts
 * const msg = useChatMessaging({ contactsStore, selectedAccount, ... })
 * ```
 */
export function useChatMessaging(options: UseChatMessagingOptions) {
  const { t, contactsStore, selectedAccount } = options

  const messageInput = ref('')
  // messageInputRef is owned by the view, passed in (template ref).
  const messageInputRef = options.messageInputRef
  const isSending = ref(false)
  const retryingMessageId = ref<string | null>(null)
  const revokingMessageId = ref<string | null>(null)

  function autoResizeTextarea() {
    const textarea = messageInputRef.value
    if (!textarea) return
    textarea.style.height = 'auto'
    textarea.style.height = Math.min(textarea.scrollHeight, 120) + 'px'
  }

  function resetTextareaHeight() {
    const textarea = messageInputRef.value
    if (!textarea) return
    textarea.style.height = 'auto'
  }

  async function sendMessage() {
    if (!messageInput.value.trim() || !contactsStore.currentContact) return

    // Status conversation posts to status@broadcast via a dedicated path.
    if (isStatusContact(contactsStore.currentContact.id)) {
      await sendStatusText(messageInput.value)
      return
    }

    isSending.value = true
    try {
      await contactsStore.sendMessage(
        contactsStore.currentContact.id,
        'text',
        { body: messageInput.value },
        contactsStore.replyingTo?.id,
        selectedAccount.value || undefined
      )
      messageInput.value = ''
      contactsStore.clearReplyingTo()
      resetTextareaHeight()
      // Sending ends any active typing session so the recipient's "typing…"
      // indicator clears as soon as the message lands.
      options.stopTypingIndicator()
      await nextTick()
      options.scrollToBottom()
    } catch (error) {
      toast.error(getErrorMessage(error, t('chat.sendMessageFailed')))
    } finally {
      isSending.value = false
    }
  }

  // sendStatusText posts a text WhatsApp Status (story) from the selected account
  // and appends it to the session-local log shown in the Status conversation.
  async function sendStatusText(text: string) {
    if (!text.trim() || !selectedAccount.value) return
    isSending.value = true
    try {
      const res = await statusService.sendText({ message: text, whatsapp_account: selectedAccount.value })
      const now = new Date().toISOString()
      contactsStore.addStatusMessage({
        id: res.data?.message_id || crypto.randomUUID(),
        contact_id: STATUS_CONTACT_ID,
        direction: 'outgoing',
        message_type: 'text',
        content: { body: text },
        status: 'sent',
        wamid: res.data?.message_id,
        whatsapp_account: selectedAccount.value,
        created_at: now,
        updated_at: now,
      } as Message)
      messageInput.value = ''
      await nextTick()
      options.scrollToBottom()
      toast.success(t('chat.statusSent'))
    } catch (error) {
      toast.error(getErrorMessage(error, t('chat.statusSendFailed')))
    } finally {
      isSending.value = false
    }
  }

  // sendStatusMedia posts an image/video WhatsApp Status and appends it to the
  // session-local log. Only image/video are supported for status (per scope).
  async function sendStatusMedia(file: File, caption: string) {
    if (!selectedAccount.value) return
    const type = options.getMediaType(file.type) as 'image' | 'video'
    if (type !== 'image' && type !== 'video') {
      toast.error(t('chat.statusUnsupportedMedia'))
      return
    }
    options.setIsUploadingMedia(true)
    try {
      const res = await statusService.sendMedia({ file, type, caption, whatsapp_account: selectedAccount.value })
      const now = new Date().toISOString()
      const objectURL = URL.createObjectURL(file)
      contactsStore.addStatusMessage({
        id: res.data?.message_id || crypto.randomUUID(),
        contact_id: STATUS_CONTACT_ID,
        direction: 'outgoing',
        message_type: type,
        content: caption ? { body: caption } : {},
        media_url: objectURL,
        media_mime_type: file.type,
        media_filename: file.name,
        status: 'sent',
        wamid: res.data?.message_id,
        whatsapp_account: selectedAccount.value,
        created_at: now,
        updated_at: now,
      } as Message)
      options.closeMediaDialog()
      await nextTick()
      options.scrollToBottom()
      toast.success(t('chat.statusSent'))
    } catch (error) {
      toast.error(getErrorMessage(error, t('chat.statusSendFailed')))
    } finally {
      options.setIsUploadingMedia(false)
    }
  }

  async function retryMessage(message: Message) {
    if (!contactsStore.currentContact || retryingMessageId.value) return

    retryingMessageId.value = message.id
    try {
      // Get the message content based on type
      const content = message.content || {}

      await contactsStore.sendMessage(
        contactsStore.currentContact.id,
        message.message_type,
        content,
        undefined,
        message.whatsapp_account || selectedAccount.value || undefined
      )

      // Remove the failed message from the list after successful retry
      const messages = (contactsStore.messages as any).get?.(contactsStore.currentContact.id) as Message[] | undefined
      if (messages) {
        const index = messages.findIndex((m: Message) => m.id === message.id)
        if (index !== -1) {
          messages.splice(index, 1)
        }
      }

      toast.success(t('chat.messageSent'))
    } catch (error) {
      toast.error(getErrorMessage(error, t('chat.sendMessageFailed')))
    } finally {
      retryingMessageId.value = null
    }
  }

  // Revoke (delete-for-everyone). GOWA-only; the backend re-validates and 400s
  // for non-GOWA. The optimistic local status is reconciled by the status_update
  // WS broadcast the handler emits on success.
  async function revokeMessage(message: Message) {
    if (!contactsStore.currentContact || revokingMessageId.value) return
    // Destructive, irreversible action — confirm before hitting GOWA.
    if (!window.confirm(t('chat.revokeConfirm'))) return

    revokingMessageId.value = message.id
    try {
      await messagesService.revokeMessage(contactsStore.currentContact.id, message.id)
      // Optimistically swap the bubble for the revoked placeholder. The backend
      // also broadcasts a status_update with status "revoked", which the WS
      // handler routes through updateMessageStatus — so this just stays ahead.
      contactsStore.updateMessageStatus(message.id, 'revoked')
      toast.success(t('chat.messageRevoked'))
    } catch (error) {
      toast.error(getErrorMessage(error, t('chat.revokeFailed')))
    } finally {
      revokingMessageId.value = null
    }
  }

  function replyToMessage(message: Message) {
    contactsStore.setReplyingTo(message)
    nextTick(() => {
      messageInputRef.value?.focus()
    })
  }

  return {
    // Composer state (messageInputRef owned by the view — not re-returned)
    messageInput,
    isSending,
    retryingMessageId,
    revokingMessageId,
    // Actions
    sendMessage,
    sendStatusText,
    sendStatusMedia,
    retryMessage,
    revokeMessage,
    replyToMessage,
    autoResizeTextarea,
    resetTextareaHeight,
  }
}
