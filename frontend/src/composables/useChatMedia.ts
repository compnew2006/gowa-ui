import { ref, computed } from 'vue'
import type { Ref } from 'vue'
import { toast } from 'vue-sonner'
import { api, getRequestHeaders } from '@/services/api'
import { mediaDisplayName, mediaUrl, saveBlob } from '@/lib/media'
import type { Message } from '@/stores/contacts'

/** Max files per send batch. Each file is sent as its own message via a
 * synchronous GOWA upload, so larger batches risk long dialogs/timeouts. */
export const MAX_BATCH_FILES = 10

export interface UseChatMediaOptions {
  /** i18n translator. */
  t: (key: string, params?: Record<string, unknown>) => string
  /** Contacts store reactive surface. */
  contactsStore: {
    currentContact: { id: string } | null
    messages: Message[]
    addMessage: (m: Message) => void
  }
  /** Selected account ref (multi-account). */
  selectedAccount: { value: string | null }
  /** Called after a media message lands so the room scrolls to the new bubble. */
  scrollToBottom: (instant?: boolean) => void
  /** Posts a media WhatsApp Status (story). Provided by the messaging composable. */
  sendStatusMedia: (file: File, caption: string) => Promise<void>
  /** True when the current contact is the virtual Status conversation. */
  isStatusContact: (id: string) => boolean
  /** Shared media-export instance (redownload). Shared with the burst-UI so both
   * single-bubble retry and burst download track progress on one state. */
  mediaExport: {
    redownloading: { value: Set<string> | undefined }
    redownload: (message: Message) => Promise<{ ok: boolean; mediaUrl?: string; mediaMimeType?: string }>
  }
  /** Hidden file-input template ref. Owned by the view (template ref). */
  fileInputRef: Ref<HTMLInputElement | null>
}

/**
 * File upload, media preview, and per-message media recovery for the chat room.
 * Owns the upload dialog state, broken-media tracking, and the media-export
 * (redownload) integration.
 *
 * @example
 * ```ts
 * const media = useChatMedia({ contactsStore, selectedAccount, ... })
 * ```
 */
