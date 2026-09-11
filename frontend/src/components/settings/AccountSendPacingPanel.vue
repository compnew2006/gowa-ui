<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { sendPacingService } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Loader2 } from 'lucide-vue-next'

// Per-account campaign send pacing: WhatsApp flags accounts that burst, so
// each account gets a messages/minute budget enforced by the campaign worker
// before every send.
const props = defineProps<{
  accountId: string
  canWrite: boolean
}>()

const emit = defineEmits<{ (e: 'saved'): void }>()

const { t } = useI18n()

const enabled = ref(false)
const messagesPerMinute = ref(60)
const effectiveDefault = ref(0)
const isSubmitting = ref(false)

// Built in script: the composer's t() has no (key, fallback, named) overload.
const defaultHint = computed(() => {
  if (pacingSource.value === 'account') {
    return `This account: ${messagesPerMinute.value}/min (overrides server default)`
  }
  return effectiveDefault.value > 0
    ? `Server default: ${effectiveDefault.value}/min`
    : 'No server default — off unless enabled here'
})
const pacingSource = ref<'account' | 'server' | 'unlimited'>('unlimited')

onMounted(async () => {
  try {
    const res = await sendPacingService.getSettings(props.accountId)
    const s = res.data.data
    // Prefer the stored account value for the switch (a server default must
    // not look like an account override). Fall back to the legacy effective
    // field for backward compatibility with older backends.
    const stored = s.stored_messages_per_minute ?? (s.source ? 0 : s.messages_per_minute)
    pacingSource.value = s.source ?? (stored > 0 ? 'account' : 'unlimited')
    effectiveDefault.value = s.effective_messages_per_minute ?? s.messages_per_minute
    if (stored > 0) {
      enabled.value = true
      messagesPerMinute.value = stored
    }
  } catch {
    toast.error('Failed to load send pacing settings')
  }
})

async function save() {
  isSubmitting.value = true
  try {
    await sendPacingService.updateSettings(props.accountId, {
      messages_per_minute: enabled.value ? messagesPerMinute.value : 0
    })
    toast.success('Send pacing saved')
    emit('saved')
  } catch {
    toast.error('Failed to save send pacing')
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>{{ t('accounts.sendPacing.title', 'Campaign Send Pacing') }}</CardTitle>
      <CardDescription>
        {{
          t(
            'accounts.sendPacing.description',
            'Cap campaign sends per minute to protect this number from WhatsApp bans.'
          )
        }}
      </CardDescription>
    </CardHeader>
    <CardContent class="space-y-4">
      <div class="flex items-center justify-between rounded-lg border p-3">
        <div class="space-y-0.5">
          <Label>{{ t('accounts.sendPacing.enable', 'Limit sending speed') }}</Label>
          <p class="text-xs text-muted-foreground">{{ defaultHint }}</p>
        </div>
        <Switch :model-value="enabled" :disabled="!props.canWrite" @update:model-value="enabled = $event" />
      </div>

      <div v-if="enabled" class="space-y-2">
        <Label for="pace-per-minute">{{ t('accounts.sendPacing.perMinute', 'Messages per minute') }}</Label>
        <Input
          id="pace-per-minute"
          v-model.number="messagesPerMinute"
          type="number"
          min="1"
          max="1000"
          :disabled="!props.canWrite"
        />
        <p class="text-xs text-muted-foreground">
          {{
            t(
              'accounts.sendPacing.hint',
              '20–60/min is a safe range for most numbers; new numbers should start lower.'
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
