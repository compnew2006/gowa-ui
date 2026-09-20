<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { mediaRetentionService } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Loader2, AlertTriangle } from 'lucide-vue-next'

// Per-account media retention: local media files older than the configured
// window are deleted by the daily backend processor. Deletion is permanent —
// the message row stays, but the file is gone (recoverable only via an
// explicit re-download while the provider still holds it).
const props = defineProps<{
  accountId: string
  canWrite: boolean
}>()

const emit = defineEmits<{ (e: 'saved'): void }>()

const { t } = useI18n()

const enabled = ref(false)
const retentionDays = ref(30)
const isSubmitting = ref(false)

onMounted(async () => {
  try {
    const res = await mediaRetentionService.getSettings(props.accountId)
    const s = res.data.data
    enabled.value = !!s.enabled
    if (s.retention_days > 0) {
      retentionDays.value = s.retention_days
    }
  } catch {
    toast.error(t('accounts.mediaRetention.loadFailed', 'Failed to load media retention settings'))
  }
})

async function save() {
  let days = 0
  if (enabled.value) {
    days = Math.floor(Number(retentionDays.value))
    if (!Number.isFinite(days) || days < 1 || days > 3650) {
      toast.error(t('accounts.mediaRetention.invalidDays', 'Retention days must be between 1 and 3650'))
      return
    }
  }
  isSubmitting.value = true
  try {
    await mediaRetentionService.updateSettings(props.accountId, {
      enabled: enabled.value,
      retention_days: days
    })
    toast.success(t('accounts.mediaRetention.saved', 'Media retention saved'))
    emit('saved')
  } catch {
    toast.error(t('accounts.mediaRetention.saveFailed', 'Failed to save media retention'))
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>{{ t('accounts.mediaRetention.title', 'Media Retention') }}</CardTitle>
      <CardDescription>
        {{
          t(
            'accounts.mediaRetention.description',
            'Automatically delete local media files older than the retention window to free disk space.'
          )
        }}
      </CardDescription>
    </CardHeader>
    <CardContent class="space-y-4">
      <div class="flex items-center justify-between rounded-lg border p-3">
        <div class="space-y-0.5">
          <Label>{{ t('accounts.mediaRetention.enable', 'Delete old media files') }}</Label>
          <p class="text-xs text-muted-foreground">
            {{ t('accounts.mediaRetention.firstRun', 'First cleanup runs within 24 hours of enabling.') }}
          </p>
        </div>
        <Switch :model-value="enabled" :disabled="!props.canWrite" @update:model-value="enabled = $event" />
      </div>

      <div v-if="enabled" class="space-y-2">
        <Label for="retention-days">{{ t('accounts.mediaRetention.days', 'Keep media for (days)') }}</Label>
        <Input
          id="retention-days"
          v-model.number="retentionDays"
          type="number"
          min="1"
          max="3650"
          :disabled="!props.canWrite"
        />
      </div>

      <div v-if="enabled" class="flex items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3">
        <AlertTriangle class="h-4 w-4 text-amber-500 shrink-0 mt-0.5" />
        <p class="text-xs text-muted-foreground">
          {{
            t(
              'accounts.mediaRetention.warning',
              'Deletion is permanent: chat history keeps the message, but the file itself cannot be recovered unless the provider still holds it (explicit re-download).'
            )
          }}
        </p>
      </div>

      <Button :disabled="!props.canWrite || isSubmitting" @click="save">
        <Loader2 v-if="isSubmitting" class="mr-2 h-4 w-4 animate-spin" />
        {{ t('common.save', 'Save') }}
      </Button>
    </CardContent>
  </Card>
</template>
