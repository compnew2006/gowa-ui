<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, onBeforeRouteLeave } from "vue-router";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
  Palette,
  MessageSquare,
  Play,
  Archive,
  ImageIcon,
  LayoutGrid,
  Upload,
  CheckCircle2,
  MoonStar,
  SunMedium,
  MonitorSmartphone,
} from "lucide-vue-next";
import { usersService, organizationService } from "@/services/api";
import { useAuthStore } from "@/stores/auth";
import { useConfigStore } from "@/stores/config";
import { useColorMode } from "@/composables/useColorMode";
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
import {
  THEME_PRESET_OPTIONS,
  getAppearanceFromSettings,
  normalizeColorMode,
  type AppearanceSettings,
} from "@/lib/theme-presets";
import type {
  ChatBackgroundSettings,
  UserSettings,
  ThemePreset,
} from "@/types/auth";

const { t } = useI18n();
const authStore = useAuthStore();
const configStore = useConfigStore();
const {
  persistedColorMode,
  persistedThemePreset,
  previewAppearance,
  restorePersistedAppearance,
  hydrateFromUserSettings,
} = useColorMode();

type NotificationSoundKey = "notification1" | "notification2" | "notification";
const DEFAULT_NOTIFICATION_SOUND: NotificationSoundKey = "notification1";
const MAX_UPLOADS_CLEANUP_RETENTION_DAYS = 3650;
const DEFAULT_UPLOADS_CLEANUP_SCHEDULE_HOUR = 3;

const activeSettingsTab = ref("general");
const isSubmitting = ref(false);
const isUploadsCleanupSubmitting = ref(false);
const isUploadsCleanupRunning = ref(false);
const isLoading = ref(true);
const isPreviewPlaying = ref(false);
let previewAudio: HTMLAudioElement | null = null;

const canViewGeneralSettings = computed(() =>
  authStore.hasPermission("settings.general", "read"),
);
const canEditGeneralSettings = computed(() =>
  authStore.hasPermission("settings.general", "write"),
);
const canViewUploadsCleanup = computed(
  () =>
    authStore.hasPermission("settings.uploads_cleanup", "read") ||
    authStore.hasPermission("settings.uploads_cleanup", "write") ||
    authStore.hasPermission("settings.uploads_cleanup", "execute"),
);
const canEditUploadsCleanup = computed(() =>
  authStore.hasPermission("settings.uploads_cleanup", "write"),
);
const canRunUploadsCleanup = computed(() =>
  authStore.hasPermission("settings.uploads_cleanup", "execute"),
);

const themePresetOptions = THEME_PRESET_OPTIONS;
const appearanceSettings = ref<AppearanceSettings>({
  theme_mode: persistedColorMode.value,
  theme_preset: persistedThemePreset.value,
});
const savedAppearanceSettings = ref<AppearanceSettings>({
  theme_mode: persistedColorMode.value,
  theme_preset: persistedThemePreset.value,
});
const isAppearanceDirty = computed(
  () =>
    appearanceSettings.value.theme_mode !==
      savedAppearanceSettings.value.theme_mode ||
    appearanceSettings.value.theme_preset !==
      savedAppearanceSettings.value.theme_preset,
);

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

function syncAppearanceSettings(value?: Partial<UserSettings> | null) {
  const nextAppearance = value
    ? getAppearanceFromSettings(value)
    : {
        theme_mode: persistedColorMode.value,
        theme_preset: persistedThemePreset.value,
      };

  savedAppearanceSettings.value = { ...nextAppearance };
  appearanceSettings.value = { ...nextAppearance };
}

function previewDraftAppearance() {
  previewAppearance(
    appearanceSettings.value.theme_mode,
    appearanceSettings.value.theme_preset,
  );
}

function selectAppearanceMode(mode: string) {
  appearanceSettings.value.theme_mode = normalizeColorMode(mode);
  previewDraftAppearance();
}

function selectThemePreset(preset: ThemePreset) {
  appearanceSettings.value.theme_preset = preset;
  previewDraftAppearance();
}

function revertAppearancePreview() {
  appearanceSettings.value = { ...savedAppearanceSettings.value };
  restorePersistedAppearance();
}

