<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue-sonner";
import { getErrorMessage } from "@/lib/api-utils";
import { Switch } from "@/components/ui/switch";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Loader2, Trash2, ChevronDown, ChevronRight } from "lucide-vue-next";
import {
  useInstanceUploadsCleanup,
  useUpdateInstanceUploadsCleanup,
  useInstanceUploadsCleanupHistory,
  useRunInstanceUploadsCleanup,
} from "@/composables/usePerInstanceUploadsCleanup";

const props = defineProps<{
  instanceId: string;
}>();

const { t } = useI18n();
const instanceId = computed(() => props.instanceId);

const { data: settings, isLoading } = useInstanceUploadsCleanup(instanceId);
const updateMutation = useUpdateInstanceUploadsCleanup(instanceId);
const { data: history } = useInstanceUploadsCleanupHistory(instanceId);
const runMutation = useRunInstanceUploadsCleanup(instanceId);

const inherit = ref(true);
const retentionDays = ref<number>(30);
const reason = ref("");

watch(
  () => settings.value,
  (s) => {
    if (s) {
      inherit.value = s.inherit;
      if (s.retention_days != null) {
        retentionDays.value = s.retention_days;
      }
    }
  },
  { immediate: true },
);

const effectiveLabel = computed(() => {
  if (!settings.value) return "";
  const src = settings.value.effective_source;
  const days = settings.value.effective_retention_days;
  if (src === "disabled") return t("settings.uploadsCleanupInstanceOverviewEffectiveDisabled");
  if (src === "custom") return t("settings.uploadsCleanupInstanceOverviewEffectiveCustom", { days });
  return t("settings.uploadsCleanupInstanceOverviewEffectiveDefault");
});

function handleSave() {
  updateMutation.mutate(
    {
      inherit: inherit.value,
      ...(inherit.value ? {} : { retention_days: retentionDays.value }),
      ...(reason.value ? { reason: reason.value } : {}),
    },
    {
      onSuccess: () => {
        toast.success(t("common.saved"));
        reason.value = "";
      },
      onError: (err: unknown) => {
        toast.error(getErrorMessage(err, t("common.error")));
      },
    },
  );
}

function handleRun() {
  runMutation.mutate(undefined, {
    onSuccess: (result) => {
      toast.success(
        `Deleted ${result.deleted_files} file(s) using ${result.retention_used}-day retention`,
      );
    },
    onError: (err: unknown) => {
      toast.error(getErrorMessage(err, t("common.error")));
    },
  });
}

const isSaving = computed(() => updateMutation.isPending.value);
const isRunning = computed(() => runMutation.isPending.value);
const showHistory = ref(false);
</script>

