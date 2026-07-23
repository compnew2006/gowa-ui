import { test, expect } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { ChatbotSettingsPage } from '../../pages'

test.describe('Chatbot Settings Page', () => {
  let chatbotSettingsPage: ChatbotSettingsPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    chatbotSettingsPage = new ChatbotSettingsPage(page)
    await chatbotSettingsPage.goto()
  })

  test('should display chatbot settings page', async () => {
    await chatbotSettingsPage.expectPageVisible()
  })

  // removed: tab-existence tests redundant with per-tab describe blocks below
  // (Messages/Agents/Hours/SLA/AI tab visibility is asserted via each
  // describe's expectXTabVisible() and the Tab Navigation describe).
})

test.describe('Messages Tab', () => {
  let chatbotSettingsPage: ChatbotSettingsPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    chatbotSettingsPage = new ChatbotSettingsPage(page)
    await chatbotSettingsPage.goto()
  })

  test('should show messages tab by default', async () => {
    await chatbotSettingsPage.expectMessagesTabVisible()
  })

  // Collapsed: previously five per-field existence tests (greeting, fallback,
  // timeout, add-greeting-button, add-fallback-button). Data-driven per Rule 4.
  test('Messages tab renders expected fields', async ({ page }) => {
    await expect(page.locator('textarea#greeting')).toBeVisible()
    await expect(page.locator('textarea#fallback')).toBeVisible()
    await expect(page.locator('input#timeout')).toBeVisible()
    await expect(page.getByRole('button', { name: /Add Button/i }).first()).toBeVisible()
    await expect(page.getByRole('button', { name: /Add Button/i }).last()).toBeVisible()
  })

  test('should fill greeting message', async ({ page }) => {
    await chatbotSettingsPage.fillGreetingMessage('Hello! Welcome to our service.')
    await expect(page.locator('textarea#greeting')).toHaveValue('Hello! Welcome to our service.')
  })

  test('should save messages settings', async () => {
    await chatbotSettingsPage.fillGreetingMessage('Test greeting')
    await chatbotSettingsPage.saveSettings()
    await chatbotSettingsPage.expectToast(/saved|success/i)
  })
})

test.describe('Agents Tab', () => {
  let chatbotSettingsPage: ChatbotSettingsPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    chatbotSettingsPage = new ChatbotSettingsPage(page)
    await chatbotSettingsPage.goto()
    await chatbotSettingsPage.switchToAgentsTab()
  })

  test('should show agents settings', async () => {
    await chatbotSettingsPage.expectAgentsTabVisible()
  })

  // Collapsed: previously three per-toggle existence tests. Data-driven per Rule 4.
  // exact:true anchors on the toggle label and avoids matching the
  // recent-activity / audit-log panel which renders entries like
  // "Assign To Same Agent: false" (different casing + trailing colon).
  test('Agents tab renders expected toggles', async ({ page }) => {
    await expect(page.getByText('Allow Agents to Pick from Queue', { exact: true })).toBeVisible()
    await expect(page.getByText('Assign to Same Agent', { exact: true })).toBeVisible()
    await expect(page.getByText('Agents See Current Conversation Only', { exact: true })).toBeVisible()
  })

  test('should toggle agent queue pickup', async ({ page }) => {
    const toggle = page.locator('button[role="switch"]').first()
    const initialState = await toggle.getAttribute('data-state')
    await toggle.click()
    const newState = await toggle.getAttribute('data-state')
    expect(newState).not.toBe(initialState)
  })

  test('should save agent settings', async () => {
    await chatbotSettingsPage.saveSettings()
    await chatbotSettingsPage.expectToast(/saved|success/i)
  })
})

test.describe('Business Hours Tab', () => {
  let chatbotSettingsPage: ChatbotSettingsPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    chatbotSettingsPage = new ChatbotSettingsPage(page)
    await chatbotSettingsPage.goto()
    await chatbotSettingsPage.switchToHoursTab()
  })

  test('should show business hours settings', async () => {
    await chatbotSettingsPage.expectHoursTabVisible()
  })

  // Collapsed: previously a standalone "have enable business hours toggle"
  // existence test. Combined with the out-of-hours message visibility check
  // into one field-render assertion per Rule 4. (Toggling business hours on
  // first is required to reveal the out-of-hours message field.)
  test('Business Hours tab renders expected fields', async ({ page }) => {
    await expect(page.getByText('Enable Business Hours')).toBeVisible()
    const toggle = page.locator('button[role="switch"]').first()
    const state = await toggle.getAttribute('data-state')
    if (state === 'unchecked') {
      await toggle.click()
    }
    await expect(page.getByText('Out of Hours Message')).toBeVisible()
  })

  test('should toggle business hours enabled', async ({ page }) => {
    const toggle = page.locator('button[role="switch"]').first()
    const initialState = await toggle.getAttribute('data-state')
    await toggle.click()
    const newState = await toggle.getAttribute('data-state')
    expect(newState).not.toBe(initialState)
  })

  test('should show day schedule when enabled', async ({ page }) => {
    const toggle = page.locator('button[role="switch"]').first()
    const state = await toggle.getAttribute('data-state')
    if (state === 'unchecked') {
      await toggle.click()
    }
    await expect(page.getByText('Monday')).toBeVisible()
    await expect(page.getByText('Tuesday')).toBeVisible()
  })

  // removed: standalone "have out of hours message field" visibility test —
  // folded into the data-driven "Business Hours tab renders expected fields"
  // existence test above.

  test('should save business hours settings', async () => {
    await chatbotSettingsPage.saveSettings()
    await chatbotSettingsPage.expectToast(/saved|success/i)
  })
})

