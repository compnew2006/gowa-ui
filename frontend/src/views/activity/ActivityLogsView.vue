<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  activityLogsService,
  type ActivityLog as ActivityLogAPI,
  type ActivityLog,
  type ActivityLogListParams,
  type CreateActivityLogPayload,
} from "@/services/api";
import {
  ActivityLogNarrator,
  type ActivityNarrative,
} from "./activity-log-narrator";
import { ActivityLogContactResolver } from "./activity-log-contact-resolver";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  CrudFormDialog,
  DataTable,
  PageHeader,
  type Column,
} from "@/components/shared";
import { ClipboardList, Plus, RotateCcw, RefreshCcw } from "lucide-vue-next";
import { toast } from "vue-sonner";
import { getErrorMessage } from "@/lib/api-utils";

const { t } = useI18n();

interface ActivityFilters {
  category: string;
  eventType: string;
  source: string;
  status: string;
  startDate: string;
  endDate: string;
}

interface CustomEventForm {
  eventType: string;
  action: string;
  contactID: string;
  messageID: string;
  metadataText: string;
}

const defaultFilters: ActivityFilters = {
  category: "all",
  eventType: "",
  source: "all",
  status: "all",
  startDate: "",
  endDate: "",
};

const defaultCustomEventForm: CustomEventForm = {
  eventType: "",
  action: "",
  contactID: "",
  messageID: "",
  metadataText: "{}",
};

const logs = ref<ActivityLog[]>([]);
const isLoading = ref(false);
const isSubmitting = ref(false);
const filters = ref<ActivityFilters>({ ...defaultFilters });
const isCreateDialogOpen = ref(false);
const customEventForm = ref<CustomEventForm>({ ...defaultCustomEventForm });

const currentPage = ref(1);
const totalItems = ref(0);
const pageSize = 50;
const narrator = new ActivityLogNarrator();
const contactResolver = new ActivityLogContactResolver();
const defaultNarrative: ActivityNarrative = {
  eventClass: "general",
  sentence: "Activity recorded",
};
const contactLabels = ref<Map<string, string>>(new Map());
let contactResolveRunID = 0;

const columns = computed<Column<ActivityLog>[]>(() => [
  {
    key: "timestamp",
    label: t("activityLogs.columns.timestamp"),
    sortable: true,
    sortKey: "created_at",
  },
  { key: "event_class", label: t("activityLogs.columns.eventClass") },
  { key: "activity", label: t("activityLogs.columns.activity") },
  { key: "status", label: t("activityLogs.columns.status") },
]);

const sortKey = ref("created_at");
const sortDirection = ref<"asc" | "desc">("desc");

function buildListParams(): ActivityLogListParams {
  const params: ActivityLogListParams = {
    page: currentPage.value,
    limit: pageSize,
  };

  if (filters.value.category !== "all")
    params.category = filters.value.category;
  if (filters.value.source !== "all") params.source = filters.value.source;
  if (filters.value.status !== "all") params.status = filters.value.status;

  const eventType = filters.value.eventType.trim();
  if (eventType) params.event_type = eventType;

  if (filters.value.startDate) params.start_date = filters.value.startDate;
  if (filters.value.endDate) params.end_date = filters.value.endDate;

  return params;
}

async function fetchLogs() {
  isLoading.value = true;
  try {
    const response = await activityLogsService.list(buildListParams());
    const data = (response.data as any).data || response.data;
    logs.value = data.logs || [];
    totalItems.value = data.total ?? logs.value.length;
    void resolveContactLabels(logs.value);
  } catch (error) {
    toast.error(getErrorMessage(error, t("activityLogs.failedLoad")));
  } finally {
    isLoading.value = false;
  }
}

