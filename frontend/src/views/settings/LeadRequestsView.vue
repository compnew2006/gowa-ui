<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ScrollArea } from '@/components/ui/scroll-area'
import { PageHeader, DataTable, SearchInput, type Column } from '@/components/shared'
import { getErrorMessage } from '@/lib/api-utils'
import { formatDate } from '@/lib/utils'
import { leadRequestsService, type LeadRequest, type LeadRequestStatus } from '@/services/api'
import { toast } from 'vue-sonner'
import { Eye, Inbox, Loader2 } from 'lucide-vue-next'

const { t } = useI18n()

const leadRequests = ref<LeadRequest[]>([])
const isLoading = ref(false)
const isUpdatingStatus = ref(false)
const isDialogOpen = ref(false)
const selectedLead = ref<LeadRequest | null>(null)
const selectedStatus = ref<LeadRequestStatus>('new')

const searchQuery = ref('')
const statusFilter = ref<'all' | LeadRequestStatus>('all')
const currentPage = ref(1)
const totalItems = ref(0)
const pageSize = 20

const sortKey = ref('created_at')
const sortDirection = ref<'asc' | 'desc'>('desc')

const statusOptions: Array<'all' | LeadRequestStatus> = ['all', 'new', 'contacted', 'qualified', 'closed']

const columns = computed<Column<LeadRequest>[]>(() => [
  { key: 'submitter', label: t('leadRequests.columns.submitter'), sortable: true },
  { key: 'company', label: t('leadRequests.columns.company'), sortable: true },
  { key: 'contact', label: t('leadRequests.columns.contact') },
  { key: 'requested_plan', label: t('leadRequests.columns.requestedPlan') },
  { key: 'status', label: t('common.status'), sortable: true },
  { key: 'created', label: t('leadRequests.columns.created'), sortable: true, sortKey: 'created_at' },
  { key: 'actions', label: t('common.actions'), align: 'right' },
])

async function fetchLeadRequests() {
  isLoading.value = true
  try {
    const response = await leadRequestsService.list({
      search: searchQuery.value || undefined,
      status: statusFilter.value === 'all' ? undefined : statusFilter.value,
      page: currentPage.value,
      limit: pageSize
    })
    const data = (response.data as any).data || response.data
    leadRequests.value = data.lead_requests || []
    totalItems.value = data.total ?? leadRequests.value.length
  } catch (error) {
    toast.error(getErrorMessage(error, t('common.failedLoad', { resource: t('resources.leadRequests') })))
  } finally {
    isLoading.value = false
  }
}

const debouncedSearch = useDebounceFn(() => {
  currentPage.value = 1
  fetchLeadRequests()
}, 300)

watch(searchQuery, () => debouncedSearch())
watch(statusFilter, () => {
  currentPage.value = 1
  fetchLeadRequests()
})

function handlePageChange(page: number) {
  currentPage.value = page
  fetchLeadRequests()
}

function openLeadDetails(lead: LeadRequest) {
  selectedLead.value = lead
  selectedStatus.value = lead.status
  isDialogOpen.value = true
}

async function saveStatus() {
  if (!selectedLead.value) return

  isUpdatingStatus.value = true
  try {
    const response = await leadRequestsService.updateStatus(selectedLead.value.id, selectedStatus.value)
    const updatedLead = ((response.data as any).data || response.data) as LeadRequest
    selectedLead.value = updatedLead

    const index = leadRequests.value.findIndex((lead) => lead.id === updatedLead.id)
    if (index !== -1) {
      leadRequests.value[index] = updatedLead
    }

    toast.success(t('common.updatedSuccess', { resource: t('resources.leadRequest') }))
  } catch (error) {
    toast.error(getErrorMessage(error, t('common.failedUpdate', { resource: t('resources.leadRequest') })))
  } finally {
    isUpdatingStatus.value = false
  }
}

