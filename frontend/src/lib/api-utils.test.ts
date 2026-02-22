import type { AxiosResponse } from 'axios'
import { describe, expect, it } from 'vitest'

import { unwrapResponse } from './api-utils'

function makeResponse(data: unknown): AxiosResponse {
  return {
    data,
    status: 200,
    statusText: 'OK',
    headers: {},
    config: {} as AxiosResponse['config']
  }
}

describe('unwrapResponse', () => {
  it('unwraps nested API envelopes', () => {
    const payload = { id: 1, name: 'Alice' }
    const response = makeResponse({ data: payload })

    expect(unwrapResponse<typeof payload>(response)).toEqual(payload)
  })

  it('returns flat response data as-is', () => {
    const payload = { id: 2, name: 'Bob' }
    const response = makeResponse(payload)

    expect(unwrapResponse<typeof payload>(response)).toEqual(payload)
  })

  it('returns primitive response data as-is', () => {
    const response = makeResponse('ok')

    expect(unwrapResponse<string>(response)).toBe('ok')
  })

  it('returns null response data as-is', () => {
    const response = makeResponse(null)

    expect(unwrapResponse<null>(response)).toBeNull()
  })

  it('prefers nested data key when present even if undefined', () => {
    const response = makeResponse({ data: undefined, meta: { total: 1 } })

    expect(unwrapResponse<undefined>(response)).toBeUndefined()
  })
})
