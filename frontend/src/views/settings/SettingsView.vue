<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { PageHeader } from "@/components/shared";
import LanguageSwitcher from "@/components/LanguageSwitcher.vue";
import { toast } from "vue-sonner";
import {
  Settings,
  Bell,
  Loader2,
  Globe,
  MessageSquare,
  Play,
  Archive,
  ImageIcon,
  LayoutGrid,
  Upload,
  CheckCircle2,
} from "lucide-vue-next";
import { usersService, organizationService } from "@/services/api";
import { useAuthStore } from "@/stores/auth";
import { useConfigStore } from "@/stores/config";
import {
  ChatSidebarUnifier,
  type ChatSidebarViewMode,
} from "@/lib/chat-sidebar-unifier";
import { getErrorMessage, unwrapResponse } from "@/lib/api-utils";
import {
  CHAT_BACKGROUND_PRESETS,
  CHAT_BACKGROUND_UPLOAD_ACCEPT,
  getChatBackgroundPreset,
  isSameChatBackgroundPreference,
  normalizeChatBackgroundPreference,
  resolveChatBackgroundAssetStyle,
  resolveChatBackgroundStyle,
  resolveChatBackgroundEditorMode,
  validateChatBackgroundFile,
  type ChatBackgroundEditorMode,
} from "@/lib/chat-backgrounds";
import type { ChatBackgroundSettings, UserSettings } from "@/types/auth";

const { t } = useI18n();
const authStore = useAuthStore();
const configStore = useConfigStore();

type NotificationSoundKey = "notification1" | "notification2" | "notification";
const DEFAULT_NOTIFICATION_SOUND: NotificationSoundKey = "notification1";
type AssignedChatResetMode = "midnight" | "custom_hour";

const DEFAULT_CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MINUTES = 15;
const CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MIN_MINUTES = 1;
const CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MAX_MINUTES = 1440;
const DEFAULT_CHAT_CLOSE_RATING_TEMPLATES: Record<string, string> = {
  en: "Hi {customer_name}, your chat {chat_id} with {agent_name} at {organization_name} is now closed. Please reply with a number from 1 to 10 to rate your experience.",
  ar: "مرحبًا {customer_name}، تم إغلاق المحادثة {chat_id} مع {agent_name} في {organization_name}. الرجاء الرد برقم من 1 إلى 10 لتقييم تجربتك.",
  es: "Hola {customer_name}, tu chat {chat_id} con {agent_name} en {organization_name} se ha cerrado. Responde con un numero del 1 al 10 para calificar tu experiencia.",
};

const isSubmitting = ref(false);
const isLoading = ref(true);
const isPreviewPlaying = ref(false);
let previewAudio: HTMLAudioElement | null = null;

function normalizeNotificationSound(value: unknown): NotificationSoundKey {
  if (value === "notification2") return "notification2";
  if (value === "notification") return "notification";
  return DEFAULT_NOTIFICATION_SOUND;
}

const rawBasePath =
  typeof window !== "undefined"
    ? ((window as any).__BASE_PATH__ ?? import.meta.env.BASE_URL ?? "/")
    : (import.meta.env.BASE_URL ?? "/");
const normalizedBasePath = String(rawBasePath).replace(/\/$/, "");

function getNotificationSoundSources(sound: NotificationSoundKey): string[] {
  return [
    normalizedBasePath ? `${normalizedBasePath}/${sound}.mp3` : `/${sound}.mp3`,
    `/${sound}.mp3`,
  ].filter((src, index, arr) => arr.indexOf(src) === index);
}

async function previewNotificationSound() {
  const selectedSound = normalizeNotificationSound(
    notificationSettings.value.notification_sound,
  );
  const sources = getNotificationSoundSources(selectedSound);

  if (previewAudio) {
    previewAudio.pause();
    previewAudio.currentTime = 0;
    previewAudio = null;
  }

  for (const src of sources) {
    const audio = new Audio(src);
    audio.volume = 0.5;
    audio.preload = "auto";

    try {
      await audio.play();
      previewAudio = audio;
      isPreviewPlaying.value = true;

      const resetState = () => {
        if (previewAudio === audio) {
          previewAudio = null;
          isPreviewPlaying.value = false;
        }
      };

      audio.addEventListener("ended", resetState, { once: true });
      audio.addEventListener("error", resetState, { once: true });
      return;
    } catch {
      // Try next source path
    }
  }

  isPreviewPlaying.value = false;
}

// General Settings
const generalSettings = ref({
  organization_name: "My Organization",
  default_timezone: "UTC",
  date_format: "YYYY-MM-DD",
  mask_phone_numbers: false,
});

interface NotificationSettings {
  email_notifications: boolean;
  new_message_alerts: boolean;
  campaign_updates: boolean;
  notification_sound: NotificationSoundKey;
}

// Notification Settings
const notificationSettings = ref<NotificationSettings>({
  email_notifications: true,
  new_message_alerts: true,
  campaign_updates: true,
  notification_sound: DEFAULT_NOTIFICATION_SOUND,
});

function normalizeAssignedChatResetMode(value: unknown): AssignedChatResetMode {
  return value === "custom_hour" ? "custom_hour" : "midnight";
}

function normalizeAssignedChatResetHour(value: unknown): number {
  const parsed =
    typeof value === "number"
      ? value
      : typeof value === "string"
        ? Number(value)
        : Number.NaN;
  if (!Number.isFinite(parsed)) return 0;
  const rounded = Math.trunc(parsed);
  return Math.min(23, Math.max(0, rounded));
}

function normalizeChatCloseRatingFollowupWindowMinutes(value: unknown): number {
  const parsed =
    typeof value === "number"
      ? value
      : typeof value === "string"
        ? Number(value)
        : Number.NaN;
  if (!Number.isFinite(parsed))
    return DEFAULT_CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MINUTES;
  const rounded = Math.trunc(parsed);
  return Math.min(
    CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MAX_MINUTES,
    Math.max(CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MIN_MINUTES, rounded),
  );
}

function normalizeChatCloseRatingTemplates(
  value: unknown,
): Record<string, string> {
  const normalized: Record<string, string> = {
    ...DEFAULT_CHAT_CLOSE_RATING_TEMPLATES,
  };

  if (!value || typeof value !== "object") {
    return normalized;
  }

  for (const [language, rawTemplate] of Object.entries(
    value as Record<string, unknown>,
  )) {
    const key = language.trim().toLowerCase();
    if (!key) continue;
    if (typeof rawTemplate !== "string") continue;
    const template = rawTemplate.trim();
    if (!template) continue;
    normalized[key] = template;
  }

  return normalized;
}

