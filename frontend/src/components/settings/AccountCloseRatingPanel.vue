<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { closeRatingService } from '@/services/api'
import type { CloseRatingResponse, CloseRatingResponseStatus, CloseRatingStats } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button, buttonVariants } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatDateTime } from '@/lib/utils'
import { Loader2, MessageSquare, Plus, Star, X } from 'lucide-vue-next'

// Per-account CSAT panel: each WhatsApp number belongs to a different branch
// with its own staff, address and wording, so prompt/thanks/lexicon are
// configured here on the account, not organization-wide.
const props = defineProps<{
  accountId: string
  canWrite: boolean
}>()

// Emitted after a successful save so the parent (AccountDetailView) can refresh
// the unified Activity Log. The close_rating audit entry shares the account's
// resource_id, so it surfaces in the parent's aggregated timeline.
const emit = defineEmits<{ (e: 'saved'): void }>()

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

// Customer ratings list: who was asked, who answered, and how to reach the chat.
const RATING_PAGE_SIZE = 20
const STAR_LEVELS = [5, 4, 3, 2, 1] as const
type StarFilter = 'all' | `${(typeof STAR_LEVELS)[number]}`
const RESPONSE_STATUSES: readonly CloseRatingResponseStatus[] = ['rated', 'pending', 'expired']

function isResponseStatus(value: unknown): value is CloseRatingResponseStatus {
  return RESPONSE_STATUSES.some((status) => status === value)
}

function isStarFilter(value: unknown): value is StarFilter {
  return value === 'all' || STAR_LEVELS.some((star) => String(star) === value)
}

const responseStatus = ref<CloseRatingResponseStatus>('rated')
const responseStar = ref<StarFilter>('all')
const responseItems = ref<CloseRatingResponse[]>([])
const responseTotal = ref(0)
const responsePage = ref(0)
const responsesLoading = ref(false)
const responsesLoadingMore = ref(false)
const responsesError = ref(false)
// Every fetch takes a sequence number and only the latest one may write state, so a
// slow response for a previous filter cannot overwrite the list being looked at.
let responsesSeq = 0

const hasMoreResponses = computed(() => responseItems.value.length < responseTotal.value)

const statusOptions = computed(() => [
  { value: 'rated', label: t('settings.ratingStatusRated'), count: ratingStats.value?.rated ?? 0 },
  { value: 'pending', label: t('settings.ratingPending'), count: ratingStats.value?.pending ?? 0 },
  { value: 'expired', label: t('settings.ratingStatusExpired'), count: ratingStats.value?.expired ?? 0 }
])

async function fetchResponses(reset: boolean) {
  const seq = ++responsesSeq
  const page = reset ? 1 : responsePage.value + 1
  const status = responseStatus.value
  const star = responseStar.value
  responsesError.value = false
  if (reset) {
    responseItems.value = []
    responseTotal.value = 0
    responsePage.value = 0
    responsesLoading.value = true
    responsesLoadingMore.value = false
  } else {
    responsesLoadingMore.value = true
  }
  try {
    const response = await closeRatingService.listResponses(props.accountId, {
      status,
      rating: status === 'rated' && star !== 'all' ? Number(star) : undefined,
      page,
      limit: RATING_PAGE_SIZE
    })
    if (seq !== responsesSeq) return
    const data = response.data.data
    const incoming = data.responses || []
    const known = new Set(responseItems.value.map((item) => item.id))
    responseItems.value = reset
      ? incoming
      : [...responseItems.value, ...incoming.filter((item) => !known.has(item.id))]
    responseTotal.value = data.total
    responsePage.value = page
  } catch (error) {
    if (seq !== responsesSeq) return
    console.error('Failed to load close-rating responses:', error)
    responsesError.value = true
  } finally {
    if (seq === responsesSeq) {
      responsesLoading.value = false
      responsesLoadingMore.value = false
    }
  }
}

function loadMoreResponses() {
  if (responsesLoadingMore.value || !hasMoreResponses.value) return
  void fetchResponses(false)
}

function retryResponses() {
  void fetchResponses(responseItems.value.length === 0)
}

// Any filter change restarts the list from page 1 of the new filter.
function applyResponseFilters(status: CloseRatingResponseStatus, star: StarFilter) {
  if (status === responseStatus.value && star === responseStar.value) return
  responseStatus.value = status
  responseStar.value = star
  void fetchResponses(true)
}

