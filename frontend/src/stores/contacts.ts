import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { contactsService, chatsService, messagesService } from '@/services/api'
import { useAuthStore } from '@/stores/auth'

export type ChatStatus = 'pending' | 'open' | 'closed'
export type ChatBucketTab = 'pending' | 'assigned'

export interface Contact {
  id: string
  phone_number: string
  instance_id?: string
  conversation_id?: string
  is_group_chat?: boolean
  name: string
  profile_name?: string
  avatar_url?: string
  status: ChatStatus
  tags: string[]
  metadata: Record<string, any>
  last_message_at?: string
  last_message_preview?: string
  last_inbound_at?: string
  service_window_open?: boolean
  unread_count: number
  assigned_user_id?: string
  assigned_user_name?: string
  is_public?: boolean
  closed_at?: string
  closed_by_user_id?: string
  closed_by_name?: string
  whatsapp_account?: string
  created_at: string
  updated_at: string
}

export interface ReplyPreview {
  id: string
  content: any
  message_type: string
  direction: 'incoming' | 'outgoing'
  sender_phone?: string
}

export interface Reaction {
  emoji: string
  from_phone?: string
  from_user?: string
}

export interface Message {
  id: string
  contact_id: string
  conversation_id?: string
  is_group_chat?: boolean
  sender_phone?: string
  sender_push_name?: string
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
  instance_id?: string
  metadata?: Record<string, any>
  whatsapp_account?: string
  created_at: string
  updated_at: string
}

interface AddMessageOptions {
  appendToActiveThread?: boolean
}

export type ChatTypeFilter = 'private' | 'group' | 'channel'

const unsupportedMessageBody = '[Unsupported message type]'
const deletedMessageBody = '(This message was deleted)'
const legacyDeletedMessageBody = 'This message was deleted'
const syntheticPlaceholderCompanionWindowMs = 3000

function normalizeChatStatus(rawStatus: unknown, assignedUserID?: string): ChatStatus {
  const normalized = typeof rawStatus === 'string' ? rawStatus.trim().toLowerCase() : ''
  if (normalized === 'closed') return 'closed'
  if (normalized === 'open') return 'open'
  if (normalized === 'pending') return assignedUserID ? 'open' : 'pending'
  return assignedUserID ? 'open' : 'pending'
}

function normalizeContact(contact: Contact): Contact {
  return {
    ...contact,
    is_public: contact.is_public === true,
    status: normalizeChatStatus(contact.status, contact.assigned_user_id)
  }
}

function normalizeContacts(contacts: Contact[]): Contact[] {
  return contacts.map(normalizeContact)
}

function normalizeSearchText(value: unknown): string {
  return typeof value === 'string' ? value.trim().toLowerCase() : ''
}

function normalizeDigits(value: string): string {
  let normalized = ''

  for (const char of value) {
    const code = char.charCodeAt(0)

    if (code >= 0x30 && code <= 0x39) {
      normalized += char
      continue
    }

    if (code >= 0x0660 && code <= 0x0669) {
      normalized += String(code - 0x0660)
      continue
    }

    if (code >= 0x06F0 && code <= 0x06F9) {
      normalized += String(code - 0x06F0)
    }
  }

  return normalized
}

function contactMatchesSearch(contact: Contact, rawQuery: string): boolean {
  const query = normalizeSearchText(rawQuery)
  if (!query) return true

  const name = normalizeSearchText(contact.name)
  const profileName = normalizeSearchText(contact.profile_name)
  const phoneNumber = normalizeSearchText(contact.phone_number)

  if (
    name.includes(query) ||
    profileName.includes(query) ||
    phoneNumber.includes(query)
  ) {
    return true
  }

  const queryDigits = normalizeDigits(rawQuery)
  if (!queryDigits) {
    return false
  }

  return normalizeDigits(contact.phone_number || '').includes(queryDigits)
}

