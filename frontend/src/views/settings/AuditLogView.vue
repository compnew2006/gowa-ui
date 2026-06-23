<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { storeToRefs } from "pinia";
import {
  PageHeader,
  DataTable,
  SearchInput,
  type Column,
} from "@/components/shared";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ShieldCheck, RotateCcw, CheckCircle2, XCircle } from "lucide-vue-next";
import { useAuditStore } from "@/stores/audit";
import type { AuditEvent } from "@/services/audit";
import { formatDate } from "@/lib/utils";

const { t } = useI18n();
const store = useAuditStore();
const { events, total, loading, filters } = storeToRefs(store);

// Filter bar local models (applied to the store on change).
const category = ref<string>("");
const action = ref<string>("");
const source = ref<string>("");
const success = ref<string>(""); // "" | "true" | "false"
const dateFrom = ref<string>("");
const dateTo = ref<string>("");

const categoryOptions = [
  "auth",
  "chat",
  "admin",
  "system",
  "campaign",
  "template",
];
const sourceOptions = ["user", "system", "worker", "scheduled"];

const categoryVariant: Record<string, string> = {
  auth: "bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300",
  chat: "bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300",
  admin:
    "bg-purple-100 text-purple-800 dark:bg-purple-900/40 dark:text-purple-300",
  system: "bg-gray-100 text-gray-800 dark:bg-gray-900/40 dark:text-gray-300",
  campaign:
    "bg-orange-100 text-orange-800 dark:bg-orange-900/40 dark:text-orange-300",
  template:
    "bg-teal-100 text-teal-800 dark:bg-teal-900/40 dark:text-teal-300",
};

const columns = computed<Column<AuditEvent>[]>(() => [
  { key: "created_at", label: t("settings.auditLog.columns.time"), width: "180px" },
  { key: "category", label: t("settings.auditLog.columns.category"), width: "120px" },
  { key: "action", label: t("settings.auditLog.columns.action"), width: "180px" },
  { key: "actor_email", label: t("settings.auditLog.columns.actor"), width: "200px" },
  { key: "target", label: t("settings.auditLog.columns.target"), width: "200px" },
  { key: "outcome", label: t("settings.auditLog.columns.outcome"), align: "center", width: "100px" },
  { key: "details", label: t("settings.auditLog.columns.details") },
]);

function formatTarget(evt: AuditEvent): string {
  if (!evt.target_type) return "—";
  return evt.target_id ? `${evt.target_type}:${evt.target_id.slice(0, 8)}` : evt.target_type;
}

function onSearch(value: string): void {
  store.setFilter("q", value || undefined);
  void store.fetch();
}

function applyCategory(v: unknown): void {
  const s = String(v);
  category.value = s === "all" ? "" : s;
  store.setFilter("category", category.value || undefined);
  void store.fetch();
}
function applyAction(): void {
  store.setFilter("action", action.value || undefined);
  void store.fetch();
}
function applySource(v: unknown): void {
  const s = String(v);
  source.value = s === "all" ? "" : s;
  store.setFilter("source", source.value || undefined);
  void store.fetch();
}
function applySuccess(v: unknown): void {
  const s = String(v);
  success.value = s === "all" ? "" : s;
  if (success.value === "") {
    store.setFilter("success", undefined);
  } else {
    store.setFilter("success", success.value === "true");
  }
  void store.fetch();
}
function applyDateRange(): void {
  store.setFilter("date_from", dateFrom.value || undefined);
  store.setFilter("date_to", dateTo.value || undefined);
  void store.fetch();
}

function resetAllFilters(): void {
  category.value = "";
  action.value = "";
  source.value = "";
  success.value = "";
  dateFrom.value = "";
  dateTo.value = "";
  store.resetFilters();
  void store.fetch();
}

function onPageChange(page: number): void {
  void store.goToPage(page);
}

onMounted(() => {
  void store.fetch();
});
</script>

