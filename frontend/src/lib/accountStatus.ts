/**
 * Task 12: one canonical enum for WhatsApp account connection state,
 * shared by every place that renders an account status badge.
 *
 * The API emits exactly these three values (see
 * `models.NormalizeAccountStatus` on the backend): `Connected`,
 * `Disconnected` and `Reconnecting`. The live GOWA probe (`gowa_connected`)
 * wins when it is known; otherwise the stored status is used — after
 * normalizing the legacy lowercase values ("active" was the old word for a
 * connected device). The raw stored string is never rendered: an account
 * with an unknown or missing status fails closed to Disconnected.
 */

export type AccountConnectionStatus = 'connected' | 'disconnected' | 'reconnecting'

export interface AccountStatusInput {
  gowa_connected?: boolean | null
  status?: string | null
}

const LEGACY_STATUS_MAP: Record<string, AccountConnectionStatus> = {
  connected: 'connected',
  active: 'connected',
  disconnected: 'disconnected',
  reconnecting: 'reconnecting',
}

/** Map a stored/API status string onto the canonical enum. */
export function normalizeAccountStatus(raw?: string | null): AccountConnectionStatus {
  if (!raw) return 'disconnected'
  return LEGACY_STATUS_MAP[raw.trim().toLowerCase()] ?? 'disconnected'
}

/**
 * Resolve the status to badge for one account: the live probe when GOWA
 * answered, the (normalized) stored status otherwise.
 */
export function resolveAccountStatus(account: AccountStatusInput): AccountConnectionStatus {
  if (account.gowa_connected === true) return 'connected'
  if (account.gowa_connected === false) return 'disconnected'
  return normalizeAccountStatus(account.status)
}

export interface AccountStatusBadge {
  status: AccountConnectionStatus
  /** i18n key for the badge label. */
  labelKey: string
  /** Badge tone classes (the Badge itself stays `variant="outline"`). */
  toneClass: string
}

const BADGE_META: Record<AccountConnectionStatus, AccountStatusBadge> = {
  connected: {
    status: 'connected',
    labelKey: 'accounts.connected',
    toneClass: 'border-emerald-600 text-emerald-600 bg-emerald-500/10',
  },
  disconnected: {
    status: 'disconnected',
    labelKey: 'accounts.disconnected',
    toneClass: 'border-amber-600 text-amber-600 bg-amber-500/10',
  },
  reconnecting: {
    status: 'reconnecting',
    labelKey: 'accounts.reconnecting',
    toneClass: 'border-sky-600 text-sky-600 bg-sky-500/10',
  },
}

/** Badge metadata (label key + tone) for one account. */
export function accountStatusBadge(account: AccountStatusInput): AccountStatusBadge {
  return BADGE_META[resolveAccountStatus(account)]
}
