<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
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
  groupJoinCampaignsService,
  accountsService,
  type GroupJoinCampaign,
  type GroupJoinRecipient,
} from "@/services/api";
import {
  Plus,
  Pencil,
  Trash2,
  Play,
  Pause,
  Users,
  Eye,
  Upload,
} from "lucide-vue-next";
import { formatDate } from "@/lib/utils";
import { useDebounceFn } from "@vueuse/core";

const { t } = useI18n();

const campaigns = ref<GroupJoinCampaign[]>([]);
const totalCampaigns = ref(0);
const loading = ref(false);
const searchQuery = ref("");
const statusFilter = ref("all");
const currentPage = ref(1);
const pageSize = ref(20);

const showCreateDialog = ref(false);
const showEditDialog = ref(false);
const showDeleteDialog = ref(false);
const showRecipientsDialog = ref(false);
const showStatsDialog = ref(false);
const selectedCampaign = ref<GroupJoinCampaign | null>(null);

const formName = ref("");
const formAccounts = ref<string[]>([]);
const formSpeed = ref<string>("slow");
const availableAccounts = ref<string[]>([]);

const recipients = ref<GroupJoinRecipient[]>([]);
const recipientStatusFilter = ref("all");
const loadingRecipients = ref(false);
const showAddRecipientsDialog = ref(false);
const newInviteLinks = ref("");
const fileInput = ref<HTMLInputElement | null>(null);
const uploadingFile = ref(false);

const campaignStats = ref<{
  total_recipients: number;
  joined_count: number;
  failed_count: number;
  skipped_count: number;
  status: string;
} | null>(null);

const columns: Column<GroupJoinCampaign>[] = [
  { key: "name", label: t("common.name"), sortable: true },
  { key: "speed", label: "Speed", sortable: true },
  { key: "status", label: t("common.status"), sortable: true },
  { key: "total_recipients", label: "Total", sortable: true },
  { key: "joined_count", label: "Joined", sortable: true },
  { key: "failed_count", label: "Failed", sortable: true },
  { key: "created_at", label: t("common.createdAt"), sortable: true },
];

const recipientColumns: Column<GroupJoinRecipient>[] = [
  { key: "invite_link", label: "Invite Link" },
  { key: "group_name", label: "Group Name" },
  { key: "status", label: t("common.status"), sortable: true },
  { key: "group_jid", label: "Group JID" },
  { key: "error_message", label: "Error" },
  { key: "processed_at", label: "Processed At" },
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
    const response = await groupJoinCampaignsService.list(params);
    campaigns.value = response.data.data;
    totalCampaigns.value = response.data.total;
  } catch (error) {
    toast.error(getErrorMessage(error));
  } finally {
    loading.value = false;
  }
};

const fetchAccounts = async () => {
  try {
    const response = await accountsService.list();
    availableAccounts.value = response.data.map((a: any) => a.name);
  } catch (error) {
    console.error("Failed to fetch accounts:", error);
  }
};

const openCreateDialog = () => {
  formName.value = "";
  formAccounts.value = [];
  formSpeed.value = "slow";
  showCreateDialog.value = true;
};

