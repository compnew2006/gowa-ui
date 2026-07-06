export function normalizeAllowedInstanceIDs(values: unknown): string[] {
  if (!Array.isArray(values)) return [];

  return Array.from(
    new Set(
      values
        .map((value) => (typeof value === "string" ? value.trim() : ""))
        .filter(Boolean),
    ),
  );
}

export function getAllowedInstanceIDsFromSettings(settings: unknown): string[] {
  if (!settings || typeof settings !== "object") return [];

  const sendRestrictions = (settings as Record<string, unknown>)
    .send_restrictions;
  if (!sendRestrictions || typeof sendRestrictions !== "object") return [];

  const raw = sendRestrictions as Record<string, unknown>;
  const fromArray = normalizeAllowedInstanceIDs(raw.allowed_instance_ids);
  if (fromArray.length > 0) {
    return fromArray;
  }

  const legacy =
    typeof raw.allowed_instance_id === "string"
      ? raw.allowed_instance_id.trim()
      : "";
  return legacy ? [legacy] : [];
}

export function canAccessInstance(
  settings: unknown,
  instanceId?: string | null,
): boolean {
  const normalizedInstanceId =
    typeof instanceId === "string" ? instanceId.trim() : "";
  if (!normalizedInstanceId) return true;

  const allowedInstanceIDs = getAllowedInstanceIDsFromSettings(settings);
  if (allowedInstanceIDs.length === 0) return true;

  return allowedInstanceIDs.includes(normalizedInstanceId);
}

export function canUserAccessInstance(
  user: { settings?: unknown } | null | undefined,
  instanceId?: string | null,
): boolean {
  return canAccessInstance(user?.settings, instanceId);
}