interface GeneralSettingsForm {
  organization_name: string;
  default_timezone: string;
  date_format: string;
  mask_phone_numbers: boolean;
}

// General Settings
const generalSettings = ref<GeneralSettingsForm>({
  organization_name: "My Organization",
  default_timezone: "UTC",
  date_format: "YYYY-MM-DD",
  mask_phone_numbers: false,
});

function parseUploadsCleanupRetentionDaysInput(value: unknown): number | null {
  if (typeof value === "number") {
    if (
      !Number.isInteger(value) ||
      value < 0 ||
      value > MAX_UPLOADS_CLEANUP_RETENTION_DAYS
    ) {
      return null;
    }

    return value;
  }

  if (typeof value !== "string") {
    return null;
  }

  const trimmed = value.trim();
  if (trimmed === "") return 0;
  if (!/^\d+$/.test(trimmed)) {
    return null;
  }

  const parsed = Number(trimmed);
  if (
    !Number.isInteger(parsed) ||
    parsed < 0 ||
    parsed > MAX_UPLOADS_CLEANUP_RETENTION_DAYS
  ) {
    return null;
  }

  return parsed;
}

function parseUploadsCleanupScheduleHourInput(value: unknown): number | null {
  if (typeof value === "number") {
    if (!Number.isInteger(value) || value < 0 || value > 23) {
      return null;
    }

    return value;
  }

  if (typeof value !== "string") {
    return null;
  }

  const trimmed = value.trim();
  if (trimmed === "") {
    return DEFAULT_UPLOADS_CLEANUP_SCHEDULE_HOUR;
  }
  if (!/^\d+$/.test(trimmed)) {
    return null;
  }

  const parsed = Number(trimmed);
  if (!Number.isInteger(parsed) || parsed < 0 || parsed > 23) {
    return null;
  }

  return parsed;
}

interface UploadsCleanupSettingsForm {
  retention_days: string | number;
  schedule_hour: string | number;
  timezone: string;
}

const uploadsCleanupSettings = ref<UploadsCleanupSettingsForm>({
  retention_days: "0",
  schedule_hour: String(DEFAULT_UPLOADS_CLEANUP_SCHEDULE_HOUR),
  timezone: "UTC",
});

const uploadsCleanupScheduleLabel = computed(() => {
  const parsedHour = parseUploadsCleanupScheduleHourInput(
    uploadsCleanupSettings.value.schedule_hour,
  );
  const hour =
    parsedHour === null ? DEFAULT_UPLOADS_CLEANUP_SCHEDULE_HOUR : parsedHour;
  return `${String(hour).padStart(2, "0")}:00`;
});

function buildUploadsCleanupPayload() {
  const retentionDays = parseUploadsCleanupRetentionDaysInput(
    uploadsCleanupSettings.value.retention_days,
  );
  if (retentionDays === null) {
    toast.error(t("settings.uploadsCleanupRetentionDaysInvalid"));
    return null;
  }

  const scheduleHour = parseUploadsCleanupScheduleHourInput(
    uploadsCleanupSettings.value.schedule_hour,
  );
  if (scheduleHour === null) {
    toast.error(t("settings.uploadsCleanupScheduleHourInvalid"));
    return null;
  }

  return {
    uploads_cleanup_retention_days: retentionDays,
    uploads_cleanup_schedule_hour: scheduleHour,
  };
}

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

// Chat Preferences (localStorage-only)
const MEDIA_GROUP_WINDOW_KEY = "chat.mediaGroupWindowSeconds";
const chatSettings = ref({
  media_group_window: 60,
  sidebar_view_mode: ChatSidebarUnifier.readViewMode() as ChatSidebarViewMode,
  show_print_buttons: configStore.showPrintButtons,
  show_download_buttons: configStore.showDownloadButtons,
});
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

// Load chat settings from localStorage
try {
  const stored = Number(localStorage.getItem(MEDIA_GROUP_WINDOW_KEY));
  if (Number.isFinite(stored) && stored >= 5 && stored <= 300) {
    chatSettings.value.media_group_window = stored;
  }
} catch {
  // Ignore localStorage errors
}

