import { expect, test, type Page, type Route } from '@playwright/test'

import { loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'

const CONTACT_ID = '99999999-0000-0000-0000-000000000201'
const IMAGE_REPLY_MESSAGE_ID = '99999999-0000-0000-0000-000000000202'
const STATUS_MEDIA_URL = '/api/statuses/00000000-0000-0000-0000-00000000a999/media'

const CONTACT = {
  id: CONTACT_ID,
  phone_number: '+15551230000',
  name: 'Status Reply Contact',
  profile_name: 'Status Reply Contact',
  status: 'pending',
  tags: [],
  metadata: {},
  unread_count: 0,
  last_message_at: '2026-03-03T10:04:00Z',
  last_message_preview: 'Reply from status',
  created_at: '2026-03-03T10:00:00Z',
  updated_at: '2026-03-03T10:04:00Z',
}

const IMAGE_REPLY_MESSAGE = {
  id: IMAGE_REPLY_MESSAGE_ID,
  contact_id: CONTACT_ID,
  direction: 'incoming',
  message_type: 'text',
  content: { body: 'Seen your status' },
  status: 'delivered',
  is_reply: true,
  reply_to_message_id: 'wamid.status.thumbnail',
  reply_to_message: {
    id: 'wamid.status.thumbnail',
    content: { body: '' },
    message_type: 'image',
    direction: 'outgoing',
    sender_phone: '+15551230000',
    media_url: STATUS_MEDIA_URL,
    media_mime_type: 'image/png',
    media_filename: 'status-thumbnail.png',
  },
  created_at: '2026-03-03T10:04:00Z',
  updated_at: '2026-03-03T10:04:00Z',
}

function chatsEnvelope(contacts: any[]) {
  return {
    status: 'success',
    data: {
      contacts,
      total: contacts.length,
      page: 1,
      limit: 50,
    },
  }
}

function messagesEnvelope(messages: any[], hasMore: boolean) {
  return {
    status: 'success',
    data: {
      messages,
      total: messages.length,
      page: 1,
      limit: 50,
      has_more: hasMore,
    },
  }
}

async function setupReplyThumbnailMocks(page: Page) {
  await page.route('**/api/chats**', async (route: Route) => {
    const requestURL = new URL(route.request().url())
    const { pathname, searchParams } = requestURL

    const messagesPathMatch = pathname.match(/\/api\/chats\/([^/]+)\/messages$/)
    if (messagesPathMatch) {
      const contactID = decodeURIComponent(messagesPathMatch[1])
      if (contactID !== CONTACT_ID) {
        await route.fallback()
        return
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(messagesEnvelope([IMAGE_REPLY_MESSAGE], false)),
      })
      return
    }

    if (pathname === '/api/chats') {
      const status = searchParams.get('status')
      const contacts = status === 'pending' ? [CONTACT] : []
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(chatsEnvelope(contacts)),
      })
      return
    }

    await route.fallback()
  })

  await page.route('**/api/statuses/*/media', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'image/png',
      body: Buffer.from(
        'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9M4WAAAAAASUVORK5CYII=',
        'base64',
      ),
    })
  })

  await page.route('**/api/contacts/**', async (route: Route) => {
    const requestURL = new URL(route.request().url())
    const { pathname } = requestURL

    if (pathname.endsWith(`/contacts/${CONTACT_ID}/session-data`)) {
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

    if (pathname.endsWith(`/contacts/${CONTACT_ID}/notes`)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: { notes: [], total: 0, has_more: false },
        }),
      })
      return
    }

    await route.fallback()
  })

  await page.route('**/api/chatbot/transfers**', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: {
          transfers: [],
          total_count: 0,
          general_queue_count: 0,
          team_queue_counts: {},
          limit: 100,
          offset: 0,
        },
      }),
    })
  })

  await page.route('**/api/custom-actions**', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'success', data: { custom_actions: [], total: 0 } }),
    })
  })

  await page.route('**/api/tags**', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'success', data: { tags: [], total: 0 } }),
    })
  })

  await page.route('**/api/users**', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'success', data: { users: [], total: 0, page: 1, limit: 50 } }),
    })
  })

  await page.route('**/api/instances**', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'success', data: [] }),
    })
  })
}

test.describe('Reply Preview Thumbnail', () => {
  test('shows image thumbnail when reply preview has media_url', async ({ page }) => {
    await loginAsAdmin(page)
    await setupReplyThumbnailMocks(page)

    const chatPage = new ChatPage(page)
    await chatPage.goto(CONTACT_ID)

    const replyPreview = page.locator(`#message-${IMAGE_REPLY_MESSAGE_ID} .reply-preview`)
    await expect(replyPreview).toBeVisible()

    const thumbnail = page.locator(`#message-${IMAGE_REPLY_MESSAGE_ID} .reply-preview-thumb`)
    await expect(thumbnail).toBeVisible()
    await expect(thumbnail).toHaveAttribute('src', STATUS_MEDIA_URL)

    await thumbnail.click()
    await expect(page.getByTestId('chat-media-viewer-dialog')).toBeVisible()
    await expect(page.getByTestId('chat-media-viewer-image')).toHaveAttribute('src', STATUS_MEDIA_URL)

    await expect(page.locator(`#message-${IMAGE_REPLY_MESSAGE_ID} .reply-preview .truncate`)).toContainText('[Photo]')
  })
})
