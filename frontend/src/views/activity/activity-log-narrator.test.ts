import { describe, expect, it } from 'vitest'

import type { ActivityLog } from '@/services/api'
import { ActivityLogNarrator } from './activity-log-narrator'
import { extractContactIDFromPath } from './activity-log-route-utils'

function makeLog(overrides: Partial<ActivityLog>): ActivityLog {
  return {
    id: 'log-id',
    created_at: '2026-02-21T14:55:00Z',
    updated_at: '2026-02-21T14:55:00Z',
    category: 'system',
    event_type: 'system.api_interaction',
    action: 'api_request',
    status: 'success',
    source: 'system',
    metadata: { http_status: 200 },
    ...overrides
  }
}

describe('ActivityLogNarrator', () => {
  it('describes contact-scoped routes with contact id fallback', () => {
    const narrator = new ActivityLogNarrator()
    const contactID = '53654e29-3f1f-4645-a39c-330da32c082f'

    const narrative = narrator.build(
      makeLog({
        method: 'GET',
        path: `/api/v1/contacts/${contactID}/session-data`
      })
    )

    expect(narrative.sentence).toContain(`viewed session summary for contact ${contactID}`)
    expect(narrative.sentence).not.toContain('viewed the contacts')
  })

  it('uses resolved contact labels when available', () => {
    const narrator = new ActivityLogNarrator()
    const contactID = '53654e29-3f1f-4645-a39c-330da32c082f'

    const narrative = narrator.build(
      makeLog({
        method: 'GET',
        path: `/api/contacts/${contactID}`
      }),
      {
        contactLabels: new Map([[contactID, 'Ralph (+966541385767)']])
      }
    )

    expect(narrative.sentence).toContain('opened contact profile for Ralph (+966541385767)')
  })
})

describe('extractContactIDFromPath', () => {
  it('extracts id from versioned api routes', () => {
    const contactID = '53654e29-3f1f-4645-a39c-330da32c082f'

    expect(extractContactIDFromPath(`/api/v1/contacts/${contactID}`)).toBe(contactID)
    expect(extractContactIDFromPath(`/api/v2/chats/${contactID}/messages?page=1`)).toBe(contactID)
  })
})
