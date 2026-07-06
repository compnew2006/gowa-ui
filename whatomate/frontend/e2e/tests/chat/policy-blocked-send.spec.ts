import { test, expect } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'

test.describe('Chat Policy-Blocked Send', () => {
  test('shows reason_code message and does not clear composer when policy blocks send', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/chat')
    await page.waitForLoadState('networkidle')

    const chatRows = page.locator('.cursor-pointer').filter({
      has: page.locator('button[aria-label^="Delete chat:"]'),
    })
    await expect(chatRows.first()).toBeVisible()
    await chatRows.first().click()
    await page.waitForLoadState('networkidle')

    let sendRequests = 0
    await page.route('**/api/contacts/*/messages', async route => {
      const request = route.request()
      if (request.method() !== 'POST') {
        await route.fallback()
        return
      }

      sendRequests += 1
      await route.fulfill({
        status: 403,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'error',
          message: 'blocked by policy',
          reason_code: 'POLICY_NO_INBOUND',
          details: {
            reason_code: 'POLICY_NO_INBOUND',
          },
        }),
      })
    })

    const messageInput = page.locator('textarea[placeholder*="message" i], input[placeholder*="message" i]').first()
    const text = `Policy blocked ${Date.now()}`

    await expect(messageInput).toBeVisible()
    await messageInput.fill(text)
    await messageInput.press('Enter')

    await expect(
      page.locator('[data-sonner-toast]').filter({
        hasText: 'Cannot send: inbound-only policy is active (POLICY_NO_INBOUND)',
      }),
    ).toBeVisible()
    await expect(messageInput).toHaveValue(text)
    expect(sendRequests).toBe(1)
  })
})
