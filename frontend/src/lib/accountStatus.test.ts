/**
 * Task 12: the account status badge renders from one canonical enum
 * (Connected / Disconnected / Reconnecting). The live GOWA probe wins when
 * known; otherwise the stored status is normalized — the legacy lowercase
 * "active" must render as the Connected badge, never as bare text.
 */
import { describe, expect, it } from 'vitest'
import {
  accountStatusBadge,
  normalizeAccountStatus,
  resolveAccountStatus,
} from './accountStatus'

describe('normalizeAccountStatus', () => {
  it.each([
    ['active', 'connected'],
    ['Connected', 'connected'],
    ['connected', 'connected'],
    ['disconnected', 'disconnected'],
    ['Disconnected', 'disconnected'],
    ['reconnecting', 'reconnecting'],
    ['Reconnecting', 'reconnecting'],
    ['', 'disconnected'],
    ['bogus', 'disconnected'],
  ] as const)('maps %s to %s', (raw, want) => {
    expect(normalizeAccountStatus(raw)).toBe(want)
  })

  it('maps null/undefined to disconnected (fail closed)', () => {
    expect(normalizeAccountStatus(null)).toBe('disconnected')
    expect(normalizeAccountStatus(undefined)).toBe('disconnected')
  })
})

describe('resolveAccountStatus', () => {
  it('prefers the live probe when GOWA answered', () => {
    expect(resolveAccountStatus({ gowa_connected: true, status: 'disconnected' })).toBe('connected')
    expect(resolveAccountStatus({ gowa_connected: false, status: 'active' })).toBe('disconnected')
  })

  it('falls back to the normalized stored status when the probe is unknown', () => {
    // The Arkan-6178 case: no live probe, legacy stored "active".
    expect(resolveAccountStatus({ gowa_connected: null, status: 'active' })).toBe('connected')
    expect(resolveAccountStatus({ status: 'reconnecting' })).toBe('reconnecting')
    expect(resolveAccountStatus({ status: 'disconnected' })).toBe('disconnected')
    expect(resolveAccountStatus({})).toBe('disconnected')
  })
})

describe('accountStatusBadge', () => {
  it('never exposes the raw stored string as a label', () => {
    const badge = accountStatusBadge({ gowa_connected: null, status: 'active' })
    expect(badge.status).toBe('connected')
    expect(badge.labelKey).toBe('accounts.connected')
    expect(badge.toneClass).toContain('emerald')
  })

  it('gives reconnecting its own badge (the "other" state)', () => {
    const badge = accountStatusBadge({ status: 'reconnecting' })
    expect(badge.status).toBe('reconnecting')
    expect(badge.labelKey).toBe('accounts.reconnecting')
  })
})
