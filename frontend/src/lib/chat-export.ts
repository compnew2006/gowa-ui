import { chatsService } from "@/services/api";
import { unwrapResponse } from "@/lib/api-utils";
import { prefetchMediaBlob } from "@/lib/media_prefetch_cache";
import type { Message } from "@/types/contacts";

interface ExportContact {
  name: string;
  phone_number: string;
  profile_name?: string;
}

/**
 * Fetch ALL messages for a contact by iterating through paginated API.
 */
export async function fetchAllMessages(
  contactId: string,
  account?: string,
  onProgress?: (loaded: number) => void,
): Promise<Message[]> {
  const allMessages: Message[] = [];
  let beforeId: string | undefined;
  let hasMore = true;

  while (hasMore) {
    const response = await chatsService.listMessages(contactId, {
      limit: 100,
      before_id: beforeId,
      account,
    });
    const data = unwrapResponse<{ messages: Message[]; has_more?: boolean }>(response);
    const messages: Message[] = data.messages || [];
    hasMore = data.has_more === true;

    if (messages.length === 0) break;

    // API returns newest-first when using before_id (cursor pagination)
    // Prepend to maintain chronological order
    allMessages.unshift(...messages);

    // The oldest message becomes the next cursor
    beforeId = messages[0].id;

    onProgress?.(allMessages.length);
  }

  return allMessages;
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function mediaLabel(msg: Message): string {
  const t = msg.message_type;
  if (t === "image" || t === "sticker") return "[Image]";
  if (t === "video") return "[Video]";
  if (t === "audio" || t === "ptt") return "[Voice/Audio]";
  if (t === "document") return msg.media_filename ? `[Document: ${msg.media_filename}]` : "[Document]";
  if (t === "location") {
    const loc = msg.interactive_data as any;
    if (loc?.latitude && loc?.longitude) return `[Location: ${loc.latitude}, ${loc.longitude}]`;
    return "[Location]";
  }
  if (t === "contact") return "[Contact]";
  if (t === "reaction") {
    const raw = msg.content;
    const emoji = typeof raw === "string" ? raw : (raw as any)?.body || "";
    return emoji ? `[Reaction: ${emoji}]` : "[Reaction]";
  }
  return "";
}

function extractText(msg: Message): string {
  const parts: string[] = [];

  // Main content – backend returns { body: "…" } for all types
  const raw = msg.content;
  const body =
    typeof raw === "string"
      ? raw.trim()
      : (raw as any)?.body && typeof (raw as any).body === "string"
        ? (raw as any).body.trim()
        : "";
  if (body) {
    parts.push(body);
  }

  // Interactive data body
  const idata = msg.interactive_data as any;
  if (idata?.body) parts.push(idata.body);

  // Button replies
  if (idata?.buttons?.length) {
    const labels = idata.buttons.map((b: any) => b.reply?.title || b.title).filter(Boolean);
    if (labels.length) parts.push(`[${labels.join(", ")}]`);
  }

  // List rows
  if (idata?.rows?.length) {
    const labels = idata.rows.map((r: any) => r.title).filter(Boolean);
    if (labels.length) parts.push(`[${labels.join(", ")}]`);
  }

  // Media label for non-text types
  if (msg.message_type !== "text" && msg.message_type !== "interactive") {
    const ml = mediaLabel(msg);
    if (ml) parts.unshift(ml);
  }

  return parts.join(" ");
}

/**
 * Build a plain text transcript of a chat.
 */
export function buildTextExport(
  contact: ExportContact,
  messages: Message[],
): string {
  const lines: string[] = [];
  const displayName = contact.name || contact.phone_number;

  lines.push(`Chat Export: ${displayName}`);
  lines.push(`Phone: ${contact.phone_number}`);
  lines.push(`Exported: ${formatTimestamp(new Date().toISOString())}`);
  lines.push(`Messages: ${messages.length}`);
  lines.push("=".repeat(60));
  lines.push("");

  for (const msg of messages) {
    const time = formatTimestamp(msg.created_at);
    const sender =
      msg.direction === "outgoing" ? "Agent" : (msg as any).sender_push_name || contact.profile_name || displayName;
    const arrow = msg.direction === "outgoing" ? "→" : "←";
    const text = extractText(msg);

    lines.push(`[${time}] ${arrow} ${sender}: ${text}`);

    // Reactions
    if (msg.reactions?.length) {
      const rText = msg.reactions.map((r) => r.emoji).join(" ");
      lines.push(`  Reactions: ${rText}`);
    }
  }

  lines.push("");
  lines.push("=".repeat(60));
  lines.push(`End of export - ${messages.length} messages`);

  return lines.join("\n");
}

/**
 * Fetch image blobs for messages that have media, returning a map of message-id → base64 data URL.
 * Limits concurrency and catches individual failures gracefully.
 */
export async function fetchImageDataUrls(
  messages: Message[],
  maxConcurrent = 4,
  onProgress?: (loaded: number) => void,
): Promise<Record<string, string>> {
  const imageMessages = messages.filter(
    (m) =>
      (m.message_type === "image" || m.message_type === "sticker") &&
      m.media_url,
  );
  const result: Record<string, string> = {};
  let loaded = 0;

  async function fetchOne(msg: Message): Promise<void> {
    try {
      const blob = await prefetchMediaBlob(msg.id);
      if (!blob) return;
      const dataUrl = await new Promise<string>((resolve) => {
        const reader = new FileReader();
        reader.onloadend = () => resolve(reader.result as string);
        reader.onerror = () => resolve("");
        reader.readAsDataURL(blob);
      });
      if (dataUrl) result[msg.id] = dataUrl;
    } catch {
      // skip failed images
    } finally {
      loaded++;
      onProgress?.(loaded);
    }
  }

  const queue = [...imageMessages];
  async function worker() {
    while (queue.length > 0) {
      const msg = queue.shift()!;
      await fetchOne(msg);
    }
  }

  await Promise.all(Array.from({ length: Math.min(maxConcurrent, queue.length) }, () => worker()));
  return result;
}

/**
 * Build an HTML document for PDF export (opened in a new window for printing).
 */
export function buildHtmlForPrint(
  contact: ExportContact,
  messages: Message[],
  locale: string = "en",
  imageDataUrls: Record<string, string> = {},
): string {
  const displayName = contact.name || contact.phone_number;
  const isRTL = locale === "ar";
  const dir = isRTL ? 'dir="rtl"' : "";

  const messageRows = messages
    .map((msg) => {
      const time = formatTimestamp(msg.created_at);
      const isOut = msg.direction === "outgoing";
      const sender = isOut
        ? "Agent"
        : (msg as any).sender_push_name || contact.profile_name || displayName;
      const text = escapeHtml(extractText(msg));
      const align = isOut ? "right" : "left";
      const bubbleBg = isOut ? "#1e9df1" : "#e5e5e6";
      const bubbleColor = isOut ? "#fff" : "#0f1419";

      const imgDataUrl = imageDataUrls[msg.id];
      const imageHtml = imgDataUrl
        ? `<img src="${imgDataUrl}" style="max-width:220px;max-height:180px;border-radius:8px;display:block;margin:4px 0;" />`
        : "";

      const hasImage = msg.message_type === "image" || msg.message_type === "sticker";
      const textContent = hasImage && imgDataUrl
        ? text.replace(/\[Image\]\s*/, "").trim()
        : text;

      return `
        <div style="display:flex;justify-content:${align};margin:4px 0;padding:0 12px;">
          <div style="max-width:70%;padding:6px 10px;border-radius:12px;background:${bubbleBg};color:${bubbleColor};font-size:13px;line-height:1.5;word-wrap:break-word;">
            <div style="font-size:11px;opacity:0.7;margin-bottom:2px;">${escapeHtml(sender)} · ${time}</div>
            ${imageHtml}${textContent ? `<div>${textContent}</div>` : ""}
          </div>
        </div>`;
    })
    .join("");

  return `<!DOCTYPE html>
<html lang="${locale}" ${dir}>
<head>
  <meta charset="UTF-8">
  <title>Chat: ${escapeHtml(displayName)}</title>
  <style>
    @media print { @page { margin: 1cm; } }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 16px; color: #0f1419; background: #fff; }
    h1 { font-size: 18px; margin: 0 0 4px; }
    .meta { font-size: 12px; color: #666; margin-bottom: 16px; }
    .separator { border-top: 1px solid #e0e0e0; margin: 12px 0; }
  </style>
</head>
<body>
  <h1>${escapeHtml(displayName)}</h1>
  <div class="meta">${escapeHtml(contact.phone_number)} · ${formatTimestamp(new Date().toISOString())} · ${messages.length} messages</div>
  <div class="separator"></div>
  ${messageRows}
  <div class="separator"></div>
  <div class="meta">End of export</div>
  <script>window.onload=function(){window.print();}</script>
</body>
</html>`;
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

export function downloadTextFile(content: string, filename: string) {
  downloadBlob(new Blob([content], { type: "text/plain;charset=utf-8" }), filename);
}

export function downloadHtmlAsPdf(html: string) {
  const w = window.open("", "_blank");
  if (!w) return;
  w.document.write(html);
  w.document.close();
}
