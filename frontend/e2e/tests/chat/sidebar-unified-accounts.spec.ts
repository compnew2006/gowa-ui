import { expect, test, type Page, type Route } from '@playwright/test'

import { loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'

const CONTACT_A_ID = '00000000-0000-0000-0000-000000000101'
const CONTACT_B_ID = '00000000-0000-0000-0000-000000000102'

const CONTACT_A = {
  id: CONTACT_A_ID,
  phone_number: '+15551230000',
  name: 'Shared Contact',
  profile_name: 'Shared Contact',
  status: 'open',
  unread_count: 1,
  whatsapp_account: 'account-a',
  assigned_user_id: '00000000-0000-0000-0000-000000000201',
  created_at: '2026-02-20T10:00:00Z',
  updated_at: '2026-02-20T10:00:00Z',
  last_message_at: '2026-02-20T10:00:00Z'
}

const CONTACT_B = {
  id: CONTACT_B_ID,
  phone_number: '+15551230000',
  name: 'Shared Contact',
  profile_name: 'Shared Contact',
  status: 'open',
  unread_count: 2,
  whatsapp_account: 'account-b',
  assigned_user_id: '00000000-0000-0000-0000-000000000201',
  created_at: '2026-02-20T10:01:00Z',
  updated_at: '2026-02-20T10:01:00Z',
  last_message_at: '2026-02-20T10:01:00Z'
}

const CONTACT_MESSAGES: Record<string, any[]> = {
  [CONTACT_A_ID]: [
    {
      id: '00000000-0000-0000-0000-000000001001',
      contact_id: CONTACT_A_ID,
      direction: 'incoming',
      message_type: 'text',
      content: { body: 'Message from account A' },
      status: 'delivered',
      whatsapp_account: 'account-a',
      created_at: '2026-02-20T10:00:00Z',
      updated_at: '2026-02-20T10:00:00Z'
    }
  ],
  [CONTACT_B_ID]: [
    {
      id: '00000000-0000-0000-0000-000000001002',
      contact_id: CONTACT_B_ID,
      direction: 'incoming',
      message_type: 'text',
      content: { body: 'Message from account B' },
      status: 'delivered',
      whatsapp_account: 'account-b',
      created_at: '2026-02-20T10:01:00Z',
      updated_at: '2026-02-20T10:01:00Z'
    }
  ]
}

function chatsEnvelope(contacts: any[]) {
  return {
    status: 'success',
    data: {
      contacts,
      total: contacts.length,
      page: 1,
      limit: 50
    }
  }
}

function contactsEnvelope(contacts: any[]) {
  return {
    status: 'success',
    data: {
      contacts,
      total: contacts.length,
      page: 1,
      limit: 50
    }
  }
}

function messagesEnvelope(messages: any[]) {
  return {
    status: 'success',
    data: {
      messages,
      total: messages.length,
      page: 1,
      limit: 50,
      has_more: false
    }
  }
}

async function setupMockRoutes(page: Page) {
  await page.route('**/api/contacts?*', async (route: Route) => {
    if (route.request().method() !== 'GET') {
      await route.continue()
      return
    }
    await route.fulfill({ json: contactsEnvelope([CONTACT_A, CONTACT_B]) })
  })

  await page.route('**/api/contacts', async (route: Route) => {
    if (route.request().method() !== 'GET') {
      await route.continue()
      return
    }
    await route.fulfill({ json: contactsEnvelope([CONTACT_A, CONTACT_B]) })
  })

  await page.route('**/api/chats*', async (route: Route) => {
    if (route.request().method() !== 'GET') {
      await route.continue()
      return
    }

    const url = new URL(route.request().url())
    const status = url.searchParams.get('status')
    if (status === 'pending') {
      await route.fulfill({ json: chatsEnvelope([]) })
      return
    }
    await route.fulfill({ json: chatsEnvelope([CONTACT_A, CONTACT_B]) })
  })

  await page.route('**/api/contacts/*', async (route: Route) => {
    if (route.request().method() !== 'GET') {
      await route.continue()
      return
    }

    const url = new URL(route.request().url())
    if (url.pathname.endsWith('/session-data')) {
      await route.fulfill({ json: { status: 'success', data: null } })
      return
    }

    if (url.pathname.endsWith(`/${CONTACT_A_ID}`)) {
      await route.fulfill({ json: { status: 'success', data: CONTACT_A } })
      return
    }
    if (url.pathname.endsWith(`/${CONTACT_B_ID}`)) {
      await route.fulfill({ json: { status: 'success', data: CONTACT_B } })
      return
    }

    await route.continue()
  })

  await page.route('**/api/contacts/**/session-data', async (route: Route) => {
    await route.fulfill({ json: { status: 'success', data: null } })
  })

  await page.route('**/api/chats/*/messages*', async (route: Route) => {
    const url = new URL(route.request().url())
    const parts = url.pathname.split('/')
    const contactID = parts[parts.indexOf('chats') + 1]
    const account = url.searchParams.get('account')

    const baseMessages = CONTACT_MESSAGES[contactID] || []
    const filteredMessages = account
      ? baseMessages.filter((message) => message.whatsapp_account === account)
      : baseMessages

    await route.fulfill({ json: messagesEnvelope(filteredMessages) })
  })

  await page.route('**/api/instances*', async (route: Route) => {
    await route.fulfill({ json: { status: 'success', data: [] } })
  })

  await page.route('**/api/chatbot/transfers*', async (route: Route) => {
    await route.fulfill({ json: { status: 'success', data: { transfers: [], total: 0 } } })
  })

  await page.route('**/api/users*', async (route: Route) => {
    await route.fulfill({ json: { status: 'success', data: { users: [], total: 0 } } })
  })

  await page.route('**/api/tags*', async (route: Route) => {
    await route.fulfill({ json: { status: 'success', data: { tags: [], total: 0 } } })
  })
}

test.describe('Unified Sidebar Contacts Across Accounts', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.evaluate(() => {
      localStorage.setItem('chat.sidebarViewMode', 'unified')
    })
  })

  test('merges same-number chats in sidebar and switches underlying account thread', async ({ page }) => {
    await setupMockRoutes(page)

    const chatPage = new ChatPage(page)
    await chatPage.goto(CONTACT_A_ID)
    await chatPage.searchContacts('Shared Contact')

    const sidebar = page.locator('[data-contacts-sidebar="true"]')
    const sharedContacts = sidebar.locator('p.text-sm.font-medium').filter({ hasText: 'Shared Contact' })

    await expect(sharedContacts).toHaveCount(1)
    await expect(chatPage.getAccountTab('account-a')).toBeVisible()
    await expect(chatPage.getAccountTab('account-b')).toBeVisible()
    await expect(page.getByText('Message from account A')).toBeVisible()

    await chatPage.switchAccount('account-b')

    await expect(page).toHaveURL(new RegExp(`/chat/${CONTACT_B_ID}$`))
    await expect(page.getByText('Message from account B')).toBeVisible()
  })
})
