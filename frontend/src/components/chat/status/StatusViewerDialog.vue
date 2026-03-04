<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { statusesService, type WhatsAppStatusGroup, type WhatsAppStatusItem } from '@/services/api'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ChevronLeft, ChevronRight, X } from 'lucide-vue-next'

const props = defineProps<{
  open: boolean
  groups: WhatsAppStatusGroup[]
}>()

const emit = defineEmits<{
  (event: 'update:open', value: boolean): void
  (event: 'refresh'): void
}>()

const { t } = useI18n()

const currentGroupIndex = ref(0)
const currentStatusIndex = ref(0)
const replyText = ref('')
const isReplying = ref(false)
let autoAdvanceTimer: ReturnType<typeof setTimeout> | null = null

const currentGroup = computed(() => props.groups[currentGroupIndex.value] || null)
const currentStatus = computed<WhatsAppStatusItem | null>(() => {
  if (!currentGroup.value) return null
  return currentGroup.value.statuses[currentStatusIndex.value] || null
})

const currentInstanceLabel = computed(() => {
  const status = currentStatus.value
  if (status) {
    const statusInstance = String(status.instance_name || '').trim()
    if (statusInstance) return statusInstance
  }
  const group = currentGroup.value
  if (group) {
    const groupInstance = String(group.instance_name || '').trim()
    if (groupInstance) return groupInstance
    const instanceID = String(group.instance_id || '').trim()
    if (instanceID) return `Instance ${instanceID.slice(0, 8)}`
  }
  return 'Unknown Instance'
})

function formatPhoneFromJID(jid: string | undefined): string {
  const raw = String(jid || '').trim()
  if (!raw) return ''
  const userPart = raw.split('@')[0]?.trim() || ''
  if (!userPart) return ''
  return userPart
}

function formatStatusTime(value: string): string {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return ''
  return parsed.toLocaleTimeString()
}

const currentStatusMetaLine = computed(() => {
  const status = currentStatus.value
  if (!status) return ''

  const displayName = String(status.sender_name || status.sender_jid || '').trim()
  const phoneNumber = formatPhoneFromJID(status.sender_jid)
  const statusTime = formatStatusTime(status.created_at)

  const parts: string[] = []
  if (displayName) parts.push(displayName)
  if (phoneNumber && phoneNumber !== displayName) parts.push(phoneNumber)
  if (currentInstanceLabel.value) parts.push(currentInstanceLabel.value)
  if (statusTime) parts.push(statusTime)

  return parts.join(' • ')
})

function clearAutoAdvanceTimer() {
  if (!autoAdvanceTimer) return
  clearTimeout(autoAdvanceTimer)
  autoAdvanceTimer = null
}

function closeDialog() {
  clearAutoAdvanceTimer()
  emit('update:open', false)
}

function advanceToNext() {
  const group = currentGroup.value
  if (!group) return

  if (currentStatusIndex.value < group.statuses.length - 1) {
    currentStatusIndex.value += 1
    return
  }

  if (currentGroupIndex.value < props.groups.length - 1) {
    currentGroupIndex.value += 1
    currentStatusIndex.value = 0
    return
  }

  closeDialog()
}

function moveToPrevious() {
  if (currentStatusIndex.value > 0) {
    currentStatusIndex.value -= 1
    return
  }
  if (currentGroupIndex.value > 0) {
    currentGroupIndex.value -= 1
    const previousGroup = props.groups[currentGroupIndex.value]
    currentStatusIndex.value = Math.max(previousGroup.statuses.length - 1, 0)
  }
}

function scheduleAutoAdvance(status: WhatsAppStatusItem | null) {
  clearAutoAdvanceTimer()
  if (!props.open || !status) return
  if (isReplying.value || replyText.value.trim().length > 0) return
  const delayMs = status.status_type === 'video' ? 8000 : 5000
  autoAdvanceTimer = setTimeout(() => {
    advanceToNext()
  }, delayMs)
}

async function markCurrentStatusAsSeen() {
  const status = currentStatus.value
  if (!status || status.is_self || status.seen_at) return
  try {
    await statusesService.markSeen(status.id)
    emit('refresh')
  } catch (error: any) {
    const message =
      error?.response?.data?.message || error?.message || t('chat.statusSeenFailed')
    toast.error(message)
  }
}

async function submitReply() {
  const status = currentStatus.value
  const text = replyText.value.trim()
  if (!status || status.is_self || text === '' || isReplying.value) return

  isReplying.value = true
  try {
    await statusesService.reply(status.id, text)
    toast.success(t('chat.statusReplySent'))
    replyText.value = ''
  } catch (error: any) {
    const message =
      error?.response?.data?.message || error?.message || t('chat.statusReplyFailed')
    toast.error(message)
  } finally {
    isReplying.value = false
    scheduleAutoAdvance(currentStatus.value)
  }
}

