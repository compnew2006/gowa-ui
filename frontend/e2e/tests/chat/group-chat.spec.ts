import { test, expect, type Page } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'

const INSTANCE_ACC1_ID = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
const INSTANCE_ACC2_ID = 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'

const GROUP_CONTACT_ACC1_ID = '11111111-1111-1111-1111-111111111111'
const GROUP_CONTACT_ACC2_ID = '22222222-2222-2222-2222-222222222222'
const DIRECT_CONTACT_ACC1_ID = '33333333-3333-3333-3333-333333333333'
const DIRECT_CONTACT_ACC2_ID = '44444444-4444-4444-4444-444444444444'

const GROUP_JID = '120363123456789012@g.us'

type SentMessageCapture = {
  contactId: string
  body: any
}

type MockChatApiOptions = {
  contacts: any[]
  messagesByContact: Record<string, any[]>
  onSendMessage?: (capture: SentMessageCapture) => void
}

function buildMessagesResponse(messages: any[]) {
  return {
    status: 'success',
    data: {
      messages,
      total: messages.length,
      page: 1,
      limit: 50,
      has_more: false,
    },
  }
}

function buildInstancesResponse() {
  return {
    status: 'success',
    data: [
      {
        id: INSTANCE_ACC1_ID,
        name: 'acc1',
        status: 'connected',
        phone_number: '15550000001',
      },
      {
        id: INSTANCE_ACC2_ID,
        name: 'acc2',
        status: 'connected',
        phone_number: '15550000002',
      },
    ],
  }
}

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
      sentMessages: string[] = []

      constructor(_url: string) {
        MockWebSocket.instances.push(this)
        setTimeout(() => {
          this.onopen?.(new Event('open'))
        }, 0)
      }

      send(data: string) {
        this.sentMessages.push(data)
      }

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

async function mockChatApi(page: Page, options: MockChatApiOptions) {
  await page.route('**/api/contacts**', async route => {
    const url = new URL(route.request().url())
    const pathname = url.pathname
    const method = route.request().method()

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

    if (pathname.includes('/notes')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: { notes: [], has_more: false },
        }),
      })
      return
    }

    const messagesMatch = pathname.match(/\/api\/contacts\/([^/]+)\/messages$/)
    if (messagesMatch) {
      const contactId = decodeURIComponent(messagesMatch[1])

      if (method === 'POST') {
        const body = route.request().postDataJSON() as any
        options.onSendMessage?.({ contactId, body })
        const createdAt = '2026-02-18T12:00:00Z'

        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            data: {
              id: `sent-${contactId}`,
              contact_id: contactId,
              instance_id: body.instance_id,
              conversation_id: options.contacts.find(contact => contact.id === contactId)?.conversation_id,
              is_group_chat: options.contacts.find(contact => contact.id === contactId)?.is_group_chat,
              direction: 'outgoing',
              message_type: body.type,
              content: body.content,
              status: 'sent',
              created_at: createdAt,
              updated_at: createdAt,
            },
          }),
        })
        return
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(buildMessagesResponse(options.messagesByContact[contactId] || [])),
      })
      return
    }

    if (pathname.endsWith('/contacts')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: {
            contacts: options.contacts,
            total: options.contacts.length,
            page: 1,
            limit: 25,
          },
        }),
      })
      return
    }

    await route.fallback()
  })

  await page.route('**/api/chatbot/transfers**', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: {
          transfers: [],
          general_queue_count: 0,
          team_queue_counts: {},
          total_count: 0,
          limit: 100,
          offset: 0,
        },
      }),
    })
  })

  await page.route('**/api/custom-actions**', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: { custom_actions: [], total: 0 },
      }),
    })
  })

  await page.route('**/api/tags**', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: { tags: [] },
      }),
    })
  })

  await page.route('**/api/users**', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: [],
      }),
    })
  })

  await page.route('**/api/instances**', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(buildInstancesResponse()),
    })
  })
}

