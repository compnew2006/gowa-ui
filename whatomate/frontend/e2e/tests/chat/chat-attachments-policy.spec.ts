import { expect, test, type Page } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'

const CONTACT_ID = 'f4e0a795-3e79-463b-9ea0-613ef4f12001'
const INSTANCE_ID = '8f8124db-2fe4-40aa-8bce-67c67c54811d'

async function installMockWebSocket(page: Page) {
  await page.addInitScript(() => {
    class MockWebSocket {
      static CONNECTING = 0
      static OPEN = 1
      static CLOSING = 2
      static CLOSED = 3

      CONNECTING = MockWebSocket.CONNECTING
      OPEN = MockWebSocket.OPEN
      CLOSING = MockWebSocket.CLOSING
      CLOSED = MockWebSocket.CLOSED

      readyState = MockWebSocket.OPEN
      onopen: ((event: Event) => void) | null = null
      onmessage: ((event: MessageEvent) => void) | null = null
      onclose: ((event: CloseEvent) => void) | null = null
      onerror: ((event: Event) => void) | null = null

      constructor(_url: string) {
        setTimeout(() => this.onopen?.(new Event('open')), 0)
      }

      send(_data: string) {}

      close() {
        this.readyState = MockWebSocket.CLOSED
        this.onclose?.(new CloseEvent('close'))
      }
    }

    ;(window as any).WebSocket = MockWebSocket
  })
}

type ChatAttachmentMockState = {
  mediaUploadCount: () => number
  lastMediaUploadBody: () => string
}

async function installChatAttachmentMocks(page: Page): Promise<ChatAttachmentMockState> {
  const contact = {
    id: CONTACT_ID,
    instance_id: INSTANCE_ID,
    phone_number: '15559991234',
    name: 'Policy Contact',
    profile_name: 'Policy Contact',
    status: 'open',
    tags: [],
    metadata: {},
    is_public: true,
    last_message_at: '2026-03-01T09:00:00Z',
    last_message_preview: 'Latest message',
    unread_count: 0,
    created_at: '2026-03-01T09:00:00Z',
    updated_at: '2026-03-01T09:00:00Z',
  }

  const messagesByContact: Record<string, any[]> = {
    [CONTACT_ID]: [],
  }

  let mediaUploadCount = 0
  let lastMediaUploadBody = ''

  await page.route('**/api/config', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: {
          whatsapp_provider: 'meta',
          features: {
            templates: true,
            flows: false,
            catalog: false,
            business_profile: false,
            campaigns: true,
            meta_insights: false,
          },
        },
      }),
    })
  })

  await page.route('**/api/chats*', async route => {
    const request = route.request()
    const method = request.method()
    const url = new URL(request.url())
    const pathname = url.pathname

    if (method === 'GET' && pathname === '/api/chats') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: {
            contacts: [contact],
            total: 1,
            page: 1,
            limit: 50,
          },
        }),
      })
      return
    }

    const messagesMatch = pathname.match(/\/api\/chats\/([^/]+)\/messages$/)
    if (method === 'GET' && messagesMatch) {
      const contactId = decodeURIComponent(messagesMatch[1])
      const messages = messagesByContact[contactId] || []
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: {
            messages,
            total: messages.length,
            page: 1,
            limit: 50,
            has_more: false,
          },
        }),
      })
      return
    }

    await route.fallback()
  })

  await page.route('**/api/contacts/**', async route => {
    const pathname = new URL(route.request().url()).pathname
    if (pathname.endsWith(`/api/contacts/${CONTACT_ID}`)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: contact,
        }),
      })
      return
    }
    if (pathname.includes('/session-data')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: { session_data: {}, panel_config: { sections: [] } },
        }),
      })
      return
    }

    await route.fallback()
  })

  await page.route('**/api/chatbot/transfers*', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: { transfers: [], total_count: 0, limit: 100, offset: 0 },
      }),
    })
  })

  await page.route('**/api/custom-actions*', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'success', data: { custom_actions: [], total: 0 } }),
    })
  })

  await page.route('**/api/tags*', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'success', data: { tags: [] } }),
    })
  })

  await page.route('**/api/users*', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'success', data: [] }),
    })
  })

  await page.route('**/api/instances*', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: [
          {
            id: INSTANCE_ID,
            name: 'Policy Instance',
            phone_number: '15559991234',
            status: 'connected',
            created_at: '2026-03-01T09:00:00Z',
            updated_at: '2026-03-01T09:00:00Z',
          },
        ],
      }),
    })
  })

  await page.route('**/api/messages/media', async route => {
    mediaUploadCount += 1
    lastMediaUploadBody = route.request().postData() || ''

    const createdMessage = {
      id: `msg-${mediaUploadCount}`,
      contact_id: CONTACT_ID,
      direction: 'outgoing',
      message_type: 'document',
      content: { body: '' },
      media_url: 'documents/uploaded.zip',
      media_mime_type: 'application/zip',
      media_filename: 'uploaded.zip',
      status: 'sent',
      created_at: '2026-03-01T09:02:00Z',
      updated_at: '2026-03-01T09:02:00Z',
    }

    messagesByContact[CONTACT_ID] = [...messagesByContact[CONTACT_ID], createdMessage]

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: createdMessage,
      }),
    })
  })

  return {
    mediaUploadCount: () => mediaUploadCount,
    lastMediaUploadBody: () => lastMediaUploadBody,
  }
}

