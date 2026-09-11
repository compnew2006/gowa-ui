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

export function formatLabel(key?: string | null): string {
  if (!key) return ''
  return key
    .replace(/_/g, ' ')
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .replace(/\b\w/g, c => c.toUpperCase())
}

export interface AuditChange {
  field: string
  old_value?: any
  new_value?: any
}

// normalizeAuditChanges flattens audit-log change entries into
// {field, old_value, new_value} records. Modern entries already have a flat
// `field` key; legacy entries are nested maps ({key: {old, new}, ...}) written
// by an older audit implementation and are expanded one record per key.
export function normalizeAuditChanges(changes?: any[] | null): AuditChange[] {
  if (!Array.isArray(changes)) return []
  const out: AuditChange[] = []
  for (const c of changes) {
    if (c && typeof c === 'object' && typeof c.field === 'string') {
      out.push({ field: c.field, old_value: c.old_value, new_value: c.new_value })
      continue
    }
    if (c && typeof c === 'object') {
      for (const [key, v] of Object.entries(c)) {
        if (v && typeof v === 'object') {
          out.push({ field: key, old_value: (v as any).old, new_value: (v as any).new })
        } else {
          out.push({ field: key, old_value: undefined, new_value: v })
        }
      }
    }
  }
  return out
}

export interface LinkSegment {
  text: string
  href?: string
  /** 'phone' segments carry normalized digits for conversation lookup. */
  kind?: 'url' | 'phone'
  phone?: string
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

// normalizePhoneDigits strips everything except digits (converting a leading
// 00 to its implicit + form) and returns '' when the result is outside the
// 7..15 E.164 digit range. WhatsApp group IDs (120362…/120363…) are not
// dialable numbers and are rejected too.
export function normalizePhoneDigits(raw: string): string {
  if (!raw) return ''
  let s = raw.trim()
  if (s.startsWith('00')) s = s.slice(2)
  const digits = s.replace(/\D/g, '')
  if (digits.length < 7 || digits.length > 15) return ''
  if (/^12036[23]/.test(digits)) return ''
  return digits
}

// samePhoneDigits reports whether two normalized digit strings belong to the
// same conversation: exact match, or a shared trailing-9-digit suffix (handles
// local 05xxxxxxxx vs international 9665xxxxxxxx variants without ever
// mismatching short codes — suffix match needs 9 overlapping digits).
export function samePhoneDigits(a: string, b: string): boolean {
  if (!a || !b) return false
  if (a === b) return true
  if (Math.min(a.length, b.length) < 9) return false
  return a.slice(-9) === b.slice(-9)
}

// phoneSearchVariants builds server-search strings for a normalized number.
// Stored numbers are usually international (9665…) while message text often
// carries local format (05…); a raw LIKE '%05…%' never matches '9665…', so
// the leading-zero-stripped form (a substring of both formats) comes first.
export function phoneSearchVariants(digits: string): string[] {
  const out: string[] = []
  const stripped = digits.replace(/^0+/, '')
  if (stripped.length >= 7 && stripped !== digits) out.push(stripped)
  out.push(digits)
  return out
}

// looksLikePhoneNumber mirrors internal/utils/phone.go LooksLikePhoneNumber
// (at least 7 digits, digits > 70% of the candidate) so frontend and backend
// agree on what counts as a phone number.
function looksLikePhoneNumber(candidate: string): boolean {
  const digits = (candidate.match(/\d/g) || []).length
  if (digits < 7 || digits > 15) return false
  if (digits / candidate.length <= 0.7) return false
  // Dates (2026-09-11, 11/09/2026) pass the ratio test but are not phones.
  if (/^\d{4}[-/.]\d{1,2}[-/.]\d{1,2}$/.test(candidate)) return false
  if (/^\d{1,2}[-/.]\d{1,2}[-/.]\d{2,4}$/.test(candidate)) return false
  return true
}

// splitPhones sub-splits a plain-text chunk on phone-number candidates.
// Candidates may carry +, spaces, dashes, dots or (parentheses) and must
// start/end with a digit so surrounding punctuation stays plain text.
function splitPhones(text: string): LinkSegment[] {
  const segments: LinkSegment[] = []
  const regex = /(?:\+|00)?\d[\d\s\-().]{5,}\d/g
  let cursor = 0
  let match: RegExpExecArray | null
  while ((match = regex.exec(text)) !== null) {
    const candidate = match[0]
    const digits = normalizePhoneDigits(candidate)
    if (!digits || !looksLikePhoneNumber(candidate)) continue
    if (match.index > cursor) {
      segments.push({ text: text.slice(cursor, match.index) })
    }
    segments.push({ text: candidate, kind: 'phone', phone: digits })
    cursor = match.index + candidate.length
    regex.lastIndex = cursor
  }
  if (cursor < text.length) {
    segments.push({ text: text.slice(cursor) })
  }
  // No phone found — return the chunk untouched (same shape as before).
  if (segments.length === 0) return [{ text }]
  return segments
}

// linkifySegments splits message text into plain-text, URL and phone-number
// segments so chat bubbles can render clickable anchors without v-html.
// URLs are extracted first (behavior unchanged); phone detection only runs
// on the remaining plain-text chunks so it can never hijack a URL.
export function linkifySegments(text: string): LinkSegment[] {
  if (!text) return []
  const raw: LinkSegment[] = []
  const regex = /(https?:\/\/[^\s<>]+|www\.[^\s<>]+)/gi
  let cursor = 0
  let match: RegExpExecArray | null
  while ((match = regex.exec(text)) !== null) {
    const url = trimTrailingPunctuation(match[0])
    if (!url) continue
    if (match.index > cursor) {
      raw.push({ text: text.slice(cursor, match.index) })
    }
    raw.push({
      text: url,
      kind: 'url',
      href: url.toLowerCase().startsWith('www.') ? `https://${url}` : url
    })
    cursor = match.index + url.length
    regex.lastIndex = cursor
  }
  if (cursor < text.length) {
    raw.push({ text: text.slice(cursor) })
  }
  return raw.flatMap(seg => (seg.href ? [seg] : splitPhones(seg.text)))
}

