<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { agentAnalyticsService } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useUsersStore } from '@/stores/users'
import { PageHeader, ErrorState, DateRangePicker } from '@/components/shared'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList
} from '@/components/ui/command'
import {
  MessageSquare,
  BarChart3,
  ChevronsUpDown,
  Check,
  Coffee,
  Star
} from 'lucide-vue-next'
// Centralized Chart.js setup (registered once)
import { Line, Bar } from '@/lib/charts'
import { useDateRange } from '@/composables/useDateRange'

interface AgentAnalyticsSummary {
  total_break_time_mins: number
  break_count: number
  avg_rating: number
  ratings_count: number
}

interface AgentPerformanceStats {
  agent_id: string
  agent_name: string
  messages_sent: number
  total_break_time_mins: number
  break_count: number
  is_available: boolean
  current_break_start?: string
  avg_rating: number
  ratings_count: number
}

interface TrendPoint {
  date: string
  ratings_count: number
  avg_rating: number
}

interface AgentAnalyticsResponse {
  summary: AgentAnalyticsSummary
  agent_stats?: AgentPerformanceStats[]
  trend_data: TrendPoint[]
  my_stats?: AgentPerformanceStats
}

const { t } = useI18n()
const authStore = useAuthStore()
const usersStore = useUsersStore()
// Permission-driven (analytics.agents:read), not role-name based — custom
// roles with the right permission should get the same view.
const isAdminOrManager = computed(() => authStore.hasPermission('analytics.agents', 'read'))

const analytics = ref<AgentAnalyticsResponse | null>(null)
const isLoading = ref(true)
const error = ref<string | null>(null)

// Agent filter for admins/managers
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

// Time range filter
const {
  selectedRange,
  customDateRange,
  isDatePickerOpen,
  dateRange,
  formatDateRangeDisplay,
  applyCustomRange: applyCustomRangeBase,
} = useDateRange({ storageKey: 'agent_analytics' })

const formatMinutes = (mins: number): string => {
  if (!mins || mins === 0) return '0m'
  if (mins < 60) return `${Math.round(mins)}m`
  const hours = Math.floor(mins / 60)
  const remainingMins = Math.round(mins % 60)
  return remainingMins > 0 ? `${hours}h ${remainingMins}m` : `${hours}h`
}

const fetchAgents = async () => {
  if (!isAdminOrManager.value) return
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
  error.value = null
  try {
    const { from, to } = dateRange.value
    const params: { from: string; to: string; agent_id?: string } = { from, to }
    if (isAdminOrManager.value && selectedAgentId.value !== 'all') {
      params.agent_id = selectedAgentId.value
    }
    const response = await agentAnalyticsService.getSummary(params)
    const data = response.data.data || response.data
    analytics.value = data
  } catch (err) {
    console.error('Failed to load agent analytics:', err)
    error.value = t('agentAnalytics.errorLoadingAnalytics')
    analytics.value = null
  } finally {
    isLoading.value = false
  }
}

const applyCustomRange = () => {
  applyCustomRangeBase()
  fetchAnalytics()
}

watch(selectedRange, (newValue) => {
  if (newValue !== 'custom') {
    fetchAnalytics()
  }
})

watch(selectedAgentId, () => {
  fetchAnalytics()
})

onMounted(() => {
  fetchAgents()
  fetchAnalytics()
})

// Messages sent: sum across agents when viewing all, else the selected agent's.
const messagesSent = computed(() => {
  if (selectedAgentId.value === 'all') {
    return (analytics.value?.agent_stats ?? []).reduce((sum, a) => sum + (a.messages_sent || 0), 0)
  }
  return analytics.value?.my_stats?.messages_sent ?? 0
})

