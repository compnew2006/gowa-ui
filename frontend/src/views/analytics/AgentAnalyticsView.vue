<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { RangeCalendar } from '@/components/ui/range-calendar'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { agentAnalyticsService } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useUsersStore } from '@/stores/users'
import { useInstancesStore } from '@/stores/instances'
import { PageHeader } from '@/components/shared'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList
} from '@/components/ui/command'
import {
  Clock,
  CheckCircle,
  MessageSquare,
  CalendarIcon,
  BarChart3,
  Activity,
  ChevronsUpDown,
  Check,
  Coffee,
  Download,
  Star,
  Loader2
} from 'lucide-vue-next'
import type { DateRange } from 'reka-ui'
import { CalendarDate } from '@internationalized/date'
// Centralized Chart.js setup (registered once)
import { Line, Bar, Doughnut } from '@/lib/charts'

interface AgentAnalyticsSummary {
  total_transfers_handled: number
  active_transfers: number
  avg_queue_time_mins: number
  avg_first_response_mins: number
  avg_resolution_mins: number
  transfers_by_source: Record<string, number>
  total_break_time_mins: number
  break_count: number
}

interface AgentPerformanceStats {
  agent_id: string
  agent_name: string
  avg_first_response_mins: number
  avg_resolution_mins: number
  transfers_handled: number
  active_transfers: number
  messages_sent: number
  total_break_time_mins: number
  break_count: number
  is_available: boolean
  current_break_start?: string
}

interface TrendPoint {
  date: string
  transfers_handled: number
  avg_response_mins: number
}

interface AgentRatingSummary {
  total_ratings: number
  average_rating: number
  ratings_by_score: Record<string, number>
}

interface AgentRatingRecord {
  id: string
  chat_id: string
  contact_id: string
  contact: string
  contact_phone: string
  agent_id?: string
  agent_name: string
  closing_agent_id: string
  closing_agent_name: string
  rating: number
  rated_at: string
  rating_message: string
  context_messages: Record<string, unknown>
}

interface AgentAnalyticsResponse {
  summary: AgentAnalyticsSummary
  agent_stats?: AgentPerformanceStats[]
  trend_data: TrendPoint[]
  my_stats?: AgentPerformanceStats
  rating_summary?: AgentRatingSummary
  rating_records?: AgentRatingRecord[]
}

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const usersStore = useUsersStore()
const instancesStore = useInstancesStore()
const canViewOrganizationAnalytics = computed(() => authStore.hasPermission('analytics', 'read'))
const canUseAgentFilter = computed(() => canViewOrganizationAnalytics.value && authStore.hasPermission('users', 'read'))

const analytics = ref<AgentAnalyticsResponse | null>(null)
const isLoading = ref(true)

// Agent filter for users who can review org-wide analytics
interface Agent {
  id: string
  full_name: string
  role: string
}
const agents = ref<Agent[]>([])
const selectedAgentId = ref<string>('all')
const agentComboboxOpen = ref(false)

const selectedAgentName = computed(() => {
  if (selectedAgentId.value === 'all') return t('agentAnalytics.allAgents')
  const agent = agents.value.find(a => a.id === selectedAgentId.value)
  return agent?.full_name || t('agentAnalytics.selectAgent')
})

const minRating = ref<string>('all')
const maxRating = ref<string>('all')
const selectedInstanceId = ref<string>('all')
const isExporting = ref(false)
const ratingOptions = Array.from({ length: 10 }, (_, idx) => {
  const value = String(idx + 1)
  return { value, label: value }
})

// Time range filter
type TimeRangePreset = 'today' | '7days' | '30days' | 'this_month' | 'custom'

const loadSavedPreferences = () => {
  const savedRange = localStorage.getItem('agent_analytics_time_range') as TimeRangePreset | null
  const savedCustomRange = localStorage.getItem('agent_analytics_custom_range')

  let customRange: DateRange = { start: undefined, end: undefined }
  if (savedCustomRange) {
    try {
      const parsed = JSON.parse(savedCustomRange)
      if (parsed.start && parsed.end) {
        customRange = {
          start: new CalendarDate(parsed.start.year, parsed.start.month, parsed.start.day),
          end: new CalendarDate(parsed.end.year, parsed.end.month, parsed.end.day)
        }
      }
    } catch (e) {
      console.error('Failed to parse saved custom range:', e)
    }
  }

  return {
    range: savedRange || 'this_month',
    customRange
  }
}

