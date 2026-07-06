import { test, expect } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'

test.describe('Chat Typing Indicator API', () => {
  test('sends composing then paused typing presence while editing message', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/chat')
    await page.waitForLoadState('networkidle')

    const chatRows = page.locator('.cursor-pointer').filter({
      has: page.locator('button[aria-label^="Delete chat:"]'),
    })
    await expect(chatRows.first()).toBeVisible()

    const states: string[] = []
    await page.route('**/api/contacts/*/typing', async route => {
      const request = route.request()
      if (request.method() !== 'POST') {
        await route.fallback()
        return
      }

      try {
        const payload = request.postDataJSON() as { state?: string }
        if (typeof payload?.state === 'string') {
          states.push(payload.state)
        }
      } catch {
        // Ignore malformed body in assertion helper
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'ok' }),
      })
    })

    let messageInput = page.locator('textarea[placeholder*="message" i], input[placeholder*="message" i]').first()
    let hasWritableComposer = false
    const attempts = Math.min(await chatRows.count(), 8)

    for (let i = 0; i < attempts; i += 1) {
      await chatRows.nth(i).click()
      await page.waitForLoadState('networkidle')

      messageInput = page.locator('textarea[placeholder*="message" i], input[placeholder*="message" i]').first()
      if (!(await messageInput.isVisible())) {
        continue
      }

      const sendButton = page.locator('button[type="submit"]').first()
      if (await sendButton.isDisabled()) {
        continue
      }

      hasWritableComposer = true
      break
    }

    test.skip(!hasWritableComposer, 'No writable chat composer found in seeded data')
    await expect(messageInput).toBeVisible()

    await messageInput.fill(`Typing indicator ${Date.now()}`)
    await page.waitForTimeout(300)
    await messageInput.fill('')
    await page.waitForTimeout(500)

    expect(states).toContain('composing')
    expect(states).toContain('paused')
  })
})
