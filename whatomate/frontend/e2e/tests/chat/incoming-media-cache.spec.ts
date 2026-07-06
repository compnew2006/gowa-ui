import { test, expect, type Page } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'

const INSTANCE_ID = '99999999-9999-9999-9999-999999999999'
const CONTACT_ID = '88888888-8888-8888-8888-888888888888'
const MESSAGE_ID = '77777777-7777-7777-7777-777777777777'

async function installMockWebSocket(page: Page) {
  await page.addInitScript(() => {
    class MockWebSocket {
      static CONNECTING = 0
      static OPEN = 1
      static CLOSING = 2
      static CLOSED = 3
      static instances: MockWebSocket[] = []

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
        MockWebSocket.instances.push(this)
        setTimeout(() => this.onopen?.(new Event('open')), 0)
      }

      send(_data: string) {}

      close() {
        this.readyState = MockWebSocket.CLOSED
        this.onclose?.(new CloseEvent('close'))
      }
    }

    ;(window as any).__mockWebSocket = {
      pushServerMessage(message: any) {
        const payload = JSON.stringify(message)
        for (const socket of MockWebSocket.instances) {
          socket.onmessage?.({ data: payload } as MessageEvent)
        }
      },
    }

    ;(window as any).WebSocket = MockWebSocket
  })
}

async function pushMockServerMessage(page: Page, message: any) {
  await page.evaluate((serverMessage) => {
    ;(window as any).__mockWebSocket.pushServerMessage(serverMessage)
  }, message)
}

test.describe('Incoming Media Cache', () => {
  test('prefetches incoming media once and reuses persistent cache on reload', async ({ page }) => {
    await installMockWebSocket(page)
    await loginAsAdmin(page)

    const contact = {
      id: CONTACT_ID,
      instance_id: INSTANCE_ID,
      phone_number: '15559990000',
      name: 'Media Contact',
      profile_name: 'Media Contact',
      status: 'pending',
      tags: [],
      metadata: {},
      last_message_at: '2026-02-28T10:00:00Z',
      last_message_preview: 'Initial',
      unread_count: 0,
      created_at: '2026-02-28T10:00:00Z',
      updated_at: '2026-02-28T10:00:00Z',
    }

    const mediaMessage = {
      id: MESSAGE_ID,
      contact_id: CONTACT_ID,
      instance_id: INSTANCE_ID,
      direction: 'incoming',
      message_type: 'image',
      content: { body: '' },
      media_url: 'images/incoming.png',
      media_mime_type: 'image/png',
      media_filename: 'incoming.png',
      status: 'received',
      created_at: '2026-02-28T10:01:00Z',
      updated_at: '2026-02-28T10:01:00Z',
    }

    const messagesByContact: Record<string, any[]> = {
      [CONTACT_ID]: [],
    }

    let mediaRequestCount = 0

    await page.route('**/api/config', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: {
            whatsapp_provider: 'whatsmeow',
            features: {
              templates: false,
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
        const status = url.searchParams.get('status')
        const contacts = status === 'pending' ? [contact] : []
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            data: {
              contacts,
              total: contacts.length,
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
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', data: { custom_actions: [], total: 0 } }) })
    })

    await page.route('**/api/tags*', async route => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', data: { tags: [] } }) })
    })

    await page.route('**/api/users*', async route => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', data: [] }) })
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
              name: 'prefetch-instance',
              status: 'connected',
              phone_number: '15551234567',
              settings: {
                auto_download_incoming_media: true,
              },
            },
          ],
        }),
      })
    })

    await page.route('**/api/media/*', async route => {
      mediaRequestCount += 1
      await route.fulfill({
        status: 200,
        contentType: 'image/png',
        body: Buffer.from('fake-image-data'),
      })
    })

    await page.goto(`/chat/${CONTACT_ID}`)
    await page.waitForLoadState('networkidle')

    messagesByContact[CONTACT_ID] = [mediaMessage]
    await pushMockServerMessage(page, {
      type: 'new_message',
      payload: mediaMessage,
    })

    await expect.poll(() => mediaRequestCount).toBe(1)
    await page.reload()
    await page.waitForLoadState('networkidle')

    await expect.poll(() => mediaRequestCount).toBe(1)
  })
})