// Chat Preferences (localStorage-only)
const MEDIA_GROUP_WINDOW_KEY = "chat.mediaGroupWindowSeconds";
const chatSettings = ref({
  media_group_window: 60,
  sidebar_view_mode: ChatSidebarUnifier.readViewMode() as ChatSidebarViewMode,
  show_print_buttons: configStore.showPrintButtons,
  show_download_buttons: configStore.showDownloadButtons,
  assigned_chat_reset_enabled: true,
  assigned_chat_reset_mode: "midnight" as AssignedChatResetMode,
  assigned_chat_reset_hour: 0,
  chat_close_rating_enabled: true,

  chat_close_rating_followup_window_minutes:
    DEFAULT_CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MINUTES,
  chat_close_rating_templates: { ...DEFAULT_CHAT_CLOSE_RATING_TEMPLATES },
});
const showChatCloseRatingTemplates = ref(false);
const savedChatBackground = ref<ChatBackgroundSettings | null>(null);
const chatBackgroundEditorMode = ref<ChatBackgroundEditorMode>("default");
const selectedChatBackgroundPresetID = ref<string | null>(null);
const stagedChatBackgroundFile = ref<File | null>(null);
const stagedChatBackgroundPreviewURL = ref<string | null>(null);
const chatBackgroundErrorKey = ref<string | null>(null);
const chatBackgroundUsesDefault = ref(true);

const imageChatBackgroundPresets = computed(() =>
  CHAT_BACKGROUND_PRESETS.filter((preset) => preset.category === "image"),
);
const patternChatBackgroundPresets = computed(() =>
  CHAT_BACKGROUND_PRESETS.filter((preset) => preset.category === "pattern"),
);
const activeChatBackgroundPresetID = computed(() =>
  chatBackgroundUsesDefault.value || chatBackgroundEditorMode.value === "upload"
    ? null
    : selectedChatBackgroundPresetID.value,
);
const defaultChatBackgroundPreviewStyle = computed(() =>
  resolveChatBackgroundStyle(null, { variant: "preview" }),
);
const savedCustomChatBackgroundStyle = computed(() => {
  if (
    chatBackgroundUsesDefault.value ||
    savedChatBackground.value?.kind !== "custom"
  ) {
    return null;
  }
  return resolveChatBackgroundStyle(savedChatBackground.value, {
    variant: "preview",
  });
});
const stagedChatBackgroundStyle = computed(() => {
  if (!stagedChatBackgroundPreviewURL.value) {
    return null;
  }
  return resolveChatBackgroundAssetStyle(
    stagedChatBackgroundPreviewURL.value,
    "image",
    "light",
    "preview",
  );
});

function clearStagedChatBackgroundPreview() {
  if (stagedChatBackgroundPreviewURL.value) {
    URL.revokeObjectURL(stagedChatBackgroundPreviewURL.value);
  }
  stagedChatBackgroundPreviewURL.value = null;
}

function clearStagedChatBackgroundSelection() {
  stagedChatBackgroundFile.value = null;
  clearStagedChatBackgroundPreview();
}

function syncChatBackgroundState(value: unknown) {
  savedChatBackground.value = normalizeChatBackgroundPreference(value);
  chatBackgroundUsesDefault.value = savedChatBackground.value === null;
  chatBackgroundEditorMode.value = resolveChatBackgroundEditorMode(
    savedChatBackground.value,
  );
  selectedChatBackgroundPresetID.value =
    savedChatBackground.value?.kind === "preset"
      ? (savedChatBackground.value.preset_id ?? null)
      : null;
  chatBackgroundErrorKey.value = null;
  clearStagedChatBackgroundSelection();
}

function setChatBackgroundMode(value: string) {
  if (
    value !== "default" &&
    value !== "images" &&
    value !== "patterns" &&
    value !== "upload"
  ) {
    return;
  }

  if (value === "default") {
    selectDefaultChatBackground();
    return;
  }

  chatBackgroundEditorMode.value = value;
  chatBackgroundUsesDefault.value = false;
  chatBackgroundErrorKey.value = null;
}

function selectDefaultChatBackground() {
  chatBackgroundUsesDefault.value = true;
  chatBackgroundEditorMode.value = "default";
  selectedChatBackgroundPresetID.value = null;
  clearStagedChatBackgroundSelection();
  chatBackgroundErrorKey.value = null;
}

function selectChatBackgroundPreset(presetID: string) {
  chatBackgroundUsesDefault.value = false;
  selectedChatBackgroundPresetID.value = presetID;
  chatBackgroundErrorKey.value = null;
}

function handleChatBackgroundFileSelection(files: FileList | null) {
  const nextFile = files?.[0];
  if (!nextFile) {
    return;
  }

  const validation = validateChatBackgroundFile(nextFile);
  if (!validation.valid) {
    clearStagedChatBackgroundSelection();
    chatBackgroundErrorKey.value = validation.errorKey ?? null;
    return;
  }

  clearStagedChatBackgroundSelection();
  stagedChatBackgroundFile.value = nextFile;
  stagedChatBackgroundPreviewURL.value = URL.createObjectURL(nextFile);
  chatBackgroundUsesDefault.value = false;
  chatBackgroundErrorKey.value = null;
  chatBackgroundEditorMode.value = "upload";
}

function resolvePendingChatBackgroundSelection(): ChatBackgroundSettings | null {
  if (
    chatBackgroundUsesDefault.value ||
    chatBackgroundEditorMode.value === "default"
  ) {
    return null;
  }

  if (chatBackgroundEditorMode.value === "upload") {
    if (stagedChatBackgroundFile.value) {
      return { kind: "custom" };
    }
    return savedChatBackground.value?.kind === "custom"
      ? savedChatBackground.value
      : null;
  }

  const preset = getChatBackgroundPreset(selectedChatBackgroundPresetID.value);
  if (!preset) {
    return null;
  }
  if (
    (chatBackgroundEditorMode.value === "images" &&
      preset.category !== "image") ||
    (chatBackgroundEditorMode.value === "patterns" &&
      preset.category !== "pattern")
  ) {
    return null;
  }

  return {
    kind: "preset",
    preset_id: preset.id,
  };
}

const timezoneOptions = [
  { value: "UTC", label: "UTC (GMT+0)" },
  { value: "Etc/GMT-2", label: "UTC+2 (Fixed Offset)" },
  { value: "Africa/Cairo", label: "UTC+2 (Cairo)" },
  { value: "Europe/Athens", label: "UTC+2 (Athens)" },
  { value: "Etc/GMT-3", label: "UTC+3 (Fixed Offset)" },
  { value: "Asia/Riyadh", label: "UTC+3 (Riyadh)" },
  { value: "Europe/Moscow", label: "UTC+3 (Moscow)" },
  { value: "America/New_York", label: "UTC-5/-4 (Eastern Time)" },
  { value: "America/Chicago", label: "UTC-6/-5 (Central Time)" },
  { value: "America/Denver", label: "UTC-7/-6 (Mountain Time)" },
  { value: "America/Los_Angeles", label: "UTC-8/-7 (Pacific Time)" },
  { value: "Europe/London", label: "UTC+0/+1 (London)" },
  { value: "Europe/Paris", label: "UTC+1/+2 (Paris)" },
  { value: "Asia/Dubai", label: "UTC+4 (Dubai)" },
  { value: "Asia/Karachi", label: "UTC+5 (Karachi)" },
  { value: "Asia/Kolkata", label: "UTC+5:30 (India)" },
  { value: "Asia/Bangkok", label: "UTC+7 (Bangkok)" },
  { value: "Asia/Singapore", label: "UTC+8 (Singapore)" },
  { value: "Asia/Tokyo", label: "UTC+9 (Tokyo)" },
  { value: "Australia/Sydney", label: "UTC+10/+11 (Sydney)" },
];

