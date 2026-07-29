<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { callAutoRejectService } from '@/services/api'
import AccountSettingsPanel from './AccountSettingsPanel.vue'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

// Per-account call auto-reject: incoming WhatsApp calls are rejected while
// still ringing and the caller optionally receives an automated text. Each
// number belongs to a different branch with its own policy and wording, so
// the toggle and the message live on the account, not organization-wide.
const props = defineProps<{
  accountId: string
  canWrite: boolean
}>()

// Emitted after a successful save so the parent (AccountDetailView) can refresh
// the unified Activity Log. The call_auto_reject audit entry shares the
// account's resource_id, so it surfaces in the parent's aggregated timeline.
const emit = defineEmits<{ (e: 'saved'): void }>()

const { t } = useI18n()

const settings = ref({
  enabled: false,
  message: ''
})
const isSubmitting = ref(false)

onMounted(async () => {
  try {
    const response = await callAutoRejectService.getSettings(props.accountId)
    const data = response.data.data
    if (data) {
      settings.value = {
        enabled: data.enabled,
        message: data.message
      }
    }
  } catch (error) {
    console.error('Failed to load call-auto-reject settings:', error)
  }
})

async function saveSettings() {
  isSubmitting.value = true
  try {
    await callAutoRejectService.updateSettings(props.accountId, {
      enabled: settings.value.enabled,
      message: settings.value.message
    })
    toast.success(t('settings.callRejectSaved'))
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
    :title="$t('settings.callRejectSettings')"
    :description="$t('settings.callRejectSettingsDesc')"
    :enable-label="$t('settings.callRejectEnable')"
    :enable-desc="$t('settings.callRejectEnableDesc')"
    v-model:enabled="settings.enabled"
    :can-write="canWrite"
    :is-submitting="isSubmitting"
    @save="saveSettings"
  >
    <div class="space-y-1.5">
      <Label for="call_reject_message" class="text-xs">{{ $t('settings.callRejectMessage') }}</Label>
      <Textarea
        id="call_reject_message"
        v-model="settings.message"
        :rows="3"
        :disabled="!canWrite"
      />
      <p class="text-[11px] text-muted-foreground">{{ $t('settings.callRejectMessageDesc') }}</p>
    </div>
  </AccountSettingsPanel>
</template>
