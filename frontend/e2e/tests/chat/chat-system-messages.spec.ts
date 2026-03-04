import { test, expect, request as playwrightRequest } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { ApiHelper } from '../../helpers/api'
import { ChatPage } from '../../pages'

test.describe('Chat System Messages', () => {
  test('shows a system message when a chat is claimed', async ({ page }) => {
    const reqContext = await playwrightRequest.newContext()
    const api = new ApiHelper(reqContext)
    await api.loginAsAdmin()

    let contacts = await api.getContacts()
    if (contacts.length === 0) {
      await api.createContact(`91${Date.now().toString().slice(-10)}`, 'System Message Contact')
      contacts = await api.getContacts()
    }

    const contactId = contacts[0].id

    // Normalize chat state so claim always executes and produces a fresh system message.
    await api.put(`/api/chats/${contactId}/reopen`).catch(() => null)
    await api.put(`/api/contacts/${contactId}/assign`, { user_id: null })

    const claimResponse = await api.put(`/api/chats/${contactId}/claim`)
    expect(claimResponse.ok()).toBeTruthy()

    await reqContext.dispose()

    await loginAsAdmin(page)
    const chatPage = new ChatPage(page)
    await chatPage.goto(contactId)

    const systemMessage = page.locator('.chat-bubble-system').filter({ hasText: 'claimed this chat' }).last()
    await expect(systemMessage).toBeVisible()
  })

  test('shows assignment system message when admin assigns chat to another agent', async ({ page }) => {
    await loginAsAdmin(page)
    await page.evaluate(() => {
      localStorage.setItem('chat.sidebarViewMode', 'unified')
    })

    const contactId = '00000000-0000-0000-0000-000000009001'
    const expectedMessage = 'System :Manager Jane has assigned this chat to Agent John'
    const contact = {
      id: contactId,
      phone_number: '+15551234567',
      name: 'Assignment Contact',
      profile_name: 'Assignment Contact',
      status: 'pending',
      unread_count: 0,
      whatsapp_account: 'account-1',
      created_at: '2026-02-20T10:00:00Z',
      updated_at: '2026-02-20T10:00:00Z',
      last_message_at: '2026-02-20T10:00:00Z',
    }

    await page.route('**/api/chats**', async (route) => {
      const url = new URL(route.request().url())
      const { pathname } = url
      const messagesPathMatch = pathname.match(/\/api\/chats\/([^/]+)\/messages$/)

      if (messagesPathMatch && decodeURIComponent(messagesPathMatch[1]) === contactId) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            data: {
              messages: [
                {
                  id: 'system-message-1',
                  contact_id: contactId,
                  direction: 'incoming',
                  message_type: 'text',
                  content: { body: expectedMessage },
                  status: 'delivered',
                  metadata: { system_event: true, event_type: 'chat_assigned' },
                  created_at: '2026-02-20T10:01:00Z',
                  updated_at: '2026-02-20T10:01:00Z',
                },
              ],
              total: 1,
              page: 1,
              limit: 50,
              has_more: false,
            },
          }),
        })
        return
      }

      if (/\/api\/chats\/?$/.test(pathname)) {
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
              limit: 25,
            },
          }),
        })
        return
      }

      await route.fallback()
    })

    await page.route('**/api/contacts/**', async (route) => {
      const url = new URL(route.request().url())
      const { pathname } = url

      if (pathname.endsWith(`/contacts/${contactId}/session-data`)) {
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

      if (pathname.endsWith(`/contacts/${contactId}/notes`)) {
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

    await page.route('**/api/chatbot/transfers**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: { transfers: [], general_queue_count: 0, team_queue_counts: {}, total_count: 0 },
        }),
      })
    })

    await page.route('**/api/custom-actions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', data: { custom_actions: [], total: 0 } }),
      })
    })

    await page.route('**/api/users**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', data: [] }),
      })
    })

    await page.route('**/api/tags**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', data: { tags: [] } }),
      })
    })

    await page.route('**/api/instances**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', data: [] }),
      })
    })

    await page.goto(`/chat/${contactId}`)
    await page.waitForLoadState('networkidle')

    const systemMessage = page
      .locator('.chat-bubble-system')
      .filter({ hasText: expectedMessage })
      .last()
    await expect(systemMessage).toBeVisible()
  })
})
