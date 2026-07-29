<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { dailyResetService } from '@/services/api'
import AccountSettingsPanel from './AccountSettingsPanel.vue'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'

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
  <AccountSettingsPanel
    :title="$t('settings.chatResetSettings')"
    :description="$t('settings.chatResetSettingsDesc')"
    :enable-label="$t('settings.chatResetEnable')"
    :enable-desc="$t('settings.chatResetEnableDesc')"
    v-model:enabled="settings.enabled"
    :can-write="canWrite"
    :is-submitting="isSubmitting"
    @save="saveSettings"
  >
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
  </AccountSettingsPanel>
</template>
