import { ref, computed, nextTick } from 'vue'
import type { Ref } from 'vue'
import { toast } from 'vue-sonner'
import { customActionsService, accountsService, type CustomAction, type ActionResult } from '@/services/api'
import { wsService } from '@/services/websocket'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'
import { STATUS_VIRTUAL_CONTACT, isStatusContact } from '@/lib/status'
import {
  Ticket, User, BarChart, Link, Phone, Mail, FileText, ExternalLink, Zap, Globe, Code,
} from 'lucide-vue-next'
import type { Contact, Message } from '@/stores/contacts'

type ListTab = 'me' | 'pending' | 'closed' | 'all'

export interface UseChatContactsListOptions {
  /** i18n translator. */
  t: (key: string, params?: Record<string, unknown>) => string
  /** Contacts store reactive surface (passed in by the view). */
  contactsStore: {
    currentContact: Contact | null
    setCurrentContact: (c: Contact | null) => void
    clearMessages: () => void
    fetchContacts: () => Promise<void>
    fetchContact: (id: string) => Promise<Contact>
    fetchMessages: (id: string, opts?: { account?: string }) => Promise<void>
    setAccountFilter: (account: string | null) => void
    contacts: Contact[]
    messages: Message[]
    selectedTags: string[]
    activeListTab: ListTab
    canSeeSupervisorTabs: boolean
    myCount: number
    pendingCount: number
    closedCount: number
    allCount: number
    hasMoreContacts: boolean
    isLoadingMoreContacts: boolean
    loadMoreContacts: () => Promise<void>
    hasMoreMessages: boolean
    isLoadingOlderMessages: boolean
    fetchOlderMessages: (id: string, account?: string) => Promise<void>
  }
  /** Called when a contact is clicked in the list (the view navigates via router). */
  onContactClick: (contact: Contact) => void
  /** Called when a contact is selected (the view wires notes/scheduled fetches + scroll setup). */
  onContactSelected: (id: string, contact: Contact) => Promise<void> | void
  /** Shared multi-account selection ref (owned by the view so every composable
   * can read/write it without an ordering dependency). */
  selectedAccount: { value: string | null }
  /** Messages infinite-scroll surface (owned by useChatScroll). */
  messagesScroll: {
    cleanup: () => void
    setup: () => void
  }
  /** Reset unread-pill state on contact switch (owned by useChatScroll). */
  resetUnreadOnSwitch: () => void
  /** Called to scroll the room to the bottom after a contact switch. */
  scrollToBottom: (instant?: boolean) => void
  /** Tab strip template ref. Owned by the view (template ref). */
  tabStripRef: Ref<HTMLElement | null>
}

/**
 * Contacts sidebar: tab navigation, tag filter, multi-account tabs, custom
 * actions, infinite-scroll (load more contacts), and the cross-contact
 * `selectContact` orchestration (account discovery + auto-select + WebSocket
 * binding). Owns the selectedAccount ref shared by the rest of the chat view.
 *
 * @example
 * ```ts
 * const list = useChatContactsList({ contactsStore, t, ... })
 * watch(contactId, (id) => id && list.selectContact(id))
 * ```
 */
