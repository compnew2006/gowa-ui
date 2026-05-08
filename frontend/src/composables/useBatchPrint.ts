import { ref, computed, type ComputedRef, type Ref } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue-sonner";
import {
  getCachedMediaBlob,
  prefetchMediaBlob,
  storeMediaBlobInPersistentCache,
} from "@/lib/media_prefetch_cache";
import {
  isMergePrintableBubbleMessage,
  toMergePrintableFile,
} from "@/lib/chat-bubble-merge-print";
import { mergePhotosAndPdfsAndOpenPrintDialog } from "@/lib/media-merge-print";
import type { Message } from "@/stores/contacts";

export function useBatchPrint(
  messages: ComputedRef<Message[]>,
  mediaBlobCache: Map<string, Blob>,
  mediaBlobUrls: Ref<Record<string, string>>,
  _getMediaBlobUrl: (message: Message) => string,
  loadMediaForMessage: (message: Message, generation?: number) => Promise<void>,
) {
  const { t } = useI18n();

  const isPreparingBatchPrint = ref(false);
  const isBatchPrintSelectionMode = ref(false);
  const selectedBatchPrintMessageIds = ref<string[]>([]);

  const selectedBatchPrintCount = computed(
    () => selectedBatchPrintMessageIds.value.length,
  );

  const hasMergePrintableBubbles = computed(() =>
    messages.value.some((message) => isMergePrintableBubbleMessage(message)),
  );

  const canMergeSelectedBubbleFiles = computed(
    () => selectedBatchPrintCount.value >= 2,
  );

  function resetBatchPrintSelection() {
    isBatchPrintSelectionMode.value = false;
    selectedBatchPrintMessageIds.value = [];
  }

  function cancelBatchPrintSelection() {
    resetBatchPrintSelection();
  }

  function isBatchPrintBubbleSelectable(message: Message): boolean {
    return isMergePrintableBubbleMessage(message);
  }

  function isBatchPrintBubbleSelected(messageId: string): boolean {
    return selectedBatchPrintMessageIds.value.includes(messageId);
  }

  function toggleBatchPrintMessageSelection(messageId: string) {
    if (isBatchPrintBubbleSelected(messageId)) {
      selectedBatchPrintMessageIds.value =
        selectedBatchPrintMessageIds.value.filter((id) => id !== messageId);
      return;
    }
    selectedBatchPrintMessageIds.value = [
      ...selectedBatchPrintMessageIds.value,
      messageId,
    ];
  }

  function isModifiedPointerEvent(event?: MouseEvent): boolean {
    if (!event) return false;
    return event.metaKey || event.ctrlKey || event.shiftKey || event.altKey;
  }

  function handleMessageBubbleClickForBatchPrint(
    message: Message,
    event?: MouseEvent,
  ) {
    if (!isBatchPrintSelectionMode.value) return;
    if (!isBatchPrintBubbleSelectable(message)) return;
    if (isModifiedPointerEvent(event)) return;
    event?.preventDefault();
    event?.stopPropagation();
    toggleBatchPrintMessageSelection(message.id);
  }

  async function resolveMessageBlobForBatchPrint(
    message: Message,
  ): Promise<Blob> {
    const cachedBlob = mediaBlobCache.get(message.id);
    if (cachedBlob) {
      return cachedBlob;
    }

    const persistentCachedBlob = await getCachedMediaBlob(message.id);
    if (persistentCachedBlob) {
      mediaBlobCache.set(message.id, persistentCachedBlob);
      if (!mediaBlobUrls.value[message.id]) {
        mediaBlobUrls.value[message.id] =
          URL.createObjectURL(persistentCachedBlob);
      }
      return persistentCachedBlob;
    }

    await loadMediaForMessage(message);
    const loadedBlob = mediaBlobCache.get(message.id);
    if (loadedBlob) {
      return loadedBlob;
    }

    const fallbackBlob = await prefetchMediaBlob(message.id);
    if (!fallbackBlob) {
      throw new Error("Failed to load media: empty response");
    }
    mediaBlobCache.set(message.id, fallbackBlob);
    void storeMediaBlobInPersistentCache(message.id, fallbackBlob);
    if (!mediaBlobUrls.value[message.id]) {
      mediaBlobUrls.value[message.id] = URL.createObjectURL(fallbackBlob);
    }
    return fallbackBlob;
  }

  async function mergeSelectedMessageBubblesAndPrint() {
    if (!canMergeSelectedBubbleFiles.value) {
      toast.error(t("chat.batchPrintMinSelection"), {
        description: t("chat.batchPrintMinSelectionDesc"),
      });
      return;
    }

    const selectedMessageIDs = new Set(selectedBatchPrintMessageIds.value);
    const selectedMessages = messages.value.filter(
      (message) =>
        selectedMessageIDs.has(message.id) &&
        isBatchPrintBubbleSelectable(message),
    );

    if (selectedMessages.length < 2) {
      toast.error(t("chat.batchPrintMinSelection"), {
        description: t("chat.batchPrintMinSelectionDesc"),
      });
      return;
    }

    isPreparingBatchPrint.value = true;
    toast.info(t("chat.batchPrintPreparing"));
    try {
      const files: File[] = [];
      for (const message of selectedMessages) {
        const blob = await resolveMessageBlobForBatchPrint(message);
        const file = toMergePrintableFile(message, blob, files.length);
        if (!file) {
          throw new Error(`Unsupported selected message: ${message.id}`);
        }
        files.push(file);
      }

      const opened = await mergePhotosAndPdfsAndOpenPrintDialog(files);
      if (!opened) {
        toast.error(t("chat.printDialogFailed"));
        return;
      }
      resetBatchPrintSelection();
    } catch (error) {
      console.error("Failed to merge selected bubbles for printing:", error);
      const errorDescription =
        error instanceof Error && error.message
          ? error.message
          : t("chat.batchPrintFailedDesc");
      toast.error(t("chat.batchPrintFailed"), {
        description: errorDescription,
      });
    } finally {
      isPreparingBatchPrint.value = false;
    }
  }

  function openBatchPrintPicker() {
    if (isPreparingBatchPrint.value) return;

    if (!isBatchPrintSelectionMode.value) {
      if (!hasMergePrintableBubbles.value) {
        toast.error(t("chat.batchPrintNoBubbleFiles"), {
          description: t("chat.batchPrintNoBubbleFilesDesc"),
        });
        return;
      }
      resetBatchPrintSelection();
      isBatchPrintSelectionMode.value = true;
      toast.info(t("chat.batchPrintSelectionMode"), {
        description: t("chat.batchPrintSelectionModeDesc"),
      });
      return;
    }

    void mergeSelectedMessageBubblesAndPrint();
  }

  function pruneSelectedIds() {
    if (selectedBatchPrintMessageIds.value.length > 0) {
      const availableMessageIDs = new Set(
        messages.value.map((message) => message.id),
      );
      selectedBatchPrintMessageIds.value =
        selectedBatchPrintMessageIds.value.filter((id) =>
          availableMessageIDs.has(id),
        );
    }
  }

  return {
    isPreparingBatchPrint,
    isBatchPrintSelectionMode,
    selectedBatchPrintMessageIds,
    selectedBatchPrintCount,
    hasMergePrintableBubbles,
    canMergeSelectedBubbleFiles,
    resetBatchPrintSelection,
    cancelBatchPrintSelection,
    isBatchPrintBubbleSelectable,
    isBatchPrintBubbleSelected,
    toggleBatchPrintMessageSelection,
    handleMessageBubbleClickForBatchPrint,
    resolveMessageBlobForBatchPrint,
    mergeSelectedMessageBubblesAndPrint,
    openBatchPrintPicker,
    isModifiedPointerEvent,
    pruneSelectedIds,
  };
}
