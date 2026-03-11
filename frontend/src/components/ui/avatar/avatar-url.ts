const WHATSAPP_AVATAR_HOST = 'pps.whatsapp.net'
const WHATSAPP_URL_EXPIRY_PARAM = 'oe'
const WHATSAPP_URL_EXPIRY_HEX = /^[0-9a-fA-F]{8}$/
const URL_EXPIRY_CLOCK_SKEW_SECONDS = 30

export function normalizeRenderableAvatarURL(raw: string | null | undefined): string {
  const trimmed = String(raw ?? '').trim()
  if (trimmed === '') return ''
  if (isExpiredWhatsAppAvatarURL(trimmed)) return ''
  return trimmed
}

export function isExpiredWhatsAppAvatarURL(url: string, nowMs: number = Date.now()): boolean {
  let parsedURL: URL
  try {
    parsedURL = new URL(url)
  } catch {
    return false
  }

  if (parsedURL.hostname !== WHATSAPP_AVATAR_HOST) {
    return false
  }

  const rawExpiry = parsedURL.searchParams.get(WHATSAPP_URL_EXPIRY_PARAM)
  if (rawExpiry === null || !WHATSAPP_URL_EXPIRY_HEX.test(rawExpiry)) {
    return false
  }

  const expiresAtSeconds = Number.parseInt(rawExpiry, 16)
  if (!Number.isFinite(expiresAtSeconds) || expiresAtSeconds <= 0) {
    return false
  }

  const nowSeconds = Math.floor(nowMs / 1000)
  return expiresAtSeconds <= nowSeconds+URL_EXPIRY_CLOCK_SKEW_SECONDS
}
