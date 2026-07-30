import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(date: string | Date, options?: Intl.DateTimeFormatOptions): string {
  const d = typeof date === 'string' ? new Date(date) : date
  return d.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    ...options
  })
}

function formatTime(date: string | Date): string {
  const d = typeof date === 'string' ? new Date(date) : date
  return d.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit'
  })
}

export function formatDateTime(date: string | Date): string {
  return `${formatDate(date)} ${formatTime(date)}`
}

export interface RelativeTimeDiff {
  date: Date
  diffMins: number
  diffHours: number
  diffDays: number
}

// relativeTimeDiff computes the elapsed minutes/hours/days between now and the
// given date. Shared by the relative "x ago" formatters (ConversationNotes,
// DashboardView) which each render the buckets differently (plain text vs
// i18n), so only the duplicated diff computation is factored out here.
export function relativeTimeDiff(date: string | Date): RelativeTimeDiff {
  const d = typeof date === 'string' ? new Date(date) : date
  const diffMs = Date.now() - d.getTime()
  return {
    date: d,
    diffMins: Math.floor(diffMs / 60000),
    diffHours: Math.floor(diffMs / 3600000),
    diffDays: Math.floor(diffMs / 86400000)
  }
}

export function getInitials(name: string): string {
  return name
    .split(' ')
    .map(n => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}

const avatarGradients = [
  'from-violet-500 to-purple-600',
  'from-blue-500 to-cyan-600',
  'from-rose-500 to-pink-600',
  'from-amber-500 to-orange-600',
  'from-emerald-500 to-teal-600',
  'from-indigo-500 to-blue-600',
  'from-fuchsia-500 to-purple-600',
  'from-cyan-500 to-blue-600',
  'from-orange-500 to-red-600',
  'from-teal-500 to-emerald-600',
]

export function getAvatarGradient(name: string): string {
  if (!name) return avatarGradients[0]
  let hash = 0
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash)
  }
  return avatarGradients[Math.abs(hash) % avatarGradients.length]
}

// avatarSrc resolves a contact's avatar_url for use in an <img>/<AvatarImage>.
// Backend now returns a stable relative route (/api/contacts/{id}/avatar/image)
// that must be prefixed with the runtime base path (subpath deployments), while
// absolute http(s) URLs (legacy rows, external avatars) pass through untouched.
// Returns undefined for empty input so the initials fallback renders.
export function avatarSrc(url?: string | null): string | undefined {
  if (!url) return undefined
  if (/^https?:\/\//i.test(url)) return url
  if (url.startsWith('/')) {
    const basePath = ((window as any).__BASE_PATH__ ?? '').replace(/\/$/, '')
    return `${basePath}${url}`
  }
  return url
}

export function formatBytes(bytes: number | undefined | null, decimals = 1): string {
  if (bytes === undefined || bytes === null || isNaN(bytes) || bytes === 0) return '0 B'
  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}

export function formatLabel(key: string): string {
  return key
    .replace(/_/g, ' ')
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .replace(/\b\w/g, c => c.toUpperCase())
}

export interface LinkSegment {
  text: string
  href?: string
}

// Trailing sentence punctuation (and unbalanced closing brackets) is usually
// not part of the URL, e.g. "see https://example.com." or "(https://example.com)".
function trimTrailingPunctuation(url: string): string {
  let result = url
  for (;;) {
    const last = result[result.length - 1]
    if ('.,!?;:\'"'.includes(last)) {
      result = result.slice(0, -1)
      continue
    }
    if (last === ')' && (result.match(/\(/g) || []).length < (result.match(/\)/g) || []).length) {
      result = result.slice(0, -1)
      continue
    }
    if (last === ']' && (result.match(/\[/g) || []).length < (result.match(/\]/g) || []).length) {
      result = result.slice(0, -1)
      continue
    }
    break
  }
  return result
}

// linkifySegments splits message text into plain-text and URL segments so chat
// bubbles can render clickable anchors without resorting to v-html.
export function linkifySegments(text: string): LinkSegment[] {
  if (!text) return []
  const segments: LinkSegment[] = []
  const regex = /(https?:\/\/[^\s<>]+|www\.[^\s<>]+)/gi
  let cursor = 0
  let match: RegExpExecArray | null
  while ((match = regex.exec(text)) !== null) {
    const url = trimTrailingPunctuation(match[0])
    if (!url) continue
    if (match.index > cursor) {
      segments.push({ text: text.slice(cursor, match.index) })
    }
    segments.push({
      text: url,
      href: url.toLowerCase().startsWith('www.') ? `https://${url}` : url
    })
    cursor = match.index + url.length
    regex.lastIndex = cursor
  }
  if (cursor < text.length) {
    segments.push({ text: text.slice(cursor) })
  }
  return segments
}