const savedPrefs = loadSavedPreferences()
const selectedRange = ref<TimeRangePreset>(savedPrefs.range as TimeRangePreset)
const customDateRange = ref<any>(savedPrefs.customRange)
const isDatePickerOpen = ref(false)

const savePreferences = () => {
  localStorage.setItem('agent_analytics_time_range', selectedRange.value)
  if (selectedRange.value === 'custom' && customDateRange.value.start && customDateRange.value.end) {
    localStorage.setItem('agent_analytics_custom_range', JSON.stringify({
      start: {
        year: customDateRange.value.start.year,
        month: customDateRange.value.start.month,
        day: customDateRange.value.start.day
      },
      end: {
        year: customDateRange.value.end.year,
        month: customDateRange.value.end.month,
        day: customDateRange.value.end.day
      }
    }))
  }
}

const formatDateLocal = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const getDateRange = computed(() => {
  const now = new Date()
  let from: Date
  let to: Date = now

  switch (selectedRange.value) {
    case 'today':
      from = new Date(now.getFullYear(), now.getMonth(), now.getDate())
      to = new Date(now.getFullYear(), now.getMonth(), now.getDate())
      break
    case '7days':
      from = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 7)
      to = new Date(now.getFullYear(), now.getMonth(), now.getDate())
      break
    case '30days':
      from = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 30)
      to = new Date(now.getFullYear(), now.getMonth(), now.getDate())
      break
    case 'this_month':
      from = new Date(now.getFullYear(), now.getMonth(), 1)
      to = new Date(now.getFullYear(), now.getMonth(), now.getDate())
      break
    case 'custom':
      if (customDateRange.value.start && customDateRange.value.end) {
        from = new Date(customDateRange.value.start.year, customDateRange.value.start.month - 1, customDateRange.value.start.day)
        to = new Date(customDateRange.value.end.year, customDateRange.value.end.month - 1, customDateRange.value.end.day)
      } else {
        from = new Date(now.getFullYear(), now.getMonth(), 1)
        to = new Date(now.getFullYear(), now.getMonth(), now.getDate())
      }
      break
    default:
      from = new Date(now.getFullYear(), now.getMonth(), 1)
      to = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  }

  return {
    from: formatDateLocal(from),
    to: formatDateLocal(to)
  }
})

const formatDateRange = computed(() => {
  if (selectedRange.value === 'custom' && customDateRange.value.start && customDateRange.value.end) {
    const start = customDateRange.value.start
    const end = customDateRange.value.end
    const startStr = `${start.month}/${start.day}/${start.year}`
    const endStr = `${end.month}/${end.day}/${end.year}`
    return `${startStr} - ${endStr}`
  }
  return ''
})

const formatDateTime = (value: string): string => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

const openRatingChat = async (record: AgentRatingRecord) => {
  if (!record.contact_id) return
  await router.push({ name: 'chat-conversation', params: { contactId: record.contact_id } })
}

const extractFollowupComments = (context: Record<string, unknown> | undefined): string[] => {
  if (!context) return []
  const rawFollowup = context.followup
  if (!rawFollowup || typeof rawFollowup !== 'object') return []

  const followup = rawFollowup as Record<string, unknown>
  const comments: string[] = []
  const seen = new Set<string>()

  const appendIfValid = (value: unknown) => {
    if (typeof value !== 'string') return
    const trimmed = value.trim()
    if (!trimmed || seen.has(trimmed)) return
    seen.add(trimmed)
    comments.push(trimmed)
  }

  const rawComments = followup.comments
  if (Array.isArray(rawComments)) {
    rawComments.forEach(appendIfValid)
  }

  const rawEntries = followup.entries
  if (Array.isArray(rawEntries)) {
    rawEntries.forEach((entry) => {
      if (!entry || typeof entry !== 'object') return
      const typed = entry as Record<string, unknown>
      if (typed.kind !== 'comment') return
      appendIfValid(typed.content)
    })
  }

  return comments
}

const formatRatingMessage = (record: AgentRatingRecord): string => {
  const base = typeof record.rating_message === 'string' ? record.rating_message.trim() : ''
  const comments = extractFollowupComments(record.context_messages)
  const merged: string[] = []

  const appendIfUnique = (value: string) => {
    const trimmed = value.trim()
    if (!trimmed) return
    if (merged.includes(trimmed)) return
    merged.push(trimmed)
  }

  if (base) {
    base
      .split('|')
      .map((part) => part.trim())
      .filter(Boolean)
      .forEach(appendIfUnique)
  }
  comments.forEach(appendIfUnique)

  return merged.join(' | ') || '-'
}

