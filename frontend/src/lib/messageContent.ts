import type { Message } from "@/stores/contacts";

/**
 * Pure, stateless helpers for extracting structured content out of chat
 * {@link Message} payloads. These transforms do not depend on any component
 * state and are safe to unit-test in isolation.
 *
 * Note: helpers that need to apply mention display-name resolution or
 * deleted-message normalization stay in {@link ChatView.vue} because those
 * depend on reactive component state (the shared `MentionContactResolver`
 * instance and the deleted-message text constants).
 */

/**
 * Extract a plain-text body string from a message `content` payload that may
 * be a string, an object with a `body` field, or anything else.
 */
export function getContentBody(content: unknown): string {
  if (typeof content === "string") {
    return content;
  }
  if (content && typeof content === "object" && "body" in content) {
    const body = (content as { body?: unknown }).body;
    return typeof body === "string" ? body : "";
  }
  return "";
}

/**
 * Raw (pre-normalization) text body for a message. Returns captions for media
 * messages and the appropriate fallback strings for media types that have no
 * textual body. Does NOT apply mention resolution or deleted-message text
 * normalization — see {@link getMessageContent} in ChatView.vue for that.
 */
export function getMessageContentRaw(message: Message): string {
  if (message.message_type === "text") {
    return getContentBody(message.content);
  }
  if (message.message_type === "button_reply") {
    // Button reply stores the selected button title in content
    return getContentBody(message.content);
  }
  if (message.message_type === "interactive") {
    // Interactive messages store body text in content (string) or content.body or interactive_data.body
    const body = getContentBody(message.content);
    if (body) return body;
    if (message.interactive_data?.body) {
      return message.interactive_data.body;
    }
    return "[Interactive Message]";
  }
  // For media messages, return caption if available (media is displayed inline)
  if (
    message.message_type === "image" ||
    message.message_type === "video" ||
    message.message_type === "sticker"
  ) {
    return getContentBody(message.content);
  }
  if (message.message_type === "audio") {
    return ""; // Audio doesn't have captions
  }
  if (message.message_type === "document") {
    return getContentBody(message.content);
  }
  if (message.message_type === "template") {
    // Show actual content if available (campaign messages), otherwise fallback
    return getContentBody(message.content) || "[Template Message]";
  }
  if (message.message_type === "location") {
    return ""; // Location is displayed as a map/card, not text
  }
  if (
    message.message_type === "contacts" ||
    message.message_type === "contact"
  ) {
    return ""; // Contacts are displayed as a card, not text
  }
  if (message.message_type === "unsupported") {
    return ""; // Displayed as a visual card, not text
  }
  if (message.message_type === "poll") {
    return getContentBody(message.content) || "";
  }
  // "ignore" is GOWA's marker for a message whose content has not synced
  // yet (first push, ~7s before the real text arrives). Return empty so the
  // bubble renders as a "loading" placeholder rather than the misleading
  // "[Message]" text. The patch broadcast flips it to real content shortly.
  if (
    message.message_type === "ignore" ||
    message.message_type === "" ||
    message.message_type === undefined ||
    message.message_type === null
  ) {
    return "";
  }
  return "[Message]";
}

/**
 * Extract a sanitised, absolute URL for the media of a message being replied
 * to. Returns "" when there is no usable URL.
 */
export function getReplyPreviewMediaURL(message: Message): string {
  const rawURL =
    typeof message.reply_to_message?.media_url === "string"
      ? message.reply_to_message.media_url.trim()
      : "";
  if (!rawURL) return "";

  const lower = rawURL.toLowerCase();
  if (
    lower.startsWith("http://") ||
    lower.startsWith("https://") ||
    lower.startsWith("data:") ||
    rawURL.startsWith("/")
  ) {
    return rawURL;
  }
  return "";
}

/** Whether the replied-to message should render an inline image thumbnail. */
export function shouldShowReplyPreviewThumbnail(message: Message): boolean {
  return (
    message.reply_to_message?.message_type === "image" &&
    getReplyPreviewMediaURL(message) !== ""
  );
}

export interface LocationData {
  latitude: number;
  longitude: number;
  name?: string;
  address?: string;
}

export interface ContactData {
  name: string;
  phones?: string[];
}

/** Parse location payload from a `location` message, or null if not parseable. */
export function getLocationData(message: Message): LocationData | null {
  if (message.message_type !== "location") return null;
  try {
    // Content is stored as JSON string in body
    const body = getContentBody(message.content) || message.content;
    if (typeof body === "string") {
      return JSON.parse(body);
    }
    return body as LocationData;
  } catch {
    return null;
  }
}

/** Parse contacts payload from a `contacts`/`contact` message, or [] if not parseable. */
export function getContactsData(message: Message): ContactData[] {
  if (message.message_type !== "contacts" && message.message_type !== "contact")
    return [];
  try {
    // Content is stored as JSON string in body
    const body = getContentBody(message.content) || message.content;
    if (typeof body === "string") {
      return JSON.parse(body);
    }
    return body as ContactData[];
  } catch {
    return [];
  }
}

