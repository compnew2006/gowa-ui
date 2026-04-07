<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { WhatsAppInstance } from "@/types/whatsmeow";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import type {
  InstanceTagColorKey,
  InstanceTagDisplayMode,
} from "@/lib/instance-tag";
import {
  cloneAutoRejectSettings,
  normalizeAutoRejectCallSettings,
  type AutoRejectCallSettings,
} from "@/lib/instance-auto-reject";
import {
  cloneAutoCampaignSettings,
  normalizeAutoCampaignSettings,
  type AutoCampaignSettings,
} from "@/lib/instance-auto-campaign";
import {
  cloneInstanceChatCloseRatingSettings,
  normalizeInstanceChatCloseRatingSettings,
  type InstanceChatCloseRatingSettings,
} from "@/lib/instance-chat-close-rating";
import {
  cloneInstanceAssignedChatResetSettings,
  normalizeInstanceAssignedChatResetSettings,
  type InstanceAssignedChatResetSettings,
} from "@/lib/instance-assigned-chat-reset";
import InstanceTagSettings from "@/components/whatsmeow/InstanceTagSettings.vue";
import AutoRejectSettingsPanel from "@/components/whatsmeow/AutoRejectSettingsPanel.vue";
import AutoCampaignSettingsPanel from "@/components/whatsmeow/AutoCampaignSettingsPanel.vue";
import InstanceChatCloseRatingPanel from "@/components/whatsmeow/InstanceChatCloseRatingPanel.vue";
import InstanceAssignedChatResetPanel from "@/components/whatsmeow/InstanceAssignedChatResetPanel.vue";
import {
  Loader2,
  Power,
  Trash2,
  Smartphone,
  QrCode,
  Pencil,
} from "lucide-vue-next";
import { toast } from "vue-sonner";

const { t } = useI18n();

const props = defineProps<{
  instance: WhatsAppInstance;
  paletteIndex?: number;
  tagSettingsSaving?: boolean;
  autoSyncSaving?: boolean;
  autoDownloadIncomingMediaSaving?: boolean;
  autoRejectSaving?: boolean;
  autoCampaignSaving?: boolean;
  autoCampaignUploading?: boolean;
  chatCloseRatingSaving?: boolean;
  assignedChatResetSaving?: boolean;
  organizationTimezone?: string;
}>();

const emit = defineEmits<{
  (e: "connect", id: string): void;
  (e: "disconnect", id: string): void;
  (e: "delete", id: string): void;
  (e: "edit", id: string): void;
  (
    e: "save-tag-settings",
    id: string,
    payload: {
      customLabel: string;
      color: InstanceTagColorKey;
      displayMode: InstanceTagDisplayMode;
    },
  ): void;
  (e: "update-auto-sync", id: string, enabled: boolean): void;
  (
    e: "update-auto-download-incoming-media",
    id: string,
    enabled: boolean,
  ): void;
  (
    e: "update-auto-reject-settings",
    id: string,
    payload: AutoRejectCallSettings,
  ): void;
  (
    e: "update-auto-campaign-settings",
    id: string,
    payload: AutoCampaignSettings,
  ): void;
  (e: "upload-auto-campaign-media", id: string, file: File): void;
  (e: "clear-auto-campaign-media", id: string): void;
  (
    e: "update-chat-close-rating-settings",
    id: string,
    payload: InstanceChatCloseRatingSettings,
  ): void;
  (
    e: "update-assigned-chat-reset-settings",
    id: string,
    payload: InstanceAssignedChatResetSettings,
  ): void;
}>();

const statusBadgeClass = computed(() => {
  switch (props.instance.status) {
    case "connected":
      return "border-primary/20 bg-primary/10 text-primary";
    case "connecting":
      return "border-primary/20 bg-primary/10 text-primary";
    case "disconnected":
      return "border-border bg-muted text-muted-foreground";
    case "banned":
      return "border-destructive/20 bg-destructive/10 text-destructive dark:text-red-300";
    case "logged_out":
      return "border-primary/15 bg-primary/5 text-primary";
    default:
      return "border-border bg-muted text-muted-foreground";
  }
});

