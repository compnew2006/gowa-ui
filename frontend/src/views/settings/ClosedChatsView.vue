<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Archive, Loader2, RotateCw } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { PageHeader } from '@/components/shared'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useContactsStore, type Contact } from '@/stores/contacts'

const router = useRouter()
const contactsStore = useContactsStore()
const isLoading = ref(false)
const reopeningChatId = ref<string | null>(null)
const searchQuery = ref('')

const filteredClosedChats = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return contactsStore.closedChats
  return contactsStore.closedChats.filter(chat =>
    (chat.name || '').toLowerCase().includes(query) ||
    (chat.phone_number || '').toLowerCase().includes(query) ||
    (chat.closed_by_name || '').toLowerCase().includes(query)
  )
})

function formatClosedDate(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString()
}

function getClosedByLabel(chat: Contact): string {
  if (chat.closed_by_name && chat.closed_by_name.trim()) return chat.closed_by_name
  if (chat.closed_by_user_id) return chat.closed_by_user_id
  if (chat.assigned_user_id) return chat.assigned_user_id
  return '—'
}

async function loadClosedChats() {
  isLoading.value = true
  try {
    await contactsStore.fetchClosedChats({ limit: 200 })
  } finally {
    isLoading.value = false
  }
}

function openReadOnlyChat(chat: Contact) {
  router.push({ name: 'chat-conversation', params: { contactId: chat.id }, query: { tab: 'assigned' } })
}

async function reopenChat(chat: Contact) {
  if (reopeningChatId.value) return
  reopeningChatId.value = chat.id
  try {
    const updated = await contactsStore.reopenChat(chat.id)
    toast.success('Chat reopened and moved to pending queue')
    await contactsStore.fetchClosedChats({ limit: 200 })
    router.push({ name: 'chat-conversation', params: { contactId: updated.id }, query: { tab: 'pending' } })
  } catch (error: any) {
    const message = error?.response?.data?.message || 'Failed to reopen chat'
    toast.error(message)
  } finally {
    reopeningChatId.value = null
  }
}

onMounted(loadClosedChats)
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader
      title="Closed Chats"
      subtitle="Review resolved conversations in read-only mode."
      :icon="Archive"
      icon-gradient="bg-gradient-to-br from-zinc-500 to-zinc-700 shadow-zinc-500/20"
    />

    <div class="p-6 space-y-4">
      <div class="flex items-center gap-2">
        <Input
          v-model="searchQuery"
          placeholder="Search by contact, phone, or closed by..."
          class="max-w-md bg-white/[0.04] border-white/[0.1] text-white placeholder:text-white/40 light:bg-white light:border-gray-200 light:text-gray-900 light:placeholder:text-gray-400"
        />
        <Button variant="outline" @click="loadClosedChats" :disabled="isLoading">
          {{ isLoading ? 'Refreshing...' : 'Refresh' }}
        </Button>
      </div>

      <div class="rounded-xl border border-white/[0.08] bg-white/[0.02] light:bg-white light:border-gray-200 overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-white/[0.04] light:bg-gray-50">
            <tr>
              <th class="text-left px-4 py-3 font-medium text-white/70 light:text-gray-700">Contact Name</th>
              <th class="text-left px-4 py-3 font-medium text-white/70 light:text-gray-700">Closed By</th>
              <th class="text-left px-4 py-3 font-medium text-white/70 light:text-gray-700">Date Closed</th>
              <th class="text-right px-4 py-3 font-medium text-white/70 light:text-gray-700">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="isLoading">
              <td colspan="4" class="px-4 py-8 text-center text-white/50 light:text-gray-500">Loading closed chats...</td>
            </tr>
            <tr
              v-for="chat in filteredClosedChats"
              :key="chat.id"
              class="border-t border-white/[0.06] light:border-gray-100 hover:bg-white/[0.04] light:hover:bg-gray-50 cursor-pointer"
              @click="openReadOnlyChat(chat)"
            >
              <td class="px-4 py-3 text-white light:text-gray-900">
                <div class="font-medium">{{ chat.name || chat.profile_name || chat.phone_number }}</div>
                <div class="text-xs text-white/50 light:text-gray-500">{{ chat.phone_number }}</div>
              </td>
              <td class="px-4 py-3 text-white/80 light:text-gray-700">{{ getClosedByLabel(chat) }}</td>
              <td class="px-4 py-3 text-white/80 light:text-gray-700">{{ formatClosedDate(chat.closed_at || chat.updated_at) }}</td>
              <td class="px-4 py-3 text-right">
                <Button
                  size="sm"
                  variant="outline"
                  class="h-7 px-2.5 bg-white/[0.04] border-white/[0.12] text-white/80 hover:bg-white/[0.08] hover:text-white light:bg-white light:border-gray-200 light:text-gray-700 light:hover:bg-gray-50"
                  :disabled="reopeningChatId === chat.id"
                  @click.stop="reopenChat(chat)"
                >
                  <Loader2 v-if="reopeningChatId === chat.id" class="mr-1.5 h-3 w-3 animate-spin" />
                  <RotateCw v-else class="mr-1.5 h-3 w-3" />
                  Reopen
                </Button>
              </td>
            </tr>
            <tr v-if="!isLoading && filteredClosedChats.length === 0">
              <td colspan="4" class="px-4 py-8 text-center text-white/50 light:text-gray-500">No closed chats found.</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
