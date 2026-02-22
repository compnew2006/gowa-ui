<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { messagesService, notificationsService } from '@/services/api'
import { wsService } from '@/services/websocket'
import { useContactsStore, type Contact } from '@/stores/contacts'
import { useAuthStore } from '@/stores/auth'
import type { InstanceNotification } from '@/types/whatsmeow'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger
} from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Bell, Loader2, X } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const router = useRouter()
const contactsStore = useContactsStore()
const authStore = useAuthStore()
const open = ref(false)
const loading = ref(false)
const isMarkingAllRead = ref(false)
const isClearingAll = ref(false)
const notifications = ref<InstanceNotification[]>([])

const undismissedInstanceCount = computed(() => notifications.value.filter(item => !item.is_dismissed).length)
const unreadConversations = computed(() => {
  const unread = contactsStore.sortedContacts.filter(item => (item.unread_count || 0) > 0)
  const currentUserId = authStore.user?.id
  const isAgent = authStore.userRole === 'agent'

  if (!isAgent || !currentUserId) {
    return unread
  }

  return unread.filter(item => item.assigned_user_id === currentUserId)
})
const hasUnreadMessages = computed(() => unreadConversations.value.length > 0)
const hasSystemNotifications = computed(() => notifications.value.length > 0)
const unreadMessagesCount = computed(() =>
  unreadConversations.value.reduce((total, item) => total + (item.unread_count || 0), 0)
)
const totalUnreadCount = computed(() => undismissedInstanceCount.value + unreadMessagesCount.value)

function formatDate(value: string) {
  return new Date(value).toLocaleString()
}

function getContactLabel(contact: Contact): string {
  return contact.profile_name || contact.name || contact.phone_number
}

function openConversation(contactId: string) {
  open.value = false
  void contactsStore.markConversationAsRead(contactId)
  router.push(`/chat/${contactId}`)
}

async function fetchNotifications() {
  loading.value = true
  try {
    const response = await notificationsService.list()
    notifications.value = (response.data.data || response.data) as InstanceNotification[]
  } catch {
    // Silent fail for header widget
  } finally {
    loading.value = false
  }
}

async function dismissNotification(id: string) {
  try {
    await notificationsService.dismiss(id)
    notifications.value = notifications.value.filter(item => item.id !== id)
  } catch {
    toast.error('Failed to dismiss notification')
  }
}

async function markAllAsRead() {
  if (!hasUnreadMessages.value || isMarkingAllRead.value) return

  isMarkingAllRead.value = true
  const targetContacts = unreadConversations.value.map(item => item.id)
  const targetSet = new Set(targetContacts)

  try {
    // Fetching a contact's messages triggers server-side mark-as-read flow for that conversation.
    await Promise.all(targetContacts.map(contactId => messagesService.list(contactId, { limit: 1 })))

    for (const contact of contactsStore.contacts) {
      if (targetSet.has(contact.id)) {
        contact.unread_count = 0
      }
    }
    if (contactsStore.currentContact && targetSet.has(contactsStore.currentContact.id)) {
      contactsStore.currentContact.unread_count = 0
    }

    toast.success('All chats marked as read')
  } catch {
    toast.error('Failed to mark all chats as read')
  } finally {
    isMarkingAllRead.value = false
  }
}

async function clearAllNotifications() {
  if (!hasSystemNotifications.value || isClearingAll.value) return

  isClearingAll.value = true
  const ids = notifications.value.map(item => item.id)
  const failedIds: string[] = []

  try {
    await Promise.all(ids.map(async id => {
      try {
        await notificationsService.dismiss(id)
      } catch {
        failedIds.push(id)
      }
    }))

    if (failedIds.length === 0) {
      notifications.value = []
      toast.success('All notifications cleared')
      return
    }

    notifications.value = notifications.value.filter(item => failedIds.includes(item.id))
    toast.error(`Failed to clear ${failedIds.length} notification${failedIds.length > 1 ? 's' : ''}`)
  } finally {
    isClearingAll.value = false
  }
}

function handleInstanceNotification() {
  fetchNotifications()
}

onMounted(() => {
  fetchNotifications()
  if (contactsStore.contacts.length === 0) {
    contactsStore.fetchContacts()
  }
  wsService.subscribe('instance_notification', handleInstanceNotification)
})

onUnmounted(() => {
  wsService.unsubscribe('instance_notification', handleInstanceNotification)
})
</script>

