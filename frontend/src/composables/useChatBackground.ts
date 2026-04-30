import { computed, ref } from "vue";
import {
  CHAT_BACKGROUND_PRESETS,
  getChatBackgroundPreset,
  isSameChatBackgroundPreference,
  normalizeChatBackgroundPreference,
  resolveChatBackgroundAssetStyle,
  resolveChatBackgroundEditorMode,
  resolveChatBackgroundStyle,
  validateChatBackgroundFile,
  type ChatBackgroundEditorMode,
} from "@/lib/chat-backgrounds";
import type { ChatBackgroundSettings } from "@/types/auth";

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

const isChatBackgroundDirty = computed(() => {
  if (stagedChatBackgroundFile.value) return true;
  return !isSameChatBackgroundPreference(
    savedChatBackground.value,
    resolvePendingChatBackgroundSelection(),
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

export function useChatBackground() {
  return {
    savedChatBackground,
    chatBackgroundEditorMode,
    selectedChatBackgroundPresetID,
    stagedChatBackgroundFile,
    stagedChatBackgroundPreviewURL,
    chatBackgroundErrorKey,
    chatBackgroundUsesDefault,
    imageChatBackgroundPresets,
    patternChatBackgroundPresets,
    activeChatBackgroundPresetID,
    defaultChatBackgroundPreviewStyle,
    savedCustomChatBackgroundStyle,
    stagedChatBackgroundStyle,
    isChatBackgroundDirty,
    clearStagedChatBackgroundPreview,
    clearStagedChatBackgroundSelection,
    syncChatBackgroundState,
    setChatBackgroundMode,
    selectDefaultChatBackground,
    selectChatBackgroundPreset,
    handleChatBackgroundFileSelection,
    resolvePendingChatBackgroundSelection,
  };
}
