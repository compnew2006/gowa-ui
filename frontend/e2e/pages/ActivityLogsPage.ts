import { Page, Locator, expect } from '@playwright/test'
import { BasePage } from './BasePage'

export class ActivityLogsPage extends BasePage {
  readonly heading: Locator
  readonly table: Locator
  readonly refreshButton: Locator
  readonly applyFiltersButton: Locator
  readonly eventTypeFilterInput: Locator

  constructor(page: Page) {
    super(page)
    this.heading = page.locator('h1').filter({ hasText: 'Activity Logs' }).first()
    this.table = page.locator('table').first()
    this.refreshButton = page.getByRole('button', { name: /^Refresh$/i }).first()
    this.applyFiltersButton = page.getByRole('button', { name: /^Apply Filters$/i })
    this.eventTypeFilterInput = page.getByPlaceholder('e.g. auth.login')
  }

  async goto() {
    await this.page.goto('/activity-logs')
    await this.page.waitForLoadState('networkidle')
  }

  async expectPageVisible() {
    await expect(this.heading).toBeVisible()
    await expect(this.table).toBeVisible()
  }

  async clickRefresh() {
    await this.refreshButton.click()
    await this.page.waitForTimeout(300)
  }

  async applyEventTypeFilter(eventType: string) {
    await this.eventTypeFilterInput.fill(eventType)
  }

  async clickApplyFilters() {
    await this.applyFiltersButton.click()
    await this.page.waitForTimeout(300)
  }

  async expectRowContains(text: string) {
    await expect(this.table).toContainText(text, { timeout: 10000 })
  }

  async expectRowNotContains(text: string) {
    await expect(this.table).not.toContainText(text)
  }
}