test.describe('Chat attachment policy', () => {
  test('accepts non-media files in the chat attachment picker', async ({ page }) => {
    await installMockWebSocket(page)
    await loginAsAdmin(page)
    await installChatAttachmentMocks(page)

    await page.goto(`/chat/${CONTACT_ID}`)
    await expect(page.locator('input[type="file"]').first()).toBeAttached()

    await page.setInputFiles('input[type="file"]', {
      name: 'archive.zip',
      mimeType: 'application/zip',
      buffer: Buffer.from('PK\x03\x04zip-content'),
    })

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByRole('heading', { name: 'Send Media' })).toBeVisible()
    await expect(dialog.getByText('archive.zip')).toBeVisible()
  })

  test('blocks oversized images with category-specific limit error', async ({ page }) => {
    await installMockWebSocket(page)
    await loginAsAdmin(page)
    await installChatAttachmentMocks(page)

    await page.goto(`/chat/${CONTACT_ID}`)
    await expect(page.locator('input[type="file"]').first()).toBeAttached()

    const oversizedImage = Buffer.alloc(5 * 1024 * 1024 + 1)
    oversizedImage[0] = 0xff
    oversizedImage[1] = 0xd8
    oversizedImage[2] = 0xff

    await page.setInputFiles('input[type="file"]', {
      name: 'oversized.jpg',
      mimeType: 'image/jpeg',
      buffer: oversizedImage,
    })

    await expect(
      page.locator('[data-sonner-toast]').filter({ hasText: 'Image files can be up to 5MB' }),
    ).toBeVisible()
  })

  test('sends valid documents through /api/messages/media with derived document type', async ({ page }) => {
    await installMockWebSocket(page)
    await loginAsAdmin(page)
    const mockState = await installChatAttachmentMocks(page)

    await page.goto(`/chat/${CONTACT_ID}`)
    await expect(page.locator('input[type="file"]').first()).toBeAttached()

    await page.setInputFiles('input[type="file"]', {
      name: 'contract.zip',
      mimeType: 'application/zip',
      buffer: Buffer.from('PK\x03\x04contract'),
    })

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByRole('heading', { name: 'Send Media' })).toBeVisible()
    await dialog.getByRole('button', { name: /^Send$/ }).click()

    await expect.poll(() => mockState.mediaUploadCount()).toBe(1)
    await expect.poll(() => mockState.lastMediaUploadBody()).toContain('name="type"')
    await expect.poll(() => mockState.lastMediaUploadBody()).toContain('document')
    await expect(
      page.locator('[data-sonner-toast]').filter({ hasText: 'Media sent successfully' }),
    ).toBeVisible()
  })
})
