export type AutoRejectCallMode = 'without_message' | 'with_message'
export type AutoRejectScheduleType = 'always' | 'custom_hours' | 'while_in_other_calls'

export interface AutoRejectSchedule {
  type: AutoRejectScheduleType
  start: string
  end: string
  days: number[]
  timezone: string
}

export interface AutoRejectCallSettings {
  enabled: boolean
  mode: AutoRejectCallMode
  message: string
  reject_individual_calls: boolean
  reject_group_calls: boolean
  bypass_contacts: string[]
  schedule: AutoRejectSchedule
}

const DEFAULT_TIMEZONE = (() => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  } catch {
    return 'UTC'
  }
})()

export const DEFAULT_AUTO_REJECT_SETTINGS: AutoRejectCallSettings = {
  enabled: false,
  mode: 'without_message',
  message: '',
  reject_individual_calls: true,
  reject_group_calls: true,
  bypass_contacts: [],
  schedule: {
    type: 'always',
    start: '09:00',
    end: '18:00',
    days: [1, 2, 3, 4, 5],
    timezone: DEFAULT_TIMEZONE
  }
}

export function normalizeAutoRejectCallSettings(raw: unknown): AutoRejectCallSettings {
  const settings = cloneAutoRejectSettings(DEFAULT_AUTO_REJECT_SETTINGS)
  if (!raw || typeof raw !== 'object') {
    return settings
  }

  const source = raw as Record<string, unknown>
  settings.enabled = booleanValue(source.enabled, settings.enabled)

  const mode = String(source.mode ?? '').trim()
  settings.mode = mode === 'with_message' ? 'with_message' : 'without_message'
  settings.message = String(source.message ?? '').trim()

  settings.reject_individual_calls = booleanValue(source.reject_individual_calls, settings.reject_individual_calls)
  settings.reject_group_calls = booleanValue(source.reject_group_calls, settings.reject_group_calls)
  settings.bypass_contacts = normalizeBypassContacts(source.bypass_contacts)

  if (source.schedule && typeof source.schedule === 'object') {
    const schedule = source.schedule as Record<string, unknown>
    const scheduleType = String(schedule.type ?? '').trim()
    if (scheduleType === 'custom_hours' || scheduleType === 'while_in_other_calls' || scheduleType === 'always') {
      settings.schedule.type = scheduleType
    }

    const start = String(schedule.start ?? '').trim()
    if (isValidHHMM(start)) {
      settings.schedule.start = start
    }

    const end = String(schedule.end ?? '').trim()
    if (isValidHHMM(end)) {
      settings.schedule.end = end
    }

    const timezone = String(schedule.timezone ?? '').trim()
    if (timezone) {
      settings.schedule.timezone = timezone
    }

    const days = normalizeDays(schedule.days)
    if (days.length > 0) {
      settings.schedule.days = days
    }
  }

  return settings
}

export function cloneAutoRejectSettings(source: AutoRejectCallSettings): AutoRejectCallSettings {
  return {
    enabled: source.enabled,
    mode: source.mode,
    message: source.message,
    reject_individual_calls: source.reject_individual_calls,
    reject_group_calls: source.reject_group_calls,
    bypass_contacts: [...source.bypass_contacts],
    schedule: {
      type: source.schedule.type,
      start: source.schedule.start,
      end: source.schedule.end,
      days: [...source.schedule.days],
      timezone: source.schedule.timezone
    }
  }
}

export function autoRejectScheduleSummary(settings: AutoRejectCallSettings): string {
  if (!settings.enabled) {
    return 'Off'
  }

  switch (settings.schedule.type) {
    case 'while_in_other_calls':
      return 'While in other calls'
    case 'custom_hours':
      return `${settings.schedule.start} - ${settings.schedule.end} (${settings.schedule.timezone})`
    default:
      return 'Always on'
  }
}

export function bypassContactsToEditorValue(contacts: string[]): string {
  return contacts.join('\n')
}

export function bypassContactsFromEditorValue(value: string): string[] {
  return normalizeBypassContacts(value)
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

function normalizeBypassContacts(value: unknown): string[] {
  const normalized = new Set<string>()

  const add = (entry: string) => {
    const cleaned = normalizePhone(entry)
    if (cleaned) {
      normalized.add(cleaned)
    }
  }

  if (Array.isArray(value)) {
    value.forEach(entry => add(String(entry ?? '')))
  } else if (typeof value === 'string') {
    value
      .replace(/\r/g, '\n')
      .split(/[,\n]/)
      .forEach(entry => add(entry))
  }

  return [...normalized].sort()
}

function normalizeDays(value: unknown): number[] {
  if (!Array.isArray(value)) {
    return []
  }

  const daySet = new Set<number>()
  value.forEach(entry => {
    const parsed = Number(entry)
    if (Number.isInteger(parsed) && parsed >= 0 && parsed <= 6) {
      daySet.add(parsed)
    }
  })

  return [...daySet].sort((a, b) => a - b)
}

function isValidHHMM(value: string): boolean {
  return /^([01]\d|2[0-3]):[0-5]\d$/.test(value)
}

function normalizePhone(value: string): string {
  return value.replace(/\D/g, '')
}
