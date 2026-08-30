import { onUnmounted, watch, type Ref } from 'vue'
import { messagesService } from '@/services/api'

export interface UseChatLiveProbeOptions {
  /** Contacts store reactive surface. */
  contactsStore: {
    currentContact: { id: string } | null
    messages: Array<{ id: string }>
    isLoadingMessages: boolean
    fetchMessages: (id: string, opts?: { account?: string }) => Promise<void>
  }
  /** Active account filter, owned by the view (same source the chat list uses). */
  selectedAccount: Ref<string | null>
}

/** Safety-net probe cadence: the WebSocket pushes instantly; this only fills
 *  the gap when a webhook or the socket is silently lost. */
export const LIVE_PROBE_INTERVAL_MS = 15_000

/**
 * Safety-net probe for the open conversation. New messages normally arrive via
 * the GOWA webhook → backend broadcast → WebSocket pipeline; when that pipeline
 * drops silently (missed webhook, dead socket), the open chat would otherwise
 * freeze until the periodic history sync (15 min) notices the gap.
 *
 * Every LIVE_PROBE_INTERVAL_MS while the tab is visible, fetch the newest
 * message only (limit 1, cheapest possible request) and compare its id with
 * the newest loaded one; on a change, re-fetch through the store's own
 * fetchMessages — the exact path a contact switch uses — so filtering,
 * read-state, and scroll behavior stay identical.
 */
export function useChatLiveProbe(options: UseChatLiveProbeOptions) {
  const { contactsStore, selectedAccount } = options
  let timer: ReturnType<typeof setInterval> | null = null
  let seenNewestId: string | null = null
  let refreshing = false

  async function probe() {
    const contact = contactsStore.currentContact
    // Skip when there is nothing to reconcile or the store is mid-flight
    // (its own fetch will land the newest message anyway).
    if (!contact || document.hidden || contactsStore.isLoadingMessages) return

    const account = selectedAccount.value || undefined
    try {
      const response = await messagesService.list(contact.id, { limit: 1, account })
      const data = response.data?.data || response.data
      const newestId = data?.messages?.[0]?.id as string | undefined
      if (!newestId) return

      // First probe after a contact switch only primes the baseline.
      if (seenNewestId === null) {
        seenNewestId = newestId
        return
      }
      if (newestId === seenNewestId) return
      seenNewestId = newestId

      if (refreshing) return
      refreshing = true
      try {
        // Re-check the contact: the user may have switched chats while the
        // probe request was in flight.
        if (contactsStore.currentContact?.id === contact.id) {
          await contactsStore.fetchMessages(contact.id, { account })
        }
      } finally {
        refreshing = false
      }
    } catch {
      // A single failed probe is expected during transient disconnects; the
      // next tick retries. Keep the baseline so a blip doesn't force a reload.
    }
  }

  // The baseline must reset on every contact switch, otherwise the first
  // probe of the next conversation would diff against the previous one.
  watch(() => contactsStore.currentContact?.id, () => {
    seenNewestId = null
  })

  timer = setInterval(probe, LIVE_PROBE_INTERVAL_MS)
  onUnmounted(() => {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  })
}
