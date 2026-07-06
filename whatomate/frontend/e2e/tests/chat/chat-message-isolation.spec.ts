import { test, expect } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'

test.describe('Chat Message Isolation', () => {
  let chatPage: ChatPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    chatPage = new ChatPage(page)
    await chatPage.goto()
  })

  test('should not append messages from other contacts into the active chat', async ({ page }) => {
    const contacts = page.locator('.cursor-pointer').filter({ has: page.locator('text=/[+0-9]/') })
    const count = await contacts.count()
    if (count === 0) test.skip()

    await contacts.first().click()
    await page.waitForLoadState('networkidle')

    // Simulate incoming message from completely different contact ID
    await page.evaluate(() => {
      if (window.__WHM_WS_TEST_EMIT__) {
         const dummyContactId = "e87f8eb4-500b-4654-8abc-03bc5561aaaa";
         window.__WHM_WS_TEST_EMIT__("new_message", {
           id: "dummy-msg-id-1234",
           contact_id: dummyContactId,
           conversation_id: "+1234567890@c.us",
           profile_name: "Contaminator",
           direction: "incoming",
           message_type: "text",
           content: { body: "DO_NOT_SHOW_THIS_MESSAGE" },
           created_at: new Date().toISOString(),
           updated_at: new Date().toISOString(),
         });
      }
    });

    await page.waitForTimeout(1000)

    // The message DO_NOT_SHOW_THIS_MESSAGE should NOT be in the DOM
    // Because it belongs to a different contact_id
    const content = await page.locator('body').innerText()
    expect(content).not.toContain("DO_NOT_SHOW_THIS_MESSAGE")
  })
})