test.describe('Group Chat Conversation', () => {
  test('shows one group row per instance and sends using the selected instance', async ({ page }) => {
    const contacts = [
      {
        id: GROUP_CONTACT_ACC1_ID,
        instance_id: INSTANCE_ACC1_ID,
        conversation_id: GROUP_JID,
        is_group_chat: true,
        phone_number: GROUP_JID,
        name: 'Support Group',
        profile_name: 'Support Group',
        status: 'active',
        tags: [],
        metadata: { is_group_chat: true, group_jid: GROUP_JID },
        last_message_at: '2026-02-18T10:00:00Z',
        last_message_preview: 'Group latest message acc1',
        unread_count: 1,
        created_at: '2026-02-17T10:00:00Z',
        updated_at: '2026-02-18T10:00:00Z',
      },
      {
        id: GROUP_CONTACT_ACC2_ID,
        instance_id: INSTANCE_ACC2_ID,
        conversation_id: GROUP_JID,
        is_group_chat: true,
        phone_number: GROUP_JID,
        name: 'Support Group',
        profile_name: 'Support Group',
        status: 'active',
        tags: [],
        metadata: { is_group_chat: true, group_jid: GROUP_JID },
        last_message_at: '2026-02-18T11:00:00Z',
        last_message_preview: 'Group latest message acc2',
        unread_count: 2,
        created_at: '2026-02-17T09:00:00Z',
        updated_at: '2026-02-18T11:00:00Z',
      },
    ]

    const messagesByContact: Record<string, any[]> = {
      [GROUP_CONTACT_ACC1_ID]: [
        {
          id: 'aaaaaaa1-aaaa-aaaa-aaaa-aaaaaaaaaaa1',
          contact_id: GROUP_CONTACT_ACC1_ID,
          instance_id: INSTANCE_ACC1_ID,
          conversation_id: GROUP_JID,
          is_group_chat: true,
          sender_phone: '15551230001',
          direction: 'incoming',
          message_type: 'text',
          content: { body: 'Message from acc1 member' },
          status: 'received',
          created_at: '2026-02-18T09:59:00Z',
          updated_at: '2026-02-18T09:59:00Z',
        },
      ],
      [GROUP_CONTACT_ACC2_ID]: [
        {
          id: 'bbbbbbb2-bbbb-bbbb-bbbb-bbbbbbbbbbb2',
          contact_id: GROUP_CONTACT_ACC2_ID,
          instance_id: INSTANCE_ACC2_ID,
          conversation_id: GROUP_JID,
          is_group_chat: true,
          sender_phone: '15551230002',
          direction: 'incoming',
          message_type: 'text',
          content: { body: 'Message from acc2 member' },
          status: 'received',
          created_at: '2026-02-18T10:00:00Z',
          updated_at: '2026-02-18T10:00:00Z',
        },
      ],
    }

    const sentMessages: SentMessageCapture[] = []

    await loginAsAdmin(page)
    await mockChatApi(page, {
      contacts,
      messagesByContact,
      onSendMessage: capture => sentMessages.push(capture),
    })

    await page.goto('/chat')
    await page.waitForLoadState('networkidle')

    const groupConversationRows = page
      .locator('.cursor-pointer')
      .filter({ hasText: 'Support Group' })
    await expect(groupConversationRows).toHaveCount(2)

    const sidebarInstanceTags = page.locator('[data-instance-tag="true"][data-placement="sidebar"]')
    await expect(sidebarInstanceTags.filter({ hasText: 'acc1' })).toHaveCount(1)
    await expect(sidebarInstanceTags.filter({ hasText: 'acc2' })).toHaveCount(1)

    await groupConversationRows.filter({ hasText: 'acc2' }).first().click()
    await expect(page.getByText('Message from acc2 member')).toBeVisible()

    const composer = page.getByPlaceholder('Type a message...')
    await composer.fill('Sent from acc2')
    await composer.press('Enter')

    await expect.poll(() => sentMessages.length).toBe(1)
    expect(sentMessages[0].contactId).toBe(GROUP_CONTACT_ACC2_ID)
    expect(sentMessages[0].body.instance_id).toBe(INSTANCE_ACC2_ID)
  })

  test('keeps websocket messages scoped to the active chat', async ({ page }) => {
    await installMockWebSocket(page)
    await loginAsAdmin(page)

    const contacts = [
      {
        id: DIRECT_CONTACT_ACC1_ID,
        instance_id: INSTANCE_ACC1_ID,
        phone_number: '15550009991',
        name: 'acc1',
        profile_name: 'acc1',
        status: 'active',
        tags: [],
        metadata: {},
        last_message_at: '2026-02-18T10:00:00Z',
        last_message_preview: 'Initial acc1 history',
        unread_count: 0,
        created_at: '2026-02-17T10:00:00Z',
        updated_at: '2026-02-18T10:00:00Z',
      },
      {
        id: DIRECT_CONTACT_ACC2_ID,
        instance_id: INSTANCE_ACC2_ID,
        phone_number: '15550009992',
        name: 'acc2',
        profile_name: 'acc2',
        status: 'active',
        tags: [],
        metadata: {},
        last_message_at: '2026-02-18T09:00:00Z',
        last_message_preview: 'Initial acc2 history',
        unread_count: 0,
        created_at: '2026-02-17T09:00:00Z',
        updated_at: '2026-02-18T09:00:00Z',
      },
    ]

    const messagesByContact: Record<string, any[]> = {
      [DIRECT_CONTACT_ACC1_ID]: [
        {
          id: 'acc1-history-00000000-0000-0000-000000000001',
          contact_id: DIRECT_CONTACT_ACC1_ID,
          instance_id: INSTANCE_ACC1_ID,
          direction: 'outgoing',
          message_type: 'text',
          content: { body: 'Initial acc1 history' },
          status: 'sent',
          created_at: '2026-02-18T10:00:00Z',
          updated_at: '2026-02-18T10:00:00Z',
        },
      ],
      [DIRECT_CONTACT_ACC2_ID]: [
        {
          id: 'acc2-history-00000000-0000-0000-000000000001',
          contact_id: DIRECT_CONTACT_ACC2_ID,
          instance_id: INSTANCE_ACC2_ID,
          direction: 'incoming',
          message_type: 'text',
          content: { body: 'Initial acc2 history' },
          status: 'received',
          created_at: '2026-02-18T09:00:00Z',
          updated_at: '2026-02-18T09:00:00Z',
        },
      ],
    }

    await mockChatApi(page, { contacts, messagesByContact })

    await page.goto(`/chat/${DIRECT_CONTACT_ACC1_ID}`)
    await page.waitForLoadState('networkidle')
    await expect(page.getByText('Initial acc1 history')).toBeVisible()

    await pushMockServerMessage(page, {
      type: 'new_message',
      payload: {
        id: 'ws-foreign-1111-1111-1111-111111111111',
        contact_id: DIRECT_CONTACT_ACC2_ID,
        instance_id: INSTANCE_ACC2_ID,
        direction: 'incoming',
        message_type: 'text',
        content: { body: 'foreign live message' },
        status: 'received',
        created_at: '2026-02-18T12:05:00Z',
        updated_at: '2026-02-18T12:05:00Z',
      },
    })

    await expect(page.getByText('foreign live message')).toHaveCount(0)

    await pushMockServerMessage(page, {
      type: 'new_message',
      payload: {
        id: 'ws-active-1111-1111-1111-111111111111',
        contact_id: DIRECT_CONTACT_ACC1_ID,
        instance_id: INSTANCE_ACC1_ID,
        direction: 'incoming',
        message_type: 'text',
        content: { body: 'active live message' },
        status: 'received',
        created_at: '2026-02-18T12:06:00Z',
        updated_at: '2026-02-18T12:06:00Z',
      },
    })

    await expect(page.getByText('active live message')).toBeVisible()
  })

  test('does not render false deleted placeholder rows for media companion messages', async ({ page }) => {
    await loginAsAdmin(page)

    const contacts = [
      {
        id: GROUP_CONTACT_ACC1_ID,
        instance_id: INSTANCE_ACC1_ID,
        conversation_id: GROUP_JID,
        is_group_chat: true,
        phone_number: GROUP_JID,
        name: 'Support Group',
        profile_name: 'Support Group',
        status: 'active',
        tags: [],
        metadata: { is_group_chat: true, group_jid: GROUP_JID },
        last_message_at: '2026-02-18T10:00:00Z',
        last_message_preview: 'Media payload',
        unread_count: 0,
        created_at: '2026-02-17T10:00:00Z',
        updated_at: '2026-02-18T10:00:00Z',
      },
    ]

    const wamid = 'wamid-paired-placeholder-123'
    const messagesByContact: Record<string, any[]> = {
      [GROUP_CONTACT_ACC1_ID]: [
        {
          id: 'placeholder-msg-1111-1111-1111-111111111111',
          contact_id: GROUP_CONTACT_ACC1_ID,
          instance_id: INSTANCE_ACC1_ID,
          conversation_id: GROUP_JID,
          is_group_chat: true,
          sender_phone: '15551230001',
          direction: 'incoming',
          message_type: 'text',
          content: { body: '[Unsupported message type]' },
          wamid,
          status: 'received',
          created_at: '2026-02-18T10:01:00Z',
          updated_at: '2026-02-18T10:01:00Z',
        },
        {
          id: 'media-msg-2222-2222-2222-222222222222',
          contact_id: GROUP_CONTACT_ACC1_ID,
          instance_id: INSTANCE_ACC1_ID,
          conversation_id: GROUP_JID,
          is_group_chat: true,
          sender_phone: '15551230001',
          direction: 'incoming',
          message_type: 'image',
          content: { body: 'Media payload' },
          wamid,
          status: 'received',
          created_at: '2026-02-18T10:01:00Z',
          updated_at: '2026-02-18T10:01:00Z',
        },
      ],
    }

    await mockChatApi(page, { contacts, messagesByContact })

    await page.goto(`/chat/${GROUP_CONTACT_ACC1_ID}`)
    await page.waitForLoadState('networkidle')

    await expect(page.getByText('Media payload')).toBeVisible()
    await expect(page.getByText('(This message was deleted)')).toHaveCount(0)
    await expect(page.getByText('[Unsupported message type]')).toHaveCount(0)
  })

  test('sends direct chat messages with the selected contact instance', async ({ page }) => {
    await loginAsAdmin(page)

    const contacts = [
      {
        id: DIRECT_CONTACT_ACC1_ID,
        instance_id: INSTANCE_ACC1_ID,
        phone_number: '15550009991',
        name: 'Direct acc1',
        profile_name: 'Direct acc1',
        status: 'active',
        tags: [],
        metadata: {},
        last_message_at: '2026-02-18T10:00:00Z',
        last_message_preview: 'Direct acc1 history',
        unread_count: 0,
        created_at: '2026-02-17T10:00:00Z',
        updated_at: '2026-02-18T10:00:00Z',
      },
      {
        id: DIRECT_CONTACT_ACC2_ID,
        instance_id: INSTANCE_ACC2_ID,
        phone_number: '15550009992',
        name: 'Direct acc2',
        profile_name: 'Direct acc2',
        status: 'active',
        tags: [],
        metadata: {},
        last_message_at: '2026-02-18T09:00:00Z',
        last_message_preview: 'Direct acc2 history',
        unread_count: 0,
        created_at: '2026-02-17T09:00:00Z',
        updated_at: '2026-02-18T09:00:00Z',
      },
    ]

    const messagesByContact: Record<string, any[]> = {
      [DIRECT_CONTACT_ACC1_ID]: [
        {
          id: 'direct-acc1-history-0001',
          contact_id: DIRECT_CONTACT_ACC1_ID,
          instance_id: INSTANCE_ACC1_ID,
          direction: 'incoming',
          message_type: 'text',
          content: { body: 'Direct acc1 history' },
          status: 'received',
          created_at: '2026-02-18T10:00:00Z',
          updated_at: '2026-02-18T10:00:00Z',
        },
      ],
      [DIRECT_CONTACT_ACC2_ID]: [
        {
          id: 'direct-acc2-history-0001',
          contact_id: DIRECT_CONTACT_ACC2_ID,
          instance_id: INSTANCE_ACC2_ID,
          direction: 'incoming',
          message_type: 'text',
          content: { body: 'Direct acc2 history' },
          status: 'received',
          created_at: '2026-02-18T09:00:00Z',
          updated_at: '2026-02-18T09:00:00Z',
        },
      ],
    }

    const sentMessages: SentMessageCapture[] = []

    await mockChatApi(page, {
      contacts,
      messagesByContact,
      onSendMessage: capture => sentMessages.push(capture),
    })

    await page.goto('/chat')
    await page.waitForLoadState('networkidle')

    const sidebarInstanceTags = page.locator('[data-instance-tag="true"][data-placement="sidebar"]')
    await expect(sidebarInstanceTags.filter({ hasText: 'acc1' })).toHaveCount(1)
    await expect(sidebarInstanceTags.filter({ hasText: 'acc2' })).toHaveCount(1)

    await page.locator('.cursor-pointer').filter({ hasText: 'Direct acc1' }).first().click()
    await expect(page.getByText('Direct acc1 history')).toBeVisible()

    const composer = page.getByPlaceholder('Type a message...')
    await composer.fill('Direct send via acc1')
    await composer.press('Enter')

    await expect.poll(() => sentMessages.length).toBe(1)
    expect(sentMessages[0].contactId).toBe(DIRECT_CONTACT_ACC1_ID)
    expect(sentMessages[0].body.instance_id).toBe(INSTANCE_ACC1_ID)

    await page.locator('.cursor-pointer').filter({ hasText: 'Direct acc2' }).first().click()
    await expect(page.getByText('Direct acc2 history')).toBeVisible()

    await composer.fill('Direct send via acc2')
    await composer.press('Enter')

    await expect.poll(() => sentMessages.length).toBe(2)
    expect(sentMessages[1].contactId).toBe(DIRECT_CONTACT_ACC2_ID)
    expect(sentMessages[1].body.instance_id).toBe(INSTANCE_ACC2_ID)
  })
})
