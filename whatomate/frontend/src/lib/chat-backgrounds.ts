import type { CSSProperties } from "vue";
import { api } from "@/services/api";
import auroraVeilUrl from "@/assets/chat-backgrounds/aurora-veil.svg";
import sunsetDunesUrl from "@/assets/chat-backgrounds/sunset-dunes.svg";
import paperGardenUrl from "@/assets/chat-backgrounds/paper-garden.svg";
import linenGridUrl from "@/assets/chat-backgrounds/linen-grid.svg";
import dotOrbitUrl from "@/assets/chat-backgrounds/dot-orbit.svg";
import rippleLinesUrl from "@/assets/chat-backgrounds/ripple-lines.svg";
import type { ChatBackgroundSettings } from "@/types/auth";

export type ChatBackgroundPresetCategory = "image" | "pattern";
export type ChatBackgroundEditorMode =
  | "default"
  | "images"
  | "patterns"
  | "upload";
export type ChatBackgroundTheme = "light" | "dark";
export type ChatBackgroundStyleVariant = "chat" | "preview";

export interface ChatBackgroundPreset {
  id: string;
  category: ChatBackgroundPresetCategory;
  labelKey: string;
  descriptionKey: string;
  assetUrl: string;
}

export interface ChatBackgroundFileValidation {
  valid: boolean;
  errorKey?: string;
}

export const CHAT_BACKGROUND_UPLOAD_MAX_BYTES = 5 * 1024 * 1024;
export const CHAT_BACKGROUND_UPLOAD_ACCEPT = "image/jpeg,image/png,image/webp";
export const CHAT_BACKGROUND_UPLOAD_MIME_TYPES = [
  "image/jpeg",
  "image/png",
  "image/webp",
] as const;
export type ChatBackgroundUploadMimeType =
  (typeof CHAT_BACKGROUND_UPLOAD_MIME_TYPES)[number];

export const CHAT_BACKGROUND_PRESETS: readonly ChatBackgroundPreset[] = [
  {
    id: "aurora-veil",
    category: "image",
    labelKey: "settings.chatBackgroundPresetAuroraVeil",
    descriptionKey: "settings.chatBackgroundPresetAuroraVeilDesc",
    assetUrl: auroraVeilUrl,
  },
  {
    id: "sunset-dunes",
    category: "image",
    labelKey: "settings.chatBackgroundPresetSunsetDunes",
    descriptionKey: "settings.chatBackgroundPresetSunsetDunesDesc",
    assetUrl: sunsetDunesUrl,
  },
  {
    id: "paper-garden",
    category: "image",
    labelKey: "settings.chatBackgroundPresetPaperGarden",
    descriptionKey: "settings.chatBackgroundPresetPaperGardenDesc",
    assetUrl: paperGardenUrl,
  },
  {
    id: "linen-grid",
    category: "pattern",
    labelKey: "settings.chatBackgroundPresetLinenGrid",
    descriptionKey: "settings.chatBackgroundPresetLinenGridDesc",
    assetUrl: linenGridUrl,
  },
  {
    id: "dot-orbit",
    category: "pattern",
    labelKey: "settings.chatBackgroundPresetDotOrbit",
    descriptionKey: "settings.chatBackgroundPresetDotOrbitDesc",
    assetUrl: dotOrbitUrl,
  },
  {
    id: "ripple-lines",
    category: "pattern",
    labelKey: "settings.chatBackgroundPresetRippleLines",
    descriptionKey: "settings.chatBackgroundPresetRippleLinesDesc",
    assetUrl: rippleLinesUrl,
  },
];

const CHAT_BACKGROUND_PRESET_MAP = new Map(
  CHAT_BACKGROUND_PRESETS.map((preset) => [preset.id, preset]),
);

function trimmedString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function isSupportedChatBackgroundMimeType(
  value: string,
): value is ChatBackgroundUploadMimeType {
  return (CHAT_BACKGROUND_UPLOAD_MIME_TYPES as readonly string[]).includes(
    value,
  );
}

