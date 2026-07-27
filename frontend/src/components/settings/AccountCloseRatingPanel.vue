<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { closeRatingService } from '@/services/api'
import type { CloseRatingStats } from '@/services/api'
import AuditLogPanel from '@/components/shared/AuditLogPanel.vue'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Loader2, Plus, X } from 'lucide-vue-next'

// Per-account CSAT panel: each WhatsApp number belongs to a different branch
// with its own staff, address and wording, so prompt/thanks/lexicon are
// configured here on the account, not organization-wide.
const props = defineProps<{
  accountId: string
  canWrite: boolean
}>()

const { t } = useI18n()

// The lexicon is edited as rows and re-assembled into a word→rating record on save.
const ratingSettings = ref({
  enabled: false,
  window_hours: 48,
  prompt: '',
  thanks: ''
})
const lexiconRows = ref<Array<{ word: string; rating: string }>>([])
const ratingStats = ref<CloseRatingStats | null>(null)
const isSubmitting = ref(false)

// Bumped after a save to remount the AuditLogPanel; the backend writes audit
// entries asynchronously, so the remount is delayed slightly.
const ratingLogKey = ref(0)

onMounted(async () => {
  // Loaded independently: a stats failure must not blank the settings form
  // (a rejected Promise.all here once made saved settings look lost).
  try {
    const ratingResponse = await closeRatingService.getSettings(props.accountId)
    const rating = ratingResponse.data.data
    if (rating) {
      ratingSettings.value = {
        enabled: rating.enabled,
        window_hours: rating.window_hours,
        prompt: rating.prompt,
        thanks: rating.thanks
      }
      lexiconRows.value = Object.entries(rating.lexicon || {}).map(([word, r]) => ({
        word,
        rating: String(r)
      }))
    }
  } catch (error) {
    console.error('Failed to load close-rating settings:', error)
  }
  try {
    const statsResponse = await closeRatingService.getStats(props.accountId)
    ratingStats.value = statsResponse.data.data || null
  } catch (error) {
    console.error('Failed to load close-rating stats:', error)
  }
})

function addLexiconRow() {
  lexiconRows.value.push({ word: '', rating: '5' })
}

function removeLexiconRow(index: number) {
  lexiconRows.value.splice(index, 1)
}

function distributionPercent(star: number): number {
  const stats = ratingStats.value
  if (!stats || stats.rated === 0) return 0
  return ((stats.distribution?.[String(star)] || 0) / stats.rated) * 100
}

