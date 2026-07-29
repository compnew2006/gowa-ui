import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { scheduledMessagesService, type ScheduledMessage, type ScheduledMessagePayload } from '@/services/api'

// Holds the pending scheduled messages for the contact currently open in the
// chat view. Fired/failed/cancelled rows drop out of the list (the panel only
// manages upcoming sends); WS events keep it live across clients.
export const useScheduledMessagesStore = defineStore('scheduledMessages', () => {
  const items = ref<ScheduledMessage[]>([])
  const isLoading = ref(false)
  const currentContactId = ref<string | null>(null)

  const pendingCount = computed(() => items.value.length)

  function sortBySchedule() {
    items.value.sort((a, b) => a.scheduled_at.localeCompare(b.scheduled_at))
  }

  // Helper: insert only if not already present (WS may race the HTTP response)
  function pushIfNew(sm: ScheduledMessage) {
    if (sm.status === 'pending' && !items.value.some(m => m.id === sm.id)) {
      items.value.push(sm)
      sortBySchedule()
    }
  }

  async function fetchForContact(contactId: string) {
    isLoading.value = true
    currentContactId.value = contactId
    try {
      const response = await scheduledMessagesService.listForContact(contactId, { status: 'pending', limit: 100 })
      const data = (response.data as any).data || response.data
      items.value = data.scheduled_messages || []
      sortBySchedule()
    } catch {
      items.value = []
    } finally {
      isLoading.value = false
    }
  }

  async function schedule(contactId: string, payload: ScheduledMessagePayload) {
    const response = await scheduledMessagesService.create(contactId, payload)
    const sm: ScheduledMessage = (response.data as any).data || response.data
    if (contactId === currentContactId.value) {
      pushIfNew(sm)
    }
    return sm
  }

  async function updateSchedule(id: string, data: { content?: { body?: string }; scheduled_at?: string }) {
    const response = await scheduledMessagesService.update(id, data)
    const updated: ScheduledMessage = (response.data as any).data || response.data
    const index = items.value.findIndex(m => m.id === id)
    if (index !== -1) {
      items.value[index] = updated
      sortBySchedule()
    }
    return updated
  }

  async function cancel(id: string) {
    await scheduledMessagesService.cancel(id)
    items.value = items.value.filter(m => m.id !== id)
  }

  // WebSocket event handlers (backend broadcasts only to clients viewing the contact)
  function onCreated(sm: ScheduledMessage) {
    if (sm.contact_id !== currentContactId.value) return
    pushIfNew(sm)
  }

  function onUpdated(sm: ScheduledMessage) {
    if (sm.contact_id !== currentContactId.value) return
    const index = items.value.findIndex(m => m.id === sm.id)
    if (sm.status !== 'pending') {
      // Fired, failed or cancelled — no longer an upcoming send.
      if (index !== -1) items.value.splice(index, 1)
      return
    }
    if (index !== -1) {
      items.value[index] = sm
      sortBySchedule()
    } else {
      pushIfNew(sm)
    }
  }

  function clear() {
    items.value = []
    currentContactId.value = null
  }

  return {
    items,
    isLoading,
    currentContactId,
    pendingCount,
    fetchForContact,
    schedule,
    updateSchedule,
    cancel,
    onCreated,
    onUpdated,
    clear
  }
})
