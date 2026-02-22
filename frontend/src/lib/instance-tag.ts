import type { WhatsAppInstance } from '@/types/whatsmeow'

export type InstanceTagDisplayMode = 'name' | 'phone' | 'custom'

export type InstanceTagColorKey =
  | 'emerald'
  | 'sky'
  | 'amber'
  | 'violet'
  | 'rose'
  | 'cyan'
  | 'orange'
  | 'indigo'
  | 'lime'
  | 'fuchsia'

export interface InstanceTagSettings {
  chat_tag_custom_label?: string
  chat_tag_color?: InstanceTagColorKey
  chat_tag_display_mode?: InstanceTagDisplayMode
}

export interface InstanceTagColorPreset {
  key: InstanceTagColorKey
  label: string
  badgeClass: string
  dotClass: string
  swatchClass: string
  ringClass: string
}

export const INSTANCE_TAG_COLOR_PRESETS: InstanceTagColorPreset[] = [
  {
    key: 'emerald',
    label: 'Emerald',
    badgeClass: 'bg-emerald-500/18 text-emerald-200 border-emerald-300/35 light:bg-emerald-100 light:text-emerald-700 light:border-emerald-200',
    dotClass: 'bg-emerald-300 light:bg-emerald-500',
    swatchClass: 'bg-emerald-500',
    ringClass: 'ring-emerald-400/40 light:ring-emerald-300'
  },
  {
    key: 'sky',
    label: 'Sky',
    badgeClass: 'bg-sky-500/18 text-sky-200 border-sky-300/35 light:bg-sky-100 light:text-sky-700 light:border-sky-200',
    dotClass: 'bg-sky-300 light:bg-sky-500',
    swatchClass: 'bg-sky-500',
    ringClass: 'ring-sky-400/40 light:ring-sky-300'
  },
  {
    key: 'amber',
    label: 'Amber',
    badgeClass: 'bg-amber-500/18 text-amber-200 border-amber-300/35 light:bg-amber-100 light:text-amber-700 light:border-amber-200',
    dotClass: 'bg-amber-300 light:bg-amber-500',
    swatchClass: 'bg-amber-500',
    ringClass: 'ring-amber-400/40 light:ring-amber-300'
  },
  {
    key: 'violet',
    label: 'Violet',
    badgeClass: 'bg-violet-500/18 text-violet-200 border-violet-300/35 light:bg-violet-100 light:text-violet-700 light:border-violet-200',
    dotClass: 'bg-violet-300 light:bg-violet-500',
    swatchClass: 'bg-violet-500',
    ringClass: 'ring-violet-400/40 light:ring-violet-300'
  },
  {
    key: 'rose',
    label: 'Rose',
    badgeClass: 'bg-rose-500/18 text-rose-200 border-rose-300/35 light:bg-rose-100 light:text-rose-700 light:border-rose-200',
    dotClass: 'bg-rose-300 light:bg-rose-500',
    swatchClass: 'bg-rose-500',
    ringClass: 'ring-rose-400/40 light:ring-rose-300'
  },
  {
    key: 'cyan',
    label: 'Cyan',
    badgeClass: 'bg-cyan-500/18 text-cyan-200 border-cyan-300/35 light:bg-cyan-100 light:text-cyan-700 light:border-cyan-200',
    dotClass: 'bg-cyan-300 light:bg-cyan-500',
    swatchClass: 'bg-cyan-500',
    ringClass: 'ring-cyan-400/40 light:ring-cyan-300'
  },
  {
    key: 'orange',
    label: 'Orange',
    badgeClass: 'bg-orange-500/18 text-orange-200 border-orange-300/35 light:bg-orange-100 light:text-orange-700 light:border-orange-200',
    dotClass: 'bg-orange-300 light:bg-orange-500',
    swatchClass: 'bg-orange-500',
    ringClass: 'ring-orange-400/40 light:ring-orange-300'
  },
  {
    key: 'indigo',
    label: 'Indigo',
    badgeClass: 'bg-indigo-500/18 text-indigo-200 border-indigo-300/35 light:bg-indigo-100 light:text-indigo-700 light:border-indigo-200',
    dotClass: 'bg-indigo-300 light:bg-indigo-500',
    swatchClass: 'bg-indigo-500',
    ringClass: 'ring-indigo-400/40 light:ring-indigo-300'
  },
  {
    key: 'lime',
    label: 'Lime',
    badgeClass: 'bg-lime-500/18 text-lime-200 border-lime-300/35 light:bg-lime-100 light:text-lime-700 light:border-lime-200',
    dotClass: 'bg-lime-300 light:bg-lime-500',
    swatchClass: 'bg-lime-500',
    ringClass: 'ring-lime-400/40 light:ring-lime-300'
  },
  {
    key: 'fuchsia',
    label: 'Fuchsia',
    badgeClass: 'bg-fuchsia-500/18 text-fuchsia-200 border-fuchsia-300/35 light:bg-fuchsia-100 light:text-fuchsia-700 light:border-fuchsia-200',
    dotClass: 'bg-fuchsia-300 light:bg-fuchsia-500',
    swatchClass: 'bg-fuchsia-500',
    ringClass: 'ring-fuchsia-400/40 light:ring-fuchsia-300'
  }
]