async function saveRatingSettings() {
  isSubmitting.value = true
  try {
    const lexicon: Record<string, number> = {}
    for (const row of lexiconRows.value) {
      const word = row.word.trim()
      if (word) lexicon[word] = Number(row.rating)
    }
    await closeRatingService.updateSettings(props.accountId, {
      enabled: ratingSettings.value.enabled,
      window_hours: Number(ratingSettings.value.window_hours) || 48,
      prompt: ratingSettings.value.prompt,
      thanks: ratingSettings.value.thanks,
      lexicon
    })
    toast.success(t('settings.ratingSaved'))
    setTimeout(() => { ratingLogKey.value++ }, 500)
  } catch (error) {
    toast.error(t('common.failedSave', { resource: t('resources.settings') }))
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <!-- Stats overview -->
    <Card v-if="ratingStats && ratingStats.total > 0">
      <CardHeader class="pb-3">
        <CardTitle class="text-sm font-medium">{{ $t('settings.ratingStats') }}</CardTitle>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div class="rounded-lg bg-muted/40 p-4">
            <p class="text-2xl font-semibold">⭐ {{ ratingStats.average.toFixed(1) }}</p>
            <p class="text-sm text-muted-foreground">{{ $t('settings.ratingAverage') }}</p>
          </div>
          <div class="rounded-lg bg-muted/40 p-4">
            <p class="text-2xl font-semibold">{{ ratingStats.rated }}</p>
            <p class="text-sm text-muted-foreground">{{ $t('settings.ratingResponses') }}</p>
          </div>
          <div class="rounded-lg bg-muted/40 p-4">
            <p class="text-2xl font-semibold">{{ ratingStats.response_rate.toFixed(0) }}%</p>
            <p class="text-sm text-muted-foreground">{{ $t('settings.ratingResponseRate') }}</p>
          </div>
          <div class="rounded-lg bg-muted/40 p-4">
            <p class="text-2xl font-semibold">{{ ratingStats.pending }}</p>
            <p class="text-sm text-muted-foreground">{{ $t('settings.ratingPending') }}</p>
          </div>
        </div>
        <div class="space-y-1.5">
          <div v-for="star in [5, 4, 3, 2, 1]" :key="star" class="flex items-center gap-3">
            <span class="w-8 text-sm text-muted-foreground">{{ star }} ★</span>
            <div class="flex-1 h-2 rounded-full bg-muted overflow-hidden">
              <div class="h-full rounded-full bg-amber-400" :style="{ width: distributionPercent(star) + '%' }" />
            </div>
            <span class="w-8 text-sm text-muted-foreground text-right">{{ ratingStats.distribution?.[String(star)] || 0 }}</span>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Settings -->
    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="text-sm font-medium">{{ $t('settings.ratingSettings') }}</CardTitle>
        <CardDescription class="text-xs">{{ $t('settings.ratingSettingsDesc') }}</CardDescription>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <Label class="text-xs">{{ $t('settings.ratingEnable') }}</Label>
            <p class="text-[11px] text-muted-foreground">{{ $t('settings.ratingEnableDesc') }}</p>
          </div>
          <Switch
            :checked="ratingSettings.enabled"
            @update:checked="ratingSettings.enabled = $event"
            :disabled="!canWrite"
          />
        </div>
        <Separator />
        <div class="space-y-1.5 max-w-xs">
          <Label for="rating_window" class="text-xs">{{ $t('settings.ratingWindowHours') }}</Label>
          <Input
            id="rating_window"
            v-model.number="ratingSettings.window_hours"
            type="number"
            min="1"
            max="720"
            :disabled="!canWrite"
          />
          <p class="text-[11px] text-muted-foreground">{{ $t('settings.ratingWindowHoursDesc') }}</p>
        </div>
        <div class="space-y-1.5">
          <Label for="rating_prompt" class="text-xs">{{ $t('settings.ratingPrompt') }}</Label>
          <Textarea
            id="rating_prompt"
            v-model="ratingSettings.prompt"
            :rows="3"
            :placeholder="$t('settings.ratingPromptPlaceholder')"
            :disabled="!canWrite"
          />
        </div>
        <div class="space-y-1.5">
          <Label for="rating_thanks" class="text-xs">{{ $t('settings.ratingThanks') }}</Label>
          <Textarea
            id="rating_thanks"
            v-model="ratingSettings.thanks"
            :rows="2"
            :disabled="!canWrite"
          />
          <p class="text-[11px] text-muted-foreground">{{ $t('settings.ratingThanksDesc') }}</p>
        </div>
        <Separator />
        <div class="space-y-3">
          <div>
            <Label class="text-xs">{{ $t('settings.ratingLexicon') }}</Label>
            <p class="text-[11px] text-muted-foreground">{{ $t('settings.ratingLexiconDesc') }}</p>
          </div>
          <div v-for="(row, index) in lexiconRows" :key="index" class="flex items-center gap-2">
            <Input
              v-model="row.word"
              class="flex-1"
              :placeholder="$t('settings.ratingLexiconWordPlaceholder')"
              :disabled="!canWrite"
            />
            <Select v-model="row.rating" :disabled="!canWrite">
              <SelectTrigger class="w-24">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="n in [5, 4, 3, 2, 1]" :key="n" :value="String(n)">{{ n }} ★</SelectItem>
              </SelectContent>
            </Select>
            <Button v-if="canWrite" variant="ghost" size="icon" @click="removeLexiconRow(index)">
              <X class="h-4 w-4" />
            </Button>
          </div>
          <Button v-if="canWrite" variant="outline" size="sm" @click="addLexiconRow">
            <Plus class="h-4 w-4 mr-1" />
            {{ $t('settings.ratingAddWord') }}
          </Button>
        </div>
        <div v-if="canWrite" class="flex justify-end">
          <Button variant="outline" size="sm" @click="saveRatingSettings" :disabled="isSubmitting">
            <Loader2 v-if="isSubmitting" class="mr-2 h-4 w-4 animate-spin" />
            {{ $t('settings.save') }}
          </Button>
        </div>
      </CardContent>
    </Card>

    <AuditLogPanel :key="ratingLogKey" resource-type="settings.close_rating" :resource-id="accountId" />
  </div>
</template>
