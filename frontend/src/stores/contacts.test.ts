import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { list } = vi.hoisted(() => ({ list: vi.fn() }))

vi.mock('@/services/api', () => ({
  contactsService: {
    list,
    listInternalConversations: () => Promise.resolve({ data: { data: { conversations: [] } } }),
  },
  messagesService: {},
  api: {},
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

describe('contacts store list paging', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
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