function extractAllowedInstanceIDsFromUserSettings(settings: unknown): string[] {
  if (!settings || typeof settings !== 'object') return []

  const sendRestrictions = (settings as Record<string, unknown>).send_restrictions
  if (!sendRestrictions || typeof sendRestrictions !== 'object') return []

  const raw = sendRestrictions as Record<string, unknown>
  const allowedInstanceIDs = raw.allowed_instance_ids
  if (Array.isArray(allowedInstanceIDs)) {
    return Array.from(new Set(allowedInstanceIDs
      .map((value) => typeof value === 'string' ? value.trim() : '')
      .filter(Boolean)))
  }

  const allowedInstanceID = raw.allowed_instance_id
  if (typeof allowedInstanceID !== 'string') return []

  const trimmed = allowedInstanceID.trim()
  return trimmed ? [trimmed] : []
}

function getMessageBody(message: Message): string {
  if (typeof message.content === 'string') {
    return message.content
  }
  return typeof message.content?.body === 'string' ? message.content.body : ''
}

function isPlaceholderMessageBody(body: string): boolean {
  const normalized = body.trim()
  return normalized === unsupportedMessageBody ||
    normalized === deletedMessageBody ||
    normalized.toLowerCase() === legacyDeletedMessageBody.toLowerCase()
}

function isSyntheticPlaceholderMessage(message: Message): boolean {
  if (message.message_type !== 'text') {
    return false
  }
  if (message.metadata?.revoked === true) {
    return false
  }
  return isPlaceholderMessageBody(getMessageBody(message))
}

function isUnsupportedPlaceholderMessage(message: Message): boolean {
  return isSyntheticPlaceholderMessage(message) && getMessageBody(message).trim() === unsupportedMessageBody
}

function isGroupMessage(message: Message): boolean {
  return message.is_group_chat === true || message.metadata?.is_group_chat === true
}

function isMediaLikeMessage(message: Message): boolean {
  const messageType = (message.message_type || '').toLowerCase()
  return messageType === 'image' ||
    messageType === 'video' ||
    messageType === 'audio' ||
    messageType === 'document' ||
    messageType === 'sticker'
}

function getMessageSenderPhone(message: Message): string {
  if (typeof message.sender_phone === 'string' && message.sender_phone.trim() !== '') {
    return message.sender_phone.trim()
  }
  if (typeof message.metadata?.sender_phone === 'string' && message.metadata.sender_phone.trim() !== '') {
    return message.metadata.sender_phone.trim()
  }
  return ''
}

function getMessageTimestamp(message: Message): number {
  if (typeof message.created_at !== 'string') return Number.NaN
  const parsed = Date.parse(message.created_at)
  return Number.isNaN(parsed) ? Number.NaN : parsed
}

function collectNearbyMediaCompanionPlaceholderIDs(messageList: Message[]): Set<string> {
  const ids = new Set<string>()
  for (let i = 0; i < messageList.length; i++) {
    const candidate = messageList[i]
    if (!candidate?.id || !isUnsupportedPlaceholderMessage(candidate) || !isGroupMessage(candidate)) {
      continue
    }

    const candidateSender = getMessageSenderPhone(candidate)
    if (candidateSender === '') {
      continue
    }

    const candidateTimestamp = getMessageTimestamp(candidate)
    if (!Number.isFinite(candidateTimestamp)) {
      continue
    }

    for (let j = i + 1; j < messageList.length; j++) {
      const next = messageList[j]
      if (!next) continue

      const nextTimestamp = getMessageTimestamp(next)
      if (!Number.isFinite(nextTimestamp)) {
        continue
      }

      if (nextTimestamp - candidateTimestamp > syntheticPlaceholderCompanionWindowMs) {
        break
      }

      if (!isMediaLikeMessage(next)) {
        continue
      }

      if (next.contact_id !== candidate.contact_id || next.direction !== candidate.direction) {
        continue
      }

      if (getMessageSenderPhone(next) !== candidateSender) {
        continue
      }

      ids.add(candidate.id)
      break
    }
  }
  return ids
}

