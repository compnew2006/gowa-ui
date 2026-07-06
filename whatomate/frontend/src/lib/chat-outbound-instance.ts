import type { Contact, Message } from "@/stores/contacts";

function normalizeInstanceID(value?: string | null): string | undefined {
  const instanceID = typeof value === "string" ? value.trim() : "";
  return instanceID || undefined;
}

function resolveLatestMessageInstanceID(
  messages: readonly Message[],
  direction?: Message["direction"],
): string | undefined {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index];
    if (direction && message.direction !== direction) {
      continue;
    }

    const instanceID = normalizeInstanceID(message.instance_id);
    if (instanceID) {
      return instanceID;
    }
  }

  return undefined;
}

export function resolvePreferredOutboundInstanceID(options: {
  messages: readonly Message[];
  selectedSourceContact?: Contact | null;
  currentContact?: Contact | null;
  selectedInstanceID?: string | null;
}): string | undefined {
  const explicitSourceInstanceID = normalizeInstanceID(
    options.selectedSourceContact?.instance_id,
  );
  if (explicitSourceInstanceID) {
    return explicitSourceInstanceID;
  }

  const latestInboundInstanceID = resolveLatestMessageInstanceID(
    options.messages,
    "incoming",
  );
  if (latestInboundInstanceID) {
    return latestInboundInstanceID;
  }

  const currentContactInstanceID = normalizeInstanceID(
    options.currentContact?.instance_id,
  );
  if (currentContactInstanceID) {
    return currentContactInstanceID;
  }

  const latestThreadInstanceID = resolveLatestMessageInstanceID(options.messages);
  if (latestThreadInstanceID) {
    return latestThreadInstanceID;
  }

  return normalizeInstanceID(options.selectedInstanceID);
}
