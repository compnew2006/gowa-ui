import type {
  ChatStatus,
  Contact,
  Message,
} from "@/types/contacts";

export type { ChatStatus, Contact, Message };

export interface AddMessageOptions {
  appendToActiveThread?: boolean;
}

export interface ContactsListPayload {
  contacts: Contact[];
  total?: number;
  page?: number;
  limit?: number;
}

export interface MessagesListPayload {
  messages: Message[];
  total?: number;
  page?: number;
  limit?: number;
  has_more?: boolean;
  next_cursor?: string;
  prev_cursor?: string;
}

export interface RecentContactFetch {
  at: number;
  cooldownMs: number;
  result: Contact | null;
}

export const unsupportedMessageBody = "[Unsupported message type]";
export const deletedMessageBody = "(This message was deleted)";
export const legacyDeletedMessageBody = "This message was deleted";
export const syntheticPlaceholderCompanionWindowMs = 3000;
export const contactFetchCooldownMs = 1500;
export const missingContactFetchCooldownMs = 30000;

export function normalizeChatStatus(
  rawStatus: unknown,
  assignedUserID?: string,
): ChatStatus {
  const normalized =
    typeof rawStatus === "string" ? rawStatus.trim().toLowerCase() : "";
  if (normalized === "closed") return "closed";
  if (normalized === "open") return "open";
  if (normalized === "pending") return assignedUserID ? "open" : "pending";
  return assignedUserID ? "open" : "pending";
}

export function normalizeContact(contact: Contact): Contact {
  return {
    ...contact,
    is_public: contact.is_public === true,
    is_collaborator: contact.is_collaborator === true,
    status: normalizeChatStatus(contact.status, contact.assigned_user_id),
  };
}

export function normalizeContacts(contacts: Contact[]): Contact[] {
  return contacts.map(normalizeContact);
}

export function normalizeSearchText(value: unknown): string {
  return typeof value === "string" ? value.trim().toLowerCase() : "";
}

export function normalizeDigits(value: string): string {
  let normalized = "";

  for (const char of value) {
    const code = char.charCodeAt(0);

    if (code >= 0x30 && code <= 0x39) {
      normalized += char;
      continue;
    }

    if (code >= 0x0660 && code <= 0x0669) {
      normalized += String(code - 0x0660);
      continue;
    }

    if (code >= 0x06f0 && code <= 0x06f9) {
      normalized += String(code - 0x06f0);
    }
  }

  return normalized;
}

export function contactMatchesSearch(contact: Contact, rawQuery: string): boolean {
  const query = normalizeSearchText(rawQuery);
  if (!query) return true;

  const name = normalizeSearchText(contact.name);
  const profileName = normalizeSearchText(contact.profile_name);
  const phoneNumber = normalizeSearchText(contact.phone_number);

  if (
    name.includes(query) ||
    profileName.includes(query) ||
    phoneNumber.includes(query)
  ) {
    return true;
  }

  const queryDigits = normalizeDigits(rawQuery);
  if (!queryDigits) {
    return false;
  }

  return normalizeDigits(contact.phone_number || "").includes(queryDigits);
}

export function extractAllowedInstanceIDsFromUserSettings(
  settings: unknown,
): string[] {
  if (!settings || typeof settings !== "object") return [];

  const sendRestrictions = (settings as Record<string, unknown>)
    .send_restrictions;
  if (!sendRestrictions || typeof sendRestrictions !== "object") return [];

  const raw = sendRestrictions as Record<string, unknown>;
  const allowedInstanceIDs = raw.allowed_instance_ids;
  if (Array.isArray(allowedInstanceIDs)) {
    return Array.from(
      new Set(
        allowedInstanceIDs
          .map((value) => (typeof value === "string" ? value.trim() : ""))
          .filter(Boolean),
      ),
    );
  }

  const allowedInstanceID = raw.allowed_instance_id;
  if (typeof allowedInstanceID !== "string") return [];

  const trimmed = allowedInstanceID.trim();
  return trimmed ? [trimmed] : [];
}

export function getMessageBody(message: Message): string {
  if (typeof message.content === "string") {
    return message.content;
  }
  const content = message.content as Record<string, unknown> | undefined;
  return typeof content?.body === "string" ? content.body : "";
}

