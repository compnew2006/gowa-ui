import { Page, Locator, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * Accounts Page - WhatsApp numbers management (DataTable + Detail Page).
 * Device lifecycle (pairing/connect) lives on the GOWA Gateway page; this
 * page links out to it.
 */
export class AccountsPage extends BasePage {
  readonly heading: Locator
  readonly addButton: Locator
  readonly alertDialog: Locator
  readonly tableBody: Locator

  constructor(page: Page) {
    super(page)
    this.heading = page.locator('h1').filter({ hasText: 'WhatsApp Accounts' })
    this.addButton = page.getByRole('button', { name: /Add WhatsApp Number/i }).first()
    this.dialog = page.locator('[role="dialog"][data-state="open"]')
    this.alertDialog = page.locator('[role="alertdialog"]')
    this.tableBody = page.locator('tbody')
  }

  async goto() {
    await this.page.goto('/settings/accounts')
    await this.page.waitForLoadState('networkidle')
  }

  async navigateToGateway() {
    await this.addButton.click()
    await this.page.waitForLoadState('networkidle')
    await expect(this.page).toHaveURL(/\/settings\/gowa-servers/)
  }

  async navigateToAccount(name: string) {
    const row = this.page.locator('tr').filter({ hasText: name })
    await row.locator('a').first().click()
    await this.page.waitForLoadState('networkidle')
  }

  async saveAccount() {
    await this.page.getByRole('button', { name: /Create|Save/i }).first().click()
    await this.page.waitForLoadState('networkidle')
  }

  async deleteAccount(name: string) {
    const row = this.page.locator('tr').filter({ hasText: name })
    await row.locator('button').filter({ has: this.page.locator('svg.text-destructive') }).click()
    await this.alertDialog.waitFor({ state: 'visible' })
  }

  async confirmDelete() {
    await this.alertDialog.getByRole('button', { name: 'Delete' }).click()
    await this.alertDialog.waitFor({ state: 'hidden' })
  }

  async cancelDelete() {
    await this.alertDialog.getByRole('button', { name: 'Cancel' }).click()
    await this.alertDialog.waitFor({ state: 'hidden' })
  }

  // Toast helpers
  async expectToast(text: string | RegExp) {
    const toast = this.page.locator('[data-sonner-toast]').filter({ hasText: text })
    await expect(toast).toBeVisible({ timeout: 5000 })
    return toast
  }

  // Assertions
  async expectPageVisible() {
    await expect(this.heading).toBeVisible()
  }

  async expectGatewayCardVisible() {
    await expect(
      this.page.getByRole('heading', { name: 'GOWA Gateway' }),
    ).toBeVisible()
  }

  async expectAccountExists(name: string) {
    await expect(this.page.locator('tr').filter({ hasText: name })).toBeVisible()
  }

  async expectAccountNotExists(name: string) {
    await expect(this.page.locator('tr').filter({ hasText: name })).not.toBeVisible()
  }

  async expectEmptyState() {
    await expect(this.page.getByText('No WhatsApp accounts')).toBeVisible()
  }
}
