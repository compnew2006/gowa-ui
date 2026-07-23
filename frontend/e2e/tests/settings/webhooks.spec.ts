import { test, expect, type Page, type Locator } from '@playwright/test'
import { TablePage } from '../../pages'
import { loginAsAdmin, createWebhookFixture, generateUniqueName, ApiHelper, verifyAuditLogged } from '../../helpers'
import { SUPER_ADMIN, createTestScope } from '../../framework'

const scope = createTestScope('webhooks')

// Read the column header's aria-sort attribute directly (the TablePage POM
// exposes icon-based sort detection; aria-sort is the stronger, accessible
// contract and toggling asc/desc on click is what we want to prove).
async function getAriaSort(page: Page, column: string): Promise<'ascending' | 'descending' | null> {
  const header = page.locator('thead th').filter({ hasText: column }).first()
  if (await header.count() === 0) return null
  const value = await header.getAttribute('aria-sort')
  if (value === 'ascending' || value === 'descending') return value
  return null
}

function nameInput(page: Page): Locator {
  return page.getByPlaceholder('My Helpdesk Integration')
}

function urlInput(page: Page): Locator {
  return page.getByPlaceholder('https://example.com/webhook')
}

function saveButton(page: Page): Locator {
  return page.getByRole('button', { name: /^(Create|Save)$/i }).first()
}

async function gotoCreateWebhook(page: Page) {
  await page.getByRole('button', { name: /^Add Webhook$/i }).first().click()
  await page.waitForURL(/\/settings\/webhooks\/new$/)
  await page.waitForLoadState('networkidle')
}

async function openWebhookDetail(tablePage: TablePage, page: Page, rowText: string) {
  await tablePage.search(rowText)
  await page.locator('tbody tr .font-medium').getByText(rowText, { exact: true }).first().click()
  await page.waitForURL(/\/settings\/webhooks\/[a-f0-9-]+$/)
  await page.waitForLoadState('networkidle')
}

test.describe('Webhooks Management', () => {
  let tablePage: TablePage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/webhooks')
    await page.waitForLoadState('networkidle')

    tablePage = new TablePage(page)
  })

  test('should display webhooks list', async ({ page }) => {
    await expect(tablePage.tableBody).toBeVisible()
  })

  test('should navigate to create webhook page', async ({ page }) => {
    await gotoCreateWebhook(page)
    expect(page.url()).toContain('/settings/webhooks/new')
  })

  test('should create a new webhook', async ({ page, request }) => {
    const webhook = createWebhookFixture()

    await gotoCreateWebhook(page)

    const nameField = nameInput(page)
    await nameField.fill(webhook.name)
    await urlInput(page).fill(webhook.url)

    // Select at least one event
    const checkbox = page.locator('button[role="checkbox"]').first()
    if (await checkbox.isVisible()) await checkbox.click()

    await saveButton(page).click()
    await page.waitForURL(/\/settings\/webhooks\/[a-f0-9-]+$/, { timeout: 10000 })
    await page.waitForLoadState('networkidle')

    // Capture the new webhook's id from the detail-page URL for the audit
    // assertion. This is the LITERAL example flow from ARCHITECTURE.md.
    const match = page.url().match(/\/settings\/webhooks\/([a-f0-9-]+)$/)
    const webhookId = match ? match[1] : null

    // Verify in list
    await page.goto('/settings/webhooks')
    await page.waitForLoadState('networkidle')
    await tablePage.search(webhook.name)
    await tablePage.expectRowExists(webhook.name)

    // API side-channel: confirm the audit trail recorded the creation.
    if (webhookId) {
      await verifyAuditLogged(request, 'webhook', webhookId, 'created')
    }
  })

  test('should edit existing webhook', async ({ page, request }) => {
    // Create a webhook first
    const webhook = createWebhookFixture()
    await gotoCreateWebhook(page)
    await nameInput(page).fill(webhook.name)
    await urlInput(page).fill(webhook.url)
    const checkbox = page.locator('button[role="checkbox"]').first()
    if (await checkbox.isVisible()) await checkbox.click()
    await saveButton(page).click()
    await page.waitForURL(/\/settings\/webhooks\/[a-f0-9-]+$/, { timeout: 10000 })
    await page.waitForLoadState('networkidle')

    // Navigate back and open detail
    await page.goto('/settings/webhooks')
    await page.waitForLoadState('networkidle')
    await openWebhookDetail(tablePage, page, webhook.name)

    const match = page.url().match(/\/settings\/webhooks\/([a-f0-9-]+)$/)

    const updatedName = webhook.name + ' Updated'
    await nameInput(page).fill(updatedName)
    await page.waitForTimeout(300)
    await saveButton(page).click()
    await page.waitForLoadState('networkidle')

    // Verify in list
    await page.goto('/settings/webhooks')
    await page.waitForLoadState('networkidle')
    await tablePage.search(updatedName)
    await tablePage.expectRowExists(updatedName)

    // API side-channel: confirm the audit trail recorded the update.
    const webhookId = match ? match[1] : null
    if (webhookId) {
      await verifyAuditLogged(request, 'webhook', webhookId, 'updated')
    }
  })

  test('should delete webhook', async ({ page, request }) => {
    const webhook = createWebhookFixture({ name: scope.name('delete') })

    await gotoCreateWebhook(page)
    await nameInput(page).fill(webhook.name)
    await urlInput(page).fill(webhook.url)
    const checkbox = page.locator('button[role="checkbox"]').first()
    if (await checkbox.isVisible()) await checkbox.click()
    await saveButton(page).click()
    await page.waitForURL(/\/settings\/webhooks\/[a-f0-9-]+$/, { timeout: 10000 })
    await page.waitForLoadState('networkidle')

    const match = page.url().match(/\/settings\/webhooks\/([a-f0-9-]+)$/)
    const webhookId = match ? match[1] : null

    // Navigate back and delete from list
    await page.goto('/settings/webhooks')
    await page.waitForLoadState('networkidle')
    await tablePage.search(webhook.name)
    await tablePage.expectRowExists(webhook.name)
    await tablePage.deleteRow(webhook.name)
    await tablePage.expectRowNotExists(webhook.name)

    // API side-channel: confirm the audit trail recorded the deletion.
    if (webhookId) {
      await verifyAuditLogged(request, 'webhook', webhookId, 'deleted')
    }
  })
})