async function resolveContactLabels(currentLogs: ActivityLogAPI[]) {
  const runID = ++contactResolveRunID;
  if (currentLogs.length === 0) {
    contactLabels.value = new Map();
    return;
  }
  try {
    const resolved = await contactResolver.resolve(currentLogs);
    if (runID !== contactResolveRunID) return;
    contactLabels.value = resolved;
  } catch {
    if (runID !== contactResolveRunID) return;
    contactLabels.value = new Map();
  }
}

function handlePageChange(page: number) {
  currentPage.value = page;
  fetchLogs();
}

function applyFilters() {
  currentPage.value = 1;
  fetchLogs();
}

function resetFilters() {
  filters.value = { ...defaultFilters };
  currentPage.value = 1;
  fetchLogs();
}

function openCreateDialog() {
  customEventForm.value = { ...defaultCustomEventForm };
  isCreateDialogOpen.value = true;
}

function parseMetadata(metadataText: string): Record<string, any> {
  const trimmed = metadataText.trim();
  if (!trimmed) return {};

  const parsed = JSON.parse(trimmed);
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(t("activityLogs.validation.metadataObject"));
  }
  return parsed;
}

async function createCustomEvent() {
  const eventType = customEventForm.value.eventType.trim();
  const action = customEventForm.value.action.trim();

  if (!eventType) {
    toast.error(t("activityLogs.validation.eventTypeRequired"));
    return;
  }
  if (!action) {
    toast.error(t("activityLogs.validation.actionRequired"));
    return;
  }

  let metadata: Record<string, any>;
  try {
    metadata = parseMetadata(customEventForm.value.metadataText);
  } catch (error) {
    toast.error(
      getErrorMessage(error, t("activityLogs.validation.metadataJson")),
    );
    return;
  }

  const payload: CreateActivityLogPayload = {
    category: "custom",
    event_type: eventType,
    action,
    metadata,
  };

  const contactID = customEventForm.value.contactID.trim();
  const messageID = customEventForm.value.messageID.trim();
  if (contactID) payload.contact_id = contactID;
  if (messageID) payload.message_id = messageID;

  isSubmitting.value = true;
  try {
    await activityLogsService.create(payload);
    toast.success(t("activityLogs.customEventCreated"));
    isCreateDialogOpen.value = false;
    await fetchLogs();
  } catch (error) {
    toast.error(getErrorMessage(error, t("activityLogs.failedCreate")));
  } finally {
    isSubmitting.value = false;
  }
}