watch(
  [persistedColorMode, persistedThemePreset],
  ([themeMode, themePreset]) => {
    if (!isAppearanceDirty.value) {
      syncAppearanceSettings({
        theme_mode: themeMode,
        theme_preset: themePreset,
      });
    }
  },
  { immediate: true },
);

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
      if (canViewGeneralSettings.value) {
        generalSettings.value = {
          organization_name: orgData.name || "My Organization",
          default_timezone: orgData.settings?.timezone || "UTC",
          date_format: orgData.settings?.date_format || "YYYY-MM-DD",
          mask_phone_numbers: orgData.settings?.mask_phone_numbers || false,
        };
      }

      if (canViewUploadsCleanup.value) {
        uploadsCleanupSettings.value = {
          retention_days: String(
            orgData.settings?.uploads_cleanup_retention_days ?? 0,
          ),
          schedule_hour: String(
            orgData.settings?.uploads_cleanup_schedule_hour ??
              DEFAULT_UPLOADS_CLEANUP_SCHEDULE_HOUR,
          ),
          timezone:
            orgData.settings?.timezone ||
            generalSettings.value.default_timezone ||
            "UTC",
        };
      }
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
    if (authStore.user) {
      hydrateFromUserSettings(user.settings);
    }
    syncAppearanceSettings(user.settings);
    syncChatBackgroundState(user.settings?.chat_background);
  } catch (error) {
    console.error("Failed to load settings:", error);
  } finally {
    isLoading.value = false;
  }
});

async function saveGeneralSettings() {
  if (!canEditGeneralSettings.value) {
    return;
  }

  isSubmitting.value = true;
  try {
    await organizationService.updateSettings({
      name: generalSettings.value.organization_name,
      timezone: generalSettings.value.default_timezone,
      date_format: generalSettings.value.date_format,
      mask_phone_numbers: generalSettings.value.mask_phone_numbers,
    });
    uploadsCleanupSettings.value.timezone =
      generalSettings.value.default_timezone;
    toast.success(t("settings.generalSaved"));
  } catch (error) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedSave", { resource: t("resources.settings") }),
      ),
    );
  } finally {
    isSubmitting.value = false;
  }
}

async function saveUploadsCleanupSettings() {
  if (!canEditUploadsCleanup.value) {
    return;
  }

  const payload = buildUploadsCleanupPayload();
  if (!payload) {
    return;
  }

  isUploadsCleanupSubmitting.value = true;
  try {
    await organizationService.updateSettings(payload);
    uploadsCleanupSettings.value.schedule_hour = String(
      payload.uploads_cleanup_schedule_hour,
    );
    uploadsCleanupSettings.value.retention_days = String(
      payload.uploads_cleanup_retention_days,
    );
    toast.success(t("settings.uploadsCleanupSaved"));
  } catch (error) {
    toast.error(getErrorMessage(error, t("settings.uploadsCleanupSaveFailed")));
  } finally {
    isUploadsCleanupSubmitting.value = false;
  }
}

async function runUploadsCleanupNow() {
  let payload: {
    uploads_cleanup_retention_days: number;
    uploads_cleanup_schedule_hour: number;
  } | null = null;

  if (canEditUploadsCleanup.value) {
    payload = buildUploadsCleanupPayload();
    if (!payload) {
      return;
    }
  }

  isUploadsCleanupRunning.value = true;
  try {
    if (payload) {
      await organizationService.updateSettings(payload);
      uploadsCleanupSettings.value.schedule_hour = String(
        payload.uploads_cleanup_schedule_hour,
      );
      uploadsCleanupSettings.value.retention_days = String(
        payload.uploads_cleanup_retention_days,
      );
    }

    const response = await organizationService.runUploadsCleanupNow();
    const result = unwrapResponse<{
      message: string;
      deleted_files: number;
      retention_days: number;
    }>(response);
    toast.success(
      result.message ||
        t("settings.uploadsCleanupRunSuccess", {
          count: result.deleted_files,
          days: result.retention_days,
        }),
    );
  } catch (error) {
    toast.error(getErrorMessage(error, t("settings.uploadsCleanupRunFailed")));
  } finally {
    isUploadsCleanupRunning.value = false;
  }
}

