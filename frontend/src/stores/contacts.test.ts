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

type ListParams = {
  page?: number
  limit?: number
  search?: string
  chat_status?: string
  has_messages?: boolean
  exclude_internal?: boolean
  exclude_closed?: boolean
  assigned?: string
  tags?: string
  accounts?: string
}

function page(ids: string[], total: number, chatStatus = 'closed') {
  const contacts = ids.map(id => ({ id, phone_number: id, name: id, chat_status: chatStatus, tags: [] }))
  return Promise.resolve({ data: { data: { contacts, total } } })
}

// Kind of a list() call, for asserting which request a call was: the Chat
// sidebar's general/closed page loads vs the limit=1 server-total ("count")
// requests that back the tab badges (Decision 3).
function callKind(params: ListParams): string {
  if (params.limit === 1) {
    if (params.chat_status === 'pending') return 'pending-count'
    if (params.assigned) return 'me-count'
    if (params.has_messages && params.exclude_internal) return 'all-count'
    return 'other-count'
  }
  return params.chat_status === 'closed' ? 'closed' : 'general'
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
      if (params.limit === 1) return page([], 0) // tab-total count request
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

    const searches = list.mock.calls.map(([params]) => [callKind(params), params.search])
    expect(searches).toEqual([
      ['general', 'zed'],
      ['closed', 'zed'],
      ['all-count', 'zed'],
      ['pending-count', 'zed'],
      ['me-count', 'zed'],
    ])
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

    const requests = list.mock.calls.map(([params]) => [callKind(params), params.page, params.tags, params.accounts])
    expect(requests).toEqual([
      ['general', 1, 'vip', 'acc1,acc2'],
      ['closed', 1, 'vip', 'acc1,acc2'],
      ['all-count', undefined, 'vip', 'acc1,acc2'],
      ['pending-count', undefined, 'vip', 'acc1,acc2'],
      ['me-count', undefined, 'vip', 'acc1,acc2'],
      ['closed', 2, 'vip', 'acc1,acc2'],
      ['general', 2, 'vip', 'acc1,acc2'],
    ])
  })

  it('omits accounts from list requests when no account is selected', async () => {
    list.mockImplementation(() => page([], 0))

    const store = useContactsStore()
    store.activeListTab = 'closed'
    await store.fetchContacts()

    // General + closed pages and the three tab-total count requests.
    expect(list.mock.calls.map(([params]) => params.accounts)).toEqual(
      [undefined, undefined, undefined, undefined, undefined])
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

describe('contacts store conversation counts (Decision 3)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listInternalConversations.mockResolvedValue({ data: { data: { conversations: [] } } })
  })

  // General page: 1 pending + 1 mine, out of a server total of 120 — the
  // badges must show the server totals, not the 2 loaded rows. The closed
  // page holds 1 row out of 33. Count requests (limit=1) answer per filter.
  function mockList() {
    list.mockImplementation((params: ListParams) => {
      if (params.limit === 1) {
        if (params.chat_status === 'pending') return page([], 27)
        if (params.assigned === 'any') return page([], 9)
        if (params.assigned === 'me') return page([], 4)
        if (params.has_messages && params.exclude_internal) return page([], 86)
        return page([], 0)
      }
      if (params.chat_status === 'closed') return page(['closed-1'], 33)
      return Promise.resolve({ data: { data: {
        contacts: [
          { id: 'open-1', phone_number: '1', name: 'Open', chat_status: 'open', last_message_at: '2026-11-22T00:00:00Z', tags: [] },
          { id: 'mine-1', phone_number: '2', name: 'Mine', chat_status: 'open', last_message_at: '2026-11-22T00:00:00Z', assigned_user_id: 'user-1', tags: [] },
        ],
        total: 120,
      } } })
    })
  }

  it('fetches only conversations for Chat and shows server totals on the tabs', async () => {
    mockList()

    const store = useContactsStore()
    await store.fetchContacts()

    // The general list, its load-more and the closed page are conversation
    // queries: never-messaged contacts are contacts, not conversations.
    const pageCalls = list.mock.calls.map(([p]) => p).filter(p => p.limit !== 1)
    expect(pageCalls).toHaveLength(2)
    for (const params of pageCalls) expect(params.has_messages).toBe(true)
    expect(store.contactsTotal).toBe(120)

    // Badges are the server totals, not the loaded rows (2 general + 1 closed).
    expect(store.allCount).toBe(86)
    expect(store.pendingCount).toBe(27)
    expect(store.myCount).toBe(9) // supervisor mock: assigned=any
    expect(store.closedCount).toBe(33)

    // The count requests carry the same filters as the list itself.
    const countCalls = list.mock.calls.map(([p]) => p).filter(p => p.limit === 1)
    expect(countCalls.map(p => callKind(p))).toEqual(['all-count', 'pending-count', 'me-count'])
    const meCount = countCalls.find(p => callKind(p) === 'me-count')!
    expect(meCount.assigned).toBe('any')
    expect(meCount.has_messages).toBe(true)
    expect(meCount.exclude_internal).toBe(true)
    expect(meCount.exclude_closed).toBe(true)
    const allCount = countCalls.find(p => callKind(p) === 'all-count')!
    expect(allCount.has_messages).toBe(true)
    expect(allCount.exclude_internal).toBe(true)
    expect(allCount.chat_status).toBeUndefined()
  })

  it('keeps has_messages on load-more so page 2 stays conversations', async () => {
    mockList()

    const store = useContactsStore()
    await store.fetchContacts()
    store.activeListTab = 'all'
    await store.loadMoreContacts()

    const generalPage2 = list.mock.calls.map(([p]) => p)
      .find(p => p.limit !== 1 && p.chat_status !== 'closed' && p.page === 2)
    expect(generalPage2?.has_messages).toBe(true)
  })

  it('falls back to loaded rows when a tab-total request fails', async () => {
    list.mockImplementation((params: ListParams) => {
      if (params.limit === 1) return Promise.reject(new Error('boom'))
      if (params.chat_status === 'closed') return page(['closed-1'], 33)
      return Promise.resolve({ data: { data: {
        contacts: [
          { id: 'open-1', phone_number: '1', name: 'Open', chat_status: 'open', last_message_at: '2026-11-22T00:00:00Z', tags: [] },
        ],
        total: 120,
      } } })
    })

    const store = useContactsStore()
    await store.fetchContacts()

    // No server totals available: badges degrade to the loaded rows rather
    // than showing 0. The closed badge still has its server total. (All
    // counts every loaded customer conversation, open and closed.)
    expect(store.allCount).toBe(2)
    expect(store.pendingCount).toBe(1)
    expect(store.myCount).toBe(0)
    expect(store.closedCount).toBe(33)
  })
})

