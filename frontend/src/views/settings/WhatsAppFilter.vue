<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Progress } from "@/components/ui/progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  PageHeader,
  SearchInput,
  DataTable,
  DeleteConfirmDialog,
  SelectableDataTable,
  type Column,
} from "@/components/shared";
import { useSelectableTable } from "@/composables/useSelectableTable";
import { toast } from "vue-sonner";
import {
  Trash2,
  Download,
  CheckCircle2,
  FileSpreadsheet,
  AlertCircle,
  UserPlus,
  Loader2,
  Calendar,
  Layers,
  Sparkles,
} from "lucide-vue-next";
import { useConfigStore } from "@/stores/config";
import { useWhatsAppFilter } from "@/composables/useWhatsAppFilter";
import {
  instancesService,
  accountsService,
  contactsService,
  whatsappFilterService,
  unwrapWhatsAppFilterResultsPage,
  type WhatsAppFilterBatch,
  type WhatsAppFilterResult,
} from "@/services/api";
import { formatDate } from "@/lib/utils";

const { t } = useI18n();
const configStore = useConfigStore();
const filterStore = useWhatsAppFilter();

// Connection states
const connections = ref<Array<{ id: string; name: string }>>([]);
const isLoadingConnections = ref(false);

// Form state
const selectedConnectionId = ref("");
const pastedNumbers = ref("");
const csvFile = ref<File | null>(null);
const fileInput = ref<HTMLInputElement | null>(null);
const activeTab = ref<"paste" | "upload">("paste");

// List campaigns pagination
const campaignsPage = ref(1);
const campaignsLimit = 10;

// Results selection table controller
const {
  items: activeBatchResults,
  totalItems: totalResults,
  isLoading: isLoadingResults,
  currentPage: resultsPage,
  pageSize: resultsLimit,
  searchQuery: resultsSearchQuery,
  filterStatus: resultsFilterStatus,
  selectedIds,
  selectedRecords,
  isAllPageSelected,
  isAllMatchingSelected,
  selectedCount,
  toggleRow,
  togglePageSelection,
  selectAllMatching,
  clearSelection,
  loadData: fetchCampaignResults,
  resetTable: resetResultsTable
} = useSelectableTable<WhatsAppFilterResult>(
  async (params) => {
    if (!viewBatchId.value) return { data: [], total: 0 }
    const response = await whatsappFilterService.listResults(viewBatchId.value, {
      page: params.page,
      limit: params.limit,
      status: params.status as any || 'all',
      q: params.q
    })
    return unwrapPaginatedResults(response)
  },
  {
    initialPageSize: 25,
    rowKey: 'id'
  }
)

const resultsTotalPages = computed(() => {
  return Math.ceil(totalResults.value / resultsLimit.value) || 1
})

function unwrapPaginatedResults(response: any): { data: WhatsAppFilterResult[]; total: number } {
  const payload = unwrapWhatsAppFilterResultsPage(response?.data);
  return {
    data: Array.isArray(payload.data) ? payload.data : [],
    total: Number(payload.total ?? 0),
  };
}

// Dialog and details state
const viewBatchId = ref<string | null>(null);
const deleteDialogOpen = ref(false);
const batchToDelete = ref<WhatsAppFilterBatch | null>(null);
const isImportingContacts = ref(false);

// Columns
const campaignColumns = computed<Column<WhatsAppFilterBatch>[]>(() => [
  { key: "created_at", label: t("whatsappFilter.date") },
  { key: "whatsapp_account", label: t("whatsappFilter.verifiedBy") },
  { key: "total_numbers", label: t("whatsappFilter.total") },
  { key: "valid_numbers", label: t("whatsappFilter.valid") },
  { key: "invalid_numbers", label: t("whatsappFilter.invalid") },
  { key: "status", label: t("whatsappFilter.status") },
  { key: "actions", label: t("whatsappFilter.actions"), align: "right" },
]);

const resultColumns = computed<Column<WhatsAppFilterResult>[]>(() => [
  { key: "phone_number", label: t("common.phone") },
  { key: "contact_name", label: t("common.name") },
  { key: "is_valid", label: t("whatsappFilter.status") },
  { key: "error_message", label: t("common.description") },
]);

// Polling interval reference
let pollingInterval: any = null;

// Helpers
const isWhatsmeow = computed(() => configStore.isWhatsmeow);