export function isPlaceholderMessageBody(body: string): boolean {
  const normalized = body.trim();
  return (
    normalized === unsupportedMessageBody ||
    normalized === deletedMessageBody ||
    normalized.toLowerCase() === legacyDeletedMessageBody.toLowerCase()
  );
}

export function isSyntheticPlaceholderMessage(message: Message): boolean {
  if (message.message_type !== "text") {
    return false;
  }
  if (message.metadata?.revoked === true) {
    return false;
  }
  return isPlaceholderMessageBody(getMessageBody(message));
}

export function isUnsupportedPlaceholderMessage(message: Message): boolean {
  return (
    isSyntheticPlaceholderMessage(message) &&
    getMessageBody(message).trim() === unsupportedMessageBody
  );
}

export function isGroupMessage(message: Message): boolean {
  return (
    message.is_group_chat === true || message.metadata?.is_group_chat === true
  );
}

export function isMediaLikeMessage(message: Message): boolean {
  const messageType = (message.message_type || "").toLowerCase();
  return (
    messageType === "image" ||
    messageType === "video" ||
    messageType === "audio" ||
    messageType === "document" ||
    messageType === "sticker"
  );
}

export function getMessageSenderPhone(message: Message): string {
  if (
    typeof message.sender_phone === "string" &&
    message.sender_phone.trim() !== ""
  ) {
    return message.sender_phone.trim();
  }
  if (
    typeof message.metadata?.sender_phone === "string" &&
    message.metadata.sender_phone.trim() !== ""
  ) {
    return message.metadata.sender_phone.trim();
  }
  return "";
}

export function getMessageTimestamp(message: Message): number {
  if (typeof message.created_at !== "string") return Number.NaN;
  const parsed = Date.parse(message.created_at);
  return Number.isNaN(parsed) ? Number.NaN : parsed;
}

export function collectNearbyMediaCompanionPlaceholderIDs(
  messageList: Message[],
): Set<string> {
  const ids = new Set<string>();
  for (let i = 0; i < messageList.length; i++) {
    const candidate = messageList[i];
    if (
      !candidate?.id ||
      !isUnsupportedPlaceholderMessage(candidate) ||
      !isGroupMessage(candidate)
    ) {
      continue;
    }

    const candidateSender = getMessageSenderPhone(candidate);
    if (candidateSender === "") {
      continue;
    }

    const candidateTimestamp = getMessageTimestamp(candidate);
    if (!Number.isFinite(candidateTimestamp)) {
      continue;
    }

    for (let j = i + 1; j < messageList.length; j++) {
      const next = messageList[j];
      if (!next) continue;

      const nextTimestamp = getMessageTimestamp(next);
      if (!Number.isFinite(nextTimestamp)) {
        continue;
      }

      if (
        nextTimestamp - candidateTimestamp >
        syntheticPlaceholderCompanionWindowMs
      ) {
        break;
      }

      if (!isMediaLikeMessage(next)) {
        continue;
      }

      if (
        next.contact_id !== candidate.contact_id ||
        next.direction !== candidate.direction
      ) {
        continue;
      }

      if (getMessageSenderPhone(next) !== candidateSender) {
        continue;
      }

      ids.add(candidate.id);
      break;
    }
  }
  return ids;
}

export function removeSyntheticPlaceholderMessages(messageList: Message[]): Message[] {
  const companionWamids = new Set(
    messageList
      .filter(
        (message) =>
          !isSyntheticPlaceholderMessage(message) &&
          typeof message.wamid === "string" &&
          message.wamid.trim() !== "",
      )
      .map((message) => message.wamid!.trim()),
  );
  const nearbyMediaCompanionPlaceholderIDs =
    collectNearbyMediaCompanionPlaceholderIDs(messageList);

  if (
    companionWamids.size === 0 &&
    nearbyMediaCompanionPlaceholderIDs.size === 0
  ) {
    return messageList;
  }

  return messageList.filter((message) => {
    if (message?.id && nearbyMediaCompanionPlaceholderIDs.has(message.id)) {
      return false;
    }
    const wamid = typeof message.wamid === "string" ? message.wamid.trim() : "";
    if (wamid === "" || !companionWamids.has(wamid)) {
      return true;
    }
    return !isSyntheticPlaceholderMessage(message);
  });
}
