import { beforeEach, describe, expect, it, vi } from 'vitest'

import { wsService } from './websocket'
import { useContactsStore } from '@/stores/contacts'

vi.mock('@/stores/contacts', () => ({
  useContactsStore: vi.fn(),
}))

vi.mock('@/stores/transfers', () => ({
  useTransfersStore: vi.fn(() => ({ fetchTransfers: vi.fn() })),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(() => ({
    user: null,
    userSettings: {},
  })),
}))

vi.mock('@/stores/notes', () => ({
  useNotesStore: vi.fn(() => ({
    addNote: vi.fn(),
    onNoteUpdated: vi.fn(),
    onNoteDeleted: vi.fn(),
  })),
}))

vi.mock('@/services/api', () => ({
  contactsService: {
    get: vi.fn(),
  },
}))

vi.mock('@/lib/incoming_media_autodownload', () => ({
  maybeAutoDownloadIncomingMedia: vi.fn(),
}))

vi.mock('vue-sonner', () => ({
  toast: {
    info: vi.fn(),
  },
}))

vi.mock('@/router', () => ({
  default: {
    push: vi.fn(),
  },
}))

describe('websocket message_media_updated', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('patches message media fields on realtime recovery event', () => {
    const patchMessage = vi.fn()
    vi.mocked(useContactsStore).mockReturnValue({
      messages: [
        {
          id: 'message-1',
          contact_id: 'contact-1',
          direction: 'incoming',
          message_type: 'document',
          content: { body: '[Document]' },
          status: 'received',
          created_at: '2026-03-03T10:00:00.000Z',
          updated_at: '2026-03-03T10:00:00.000Z',
        },
      ],
      patchMessage,
    } as unknown as ReturnType<typeof useContactsStore>)

    ;(wsService as any).handleMessage(JSON.stringify({
      type: 'message_media_updated',
      payload: {
        id: 'message-1',
        contact_id: 'contact-1',
        media_url: 'documents/recovered.pdf',
        media_mime_type: 'application/pdf',
        media_filename: 'recovered.pdf',
        updated_at: '2026-03-03T10:01:00.000Z',
      },
    }))

    expect(patchMessage).toHaveBeenCalledTimes(1)
    expect(patchMessage).toHaveBeenCalledWith(expect.objectContaining({
      id: 'message-1',
      contact_id: 'contact-1',
      media_url: 'documents/recovered.pdf',
      media_mime_type: 'application/pdf',
      media_filename: 'recovered.pdf',
      message_type: 'document',
      direction: 'incoming',
    }))
  })
})