function onStatusTabChange(value: unknown) {
  if (!isResponseStatus(value)) return
  applyResponseFilters(value, responseStar.value)
}

function onStarFilterChange(value: unknown) {
  if (!isStarFilter(value)) return
  applyResponseFilters(responseStatus.value, value)
}

// Distribution rows are shortcuts into the rated list filtered to one star level.
function selectDistributionStar(star: number) {
  applyResponseFilters('rated', String(star) as StarFilter)
}

function isDistributionSelected(star: number): boolean {
  return responseStatus.value === 'rated' && responseStar.value === String(star)
}

// Rated rows show when the customer rated; the other rows show when the request was sent.
function responseTime(item: CloseRatingResponse): { label: string; value: string } {
  if (item.status === 'rated' && item.rated_at) {
    return { label: t('settings.ratingRatedAt'), value: item.rated_at }
  }
  return { label: t('settings.ratingSentAt'), value: item.created_at }
}

function chatLink(item: CloseRatingResponse) {
  return {
    name: 'chat-conversation',
    params: { contactId: item.contact_id },
    query: item.whatsapp_account ? { account: item.whatsapp_account } : undefined
  }
}

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
  // The list only shows when the account has rating activity, so it loads after the
  // stats and a failure there never touches them.
  if (ratingStats.value && ratingStats.value.total > 0) {
    void fetchResponses(true)
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
    <!-- Stats overview -->
    <Card v-if="ratingStats && ratingStats.total > 0">
      <CardHeader class="pb-3">
        <CardTitle class="text-sm font-medium">{{ $t('settings.ratingStats') }}</CardTitle>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div class="rounded-lg bg-muted/40 p-4">
            <p class="flex items-center gap-2 text-2xl font-semibold">
              <Star class="h-5 w-5 shrink-0 fill-amber-400 text-amber-400" aria-hidden="true" />
              {{ ratingStats.average.toFixed(1) }}
            </p>
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
          <button
            v-for="star in STAR_LEVELS"
            :key="star"
            type="button"
            :aria-pressed="isDistributionSelected(star)"
            :aria-label="`${star}/5: ${ratingStats.distribution?.[String(star)] || 0}`"
            class="-mx-2 flex w-[calc(100%+1rem)] items-center gap-3 rounded-md px-2 py-1 text-start transition-colors hover:bg-muted/40 aria-pressed:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            @click="selectDistributionStar(star)"
          >
            <span class="inline-flex w-10 items-center gap-1 text-sm text-muted-foreground">
              {{ star }}
              <Star class="h-3.5 w-3.5 fill-amber-400 text-amber-400" aria-hidden="true" />
            </span>
            <div class="flex-1 h-2 rounded-full bg-muted overflow-hidden">
              <div class="h-full rounded-full bg-amber-400" :style="{ width: distributionPercent(star) + '%' }" />
            </div>
            <span class="w-8 text-sm text-muted-foreground text-end tabular-nums">{{ ratingStats.distribution?.[String(star)] || 0 }}</span>
          </button>
        </div>
      </CardContent>
    </Card>

    <!-- Customer ratings: who answered, their number, and a link to the conversation -->
    <Card v-if="ratingStats && ratingStats.total > 0">
      <CardHeader class="pb-3">
        <CardTitle class="text-sm font-medium">{{ $t('settings.ratingResponsesList') }}</CardTitle>
        <CardDescription class="text-xs">{{ $t('settings.ratingResponsesListDesc') }}</CardDescription>
      </CardHeader>
      <CardContent class="space-y-3">
        <div class="flex flex-wrap items-center gap-2">
          <Tabs :model-value="responseStatus" @update:model-value="onStatusTabChange">
            <TabsList class="bg-muted/50 p-1">
              <TabsTrigger
                v-for="option in statusOptions"
                :key="option.value"
                :value="option.value"
                class="group gap-1.5 text-xs text-muted-foreground hover:text-foreground data-[state=active]:bg-emerald-600 data-[state=active]:text-white data-[state=active]:shadow-sm"
              >
                {{ option.label }}
                <span class="ms-1.5 text-[10px] tabular-nums text-muted-foreground group-data-[state=active]:text-white/80">{{ option.count }}</span>
              </TabsTrigger>
            </TabsList>
          </Tabs>
          <Select
            v-if="responseStatus === 'rated'"
            :model-value="responseStar"
            @update:model-value="onStarFilterChange"
          >
            <SelectTrigger class="ms-auto w-[10.5rem] text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ $t('settings.ratingAllStars') }}</SelectItem>
              <SelectItem v-for="star in STAR_LEVELS" :key="star" :value="String(star)" :text-value="`${star}/5`">
                <span role="img" :aria-label="`${star}/5`" class="inline-flex items-center gap-0.5">
                  <Star
                    v-for="n in 5"
                    :key="n"
                    aria-hidden="true"
                    class="h-3.5 w-3.5"
                    :class="n <= star ? 'fill-amber-400 text-amber-400' : 'text-muted-foreground/40'"
                  />
                </span>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div v-if="responsesLoading && responseItems.length === 0" class="divide-y divide-border/60" aria-busy="true">
          <div v-for="n in 3" :key="n" class="flex flex-col gap-2 py-3">
            <Skeleton class="h-4 w-40" />
            <Skeleton class="h-3 w-full" />
          </div>
        </div>

        <ul v-else-if="responseItems.length > 0" class="divide-y divide-border/60">
          <li
            v-for="item in responseItems"
            :key="item.id"
            class="flex flex-col gap-2 py-3 md:max-xl:flex-row md:max-xl:items-center md:max-xl:gap-4"
          >
            <!-- Customer and rating -->
            <div class="flex min-w-0 items-start justify-between gap-3 md:max-xl:w-56 md:max-xl:shrink-0">
              <div class="min-w-0">
                <p class="truncate text-sm font-medium" :dir="item.contact_name ? undefined : 'ltr'">
                  {{ item.contact_name || item.phone_number }}
                </p>
                <p v-if="item.contact_name" dir="ltr" class="truncate text-start text-xs text-muted-foreground">
                  {{ item.phone_number }}
                </p>
              </div>
              <span
                v-if="item.status === 'rated' && item.rating"
                role="img"
                :aria-label="`${item.rating}/5`"
                class="inline-flex shrink-0 items-center gap-0.5 pt-0.5"
              >
                <Star
                  v-for="n in 5"
                  :key="n"
                  aria-hidden="true"
                  class="h-3.5 w-3.5"
                  :class="n <= item.rating ? 'fill-amber-400 text-amber-400' : 'text-muted-foreground/40'"
                />
              </span>
            </div>

            <!-- Customer reply -->
            <p
              class="line-clamp-2 min-w-0 break-words text-sm md:max-xl:flex-1"
              :title="item.raw_reply || undefined"
            >{{ item.raw_reply }}</p>

            <!-- Meta and action -->
            <div class="flex shrink-0 items-center justify-between gap-3 md:max-xl:flex-col md:max-xl:items-end">
              <div class="min-w-0 space-y-0.5 text-xs text-muted-foreground">
                <p v-if="item.closed_by_user_name">{{ $t('settings.ratingClosedBy') }} {{ item.closed_by_user_name }}</p>
                <p>
                  {{ responseTime(item).label }}
                  <time :datetime="responseTime(item).value" class="tabular-nums">{{ formatDateTime(responseTime(item).value) }}</time>
                </p>
              </div>
              <RouterLink :to="chatLink(item)" :class="buttonVariants({ variant: 'outline', size: 'sm' })">
                <MessageSquare class="h-4 w-4" aria-hidden="true" />
                {{ $t('settings.ratingOpenChat') }}
              </RouterLink>
            </div>
          </li>
        </ul>

        <p v-else-if="!responsesError" class="py-8 text-center text-sm text-muted-foreground">
          {{ $t('settings.ratingNoResponses') }}
        </p>

        <div
          v-if="responsesError"
          role="alert"
          class="flex flex-wrap items-center justify-between gap-3 rounded-md bg-muted/40 px-3 py-2.5"
        >
          <p class="text-sm text-muted-foreground">{{ $t('settings.ratingResponsesFailed') }}</p>
          <Button variant="outline" size="sm" @click="retryResponses">{{ $t('settings.ratingRetry') }}</Button>
        </div>

        <div v-if="hasMoreResponses && !responsesError" class="flex justify-center">
          <Button variant="outline" size="sm" :loading="responsesLoadingMore" @click="loadMoreResponses">
            {{ $t('settings.ratingLoadMore') }}
          </Button>
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
  </div>
</template>
