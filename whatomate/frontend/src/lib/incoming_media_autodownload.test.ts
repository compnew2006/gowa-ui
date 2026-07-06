import { beforeEach, describe, expect, it, vi } from 'vitest'

import { maybeAutoDownloadIncomingMedia, shouldAutoDownloadIncomingMedia } from './incoming_media_autodownload'
import { useConfigStore } from '@/stores/config'
import { useInstancesStore } from '@/stores/instances'
import { prefetchMediaBlob } from '@/lib/media_prefetch_cache'

vi.mock('@/stores/config', () => ({
  useConfigStore: vi.fn(),
}))

vi.mock('@/stores/instances', () => ({
  useInstancesStore: vi.fn(),
}))

vi.mock('@/lib/media_prefetch_cache', () => ({
  prefetchMediaBlob: vi.fn(async () => new Blob(['ok'], { type: 'text/plain' })),
  isMediaMessageType: (messageType: unknown) => {
    if (typeof messageType !== 'string') return false
    return ['image', 'video', 'audio', 'document', 'sticker'].includes(messageType.trim().toLowerCase())
  },
}))

describe('incoming_media_autodownload', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    vi.mocked(useConfigStore).mockReturnValue({
      provider: 'whatsmeow',
    } as unknown as ReturnType<typeof useConfigStore>)

    vi.mocked(useInstancesStore).mockReturnValue({
      instances: [
        {
          id: 'instance-1',
          settings: {
            auto_download_incoming_media: true,
          },
        },
      ],
    } as unknown as ReturnType<typeof useInstancesStore>)
  })

  it('returns true only when all auto-download gates pass', () => {
    expect(shouldAutoDownloadIncomingMedia({
      provider: 'whatsmeow',
      instanceAutoDownloadEnabled: true,
      payload: {
        id: 'msg-1',
        direction: 'incoming',
        message_type: 'image',
      },
    })).toBe(true)

    expect(shouldAutoDownloadIncomingMedia({
      provider: 'meta',
      instanceAutoDownloadEnabled: true,
      payload: {
        id: 'msg-1',
        direction: 'incoming',
        message_type: 'image',
      },
    })).toBe(false)

    expect(shouldAutoDownloadIncomingMedia({
      provider: 'whatsmeow',
      instanceAutoDownloadEnabled: false,
      payload: {
        id: 'msg-1',
        direction: 'incoming',
        message_type: 'image',
      },
    })).toBe(false)

    expect(shouldAutoDownloadIncomingMedia({
      provider: 'whatsmeow',
      instanceAutoDownloadEnabled: true,
      payload: {
        id: 'msg-1',
        direction: 'outgoing',
        message_type: 'image',
      },
    })).toBe(false)

    expect(shouldAutoDownloadIncomingMedia({
      provider: 'whatsmeow',
      instanceAutoDownloadEnabled: true,
      payload: {
        id: 'msg-1',
        direction: 'incoming',
        message_type: 'text',
      },
    })).toBe(false)
  })

  it('triggers prefetch for eligible incoming media payload', () => {
    maybeAutoDownloadIncomingMedia({
      id: 'msg-prefetch',
      direction: 'incoming',
      message_type: 'document',
      instance_id: 'instance-1',
    })

    expect(prefetchMediaBlob).toHaveBeenCalledTimes(1)
    expect(prefetchMediaBlob).toHaveBeenCalledWith('msg-prefetch')
  })

  it('skips prefetch for non-eligible payloads', () => {
    maybeAutoDownloadIncomingMedia({
      id: 'msg-skip-toggle',
      direction: 'incoming',
      message_type: 'image',
      instance_id: 'instance-unknown',
    })

    maybeAutoDownloadIncomingMedia({
      id: 'msg-skip-direction',
      direction: 'outgoing',
      message_type: 'image',
      instance_id: 'instance-1',
    })

    maybeAutoDownloadIncomingMedia({
      id: 'msg-skip-type',
      direction: 'incoming',
      message_type: 'text',
      instance_id: 'instance-1',
    })

    expect(prefetchMediaBlob).not.toHaveBeenCalled()
  })
})
