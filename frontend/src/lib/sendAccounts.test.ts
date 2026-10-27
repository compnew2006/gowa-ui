import { describe, expect, it } from 'vitest'
import { apiErrorCode, isSendDenied, pickSendAccount, sendAccountErrorKey, toSendAccountStatus } from './sendAccounts'

const base = {
  preferred: null,
  contactAccount: 'B',
  incomingAccounts: ['A', 'C'],
  accounts: ['A', 'B', 'C'],
}

describe('pickSendAccount', () => {
  it('never auto-selects an account the server refused', () => {
    const status = { A: false, B: false, C: false }
    expect(pickSendAccount({ ...base, status })).toBeNull()
  })

  it('ignores a ?account= the user may not send from', () => {
    const status = { A: true, B: false, C: false }
    expect(pickSendAccount({ ...base, preferred: 'C', status })).toBe('A')
  })

  it('honours a permitted ?account= first', () => {
    const status = { A: true, B: true, C: true }
    expect(pickSendAccount({ ...base, preferred: 'C', status })).toBe('C')
  })

  it('prefers the contact account, then the latest incoming permitted account', () => {
    expect(pickSendAccount({ ...base, status: { A: true, B: true, C: true } })).toBe('B')
    expect(pickSendAccount({ ...base, status: { A: true, B: false, C: true } })).toBe('C')
    expect(pickSendAccount({ ...base, incomingAccounts: [], status: { A: false, B: false, C: true } })).toBe('C')
  })

  it('keeps the legacy viewing order when the decision is unknown', () => {
    expect(pickSendAccount({ ...base, status: null })).toBe('C')
    expect(pickSendAccount({ ...base, preferred: 'A', status: null })).toBe('A')
  })
})

describe('isSendDenied', () => {
  it('locks only on an explicit refusal', () => {
    expect(isSendDenied(null, 'A')).toBe(false)
    expect(isSendDenied({ A: true }, 'A')).toBe(false)
    expect(isSendDenied({ A: false }, 'A')).toBe(true)
    expect(isSendDenied({ A: true }, 'unlisted')).toBe(false)
  })

  it('locks with no selection when no account may be sent from', () => {
    expect(isSendDenied({ A: false, B: false }, null)).toBe(true)
    expect(isSendDenied({ A: false, B: true }, null)).toBe(false)
  })
})

describe('toSendAccountStatus / apiErrorCode', () => {
  it('maps server options to a lookup', () => {
    expect(toSendAccountStatus([
      { name: 'A', can_send: true, reason: null },
      { name: 'B', can_send: false, reason: 'account_not_allowed' },
    ])).toEqual({ A: true, B: false })
  })

  it('reads the envelope error_type', () => {
    expect(apiErrorCode({ response: { data: { error_type: 'account_not_allowed' } } })).toBe('account_not_allowed')
    expect(apiErrorCode(new Error('x'))).toBeUndefined()
    expect(apiErrorCode(null)).toBeUndefined()
  })
})

describe('sendAccountErrorKey', () => {
  it('maps send-account refusals to their messages', () => {
    expect(sendAccountErrorKey({ response: { data: { error_type: 'account_not_allowed' } } })).toBe('chat.sendAccountNotAllowed')
    expect(sendAccountErrorKey({ response: { data: { error_type: 'account_required' } } })).toBe('chat.sendAccountRequired')
  })

  it('ignores other errors', () => {
    expect(sendAccountErrorKey({ response: { data: { error_type: 'account_not_found' } } })).toBeNull()
    expect(sendAccountErrorKey(new Error('x'))).toBeNull()
  })
})
