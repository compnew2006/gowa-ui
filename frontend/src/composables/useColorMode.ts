import { computed, onMounted, ref } from "vue";
import type { ThemeMode, ThemePreset, UserSettings } from "@/types/auth";
import {
  COLOR_MODE_STORAGE_KEY,
  DEFAULT_COLOR_MODE,
  DEFAULT_THEME_PRESET,
  THEME_PRESET_STORAGE_KEY,
  getAppearanceFromSettings,
  getStoredAppearance,
  normalizeColorMode,
  normalizeThemePreset,
  type AppearanceSettings,
  type ResolvedColorMode,
} from "@/lib/theme-presets";

const colorMode = ref<ThemeMode>(DEFAULT_COLOR_MODE);
const themePreset = ref<ThemePreset>(DEFAULT_THEME_PRESET);
const persistedColorMode = ref<ThemeMode>(DEFAULT_COLOR_MODE);
const persistedThemePreset = ref<ThemePreset>(DEFAULT_THEME_PRESET);
const resolvedColorMode = ref<ResolvedColorMode>("light");
const isDark = computed(() => resolvedColorMode.value === "dark");
const hasUnsavedAppearance = computed(
  () =>
    colorMode.value !== persistedColorMode.value ||
    themePreset.value !== persistedThemePreset.value,
);

let mediaQuery: MediaQueryList | null = null;
let mediaListenerAttached = false;
let isInitialized = false;

function canUseDOM(): boolean {
  return typeof window !== "undefined" && typeof document !== "undefined";
}

function resolveMode(mode: ThemeMode): ResolvedColorMode {
  if (!canUseDOM()) {
    return mode === "dark" ? "dark" : "light";
  }

  if (mode === "system") {
    return window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "dark"
      : "light";
  }

  return mode;
}

function applyToDocument(mode: ThemeMode, preset: ThemePreset) {
  if (!canUseDOM()) {
    resolvedColorMode.value = resolveMode(mode);
    return;
  }

  const root = document.documentElement;
  const resolved = resolveMode(mode);

  resolvedColorMode.value = resolved;
  root.classList.toggle("dark", resolved === "dark");
  root.classList.toggle("light", resolved === "light");
  root.dataset.themePreset = preset;
  root.style.colorScheme = resolved;
}

function persistAppearance(mode: ThemeMode, preset: ThemePreset) {
  if (!canUseDOM()) {
    return;
  }

  window.localStorage.setItem(COLOR_MODE_STORAGE_KEY, mode);
  window.localStorage.setItem(THEME_PRESET_STORAGE_KEY, preset);
}

function setActiveAppearance(mode: ThemeMode, preset: ThemePreset) {
  colorMode.value = normalizeColorMode(mode);
  themePreset.value = normalizeThemePreset(preset);
  applyToDocument(colorMode.value, themePreset.value);
}

function setPersistedAppearance(mode: ThemeMode, preset: ThemePreset) {
  persistedColorMode.value = normalizeColorMode(mode);
  persistedThemePreset.value = normalizeThemePreset(preset);
  persistAppearance(persistedColorMode.value, persistedThemePreset.value);
}

function ensureSystemListener() {
  if (!canUseDOM() || mediaListenerAttached) {
    return;
  }

  mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
  const handleChange = () => {
    if (colorMode.value === "system") {
      applyToDocument(colorMode.value, themePreset.value);
    }
  };

  if (typeof mediaQuery.addEventListener === "function") {
    mediaQuery.addEventListener("change", handleChange);
  } else if (typeof mediaQuery.addListener === "function") {
    mediaQuery.addListener(handleChange);
  }

  mediaListenerAttached = true;
}

function initializeAppearance() {
  if (isInitialized) {
    return;
  }

  const stored = getStoredAppearance();
  setPersistedAppearance(stored.theme_mode, stored.theme_preset);
  setActiveAppearance(stored.theme_mode, stored.theme_preset);
  ensureSystemListener();
  isInitialized = true;
}

export function useColorMode() {
  onMounted(() => {
    initializeAppearance();
  });

  function previewAppearance(
    mode: ThemeMode = colorMode.value,
    preset: ThemePreset = themePreset.value,
  ) {
    setActiveAppearance(mode, preset);
  }

  function commitAppearance(
    mode: ThemeMode = colorMode.value,
    preset: ThemePreset = themePreset.value,
  ) {
    const nextMode = normalizeColorMode(mode);
    const nextPreset = normalizeThemePreset(preset);
    setPersistedAppearance(nextMode, nextPreset);
    setActiveAppearance(nextMode, nextPreset);
  }

  function restorePersistedAppearance() {
    setActiveAppearance(persistedColorMode.value, persistedThemePreset.value);
  }

  function hydrateFromUserSettings(settings?: Partial<UserSettings> | null) {
    const appearance = getAppearanceFromSettings(settings);
    commitAppearance(appearance.theme_mode, appearance.theme_preset);
  }

  function setColorMode(mode: ThemeMode) {
    commitAppearance(mode, themePreset.value);
  }

  function setThemePreset(preset: ThemePreset) {
    commitAppearance(colorMode.value, preset);
  }

  return {
    colorMode,
    themePreset,
    persistedColorMode,
    persistedThemePreset,
    resolvedColorMode,
    isDark,
    hasUnsavedAppearance,
    setColorMode,
    setThemePreset,
    previewAppearance,
    commitAppearance,
    restorePersistedAppearance,
    hydrateFromUserSettings,
    initializeAppearance,
  };
}

export type { ThemeMode, ThemePreset, AppearanceSettings, ResolvedColorMode };
