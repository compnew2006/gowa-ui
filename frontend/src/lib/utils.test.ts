import { describe, expect, it } from 'vitest'

import { formatLabel } from './utils'

describe('formatLabel', () => {
  it.each([
    { input: 'first_name', expected: 'First Name' },
    { input: 'createdAt', expected: 'Created At' },
    { input: 'user_firstName', expected: 'User First Name' },
    { input: 'groupChatID', expected: 'Group Chat ID' },
    { input: 'userID', expected: 'User ID' },
    { input: 'metadata_is_group_chat', expected: 'Metadata Is Group Chat' },
    { input: 'api_responseTimeMs', expected: 'Api Response Time Ms' },
    { input: 'support_email_address', expected: 'Support Email Address' },
    { input: 'already formatted', expected: 'Already Formatted' },
    { input: '', expected: '' }
  ])('formats "$input" as "$expected"', ({ input, expected }) => {
    expect(formatLabel(input)).toBe(expected)
  })
})