const formatContextMessages = (context: Record<string, unknown> | undefined): string => {
  if (!context) return '-'

  const collect = (key: string): string[] => {
    const value = context[key]
    if (!Array.isArray(value)) return []
    return value
      .map((entry) => {
        if (entry && typeof entry === 'object' && 'content' in entry) {
          return String((entry as { content?: unknown }).content ?? '').trim()
        }
        return ''
      })
      .filter(Boolean)
  }

  let ratingPart = ''
  const ratingValue = context.rating
  if (ratingValue && typeof ratingValue === 'object' && 'content' in ratingValue) {
    ratingPart = String((ratingValue as { content?: unknown }).content ?? '').trim()
  }

  const parts = [...collect('before')]
  if (ratingPart) parts.push(ratingPart)
  parts.push(...collect('after'))
  parts.push(...extractFollowupComments(context))

  return parts.join(' | ') || '-'
}

const formatMinutes = (mins: number): string => {
  if (!mins || mins === 0) return '0m'
  if (mins < 60) return `${Math.round(mins)}m`
  const hours = Math.floor(mins / 60)
  const remainingMins = Math.round(mins % 60)
  return remainingMins > 0 ? `${hours}h ${remainingMins}m` : `${hours}h`
}

const fetchAgents = async () => {
  if (!canUseAgentFilter.value) return
  try {
    await usersStore.fetchUsers()
    agents.value = usersStore.users
      .map((u) => ({ id: u.id, full_name: u.full_name, role: u.role?.name || '' }))
  } catch (error) {
    console.error('Failed to load agents:', error)
  }
}

const fetchAnalytics = async () => {
  isLoading.value = true
  try {
    const { from, to } = getDateRange.value
    const params: { from: string; to: string; agent_id?: string; instance_id?: string; min_rating?: number; max_rating?: number } = { from, to }
    if (canViewOrganizationAnalytics.value && selectedAgentId.value !== 'all') {
      params.agent_id = selectedAgentId.value
    }
    if (canViewOrganizationAnalytics.value && selectedInstanceId.value !== 'all') {
      params.instance_id = selectedInstanceId.value
    }
    if (canViewOrganizationAnalytics.value && minRating.value !== 'all') {
      params.min_rating = Number(minRating.value)
    }
    if (canViewOrganizationAnalytics.value && maxRating.value !== 'all') {
      params.max_rating = Number(maxRating.value)
    }
    const response = await agentAnalyticsService.getSummary(params)
    const data = response.data.data || response.data
    analytics.value = data
  } catch (error) {
    console.error('Failed to load agent analytics:', error)
    analytics.value = null
  } finally {
    isLoading.value = false
  }
}

const exportRatings = async () => {
  if (!canViewOrganizationAnalytics.value) return

  isExporting.value = true
  try {
    const { from, to } = getDateRange.value
    const params: { from: string; to: string; agent_id?: string; instance_id?: string; min_rating?: number; max_rating?: number } = { from, to }
    if (selectedAgentId.value !== 'all') {
      params.agent_id = selectedAgentId.value
    }
    if (selectedInstanceId.value !== 'all') {
      params.instance_id = selectedInstanceId.value
    }
    if (minRating.value !== 'all') {
      params.min_rating = Number(minRating.value)
    }
    if (maxRating.value !== 'all') {
      params.max_rating = Number(maxRating.value)
    }

    const response = await agentAnalyticsService.exportRatings(params)
    const csvBlob = response.data instanceof Blob
      ? response.data
      : new Blob([response.data], { type: 'text/csv' })

    const url = URL.createObjectURL(csvBlob)
    const link = document.createElement('a')
    link.href = url
    link.download = `agent-ratings-${new Date().toISOString().replace(/[:.]/g, '-')}.csv`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Failed to export ratings:', error)
  } finally {
    isExporting.value = false
  }
}

const applyCustomRange = () => {
  if (customDateRange.value.start && customDateRange.value.end) {
    isDatePickerOpen.value = false
    savePreferences()
    fetchAnalytics()
  }
}

watch(selectedRange, (newValue) => {
  savePreferences()
  if (newValue !== 'custom') {
    fetchAnalytics()
  }
})

watch(selectedAgentId, () => {
  fetchAnalytics()
})

watch(selectedInstanceId, () => {
  fetchAnalytics()
})

watch([minRating, maxRating], ([nextMin, nextMax]) => {
  if (nextMin !== 'all' && nextMax !== 'all' && Number(nextMin) > Number(nextMax)) {
    maxRating.value = nextMin
    return
  }
  fetchAnalytics()
})