export function useChatContactsList(options: UseChatContactsListOptions) {
  const { t, contactsStore } = options

  // ─── Multi-account state ───
  // selectedAccount is owned by the view and passed in so every composable can
  // read it. contactAccounts/orgAccounts stay local (only this list uses them).
  const selectedAccount = options.selectedAccount
  const contactAccounts = ref<string[]>([])
  const orgAccounts = ref<any[]>([])

  // ─── Tag filter state ───
  const isTagFilterOpen = ref(false)

  // ─── Custom actions state ───
  const customActions = ref<CustomAction[]>([])
  const executingActionId = ref<string | null>(null)

  // Icon mapping for custom actions
  const actionIconMap: Record<string, any> = {
    'ticket': Ticket,
    'user': User,
    'bar-chart': BarChart,
    'link': Link,
    'phone': Phone,
    'mail': Mail,
    'file-text': FileText,
    'external-link': ExternalLink,
    'zap': Zap,
    'globe': Globe,
    'code': Code,
  }

  function getActionIcon(iconName: string) {
    return actionIconMap[iconName] || Zap
  }

  async function fetchCustomActions() {
    try {
      const response = await customActionsService.list()
      const data = (response.data as any).data || response.data
      customActions.value = (data.custom_actions || []).filter((a: CustomAction) => a.is_active)
    } catch (error) {
      // Silently fail - custom actions are optional
      console.error('Failed to fetch custom actions:', error)
    }
  }

  function toggleTagFilter(tagName: string) {
    const index = contactsStore.selectedTags.indexOf(tagName)
    if (index === -1) {
      contactsStore.selectedTags.push(tagName)
    } else {
      contactsStore.selectedTags.splice(index, 1)
    }
    // Refetch contacts with new filter
    contactsStore.fetchContacts()
  }

  function clearTagFilter() {
    contactsStore.selectedTags = []
    contactsStore.fetchContacts()
  }

  async function executeCustomAction(action: CustomAction) {
    if (!contactsStore.currentContact || executingActionId.value) return

    executingActionId.value = action.id
    try {
      const response = await customActionsService.execute(action.id, contactsStore.currentContact.id)
      let result: ActionResult = (response.data as any).data || response.data

      // JavaScript actions are now executed server-side via goja.
      // The response already contains structured result fields (toast, clipboard, redirect_url, message).

      // Handle different result types
      if (result.redirect_url) {
        // Open URL action result - prepend base path for relative URLs
        let redirectUrl = result.redirect_url
        if (redirectUrl.startsWith('/api/')) {
          const basePath = ((window as any).__BASE_PATH__ ?? '').replace(/\/$/, '')
          redirectUrl = basePath + redirectUrl
        }
        try {
          const parsed = new URL(redirectUrl, window.location.origin)
          if (parsed.protocol === 'http:' || parsed.protocol === 'https:') {
            window.open(parsed.href, '_blank')
          }
        } catch {
          // Invalid URL, ignore
        }
      }

      if (result.clipboard) {
        // Copy to clipboard
        await navigator.clipboard.writeText(result.clipboard)
        toast.success(t('common.copiedToClipboard'))
      }

      if (result.toast) {
        // Show toast notification
        if (result.toast.type === 'success') {
          toast.success(result.toast.message)
        } else if (result.toast.type === 'error') {
          toast.error(result.toast.message)
        } else {
          toast.info(result.toast.message)
        }
      } else if (result.success && !result.redirect_url && !result.clipboard) {
        // Default success message
        toast.success(result.message || t('chat.actionExecuted'))
      } else if (!result.success) {
        toast.error(result.message || t('chat.actionFailed'))
      }
    } catch (error: any) {
      const message = error.response?.data?.message || 'Failed to execute action'
      toast.error(message)
    } finally {
      executingActionId.value = null
    }
  }

  // ─── Tab keyboard navigation (M5 a11y) ───
  // Arrow Left/Right move focus between tabs and activate them, mirroring the
  // WAI-ARIA tabs pattern. Home/End jump to first/last. The roving tabindex on
  // the buttons themselves handles the rest. tabStripRef is owned by the view.
  const tabStripRef = options.tabStripRef
  const TAB_ORDER = ['me', 'pending', 'closed', 'all'] as const
  // 'closed' and 'all' are supervisor tabs (contacts:write — the admin/manager
  // marker, matching canManageAllChats), so agents keep the two-tab strip.
  function visibleTabOrder(): ListTab[] {
    return contactsStore.canSeeSupervisorTabs
      ? [...TAB_ORDER]
      : ['me', 'pending']
  }
  function onTabKeydown(e: KeyboardEvent) {
    const order = visibleTabOrder()
    const idx = order.indexOf(contactsStore.activeListTab as any)
    if (idx === -1) return
    let next = idx
    if (e.key === 'ArrowRight') next = (idx + 1) % order.length
    else if (e.key === 'ArrowLeft') next = (idx - 1 + order.length) % order.length
    else if (e.key === 'Home') next = 0
    else if (e.key === 'End') next = order.length - 1
    else return
    e.preventDefault()
    contactsStore.activeListTab = order[next]
    nextTick(() => {
      const el = tabStripRef.value?.querySelector<HTMLButtonElement>(`#tab-${order[next]}`)
      el?.focus()
    })
  }
  function tabLabel(tab: ListTab): string {
    switch (tab) {
      case 'pending': return t('chat.tabPending')
      case 'closed': return t('chat.tabClosed')
      case 'all': return t('chat.tabAll')
      default: return t('chat.tabMe')
    }
  }
  function tabCount(tab: ListTab): number {
    switch (tab) {
      case 'pending': return contactsStore.pendingCount
      case 'closed': return contactsStore.closedCount
      case 'all': return contactsStore.allCount
      default: return contactsStore.myCount
    }
  }

  // ─── Infinite scroll for contacts (load more at bottom) ───
  const contactsScroll = useInfiniteScroll({
    direction: 'bottom',
    onLoadMore: () => contactsStore.loadMoreContacts(),
    hasMore: computed(() => contactsStore.hasMoreContacts),
    isLoading: computed(() => contactsStore.isLoadingMoreContacts),
  })

  async function switchAccount(accountName: string) {
    if (!contactsStore.currentContact || accountName === selectedAccount.value) return
    selectedAccount.value = accountName
    contactsStore.setAccountFilter(accountName)
    await contactsStore.fetchMessages(contactsStore.currentContact.id, { account: accountName })
    await nextTick()
    options.scrollToBottom(true)
  }

  function handleContactClick(contact: Contact) {
    options.onContactClick(contact)
  }

  /**
   * Select a contact by id: discover its accounts, auto-select the most likely
   * sending account, bind the WebSocket "current contact", and hand control
   * back to the view for notes/scheduled fetches + scroll setup.
   */
  async function selectContact(id: string) {
    // The virtual Status conversation is send-only and has no backend row:
    // short-circuit the normal fetch/contact-discovery/account-detection flow.
    if (isStatusContact(id)) {
      options.resetUnreadOnSwitch()
      selectedAccount.value = null
      contactAccounts.value = []
      contactsStore.setAccountFilter(null)
      contactsStore.setCurrentContact({ ...STATUS_VIRTUAL_CONTACT } as Contact)
      await contactsStore.fetchMessages(id)
      // Status needs a sending account — default to the first org account.
      if (orgAccounts.value.length > 0) {
        selectedAccount.value = orgAccounts.value[0].name
      }
      wsService.setCurrentContact(id)
      options.onContactSelected(id, contactsStore.currentContact as Contact)
      return
    }
    // Direct deep links to /chat/:id may target a contact that isn't in the
    // currently-loaded (paginated) list — fall back to fetching it directly.
    let contact = contactsStore.contacts.find(c => c.id === id)
    if (!contact) {
      contact = await contactsStore.fetchContact(id)
    }
    if (contact) {
      options.resetUnreadOnSwitch()

      // Reset account selection when switching contacts
      selectedAccount.value = null
      contactAccounts.value = []
      contactsStore.setAccountFilter(null)

      contactsStore.setCurrentContact(contact)
      await contactsStore.fetchMessages(id)

      // Discover distinct accounts from the unfiltered message set
      const accounts = new Set<string>()
      for (const msg of contactsStore.messages) {
        if (msg.whatsapp_account) accounts.add(msg.whatsapp_account)
      }
      contactAccounts.value = Array.from(accounts).sort()

      // Auto-select account and filter client-side (avoids a second fetch)
      if (orgAccounts.value.length > 1) {
        // Find account of the most recent incoming message
        for (let i = contactsStore.messages.length - 1; i >= 0; i--) {
          const msg = contactsStore.messages[i]
          if (msg.direction === 'incoming' && msg.whatsapp_account) {
            selectedAccount.value = msg.whatsapp_account
            break
          }
        }
        // Fallback to contact's default account, then first org account
        if (!selectedAccount.value) {
          selectedAccount.value = contact.whatsapp_account || contactAccounts.value[0] || orgAccounts.value[0]?.name
        }
        if (selectedAccount.value) {
          contactsStore.setAccountFilter(selectedAccount.value)
          // Filter messages client-side instead of re-fetching. System messages
          // (claim/close/release/reopen) carry no whatsapp_account — keep them so
          // lifecycle events don't disappear when an account is selected.
          contactsStore.messages = contactsStore.messages.filter(
            (m: any) => m.whatsapp_account === selectedAccount.value || m.metadata?.is_system_message
          )
        }
      } else if (contactAccounts.value.length === 1) {
        selectedAccount.value = contactAccounts.value[0]
      } else if (contact.whatsapp_account) {
        selectedAccount.value = contact.whatsapp_account
      }

      // Tell WebSocket server which contact we're viewing
      wsService.setCurrentContact(id)
      options.onContactSelected(id, contact)
    }
  }

  async function fetchOrgAccounts() {
    try {
      const res = await accountsService.list()
      orgAccounts.value = res.data.data?.accounts || []
    } catch {
      orgAccounts.value = []
    }
  }

  return {
    // Multi-account (selectedAccount is owned by the view; not re-returned)
    contactAccounts,
    orgAccounts,
    // Tag filter
    isTagFilterOpen,
    toggleTagFilter,
    clearTagFilter,
  // Custom actions
  customActions,
  executingActionId,
  getActionIcon,
  fetchCustomActions,
  executeCustomAction,
  // Tabs (tabStripRef owned by the view — not re-returned)
  visibleTabOrder,
  onTabKeydown,
  tabLabel,
  tabCount,
    // Contacts infinite scroll
    contactsScroll,
    // Actions
    switchAccount,
    handleContactClick,
    selectContact,
    fetchOrgAccounts,
  }
}
