import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { contactsService, messagesService, api } from '@/services/api'
import { useAuthStore } from '@/stores/auth'

// Phones are stored without leading + or whitespace (see CreateContact in
// internal/handlers/contacts.go). Strip them from a digit-only query so a user
// typing "+91 98765 43210" still matches a stored "919876543210" via the
// server's substring LIKE.
function normalizeContactSearch(raw: string): string {
  const trimmed = raw.trim().replace(/^\+/, '')
  if (trimmed && /^[\d\s+()-]+$/.test(trimmed)) {
    return trimmed.replace(/[\s+()-]/g, '')
  }
  return trimmed
}

export interface Contact {
  id: string
  phone_number: string
  name: string
  profile_name?: string
  avatar_url?: string
  status: string
  tags: string[]
  metadata: Record<string, any>
  last_message_at?: string
  last_inbound_at?: string
  unread_count: number
  assigned_user_id?: string
  assigned_user_name?: string
  whatsapp_account?: string
  marketing_opt_out?: boolean
  is_group_chat?: boolean
  is_newsletter?: boolean
  chat_status?: 'pending' | 'open' | 'closed'
  collaborators?: Collaborator[]
  created_at: string
  updated_at: string
}

export interface Collaborator {
  user_id: string
  name: string
  role: string
  joined_at: string
}

export interface ReplyPreview {
  id: string
  content: any
  message_type: string
  direction: 'incoming' | 'outgoing'
}

export interface Reaction {
  emoji: string
  from_phone?: string
  from_user?: string
}

export interface Message {
  id: string
  contact_id: string
  direction: 'incoming' | 'outgoing'
  message_type: string
  content: any
  media_url?: string
  media_mime_type?: string
  media_filename?: string
  interactive_data?: {
    type?: string
    body?: string
    buttons?: Array<{
      type?: string
      reply?: { id: string; title: string }
      id?: string
      title?: string
    }>
    rows?: Array<{
      id?: string
      title?: string
    }>
  }
  status: string
  wamid?: string
  error_message?: string
  is_reply?: boolean
  reply_to_message_id?: string
  reply_to_message?: ReplyPreview
  reactions?: Reaction[]
  whatsapp_account?: string
  is_group_chat?: boolean
  sender_phone?: string
  sender_push_name?: string
  metadata?: Record<string, any>
  created_at: string
  updated_at: string
}

