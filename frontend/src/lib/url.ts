export function normalizeBaseUrl(url: string): string {
  let cleaned = url.trim()
  if (!cleaned) return ''
  if (!cleaned.startsWith('http://') && !cleaned.startsWith('https://')) {
    cleaned = `http://${cleaned}`
  }
  return cleaned.replace(/\/+$/, '')
}

export function sameOriginBaseUrl(): string {
  if (typeof window !== 'undefined' && window.location) {
    return window.location.origin
  }
  return 'http://localhost:3000'
}
