import type { Message } from '@/stores/contacts'

/**
 * Media URL + display-name helpers, shared by the chat bubbles
 * (useMessageFormat), the preview tab (useChatMedia), and the burst download
 * (useMediaExport) so every /api/media request carries the same filename.
 *
 * The filename is appended as a decorative last path segment: the backend
 * ignores it, but it makes the URL's basename a real file name — which is
 * what browsers propose for "Save Image As" / drag-out from the preview tab
 * (the Content-Disposition header alone is not honored there, notably in
 * Safari) — and it versioned the URL when naming was introduced, busting
 * stale cached responses that predate the header.
 */

// Mirrors mimeExts in internal/handlers/media.go — keep both in sync.
const MIME_EXTS: Array<[string, string]> = [
  ['image/jpeg', '.jpg'],
  ['image/png', '.png'],
  ['image/gif', '.gif'],
  ['image/webp', '.webp'],
  ['video/mp4', '.mp4'],
  ['video/3gpp', '.3gp'],
  ['audio/aac', '.aac'],
  ['audio/mp4', '.m4a'],
  ['audio/mpeg', '.mp3'],
  ['audio/amr', '.amr'],
  ['audio/ogg', '.ogg'],
  ['application/pdf', '.pdf'],
  ['text/plain', '.txt'],
]

// Fallback extension per message type when the MIME is missing/unknown.
const TYPE_EXTS: Record<string, string> = {
  image: '.jpg',
  sticker: '.webp',
  video: '.mp4',
  audio: '.mp3',
}

function extForMessage(message: Pick<Message, 'media_mime_type' | 'message_type'>): string {
  const mime = (message.media_mime_type || '').split(';')[0].trim().toLowerCase()
  for (const [prefix, ext] of MIME_EXTS) {
    if (mime.startsWith(prefix)) return ext
  }
  return TYPE_EXTS[message.message_type] || '.bin'
}

/**
 * The file name a download/save should propose for a message's media: the
 * original WhatsApp filename when stored, else `<type>_<short-id><ext>`
 * (same rule as the backend's defaultZipEntryName).
 */
export function mediaDisplayName(
  message: Pick<Message, 'id' | 'media_filename' | 'media_mime_type' | 'message_type'>
): string {
  const name = (message.media_filename || '').trim()
  if (name) return name
  const ext = extForMessage(message)
  return `${message.message_type || 'file'}_${message.id.slice(0, 8)}${ext}`
}

/**
 * Per-message authenticated media URL with the display filename as the last
 * path segment (identical contract to the previous `/api/media/{id}` URL —
 * the backend resolves access by message id and ignores the name).
 */
export function mediaUrl(message: Pick<Message, 'id' | 'media_filename' | 'media_mime_type' | 'message_type'>): string {
  const basePath = ((window as any).__BASE_PATH__ ?? '').replace(/\/$/, '')
  return `${basePath}/api/media/${message.id}/${encodeURIComponent(mediaDisplayName(message))}`
}

/**
 * Trigger a browser download from a Blob under the given filename (the app's
 * standard idiom — shared by the per-bubble download button and the burst
 * download).
 */
export function saveBlob(blob: Blob, filename: string): void {
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}
