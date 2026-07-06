import { test, expect } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'

test.describe('Chat Bubble Phone Number Masking', () => {
  let chatPage: ChatPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    
    // Enable "Mask Phone Numbers" in settings
    await page.goto('/settings/general')
    await page.waitForLoadState('networkidle')
    
    const maskSwitch = page.locator('button[role="switch"]').filter({ hasText: /Mask Phone/i })
    const isChecked = await maskSwitch.getAttribute('aria-checked')
    if (isChecked === 'false') {
      await maskSwitch.click()
      const saveBtn = page.locator('button').filter({ hasText: /^Save/i })
      if (await saveBtn.isVisible()) {
        await saveBtn.click()
      }
      await page.waitForTimeout(1000)
    }

    chatPage = new ChatPage(page)
    await chatPage.goto()
  })

  test.afterEach(async ({ page }) => {
    // Disable "Mask Phone Numbers" in settings to clean up
    await page.goto('/settings/general')
    await page.waitForLoadState('networkidle')
    const maskSwitch = page.locator('button[role="switch"]').filter({ hasText: /Mask Phone/i })
    const isChecked = await maskSwitch.getAttribute('aria-checked')
    if (isChecked === 'true') {
      await maskSwitch.click()
      const saveBtn = page.locator('button').filter({ hasText: /^Save/i })
      if (await saveBtn.isVisible()) {
        await saveBtn.click()
      }
      await page.waitForTimeout(1000)
    }
  })

  test('should mask international phone numbers in chat messages', async ({ page }) => {
    const contacts = page.locator('.cursor-pointer').filter({ has: page.locator('text=/[+0-9]|contact/i') })
    const count = await contacts.count()
    if (count === 0) test.skip()

    await contacts.first().click()
    await page.waitForLoadState('networkidle')
    
    const messageInput = page.locator('textarea, input[placeholder*="message" i]').first()
    await expect(messageInput).toBeVisible()
    
    const uniqueMsg = `Test msg with phone +1 234 567 8900 - ${Date.now()}`
    await messageInput.fill(uniqueMsg)
    
    const sendBtn = page.locator('button').filter({ has: page.locator('.lucide-send') })
    await sendBtn.click()
    
    // Wait for the message to appear in the chat
    await page.waitForTimeout(2000)
    
    // Verify that the unmasked string DOES NOT exist in the DOM
    const rawContent = await page.locator('body').innerText()
    expect(rawContent).not.toContain('+1 234 567 8900')
    
    // Verify that the MASKED string DOES exist in the DOM
    // For +1 234 567 8900 (len=15), we expect 11 stars + 8900
    expect(rawContent).toContain('***********8900')
  })
})
