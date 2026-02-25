import { resolveMediaFilename } from "@/lib/media-actions";
import { isMergePrintableMimeType } from "@/lib/media-merge-print";

const PDF_MIME_TYPE = "application/pdf";
const IMAGE_FALLBACK_MIME_TYPE = "image/jpeg";

const MIME_TO_EXTENSION: Record<string, string> = {
  "application/pdf": "pdf",
  "image/jpeg": "jpg",
  "image/png": "png",
  "image/webp": "webp",
  "image/gif": "gif",
  "image/bmp": "bmp",
  "image/tiff": "tiff",
};

const EXTENSION_TO_MIME: Record<string, string> = {
  pdf: "application/pdf",
  jpg: "image/jpeg",
  jpeg: "image/jpeg",
  png: "image/png",
  webp: "image/webp",
  gif: "image/gif",
  bmp: "image/bmp",
  tif: "image/tiff",
  tiff: "image/tiff",
};

export interface MergePrintableChatMessage {
  id: string;
  message_type?: string;
  media_url?: string;
  media_mime_type?: string;
  media_filename?: string;
}

function normalize(value?: string | null): string {
  return String(value || "").trim().toLowerCase();
}

function extensionFromFilename(filename: string): string {
  const normalized = filename.trim();
  const dotIndex = normalized.lastIndexOf(".");
  if (dotIndex < 0 || dotIndex === normalized.length - 1) return "";
  return normalized.slice(dotIndex + 1).toLowerCase();
}

function hasKnownExtension(filename: string): boolean {
  return extensionFromFilename(filename) !== "";
}

function mimeTypeFromFilename(filename: string): string {
  const extension = extensionFromFilename(filename);
  if (!extension) return "";
  return EXTENSION_TO_MIME[extension] || "";
}

function fallbackMimeTypeFromMessageType(
  message: MergePrintableChatMessage,
): string {
  const messageType = normalize(message.message_type);
  if (messageType === "image") {
    return IMAGE_FALLBACK_MIME_TYPE;
  }
  if (messageType === "document") {
    const filename = resolveMediaFilename(message);
    if (normalize(filename).endsWith(".pdf")) {
      return PDF_MIME_TYPE;
    }
  }
  return "";
}

export function isMergePrintableBubbleMessage(
  message: MergePrintableChatMessage,
): boolean {
  if (!String(message.media_url || "").trim()) {
    return false;
  }

  const explicitMimeType = normalize(message.media_mime_type);
  if (isMergePrintableMimeType(explicitMimeType)) {
    return true;
  }

  const filenameMimeType = mimeTypeFromFilename(resolveMediaFilename(message));
  if (isMergePrintableMimeType(filenameMimeType)) {
    return true;
  }

  return normalize(message.message_type) === "image";
}

export function inferMergePrintableMimeType(
  message: MergePrintableChatMessage,
  blob: Blob,
): string {
  const blobMimeType = normalize(blob.type);
  if (isMergePrintableMimeType(blobMimeType)) {
    return blobMimeType;
  }

  const explicitMimeType = normalize(message.media_mime_type);
  if (isMergePrintableMimeType(explicitMimeType)) {
    return explicitMimeType;
  }

  const filenameMimeType = mimeTypeFromFilename(resolveMediaFilename(message));
  if (isMergePrintableMimeType(filenameMimeType)) {
    return filenameMimeType;
  }

  return fallbackMimeTypeFromMessageType(message);
}

export function resolveMergePrintableFilename(
  message: MergePrintableChatMessage,
  mimeType: string,
  index: number,
): string {
  const fallbackName = `attachment_${index + 1}`;
  const baseName = resolveMediaFilename(message).trim() || fallbackName;
  if (hasKnownExtension(baseName)) {
    return baseName;
  }

  const extension = MIME_TO_EXTENSION[normalize(mimeType)] || "";
  if (!extension) {
    return baseName;
  }
  return `${baseName}.${extension}`;
}

export function toMergePrintableFile(
  message: MergePrintableChatMessage,
  blob: Blob,
  index: number,
): File | null {
  const mimeType = inferMergePrintableMimeType(message, blob);
  if (!isMergePrintableMimeType(mimeType)) {
    return null;
  }

  const filename = resolveMergePrintableFilename(message, mimeType, index);
  return new File([blob], filename, {
    type: mimeType,
    lastModified: Date.now(),
  });
}
