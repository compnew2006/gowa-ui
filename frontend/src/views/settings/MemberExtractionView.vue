<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "vue-sonner";
import {
  PageHeader,
  DataTable,
  SearchInput,
  type Column,
} from "@/components/shared";
import { getErrorMessage } from "@/lib/api-utils";
import {
  memberExtractionService,
  instancesService,
  type ExtractionCampaign,
  type MemberExtractionResult,
} from "@/services/api";
import {
  Plus,
  Pencil,
  Trash2,
  Play,
  Pause,
  Eye,
  Download,
} from "lucide-vue-next";
import { formatDate } from "@/lib/utils";
import { useDebounceFn } from "@vueuse/core";

const { t } = useI18n();

const campaigns = ref<ExtractionCampaign[]>([]);
const totalCampaigns = ref(0);
const loading = ref(false);
const searchQuery = ref("");
const statusFilter = ref("all");
const currentPage = ref(1);
const pageSize = ref(20);

const showCreateDialog = ref(false);
const showEditDialog = ref(false);
const showDeleteDialog = ref(false);
const showResultsDialog = ref(false);
const showStatsDialog = ref(false);
const selectedCampaign = ref<ExtractionCampaign | null>(null);

const formName = ref("");
const formInstanceId = ref("");
const formGroupJID = ref("");
const availableInstances = ref<{ id: string; name: string }[]>([]);

const results = ref<MemberExtractionResult[]>([]);
const totalResults = ref(0);
const resultStatusFilter = ref("all");
const loadingResults = ref(false);
const resultsPage = ref(1);

const campaignStats = ref<{
  total_members: number;
  extracted_count: number;
  failed_count: number;
  status: string;
} | null>(null);

const columns: Column<ExtractionCampaign>[] = [
  { key: "name", label: t("common.name"), sortable: true },
  { key: "instance_name", label: t("extraction.instance"), sortable: true },
  { key: "group_name", label: t("extraction.groupName"), sortable: true },
  { key: "status", label: t("common.status"), sortable: true },
  { key: "extracted_count", label: t("extraction.extracted"), sortable: true },
  { key: "failed_count", label: t("extraction.failed"), sortable: true },
  { key: "created_at", label: t("common.createdAt"), sortable: true },
];

const resultColumns: Column<MemberExtractionResult>[] = [
  { key: "phone_number", label: t("extraction.phoneNumber") },
  { key: "push_name", label: t("extraction.pushName") },
  { key: "participant_jid", label: "JID" },
  { key: "is_admin", label: t("extraction.isAdmin") },
  { key: "is_super_admin", label: t("extraction.isSuperAdmin") },
  { key: "status", label: t("common.status"), sortable: true },
  { key: "created_at", label: t("common.createdAt") },
];

const fetchCampaigns = async () => {
  loading.value = true;
  try {
    const params: Record<string, any> = {
      page: currentPage.value,
      limit: pageSize.value,
    };
    if (searchQuery.value) params.search = searchQuery.value;
    if (statusFilter.value && statusFilter.value !== "all") params.status = statusFilter.value;
    const response = await memberExtractionService.list(params);
    campaigns.value = response.data.data;
    totalCampaigns.value = response.data.total;
  } catch (error) {
    toast.error(getErrorMessage(error));
  } finally {
    loading.value = false;
  }
};

const fetchInstances = async () => {
  try {
    const response = await instancesService.list();
    availableInstances.value = (response.data as any[]).map((i: any) => ({ id: i.id, name: i.name }));
  } catch (error) {
    console.error("Failed to fetch instances:", error);
  }
};

const openCreateDialog = () => {
  formName.value = "";
  formInstanceId.value = "";
  formGroupJID.value = "";
  showCreateDialog.value = true;
};

