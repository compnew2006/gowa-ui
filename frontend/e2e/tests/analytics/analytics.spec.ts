import { test, expect } from '@playwright/test'
import { loginAsAdmin, loginAsAgent } from '../../helpers'

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
    const timeRangeSelect = page.locator('button[role="combobox"]').first()
    // Open time range dropdown
    await timeRangeSelect.click()

    // Select different option
    const options = page.locator('[role="option"]')
    const count = await options.count()
    // Rule 4: original used `if (count > 1)` with no else — a no-op when
    // the dropdown had 0-1 options. Now we assert the dropdown actually
    // exposes more than one option (the filter is meaningful) and that
    // selecting the second one updates the trigger's label.
    expect(count).toBeGreaterThan(1)

    const selectedLabel = (await options.nth(1).textContent()) ?? ''
    expect(selectedLabel.trim().length).toBeGreaterThan(0)
    await options.nth(1).click()
    await page.waitForLoadState('networkidle')

    // Assert the combobox now reflects the newly-selected label.
    await expect(timeRangeSelect).toContainText(selectedLabel.trim())
  })

  test('should display agent performance metrics', async ({ page }) => {
    // Wait for loading to complete (skeleton should disappear)
    await page.waitForSelector('.card-depth', { timeout: 15000 })

    // Check stat card labels are visible (use exact match to avoid matching chart descriptions)
    await expect(page.getByText('Messages Sent', { exact: true })).toBeVisible()
    await expect(page.getByText('Break Time', { exact: true })).toBeVisible()
  })
})

test.describe('Agent Analytics - Agent Role', () => {
  test('should allow agents to view their own analytics', async ({ page }) => {
    await loginAsAgent(page)
    await page.goto('/analytics/agents')
    await page.waitForLoadState('networkidle')

    // Agents should be able to see the analytics page (with limited data)
    await expect(page).toHaveURL(/\/analytics\/agents/)

    // Rule 4: original only asserted the URL — a permission gate that
    // never asserted the gated content actually rendered. Assert the page
    // heading (the same one the admin-role test checks) is visible, which
    // proves the agent wasn't bounced to a permission-denied screen.
    await expect(page.locator('h1')).toContainText('Agent Analytics')
    // And explicitly: no permission-denied message surfaces.
    await expect(page.getByText(/you do not have|permission denied|forbidden/i)).toHaveCount(0)
  })
})
