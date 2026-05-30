<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
  ImportExportDialog,
  type Column,
} from "@/components/shared";
import { toast } from "vue-sonner";
import {
  Smartphone,
  Users,
  MessageSquare,
  Download,
  RefreshCw,
  Loader2,
  Activity,
} from "lucide-vue-next";
import {
  extractService,
  unwrapEnvelope,
  type ExtractContact,
  type ExtractStats,
  type ExtractContactsResponse,
  type ExtractStatsResponse,
} from "@/services/extractService";
import { instancesService } from "@/services/api";
import { formatDate } from "@/lib/utils";

const { t, locale } = useI18n();
const isRtl = computed(() => locale.value === "ar");

const instances = ref<Array<{ id: string; name: string; phone_number: string }>>([]);
const selectedInstanceId = ref("");
const isLoading = ref(false);
const isSyncing = ref(false);
const isImportExportOpen = ref(false);

const contacts = ref<ExtractContact[]>([]);
const totalContacts = ref(0);
const currentPage = ref(1);
const pageSize = 50;
const searchQuery = ref("");

const stats = ref<ExtractStats[]>([]);
const isLoadingStats = ref(false);

const columns = computed<Column<ExtractContact>[]>(() => [
  {
    key: "phone_number",
    label: t("extract.phoneNumber"),
    sortable: true,
  },
  {
    key: "profile_name",
    label: t("extract.profileName"),
    sortable: true,
  },
  {
    key: "last_message_at",
    label: t("extract.lastMessage"),
    sortable: true,
  },
  {
    key: "message_count",
    label: t("extract.messageCount"),
    align: "center",
    sortable: true,
  },
]);

async function loadInstances() {
  try {
    const response = await instancesService.list();
    instances.value = unwrapEnvelope<Array<{ id: string; name: string; phone_number: string }>>(response) ?? [];
  } catch (err: any) {
    toast.error(err?.response?.data?.message || t("extract.failedToLoad"));
  }
}

async function loadStats() {
  isLoadingStats.value = true;
  try {
    const params: Record<string, string> = {};
    if (selectedInstanceId.value) {
      params.instance_id = selectedInstanceId.value;
    }
    const response = await extractService.getStats(params);
    stats.value = unwrapEnvelope<ExtractStatsResponse>(response).stats ?? [];
  } catch (err: any) {
    toast.error(err?.response?.data?.message || t("extract.failedToLoadStats"));
  } finally {
    isLoadingStats.value = false;
  }
}

async function loadContacts() {
  isLoading.value = true;
  try {
    const params: Record<string, any> = {
      page: currentPage.value,
      limit: pageSize,
    };
    if (selectedInstanceId.value) {
      params.instance_id = selectedInstanceId.value;
    }
    if (searchQuery.value.trim()) {
      params.search = searchQuery.value.trim();
    }
    const response = await extractService.listContacts(params);
    const payload = unwrapEnvelope<ExtractContactsResponse>(response);
    contacts.value = payload?.data ?? [];
    totalContacts.value = payload?.total ?? 0;
  } catch (err: any) {
    toast.error(err?.response?.data?.message || t("extract.failedToLoad"));
  } finally {
    isLoading.value = false;
  }
}

function onSearch(value: string) {
  searchQuery.value = value;
  currentPage.value = 1;
  loadContacts();
}

function onPageChange(page: number) {
  currentPage.value = page;
  loadContacts();
}

function onInstanceChange(value: any) {
  selectedInstanceId.value = String(value ?? "");
  currentPage.value = 1;
  loadContacts();
  loadStats();
}

async function handleSync() {
  if (!selectedInstanceId.value) {
    toast.error(t("extract.selectInstanceFirst"));
    return;
  }
  isSyncing.value = true;
  try {
    await extractService.triggerSync({
      instance_id: selectedInstanceId.value,
    });
    toast.success(t("extract.syncSuccess"));
    loadStats();
  } catch (err: any) {
    toast.error(err?.response?.data?.message || t("extract.syncFailed"));
  } finally {
    isSyncing.value = false;
  }
}

onMounted(async () => {
  await loadInstances();
  await Promise.all([loadStats(), loadContacts()]);
});
</script>

