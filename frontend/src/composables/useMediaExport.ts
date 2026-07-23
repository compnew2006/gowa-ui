import { ref } from 'vue'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import type { Message } from '@/stores/contacts'
import { getRequestHeaders } from '@/services/api'
import { useAuthStore } from '@/stores/auth'

export interface RedownloadResult {
  ok: boolean
  /** Updated media fields to patch into the message, when ok. */
  mediaUrl?: string
  mediaMimeType?: string
}

export interface UseMediaExportResult {
  /** True while a download (zip or sequential) is in progress. */
  isDownloading: ReturnType<typeof ref<boolean>>
  /** `{ current, total }` progress for UI display. */
  progress: ReturnType<typeof ref<{ current: number; total: number }>>
  /** Set of message IDs currently being re-downloaded (per-message spinners). */
  redownloading: ReturnType<typeof ref<Set<string>>>
  /** Download the given messages as a single server-built ZIP. */
  downloadAsZip: (messages: Message[]) => Promise<void>
  /** Download each message's media as a separate file, sequentially. */
  downloadSeparately: (messages: Message[]) => Promise<void>
  /** Re-fetch a message's media from the provider. Returns updated fields. */
  redownload: (message: Message) => Promise<RedownloadResult>
}

/** Gap (ms) between separate downloads so the browser doesn't merge them. */
const SEPARATE_DOWNLOAD_GAP_MS = 450

/**
 * Orchestrates downloading a burst of media files — either bundled as a ZIP
 * (built server-side) or as separate sequential downloads. Mirrors the
 * anchor-click download idiom used throughout the app (see
 * ImportExportDialog.vue) and reuses the existing per-file authenticated
 * endpoint, so no new auth path is required.
 */
export function useMediaExport(): UseMediaExportResult {
  const { t } = useI18n()
  const authStore = useAuthStore()
  const isDownloading = ref(false)
  const progress = ref({ current: 0, total: 0 })
  const redownloading = ref(new Set<string>())
  async function downloadAsZip(messages: Message[]): Promise<void> {
    // Defense-in-depth: verify export permission before fetching (FR-013).
    // The backend also gates on contacts:export, but this prevents a wasted
    // round-trip and shows a toast immediately.
    if (!authStore.hasPermission('contacts', 'export')) {
      toast.error('You do not have permission to export media')
      return
    }
    const eligible = messages.filter((m) => !!m.media_url)
    if (eligible.length === 0) return

    isDownloading.value = true
    progress.value = { current: 0, total: eligible.length }
    try {
      const ids = eligible.map((m) => m.id).join(',')
      const url = `${getBasePath()}/api/media/zip?ids=${encodeURIComponent(ids)}`
      const res = await fetch(url, { credentials: 'include' })
      if (!res.ok) {
        throw new Error(`Server responded ${res.status}`)
      }
      const blob = await res.blob()
      progress.value = { current: eligible.length, total: eligible.length }
      saveBlob(blob, zipFilename())
      toast.success('Files downloaded')
    } catch (e: any) {
      toast.error('Download failed', { description: e?.message || 'Please try again' })
    } finally {
      isDownloading.value = false
    }
  }

  async function downloadSeparately(messages: Message[]): Promise<void> {
    const eligible = messages.filter((m) => !!m.media_url)
    if (eligible.length === 0) return

    isDownloading.value = true
    progress.value = { current: 0, total: eligible.length }
    try {
      for (let i = 0; i < eligible.length; i++) {
        const message = eligible[i]
        const url = mediaUrlFor(message)
        const res = await fetch(url, { credentials: 'include' })
        if (!res.ok) {
          throw new Error(`Server responded ${res.status}`)
        }
        const blob = await res.blob()
        saveBlob(blob, message.media_filename || defaultFilenameFor(message))
        progress.value = { current: i + 1, total: eligible.length }
        if (i < eligible.length - 1) {
          await delay(SEPARATE_DOWNLOAD_GAP_MS)
        }
      }
      toast.success('Files downloaded')
    } catch (e: any) {
      toast.error('Download failed', { description: e?.message || 'Please try again' })
    } finally {
      isDownloading.value = false
    }
  }

  async function redownload(message: Message): Promise<RedownloadResult> {
    // Track this message's in-flight state for an independent per-bubble spinner.
    const next = new Set(redownloading.value)
    next.add(message.id)
    redownloading.value = next
    try {
      const res = await fetch(`${getBasePath()}/api/media/${message.id}/redownload`, {
        method: 'POST',
        credentials: 'include',
        headers: getRequestHeaders({ csrf: true })
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        // Map the backend's plain message to a translated, user-facing one.
        throw new Error(body?.message || `Server responded ${res.status}`)
      }
      const j = await res.json()
      const data = j?.data ?? j
      toast.success(t('chat.retrySuccess'))
      return {
        ok: true,
        mediaUrl: data?.media_url,
        mediaMimeType: data?.media_type
      }
    } catch (e: any) {
      const { title, desc } = translateRedownloadError(e?.message, t)
      toast.error(title, { description: desc })
      return { ok: false }
    } finally {
      const after = new Set(redownloading.value)
      after.delete(message.id)
      redownloading.value = after
    }
  }

  return { isDownloading, progress, redownloading, downloadAsZip, downloadSeparately, redownload }
}

/** Resolve the app base path injected at runtime (mirrors getMediaUrl in ChatView). */
function getBasePath(): string {
  return ((window as any).__BASE_PATH__ ?? '').replace(/\/$/, '')
}

/** Per-file authenticated media URL (identical to ChatView.getMediaUrl). */
export function mediaUrlFor(message: Message): string {
  return `${getBasePath()}/api/media/${message.id}`
}

/** Trigger a browser download from a Blob (the app's standard idiom). */
function saveBlob(blob: Blob, filename: string): void {
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}

function zipFilename(): string {
  const stamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
  return `files_${stamp}.zip`
}

function defaultFilenameFor(message: Message): string {
  const type = message.message_type || 'file'
  return `${type}_${message.id.slice(0, 8)}`
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/**
 * Map a backend redownload error message to a translated, user-facing title
 * and description. The raw provider payload stays server-side; the user only
 * ever sees a short, friendly sentence in their language.
 */
function translateRedownloadError(
  raw: string | undefined,
  t: (key: string, named?: Record<string, unknown>) => string
): { title: string; desc: string } {
  const low = (raw || '').toLowerCase()
  // Media is permanently gone from the provider.
  const gone =
    low.includes('no longer available') ||
    low.includes('does not belong') ||
    low.includes('not found') ||
    low.includes('expired') ||
    low.includes('deleted') ||
    low.includes('cannot be recovered')
  if (gone) {
    return { title: t('chat.mediaGone'), desc: t('chat.mediaGoneDesc') }
  }
  // Anything else is transient.
  return { title: t('chat.retryFailed'), desc: t('chat.retryFailedDesc') }
}
