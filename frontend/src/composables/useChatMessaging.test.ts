import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive, ref } from 'vue'

vi.mock('@/services/api', () => ({
  statusService: { sendText: vi.fn(), sendMedia: vi.fn() },
  messagesService: { revokeMessage: vi.fn() },
}))
vi.mock('vue-sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn() }
}))

import { toast } from 'vue-sonner'
import { useChatMessaging } from './useChatMessaging'

function setup(opts: { locked?: boolean; sendError?: unknown } = {}) {
  const contactsStore = reactive({
    currentContact: { id: 'contact-1' },
    replyingTo: null,
    setReplyingTo: vi.fn(),
    clearReplyingTo: vi.fn(),
    sendMessage: opts.sendError ? vi.fn().mockRejectedValue(opts.sendError) : vi.fn().mockResolvedValue(undefined),
    addStatusMessage: vi.fn(),
    updateMessageStatus: vi.fn(),
    messages: [],
  })
  const onSendAccountDenied = vi.fn()
  const messaging = useChatMessaging({
    t: (key: string) => key,
    contactsStore: contactsStore as any,
    selectedAccount: ref<string | null>('acct-b'),
    isCurrentAccountGowa: ref(true),
    stopTypingIndicator: vi.fn(),
    scrollToBottom: vi.fn(),
    getMediaType: () => 'image',
    closeMediaDialog: vi.fn(),
    getIsUploadingMedia: () => false,
    setIsUploadingMedia: vi.fn(),
    messageInputRef: ref(null),
    isSendAccountLocked: () => !!opts.locked,
    onSendAccountDenied,
  })
  messaging.messageInput.value = 'hello'
  return { contactsStore, messaging, onSendAccountDenied }
}

describe('useChatMessaging send-account policy', () => {
  beforeEach(() => vi.clearAllMocks())

  it('does not send from an account the server refused', async () => {
    const { contactsStore, messaging } = setup({ locked: true })
    await messaging.sendMessage()
    expect(contactsStore.sendMessage).not.toHaveBeenCalled()
    expect(toast.error).toHaveBeenCalledWith('chat.sendAccountNotAllowed')
    expect(messaging.messageInput.value).toBe('hello')
  })

  it('sends with the selected account when permitted', async () => {
    const { contactsStore, messaging } = setup()
    await messaging.sendMessage()
    expect(contactsStore.sendMessage).toHaveBeenCalledWith('contact-1', 'text', { body: 'hello' }, undefined, 'acct-b')
  })

  it('shows account_not_allowed and reloads the account decisions', async () => {
    const { messaging, onSendAccountDenied } = setup({
      sendError: { isAxiosError: true, response: { status: 403, data: { error_type: 'account_not_allowed', message: 'x' } } },
    })
    await messaging.sendMessage()
    expect(toast.error).toHaveBeenCalledWith('chat.sendAccountNotAllowed')
    expect(onSendAccountDenied).toHaveBeenCalledOnce()
  })

  it('keeps the generic error for other failures', async () => {
    const { messaging, onSendAccountDenied } = setup({ sendError: new Error('boom') })
    await messaging.sendMessage()
    expect(toast.error).toHaveBeenCalledWith('boom')
    expect(onSendAccountDenied).not.toHaveBeenCalled()
  })
})
