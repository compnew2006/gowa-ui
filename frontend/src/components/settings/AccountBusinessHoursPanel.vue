<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { businessHoursService } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { Loader2 } from 'lucide-vue-next'

// Per-account business hours: inbound 1:1 messages arriving while closed get
// the away reply (once per contact per cooldown — server-side guard).
const props = defineProps<{
  accountId: string
  canWrite: boolean
}>()

const emit = defineEmits<{ (e: 'saved'): void }>()

const { t } = useI18n()

const WEEKDAYS = [
  { value: 6, label: t('days.sat', 'Sat') },
  { value: 0, label: t('days.sun', 'Sun') },
  { value: 1, label: t('days.mon', 'Mon') },
  { value: 2, label: t('days.tue', 'Tue') },
  { value: 3, label: t('days.wed', 'Wed') },
  { value: 4, label: t('days.thu', 'Thu') },
  { value: 5, label: t('days.fri', 'Fri') }
]

const settings = ref({
  enabled: false,
  start_time: '09:00',
  end_time: '17:00',
  days: [0, 1, 2, 3, 4] as number[], // Egyptian work week default: Sun–Thu
  utc_offset_min: 120,
  away_message: ''
})
const isSubmitting = ref(false)
// The shadcn Input models strings; the numeric offset converts on save.
const utcOffsetInput = ref('120')

onMounted(async () => {
  try {
    const res = await businessHoursService.getSettings(props.accountId)
    const s = res.data.data
    if (s && s.start_time) {
      settings.value = {
        enabled: s.enabled,
        start_time: s.start_time,
        end_time: s.end_time,
        days: s.days?.length ? s.days : settings.value.days,
        utc_offset_min: s.utc_offset_min,
        away_message: s.away_message
      }
      utcOffsetInput.value = String(s.utc_offset_min)
    }
  } catch {
    toast.error('Failed to load business hours')
  }
})

function toggleDay(day: number, on: boolean) {
  if (on && !settings.value.days.includes(day)) {
    settings.value.days = [...settings.value.days, day].sort()
  } else if (!on) {
    settings.value.days = settings.value.days.filter((d) => d !== day)
  }
}

async function save() {
  isSubmitting.value = true
  try {
    await businessHoursService.updateSettings(props.accountId, {
      ...settings.value,
      utc_offset_min: parseInt(utcOffsetInput.value, 10) || 0
    })
    toast.success('Business hours saved')
    emit('saved')
  } catch {
    toast.error('Failed to save business hours')
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>{{ t('accounts.businessHours.title', 'Business Hours & Auto-Reply') }}</CardTitle>
      <CardDescription>
        {{
          t(
            'accounts.businessHours.description',
            'Customers writing outside these hours get the away message once, until you reopen.'
          )
        }}
      </CardDescription>
    </CardHeader>
    <CardContent class="space-y-4">
      <div class="flex items-center justify-between rounded-lg border p-3">
        <Label>{{ t('accounts.businessHours.enable', 'Enable business hours') }}</Label>
        <Switch
          :model-value="settings.enabled"
          :disabled="!props.canWrite"
          @update:model-value="settings.enabled = $event"
        />
      </div>

      <template v-if="settings.enabled">
        <div class="grid grid-cols-3 gap-3">
          <div class="space-y-2">
            <Label for="bh-start">{{ t('accounts.businessHours.open', 'Opens') }}</Label>
            <Input id="bh-start" v-model="settings.start_time" type="time" :disabled="!props.canWrite" />
          </div>
          <div class="space-y-2">
            <Label for="bh-end">{{ t('accounts.businessHours.close', 'Closes') }}</Label>
            <Input id="bh-end" v-model="settings.end_time" type="time" :disabled="!props.canWrite" />
            <p class="text-xs text-muted-foreground">
              {{ t('accounts.businessHours.overnightHint', 'Later than open = overnight window') }}
            </p>
          </div>
          <div class="space-y-2">
            <Label for="bh-tz">{{ t('accounts.businessHours.utcOffset', 'UTC offset (min)') }}</Label>
            <Input
              id="bh-tz"
              v-model="utcOffsetInput"
              type="number"
              min="-720"
              max="840"
              :disabled="!props.canWrite"
            />
            <p class="text-xs text-muted-foreground">{{ t('accounts.businessHours.tzHint', 'Egypt: 120 (winter) / 180 (summer)') }}</p>
          </div>
        </div>

        <div class="space-y-2">
          <Label>{{ t('accounts.businessHours.days', 'Open days') }}</Label>
          <div class="flex flex-wrap gap-2">
            <Button
              v-for="d in WEEKDAYS"
              :key="d.value"
              type="button"
              size="sm"
              :variant="settings.days.includes(d.value) ? 'default' : 'outline'"
              :disabled="!props.canWrite"
              @click="toggleDay(d.value, !settings.days.includes(d.value))"
            >
              {{ d.label }}
            </Button>
          </div>
        </div>

        <div class="space-y-2">
          <Label for="bh-away">{{ t('accounts.businessHours.awayMessage', 'Away message') }}</Label>
          <Textarea
            id="bh-away"
            v-model="settings.away_message"
            :rows="3"
            :placeholder="t('accounts.businessHours.awayPlaceholder', 'شكراً لرسالتك! ساعات العمل من ٩ صباحاً حتى ٥ مساءً وسنرد عليك في أقرب وقت.')"
            :disabled="!props.canWrite"
          />
        </div>
      </template>

      <Button :disabled="!props.canWrite || isSubmitting" @click="save">
        <Loader2 v-if="isSubmitting" class="mr-2 h-4 w-4 animate-spin" />
        {{ t('common.save', 'Save') }}
      </Button>
    </CardContent>
  </Card>
</template>