<template>
  <div :dir="isRtl ? 'rtl' : 'ltr'">
    <PageHeader :title="t('extract.title')" :description="t('extract.description')">
      <div class="flex items-center gap-2">
        <Select :model-value="selectedInstanceId" @update:model-value="onInstanceChange">
          <SelectTrigger class="w-[220px]">
            <SelectValue :placeholder="t('extract.selectInstance')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">
              {{ t("extract.allInstances") }}
            </SelectItem>
            <SelectItem
              v-for="inst in instances"
              :key="inst.id"
              :value="inst.id"
            >
              {{ inst.name }}
            </SelectItem>
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          :disabled="!selectedInstanceId || isSyncing"
          @click="handleSync"
        >
          <RefreshCw v-if="!isSyncing" class="h-4 w-4 me-1" />
          <Loader2 v-else class="h-4 w-4 me-1 animate-spin" />
          {{ t("extract.sync") }}
        </Button>
      </div>
    </PageHeader>

    <div class="p-6 space-y-6">
      <!-- Stats Cards -->
      <div v-if="isLoadingStats" class="flex justify-center py-4">
        <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
      <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card v-for="stat in stats" :key="stat.instance_id">
          <CardHeader class="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle class="text-sm font-medium truncate">
              {{ stat.instance_name }}
            </CardTitle>
            <Smartphone class="h-4 w-4 text-muted-foreground shrink-0" />
          </CardHeader>
          <CardContent>
            <div class="grid grid-cols-2 gap-2 text-sm">
              <div class="flex items-center gap-1">
                <Users class="h-3 w-3 text-muted-foreground" />
                <span>{{ stat.total_contacts }}</span>
                <span class="text-muted-foreground text-xs">{{ t("extract.contacts") }}</span>
              </div>
              <div class="flex items-center gap-1">
                <MessageSquare class="h-3 w-3 text-muted-foreground" />
                <span>{{ stat.total_messages }}</span>
                <span class="text-muted-foreground text-xs">{{ t("extract.messages") }}</span>
              </div>
            </div>
            <div class="flex items-center gap-1 mt-2 text-xs text-muted-foreground">
              <Activity class="h-3 w-3" />
              <Badge
                :variant="stat.status === 'connected' ? 'success' : 'secondary'"
                class="text-xs px-1.5 py-0"
              >
                {{ stat.status }}
              </Badge>
            </div>
          </CardContent>
        </Card>
        <div
          v-if="stats.length === 0"
          class="col-span-full text-center py-8 text-muted-foreground"
        >
          {{ t("extract.noData") }}
        </div>
      </div>

      <!-- Contacts Table -->
      <Card>
        <CardHeader class="flex flex-row items-center justify-between">
          <div class="flex items-center gap-4">
            <SearchInput
              :model-value="searchQuery"
              :placeholder="t('extract.searchPlaceholder')"
              @search="onSearch"
              @clear="onSearch('')"
            />
          </div>
          <Button variant="outline" @click="isImportExportOpen = true">
            <Download class="h-4 w-4 me-1" />
            {{ t("extract.export") }}
          </Button>
        </CardHeader>
        <CardContent>
          <DataTable
            :items="contacts"
            :columns="columns"
            :is-loading="isLoading"
            :server-pagination="true"
            :current-page="currentPage"
            :total-items="totalContacts"
            :page-size="pageSize"
            :item-name="t('extract.contacts')"
            @page-change="onPageChange"
          >
            <template #cell-last_message_at="{ item }">
              <span class="text-muted-foreground text-sm">
                {{ item.last_message_at ? formatDate(item.last_message_at) : "—" }}
              </span>
            </template>
            <template #cell-message_count="{ item }">
              <Badge variant="secondary" class="font-mono">
                {{ item.message_count }}
              </Badge>
            </template>
            <template #empty>
              <div class="text-center py-12">
                <MessageSquare class="h-12 w-12 mx-auto text-muted-foreground/50" />
                <p class="mt-4 text-muted-foreground">{{ t("extract.noContacts") }}</p>
              </div>
            </template>
          </DataTable>
        </CardContent>
      </Card>
    </div>

    <ImportExportDialog
      v-model:open="isImportExportOpen"
      table="extracted_messages"
      :table-label="t('extract.title')"
      :filters="
        selectedInstanceId || searchQuery
          ? {
              ...(searchQuery ? { search: searchQuery } : {}),
              ...(selectedInstanceId ? { instance_id: selectedInstanceId } : {}),
            }
          : undefined
      "
      :can-import="false"
      :can-export="true"
    />
  </div>
</template>
