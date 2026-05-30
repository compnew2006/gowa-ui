import { Page, Locator, expect } from '@playwright/test'
import { BasePage } from './BasePage'

export class WhatsAppFilterPage extends BasePage {
  readonly heading: Locator
  readonly connectionSelect: Locator
  readonly pasteNumbersTab: Locator
  readonly uploadCsvTab: Locator
  readonly textarea: Locator
  readonly fileInput: Locator
  readonly submitButton: Locator
  readonly table: Locator
  readonly alertDialog: Locator

  constructor(page: Page) {
    super(page)
    this.heading = page.locator('h1').filter({ hasText: 'WhatsApp Number Filter' }).first()
    this.connectionSelect = page.locator('button[role="combobox"]').first()
    this.pasteNumbersTab = page.getByRole('button', { name: 'Paste Numbers' })
    this.uploadCsvTab = page.getByRole('button', { name: 'Upload CSV' })
    this.textarea = page.locator('textarea')
    this.fileInput = page.locator('input[type="file"]')
    this.submitButton = page.getByRole('button', { name: 'Start Verification' })
    this.table = page.locator('table')
    this.alertDialog = page.locator('[role="alertdialog"]')
  }

  async goto() {
    await this.page.goto('/settings/whatsapp-filter')
    await this.page.waitForLoadState('networkidle')
  }

  async expectPageVisible() {
    await expect(this.heading).toBeVisible()
  }

  async selectConnection(name: string) {
    await this.connectionSelect.click()
    await this.page.locator('[role="option"]').filter({ hasText: name }).click()
  }

  async fillPasteNumbers(numbers: string[]) {
    await this.pasteNumbersTab.click()
    await this.textarea.fill(numbers.join('\n'))
  }

  async startCheck() {
    await this.submitButton.click()
    await this.page.waitForLoadState('networkidle')
  }

  async expectBatchResultsPanel() {
    await expect(this.page.locator('text=Verification Results - Batch')).toBeVisible({ timeout: 10000 })
  }

  async expectRowExists(text: string) {
    await expect(this.table).toContainText(text, { timeout: 10000 })
  }

  async clickCampaignsTab() {
    await this.page.getByRole('button', { name: 'Back to Campaigns' }).click()
    await this.page.waitForLoadState('networkidle')
  }

  async deleteBatchRow(batchIdSlice: string) {
    const row = this.page.locator('tbody tr').filter({ hasText: batchIdSlice })
    await expect(row).toBeVisible({ timeout: 10000 })
    // Delete icon button is second button in actions column
    await row.locator('td:last-child button').nth(1).click()
    await this.alertDialog.waitFor({ state: 'visible' })
    await this.alertDialog.getByRole('button', { name: 'Delete' }).click()
    await this.alertDialog.waitFor({ state: 'hidden' })
  }
}
