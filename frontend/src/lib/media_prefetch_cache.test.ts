import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  __resetMediaPrefetchCacheForTests,
  buildMediaMessageURL,
  clearMissingMediaPrefetch,
  getCachedMediaBlob,
  isMediaMessageType,
  MEDIA_PREFETCH_CACHE_NAME,
  prefetchMediaBlob,
  storeMediaBlobInPersistentCache,
} from "./media_prefetch_cache";

type CacheRecordMap = Map<string, Response>;

function createMockCache(records: CacheRecordMap): Cache {
  return {
    add: vi.fn(),
    addAll: vi.fn(),
    delete: vi.fn(),
    keys: vi.fn(),
    match: vi.fn(async (request: RequestInfo | URL) => {
      const key = String(request);
      const response = records.get(key);
      return response ? response.clone() : undefined;
    }),
    matchAll: vi.fn(),
    put: vi.fn(async (request: RequestInfo | URL, response: Response) => {
      records.set(String(request), response.clone());
    }),
  } as unknown as Cache;
}

describe("media_prefetch_cache", () => {
  const records: CacheRecordMap = new Map();
  const originalFetch = globalThis.fetch;
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    records.clear();
    __resetMediaPrefetchCacheForTests();

    const cache = createMockCache(records);
    Object.defineProperty(globalThis, "caches", {
      value: {
        open: vi.fn(async (name: string) => {
          expect(name).toBe(MEDIA_PREFETCH_CACHE_NAME);
          return cache;
        }),
      },
      configurable: true,
      writable: true,
    });

    fetchMock = vi.fn();
    (globalThis as any).fetch = fetchMock;
  });

  afterEach(() => {
    __resetMediaPrefetchCacheForTests();
    (globalThis as any).fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("identifies supported media message types", () => {
    expect(isMediaMessageType("image")).toBe(true);
    expect(isMediaMessageType(" video ")).toBe(true);
    expect(isMediaMessageType("document")).toBe(true);
    expect(isMediaMessageType("text")).toBe(false);
    expect(isMediaMessageType(null)).toBe(false);
  });

  it("builds media URL with base path and encoded message id", () => {
    expect(buildMediaMessageURL("abc 123", "/base/")).toBe(
      "/base/api/media/abc%20123",
    );
  });

  it("returns cached blob without network fetch when cache entry exists", async () => {
    const blob = new Blob(["cached-data"], { type: "text/plain" });
    await storeMediaBlobInPersistentCache("msg-cache-hit", blob, "/base");

    const result = await prefetchMediaBlob("msg-cache-hit", {
      basePath: "/base",
    });
    expect(fetchMock).not.toHaveBeenCalled();
    expect(result).not.toBeNull();
    expect(await result?.text()).toBe("cached-data");
  });

  it("deduplicates in-flight requests for the same message id", async () => {
    let resolveFetch: ((value: Response) => void) | undefined;
    fetchMock.mockImplementation(() => {
      return new Promise<Response>((resolve) => {
        resolveFetch = resolve;
      });
    });

    const requestA = prefetchMediaBlob("msg-dedupe", { basePath: "/base" });
    const requestB = prefetchMediaBlob("msg-dedupe", { basePath: "/base" });

    await vi.waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(1);
    });
    if (typeof resolveFetch !== "function") {
      throw new Error("Fetch resolver was not captured");
    }
    resolveFetch(new Response("deduped-data", { status: 200 }));

    const [blobA, blobB] = await Promise.all([requestA, requestB]);
    expect(blobA).not.toBeNull();
    expect(blobB).not.toBeNull();
    expect(await blobA?.text()).toBe("deduped-data");
    expect(await blobB?.text()).toBe("deduped-data");
  });

  it("exposes cached blob reads through getCachedMediaBlob", async () => {
    await storeMediaBlobInPersistentCache(
      "msg-cache-read",
      new Blob(["from-cache"], { type: "text/plain" }),
      "/base",
    );

    const cached = await getCachedMediaBlob("msg-cache-read", "/base");
    expect(cached).not.toBeNull();
    expect(await cached?.text()).toBe("from-cache");
  });

  it("cools down repeated 404 media prefetch retries", async () => {
    fetchMock.mockResolvedValue(new Response("missing", { status: 404 }));

    const firstResult = await prefetchMediaBlob("msg-missing", {
      basePath: "/base",
    });
    const secondResult = await prefetchMediaBlob("msg-missing", {
      basePath: "/base",
    });

    expect(firstResult).toBeNull();
    expect(secondResult).toBeNull();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("treats gone media responses like missing media", async () => {
    fetchMock.mockResolvedValue(new Response("gone", { status: 410 }));

    const firstResult = await prefetchMediaBlob("msg-gone", {
      basePath: "/base",
    });
    const secondResult = await prefetchMediaBlob("msg-gone", {
      basePath: "/base",
    });

    expect(firstResult).toBeNull();
    expect(secondResult).toBeNull();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("clears the missing-media cooldown when media becomes available", async () => {
    fetchMock.mockResolvedValue(new Response("missing", { status: 404 }));
    await prefetchMediaBlob("msg-recovered", { basePath: "/base" });
    expect(fetchMock).toHaveBeenCalledTimes(1);

    fetchMock.mockResolvedValue(new Response("recovered", { status: 200 }));
    clearMissingMediaPrefetch("msg-recovered", "/base");

    const recovered = await prefetchMediaBlob("msg-recovered", {
      basePath: "/base",
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(recovered).not.toBeNull();
    expect(await recovered?.text()).toBe("recovered");
  });
});