async function saveAppearanceSettings() {
  isSubmitting.value = true;
  try {
    const response = await usersService.updateSettings({
      theme_mode: appearanceSettings.value.theme_mode,
      theme_preset: appearanceSettings.value.theme_preset,
    });
    const payload = unwrapResponse<{
      message: string;
      settings: UserSettings;
    }>(response);

    authStore.replaceUserSettings(payload.settings);
    hydrateFromUserSettings(payload.settings);
    syncAppearanceSettings(payload.settings);

    toast.success(t("settings.appearanceSaved"));
  } catch (error) {
    revertAppearancePreview();
    toast.error(getErrorMessage(error, t("settings.appearanceSaveFailed")));
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

  chatSettings.value.media_group_window = clamped;
  chatSettings.value.sidebar_view_mode = sidebarViewMode;

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
  if (isAppearanceDirty.value) {
    revertAppearancePreview();
  }
  if (previewAudio) {
    previewAudio.pause();
    previewAudio = null;
  }
  isPreviewPlaying.value = false;
  clearStagedChatBackgroundSelection();
});

onBeforeRouteLeave(() => {
  if (isAppearanceDirty.value) {
    revertAppearancePreview();
  }
});
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="$t('settings.title')"
      :subtitle="$t('settings.subtitle')"
      :icon="Settings"
      icon-gradient="bg-gradient-to-br from-blue-500 to-sky-600 shadow-blue-500/20"
    />
    <ScrollArea class="flex-1">
      <div class="p-6 space-y-4 max-w-4xl mx-auto">
        <Tabs v-model="activeSettingsTab" class="w-full">
          <TabsList
            class="mb-6 grid w-full grid-cols-2 gap-1 rounded-full border border-border bg-accent/70 p-1 sm:grid-cols-4"
          >
            <TabsTrigger
              value="general"
              class="text-muted-foreground data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm"
            >
              <Settings class="h-4 w-4 mr-2" />
              {{ $t("settings.general") }}
            </TabsTrigger>
            <TabsTrigger
              value="appearance"
              class="text-muted-foreground data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm"
              data-testid="settings-tab-appearance"
            >
              <Palette class="h-4 w-4 mr-2" />
              {{ $t("settings.appearance") }}
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
            <div class="space-y-6">
              <div
                v-if="canViewGeneralSettings"
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
                      variant="default"
                      size="sm"
                      class="shadow-sm"
                      @click="saveGeneralSettings"
                      :disabled="isSubmitting || !canEditGeneralSettings"
                      data-testid="settings-general-save"
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

              <div
                v-if="canViewUploadsCleanup"
                class="rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card/95 shadow-sm"
              >
                <div class="p-6 pb-3">
                  <h3 class="text-lg font-semibold text-foreground">
                    {{ $t("settings.uploadsCleanupTitle") }}
                  </h3>
                  <p class="text-sm text-muted-foreground">
                    {{ $t("settings.uploadsCleanupDesc") }}
                  </p>
                </div>
                <div class="p-6 pt-3 space-y-4">
                  <div class="grid gap-4 md:grid-cols-2">
                    <div class="space-y-2 max-w-xs">
                      <Label
                        for="uploads_cleanup_retention_days"
                        class="text-foreground/80"
                      >
                        {{ $t("settings.uploadsCleanupRetentionDays") }}
                      </Label>
                      <Input
                        id="uploads_cleanup_retention_days"
                        v-model="uploadsCleanupSettings.retention_days"
                        type="number"
                        min="0"
                        :max="String(MAX_UPLOADS_CLEANUP_RETENTION_DAYS)"
                        step="1"
                        :disabled="!canEditUploadsCleanup"
                        data-testid="uploads-cleanup-retention-days-input"
                      />
                      <p class="text-xs text-muted-foreground">
                        {{ $t("settings.uploadsCleanupRetentionDaysDesc") }}
                      </p>
                    </div>
                    <div class="space-y-2 max-w-xs">
                      <Label
                        for="uploads_cleanup_schedule_hour"
                        class="text-foreground/80"
                      >
                        {{ $t("settings.uploadsCleanupScheduleHour") }}
                      </Label>
                      <Input
                        id="uploads_cleanup_schedule_hour"
                        v-model="uploadsCleanupSettings.schedule_hour"
                        type="number"
                        min="0"
                        max="23"
                        step="1"
                        :disabled="!canEditUploadsCleanup"
                        data-testid="uploads-cleanup-schedule-hour-input"
                      />
                      <p class="text-xs text-muted-foreground">
                        {{
                          $t("settings.uploadsCleanupScheduleHourDesc", {
                            time: uploadsCleanupScheduleLabel,
                          })
                        }}
                      </p>
                    </div>
                  </div>
                  <div
                    class="rounded-xl border border-border/60 bg-background/70 p-4"
                  >
                    <p class="text-sm font-medium text-foreground">
                      {{ $t("settings.uploadsCleanupTimezone") }}
                    </p>
                    <p class="mt-1 text-sm text-muted-foreground">
                      {{
                        $t("settings.uploadsCleanupTimezoneDesc", {
                          timezone: uploadsCleanupSettings.timezone,
                          time: uploadsCleanupScheduleLabel,
                        })
                      }}
                    </p>
                  </div>
                  <div class="flex flex-wrap justify-end gap-2">
                    <Button
                      v-if="canRunUploadsCleanup"
                      variant="outline"
                      size="sm"
                      class="shadow-sm"
                      @click="runUploadsCleanupNow"
                      :disabled="
                        isUploadsCleanupRunning || isUploadsCleanupSubmitting
                      "
                      data-testid="uploads-cleanup-run-now"
                    >
                      <Loader2
                        v-if="isUploadsCleanupRunning"
                        class="mr-2 h-4 w-4 animate-spin"
                      />
                      {{ $t("settings.uploadsCleanupRunNow") }}
                    </Button>
                    <Button
                      v-if="canEditUploadsCleanup"
                      variant="default"
                      size="sm"
                      class="shadow-sm"
                      @click="saveUploadsCleanupSettings"
                      :disabled="
                        isUploadsCleanupSubmitting || isUploadsCleanupRunning
                      "
                      data-testid="uploads-cleanup-save"
                    >
                      <Loader2
                        v-if="isUploadsCleanupSubmitting"
                        class="mr-2 h-4 w-4 animate-spin"
                      />
                      {{ $t("settings.save") }}
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="appearance">
            <div
              class="rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card/95 shadow-sm"
            >
              <div class="p-6 pb-3">
                <h3 class="text-lg font-semibold text-foreground">
                  {{ $t("settings.appearance") }}
                </h3>
                <p class="text-sm text-muted-foreground">
                  {{ $t("settings.appearanceDesc") }}
                </p>
              </div>
              <div class="space-y-5 p-6 pt-3">
                <div class="space-y-3">
                  <div class="space-y-1">
                    <Label class="text-foreground/80">{{
                      $t("settings.themeMode")
                    }}</Label>
                    <p class="text-xs text-muted-foreground">
                      {{ $t("settings.themeModeDesc") }}
                    </p>
                  </div>
                  <div class="grid gap-2 sm:grid-cols-3">
                    <button
                      type="button"
                      class="flex items-center gap-3 rounded-xl border px-4 py-3 text-left transition hover:-translate-y-0.5 hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60"
                      :class="
                        appearanceSettings.theme_mode === 'light'
                          ? 'border-primary bg-primary/10 text-foreground shadow-sm'
                          : 'border-border bg-background text-muted-foreground'
                      "
                      data-testid="appearance-mode-light"
                      @click="selectAppearanceMode('light')"
                    >
                      <div class="rounded-full bg-background/80 p-2 shadow-sm">
                        <SunMedium class="h-4 w-4" />
                      </div>
                      <div>
                        <p class="text-sm font-medium text-foreground">
                          {{ $t("settings.lightMode") }}
                        </p>
                      </div>
                    </button>
                    <button
                      type="button"
                      class="flex items-center gap-3 rounded-xl border px-4 py-3 text-left transition hover:-translate-y-0.5 hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60"
                      :class="
                        appearanceSettings.theme_mode === 'dark'
                          ? 'border-primary bg-primary/10 text-foreground shadow-sm'
                          : 'border-border bg-background text-muted-foreground'
                      "
                      data-testid="appearance-mode-dark"
                      @click="selectAppearanceMode('dark')"
                    >
                      <div class="rounded-full bg-background/80 p-2 shadow-sm">
                        <MoonStar class="h-4 w-4" />
                      </div>
                      <div>
                        <p class="text-sm font-medium text-foreground">
                          {{ $t("settings.darkMode") }}
                        </p>
                      </div>
                    </button>
                    <button
                      type="button"
                      class="flex items-center gap-3 rounded-xl border px-4 py-3 text-left transition hover:-translate-y-0.5 hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60"
                      :class="
                        appearanceSettings.theme_mode === 'system'
                          ? 'border-primary bg-primary/10 text-foreground shadow-sm'
                          : 'border-border bg-background text-muted-foreground'
                      "
                      data-testid="appearance-mode-system"
                      @click="selectAppearanceMode('system')"
                    >
                      <div class="rounded-full bg-background/80 p-2 shadow-sm">
                        <MonitorSmartphone class="h-4 w-4" />
                      </div>
                      <div>
                        <p class="text-sm font-medium text-foreground">
                          {{ $t("settings.systemTheme") }}
                        </p>
                      </div>
                    </button>
                  </div>
                </div>

                <Separator class="bg-border" />

                <div class="space-y-3">
                  <div class="space-y-1">
                    <Label class="text-foreground/80">{{
                      $t("settings.themePreset")
                    }}</Label>
                    <p class="text-xs text-muted-foreground">
                      {{ $t("settings.themePresetDesc") }}
                    </p>
                  </div>
                  <div class="grid gap-3 sm:grid-cols-2">
                    <button
                      v-for="option in themePresetOptions"
                      :key="option.id"
                      type="button"
                      class="group overflow-hidden rounded-[calc(var(--radius)+0.2rem)] border bg-card text-left transition hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60"
                      :class="
                        appearanceSettings.theme_preset === option.id
                          ? 'border-primary shadow-sm ring-1 ring-primary/35'
                          : 'border-border'
                      "
                      :data-testid="`appearance-preset-${option.id}`"
                      @click="selectThemePreset(option.id)"
                    >
                      <div
                        class="flex min-h-[132px] items-end justify-between border-b border-black/5 px-4 py-4"
                        :style="{ background: option.previewBackground }"
                      >
                        <div class="space-y-2">
                          <div
                            class="h-3 w-16 rounded-full"
                            :style="{ backgroundColor: option.previewAccent }"
                          />
                          <div class="h-3 w-24 rounded-full bg-white/65" />
                          <div class="h-3 w-20 rounded-full bg-white/45" />
                        </div>
                        <div
                          class="rounded-full px-3 py-1 text-xs font-semibold shadow-sm"
                          :style="{
                            backgroundColor: option.previewAccent,
                            color: 'rgb(255 255 255)',
                          }"
                        >
                          {{ $t(option.labelKey) }}
                        </div>
                      </div>
                      <div class="flex items-start justify-between gap-3 p-4">
                        <div>
                          <p class="text-sm font-semibold text-foreground">
                            {{ $t(option.labelKey) }}
                          </p>
                          <p
                            class="mt-1 text-xs leading-5 text-muted-foreground"
                          >
                            {{ $t(option.descriptionKey) }}
                          </p>
                        </div>
                        <CheckCircle2
                          v-if="appearanceSettings.theme_preset === option.id"
                          class="mt-0.5 h-4 w-4 shrink-0 text-primary"
                        />
                      </div>
                    </button>
                  </div>
                  <p
                    class="rounded-lg border border-dashed border-border bg-muted/40 px-3 py-2 text-xs text-muted-foreground"
                  >
                    {{ $t("settings.appearancePreviewHint") }}
                  </p>
                </div>

                <div class="flex justify-end gap-2 pt-2">
                  <Button
                    variant="outline"
                    size="sm"
                    class="border-input bg-input text-foreground hover:bg-accent"
                    :disabled="isSubmitting || !isAppearanceDirty"
                    data-testid="settings-appearance-revert"
                    @click="revertAppearancePreview"
                  >
                    {{ $t("common.cancel") }}
                  </Button>
                  <Button
                    variant="default"
                    size="sm"
                    class="shadow-sm"
                    :disabled="isSubmitting || !isAppearanceDirty"
                    data-testid="settings-appearance-save"
                    @click="saveAppearanceSettings"
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