test.describe('SLA Tab', () => {
  let chatbotSettingsPage: ChatbotSettingsPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    chatbotSettingsPage = new ChatbotSettingsPage(page)
    await chatbotSettingsPage.goto()
    await chatbotSettingsPage.switchToSLATab()
  })

  test('should show SLA settings', async () => {
    await chatbotSettingsPage.expectSLATabVisible()
  })

  // Collapsed: previously two per-toggle existence tests (enable SLA, client
  // inactivity reminders). Combined with the response/escalation field labels
  // into one data-driven render check per Rule 4. Toggling SLA on first is
  // required to reveal the time fields.
  test('SLA tab renders expected fields and toggles', async ({ page }) => {
    await expect(page.getByText('Enable SLA Tracking')).toBeVisible()
    await expect(page.getByText('Client Inactivity Reminders')).toBeVisible()
    const toggle = page.locator('button[role="switch"]').first()
    const state = await toggle.getAttribute('data-state')
    if (state === 'unchecked') {
      await toggle.click()
    }
    // Target labels specifically to avoid matching description text
    await expect(page.locator('label').filter({ hasText: /Response Time/i })).toBeVisible()
    await expect(page.locator('label').filter({ hasText: /Escalation Time/i })).toBeVisible()
  })

  test('should toggle SLA enabled', async ({ page }) => {
    const toggle = page.locator('button[role="switch"]').first()
    const initialState = await toggle.getAttribute('data-state')
    await toggle.click()
    const newState = await toggle.getAttribute('data-state')
    expect(newState).not.toBe(initialState)
  })

  test('should save SLA settings', async () => {
    await chatbotSettingsPage.saveSettings()
    await chatbotSettingsPage.expectToast(/saved|success/i)
  })
})

test.describe('AI Tab', () => {
  let chatbotSettingsPage: ChatbotSettingsPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    chatbotSettingsPage = new ChatbotSettingsPage(page)
    await chatbotSettingsPage.goto()
    await chatbotSettingsPage.switchToAITab()
  })

  test('should show AI settings', async () => {
    await chatbotSettingsPage.expectAITabVisible()
  })

  // Collapsed: previously four per-field existence tests (enable AI toggle,
  // AI Provider/Model labels, API key field, system prompt field). Data-driven
  // per Rule 4. Toggling AI on first is required to reveal the config fields.
  test('AI tab renders expected fields when enabled', async ({ page }) => {
    await expect(page.getByText('Enable AI Responses')).toBeVisible()
    const toggle = page.locator('button[role="switch"]').first()
    const state = await toggle.getAttribute('data-state')
    if (state === 'unchecked') {
      await toggle.click()
    }
    await expect(page.locator('label').filter({ hasText: /^AI Provider$/ })).toBeVisible()
    await expect(page.locator('label').filter({ hasText: /^Model$/ })).toBeVisible()
    await expect(page.locator('label').filter({ hasText: /^API Key$/ })).toBeVisible()
    await expect(page.getByText('System Prompt')).toBeVisible()
  })

  test('should toggle AI enabled', async ({ page }) => {
    const toggle = page.locator('button[role="switch"]').first()
    const initialState = await toggle.getAttribute('data-state')
    await toggle.click()
    const newState = await toggle.getAttribute('data-state')
    expect(newState).not.toBe(initialState)
  })

  test('should show AI providers', async ({ page }) => {
    const toggle = page.locator('button[role="switch"]').first()
    const state = await toggle.getAttribute('data-state')
    if (state === 'unchecked') {
      await toggle.click()
    }
    await page.locator('button[role="combobox"]').first().click()
    await expect(page.locator('[role="option"]').filter({ hasText: 'OpenAI' })).toBeVisible()
    await expect(page.locator('[role="option"]').filter({ hasText: 'Anthropic' })).toBeVisible()
    await page.keyboard.press('Escape')
  })

  test('should save AI settings', async () => {
    await chatbotSettingsPage.saveSettings()
    await chatbotSettingsPage.expectToast(/saved|success/i)
  })
})

test.describe('Tab Navigation', () => {
  let chatbotSettingsPage: ChatbotSettingsPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    chatbotSettingsPage = new ChatbotSettingsPage(page)
    await chatbotSettingsPage.goto()
  })

  test('should switch to Agents tab', async () => {
    await chatbotSettingsPage.switchToAgentsTab()
    await chatbotSettingsPage.expectAgentsTabVisible()
  })

  test('should switch to Hours tab', async () => {
    await chatbotSettingsPage.switchToHoursTab()
    await chatbotSettingsPage.expectHoursTabVisible()
  })

  test('should switch to SLA tab', async () => {
    await chatbotSettingsPage.switchToSLATab()
    await chatbotSettingsPage.expectSLATabVisible()
  })

  test('should switch to AI tab', async () => {
    await chatbotSettingsPage.switchToAITab()
    await chatbotSettingsPage.expectAITabVisible()
  })

  test('should switch back to Messages tab', async () => {
    await chatbotSettingsPage.switchToAITab()
    await chatbotSettingsPage.switchToMessagesTab()
    await chatbotSettingsPage.expectMessagesTabVisible()
  })
})
