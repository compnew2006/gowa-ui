<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { FileText, Film, Image as ImageIcon, Music, Download, Loader2, Package } from 'lucide-vue-next'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { Message } from '@/stores/contacts'

const props = defineProps<{
  /** Two-way bound open state. */
  open: boolean
  /** The burst of messages to offer for download. */
  messages: Message[]
  /** Whether a download is currently in flight (drives button spinners). */
  isDownloading?: boolean
  /** `{ current, total }` progress for the active download. */
  progress?: { current: number; total: number }
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'zip'): void
  (e: 'separate'): void
}>()

const { t } = useI18n()

const open = computed({
  get: () => props.open,
  set: (v) => emit('update:open', v)
})

const hasFiles = computed(() => props.messages.some((m) => !!m.media_url))

function iconFor(message: Message) {
  switch (message.message_type) {
    case 'image':
    case 'sticker':
      return ImageIcon
    case 'video':
      return Film
    case 'audio':
      return Music
    default:
      return FileText
  }
}

function labelFor(message: Message): string {
  return message.media_filename || `${message.message_type}_${message.id.slice(0, 8)}`
}

function timeFor(message: Message): string {
  const d = new Date(message.created_at)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function progressLabel(): string {
  const p = props.progress ?? { current: 0, total: 0 }
  return t('chat.downloadingBurst', { current: p.current, total: p.total })
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="max-w-lg">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <Package class="h-5 w-5" />
          {{ $t('chat.mediaBurstTitle') }}
        </DialogTitle>
        <DialogDescription>{{ $t('chat.mediaBurstDesc') }}</DialogDescription>
      </DialogHeader>

      <ScrollArea class="max-h-[320px] -mx-2 px-2">
        <ul class="space-y-1.5">
          <li
            v-for="message in messages"
            :key="message.id"
            class="flex items-center gap-3 rounded-lg px-2 py-1.5 hover:bg-muted/50"
          >
            <component :is="iconFor(message)" class="h-4 w-4 shrink-0 text-muted-foreground" />
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium">{{ labelFor(message) }}</p>
              <p v-if="message.media_mime_type" class="truncate text-xs text-muted-foreground">
                {{ message.media_mime_type }}
              </p>
            </div>
            <span class="shrink-0 text-xs text-muted-foreground">{{ timeFor(message) }}</span>
          </li>
        </ul>
      </ScrollArea>

      <p v-if="isDownloading" class="flex items-center gap-2 text-xs text-muted-foreground">
        <Loader2 class="h-3.5 w-3.5 animate-spin" />
        {{ progressLabel() }}
      </p>

      <div class="flex justify-end gap-2 pt-1">
        <Button variant="outline" :disabled="!hasFiles || isDownloading" @click="emit('separate')">
          <Download v-if="!isDownloading" class="mr-2 h-4 w-4" />
          {{ $t('chat.downloadSeparately') }}
        </Button>
        <Button :disabled="!hasFiles || isDownloading" @click="emit('zip')">
          <Loader2 v-if="isDownloading" class="mr-2 h-4 w-4 animate-spin" />
          <Package v-else class="mr-2 h-4 w-4" />
          {{ $t('chat.downloadZip') }}
        </Button>
      </div>
    </DialogContent>
  </Dialog>
</template>
