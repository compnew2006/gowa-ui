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

function setup(opts: { role?: string; assignee?: string; status?: string; canAssign?: boolean; isPendingClaim?: boolean; isAdminOrManager?: boolean; isCollaborator?: boolean } = {}) {
  // 'assignee' in opts lets a test pass assignee: undefined for unclaimed.
  const assignee = 'assignee' in opts ? opts.assignee : ME
  const status = opts.status ?? 'open'
  const isAdminOrManager = opts.isAdminOrManager ?? (opts.role === 'admin' || opts.role === 'manager')
  const contactsStore = reactive({
    currentContact: { id: 'contact-1', assigned_user_id: assignee, chat_status: status } as any,
    // Close-gate flags, mirroring the contacts store computeds. Defaults
    // derive from role/assignee/status the way the real store does.
    isPendingClaim: opts.isPendingClaim ?? (status === 'pending' && !assignee && !isAdminOrManager),
    isAdminOrManager,
    isAssignedToMe: assignee === ME,
    isCollaborator: opts.isCollaborator ?? false,
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

describe('useChatLifecycle close', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  // Regression pin for the solved Close-button mystery (live evidence
  // 2026-11-21, salama@arkan.com): an unclaimed conversation shows no Close
  // affordance — the v-if fails both isPendingClaim and the participant
  // check. The gate below must encode exactly that condition.
  it('shows no Close affordance for an unclaimed (pending) conversation', () => {
    expect(setup({ status: 'pending', assignee: undefined }).life.canCloseConversation.value).toBe(false)
  })

  it('shows Close for the claimed owner of an open conversation', () => {
    expect(setup().life.canCloseConversation.value).toBe(true)
  })

  it('shows Close for an admin/manager on an open conversation they do not own', () => {
    expect(setup({ role: 'admin', assignee: 'someone-else' }).life.canCloseConversation.value).toBe(true)
    expect(setup({ role: 'manager', assignee: 'someone-else' }).life.canCloseConversation.value).toBe(true)
  })

  it('shows Close for a collaborator on an open conversation', () => {
    expect(setup({ assignee: 'someone-else', isCollaborator: true }).life.canCloseConversation.value).toBe(true)
  })

  it('shows no Close for an uninvolved agent, or on a closed conversation', () => {
    expect(setup({ assignee: 'someone-else' }).life.canCloseConversation.value).toBe(false)
    expect(setup({ status: 'closed' }).life.canCloseConversation.value).toBe(false)
  })

  it('clicking Close opens a confirmation dialog and does NOT close yet', async () => {
    const { life, contactsStore } = setup()
    await life.handleClose()
    expect(life.isCloseDialogOpen.value).toBe(true)
    expect(contactsStore.closeChat).not.toHaveBeenCalled()
  })

  it('confirming closes the conversation once and dismisses the dialog', async () => {
    const { life, contactsStore } = setup()
    await life.handleClose()
    await life.confirmClose()
    expect(contactsStore.closeChat).toHaveBeenCalledTimes(1)
    expect(contactsStore.closeChat).toHaveBeenCalledWith('contact-1')
    expect(life.isCloseDialogOpen.value).toBe(false)
    expect(life.isClosing.value).toBe(false)
  })

  it('confirming without opening the dialog first does nothing', async () => {
    const { life, contactsStore } = setup()
    await life.confirmClose()
    expect(contactsStore.closeChat).not.toHaveBeenCalled()
  })

  it('a failed close keeps the dialog open so the user can retry or cancel', async () => {
    const { life, contactsStore } = setup()
    vi.mocked(contactsStore.closeChat).mockRejectedValueOnce(new Error('boom'))
    await life.handleClose()
    await life.confirmClose()
    expect(life.isCloseDialogOpen.value).toBe(true)
    expect(life.isClosing.value).toBe(false)
  })

  it('handleClose with no current contact opens nothing', async () => {
    const { life, contactsStore } = setup()
    contactsStore.currentContact = null
    await life.handleClose()
    expect(life.isCloseDialogOpen.value).toBe(false)
    expect(contactsStore.closeChat).not.toHaveBeenCalled()
  })
})
