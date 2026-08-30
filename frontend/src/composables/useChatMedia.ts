import { ref } from 'vue'
import type { Ref } from 'vue'
import { toast } from 'vue-sonner'
import { api, getRequestHeaders } from '@/services/api'
import { getErrorMessage } from '@/lib/api-utils'
import type { Message } from '@/stores/contacts'

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

  // File upload state (fileInputRef is owned by the view, passed in)
  const fileInputRef = options.fileInputRef
  const selectedFile = ref<File | null>(null)
  const filePreviewUrl = ref<string | null>(null)
  const isMediaDialogOpen = ref(false)
  const mediaCaption = ref('')
  const isUploadingMedia = ref(false)
  // 0..100 upload progress (bytes sent to the server). Reaches 100 while the
  // server still processes (GOWA send), during which isUploadingMedia keeps
  // the "sending" state on the button.
  const uploadProgress = ref(0)
  // Aborts the in-flight upload when the user cancels mid-transfer. Non-null
  // only while an upload request is running.
  let uploadAbort: AbortController | null = null

  // Messages whose media failed to load in the DOM (video error, image error).
  // Keyed by message id so the "Retry download" affordance only shows on broken bubbles.
  const brokenMediaIds = ref(new Set<string>())

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
    const basePath = ((window as any).__BASE_PATH__ ?? '').replace(/\/$/, '')
    const url = `${basePath}/api/media/${message.id}`
    if (url) {
      window.open(url, '_blank')
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

  function handleFileSelect(event: Event) {
    const input = event.target as HTMLInputElement
    const file = input.files?.[0]
    if (!file) return

    // Validate file type. The MIME check alone is not enough: browsers report
    // empty or generic application/octet-stream MIME for archives (.zip/.rar/
    // .7z) on several platforms, which used to reject valid documents. Fall
    // back to the file extension — kept in sync with the file input's accept
    // attribute in ChatView.vue.
    const allowedMimePrefixes = ['image/', 'video/', 'audio/', 'text/', 'application/pdf', 'application/msword', 'application/vnd.openxmlformats-officedocument', 'application/zip', 'application/x-zip-compressed', 'application/x-7z-compressed', 'application/x-rar-compressed', 'application/rar', 'application/rtf', 'application/json', 'application/xml', 'application/vnd.oasis.opendocument']
    const allowedExtensions = ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'csv', 'html', 'htm', 'zip', 'rar', '7z', 'md', 'json', 'xml', 'rtf', 'odt', 'ods', 'odp']
    const ext = (file.name.split('.').pop() || '').toLowerCase()
    const isAllowed = allowedMimePrefixes.some(type => file.type.startsWith(type)) || allowedExtensions.includes(ext)
    if (!isAllowed) {
      toast.error(t('chat.unsupportedFileType'), {
        description: t('chat.unsupportedFileTypeDesc')
      })
      return
    }

    // Validate file size — aligned with the engine: GOWA (go-whatsapp-web-
    // multidevice) enforces a hard 50MB upload limit ("max file upload is
    // 50 MB"); media (image/video/audio) stay at WhatsApp's 16MB. Everything
    // upstream (nginx 110M, server body 110MB) only provides headroom.
    const isMediaType = file.type.startsWith('image/') || file.type.startsWith('video/') || file.type.startsWith('audio/')
    const maxSize = isMediaType ? 16 * 1024 * 1024 : 50 * 1024 * 1024
    if (file.size > maxSize) {
      toast.error(t('chat.fileTooLarge'), {
        description: t('chat.fileTooLargeDesc')
      })
      return
    }

    selectedFile.value = file
    mediaCaption.value = ''

    // Create preview URL for images and videos
    if (file.type.startsWith('image/') || file.type.startsWith('video/')) {
      filePreviewUrl.value = URL.createObjectURL(file)
    } else {
      filePreviewUrl.value = null
    }

    isMediaDialogOpen.value = true

    // Reset input so same file can be selected again
    input.value = ''
  }

  function closeMediaDialog() {
    // Cancel during an active upload aborts the transfer itself — the dialog
    // must not stay hostage until the last byte lands (large files can take
    // minutes on slow links).
    if (uploadAbort) {
      uploadAbort.abort()
      uploadAbort = null
    }
    isMediaDialogOpen.value = false
    if (filePreviewUrl.value) {
      URL.revokeObjectURL(filePreviewUrl.value)
      filePreviewUrl.value = null
    }
    selectedFile.value = null
    mediaCaption.value = ''
  }

  async function sendMediaMessage() {
    if (!selectedFile.value || !contactsStore.currentContact) return

    // Status conversation posts media to status@broadcast via a dedicated path.
    if (options.isStatusContact(contactsStore.currentContact.id)) {
      await options.sendStatusMedia(selectedFile.value, mediaCaption.value.trim())
      return
    }

    isUploadingMedia.value = true
    uploadProgress.value = 0
    uploadAbort = new AbortController()
    try {
      const formData = new FormData()
      formData.append('file', selectedFile.value)
      formData.append('contact_id', contactsStore.currentContact.id)
      formData.append('type', getMediaType(selectedFile.value.type))
      if (mediaCaption.value.trim()) {
        formData.append('caption', mediaCaption.value.trim())
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
            uploadProgress.value = Math.min(100, Math.round((e.loaded / e.total) * 100))
          }
        }
      })

      const result = response.data

      // Add the message to the store (addMessage has duplicate checking for WebSocket)
      if (result.data) {
        contactsStore.addMessage(result.data)
        options.scrollToBottom()
      }

      toast.success(t('chat.mediaSent'))
      closeMediaDialog()
    } catch (error: any) {
      // A user-initiated abort is not a failure — closeMediaDialog already
      // reset the dialog; an error toast for it would be noise.
      if (error?.code === 'ERR_CANCELED' || error?.name === 'CanceledError') {
        return
      }
      toast.error(t('chat.mediaFailed'), {
        description: error?.response?.data?.message || error.message || getErrorMessage(error, t('chat.mediaFailedDesc'))
      })
    } finally {
      isUploadingMedia.value = false
      uploadProgress.value = 0
      uploadAbort = null
    }
  }

  return {
    // Upload dialog state (fileInputRef owned by the view — not re-returned)
    selectedFile,
    filePreviewUrl,
    isMediaDialogOpen,
    mediaCaption,
    isUploadingMedia,
    uploadProgress,
    // Broken-media / redownload
    brokenMediaIds,
    retryMediaDownload,
    markMediaBroken,
    isRedownloading,
    // Actions
    openFilePicker,
    handleFileSelect,
    closeMediaDialog,
    sendMediaMessage,
    openMediaPreview,
    handleImageError,
    handleMediaError,
    // Helper (exposed for messaging composable's status path)
    getMediaType,
  }
}