function formatTimestamp(value: string): string {
  return new Date(value).toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

const narrativesByID = computed(() => {
  const narrativeMap = new Map<string, ActivityNarrative>();
  for (const log of logs.value) {
    narrativeMap.set(
      log.id,
      narrator.build(log, { contactLabels: contactLabels.value }),
    );
  }
  return narrativeMap;
});

function getNarrative(log: ActivityLog): ActivityNarrative {
  return narrativesByID.value.get(log.id) || defaultNarrative;
}

function getActivityDescription(log: ActivityLog): string {
  return getNarrative(log).sentence;
}

function getEventClassLabel(log: ActivityLog): string {
  return t(`activityLogs.eventClasses.${getNarrative(log).eventClass}`);
}

function getEventClassClass(log: ActivityLog): string {
  const eventClass = getNarrative(log).eventClass;
  if (eventClass === "user") return "border-emerald-600 text-emerald-600";
  if (eventClass === "auth") return "border-blue-600 text-blue-600";
  if (eventClass === "system") return "border-amber-600 text-amber-600";
  if (eventClass === "custom") return "border-violet-600 text-violet-600";
  return "";
}

function getStatusLabel(status: string): string {
  if (status === "success") return t("common.success");
  if (status === "failure") return t("common.failed");
  return status || "—";
}

function getStatusClass(status: string): string {
  if (status === "success") return "border-emerald-600 text-emerald-600";
  if (status === "failure") return "border-destructive text-destructive";
  return "";
}

onMounted(() => fetchLogs());
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="$t('activityLogs.title')"
      :subtitle="$t('activityLogs.subtitle')"
      :icon="ClipboardList"
      icon-gradient="bg-gradient-to-br from-cyan-500 to-sky-600 shadow-cyan-500/20"
    >
      <template #actions>
        <div class="flex items-center gap-2">
          <Button variant="outline" size="sm" @click="fetchLogs">
            <RefreshCcw class="h-4 w-4 mr-2" />
            {{ $t("activityLogs.refresh") }}
          </Button>
          <Button variant="outline" size="sm" @click="openCreateDialog">
            <Plus class="h-4 w-4 mr-2" />
            {{ $t("activityLogs.logCustomEvent") }}
          </Button>
        </div>
      </template>
    </PageHeader>

    <ScrollArea class="flex-1">
      <div class="p-6">
        <div class="max-w-7xl mx-auto space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>{{ $t("activityLogs.filtersTitle") }}</CardTitle>
              <CardDescription>{{
                $t("activityLogs.filtersDescription")
              }}</CardDescription>
            </CardHeader>
            <CardContent>
              <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <div class="space-y-2">
                  <Label>{{ $t("activityLogs.filters.category") }}</Label>
                  <Select v-model="filters.category">
                    <SelectTrigger>
                      <SelectValue :placeholder="$t('common.select')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">{{
                        $t("activityLogs.options.all")
                      }}</SelectItem>
                      <SelectItem value="auth">{{
                        $t("activityLogs.options.auth")
                      }}</SelectItem>
                      <SelectItem value="engagement">{{
                        $t("activityLogs.options.engagement")
                      }}</SelectItem>
                      <SelectItem value="system">{{
                        $t("activityLogs.options.system")
                      }}</SelectItem>
                      <SelectItem value="custom">{{
                        $t("activityLogs.options.custom")
                      }}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div class="space-y-2">
                  <Label>{{ $t("activityLogs.filters.eventType") }}</Label>
                  <Input
                    v-model="filters.eventType"
                    :placeholder="
                      $t('activityLogs.filters.eventTypePlaceholder')
                    "
                  />
                </div>

                <div class="space-y-2">
                  <Label>{{ $t("activityLogs.filters.source") }}</Label>
                  <Select v-model="filters.source">
                    <SelectTrigger>
                      <SelectValue :placeholder="$t('common.select')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">{{
                        $t("activityLogs.options.all")
                      }}</SelectItem>
                      <SelectItem value="auth">{{
                        $t("activityLogs.options.auth")
                      }}</SelectItem>
                      <SelectItem value="engagement">{{
                        $t("activityLogs.options.engagement")
                      }}</SelectItem>
                      <SelectItem value="system">{{
                        $t("activityLogs.options.system")
                      }}</SelectItem>
                      <SelectItem value="custom">{{
                        $t("activityLogs.options.custom")
                      }}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div class="space-y-2">
                  <Label>{{ $t("activityLogs.filters.status") }}</Label>
                  <Select v-model="filters.status">
                    <SelectTrigger>
                      <SelectValue :placeholder="$t('common.select')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">{{
                        $t("activityLogs.options.all")
                      }}</SelectItem>
                      <SelectItem value="success">{{
                        $t("common.success")
                      }}</SelectItem>
                      <SelectItem value="failure">{{
                        $t("common.failed")
                      }}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div class="space-y-2">
                  <Label>{{ $t("activityLogs.filters.startDate") }}</Label>
                  <Input v-model="filters.startDate" type="date" />
                </div>

                <div class="space-y-2">
                  <Label>{{ $t("activityLogs.filters.endDate") }}</Label>
                  <Input v-model="filters.endDate" type="date" />
                </div>
              </div>

              <div class="mt-4 flex items-center gap-2">
                <Button variant="outline" size="sm" @click="applyFilters">
                  {{ $t("activityLogs.applyFilters") }}
                </Button>
                <Button variant="ghost" size="sm" @click="resetFilters">
                  <RotateCcw class="h-4 w-4 mr-2" />
                  {{ $t("activityLogs.resetFilters") }}
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{{ $t("activityLogs.historyTitle") }}</CardTitle>
              <CardDescription>{{
                $t("activityLogs.historyDescription")
              }}</CardDescription>
            </CardHeader>
            <CardContent>
              <DataTable
                :items="logs"
                :columns="columns"
                :is-loading="isLoading"
                :empty-icon="ClipboardList"
                :empty-title="$t('activityLogs.noLogsFound')"
                :empty-description="$t('activityLogs.noLogsFoundDesc')"
                v-model:sort-key="sortKey"
                v-model:sort-direction="sortDirection"
                server-pagination
                :current-page="currentPage"
                :total-items="totalItems"
                :page-size="pageSize"
                :item-name="$t('activityLogs.itemName')"
                @page-change="handlePageChange"
              >
                <template #cell-timestamp="{ item: log }">
                  <span class="whitespace-nowrap">{{
                    formatTimestamp(log.created_at)
                  }}</span>
                </template>

                <template #cell-event_class="{ item: log }">
                  <Badge
                    variant="outline"
                    :class="getEventClassClass(log)"
                    class="capitalize"
                  >
                    {{ getEventClassLabel(log) }}
                  </Badge>
                </template>

                <template #cell-activity="{ item: log }">
                  <span class="text-sm leading-6">{{
                    getActivityDescription(log)
                  }}</span>
                </template>

                <template #cell-status="{ item: log }">
                  <Badge variant="outline" :class="getStatusClass(log.status)">
                    {{ getStatusLabel(log.status) }}
                  </Badge>
                </template>
              </DataTable>
            </CardContent>
          </Card>
        </div>
      </div>
    </ScrollArea>

    <CrudFormDialog
      v-model:open="isCreateDialogOpen"
      :is-editing="false"
      :is-submitting="isSubmitting"
      :create-title="$t('activityLogs.customEventTitle')"
      :create-description="$t('activityLogs.customEventDescription')"
      :create-submit-label="$t('activityLogs.submitCustomEvent')"
      @submit="createCustomEvent"
    >
      <div class="space-y-4 py-2">
        <div class="space-y-2">
          <Label for="custom-event-type">{{
            $t("activityLogs.customForm.eventType")
          }}</Label>
          <Input
            id="custom-event-type"
            v-model="customEventForm.eventType"
            :placeholder="$t('activityLogs.customForm.eventTypePlaceholder')"
          />
        </div>

        <div class="space-y-2">
          <Label for="custom-action">{{
            $t("activityLogs.customForm.action")
          }}</Label>
          <Input
            id="custom-action"
            v-model="customEventForm.action"
            :placeholder="$t('activityLogs.customForm.actionPlaceholder')"
          />
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label for="custom-contact-id">{{
              $t("activityLogs.customForm.contactId")
            }}</Label>
            <Input
              id="custom-contact-id"
              v-model="customEventForm.contactID"
              :placeholder="$t('activityLogs.customForm.optionalUuid')"
            />
          </div>

          <div class="space-y-2">
            <Label for="custom-message-id">{{
              $t("activityLogs.customForm.messageId")
            }}</Label>
            <Input
              id="custom-message-id"
              v-model="customEventForm.messageID"
              :placeholder="$t('activityLogs.customForm.optionalUuid')"
            />
          </div>
        </div>

        <div class="space-y-2">
          <Label for="custom-metadata">{{
            $t("activityLogs.customForm.metadata")
          }}</Label>
          <Textarea
            id="custom-metadata"
            v-model="customEventForm.metadataText"
            :rows="5"
            :placeholder="$t('activityLogs.customForm.metadataPlaceholder')"
            class="font-mono text-xs"
          />
        </div>
      </div>
    </CrudFormDialog>
  </div>
</template>