function formatDateTime(value: string) {
  return formatDate(value, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function statusLabel(status: LeadRequestStatus | 'all') {
  if (status === 'all') return t('common.all')
  return t(`leadRequests.status.${status}`)
}

function planLabel(plan?: string) {
  if (!plan) return t('leadRequests.plan.unspecified')
  return t(`leadRequests.plan.${plan}`)
}

function statusBadgeClass(status: LeadRequestStatus) {
  switch (status) {
    case 'new':
      return 'border-blue-600 text-blue-600'
    case 'contacted':
      return 'border-amber-600 text-amber-600'
    case 'qualified':
      return 'border-emerald-600 text-emerald-600'
    case 'closed':
      return 'border-zinc-500 text-zinc-500'
    default:
      return ''
  }
}

onMounted(() => fetchLeadRequests())
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader
      :title="$t('leadRequests.title')"
      :subtitle="$t('leadRequests.subtitle')"
      :icon="Inbox"
      icon-gradient="bg-gradient-to-br from-cyan-500 to-blue-600 shadow-cyan-500/20"
    />

    <ScrollArea class="flex-1">
      <div class="p-6">
        <div class="max-w-6xl mx-auto">
          <Card>
            <CardHeader>
              <div class="flex items-center justify-between flex-wrap gap-4">
                <div>
                  <CardTitle>{{ $t('leadRequests.inboxTitle') }}</CardTitle>
                  <CardDescription>{{ $t('leadRequests.inboxDescription') }}</CardDescription>
                </div>
                <div class="flex items-center gap-3 flex-wrap">
                  <SearchInput
                    v-model="searchQuery"
                    :placeholder="$t('leadRequests.searchPlaceholder')"
                    class="w-64"
                  />
                  <Select v-model="statusFilter">
                    <SelectTrigger class="w-[180px]">
                      <SelectValue :placeholder="$t('leadRequests.filterStatus')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="status in statusOptions" :key="status" :value="status">
                        {{ statusLabel(status) }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <DataTable
                :items="leadRequests"
                :columns="columns"
                :is-loading="isLoading"
                :empty-icon="Inbox"
                :empty-title="searchQuery || statusFilter !== 'all' ? $t('leadRequests.noMatching') : $t('leadRequests.emptyTitle')"
                :empty-description="searchQuery || statusFilter !== 'all' ? $t('leadRequests.noMatchingDescription') : $t('leadRequests.emptyDescription')"
                v-model:sort-key="sortKey"
                v-model:sort-direction="sortDirection"
                server-pagination
                :current-page="currentPage"
                :total-items="totalItems"
                :page-size="pageSize"
                item-name="lead requests"
                @page-change="handlePageChange"
              >
                <template #cell-submitter="{ item: lead }">
                  <div>
                    <div class="font-medium">{{ lead.full_name }}</div>
                    <div class="text-xs text-muted-foreground">{{ lead.country || $t('leadRequests.noCountry') }}</div>
                  </div>
                </template>

                <template #cell-company="{ item: lead }">
                  <span class="font-medium">{{ lead.company_name }}</span>
                </template>

                <template #cell-contact="{ item: lead }">
                  <div class="space-y-1">
                    <div class="text-sm">{{ lead.work_email }}</div>
                    <div class="text-xs text-muted-foreground">{{ lead.phone_whatsapp }}</div>
                  </div>
                </template>

                <template #cell-requested_plan="{ item: lead }">
                  <Badge variant="outline" class="text-xs">
                    {{ planLabel(lead.requested_plan) }}
                  </Badge>
                </template>

                <template #cell-status="{ item: lead }">
                  <Badge variant="outline" :class="statusBadgeClass(lead.status)">
                    {{ statusLabel(lead.status) }}
                  </Badge>
                </template>

                <template #cell-created="{ item: lead }">
                  <span class="text-muted-foreground">{{ formatDateTime(lead.created_at) }}</span>
                </template>

                <template #cell-actions="{ item: lead }">
                  <Button variant="ghost" size="icon" @click="openLeadDetails(lead)">
                    <Eye class="h-4 w-4" />
                  </Button>
                </template>
              </DataTable>
            </CardContent>
          </Card>
        </div>
      </div>
    </ScrollArea>

    <Dialog v-model:open="isDialogOpen">
      <DialogContent class="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{{ selectedLead?.full_name || $t('leadRequests.detailsTitle') }}</DialogTitle>
          <DialogDescription>
            {{ selectedLead?.company_name || $t('leadRequests.detailsDescription') }}
          </DialogDescription>
        </DialogHeader>

        <div v-if="selectedLead" class="space-y-6 py-2">
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-1">
              <Label>{{ $t('leadRequests.fields.email') }}</Label>
              <div class="text-sm text-muted-foreground">{{ selectedLead.work_email }}</div>
            </div>
            <div class="space-y-1">
              <Label>{{ $t('leadRequests.fields.phoneWhatsApp') }}</Label>
              <div class="text-sm text-muted-foreground">{{ selectedLead.phone_whatsapp }}</div>
            </div>
            <div class="space-y-1">
              <Label>{{ $t('leadRequests.fields.country') }}</Label>
              <div class="text-sm text-muted-foreground">{{ selectedLead.country || $t('leadRequests.noCountry') }}</div>
            </div>
            <div class="space-y-1">
              <Label>{{ $t('leadRequests.fields.requestedPlan') }}</Label>
              <div class="text-sm text-muted-foreground">{{ planLabel(selectedLead.requested_plan) }}</div>
            </div>
            <div class="space-y-1">
              <Label>{{ $t('leadRequests.fields.source') }}</Label>
              <div class="text-sm text-muted-foreground">{{ selectedLead.source_route }}</div>
            </div>
            <div class="space-y-1">
              <Label>{{ $t('leadRequests.fields.submittedAt') }}</Label>
              <div class="text-sm text-muted-foreground">{{ formatDateTime(selectedLead.created_at) }}</div>
            </div>
          </div>

          <div class="space-y-2">
            <Label>{{ $t('leadRequests.fields.message') }}</Label>
            <div class="rounded-lg border bg-muted/30 px-4 py-3 text-sm text-muted-foreground whitespace-pre-wrap min-h-[120px]">
              {{ selectedLead.message || $t('leadRequests.noMessage') }}
            </div>
          </div>

          <div class="space-y-2">
            <Label for="lead-status">{{ $t('leadRequests.fields.status') }}</Label>
            <Select v-model="selectedStatus">
              <SelectTrigger id="lead-status">
                <SelectValue :placeholder="$t('leadRequests.filterStatus')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="status in statusOptions.filter((value) => value !== 'all')" :key="status" :value="status">
                  {{ statusLabel(status) }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="isDialogOpen = false">
            {{ $t('common.close') }}
          </Button>
          <Button @click="saveStatus" :disabled="isUpdatingStatus || !selectedLead || selectedLead.status === selectedStatus">
            <Loader2 v-if="isUpdatingStatus" class="h-4 w-4 mr-2 animate-spin" />
            {{ $t('leadRequests.updateStatus') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