onMounted(() => {
  fetchAgents()
  if (canViewOrganizationAnalytics.value) {
    instancesStore.fetchInstances()
  }
  fetchAnalytics()
})

// Chart configurations
const trendChartData = computed(() => {
  if (!analytics.value?.trend_data?.length) {
    return {
      labels: [],
      datasets: []
    }
  }

  return {
    labels: analytics.value.trend_data.map(t => {
      const date = new Date(t.date)
      return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
    }),
    datasets: [
      {
        label: t('agentAnalytics.transfersHandled'),
        data: analytics.value.trend_data.map(d => d.transfers_handled),
        borderColor: 'rgb(59, 130, 246)',
        backgroundColor: 'rgba(59, 130, 246, 0.1)',
        fill: true,
        tension: 0.3
      }
    ]
  }
})

const trendChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false
    }
  },
  scales: {
    y: {
      beginAtZero: true,
      ticks: {
        stepSize: 1
      }
    }
  }
}

const sourceChartData = computed(() => {
  if (!analytics.value?.summary?.transfers_by_source) {
    return {
      labels: [],
      datasets: []
    }
  }

  const sources = analytics.value.summary.transfers_by_source
  const labels = Object.keys(sources).map(s => s.charAt(0).toUpperCase() + s.slice(1))
  const data = Object.values(sources)

  return {
    labels,
    datasets: [
      {
        data,
        backgroundColor: [
          'rgba(59, 130, 246, 0.8)',
          'rgba(16, 185, 129, 0.8)',
          'rgba(245, 158, 11, 0.8)',
          'rgba(139, 92, 246, 0.8)'
        ],
        borderWidth: 0
      }
    ]
  }
})

const sourceChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      position: 'bottom' as const
    }
  }
}

const comparisonChartData = computed(() => {
  if (!analytics.value?.agent_stats?.length) {
    return {
      labels: [],
      datasets: []
    }
  }

  return {
    labels: analytics.value.agent_stats.map(a => a.agent_name || 'Unknown'),
    datasets: [
      {
        label: t('agentAnalytics.transfersHandled'),
        data: analytics.value.agent_stats.map(a => a.transfers_handled),
        backgroundColor: 'rgba(59, 130, 246, 0.8)'
      },
      {
        label: t('agentAnalytics.messagesSent'),
        data: analytics.value.agent_stats.map(a => a.messages_sent),
        backgroundColor: 'rgba(16, 185, 129, 0.8)'
      }
    ]
  }
})

const comparisonChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      position: 'bottom' as const
    }
  },
  scales: {
    y: {
      beginAtZero: true
    }
  }
}

// Stats to display based on role (reserved for future use)
const _displayStats = computed(() => {
  if (canViewOrganizationAnalytics.value) {
    return analytics.value?.summary
  }
  return analytics.value?.my_stats
})
void _displayStats.value // Suppress unused warning
</script>

