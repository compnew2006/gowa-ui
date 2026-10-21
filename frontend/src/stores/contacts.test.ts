import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { list, listInternalConversations, get, messagesList, put } = vi.hoisted(() => ({
  list: vi.fn(),
  listInternalConversations: vi.fn(),
  get: vi.fn(),
  messagesList: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/services/api', () => ({
  contactsService: {
    list,
    listInternalConversations,
    get,
  },
  messagesService: {
    list: messagesList,
  },
  api: {
    put,
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: { id: 'user-1' },
    userSettings: {},
    hasPermission: () => true,
  }),
}))

import { useContactsStore } from './contacts'

type ListParams = { page: number; search?: string; chat_status?: string }

function page(ids: string[], total: number, chatStatus = 'closed') {
  const contacts = ids.map(id => ({ id, phone_number: id, name: id, chat_status: chatStatus, tags: [] }))
  return Promise.resolve({ data: { data: { contacts, total } } })
}

// A Private-tab pair: the two org accounts' copies of one conversation.
function pairConversation(a: string, b: string) {
  const side = (account: string, peer: string) => ({
    contact_id: `${account}-side`,
    account,
    peer_account: peer,
    last_message_preview: '',
    unread_count: 0,
  })
  return {
    key: `${a}|${b}`,
    accounts: [a, b] as [string, string],
    last_message_preview: '',
    unread_count: 0,
    sides: [side(a, b), side(b, a)],
  }
}

describe('contacts store list paging', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listInternalConversations.mockResolvedValue({ data: { data: { conversations: [] } } })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('appends the next closed page when no list fetch intervenes', async () => {
    list.mockImplementation((params: ListParams) => {
      if (params.chat_status !== 'closed') return page([], 0)
      return page([`closed-${params.page}`], 120)
    })

    const store = useContactsStore()
    store.activeListTab = 'closed'
    await store.fetchContacts()
    await store.loadMoreContacts()

    expect(store.contacts.map(c => c.id)).toEqual(['closed-1', 'closed-2'])
    expect(store.hasMoreContacts).toBe(true)
  })

  it('drops a closed load-more response that a newer list fetch superseded', async () => {
    let resolveStalePage!: (value: unknown) => void
    list.mockImplementation((params: ListParams) => {
      if (params.chat_status !== 'closed') return page([], 0)
      if (params.search === 'zed') return page(['closed-zed'], 1)
      if (params.page === 1) return page(['closed-1'], 120)
      return new Promise(resolve => { resolveStalePage = resolve })
    })

    const store = useContactsStore()
    store.activeListTab = 'closed'
    await store.fetchContacts()
    expect(store.hasMoreContacts).toBe(true)

    // Scroll the Closed tab, then search before page 2 comes back.
    const loadMore = store.loadMoreContacts()
    await store.fetchContacts({ search: 'zed' })
    resolveStalePage(await page(['closed-2'], 120))
    await loadMore

    expect(store.contacts.map(c => c.id)).toEqual(['closed-zed'])
    expect(store.hasMoreContacts).toBe(false)
  })

  it('drops a general load-more response that a newer list fetch superseded', async () => {
    let resolveStalePage!: (value: unknown) => void
    list.mockImplementation((params: ListParams) => {
      if (params.chat_status === 'closed') return page([], 0)
      if (params.search === 'zed') return page(['open-zed'], 1, 'open')
      if (params.page === 1) return page(['open-1'], 120, 'open')
      return new Promise(resolve => { resolveStalePage = resolve })
    })

    const store = useContactsStore()
    store.activeListTab = 'all'
    await store.fetchContacts()

    const loadMore = store.loadMoreContacts()
    await store.fetchContacts({ search: 'zed' })
    resolveStalePage(await page(['open-2'], 120, 'open'))
    await loadMore

    expect(store.contacts.map(c => c.id)).toEqual(['open-zed'])
    expect(store.hasMoreContacts).toBe(false)
  })

  it('keeps the active search on a refresh that passes no search', async () => {
    // Fake timers park the debounced search watcher, so only the explicit
    // refresh below (e.g. a websocket new_message) hits the API.
    vi.useFakeTimers()
    list.mockImplementation(() => page([], 0))

    const store = useContactsStore()
    store.searchQuery = 'zed'
    await store.fetchContacts()

    const searches = list.mock.calls.map(([params]) => [params.chat_status ?? 'general', params.search])
    expect(searches).toEqual([['general', 'zed'], ['closed', 'zed']])
  })
})