// Load available senders
async function loadConnections() {
  isLoadingConnections.value = true;
  connections.value = [];
  try {
    if (isWhatsmeow.value) {
      const response = await instancesService.list();
      const data = (response.data as any).data ?? response.data;
      const list = Array.isArray(data) ? data : data.instances ?? [];
      connections.value = list
        .filter((inst: any) => inst.status === "connected")
        .map((inst: any) => ({
          id: inst.id,
          name: `${inst.name} (${inst.phone_number || "whatsmeow"})`,
        }));
    } else {
      const [accRes, instRes] = await Promise.allSettled([
        accountsService.list(),
        instancesService.list(),
      ]);
      const accounts = accRes.status === "fulfilled"
        ? (accRes.value.data as any).data?.accounts ?? []
        : [];
      const data = instRes.status === "fulfilled"
        ? (instRes.value.data as any).data ?? instRes.value.data
        : null;
      const instances = Array.isArray(data) ? data : data?.instances ?? [];
      const mapped = [
        ...accounts.map((acc: any) => ({ id: acc.id, name: acc.name })),
        ...instances
          .filter((inst: any) => inst.status === "connected")
          .map((inst: any) => ({
            id: inst.id,
            name: `${inst.name} (${inst.phone_number || "whatsmeow"})`,
          })),
      ];
      connections.value = mapped;
    }
  } catch (err) {
    console.error("Failed to load connections:", err);
    toast.error(t("whatsappFilter.noConnectedWhatsApp"));
  } finally {
    isLoadingConnections.value = false;
  }
}

// Drag & Drop
const isDragActive = ref(false);
function onDragOver(e: DragEvent) {
  e.preventDefault();
  isDragActive.value = true;
}
function onDragLeave() {
  isDragActive.value = false;
}
function onDrop(e: DragEvent) {
  e.preventDefault();
  isDragActive.value = false;
  const files = e.dataTransfer?.files;
  if (files && files.length > 0) {
    const file = files[0];
    if (file.name.endsWith(".csv")) {
      csvFile.value = file;
      toast.success(`${file.name} selected successfully.`);
    } else {
      toast.error("Only CSV files are supported.");
    }
  }
}
function onFileSelect(e: Event) {
  const target = e.target as HTMLInputElement;
  const files = target.files;
  if (files && files.length > 0) {
    csvFile.value = files[0];
  }
}

function triggerFileSelect() {
  fileInput.value?.click();
}

// Form Submission
async function startVerification() {
  if (!selectedConnectionId.value) {
    toast.error(t("whatsappFilter.connectionRequired"));
    return;
  }

  try {
    let batch: WhatsAppFilterBatch;
    if (activeTab.value === "paste") {
      const phones = pastedNumbers.value
        .split("\n")
        .map((num) => num.trim())
        .filter((num) => num.length > 0);

      if (phones.length === 0) {
        toast.error(t("whatsappFilter.errorEmptyInput"));
        return;
      }
      batch = await filterStore.createCampaignJSON(selectedConnectionId.value, phones);
    } else {
      if (!csvFile.value) {
        toast.error(t("whatsappFilter.errorEmptyInput"));
        return;
      }
      batch = await filterStore.createCampaignCSV(selectedConnectionId.value, csvFile.value);
    }

    toast.success(t("common.createdSuccess", { resource: t("whatsappFilter.batchName") }));
    
    // Clear inputs
    pastedNumbers.value = "";
    csvFile.value = null;

    // View the batch results page and start polling
    viewBatch(batch.id);
  } catch (err: any) {
    toast.error(err.response?.data?.message || "Failed to initiate number verification.");
  }
}

// View campaign details & results
async function viewBatch(id: string) {
  // Fetch details BEFORE showing the panel so activeBatch is always populated
  // when the v-if renders the detail cards.
  await filterStore.fetchBatchDetails(id);
  viewBatchId.value = id;
  resetResultsTable();
  void fetchCampaignResults();
  startPolling(id);
}

// Polling manager
function startPolling(id: string) {
  stopPolling();
  pollingInterval = setInterval(async () => {
    const details = await filterStore.fetchBatchDetails(id);
    if (!details) {
      stopPolling();
      return;
    }
    // Refresh results list concurrently to show live rows populated
    void fetchCampaignResults();
    
    if (details.status === "completed" || details.status === "failed") {
      stopPolling();
      // Reload batches list
      void filterStore.fetchBatches(campaignsPage.value, campaignsLimit);
    }
  }, 2500);
}