const chatResetHourOptions = Array.from({ length: 24 }, (_, hour) => ({
  value: String(hour),
  label: `${String(hour).padStart(2, "0")}:00`,
}));

// Load chat settings from localStorage
try {
  const stored = Number(localStorage.getItem(MEDIA_GROUP_WINDOW_KEY));
  if (Number.isFinite(stored) && stored >= 5 && stored <= 300) {
    chatSettings.value.media_group_window = stored;
  }
} catch {
  // Ignore localStorage errors
}

onMounted(async () => {
  chatSettings.value.show_print_buttons = configStore.showPrintButtons;
  chatSettings.value.show_download_buttons = configStore.showDownloadButtons;
  try {
    const [orgResponse, userResponse] = await Promise.all([
      organizationService.getSettings(),
      usersService.me(),
    ]);

    // Organization settings
    const orgData = orgResponse.data.data || orgResponse.data;
    if (orgData) {
      generalSettings.value = {
        organization_name: orgData.name || "My Organization",
        default_timezone: orgData.settings?.timezone || "UTC",
        date_format: orgData.settings?.date_format || "YYYY-MM-DD",
        mask_phone_numbers: orgData.settings?.mask_phone_numbers || false,
      };

      const resetMode = normalizeAssignedChatResetMode(
        orgData.settings?.assigned_chat_reset_mode,
      );
      const resetHour = normalizeAssignedChatResetHour(
        orgData.settings?.assigned_chat_reset_hour,
      );
      chatSettings.value.assigned_chat_reset_enabled =
        orgData.settings?.assigned_chat_reset_enabled !== false;
      chatSettings.value.assigned_chat_reset_mode = resetMode;
      chatSettings.value.assigned_chat_reset_hour =
        resetMode === "midnight" ? 0 : resetHour;
      chatSettings.value.chat_close_rating_enabled =
        orgData.settings?.chat_close_rating_enabled !== false;

      chatSettings.value.chat_close_rating_followup_window_minutes =
        normalizeChatCloseRatingFollowupWindowMinutes(
          orgData.settings?.chat_close_rating_followup_window_minutes,
        );
      chatSettings.value.chat_close_rating_templates =
        normalizeChatCloseRatingTemplates(
          orgData.settings?.chat_close_rating_templates,
        );
    }

    // User notification settings
    const user = userResponse.data.data || userResponse.data;
    if (user.settings) {
      notificationSettings.value = {
        email_notifications: user.settings.email_notifications ?? true,
        new_message_alerts: user.settings.new_message_alerts ?? true,
        campaign_updates: user.settings.campaign_updates ?? true,
        notification_sound: normalizeNotificationSound(
          user.settings.notification_sound,
        ),
      };
    }
    syncChatBackgroundState(user.settings?.chat_background);
  } catch (error) {
    console.error("Failed to load settings:", error);
  } finally {
    isLoading.value = false;
  }
});

async function saveGeneralSettings() {
  isSubmitting.value = true;
  try {
    await organizationService.updateSettings({
      name: generalSettings.value.organization_name,
      timezone: generalSettings.value.default_timezone,
      date_format: generalSettings.value.date_format,
      mask_phone_numbers: generalSettings.value.mask_phone_numbers,
    });
    toast.success(t("settings.generalSaved"));
  } catch (error) {
    toast.error(t("common.failedSave", { resource: t("resources.settings") }));
  } finally {
    isSubmitting.value = false;
  }
}

async function saveNotificationSettings() {
  isSubmitting.value = true;
  try {
    const notificationSound = normalizeNotificationSound(
      notificationSettings.value.notification_sound,
    );
    const response = await usersService.updateSettings({
      email_notifications: notificationSettings.value.email_notifications,
      new_message_alerts: notificationSettings.value.new_message_alerts,
      campaign_updates: notificationSettings.value.campaign_updates,
      notification_sound: notificationSound,
    });
    const payload = unwrapResponse<{
      message: string;
      settings: UserSettings;
    }>(response);
    authStore.replaceUserSettings(payload.settings);

    toast.success(t("settings.notificationsSaved"));
  } catch (error) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedSave", {
          resource: t("resources.notificationSettings"),
        }),
      ),
    );
  } finally {
    isSubmitting.value = false;
  }
}
async function saveChatSettings() {
  const clamped = Math.min(
    300,
    Math.max(5, Math.round(chatSettings.value.media_group_window)),
  );
  const sidebarViewMode = ChatSidebarUnifier.normalizeViewMode(
    chatSettings.value.sidebar_view_mode,
  );
  const normalizedMode = normalizeAssignedChatResetMode(
    chatSettings.value.assigned_chat_reset_mode,
  );
  const normalizedHour =
    normalizedMode === "midnight"
      ? 0
      : normalizeAssignedChatResetHour(
          chatSettings.value.assigned_chat_reset_hour,
        );

  const normalizedChatCloseRatingFollowupWindowMinutes =
    normalizeChatCloseRatingFollowupWindowMinutes(
      chatSettings.value.chat_close_rating_followup_window_minutes,
    );
  const normalizedChatCloseRatingTemplates = normalizeChatCloseRatingTemplates(
    chatSettings.value.chat_close_rating_templates,
  );

  chatSettings.value.media_group_window = clamped;
  chatSettings.value.sidebar_view_mode = sidebarViewMode;
  chatSettings.value.assigned_chat_reset_mode = normalizedMode;
  chatSettings.value.assigned_chat_reset_hour = normalizedHour;

  chatSettings.value.chat_close_rating_followup_window_minutes =
    normalizedChatCloseRatingFollowupWindowMinutes;
  chatSettings.value.chat_close_rating_templates =
    normalizedChatCloseRatingTemplates;

  const nextChatBackground = resolvePendingChatBackgroundSelection();
  const shouldClearChatBackground = chatBackgroundUsesDefault.value;
  if (
    !shouldClearChatBackground &&
    chatBackgroundEditorMode.value === "upload" &&
    !nextChatBackground
  ) {
    chatBackgroundErrorKey.value = "settings.chatBackgroundUploadRequired";
    toast.error(t("settings.chatBackgroundUploadRequired"));
    return;
  }
  if (
    !shouldClearChatBackground &&
    (chatBackgroundEditorMode.value === "images" ||
      chatBackgroundEditorMode.value === "patterns") &&
    !nextChatBackground
  ) {
    chatBackgroundErrorKey.value = "settings.chatBackgroundSelectPreset";
    toast.error(t("settings.chatBackgroundSelectPreset"));
    return;
  }

  isSubmitting.value = true;
  try {
    if (shouldClearChatBackground && savedChatBackground.value) {
      const settingsResponse = await usersService.updateSettings({
        chat_background: null,
      });
      const payload = unwrapResponse<{
        message: string;
        settings: UserSettings;
      }>(settingsResponse);
      authStore.replaceUserSettings(payload.settings);
      syncChatBackgroundState(payload.settings?.chat_background);
    } else if (
      nextChatBackground?.kind === "preset" &&
      !isSameChatBackgroundPreference(
        savedChatBackground.value,
        nextChatBackground,
      )
    ) {
      const settingsResponse = await usersService.updateSettings({
        chat_background: nextChatBackground,
      });
      const payload = unwrapResponse<{
        message: string;
        settings: UserSettings;
      }>(settingsResponse);
      authStore.replaceUserSettings(payload.settings);
      syncChatBackgroundState(payload.settings?.chat_background);
    }

    if (
      !shouldClearChatBackground &&
      chatBackgroundEditorMode.value === "upload" &&
      stagedChatBackgroundFile.value
    ) {
      const uploadResponse = await usersService.uploadChatBackground(
        stagedChatBackgroundFile.value,
      );
      const payload = unwrapResponse<{
        message: string;
        chat_background: ChatBackgroundSettings;
      }>(uploadResponse);
      authStore.replaceUserSettings({
        ...(authStore.user?.settings || {}),
        chat_background: payload.chat_background,
      });
      syncChatBackgroundState(payload.chat_background);
    }

    await organizationService.updateSettings({
      assigned_chat_reset_enabled:
        chatSettings.value.assigned_chat_reset_enabled,
      assigned_chat_reset_mode: normalizedMode,
      assigned_chat_reset_hour: normalizedHour,
      chat_close_rating_enabled: chatSettings.value.chat_close_rating_enabled,

      chat_close_rating_followup_window_minutes:
        normalizedChatCloseRatingFollowupWindowMinutes,
      chat_close_rating_templates: normalizedChatCloseRatingTemplates,
    });

    localStorage.setItem(MEDIA_GROUP_WINDOW_KEY, String(clamped));
    ChatSidebarUnifier.saveViewMode(sidebarViewMode);
    configStore.setShowPrintButtons(chatSettings.value.show_print_buttons);
    configStore.setShowDownloadButtons(
      chatSettings.value.show_download_buttons,
    );

    toast.success(t("settings.chatPreferencesSaved"));
  } catch (error) {
    toast.error(
      getErrorMessage(error, t("settings.chatPreferencesSaveFailed")),
    );
  } finally {
    isSubmitting.value = false;
  }
}

