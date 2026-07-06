import { resolveMediaFilename, openPrintDialogForMedia } from "@/lib/media-actions";
import { mergePhotosAndPdfsAndOpenPrintDialog } from "@/lib/media-merge-print";
import {
  toMergePrintableFile,
  type MergePrintableChatMessage,
} from "@/lib/chat-bubble-merge-print";

const TIFF_MIME_TYPES = new Set([
  "image/tiff",
  "image/tif",
  "application/tiff",
  "application/x-tiff",
]);
const PDF_MIME_TYPES = new Set(["application/pdf", "application/x-pdf"]);

function normalize(value?: string | null): string {
  return String(value || "").trim().toLowerCase();
}

function hasTiffFilenameExtension(filename: string): boolean {
  const normalized = normalize(filename);
  return normalized.endsWith(".tif") || normalized.endsWith(".tiff");
}

function hasPdfFilenameExtension(filename: string): boolean {
  return normalize(filename).endsWith(".pdf");
}

export function isMessagePrintSupported(
  message: Pick<
    MergePrintableChatMessage,
    "id" | "message_type" | "media_filename" | "media_mime_type"
  >,
): boolean {
  const mimeType = normalize(message.media_mime_type);
  if (mimeType.startsWith("image/")) {
    return true;
  }
  if (PDF_MIME_TYPES.has(mimeType)) {
    return true;
  }

  const filename = resolveMediaFilename(message);
  if (hasTiffFilenameExtension(filename) || hasPdfFilenameExtension(filename)) {
    return true;
  }

  return normalize(message.message_type) === "image";
}

export function isTiffLikeMessage(
  message: Pick<
    MergePrintableChatMessage,
    "id" | "message_type" | "media_filename" | "media_mime_type"
  >,
  blob?: Blob | null,
): boolean {
  if (TIFF_MIME_TYPES.has(normalize(blob?.type))) {
    return true;
  }
  if (TIFF_MIME_TYPES.has(normalize(message.media_mime_type))) {
    return true;
  }
  return hasTiffFilenameExtension(resolveMediaFilename(message));
}

export async function openPrintDialogForSingleMessage(options: {
  mediaUrl: string;
  message: Pick<
    MergePrintableChatMessage,
    "id" | "message_type" | "media_filename" | "media_mime_type"
  >;
  resolveBlob: () => Promise<Blob>;
}): Promise<boolean> {
  const { mediaUrl, message, resolveBlob } = options;
  if (!mediaUrl) return false;
  if (!isMessagePrintSupported(message)) {
    return false;
  }

  if (!isTiffLikeMessage(message)) {
    return openPrintDialogForMedia(mediaUrl);
  }

  try {
    const blob = await resolveBlob();
    const file = toMergePrintableFile(message, blob, 0);
    if (!file) return false;
    return await mergePhotosAndPdfsAndOpenPrintDialog([file]);
  } catch {
    return false;
  }
}
