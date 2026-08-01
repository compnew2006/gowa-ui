import { ref } from 'vue'
import type { Ref } from 'vue'
import { toast } from 'vue-sonner'
import { getRequestHeaders } from '@/services/api'
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

    // Validate file type
    const allowedTypes = ['image/', 'video/', 'audio/', 'application/pdf', 'application/msword', 'application/vnd.openxmlformats-officedocument']
    const isAllowed = allowedTypes.some(type => file.type.startsWith(type))
    if (!isAllowed) {
      toast.error(t('chat.unsupportedFileType'), {
        description: t('chat.unsupportedFileTypeDesc')
      })
      return
    }

    // Validate file size (16MB limit for WhatsApp)
    const maxSize = 16 * 1024 * 1024
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

      const basePath = ((window as any).__BASE_PATH__ ?? '').replace(/\/$/, '')
      const response = await fetch(`${basePath}/api/messages/media`, {
        method: 'POST',
        credentials: 'include',
        headers: getRequestHeaders({ csrf: true }),
        body: formData
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.message || 'Failed to send media')
      }

      const result = await response.json()

      // Add the message to the store (addMessage has duplicate checking for WebSocket)
      if (result.data) {
        contactsStore.addMessage(result.data)
        options.scrollToBottom()
      }

      toast.success(t('chat.mediaSent'))
      closeMediaDialog()
    } catch (error: any) {
      toast.error(t('chat.mediaFailed'), {
        description: error.message || getErrorMessage(error, t('chat.mediaFailedDesc'))
      })
    } finally {
      isUploadingMedia.value = false
    }
  }

  return {
    // Upload dialog state (fileInputRef owned by the view — not re-returned)
    selectedFile,
    filePreviewUrl,
    isMediaDialogOpen,
    mediaCaption,
    isUploadingMedia,
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
