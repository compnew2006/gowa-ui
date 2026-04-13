export const MEDIA_PREFETCH_CACHE_NAME = "whatomate-media-v1";

const MEDIA_MESSAGE_TYPES = new Set([
  "image",
  "video",
  "audio",
  "document",
  "sticker",
]);
const inFlightMediaPrefetch = new Map<string, Promise<Blob | null>>();
const recentMissingMediaPrefetch = new Map<string, number>();
const missingMediaPrefetchCooldownMs = 10_000;

function normalizeBasePath(basePath?: string): string {
  const runtimeBasePath =
    typeof window !== "undefined"
      ? String((window as any).__BASE_PATH__ ?? "").trim()
      : "";
  const raw =
    typeof basePath === "string" && basePath.trim() !== ""
      ? basePath
      : runtimeBasePath;
  return raw.replace(/\/$/, "");
}

function normalizeMessageID(messageID: string): string {
  return messageID.trim();
}

function hasCacheStorage(): boolean {
  return typeof caches !== "undefined" && typeof caches.open === "function";
}

async function openMediaPrefetchCache(): Promise<Cache | null> {
  if (!hasCacheStorage()) return null;
  try {
    return await caches.open(MEDIA_PREFETCH_CACHE_NAME);
  } catch {
    return null;
  }
}

async function readCachedMediaResponse(
  requestURL: string,
): Promise<Response | null> {
  const cache = await openMediaPrefetchCache();
  if (!cache) return null;
  try {
    return (await cache.match(requestURL)) || null;
  } catch {
    return null;
  }
}

export function isMediaMessageType(messageType: unknown): boolean {
  if (typeof messageType !== "string") return false;
  return MEDIA_MESSAGE_TYPES.has(messageType.trim().toLowerCase());
}

export function buildMediaMessageURL(
  messageID: string,
  basePath?: string,
): string {
  const normalizedMessageID = normalizeMessageID(messageID);
  const normalizedBasePath = normalizeBasePath(basePath);
  return `${normalizedBasePath}/api/media/${encodeURIComponent(normalizedMessageID)}`;
}

function hasRecentMissingMediaPrefetch(requestURL: string): boolean {
  const expiresAt = recentMissingMediaPrefetch.get(requestURL);
  if (typeof expiresAt !== "number") return false;
  if (expiresAt <= Date.now()) {
    recentMissingMediaPrefetch.delete(requestURL);
    return false;
  }
  return true;
}

function markMissingMediaPrefetch(requestURL: string): void {
  recentMissingMediaPrefetch.set(
    requestURL,
    Date.now() + missingMediaPrefetchCooldownMs,
  );
}

export function clearMissingMediaPrefetch(
  messageID: string,
  basePath?: string,
): void {
  const normalizedMessageID = normalizeMessageID(messageID);
  if (!normalizedMessageID) return;
  recentMissingMediaPrefetch.delete(
    buildMediaMessageURL(normalizedMessageID, basePath),
  );
}

export async function getCachedMediaBlob(
  messageID: string,
  basePath?: string,
): Promise<Blob | null> {
  const normalizedMessageID = normalizeMessageID(messageID);
  if (!normalizedMessageID) return null;

  const requestURL = buildMediaMessageURL(normalizedMessageID, basePath);
  const cachedResponse = await readCachedMediaResponse(requestURL);
  if (!cachedResponse || !cachedResponse.ok) return null;

  try {
    return await cachedResponse.blob();
  } catch {
    return null;
  }
}

export async function storeMediaBlobInPersistentCache(
  messageID: string,
  blob: Blob,
  basePath?: string,
): Promise<void> {
  const normalizedMessageID = normalizeMessageID(messageID);
  if (!normalizedMessageID) return;
  clearMissingMediaPrefetch(normalizedMessageID, basePath);

  const cache = await openMediaPrefetchCache();
  if (!cache) return;

  const requestURL = buildMediaMessageURL(normalizedMessageID, basePath);
  const response = new Response(blob, {
    status: 200,
    headers: {
      "Content-Type": blob.type || "application/octet-stream",
    },
  });

  try {
    await cache.put(requestURL, response);
  } catch {
    // Ignore cache persistence failures.
  }
}

export async function prefetchMediaBlob(
  messageID: string,
  options?: { basePath?: string; signal?: AbortSignal },
): Promise<Blob | null> {
  const normalizedMessageID = normalizeMessageID(messageID);
  if (!normalizedMessageID) return null;

  const requestURL = buildMediaMessageURL(
    normalizedMessageID,
    options?.basePath,
  );
  const cachedBlob = await getCachedMediaBlob(
    normalizedMessageID,
    options?.basePath,
  );
  if (cachedBlob) return cachedBlob;
  if (hasRecentMissingMediaPrefetch(requestURL)) return null;
  if (typeof globalThis.fetch !== "function") return null;

  const existingInFlight = inFlightMediaPrefetch.get(requestURL);
  if (existingInFlight) {
    return existingInFlight;
  }

  const nextRequest = (async (): Promise<Blob | null> => {
    const response = await globalThis.fetch(requestURL, {
      credentials: "include",
      signal: options?.signal,
    });
    if (!response.ok) {
      if (response.status === 404 || response.status === 410) {
        markMissingMediaPrefetch(requestURL);
        return null;
      }
      throw new Error(`Failed to prefetch media: ${response.status}`);
    }

    recentMissingMediaPrefetch.delete(requestURL);
    const responseForCache = response.clone();
    const blob = await response.blob();

    const cache = await openMediaPrefetchCache();
    if (cache) {
      try {
        await cache.put(requestURL, responseForCache);
      } catch {
        // Ignore cache persistence failures.
      }
    }

    return blob;
  })();

  inFlightMediaPrefetch.set(requestURL, nextRequest);

  try {
    return await nextRequest;
  } finally {
    if (inFlightMediaPrefetch.get(requestURL) === nextRequest) {
      inFlightMediaPrefetch.delete(requestURL);
    }
  }
}

export function __resetMediaPrefetchCacheForTests(): void {
  inFlightMediaPrefetch.clear();
  recentMissingMediaPrefetch.clear();
}
