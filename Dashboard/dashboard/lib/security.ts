const MAX_URL_LENGTH = 2048;

const IMAGE_TYPES = new Set(["image/jpeg", "image/png", "image/gif", "image/webp"]);
const VIDEO_TYPES = new Set(["video/mp4", "video/quicktime", "video/webm"]);

export const MAX_IMAGE_BYTES = 5 * 1024 * 1024;
export const MAX_VIDEO_BYTES = 50 * 1024 * 1024;

function isLoopback(hostname: string) {
  return ["localhost", "127.0.0.1", "::1"].includes(hostname);
}

export function getSafeDisplayUrl(value?: string | null): string | undefined {
  if (!value || value.length > MAX_URL_LENGTH) return undefined;
  try {
    const base = typeof window !== "undefined" ? window.location.origin : "https://example.com";
    const url = new URL(value, base);
    if (url.protocol === "blob:") return url.href;
    if (!["http:", "https:"].includes(url.protocol)) return undefined;
    if (url.protocol === "http:" && !isLoopback(url.hostname)) return undefined;
    return url.href;
  } catch {
    return undefined;
  }
}

export function isSafeDisplayUrl(value?: string | null): boolean {
  return Boolean(getSafeDisplayUrl(value));
}

export function validateImageFile(file: File): string | null {
  if (!IMAGE_TYPES.has(file.type)) return "نوع الصورة غير مدعوم";
  if (file.size > MAX_IMAGE_BYTES) return "حجم الصورة يجب ألا يتجاوز 5MB";
  return null;
}

export function validateCampaignMediaFile(file: File): string | null {
  if (IMAGE_TYPES.has(file.type)) {
    return file.size > MAX_IMAGE_BYTES ? "حجم الصورة يجب ألا يتجاوز 5MB" : null;
  }
  if (VIDEO_TYPES.has(file.type)) {
    return file.size > MAX_VIDEO_BYTES ? "حجم الفيديو يجب ألا يتجاوز 50MB" : null;
  }
  return "نوع الملف غير مدعوم";
}
