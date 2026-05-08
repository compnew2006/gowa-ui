import { ref, type ComputedRef } from "vue";
import {
  prefetchMediaBlob,
  storeMediaBlobInPersistentCache,
  getCachedMediaBlob,
} from "@/lib/media_prefetch_cache";
import type { Message } from "@/stores/contacts";

export type ChatMediaViewerType = "image" | "video" | "audio" | "document";

const MAX_MEDIA_LOAD_CONCURRENCY = 4;

export function useChatMedia(messages: ComputedRef<Message[]>) {
  const mediaBlobUrls = ref<Record<string, string>>({});
  const mediaLoadingStates = ref<Record<string, boolean>>({});
  const mediaBlobCache = new Map<string, Blob>();
  const pendingMediaQueue: Message[] = [];
  const queuedMediaMessageIDs = new Set<string>();
  const inFlightMediaRequests = new Map<string, AbortController>();
  let activeMediaLoadCount = 0;
  let mediaLoadGeneration = 0;

  const isChatMediaViewerOpen = ref(false);
  const chatMediaViewerURL = ref("");
  const chatMediaViewerType = ref<ChatMediaViewerType>("image");
  const chatMediaViewerTitle = ref("");

  function resetMediaLoadingPipeline() {
    mediaLoadGeneration++;
    pendingMediaQueue.length = 0;
    queuedMediaMessageIDs.clear();
    for (const controller of inFlightMediaRequests.values()) {
      controller.abort();
    }
    inFlightMediaRequests.clear();
  }

  function enqueueMediaForBackgroundLoad(message: Message) {
    if (queuedMediaMessageIDs.has(message.id)) return;
    queuedMediaMessageIDs.add(message.id);
    pendingMediaQueue.push(message);
  }

  function pumpMediaLoadQueue() {
    while (
      activeMediaLoadCount < MAX_MEDIA_LOAD_CONCURRENCY &&
      pendingMediaQueue.length > 0
    ) {
      const message = pendingMediaQueue.shift();
      if (!message) continue;
      queuedMediaMessageIDs.delete(message.id);

      if (
        !message.media_url ||
        mediaBlobUrls.value[message.id] ||
        mediaLoadingStates.value[message.id]
      ) {
        continue;
      }

      const generationAtStart = mediaLoadGeneration;
      activeMediaLoadCount++;
      void loadMediaForMessage(message, generationAtStart).finally(() => {
        activeMediaLoadCount = Math.max(0, activeMediaLoadCount - 1);
        pumpMediaLoadQueue();
      });
    }
  }

  async function loadMediaForMessage(
    message: Message,
    generation: number = mediaLoadGeneration,
  ) {
    if (
      generation !== mediaLoadGeneration ||
      !message.media_url ||
      mediaLoadingStates.value[message.id]
    ) {
      return;
    }

    const cachedBlob = mediaBlobCache.get(message.id);
    if (cachedBlob) {
      if (!mediaBlobUrls.value[message.id]) {
        mediaBlobUrls.value[message.id] = URL.createObjectURL(cachedBlob);
      }
      return;
    }

    const persistentCachedBlob = await getCachedMediaBlob(message.id);
    if (persistentCachedBlob) {
      mediaBlobCache.set(message.id, persistentCachedBlob);
      if (generation !== mediaLoadGeneration) {
        return;
      }
      if (!mediaBlobUrls.value[message.id]) {
        mediaBlobUrls.value[message.id] =
          URL.createObjectURL(persistentCachedBlob);
      }
      return;
    }

    if (mediaBlobUrls.value[message.id]) {
      return;
    }

    const controller = new AbortController();
    inFlightMediaRequests.set(message.id, controller);
    mediaLoadingStates.value[message.id] = true;

    try {
      const blob = await prefetchMediaBlob(message.id, {
        signal: controller.signal,
      });
      if (!blob) {
        throw new Error("Failed to load media: empty response");
      }
      mediaBlobCache.set(message.id, blob);
      void storeMediaBlobInPersistentCache(message.id, blob);
      if (generation !== mediaLoadGeneration) {
        return;
      }
      if (mediaBlobUrls.value[message.id]) {
        URL.revokeObjectURL(mediaBlobUrls.value[message.id]);
      }
      const blobUrl = URL.createObjectURL(blob);
      mediaBlobUrls.value[message.id] = blobUrl;
    } catch (error: any) {
      if (error?.name === "AbortError") {
        return;
      }
      console.error("Failed to load media:", error, "message_id:", message.id);
    } finally {
      if (inFlightMediaRequests.get(message.id) === controller) {
        inFlightMediaRequests.delete(message.id);
      }
      mediaLoadingStates.value[message.id] = false;
    }
  }

  function loadMediaForMessages() {
    try {
      for (const message of messages.value) {
        if (message.media_url && !mediaBlobUrls.value[message.id]) {
          enqueueMediaForBackgroundLoad(message);
        }
      }
      pumpMediaLoadQueue();
    } catch (e) {
      console.error("Error in loadMediaForMessages:", e);
    }
  }

  function getMediaBlobUrl(message: Message): string {
    return mediaBlobUrls.value[message.id] || "";
  }

  function isMediaLoading(message: Message): boolean {
    return mediaLoadingStates.value[message.id] || false;
  }

  function openChatMediaViewer(
    url: string,
    type: ChatMediaViewerType,
    title: string,
  ) {
    chatMediaViewerURL.value = url;
    chatMediaViewerType.value = type;
    chatMediaViewerTitle.value = title;
    isChatMediaViewerOpen.value = true;
  }

  function closeChatMediaViewer() {
    isChatMediaViewerOpen.value = false;
    chatMediaViewerURL.value = "";
    chatMediaViewerType.value = "image";
    chatMediaViewerTitle.value = "";
  }

  function cleanupBlobUrls() {
    Object.values(mediaBlobUrls.value).forEach((url) => {
      URL.revokeObjectURL(url);
    });
    mediaBlobUrls.value = {};
    mediaBlobCache.clear();
  }

  return {
    mediaBlobUrls,
    mediaLoadingStates,
    mediaBlobCache,
    isChatMediaViewerOpen,
    chatMediaViewerURL,
    chatMediaViewerType,
    chatMediaViewerTitle,
    resetMediaLoadingPipeline,
    loadMediaForMessage,
    loadMediaForMessages,
    getMediaBlobUrl,
    isMediaLoading,
    openChatMediaViewer,
    closeChatMediaViewer,
    cleanupBlobUrls,
  };
}