function stopPolling() {
  if (pollingInterval) {
    clearInterval(pollingInterval);
    pollingInterval = null;
  }
}

// Actions
async function exportResults() {
  if (!viewBatchId.value || selectedCount.value === 0) return;
  try {
    if (isAllMatchingSelected.value) {
      await filterStore.downloadResults(
        viewBatchId.value,
        resultsFilterStatus.value as "all" | "valid" | "invalid",
        resultsSearchQuery.value
      );
    } else {
      let csvContent = 'Phone Number,Contact Name,Registered on WhatsApp,Checked At,Error Message\n';
      selectedRecords.value.forEach((row) => {
        const isValidStr = row.is_valid ? 'true' : 'false';
        const checkedStr = row.checked_at ? new Date(row.checked_at).toISOString() : '';
        const errorMsg = row.error_message || '';
        csvContent += `"${row.phone_number}","${row.contact_name || ''}","${isValidStr}","${checkedStr}","${errorMsg}"\n`;
      });
      const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.setAttribute('href', url);
      link.setAttribute('download', `whatsapp_filter_selected_${(viewBatchId.value ?? '').slice(0, 8)}.csv`);
      link.style.visibility = 'hidden';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    }
    toast.success("CSV exported successfully.");
  } catch (err) {
    toast.error("Failed to export results CSV.");
  }
}

async function importToContacts() {
  if (!viewBatchId.value || selectedCount.value === 0) return;

  isImportingContacts.value = true;
  let importedCount = 0;
  let failedCount = 0;

  try {
    let listToImport: WhatsAppFilterResult[] = [];
    if (isAllMatchingSelected.value) {
      const res = await whatsappFilterService.listResults(viewBatchId.value, {
        page: 1,
        limit: 10000,
        status: 'valid' as const
      });
      listToImport = unwrapPaginatedResults(res).data;
    } else {
      listToImport = selectedRecords.value.filter((r) => r.is_valid);
    }

    if (listToImport.length === 0) {
      toast.error("No valid/registered phone numbers selected to import.");
      isImportingContacts.value = false;
      return;
    }

    toast.info(`Importing ${listToImport.length} valid numbers into contacts...`);

    for (const res of listToImport) {
      try {
        await contactsService.create({
          phone_number: res.phone_number,
          profile_name: res.contact_name || `WA Verified ${res.phone_number}`,
          whatsapp_account: filterStore.activeBatch.value?.whatsapp_account || "",
          instance_id: filterStore.activeBatch.value?.instance_id || undefined,
          start_chat: false,
          tags: ["whatsapp_verified"],
          metadata: {
            verified_batch_id: viewBatchId.value,
            verified_at: res.checked_at,
          },
        });
        importedCount++;
      } catch (err: any) {
        if (err.response?.status === 409) {
          importedCount++;
        } else {
          failedCount++;
        }
      }
    }

    toast.success(t("whatsappFilter.importSuccess", { count: importedCount }));
    if (failedCount > 0) {
      toast.warning(`Note: ${failedCount} contacts failed to import.`);
    }
    clearSelection();
  } catch (err) {
    toast.error(t("whatsappFilter.importFailed"));
  } finally {
    isImportingContacts.value = false;
  }
}

function openDeleteDialog(batch: WhatsAppFilterBatch) {
  batchToDelete.value = batch;
  deleteDialogOpen.value = true;
}

async function confirmDelete() {
  if (!batchToDelete.value) return;
  try {
    await filterStore.deleteCampaign(batchToDelete.value.id);
    toast.success(t("whatsappFilter.deleteSuccess"));
    deleteDialogOpen.value = false;
    batchToDelete.value = null;
    void filterStore.fetchBatches(campaignsPage.value, campaignsLimit);
  } catch (err: any) {
    const serverMsg = err.response?.data?.message;
    toast.error(serverMsg || t("whatsappFilter.deleteFailed"));
  }
}

// Progress calculation
const batchProgress = computed(() => {
  if (!filterStore.activeBatch.value) return 0;
  const batch = filterStore.activeBatch.value;
  if (batch.total_numbers === 0) return 0;
  const processed = batch.valid_numbers + batch.invalid_numbers;
  return Math.round((processed / batch.total_numbers) * 100);
});

