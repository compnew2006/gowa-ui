import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, reactive } from 'vue'

vi.mock('@/services/api', () => ({
  api: { get: vi.fn(), delete: vi.fn() },
  contactsService: { assign: vi.fn(), transfer: vi.fn(), transferTargets: vi.fn() }
}))
vi.mock('vue-sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn() }
}))

import { toast } from 'vue-sonner'
import { contactsService } from '@/services/api'
import { useChatLifecycle } from './useChatLifecycle'

const ME = 'user-me'
const COLLEAGUE = { id: 'user-mona', full_name: 'Mona', role_name: 'agent', is_available: true }

function setup(opts: { role?: string; assignee?: string; status?: string; canAssign?: boolean } = {}) {
  const contactsStore = reactive({
    currentContact: { id: 'contact-1', assigned_user_id: opts.assignee ?? ME, chat_status: opts.status ?? 'open' } as any,
    selectedContactIds: new Set<string>(),
    setCurrentContact: vi.fn(),
    fetchContacts: vi.fn().mockResolvedValue(undefined),
    fetchContact: vi.fn().mockResolvedValue({ id: 'contact-1', assigned_user_id: COLLEAGUE.id }),
    fetchMessages: vi.fn(),
    claimChat: vi.fn(),
    joinChat: vi.fn(),
    leaveChat: vi.fn(),
    releaseChat: vi.fn(),
    closeChat: vi.fn(),
    reopenChat: vi.fn(),
    bulkReleaseChats: vi.fn(),
    inviteCollaborator: vi.fn(),
    removeCollaborator: vi.fn()
  })
  const authStore = {
    userRole: opts.role ?? 'agent',
    user: { id: ME },
    hasPermission: (resource: string, action: string) =>
      resource === 'chat.assign' && action === 'write' && (opts.canAssign ?? true)
  }
  const usersStore = { users: [], fetchUsers: vi.fn() }
  const life = useChatLifecycle({ t: (key: string) => key, contactsStore, authStore, usersStore })
  return { life, contactsStore }
}

describe('useChatLifecycle transfer', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    vi.mocked(contactsService.transferTargets).mockResolvedValue({ data: { data: { users: [COLLEAGUE] } } } as any)
    vi.mocked(contactsService.transfer).mockResolvedValue({} as any)
  })

  it('offers transfer only to the assigned agent of an open chat with chat.assign:write', () => {
    expect(setup().life.canTransferChat.value).toBe(true)
    expect(setup({ assignee: 'someone-else' }).life.canTransferChat.value).toBe(false)
    expect(setup({ status: 'closed' }).life.canTransferChat.value).toBe(false)
    expect(setup({ canAssign: false }).life.canTransferChat.value).toBe(false)
    // Admins/managers use the Assign dialog instead.
    expect(setup({ role: 'manager' }).life.canTransferChat.value).toBe(false)
  })

  it('loads colleagues on open and resets selection on close', async () => {
    const { life } = setup()
    life.isTransferDialogOpen.value = true
    await nextTick()
    await vi.waitFor(() => expect(life.transferTargets.value).toEqual([COLLEAGUE]))

    life.selectedTransferTargetId.value = COLLEAGUE.id
    life.transferSearchQuery.value = 'mo'
    life.isTransferDialogOpen.value = false
    await nextTick()
    expect(life.selectedTransferTargetId.value).toBeNull()
    expect(life.transferSearchQuery.value).toBe('')
  })

  it('flags a failed colleague load so the dialog can offer a retry', async () => {
    vi.mocked(contactsService.transferTargets).mockRejectedValueOnce(new Error('boom'))
    const { life } = setup()
    await life.loadTransferTargets()
    expect(life.transferTargetsFailed.value).toBe(true)
    expect(life.transferTargets.value).toEqual([])
  })

  it('transfers to the selected colleague and resyncs the conversation', async () => {
    const { life, contactsStore } = setup()
    await life.loadTransferTargets()
    life.selectedTransferTargetId.value = COLLEAGUE.id
    life.isTransferDialogOpen.value = true

    await life.transferChat()

    expect(contactsService.transfer).toHaveBeenCalledWith('contact-1', COLLEAGUE.id)
    expect(toast.success).toHaveBeenCalledWith('chat.conversationTransferred')
    expect(life.isTransferDialogOpen.value).toBe(false)
    expect(contactsStore.fetchContact).toHaveBeenCalledWith('contact-1')
    expect(contactsStore.setCurrentContact).toHaveBeenCalledWith({ id: 'contact-1', assigned_user_id: COLLEAGUE.id })
  })

  it('surfaces the server refusal and keeps the dialog open', async () => {
    vi.mocked(contactsService.transfer).mockRejectedValueOnce({
      response: { data: { message: 'This colleague cannot receive the conversation' } }
    })
    const { life } = setup()
    await life.loadTransferTargets()
    life.selectedTransferTargetId.value = COLLEAGUE.id
    life.isTransferDialogOpen.value = true

    await life.transferChat()

    expect(toast.error).toHaveBeenCalledWith('This colleague cannot receive the conversation')
    expect(life.isTransferDialogOpen.value).toBe(true)
    expect(life.isTransferring.value).toBe(false)
  })
})
