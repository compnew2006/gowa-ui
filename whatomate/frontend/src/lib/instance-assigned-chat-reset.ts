export type InstanceAssignedChatResetMode = "midnight" | "custom_hour";

export interface InstanceAssignedChatResetSettings {
  enabled: boolean;
  mode: InstanceAssignedChatResetMode;
  hour: number;
}

export const DEFAULT_INSTANCE_ASSIGNED_CHAT_RESET_SETTINGS: InstanceAssignedChatResetSettings =
  {
    enabled: true,
    mode: "midnight",
    hour: 0,
  };

export function normalizeInstanceAssignedChatResetMode(
  value: unknown,
): InstanceAssignedChatResetMode {
  return value === "custom_hour" ? "custom_hour" : "midnight";
}

export function normalizeInstanceAssignedChatResetHour(value: unknown): number {
  const parsed =
    typeof value === "number"
      ? value
      : typeof value === "string"
        ? Number(value)
        : Number.NaN;

  if (!Number.isFinite(parsed)) {
    return 0;
  }

  const rounded = Math.trunc(parsed);
  return Math.min(23, Math.max(0, rounded));
}

export function normalizeInstanceAssignedChatResetSettings(
  raw: Record<string, unknown> | null | undefined,
): InstanceAssignedChatResetSettings {
  const mode = normalizeInstanceAssignedChatResetMode(
    raw?.assigned_chat_reset_mode,
  );
  const hour =
    mode === "midnight"
      ? 0
      : normalizeInstanceAssignedChatResetHour(raw?.assigned_chat_reset_hour);

  return {
    enabled:
      typeof raw?.assigned_chat_reset_enabled === "boolean"
        ? raw.assigned_chat_reset_enabled
        : DEFAULT_INSTANCE_ASSIGNED_CHAT_RESET_SETTINGS.enabled,
    mode,
    hour,
  };
}

export function cloneInstanceAssignedChatResetSettings(
  current: InstanceAssignedChatResetSettings,
): InstanceAssignedChatResetSettings {
  return {
    enabled: current.enabled,
    mode: current.mode,
    hour: current.hour,
  };
}

export function sanitizeInstanceAssignedChatResetSettings(
  current: InstanceAssignedChatResetSettings,
): InstanceAssignedChatResetSettings {
  const mode = normalizeInstanceAssignedChatResetMode(current.mode);
  return {
    enabled: current.enabled !== false,
    mode,
    hour:
      mode === "midnight"
        ? 0
        : normalizeInstanceAssignedChatResetHour(current.hour),
  };
}