describe('contacts store account filter', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listInternalConversations.mockResolvedValue({ data: { data: { conversations: [] } } })
  })

  it('sends selected accounts with the general, closed and load-more requests', async () => {
    list.mockImplementation((params: ListParams) =>
      params.chat_status === 'closed' ? page([`closed-${params.page}`], 120) : page([`open-${params.page}`], 120, 'open'))

    const store = useContactsStore()
    store.selectedAccounts = ['acc1', 'acc2']
    store.selectedTags = ['vip']
    store.activeListTab = 'closed'
    await store.fetchContacts() // general page 1 + closed page 1
    await store.loadMoreContacts() // closed page 2
    store.activeListTab = 'all'
    await store.loadMoreContacts() // general page 2

    const requests = list.mock.calls.map(([params]) => [params.chat_status ?? 'general', params.page, params.tags, params.accounts])
    expect(requests).toEqual([
      ['general', 1, 'vip', 'acc1,acc2'],
      ['closed', 1, 'vip', 'acc1,acc2'],
      ['closed', 2, 'vip', 'acc1,acc2'],
      ['general', 2, 'vip', 'acc1,acc2'],
    ])
  })

  it('omits accounts from list requests when no account is selected', async () => {
    list.mockImplementation(() => page([], 0))

    const store = useContactsStore()
    store.activeListTab = 'closed'
    await store.fetchContacts()

    expect(list.mock.calls.map(([params]) => params.accounts)).toEqual([undefined, undefined])
  })

  it('keeps only Private pair conversations on a selected account', async () => {
    listInternalConversations.mockResolvedValue({
      data: { data: { conversations: [pairConversation('acc1', 'acc2'), pairConversation('acc3', 'acc4')] } },
    })

    const store = useContactsStore()
    store.activeListTab = 'internal'
    await store.fetchInternalConversations()
    // Private rows as the sidebar renders them (displayedContacts minus the pinned Status row).
    const privateRows = () => store.displayedContacts.filter(c => c.is_internal).map(c => c.name)
    expect(store.internalCount).toBe(2)

    store.selectedAccounts = ['acc3']
    expect(store.internalCount).toBe(1)
    expect(privateRows()).toEqual(['acc3 ↔ acc4'])

    store.selectedAccounts = ['acc2', 'acc4']
    expect(privateRows()).toEqual(['acc1 ↔ acc2', 'acc3 ↔ acc4'])

    store.selectedAccounts = []
    expect(store.internalCount).toBe(2)
  })
})

describe('contacts store claim', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listInternalConversations.mockResolvedValue({ data: { data: { conversations: [] } } })
    messagesList.mockResolvedValue({ data: { data: { messages: [], total: 0, has_more: false } } })
  })

  it('re-reads the claimed conversation so its access mode comes from the server', async () => {
    put.mockResolvedValue({ data: { data: { status: 'success' } } })
    get.mockResolvedValue({
      data: { data: { id: 'contact-1', assigned_user_id: 'user-1', chat_status: 'open', access_mode: 'current_assignee', can_reply: true } },
    })

    const store = useContactsStore()
    store.setCurrentContact({
      id: 'contact-1',
      phone_number: '1',
      name: 'Contact',
      chat_status: 'pending',
      access_mode: 'restricted',
      can_reply: false,
      tags: [],
    } as never)

    await store.claimChat('contact-1')

    expect(get).toHaveBeenCalledWith('contact-1')
    expect(store.currentContact?.access_mode).toBe('current_assignee')
    expect(store.currentContact?.can_reply).toBe(true)
  })
})