watch(
  () => props.open,
  (open) => {
    if (!open) {
      clearAutoAdvanceTimer()
      replyText.value = ''
      isReplying.value = false
      return
    }
    currentGroupIndex.value = 0
    currentStatusIndex.value = 0
    replyText.value = ''
  },
)

watch(currentStatus, (status) => {
  if (!props.open || !status) return
  replyText.value = ''
  void markCurrentStatusAsSeen()
  scheduleAutoAdvance(status)
})

watch(replyText, () => {
  if (!props.open) return
  scheduleAutoAdvance(currentStatus.value)
})

onBeforeUnmount(() => {
  clearAutoAdvanceTimer()
})
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-w-3xl overflow-hidden p-0" data-testid="status-viewer-dialog">
      <div class="relative h-[70vh] bg-black text-white">
        <div class="absolute left-0 top-0 z-20 flex w-full items-center gap-2 px-3 py-2">
          <template v-if="currentGroup">
            <div
              v-for="(status, index) in currentGroup.statuses"
              :key="status.id"
              class="h-1 flex-1 rounded-full bg-white/20"
            >
              <div
                class="h-full rounded-full bg-white transition-all"
                :style="{
                  width:
                    index < currentStatusIndex
                      ? '100%'
                      : index === currentStatusIndex
                        ? '60%'
                        : '0%',
                }"
              />
            </div>
          </template>
        </div>

        <button
          class="absolute right-2 top-2 z-30 rounded-md bg-black/40 p-1 text-white/80 transition-colors hover:text-white"
          :aria-label="$t('common.close')"
          @click="closeDialog"
        >
          <X class="h-4 w-4" />
        </button>

        <div
          v-if="currentStatus"
          class="flex h-full items-center justify-center px-4 pt-10 text-center"
          data-testid="status-viewer-content"
        >
          <div class="w-full max-w-2xl space-y-4">
            <p class="text-sm text-white/70">
              {{ currentStatusMetaLine }}
            </p>

            <div
              v-if="currentStatus.status_type === 'text'"
              class="rounded-2xl px-8 py-14 text-xl font-semibold shadow-lg"
              :style="{
                background:
                  currentStatus.background_argb !== undefined
                    ? `#${Number(currentStatus.background_argb)
                        .toString(16)
                        .padStart(8, '0')
                        .slice(2)}`
                    : '#1d4ed8',
              }"
            >
              {{ currentStatus.content }}
            </div>

            <img
              v-else-if="currentStatus.status_type === 'image'"
              :src="currentStatus.media_url"
              class="mx-auto max-h-[54vh] rounded-xl object-contain"
              :alt="currentStatus.content || 'status-image'"
            />

            <video
              v-else
              :src="currentStatus.media_url"
              class="mx-auto max-h-[54vh] rounded-xl object-contain"
              controls
              autoplay
              playsinline
            />

            <p v-if="currentStatus.content && currentStatus.status_type !== 'text'" class="text-sm text-white/80">
              {{ currentStatus.content }}
            </p>
          </div>
        </div>

        <div
          v-if="currentStatus && !currentStatus.is_self"
          class="absolute bottom-3 left-3 right-3 z-20 flex items-center gap-2 rounded-lg bg-black/45 p-2 backdrop-blur-sm"
        >
          <Input
            v-model="replyText"
            data-testid="status-reply-input"
            class="border-white/20 bg-black/30 text-white placeholder:text-white/60"
            :placeholder="$t('chat.statusReplyPlaceholder')"
            @keydown.enter.prevent="submitReply"
          />
          <Button
            data-testid="status-reply-send-button"
            :disabled="replyText.trim().length === 0 || isReplying"
            @click="submitReply"
          >
            {{ isReplying ? $t('common.loading') : $t('chat.statusReplySend') }}
          </Button>
        </div>

        <div class="absolute inset-y-0 left-0 flex items-center px-2">
          <Button
            size="icon"
            variant="ghost"
            class="text-white hover:bg-white/10"
            @click="moveToPrevious"
          >
            <ChevronLeft class="h-5 w-5" />
          </Button>
        </div>

        <div class="absolute inset-y-0 right-0 flex items-center px-2">
          <Button
            size="icon"
            variant="ghost"
            class="text-white hover:bg-white/10"
            @click="advanceToNext"
          >
            <ChevronRight class="h-5 w-5" />
          </Button>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
