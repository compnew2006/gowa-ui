import { test, expect } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/')
    await page.waitForLoadState('networkidle')
  })

  test('should display dashboard page', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Dashboard')
    await expect(page.getByText('Customizable analytics overview')).toBeVisible()
  })

  test('should display stat cards', async ({ page }) => {
    // Wait for widget cards to load (not skeleton) - use exact text matching to avoid duplicates
    await expect(page.getByText('Total Messages', { exact: true })).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Active Contacts', { exact: true })).toBeVisible()
    await expect(page.getByText('Total Campaigns', { exact: true })).toBeVisible()
  })

  test('should display time range filter', async ({ page }) => {
    // Check for time range selector
    const timeRangeSelect = page.locator('button[role="combobox"]').first()
    await expect(timeRangeSelect).toBeVisible()
  })

  test('should change time range', async ({ page }) => {
    // Open time range dropdown
    await page.locator('button[role="combobox"]').first().click()

    // Select "Last 7 Days"
    await page.locator('[role="option"]').filter({ hasText: 'Last 7 Days' }).click()

    // Should reload data
    await page.waitForLoadState('networkidle')

    // Check that the selection is updated
    await expect(page.locator('button[role="combobox"]').first()).toContainText('Last 7 Days')
  })

  test('should display recent messages widget', async ({ page }) => {
    // Widget names are displayed in spans, not headings
    await expect(page.getByText('Recent Messages', { exact: true })).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Latest conversations from your contacts')).toBeVisible()
  })

  test('should display quick actions widget', async ({ page }) => {
    const main = page.locator('main')
    await expect(main.getByText('Quick Actions', { exact: true })).toBeVisible({ timeout: 15000 })
    await expect(main.getByText('Common tasks and shortcuts')).toBeVisible()

    // Check for quick action links - scope to main to avoid sidebar duplicates
    await expect(main.locator('a[href="/chat"]')).toBeVisible()
    await expect(main.locator('a[href="/campaigns"]')).toBeVisible()
    await expect(main.locator('a[href="/templates"]')).toBeVisible()
  })

  // Rule 3: collapsed three near-duplicate quick-action navigation tests
  // (chat / campaigns / templates — identical bodies, differing
  // only by href) into one data-driven test.
  test('quick actions navigate to each destination', async ({ page }) => {
    const destinations: Array<{ href: string; urlRegex: RegExp }> = [
      { href: '/chat', urlRegex: /\/chat/ },
      { href: '/campaigns', urlRegex: /\/campaigns/ },
      { href: '/templates', urlRegex: /\/templates/ },
    ]

    for (const { href, urlRegex } of destinations) {
      // Re-navigate to the dashboard between iterations so the quick-action
      // links are present for the next click.
      await page.goto('/')
      await page.waitForLoadState('networkidle')

      await page.locator(`main a[href="${href}"]`).click()
      await expect(page).toHaveURL(urlRegex)
    }
  })

  test('should show custom date range picker', async ({ page }) => {
    // Open time range dropdown
    await page.locator('button[role="combobox"]').first().click()

    // Select "Custom Range"
    await page.locator('[role="option"]').filter({ hasText: 'Custom Range' }).click()

    // Should show date picker button
    await expect(page.getByRole('button', { name: /Select|Calendar/i })).toBeVisible()
  })

  test('should display percentage change indicators', async ({ page }) => {
    // Wait for stats to load
    await expect(page.locator('text=Total Messages')).toBeVisible({ timeout: 15000 })

    // Should show percentage changes with comparison text
    await expect(page.locator('text=/from (yesterday|previous|last month)/i').first()).toBeVisible()
  })
})
