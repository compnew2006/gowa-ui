<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useScheduledMessagesStore } from '@/stores/scheduledMessages'
import {
  Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogFooter
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toast } from 'vue-sonner'
import { getErrorMessage } from '@/lib/api-utils'
import { CalendarClock, Loader2, FileText } from 'lucide-vue-next'

const props = defineProps<{
  contactId: string
  body: string
  file?: File | null
  whatsappAccount?: string | null
}>()

const emit = defineEmits<{
  scheduled: []
}>()

const open = defineModel<boolean>('open', { required: true })

const { t } = useI18n()
const scheduledStore = useScheduledMessagesStore()

const scheduledAtLocal = ref('')
const isSubmitting = ref(false)

// Format a Date as the local "YYYY-MM-DDTHH:mm" string datetime-local expects
function toLocalInputValue(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const minLocal = computed(() => toLocalInputValue(new Date(Date.now() + 2 * 60 * 1000)))

// Default to one hour from now every time the dialog opens
watch(open, (isOpen) => {
  if (isOpen) {
    scheduledAtLocal.value = toLocalInputValue(new Date(Date.now() + 60 * 60 * 1000))
  }
})

function applyPreset(preset: 'hour' | 'tomorrow' | 'monday') {
  const d = new Date()
  if (preset === 'hour') {
    d.setHours(d.getHours() + 1)
  } else if (preset === 'tomorrow') {
    d.setDate(d.getDate() + 1)
    d.setHours(9, 0, 0, 0)
  } else {
    // Next Monday 9:00
    const daysUntilMonday = ((8 - d.getDay()) % 7) || 7
    d.setDate(d.getDate() + daysUntilMonday)
    d.setHours(9, 0, 0, 0)
  }
  scheduledAtLocal.value = toLocalInputValue(d)
}

function getMediaType(mimeType: string): string {
  if (mimeType.startsWith('image/')) return 'image'
  if (mimeType.startsWith('video/')) return 'video'
  if (mimeType.startsWith('audio/')) return 'audio'
  return 'document'
}

// Read a File as base64 (without the data: URL prefix)
function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      resolve(result.slice(result.indexOf(',') + 1))
    }
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

const isValid = computed(() => {
  if (!scheduledAtLocal.value) return false
  return new Date(scheduledAtLocal.value).getTime() > Date.now() + 60 * 1000
})

async function submit() {
  if (!isValid.value) {
    toast.error(t('chat.schedulePastError'))
    return
  }

  isSubmitting.value = true
  try {
    const scheduledAt = new Date(scheduledAtLocal.value).toISOString()
    if (props.file) {
      const mediaData = await fileToBase64(props.file)
      await scheduledStore.schedule(props.contactId, {
        type: getMediaType(props.file.type),
        content: {
          body: props.body,
          media_data: mediaData,
          media_mime_type: props.file.type,
          media_filename: props.file.name
        },
        whatsapp_account: props.whatsappAccount || undefined,
        scheduled_at: scheduledAt
      })
    } else {
      await scheduledStore.schedule(props.contactId, {
        type: 'text',
        content: { body: props.body },
        whatsapp_account: props.whatsappAccount || undefined,
        scheduled_at: scheduledAt
      })
    }
    toast.success(t('chat.messageScheduled'))
    open.value = false
    emit('scheduled')
  } catch (error) {
    toast.error(getErrorMessage(error, t('chat.scheduleFailed')))
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="max-w-sm">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <CalendarClock class="h-4 w-4" />
          {{ $t('chat.scheduleMessageTitle') }}
        </DialogTitle>
        <DialogDescription>{{ $t('chat.scheduleMessageDesc') }}</DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2">
        <!-- Preview of what will be sent -->
        <div class="rounded-lg bg-muted/50 border border-border px-3 py-2 text-sm space-y-1">
          <div v-if="file" class="flex items-center gap-2 text-muted-foreground">
            <FileText class="h-4 w-4 shrink-0" />
            <span class="truncate">{{ file.name }}</span>
          </div>
          <p v-if="body" class="line-clamp-3 whitespace-pre-wrap break-words">{{ body }}</p>
        </div>

        <div class="space-y-1.5">
          <Label class="text-xs">{{ $t('chat.scheduledAtLabel') }}</Label>
          <Input v-model="scheduledAtLocal" type="datetime-local" :min="minLocal" />
        </div>

        <div class="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" type="button" @click="applyPreset('hour')">
            {{ $t('chat.scheduleInOneHour') }}
          </Button>
          <Button variant="outline" size="sm" type="button" @click="applyPreset('tomorrow')">
            {{ $t('chat.scheduleTomorrowMorning') }}
          </Button>
          <Button variant="outline" size="sm" type="button" @click="applyPreset('monday')">
            {{ $t('chat.scheduleMondayMorning') }}
          </Button>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" type="button" @click="open = false">
          {{ $t('common.cancel') }}
        </Button>
        <Button type="button" :disabled="!isValid || isSubmitting" @click="submit">
          <Loader2 v-if="isSubmitting" class="h-4 w-4 me-1.5 animate-spin" />
          <CalendarClock v-else class="h-4 w-4 me-1.5" />
          {{ $t('chat.scheduleSend') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
