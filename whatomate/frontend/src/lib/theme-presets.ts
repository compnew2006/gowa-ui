import type { ThemeMode, ThemePreset, UserSettings } from "@/types/auth";

export const COLOR_MODE_STORAGE_KEY = "color-mode";
export const THEME_PRESET_STORAGE_KEY = "theme-preset";

export const DEFAULT_COLOR_MODE: ThemeMode = "system";
export const DEFAULT_THEME_PRESET: ThemePreset = "twitter";

export type ResolvedColorMode = "light" | "dark";

export interface ThemePresetOption {
  id: ThemePreset;
  labelKey: string;
  descriptionKey: string;
  previewBackground: string;
  previewAccent: string;
  previewForeground: string;
}

export const THEME_PRESET_OPTIONS: ThemePresetOption[] = [
  {
    id: "twitter",
    labelKey: "settings.themePresetTwitter",
    descriptionKey: "settings.themePresetTwitterDesc",
    previewBackground:
      "linear-gradient(135deg, rgb(240 248 255) 0%, rgb(247 248 248) 100%)",
    previewAccent: "rgb(30 157 241)",
    previewForeground: "rgb(15 20 25)",
  },
  {
    id: "ocean-breeze",
    labelKey: "settings.themePresetOceanBreeze",
    descriptionKey: "settings.themePresetOceanBreezeDesc",
    previewBackground:
      "linear-gradient(135deg, rgb(224 242 254) 0%, rgb(209 250 229) 100%)",
    previewAccent: "rgb(34 197 94)",
    previewForeground: "rgb(55 65 81)",
  },
  {
    id: "soft-pop",
    labelKey: "settings.themePresetSoftPop",
    descriptionKey: "settings.themePresetSoftPopDesc",
    previewBackground:
      "linear-gradient(135deg, rgb(247 249 243) 0%, rgb(255 255 255) 100%)",
    previewAccent: "rgb(79 70 229)",
    previewForeground: "rgb(0 0 0)",
  },
  {
    id: "amber-minimal",
    labelKey: "settings.themePresetAmberMinimal",
    descriptionKey: "settings.themePresetAmberMinimalDesc",
    previewBackground:
      "linear-gradient(135deg, rgb(255 251 235) 0%, rgb(249 250 251) 100%)",
    previewAccent: "rgb(245 158 11)",
    previewForeground: "rgb(38 38 38)",
  },
];

export interface AppearanceSettings {
  theme_mode: ThemeMode;
  theme_preset: ThemePreset;
}

export function normalizeColorMode(value: unknown): ThemeMode {
  if (value === "light" || value === "dark" || value === "system") {
    return value;
  }
  return DEFAULT_COLOR_MODE;
}

export function normalizeThemePreset(value: unknown): ThemePreset {
  if (value === "caffeine") {
    return "soft-pop";
  }

  if (
    value === "amber-minimal" ||
    value === "ocean-breeze" ||
    value === "twitter" ||
    value === "soft-pop"
  ) {
    return value;
  }
  return DEFAULT_THEME_PRESET;
}

export function getAppearanceFromSettings(
  settings?: Partial<UserSettings> | null,
): AppearanceSettings {
  return {
    theme_mode: normalizeColorMode(settings?.theme_mode),
    theme_preset: normalizeThemePreset(settings?.theme_preset),
  };
}

export function getStoredAppearance(): AppearanceSettings {
  if (typeof window === "undefined") {
    return {
      theme_mode: DEFAULT_COLOR_MODE,
      theme_preset: DEFAULT_THEME_PRESET,
    };
  }

  return {
    theme_mode: normalizeColorMode(
      window.localStorage.getItem(COLOR_MODE_STORAGE_KEY),
    ),
    theme_preset: normalizeThemePreset(
      window.localStorage.getItem(THEME_PRESET_STORAGE_KEY),
    ),
  };
}
