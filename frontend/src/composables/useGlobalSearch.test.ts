/**
 * Task 10: Global search — composable unit tests.
 *
 * Searching "Sara" returns the contact AND the conversation, each labeled
 * by scope; an unknown string shows a teaching empty state. Contacts and
 * conversations come from the existing contactsService.list endpoint, which
 * the server already scopes through scopeAssignedContact — no new gate.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

const allowedPermissions = new Set<string>([
  'chat',
  'contacts.manage',
  'settings.general',
  'tags',
])
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    hasPermission: (permission: string) => allowedPermissions.has(permission),
  }),
}))

const listMock = vi.fn()
vi.mock('@/services/api', () => ({
  contactsService: { list: (...args: unknown[]) => listMock(...args) },
}))

import { useGlobalSearch } from './useGlobalSearch'

const saraContact = {
  id: 'contact-sara',
  phone_number: '966500000001',
  name: 'Sara Ahmed',
  last_message_at: '2026-11-22T08:00:00.000Z',
  last_message_preview: 'Hello, I need help',
}

function mockSaraResponse() {
  listMock.mockResolvedValue({ data: { data: { contacts: [saraContact] } } })
}

async function searchFor(search: ReturnType<typeof useGlobalSearch>, query: string) {
  search.query.value = query
  // debounceMs: 0 still defers one macrotask
  await new Promise((resolve) => setTimeout(resolve, 10))
  await flushPromises()
}

describe('useGlobalSearch', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    allowedPermissions.clear()
    ;['chat', 'contacts.manage', 'settings.general', 'tags'].forEach((p) =>
      allowedPermissions.add(p),
    )
  })

  it('searching "Sara" returns the contact and the conversation, each labeled by scope', async () => {
    mockSaraResponse()
    const search = useGlobalSearch({ debounceMs: 0 })

    await searchFor(search, 'Sara')

    expect(listMock).toHaveBeenCalledWith(
      expect.objectContaining({ search: 'Sara' }),
    )
    expect(search.contactItems.value).toHaveLength(1)
    expect(search.contactItems.value[0]).toMatchObject({
      kind: 'contact',
      title: 'Sara Ahmed',
    })
    expect(search.conversationItems.value).toHaveLength(1)
    expect(search.conversationItems.value[0]).toMatchObject({
      kind: 'conversation',
      title: 'Sara Ahmed',
    })
    // Scope labels shown as group headings in the UI.
    expect(search.contactGroupLabel.value).toBe('globalSearch.contacts')
    expect(search.conversationGroupLabel.value).toBe(
      'globalSearch.conversations',
    )
  })

  it('a contact with no messages appears only as a contact, not a conversation', async () => {
    listMock.mockResolvedValue({
      data: {
        data: {
          contacts: [{ ...saraContact, last_message_at: undefined, last_message_preview: undefined }],
        },
      },
    })
    const search = useGlobalSearch({ debounceMs: 0 })

    await searchFor(search, 'Sara')

    expect(search.contactItems.value).toHaveLength(1)
    expect(search.conversationItems.value).toHaveLength(0)
  })

  it('an unknown string shows a teaching empty state', async () => {
    listMock.mockResolvedValue({ data: { data: { contacts: [] } } })
    const search = useGlobalSearch({ debounceMs: 0 })

    await searchFor(search, 'ZzqxUnknown')

    expect(search.contactItems.value).toHaveLength(0)
    expect(search.conversationItems.value).toHaveLength(0)
    expect(search.settingsItems.value).toHaveLength(0)
    expect(search.isEmpty.value).toBe(true)
    expect(search.emptyTitle.value).toBe('globalSearch.noResultsTitle')
    expect(search.emptyDescription.value).toBe(
      'globalSearch.noResultsDescription',
    )
  })

  it('matches settings pages by name, permission-filtered', async () => {
    listMock.mockResolvedValue({ data: { data: { contacts: [] } } })
    const search = useGlobalSearch({ debounceMs: 0 })

    await searchFor(search, 'tags')

    expect(search.settingsItems.value).toHaveLength(1)
    expect(search.settingsItems.value[0]).toMatchObject({
      kind: 'settings',
      to: { path: '/settings/tags' },
    })
  })

  it('hides the Contacts group when the user lacks contacts.manage:read', async () => {
    mockSaraResponse()
    allowedPermissions.delete('contacts.manage')
    const search = useGlobalSearch({ debounceMs: 0 })

    await searchFor(search, 'Sara')

    expect(search.contactItems.value).toHaveLength(0)
    // The conversation is still visible: chat visibility is separate.
    expect(search.conversationItems.value).toHaveLength(1)
  })

  it('selecting a conversation navigates to chat and closes the palette', async () => {
    mockSaraResponse()
    const search = useGlobalSearch({ debounceMs: 0 })
    search.open.value = true

    await searchFor(search, 'Sara')
    search.select(search.conversationItems.value[0])

    expect(push).toHaveBeenCalledWith({
      name: 'chat-conversation',
      params: { contactId: 'contact-sara' },
    })
    expect(search.open.value).toBe(false)
    expect(search.query.value).toBe('')
  })

  it('selecting a contact navigates to the contact detail page', async () => {
    mockSaraResponse()
    const search = useGlobalSearch({ debounceMs: 0 })

    await searchFor(search, 'Sara')
    search.select(search.contactItems.value[0])

    expect(push).toHaveBeenCalledWith({
      name: 'contact-detail',
      params: { id: 'contact-sara' },
    })
  })

  it('does not query the server for a query shorter than 2 characters', async () => {
    const search = useGlobalSearch({ debounceMs: 0 })

    await searchFor(search, 'S')

    expect(listMock).not.toHaveBeenCalled()
    expect(search.isEmpty.value).toBe(false)
  })
})
