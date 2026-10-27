import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive, ref } from 'vue'

vi.mock('@/services/api', () => ({
  cannedResponsesService: { use: vi.fn().mockResolvedValue(undefined) },
}))
vi.mock('vue-sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn() }
}))

import { toast } from 'vue-sonner'
import { useChatCannedTemplates } from './useChatCannedTemplates'

const denied = { response: { data: { error_type: 'account_not_allowed', message: 'server text' } } }

function setup(opts: { locked?: boolean; sendError?: unknown } = {}) {
  const send = () => (opts.sendError ? vi.fn().mockRejectedValue(opts.sendError) : vi.fn().mockResolvedValue(undefined))
  const contactsStore = reactive({
    currentContact: { id: 'contact-1' },
    replyingTo: null,
    clearReplyingTo: vi.fn(),
    sendMessage: send(),
    sendTemplate: send(),
  })
  const onSendAccountDenied = vi.fn()
  const composable = useChatCannedTemplates({
    t: (key: string) => key,
    contactsStore: contactsStore as any,
    selectedAccount: ref<string | null>('acct-b'),
    getCurrentContact: () => null,
    getCurrentUser: () => null,
    messageInput: ref(''),
    resetTextareaHeight: vi.fn(),
    scrollToBottom: vi.fn(),
    isSendAccountLocked: () => !!opts.locked,
    onSendAccountDenied,
  })
  composable.selectedCannedResponse.value = { id: 'canned-1', content: 'Hi there' } as any
  composable.selectedTemplate.value = { name: 'welcome' } as any
  return { contactsStore, composable, onSendAccountDenied }
}

describe('useChatCannedTemplates send-account policy', () => {
  beforeEach(() => vi.clearAllMocks())

  it('does not send a canned response from a refused account', async () => {
    const { contactsStore, composable } = setup({ locked: true })
    await composable.sendCannedResponse()
    expect(contactsStore.sendMessage).not.toHaveBeenCalled()
    expect(toast.error).toHaveBeenCalledWith('chat.sendAccountNotAllowed')
  })

  it('does not send a template from a refused account', async () => {
    const { contactsStore, composable } = setup({ locked: true })
    await composable.sendTemplateMessage()
    expect(contactsStore.sendTemplate).not.toHaveBeenCalled()
    expect(toast.error).toHaveBeenCalledWith('chat.sendAccountNotAllowed')
  })

  it('reloads the decisions when the server refuses a canned response', async () => {
    const { composable, onSendAccountDenied } = setup({ sendError: denied })
    await composable.sendCannedResponse()
    expect(toast.error).toHaveBeenCalledWith('chat.sendAccountNotAllowed')
    expect(onSendAccountDenied).toHaveBeenCalledOnce()
  })

  it('reloads the decisions when the server refuses a template', async () => {
    const { composable, onSendAccountDenied } = setup({ sendError: denied })
    await composable.sendTemplateMessage()
    expect(toast.error).toHaveBeenCalledWith('chat.sendAccountNotAllowed')
    expect(toast.error).not.toHaveBeenCalledWith('server text')
    expect(onSendAccountDenied).toHaveBeenCalledOnce()
  })
})