function removeSyntheticPlaceholderMessages(messageList: Message[]): Message[] {
  const companionWamids = new Set(
    messageList
      .filter(message => !isSyntheticPlaceholderMessage(message) && typeof message.wamid === 'string' && message.wamid.trim() !== '')
      .map(message => message.wamid!.trim())
  )
  const nearbyMediaCompanionPlaceholderIDs = collectNearbyMediaCompanionPlaceholderIDs(messageList)

  if (companionWamids.size === 0 && nearbyMediaCompanionPlaceholderIDs.size === 0) {
    return messageList
  }

  return messageList.filter(message => {
    if (message?.id && nearbyMediaCompanionPlaceholderIDs.has(message.id)) {
      return false
    }
    const wamid = typeof message.wamid === 'string' ? message.wamid.trim() : ''
    if (wamid === '' || !companionWamids.has(wamid)) {
      return true
    }
    return !isSyntheticPlaceholderMessage(message)
  })
}

export const useContactsStore = defineStore('contacts', () => {
  const authStore = useAuthStore()
  const contacts = ref<Contact[]>([])
  const pendingChats = ref<Contact[]>([])
  const assignedChats = ref<Contact[]>([])
  const closedChats = ref<Contact[]>([])
  const activeChatTab = ref<ChatBucketTab>('pending')
  const currentContact = ref<Contact | null>(null)
  const messages = ref<Message[]>([])
  const isLoading = ref(false)
  const isLoadingMessages = ref(false)
  const isLoadingOlderMessages = ref(false)
  const isMessageAccessRestricted = ref(false)
  const hasMoreMessages = ref(false)
  let messageFetchSequence = 0
  let latestMessageFetchSequence = 0
  const searchQuery = ref('')
  const selectedTags = ref<string[]>([])
  const selectedInstanceId = ref('')
  const selectedChatTypes = ref<ChatTypeFilter[]>([])
  const replyingTo = ref<Message | null>(null)
  const accountFilter = ref<string | null>(null)

  // Contacts pagination
  const contactsPage = ref(1)
  const contactsLimit = ref(50)
  const contactsTotal = ref(0)
  const pendingChatsTotal = ref(0)
  const assignedChatsTotal = ref(0)
  const isLoadingMoreContacts = ref(false)
  const restrictedAllowedInstanceIDs = computed(() =>
    extractAllowedInstanceIDsFromUserSettings(authStore.user?.settings)
  )
  const effectiveInstanceFilterID = computed(() => {
    const selected = selectedInstanceId.value.trim()
    if (selected !== '') {
      return selected
    }
    return restrictedAllowedInstanceIDs.value.length === 1
      ? restrictedAllowedInstanceIDs.value[0]
      : ''
  })
  const isAdminOrSuperAdmin = computed(() => {
    if (authStore.user?.is_super_admin === true) return true
    return (authStore.userRole || '').toLowerCase() === 'admin'
  })
  const hasMoreContacts = computed(() => {
    const activeCount = activeChatTab.value === 'assigned'
      ? assignedChats.value.length
      : pendingChats.value.length
    return activeCount < contactsTotal.value
  })

  function rebuildChatBucketsFromContacts() {
    pendingChats.value = contacts.value.filter(c => c.status === 'pending' && !c.assigned_user_id)
    assignedChats.value = contacts.value.filter(c => c.status === 'open' && !!c.assigned_user_id)
  }

  function mergeContactsIntoStore(nextContacts: Contact[]) {
    const normalized = normalizeContacts(nextContacts)
    const merged = new Map<string, Contact>()

    for (const existing of contacts.value) {
      merged.set(existing.id, existing)
    }

    for (const next of normalized) {
      const current = merged.get(next.id)
      merged.set(next.id, current ? { ...current, ...next } : next)
    }

    contacts.value = Array.from(merged.values())
    rebuildChatBucketsFromContacts()
  }

  function upsertContact(contact: Contact) {
    mergeContactsIntoStore([contact])
    if (currentContact.value?.id === contact.id) {
      currentContact.value = {
        ...currentContact.value,
        ...normalizeContact(contact)
      }
    }
  }

  function replaceContacts(nextContacts: Contact[]) {
    contacts.value = normalizeContacts(nextContacts)
    rebuildChatBucketsFromContacts()
  }

  const activeTabContacts = computed(() => {
    if (activeChatTab.value === 'assigned') return assignedChats.value
    return pendingChats.value
  })

  const searchedContacts = computed(() => {
    const trimmedQuery = searchQuery.value.trim()
    if (!trimmedQuery) return activeTabContacts.value

    // While searching from /chat, include recently-fetched closed chats so
    // agents can locate and open historical conversations without switching pages.
    const merged = new Map<string, Contact>()
    for (const contact of activeTabContacts.value) {
      merged.set(contact.id, contact)
    }
    for (const contact of closedChats.value) {
      merged.set(contact.id, contact)
    }

    return Array.from(merged.values()).filter(c => contactMatchesSearch(c, trimmedQuery))
  })

  function getConversationId(contact: Contact): string {
    return (
      contact.conversation_id ||
      (typeof contact.metadata?.group_jid === 'string' ? contact.metadata.group_jid : '') ||
      (typeof contact.metadata?.channel_jid === 'string' ? contact.metadata.channel_jid : '') ||
      (contact.phone_number.endsWith('@g.us') || contact.phone_number.endsWith('@newsletter') ? contact.phone_number : '')
    )
  }

  function isGroupConversation(contact: Contact): boolean {
    const conversationId = getConversationId(contact)
    return Boolean(
      contact.is_group_chat === true ||
      contact.metadata?.is_group_chat === true ||
      (conversationId && conversationId.endsWith('@g.us'))
    )
  }

  function matchesActiveFilters(contact: Contact): boolean {
    if (effectiveInstanceFilterID.value && contact.instance_id !== effectiveInstanceFilterID.value) {
      return false
    }
    if (!effectiveInstanceFilterID.value && restrictedAllowedInstanceIDs.value.length > 0) {
      const instanceID = typeof contact.instance_id === 'string' ? contact.instance_id.trim() : ''
      if (!instanceID || !restrictedAllowedInstanceIDs.value.includes(instanceID)) {
        return false
      }
    }

    return true
  }

  const filteredContacts = computed(() => {
    const grouped = new Map<string, Contact>()
    for (const contact of searchedContacts.value) {
      if (!matchesActiveFilters(contact)) {
        continue
      }

      const conversationId = getConversationId(contact)
      const isGroupChat = isGroupConversation(contact)

      const groupKey = isGroupChat && conversationId
        ? `group:${conversationId}:${contact.instance_id || 'no-instance'}`
        : `contact:${contact.id}`

      const existing = grouped.get(groupKey)
      if (!existing) {
        grouped.set(groupKey, { ...contact })
        continue
      }

      const existingTime = existing.last_message_at ? new Date(existing.last_message_at).getTime() : 0
      const contactTime = contact.last_message_at ? new Date(contact.last_message_at).getTime() : 0
      const latest = contactTime >= existingTime ? contact : existing

      grouped.set(groupKey, {
        ...existing,
        ...latest,
        conversation_id: conversationId || latest.conversation_id || existing.conversation_id,
        is_group_chat: isGroupChat || latest.is_group_chat || existing.is_group_chat,
        is_public: existing.is_public === true || latest.is_public === true || contact.is_public === true,
        unread_count: (existing.unread_count || 0) + (contact.unread_count || 0),
        tags: Array.from(new Set([...(existing.tags || []), ...(contact.tags || [])])),
      })
    }

    return Array.from(grouped.values())
  })

  const sortedContacts = computed(() => {
    return [...filteredContacts.value].sort((a, b) => {
      const publicA = a.is_public === true ? 1 : 0
      const publicB = b.is_public === true ? 1 : 0
      if (publicA !== publicB) {
        return publicB - publicA
      }
      const dateA = a.last_message_at ? new Date(a.last_message_at).getTime() : 0
      const dateB = b.last_message_at ? new Date(b.last_message_at).getTime() : 0
      return dateB - dateA
    })
  })

  function setActiveChatTab(tab: ChatBucketTab) {
    activeChatTab.value = tab
    contactsPage.value = 1
    contactsTotal.value = tab === 'assigned'
      ? assignedChatsTotal.value
      : pendingChatsTotal.value
  }

  function buildListParams() {
    return {
      tags: selectedTags.value.length > 0 ? selectedTags.value.join(',') : undefined,
      instance_id: effectiveInstanceFilterID.value || undefined,
      chat_types: selectedChatTypes.value.length > 0 ? selectedChatTypes.value.join(',') : undefined
    }
  }

  async function fetchContacts(params?: {
    search?: string
    page?: number
    limit?: number
    tags?: string
    instance_id?: string
    chat_types?: string
    status?: ChatStatus
    assigned_to?: 'me' | 'unassigned' | string
  }) {
    isLoading.value = true
    try {
      const response = await contactsService.list({
        ...buildListParams(),
        page: params?.page ?? 1,
        limit: contactsLimit.value,
        ...params
      })
      const data = response.data.data || response.data
      replaceContacts(data.contacts || [])
      contactsTotal.value = data.total ?? contacts.value.length
      pendingChatsTotal.value = pendingChats.value.length
      assignedChatsTotal.value = assignedChats.value.length
      contactsPage.value = params?.page ?? 1
    } catch (error) {
      console.error('Failed to fetch contacts:', error)
    } finally {
      isLoading.value = false
    }
  }

  async function fetchChats(params?: {
    search?: string
    limit?: number
  }) {
    isLoading.value = true
    try {
      const trimmedSearch = typeof params?.search === 'string'
        ? params.search.trim()
        : ''
      const includeClosedInSearch = trimmedSearch !== ''
      const closedSearchLimit = Math.max(params?.limit ?? contactsLimit.value, 500)
      const listParams = {
        ...buildListParams(),
        search: trimmedSearch || undefined,
        page: 1,
        limit: params?.limit ?? contactsLimit.value
      }

      const [pendingResponse, assignedResponse, closedResponse] = await Promise.all([
        chatsService.list({
          ...listParams,
          status: 'pending'
        }),
        chatsService.list({
          ...listParams,
          status: 'open'
        }),
        includeClosedInSearch
          ? chatsService.list({
            ...listParams,
            limit: closedSearchLimit,
            status: 'closed'
          })
          : Promise.resolve(null)
      ])

      const pendingData = pendingResponse.data.data || pendingResponse.data
      const assignedData = assignedResponse.data.data || assignedResponse.data
      const pendingList = normalizeContacts(pendingData.contacts || [])
      const assignedList = normalizeContacts(assignedData.contacts || [])
      pendingChatsTotal.value = pendingData.total ?? pendingList.length
      assignedChatsTotal.value = assignedData.total ?? assignedList.length
      const searchedClosed = includeClosedInSearch && closedResponse
        ? normalizeContacts((closedResponse.data.data || closedResponse.data).contacts || [])
        : null
      if (searchedClosed) {
        closedChats.value = searchedClosed
      }

      // Preserve already-fetched closed chats while refreshing active buckets.
      const retainedClosed = searchedClosed ?? contacts.value.filter(c => c.status === 'closed')
      const merged = new Map<string, Contact>()
      for (const item of retainedClosed) {
        merged.set(item.id, item)
      }
      for (const item of [...pendingList, ...assignedList]) {
        merged.set(item.id, item)
      }

      contacts.value = Array.from(merged.values())
      rebuildChatBucketsFromContacts()
      contactsTotal.value = activeChatTab.value === 'assigned'
        ? assignedChatsTotal.value
        : pendingChatsTotal.value
      contactsPage.value = 1
    } catch (error) {
      console.error('Failed to fetch chats:', error)
    } finally {
      isLoading.value = false
    }
  }

  async function fetchPendingChats(params?: { search?: string; limit?: number }) {
    isLoading.value = true
    try {
      const response = await chatsService.list({
        ...buildListParams(),
        search: params?.search,
        page: 1,
        limit: params?.limit ?? contactsLimit.value,
        status: 'pending'
      })
      const data = response.data.data || response.data
      const nextPending = normalizeContacts(data.contacts || [])

      mergeContactsIntoStore(nextPending)
      pendingChats.value = nextPending
      pendingChatsTotal.value = data.total ?? nextPending.length
      if (activeChatTab.value === 'pending') {
        contactsTotal.value = pendingChatsTotal.value
      }
      return nextPending
    } catch (error) {
      console.error('Failed to fetch pending chats:', error)
      return []
    } finally {
      isLoading.value = false
    }
  }

  async function fetchAssignedChats(params?: { search?: string; limit?: number; assigned_to?: 'me' | string }) {
    isLoading.value = true
    try {
      const response = await chatsService.list({
        ...buildListParams(),
        search: params?.search,
        page: 1,
        limit: params?.limit ?? contactsLimit.value,
        status: 'open',
        assigned_to: params?.assigned_to
      })
      const data = response.data.data || response.data
      const nextAssigned = normalizeContacts(data.contacts || [])

      mergeContactsIntoStore(nextAssigned)
      assignedChats.value = nextAssigned
      assignedChatsTotal.value = data.total ?? nextAssigned.length
      if (activeChatTab.value === 'assigned') {
        contactsTotal.value = assignedChatsTotal.value
      }
      return nextAssigned
    } catch (error) {
      console.error('Failed to fetch assigned chats:', error)
      return []
    } finally {
      isLoading.value = false
    }
  }
  async function loadMoreContacts() {
    if (isLoadingMoreContacts.value || !hasMoreContacts.value) return

    isLoadingMoreContacts.value = true
    try {
      const nextPage = contactsPage.value + 1
      const response = await chatsService.list({
        ...buildListParams(),
        search: searchQuery.value || undefined,
        page: nextPage,
        limit: contactsLimit.value,
        status: activeChatTab.value === 'assigned' ? 'open' : 'pending'
      })
      const data = response.data.data || response.data
      const newContacts = normalizeContacts(data.contacts || [])

      if (newContacts.length > 0) {
        mergeContactsIntoStore(newContacts)
        contactsPage.value = nextPage
      }
      if (activeChatTab.value === 'assigned') {
        assignedChatsTotal.value = data.total ?? assignedChatsTotal.value
      } else {
        pendingChatsTotal.value = data.total ?? pendingChatsTotal.value
      }
      contactsTotal.value = activeChatTab.value === 'assigned'
        ? assignedChatsTotal.value
        : pendingChatsTotal.value
    } catch (error) {
      console.error('Failed to load more contacts:', error)
    } finally {
      isLoadingMoreContacts.value = false
    }
  }

  async function fetchContact(id: string) {
    try {
      const response = await contactsService.get(id)
      const data = normalizeContact(response.data.data || response.data)
      upsertContact(data)
      currentContact.value = data
      return data
    } catch (error) {
      console.error('Failed to fetch contact:', error)
      return null
    }
  }

  async function fetchClosedChats(params?: {
    search?: string
    page?: number
    limit?: number
    closed_by?: string
    closed_from?: string
    closed_to?: string
  }) {
    isLoading.value = true
    try {
      const response = await chatsService.list({
        ...buildListParams(),
        search: params?.search,
        page: params?.page ?? 1,
        limit: params?.limit ?? contactsLimit.value,
        status: 'closed',
        closed_by: params?.closed_by,
        closed_from: params?.closed_from,
        closed_to: params?.closed_to
      })
      const data = response.data.data || response.data
      const nextClosed = normalizeContacts(data.contacts || [])

      mergeContactsIntoStore(nextClosed)
      closedChats.value = nextClosed
      return {
        chats: nextClosed,
        total: data.total ?? nextClosed.length,
        page: data.page ?? (params?.page ?? 1),
        limit: data.limit ?? (params?.limit ?? contactsLimit.value)
      }
    } catch (error) {
      console.error('Failed to fetch closed chats:', error)
      closedChats.value = []
      return {
        chats: [],
        total: 0,
        page: params?.page ?? 1,
        limit: params?.limit ?? contactsLimit.value
      }
    } finally {
      isLoading.value = false
    }
  }

  async function claimChat(chatId: string) {
    const response = await chatsService.claim(chatId)
    const updated = normalizeContact((response.data.data || response.data) as Contact)
    upsertContact(updated)
    isMessageAccessRestricted.value = false
    return updated
  }

  async function closeChat(chatId: string) {
    const response = await chatsService.close(chatId)
    const updated = normalizeContact((response.data.data || response.data) as Contact)
    upsertContact(updated)
    return updated
  }

  async function reopenChat(chatId: string) {
    const response = await chatsService.reopen(chatId)
    const updated = normalizeContact((response.data.data || response.data) as Contact)
    upsertContact(updated)
    return updated
  }

  async function setChatPublic(chatId: string, isPublic: boolean) {
    const response = await chatsService.setPublic(chatId, isPublic)
    const updated = normalizeContact((response.data.data || response.data) as Contact)
    upsertContact(updated)
    return updated
  }

  async function fetchMessages(contactId: string, params?: { page?: number; limit?: number; account?: string }) {
    const requestSequence = ++messageFetchSequence
    latestMessageFetchSequence = requestSequence
    isLoadingMessages.value = true
    isMessageAccessRestricted.value = false
    // Prevent stale thread content from staying visible while switching chats.
    messages.value = []
    hasMoreMessages.value = false
    try {
      const response = await chatsService.listMessages(contactId, params)
      if (requestSequence !== latestMessageFetchSequence) {
        return
      }
      const data = response.data.data || response.data
      messages.value = removeSyntheticPlaceholderMessages(data.messages || [])
      hasMoreMessages.value = data.has_more === true
      const contact = contacts.value.find(c => c.id === contactId)
      if (contact) {
        contact.unread_count = 0
      }
      if (currentContact.value?.id === contactId) {
        currentContact.value.unread_count = 0
      }
    } catch (error: any) {
      if (requestSequence !== latestMessageFetchSequence) {
        return
      }
      if (error?.response?.status === 403) {
        messages.value = []
        hasMoreMessages.value = false
        isMessageAccessRestricted.value = true
      }
      console.error('Failed to fetch messages:', error)
    } finally {
      if (requestSequence === latestMessageFetchSequence) {
        isLoadingMessages.value = false
      }
    }
  }

  async function fetchOlderMessages(contactId: string, account?: string) {
    if (isMessageAccessRestricted.value || isLoadingOlderMessages.value || !hasMoreMessages.value || messages.value.length === 0) {
      return
    }

    isLoadingOlderMessages.value = true
    try {
      // Get the oldest message ID for cursor-based pagination
      const oldestMessageId = messages.value[0].id
      const response = await chatsService.listMessages(contactId, { before_id: oldestMessageId, account })
      const data = response.data.data || response.data
      const olderMessages = data.messages || []
      if (currentContact.value?.id !== contactId) {
        return
      }

      if (olderMessages.length > 0) {
        // Prepend older messages (they come in chronological order, oldest first)
        messages.value = removeSyntheticPlaceholderMessages([...olderMessages, ...messages.value])
      }
      hasMoreMessages.value = data.has_more === true
    } catch (error) {
      console.error('Failed to fetch older messages:', error)
    } finally {
      isLoadingOlderMessages.value = false
    }
  }

  async function sendMessage(contactId: string, type: string, content: any, replyToMessageId?: string, whatsappAccount?: string) {
    try {
      const contact = contacts.value.find(item => item.id === contactId)
      const response = await messagesService.send(contactId, {
        type,
        content,
        reply_to_message_id: replyToMessageId,
        instance_id: contact?.instance_id,
        whatsapp_account: whatsappAccount
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
    accountName?: string
  ) {
    try {
      const response = await messagesService.sendTemplate(contactId, {
        template_name: templateName,
        template_params: templateParams,
        account_name: accountName
      })
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

  function addMessage(message: Message, options: AddMessageOptions = {}): boolean {
    const { appendToActiveThread = true } = options

    // Update contact metadata regardless of account filter.
 
    const contact = contacts.value.find(c => c.id === message.contact_id)
    if (contact) {
      contact.last_message_at = message.created_at
      if (message.direction === 'incoming') {
        contact.unread_count++
        contact.last_inbound_at = message.created_at
        contact.service_window_open = true
      }
    }
    // Also update currentContact if it matches
    if (currentContact.value && currentContact.value.id === message.contact_id && message.direction === 'incoming') {
      currentContact.value.last_inbound_at = message.created_at
      currentContact.value.service_window_open = true
    }

    // Skip adding to messages array if account filter is active and doesn't match
    if (accountFilter.value && message.whatsapp_account && message.whatsapp_account !== accountFilter.value) {
      return false
    }

    const existingIndex = messages.value.findIndex(m => m.id === message.id)
    if (existingIndex !== -1) {
      if (appendToActiveThread) {
        messages.value[existingIndex] = { ...messages.value[existingIndex], ...message }
        messages.value = removeSyntheticPlaceholderMessages(messages.value)
      }
      return false
    }

    if (appendToActiveThread) {
      messages.value.push(message)
      messages.value = removeSyntheticPlaceholderMessages(messages.value)
    }

    return true
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

  function patchMessage(updatedMessage: Message) {
    const index = messages.value.findIndex(m => m.id === updatedMessage.id)
    if (index !== -1) {
      messages.value[index] = { ...messages.value[index], ...updatedMessage }
      messages.value = removeSyntheticPlaceholderMessages(messages.value)
    }

    const contact = contacts.value.find(c => c.id === updatedMessage.contact_id)
    if (contact) {
      const existingLastAt = contact.last_message_at ? new Date(contact.last_message_at).getTime() : 0
      const updatedLastAt = updatedMessage.created_at ? new Date(updatedMessage.created_at).getTime() : 0
      if (updatedLastAt >= existingLastAt) {
        contact.last_message_at = updatedMessage.created_at
        contact.last_message_preview = getMessageBody(updatedMessage)
      }
    }
  }

  function patchContact(updatedContact: Partial<Contact> & { id: string }) {
    const normalizedPartial: Partial<Contact> & { id: string } = { ...updatedContact }
    if (updatedContact.status !== undefined || updatedContact.assigned_user_id !== undefined) {
      normalizedPartial.status = normalizeChatStatus(updatedContact.status, updatedContact.assigned_user_id)
    }

    const index = contacts.value.findIndex(c => c.id === updatedContact.id)
    if (index !== -1) {
      contacts.value[index] = {
        ...contacts.value[index],
        ...normalizedPartial
      }
      rebuildChatBucketsFromContacts()
    }

    if (currentContact.value?.id === updatedContact.id) {
      currentContact.value = {
        ...currentContact.value,
        ...normalizedPartial
      }
    }
  }

  function setCurrentContact(contact: Contact | null) {
    currentContact.value = contact ? normalizeContact(contact) : null
    replyingTo.value = null // Clear reply state when switching contacts
    isMessageAccessRestricted.value = false
    if (currentContact.value) {
      currentContact.value.unread_count = 0
    }
  }

  async function markConversationAsRead(contactId: string) {
    if (!contactId) return

    try {
      // Server marks this conversation as read when listing messages.
      await messagesService.list(contactId, { limit: 1 })
    } catch (error) {
      console.error('Failed to mark conversation as read:', error)
    }

    const contact = contacts.value.find(c => c.id === contactId)
    if (contact) {
      contact.unread_count = 0
    }
    if (currentContact.value?.id === contactId) {
      currentContact.value.unread_count = 0
    }
  }

  function setAccountFilter(account: string | null) {
    accountFilter.value = account
  }

  function clearMessages() {
    latestMessageFetchSequence = ++messageFetchSequence
    messages.value = []
    hasMoreMessages.value = false
    isMessageAccessRestricted.value = false
    isLoadingMessages.value = false
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

  return {
    contacts,
    pendingChats,
    assignedChats,
    closedChats,
    activeChatTab,
    currentContact,
    messages,
    isLoading,
    isLoadingMessages,
    isLoadingOlderMessages,
    isMessageAccessRestricted,
    hasMoreMessages,
    searchQuery,
    selectedTags,
    selectedInstanceId,
    selectedChatTypes,
    replyingTo,
    filteredContacts,
    sortedContacts,
    // Contacts pagination
    contactsTotal,
    hasMoreContacts,
    isLoadingMoreContacts,
    setActiveChatTab,
    fetchContacts,
    fetchChats,
    fetchPendingChats,
    fetchAssignedChats,
    fetchClosedChats,
    loadMoreContacts,
    // Other
    fetchContact,
    fetchMessages,
    fetchOlderMessages,
    claimChat,
    closeChat,
    reopenChat,
    setChatPublic,
    sendMessage,
    sendTemplate,
    addMessage,
    updateMessageStatus,
    patchMessage,
    patchContact,
    setCurrentContact,
    clearMessages,
    setAccountFilter,
    setReplyingTo,
    clearReplyingTo,
    markConversationAsRead,
    updateMessageReactions,
    updateContactTags
  }
})
