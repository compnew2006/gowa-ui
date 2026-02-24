export type AutoCampaignTargetStatus = 'draft' | 'run'

export interface AutoCampaignSettings {
  enabled: boolean
  name_prefix: string
  message: string
  interval_days: number
  min_delay_minutes: number
  max_delay_minutes: number
  target_status: AutoCampaignTargetStatus
  media_local_path: string
  media_mime_type: string
  media_filename: string
  last_generated_at?: string
}

export const DEFAULT_AUTO_CAMPAIGN_SETTINGS: AutoCampaignSettings = {
  enabled: false,
  name_prefix: '',
  message: '',
  interval_days: 7,
  min_delay_minutes: 0,
  max_delay_minutes: 0,
  target_status: 'draft',
  media_local_path: '',
  media_mime_type: '',
  media_filename: ''
}

export function normalizeAutoCampaignSettings(raw: unknown): AutoCampaignSettings {
  const settings = cloneAutoCampaignSettings(DEFAULT_AUTO_CAMPAIGN_SETTINGS)
  if (!raw || typeof raw !== 'object') {
    return settings
  }

  const source = raw as Record<string, unknown>
  settings.enabled = booleanValue(source.enabled, settings.enabled)
  settings.name_prefix = String(source.name_prefix ?? '').trim()
  settings.message = String(source.message ?? '').trim()

  const intervalRaw = Number(source.interval_days)
  if (Number.isFinite(intervalRaw) && intervalRaw >= 1) {
    settings.interval_days = Math.floor(intervalRaw)
  }
  if (settings.interval_days > 365) {
    settings.interval_days = 365
  }

  const minDelayRaw = Number(source.min_delay_minutes)
  if (Number.isFinite(minDelayRaw) && minDelayRaw >= 0) {
    settings.min_delay_minutes = Math.floor(minDelayRaw)
  }

  const maxDelayRaw = Number(source.max_delay_minutes)
  if (Number.isFinite(maxDelayRaw) && maxDelayRaw >= 0) {
    settings.max_delay_minutes = Math.floor(maxDelayRaw)
  }
  if (settings.max_delay_minutes < settings.min_delay_minutes) {
    settings.max_delay_minutes = settings.min_delay_minutes
  }

  const targetStatus = String(source.target_status ?? '').trim().toLowerCase()
  settings.target_status = targetStatus === 'run' ? 'run' : 'draft'

  settings.media_local_path = String(source.media_local_path ?? '').trim()
  settings.media_mime_type = String(source.media_mime_type ?? '').trim()
  settings.media_filename = String(source.media_filename ?? '').trim()
  if (!settings.media_local_path) {
    settings.media_mime_type = ''
    settings.media_filename = ''
  }

  const lastGeneratedRaw = String(source.last_generated_at ?? '').trim()
  if (lastGeneratedRaw) {
    const parsed = new Date(lastGeneratedRaw)
    if (!Number.isNaN(parsed.getTime())) {
      settings.last_generated_at = parsed.toISOString()
    }
  }

  return settings
}

export function cloneAutoCampaignSettings(source: AutoCampaignSettings): AutoCampaignSettings {
  return {
    enabled: source.enabled,
    name_prefix: source.name_prefix,
    message: source.message,
    interval_days: source.interval_days,
    min_delay_minutes: source.min_delay_minutes,
    max_delay_minutes: source.max_delay_minutes,
    target_status: source.target_status,
    media_local_path: source.media_local_path,
    media_mime_type: source.media_mime_type,
    media_filename: source.media_filename,
    last_generated_at: source.last_generated_at
  }
}

function booleanValue(value: unknown, fallback: boolean): boolean {
  if (typeof value === 'boolean') {
    return value
  }
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    if (normalized === 'true') return true
    if (normalized === 'false') return false
  }
  if (typeof value === 'number') {
    return value !== 0
  }
  return fallback
}