/** Build a Google Maps URL pointing at the given location. */
export function getGoogleMapsUrl(location: LocationData): string {
  return `https://www.google.com/maps?q=${location.latitude},${location.longitude}`;
}

export interface PollData {
  question: string;
  options: string[];
  max_selections: number;
  votes: Record<string, number>;
  total_votes: number;
  selected_options: string[];
}

/** Normalize a single poll option value (string or {option_name|name|...}). */
export function normalizePollOption(raw: unknown): string {
  if (typeof raw === "string") return raw.trim();
  if (!raw || typeof raw !== "object") return "";

  const option = raw as Record<string, unknown>;
  for (const key of ["option_name", "name", "title", "text", "label"]) {
    const value = option[key];
    if (typeof value === "string" && value.trim() !== "") {
      return value.trim();
    }
  }

  const nestedReply = option.reply;
  if (nestedReply && typeof nestedReply === "object") {
    return normalizePollOption(nestedReply);
  }

  return "";
}

/** Normalize and de-duplicate a list of poll options. */
export function normalizePollOptions(raw: unknown): string[] {
  if (!Array.isArray(raw)) return [];
  const seen = new Set<string>();
  const options: string[] = [];

  for (const item of raw) {
    const option = normalizePollOption(item);
    if (!option || seen.has(option)) continue;
    seen.add(option);
    options.push(option);
  }

  return options;
}

/** Coerce a raw value into a finite integer poll max-selections count. */
export function normalizePollMaxSelections(raw: unknown): number {
  if (typeof raw === "number" && Number.isFinite(raw)) return raw;
  if (typeof raw === "string") {
    const parsed = Number.parseInt(raw, 10);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

/** Normalize a raw votes mapping into a clean Record<string, number>. */
export function normalizePollVotes(raw: unknown): Record<string, number> {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return {};
  const votes: Record<string, number> = {};
  for (const [option, count] of Object.entries(raw as Record<string, unknown>)) {
    const normalizedOption = option.trim();
    const normalizedCount =
      typeof count === "number" ? count : Number.parseInt(String(count), 10);
    if (normalizedOption && Number.isFinite(normalizedCount)) {
      votes[normalizedOption] = Math.max(0, normalizedCount);
    }
  }
  return votes;
}

/** Normalize a raw selected-options array into a trimmed, non-empty string[]. */
export function normalizePollSelectedOptions(raw: unknown): string[] {
  if (!Array.isArray(raw)) return [];
  return raw
    .filter((value): value is string => typeof value === "string")
    .map((value) => value.trim())
    .filter(Boolean);
}

/**
 * Walk a poll message's content payload to extract the embedded poll object.
 * Handles strings (JSON), `{ body: ... }` wrappers, and bare objects.
 */
export function parsePollContentData(
  content: unknown,
): Record<string, unknown> | null {
  if (!content || typeof content !== "object") {
    if (typeof content !== "string") return null;
    const trimmed = content.trim();
    if (!trimmed.startsWith("{") || !trimmed.endsWith("}")) return null;
    try {
      const parsed = JSON.parse(trimmed);
      return parsed && typeof parsed === "object"
        ? (parsed as Record<string, unknown>)
        : null;
    } catch {
      return null;
    }
  }

  if ("body" in content) {
    return parsePollContentData((content as Record<string, unknown>).body);
  }

  return content as Record<string, unknown>;
}

/** Parse a structured poll payload from a `poll` message, or null if none. */
export function getPollData(message: Message): PollData | null {
  if (message.message_type !== "poll") return null;

  const interactive = (message.interactive_data || {}) as Record<string, unknown>;
  if (interactive.type === "poll_vote") return null;

  const nestedPoll = interactive.poll;
  const d =
    nestedPoll && typeof nestedPoll === "object"
      ? (nestedPoll as Record<string, unknown>)
      : interactive;
  const contentData = parsePollContentData(message.content);

  const options = normalizePollOptions(
    d.options || d.poll_options || contentData?.options || contentData?.poll_options,
  );

  if (interactive.type && interactive.type !== "poll" && options.length === 0) {
    return null;
  }

  const rawQuestion =
    d.question || d.name || d.title || contentData?.question || contentData?.name;
  const question =
    typeof rawQuestion === "string" && rawQuestion.trim() !== ""
      ? rawQuestion.trim()
      : getContentBody(message.content) || "Poll";

  const votes = normalizePollVotes(d.votes || d.vote_counts || contentData?.votes);
  const selectedOptions = normalizePollSelectedOptions(
    d.selected_options || d.last_selected_options || contentData?.selected_options,
  );
  const totalVotes =
    normalizePollMaxSelections(d.total_votes || d.totalVotes || contentData?.total_votes) ||
    Object.values(votes).reduce((sum, count) => sum + count, 0);

  return {
    question,
    options,
    max_selections: normalizePollMaxSelections(
      d.max_selections ||
        d.maxSelections ||
        d.selectable_options_count ||
        contentData?.max_selections ||
        contentData?.selectable_options_count,
    ),
    votes,
    total_votes: totalVotes,
    selected_options: selectedOptions,
  };
}