const handleCreate = async () => {
  if (!formName.value.trim() || !formInstanceId.value || !formGroupJID.value.trim()) {
    toast.error(t("extraction.nameInstanceGroupRequired"));
    return;
  }
  try {
    await memberExtractionService.create({
      name: formName.value.trim(),
      instance_id: formInstanceId.value,
      group_jid: formGroupJID.value.trim(),
    });
    toast.success(t("extraction.campaignCreated"));
    showCreateDialog.value = false;
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const openEditDialog = (campaign: ExtractionCampaign) => {
  selectedCampaign.value = campaign;
  formName.value = campaign.name;
  formInstanceId.value = campaign.instance_id;
  formGroupJID.value = campaign.group_jid || "";
  showEditDialog.value = true;
};

const handleEdit = async () => {
  if (!selectedCampaign.value) return;
  try {
    await memberExtractionService.update(selectedCampaign.value.id, {
      name: formName.value.trim(),
      instance_id: formInstanceId.value,
      group_jid: formGroupJID.value.trim(),
    });
    toast.success(t("extraction.campaignUpdated"));
    showEditDialog.value = false;
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const openDeleteDialog = (campaign: ExtractionCampaign) => {
  selectedCampaign.value = campaign;
  showDeleteDialog.value = true;
};

const handleDelete = async () => {
  if (!selectedCampaign.value) return;
  try {
    await memberExtractionService.delete(selectedCampaign.value.id);
    toast.success(t("extraction.campaignDeleted"));
    showDeleteDialog.value = false;
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const handleStart = async (campaign: ExtractionCampaign) => {
  try {
    await memberExtractionService.start(campaign.id);
    toast.success(t("extraction.campaignStarted"));
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const handlePause = async (campaign: ExtractionCampaign) => {
  try {
    await memberExtractionService.pause(campaign.id);
    toast.success(t("extraction.campaignPaused"));
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const openResultsDialog = async (campaign: ExtractionCampaign) => {
  selectedCampaign.value = campaign;
  resultsPage.value = 1;
  resultStatusFilter.value = "all";
  showResultsDialog.value = true;
  await fetchResults();
};

const fetchResults = async () => {
  if (!selectedCampaign.value) return;
  loadingResults.value = true;
  try {
    const params: Record<string, any> = {
      page: resultsPage.value,
      limit: 20,
    };
    if (resultStatusFilter.value && resultStatusFilter.value !== "all") params.status = resultStatusFilter.value;
    const response = await memberExtractionService.getResults(selectedCampaign.value.id, params);
    results.value = response.data.data;
    totalResults.value = response.data.total;
  } catch (error) {
    toast.error(getErrorMessage(error));
  } finally {
    loadingResults.value = false;
  }
};

const handleExport = async (campaign: ExtractionCampaign) => {
  try {
    const response = await memberExtractionService.exportCSV(campaign.id);
    const url = window.URL.createObjectURL(new Blob([response.data as any]));
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute("download", `member-extraction-${campaign.name}.csv`);
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(url);
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const openStatsDialog = async (campaign: ExtractionCampaign) => {
  selectedCampaign.value = campaign;
  try {
    const response = await memberExtractionService.getStats(campaign.id);
    campaignStats.value = response.data;
    showStatsDialog.value = true;
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const getStatusBadgeVariant = (status: string) => {
  switch (status) {
    case "completed": return "default";
    case "processing": return "secondary";
    case "draft": return "outline";
    case "paused": return "destructive";
    case "failed": return "destructive";
    default: return "outline";
  }
};

const debouncedSearch = useDebounceFn(() => {
  currentPage.value = 1;
  fetchCampaigns();
}, 300);

watch(searchQuery, debouncedSearch);
watch(statusFilter, () => {
  currentPage.value = 1;
  fetchCampaigns();
});

onMounted(() => {
  fetchCampaigns();
  fetchInstances();
});
</script>

<template>
  <div class="flex flex-col h-full">
    <PageHeader :title="t('extraction.memberTitle')" :description="t('extraction.memberDescription')">
      <template #actions>
        <Button @click="openCreateDialog">
          <Plus class="w-4 h-4 mr-2" />
          {{ t("extraction.newCampaign") }}
        </Button>
      </template>
    </PageHeader>

    <div class="flex-1 overflow-hidden p-4">
      <div class="flex items-center gap-4 mb-4">
        <SearchInput v-model="searchQuery" :placeholder="t('extraction.searchPlaceholder')" class="w-64" />
        <Select v-model="statusFilter">
          <SelectTrigger class="w-40">
            <SelectValue :placeholder="t('extraction.allStatuses')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{{ t("extraction.allStatuses") }}</SelectItem>
            <SelectItem value="draft">{{ t("extraction.draft") }}</SelectItem>
            <SelectItem value="processing">{{ t("extraction.processing") }}</SelectItem>
            <SelectItem value="paused">{{ t("extraction.paused") }}</SelectItem>
            <SelectItem value="completed">{{ t("extraction.completed") }}</SelectItem>
            <SelectItem value="failed">{{ t("extraction.failed") }}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <DataTable
        :items="campaigns"
        :columns="columns"
        :is-loading="loading"
        :server-pagination="true"
        :current-page="currentPage"
        :total-items="totalCampaigns"
        :page-size="pageSize"
        @page-change="(page: number) => { currentPage = page; fetchCampaigns(); }"
      >
        <template #cell-status="{ item }">
          <Badge :variant="getStatusBadgeVariant(item.status)">
            {{ item.status }}
          </Badge>
        </template>
        <template #cell-name="{ item }">
          <span class="font-medium">{{ item.name }}</span>
        </template>
        <template #cell-instance_name="{ item }">
          {{ item.instance_name || "-" }}
        </template>
        <template #cell-group_name="{ item }">
          {{ item.group_name || "-" }}
        </template>
        <template #cell-created_at="{ item }">
          {{ formatDate(item.created_at) }}
        </template>
        <template #cell-actions="{ item }">
          <div class="flex items-center gap-1">
            <Button variant="ghost" size="icon" @click="openStatsDialog(item)" title="Stats">
              <Eye class="w-4 h-4" />
            </Button>
            <Button variant="ghost" size="icon" @click="openResultsDialog(item)" title="Results">
              <Eye class="w-4 h-4" />
            </Button>
            <Button variant="ghost" size="icon" @click="handleExport(item)" title="Export">
              <Download class="w-4 h-4" />
            </Button>
            <Button
              v-if="item.status === 'draft' || item.status === 'paused'"
              variant="ghost"
              size="icon"
              @click="handleStart(item)"
              title="Start"
            >
              <Play class="w-4 h-4 text-green-600" />
            </Button>
            <Button
              v-if="item.status === 'processing'"
              variant="ghost"
              size="icon"
              @click="handlePause(item)"
              title="Pause"
            >
              <Pause class="w-4 h-4 text-yellow-600" />
            </Button>
            <Button
              v-if="item.status !== 'processing'"
              variant="ghost"
              size="icon"
              @click="openEditDialog(item)"
              title="Edit"
            >
              <Pencil class="w-4 h-4" />
            </Button>
            <Button
              v-if="item.status !== 'processing'"
              variant="ghost"
              size="icon"
              @click="openDeleteDialog(item)"
              title="Delete"
            >
              <Trash2 class="w-4 h-4 text-red-600" />
            </Button>
          </div>
        </template>
      </DataTable>
    </div>

    <!-- Create Dialog -->
    <Dialog v-model:open="showCreateDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ t("extraction.newCampaign") }}</DialogTitle>
          <DialogDescription>{{ t("extraction.createMemberDescription") }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-4">
          <div>
            <Label>{{ t("extraction.campaignName") }}</Label>
            <Input v-model="formName" :placeholder="t('extraction.campaignNamePlaceholder')" />
          </div>
          <div>
            <Label>{{ t("extraction.instance") }}</Label>
            <Select v-model="formInstanceId">
              <SelectTrigger>
                <SelectValue :placeholder="t('extraction.selectInstance')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="inst in availableInstances" :key="inst.id" :value="inst.id">
                  {{ inst.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label>{{ t("extraction.groupJID") }}</Label>
            <Input v-model="formGroupJID" placeholder="123456789-1234567890@g.us" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showCreateDialog = false">{{ t("common.cancel") }}</Button>
          <Button @click="handleCreate">{{ t("extraction.createCampaign") }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Edit Dialog -->
    <Dialog v-model:open="showEditDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ t("extraction.editCampaign") }}</DialogTitle>
          <DialogDescription>{{ t("extraction.editDescription") }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-4">
          <div>
            <Label>{{ t("extraction.campaignName") }}</Label>
            <Input v-model="formName" :placeholder="t('extraction.campaignNamePlaceholder')" />
          </div>
          <div>
            <Label>{{ t("extraction.instance") }}</Label>
            <Select v-model="formInstanceId">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="inst in availableInstances" :key="inst.id" :value="inst.id">
                  {{ inst.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label>{{ t("extraction.groupJID") }}</Label>
            <Input v-model="formGroupJID" placeholder="123456789-1234567890@g.us" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showEditDialog = false">{{ t("common.cancel") }}</Button>
          <Button @click="handleEdit">{{ t("common.save") }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Delete Dialog -->
    <AlertDialog v-model:open="showDeleteDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ t("extraction.deleteCampaign") }}</AlertDialogTitle>
          <AlertDialogDescription>
            {{ t("extraction.deleteConfirmation", { name: selectedCampaign?.name }) }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ t("common.cancel") }}</AlertDialogCancel>
          <AlertDialogAction @click="handleDelete" class="bg-red-600 hover:bg-red-700">
            {{ t("common.delete") }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Results Dialog -->
    <Dialog v-model:open="showResultsDialog">
      <DialogContent class="max-w-4xl">
        <DialogHeader>
          <DialogTitle>{{ t("extraction.results") }} - {{ selectedCampaign?.name }}</DialogTitle>
          <DialogDescription>{{ t("extraction.memberResultsDescription") }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-4">
          <div class="flex items-center gap-4">
            <Select v-model="resultStatusFilter" @update:model-value="fetchResults">
              <SelectTrigger class="w-40">
                <SelectValue :placeholder="t('extraction.allStatuses')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{{ t("extraction.allStatuses") }}</SelectItem>
                <SelectItem value="extracted">{{ t("extraction.extracted") }}</SelectItem>
                <SelectItem value="failed">{{ t("extraction.failed") }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <DataTable
            :items="results"
            :columns="resultColumns"
            :is-loading="loadingResults"
            :server-pagination="true"
            :current-page="resultsPage"
            :total-items="totalResults"
            :page-size="20"
            @page-change="(page: number) => { resultsPage = page; fetchResults(); }"
          >
            <template #cell-status="{ item }">
              <Badge :variant="item.status === 'extracted' ? 'default' : 'destructive'">
                {{ item.status }}
              </Badge>
            </template>
            <template #cell-participant_jid="{ item }">
              <span class="font-mono text-xs">{{ item.participant_jid }}</span>
            </template>
            <template #cell-phone_number="{ item }">
              {{ item.phone_number || "-" }}
            </template>
            <template #cell-push_name="{ item }">
              {{ item.push_name || "-" }}
            </template>
            <template #cell-is_admin="{ item }">
              <Badge :variant="item.is_admin ? 'default' : 'outline'">
                {{ item.is_admin ? t("common.yes") : t("common.no") }}
              </Badge>
            </template>
            <template #cell-is_super_admin="{ item }">
              <Badge :variant="item.is_super_admin ? 'default' : 'outline'">
                {{ item.is_super_admin ? t("common.yes") : t("common.no") }}
              </Badge>
            </template>
            <template #cell-created_at="{ item }">
              {{ formatDate(item.created_at) }}
            </template>
          </DataTable>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Stats Dialog -->
    <Dialog v-model:open="showStatsDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ t("extraction.campaignStats") }}</DialogTitle>
          <DialogDescription>{{ selectedCampaign?.name }}</DialogDescription>
        </DialogHeader>
        <div v-if="campaignStats" class="grid grid-cols-3 gap-4">
          <div class="text-center p-4 border rounded-lg">
            <div class="text-3xl font-bold">{{ campaignStats.total_members }}</div>
            <div class="text-sm text-muted-foreground">{{ t("extraction.totalMembers") }}</div>
          </div>
          <div class="text-center p-4 border rounded-lg">
            <div class="text-3xl font-bold text-green-600">{{ campaignStats.extracted_count }}</div>
            <div class="text-sm text-muted-foreground">{{ t("extraction.extracted") }}</div>
          </div>
          <div class="text-center p-4 border rounded-lg">
            <div class="text-3xl font-bold text-red-600">{{ campaignStats.failed_count }}</div>
            <div class="text-sm text-muted-foreground">{{ t("extraction.failed") }}</div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
