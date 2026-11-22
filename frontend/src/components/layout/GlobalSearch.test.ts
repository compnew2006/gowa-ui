/**
 * Task 10: Global search — GlobalSearch palette component tests.
 *
 * Mounts the real palette (shadcn-vue CommandDialog primitives) with a real
 * useGlobalSearch instance and a mocked contactsService. Covers rendering
 * grouped results, the teaching empty state, and keyboard-only use
 * (Ctrl+K to open, arrows + Enter to pick, Esc to close).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'

// jsdom does not implement scrollIntoView; Reka's listbox calls it when the
// highlight moves. Stub it so keyboard navigation can be exercised.
Element.prototype.scrollIntoView = vi.fn()

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    hasPermission: () => true,
  }),
}))

const listMock = vi.fn()
vi.mock('@/services/api', () => ({
  contactsService: { list: (...args: unknown[]) => listMock(...args) },
}))

import { useGlobalSearch } from '@/composables/useGlobalSearch'
import GlobalSearch from './GlobalSearch.vue'

const saraContact = {
  id: 'contact-sara',
  phone_number: '966500000001',
  name: 'Sara Ahmed',
  last_message_at: '2026-11-22T08:00:00.000Z',
  last_message_preview: 'Hello, I need help',
}

function bodyText() {
  return document.body.textContent ?? ''
}

function paletteInput(): HTMLInputElement | null {
  return document.body.querySelector('input')
}

async function typeQuery(text: string) {
  const input = paletteInput()
  expect(input).not.toBeNull()
  input!.value = text
  input!.dispatchEvent(new Event('input', { bubbles: true }))
  await nextTick()
  // Default 150ms debounce + server round-trip.
  await new Promise((resolve) => setTimeout(resolve, 250))
  await flushPromises()
  await nextTick()
}

describe('GlobalSearch', () => {
  let wrapper: VueWrapper | null = null
  let search: ReturnType<typeof useGlobalSearch>

  beforeEach(() => {
    vi.clearAllMocks()
    listMock.mockResolvedValue({ data: { data: { contacts: [saraContact] } } })
    search = useGlobalSearch()
    wrapper = mount(GlobalSearch, {
      props: { search },
      attachTo: document.body,
    })
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
    document.body.innerHTML = ''
  })

  it('opens on Ctrl+K and focuses the search input', async () => {
    expect(search.open.value).toBe(false)

    window.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'k', ctrlKey: true, bubbles: true }),
    )
    await flushPromises()
    await nextTick()

    expect(search.open.value).toBe(true)
    expect(paletteInput()).not.toBeNull()
    expect(document.activeElement).toBe(paletteInput())
  })

  it('searching "Sara" shows the contact and the conversation, each under its scope heading', async () => {
    search.openPalette()
    await flushPromises()
    await nextTick()

    await typeQuery('Sara')

    expect(listMock).toHaveBeenCalledWith(
      expect.objectContaining({ search: 'Sara' }),
    )
    expect(bodyText()).toContain('globalSearch.contacts')
    expect(bodyText()).toContain('globalSearch.conversations')
    // One Sara row per scope group.
    expect(bodyText().match(/Sara Ahmed/g)?.length).toBe(2)
  })

  it('an unknown string shows the teaching empty state', async () => {
    listMock.mockResolvedValue({ data: { data: { contacts: [] } } })
    search.openPalette()
    await flushPromises()
    await nextTick()

    await typeQuery('ZzqxUnknown')

    expect(bodyText()).toContain('globalSearch.noResultsTitle')
    expect(bodyText()).toContain('globalSearch.noResultsDescription')
  })

  it('keyboard-only: ArrowDown then Enter opens the first result', async () => {
    search.openPalette()
    await flushPromises()
    await nextTick()

    await typeQuery('Sara')
    const input = paletteInput()
    expect(input).not.toBeNull()

    input!.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }),
    )
    await nextTick()
    input!.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }),
    )
    await flushPromises()
    await nextTick()

    expect(push).toHaveBeenCalled()
    expect(search.open.value).toBe(false)
  })

  it('Escape closes the palette', async () => {
    search.openPalette()
    await flushPromises()
    await nextTick()
    expect(search.open.value).toBe(true)

    document.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }),
    )
    await flushPromises()
    await nextTick()

    expect(search.open.value).toBe(false)
  })
})
