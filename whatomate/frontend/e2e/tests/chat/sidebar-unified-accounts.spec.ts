import { expect, test, type Page, type Route } from '@playwright/test'

import { loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'

const CONTACT_A_ID = '00000000-0000-0000-0000-000000000101'
const CONTACT_B_ID = '00000000-0000-0000-0000-000000000102'
const INSTANCE_A_ID = '00000000-0000-0000-0000-000000000201'
const INSTANCE_B_ID = '00000000-0000-0000-0000-000000000202'

const CONTACT_A = {
  id: CONTACT_A_ID,
  instance_id: INSTANCE_A_ID,
  phone_number: '+15551230000',
  name: 'Shared Contact',
  profile_name: 'Shared Contact',
  status: 'pending',
  unread_count: 1,
  whatsapp_account: 'account-a',
  created_at: '2026-02-20T10:00:00Z',
  updated_at: '2026-02-20T10:00:00Z',
  last_message_at: '2026-02-20T10:00:00Z'
}

const CONTACT_B = {
  id: CONTACT_B_ID,
  instance_id: INSTANCE_B_ID,
  phone_number: '+15551230000',
  name: 'Shared Contact',
  profile_name: 'Shared Contact',
  status: 'pending',
  unread_count: 2,
  whatsapp_account: 'account-b',
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
      instance_id: INSTANCE_A_ID,
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
      instance_id: INSTANCE_B_ID,
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

async function setupMockRoutes(
  page: Page,
  options?: {
    messageDelayByContactID?: Partial<Record<string, number>>
    onSendMessage?: (capture: { contactID: string; body: any }) => void
  }
) {
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
    if (status === 'open') {
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
    const delayMs = options?.messageDelayByContactID?.[contactID] ?? 0

    const baseMessages = CONTACT_MESSAGES[contactID] || []
    const filteredMessages = account
      ? baseMessages.filter((message) => message.whatsapp_account === account)
      : baseMessages

    if (delayMs > 0) {
      await new Promise((resolve) => setTimeout(resolve, delayMs))
    }
    await route.fulfill({ json: messagesEnvelope(filteredMessages) })
  })

  await page.route('**/api/contacts/*/messages', async (route: Route) => {
    if (route.request().method() !== 'POST') {
      await route.fallback()
      return
    }

    const url = new URL(route.request().url())
    const parts = url.pathname.split('/')
    const contactID = parts[parts.indexOf('contacts') + 1]
    const body = route.request().postDataJSON() as any
    options?.onSendMessage?.({ contactID, body })

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: {
          id: `sent-${contactID}`,
          contact_id: contactID,
          direction: 'outgoing',
          message_type: body?.type || 'text',
          content: body?.content || { body: '' },
          status: 'sent',
          whatsapp_account: body?.whatsapp_account,
          instance_id: body?.instance_id,
          created_at: '2026-02-20T10:02:00Z',
          updated_at: '2026-02-20T10:02:00Z',
        },
      }),
    })
  })

  await page.route('**/api/instances*', async (route: Route) => {
    await route.fulfill({
      json: {
        status: 'success',
        data: [
          { id: INSTANCE_A_ID, name: 'Instance A', status: 'connected' },
          { id: INSTANCE_B_ID, name: 'Instance B', status: 'connected' },
        ],
      },
    })
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
    const sharedContacts = sidebar.locator('[data-testid="chat-sidebar-entry"]').filter({ hasText: 'Shared Contact' })

    await expect(sharedContacts).toHaveCount(1)
    await expect(sharedContacts.first().locator('[data-testid="sidebar-multi-instance-tags"]')).toBeVisible()
    const instanceTags = sharedContacts.first().locator('[data-instance-tag="true"][data-placement="sidebar"]')
    await expect(instanceTags).toHaveCount(2)
    const tagTexts = (await instanceTags.allTextContents()).map(text => text.trim())
    expect(tagTexts).toContain('Instance A')
    expect(tagTexts).toContain('Instance B')
    await expect(chatPage.getAccountTab('Instance A')).toBeVisible()
    await expect(chatPage.getAccountTab('Instance B')).toBeVisible()
    await expect(page.getByText('Message from account A')).toBeVisible()

    await chatPage.switchAccount('Instance B')

    await expect(page).toHaveURL(new RegExp(`/chat/${CONTACT_B_ID}$`))
    await expect(page.getByText('Message from account B')).toBeVisible()
  })

  test('routes outgoing send from the selected merged-instance tab', async ({ page }) => {
    const sentPayloads: Array<{ contactID: string; body: any }> = []
    await setupMockRoutes(page, {
      onSendMessage: capture => sentPayloads.push(capture),
    })

    const chatPage = new ChatPage(page)
    await chatPage.goto(CONTACT_A_ID)
    await chatPage.searchContacts('Shared Contact')
    await expect(chatPage.getAccountTab('Instance B')).toBeVisible()

    await chatPage.switchAccount('Instance B')
    await page.getByPlaceholder('Type a message...').fill('Sending from B instance')
    await page.getByPlaceholder('Type a message...').press('Enter')

    await expect.poll(() => sentPayloads.length).toBe(1)
    expect(sentPayloads[0].contactID).toBe(CONTACT_B_ID)
    expect(sentPayloads[0].body.instance_id).toBe(INSTANCE_B_ID)
  })

  test('keeps latest selected chat visible when an older message request resolves late', async ({ page }) => {
    await setupMockRoutes(page, {
      messageDelayByContactID: {
        [CONTACT_A_ID]: 900,
        [CONTACT_B_ID]: 50
      }
    })

    await page.goto(`/chat/${CONTACT_A_ID}`)
    await page.waitForTimeout(80)
    await page.goto(`/chat/${CONTACT_B_ID}`)

    await expect(page).toHaveURL(new RegExp(`/chat/${CONTACT_B_ID}$`))
    await expect(page.getByText('Message from account B')).toBeVisible()
    await page.waitForTimeout(1000)
    await expect(page.getByText('Message from account B')).toBeVisible()
    await expect(page.getByText('Message from account A')).not.toBeVisible()
  })
})