<template>
  <div>
    <PageHeader
      :title="t('settings.auditLog.title')"
      :description="t('settings.auditLog.subtitle')"
      :icon="ShieldCheck"
      icon-gradient="bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300"
    />

    <div class="p-6 space-y-4">
      <!-- Filter bar -->
      <div
        class="flex flex-wrap items-end gap-3 rounded-lg border border-border bg-card p-4"
      >
        <div class="flex flex-col gap-1">
          <span class="text-xs text-muted-foreground">{{ t("settings.auditLog.filters.search") }}</span>
          <SearchInput
            :model-value="(filters.q as string) ?? ''"
            :placeholder="t('settings.auditLog.filters.search')"
            class="w-64"
            @update:model-value="onSearch"
          />
        </div>

        <div class="flex flex-col gap-1">
          <span class="text-xs text-muted-foreground">{{ t("settings.auditLog.filters.category") }}</span>
          <Select :model-value="category || 'all'" @update:model-value="applyCategory">
            <SelectTrigger class="w-40">
              <SelectValue :placeholder="t('settings.auditLog.filters.any')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ t("settings.auditLog.filters.any") }}</SelectItem>
              <SelectItem v-for="c in categoryOptions" :key="c" :value="c">{{ c }}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="flex flex-col gap-1">
          <span class="text-xs text-muted-foreground">{{ t("settings.auditLog.filters.action") }}</span>
          <Input
            v-model="action"
            class="w-40"
            placeholder="login_success"
            @change="applyAction"
          />
        </div>

        <div class="flex flex-col gap-1">
          <span class="text-xs text-muted-foreground">{{ t("settings.auditLog.filters.source") }}</span>
          <Select :model-value="source || 'all'" @update:model-value="applySource">
            <SelectTrigger class="w-36">
              <SelectValue :placeholder="t('settings.auditLog.filters.any')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ t("settings.auditLog.filters.any") }}</SelectItem>
              <SelectItem v-for="s in sourceOptions" :key="s" :value="s">{{ s }}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="flex flex-col gap-1">
          <span class="text-xs text-muted-foreground">{{ t("settings.auditLog.filters.success") }}</span>
          <Select :model-value="success || 'all'" @update:model-value="applySuccess">
            <SelectTrigger class="w-40">
              <SelectValue :placeholder="t('settings.auditLog.filters.any')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ t("settings.auditLog.filters.any") }}</SelectItem>
              <SelectItem value="true">{{ t("settings.auditLog.filters.successOnly") }}</SelectItem>
              <SelectItem value="false">{{ t("settings.auditLog.filters.failureOnly") }}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="flex flex-col gap-1">
          <span class="text-xs text-muted-foreground">{{ t("settings.auditLog.filters.dateFrom") }}</span>
          <Input v-model="dateFrom" type="date" class="w-40" @change="applyDateRange" />
        </div>

        <div class="flex flex-col gap-1">
          <span class="text-xs text-muted-foreground">{{ t("settings.auditLog.filters.dateTo") }}</span>
          <Input v-model="dateTo" type="date" class="w-40" @change="applyDateRange" />
        </div>

        <Button variant="outline" size="sm" @click="resetAllFilters">
          <RotateCcw class="h-4 w-4 mr-1" />
          {{ t("settings.auditLog.filters.reset") }}
        </Button>
      </div>

      <!-- Data table -->
      <DataTable
        :items="events"
        :columns="columns"
        :is-loading="loading"
        :empty-title="t('settings.auditLog.empty')"
        :empty-description="''"
        server-pagination
        :current-page="filters.page ?? 1"
        :page-size="filters.per_page ?? 50"
        :total-items="total"
        @page-change="onPageChange"
      >
        <template #cell-created_at="{ item }">
          <span class="text-xs text-muted-foreground">{{ formatDate(item.created_at) }}</span>
        </template>

        <template #cell-category="{ item }">
          <Badge
            variant="outline"
            :class="categoryVariant[item.category] || categoryVariant.system"
          >
            {{ item.category }}
          </Badge>
        </template>

        <template #cell-action="{ item }">
          <span class="font-mono text-xs">{{ item.action }}</span>
        </template>

        <template #cell-actor_email="{ item }">
          <div class="flex flex-col">
            <span class="text-sm">{{ item.actor_email || t("settings.auditLog.source.system") }}</span>
            <span v-if="item.actor_role" class="text-xs text-muted-foreground">{{ item.actor_role }}</span>
          </div>
        </template>

        <template #cell-target="{ item }">
          <span class="text-xs font-mono">{{ formatTarget(item) }}</span>
        </template>

        <template #cell-outcome="{ item }">
          <CheckCircle2 v-if="item.success" class="h-4 w-4 text-green-600 inline" />
          <XCircle v-else class="h-4 w-4 text-red-600 inline" />
        </template>

        <template #cell-details="{ item }">
          <details v-if="item.details && Object.keys(item.details).length > 0">
            <summary class="cursor-pointer text-xs text-primary hover:underline">
              {{ t("settings.auditLog.columns.details") }}
            </summary>
            <pre class="mt-1 p-2 text-xs bg-muted rounded overflow-auto max-h-48">{{ JSON.stringify(item.details, null, 2) }}</pre>
          </details>
          <span v-else class="text-xs text-muted-foreground">—</span>
        </template>
      </DataTable>
    </div>
  </div>
</template>