test.describe('Webhook Toggle Confirmation', () => {
  let api: ApiHelper
  let webhookId: string

  test.beforeEach(async ({ request }) => {
    // Seed a webhook via the API so the toggle test has a real, enabled subject.
    // Previously the beforeEach didn't seed anything and the test was a chain of
    // `if (toggle visible) { if (dialog visible) { cancel } }` — it asserted
    // nothing and passed on an empty list.
    api = new ApiHelper(request)
    await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
    const resp = await api.post('/api/webhooks', {
      name: generateUniqueName('Hook'),
      url: 'https://example.com/hook-' + Date.now(),
    })
    expect(resp.ok(), `seed webhook: ${await resp.text()}`).toBe(true)
    const body = await resp.json()
    webhookId = body.data?.id ?? body.data?.webhook?.id ?? body.data?.webhook?.uuid
  })

  test.afterEach(async () => {
    if (webhookId) {
      await api.del('/api/webhooks/' + webhookId).catch(() => {})
    }
  })

  test('should show confirmation when disabling webhook', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/webhooks')
    await page.waitForLoadState('networkidle')

    // The seeded webhook gives a guaranteed toggle, so the confirmation flow is
    // asserted unconditionally instead of skipped when no toggle is visible.
    const toggleSwitch = page.getByRole('switch').first()
    await expect(toggleSwitch).toBeVisible({ timeout: 10000 })
    await toggleSwitch.click()

    // Disabling an active webhook prompts a confirmation alert dialog.
    const alertDialog = page.locator('[role="alertdialog"]')
    await expect(alertDialog).toBeVisible({ timeout: 5000 })

    // Cancel so the webhook stays enabled (cleanup via afterEach).
    await alertDialog.getByRole('button', { name: /cancel/i }).click()
    await alertDialog.waitFor({ state: 'hidden' })
  })
})

test.describe('Webhooks - Table Sorting', () => {
  let tablePage: TablePage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/webhooks')
    await page.waitForLoadState('networkidle')
    tablePage = new TablePage(page)
  })

  test('webhooks table sorts by each column', async ({ page }) => {
    // Collapsed from four near-identical tests (sort by Name / URL / Status /
    // Created). Each used to assert only that the sort indicator was non-null;
    // this loop additionally asserts the aria-sort attribute toggles between
    // ascending and descending on successive clicks.
    for (const column of ['Name', 'URL', 'Status', 'Created']) {
      await tablePage.clickColumnHeader(column)
      const firstSort = await getAriaSort(page, column)
      expect(firstSort, `${column}: expected a sort direction after first click`).not.toBeNull()

      await tablePage.clickColumnHeader(column)
      const secondSort = await getAriaSort(page, column)
      expect(secondSort, `${column}: expected a sort direction after second click`).not.toBeNull()
      expect(secondSort, `${column}: sort should toggle on second click`).not.toEqual(firstSort)
    }
  })

  test('should toggle sort direction', async () => {
    await tablePage.clickColumnHeader('Name')
    const firstDirection = await tablePage.getSortDirection('Name')

    await tablePage.clickColumnHeader('Name')
    const secondDirection = await tablePage.getSortDirection('Name')

    expect(firstDirection).not.toEqual(secondDirection)
  })
})