// CSAT values for the stat card: org-wide when viewing all agents,
// per-agent otherwise (my_stats carries the filtered agent's numbers).
const csatAvgRating = computed(() => {
  if (selectedAgentId.value === 'all') return analytics.value?.summary?.avg_rating ?? 0
  return analytics.value?.my_stats?.avg_rating ?? 0
})
const csatRatingsCount = computed(() => {
  if (selectedAgentId.value === 'all') return analytics.value?.summary?.ratings_count ?? 0
  return analytics.value?.my_stats?.ratings_count ?? 0
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
        label: t('agentAnalytics.ratingsReceivedLabel'),
        data: analytics.value.trend_data.map(d => d.ratings_count),
        borderColor: 'rgb(234, 179, 8)',
        backgroundColor: 'rgba(234, 179, 8, 0.1)',
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
        label: t('agentAnalytics.messagesSent'),
        data: analytics.value.agent_stats.map(a => a.messages_sent),
        backgroundColor: 'rgba(16, 185, 129, 0.8)'
      },
      {
        label: t('agentAnalytics.customerRating'),
        data: analytics.value.agent_stats.map(a => a.ratings_count),
        backgroundColor: 'rgba(234, 179, 8, 0.8)'
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
</script>

<template>
  <div class="flex flex-col h-full">
    <PageHeader
      :title="$t('agentAnalytics.title')"
      :description="isAdminOrManager ? $t('agentAnalytics.subtitle') : $t('agentAnalytics.myMetrics')"
      :icon="BarChart3"
      icon-gradient="bg-gradient-to-br from-blue-500 to-indigo-600 shadow-blue-500/20"
    >
      <template #actions>
        <!-- Agent Filter (Admin/Manager only) -->
        <div v-if="isAdminOrManager" class="flex items-center gap-2 mr-4">
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

        <!-- Time Range Filter -->
        <div class="flex items-center gap-2">
          <DateRangePicker
            v-model:selected-range="selectedRange"
            v-model:custom-date-range="customDateRange"
            v-model:is-date-picker-open="isDatePickerOpen"
            :format-date-range-display="formatDateRangeDisplay"
            @apply-custom="applyCustomRange"
          />
        </div>
      </template>
    </PageHeader>

    <!-- Content -->
    <ScrollArea class="flex-1">
      <div class="p-6 space-y-6">
        <!-- Error State -->
        <ErrorState
          v-if="error && !isLoading"
          :title="$t('common.loadErrorTitle')"
          :description="error"
          :retry-label="$t('common.retry')"
          @retry="fetchAnalytics"
        />

        <!-- Stats Cards -->
        <div v-if="!error" class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <template v-if="isLoading">
            <div v-for="i in 3" :key="i" class="rounded-xl border border-white/[0.08] bg-white/[0.02] p-6 light:bg-white light:border-gray-200">
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
            <!-- Messages Sent -->
            <div class="card-depth rounded-xl border border-white/[0.08] bg-white/[0.04] p-6 light:bg-white light:border-gray-200">
              <div class="flex flex-row items-center justify-between space-y-0 pb-2">
                <span class="text-sm font-medium text-white/50 light:text-gray-500">{{ $t('agentAnalytics.messagesSent') }}</span>
                <div class="h-10 w-10 rounded-lg bg-purple-500/20 flex items-center justify-center">
                  <MessageSquare class="h-5 w-5 text-purple-400" />
                </div>
              </div>
              <div class="pt-2">
                <div class="text-3xl font-bold text-white light:text-gray-900">
                  {{ messagesSent }}
                </div>
                <p class="text-xs text-white/40 light:text-gray-500 mt-1">{{ $t('agentAnalytics.outgoingMessages') }}</p>
              </div>
            </div>

            <!-- Customer Rating (CSAT) -->
            <div class="card-depth rounded-xl border border-white/[0.08] bg-white/[0.04] p-6 light:bg-white light:border-gray-200">
              <div class="flex flex-row items-center justify-between space-y-0 pb-2">
                <span class="text-sm font-medium text-white/50 light:text-gray-500">{{ $t('agentAnalytics.customerRating') }}</span>
                <div class="h-10 w-10 rounded-lg bg-yellow-500/20 flex items-center justify-center">
                  <Star class="h-5 w-5 text-yellow-400" />
                </div>
              </div>
              <div class="pt-2">
                <div class="text-3xl font-bold text-white light:text-gray-900">
                  {{ csatRatingsCount > 0 ? `${csatAvgRating.toFixed(1)} / 5` : '—' }}
                </div>
                <p class="text-xs text-white/40 light:text-gray-500 mt-1">
                  {{ csatRatingsCount > 0 ? $t('agentAnalytics.ratingsReceived', { count: csatRatingsCount }) : $t('agentAnalytics.noRatingsYet') }}
                </p>
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
          </template>
        </div>

        <!-- Charts Row -->
        <div v-if="!error" class="grid gap-4">
          <!-- Ratings Trend Chart -->
          <Card>
            <CardHeader>
              <CardTitle>{{ $t('agentAnalytics.ratingTrends') }}</CardTitle>
              <CardDescription>{{ $t('agentAnalytics.ratingsOverTime') }}</CardDescription>
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
        </div>

        <!-- Agent Comparison (Admin/Manager only, when viewing all agents) -->
        <template v-if="!error && isAdminOrManager && selectedAgentId === 'all'">
          <!-- Per-agent performance table with CSAT rating -->
          <Card>
            <CardHeader>
              <CardTitle>{{ $t('agentAnalytics.agentPerformance') }}</CardTitle>
              <CardDescription>{{ $t('agentAnalytics.agentPerformanceDesc') }}</CardDescription>
            </CardHeader>
            <CardContent>
              <template v-if="isLoading">
                <Skeleton class="h-40 w-full" />
              </template>
              <template v-else-if="analytics?.agent_stats?.length">
                <table class="w-full text-sm">
                  <thead>
                    <tr class="border-b border-white/[0.08] light:border-gray-200 text-left rtl:text-right text-white/50 light:text-gray-500">
                      <th class="py-2 pr-4 font-medium">{{ $t('agentAnalytics.agent') }}</th>
                      <th class="py-2 pr-4 font-medium">{{ $t('agentAnalytics.customerRating') }}</th>
                      <th class="py-2 pr-4 font-medium">{{ $t('agentAnalytics.messagesSent') }}</th>
                      <th class="py-2 font-medium">{{ $t('agentAnalytics.breakTime') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="agent in analytics.agent_stats"
                      :key="agent.agent_id"
                      class="border-b border-white/[0.04] light:border-gray-100 text-white light:text-gray-900"
                    >
                      <td class="py-2.5 pr-4 font-medium">{{ agent.agent_name || $t('agentAnalytics.selectAgent') }}</td>
                      <td class="py-2.5 pr-4">
                        <span v-if="agent.ratings_count > 0" class="inline-flex items-center gap-1">
                          <Star class="h-4 w-4 text-yellow-400 fill-yellow-400" />
                          {{ agent.avg_rating.toFixed(1) }}
                          <span class="text-white/40 light:text-gray-500">({{ agent.ratings_count }})</span>
                        </span>
                        <span v-else class="text-white/40 light:text-gray-500">{{ $t('agentAnalytics.noRatingsYet') }}</span>
                      </td>
                      <td class="py-2.5 pr-4">{{ agent.messages_sent }}</td>
                      <td class="py-2.5">{{ formatMinutes(agent.total_break_time_mins) }}</td>
                    </tr>
                  </tbody>
                </table>
              </template>
              <template v-else>
                <div class="py-8 text-center text-muted-foreground">
                  {{ $t('agentAnalytics.noAgentsFound') }}
                </div>
              </template>
            </CardContent>
          </Card>

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
      </div>
    </ScrollArea>
  </div>
</template>