<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        variant="ghost"
        size="icon"
        class="relative h-8 w-8 text-white/70 hover:text-white hover:bg-white/[0.08] light:text-gray-600 light:hover:text-gray-900 light:hover:bg-gray-100"
        aria-label="Notifications"
      >
        <Bell class="h-4 w-4" />
        <Badge
          v-if="totalUnreadCount > 0"
          class="absolute -top-1 -right-1 min-w-[18px] h-[18px] px-1 text-[10px] bg-red-500 text-white border-0 flex items-center justify-center"
        >
          {{ totalUnreadCount > 99 ? '99+' : totalUnreadCount }}
        </Badge>
      </Button>
    </PopoverTrigger>
    <PopoverContent
      align="end"
      class="w-[360px] max-w-[92vw] p-0 bg-[#141414] light:bg-white border-white/[0.08] light:border-gray-200 overflow-hidden"
    >
      <div class="px-4 py-3 border-b border-white/[0.08] light:border-gray-200">
        <h3 class="text-sm font-medium text-white light:text-gray-900">Notifications</h3>
      </div>
      <div class="px-3 py-2 border-b border-white/[0.08] light:border-gray-200 flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          class="h-7 px-2 text-[11px]"
          :disabled="!hasUnreadMessages || isMarkingAllRead"
          @click="markAllAsRead"
        >
          <Loader2 v-if="isMarkingAllRead" class="h-3 w-3 mr-1 animate-spin" />
          Mark all as read
        </Button>
        <Button
          variant="outline"
          size="sm"
          class="h-7 px-2 text-[11px]"
          :disabled="!hasSystemNotifications || isClearingAll"
          @click="clearAllNotifications"
        >
          <Loader2 v-if="isClearingAll" class="h-3 w-3 mr-1 animate-spin" />
          Clear all
        </Button>
      </div>

      <div v-if="loading && notifications.length === 0 && unreadConversations.length === 0" class="flex justify-center items-center py-8">
        <Loader2 class="h-5 w-5 animate-spin text-white/40 light:text-gray-400" />
      </div>

      <div v-else-if="notifications.length === 0 && unreadConversations.length === 0" class="px-4 py-8 text-center text-sm text-white/50 light:text-gray-500">
        No notifications
      </div>

      <ScrollArea v-else class="h-[360px]">
        <div>
          <div v-if="unreadConversations.length > 0" class="border-b border-white/[0.08] light:border-gray-200">
            <div class="px-4 py-2 text-[11px] uppercase tracking-wide text-white/40 light:text-gray-500">
              Unread Messages
            </div>
            <button
              v-for="contact in unreadConversations"
              :key="`chat-${contact.id}`"
              type="button"
              class="w-full px-4 py-3 text-left hover:bg-white/[0.04] light:hover:bg-gray-50 transition-colors border-t border-white/[0.04] light:border-gray-100"
              @click="openConversation(contact.id)"
            >
              <div class="flex items-start justify-between gap-2">
                <div class="min-w-0 flex-1">
                  <p class="text-sm font-medium text-white light:text-gray-900 truncate">
                    {{ getContactLabel(contact) }}
                  </p>
                  <p class="text-xs text-white/60 light:text-gray-600 truncate">
                    {{ contact.last_message_preview || contact.phone_number }}
                  </p>
                  <p class="text-[11px] text-white/40 light:text-gray-500 mt-1">
                    {{ formatDate(contact.last_message_at || contact.updated_at) }}
                  </p>
                </div>
                <Badge class="min-w-[18px] h-[18px] px-1 text-[10px] bg-emerald-500 text-white border-0">
                  {{ contact.unread_count }}
                </Badge>
              </div>
            </button>
          </div>

          <div v-if="notifications.length > 0">
            <div class="px-4 py-2 text-[11px] uppercase tracking-wide text-white/40 light:text-gray-500">
              System Notifications
            </div>
            <div class="divide-y divide-white/[0.08] light:divide-gray-200">
              <div
                v-for="notification in notifications"
                :key="notification.id"
                class="px-4 py-3 flex items-start gap-2"
              >
                <div class="flex-1 min-w-0">
                  <div class="text-xs uppercase tracking-wide text-white/40 light:text-gray-500">
                    {{ notification.event_type.replace('_', ' ') }}
                  </div>
                  <p class="text-sm text-white light:text-gray-900 mt-1 break-words">
                    {{ notification.message }}
                  </p>
                  <p class="text-[11px] text-white/40 light:text-gray-500 mt-1">
                    {{ formatDate(notification.created_at) }}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-7 w-7 text-white/40 hover:text-white hover:bg-white/[0.08] light:text-gray-400 light:hover:text-gray-700 light:hover:bg-gray-100"
                  @click="dismissNotification(notification.id)"
                >
                  <X class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
          </div>
        </div>
      </ScrollArea>
    </PopoverContent>
  </Popover>
</template>
