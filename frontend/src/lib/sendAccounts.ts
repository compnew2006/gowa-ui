/**
 * Send-account policy helpers for the chat composer.
 *
 * The server decides which WhatsApp accounts the current user may send from
 * (GET /contacts/{id}/send-accounts). The UI only reads that decision: it never
 * auto-selects an account the server marked can_send=false.
 */

export interface SendAccountOption {
  name: string
  can_send: boolean
  reason: string | null
}

/** account name → may the user send from it. `null` = unknown (not loaded). */
export type SendAccountStatus = Record<string, boolean> | null

/** Stable server error code for a send from a non-permitted account. */
export const ACCOUNT_NOT_ALLOWED = 'account_not_allowed'
/** Stable server error code when no permitted account could be chosen. */
export const ACCOUNT_REQUIRED = 'account_required'

export function toSendAccountStatus(options: SendAccountOption[]): Record<string, boolean> {
  const status: Record<string, boolean> = {}
  for (const opt of options) status[opt.name] = opt.can_send
  return status
}

/** True only when the server explicitly refused sending from `account`. */
export function isSendDenied(status: SendAccountStatus, account: string | null): boolean {
  if (!status) return false
  if (!account) return !Object.values(status).some(Boolean)
  return status[account] === false
}

export interface PickSendAccountInput {
  /** Account requested by the route (?account=). */
  preferred?: string | null
  /** The contact's own account. */
  contactAccount?: string | null
  /** Accounts of incoming messages, oldest first. */
  incomingAccounts: string[]
  /** Account tabs of the conversation, in display order. */
  accounts: string[]
  status: SendAccountStatus
}

/**
 * Chooses the account to select for a conversation.
 *
 * With the server's decision loaded: the route's account, the contact's
 * account, the latest incoming account, then the first tab — each only when
 * it may be sent from — else null. Without it (request failed or the viewer is
 * read-only) the legacy order is kept for viewing; the server still enforces
 * the policy on send.
 */
export function pickSendAccount(input: PickSendAccountInput): string | null {
  const { preferred, contactAccount, incomingAccounts, accounts, status } = input
  const latestIncoming = [...incomingAccounts].reverse()

  if (!status) {
    if (preferred && accounts.includes(preferred)) return preferred
    return latestIncoming[0] || contactAccount || accounts[0] || null
  }

  const allowed = (name?: string | null): name is string => !!name && status[name] === true
  const ordered = [preferred, contactAccount, ...latestIncoming, ...accounts]
  return ordered.find(allowed) ?? null
}

/** Extracts the server's error code from an API error envelope. */
export function apiErrorCode(error: unknown): string | undefined {
  const data = (error as { response?: { data?: { error_type?: unknown } } } | null)?.response?.data
  return typeof data?.error_type === 'string' ? data.error_type : undefined
}

/**
 * i18n key for a send-account refusal from the server, or null when the error
 * is something else (callers then fall back to their generic message).
 */
export function sendAccountErrorKey(error: unknown): string | null {
  const code = apiErrorCode(error)
  if (code === ACCOUNT_REQUIRED) return 'chat.sendAccountRequired'
  if (code === ACCOUNT_NOT_ALLOWED) return 'chat.sendAccountNotAllowed'
  return null
}