export function useChatMedia(options: UseChatMediaOptions) {
  const { t, contactsStore, selectedAccount } = options

  // File upload state (fileInputRef is owned by the view, passed in).
  const fileInputRef = options.fileInputRef
  // Multi-file queue: the picker accepts several files, each sent as its own
  // message (caption goes on the first only, so the rest can still fold into
  // an album bubble at render time — see useChatAlbums).
  const selectedFiles = ref<File[]>([])
  const activeFileIndex = ref(0)
  const filePreviewUrls = ref<(string | null)[]>([])
  // Back-compat selectors for the single active file (the view's preview
  // blocks keep working unchanged).
  const selectedFile = computed(() => selectedFiles.value[activeFileIndex.value] ?? null)
  const filePreviewUrl = computed(() => filePreviewUrls.value[activeFileIndex.value] ?? null)
  const isMediaDialogOpen = ref(false)
  const mediaCaption = ref('')
  const isUploadingMedia = ref(false)
  // 0..100 overall batch progress (completed files + current file fraction).
  const uploadProgress = ref(0)
  // 1-based position inside the batch while sending ("file 2 of 5").
  const uploadCurrent = ref(0)
  const uploadTotal = ref(0)
  // Aborts the in-flight upload when the user cancels mid-transfer. Non-null
  // only while an upload request is running. batchCancelled stops the queue
  // loop after the current file finishes aborting.
  let uploadAbort: AbortController | null = null
  let batchCancelled = false

  // Messages whose media failed to load in the DOM (video error, image error).
  // Keyed by message id so the "Retry download" affordance only shows on broken bubbles.
  const brokenMediaIds = ref(new Set<string>())

  // Messages whose file a per-bubble Download click is currently fetching.
  // Keyed by message id — drives the button spinner and blocks double clicks.
  const downloadingMessageIds = ref(new Set<string>())

  const {
    redownloading: redownloadingIds,
    redownload
  } = options.mediaExport

  function getMediaType(mimeType: string): string {
    if (mimeType.startsWith('image/')) return 'image'
    if (mimeType.startsWith('video/')) return 'video'
    if (mimeType.startsWith('audio/')) return 'audio'
    return 'document'
  }

  /**
   * Re-fetch a message's media from the provider and, on success, patch the
   * updated media_url into the store so the bubble re-renders with live media.
   */
  async function retryMediaDownload(message: Message) {
    const result = await redownload(message)
    if (!result.ok) return
    // Patch the message in the store so the bubble re-renders.
    const idx = contactsStore.messages.findIndex((m) => m.id === message.id)
    if (idx !== -1) {
      const updated = {
        ...contactsStore.messages[idx],
        media_url: result.mediaUrl || contactsStore.messages[idx].media_url,
        media_mime_type: result.mediaMimeType || contactsStore.messages[idx].media_mime_type
      }
      const fresh = [...contactsStore.messages]
      fresh[idx] = updated
      contactsStore.messages = fresh
    }
    // Clear the broken flag so any error placeholder hides.
    const cleared = new Set(brokenMediaIds.value)
    cleared.delete(message.id)
    brokenMediaIds.value = cleared
  }

  function markMediaBroken(message: Message) {
    if (brokenMediaIds.value.has(message.id)) return
    const next = new Set(brokenMediaIds.value)
    next.add(message.id)
    brokenMediaIds.value = next
  }

  function isRedownloading(message: Message): boolean {
    return !!redownloadingIds.value?.has(message.id)
  }

  function openMediaPreview(message: Message) {
    // The URL carries the display filename as its last segment so the opened
    // tab — and any save/drag-out from it — proposes a real file name instead
    // of the message UUID.
    window.open(mediaUrl(message), '_blank')
  }

  /** Message types that carry a downloadable media file (mirrors the chat
   * renderer's media bubbles; sticker included since it renders as media). */
  function canDownloadMedia(message: Message): boolean {
    return ['image', 'video', 'audio', 'document', 'sticker'].includes(message.message_type)
      || (message.message_type === 'template' && !!message.media_url)
  }

  function isDownloadingMedia(message: Message): boolean {
    return downloadingMessageIds.value.has(message.id)
  }

  /**
   * Per-bubble Download action: fetch the message's media over the
   * authenticated endpoint (which also triggers lazy recovery for
   * history-synced files) and save it under its display filename.
   */
  async function downloadMessageFile(message: Message) {
    if (downloadingMessageIds.value.has(message.id)) return
    const next = new Set(downloadingMessageIds.value)
    next.add(message.id)
    downloadingMessageIds.value = next
    try {
      const res = await fetch(mediaUrl(message), { credentials: 'include' })
      if (!res.ok) {
        throw new Error(`Server responded ${res.status}`)
      }
      const blob = await res.blob()
      saveBlob(blob, mediaDisplayName(message))
    } catch (e: any) {
      toast.error(t('chat.downloadFailed'), { description: e?.message || t('chat.tryAgain') })
    } finally {
      const after = new Set(downloadingMessageIds.value)
      after.delete(message.id)
      downloadingMessageIds.value = after
    }
  }

  function handleImageError(event: Event) {
    const img = event.target as HTMLImageElement
    img.style.display = 'none'
  }

  function handleMediaError(event: Event, mediaType: string) {
    console.error(`Failed to load ${mediaType}:`, event)
  }

  // File upload functions
  function openFilePicker() {
    fileInputRef.value?.click()
  }

  // Per-file validation shared by single and batch picks. Returns the
  // failure reason so the caller can group offending names in one toast.
  // Extension fallback kept in sync with the file input's accept attribute
  // in ChatView.vue (browsers report empty/generic MIME for archives).
  function validateFile(file: File): 'type' | 'size' | null {
    const allowedMimePrefixes = ['image/', 'video/', 'audio/', 'text/', 'application/pdf', 'application/msword', 'application/vnd.openxmlformats-officedocument', 'application/zip', 'application/x-zip-compressed', 'application/x-7z-compressed', 'application/x-rar-compressed', 'application/rar', 'application/rtf', 'application/json', 'application/xml', 'application/vnd.oasis.opendocument']
    const allowedExtensions = ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'csv', 'html', 'htm', 'zip', 'rar', '7z', 'md', 'json', 'xml', 'rtf', 'odt', 'ods', 'odp']
    const ext = (file.name.split('.').pop() || '').toLowerCase()
    const isAllowed = allowedMimePrefixes.some(type => file.type.startsWith(type)) || allowedExtensions.includes(ext)
    if (!isAllowed) return 'type'

    // Size limits aligned with the engine: GOWA enforces a hard 50MB upload
    // limit; media (image/video/audio) stay at WhatsApp's 16MB.
    const isMediaType = file.type.startsWith('image/') || file.type.startsWith('video/') || file.type.startsWith('audio/')
    const maxSize = isMediaType ? 16 * 1024 * 1024 : 50 * 1024 * 1024
    if (file.size > maxSize) return 'size'
    return null
  }

  function revokePreviews() {
    for (const url of filePreviewUrls.value) {
      if (url) URL.revokeObjectURL(url)
    }
    filePreviewUrls.value = []
  }

  function handleFileSelect(event: Event) {
    const input = event.target as HTMLInputElement
    const picked = Array.from(input.files ?? [])
    // Reset input so the same files can be selected again
    input.value = ''
    if (!picked.length) return

    if (picked.length > MAX_BATCH_FILES) {
      toast.error(t('chat.tooManyFiles', { max: MAX_BATCH_FILES }))
      return
    }
    // Validate the whole batch upfront — a batch with any invalid file does
    // not start, so the user fixes the pick instead of getting a partial send.
    const badType = picked.filter((f) => validateFile(f) === 'type')
    if (badType.length) {
      toast.error(t('chat.unsupportedFileType'), {
        description: badType.map((f) => f.name).join(', ')
      })
      return
    }
    const tooBig = picked.filter((f) => validateFile(f) === 'size')
    if (tooBig.length) {
      toast.error(t('chat.fileTooLarge'), {
        description: tooBig.map((f) => f.name).join(', ')
      })
      return
    }

    revokePreviews()
    selectedFiles.value = picked
    activeFileIndex.value = 0
    // Create preview URLs for images and videos only
    filePreviewUrls.value = picked.map((f) =>
      f.type.startsWith('image/') || f.type.startsWith('video/') ? URL.createObjectURL(f) : null
    )
    mediaCaption.value = ''

    isMediaDialogOpen.value = true
  }

  /** Drop a queued file before sending (releases its preview URL). */
  function removeFile(index: number) {
    const remaining = selectedFiles.value.filter((_, i) => i !== index)
    const url = filePreviewUrls.value[index]
    if (url) URL.revokeObjectURL(url)
    filePreviewUrls.value = filePreviewUrls.value.filter((_, i) => i !== index)
    selectedFiles.value = remaining
    if (activeFileIndex.value >= remaining.length) {
      activeFileIndex.value = Math.max(0, remaining.length - 1)
    }
    if (!remaining.length) {
      isMediaDialogOpen.value = false
      mediaCaption.value = ''
    }
  }

  function setActiveFile(index: number) {
    if (index >= 0 && index < selectedFiles.value.length) {
      activeFileIndex.value = index
    }
  }

  function closeMediaDialog() {
    // Cancel during an active upload aborts the transfer itself — the dialog
    // must not stay hostage until the last byte lands (large files can take
    // minutes on slow links). The batch loop observes batchCancelled and
    // stops after the current file.
    if (uploadAbort) {
      batchCancelled = true
      uploadAbort.abort()
      uploadAbort = null
    }
    isMediaDialogOpen.value = false
    revokePreviews()
    selectedFiles.value = []
    activeFileIndex.value = 0
    mediaCaption.value = ''
    uploadCurrent.value = 0
    uploadTotal.value = 0
  }

  /**
   * Upload one queued file as its own message. Throws on failure so the
   * batch loop can record it and continue with the rest. `fileIndex` drives
   * the overall progress bar.
   */
  async function sendSingleFile(file: File, caption: string, fileIndex: number, total: number) {
    uploadAbort = new AbortController()
    try {
      const formData = new FormData()
      formData.append('file', file)
      formData.append('contact_id', contactsStore.currentContact!.id)
      formData.append('type', getMediaType(file.type))
      if (caption) {
        formData.append('caption', caption)
      }
      if (selectedAccount.value) {
        formData.append('whatsapp_account', selectedAccount.value)
      }

      // axios (XHR) instead of raw fetch: only XHR reports upload progress,
      // which the media dialog's progress bar renders. Same credentials/CSRF
      // contract as the fetch path it replaces. The explicit multipart
      // Content-Type is REQUIRED with this axios instance: its default is
      // application/json, and axios converts FormData to a JSON string when
      // that default wins (utils.formDataToJSON) — fasthttp then rejects the
      // body with "Invalid multipart form". The manual header disables the
      // JSON path and the browser adapter swaps in the real boundary (same
      // pattern as importData/uploadMedia/sendTemplate). The signal lets the
      // Cancel button abort a long upload mid-transfer.
      const response = await api.post('/messages/media', formData, {
        signal: uploadAbort.signal,
        headers: { ...getRequestHeaders({ csrf: true }), 'Content-Type': 'multipart/form-data' },
        onUploadProgress: (e) => {
          if (e.total) {
            const frac = Math.min(1, e.loaded / e.total)
            uploadProgress.value = Math.min(100, Math.round(((fileIndex + frac) / total) * 100))
          }
        }
      })

      const result = response.data

      // Add the message to the store (addMessage has duplicate checking for WebSocket)
      if (result.data) {
        contactsStore.addMessage(result.data)
        options.scrollToBottom()
      }
      uploadProgress.value = Math.round(((fileIndex + 1) / total) * 100)
    } finally {
      uploadAbort = null
    }
  }

  async function sendMediaMessage() {
    if (!selectedFiles.value.length || !contactsStore.currentContact) return

    // Status conversation posts media to status@broadcast via a dedicated path
    // that supports a single file — batch sends stay in normal chats.
    if (options.isStatusContact(contactsStore.currentContact.id)) {
      await options.sendStatusMedia(selectedFiles.value[0], mediaCaption.value.trim())
      return
    }

    isUploadingMedia.value = true
    batchCancelled = false
    uploadProgress.value = 0
    const files = [...selectedFiles.value]
    const total = files.length
    uploadTotal.value = total
    // The caption rides on the FIRST file only: any captioned file renders as
    // its own bubble (see useChatAlbums), so this keeps the rest eligible to
    // fold into one album grid.
    const caption = mediaCaption.value.trim()
    const failedIndexes: number[] = []
    try {
      for (let i = 0; i < files.length; i++) {
        if (batchCancelled) break
        uploadCurrent.value = i + 1
        try {
          await sendSingleFile(files[i], i === 0 ? caption : '', i, total)
        } catch (error: any) {
          // A user-initiated abort is not a failure — closeMediaDialog already
          // reset the dialog; stop the loop quietly.
          if (error?.code === 'ERR_CANCELED' || error?.name === 'CanceledError' || batchCancelled) {
            return
          }
          failedIndexes.push(i)
          toast.error(t('chat.mediaFailed'), {
            description: files[i].name
          })
        }
      }
      if (batchCancelled) return
      const sent = total - failedIndexes.length
      if (failedIndexes.length === 0) {
        toast.success(t('chat.mediaSent'))
        closeMediaDialog()
      } else {
        // Keep only the failed files queued so the user can retry them
        // directly; succeeded ones are already bubbles in the room.
        const failedFiles = failedIndexes.map((i) => files[i])
        revokePreviews()
        selectedFiles.value = failedFiles
        activeFileIndex.value = 0
        filePreviewUrls.value = failedFiles.map((f) =>
          f.type.startsWith('image/') || f.type.startsWith('video/') ? URL.createObjectURL(f) : null
        )
        // The caption was consumed by the first file unless it also failed.
        if (!failedIndexes.includes(0)) mediaCaption.value = ''
        toast.error(t('chat.partialSent', { sent, total }))
      }
    } finally {
      isUploadingMedia.value = false
      if (failedIndexes.length === 0) {
        uploadProgress.value = 0
        uploadCurrent.value = 0
        uploadTotal.value = 0
      }
      uploadAbort = null
    }
  }

  return {
    // Upload dialog state (fileInputRef owned by the view — not re-returned)
    selectedFile,
    selectedFiles,
    activeFileIndex,
    filePreviewUrl,
    isMediaDialogOpen,
    mediaCaption,
    isUploadingMedia,
    uploadProgress,
    uploadCurrent,
    uploadTotal,
    // Broken-media / redownload
    brokenMediaIds,
    retryMediaDownload,
    markMediaBroken,
    isRedownloading,
    // Per-bubble file download
    canDownloadMedia,
    isDownloadingMedia,
    downloadMessageFile,
    // Actions
    openFilePicker,
    handleFileSelect,
    removeFile,
    setActiveFile,
    closeMediaDialog,
    sendMediaMessage,
    openMediaPreview,
    handleImageError,
    handleMediaError,
    // Helper (exposed for messaging composable's status path)
    getMediaType,
  }
}