// Watch page settings for campaign batches
watch(campaignsPage, () => {
  void filterStore.fetchBatches(campaignsPage.value, campaignsLimit);
});

onMounted(async () => {
  await configStore.fetchConfig();
  await Promise.all([
    loadConnections(),
    filterStore.fetchBatches(campaignsPage.value, campaignsLimit),
  ]);
});

onUnmounted(() => {
  stopPolling();
});
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="$t('whatsappFilter.title')"
      :subtitle="$t('whatsappFilter.subtitle')"
      :icon="Layers"
      icon-gradient="bg-gradient-to-br from-emerald-500 to-teal-600 shadow-emerald-500/20"
      :back-link="viewBatchId ? undefined : '/settings'"
      @back="viewBatchId = null; stopPolling(); filterStore.fetchBatches(campaignsPage, campaignsLimit)"
    >
      <template #actions v-if="viewBatchId">
        <Button variant="outline" size="sm" @click="viewBatchId = null; stopPolling(); filterStore.fetchBatches(campaignsPage, campaignsLimit)">
          Back to Campaigns
        </Button>
      </template>
    </PageHeader>

    <ScrollArea class="flex-1">
      <div class="p-6 max-w-7xl mx-auto space-y-6">
        <!-- 1. CAMPAIGN BATCH RESULTS PANEL (When viewing details) -->
        <div v-if="viewBatchId && filterStore.activeBatch.value" class="space-y-6">
          <div class="grid gap-6 md:grid-cols-3">
            <!-- Summary Stats Card -->
            <Card class="md:col-span-2 relative overflow-hidden bg-card border shadow-sm">
              <div class="absolute top-0 right-0 p-4 opacity-10 pointer-events-none">
                <Sparkles class="h-24 w-24 text-emerald-500" />
              </div>
              <CardHeader>
                <div class="flex items-center justify-between">
                  <div>
                    <CardTitle class="text-xl flex items-center gap-2">
                      {{ $t('whatsappFilter.resultsTitle', { id: (filterStore.activeBatch.value.id ?? '').slice(0, 8) }) }}
                    </CardTitle>
                    <CardDescription class="text-sm mt-1">
                      Account: <span class="font-medium text-foreground">{{ filterStore.activeBatch.value.whatsapp_account }}</span> |
                      Date: <span class="font-medium text-foreground">{{ formatDate(filterStore.activeBatch.value.created_at) }}</span>
                    </CardDescription>
                  </div>
                  <span
                    :class="[
                      'px-3 py-1 rounded-full text-xs font-semibold border flex items-center gap-1.5 shadow-sm',
                      filterStore.activeBatch.value.status === 'completed' && 'bg-emerald-500/10 border-emerald-500/30 text-emerald-600 dark:text-emerald-400',
                      filterStore.activeBatch.value.status === 'failed' && 'bg-rose-500/10 border-rose-500/30 text-rose-600 dark:text-rose-400',
                      (filterStore.activeBatch.value.status === 'processing' || filterStore.activeBatch.value.status === 'pending') && 'bg-blue-500/10 border-blue-500/30 text-blue-600 dark:text-blue-400 animate-pulse'
                    ]"
                  >
                    <span v-if="filterStore.activeBatch.value.status === 'processing'" class="relative flex h-2 w-2">
                      <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span>
                      <span class="relative inline-flex rounded-full h-2 w-2 bg-blue-500"></span>
                    </span>
                    {{ $t(`whatsappFilter.status${(filterStore.activeBatch.value.status || 'Pending').charAt(0).toUpperCase() + (filterStore.activeBatch.value.status || 'pending').slice(1)}`, { percent: batchProgress }) }}
                  </span>
                </div>
              </CardHeader>
              <CardContent class="space-y-6">
                <!-- Polling / Running progress bar -->
                <div v-if="filterStore.activeBatch.value.status === 'pending' || filterStore.activeBatch.value.status === 'processing'" class="space-y-2">
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-muted-foreground">Checking numbers...</span>
                    <span class="font-medium text-emerald-500 animate-pulse">{{ batchProgress }}%</span>
                  </div>
                  <Progress :model-value="batchProgress" class="h-2.5 bg-secondary overflow-hidden rounded-full transition-all duration-500" />
                </div>

                <div v-else-if="filterStore.activeBatch.value.status === 'failed' && filterStore.activeBatch.value.error_message" class="p-3 bg-rose-500/15 border border-rose-500/30 rounded-lg text-sm text-rose-600 dark:text-rose-400 flex gap-2 items-start">
                  <AlertCircle class="h-5 w-5 shrink-0" />
                  <div>
                    <h4 class="font-semibold">Verification Campaign Stopped</h4>
                    <p class="mt-0.5">{{ filterStore.activeBatch.value.error_message }}</p>
                  </div>
                </div>

                <!-- Stats breakdown widgets -->
                <div class="grid grid-cols-3 gap-4">
                  <div class="p-4 rounded-xl border bg-slate-500/5 hover:bg-slate-500/10 transition-colors flex flex-col items-center justify-center">
                    <span class="text-xs text-muted-foreground font-medium uppercase tracking-wider">Total</span>
                    <span class="text-2xl font-bold mt-1 text-slate-800 dark:text-slate-100">{{ filterStore.activeBatch.value.total_numbers }}</span>
                  </div>
                  <div class="p-4 rounded-xl border bg-emerald-500/5 border-emerald-500/10 hover:bg-emerald-500/10 transition-colors flex flex-col items-center justify-center">
                    <span class="text-xs text-muted-foreground font-medium uppercase tracking-wider">Registered</span>
                    <span class="text-2xl font-bold mt-1 text-emerald-600 dark:text-emerald-400">{{ filterStore.activeBatch.value.valid_numbers }}</span>
                  </div>
                  <div class="p-4 rounded-xl border bg-rose-500/5 border-rose-500/10 hover:bg-rose-500/10 transition-colors flex flex-col items-center justify-center">
                    <span class="text-xs text-muted-foreground font-medium uppercase tracking-wider">Not Registered</span>
                    <span class="text-2xl font-bold mt-1 text-rose-600 dark:text-rose-400">{{ filterStore.activeBatch.value.invalid_numbers }}</span>
                  </div>
                </div>
              </CardContent>
            </Card>

            <!-- Actions Card -->
            <Card class="flex flex-col justify-between border shadow-sm">
              <CardHeader>
                <CardTitle class="text-lg">Campaign Operations</CardTitle>
                <CardDescription>Export results or save valid contacts to your registry.</CardDescription>
              </CardHeader>
              <CardContent class="space-y-3 flex-1 flex flex-col justify-end">
                <Button variant="outline" class="w-full justify-start gap-2 shadow-sm" @click="exportResults" :disabled="selectedCount === 0">
                  <Download class="h-4 w-4" />
                  Export Selected ({{ selectedCount }})
                </Button>
                <Button variant="default" class="w-full justify-start gap-2 shadow-sm bg-gradient-to-br from-emerald-500 to-emerald-600 hover:from-emerald-600 hover:to-emerald-700" @click="importToContacts" :disabled="isImportingContacts || selectedCount === 0">
                  <Loader2 v-if="isImportingContacts" class="h-4 w-4 animate-spin" />
                  <UserPlus v-else class="h-4 w-4" />
                  Import Selected ({{ selectedCount }}) to Contacts
                </Button>
              </CardContent>
            </Card>
          </div>

          <!-- Paginated results table card -->
          <Card class="border shadow-sm">
            <CardHeader class="pb-3 border-b">
              <div class="flex items-center justify-between flex-wrap gap-4">
                <div class="flex items-center gap-3">
                  <span class="text-lg font-bold">Verification Log</span>
                  <!-- Filters tab -->
                  <div class="flex rounded-lg bg-secondary p-1 text-xs">
                    <button
                      v-for="st in ['all', 'valid', 'invalid']"
                      :key="st"
                      @click="resultsFilterStatus = st as any; resultsPage = 1"
                      :class="[
                        'px-3 py-1 rounded-md transition-all font-medium',
                        resultsFilterStatus === st ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
                      ]"
                    >
                      {{ st === 'all' ? $t('whatsappFilter.allResults') : st === 'valid' ? $t('whatsappFilter.validResults') : $t('whatsappFilter.invalidResults') }}
                    </button>
                  </div>
                </div>
                <SearchInput
                  v-model="resultsSearchQuery"
                  :placeholder="$t('whatsappFilter.searchPlaceholder')"
                  class="w-72"
                />
              </div>
            </CardHeader>
            <CardContent class="p-0">
              <SelectableDataTable
                :items="activeBatchResults"
                :columns="resultColumns"
                :is-loading="isLoadingResults"
                row-key="id"
                
                :current-page="resultsPage"
                :total-items="totalResults"
                :page-size="resultsLimit"
                :total-pages="resultsTotalPages"
                
                :selected-ids="selectedIds"
                :is-all-matching-selected="isAllMatchingSelected"
                :is-all-page-selected="isAllPageSelected"
                :selected-count="selectedCount"
                
                @page-change="(p) => resultsPage = p"
                @pageSize-change="(s) => resultsLimit = s"
                @toggle-row="toggleRow"
                @toggle-page="togglePageSelection"
                @select-all-matching="selectAllMatching"
                @clear-selection="clearSelection"
              >
                <!-- status cell -->
                <template #cell-is_valid="{ item: row }">
                  <div class="flex items-center gap-1.5">
                    <CheckCircle2 class="h-4 w-4 text-emerald-500 fill-emerald-500/10" />
                    <span class="text-emerald-600 dark:text-emerald-400 font-medium">
                      {{ row.is_valid ? $t('whatsappFilter.validResults') : $t('whatsappFilter.invalidResults') }}
                    </span>
                  </div>
                </template>

                <!-- error message cell -->
                <template #cell-error_message="{ item: row }">
                  <span class="text-sm text-muted-foreground font-mono">{{ row.error_message || '-' }}</span>
                </template>
              </SelectableDataTable>
            </CardContent>
          </Card>
        </div>

        <!-- 2. CREATION VIEW & CAMPAIGNS LOG (When not viewing details) -->
        <div v-else class="grid gap-6 lg:grid-cols-3">
          <!-- Create batch form card -->
          <Card class="lg:col-span-1 border shadow-sm h-fit">
            <CardHeader>
              <CardTitle class="text-lg flex items-center gap-2">
                <Sparkles class="h-5 w-5 text-emerald-500" />
                {{ $t('whatsappFilter.newCheck') }}
              </CardTitle>
              <CardDescription>Setup a new filtering operation to verify phone registries.</CardDescription>
            </CardHeader>
            <CardContent class="space-y-4">
              <!-- Select Connection -->
              <div class="space-y-2">
                <Label class="text-sm font-semibold flex items-center gap-1">
                  {{ $t('whatsappFilter.connection') }}
                  <span class="text-destructive">*</span>
                </Label>
                <Select v-model="selectedConnectionId">
                  <SelectTrigger class="w-full">
                    <SelectValue :placeholder="$t('whatsappFilter.selectConnection')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="conn in connections" :key="conn.id" :value="conn.id">
                      {{ conn.name }}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <!-- Input Tab selection -->
              <div class="space-y-2">
                <Label class="text-sm font-semibold">Verification Method</Label>
                <div class="grid grid-cols-2 rounded-lg bg-secondary p-1 text-xs">
                  <button
                    @click="activeTab = 'paste'"
                    :class="[
                      'py-1.5 rounded-md transition-all font-semibold',
                      activeTab === 'paste' ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
                    ]"
                  >
                    Paste Numbers
                  </button>
                  <button
                    @click="activeTab = 'upload'"
                    :class="[
                      'py-1.5 rounded-md transition-all font-semibold',
                      activeTab === 'upload' ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
                    ]"
                  >
                    Upload CSV
                  </button>
                </div>
              </div>

              <!-- Tab Contents -->
              <div v-if="activeTab === 'paste'" class="space-y-2">
                <Label class="text-sm font-semibold">{{ $t('whatsappFilter.inputNumbers') }}</Label>
                <Textarea
                  v-model="pastedNumbers"
                  :rows="8"
                  :placeholder="$t('whatsappFilter.inputNumbersPlaceholder')"
                  class="font-mono text-sm resize-none rounded-lg"
                />
              </div>

              <div v-else class="space-y-2">
                <Label class="text-sm font-semibold">{{ $t('whatsappFilter.csvFile') }}</Label>
                <div
                  @dragover="onDragOver"
                  @dragleave="onDragLeave"
                  @drop="onDrop"
                  :class="[
                    'border-2 border-dashed rounded-xl p-6 flex flex-col items-center justify-center text-center cursor-pointer transition-all hover:bg-slate-500/5',
                    isDragActive ? 'border-emerald-500 bg-emerald-500/5' : 'border-slate-300 dark:border-slate-800'
                  ]"
                  @click="triggerFileSelect"
                >
                  <input type="file" ref="fileInput" accept=".csv" class="hidden" @change="onFileSelect" />
                  <FileSpreadsheet class="h-10 w-10 text-emerald-500 animate-bounce" />
                  <h4 class="font-bold text-sm mt-2">{{ csvFile ? csvFile.name : $t('whatsappFilter.csvFilePlaceholder') }}</h4>
                  <p class="text-xs text-muted-foreground mt-1 px-4 leading-relaxed">{{ $t('whatsappFilter.csvFileHint') }}</p>
                </div>
              </div>

              <Button
                variant="default"
                class="w-full gap-2 bg-gradient-to-br from-emerald-500 to-teal-600 hover:from-emerald-600 hover:to-teal-700 shadow-md text-white font-semibold pt-1.5"
                @click="startVerification"
                :disabled="filterStore.isSubmitting.value"
              >
                <Loader2 v-if="filterStore.isSubmitting.value" class="h-4 w-4 animate-spin" />
                <Sparkles v-else class="h-4 w-4" />
                {{ $t('whatsappFilter.startCheck') }}
              </Button>
            </CardContent>
          </Card>

          <!-- History Logs card -->
          <Card class="lg:col-span-2 border shadow-sm">
            <CardHeader class="border-b">
              <CardTitle class="text-lg flex items-center gap-2">
                <Calendar class="h-5 w-5 text-muted-foreground" />
                {{ $t('whatsappFilter.campaigns') }}
              </CardTitle>
              <CardDescription>View and manage previously run number filtering operations.</CardDescription>
            </CardHeader>
            <CardContent class="p-0">
              <DataTable
                :items="filterStore.batches.value"
                :columns="campaignColumns"
                :is-loading="filterStore.isLoadingBatches.value"
                :empty-icon="Layers"
                :empty-title="$t('whatsappFilter.noCampaigns')"
                empty-description="Create your first campaign check using the sidebar form."
                server-pagination
                :current-page="campaignsPage"
                :total-items="filterStore.totalBatches.value"
                :page-size="campaignsLimit"
                item-name="campaigns"
                @page-change="(p) => campaignsPage = p"
              >
                <!-- date cell -->
                <template #cell-created_at="{ item: row }">
                  <span class="font-semibold text-foreground text-sm">{{ formatDate(row.created_at) }}</span>
                </template>

                <!-- status cell -->
                <template #cell-status="{ item: row }">
                  <span
                    :class="[
                      'px-2 py-0.5 rounded-full text-xs font-semibold border flex items-center w-fit gap-1 shadow-sm',
                      row.status === 'completed' && 'bg-emerald-500/10 border-emerald-500/30 text-emerald-600 dark:text-emerald-400',
                      row.status === 'failed' && 'bg-rose-500/10 border-rose-500/30 text-rose-600 dark:text-rose-400',
                      (row.status === 'processing' || row.status === 'pending') && 'bg-blue-500/10 border-blue-500/30 text-blue-600 dark:text-blue-400 animate-pulse'
                    ]"
                  >
                    {{ $t(`whatsappFilter.status${(row.status || 'Pending').charAt(0).toUpperCase() + (row.status || 'pending').slice(1)}`, { percent: 100 }) }}
                  </span>
                </template>

                <!-- actions cell -->
                <template #cell-actions="{ item: row }">
                  <div class="flex items-center justify-end gap-1.5">
                    <Button variant="outline" size="sm" @click="viewBatch(row.id)">
                      View Logs
                    </Button>
                    <Button variant="ghost" size="icon" class="h-8 w-8 text-destructive hover:bg-destructive/10 hover:text-destructive" @click="openDeleteDialog(row)">
                      <Trash2 class="h-4 w-4" />
                    </Button>
                  </div>
                </template>
              </DataTable>
            </CardContent>
          </Card>
        </div>
      </div>
    </ScrollArea>

    <!-- Delete confirm dialog -->
    <DeleteConfirmDialog
      v-model:open="deleteDialogOpen"
      :title="$t('whatsappFilter.deleteConfirmTitle')"
      :item-name="batchToDelete ? `Batch ${(batchToDelete.id ?? '').slice(0, 8)}` : ''"
      @confirm="confirmDelete"
    >
      <p class="text-sm text-muted-foreground leading-relaxed">
        {{ $t("whatsappFilter.deleteConfirmDesc") }}
      </p>
    </DeleteConfirmDialog>
  </div>
</template>
