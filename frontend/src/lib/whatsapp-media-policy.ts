export type WhatsAppMediaCategory = 'image' | 'video' | 'audio' | 'document'

const BYTES_PER_MB = 1024 * 1024
const APPLICATION_OCTET_STREAM = 'application/octet-stream'

const IMAGE_MIME_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp'])
const VIDEO_MIME_TYPES = new Set(['video/mp4', 'video/3gpp'])
const AUDIO_MIME_TYPES = new Set(['audio/aac', 'audio/amr', 'audio/mpeg', 'audio/mp4', 'audio/ogg'])

const EXTENSION_TO_MIME: Record<string, string> = {
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.png': 'image/png',
  '.webp': 'image/webp',
  '.mp4': 'video/mp4',
  '.3gp': 'video/3gpp',
  '.3gpp': 'video/3gpp',
  '.aac': 'audio/aac',
  '.amr': 'audio/amr',
  '.mp3': 'audio/mpeg',
  '.m4a': 'audio/mp4',
  '.ogg': 'audio/ogg',
}

const MAX_BYTES_BY_CATEGORY: Record<WhatsAppMediaCategory, number> = {
  image: 5 * BYTES_PER_MB,
  video: 16 * BYTES_PER_MB,
  audio: 16 * BYTES_PER_MB,
  document: 100 * BYTES_PER_MB,
}

function normalizeMimeType(value: string | null | undefined): string {
  const raw = String(value ?? '').trim().toLowerCase()
  if (!raw) return ''

  const [mediaType] = raw.split(';', 1)
  return mediaType.trim()
}

function mimeTypeFromFilename(filename: string): string {
  const trimmedName = String(filename || '').trim().toLowerCase()
  if (!trimmedName) return ''

  const dotIndex = trimmedName.lastIndexOf('.')
  if (dotIndex < 0) return ''

  const extension = trimmedName.slice(dotIndex)
  return EXTENSION_TO_MIME[extension] || ''
}

function categoryFromMimeType(mimeType: string): WhatsAppMediaCategory {
  if (IMAGE_MIME_TYPES.has(mimeType)) return 'image'
  if (VIDEO_MIME_TYPES.has(mimeType)) return 'video'
  if (AUDIO_MIME_TYPES.has(mimeType)) return 'audio'
  return 'document'
}

export function resolveWhatsAppMediaCategory(input: {
  mimeType?: string | null
  filename?: string | null
}): WhatsAppMediaCategory {
  const normalizedMime = normalizeMimeType(input.mimeType)
  if (normalizedMime && normalizedMime !== APPLICATION_OCTET_STREAM) {
    return categoryFromMimeType(normalizedMime)
  }

  const extensionMimeType = mimeTypeFromFilename(String(input.filename || ''))
  if (extensionMimeType) {
    return categoryFromMimeType(extensionMimeType)
  }

  return 'document'
}

export function resolveWhatsAppMediaCategoryForFile(file: Pick<File, 'type' | 'name'>): WhatsAppMediaCategory {
  return resolveWhatsAppMediaCategory({
    mimeType: file.type,
    filename: file.name,
  })
}

export function getWhatsAppMediaMaxSizeBytes(category: WhatsAppMediaCategory): number {
  return MAX_BYTES_BY_CATEGORY[category]
}

export function getWhatsAppMediaMaxSizeMB(category: WhatsAppMediaCategory): number {
  return getWhatsAppMediaMaxSizeBytes(category) / BYTES_PER_MB
}

export interface WhatsAppMediaValidationResult {
  category: WhatsAppMediaCategory
  fileSizeBytes: number
  maxSizeBytes: number
  maxSizeMB: number
  isValid: boolean
  errorCode?: 'FILE_TOO_LARGE'
}

export function validateWhatsAppMediaFile(file: Pick<File, 'size' | 'type' | 'name'>): WhatsAppMediaValidationResult {
  const category = resolveWhatsAppMediaCategoryForFile(file)
  const maxSizeBytes = getWhatsAppMediaMaxSizeBytes(category)
  const fileSizeBytes = Number(file.size)
  const isValid = fileSizeBytes <= maxSizeBytes

  return {
    category,
    fileSizeBytes,
    maxSizeBytes,
    maxSizeMB: getWhatsAppMediaMaxSizeMB(category),
    isValid,
    errorCode: isValid ? undefined : 'FILE_TOO_LARGE',
  }
}
