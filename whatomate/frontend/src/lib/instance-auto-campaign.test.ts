import { describe, expect, it } from 'vitest'

import {
  DEFAULT_AUTO_CAMPAIGN_SETTINGS,
  cloneAutoCampaignSettings,
  getAutoCampaignEvaluationSchedule,
  normalizeAutoCampaignSettings,
} from './instance-auto-campaign'

describe('instance-auto-campaign', () => {
  it('returns defaults for invalid input', () => {
    expect(normalizeAutoCampaignSettings(null)).toEqual(DEFAULT_AUTO_CAMPAIGN_SETTINGS)
    expect(normalizeAutoCampaignSettings(undefined)).toEqual(DEFAULT_AUTO_CAMPAIGN_SETTINGS)
    expect(normalizeAutoCampaignSettings('invalid')).toEqual(DEFAULT_AUTO_CAMPAIGN_SETTINGS)
  })

  it('normalizes interval and delay boundaries', () => {
    const normalized = normalizeAutoCampaignSettings({
      enabled: true,
      message: 'Hello',
      interval_days: 999,
      min_delay_minutes: 15,
      max_delay_minutes: 2,
    })

    expect(normalized.enabled).toBe(true)
    expect(normalized.interval_days).toBe(365)
    expect(normalized.min_delay_minutes).toBe(15)
    expect(normalized.max_delay_minutes).toBe(15)
  })

  it('normalizes target status and media linkage', () => {
    const normalizedRun = normalizeAutoCampaignSettings({
      target_status: 'RUN',
      media_local_path: '/tmp/media/file.pdf',
      media_mime_type: 'application/pdf',
      media_filename: 'file.pdf',
    })
    expect(normalizedRun.target_status).toBe('run')
    expect(normalizedRun.media_local_path).toBe('/tmp/media/file.pdf')
    expect(normalizedRun.media_mime_type).toBe('application/pdf')
    expect(normalizedRun.media_filename).toBe('file.pdf')

    const normalizedDraft = normalizeAutoCampaignSettings({
      target_status: 'invalid-status',
      media_local_path: '',
      media_mime_type: 'application/pdf',
      media_filename: 'file.pdf',
    })
    expect(normalizedDraft.target_status).toBe('draft')
    expect(normalizedDraft.media_mime_type).toBe('')
    expect(normalizedDraft.media_filename).toBe('')
  })

  it('keeps valid ISO last_generated_at and removes invalid values', () => {
    const valid = normalizeAutoCampaignSettings({
      last_generated_at: '2026-03-01T12:00:00Z',
    })
    expect(valid.last_generated_at).toBe('2026-03-01T12:00:00.000Z')

    const invalid = normalizeAutoCampaignSettings({
      last_generated_at: 'not-a-date',
    })
    expect(invalid.last_generated_at).toBeUndefined()
  })

  it('clone creates an independent copy', () => {
    const original = normalizeAutoCampaignSettings({
      enabled: true,
      name_prefix: 'Promo',
      message: 'Hello {{name}}',
      interval_days: 7,
      min_delay_minutes: 1,
      max_delay_minutes: 3,
      target_status: 'draft',
      media_local_path: '/tmp/a.png',
      media_mime_type: 'image/png',
      media_filename: 'a.png',
    })

    const cloned = cloneAutoCampaignSettings(original)
    cloned.name_prefix = 'Changed'

    expect(original.name_prefix).toBe('Promo')
    expect(cloned.name_prefix).toBe('Changed')
  })

  it('computes evaluation timestamps from last_generated_at', () => {
    const schedule = getAutoCampaignEvaluationSchedule({
      interval_days: 7,
      last_generated_at: '2026-03-01T12:00:00Z',
    })

    expect(schedule).toEqual({
      lastEvaluationAt: '2026-03-01T12:00:00.000Z',
      nextEvaluationAt: '2026-03-08T12:00:00.000Z',
    })
  })

  it('returns an empty schedule before the first evaluation', () => {
    expect(getAutoCampaignEvaluationSchedule({
      enabled: true,
      interval_days: 7,
      message: 'Hello',
    })).toEqual({})
  })
})