<template>
  <div
    data-testid="per-instance-uploads-cleanup"
    class="rounded-xl border border-border/70 bg-background/70 p-3"
  >
    <!-- Header -->
    <div class="flex items-start justify-between gap-3">
      <div class="min-w-0">
        <p class="text-xs font-medium text-foreground">
          {{ t("settings.uploadsCleanupInstanceRetentionLabel") }}
        </p>
        <p class="mt-1 text-[11px] text-muted-foreground line-clamp-1">
          {{ t("settings.uploadsCleanupInstanceRetentionDesc") }}
        </p>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <Loader2
          v-if="isLoading"
          class="h-3.5 w-3.5 animate-spin text-muted-foreground"
        />
        <Badge
          variant="outline"
          class="border-primary/20 bg-primary/10 text-[10px] text-primary"
        >
          {{ effectiveLabel }}
        </Badge>
      </div>
    </div>

    <!-- Controls -->
    <div class="mt-2.5 space-y-2">
      <!-- Inherit toggle -->
      <div class="flex items-center justify-between gap-3">
        <label
          class="text-[11px] text-muted-foreground cursor-pointer"
          for="uploads-cleanup-inherit"
        >
          {{ t("settings.uploadsCleanupInstanceInheritLabel") }}
        </label>
        <Switch
          id="uploads-cleanup-inherit"
          data-testid="uploads-cleanup-inherit-toggle"
          :checked="inherit"
          :disabled="isSaving"
          @update:checked="inherit = $event"
        />
      </div>

      <!-- Retention input + actions row (when inherit OFF) -->
      <div v-if="!inherit" class="flex items-center gap-2">
        <Input
          id="uploads-cleanup-days"
          v-model.number="retentionDays"
          data-testid="uploads-cleanup-days-input"
          type="number"
          min="0"
          max="3650"
          class="h-7 w-20 text-xs"
          :disabled="isSaving"
        />
        <span class="text-[11px] text-muted-foreground">days</span>
        <div class="ml-auto flex gap-1.5">
          <Button
            variant="outline"
            size="sm"
            class="h-7 px-2 text-[11px]"
            data-testid="uploads-cleanup-run-btn"
            :disabled="isRunning"
            @click="handleRun"
          >
            <Loader2 v-if="isRunning" class="mr-1 h-3 w-3 animate-spin" />
            <Trash2 v-else class="mr-1 h-3 w-3" />
            {{ t("settings.uploadsCleanupInstanceRunNow") }}
          </Button>
          <Button
            size="sm"
            class="h-7 px-2 text-[11px]"
            data-testid="uploads-cleanup-save-btn"
            :disabled="isSaving"
            @click="handleSave"
          >
            <Loader2 v-if="isSaving" class="mr-1 h-3 w-3 animate-spin" />
            {{ t("common.save") }}
          </Button>
        </div>
      </div>

      <!-- When inheriting: effective preview + compact actions -->
      <div v-else class="flex items-center gap-2">
        <p
          v-if="settings?.effective_retention_days"
          class="text-[11px] text-muted-foreground"
        >
          {{ t("settings.uploadsCleanupInstanceEffectivePreview", { days: settings.effective_retention_days }) }}
        </p>
        <div class="ml-auto flex gap-1.5">
          <Button
            variant="outline"
            size="sm"
            class="h-7 px-2 text-[11px]"
            data-testid="uploads-cleanup-run-btn"
            :disabled="isRunning"
            @click="handleRun"
          >
            <Loader2 v-if="isRunning" class="mr-1 h-3 w-3 animate-spin" />
            <Trash2 v-else class="mr-1 h-3 w-3" />
            {{ t("settings.uploadsCleanupInstanceRunNow") }}
          </Button>
        </div>
      </div>

      <!-- Reason (only when custom + saving) -->
      <Input
        v-if="!inherit"
        v-model="reason"
        data-testid="uploads-cleanup-reason-input"
        class="h-7 text-xs"
        :disabled="isSaving"
        :placeholder="t('settings.uploadsCleanupInstanceHistoryReason')"
      />

      <!-- Collapsible history toggle -->
      <button
        v-if="history?.entries?.length"
        type="button"
        class="flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground transition-colors"
        @click="showHistory = !showHistory"
      >
        <component :is="showHistory ? ChevronDown : ChevronRight" class="h-3 w-3" />
        {{ t("settings.uploadsCleanupInstanceHistoryTitle") }} ({{ history.entries.length }})
      </button>

      <!-- History entries (collapsed by default) -->
      <div v-if="showHistory && history?.entries?.length" class="space-y-1">
        <div
          v-for="entry in history.entries"
          :key="entry.id"
          data-testid="uploads-cleanup-history-entry"
          class="flex items-center gap-2 rounded border border-border/40 bg-background/40 px-2 py-1 text-[11px]"
        >
          <span class="text-muted-foreground shrink-0">
            {{ new Date(entry.created_at).toLocaleDateString() }}
          </span>
          <span class="text-muted-foreground">
            {{ entry.actor_email || "System" }}
          </span>
          <span class="ml-auto text-foreground tabular-nums">
            <span v-if="entry.old_retention_days != null">{{ entry.old_retention_days }}d</span>
            <span v-if="entry.new_retention_days != null" class="ml-1">&rarr; {{ entry.new_retention_days }}d</span>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
