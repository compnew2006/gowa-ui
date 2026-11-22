import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, reactive } from 'vue'

vi.mock('@/services/api', () => ({
  api: { get: vi.fn(), delete: vi.fn() },
  contactsService: { assign: vi.fn(), transfer: vi.fn(), transferTargets: vi.fn() }
}))
vi.mock('vue-sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), warning: vi.fn() }
}))

import { toast } from 'vue-sonner'
import { contactsService } from '@/services/api'
import { useChatLifecycle } from './useChatLifecycle'

const ME = 'user-me'
const COLLEAGUE = { id: 'user-mona', full_name: 'Mona', role_name: 'agent', is_available: true }

function setup(opts: { role?: string; assignee?: string; status?: string; canAssign?: boolean; isPendingClaim?: boolean; isAdminOrManager?: boolean; isCollaborator?: boolean; contacts?: any[]; selected?: string[]; bulkCloseResult?: { closed: number; failed: number } } = {}) {
  // 'assignee' in opts lets a test pass assignee: undefined for unclaimed.
  const assignee = 'assignee' in opts ? opts.assignee : ME
  const status = opts.status ?? 'open'
  const isAdminOrManager = opts.isAdminOrManager ?? (opts.role === 'admin' || opts.role === 'manager')
  const t = vi.fn((key: string) => key)
  const contactsStore = reactive({
    currentContact: { id: 'contact-1', assigned_user_id: assignee, chat_status: status } as any,
    // Close-gate flags, mirroring the contacts store computeds. Defaults
    // derive from role/assignee/status the way the real store does.
    isPendingClaim: opts.isPendingClaim ?? (status === 'pending' && !assignee && !isAdminOrManager),
    isAdminOrManager,
    isAssignedToMe: assignee === ME,
    isCollaborator: opts.isCollaborator ?? false,
    contacts: opts.contacts ?? [],
    selectedContactIds: new Set<string>(opts.selected ?? []),
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
    bulkCloseChats: vi.fn().mockResolvedValue(opts.bulkCloseResult ?? { closed: 0, failed: 0 }),
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
  const life = useChatLifecycle({ t, contactsStore, authStore, usersStore })
  return { life, contactsStore, t }
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

describe('useChatLifecycle bulk close', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  // One row per gate outcome. Assignee/collaborators/access_mode mirror the
  // Contact rows the contacts store list carries.
  function bulkRows() {
    return [
      { id: 'c-mine-1', assigned_user_id: ME, chat_status: 'open', collaborators: [] },
      { id: 'c-mine-2', assigned_user_id: ME, chat_status: 'open', collaborators: [] },
      { id: 'c-other', assigned_user_id: 'someone-else', chat_status: 'open', collaborators: [] },
      { id: 'c-collab', assigned_user_id: 'someone-else', chat_status: 'open', collaborators: [{ user_id: ME }] },
      { id: 'c-pending', assigned_user_id: undefined, chat_status: 'pending', collaborators: [] },
      { id: 'c-closed', assigned_user_id: ME, chat_status: 'closed', collaborators: [] },
      { id: 'c-readonly', assigned_user_id: 'someone-else', chat_status: 'open', collaborators: [], access_mode: 'historical_assignment_read_only' },
    ]
  }

  // Per-row form of the Task 5 gate pin: an unclaimed (pending) conversation
  // is refused by the bulk gate exactly as the single-close gate refuses it.
  it('refuses an unclaimed (pending) conversation in the per-row gate', () => {
    const { life } = setup({ contacts: bulkRows() })
    const pending = bulkRows().find(c => c.id === 'c-pending')!
    expect(life.canCloseContact(pending)).toBe(false)
  })

  it('applies the single-close gate per row: owner/collaborator yes, uninvolved/closed/read-only no', () => {
    const { life } = setup({ contacts: bulkRows() })
    const byId = new Map(bulkRows().map(c => [c.id, c]))
    expect(life.canCloseContact(byId.get('c-mine-1')!)).toBe(true)
    expect(life.canCloseContact(byId.get('c-collab')!)).toBe(true)
    expect(life.canCloseContact(byId.get('c-other')!)).toBe(false)
    expect(life.canCloseContact(byId.get('c-closed')!)).toBe(false)
    expect(life.canCloseContact(byId.get('c-readonly')!)).toBe(false)
    expect(life.canCloseContact(undefined)).toBe(false)
  })

  it('lets an admin/manager close another agent\'s open conversation', () => {
    const { life } = setup({ role: 'admin', contacts: bulkRows() })
    const byId = new Map(bulkRows().map(c => [c.id, c]))
    expect(life.canCloseContact(byId.get('c-other')!)).toBe(true)
    // …but not a pending unclaimed one, and not a read-only one.
    expect(life.canCloseContact(byId.get('c-pending')!)).toBe(false)
    expect(life.canCloseContact(byId.get('c-readonly')!)).toBe(false)
  })

  it('enables bulk close only when at least one selected conversation is closable', () => {
    expect(setup({ contacts: bulkRows(), selected: [] }).life.canBulkClose.value).toBe(false)
    expect(setup({ contacts: bulkRows(), selected: ['c-other', 'c-pending', 'c-closed'] }).life.canBulkClose.value).toBe(false)
    expect(setup({ contacts: bulkRows(), selected: ['c-other', 'c-mine-1'] }).life.canBulkClose.value).toBe(true)
    expect(setup({ contacts: bulkRows(), selected: ['c-mine-1'] }).life.canBulkClose.value).toBe(true)
  })

  it('clicking bulk Close opens ONE confirmation dialog and closes nothing yet', async () => {
    const { life, contactsStore } = setup({ contacts: bulkRows(), selected: ['c-mine-1', 'c-mine-2'] })
    await life.handleBulkClose()
    expect(life.isBulkCloseDialogOpen.value).toBe(true)
    expect(contactsStore.bulkCloseChats).not.toHaveBeenCalled()
  })

  it('handleBulkClose with no closable selection opens nothing', async () => {
    const { life, contactsStore } = setup({ contacts: bulkRows(), selected: ['c-other', 'c-pending'] })
    await life.handleBulkClose()
    expect(life.isBulkCloseDialogOpen.value).toBe(false)
    expect(contactsStore.bulkCloseChats).not.toHaveBeenCalled()
  })

  it('confirming closes the whole selected batch — all N and only those N', async () => {
    const selected = ['c-mine-1', 'c-mine-2']
    const { life, contactsStore, t } = setup({
      contacts: bulkRows(), selected, bulkCloseResult: { closed: 2, failed: 0 },
    })
    await life.handleBulkClose()
    await life.confirmBulkClose()
    expect(contactsStore.bulkCloseChats).toHaveBeenCalledTimes(1)
    expect(contactsStore.bulkCloseChats).toHaveBeenCalledWith(selected)
    expect(life.isBulkCloseDialogOpen.value).toBe(false)
    expect(life.isBulkClosing.value).toBe(false)
    expect(toast.success).toHaveBeenCalledWith('chat.bulkCloseSuccess')
    expect(t).toHaveBeenCalledWith('chat.bulkCloseSuccess', { count: 2 })
  })

  it('sends the unclaimed conversation too — the server refuses it — and reports the partial result honestly', async () => {
    const selected = ['c-mine-1', 'c-mine-2', 'c-pending']
    const { life, contactsStore, t } = setup({
      contacts: bulkRows(), selected, bulkCloseResult: { closed: 2, failed: 1 },
    })
    await life.handleBulkClose()
    await life.confirmBulkClose()
    // The whole selection goes to the server; the per-conversation gate
    // lives there, so the unclaimed row is refused, not silently skipped.
    expect(contactsStore.bulkCloseChats).toHaveBeenCalledWith(selected)
    expect(life.isBulkCloseDialogOpen.value).toBe(false)
    expect(toast.success).not.toHaveBeenCalled()
    expect(toast.warning).toHaveBeenCalledWith('chat.bulkClosePartial')
    expect(t).toHaveBeenCalledWith('chat.bulkClosePartial', { count: 2, failed: 1 })
  })

  it('confirming without opening the dialog first does nothing', async () => {
    const { life, contactsStore } = setup({ contacts: bulkRows(), selected: ['c-mine-1'] })
    await life.confirmBulkClose()
    expect(contactsStore.bulkCloseChats).not.toHaveBeenCalled()
  })

  it('a failed bulk close keeps the dialog open so the user can retry or cancel', async () => {
    const { life, contactsStore } = setup({ contacts: bulkRows(), selected: ['c-mine-1'] })
    vi.mocked(contactsStore.bulkCloseChats).mockRejectedValueOnce(new Error('boom'))
    await life.handleBulkClose()
    await life.confirmBulkClose()
    expect(life.isBulkCloseDialogOpen.value).toBe(true)
    expect(life.isBulkClosing.value).toBe(false)
  })
})