describe('contacts store claim', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listInternalConversations.mockResolvedValue({ data: { data: { conversations: [] } } })
    messagesList.mockResolvedValue({ data: { data: { messages: [], total: 0, has_more: false } } })
    // claimChat re-reads the tab totals (Task 7a), so the list mock must
    // answer the limit=1 count requests instead of returning undefined.
    list.mockImplementation(() => page([], 0))
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

  it('claiming a pending conversation moves the tab counters from pending to me without a reload (Task 7a)', async () => {
    // Live 2026-11-21: after salama claimed, the badges stayed "Pending 30"
    // and "My conversations 0". The badges read the server totals first
    // (Decision 3), so the local assignment update alone is not enough —
    // the claim must re-read the totals. The mock server totals move when
    // the claim PUT lands, like the real backend does.
    let serverPending = 30
    let serverMe = 0
    const serverAll = 31
    list.mockImplementation((params: ListParams) => {
      if (params.limit === 1) {
        if (params.chat_status === 'pending') return page([], serverPending)
        if (params.assigned) return page([], serverMe)
        return page([], serverAll)
      }
      if (params.chat_status === 'closed') return page([], 0)
      return Promise.resolve({ data: { data: {
        contacts: [
          { id: 'contact-1', phone_number: '1', name: 'Contact', chat_status: 'pending', last_message_at: '2026-11-21T00:00:00Z', tags: [] },
        ],
        total: 31,
      } } })
    })
    put.mockImplementation(async () => {
      serverPending = 29
      serverMe = 1
      return { data: { data: { status: 'success' } } }
    })
    get.mockResolvedValue({
      data: { data: { id: 'contact-1', assigned_user_id: 'user-1', chat_status: 'open', access_mode: 'current_assignee', can_reply: true } },
    })

    const store = useContactsStore()
    await store.fetchContacts()

    // Badges show the server totals, not the 1 loaded row.
    expect(store.pendingCount).toBe(30)
    expect(store.myCount).toBe(0)
    expect(store.allCount).toBe(31)

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

    // The conversation moved tabs locally …
    expect(store.pendingContacts).toHaveLength(0)
    expect(store.myContacts).toHaveLength(1)
    // … and the badges the user sees moved with it, without a reload.
    expect(store.pendingCount).toBe(29)
    expect(store.myCount).toBe(1)
    expect(store.allCount).toBe(31)
    // The claim re-asked the server for the totals (3 count requests on top
    // of the 3 from the initial fetch).
    expect(list.mock.calls.map(([p]) => p).filter(p => p.limit === 1)).toHaveLength(6)
  })

  it('moves the tab counters before the message re-read finishes (Task 7a fix round 1)', async () => {
    // The totals refresh must not sit behind the current-contact re-reads in
    // claimChat: a slow (or failing) message re-fetch would otherwise leave
    // the badges stale even though the claim PUT already succeeded. The
    // message request is deferred here, so the claim blocks on it — the
    // counters must already have moved by then.
    let serverPending = 30
    let serverMe = 0
    list.mockImplementation((params: ListParams) => {
      if (params.limit === 1) {
        if (params.chat_status === 'pending') return page([], serverPending)
        if (params.assigned) return page([], serverMe)
        return page([], 31)
      }
      if (params.chat_status === 'closed') return page([], 0)
      return Promise.resolve({ data: { data: {
        contacts: [
          { id: 'contact-1', phone_number: '1', name: 'Contact', chat_status: 'pending', last_message_at: '2026-11-21T00:00:00Z', tags: [] },
        ],
        total: 31,
      } } })
    })
    put.mockImplementation(async () => {
      serverPending = 29
      serverMe = 1
      return { data: { data: { status: 'success' } } }
    })
    get.mockResolvedValue({
      data: { data: { id: 'contact-1', assigned_user_id: 'user-1', chat_status: 'open', access_mode: 'current_assignee', can_reply: true } },
    })
    let resolveMessages!: (value: unknown) => void
    const messagesDeferred = new Promise(resolve => { resolveMessages = resolve })
    messagesList.mockImplementation(() => messagesDeferred)

    const store = useContactsStore()
    await store.fetchContacts()
    expect(store.pendingCount).toBe(30)
    expect(store.myCount).toBe(0)

    store.setCurrentContact({
      id: 'contact-1',
      phone_number: '1',
      name: 'Contact',
      chat_status: 'pending',
      access_mode: 'restricted',
      can_reply: false,
      tags: [],
    } as never)

    const claiming = store.claimChat('contact-1')
    // Let the claim run until it blocks on the deferred message re-fetch;
    // the PUT and the totals refresh are plain microtasks, so two timer
    // turns are enough for them to land.
    await new Promise(resolve => setTimeout(resolve, 0))
    await new Promise(resolve => setTimeout(resolve, 0))

    // Counters moved already, without waiting for the re-read to finish.
    expect(store.pendingCount).toBe(29)
    expect(store.myCount).toBe(1)

    resolveMessages({ data: { data: { messages: [], total: 0, has_more: false } } })
    await claiming
    expect(store.pendingCount).toBe(29)
    expect(store.myCount).toBe(1)
  })
})