export const useContactsStore = defineStore('contacts', () => {
  const authStore = useAuthStore()
  const contacts = ref<Contact[]>([])
  const currentContact = ref<Contact | null>(null)
  const messages = ref<Message[]>([])
  const isLoading = ref(false)
  const isLoadingMessages = ref(false)
  const isLoadingOlderMessages = ref(false)
  const hasMoreMessages = ref(false)
  const searchQuery = ref('')
  const selectedTags = ref<string[]>([])
  // Sidebar visibility toggles. Default false (show everything) preserves the
  // pre-existing behavior. Toggling true hides the matching chats from the
  // sortedContacts list (and thus from the sidebar).
  const hideGroupChats = ref(false)
  const hideNewsletterChats = ref(false)
  const replyingTo = ref<Message | null>(null)
  const accountFilter = ref<string | null>(null)

  // Me/Pending/Closed/All tab selector for the chat sidebar. Persisted to
  // localStorage (M2) so the agent's preference survives reloads. The default
  // (when there is no valid stored preference) is role-aware: admins/managers
  // land on 'pending' (the unassigned queue they manage — they have no chats
  // assigned to them directly, so 'me' would show an empty list); agents land
  // on 'me' (their own assigned conversations — the primary working surface).
  // 'closed' and 'all' are supervisor tabs, gated on contacts:write in the
  // view (the admin/manager marker — see canSeeSupervisorTabs below).
  const VALID_TABS = ['me', 'pending', 'closed', 'all'] as const
  type ListTab = typeof VALID_TABS[number]

  // One-time migration: an earlier build defaulted everyone (including admins)
  // to the 'me' tab, which is empty for admins (no chats assigned to them) and
  // left them staring at an empty sidebar / no messages. Bumping this version
  // flag clears the stale 'me' preference so the role-aware default below takes
  // over for everyone once. Users can still re-pick a tab afterwards.
  const TAB_PREF_VERSION = '2'
  const TAB_PREF_VERSION_KEY = 'whatomate.chat.activeListTab.v'
  if (typeof localStorage !== 'undefined'
    && localStorage.getItem(TAB_PREF_VERSION_KEY) !== TAB_PREF_VERSION) {
    localStorage.removeItem('whatomate.chat.activeListTab')
    localStorage.setItem(TAB_PREF_VERSION_KEY, TAB_PREF_VERSION)
  }

  function loadStoredTab(): ListTab {
    const stored = typeof localStorage !== 'undefined'
      ? localStorage.getItem('whatomate.chat.activeListTab') as ListTab | null
      : null
    if (stored && (VALID_TABS as readonly string[]).includes(stored)) {
      return stored
    }
    // Role-aware default. Admins/managers have contacts:write and manage the
    // unassigned queue; agents work their own assigned chats.
    const isManager = authStore.hasPermission('contacts', 'write')
    return isManager ? 'pending' : 'me'
  }
  const activeListTab = ref<ListTab>(loadStoredTab())
  // Persist tab choice (M2). `watch` re-fires on every change, so the stored
  // value always mirrors the live one.
  watch(activeListTab, (tab) => {
    try { localStorage.setItem('whatomate.chat.activeListTab', tab) } catch { /* quota / private mode */ }
  })

  // Supervisor visibility gate for the 'closed' and 'all' tabs. Gated on
  // contacts:write (the admin/manager marker used by canManageAllChats and
  // the role-aware tab default below) rather than contacts:read, because the
  // seeded agent role DOES carry contacts:read — gating on read would surface
  // the supervisor tabs to every agent. Agents keep the two-tab Me/Pending
  // strip focused on their own work.
  const canSeeSupervisorTabs = computed(() =>
    authStore.hasPermission('contacts', 'write')
  )

  // Role-aware default correction. `loadStoredTab()` may run before the auth
  // session is restored (e.g. on cold load, when the contacts store is created
  // before `restoreSession()` resolves), so a manager with no explicit saved
  // preference can land on 'me' and see an empty list. Once the user object is
  // available, if the user never made an explicit choice (no stored tab), flip
  // managers to 'pending' (the queue they manage) and leave agents on 'me'.
  const hasExplicitTabChoice = typeof localStorage !== 'undefined'
    && !!localStorage.getItem('whatomate.chat.activeListTab')
  watch(() => authStore.user, (user) => {
    if (!user) return
    // A stored 'closed'/'all' preference is only honored for users who can
    // actually see those tabs (e.g. after a role downgrade); others fall back
    // to 'me' so they aren't stuck on an invisible tab.
    if (!canSeeSupervisorTabs.value
      && (activeListTab.value === 'closed' || activeListTab.value === 'all')) {
      activeListTab.value = 'me'
      return
    }
    if (hasExplicitTabChoice) return
    const isManager = authStore.hasPermission('contacts', 'write')
    if (isManager && activeListTab.value === 'me') {
      activeListTab.value = 'pending'
    } else if (!isManager && activeListTab.value === 'pending') {
      activeListTab.value = 'me'
    }
  }, { immediate: true })

  // Contacts pagination
  const contactsPage = ref(1)
  const contactsLimit = ref(50)
  const contactsTotal = ref(0)
  const isLoadingMoreContacts = ref(false)
  const hasMoreContacts = computed(() => contacts.value.length < contactsTotal.value)

  // Search is now driven server-side via fetchContacts({ search }), so the
  // visible list is whatever the server returned — no extra local filtering.
  const filteredContacts = computed(() => contacts.value)

  const sortedContacts = computed(() => {
    let list = [...filteredContacts.value]
    // Apply sidebar visibility filters. is_newsletter may be undefined for
    // older data — treat that as "not a newsletter" (falsy).
    if (hideGroupChats.value) {
      list = list.filter(c => !c.is_group_chat)
    }
    if (hideNewsletterChats.value) {
      list = list.filter(c => !c.is_newsletter)
    }
    return list.sort((a, b) => {
      const dateA = a.last_message_at ? new Date(a.last_message_at).getTime() : 0
      const dateB = b.last_message_at ? new Date(b.last_message_at).getTime() : 0
      return dateB - dateA
    })
  })

  // ─── Pending / Me / Closed / All tab membership (client-side filtering per D4) ───
  // Membership is derived from ASSIGNMENT (the source of truth) plus the explicit
  // `chat_status` for the closed-state check. The backend's EffectiveStatus()
  // defaults to "open" for legacy rows that never had chat_status set, so a
  // filter on `chat_status === 'pending'` alone would hide most legacy
  // unassigned chats. We therefore treat "pending" as `!assigned && !closed`.
  //   pending → not assigned to anyone AND not closed (awaiting a claim)
  //   me      → assigned to the current user (closed or not — owner sees their own)
  //   closed  → chat_status === 'closed' (supervisors only; legacy rows without
  //             chat_status default to open and correctly stay out of here)
  //   all     → every loaded chat, no filter (supervisors only — the backend
  //             already returns everything for contacts:read holders)
  const pendingContacts = computed(() =>
    sortedContacts.value.filter(c => !c.assigned_user_id && c.chat_status !== 'closed')
  )
  const myContacts = computed(() =>
    sortedContacts.value.filter(c => c.assigned_user_id === authStore.user?.id)
  )
  const closedContacts = computed(() =>
    sortedContacts.value.filter(c => c.chat_status === 'closed')
  )
  const allContacts = computed(() => sortedContacts.value)
  const pendingCount = computed(() => pendingContacts.value.length)
  const myCount = computed(() => myContacts.value.length)
  const closedCount = computed(() => closedContacts.value.length)
  const allCount = computed(() => allContacts.value.length)

  // The list to render in the sidebar for the active tab.
  const displayedContacts = computed(() => {
    switch (activeListTab.value) {
      case 'me': return myContacts.value
      case 'closed': return closedContacts.value
      case 'all': return allContacts.value
      case 'pending':
      default: return pendingContacts.value
    }
  })

  // ─── Cross-tab search fallback (M3) ───
  // When a search is active, the user is looking for a specific contact
  // regardless of which tab they are on. We search across ALL loaded contacts
  // (not just the active tab) and surface a hint when the query has hits in
  // other tabs but not the current one, so a "missing" result is explained.
  const searchResultsAcrossTabs = computed(() => {
    const q = normalizeContactSearch(searchQuery.value)
    if (!q) return null // no active search → caller should use displayedContacts
    const needle = q.toLowerCase()
    const matches = sortedContacts.value.filter(c =>
      (c.name?.toLowerCase().includes(needle)) ||
      (c.phone_number?.toLowerCase().includes(needle)) ||
      (c.assigned_user_name?.toLowerCase().includes(needle))
    )
    return matches
  })
  // Contacts to actually render when a search is active; otherwise the active
  // tab's list. The view consumes this single computed.
  const visibleContacts = computed(() => {
    if (searchQuery.value.trim()) {
      const r = searchResultsAcrossTabs.value
      if (r && r.length) return r
      return [] // search active, no matches
    }
    return displayedContacts.value
  })
  // True when the current search query has hits in OTHER tabs but not the
  // current one — drives the "found in: Me" hint in the view. The 'all' tab
  // shows every match by definition, so it never needs (or produces) a hint;
  // 'closed' participates only for supervisors who can see that tab.
  const searchHint = computed<{ show: boolean; tabs: ListTab[] } | null>(() => {
    const q = searchQuery.value.trim()
    if (!q) return null
    const r = searchResultsAcrossTabs.value ?? []
    if (!r.length) return null
    const inPending = r.some(c => !c.assigned_user_id && c.chat_status !== 'closed')
    const inMe = r.some(c => c.assigned_user_id === authStore.user?.id)
    const inClosed = canSeeSupervisorTabs.value && r.some(c => c.chat_status === 'closed')
    const current = activeListTab.value
    const currentHasHits =
      current === 'all' ||
      (current === 'pending' && inPending) ||
      (current === 'me' && inMe) ||
      (current === 'closed' && inClosed)
    if (currentHasHits) return null
    const tabs: ListTab[] = []
    if (inMe) tabs.push('me')
    if (inPending) tabs.push('pending')
    if (inClosed) tabs.push('closed')
    return { show: tabs.length > 0, tabs }
  })

  // ─── Chat lifecycle computed properties ───
  const pendingMessageCount = ref(0)

  const isPendingClaim = computed(() => {
    const c = currentContact.value
    if (!c) return false
    // Managers/admins can see any chat — no claim screen for them
    if (canManageAllChats.value) return false
    return c.chat_status === 'pending' && !c.assigned_user_id
  })

  const isChatClosed = computed(() => {
    const c = currentContact.value
    if (!c) return false
    return c.chat_status === 'closed'
  })

  // Managers/admins have contacts:write — they can see any chat without joining
  const canManageAllChats = computed(() =>
    authStore.hasPermission('contacts', 'write')
  )

  const isAssignedToMe = computed(() => {
    const c = currentContact.value
    if (!c) return false
    return c.assigned_user_id === authStore.user?.id
  })

  const isAssignedToOther = computed(() => {
    const c = currentContact.value
    if (!c || !c.assigned_user_id) return false
    return c.assigned_user_id !== authStore.user?.id
  })

  const isCollaborator = computed(() => {
    const c = currentContact.value
    if (!c) return false
    return c.collaborators?.some(collab => collab.user_id === authStore.user?.id) ?? false
  })

  const canViewMessages = computed(() =>
    isAssignedToMe.value || isCollaborator.value ||
    authStore.hasPermission('contacts', 'read') ||
    authStore.hasPermission('chat.collaborate', 'write')
  )

  const canCollaborate = computed(() =>
    authStore.hasPermission('chat.collaborate', 'write')
  )

  // Admins/managers ghost-view chats: they see content without being a
  // collaborator. Used to gate the Leave (ghost exit) button and ghost UI.
  const isAdminOrManager = computed(() =>
    authStore.hasPermission('contacts', 'write')
  )

  // The current user is the last remaining participant (owner + no other
  // collaborators). The Leave button label swaps to "Leave & Close" and the
  // action closes the conversation instead of orphaning it.
  const isLastParticipant = computed(() => {
    const c = currentContact.value
    if (!c) return false
    return isAssignedToMe.value && (c.collaborators?.length ?? 0) === 0
  })

  async function fetchContacts(params?: { search?: string; page?: number; limit?: number; tags?: string }) {
    isLoading.value = true
    try {
      const tagsParam = selectedTags.value.length > 0 ? selectedTags.value.join(',') : undefined
      const response = await contactsService.list({
        page: 1,
        limit: contactsLimit.value,
        tags: tagsParam,
        ...params
      })
      // API returns { status: "success", data: { contacts: [...], total: number } }
      const data = response.data.data || response.data
      contacts.value = data.contacts || []
      contactsTotal.value = data.total ?? contacts.value.length
      contactsPage.value = 1
    } catch (error) {
      console.error('Failed to fetch contacts:', error)
    } finally {
      isLoading.value = false
    }
  }

  async function loadMoreContacts() {
    if (isLoadingMoreContacts.value || !hasMoreContacts.value) return

    isLoadingMoreContacts.value = true
    try {
      const nextPage = contactsPage.value + 1
      const tagsParam = selectedTags.value.length > 0 ? selectedTags.value.join(',') : undefined
      const search = normalizeContactSearch(searchQuery.value) || undefined
      const response = await contactsService.list({
        page: nextPage,
        limit: contactsLimit.value,
        tags: tagsParam,
        search
      })
      const data = response.data.data || response.data
      const newContacts = data.contacts || []

      if (newContacts.length > 0) {
        // Append new contacts, avoiding duplicates
        const existingIds = new Set(contacts.value.map(c => c.id))
        const uniqueNew = newContacts.filter((c: Contact) => !existingIds.has(c.id))
        contacts.value = [...contacts.value, ...uniqueNew]
        contactsPage.value = nextPage
      }
      contactsTotal.value = data.total ?? contactsTotal.value
    } catch (error) {
      console.error('Failed to load more contacts:', error)
    } finally {
      isLoadingMoreContacts.value = false
    }
  }

  async function fetchContact(id: string) {
    try {
      const response = await contactsService.get(id)
      // API returns { status: "success", data: { ... } }
      const data = response.data.data || response.data
      currentContact.value = data
      return data
    } catch (error) {
      console.error('Failed to fetch contact:', error)
      return null
    }
  }

  async function fetchMessages(contactId: string, params?: { page?: number; limit?: number; account?: string }) {
    isLoadingMessages.value = true
    // Drop the previous contact's messages immediately so the list doesn't show
    // stale content under the new contact's header while the fetch is in flight.
    messages.value = []
    pendingMessageCount.value = 0

    // Skip the API call entirely for pending unclaimed chats — the privacy guard
    // would return 403, and the browser logs that as a console error. Instead,
    // read the unread count from the contact list item (already fetched).
    if (isPendingClaim.value) {
      pendingMessageCount.value = currentContact.value?.unread_count || 0
      isLoadingMessages.value = false
      return
    }

    try {
      // Default to the active account filter when a caller (lifecycle actions,
      // websocket re-fetches) doesn't pass one explicitly, so the live view
      // matches what a page refresh would render. The select-contact flow
      // clears the filter first, so it still fetches the unfiltered set.
      const account = params?.account ?? (accountFilter.value || undefined)
      const response = await messagesService.list(contactId, { ...params, account })
      // API returns { status: "success", data: { messages: [...], has_more: boolean } }
      const data = response.data.data || response.data
      messages.value = data.messages || []
      hasMoreMessages.value = data.has_more === true
      // A successful fetch marks the conversation read server-side
      // (GetMessages → markMessagesAsRead); mirror that on the sidebar badge.
      // Claim-gated chats never reach here (403 / early return above), so
      // their badge survives until the chat is actually claimed and read.
      const listEntry = contacts.value.find(c => c.id === contactId)
      if (listEntry) listEntry.unread_count = 0
      if (currentContact.value?.id === contactId) currentContact.value.unread_count = 0
    } catch (error: any) {
      // Privacy guard: conversation is pending and unclaimed
      if (error.response?.status === 403 && error.response.data?.code === 'chat_not_claimed') {
        pendingMessageCount.value = error.response.data.data?.pending_message_count || 0
        // messages.value stays empty → frontend shows claim screen
      } else {
        console.error('Failed to fetch messages:', error)
      }
    } finally {
      isLoadingMessages.value = false
    }
  }

  async function fetchOlderMessages(contactId: string, account?: string) {
    if (isLoadingOlderMessages.value || !hasMoreMessages.value || messages.value.length === 0) {
      return
    }

    isLoadingOlderMessages.value = true
    try {
      // Get the oldest message ID for cursor-based pagination
      const oldestMessageId = messages.value[0].id
      const response = await messagesService.list(contactId, { before_id: oldestMessageId, account })
      const data = response.data.data || response.data
      const olderMessages = data.messages || []

      if (olderMessages.length > 0) {
        // Prepend older messages (they come in chronological order, oldest first)
        messages.value = [...olderMessages, ...messages.value]
      }
      hasMoreMessages.value = data.has_more === true
    } catch (error) {
      console.error('Failed to fetch older messages:', error)
    } finally {
      isLoadingOlderMessages.value = false
    }
  }

  async function sendMessage(
    contactId: string,
    type: string,
    content: any,
    replyToMessageId?: string,
    whatsappAccount?: string,
    extra?: { interactive?: Parameters<typeof messagesService.send>[1]['interactive'] },
  ) {
    try {
      const response = await messagesService.send(contactId, {
        type,
        content,
        reply_to_message_id: replyToMessageId,
        whatsapp_account: whatsappAccount,
        ...(extra?.interactive ? { interactive: extra.interactive } : {}),
      })
      // API returns { status: "success", data: { ... } }
      const newMessage = response.data.data || response.data
      // Use addMessage which has duplicate checking (WebSocket may also broadcast this)
      addMessage(newMessage)

      return newMessage
    } catch (error) {
      console.error('Failed to send message:', error)
      throw error
    }
  }

  async function sendTemplate(
    contactId: string,
    templateName: string,
    templateParams?: Record<string, string>,
    accountName?: string,
    headerFile?: File,
    buttonParams?: Record<string, string>,
    headerParams?: Record<string, string>
  ) {
    try {
      const response = await messagesService.sendTemplate(contactId, {
        template_name: templateName,
        template_params: templateParams,
        header_params: headerParams,
        button_params: buttonParams,
        account_name: accountName
      }, headerFile)
      const data = response.data.data || response.data
      // Use addMessage which has duplicate checking (WebSocket may also broadcast this)
      addMessage(data)
      return data
    } catch (error) {
      console.error('Failed to send template:', error)
      throw error
    }
  }

  function setReplyingTo(message: Message | null) {
    replyingTo.value = message
  }

  function clearReplyingTo() {
    replyingTo.value = null
  }

  function addMessage(message: Message) {
    // Update contact metadata regardless of account filter
    const contact = contacts.value.find(c => c.id === message.contact_id)
    if (contact) {
      contact.last_message_at = message.created_at
      if (message.direction === 'incoming') {
        contact.unread_count++
        contact.last_inbound_at = message.created_at
      }
    }
    // Also update currentContact if it matches
    if (currentContact.value && currentContact.value.id === message.contact_id && message.direction === 'incoming') {
      currentContact.value.last_inbound_at = message.created_at
    }

    // Skip adding to messages array if account filter is active and doesn't match
    if (accountFilter.value && message.whatsapp_account && message.whatsapp_account !== accountFilter.value) {
      return
    }

    // Check if message already exists
    const exists = messages.value.some(m => m.id === message.id)
    if (!exists) {
      messages.value.push(message)
    }
  }

  function updateMessageStatus(messageId: string, status: string, errorMessage?: string) {
    const index = messages.value.findIndex(m => m.id === messageId)
    if (index !== -1) {
      messages.value[index] = {
        ...messages.value[index],
        status,
        ...(errorMessage ? { error_message: errorMessage } : {})
      }
    }
  }

  function setCurrentContact(contact: Contact | null) {
    currentContact.value = contact
    replyingTo.value = null // Clear reply state when switching contacts
    if (contact) {
      // Keep the unread badge for claim-gated conversations: the claim screen
      // hides the content, so nothing has actually been read yet. The badge
      // clears in fetchMessages once a successful (post-claim) fetch marks the
      // conversation read server-side.
      const claimGated =
        !canManageAllChats.value && contact.chat_status === 'pending' && !contact.assigned_user_id
      if (!claimGated) contact.unread_count = 0
      // Lazily fetch the contact's WhatsApp profile picture when it isn't
      // cached yet (e.g. the chat was created by an inbound message before a
      // GOWA contact sync). Best-effort: failures are swallowed so the chat
      // still opens — the initials fallback covers the no-avatar case.
      refreshAvatar(contact.id).catch(() => {})
    }
  }

  // refreshAvatar asks the backend to (re)fetch the contact's WhatsApp profile
  // picture and updates the cached contact rows with the returned avatar_url.
  // It returns the fetched URL (possibly empty when the contact has no
  // picture or no GOWA provider is available).
  async function refreshAvatar(contactId: string): Promise<string> {
    const res = await contactsService.refreshAvatar(contactId)
    const avatarUrl = (res.data as any)?.avatar_url ?? ''
    // Patch both the list entry and the active contact so the avatar appears
    // without a full refetch.
    const inList = contacts.value.find((c) => c.id === contactId)
    if (inList && inList.avatar_url !== avatarUrl) {
      inList.avatar_url = avatarUrl
    }
    if (currentContact.value?.id === contactId && currentContact.value.avatar_url !== avatarUrl) {
      currentContact.value = { ...currentContact.value, avatar_url: avatarUrl }
    }
    return avatarUrl
  }

  function setAccountFilter(account: string | null) {
    accountFilter.value = account
  }

  function clearMessages() {
    messages.value = []
    hasMoreMessages.value = false
    accountFilter.value = null
  }

  function updateMessageReactions(messageId: string, reactions: Reaction[]) {
    const message = messages.value.find(m => m.id === messageId)
    if (message) {
      message.reactions = reactions
    }
  }

  function updateContactTags(contactId: string, tags: string[]) {
    // Update in contacts list
    const contact = contacts.value.find(c => c.id === contactId)
    if (contact) {
      contact.tags = tags
    }
    // Update current contact if it matches
    if (currentContact.value?.id === contactId) {
      currentContact.value = { ...currentContact.value, tags }
    }
  }

  // Debounce server-side search so each keystroke doesn't fire a request.
  let searchDebounceHandle: ReturnType<typeof setTimeout> | null = null
  watch(searchQuery, (query) => {
    if (searchDebounceHandle) clearTimeout(searchDebounceHandle)
    searchDebounceHandle = setTimeout(() => {
      const search = normalizeContactSearch(query) || undefined
      fetchContacts({ search })
    }, 300)
  })

  // ─── Chat lifecycle actions ───
  async function claimChat(contactId: string) {
    const response = await api.put(`/contacts/${contactId}/claim`)
    const data = response.data.data || response.data
    // Update contact locally
    const contact = contacts.value.find(c => c.id === contactId)
    if (contact) {
      contact.assigned_user_id = authStore.user?.id
      contact.assigned_user_name = authStore.user?.full_name || ''
      contact.chat_status = 'open'
    }
    if (currentContact.value?.id === contactId) {
      currentContact.value.assigned_user_id = authStore.user?.id
      currentContact.value.assigned_user_name = authStore.user?.full_name || ''
      currentContact.value.chat_status = 'open'
      // Re-fetch messages to show the system message immediately
      await fetchMessages(contactId)
    }
    return data
  }

  // Release a conversation back to the pending pool. Mirrors claimChat's shape:
  // PUT, optimistic local update of status/assignment, then re-fetch messages.
  // Also clears collaborators locally (G4) — the backend wipes them on release
  // and broadcasts an empty collaborators array, but the optimistic update must
  // match so the UI does not flash stale collaborator avatars.
  async function releaseChat(contactId: string) {
    const response = await api.put(`/contacts/${contactId}/release`)
    const data = response.data.data || response.data
    const contact = contacts.value.find(c => c.id === contactId)
    if (contact) {
      contact.assigned_user_id = undefined
      contact.assigned_user_name = undefined
      contact.chat_status = 'pending'
      contact.collaborators = []
    }
    if (currentContact.value?.id === contactId) {
      currentContact.value.assigned_user_id = undefined
      currentContact.value.assigned_user_name = undefined
      currentContact.value.chat_status = 'pending'
      currentContact.value.collaborators = []
      // Re-fetch messages to show the system message immediately
      await fetchMessages(contactId)
    }
    return data
  }

  // Bulk release (M4). Releases many conversations back to pending in one call.
  // Server-side loop (the endpoint accepts an array of contact IDs) so we get a
  // single audit batch and one WS broadcast per chat. Used by the sidebar
  // multi-select mode. Returns per-id results so the UI can report partial fails.
  const bulkReleaseInProgress = ref(false)
  const selectedContactIds = ref<Set<string>>(new Set())
  const bulkSelectMode = ref(false)

  function toggleBulkSelect(contactId: string) {
    if (selectedContactIds.value.has(contactId)) {
      selectedContactIds.value.delete(contactId)
    } else {
      selectedContactIds.value.add(contactId)
    }
    // Trigger reactivity for Set mutation.
    selectedContactIds.value = new Set(selectedContactIds.value)
  }
  function clearBulkSelection() {
    selectedContactIds.value = new Set()
    bulkSelectMode.value = false
  }

  async function bulkReleaseChats(contactIds: string[]) {
    if (!contactIds.length) return { released: 0, failed: 0 }
    bulkReleaseInProgress.value = true
    try {
      const response = await api.post('/contacts/bulk-release', { contact_ids: contactIds })
      const data = response.data.data || response.data
      const releasedIds: string[] = data?.released_ids || data?.released || []
      // Optimistically apply the same local mutation as releaseChat for each
      // successfully released chat. Failed ones keep their current state.
      for (const id of releasedIds) {
        const c = contacts.value.find(x => x.id === id)
        if (c) {
          c.assigned_user_id = undefined
          c.assigned_user_name = undefined
          c.chat_status = 'pending'
          c.collaborators = []
        }
      }
      clearBulkSelection()
      return {
        released: releasedIds.length,
        failed: contactIds.length - releasedIds.length,
      }
    } finally {
      bulkReleaseInProgress.value = false
    }
  }

  async function joinChat(contactId: string) {
    const response = await api.post(`/contacts/${contactId}/join`)
    const data = response.data.data || response.data
    // Re-fetch contact to get updated collaborators
    await fetchContact(contactId)
    await fetchMessages(contactId)
    return data
  }

  async function leaveChat(contactId: string) {
    await api.delete(`/contacts/${contactId}/join`)
    // Remove self from collaborators locally
    const userId = authStore.user?.id
    if (currentContact.value?.id === contactId && currentContact.value.collaborators) {
      currentContact.value.collaborators = currentContact.value.collaborators.filter(
        c => c.user_id !== userId
      )
    }
    const contact = contacts.value.find(c => c.id === contactId)
    if (contact?.collaborators) {
      contact.collaborators = contact.collaborators.filter(c => c.user_id !== userId)
    }
  }

  async function closeChat(contactId: string) {
    await api.put(`/contacts/${contactId}/close`)
    // Update locally
    const contact = contacts.value.find(c => c.id === contactId)
    if (contact) {
      contact.chat_status = 'closed'
    }
    if (currentContact.value?.id === contactId) {
      currentContact.value.chat_status = 'closed'
      // Re-fetch messages to show the system message immediately
      await fetchMessages(contactId)
    }
  }

  // Reopen a closed conversation. Admins/managers only (backend enforces
  // contacts:write). Unlike claim, this does NOT assign ownership — it just
  // flips status back to 'open' so content is viewable/sendable again.
  async function reopenChat(contactId: string) {
    await api.put(`/contacts/${contactId}/reopen`)
    const contact = contacts.value.find(c => c.id === contactId)
    if (contact) {
      contact.chat_status = 'open'
    }
    if (currentContact.value?.id === contactId) {
      currentContact.value.chat_status = 'open'
      await fetchMessages(contactId)
    }
  }

  async function removeCollaborator(contactId: string, userId: string) {
    await api.delete(`/contacts/${contactId}/collaborators/${userId}`)
    // Update locally
    if (currentContact.value?.id === contactId && currentContact.value.collaborators) {
      currentContact.value.collaborators = currentContact.value.collaborators.filter(
        c => c.user_id !== userId
      )
    }
  }

  async function inviteCollaborator(contactId: string, userId: string) {
    await api.post(`/contacts/${contactId}/collaborators/${userId}`)
    // Re-fetch contact to get updated collaborators
    await fetchContact(contactId)
  }

  return {
    contacts,
    currentContact,
    messages,
    isLoading,
    isLoadingMessages,
    isLoadingOlderMessages,
    hasMoreMessages,
    searchQuery,
    selectedTags,
    hideGroupChats,
    hideNewsletterChats,
    replyingTo,
    filteredContacts,
    sortedContacts,
    // Pending / Me / Closed / All tabs
    activeListTab,
    canSeeSupervisorTabs,
    pendingContacts,
    myContacts,
    closedContacts,
    allContacts,
    pendingCount,
    myCount,
    closedCount,
    allCount,
    displayedContacts,
    // Cross-tab search (M3)
    visibleContacts,
    searchHint,
    // Bulk release (M4)
    bulkSelectMode,
    selectedContactIds,
    bulkReleaseInProgress,
    toggleBulkSelect,
    clearBulkSelection,
    bulkReleaseChats,
    // Chat lifecycle
    pendingMessageCount,
    isPendingClaim,
    isChatClosed,
    canManageAllChats,
    isAssignedToMe,
    isAssignedToOther,
    isCollaborator,
    canViewMessages,
    canCollaborate,
    isAdminOrManager,
    isLastParticipant,
    claimChat,
    releaseChat,
    closeChat,
    reopenChat,
    joinChat,
    leaveChat,
    removeCollaborator,
    inviteCollaborator,
    // Contacts pagination
    contactsTotal,
    hasMoreContacts,
    isLoadingMoreContacts,
    fetchContacts,
    loadMoreContacts,
    // Other
    fetchContact,
    fetchMessages,
    fetchOlderMessages,
    sendMessage,
    sendTemplate,
    addMessage,
    updateMessageStatus,
    setCurrentContact,
    refreshAvatar,
    clearMessages,
    setAccountFilter,
    setReplyingTo,
    clearReplyingTo,
    updateMessageReactions,
    updateContactTags
  }
})
