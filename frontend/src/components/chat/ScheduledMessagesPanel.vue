<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useScheduledMessagesStore } from '@/stores/scheduledMessages'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Input } from '@/components/ui/input'
import { toast } from 'vue-sonner'
import { getErrorMessage } from '@/lib/api-utils'
import {
  CalendarClock, Pencil, Trash2, X, Check, Loader2,
  Image as ImageIcon, Video, Mic, FileText, MessageSquare
} from 'lucide-vue-next'

defineProps<{
  contactId: string
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const scheduledStore = useScheduledMessagesStore()

const editingId = ref<string | null>(null)
const editingTime = ref('')
const isSaving = ref(false)
const cancellingId = ref<string | null>(null)

// Format a Date as the local "YYYY-MM-DDTHH:mm" string datetime-local expects
function toLocalInputValue(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const minLocal = computed(() => toLocalInputValue(new Date(Date.now() + 2 * 60 * 1000)))

function typeIcon(messageType: string) {
  switch (messageType) {
    case 'image': return ImageIcon
    case 'video': return Video
    case 'audio': return Mic
    case 'document': return FileText
    default: return MessageSquare
  }
}

function formatScheduledTime(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date()
  const sameYear = date.getFullYear() === now.getFullYear()
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    ...(sameYear ? {} : { year: 'numeric' }),
    hour: '2-digit',
    minute: '2-digit'
  })
}

function startEditing(id: string, scheduledAt: string) {
  editingId.value = id
  editingTime.value = toLocalInputValue(new Date(scheduledAt))
}

function cancelEditing() {
  editingId.value = null
  editingTime.value = ''
}

async function saveTime(id: string) {
  if (!editingTime.value) return
  const newDate = new Date(editingTime.value)
  if (newDate.getTime() <= Date.now() + 60 * 1000) {
    toast.error(t('chat.schedulePastError'))
    return
  }
  isSaving.value = true
  try {
    await scheduledStore.updateSchedule(id, { scheduled_at: newDate.toISOString() })
    toast.success(t('chat.scheduleUpdated'))
    cancelEditing()
  } catch (error) {
    toast.error(getErrorMessage(error, t('chat.scheduleUpdateFailed')))
  } finally {
    isSaving.value = false
  }
}

async function cancelSchedule(id: string) {
  if (!confirm(t('chat.cancelScheduleConfirm'))) return
  cancellingId.value = id
  try {
    await scheduledStore.cancel(id)
    toast.success(t('chat.scheduleCancelled'))
  } catch (error) {
    toast.error(getErrorMessage(error, t('chat.scheduleCancelFailed')))
  } finally {
    cancellingId.value = null
  }
}
</script>

