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
  service_window_open?: boolean
  unread_count: number
  assigned_user_id?: string
  assigned_user_name?: string
  whatsapp_account?: string
  marketing_opt_out?: boolean
  is_group_chat?: boolean
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
  const replyingTo = ref<Message | null>(null)
  const accountFilter = ref<string | null>(null)

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
    return [...filteredContacts.value].sort((a, b) => {
      const dateA = a.last_message_at ? new Date(a.last_message_at).getTime() : 0
      const dateB = b.last_message_at ? new Date(b.last_message_at).getTime() : 0
      return dateB - dateA
    })
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
      const response = await messagesService.list(contactId, params)
      // API returns { status: "success", data: { messages: [...], has_more: boolean } }
      const data = response.data.data || response.data
      messages.value = data.messages || []
      hasMoreMessages.value = data.has_more === true
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
      contact.unread_count = 0
    }
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
    replyingTo,
    filteredContacts,
    sortedContacts,
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
    clearMessages,
    setAccountFilter,
    setReplyingTo,
    clearReplyingTo,
    updateMessageReactions,
    updateContactTags
  }
})
