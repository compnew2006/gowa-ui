import { ref, computed, type Ref, type ComputedRef } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue-sonner";
import { resolveMediaFilename } from "@/lib/media-actions";
import { getErrorMessage } from "@/lib/api-utils";
import {
  resolveWhatsAppMediaCategoryForFile,
  validateWhatsAppMediaFile,
  type WhatsAppMediaCategory,
} from "@/lib/whatsapp-media-policy";
import {
  isMessagePrintSupported,
  openPrintDialogForSingleMessage,
} from "@/lib/single-media-print";
import { downloadMessageMedia } from "@/lib/media-actions";
import type { Contact, Message } from "@/stores/contacts";
import type { CannedResponseAttachment } from "@/services/api";
import type { ChatMediaViewerType } from "./useChatMedia";

interface PendingMediaUpload {
  id: string;
  file: File;
  category: WhatsAppMediaCategory;
  previewUrl: string | null;
}

export function useChatMessaging(
  currentContact: ComputedRef<Contact | null>,
  options: {
    isCurrentChatSendRestricted: ComputedRef<boolean>;
    isCurrentChatClosed: ComputedRef<boolean>;
    resolveOutboundInstanceID: (contact: Contact | null) => string | undefined;
    resolveOutboundWhatsAppAccount: (contact: Contact | null) => string | undefined;
    scrollToBottom: (instant?: boolean) => void;
    addMessage: (message: Message) => void;
    loadMediaForMessage: (message: Message) => Promise<void>;
    openChatMediaViewer: (url: string, type: ChatMediaViewerType, title: string) => void;
    resolveMessageBlobForBatchPrint: (message: Message) => Promise<Blob>;
    isBatchPrintSelectionMode: Ref<boolean>;
    isBatchPrintBubbleSelectable: (message: Message) => boolean;
    isModifiedPointerEvent: (event?: MouseEvent) => boolean;
    handleMessageBubbleClickForBatchPrint: (message: Message, event?: MouseEvent) => void;
  },
) {
  const { t } = useI18n();

  const fileInputRef = ref<HTMLInputElement | null>(null);
  const selectedMediaUploads = ref<PendingMediaUpload[]>([]);
  const activeMediaPreviewID = ref<string | null>(null);
  const isMediaDialogOpen = ref(false);
  const mediaCaption = ref("");
  const isUploadingMedia = ref(false);
  const mediaUploadProgress = ref<{ current: number; total: number } | null>(null);

  const cannedPickerOpen = ref(false);
  const cannedSearchQuery = ref("");
  const pendingCannedResponse = ref<{
    id: string;
    attachments: CannedResponseAttachment[];
  } | null>(null);

  const emojiPickerOpen = ref(false);
  const reactionPickerMessageId = ref<string | null>(null);
  const quickReactionEmojis = ["👍", "❤️", "😂", "😮", "😢", "🙏"];

  const retryingMessageId = ref<string | null>(null);
  const revokingMessageId = ref<string | null>(null);

  const selectedMediaCount = computed(() => selectedMediaUploads.value.length);

  const activeMediaUpload = computed<PendingMediaUpload | null>(() => {
    if (activeMediaPreviewID.value) {
      const matchedUpload = selectedMediaUploads.value.find(
        (upload) => upload.id === activeMediaPreviewID.value,
      );
      if (matchedUpload) {
        return matchedUpload;
      }
    }
    return selectedMediaUploads.value[0] ?? null;
  });

  const canApplyMediaCaption = computed(
    () =>
      selectedMediaCount.value === 1 &&
      activeMediaUpload.value?.category !== "audio",
  );

  const mediaDialogDescription = computed(() => {
    if (selectedMediaCount.value === 0) return "";
    if (selectedMediaCount.value === 1) {
      return activeMediaUpload.value?.file.name ?? "";
    }
    return t("chat.mediaFilesSelected", { count: selectedMediaCount.value });
  });

  const mediaSendButtonLabel = computed(() =>
    selectedMediaCount.value > 1 ? t("chat.sendFiles") : t("chat.send"),
  );

  const mediaUploadingLabel = computed(() => {
    if (selectedMediaCount.value > 1 && mediaUploadProgress.value) {
      return t("chat.mediaSendingProgress", mediaUploadProgress.value);
    }
    return `${t("chat.sending")}...`;
  });

  const hasPendingCannedAttachments = computed(() => {
    return (pendingCannedResponse.value?.attachments.length ?? 0) > 0;
  });

  function insertCannedResponse(payload: {
    id: string;
    content: string;
    attachments: CannedResponseAttachment[];
  }) {
    if (payload.attachments && payload.attachments.length > 0) {
      pendingCannedResponse.value = {
        id: payload.id,
        attachments: [...payload.attachments],
      };
    } else {
      pendingCannedResponse.value = null;
    }
    cannedPickerOpen.value = false;
    cannedSearchQuery.value = "";
  }

  function closeCannedPicker() {
    cannedPickerOpen.value = false;
    cannedSearchQuery.value = "";
  }

  function clearPendingCannedAttachments() {
    pendingCannedResponse.value = null;
  }

  function removePendingCannedAttachment(index: number) {
    if (!pendingCannedResponse.value) return;
    pendingCannedResponse.value.attachments =
      pendingCannedResponse.value.attachments.filter(
        (_, currentIndex) => currentIndex !== index,
      );
    if (pendingCannedResponse.value.attachments.length === 0) {
      pendingCannedResponse.value = null;
    }
  }

  function getPendingAttachmentIcon(type: string) {
    return type === "video"
      ? () => import("lucide-vue-next").then((m) => m.Play)
      : () => import("lucide-vue-next").then((m) => m.Image);
  }

  function insertEmoji(emoji: string) {
    emojiPickerOpen.value = false;
    return emoji;
  }

  async function sendReaction(messageId: string, emoji: string) {
    if (!currentContact.value) return;

    try {
      const { useContactsStore } = await import("@/stores/contacts");
      const contactsStore = useContactsStore();
      const { messagesService } = await import("@/services/api");
      const response = await messagesService.sendReaction(
        currentContact.value.id,
        messageId,
        emoji,
      );
      const data = response.data.data || response.data;
      contactsStore.updateMessageReactions(messageId, data.reactions);
    } catch (error) {
      toast.error(t("chat.reactionFailed"));
    }
    reactionPickerMessageId.value = null;
  }

  function getMediaSizeErrorKey(category: WhatsAppMediaCategory) {
    if (category === "image") return "chat.fileTooLargeImageDesc";
    if (category === "video") return "chat.fileTooLargeVideoDesc";
    if (category === "audio") return "chat.fileTooLargeAudioDesc";
    return "chat.fileTooLargeDocumentDesc";
  }

  function buildPendingMediaUpload(file: File, index: number): PendingMediaUpload {
    const category = resolveWhatsAppMediaCategoryForFile(file);
    const shouldPreview = category === "image" || category === "video";
    return {
      id: `${file.name}-${file.size}-${file.lastModified}-${index}`,
      file,
      category,
      previewUrl: shouldPreview ? URL.createObjectURL(file) : null,
    };
  }

  function revokePendingMediaUpload(upload: PendingMediaUpload) {
    if (upload.previewUrl) {
      URL.revokeObjectURL(upload.previewUrl);
    }
  }

  function formatMediaUploadSize(sizeBytes: number) {
    const kilobyte = 1024;
    const megabyte = kilobyte * 1024;
    if (sizeBytes >= megabyte) return `${(sizeBytes / megabyte).toFixed(1)} MB`;
    if (sizeBytes >= kilobyte) return `${(sizeBytes / kilobyte).toFixed(1)} KB`;
    return `${sizeBytes} B`;
  }

  function setActiveMediaPreview(uploadID: string) {
    activeMediaPreviewID.value = uploadID;
  }

  function removeSelectedMediaUpload(uploadID: string) {
    const removedUpload = selectedMediaUploads.value.find(
      (upload) => upload.id === uploadID,
    );
    if (!removedUpload) return;
    revokePendingMediaUpload(removedUpload);
    const remainingUploads = selectedMediaUploads.value.filter(
      (upload) => upload.id !== uploadID,
    );
    selectedMediaUploads.value = remainingUploads;
    if (remainingUploads.length === 0) {
      closeMediaDialog();
      return;
    }
    if (activeMediaPreviewID.value === uploadID) {
      activeMediaPreviewID.value = remainingUploads[0].id;
    }
    if (remainingUploads.length > 1) {
      mediaCaption.value = "";
    }
  }

  function handleMediaDialogOpenChange(open: boolean) {
    if (open) {
      isMediaDialogOpen.value = true;
      return;
    }
    if (isUploadingMedia.value) {
      isMediaDialogOpen.value = true;
      return;
    }
    closeMediaDialog();
  }

  function openFilePicker() {
    fileInputRef.value?.click();
  }

  function handleFileSelect(event: Event) {
    const input = event.target as HTMLInputElement;
    const files = Array.from(input.files ?? []);
    if (files.length === 0) return;

    const acceptedUploads: PendingMediaUpload[] = [];
    files.forEach((file, index) => {
      const validation = validateWhatsAppMediaFile(file);
      if (!validation.isValid) {
        toast.error(t("chat.fileTooLarge"), {
          description: `${file.name}: ${t(getMediaSizeErrorKey(validation.category))}`,
        });
        return;
      }
      acceptedUploads.push(buildPendingMediaUpload(file, index));
    });

    input.value = "";
    if (acceptedUploads.length === 0) return;

    selectedMediaUploads.value.forEach(revokePendingMediaUpload);
    selectedMediaUploads.value = acceptedUploads;
    activeMediaPreviewID.value = acceptedUploads[0]?.id ?? null;
    mediaCaption.value = "";
    isMediaDialogOpen.value = true;
  }

  function closeMediaDialog() {
    selectedMediaUploads.value.forEach(revokePendingMediaUpload);
    selectedMediaUploads.value = [];
    activeMediaPreviewID.value = null;
    mediaUploadProgress.value = null;
    mediaCaption.value = "";
    isMediaDialogOpen.value = false;
  }

  async function sendMediaMessage() {
    if (options.isCurrentChatSendRestricted.value || options.isCurrentChatClosed.value) return;
    if (selectedMediaUploads.value.length === 0 || !currentContact.value) return;

    const uploads = [...selectedMediaUploads.value];
    const outboundInstanceID = options.resolveOutboundInstanceID(currentContact.value);
    const accountFilter = options.resolveOutboundWhatsAppAccount(currentContact.value);
    const shouldApplyCaption = uploads.length === 1 && uploads[0].category !== "audio";
    const caption = shouldApplyCaption ? mediaCaption.value : "";
    const sentMessages: Message[] = [];
    const successfulUploadIDs = new Set<string>();
    let firstError: unknown = null;

    isUploadingMedia.value = true;
    try {
      const { messagesService } = await import("@/services/api");
      for (const [index, upload] of uploads.entries()) {
        mediaUploadProgress.value = { current: index + 1, total: uploads.length };
        try {
          const response = await messagesService.sendMedia({
            contactId: currentContact.value.id,
            file: upload.file,
            type: upload.category,
            caption,
            instance_id: outboundInstanceID,
            whatsapp_account: accountFilter,
          });
          const result = response.data.data || response.data;
          successfulUploadIDs.add(upload.id);
          if (result) sentMessages.push(result);
        } catch (error) {
          if (!firstError) firstError = error;
        }
      }

      sentMessages.forEach((message) => options.addMessage(message));

      if (sentMessages.length > 0) {
        options.scrollToBottom();
        await import("vue").then(({ nextTick }) =>
          nextTick(() => {
            sentMessages.forEach((message) => {
              if (message.media_url) options.loadMediaForMessage(message);
            });
          }),
        );
      }

      if (successfulUploadIDs.size === uploads.length) {
        toast.success(
          uploads.length > 1
            ? t("chat.mediaBatchSent", { count: uploads.length })
            : t("chat.mediaSent"),
        );
        closeMediaDialog();
        return;
      }

      if (successfulUploadIDs.size === 0) {
        throw firstError;
      }

      const failedUploads = uploads.filter(
        (upload) => !successfulUploadIDs.has(upload.id),
      );
      uploads
        .filter((upload) => successfulUploadIDs.has(upload.id))
        .forEach(revokePendingMediaUpload);

      selectedMediaUploads.value = failedUploads;
      activeMediaPreviewID.value = failedUploads[0]?.id ?? null;
      mediaCaption.value = "";

      toast.warning(t("chat.mediaBatchPartialFailed"), {
        description: t("chat.mediaBatchPartialFailedDesc", {
          sent: successfulUploadIDs.size,
          failed: failedUploads.length,
        }),
      });
    } catch (error: any) {
      toast.error(t("chat.mediaFailed"), {
        description: getErrorMessage(error, t("chat.mediaFailedDesc")),
      });
    } finally {
      mediaUploadProgress.value = null;
      isUploadingMedia.value = false;
    }
  }

  function openMediaPreview(
    message: Message,
    event?: MouseEvent,
    getMediaBlobUrl: (message: Message) => string = () => "",
  ) {
    if (options.isBatchPrintSelectionMode.value) {
      options.handleMessageBubbleClickForBatchPrint(message, event);
      return;
    }
    if (options.isModifiedPointerEvent(event)) return;
    const url = getMediaBlobUrl(message);
    if (!url) return;
    options.openChatMediaViewer(
      url,
      message.message_type === "video" ? "video" : "image",
      resolveMediaFilename(message),
    );
  }

  function downloadAttachment(
    message: Message,
    event?: MouseEvent,
    getMediaBlobUrl: (message: Message) => string = () => "",
  ) {
    if (options.isBatchPrintSelectionMode.value) return;
    if (options.isModifiedPointerEvent(event)) return;
    const mediaUrl = getMediaBlobUrl(message);
    if (!mediaUrl) return;
    downloadMessageMedia(mediaUrl, message);
  }

  function printAttachment(
    message: Message,
    event?: MouseEvent,
    getMediaBlobUrl: (message: Message) => string = () => "",
  ) {
    if (options.isBatchPrintSelectionMode.value) return;
    if (options.isModifiedPointerEvent(event)) return;
    const mediaUrl = getMediaBlobUrl(message);
    if (!mediaUrl) return;
    if (!isMessagePrintSupported(message)) {
      toast.error(t("chat.printDialogFailed"), {
        description: t("chat.batchPrintUnsupportedDesc"),
      });
      return;
    }
    void (async () => {
      const opened = await openPrintDialogForSingleMessage({
        message,
        mediaUrl,
        resolveBlob: () => options.resolveMessageBlobForBatchPrint(message),
      });
      if (!opened) {
        toast.error(t("chat.printDialogFailed"));
      }
    })();
  }

  return {
    fileInputRef,
    selectedMediaUploads,
    activeMediaPreviewID,
    isMediaDialogOpen,
    mediaCaption,
    isUploadingMedia,
    mediaUploadProgress,
    cannedPickerOpen,
    cannedSearchQuery,
    pendingCannedResponse,
    emojiPickerOpen,
    reactionPickerMessageId,
    quickReactionEmojis,
    retryingMessageId,
    revokingMessageId,
    selectedMediaCount,
    activeMediaUpload,
    canApplyMediaCaption,
    mediaDialogDescription,
    mediaSendButtonLabel,
    mediaUploadingLabel,
    hasPendingCannedAttachments,
    insertCannedResponse,
    closeCannedPicker,
    clearPendingCannedAttachments,
    removePendingCannedAttachment,
    getPendingAttachmentIcon,
    insertEmoji,
    sendReaction,
    openFilePicker,
    handleFileSelect,
    closeMediaDialog,
    handleMediaDialogOpenChange,
    sendMediaMessage,
    setActiveMediaPreview,
    removeSelectedMediaUpload,
    formatMediaUploadSize,
    openMediaPreview,
    downloadAttachment,
    printAttachment,
  };
}
