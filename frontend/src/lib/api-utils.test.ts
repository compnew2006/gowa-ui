import type { AxiosResponse } from 'axios'
import { describe, expect, it } from 'vitest'

import { getErrorMessage, unwrapResponse } from './api-utils'

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

describe('getErrorMessage', () => {
  it('returns a helpful proxy limit message for 413 upload failures', () => {
    const error = {
      isAxiosError: true,
      response: {
        status: 413,
        data: '<html><title>413 Request Entity Too Large</title></html>'
      }
    }

    expect(getErrorMessage(error)).toContain('client_max_body_size')
  })

  it('extracts a human-readable title from HTML error pages', () => {
    const error = {
      isAxiosError: true,
      response: {
        status: 502,
        data: '<html><head><title>Bad Gateway</title></head></html>'
      }
    }

    expect(getErrorMessage(error)).toBe('Bad Gateway')
  })
})
