import { expect, test, type Page, type Route } from '@playwright/test'

import { ChatPage } from '../../pages'

const INSTANCE_ID = '00000000-0000-0000-0000-000000000901'
const INSTANCE_NAME = 'Primary WA'
const SELF_JID = '15550000001@s.whatsapp.net'
const CONTACT_JID = '15550000002@s.whatsapp.net'

interface StatusItem {
  id: string
  instance_id: string
  instance_name: string
  sender_jid: string
  sender_name: string
  whatsapp_message_id: string
  status_type: 'text' | 'image' | 'video'
  content: string
  media_url?: string
  media_mime_type?: string
  media_filename?: string
  text_argb?: number
  background_argb?: number
  font?: string
  is_self: boolean
  seen_at?: string
  created_at: string
  expires_at: string
}

function successEnvelope<T>(data: T) {
  return {
    status: 'success',
    data,
  }
}

function buildStatusGroups(selfStatuses: StatusItem[], otherStatuses: StatusItem[]) {
  const groups: Array<{
    group_id: string
    instance_id: string
    instance_name: string
    sender_jid: string
    sender_name: string
    is_self: boolean
    statuses: StatusItem[]
  }> = []

  if (selfStatuses.length > 0) {
    groups.push({
      group_id: `${INSTANCE_ID}:${SELF_JID}`,
      instance_id: INSTANCE_ID,
      instance_name: INSTANCE_NAME,
      sender_jid: SELF_JID,
      sender_name: 'Test Admin',
      is_self: true,
      statuses: selfStatuses,
    })
  }

  if (otherStatuses.length > 0) {
    groups.push({
      group_id: `${INSTANCE_ID}:${CONTACT_JID}`,
      instance_id: INSTANCE_ID,
      instance_name: INSTANCE_NAME,
      sender_jid: CONTACT_JID,
      sender_name: 'External Contact',
      is_self: false,
      statuses: otherStatuses,
    })
  }

  return groups
}

async function setupMockRoutes(page: Page) {
  const selfStatuses: StatusItem[] = []
  const otherStatuses: StatusItem[] = [
    {
      id: '00000000-0000-0000-0000-00000000a001',
      instance_id: INSTANCE_ID,
      instance_name: INSTANCE_NAME,
      sender_jid: CONTACT_JID,
      sender_name: 'External Contact',
      whatsapp_message_id: 'wamid.status.1',
      status_type: 'text',
      content: 'Morning update',
      is_self: false,
      created_at: '2026-03-03T09:00:00Z',
      expires_at: '2026-03-04T09:00:00Z',
    },
  ]

  let sendTextPayload: Record<string, unknown> | null = null
  let sendReplyPayload: Record<string, unknown> | null = null
  let markSeenCalls = 0

  await page.route('**/api/instances/*/status/send', async (route: Route) => {
    if (route.request().method() !== 'POST') {
      await route.continue()
      return
    }

    const payload = route.request().postDataJSON() as Record<string, unknown>
    sendTextPayload = payload

    const id = `00000000-0000-0000-0000-00000000b00${selfStatuses.length + 1}`
    const createdAt = new Date().toISOString()
    selfStatuses.push({
      id,
      instance_id: INSTANCE_ID,
      instance_name: INSTANCE_NAME,
      sender_jid: SELF_JID,
      sender_name: 'Test Admin',
      whatsapp_message_id: `wamid.self.${selfStatuses.length + 1}`,
      status_type: String(payload.type || 'text') as 'text' | 'image' | 'video',
      content: String(payload.text || ''),
      background_argb: payload.background_argb ? Number(payload.background_argb) : undefined,
      font: payload.font ? String(payload.font) : undefined,
      is_self: true,
      created_at: createdAt,
      expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
    })

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({
        id,
        content: String(payload.text || ''),
        created_at: createdAt,
      })),
    })
  })

  await page.route('**/api/statuses/*/mark-seen', async (route: Route) => {
    if (route.request().method() !== 'POST') {
      await route.continue()
      return
    }

    markSeenCalls += 1

    const url = new URL(route.request().url())
    const parts = url.pathname.split('/')
    const statusID = parts[parts.indexOf('statuses') + 1]

    const target = otherStatuses.find((status) => status.id === statusID)
    if (target) {
      target.seen_at = new Date().toISOString()
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({ status: 'ok', seen_at: new Date().toISOString() })),
    })
  })

  await page.route('**/api/statuses/*/reply', async (route: Route) => {
    if (route.request().method() !== 'POST') {
      await route.continue()
      return
    }

    sendReplyPayload = route.request().postDataJSON() as Record<string, unknown>
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({ status: 'ok', message_id: '00000000-0000-0000-0000-00000000r001' })),
    })
  })

  await page.route('**/api/statuses*', async (route: Route) => {
    if (route.request().method() !== 'GET') {
      await route.continue()
      return
    }

    const groups = buildStatusGroups(selfStatuses, otherStatuses)
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({
        groups,
        total: selfStatuses.length + otherStatuses.length,
      })),
    })
  })

  await page.route('**/api/chats*', async (route: Route) => {
    if (route.request().method() !== 'GET') {
      await route.continue()
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({ contacts: [], total: 0, page: 1, limit: 50 })),
    })
  })

  await page.route('**/api/contacts?*', async (route: Route) => {
    if (route.request().method() !== 'GET') {
      await route.continue()
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({ contacts: [], total: 0, page: 1, limit: 50 })),
    })
  })

  await page.route('**/api/contacts', async (route: Route) => {
    if (route.request().method() !== 'GET') {
      await route.continue()
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({ contacts: [], total: 0, page: 1, limit: 50 })),
    })
  })

  await page.route('**/api/instances*', async (route: Route) => {
    if (route.request().method() !== 'GET') {
      await route.continue()
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope([
        {
          id: INSTANCE_ID,
          name: INSTANCE_NAME,
          account_name: 'primary',
          status: 'connected',
        },
      ])),
    })
  })

  await page.route('**/api/chatbot/transfers*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({ transfers: [], total: 0 })),
    })
  })

  await page.route('**/api/users*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({ users: [], total: 0 })),
    })
  })

  await page.route('**/api/tags*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({ tags: [], total: 0 })),
    })
  })

  await page.route('**/api/instances/notifications*', async (route: Route) => {
    if (route.request().method() !== 'GET') {
      await route.continue()
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope([])),
    })
  })

  await page.route('**/api/auth/ws-token', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({ token: null })),
    })
  })

  await page.route('**/api/config', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({
        whatsapp_provider: 'whatsmeow',
        features: {
          templates: false,
          flows: false,
          catalog: false,
          business_profile: false,
          campaigns: false,
          meta_insights: false,
        },
      })),
    })
  })

  await page.route('**/api/me', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({
        id: '00000000-0000-0000-0000-000000000111',
        email: 'admin@test.com',
        full_name: 'Test Admin',
        organization_id: '00000000-0000-0000-0000-000000000211',
        role: {
          id: '00000000-0000-0000-0000-000000000311',
          name: 'admin',
          is_system: false,
          permissions: [
            { id: 'perm-chat-read', resource: 'chat', action: 'read' },
            { id: 'perm-chat-write', resource: 'chat', action: 'write' },
          ],
        },
      })),
    })
  })

  await page.route('**/api/me/organizations', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope([])),
    })
  })

  await page.route('**/api/notifications*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({ notifications: [], total: 0 })),
    })
  })

  await page.route('**/api/custom-actions*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(successEnvelope({ custom_actions: [], total: 0 })),
    })
  })

  return {
    getSendTextPayload: () => sendTextPayload,
    getSendReplyPayload: () => sendReplyPayload,
    getMarkSeenCalls: () => markSeenCalls,
  }
}

