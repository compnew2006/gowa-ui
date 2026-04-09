import { beforeEach, describe, expect, it, vi } from 'vitest'

import { wsService } from './websocket'
import { useContactsStore } from '@/stores/contacts'
import { useAuthStore } from '@/stores/auth'

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

  it('fetches the newly assigned contact for the assignee when it is not in the local store', () => {
    const patchContact = vi.fn()
    const fetchContact = vi.fn()
    vi.mocked(useContactsStore).mockReturnValue({
      contacts: [],
      patchContact,
      fetchContact,
      messages: [],
    } as unknown as ReturnType<typeof useContactsStore>)
    vi.mocked(useAuthStore).mockReturnValue({
      user: {
        id: 'agent-1',
      },
      userSettings: {},
    } as unknown as ReturnType<typeof useAuthStore>)

    ;(wsService as any).handleMessage(JSON.stringify({
      type: 'contact_update',
      payload: {
        id: 'contact-99',
        assigned_user_id: 'agent-1',
        status: 'open',
        notify_assignment: true,
      },
    }))

    expect(patchContact).toHaveBeenCalledWith(expect.objectContaining({
      id: 'contact-99',
      assigned_user_id: 'agent-1',
      status: 'open',
    }))
    expect(fetchContact).toHaveBeenCalledWith('contact-99')
  })

  it('deduplicates unknown-contact fetches during a single incoming message event', async () => {
    const fetchContact = vi.fn().mockResolvedValue({
      id: 'contact-42',
      assigned_user_id: 'agent-1',
      status: 'open',
    })
    const addMessage = vi.fn(() => true)

    vi.mocked(useContactsStore).mockReturnValue({
      currentContact: null,
      contacts: [],
      addMessage,
      fetchContact,
      patchContact: vi.fn(),
      messages: [],
    } as unknown as ReturnType<typeof useContactsStore>)
    vi.mocked(useAuthStore).mockReturnValue({
      user: {
        id: 'agent-1',
        role: {
          name: 'agent',
        },
        is_super_admin: false,
      },
      userSettings: {
        new_message_alerts: false,
      },
    } as unknown as ReturnType<typeof useAuthStore>)

    ;(wsService as any).handleMessage(JSON.stringify({
      type: 'new_message',
      payload: {
        id: 'message-42',
        contact_id: 'contact-42',
        direction: 'incoming',
        content: { body: 'hello' },
        message_type: 'text',
        status: 'received',
        created_at: '2026-03-03T10:00:00.000Z',
        updated_at: '2026-03-03T10:00:00.000Z',
      },
    }))

    await Promise.resolve()
    await Promise.resolve()

    expect(addMessage).toHaveBeenCalledTimes(1)
    expect(fetchContact).toHaveBeenCalledTimes(1)
    expect(fetchContact).toHaveBeenCalledWith('contact-42')
  })
})
