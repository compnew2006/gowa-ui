import { ref, nextTick, onUnmounted } from 'vue'
import type { ComputedRef, Ref } from 'vue'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'
import { contactsService } from '@/services/api'

export interface UseChatScrollOptions {
  /** Current contact id getter — drives read-receipts on focus. */
  getFirstUnreadId: () => string | null
  /** Current contact id getter (for markRead). */
  getCurrentContactId: () => string | undefined
  /** Selected account ref — used by the messages infinite scroll loader. */
  selectedAccount: { value: string | null }
  /** Contacts store reactive surface. */
  contactsStore: {
    currentContact: { id: string } | null
    fetchOlderMessages: (id: string, account?: string) => Promise<void>
  }
  /** Reactive hasMore flag for messages (top) infinite scroll. */
  hasMoreMessages: ComputedRef<boolean> | Ref<boolean>
  /** Reactive isLoading flag for messages (top) infinite scroll. */
  isLoadingOlderMessages: ComputedRef<boolean> | Ref<boolean>
  /** DOM anchor for scroll-to-bottom. Owned by the view (template ref). */
  messagesEndRef: Ref<HTMLElement | null>
}

/**
 * Scrolling, sticky-date header, unread-pill, and "load older messages"
 * infinite scroll for the chat room. Owns the messagesScroll instance and the
 * DOM anchor refs the template binds to.
 *
 * @example
 * ```ts
 * const scroll = useChatScroll({ contactsStore, ... })
 * onMounted(() => scroll.setup())
 * ```
 */
