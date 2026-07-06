import { test, expect, type Page, type Route } from '@playwright/test'

import { loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'

const CONTACT_ID = '99999999-0000-0000-0000-000000000001'
const TARGET_MESSAGE_ID = '99999999-0000-0000-0000-000000000101'
const TARGET_FOLLOWUP_ID = '99999999-0000-0000-0000-000000000102'
const INITIAL_OLDEST_MESSAGE_ID = '99999999-0000-0000-0000-000000000103'
const REPLY_MESSAGE_ID = '99999999-0000-0000-0000-000000000104'

const CONTACT = {
  id: CONTACT_ID,
  phone_number: '+15551234567',
  name: 'Quoted Message Contact',
  profile_name: 'Quoted Message Contact',
  status: 'pending',
  tags: [],
  metadata: {},
  unread_count: 0,
  last_message_at: '2026-02-18T10:04:00Z',
  last_message_preview: 'Reply that references older message',
  created_at: '2026-02-18T10:00:00Z',
  updated_at: '2026-02-18T10:04:00Z',
}

const TARGET_MESSAGE = {
  id: TARGET_MESSAGE_ID,
  contact_id: CONTACT_ID,
  direction: 'incoming',
  message_type: 'text',
  content: { body: 'Target quoted message in older history' },
  status: 'delivered',
  created_at: '2026-02-18T10:01:00Z',
  updated_at: '2026-02-18T10:01:00Z',
}

const TARGET_FOLLOWUP_MESSAGE = {
  id: TARGET_FOLLOWUP_ID,
  contact_id: CONTACT_ID,
  direction: 'incoming',
  message_type: 'text',
  content: { body: 'Another older message after target' },
  status: 'delivered',
  created_at: '2026-02-18T10:02:00Z',
  updated_at: '2026-02-18T10:02:00Z',
}

const INITIAL_OLDEST_MESSAGE = {
  id: INITIAL_OLDEST_MESSAGE_ID,
  contact_id: CONTACT_ID,
  direction: 'incoming',
  message_type: 'text',
  content: { body: 'Most recent page starts here' },
  status: 'delivered',
  created_at: '2026-02-18T10:03:00Z',
  updated_at: '2026-02-18T10:03:00Z',
}

const REPLY_MESSAGE = {
  id: REPLY_MESSAGE_ID,
  contact_id: CONTACT_ID,
  direction: 'outgoing',
  message_type: 'text',
  content: { body: 'Replying to an old message' },
  status: 'sent',
  is_reply: true,
  reply_to_message_id: TARGET_MESSAGE_ID,
  reply_to_message: {
    id: TARGET_MESSAGE_ID,
    content: { body: 'Target quoted message in older history' },
    message_type: 'text',
    direction: 'incoming',
    sender_phone: '+15551234567',
  },
  created_at: '2026-02-18T10:04:00Z',
  updated_at: '2026-02-18T10:04:00Z',
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

async function setupQuoteNavigationMocks(page: Page) {
  const beforeIdRequests: string[] = []

  await page.route('**/api/chats**', async (route: Route) => {
    const requestUrl = new URL(route.request().url())
    const { pathname, searchParams } = requestUrl

    const messagesPathMatch = pathname.match(/\/api\/chats\/([^/]+)\/messages$/)
    if (messagesPathMatch) {
      const contactId = decodeURIComponent(messagesPathMatch[1])
      if (contactId !== CONTACT_ID) {
        await route.fallback()
        return
      }

      const beforeId = searchParams.get('before_id')
      if (beforeId) {
        beforeIdRequests.push(beforeId)
      }

      if (beforeId === INITIAL_OLDEST_MESSAGE_ID) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(messagesEnvelope([TARGET_MESSAGE, TARGET_FOLLOWUP_MESSAGE], false)),
        })
        return
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(messagesEnvelope([INITIAL_OLDEST_MESSAGE, REPLY_MESSAGE], true)),
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

  await page.route('**/api/contacts/**', async (route: Route) => {
    const requestUrl = new URL(route.request().url())
    const { pathname } = requestUrl

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

  return { beforeIdRequests }
}

test.describe('Quoted Message Navigation', () => {
  test('loads missing history and jumps to the quoted original message', async ({ page }) => {
    await loginAsAdmin(page)

    const { beforeIdRequests } = await setupQuoteNavigationMocks(page)
    const chatPage = new ChatPage(page)

    await chatPage.goto(CONTACT_ID)

    const missingTarget = page.locator(`#message-${TARGET_MESSAGE_ID}`)
    await expect(missingTarget).toHaveCount(0)

    const replyPreview = page.locator(`#message-${REPLY_MESSAGE_ID} .reply-preview`)
    await expect(replyPreview).toBeVisible()

    await replyPreview.click()

    await expect.poll(() => beforeIdRequests).toContain(INITIAL_OLDEST_MESSAGE_ID)
    await expect(missingTarget).toBeVisible()
    await expect(missingTarget).toHaveClass(/highlight-message/)
  })
})