onBeforeUnmount(() => {
  if (previewAudio) {
    previewAudio.pause();
    previewAudio = null;
  }
  isPreviewPlaying.value = false;
  clearStagedChatBackgroundSelection();
});
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="$t('settings.title')"
      :subtitle="$t('settings.subtitle')"
      :icon="Settings"
      icon-gradient="bg-gradient-to-br from-gray-500 to-gray-600 shadow-gray-500/20"
    />
    <ScrollArea class="flex-1">
      <div class="p-6 space-y-4 max-w-4xl mx-auto">
        <Tabs default-value="general" class="w-full">
          <TabsList
            class="mb-6 grid w-full grid-cols-3 rounded-full border border-border bg-accent/70 p-1"
          >
            <TabsTrigger
              value="general"
              class="text-muted-foreground data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm"
            >
              <Settings class="h-4 w-4 mr-2" />
              {{ $t("settings.general") }}
            </TabsTrigger>
            <TabsTrigger
              value="chat"
              class="text-muted-foreground data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm"
            >
              <MessageSquare class="h-4 w-4 mr-2" />
              {{ $t("settings.chat") }}
            </TabsTrigger>
            <TabsTrigger
              value="notifications"
              class="text-muted-foreground data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm"
            >
              <Bell class="h-4 w-4 mr-2" />
              {{ $t("settings.notifications") }}
            </TabsTrigger>
          </TabsList>

          <!-- General Settings Tab -->
          <TabsContent value="general">
            <div
              class="rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card/95 shadow-sm"
            >
              <div class="p-6 pb-3">
                <h3 class="text-lg font-semibold text-foreground">
                  {{ $t("settings.generalSettings") }}
                </h3>
                <p class="text-sm text-muted-foreground">
                  {{ $t("settings.generalSettingsDesc") }}
                </p>
              </div>
              <div class="p-6 pt-3 space-y-4">
                <div class="space-y-2">
                  <Label for="org_name" class="text-foreground/80">{{
                    $t("settings.organizationName")
                  }}</Label>
                  <Input
                    id="org_name"
                    v-model="generalSettings.organization_name"
                    :placeholder="$t('settings.organizationPlaceholder')"
                  />
                </div>
                <div class="grid grid-cols-2 gap-4">
                  <div class="space-y-2">
                    <Label for="timezone" class="text-foreground/80">{{
                      $t("settings.defaultTimezone")
                    }}</Label>
                    <Select v-model="generalSettings.default_timezone">
                      <SelectTrigger
                        class="border-input bg-input text-foreground"
                      >
                        <SelectValue
                          :placeholder="$t('settings.selectTimezone')"
                        />
                      </SelectTrigger>
                      <SelectContent
                        class="border-border bg-popover text-popover-foreground"
                      >
                        <SelectItem
                          v-for="option in timezoneOptions"
                          :key="option.value"
                          :value="option.value"
                          class="text-foreground/80 focus:bg-accent focus:text-foreground"
                        >
                          {{ option.label }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div class="space-y-2">
                    <Label for="date_format" class="text-foreground/80">{{
                      $t("settings.dateFormat")
                    }}</Label>
                    <Select v-model="generalSettings.date_format">
                      <SelectTrigger
                        class="border-input bg-input text-foreground"
                      >
                        <SelectValue
                          :placeholder="$t('settings.selectFormat')"
                        />
                      </SelectTrigger>
                      <SelectContent
                        class="border-border bg-popover text-popover-foreground"
                      >
                        <SelectItem
                          value="YYYY-MM-DD"
                          class="text-foreground/80 focus:bg-accent focus:text-foreground"
                          >YYYY-MM-DD</SelectItem
                        >
                        <SelectItem
                          value="DD/MM/YYYY"
                          class="text-foreground/80 focus:bg-accent focus:text-foreground"
                          >DD/MM/YYYY</SelectItem
                        >
                        <SelectItem
                          value="MM/DD/YYYY"
                          class="text-foreground/80 focus:bg-accent focus:text-foreground"
                          >MM/DD/YYYY</SelectItem
                        >
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <div class="space-y-2">
                  <Label class="text-foreground/80">
                    <Globe class="h-4 w-4 inline mr-1" />
                    {{ $t("settings.language") }}
                  </Label>
                  <LanguageSwitcher class="max-w-xs" />
                  <p class="text-xs text-muted-foreground">
                    {{ $t("settings.languageDesc") }}
                  </p>
                </div>
                <Separator class="bg-border" />
                <div class="flex items-center justify-between">
                  <div>
                    <p class="font-medium text-foreground">
                      {{ $t("settings.maskPhoneNumbers") }}
                    </p>
                    <p class="text-sm text-muted-foreground">
                      {{ $t("settings.maskPhoneNumbersDesc") }}
                    </p>
                  </div>
                  <Switch
                    :checked="generalSettings.mask_phone_numbers"
                    @update:checked="
                      generalSettings.mask_phone_numbers = $event
                    "
                  />
                </div>
                <Separator class="bg-border" />
                <div class="flex justify-end">
                  <Button
                    variant="outline"
                    size="sm"
                    class="border-input bg-input text-foreground hover:bg-accent"
                    @click="saveGeneralSettings"
                    :disabled="isSubmitting"
                  >
                    <Loader2
                      v-if="isSubmitting"
                      class="mr-2 h-4 w-4 animate-spin"
                    />
                    {{ $t("settings.save") }}
                  </Button>
                </div>
              </div>
            </div>
          </TabsContent>

          <!-- Notification Settings Tab -->
          <TabsContent value="notifications">
            <div
              class="rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card/95 shadow-sm"
            >
              <div class="p-6 pb-3">
                <h3 class="text-lg font-semibold text-foreground">
                  {{ $t("settings.notifications") }}
                </h3>
                <p class="text-sm text-muted-foreground">
                  {{ $t("settings.notificationsDesc") }}
                </p>
              </div>
              <div class="p-6 pt-3 space-y-4">
                <div class="flex items-center justify-between">
                  <div>
                    <p class="font-medium text-foreground">
                      {{ $t("settings.emailNotifications") }}
                    </p>
                    <p class="text-sm text-muted-foreground">
                      {{ $t("settings.emailNotificationsDesc") }}
                    </p>
                  </div>
                  <Switch
                    :checked="notificationSettings.email_notifications"
                    @update:checked="
                      notificationSettings.email_notifications = $event
                    "
                  />
                </div>
                <Separator class="bg-border" />
                <div class="flex items-center justify-between">
                  <div>
                    <p class="font-medium text-foreground">
                      {{ $t("settings.newMessageAlerts") }}
                    </p>
                    <p class="text-sm text-muted-foreground">
                      {{ $t("settings.newMessageAlertsDesc") }}
                    </p>
                  </div>
                  <Switch
                    :checked="notificationSettings.new_message_alerts"
                    @update:checked="
                      notificationSettings.new_message_alerts = $event
                    "
                  />
                </div>
                <Separator class="bg-border" />
                <div class="space-y-2">
                  <div>
                    <p class="font-medium text-foreground">
                      {{ $t("settings.notificationSound") }}
                    </p>
                    <p class="text-sm text-muted-foreground">
                      {{ $t("settings.notificationSoundDesc") }}
                    </p>
                  </div>
                  <div class="flex items-center gap-2">
                    <Select v-model="notificationSettings.notification_sound">
                      <SelectTrigger
                        class="w-full max-w-xs border-input bg-input text-foreground"
                      >
                        <SelectValue
                          :placeholder="$t('settings.selectNotificationSound')"
                        />
                      </SelectTrigger>
                      <SelectContent
                        class="border-border bg-popover text-popover-foreground"
                      >
                        <SelectItem
                          value="notification1"
                          class="text-foreground/80 focus:bg-accent focus:text-foreground"
                        >
                          {{ $t("settings.notificationSound1") }}
                        </SelectItem>
                        <SelectItem
                          value="notification2"
                          class="text-foreground/80 focus:bg-accent focus:text-foreground"
                        >
                          {{ $t("settings.notificationSound2") }}
                        </SelectItem>
                        <SelectItem
                          value="notification"
                          class="text-foreground/80 focus:bg-accent focus:text-foreground"
                        >
                          {{ $t("settings.notificationSoundOriginal") }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      class="shrink-0 border-input bg-input text-foreground hover:bg-accent"
                      :disabled="isPreviewPlaying"
                      @click="previewNotificationSound"
                    >
                      <Loader2
                        v-if="isPreviewPlaying"
                        class="h-4 w-4 mr-1 animate-spin"
                      />
                      <Play v-else class="h-4 w-4 mr-1" />
                      {{ $t("settings.previewSound") }}
                    </Button>
                  </div>
                </div>
                <Separator class="bg-border" />
                <div class="flex items-center justify-between">
                  <div>
                    <p class="font-medium text-foreground">
                      {{ $t("settings.campaignUpdates") }}
                    </p>
                    <p class="text-sm text-muted-foreground">
                      {{ $t("settings.campaignUpdatesDesc") }}
                    </p>
                  </div>
                  <Switch
                    :checked="notificationSettings.campaign_updates"
                    @update:checked="
                      notificationSettings.campaign_updates = $event
                    "
                  />
                </div>
                <div class="flex justify-end pt-4">
                  <Button
                    variant="outline"
                    size="sm"
                    class="border-input bg-input text-foreground hover:bg-accent"
                    @click="saveNotificationSettings"
                    :disabled="isSubmitting"
                  >
                    <Loader2
                      v-if="isSubmitting"
                      class="mr-2 h-4 w-4 animate-spin"
                    />
                    {{ $t("settings.save") }}
                  </Button>
                </div>
              </div>
            </div>
          </TabsContent>

          <!-- Chat Preferences Tab -->
          <TabsContent value="chat">
            <div
              class="rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card/95 shadow-sm"
            >
              <div class="p-6 pb-3">
                <h3 class="text-lg font-semibold text-foreground">
                  {{ $t("settings.chatPreferences") }}
                </h3>
                <p class="text-sm text-muted-foreground">
                  {{ $t("settings.chatPreferencesDesc") }}
                </p>
              </div>
              <div class="p-6 pt-3 space-y-4">
                <div class="space-y-2">
                  <Label for="media_group_window" class="text-foreground/80">{{
                    $t("settings.mediaGroupingWindow")
                  }}</Label>
                  <p class="text-xs text-muted-foreground">
                    {{ $t("settings.mediaGroupingWindowDesc") }}
                  </p>
                  <Select
                    :model-value="String(chatSettings.media_group_window)"
                    @update:model-value="
                      (v: unknown) => {
                        if (typeof v === 'string')
                          chatSettings.media_group_window = Number(v);
                      }
                    "
                  >
                    <SelectTrigger
                      class="w-full max-w-xs border-input bg-input text-foreground"
                    >
                      <SelectValue
                        :placeholder="$t('settings.selectGroupingWindow')"
                      />
                    </SelectTrigger>
                    <SelectContent
                      class="border-border bg-popover text-popover-foreground"
                    >
                      <SelectItem
                        value="15"
                        class="text-foreground/80 focus:bg-accent focus:text-foreground"
                        >{{
                          $t("settings.mediaGroupingWindow15Seconds")
                        }}</SelectItem
                      >
                      <SelectItem
                        value="30"
                        class="text-foreground/80 focus:bg-accent focus:text-foreground"
                        >{{
                          $t("settings.mediaGroupingWindow30Seconds")
                        }}</SelectItem
                      >
                      <SelectItem
                        value="60"
                        class="text-foreground/80 focus:bg-accent focus:text-foreground"
                        >{{
                          $t("settings.mediaGroupingWindow60SecondsDefault")
                        }}</SelectItem
                      >
                      <SelectItem
                        value="120"
                        class="text-foreground/80 focus:bg-accent focus:text-foreground"
                        >{{
                          $t("settings.mediaGroupingWindow2Minutes")
                        }}</SelectItem
                      >
                      <SelectItem
                        value="180"
                        class="text-foreground/80 focus:bg-accent focus:text-foreground"
                        >{{
                          $t("settings.mediaGroupingWindow3Minutes")
                        }}</SelectItem
                      >
                      <SelectItem
                        value="300"
                        class="text-foreground/80 focus:bg-accent focus:text-foreground"
                        >{{
                          $t("settings.mediaGroupingWindow5Minutes")
                        }}</SelectItem
                      >
                    </SelectContent>
                  </Select>
                </div>
                <div class="space-y-2">
                  <Label class="text-foreground/80">{{
                    $t("settings.sidebarViewMode")
                  }}</Label>
                  <p class="text-xs text-muted-foreground">
                    {{ $t("settings.sidebarViewModeDesc") }}
                  </p>
                  <Select v-model="chatSettings.sidebar_view_mode">
                    <SelectTrigger
                      class="w-full max-w-xs border-input bg-input text-foreground"
                    >
                      <SelectValue
                        :placeholder="$t('settings.selectSidebarViewMode')"
                      />
                    </SelectTrigger>
                    <SelectContent
                      class="border-border bg-popover text-popover-foreground"
                    >
                      <SelectItem
                        value="unified"
                        class="text-foreground/80 focus:bg-accent focus:text-foreground"
                        >{{ $t("settings.sidebarViewModeUnified") }}</SelectItem
                      >
                      <SelectItem
                        value="separate"
                        class="text-foreground/80 focus:bg-accent focus:text-foreground"
                        >{{
                          $t("settings.sidebarViewModeSeparate")
                        }}</SelectItem
                      >
                    </SelectContent>
                  </Select>
                </div>
                <Separator class="bg-border" />
                <div class="space-y-4">
                  <div class="space-y-1">
                    <Label class="text-foreground/80">{{
                      $t("settings.chatBackground")
                    }}</Label>
                    <p class="text-xs text-muted-foreground">
                      {{ $t("settings.chatBackgroundDesc") }}
                    </p>
                  </div>

                  <ToggleGroup
                    type="single"
                    :model-value="chatBackgroundEditorMode"
                    class="grid w-full grid-cols-1 gap-2 rounded-[calc(var(--radius)-0.1rem)] border border-border bg-muted/40 p-1 sm:grid-cols-4"
                    @update:model-value="
                      (value) =>
                        setChatBackgroundMode(
                          typeof value === 'string' ? value : '',
                        )
                    "
                  >
                    <ToggleGroupItem
                      value="default"
                      class="h-auto justify-start gap-2 rounded-md px-3 py-2 text-left data-[state=on]:bg-background data-[state=on]:shadow-sm"
                      data-testid="chat-background-mode-default"
                    >
                      <MessageSquare class="h-4 w-4" />
                      <span>{{ $t("settings.chatBackgroundDefault") }}</span>
                    </ToggleGroupItem>
                    <ToggleGroupItem
                      value="images"
                      class="h-auto justify-start gap-2 rounded-md px-3 py-2 text-left data-[state=on]:bg-background data-[state=on]:shadow-sm"
                      data-testid="chat-background-mode-images"
                    >
                      <ImageIcon class="h-4 w-4" />
                      <span>{{ $t("settings.chatBackgroundModeImages") }}</span>
                    </ToggleGroupItem>
                    <ToggleGroupItem
                      value="patterns"
                      class="h-auto justify-start gap-2 rounded-md px-3 py-2 text-left data-[state=on]:bg-background data-[state=on]:shadow-sm"
                      data-testid="chat-background-mode-patterns"
                    >
                      <LayoutGrid class="h-4 w-4" />
                      <span>{{
                        $t("settings.chatBackgroundModePatterns")
                      }}</span>
                    </ToggleGroupItem>
                    <ToggleGroupItem
                      value="upload"
                      class="h-auto justify-start gap-2 rounded-md px-3 py-2 text-left data-[state=on]:bg-background data-[state=on]:shadow-sm"
                      data-testid="chat-background-mode-upload"
                    >
                      <Upload class="h-4 w-4" />
                      <span>{{ $t("settings.chatBackgroundModeUpload") }}</span>
                    </ToggleGroupItem>
                  </ToggleGroup>

                  <div
                    v-if="chatBackgroundEditorMode === 'default'"
                    class="overflow-hidden rounded-xl border border-border bg-card"
                    data-testid="chat-background-default-preview"
                  >
                    <div
                      class="flex h-28 items-end justify-between border-b border-black/5 px-4 py-3"
                      :style="defaultChatBackgroundPreviewStyle"
                    >
                      <div
                        class="rounded-full bg-background/80 p-2 text-foreground shadow-sm"
                      >
                        <MessageSquare class="h-4 w-4" />
                      </div>
                    </div>
                    <div class="flex items-start justify-between gap-3 p-3">
                      <div>
                        <p class="text-sm font-medium text-foreground">
                          {{ $t("settings.chatBackgroundDefault") }}
                        </p>
                        <p class="mt-1 text-xs text-muted-foreground">
                          {{ $t("settings.chatBackgroundDefaultDesc") }}
                        </p>
                      </div>
                      <CheckCircle2 class="mt-0.5 h-4 w-4 text-primary" />
                    </div>
                  </div>

                  <div
                    v-else-if="chatBackgroundEditorMode === 'images'"
                    class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3"
                  >
                    <button
                      v-for="preset in imageChatBackgroundPresets"
                      :key="preset.id"
                      type="button"
                      class="group overflow-hidden rounded-xl border bg-card text-left transition hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60"
                      :class="
                        activeChatBackgroundPresetID === preset.id
                          ? 'border-primary shadow-sm ring-1 ring-primary/40'
                          : 'border-border'
                      "
                      :data-testid="`chat-background-preset-${preset.id}`"
                      @click="selectChatBackgroundPreset(preset.id)"
                    >
                      <div
                        class="h-32 border-b border-black/5"
                        :style="
                          resolveChatBackgroundAssetStyle(
                            preset.assetUrl,
                            preset.category,
                            'light',
                            'preview',
                          )
                        "
                      />
                      <div class="flex items-start justify-between gap-3 p-3">
                        <div>
                          <p class="text-sm font-medium text-foreground">
                            {{ $t(preset.labelKey) }}
                          </p>
                          <p class="mt-1 text-xs text-muted-foreground">
                            {{ $t(preset.descriptionKey) }}
                          </p>
                        </div>
                        <CheckCircle2
                          v-if="activeChatBackgroundPresetID === preset.id"
                          class="mt-0.5 h-4 w-4 text-primary"
                        />
                      </div>
                    </button>
                  </div>

                  <div
                    v-else-if="chatBackgroundEditorMode === 'patterns'"
                    class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3"
                  >
                    <button
                      v-for="preset in patternChatBackgroundPresets"
                      :key="preset.id"
                      type="button"
                      class="group overflow-hidden rounded-xl border bg-card text-left transition hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60"
                      :class="
                        activeChatBackgroundPresetID === preset.id
                          ? 'border-primary shadow-sm ring-1 ring-primary/40'
                          : 'border-border'
                      "
                      :data-testid="`chat-background-preset-${preset.id}`"
                      @click="selectChatBackgroundPreset(preset.id)"
                    >
                      <div
                        class="h-32 border-b border-black/5"
                        :style="
                          resolveChatBackgroundAssetStyle(
                            preset.assetUrl,
                            preset.category,
                            'light',
                            'preview',
                          )
                        "
                      />
                      <div class="flex items-start justify-between gap-3 p-3">
                        <div>
                          <p class="text-sm font-medium text-foreground">
                            {{ $t(preset.labelKey) }}
                          </p>
                          <p class="mt-1 text-xs text-muted-foreground">
                            {{ $t(preset.descriptionKey) }}
                          </p>
                        </div>
                        <CheckCircle2
                          v-if="activeChatBackgroundPresetID === preset.id"
                          class="mt-0.5 h-4 w-4 text-primary"
                        />
                      </div>
                    </button>
                  </div>

                  <div
                    v-else
                    class="space-y-3 rounded-[calc(var(--radius)-0.1rem)] border border-dashed border-border bg-muted/30 p-4"
                  >
                    <div class="space-y-1">
                      <Label
                        for="chat-background-upload"
                        class="text-foreground/80"
                        >{{ $t("settings.chatBackgroundUploadLabel") }}</Label
                      >
                      <p class="text-xs text-muted-foreground">
                        {{ $t("settings.chatBackgroundUploadDesc") }}
                      </p>
                    </div>
                    <Input
                      id="chat-background-upload"
                      type="file"
                      :accept="CHAT_BACKGROUND_UPLOAD_ACCEPT"
                      class="border-input bg-input text-foreground file:mr-3 file:rounded-md file:border-0 file:bg-primary/10 file:px-3 file:py-2 file:text-sm file:font-medium file:text-primary"
                      data-testid="chat-background-upload-input"
                      @change="
                        (event: Event) =>
                          handleChatBackgroundFileSelection(
                            (event.target as HTMLInputElement).files,
                          )
                      "
                    />
                    <p
                      v-if="chatBackgroundErrorKey"
                      class="text-sm text-destructive"
                      data-testid="chat-background-upload-error"
                    >
                      {{ $t(chatBackgroundErrorKey) }}
                    </p>
                    <div
                      v-if="
                        stagedChatBackgroundStyle ||
                        savedCustomChatBackgroundStyle
                      "
                      class="space-y-2"
                    >
                      <div
                        class="overflow-hidden rounded-xl border border-border bg-card"
                      >
                        <div
                          class="h-40"
                          :style="
                            stagedChatBackgroundStyle ||
                            savedCustomChatBackgroundStyle ||
                            {}
                          "
                        />
                        <div class="flex items-start justify-between gap-3 p-3">
                          <div>
                            <p class="text-sm font-medium text-foreground">
                              {{
                                stagedChatBackgroundFile
                                  ? stagedChatBackgroundFile.name
                                  : savedChatBackground?.custom_filename ||
                                    $t("settings.chatBackgroundCurrentCustom")
                              }}
                            </p>
                            <p class="mt-1 text-xs text-muted-foreground">
                              {{
                                stagedChatBackgroundFile
                                  ? $t("settings.chatBackgroundUploadPending")
                                  : $t("settings.chatBackgroundCurrentCustom")
                              }}
                            </p>
                          </div>
                          <CheckCircle2 class="mt-0.5 h-4 w-4 text-primary" />
                        </div>
                      </div>
                    </div>
                    <p v-else class="text-xs text-muted-foreground">
                      {{ $t("settings.chatBackgroundUploadEmpty") }}
                    </p>
                  </div>
                </div>
                <Separator class="bg-border" />
                <div class="space-y-3">
                  <div class="flex items-center justify-between">
                    <div>
                      <p class="font-medium text-foreground">
                        {{ $t("settings.showPrintButtons") }}
                      </p>
                      <p class="text-sm text-muted-foreground">
                        {{ $t("settings.showPrintButtonsDesc") }}
                      </p>
                    </div>
                    <Switch
                      :checked="chatSettings.show_print_buttons"
                      @update:checked="chatSettings.show_print_buttons = $event"
                    />
                  </div>
                  <Separator class="bg-border" />
                  <div class="flex items-center justify-between">
                    <div>
                      <p class="font-medium text-foreground">
                        {{ $t("settings.showDownloadButtons") }}
                      </p>
                      <p class="text-sm text-muted-foreground">
                        {{ $t("settings.showDownloadButtonsDesc") }}
                      </p>
                    </div>
                    <Switch
                      :checked="chatSettings.show_download_buttons"
                      @update:checked="
                        chatSettings.show_download_buttons = $event
                      "
                    />
                  </div>
                </div>
                <Separator class="bg-border" />
                <div class="space-y-2">
                  <div class="flex items-center justify-between gap-3">
                    <div>
                      <Label class="text-foreground/80">{{
                        $t("settings.assignedChatResetEnabled")
                      }}</Label>
                      <p class="text-xs text-muted-foreground">
                        {{ $t("settings.assignedChatResetEnabledDesc") }}
                      </p>
                    </div>
                    <Switch
                      :checked="chatSettings.assigned_chat_reset_enabled"
                      @update:checked="
                        chatSettings.assigned_chat_reset_enabled = $event
                      "
                    />
                  </div>
                  <Label class="text-foreground/80">{{
                    $t("settings.assignedChatResetSchedule")
                  }}</Label>
                  <p class="text-xs text-muted-foreground">
                    {{ $t("settings.assignedChatResetScheduleDesc") }}
                  </p>
                  <Select
                    v-model="chatSettings.assigned_chat_reset_mode"
                    :disabled="!chatSettings.assigned_chat_reset_enabled"
                  >
                    <SelectTrigger
                      class="w-full max-w-xs border-input bg-input text-foreground"
                    >
                      <SelectValue
                        :placeholder="$t('settings.selectResetSchedule')"
                      />
                    </SelectTrigger>
                    <SelectContent
                      class="border-border bg-popover text-popover-foreground"
                    >
                      <SelectItem
                        value="midnight"
                        class="text-foreground/80 focus:bg-accent focus:text-foreground"
                      >
                        {{ $t("settings.defaultMidnight") }}
                      </SelectItem>
                      <SelectItem
                        value="custom_hour"
                        class="text-foreground/80 focus:bg-accent focus:text-foreground"
                      >
                        {{ $t("settings.customHour") }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div
                  v-if="
                    chatSettings.assigned_chat_reset_enabled &&
                    chatSettings.assigned_chat_reset_mode === 'custom_hour'
                  "
                  class="space-y-2"
                >
                  <Label class="text-foreground/80">{{
                    $t("settings.customResetHour")
                  }}</Label>
                  <Select
                    :model-value="String(chatSettings.assigned_chat_reset_hour)"
                    @update:model-value="
                      (v: unknown) => {
                        if (typeof v === 'string')
                          chatSettings.assigned_chat_reset_hour = Number(v);
                      }
                    "
                    :disabled="!chatSettings.assigned_chat_reset_enabled"
                  >
                    <SelectTrigger
                      class="w-full max-w-xs border-input bg-input text-foreground"
                    >
                      <SelectValue
                        :placeholder="$t('settings.selectResetHour')"
                      />
                    </SelectTrigger>
                    <SelectContent
                      class="border-border bg-popover text-popover-foreground"
                    >
                      <SelectItem
                        v-for="option in chatResetHourOptions"
                        :key="option.value"
                        :value="option.value"
                        class="text-foreground/80 focus:bg-accent focus:text-foreground"
                      >
                        {{ option.label }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <Separator class="bg-border" />
                <div class="space-y-3">
                  <div class="flex items-center justify-between gap-3">
                    <div>
                      <Label class="text-foreground/80">{{
                        $t("settings.chatCloseRatingEnabled")
                      }}</Label>
                      <p class="text-xs text-muted-foreground">
                        {{ $t("settings.chatCloseRatingEnabledDesc") }}
                      </p>
                    </div>
                    <Switch
                      :checked="chatSettings.chat_close_rating_enabled"
                      @update:checked="
                        chatSettings.chat_close_rating_enabled = $event
                      "
                    />
                  </div>

                  <div class="space-y-2">
                    <Label class="text-foreground/80">{{
                      $t("settings.chatCloseRatingFollowupWindowMinutes")
                    }}</Label>
                    <p class="text-xs text-muted-foreground">
                      {{
                        $t("settings.chatCloseRatingFollowupWindowMinutesDesc")
                      }}
                    </p>
                    <Input
                      type="number"
                      min="1"
                      max="1440"
                      step="1"
                      class="w-full max-w-xs border-input bg-input text-foreground"
                      :model-value="
                        String(
                          chatSettings.chat_close_rating_followup_window_minutes,
                        )
                      "
                      :disabled="!chatSettings.chat_close_rating_enabled"
                      @update:model-value="
                        (v: unknown) => {
                          if (typeof v === 'string') {
                            chatSettings.chat_close_rating_followup_window_minutes =
                              Number(v);
                          }
                        }
                      "
                    />
                  </div>

                  <div class="space-y-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      class="border-input bg-input text-foreground hover:bg-accent"
                      :disabled="!chatSettings.chat_close_rating_enabled"
                      @click="
                        showChatCloseRatingTemplates =
                          !showChatCloseRatingTemplates
                      "
                    >
                      {{ $t("settings.chatCloseRatingTemplates") }}
                    </Button>
                    <div
                      v-if="showChatCloseRatingTemplates"
                      class="space-y-3 rounded-[calc(var(--radius)-0.1rem)] border border-border bg-card/80 p-3"
                    >
                      <p class="text-xs text-muted-foreground">
                        {{ $t("settings.chatCloseRatingTemplatesDesc") }}
                      </p>

                      <div class="space-y-2">
                        <Label class="text-foreground/80">{{
                          $t("settings.chatCloseRatingTemplateEn")
                        }}</Label>
                        <Textarea
                          v-model="chatSettings.chat_close_rating_templates.en"
                          :rows="3"
                          class="border-input bg-input text-foreground placeholder:text-muted-foreground"
                          :placeholder="
                            $t('settings.chatCloseRatingTemplatePlaceholder')
                          "
                          :disabled="!chatSettings.chat_close_rating_enabled"
                        />
                      </div>

                      <div class="space-y-2">
                        <Label class="text-foreground/80">{{
                          $t("settings.chatCloseRatingTemplateAr")
                        }}</Label>
                        <Textarea
                          v-model="chatSettings.chat_close_rating_templates.ar"
                          :rows="3"
                          class="border-input bg-input text-foreground placeholder:text-muted-foreground"
                          :placeholder="
                            $t('settings.chatCloseRatingTemplatePlaceholder')
                          "
                          :disabled="!chatSettings.chat_close_rating_enabled"
                        />
                      </div>

                      <div class="space-y-2">
                        <Label class="text-foreground/80">{{
                          $t("settings.chatCloseRatingTemplateEs")
                        }}</Label>
                        <Textarea
                          v-model="chatSettings.chat_close_rating_templates.es"
                          :rows="3"
                          class="border-input bg-input text-foreground placeholder:text-muted-foreground"
                          :placeholder="
                            $t('settings.chatCloseRatingTemplatePlaceholder')
                          "
                          :disabled="!chatSettings.chat_close_rating_enabled"
                        />
                      </div>

                      <p class="font-mono text-[11px] text-muted-foreground">
                        {{ $t("settings.chatCloseRatingPlaceholders") }}
                      </p>
                    </div>
                  </div>
                </div>
                <Separator class="bg-border" />
                <div class="space-y-3">
                  <div>
                    <h4 class="text-sm font-medium text-foreground">
                      {{ $t("settings.chatQueues") }}
                    </h4>
                    <p class="text-xs text-muted-foreground">
                      {{ $t("settings.chatQueuesDesc") }}
                    </p>
                  </div>
                  <div class="grid gap-2 sm:grid-cols-1">
                    <RouterLink
                      to="/settings/closed-chats"
                      class="rounded-lg border border-border bg-muted px-3 py-2 text-xs text-muted-foreground hover:bg-accent"
                    >
                      <span class="inline-flex items-center gap-1.5 font-medium"
                        ><Archive class="h-3.5 w-3.5" />
                        {{ $t("settings.closedChats") }}</span
                      >
                    </RouterLink>
                  </div>
                </div>
                <div class="flex justify-end pt-4">
                  <Button
                    variant="outline"
                    size="sm"
                    class="border-input bg-input text-foreground hover:bg-accent"
                    @click="saveChatSettings"
                    :disabled="isSubmitting"
                    data-testid="settings-chat-save"
                  >
                    <Loader2
                      v-if="isSubmitting"
                      class="mr-2 h-4 w-4 animate-spin"
                    />
                    {{ $t("settings.save") }}
                  </Button>
                </div>
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </ScrollArea>
  </div>
</template>