const handleCreate = async () => {
  if (!formName.value.trim()) {
    toast.error("Name is required");
    return;
  }
  try {
    await groupJoinCampaignsService.create({
      name: formName.value.trim(),
      accounts: formAccounts.value,
      speed: formSpeed.value,
    });
    toast.success("Campaign created");
    showCreateDialog.value = false;
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const openEditDialog = (campaign: GroupJoinCampaign) => {
  selectedCampaign.value = campaign;
  formName.value = campaign.name;
  formAccounts.value = [...campaign.accounts];
  formSpeed.value = campaign.speed;
  showEditDialog.value = true;
};

const handleEdit = async () => {
  if (!selectedCampaign.value) return;
  try {
    await groupJoinCampaignsService.update(selectedCampaign.value.id, {
      name: formName.value.trim(),
      accounts: formAccounts.value,
      speed: formSpeed.value,
    });
    toast.success("Campaign updated");
    showEditDialog.value = false;
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const openDeleteDialog = (campaign: GroupJoinCampaign) => {
  selectedCampaign.value = campaign;
  showDeleteDialog.value = true;
};

const handleDelete = async () => {
  if (!selectedCampaign.value) return;
  try {
    await groupJoinCampaignsService.delete(selectedCampaign.value.id);
    toast.success("Campaign deleted");
    showDeleteDialog.value = false;
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const handleStart = async (campaign: GroupJoinCampaign) => {
  try {
    await groupJoinCampaignsService.start(campaign.id);
    toast.success("Campaign started");
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const handlePause = async (campaign: GroupJoinCampaign) => {
  try {
    await groupJoinCampaignsService.pause(campaign.id);
    toast.success("Campaign paused");
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const openRecipientsDialog = async (campaign: GroupJoinCampaign) => {
  selectedCampaign.value = campaign;
  showRecipientsDialog.value = true;
  await fetchRecipients();
};

const fetchRecipients = async () => {
  if (!selectedCampaign.value) return;
  loadingRecipients.value = true;
  try {
    const params: Record<string, any> = {};
    if (recipientStatusFilter.value && recipientStatusFilter.value !== "all") params.status = recipientStatusFilter.value;
    const response = await groupJoinCampaignsService.getRecipients(selectedCampaign.value.id, params);
    recipients.value = response.data.data;
  } catch (error) {
    toast.error(getErrorMessage(error));
  } finally {
    loadingRecipients.value = false;
  }
};

const openAddRecipientsDialog = () => {
  newInviteLinks.value = "";
  showAddRecipientsDialog.value = true;
};

const handleAddRecipients = async () => {
  if (!selectedCampaign.value) return;
  const links = newInviteLinks.value
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l.length > 0);
  if (links.length === 0) {
    toast.error("No invite links provided");
    return;
  }
  try {
    await groupJoinCampaignsService.addRecipients(selectedCampaign.value.id, {
      invite_links: links,
    });
    toast.success(`${links.length} recipients added`);
    showAddRecipientsDialog.value = false;
    fetchRecipients();
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const handleFileUpload = async (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file || !selectedCampaign.value) return;
  uploadingFile.value = true;
  try {
    await groupJoinCampaignsService.uploadRecipientsCSV(selectedCampaign.value.id, file);
    toast.success("File uploaded");
    fetchRecipients();
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  } finally {
    uploadingFile.value = false;
    if (fileInput.value) fileInput.value.value = "";
  }
};

const handleDeleteRecipient = async (recipient: GroupJoinRecipient) => {
  if (!selectedCampaign.value) return;
  try {
    await groupJoinCampaignsService.deleteRecipient(selectedCampaign.value.id, recipient.id);
    toast.success("Recipient deleted");
    fetchRecipients();
    fetchCampaigns();
  } catch (error) {
    toast.error(getErrorMessage(error));
  }
};

const openStatsDialog = async (campaign: GroupJoinCampaign) => {
  selectedCampaign.value = campaign;
  try {
    const response = await groupJoinCampaignsService.getStats(campaign.id);
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
  fetchAccounts();
});
</script>

<template>
  <div class="flex flex-col h-full">
    <PageHeader title="Group Join Campaigns" description="Manage WhatsApp group join campaigns">
      <template #actions>
        <Button @click="openCreateDialog">
          <Plus class="w-4 h-4 mr-2" />
          New Campaign
        </Button>
      </template>
    </PageHeader>

    <div class="flex-1 overflow-hidden p-4">
      <div class="flex items-center gap-4 mb-4">
        <SearchInput v-model="searchQuery" placeholder="Search campaigns..." class="w-64" />
        <Select v-model="statusFilter">
            <SelectTrigger class="w-40">
              <SelectValue placeholder="All Statuses" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Statuses</SelectItem>
              <SelectItem value="draft">Draft</SelectItem>
              <SelectItem value="processing">Processing</SelectItem>
              <SelectItem value="paused">Paused</SelectItem>
              <SelectItem value="completed">Completed</SelectItem>
              <SelectItem value="failed">Failed</SelectItem>
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
        <template #cell-speed="{ item }">
          {{ item.speed === "fast" ? "Fast ⚡" : "Slow 🐢" }}
        </template>
        <template #cell-created_at="{ item }">
          {{ formatDate(item.created_at) }}
        </template>
        <template #cell-name="{ item }">
          <span class="font-medium">{{ item.name }}</span>
        </template>
        <template #cell-actions="{ item }">
          <div class="flex items-center gap-1">
            <Button variant="ghost" size="icon" @click="openStatsDialog(item)" title="Stats">
              <Eye class="w-4 h-4" />
            </Button>
            <Button variant="ghost" size="icon" @click="openRecipientsDialog(item)" title="Recipients">
              <Users class="w-4 h-4" />
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
          <DialogTitle>New Campaign</DialogTitle>
          <DialogDescription>Create a new group join campaign</DialogDescription>
        </DialogHeader>
        <div class="space-y-4">
          <div>
            <Label>Campaign Name</Label>
            <Input v-model="formName" placeholder="Enter campaign name" />
          </div>
          <div>
            <Label>WhatsApp Accounts</Label>
            <div class="space-y-2">
              <div v-for="account in availableAccounts" :key="account" class="flex items-center gap-2">
                <input type="checkbox" :id="'create-' + account" :value="account" v-model="formAccounts" class="rounded" />
                <Label :for="'create-' + account">{{ account }}</Label>
              </div>
              <p v-if="availableAccounts.length === 0" class="text-sm text-muted-foreground">
                No WhatsApp accounts configured
              </p>
            </div>
          </div>
          <div>
            <Label>Speed</Label>
            <Select v-model="formSpeed">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="slow">Slow (30s delay, free)</SelectItem>
                <SelectItem value="fast">Fast (5s delay, costs points)</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showCreateDialog = false">Cancel</Button>
          <Button @click="handleCreate">Create Campaign</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Edit Dialog -->
    <Dialog v-model:open="showEditDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Campaign</DialogTitle>
          <DialogDescription>Update campaign settings</DialogDescription>
        </DialogHeader>
        <div class="space-y-4">
          <div>
            <Label>Campaign Name</Label>
            <Input v-model="formName" placeholder="Enter campaign name" />
          </div>
          <div>
            <Label>WhatsApp Accounts</Label>
            <div class="space-y-2">
              <div v-for="account in availableAccounts" :key="account" class="flex items-center gap-2">
                <input type="checkbox" :id="'edit-' + account" :value="account" v-model="formAccounts" class="rounded" />
                <Label :for="'edit-' + account">{{ account }}</Label>
              </div>
              <p v-if="availableAccounts.length === 0" class="text-sm text-muted-foreground">
                No WhatsApp accounts configured
              </p>
            </div>
          </div>
          <div>
            <Label>Speed</Label>
            <Select v-model="formSpeed">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="slow">Slow (30s delay, free)</SelectItem>
                <SelectItem value="fast">Fast (5s delay, costs points)</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showEditDialog = false">Cancel</Button>
          <Button @click="handleEdit">Save Changes</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Delete Dialog -->
    <AlertDialog v-model:open="showDeleteDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete Campaign</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to delete "{{ selectedCampaign?.name }}"? This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction @click="handleDelete" class="bg-red-600 hover:bg-red-700">
            Delete
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Recipients Dialog -->
    <Dialog v-model:open="showRecipientsDialog">
      <DialogContent class="max-w-4xl">
        <DialogHeader>
          <DialogTitle>Recipients - {{ selectedCampaign?.name }}</DialogTitle>
          <DialogDescription>Manage invite links for this campaign</DialogDescription>
        </DialogHeader>
        <div class="space-y-4">
          <div class="flex items-center gap-4">
            <Button @click="openAddRecipientsDialog">
              <Plus class="w-4 h-4 mr-2" />
              Add Links
            </Button>
            <div>
              <input ref="fileInput" type="file" accept=".csv" @change="handleFileUpload" class="hidden" />
              <Button variant="outline" @click="fileInput?.click()" :disabled="uploadingFile">
                <Upload class="w-4 h-4 mr-2" />
                {{ uploadingFile ? "Uploading..." : "Upload CSV" }}
              </Button>
            </div>
            <Select v-model="recipientStatusFilter" @update:model-value="fetchRecipients">
              <SelectTrigger class="w-40">
                <SelectValue placeholder="All Statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Statuses</SelectItem>
                <SelectItem value="pending">Pending</SelectItem>
                <SelectItem value="joined">Joined</SelectItem>
                <SelectItem value="failed">Failed</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <DataTable
            :items="recipients"
            :columns="recipientColumns"
            :is-loading="loadingRecipients"
          >
            <template #cell-status="{ item }">
              <Badge :variant="item.status === 'joined' ? 'default' : item.status === 'failed' ? 'destructive' : 'outline'">
                {{ item.status }}
              </Badge>
            </template>
            <template #cell-invite_link="{ item }">
              <span class="font-mono text-xs">{{ item.invite_link.substring(0, 40) }}...</span>
            </template>
            <template #cell-group_name="{ item }">
              {{ item.group_name || "-" }}
            </template>
            <template #cell-group_jid="{ item }">
              <span class="font-mono text-xs">{{ item.group_jid || "-" }}</span>
            </template>
            <template #cell-error_message="{ item }">
              <span class="text-red-600 text-xs">{{ item.error_message || "-" }}</span>
            </template>
            <template #cell-processed_at="{ item }">
              {{ item.processed_at ? formatDate(item.processed_at) : "-" }}
            </template>
            <template #cell-actions="{ item }">
              <Button
                v-if="item.status === 'pending'"
                variant="ghost"
                size="icon"
                @click="handleDeleteRecipient(item)"
              >
                <Trash2 class="w-4 h-4 text-red-600" />
              </Button>
            </template>
          </DataTable>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Add Recipients Dialog -->
    <Dialog v-model:open="showAddRecipientsDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Invite Links</DialogTitle>
          <DialogDescription>Paste invite links (one per line)</DialogDescription>
        </DialogHeader>
        <div>
          <Textarea
            v-model="newInviteLinks"
            placeholder="https://chat.whatsapp.com/XXXXX&#10;https://chat.whatsapp.com/YYYYY"
            :rows="8"
          />
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showAddRecipientsDialog = false">Cancel</Button>
          <Button @click="handleAddRecipients">Add Links</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Stats Dialog -->
    <Dialog v-model:open="showStatsDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Campaign Stats</DialogTitle>
          <DialogDescription>{{ selectedCampaign?.name }}</DialogDescription>
        </DialogHeader>
        <div v-if="campaignStats" class="grid grid-cols-2 gap-4">
          <div class="text-center p-4 border rounded-lg">
            <div class="text-3xl font-bold">{{ campaignStats.total_recipients }}</div>
            <div class="text-sm text-muted-foreground">Total Recipients</div>
          </div>
          <div class="text-center p-4 border rounded-lg">
            <div class="text-3xl font-bold text-green-600">{{ campaignStats.joined_count }}</div>
            <div class="text-sm text-muted-foreground">Joined</div>
          </div>
          <div class="text-center p-4 border rounded-lg">
            <div class="text-3xl font-bold text-red-600">{{ campaignStats.failed_count }}</div>
            <div class="text-sm text-muted-foreground">Failed</div>
          </div>
          <div class="text-center p-4 border rounded-lg">
            <div class="text-3xl font-bold text-yellow-600">{{ campaignStats.skipped_count }}</div>
            <div class="text-sm text-muted-foreground">Skipped</div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
