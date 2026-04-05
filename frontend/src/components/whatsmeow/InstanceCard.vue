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

const statusColor = computed(() => {
  switch (props.instance.status) {
    case "connected":
      return "bg-green-500";
    case "connecting":
      return "bg-yellow-500";
    case "disconnected":
      return "bg-gray-500";
    case "banned":
      return "bg-red-500";
    case "logged_out":
      return "bg-orange-500";
    default:
      return "bg-gray-500";
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
</script>

<template>
  <Card
    class="bg-white/[0.04] border-white/[0.08] light:bg-white light:border-gray-200"
  >
    <CardHeader class="pb-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center space-x-2">
          <Badge :class="[statusColor, 'text-white border-0']">{{
            $t(`instances.status.${instance.status}`)
          }}</Badge>
          <div
            v-if="instance.is_default"
            class="text-xs text-emerald-400 font-medium border border-emerald-400/20 bg-emerald-400/10 px-2 py-0.5 rounded"
          >
            {{ $t("instances.card.default") }}
          </div>
        </div>
        <div class="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            class="text-white/40 hover:text-emerald-400 hover:bg-emerald-400/10 light:text-gray-500 light:hover:text-emerald-600 light:hover:bg-emerald-50"
            :aria-label="$t('instances.card.editAria')"
            @click="emit('edit', instance.id)"
          >
            <Pencil class="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            class="text-white/40 hover:text-red-400 hover:bg-red-400/10 light:text-gray-500 light:hover:text-red-600 light:hover:bg-red-50"
            :aria-label="$t('instances.card.deleteAria')"
            @click="emit('delete', instance.id)"
          >
            <Trash2 class="h-4 w-4" />
          </Button>
        </div>
      </div>
      <CardTitle
        class="text-lg font-semibold text-white mt-2 light:text-gray-900"
        >{{ instance.name }}</CardTitle
      >
      <CardDescription class="text-white/40 light:text-gray-500">
        {{ instance.phone_number || $t("instances.status.no_phone") }}
      </CardDescription>
    </CardHeader>
    <CardContent>
      <div class="text-sm text-white/60 light:text-gray-600 space-y-3">
        <div class="flex items-center">
          <Smartphone class="h-4 w-4 mr-2 opacity-70" />
          <span>{{ instance.jid || $t("instances.status.not_paired") }}</span>
        </div>
        <div
          v-if="sendBlockedNotice"
          class="rounded-md border border-red-400/30 bg-red-500/10 px-2 py-1 text-xs text-red-200 light:border-red-200 light:bg-red-50 light:text-red-700"
        >
          {{ sendBlockedNotice }}
        </div>
        <div v-if="instance.health" class="grid grid-cols-2 gap-2 text-xs">
          <div
            class="rounded-md bg-white/[0.03] border border-white/[0.06] p-2 light:bg-gray-50 light:border-gray-200"
          >
            <div class="text-white/40 light:text-gray-500">
              {{ $t("instances.card.uptime") }}
            </div>
            <div class="font-medium text-white light:text-gray-900">
              {{ formatUptime(instance.health.uptime_seconds) }}
            </div>
          </div>
          <div
            class="rounded-md bg-white/[0.03] border border-white/[0.06] p-2 light:bg-gray-50 light:border-gray-200"
          >
            <div class="text-white/40 light:text-gray-500">
              {{ $t("instances.card.queue") }}
            </div>
            <div class="font-medium text-white light:text-gray-900">
              {{ instance.health.queue_depth }}
            </div>
          </div>
          <div
            class="rounded-md bg-white/[0.03] border border-white/[0.06] p-2 light:bg-gray-50 light:border-gray-200"
          >
            <div class="text-white/40 light:text-gray-500">
              {{ $t("instances.card.sentReceived") }}
            </div>
            <div class="font-medium text-white light:text-gray-900">
              {{ instance.health.messages_sent_today }} /
              {{ instance.health.messages_received_today }}
            </div>
          </div>
          <div
            class="rounded-md bg-white/[0.03] border border-white/[0.06] p-2 light:bg-gray-50 light:border-gray-200"
          >
            <div class="text-white/40 light:text-gray-500">
              {{ $t("instances.card.errorRate") }}
            </div>
            <div class="font-medium text-white light:text-gray-900">
              {{ instance.health.error_rate_percent.toFixed(1) }}%
            </div>
          </div>
        </div>
        <div class="grid grid-cols-1 gap-3 xl:grid-cols-2">
          <div
            class="rounded-md bg-white/[0.03] border border-white/[0.06] p-2 light:bg-gray-50 light:border-gray-200"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <p class="text-xs font-medium text-white light:text-gray-900">
                  {{ $t("instances.card.autoSync") }}
                </p>
                <p
                  class="text-[11px] text-white/45 light:text-gray-500 line-clamp-2"
                >
                  {{ $t("instances.card.autoSyncDesc") }}
                </p>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <Loader2
                  v-if="autoSyncSaving"
                  class="h-3.5 w-3.5 animate-spin text-white/50 light:text-gray-500"
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
          <div
            class="rounded-md bg-white/[0.03] border border-white/[0.06] p-2 light:bg-gray-50 light:border-gray-200"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <p class="text-xs font-medium text-white light:text-gray-900">
                  {{ $t("instances.card.autoDownloadIncomingMedia") }}
                </p>
                <p
                  class="text-[11px] text-white/45 light:text-gray-500 line-clamp-2"
                >
                  {{ $t("instances.card.autoDownloadIncomingMediaDesc") }}
                </p>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <Loader2
                  v-if="autoDownloadIncomingMediaSaving"
                  class="h-3.5 w-3.5 animate-spin text-white/50 light:text-gray-500"
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
          <div
            class="rounded-md bg-white/[0.03] border border-white/[0.06] p-2 space-y-2 light:bg-gray-50 light:border-gray-200"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <p class="text-xs font-medium text-white light:text-gray-900">
                    {{ $t("instances.card.callAutoReject") }}
                  </p>
                  <Badge
                    v-if="autoRejectSettings.enabled"
                    variant="default"
                    class="text-[10px] px-1.5 py-0"
                    >{{ $t("common.on") }}</Badge
                  >
                </div>
                <p
                  class="text-[11px] text-white/45 light:text-gray-500 line-clamp-2"
                >
                  {{ autoRejectSchedule }}
                </p>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <Loader2
                  v-if="autoRejectSaving"
                  class="h-3.5 w-3.5 animate-spin text-white/50 light:text-gray-500"
                />
                <Switch
                  :checked="autoRejectSettings.enabled"
                  :disabled="autoRejectSaving"
                  @update:checked="
                    (enabled) =>
                      emit('update-auto-reject-settings', instance.id, {
                        ...cloneAutoRejectSettings(autoRejectSettings),
                        enabled,
                      })
                  "
                />
              </div>
            </div>

            <AutoRejectSettingsPanel
              :settings="autoRejectSettings"
              :saving="autoRejectSaving || false"
              @save="
                (payload) =>
                  emit('update-auto-reject-settings', instance.id, payload)
              "
            />
          </div>
          <div
            class="rounded-md bg-white/[0.03] border border-white/[0.06] p-2 space-y-2 light:bg-gray-50 light:border-gray-200"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <p class="text-xs font-medium text-white light:text-gray-900">
                    {{ $t("instances.card.autoCampaign") }}
                  </p>
                  <Badge
                    v-if="autoCampaignSettings.enabled"
                    variant="default"
                    class="text-[10px] px-1.5 py-0"
                    >{{ $t("common.on") }}</Badge
                  >
                </div>
                <p
                  class="text-[11px] text-white/45 light:text-gray-500 line-clamp-2"
                >
                  {{ autoCampaignSummary }}
                </p>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <Loader2
                  v-if="autoCampaignSaving"
                  class="h-3.5 w-3.5 animate-spin text-white/50 light:text-gray-500"
                />
                <Switch
                  :checked="autoCampaignSettings.enabled"
                  :disabled="autoCampaignSaving"
                  @update:checked="
                    (enabled) =>
                      emit('update-auto-campaign-settings', instance.id, {
                        ...cloneAutoCampaignSettings(autoCampaignSettings),
                        enabled,
                      })
                  "
                />
              </div>
            </div>

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
          <div
            class="rounded-md bg-white/[0.03] border border-white/[0.06] p-2 space-y-2 light:bg-gray-50 light:border-gray-200"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <p class="text-xs font-medium text-white light:text-gray-900">
                    {{ $t("instances.chat_close_rating.title") }}
                  </p>
                  <Badge
                    v-if="chatCloseRatingSettings.enabled"
                    variant="default"
                    class="text-[10px] px-1.5 py-0"
                    >{{ $t("common.on") }}</Badge
                  >
                </div>
                <p
                  class="text-[11px] text-white/45 light:text-gray-500 line-clamp-2"
                >
                  {{ chatCloseRatingSummary }}
                </p>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <Loader2
                  v-if="chatCloseRatingSaving"
                  class="h-3.5 w-3.5 animate-spin text-white/50 light:text-gray-500"
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
          <div
            class="rounded-md bg-white/[0.03] border border-white/[0.06] p-2 space-y-2 light:bg-gray-50 light:border-gray-200"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <p class="text-xs font-medium text-white light:text-gray-900">
                    {{ $t("instances.assigned_chat_reset.title") }}
                  </p>
                  <Badge
                    v-if="assignedChatResetSettings.enabled"
                    variant="default"
                    class="text-[10px] px-1.5 py-0"
                  >
                    {{ $t("common.on") }}
                  </Badge>
                </div>
                <p
                  class="text-[11px] text-white/45 light:text-gray-500 line-clamp-2"
                >
                  {{ assignedChatResetSummary }}
                </p>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <Loader2
                  v-if="assignedChatResetSaving"
                  class="h-3.5 w-3.5 animate-spin text-white/50 light:text-gray-500"
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
    <CardFooter class="pt-2">
      <Button
        v-if="!isConnected && !isConnecting"
        class="w-full bg-emerald-600 hover:bg-emerald-700 text-white"
        @click="emit('connect', instance.id)"
      >
        <QrCode class="h-4 w-4 mr-2" />
        {{ $t("instances.card.connectScan") }}
      </Button>
      <Button
        v-else-if="isConnected"
        variant="destructive"
        class="w-full bg-red-500/10 text-red-400 hover:bg-red-500/20 border border-red-500/50"
        @click="emit('disconnect', instance.id)"
      >
        <Power class="h-4 w-4 mr-2" />
        {{ $t("instances.card.disconnect") }}
      </Button>
      <div
        v-else
        class="w-full text-center text-sm text-yellow-400 animate-pulse"
      >
        {{ $t("instances.card.connecting") }}
      </div>
    </CardFooter>
  </Card>
</template>