<template>
  <div id="scheduled-panel" class="w-80 border-l border-white/[0.08] light:border-gray-200 bg-[#111113] light:bg-white flex flex-col">
    <!-- Header -->
    <div class="px-4 py-3 border-b border-white/[0.08] light:border-gray-200 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <div class="h-7 w-7 rounded-lg bg-sky-500/15 flex items-center justify-center">
          <CalendarClock class="h-4 w-4 text-sky-400 light:text-sky-600" />
        </div>
        <span class="text-sm font-semibold text-white light:text-gray-900">{{ t('chat.scheduledMessages') }}</span>
        <Badge v-if="scheduledStore.items.length > 0" class="bg-sky-500/20 text-sky-400 light:bg-sky-100 light:text-sky-700 border-0 text-[10px] px-1.5 py-0">
          {{ scheduledStore.items.length }}
        </Badge>
      </div>
      <Button
        variant="ghost"
        size="icon"
        class="h-7 w-7 text-white/40 hover:text-white hover:bg-white/[0.08] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-100"
        @click="emit('close')"
      >
        <X class="h-4 w-4" />
      </Button>
    </div>

    <!-- Scheduled list -->
    <ScrollArea class="flex-1 p-3">
      <div class="space-y-3">
        <!-- Loading state -->
        <div v-if="scheduledStore.isLoading" class="flex justify-center py-8">
          <Loader2 class="h-5 w-5 animate-spin text-white/30 light:text-gray-400" />
        </div>

        <!-- Scheduled messages (soonest first) -->
        <template v-else-if="scheduledStore.items.length > 0">
          <div
            v-for="sm in scheduledStore.items"
            :key="sm.id"
            class="group relative rounded-xl p-3 backdrop-blur-sm border border-white/[0.06] light:border-gray-200 bg-gradient-to-br from-white/[0.04] to-white/[0.02] light:from-gray-50 light:to-white hover:from-white/[0.06] hover:to-white/[0.03] light:hover:from-gray-100 light:hover:to-gray-50 transition-all duration-200"
          >
            <!-- Gradient accent line -->
            <div class="absolute top-0 left-3 right-3 h-[2px] rounded-full bg-gradient-to-r from-sky-500/60 via-blue-500/40 to-transparent" />

            <div class="flex items-start gap-2.5 mt-1">
              <div class="h-6 w-6 shrink-0 rounded-md bg-sky-500/10 light:bg-sky-50 flex items-center justify-center">
                <component :is="typeIcon(sm.message_type)" class="h-3.5 w-3.5 text-sky-400/70 light:text-sky-500" />
              </div>
              <div class="flex-1 min-w-0">
                <div class="flex items-center justify-between mb-1">
                  <span class="text-xs font-medium text-sky-400/90 light:text-sky-600 flex items-center gap-1">
                    <CalendarClock class="h-3 w-3" />
                    {{ formatScheduledTime(sm.scheduled_at) }}
                  </span>
                  <!-- Hover actions -->
                  <div
                    v-if="editingId !== sm.id"
                    class="opacity-0 group-hover:opacity-100 transition-opacity flex gap-0.5"
                  >
                    <button
                      class="h-5 w-5 rounded-md flex items-center justify-center hover:bg-white/[0.08] light:hover:bg-gray-200 text-white/30 hover:text-white/60 light:text-gray-400 light:hover:text-gray-600 transition-colors"
                      :title="t('chat.editScheduleTime')"
                      @click="startEditing(sm.id, sm.scheduled_at)"
                    >
                      <Pencil class="h-3 w-3" />
                    </button>
                    <button
                      class="h-5 w-5 rounded-md flex items-center justify-center hover:bg-red-500/10 text-white/30 hover:text-red-400 light:text-gray-400 light:hover:text-red-500 transition-colors"
                      :title="t('chat.cancelSchedule')"
                      :disabled="cancellingId === sm.id"
                      @click="cancelSchedule(sm.id)"
                    >
                      <Loader2 v-if="cancellingId === sm.id" class="h-3 w-3 animate-spin" />
                      <Trash2 v-else class="h-3 w-3" />
                    </button>
                  </div>
                </div>

                <!-- Media filename -->
                <div v-if="sm.media_filename" class="flex items-center gap-1.5 text-xs text-white/40 light:text-gray-500 mb-1">
                  <FileText class="h-3 w-3 shrink-0" />
                  <span class="truncate">{{ sm.media_filename }}</span>
                </div>

                <p v-if="sm.content" class="text-[13px] text-white/60 light:text-gray-600 leading-relaxed whitespace-pre-wrap break-words line-clamp-4">{{ sm.content }}</p>

                <!-- Inline time edit -->
                <div v-if="editingId === sm.id" class="mt-2 space-y-2">
                  <Input
                    v-model="editingTime"
                    type="datetime-local"
                    :min="minLocal"
                    class="h-8 text-xs bg-white/[0.04] light:bg-gray-50 border-sky-500/20 light:border-sky-200"
                  />
                  <div class="flex justify-end gap-1.5">
                    <Button variant="ghost" size="sm" class="h-7 text-xs" @click="cancelEditing">
                      {{ t('common.cancel') }}
                    </Button>
                    <Button
                      size="sm"
                      class="h-7 text-xs bg-sky-600 hover:bg-sky-500 text-white"
                      :disabled="!editingTime || isSaving"
                      @click="saveTime(sm.id)"
                    >
                      <Loader2 v-if="isSaving" class="h-3 w-3 mr-1 animate-spin" />
                      <Check v-else class="h-3 w-3 mr-1" />
                      {{ t('common.save') }}
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- Empty state -->
        <div v-else class="flex flex-col items-center justify-center py-12 text-center">
          <div class="h-12 w-12 rounded-xl bg-sky-500/10 light:bg-sky-50 flex items-center justify-center mb-3">
            <CalendarClock class="h-6 w-6 text-sky-400/50 light:text-sky-400" />
          </div>
          <p class="text-sm font-medium text-white/40 light:text-gray-500 mb-1">{{ t('chat.noScheduledMessages') }}</p>
          <p class="text-xs text-white/25 light:text-gray-400">{{ t('chat.noScheduledMessagesHint') }}</p>
        </div>
      </div>
    </ScrollArea>
  </div>
</template>
