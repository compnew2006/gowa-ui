import { ref } from 'vue'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import type { Message } from '@/stores/contacts'
import { getRequestHeaders } from '@/services/api'
import { mediaDisplayName, mediaUrl, saveBlob } from '@/lib/media'

interface RedownloadResult {
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
  /** Download the given messages as a single server-built ZIP. Resolves true
   *  only when the archive was actually saved (used to "seal" the album so
   *  later files group separately — sealing never blocks re-download). */
  downloadAsZip: (messages: Message[]) => Promise<boolean>
  /** Download each message's media as a separate file, sequentially.
   *  Resolves true only when every file was saved. */
  downloadSeparately: (messages: Message[]) => Promise<boolean>
  /** Re-fetch a message's media from the provider. Returns updated fields. */
  redownload: (message: Message) => Promise<RedownloadResult>
}

/** Gap (ms) between separate downloads so the browser doesn't merge them. */
const SEPARATE_DOWNLOAD_GAP_MS = 450

/** Message types whose media the server can serve, even when the message's
 *  media_url is still empty in the store (a download that failed at ingest
 *  or a history-synced file is recovered lazily server-side). */
const MEDIA_TYPES = ['image', 'video', 'audio', 'document', 'sticker']

/** Messages that carry a downloadable file. Deliberately NOT keyed on
 *  media_url alone: that dropped not-yet-downloaded files from album
 *  downloads although the chat showed them. */
export function downloadableMedia(messages: Message[]): Message[] {
  return messages.filter((m) => !!m.media_url || MEDIA_TYPES.includes(m.message_type))
}

/**
 * Orchestrates downloading a burst of media files — either bundled as a ZIP
 * (built server-side) or as separate sequential downloads. Mirrors the
 * anchor-click download idiom used throughout the app (see
 * ImportExportDialog.vue) and reuses the existing per-file authenticated
 * endpoint, so no new auth path is required.
 */
export function useMediaExport(): UseMediaExportResult {
  const { t } = useI18n()
  const isDownloading = ref(false)
  const progress = ref({ current: 0, total: 0 })
  const redownloading = ref(new Set<string>())
  async function downloadAsZip(messages: Message[]): Promise<boolean> {
    // No export-permission pre-check here: the ZIP bundles the same files the
    // per-message endpoint already serves to anyone who can view the chat —
    // the backend enforces identical org + visibility scoping on /api/media/zip.
    const eligible = downloadableMedia(messages)
    if (eligible.length === 0) return false

    isDownloading.value = true
    progress.value = { current: 0, total: eligible.length }
    try {
      // IDs travel in a JSON body: an album can hold hundreds of files, far
      // past what a query string comfortably carries.
      const res = await fetch(`${getBasePath()}/api/media/zip`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', ...getRequestHeaders({ csrf: true }) },
        body: JSON.stringify({ ids: eligible.map((m) => m.id) })
      })
      if (!res.ok) {
        throw new Error(`Server responded ${res.status}`)
      }
      const blob = await res.blob()
      progress.value = { current: eligible.length, total: eligible.length }
      saveBlob(blob, zipFilename())
      // The server reports files it could not include (gone from the
      // provider, retention-purged, over the size budget) — say so instead of
      // silently delivering fewer files than the chat shows.
      const included = Number(res.headers.get('X-Media-Zip-Included'))
      const skipped = Number(res.headers.get('X-Media-Zip-Skipped'))
      if (skipped > 0 && Number.isFinite(included)) {
        toast.warning(t('chat.zipPartial', { included, total: included + skipped }), {
          description: t('chat.zipPartialDesc')
        })
      } else {
        toast.success(t('chat.filesDownloaded'))
      }
      return true
    } catch (e: any) {
      toast.error(t('chat.downloadFailed'), { description: e?.message || t('chat.tryAgain') })
      return false
    } finally {
      isDownloading.value = false
    }
  }

  async function downloadSeparately(messages: Message[]): Promise<boolean> {
    const eligible = downloadableMedia(messages)
    if (eligible.length === 0) return false

    isDownloading.value = true
    progress.value = { current: 0, total: eligible.length }
    let failed = 0
    let lastError = ''
    try {
      for (let i = 0; i < eligible.length; i++) {
        const message = eligible[i]
        // One unavailable file must not abort the rest of the batch.
        try {
          const res = await fetch(mediaUrl(message), { credentials: 'include' })
          if (!res.ok) {
            throw new Error(`Server responded ${res.status}`)
          }
          const blob = await res.blob()
          saveBlob(blob, mediaDisplayName(message))
        } catch (e: any) {
          failed++
          lastError = e?.message || ''
        }
        progress.value = { current: i + 1, total: eligible.length }
        if (i < eligible.length - 1) {
          await delay(SEPARATE_DOWNLOAD_GAP_MS)
        }
      }
      if (failed === eligible.length) {
        toast.error(t('chat.downloadFailed'), { description: lastError || t('chat.tryAgain') })
      } else if (failed > 0) {
        toast.warning(t('chat.zipPartial', { included: eligible.length - failed, total: eligible.length }), {
          description: t('chat.zipPartialDesc')
        })
      } else {
        toast.success(t('chat.filesDownloaded'))
      }
      return failed === 0
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

function zipFilename(): string {
  const stamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
  return `files_${stamp}.zip`
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
