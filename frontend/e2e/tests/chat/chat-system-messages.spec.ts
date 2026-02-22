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
})