function normalizeCustomChatBackgroundURL(
  chatBackground: ChatBackgroundSettings,
): string | null {
  const assetID = trimmedString(chatBackground.custom_asset_id);
  if (assetID === "") {
    return null;
  }

  const baseURL = String(api.defaults.baseURL ?? "/api").replace(/\/+$/, "");
  return `${baseURL}/me/chat-background?asset=${encodeURIComponent(assetID)}`;
}

function buildDefaultLayers(theme: ChatBackgroundTheme): {
  backgroundColor: string;
  layers: string[];
} {
  if (theme === "dark") {
    return {
      backgroundColor: "rgb(var(--background))",
      layers: [
        "radial-gradient(circle at top, rgb(var(--primary) / 0.12), transparent 24%)",
        "linear-gradient(180deg, rgb(var(--background)), rgb(var(--card)))",
      ],
    };
  }

  return {
    backgroundColor: "rgb(var(--background))",
    layers: [
      "radial-gradient(circle at top, rgb(var(--primary) / 0.08), transparent 28%)",
      "linear-gradient(180deg, rgb(var(--background)), rgb(var(--muted) / 0.42))",
    ],
  };
}

function buildOverlayLayer(
  theme: ChatBackgroundTheme,
  category: ChatBackgroundPresetCategory,
  variant: ChatBackgroundStyleVariant,
): string {
  if (category === "pattern") {
    if (theme === "dark") {
      return variant === "preview"
        ? "linear-gradient(180deg, rgba(15, 23, 42, 0.38), rgba(15, 23, 42, 0.56))"
        : "linear-gradient(180deg, rgba(15, 23, 42, 0.68), rgba(15, 23, 42, 0.82))";
    }
    return variant === "preview"
      ? "linear-gradient(180deg, rgba(255, 250, 244, 0.36), rgba(255, 255, 255, 0.52))"
      : "linear-gradient(180deg, rgba(255, 250, 244, 0.78), rgba(255, 255, 255, 0.9))";
  }

  if (theme === "dark") {
    return variant === "preview"
      ? "linear-gradient(180deg, rgba(15, 23, 42, 0.24), rgba(15, 23, 42, 0.44))"
      : "linear-gradient(180deg, rgba(15, 23, 42, 0.5), rgba(15, 23, 42, 0.72))";
  }

  return variant === "preview"
    ? "linear-gradient(180deg, rgba(252, 247, 240, 0.2), rgba(255, 255, 255, 0.34))"
    : "linear-gradient(180deg, rgba(252, 247, 240, 0.62), rgba(255, 255, 255, 0.82))";
}

export function resolveChatBackgroundAssetStyle(
  assetUrl: string,
  category: ChatBackgroundPresetCategory,
  theme: ChatBackgroundTheme,
  variant: ChatBackgroundStyleVariant,
): CSSProperties {
  const defaults = buildDefaultLayers(theme);
  const patternSize = variant === "preview" ? "220px 220px" : "360px 360px";
  const backgroundImage = [
    buildOverlayLayer(theme, category, variant),
    `url("${assetUrl}")`,
    ...defaults.layers,
  ].join(", ");

  return {
    backgroundColor: defaults.backgroundColor,
    backgroundImage,
    backgroundPosition: "center, center, top center, center",
    backgroundRepeat:
      category === "pattern"
        ? "no-repeat, repeat, no-repeat, no-repeat"
        : "no-repeat, no-repeat, no-repeat, no-repeat",
    backgroundSize:
      category === "pattern"
        ? `cover, ${patternSize}, auto, auto`
        : "cover, cover, auto, auto",
  };
}

export function getChatBackgroundPreset(
  presetID: string | null | undefined,
): ChatBackgroundPreset | undefined {
  if (!presetID) {
    return undefined;
  }
  return CHAT_BACKGROUND_PRESET_MAP.get(presetID);
}