const isConnected = computed(() => props.instance.status === "connected");
const isConnecting = computed(() => props.instance.status === "connecting");
const autoSyncEnabled = computed(() => {
  const setting = props.instance.settings?.auto_sync_history;
  return typeof setting === "boolean" ? setting : true;
});
const autoDownloadIncomingMediaEnabled = computed(() => {
  const setting = props.instance.settings?.auto_download_incoming_media;
  return typeof setting === "boolean" ? setting : false;
});

const autoRejectSettings = computed(() =>
  normalizeAutoRejectCallSettings(props.instance.settings?.auto_reject_calls),
);
const autoCampaignSettings = computed(() =>
  normalizeAutoCampaignSettings(props.instance.settings?.auto_campaign),
);
const autoRejectSchedule = computed(() => {
  const s = autoRejectSettings.value;
  if (!s.enabled) return t("common.off");

  switch (s.schedule.type) {
    case "while_in_other_calls":
      return t("instances.auto_reject.scheduleOtherCalls");
    case "custom_hours":
      return `${s.schedule.start} - ${s.schedule.end} (${s.schedule.timezone})`;
    default:
      return t("instances.auto_reject.scheduleAlways");
  }
});
const autoCampaignSummary = computed(() => {
  const s = autoCampaignSettings.value;
  if (!s.enabled) {
    return t("common.off");
  }

  return t("instances.auto_campaign.summary", {
    days: s.interval_days,
    status:
      s.target_status === "run"
        ? t("instances.auto_campaign.statusRun")
        : t("instances.auto_campaign.statusDraft"),
  });
});

const chatCloseRatingSettings = computed(() =>
  normalizeInstanceChatCloseRatingSettings(props.instance.settings),
);
const chatCloseRatingSummary = computed(() => {
  const s = chatCloseRatingSettings.value;
  if (!s.enabled) return t("common.off");
  return t("instances.chat_close_rating.summary", {
    minutes: s.followup_window_minutes,
  });
});
const assignedChatResetSettings = computed(() =>
  normalizeInstanceAssignedChatResetSettings(props.instance.settings),
);
const assignedChatResetSummary = computed(() => {
  const s = assignedChatResetSettings.value;
  if (!s.enabled) return t("common.off");

  const hourLabel = `${String(s.hour).padStart(2, "0")}:00`;
  if (s.mode === "custom_hour") {
    return t("instances.assigned_chat_reset.summaryCustomHour", {
      hour: hourLabel,
      timezone: props.organizationTimezone || "UTC",
    });
  }

  return t("instances.assigned_chat_reset.summaryMidnight", {
    timezone: props.organizationTimezone || "UTC",
  });
});

const sendBlockedNotice = computed(() => {
  const blockedUntilRaw = props.instance.send_blocked_until;
  if (!blockedUntilRaw) return "";

  const blockedUntil = new Date(blockedUntilRaw);
  if (Number.isNaN(blockedUntil.getTime())) return "";
  if (blockedUntil.getTime() <= Date.now()) return "";

  const reason = (
    props.instance.send_block_reason ||
    "Instance sending is temporarily blocked"
  ).trim();
  return `${reason} (${blockedUntil.toLocaleString()})`;
});