test.describe('Chat Statuses', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      const user = {
        id: '00000000-0000-0000-0000-000000000111',
        email: 'admin@test.com',
        full_name: 'Test Admin',
        organization_id: '00000000-0000-0000-0000-000000000211',
        role: {
          id: '00000000-0000-0000-0000-000000000311',
          name: 'admin',
          is_system: false,
          permissions: [
            { id: 'perm-chat-read', resource: 'chat', action: 'read' },
            { id: 'perm-chat-write', resource: 'chat', action: 'write' },
          ],
        },
      }
      window.localStorage.setItem('user', JSON.stringify(user))
    })
  })

  test('shows statuses, opens viewer, sends text status, and marks seen', async ({ page }) => {
    const tracker = await setupMockRoutes(page)

    const chatPage = new ChatPage(page)
    await chatPage.goto()

    const storiesBar = page.getByTestId('status-stories-bar')
    await expect(storiesBar).toBeVisible()

    const refreshButton = storiesBar.getByRole('button', { name: 'Refresh' })
    const notificationBell = storiesBar.getByTestId('notification-bell-button')
    const drawerToggle = storiesBar.getByTestId('status-drawer-toggle')

    await expect(refreshButton).toBeVisible()
    await expect(notificationBell).toBeVisible()
    await expect(drawerToggle).toBeVisible()

    const refreshBox = await refreshButton.boundingBox()
    const bellBox = await notificationBell.boundingBox()
    const drawerBox = await drawerToggle.boundingBox()

    expect(refreshBox).not.toBeNull()
    expect(bellBox).not.toBeNull()
    expect(drawerBox).not.toBeNull()
    expect(refreshBox!.x).toBeLessThan(bellBox!.x)
    expect(bellBox!.x).toBeLessThan(drawerBox!.x)

    await notificationBell.click()
    await expect(page.getByText('No notifications')).toBeVisible()
    await expect(page.getByTestId('status-create-button')).toBeHidden()
    await page.keyboard.press('Escape')

    await storiesBar.click({ position: { x: 12, y: 12 } })

    await expect(page.getByTestId('status-create-button')).toBeVisible()

    const storyButtons = page.getByTestId('status-story-button')
    await expect(storyButtons).toHaveCount(1)

    await storyButtons.first().click()
    await expect(page.getByText('Morning update')).toBeVisible()
    await expect.poll(() => tracker.getMarkSeenCalls()).toBeGreaterThan(0)

    await page.getByTestId('status-reply-input').fill('Nice status')
    await page.getByTestId('status-reply-send-button').click()
    await expect.poll(() => tracker.getSendReplyPayload()).not.toBeNull()

    const sentReplyPayload = tracker.getSendReplyPayload() as Record<string, unknown>
    expect(sentReplyPayload.text).toBe('Nice status')

    await page.keyboard.press('Escape')

    await page.getByTestId('status-create-button').click()
    await expect(page.getByRole('dialog')).toBeVisible()

    await page.getByRole('dialog').locator('textarea').first().fill('Team sync at 2 PM')
    await page.getByTestId('status-submit-button').click()

    await expect.poll(() => tracker.getSendTextPayload()).not.toBeNull()

    const sentPayload = tracker.getSendTextPayload() as Record<string, unknown>
    expect(sentPayload.type).toBe('text')
    expect(sentPayload.text).toBe('Team sync at 2 PM')

    await storiesBar.click({ position: { x: 12, y: 12 } })
    await expect(page.getByTestId('status-create-button')).toBeHidden()
  })
})