export function normalizeChatBackgroundPreference(
  value: unknown,
): ChatBackgroundSettings | null {
  if (!value || typeof value !== "object") {
    return null;
  }

  const input = value as Record<string, unknown>;
  const kind = trimmedString(input.kind).toLowerCase();

  if (kind === "preset") {
    const presetID = trimmedString(input.preset_id);
    if (!getChatBackgroundPreset(presetID)) {
      return null;
    }
    return {
      kind: "preset",
      preset_id: presetID,
    };
  }

  if (kind === "custom") {
    const assetID = trimmedString(input.custom_asset_id);
    const filename = trimmedString(input.custom_filename);
    const mimeType = trimmedString(input.custom_mime_type).toLowerCase();
    if (assetID === "" || !isSupportedChatBackgroundMimeType(mimeType)) {
      return null;
    }
    return {
      kind: "custom",
      custom_asset_id: assetID,
      custom_filename: filename || undefined,
      custom_mime_type: mimeType,
    };
  }

  return null;
}

export function resolveChatBackgroundEditorMode(
  value: unknown,
): ChatBackgroundEditorMode {
  const normalized = normalizeChatBackgroundPreference(value);
  if (normalized?.kind === "custom") {
    return "upload";
  }
  if (normalized?.kind === "preset") {
    return getChatBackgroundPreset(normalized.preset_id)?.category === "pattern"
      ? "patterns"
      : "images";
  }
  return "default";
}

export function resolveChatBackgroundStyle(
  value: unknown,
  options: {
    theme?: ChatBackgroundTheme;
    variant?: ChatBackgroundStyleVariant;
  } = {},
): CSSProperties {
  const theme = options.theme ?? "light";
  const variant = options.variant ?? "chat";
  const normalized = normalizeChatBackgroundPreference(value);

  if (!normalized) {
    const defaults = buildDefaultLayers(theme);
    return {
      backgroundColor: defaults.backgroundColor,
      backgroundImage: defaults.layers.join(", "),
      backgroundPosition: "top center, center",
      backgroundRepeat: "no-repeat, no-repeat",
      backgroundSize: "auto, auto",
    };
  }

  if (normalized.kind === "preset") {
    const preset = getChatBackgroundPreset(normalized.preset_id);
    if (!preset) {
      const defaults = buildDefaultLayers(theme);
      return {
        backgroundColor: defaults.backgroundColor,
        backgroundImage: defaults.layers.join(", "),
        backgroundPosition: "top center, center",
        backgroundRepeat: "no-repeat, no-repeat",
        backgroundSize: "auto, auto",
      };
    }
    return resolveChatBackgroundAssetStyle(
      preset.assetUrl,
      preset.category,
      theme,
      variant,
    );
  }

  const customUrl = normalizeCustomChatBackgroundURL(normalized);
  if (!customUrl) {
    const defaults = buildDefaultLayers(theme);
    return {
      backgroundColor: defaults.backgroundColor,
      backgroundImage: defaults.layers.join(", "),
      backgroundPosition: "top center, center",
      backgroundRepeat: "no-repeat, no-repeat",
      backgroundSize: "auto, auto",
    };
  }

  return resolveChatBackgroundAssetStyle(customUrl, "image", theme, variant);
}

export function resolvePreviewBackgroundStyle(
  assetUrl: string,
  category: ChatBackgroundPresetCategory,
  theme: ChatBackgroundTheme = "light",
): CSSProperties {
  return resolveChatBackgroundAssetStyle(assetUrl, category, theme, "preview");
}

export function validateChatBackgroundFile(
  file: File,
): ChatBackgroundFileValidation {
  if (!isSupportedChatBackgroundMimeType(file.type)) {
    return {
      valid: false,
      errorKey: "settings.chatBackgroundUploadInvalidType",
    };
  }

  if (file.size > CHAT_BACKGROUND_UPLOAD_MAX_BYTES) {
    return {
      valid: false,
      errorKey: "settings.chatBackgroundUploadTooLarge",
    };
  }

  return { valid: true };
}

export function isSameChatBackgroundPreference(
  left: ChatBackgroundSettings | null,
  right: ChatBackgroundSettings | null,
): boolean {
  if (left === right) {
    return true;
  }
  if (!left || !right) {
    return false;
  }
  if (left.kind !== right.kind) {
    return false;
  }
  if (left.kind === "preset" && right.kind === "preset") {
    return left.preset_id === right.preset_id;
  }
  return (
    left.custom_asset_id === right.custom_asset_id &&
    left.custom_filename === right.custom_filename &&
    left.custom_mime_type === right.custom_mime_type
  );
}