<template>
  <div class="flex flex-col h-full">
    <PageHeader
      :title="$t('agentAnalytics.title')"
      :description="canViewOrganizationAnalytics ? $t('agentAnalytics.subtitle') : $t('agentAnalytics.myMetrics')"
      :icon="BarChart3"
      icon-gradient="bg-gradient-to-br from-blue-500 to-indigo-600 shadow-blue-500/20"
    >
      <template #actions>
        <!-- Agent Filter -->
        <div v-if="canUseAgentFilter" class="flex items-center gap-2 mr-4">
          <Popover v-model:open="agentComboboxOpen">
            <PopoverTrigger as-child>
              <Button variant="outline" role="combobox" :aria-expanded="agentComboboxOpen" class="w-[200px] justify-between">
                <span class="truncate">{{ selectedAgentName }}</span>
                <ChevronsUpDown class="ml-2 h-4 w-4 shrink-0 opacity-50" />
              </Button>
            </PopoverTrigger>
            <PopoverContent class="w-[200px] p-0">
              <Command>
                <CommandInput :placeholder="$t('agentAnalytics.searchAgent')" />
                <CommandList>
                  <CommandEmpty>{{ $t('agentAnalytics.noAgentFound') }}</CommandEmpty>
                  <CommandGroup>
                    <CommandItem
                      value="all"
                      @select="() => { selectedAgentId = 'all'; agentComboboxOpen = false }"
                    >
                      <Check :class="['mr-2 h-4 w-4', selectedAgentId === 'all' ? 'opacity-100' : 'opacity-0']" />
                      {{ $t('agentAnalytics.allAgents') }}
                    </CommandItem>
                    <CommandItem
                      v-for="agent in agents"
                      :key="agent.id"
                      :value="agent.full_name"
                      @select="() => { selectedAgentId = agent.id; agentComboboxOpen = false }"
                    >
                      <Check :class="['mr-2 h-4 w-4', selectedAgentId === agent.id ? 'opacity-100' : 'opacity-0']" />
                      {{ agent.full_name }}
                    </CommandItem>
                  </CommandGroup>
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>
        </div>

        <div v-if="canViewOrganizationAnalytics" class="flex items-center gap-2 mr-4">
          <Select v-model="selectedInstanceId">
            <SelectTrigger class="w-[220px]" data-testid="agent-analytics-instance-filter">
              <SelectValue :placeholder="$t('chat.instance')">
                <span v-if="selectedInstanceId === 'all'">{{ $t('common.all') }}</span>
                <span v-else>
                  {{
                    instancesStore.instances.find((instance) => instance.id === selectedInstanceId)?.name
                    || selectedInstanceId
                  }}
                </span>
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ $t('common.all') }}</SelectItem>
              <SelectItem
                v-for="instance in instancesStore.instances"
                :key="instance.id"
                :value="instance.id"
              >
                {{ instance.name || instance.id }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- Time Range Filter -->
        <div class="flex items-center gap-2">
          <Select v-model="selectedRange">
            <SelectTrigger class="w-[180px]">
              <SelectValue :placeholder="$t('agentAnalytics.selectRange')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="today">{{ $t('agentAnalytics.today') }}</SelectItem>
              <SelectItem value="7days">{{ $t('agentAnalytics.last7Days') }}</SelectItem>
              <SelectItem value="30days">{{ $t('agentAnalytics.last30Days') }}</SelectItem>
              <SelectItem value="this_month">{{ $t('agentAnalytics.thisMonth') }}</SelectItem>
              <SelectItem value="custom">{{ $t('agentAnalytics.customRange') }}</SelectItem>
            </SelectContent>
          </Select>

          <Popover v-if="selectedRange === 'custom'" v-model:open="isDatePickerOpen">
            <PopoverTrigger as-child>
              <Button variant="outline" class="w-auto">
                <CalendarIcon class="h-4 w-4 mr-2" />
                {{ formatDateRange || $t('agentAnalytics.selectDates') }}
              </Button>
            </PopoverTrigger>
            <PopoverContent class="w-auto p-4" align="end">
              <div class="space-y-4">
                <RangeCalendar v-model="customDateRange" :number-of-months="2" />
                <Button class="w-full" @click="applyCustomRange" :disabled="!customDateRange.start || !customDateRange.end">
                  {{ $t('agentAnalytics.applyRange') }}
                </Button>
              </div>
            </PopoverContent>
          </Popover>
        </div>

        <div v-if="canViewOrganizationAnalytics" class="flex items-center gap-2">
          <Select v-model="minRating">
            <SelectTrigger class="w-[120px]">
              <SelectValue :placeholder="$t('agentAnalytics.minRating')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ $t('agentAnalytics.anyRating') }}</SelectItem>
              <SelectItem
                v-for="option in ratingOptions"
                :key="`min-${option.value}`"
                :value="option.value"
              >
                {{ option.label }}
              </SelectItem>
            </SelectContent>
          </Select>

          <Select v-model="maxRating">
            <SelectTrigger class="w-[120px]">
              <SelectValue :placeholder="$t('agentAnalytics.maxRating')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ $t('agentAnalytics.anyRating') }}</SelectItem>
              <SelectItem
                v-for="option in ratingOptions"
                :key="`max-${option.value}`"
                :value="option.value"
              >
                {{ option.label }}
              </SelectItem>
            </SelectContent>
          </Select>

          <Button variant="outline" size="sm" :disabled="isExporting" @click="exportRatings">
            <Loader2 v-if="isExporting" class="mr-2 h-4 w-4 animate-spin" />
            <Download v-else class="mr-2 h-4 w-4" />
            {{ $t('agentAnalytics.exportRatings') }}
          </Button>
        </div>
      </template>
    </PageHeader>

    <!-- Content -->
    <ScrollArea class="flex-1">
      <div class="p-6 space-y-6">
        <!-- Stats Cards -->
        <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
          <template v-if="isLoading">
            <div v-for="i in 5" :key="i" class="rounded-xl border border-white/[0.08] bg-white/[0.02] p-6 light:bg-white light:border-gray-200">
              <div class="flex flex-row items-center justify-between space-y-0 pb-2">
                <Skeleton class="h-4 w-24 bg-white/[0.08] light:bg-gray-200" />
                <Skeleton class="h-10 w-10 rounded-lg bg-white/[0.08] light:bg-gray-200" />
              </div>
              <div class="pt-2">
                <Skeleton class="h-8 w-20 mb-2 bg-white/[0.08] light:bg-gray-200" />
                <Skeleton class="h-3 w-32 bg-white/[0.08] light:bg-gray-200" />
              </div>
            </div>
          </template>
          <template v-else-if="analytics">
            <!-- Transfers Handled -->
            <div class="card-depth rounded-xl border border-white/[0.08] bg-white/[0.04] p-6 light:bg-white light:border-gray-200">
              <div class="flex flex-row items-center justify-between space-y-0 pb-2">
                <span class="text-sm font-medium text-white/50 light:text-gray-500">{{ $t('agentAnalytics.transfersHandled') }}</span>
                <div class="h-10 w-10 rounded-lg bg-emerald-500/20 flex items-center justify-center">
                  <CheckCircle class="h-5 w-5 text-emerald-400" />
                </div>
              </div>
              <div class="pt-2">
                <div class="text-3xl font-bold text-white light:text-gray-900">
                  {{ selectedAgentId === 'all'
                    ? (analytics.summary?.total_transfers_handled ?? 0)
                    : (analytics.my_stats?.transfers_handled ?? 0) }}
                </div>
                <p class="text-xs text-white/40 light:text-gray-500 mt-1">{{ $t('agentAnalytics.completedConversations') }}</p>
              </div>
            </div>

            <!-- Active Conversations -->
            <div class="card-depth rounded-xl border border-white/[0.08] bg-white/[0.04] p-6 light:bg-white light:border-gray-200">
              <div class="flex flex-row items-center justify-between space-y-0 pb-2">
                <span class="text-sm font-medium text-white/50 light:text-gray-500">{{ $t('agentAnalytics.activeConversations') }}</span>
                <div class="h-10 w-10 rounded-lg bg-blue-500/20 flex items-center justify-center">
                  <Activity class="h-5 w-5 text-blue-400" />
                </div>
              </div>
              <div class="pt-2">
                <div class="text-3xl font-bold text-white light:text-gray-900">
                  {{ selectedAgentId === 'all'
                    ? (analytics.summary?.active_transfers ?? 0)
                    : (analytics.my_stats?.active_transfers ?? 0) }}
                </div>
                <p class="text-xs text-white/40 light:text-gray-500 mt-1">{{ $t('agentAnalytics.currentlyInProgress') }}</p>
              </div>
            </div>

            <!-- Avg Resolution Time -->
            <div class="card-depth rounded-xl border border-white/[0.08] bg-white/[0.04] p-6 light:bg-white light:border-gray-200">
              <div class="flex flex-row items-center justify-between space-y-0 pb-2">
                <span class="text-sm font-medium text-white/50 light:text-gray-500">{{ $t('agentAnalytics.avgResolutionTime') }}</span>
                <div class="h-10 w-10 rounded-lg bg-orange-500/20 flex items-center justify-center">
                  <Clock class="h-5 w-5 text-orange-400" />
                </div>
              </div>
              <div class="pt-2">
                <div class="text-3xl font-bold text-white light:text-gray-900">
                  {{ formatMinutes(selectedAgentId === 'all'
                    ? (analytics.summary?.avg_resolution_mins ?? 0)
                    : (analytics.my_stats?.avg_resolution_mins ?? 0)) }}
                </div>
                <p class="text-xs text-white/40 light:text-gray-500 mt-1">{{ $t('agentAnalytics.timeToResolve') }}</p>
              </div>
            </div>

            <!-- Messages Sent (for specific agent) or Queue Time (for all agents) -->
            <div v-if="canViewOrganizationAnalytics && selectedAgentId === 'all'" class="card-depth rounded-xl border border-white/[0.08] bg-white/[0.04] p-6 light:bg-white light:border-gray-200">
              <div class="flex flex-row items-center justify-between space-y-0 pb-2">
                <span class="text-sm font-medium text-white/50 light:text-gray-500">{{ $t('agentAnalytics.avgQueueTime') }}</span>
                <div class="h-10 w-10 rounded-lg bg-purple-500/20 flex items-center justify-center">
                  <Clock class="h-5 w-5 text-purple-400" />
                </div>
              </div>
              <div class="pt-2">
                <div class="text-3xl font-bold text-white light:text-gray-900">
                  {{ formatMinutes(analytics.summary?.avg_queue_time_mins || 0) }}
                </div>
                <p class="text-xs text-white/40 light:text-gray-500 mt-1">{{ $t('agentAnalytics.waitBeforeAssignment') }}</p>
              </div>
            </div>
            <div v-else class="card-depth rounded-xl border border-white/[0.08] bg-white/[0.04] p-6 light:bg-white light:border-gray-200">
              <div class="flex flex-row items-center justify-between space-y-0 pb-2">
                <span class="text-sm font-medium text-white/50 light:text-gray-500">{{ $t('agentAnalytics.messagesSent') }}</span>
                <div class="h-10 w-10 rounded-lg bg-purple-500/20 flex items-center justify-center">
                  <MessageSquare class="h-5 w-5 text-purple-400" />
                </div>
              </div>
              <div class="pt-2">
                <div class="text-3xl font-bold text-white light:text-gray-900">
                  {{ analytics.my_stats?.messages_sent || 0 }}
                </div>
                <p class="text-xs text-white/40 light:text-gray-500 mt-1">{{ $t('agentAnalytics.outgoingMessages') }}</p>
              </div>
            </div>

            <!-- Break Time -->
            <div class="card-depth rounded-xl border border-white/[0.08] bg-white/[0.04] p-6 light:bg-white light:border-gray-200">
              <div class="flex flex-row items-center justify-between space-y-0 pb-2">
                <span class="text-sm font-medium text-white/50 light:text-gray-500">{{ $t('agentAnalytics.breakTime') }}</span>
                <div class="h-10 w-10 rounded-lg bg-amber-500/20 flex items-center justify-center">
                  <Coffee class="h-5 w-5 text-amber-400" />
                </div>
              </div>
              <div class="pt-2">
                <div class="text-3xl font-bold text-white light:text-gray-900">
                  {{ formatMinutes(analytics.my_stats?.total_break_time_mins ?? analytics.summary?.total_break_time_mins ?? 0) }}
                </div>
                <p class="text-xs text-white/40 light:text-gray-500 mt-1">
                  {{ $t('agentAnalytics.breaksTaken', { count: analytics.my_stats?.break_count ?? analytics.summary?.break_count ?? 0 }) }}
                </p>
              </div>
            </div>

            <div
              class="card-depth rounded-xl border border-white/[0.08] bg-white/[0.04] p-6 light:bg-white light:border-gray-200"
            >
              <div class="flex flex-row items-center justify-between space-y-0 pb-2">
                <span class="text-sm font-medium text-white/50 light:text-gray-500">{{ $t('agentAnalytics.averageRating') }}</span>
                <div class="h-10 w-10 rounded-lg bg-yellow-500/20 flex items-center justify-center">
                  <Star class="h-5 w-5 text-yellow-400" />
                </div>
              </div>
              <div class="pt-2">
                <div class="text-3xl font-bold text-white light:text-gray-900">
                  {{ Number(analytics.rating_summary?.average_rating || 0).toFixed(1) }}
                </div>
                <p class="text-xs text-white/40 light:text-gray-500 mt-1">
                  {{ $t('agentAnalytics.totalRatingsLabel', { count: analytics.rating_summary?.total_ratings || 0 }) }}
                </p>
              </div>
            </div>
          </template>
        </div>

        <!-- Charts Row -->
        <div class="grid gap-4 md:grid-cols-2">
          <!-- Trend Chart -->
          <Card>
            <CardHeader>
              <CardTitle>{{ $t('agentAnalytics.transferTrends') }}</CardTitle>
              <CardDescription>{{ $t('agentAnalytics.transfersOverTime') }}</CardDescription>
            </CardHeader>
            <CardContent>
              <div class="h-64">
                <template v-if="isLoading">
                  <Skeleton class="h-full w-full" />
                </template>
                <template v-else-if="trendChartData.labels.length > 0">
                  <Line :data="trendChartData" :options="trendChartOptions" />
                </template>
                <template v-else>
                  <div class="h-full flex items-center justify-center text-muted-foreground">
                    {{ $t('agentAnalytics.noDataAvailable') }}
                  </div>
                </template>
              </div>
            </CardContent>
          </Card>

          <!-- Source Distribution -->
          <Card>
            <CardHeader>
              <CardTitle>{{ $t('agentAnalytics.conversationSources') }}</CardTitle>
              <CardDescription>{{ $t('agentAnalytics.howConversationsInitiated') }}</CardDescription>
            </CardHeader>
            <CardContent>
              <div class="h-64">
                <template v-if="isLoading">
                  <Skeleton class="h-full w-full" />
                </template>
                <template v-else-if="sourceChartData.labels.length > 0">
                  <Doughnut :data="sourceChartData" :options="sourceChartOptions" />
                </template>
                <template v-else>
                  <div class="h-full flex items-center justify-center text-muted-foreground">
                    {{ $t('agentAnalytics.noDataAvailable') }}
                  </div>
                </template>
              </div>
            </CardContent>
          </Card>
        </div>

        <!-- Agent Comparison -->
        <template v-if="canViewOrganizationAnalytics && selectedAgentId === 'all'">
          <Card>
            <CardHeader>
              <CardTitle>{{ $t('agentAnalytics.agentComparison') }}</CardTitle>
              <CardDescription>{{ $t('agentAnalytics.performanceComparison') }}</CardDescription>
            </CardHeader>
            <CardContent>
              <div class="h-64">
                <template v-if="isLoading">
                  <Skeleton class="h-full w-full" />
                </template>
                <template v-else-if="comparisonChartData.labels.length > 0">
                  <Bar :data="comparisonChartData" :options="comparisonChartOptions" />
                </template>
                <template v-else>
                  <div class="h-full flex items-center justify-center text-muted-foreground">
                    {{ $t('agentAnalytics.noAgentsFound') }}
                  </div>
                </template>
              </div>
            </CardContent>
          </Card>
        </template>

        <template v-if="canViewOrganizationAnalytics">
          <Card>
            <CardHeader>
              <CardTitle>{{ $t('agentAnalytics.ratingsTableTitle') }}</CardTitle>
              <CardDescription>{{ $t('agentAnalytics.ratingsTableSubtitle') }}</CardDescription>
            </CardHeader>
            <CardContent>
              <template v-if="isLoading">
                <Skeleton class="h-64 w-full" />
              </template>
              <template v-else-if="analytics?.rating_records?.length">
                <div class="overflow-x-auto rounded-md border border-white/[0.08] light:border-gray-200">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{{ $t('agentAnalytics.agentColumn') }}</TableHead>
                        <TableHead>{{ $t('agentAnalytics.chatIdColumn') }}</TableHead>
                        <TableHead>{{ $t('agentAnalytics.contactColumn') }}</TableHead>
                        <TableHead>{{ $t('agentAnalytics.ratingColumn') }}</TableHead>
                        <TableHead>{{ $t('agentAnalytics.ratedAtColumn') }}</TableHead>
                        <TableHead>{{ $t('agentAnalytics.closingAgentColumn') }}</TableHead>
                        <TableHead>{{ $t('agentAnalytics.ratingMessageColumn') }}</TableHead>
                        <TableHead>{{ $t('agentAnalytics.contextMessagesColumn') }}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      <TableRow v-for="record in analytics.rating_records" :key="record.id">
                        <TableCell>{{ record.agent_name || '-' }}</TableCell>
                        <TableCell class="font-mono text-xs">
                          <button
                            type="button"
                            class="text-primary hover:underline disabled:cursor-not-allowed disabled:no-underline disabled:text-muted-foreground"
                            :disabled="!record.contact_id"
                            @click="openRatingChat(record)"
                          >
                            {{ record.contact_phone || '-' }}
                          </button>
                        </TableCell>
                        <TableCell>{{ record.contact || record.contact_phone }}</TableCell>
                        <TableCell>{{ record.rating }}</TableCell>
                        <TableCell>{{ formatDateTime(record.rated_at) }}</TableCell>
                        <TableCell>{{ record.closing_agent_name || '-' }}</TableCell>
                        <TableCell class="max-w-[220px] truncate" :title="formatRatingMessage(record)">
                          {{ formatRatingMessage(record) }}
                        </TableCell>
                        <TableCell class="max-w-[280px] truncate" :title="formatContextMessages(record.context_messages)">
                          {{ formatContextMessages(record.context_messages) }}
                        </TableCell>
                      </TableRow>
                    </TableBody>
                  </Table>
                </div>
              </template>
              <template v-else>
                <div class="h-40 flex items-center justify-center text-muted-foreground">
                  {{ $t('agentAnalytics.noDataAvailable') }}
                </div>
              </template>
            </CardContent>
          </Card>
        </template>
      </div>
    </ScrollArea>
  </div>
</template>
