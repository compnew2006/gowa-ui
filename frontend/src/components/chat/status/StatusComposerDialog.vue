<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { statusesService } from '@/services/api'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Textarea } from '@/components/ui/textarea'
import { Input } from '@/components/ui/input'

type InstanceOption = {
  id: string
  name: string
  status?: string
}

const props = defineProps<{
  open: boolean
  instances: InstanceOption[]
}>()

const emit = defineEmits<{
  (event: 'update:open', value: boolean): void
  (event: 'submitted'): void
}>()

const { t } = useI18n()

const mode = ref<'text' | 'image' | 'video'>('text')
const selectedInstanceID = ref('')
const textBody = ref('')
const textBackground = ref('#1d4ed8')
const textFont = ref('SYSTEM')
const mediaCaption = ref('')
const mediaFile = ref<File | null>(null)
const isSubmitting = ref(false)

const canSubmit = computed(() => {
  if (!selectedInstanceID.value) return false
  if (mode.value === 'text') {
    return textBody.value.trim().length > 0
  }
  return mediaFile.value !== null
})

watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (!selectedInstanceID.value && props.instances.length > 0) {
      selectedInstanceID.value = props.instances[0].id
    }
  },
  { immediate: true },
)

function closeDialog() {
  emit('update:open', false)
}

function resetForm() {
  mode.value = 'text'
  textBody.value = ''
  textBackground.value = '#1d4ed8'
  textFont.value = 'SYSTEM'
  mediaCaption.value = ''
  mediaFile.value = null
}

function hexToARGB(value: string): number {
  const hex = value.trim().replace('#', '')
  if (!/^[0-9a-fA-F]{6}$/.test(hex)) {
    return 0xff1d4ed8
  }
  return Number.parseInt(`ff${hex}`, 16)
}

function onFileSelected(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0] || null
  mediaFile.value = file
  if (!file) return

  if (file.type.startsWith('video/')) {
    mode.value = 'video'
    return
  }
  mode.value = 'image'
}

async function submitStatus() {
  if (!canSubmit.value || isSubmitting.value) return
  isSubmitting.value = true

  try {
    if (mode.value === 'text') {
      await statusesService.sendText(selectedInstanceID.value, {
        text: textBody.value.trim(),
        background_argb: hexToARGB(textBackground.value),
        font: textFont.value,
      })
    } else if (mediaFile.value) {
      await statusesService.sendMedia(selectedInstanceID.value, mediaFile.value, {
        type: mode.value,
        caption: mediaCaption.value.trim() || undefined,
      })
    }

    toast.success(t('chat.statusSent'))
    emit('submitted')
    closeDialog()
    resetForm()
  } catch (error: any) {
    const message =
      error?.response?.data?.message || error?.message || t('chat.statusSendFailed')
    toast.error(message)
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-w-lg" data-testid="status-composer-dialog">
      <DialogHeader>
        <DialogTitle>{{ $t('chat.createStatus') }}</DialogTitle>
      </DialogHeader>

      <div class="space-y-4">
        <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <label class="text-xs text-muted-foreground">{{ $t('chat.instance') }}</label>
          <select
            v-model="selectedInstanceID"
            data-testid="status-instance-select"
            class="h-9 rounded-md border border-input bg-background px-2 text-sm"
          >
            <option value="" disabled>{{ $t('chat.selectInstance') }}</option>
            <option
              v-for="instance in instances"
              :key="instance.id"
              :value="instance.id"
            >
              {{ instance.name || instance.id }}
            </option>
          </select>
        </div>

        <div class="flex flex-wrap gap-2">
          <Button
            size="sm"
            :variant="mode === 'text' ? 'default' : 'outline'"
            data-testid="status-mode-text"
            @click="mode = 'text'"
          >
            {{ $t('chat.statusText') }}
          </Button>
          <Button
            size="sm"
            :variant="mode === 'image' ? 'default' : 'outline'"
            data-testid="status-mode-image"
            @click="mode = 'image'"
          >
            {{ $t('chat.statusImage') }}
          </Button>
          <Button
            size="sm"
            :variant="mode === 'video' ? 'default' : 'outline'"
            data-testid="status-mode-video"
            @click="mode = 'video'"
          >
            {{ $t('chat.statusVideo') }}
          </Button>
        </div>

        <div v-if="mode === 'text'" class="space-y-3">
          <Textarea
            v-model="textBody"
            :rows="4"
            data-testid="status-text-input"
            :placeholder="$t('chat.statusTextPlaceholder')"
          />
          <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <label class="text-xs text-muted-foreground">{{ $t('chat.statusBackgroundColor') }}</label>
            <Input
              v-model="textBackground"
              type="color"
              data-testid="status-bg-color-input"
              class="h-9 p-1"
            />
          </div>
          <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <label class="text-xs text-muted-foreground">{{ $t('chat.statusFont') }}</label>
            <select
              v-model="textFont"
              data-testid="status-font-select"
              class="h-9 rounded-md border border-input bg-background px-2 text-sm"
            >
              <option value="SYSTEM">System</option>
              <option value="SYSTEM_TEXT">System Text</option>
              <option value="FB_SCRIPT">Script</option>
              <option value="SYSTEM_BOLD">Bold</option>
              <option value="MORNINGBREEZE_REGULAR">Morning Breeze</option>
              <option value="CALISTOGA_REGULAR">Calistoga</option>
              <option value="EXO2_EXTRABOLD">Exo2</option>
              <option value="COURIERPRIME_BOLD">Courier Prime</option>
            </select>
          </div>
        </div>

        <div v-else class="space-y-3">
          <Input
            type="file"
            data-testid="status-media-file-input"
            :accept="mode === 'video' ? 'video/*' : 'image/*'"
            @change="onFileSelected"
          />
          <Textarea
            v-model="mediaCaption"
            :rows="3"
            data-testid="status-caption-input"
            :placeholder="$t('chat.statusCaptionPlaceholder')"
          />
          <p v-if="mediaFile" class="text-xs text-muted-foreground">
            {{ mediaFile.name }}
          </p>
        </div>

        <div class="flex items-center justify-end gap-2">
          <Button variant="outline" @click="closeDialog">
            {{ $t('common.cancel') }}
          </Button>
          <Button
            data-testid="status-submit-button"
            :disabled="!canSubmit || isSubmitting"
            @click="submitStatus"
          >
            {{ isSubmitting ? $t('common.loading') : $t('chat.postStatus') }}
          </Button>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
