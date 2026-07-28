<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { dailyResetService } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { Loader2 } from 'lucide-vue-next'

// Per-account daily assigned-chat reset: at the configured wall-clock time,
// once per day, every assigned (open) conversation is returned to the pending
// pool. Each number belongs to a different branch with its own operating
// hours and staffing, so the toggle and the reset time live on the account,
// not organization-wide.
const props = defineProps<{
  accountId: string
  canWrite: boolean
}>()

// Emitted after a successful save so the parent (AccountDetailView) can refresh
// the unified Activity Log. The daily_reset audit entry shares the account's
// resource_id, so it surfaces in the parent's aggregated timeline.
const emit = defineEmits<{ (e: 'saved'): void }>()

const { t } = useI18n()

const settings = ref({
  enabled: false,
  time: '02:00',
  timezone: ''
})
const isSubmitting = ref(false)

// Time validation: HH:MM, 24-hour. Matches the backend's time.Parse("15:04").
const TIME_RE = /^([01]\d|2[0-3]):[0-5]\d$/
const timeError = ref('')

function validateTime() {
  if (!TIME_RE.test(settings.value.time)) {
    timeError.value = t('settings.chatResetTimeInvalid')
    return false
  }
  timeError.value = ''
  return true
}

onMounted(async () => {
  try {
    const response = await dailyResetService.getSettings(props.accountId)
    const data = response.data.data
    if (data) {
      settings.value = {
        enabled: data.enabled,
        time: data.time,
        timezone: data.timezone
      }
    }
  } catch (error) {
    console.error('Failed to load daily-reset settings:', error)
  }
})

async function saveSettings() {
  if (!validateTime()) return
  isSubmitting.value = true
  try {
    await dailyResetService.updateSettings(props.accountId, {
      enabled: settings.value.enabled,
      time: settings.value.time,
      timezone: settings.value.timezone
    })
    toast.success(t('settings.chatResetSaved'))
    // Give the backend a moment to write the audit entry, then let the parent
    // refresh its unified Activity Log.
    setTimeout(() => emit('saved'), 500)
  } catch (error) {
    toast.error(t('common.failedSave', { resource: t('resources.settings') }))
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="text-sm font-medium">{{ $t('settings.chatResetSettings') }}</CardTitle>
        <CardDescription class="text-xs">{{ $t('settings.chatResetSettingsDesc') }}</CardDescription>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <Label class="text-xs">{{ $t('settings.chatResetEnable') }}</Label>
            <p class="text-[11px] text-muted-foreground">{{ $t('settings.chatResetEnableDesc') }}</p>
          </div>
          <Switch
            :checked="settings.enabled"
            @update:checked="settings.enabled = $event"
            :disabled="!canWrite"
          />
        </div>
        <Separator />
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div class="space-y-1.5">
            <Label for="chat_reset_time" class="text-xs">{{ $t('settings.chatResetTime') }}</Label>
            <Input
              id="chat_reset_time"
              v-model="settings.time"
              type="time"
              placeholder="02:00"
              :disabled="!canWrite"
              @blur="validateTime"
            />
            <p v-if="timeError" class="text-[11px] text-destructive">{{ timeError }}</p>
            <p v-else class="text-[11px] text-muted-foreground">{{ $t('settings.chatResetTimeDesc') }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="chat_reset_tz" class="text-xs">{{ $t('settings.chatResetTimezone') }}</Label>
            <Input
              id="chat_reset_tz"
              v-model="settings.timezone"
              placeholder="Asia/Dubai"
              :disabled="!canWrite"
            />
            <p class="text-[11px] text-muted-foreground">{{ $t('settings.chatResetTimezoneDesc') }}</p>
          </div>
        </div>
        <div v-if="canWrite" class="flex justify-end">
          <Button variant="outline" size="sm" @click="saveSettings" :disabled="isSubmitting">
            <Loader2 v-if="isSubmitting" class="mr-2 h-4 w-4 animate-spin" />
            {{ $t('settings.save') }}
          </Button>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
