import { ref, type ComputedRef } from "vue";
import { getMessageSenderPhone } from "@/lib/group-chat";
import { MentionContactResolver } from "@/lib/mention-contact-resolver";
import type { Contact, Message } from "@/stores/contacts";

const deletedMessageText = "(This message was deleted)";
const legacyDeletedMessageText = "This message was deleted";

interface LocationData {
  latitude: number;
  longitude: number;
  name?: string;
  address?: string;
}

interface ContactData {
  name: string;
  phones?: string[];
}

interface CTAUrlData {
  type: "cta_url";
  body: string;
  button_text: string;
  url: string;
}

function extractBody(content: unknown): string {
  if (typeof content === "string") return content;
  if (content && typeof content === "object" && "body" in content) {
    return String((content as Record<string, unknown>).body ?? "");
  }
  return "";
}

export function useMessageContent(
  contacts: ComputedRef<Contact[]>,
  pendingChats: ComputedRef<Contact[]>,
  assignedChats: ComputedRef<Contact[]>,
  closedChats: ComputedRef<Contact[]>,
  messages: ComputedRef<Message[]>,
  currentContact: ComputedRef<Contact | null>,
  isCurrentGroupChat: ComputedRef<boolean>,
) {
  const mentionContactResolver = new MentionContactResolver();
  const mentionResolutionVersion = ref(0);

  function preloadMentionResolverFromKnownContacts(): void {
    const changed = mentionContactResolver.preloadContacts([
      ...contacts.value,
      ...pendingChats.value,
      ...assignedChats.value,
      ...closedChats.value,
    ]);

    if (changed) {
      mentionResolutionVersion.value += 1;
    }
  }

  function applyMentionDisplayNames(content: string): string {
    if (!content || !content.includes("@")) {
      return content;
    }

    const revision = mentionResolutionVersion.value;
    if (revision < 0) {
      return content;
    }

    return mentionContactResolver.replaceMentions(content);
  }

  async function resolveMentionsForCurrentMessages(): Promise<void> {
    preloadMentionResolverFromKnownContacts();

    const texts: string[] = [];
    for (const message of messages.value) {
      const raw = getMessageContentRaw(message);
      if (raw && raw.includes("@")) {
        texts.push(raw);
      }
    }

    if (texts.length === 0) {
      return;
    }

    const changed = await mentionContactResolver.resolveMentionsInTexts(texts);
    if (changed) {
      mentionResolutionVersion.value += 1;
    }
  }

  function normalizeDeletedMessageText(content: string): string {
    if (content.trim().toLowerCase() === legacyDeletedMessageText.toLowerCase()) {
      return deletedMessageText;
    }
    return content;
  }

  function isDeletedMessage(message: Message): boolean {
    if (message.content && typeof message.content === "object") {
      const metadata = (message.content as { metadata?: Record<string, any> })
        .metadata;
      if (metadata?.revoked === true) {
        return true;
      }
    }

    const body = getMessageContent(message).trim();
    if (!body) {
      return false;
    }

    return (
      body.includes(deletedMessageText) || body.includes(legacyDeletedMessageText)
    );
  }

  function isSystemEventMessage(message: Message): boolean {
    const rawValue = message.metadata?.system_event;
    return (
      rawValue === true ||
      rawValue === "true" ||
      rawValue === 1 ||
      rawValue === "1"
    );
  }

  function isGroupMessage(message: Message): boolean {
    if (message.is_group_chat === true) {
      return true;
    }
    if (
      typeof message.conversation_id === "string" &&
      message.conversation_id.endsWith("@g.us")
    ) {
      return true;
    }
    return isCurrentGroupChat.value;
  }

  function shouldShowGroupSenderPhone(message: Message): boolean {
    if (message.direction !== "incoming" || !isGroupMessage(message)) {
      return false;
    }
    return getGroupSenderPhone(message) !== "";
  }

  function getGroupSenderPhone(message: Message): string {
    return getMessageSenderPhone(message);
  }

  function getMessageContentRaw(message: Message): string {
    if (message.message_type === "text") {
      return extractBody(message.content);
    }
    if (message.message_type === "button_reply") {
      return extractBody(message.content);
    }
    if (message.message_type === "interactive") {
      const body = extractBody(message.content);
      if (body) return body;
      if (message.interactive_data?.body) {
        return message.interactive_data.body;
      }
      return "[Interactive Message]";
    }
    if (
      message.message_type === "image" ||
      message.message_type === "video" ||
      message.message_type === "sticker"
    ) {
      return extractBody(message.content);
    }
    if (message.message_type === "audio") {
      return "";
    }
    if (message.message_type === "document") {
      return extractBody(message.content);
    }
    if (message.message_type === "template") {
      const body = extractBody(message.content);
      return body || "[Template Message]";
    }
    if (message.message_type === "location") {
      return "";
    }
    if (
      message.message_type === "contacts" ||
      message.message_type === "contact"
    ) {
      return "";
    }
    if (message.message_type === "unsupported") {
      return "";
    }
    return "[Message]";
  }

  function getMessageContent(message: Message): string {
    const rawContent = getMessageContentRaw(message);
    const normalizedContent = normalizeDeletedMessageText(rawContent);
    return applyMentionDisplayNames(normalizedContent);
  }

  function getLocationData(message: Message): LocationData | null {
    if (message.message_type !== "location") return null;
    try {
      const body = extractBody(message.content) || message.content;
      if (typeof body === "string") {
        return JSON.parse(body);
      }
      return body as LocationData;
    } catch {
      return null;
    }
  }

  function getContactsData(message: Message): ContactData[] {
    if (message.message_type !== "contacts" && message.message_type !== "contact")
      return [];
    try {
      const body = extractBody(message.content) || message.content;
      if (typeof body === "string") {
        return JSON.parse(body);
      }
      return body as ContactData[];
    } catch {
      return [];
    }
  }

  function getGoogleMapsUrl(location: LocationData): string {
    return `https://www.google.com/maps?q=${location.latitude},${location.longitude}`;
  }

  function getInteractiveButtons(
    message: Message,
  ): Array<{ id: string; title: string }> {
    if (!message.interactive_data) {
      return [];
    }
    if (
      message.message_type !== "interactive" &&
      message.message_type !== "template"
    ) {
      return [];
    }
    const items =
      message.interactive_data.buttons || message.interactive_data.rows;
    if (!items || !Array.isArray(items)) {
      return [];
    }
    return items.map((btn: any) => ({
      id: btn.reply?.id || btn.id || "",
      title: btn.reply?.title || btn.title || btn.text || "",
    }));
  }

  function sanitizeCTAUrl(raw: unknown): string {
    const candidate = typeof raw === "string" ? raw.trim() : "";
    if (!candidate) return "";
    try {
      const parsed = new URL(candidate);
      if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
        return "";
      }
      return parsed.toString();
    } catch {
      return "";
    }
  }

  function getCTAUrlData(message: Message): CTAUrlData | null {
    if (message.message_type !== "interactive" || !message.interactive_data) {
      return null;
    }
    if (message.interactive_data.type !== "cta_url") {
      return null;
    }
    const safeURL = sanitizeCTAUrl((message.interactive_data as any).url);
    if (!safeURL) {
      return null;
    }
    return {
      type: "cta_url",
      body: message.interactive_data.body || "",
      button_text: (message.interactive_data as any).button_text || "Open",
      url: safeURL,
    };
  }

  function isMediaMessage(message: Message): boolean {
    return ["image", "video", "audio", "document", "sticker"].includes(
      message.message_type,
    );
  }

  function shouldShowDateSeparator(index: number): boolean {
    const msgs = messages.value;
    if (index === 0) return true;

    const currentDate = new Date(msgs[index].created_at);
    const prevDate = new Date(msgs[index - 1].created_at);

    return currentDate.toDateString() !== prevDate.toDateString();
  }

  function getDateLabel(dateStr: string): string {
    const date = new Date(dateStr);
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const messageDate = new Date(
      date.getFullYear(),
      date.getMonth(),
      date.getDate(),
    );
    const diffDays = Math.floor(
      (today.getTime() - messageDate.getTime()) / 86400000,
    );

    if (diffDays === 0) {
      return "Today";
    } else if (diffDays === 1) {
      return "Yesterday";
    }
    return date.toLocaleDateString("en-US", {
      weekday: "long",
      month: "long",
      day: "numeric",
      year: "numeric",
    });
  }

  function formatMessageTime(dateStr: string) {
    const date = new Date(dateStr);
    return date.toLocaleTimeString("en-US", {
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  function getReplyAuthorLabel(message: Message | null): string {
    if (!message) {
      return "Yourself";
    }
    if (message.direction === "outgoing") {
      return "Yourself";
    }
    if (isGroupMessage(message) || isCurrentGroupChat.value) {
      const senderPhone = getGroupSenderPhone(message);
      if (senderPhone) {
        return senderPhone;
      }
    }
    if (
      message.reply_to_message &&
      message.direction === "incoming"
    ) {
      return message.reply_to_message.sender_phone ?? "Customer";
    }
    return (
      currentContact.value?.profile_name ||
      currentContact.value?.name ||
      "Customer"
    );
  }

  function getReplyingToAuthorLabel(message: Message | null): string {
    if (!message) {
      return "Yourself";
    }
    if (message.direction === "outgoing") {
      return "Yourself";
    }
    if (isGroupMessage(message) || isCurrentGroupChat.value) {
      const senderPhone = getGroupSenderPhone(message);
      if (senderPhone) {
        return senderPhone;
      }
    }
    return (
      currentContact.value?.profile_name ||
      currentContact.value?.name ||
      "Customer"
    );
  }

  function shouldShowReplyPreviewThumbnail(message: Message): boolean {
    if (!message.reply_to_message) return false;
    const replyType = message.reply_to_message.message_type;
    return replyType === "image" || replyType === "sticker" || replyType === "video";
  }

  function getReplyPreviewMediaURL(message: Message): string {
    if (!message.reply_to_message?.media_url) return "";
    return message.reply_to_message.media_url;
  }

  function getReplyPreviewContent(message: Message): string {
    if (!message.reply_to_message) return "";
    const replyMsg = message.reply_to_message;
    return extractBody(replyMsg.content) || `[${replyMsg.message_type || "Media"}]`;
  }

  return {
    mentionResolutionVersion,
    preloadMentionResolverFromKnownContacts,
    resolveMentionsForCurrentMessages,
    normalizeDeletedMessageText,
    isDeletedMessage,
    isSystemEventMessage,
    isGroupMessage,
    shouldShowGroupSenderPhone,
    getGroupSenderPhone,
    getMessageContentRaw,
    getMessageContent,
    getLocationData,
    getContactsData,
    getGoogleMapsUrl,
    getInteractiveButtons,
    getCTAUrlData,
    isMediaMessage,
    shouldShowDateSeparator,
    getDateLabel,
    formatMessageTime,
    getReplyAuthorLabel,
    getReplyingToAuthorLabel,
    shouldShowReplyPreviewThumbnail,
    getReplyPreviewMediaURL,
    getReplyPreviewContent,
    deletedMessageText,
  };
}
