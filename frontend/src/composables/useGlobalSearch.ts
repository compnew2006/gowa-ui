import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { contactsService } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import type { Contact } from '@/stores/contacts'
import { navigationSections } from '@/components/layout/navigation'

export type GlobalSearchItemKind = 'contact' | 'conversation' | 'settings'

export interface GlobalSearchItem {
  kind: GlobalSearchItemKind
  id: string
  title: string
  subtitle?: string
  to: { name?: string; params?: Record<string, string>; path?: string }
}

/** Minimum characters before the server is queried. */
export const GLOBAL_SEARCH_MIN_QUERY = 2

/**
 * Global search (Task 10): contacts + conversations + settings pages.
 *
 * Contacts and conversations reuse the existing `contactsService.list`
 * endpoint (`GET /contacts?search=…`), which the server already scopes
 * through `scopeAssignedContact` — the same gate the chat list uses — so
 * this task adds no second visibility gate and no new backend search
 * index. A row is a *conversation* when it has message history
 * (`last_message_at` / `last_message_preview`); every row is a *contact*.
 * Settings pages are matched client-side against the settings children in
 * `navigation.ts`, filtered by the viewer's permissions.
 */
export function useGlobalSearch(options?: { debounceMs?: number }) {
  const debounceMs = options?.debounceMs ?? 150
  const { t } = useI18n()
  const router = useRouter()
  const authStore = useAuthStore()

  const open = ref(false)
  const query = ref('')
  const isLoading = ref(false)
  const contacts = ref<Contact[]>([])
  const searchedQuery = ref('')

  const trimmedQuery = computed(() => query.value.trim())
  const canSearch = computed(
    () => trimmedQuery.value.length >= GLOBAL_SEARCH_MIN_QUERY,
  )

  let debounceTimer: ReturnType<typeof setTimeout> | undefined
  let requestSeq = 0

  async function runSearch(q: string) {
    const seq = ++requestSeq
    isLoading.value = true
    try {
      const response = await contactsService.list({ search: q, limit: 8 })
      if (seq !== requestSeq) return
      const data = response.data?.data || response.data
      contacts.value = (data?.contacts || []) as Contact[]
      searchedQuery.value = q
    } catch {
      if (seq !== requestSeq) return
      contacts.value = []
      searchedQuery.value = q
    } finally {
      if (seq === requestSeq) isLoading.value = false
    }
  }

  watch(query, (q) => {
    clearTimeout(debounceTimer)
    const trimmed = q.trim()
    if (trimmed.length < GLOBAL_SEARCH_MIN_QUERY) {
      contacts.value = []
      searchedQuery.value = ''
      isLoading.value = false
      return
    }
    debounceTimer = setTimeout(() => void runSearch(trimmed), debounceMs)
  })

  const canSeeContactsDirectory = computed(() =>
    authStore.hasPermission('contacts.manage', 'read'),
  )
  const canSeeChat = computed(() => authStore.hasPermission('chat', 'read'))

  const contactItems = computed<GlobalSearchItem[]>(() => {
    if (!canSeeContactsDirectory.value || !searchedQuery.value) return []
    return contacts.value.map((c) => ({
      kind: 'contact',
      id: `contact-${c.id}`,
      title: c.name || c.phone_number,
      subtitle: c.phone_number,
      to: { name: 'contact-detail', params: { id: c.id } },
    }))
  })

  const conversationItems = computed<GlobalSearchItem[]>(() => {
    if (!canSeeChat.value || !searchedQuery.value) return []
    return contacts.value
      .filter((c) => c.last_message_at || c.last_message_preview)
      .map((c) => ({
        kind: 'conversation',
        id: `conversation-${c.id}`,
        title: c.name || c.phone_number,
        subtitle: c.last_message_preview || c.phone_number,
        to: { name: 'chat-conversation', params: { contactId: c.id } },
      }))
  })

  const settingsItems = computed<GlobalSearchItem[]>(() => {
    if (!canSearch.value) return []
    const needle = trimmedQuery.value.toLowerCase()
    const settingsSection = navigationSections.find((s) =>
      s.items.some((item) => item.path === '/settings'),
    )
    const children =
      settingsSection?.items.find((item) => item.path === '/settings')
        ?.children ?? []
    return children
      .filter(
        (child) =>
          !child.permission ||
          authStore.hasPermission(child.permission, 'read'),
      )
      .filter((child) =>
        t(child.name).toLowerCase().includes(needle),
      )
      .map((child) => ({
        kind: 'settings',
        id: `settings-${child.path}`,
        title: t(child.name),
        subtitle: child.path,
        to: { path: child.path },
      }))
  })

  const totalResults = computed(
    () =>
      contactItems.value.length +
      conversationItems.value.length +
      settingsItems.value.length,
  )

  /** Server answered (or nothing was searchable) and nothing matched. */
  const isEmpty = computed(
    () => canSearch.value && !isLoading.value && totalResults.value === 0,
  )

  const contactGroupLabel = computed(() => t('globalSearch.contacts'))
  const conversationGroupLabel = computed(() =>
    t('globalSearch.conversations'),
  )
  const settingsGroupLabel = computed(() => t('globalSearch.settingsPages'))
  const emptyTitle = computed(() =>
    t('globalSearch.noResultsTitle', { query: searchedQuery.value }),
  )
  const emptyDescription = computed(() =>
    t('globalSearch.noResultsDescription'),
  )

  function openPalette() {
    open.value = true
  }

  function closePalette() {
    open.value = false
    query.value = ''
    contacts.value = []
    searchedQuery.value = ''
  }

  function select(item: GlobalSearchItem) {
    closePalette()
    void router.push(item.to as never)
  }

  return {
    open,
    query,
    isLoading,
    contacts,
    searchedQuery,
    canSearch,
    contactItems,
    conversationItems,
    settingsItems,
    totalResults,
    isEmpty,
    contactGroupLabel,
    conversationGroupLabel,
    settingsGroupLabel,
    emptyTitle,
    emptyDescription,
    openPalette,
    closePalette,
    select,
  }
}

export type UseGlobalSearch = ReturnType<typeof useGlobalSearch>