const COLOR_PRESET_BY_KEY: Record<InstanceTagColorKey, InstanceTagColorPreset> = INSTANCE_TAG_COLOR_PRESETS.reduce((acc, preset) => {
  acc[preset.key] = preset
  return acc
}, {} as Record<InstanceTagColorKey, InstanceTagColorPreset>)

const DEFAULT_TAG_LABEL = 'Instance'

function normalizeOptionalLabel(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  return trimmed.length > 0 ? trimmed : undefined
}

function isInstanceTagColorKey(value: unknown): value is InstanceTagColorKey {
  return typeof value === 'string' && value in COLOR_PRESET_BY_KEY
}

function isInstanceTagDisplayMode(value: unknown): value is InstanceTagDisplayMode {
  return value === 'name' || value === 'phone' || value === 'custom'
}

export function readInstanceTagSettings(instance?: Pick<WhatsAppInstance, 'settings'> | null): InstanceTagSettings {
  const settings = (instance?.settings || {}) as Record<string, unknown>
  const customLabel = normalizeOptionalLabel(settings.chat_tag_custom_label)
  const color = isInstanceTagColorKey(settings.chat_tag_color) ? settings.chat_tag_color : undefined
  const displayMode = isInstanceTagDisplayMode(settings.chat_tag_display_mode) ? settings.chat_tag_display_mode : undefined
  return {
    chat_tag_custom_label: customLabel,
    chat_tag_color: color,
    chat_tag_display_mode: displayMode
  }
}

export function resolveInstanceTagColorKey(
  instance?: Pick<WhatsAppInstance, 'settings'> | null,
  fallbackIndex = 0
): InstanceTagColorKey {
  const settings = readInstanceTagSettings(instance)
  if (settings.chat_tag_color) {
    return settings.chat_tag_color
  }
  const normalizedIndex = Math.abs(fallbackIndex) % INSTANCE_TAG_COLOR_PRESETS.length
  return INSTANCE_TAG_COLOR_PRESETS[normalizedIndex].key
}

export function getInstanceTagPresetByKey(key: InstanceTagColorKey): InstanceTagColorPreset {
  return COLOR_PRESET_BY_KEY[key] || INSTANCE_TAG_COLOR_PRESETS[0]
}

export function resolveInstanceTagDisplayMode(
  instance?: Pick<WhatsAppInstance, 'settings'> | null,
  fallback: InstanceTagDisplayMode = 'name'
): InstanceTagDisplayMode {
  const settings = readInstanceTagSettings(instance)
  return settings.chat_tag_display_mode || fallback
}

export function getInstanceTagLabel(
  instance: Pick<WhatsAppInstance, 'name' | 'phone_number' | 'settings'>,
  displayMode: InstanceTagDisplayMode
): string {
  const settings = readInstanceTagSettings(instance)
  const customLabel = settings.chat_tag_custom_label
  const name = normalizeOptionalLabel(instance.name)
  const phone = normalizeOptionalLabel(instance.phone_number)

  if (displayMode === 'custom') {
    return customLabel || name || phone || DEFAULT_TAG_LABEL
  }
  if (displayMode === 'phone') {
    return phone || name || customLabel || DEFAULT_TAG_LABEL
  }
  return name || phone || customLabel || DEFAULT_TAG_LABEL
}

export function upsertInstanceTagSettings(
  currentSettings: Record<string, any> | undefined,
  updates: InstanceTagSettings
): Record<string, any> {
  const nextSettings = { ...(currentSettings || {}) }
  const customLabel = normalizeOptionalLabel(updates.chat_tag_custom_label)

  if (customLabel) {
    nextSettings.chat_tag_custom_label = customLabel
  } else {
    delete nextSettings.chat_tag_custom_label
  }

  if (updates.chat_tag_color && isInstanceTagColorKey(updates.chat_tag_color)) {
    nextSettings.chat_tag_color = updates.chat_tag_color
  } else {
    delete nextSettings.chat_tag_color
  }

  if (updates.chat_tag_display_mode && isInstanceTagDisplayMode(updates.chat_tag_display_mode)) {
    nextSettings.chat_tag_display_mode = updates.chat_tag_display_mode
  } else {
    delete nextSettings.chat_tag_display_mode
  }

  return nextSettings
}
