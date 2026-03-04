import { test, expect } from '@playwright/test'
import { loginAsAdmin, loginAsAgent, loginAsManager } from '../../helpers'

test.describe('Agent Analytics', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/analytics/agents')
    await page.waitForLoadState('networkidle')
  })

  test('should display agent analytics page', async ({ page }) => {
    // Check for page header
    await expect(page.locator('h1')).toContainText('Agent Analytics')
  })

  test('should display time range filter', async ({ page }) => {
    // Check for time range selector
    const timeRangeSelect = page.locator('button[role="combobox"]').first()
    await expect(timeRangeSelect).toBeVisible()
  })

  test('should change time range filter', async ({ page }) => {
    // Open time range dropdown
    await page.locator('button[role="combobox"]').first().click()

    // Select different option
    const options = page.locator('[role="option"]')
    if (await options.count() > 1) {
      await options.nth(1).click()
      await page.waitForLoadState('networkidle')
    }
  })

  test('should display agent performance metrics', async ({ page }) => {
    // Wait for loading to complete (skeleton should disappear)
    await page.waitForSelector('.card-depth', { timeout: 15000 })

    // Check stat card labels are visible (use exact match to avoid matching chart descriptions)
    await expect(page.getByText('Transfers Handled', { exact: true })).toBeVisible()
    await expect(page.getByText('Active Conversations', { exact: true })).toBeVisible()
  })

  test('should apply agent + instance + date range filters together', async ({ page }) => {
    let latestAnalyticsUrl = ''
    page.on('request', (request) => {
      if (request.url().includes('/api/analytics/agents') && !request.url().includes('/ratings/export')) {
        latestAnalyticsUrl = request.url()
      }
    })

    const comboboxes = page.locator('button[role="combobox"]')
    await expect(comboboxes.first()).toBeVisible()

    let selectedAgent = false
    await comboboxes.nth(0).click()
    const agentItems = page.locator('[cmdk-item]')
    const agentCount = await agentItems.count()
    if (agentCount > 1) {
      await agentItems.nth(1).click()
      selectedAgent = true
    } else {
      await page.keyboard.press('Escape')
    }

    let selectedInstance = false
    await comboboxes.nth(1).click()
    const instanceOptions = page.locator('[role="option"]')
    const instanceCount = await instanceOptions.count()
    if (instanceCount > 1) {
      await instanceOptions.nth(1).click()
      selectedInstance = true
    } else {
      await page.keyboard.press('Escape')
    }

    await comboboxes.nth(2).click()
    const rangeOption = page.getByRole('option').filter({ hasText: /7|Last 7 Days/i }).first()
    if (await rangeOption.count()) {
      await rangeOption.click()
    } else {
      await page.keyboard.press('Escape')
    }

    await page.waitForLoadState('networkidle')
    expect(latestAnalyticsUrl).toContain('/api/analytics/agents')
    expect(latestAnalyticsUrl).toMatch(/from=\d{4}-\d{2}-\d{2}/)
    expect(latestAnalyticsUrl).toMatch(/to=\d{4}-\d{2}-\d{2}/)
    if (selectedAgent) {
      expect(latestAnalyticsUrl).toContain('agent_id=')
    }
    if (selectedInstance) {
      expect(latestAnalyticsUrl).toContain('instance_id=')
    }
  })
})

test.describe('Agent Analytics - Agent Role', () => {
  test('should allow agents to view their own analytics', async ({ page }) => {
    await loginAsAgent(page)
    await page.goto('/analytics/agents')
    await page.waitForLoadState('networkidle')

    // Agents should be able to see the analytics page (with limited data)
    await expect(page).toHaveURL(/\/analytics\/agents/)
  })
})

test.describe('Agent Analytics - Manager Role', () => {
  test('should show instance filter for manager users', async ({ page }) => {
    await loginAsManager(page)
    await page.goto('/analytics/agents')
    await page.waitForLoadState('networkidle')

    await expect(page.getByTestId('agent-analytics-instance-filter')).toBeVisible()
  })
})