function formatUptime(totalSeconds?: number) {
  const seconds = totalSeconds || 0;
  if (seconds <= 0) {
    return "0m";
  }
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${minutes}m`;
}

function handleAutoRejectToggle(enabled: boolean) {
  const nextSettings = {
    ...cloneAutoRejectSettings(autoRejectSettings.value),
    enabled,
  };

  if (
    nextSettings.enabled &&
    nextSettings.mode === "with_message" &&
    !nextSettings.message.trim()
  ) {
    toast.error(t("instances.auto_reject.validation.messageRequired"));
    return;
  }

  emit("update-auto-reject-settings", props.instance.id, nextSettings);
}

function handleAutoCampaignToggle(enabled: boolean) {
  const nextSettings = {
    ...cloneAutoCampaignSettings(autoCampaignSettings.value),
    enabled,
  };

  if (nextSettings.enabled && !nextSettings.message.trim()) {
    toast.error(t("instances.auto_campaign.validation.messageRequired"));
    return;
  }

  emit("update-auto-campaign-settings", props.instance.id, nextSettings);
}
</script>

<template>
  <Card
    data-testid="instance-card"
    class="instance-card flex h-full flex-col overflow-hidden"
  >
    <CardHeader class="space-y-5 border-b border-border/70 pb-5">
      <div class="flex items-start justify-between gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <Badge variant="outline" :class="statusBadgeClass">
            {{ $t(`instances.status.${instance.status}`) }}
          </Badge>
          <Badge
            v-if="instance.is_default"
            variant="outline"
            class="border-primary/20 bg-primary/10 text-primary"
          >
            {{ $t("instances.card.default") }}
          </Badge>
        </div>

        <div class="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon-sm"
            class="text-muted-foreground hover:text-foreground"
            :aria-label="$t('instances.card.editAria')"
            @click="emit('edit', instance.id)"
          >
            <Pencil class="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            class="text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
            :aria-label="$t('instances.card.deleteAria')"
            @click="emit('delete', instance.id)"
          >
            <Trash2 class="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div class="space-y-1.5">
        <CardTitle class="text-lg font-semibold text-foreground">
          {{ instance.name }}
        </CardTitle>
        <CardDescription>
          {{ instance.phone_number || $t("instances.status.no_phone") }}
        </CardDescription>
      </div>

      <div class="flex items-center gap-2 text-sm text-muted-foreground">
        <Smartphone class="h-4 w-4 shrink-0" />
        <span class="min-w-0 truncate">{{
          instance.jid || $t("instances.status.not_paired")
        }}</span>
      </div>

      <div
        v-if="sendBlockedNotice"
        class="rounded-xl border border-destructive/20 bg-destructive/5 px-3 py-2 text-xs text-destructive"
      >
        {{ sendBlockedNotice }}
      </div>

      <div v-if="instance.health" class="grid grid-cols-2 gap-3 text-xs">
        <div class="rounded-xl border border-border/70 bg-background/70 p-3">
          <div class="text-[11px] text-muted-foreground">
            {{ $t("instances.card.uptime") }}
          </div>
          <div class="mt-1 font-medium text-foreground">
            {{ formatUptime(instance.health.uptime_seconds) }}
          </div>
        </div>
        <div class="rounded-xl border border-border/70 bg-background/70 p-3">
          <div class="text-[11px] text-muted-foreground">
            {{ $t("instances.card.queue") }}
          </div>
          <div class="mt-1 font-medium text-foreground">
            {{ instance.health.queue_depth }}
          </div>
        </div>
        <div class="rounded-xl border border-border/70 bg-background/70 p-3">
          <div class="text-[11px] text-muted-foreground">
            {{ $t("instances.card.sentReceived") }}
          </div>
          <div class="mt-1 font-medium text-foreground">
            {{ instance.health.messages_sent_today }} /
            {{ instance.health.messages_received_today }}
          </div>
        </div>
        <div class="rounded-xl border border-border/70 bg-background/70 p-3">
          <div class="text-[11px] text-muted-foreground">
            {{ $t("instances.card.errorRate") }}
          </div>
          <div class="mt-1 font-medium text-foreground">
            {{ instance.health.error_rate_percent.toFixed(1) }}%
          </div>
        </div>
      </div>
    </CardHeader>

    <CardContent class="flex flex-1 flex-col gap-5 pt-5">
      <div class="instance-card-settings-grid grid grid-cols-1 gap-3">
        <div class="rounded-xl border border-border/70 bg-background/70 p-3">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <p class="text-xs font-medium text-foreground">
                {{ $t("instances.card.autoSync") }}
              </p>
              <p class="mt-1 text-[11px] text-muted-foreground line-clamp-2">
                {{ $t("instances.card.autoSyncDesc") }}
              </p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <Loader2
                v-if="autoSyncSaving"
                class="h-3.5 w-3.5 animate-spin text-muted-foreground"
              />
              <Switch
                :checked="autoSyncEnabled"
                :disabled="autoSyncSaving"
                @update:checked="
                  (enabled) => emit('update-auto-sync', instance.id, enabled)
                "
              />
            </div>
          </div>
        </div>

        <div class="rounded-xl border border-border/70 bg-background/70 p-3">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <p class="text-xs font-medium text-foreground">
                {{ $t("instances.card.autoDownloadIncomingMedia") }}
              </p>
              <p class="mt-1 text-[11px] text-muted-foreground line-clamp-2">
                {{ $t("instances.card.autoDownloadIncomingMediaDesc") }}
              </p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <Loader2
                v-if="autoDownloadIncomingMediaSaving"
                class="h-3.5 w-3.5 animate-spin text-muted-foreground"
              />
              <Switch
                :checked="autoDownloadIncomingMediaEnabled"
                :disabled="autoDownloadIncomingMediaSaving"
                @update:checked="
                  (enabled) =>
                    emit(
                      'update-auto-download-incoming-media',
                      instance.id,
                      enabled,
                    )
                "
              />
            </div>
          </div>
        </div>

        <div class="rounded-xl border border-border/70 bg-background/70 p-3">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <p class="text-xs font-medium text-foreground">
                  {{ $t("instances.card.callAutoReject") }}
                </p>
                <Badge
                  v-if="autoRejectSettings.enabled"
                  variant="outline"
                  class="border-primary/20 bg-primary/10 text-[10px] text-primary"
                >
                  {{ $t("common.on") }}
                </Badge>
              </div>
              <p class="mt-1 text-[11px] text-muted-foreground line-clamp-2">
                {{ autoRejectSchedule }}
              </p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <Loader2
                v-if="autoRejectSaving"
                class="h-3.5 w-3.5 animate-spin text-muted-foreground"
              />
              <Switch
                data-testid="instance-auto-reject-toggle"
                :checked="autoRejectSettings.enabled"
                :disabled="autoRejectSaving"
                @update:checked="handleAutoRejectToggle"
              />
            </div>
          </div>

          <div class="mt-3">
            <AutoRejectSettingsPanel
              :settings="autoRejectSettings"
              :saving="autoRejectSaving || false"
              @save="
                (payload) =>
                  emit('update-auto-reject-settings', instance.id, payload)
              "
            />
          </div>
        </div>

        <div class="rounded-xl border border-border/70 bg-background/70 p-3">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <p class="text-xs font-medium text-foreground">
                  {{ $t("instances.card.autoCampaign") }}
                </p>
                <Badge
                  v-if="autoCampaignSettings.enabled"
                  variant="outline"
                  class="border-primary/20 bg-primary/10 text-[10px] text-primary"
                >
                  {{ $t("common.on") }}
                </Badge>
              </div>
              <p class="mt-1 text-[11px] text-muted-foreground line-clamp-2">
                {{ autoCampaignSummary }}
              </p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <Loader2
                v-if="autoCampaignSaving"
                class="h-3.5 w-3.5 animate-spin text-muted-foreground"
              />
              <Switch
                data-testid="instance-auto-campaign-toggle"
                :checked="autoCampaignSettings.enabled"
                :disabled="autoCampaignSaving"
                @update:checked="handleAutoCampaignToggle"
              />
            </div>
          </div>

          <div class="mt-3">
            <AutoCampaignSettingsPanel
              :settings="autoCampaignSettings"
              :saving="autoCampaignSaving || false"
              :uploading="autoCampaignUploading || false"
              @save="
                (payload) =>
                  emit('update-auto-campaign-settings', instance.id, payload)
              "
              @upload-media="
                (file) => emit('upload-auto-campaign-media', instance.id, file)
              "
              @clear-media="
                () => emit('clear-auto-campaign-media', instance.id)
              "
            />
          </div>
        </div>

        <div class="rounded-xl border border-border/70 bg-background/70 p-3">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <p class="text-xs font-medium text-foreground">
                  {{ $t("instances.chat_close_rating.title") }}
                </p>
                <Badge
                  v-if="chatCloseRatingSettings.enabled"
                  variant="outline"
                  class="border-primary/20 bg-primary/10 text-[10px] text-primary"
                >
                  {{ $t("common.on") }}
                </Badge>
              </div>
              <p class="mt-1 text-[11px] text-muted-foreground line-clamp-2">
                {{ chatCloseRatingSummary }}
              </p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <Loader2
                v-if="chatCloseRatingSaving"
                class="h-3.5 w-3.5 animate-spin text-muted-foreground"
              />
              <Switch
                :checked="chatCloseRatingSettings.enabled"
                :disabled="chatCloseRatingSaving"
                @update:checked="
                  (enabled) =>
                    emit('update-chat-close-rating-settings', instance.id, {
                      ...cloneInstanceChatCloseRatingSettings(
                        chatCloseRatingSettings,
                      ),
                      enabled,
                    })
                "
              />
            </div>
          </div>

          <div class="mt-3">
            <InstanceChatCloseRatingPanel
              :settings="chatCloseRatingSettings"
              :saving="chatCloseRatingSaving || false"
              @save="
                (payload) =>
                  emit(
                    'update-chat-close-rating-settings',
                    instance.id,
                    payload,
                  )
              "
            />
          </div>
        </div>

        <div class="rounded-xl border border-border/70 bg-background/70 p-3">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <p class="text-xs font-medium text-foreground">
                  {{ $t("instances.assigned_chat_reset.title") }}
                </p>
                <Badge
                  v-if="assignedChatResetSettings.enabled"
                  variant="outline"
                  class="border-primary/20 bg-primary/10 text-[10px] text-primary"
                >
                  {{ $t("common.on") }}
                </Badge>
              </div>
              <p class="mt-1 text-[11px] text-muted-foreground line-clamp-2">
                {{ assignedChatResetSummary }}
              </p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <Loader2
                v-if="assignedChatResetSaving"
                class="h-3.5 w-3.5 animate-spin text-muted-foreground"
              />
              <Switch
                :checked="assignedChatResetSettings.enabled"
                :disabled="assignedChatResetSaving"
                @update:checked="
                  (enabled) =>
                    emit('update-assigned-chat-reset-settings', instance.id, {
                      ...cloneInstanceAssignedChatResetSettings(
                        assignedChatResetSettings,
                      ),
                      enabled,
                    })
                "
              />
            </div>
          </div>

          <div class="mt-3">
            <InstanceAssignedChatResetPanel
              :settings="assignedChatResetSettings"
              :saving="assignedChatResetSaving || false"
              :organization-timezone="organizationTimezone"
              @save="
                (payload) =>
                  emit(
                    'update-assigned-chat-reset-settings',
                    instance.id,
                    payload,
                  )
              "
            />
          </div>
        </div>
      </div>

      <InstanceTagSettings
        :instance="instance"
        :palette-index="paletteIndex || 0"
        :saving="tagSettingsSaving || false"
        @save="(payload) => emit('save-tag-settings', instance.id, payload)"
      />
    </CardContent>

    <CardFooter class="border-t border-border/70 bg-muted/20 pt-5">
      <Button
        v-if="!isConnected && !isConnecting"
        class="w-full"
        @click="emit('connect', instance.id)"
      >
        <QrCode class="h-4 w-4 mr-2" />
        {{ $t("instances.card.connectScan") }}
      </Button>
      <Button
        v-else-if="isConnected"
        variant="destructive"
        class="w-full"
        @click="emit('disconnect', instance.id)"
      >
        <Power class="h-4 w-4 mr-2" />
        {{ $t("instances.card.disconnect") }}
      </Button>
      <div
        v-else
        class="flex w-full items-center justify-center gap-2 rounded-full border border-primary/20 bg-primary/10 px-4 py-2 text-sm font-medium text-primary"
      >
        <Loader2 class="h-4 w-4 animate-spin" />
        <span>{{ $t("instances.card.connecting") }}</span>
      </div>
    </CardFooter>
  </Card>
</template>

<style scoped>
.instance-card {
  container-type: inline-size;
}

@container (min-width: 25rem) {
  .instance-card-settings-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