export function useChatScroll(options: UseChatScrollOptions) {
  const { contactsStore } = options

  // ─── State ───
  // Tracks incoming messages that arrived while the chat is open.
  // Surfaced as a "N unread messages" pill at the top of the chat panel
  // (WhatsApp-style). Click the pill to jump up to the first message of
  // the unread batch; cleared on click or contact switch. See issue #280.
  const newMessagesCount = ref(0)
  const firstUnreadId = ref<string | null>(null)
  const isAtBottom = ref(true)
  const SCROLL_BOTTOM_THRESHOLD = 80

  // Sticky date header
  const stickyDate = ref('')
  const showStickyDate = ref(false)
  let stickyDateTimeout: ReturnType<typeof setTimeout> | null = null

  // DOM anchor owned by the view.
  const messagesEndRef = options.messagesEndRef

  // ─── Scroll primitives ───
  function scrollToBottom(instant = false) {
    nextTick(() => {
      if (messagesEndRef.value) {
        messagesEndRef.value.scrollIntoView({
          behavior: instant ? 'instant' : 'smooth',
          block: 'end'
        })
      }
    })
  }

  function scrollToMessage(messageId: string | undefined) {
    if (!messageId) return
    const messageEl = document.getElementById(`message-${messageId}`)
    if (messageEl) {
      messageEl.scrollIntoView({ behavior: 'smooth', block: 'center' })
      messageEl.classList.add('highlight-message')
      setTimeout(() => messageEl.classList.remove('highlight-message'), 2000)
    }
  }

  function updateAtBottom(el: HTMLElement) {
    const distanceFromBottom = el.scrollHeight - el.clientHeight - el.scrollTop
    isAtBottom.value = distanceFromBottom < SCROLL_BOTTOM_THRESHOLD
  }

  function updateStickyDate(scrollContainer: HTMLElement) {
    // Find all date separator elements
    const dateSeparators = scrollContainer.querySelectorAll('[data-date-separator]')
    if (dateSeparators.length === 0) return

    const containerRect = scrollContainer.getBoundingClientRect()
    const containerTop = containerRect.top + 60 // Offset for sticky header position

    // Find the last date separator that's above the viewport top
    let currentDate = ''
    for (const separator of dateSeparators) {
      const rect = separator.getBoundingClientRect()
      if (rect.top < containerTop) {
        currentDate = separator.getAttribute('data-date-separator') || ''
      } else {
        break
      }
    }

    // Show sticky date if we have scrolled past at least one date separator
    if (currentDate && scrollContainer.scrollTop > 50) {
      stickyDate.value = currentDate
      showStickyDate.value = true

      // Hide after scrolling stops
      if (stickyDateTimeout) clearTimeout(stickyDateTimeout)
      stickyDateTimeout = setTimeout(() => {
        showStickyDate.value = false
      }, 1500)
    } else {
      showStickyDate.value = false
    }
  }

  // ─── Infinite scroll (load older messages at top) ───
  const messagesScroll = useInfiniteScroll({
    direction: 'top',
    onLoadMore: async () => {
      if (!contactsStore.currentContact) return
      await messagesScroll.preserveScrollPosition(async () => {
        await contactsStore.fetchOlderMessages(contactsStore.currentContact!.id, options.selectedAccount.value || undefined)
        await nextTick()
      })
    },
    hasMore: options.hasMoreMessages,
    isLoading: options.isLoadingOlderMessages,
    onScroll: (event: Event) => {
      const el = event.target as HTMLElement
      updateStickyDate(el)
      updateAtBottom(el)
    }
  })

  // ─── "Agent returned" → mark read + jump to unread divider ───
  function onUserActive() {
    if (document.visibilityState !== 'visible' || !document.hasFocus()) return
    if (!options.getFirstUnreadId()) return
    const contactId = options.getCurrentContactId()
    if (contactId) {
      contactsService.markRead(contactId).catch(() => { /* non-critical */ })
    }
    nextTick(() => {
      const unreadId = options.getFirstUnreadId()
      const el = unreadId ? document.getElementById(`message-${unreadId}`) : null
      if (el) {
        el.scrollIntoView({ behavior: 'smooth', block: 'start' })
      }
    })
  }

  // ─── New-message watcher logic (called by the caller's watcher) ───
  /**
   * Decides what to do when the messages array grows: pile up an unread pill
   * when the agent is away, otherwise auto-scroll if at the bottom.
   * Returns true if the caller should stop (handled the "away" branch).
   */
  function handleMessagesLengthChange(newLen: number, oldLen: number, latestIsIncoming: boolean): boolean {
    if (newLen <= oldLen) return false
    // "Not actively looking" covers both other-tab (hidden) and other-window
    // (visible but unfocused). The divider should pile in either case.
    const userAway = typeof document !== 'undefined'
      && (document.visibilityState === 'hidden' || !document.hasFocus())
    if (latestIsIncoming && userAway) {
      // Caller sets firstUnreadId on the first unread of the batch
      newMessagesCount.value += 1
      return true
    }
    // Outgoing (the agent replied) — they've seen the unread, drop the divider.
    if (!latestIsIncoming && newMessagesCount.value > 0) {
      newMessagesCount.value = 0
      firstUnreadId.value = null
    }
    if (isAtBottom.value || !latestIsIncoming) {
      scrollToBottom()
    }
    return false
  }

  function clearUnread() {
    newMessagesCount.value = 0
    firstUnreadId.value = null
  }

  function resetOnContactSwitch() {
    newMessagesCount.value = 0
    firstUnreadId.value = null
    isAtBottom.value = true
    messagesScroll.cleanup()
  }

  onUnmounted(() => {
    if (stickyDateTimeout) clearTimeout(stickyDateTimeout)
  })

  return {
    // State
    newMessagesCount,
    firstUnreadId,
    isAtBottom,
    stickyDate,
    showStickyDate,
    // messagesEndRef is owned by the view (passed in) — not re-returned.
    // Scroll instance (template binds messagesScroll.scrollAreaRef)
    messagesScroll,
    // Imperative
    scrollToBottom,
    scrollToMessage,
    updateStickyDate,
    updateAtBottom,
    onUserActive,
    // Watcher helpers
    handleMessagesLengthChange,
    clearUnread,
    resetOnContactSwitch,
  }
}
